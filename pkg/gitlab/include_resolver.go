package gitlab

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/m13tLabs/glab-docs/pkg/util"
)

// IncludeSummary is what an included file (plus everything it transitively includes) adds to the
// including pipeline, for the Description column of the Includes table.
type IncludeSummary struct {
	// Description is the included file's own leading `# --` comment, if any.
	Description string
	// Files are the included file followed by everything it (transitively) includes, each parsed
	// like a documented component, in include order. Their variables and jobs are what the
	// include adds; deduplicating names across files is left to the renderer.
	Files []ComponentDocumentationInfo
}

// maxIncludeDepth bounds how deep nested includes are followed. GitLab itself caps the total
// number of includes, not the depth, but anything this deep is almost certainly a cycle the
// visited-set didn't catch (e.g. the same file reached through differently-spelled refs).
const maxIncludeDepth = 10

// includeSource identifies one concrete file an include resolves to. BaseURL is empty for a file
// of the documented repository itself, in which case Project is empty too and File is relative to
// IncludeResolver.LocalRoot (or absolute, for one of LocalComponents). Such a file is read from
// the working tree when Ref is empty, otherwise from local git at Ref.
type includeSource struct {
	BaseURL string
	Project string
	Ref     string
	File    string
}

func (s includeSource) key() string {
	if s.BaseURL == "" && s.Ref != "" {
		return "git:" + s.Ref + ":" + s.File
	}
	if s.BaseURL == "" {
		return "local:" + s.File
	}
	return fmt.Sprintf("%s/%s@%s:%s", s.BaseURL, s.Project, s.Ref, s.File)
}

type fetchResult struct {
	contents []byte
	err      error
}

// IncludeResolver fetches the files a pipeline's `include:` entries point at - from the local
// checkout for `local:` includes, and through the GitLab repository files API for `project:` and
// `component:` includes - and summarizes the variables and jobs they add. Fetched files are
// cached, so a project included by many components is only downloaded once. `remote:` and
// `template:` includes aren't resolved.
type IncludeResolver struct {
	// ServerURL is the GitLab base URL (e.g. https://gitlab.com), without a trailing slash. It
	// resolves `project:` includes and `$CI_SERVER_FQDN` component addresses; when empty only
	// `local:` includes and components with a literal host are resolved.
	ServerURL string
	// Token is sent as PRIVATE-TOKEN, JobToken (when Token is empty) as JOB-TOKEN - and either
	// only to ServerURL, never to another host named in a component address.
	Token    string
	JobToken string
	// LocalRoot is the directory `local:` includes of the documented repository are relative to.
	LocalRoot string
	// LocalProjects are the path(s) the documented repository is known by (e.g. from
	// $CI_PROJECT_PATH, --component-prefix, the git remote), compared case-insensitively. A
	// `component:` include whose project is one of them is read from LocalComponents instead of
	// being fetched, so a repository's own components resolve without any API access.
	LocalProjects []string
	// LocalComponents maps a discovered component's name to its file on disk.
	LocalComponents map[string]string
	// CurrentRefs are the names of the checked-out revision (HEAD, its branch, its SHA, ...). A
	// `project:` include of the documented repository at one of them is read from the working
	// tree - uncommitted edits included - rather than from git.
	CurrentRefs []string
	Client      *http.Client

	mu    sync.Mutex
	cache map[string]fetchResult
}

// NewIncludeResolver returns a resolver with a default HTTP client.
func NewIncludeResolver(serverURL, token, jobToken, localRoot string) *IncludeResolver {
	return &IncludeResolver{
		ServerURL: strings.TrimRight(serverURL, "/"),
		Token:     token,
		JobToken:  jobToken,
		LocalRoot: localRoot,
		Client:    &http.Client{Timeout: 15 * time.Second},
		cache:     map[string]fetchResult{},
	}
}

// ResolveAll fills in the Summary of every include of every component in infoByFile, leaving it
// nil where the include can't be resolved (a failed fetch is logged, never fatal).
func (r *IncludeResolver) ResolveAll(infoByFile map[string]ComponentDocumentationInfo) {
	for relFile, info := range infoByFile {
		for i, item := range info.Includes {
			info.Includes[i].Summary = r.Summarize(item)
		}
		infoByFile[relFile] = info
	}
}

// Summarize resolves a single top-level include of the documented repository.
func (r *IncludeResolver) Summarize(item IncludeItem) *IncludeSummary {
	sources := r.sourcesFor(item, includeSource{})
	if len(sources) == 0 {
		return nil
	}

	for _, src := range sources {
		summary := &IncludeSummary{}
		info, err := r.parse(src)
		if err != nil {
			log.Debugf("Could not resolve include %s: %s", src.key(), err)
			continue // e.g. a component in templates/<name>/template.yml rather than templates/<name>.yml
		}
		summary.Description = info.Description
		r.collect(src, info, summary, map[string]bool{src.key(): true}, 0)
		return summary
	}

	log.Warnf("Could not resolve %s include %s for the Includes table description", item.Kind, item.Location)
	return nil
}

func (r *IncludeResolver) collect(src includeSource, info ComponentDocumentationInfo, summary *IncludeSummary, visited map[string]bool, depth int) {
	summary.Files = append(summary.Files, info)

	if depth >= maxIncludeDepth {
		log.Warnf("Not following includes of %s any deeper (max depth %d)", src.key(), maxIncludeDepth)
		return
	}

	for _, nested := range info.Includes {
		for _, nestedSrc := range r.sourcesFor(nested, src) {
			if visited[nestedSrc.key()] {
				break
			}
			nestedInfo, err := r.parse(nestedSrc)
			if err != nil {
				log.Debugf("Could not resolve nested include %s: %s", nestedSrc.key(), err)
				continue
			}
			visited[nestedSrc.key()] = true
			r.collect(nestedSrc, nestedInfo, summary, visited, depth+1)
			break
		}
	}
}

// sourcesFor returns the candidate files an include resolves to, in the order they should be
// tried, relative to parent - the file doing the including (the zero value for the documented
// repository itself). An empty result means the include can't be resolved at all.
func (r *IncludeResolver) sourcesFor(item IncludeItem, parent includeSource) []includeSource {
	switch item.Kind {
	case "local":
		file := strings.TrimPrefix(item.Location, "/")
		if file == "" || strings.ContainsAny(file, "$*") {
			return nil
		}
		// A `local:` include is relative to the root of whichever project the including file
		// lives in - for a file fetched from another project, that project at the same ref.
		return []includeSource{{BaseURL: parent.BaseURL, Project: parent.Project, Ref: parent.Ref, File: file}}

	case "project":
		project, file := strings.Trim(item.Location, "/"), strings.TrimPrefix(item.File, "/")
		if project == "" || file == "" {
			return nil
		}
		ref := item.Ref
		if ref == "" {
			ref = "HEAD" // GitLab uses the project's default branch; HEAD is the API's name for it
		}

		sources := make([]includeSource, 0, 2)
		// The documented repository itself: read it from local git at the include's ref, falling
		// back to the API below when git doesn't have that ref (e.g. a shallow CI clone).
		if r.isOwnProject(project) && !strings.Contains(file+item.Ref, "$") {
			if item.Ref == "" || slices.Contains(r.CurrentRefs, item.Ref) {
				sources = append(sources, includeSource{File: file})
			} else {
				sources = append(sources, includeSource{Ref: item.Ref, File: file})
			}
		}
		if r.ServerURL != "" && !strings.Contains(project+file+item.Ref, "$") {
			sources = append(sources, includeSource{BaseURL: r.ServerURL, Project: project, Ref: ref, File: file})
		}
		return sources

	case "component":
		return r.componentSources(item)
	}
	return nil
}

// componentSources maps a `component:` address (`<host>/<project path>/<name>`, with Ref split
// off by parseIncludes) to the two file layouts GitLab accepts for it:
// https://docs.gitlab.com/ci/components/#directory-structure
func (r *IncludeResolver) componentSources(item IncludeItem) []includeSource {
	if src, ok := r.localComponentSource(item); ok {
		return []includeSource{src}
	}

	// `~latest` and other `~` refs are resolved by GitLab against the project's releases, which
	// the repository files API knows nothing about.
	if item.Ref == "" || strings.HasPrefix(item.Ref, "~") {
		return nil
	}

	var baseURL, path string
	if rest, ok := strings.CutPrefix(item.Location, "$CI_SERVER_FQDN/"); ok {
		baseURL, path = r.ServerURL, rest
	} else {
		host, rest, found := strings.Cut(item.Location, "/")
		if !found {
			return nil
		}
		baseURL, path = "https://"+host, rest
		if server, err := url.Parse(r.ServerURL); err == nil && server.Host == host {
			baseURL = r.ServerURL
		}
	}
	if baseURL == "" || strings.Contains(path+item.Ref, "$") {
		return nil
	}

	idx := strings.LastIndex(path, "/")
	if idx <= 0 {
		return nil
	}
	project, name := path[:idx], path[idx+1:]
	return []includeSource{
		{BaseURL: baseURL, Project: project, Ref: item.Ref, File: "templates/" + name + ".yml"},
		{BaseURL: baseURL, Project: project, Ref: item.Ref, File: "templates/" + name + "/template.yml"},
	}
}

// localComponentSource resolves a `component:` include of one of the documented repository's own
// components (see LocalProjects) to its file on disk. The address's ref is ignored - the checkout
// being documented is the best available version of it - and `$CI_PROJECT_PATH` in the address is
// taken to be the local project.
func (r *IncludeResolver) localComponentSource(item IncludeItem) (includeSource, bool) {
	address := item.Location
	if rest, ok := strings.CutPrefix(address, "$CI_SERVER_FQDN/"); ok {
		address = rest
	} else if _, rest, found := strings.Cut(address, "/"); found {
		address = rest // drop the literal host
	} else {
		return includeSource{}, false
	}

	idx := strings.LastIndex(address, "/")
	if idx <= 0 {
		return includeSource{}, false
	}
	project, name := address[:idx], address[idx+1:]

	file, ok := r.LocalComponents[name]
	if !ok {
		return includeSource{}, false
	}
	return includeSource{File: file}, r.isOwnProject(project)
}

// isOwnProject reports whether a project path in an include refers to the documented repository.
func (r *IncludeResolver) isOwnProject(project string) bool {
	for _, local := range r.LocalProjects {
		if local != "" && (project == "$CI_PROJECT_PATH" || strings.EqualFold(project, local)) {
			return true
		}
	}
	return false
}

func (r *IncludeResolver) parse(src includeSource) (ComponentDocumentationInfo, error) {
	contents, err := r.fetch(src)
	if err != nil {
		return ComponentDocumentationInfo{}, err
	}
	return parseComponentContents(ComponentDocumentationInfo{
		SourceFile:           src.key(),
		Name:                 deriveComponentName(src.File),
		InputDescriptions:    map[string]ValueDescription{},
		VariableDescriptions: map[string]ValueDescription{},
	}, contents, DocumentationParsingConfig{})
}

func (r *IncludeResolver) fetch(src includeSource) ([]byte, error) {
	key := src.key()

	r.mu.Lock()
	if cached, ok := r.cache[key]; ok {
		r.mu.Unlock()
		return cached.contents, cached.err
	}
	r.mu.Unlock()

	var contents []byte
	var err error
	if src.BaseURL == "" && src.Ref != "" {
		contents, err = r.gitShow(src.Ref, src.File)
	} else if src.BaseURL == "" {
		path := src.File
		if !filepath.IsAbs(path) {
			path = filepath.Join(r.LocalRoot, filepath.FromSlash(path))
		}
		contents, err = os.ReadFile(path)
	} else {
		contents, err = r.fetchRemote(src)
	}

	r.mu.Lock()
	r.cache[key] = fetchResult{contents: contents, err: err}
	r.mu.Unlock()
	return contents, err
}

// gitShow reads file at ref from the local repository, trying ref as given and then as a branch
// of the "origin" remote (CI clones and fresh checkouts often lack a local branch of that name).
func (r *IncludeResolver) gitShow(ref, file string) ([]byte, error) {
	var lastErr error
	for _, rev := range []string{ref, "origin/" + ref} {
		contents, err := util.GitShow(r.LocalRoot, rev, file)
		if err == nil {
			return contents, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func (r *IncludeResolver) fetchRemote(src includeSource) ([]byte, error) {
	endpoint := fmt.Sprintf("%s/api/v4/projects/%s/repository/files/%s/raw?ref=%s",
		src.BaseURL, url.PathEscape(src.Project), url.PathEscape(src.File), url.QueryEscape(src.Ref))

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if src.BaseURL == r.ServerURL {
		if r.Token != "" {
			req.Header.Set("PRIVATE-TOKEN", r.Token)
		} else if r.JobToken != "" {
			req.Header.Set("JOB-TOKEN", r.JobToken)
		}
	}

	resp, err := r.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", endpoint, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

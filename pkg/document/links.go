package document

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/m13tLabs/glab-docs/pkg/gitlab"
)

// ComponentLink is where a discovered component's documentation ends up, keyed by the same
// search-root-relative path used as the key of the infoByFile map (e.g. "templates/build.yml").
// It lets other components' Includes tables link to it.
type ComponentLink struct {
	// OutputPath is the rendered doc's path, as the component's ComponentDirectory gives it
	// (i.e. including the search root).
	OutputPath string
	// Anchor is the Markdown heading anchor for this component's section within OutputPath, or
	// "" when the component has that whole output file to itself.
	Anchor string
}

// ComponentLinks is every discovered component's ComponentLink, plus where the search root sits
// within the project, since `local:` includes are addressed relative to the project root rather
// than to the search root (e.g. `templates/build.yml` for a component found as "build.yml" when
// --search-root points straight at `templates/`).
type ComponentLinks struct {
	// ByFile maps a component's search-root-relative path to its ComponentLink.
	ByFile map[string]ComponentLink
	// SearchRoot is the search root relative to ProjectDir, slash-separated ("." when they're
	// the same directory).
	SearchRoot string
	// ProjectDir is the absolute project root `local:` includes resolve against, used to check
	// that an included file that isn't itself a documented component exists before linking to
	// it. Empty disables those links.
	ProjectDir string
}

// NewComponentLinks returns the ComponentLinks for byFile, with the search root placed within
// projectDir. When the search root can't be placed inside projectDir, `local:` include paths are
// matched against the search-root-relative keys as-is and undocumented files aren't linked.
func NewComponentLinks(byFile map[string]ComponentLink, searchRoot, projectDir string) ComponentLinks {
	links := ComponentLinks{ByFile: byFile, SearchRoot: "."}
	absSearchRoot, err := realAbsPath(searchRoot)
	if err != nil {
		return links
	}
	absProjectDir, err := realAbsPath(projectDir)
	if err != nil {
		return links
	}
	rel, err := filepath.Rel(absProjectDir, absSearchRoot)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return links
	}
	links.SearchRoot = filepath.ToSlash(rel)
	links.ProjectDir = absProjectDir
	return links
}

// realAbsPath makes path absolute with symlinks resolved, so e.g. a search root under macOS'
// /tmp and a git toplevel reported as /private/tmp still compare equal.
func realAbsPath(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(abs)
}

// GroupComponentsByOutput buckets components by the output file their ComponentDirectory
// resolves to given outputFileName. Several sibling component files (e.g. flat `templates/*.yml`
// files, which all share the "templates" directory) can resolve to the same output path; each
// group is returned sorted by component name so callers render/link them in a stable order.
func GroupComponentsByOutput(infoByFile map[string]gitlab.ComponentDocumentationInfo, outputFileName string) map[string][]string {
	groups := make(map[string][]string)
	for relFile, info := range infoByFile {
		outputPath := filepath.Join(info.ComponentDirectory, outputFileName)
		groups[outputPath] = append(groups[outputPath], relFile)
	}
	for outputPath := range groups {
		group := groups[outputPath]
		sort.Slice(group, func(i, j int) bool {
			return infoByFile[group[i]].Name < infoByFile[group[j]].Name
		})
	}
	return groups
}

// BuildComponentLinks resolves where every component in groups will end up, for Includes tables
// to link to. The result is keyed like infoByFile, by search-root-relative path.
func BuildComponentLinks(infoByFile map[string]gitlab.ComponentDocumentationInfo, groups map[string][]string) map[string]ComponentLink {
	links := make(map[string]ComponentLink, len(infoByFile))
	for outputPath, relFiles := range groups {
		combined := len(relFiles) > 1
		for _, relFile := range relFiles {
			anchor := ""
			if combined {
				anchor = markdownAnchor(infoByFile[relFile].Name)
			}
			links[relFile] = ComponentLink{OutputPath: outputPath, Anchor: anchor}
		}
	}
	return links
}

var anchorStripRegex = regexp.MustCompile(`[^\w\- ]`)

// markdownAnchor approximates GitHub/GitLab's heading-to-anchor slugification, close enough for
// linking between sections of a document this tool itself generated.
func markdownAnchor(name string) string {
	s := strings.ToLower(name)
	s = anchorStripRegex.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, " ", "-")
	return s
}

// resolveIncludeLink returns the relative link that fromRelFile's Includes table should use to
// point at a `local:` include item: to the included component's documentation (with a "#anchor"
// suffix when the target shares a combined doc with something else) when it is a discovered
// component, otherwise straight to the included file when it exists in the project (e.g. a
// helper file outside the search root). It returns "" for a self-include, a file that can't be
// found, a wildcard or variable path, and any non-`local:` include, which can't be resolved to a
// local path.
func resolveIncludeLink(fromRelFile string, item gitlab.IncludeItem, links ComponentLinks) string {
	if item.Kind != "local" {
		return ""
	}
	from, ok := links.ByFile[fromRelFile]
	if !ok {
		return ""
	}

	// `local:` paths are relative to the project root; map them onto the search root.
	projectRelFile := path.Clean(filepath.ToSlash(strings.TrimPrefix(item.Location, "/")))
	targetRelFile := projectRelFile
	if links.SearchRoot != "." {
		var found bool
		targetRelFile, found = strings.CutPrefix(projectRelFile, links.SearchRoot+"/")
		if !found {
			targetRelFile = ""
		}
	}
	if targetRelFile == fromRelFile {
		return ""
	}

	target, ok := links.ByFile[targetRelFile]
	if !ok {
		return resolveIncludedFileLink(filepath.Dir(from.OutputPath), projectRelFile, links.ProjectDir)
	}

	if from.OutputPath == target.OutputPath {
		if target.Anchor == "" {
			return ""
		}
		return "#" + target.Anchor
	}

	rel, err := filepath.Rel(filepath.Dir(from.OutputPath), target.OutputPath)
	if err != nil {
		return ""
	}
	rel = filepath.ToSlash(rel)
	if target.Anchor != "" {
		rel += "#" + target.Anchor
	}
	return rel
}

// resolveIncludedFileLink returns the link from the doc in fromDir to the included file itself
// (relative to projectDir), or "" when projectDir is unknown, the path is a wildcard or holds a
// variable / input interpolation, or no such file exists.
func resolveIncludedFileLink(fromDir, projectRelFile, projectDir string) string {
	if projectDir == "" || strings.ContainsAny(projectRelFile, "*?[$") || strings.HasPrefix(projectRelFile, "../") {
		return ""
	}
	targetFile := filepath.Join(projectDir, filepath.FromSlash(projectRelFile))
	if stat, err := os.Stat(targetFile); err != nil || stat.IsDir() {
		return ""
	}
	absFromDir, err := realAbsPath(fromDir)
	if err != nil {
		return ""
	}
	rel, err := filepath.Rel(absFromDir, targetFile)
	if err != nil {
		return ""
	}
	return filepath.ToSlash(rel)
}

// ciServerFQDNPlaceholder is the predefined GitLab CI/CD variable authors commonly hardcode at
// the front of a `component:` include address (GitLab itself only resolves it at pipeline
// runtime), rather than using this tool's own --component-prefix. glab-docs sees it as literal
// text when parsing the source YAML.
const ciServerFQDNPlaceholder = "$CI_SERVER_FQDN"

// resolveComponentLocation strips a literal "$CI_SERVER_FQDN/" prefix from a `component:` include
// address for display and, when serverURL is configured (--gitlab-server-url), resolves it into a
// link straight to the component's source file rather than just the raw address (which isn't
// itself a browsable path - GitLab CI/CD components live under a project's `templates/`
// directory, per https://docs.gitlab.com/ci/components/#directory-structure). Per that same
// address format, the address's last path segment is always the component name and everything
// before it the project path, so the file is linked as `templates/<component-name>.yml`;
// glab-docs itself uses that flat layout (see templates/*.yml in this repo), which is also
// GitLab's simpler/more common convention over the alternative `templates/<name>/template.yml`
// subdirectory form, so it's the best default guess for an external, undocumented project. Ref
// (falling back to "HEAD" when no `ref:`/`@<ref>` was given) is used as the blob's ref. serverURL
// is expected caller-trimmed of any trailing slash.
func resolveComponentLocation(item gitlab.IncludeItem, serverURL string) (displayLocation, link string) {
	location := item.Location
	if !strings.HasPrefix(location, ciServerFQDNPlaceholder) {
		return location, ""
	}

	path := strings.TrimPrefix(strings.TrimPrefix(location, ciServerFQDNPlaceholder), "/")
	if serverURL == "" {
		return location, ""
	}

	idx := strings.LastIndex(path, "/")
	// no "/" at all means a single-segment address, which isn't a valid project path on its own
	// - link to it as-is rather than guessing at a file that can't exist.
	if idx == -1 {
		return path, serverURL + "/" + path
	}
	projectPath, componentName := path[:idx], path[idx+1:]

	ref := item.Ref
	if ref == "" {
		ref = "HEAD"
	}
	return path, fmt.Sprintf("%s/%s/-/blob/%s/templates/%s.yml", serverURL, projectPath, ref, componentName)
}

// resolveProjectLocation builds the display text and, when serverURL is configured
// (--gitlab-server-url), the link for a `project:` include - appending "(file: ...)" to the
// display text when the include also names a `file:`, and linking straight to that file's blob
// at Ref (falling back to "HEAD" when no `ref:` was given) rather than just the project root.
// serverURL is expected caller-trimmed of any trailing slash.
func resolveProjectLocation(item gitlab.IncludeItem, serverURL string) (displayLocation, link string) {
	displayLocation = item.Location
	if item.File != "" {
		displayLocation = fmt.Sprintf("%s (file: %s)", item.Location, item.File)
	}

	if serverURL == "" || item.Location == "" {
		return displayLocation, ""
	}

	link = serverURL + "/" + strings.TrimPrefix(item.Location, "/")
	if item.File != "" {
		ref := item.Ref
		if ref == "" {
			ref = "HEAD"
		}
		link += "/-/blob/" + ref + "/" + strings.TrimPrefix(item.File, "/")
	}
	return displayLocation, link
}

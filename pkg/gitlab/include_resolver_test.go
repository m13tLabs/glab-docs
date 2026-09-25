package gitlab

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helpersProject mimics a shared helpers project whose entry file only includes other local files
// and exports a `.load` template built from `!reference`s to them.
var helpersProject = map[string]string{
	".gitlab-ci/include.yml": `# -- Shared shell helpers.
include:
  - local: .gitlab-ci/helpers/common.yml
  - local: .gitlab-ci/variables/common.yml

.load:
  helpers:
    - !reference [.logging, script]
    - !reference [.git, script]
`,
	".gitlab-ci/helpers/common.yml": `# -- Logging helpers.
.logging:
  script:
    - echo log
.git:
  script:
    - git --version
`,
	".gitlab-ci/variables/common.yml": `variables:
  LOG_LEVEL: info
  GIT_DEPTH: "0"
`,
}

func variableNames(summary *IncludeSummary) []string {
	names := make([]string, 0)
	for _, f := range summary.Files {
		names = append(names, mappingKeyNames(f.Variables)...)
	}
	return names
}

func jobNames(summary *IncludeSummary) []string {
	names := make([]string, 0)
	for _, f := range summary.Files {
		for _, j := range f.Jobs {
			names = append(names, j.Name)
		}
	}
	return names
}

func fakeGitLab(t *testing.T, files map[string]string, token *string) *httptest.Server {
	t.Helper()
	const prefix = "/api/v4/projects/infra/jobs/helpers/repository/files/"
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*token = r.Header.Get("PRIVATE-TOKEN")
		// r.URL.Path is already unescaped, so project and file paths come back slash-separated.
		if len(r.URL.Path) <= len(prefix) || r.URL.Path[:len(prefix)] != prefix || r.URL.Query().Get("ref") != "main" {
			http.NotFound(w, r)
			return
		}
		file := r.URL.Path[len(prefix) : len(r.URL.Path)-len("/raw")]
		contents, ok := files[file]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(contents))
	}))
}

func TestSummarizeProjectIncludeFollowsNestedLocalIncludes(t *testing.T) {
	var token string
	server := fakeGitLab(t, helpersProject, &token)
	defer server.Close()

	resolver := NewIncludeResolver(server.URL+"/", "secret", "", t.TempDir())
	summary := resolver.Summarize(IncludeItem{Kind: "project", Location: "infra/jobs/helpers", File: ".gitlab-ci/include.yml", Ref: "main"})

	require.NotNil(t, summary)
	assert.Equal(t, "Shared shell helpers.", summary.Description)
	assert.Equal(t, []string{"LOG_LEVEL", "GIT_DEPTH"}, variableNames(summary))
	assert.Equal(t, []string{".load", ".logging", ".git"}, jobNames(summary))
	assert.Equal(t, "secret", token)
	// Descriptions of jobs from nested includes come through too.
	assert.Equal(t, "Logging helpers.", summary.Files[1].Jobs[0].Description)
}

func TestSummarizeComponentThroughServerFQDN(t *testing.T) {
	// A component living in the helpers project itself, which includes the project's shared
	// helpers - so both the component and its nested `project:` include hit the same fake server.
	files := map[string]string{
		"templates/update-docs.yml": `spec:
  inputs:
    stage:
      default: test
---
include:
  - project: infra/jobs/helpers
    ref: main
    file: .gitlab-ci/include.yml
update-docs:
  script: [echo]
`,
	}
	for k, v := range helpersProject {
		files[k] = v
	}
	var token string
	server := fakeGitLab(t, files, &token)
	defer server.Close()

	resolver := NewIncludeResolver(server.URL, "", "", t.TempDir())
	summary := resolver.Summarize(IncludeItem{Kind: "component", Location: "$CI_SERVER_FQDN/infra/jobs/helpers/update-docs", Ref: "main"})

	require.NotNil(t, summary)
	assert.Equal(t, []string{"LOG_LEVEL", "GIT_DEPTH"}, variableNames(summary))
	assert.Equal(t, []string{"update-docs", ".load", ".logging", ".git"}, jobNames(summary))
}

func TestSummarizeLocalIncludeFromCheckout(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "ci"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "ci", "lint.yml"), []byte("variables:\n  LINT_IMAGE: golang\nlint:\n  script: [make lint]\n"), 0o644))

	resolver := NewIncludeResolver("", "", "", root)
	summary := resolver.Summarize(IncludeItem{Kind: "local", Location: "/ci/lint.yml"})

	require.NotNil(t, summary)
	assert.Equal(t, []string{"LINT_IMAGE"}, variableNames(summary))
	assert.Equal(t, []string{"lint"}, jobNames(summary))
}

func TestSummarizeUnresolvableIncludes(t *testing.T) {
	resolver := NewIncludeResolver("", "", "", t.TempDir())

	assert.Nil(t, resolver.Summarize(IncludeItem{Kind: "project", Location: "infra/jobs/helpers", File: "x.yml"}), "no server URL")
	assert.Nil(t, resolver.Summarize(IncludeItem{Kind: "component", Location: "$CI_SERVER_FQDN/a/b/c", Ref: "~latest"}), "~latest ref")
	assert.Nil(t, resolver.Summarize(IncludeItem{Kind: "template", Location: "Security/SAST.gitlab-ci.yml"}))
	assert.Nil(t, resolver.Summarize(IncludeItem{Kind: "local", Location: "missing.yml"}))
}

func TestSummarizeOwnComponentFromCheckout(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "build-image", "templates", "build-image.yml")
	require.NoError(t, os.MkdirAll(filepath.Dir(file), 0o755))
	require.NoError(t, os.WriteFile(file, []byte("spec:\n  inputs: {}\n---\nvariables:\n  CI_DEBUG:\n    value: 'true'\n    description: Debug logging\nbuild-image:\n  script: [echo]\n"), 0o644))

	// No server URL: the component must resolve from disk alone.
	resolver := NewIncludeResolver("", "", "", root)
	resolver.LocalProjects = []string{"m13tLabs/glab-docs"}
	resolver.LocalComponents = map[string]string{"build-image": file}

	for _, location := range []string{
		"$CI_SERVER_FQDN/m13tlabs/glab-docs/build-image", // project path matched case-insensitively
		"$CI_SERVER_FQDN/$CI_PROJECT_PATH/build-image",
		"gitlab.com/m13tLabs/glab-docs/build-image",
	} {
		summary := resolver.Summarize(IncludeItem{Kind: "component", Location: location, Ref: "$CI_COMMIT_SHA"})
		require.NotNil(t, summary, location)
		assert.Equal(t, []string{"CI_DEBUG"}, variableNames(summary), location)
		assert.Equal(t, []string{"build-image"}, jobNames(summary), location)
	}

	assert.Nil(t, resolver.Summarize(IncludeItem{Kind: "component", Location: "$CI_SERVER_FQDN/other/project/build-image", Ref: "~latest"}), "other project")
}

func TestSummarizeOwnProjectIncludeRespectsGitRef(t *testing.T) {
	root := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)...)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
	}
	write := func(file, contents string) {
		t.Helper()
		require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(root, file)), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(root, file), []byte(contents), 0o644))
	}

	// develop: include.yml pulls in a nested local file.
	git("init", "-q", "-b", "develop")
	write("ci/include.yml", "include:\n  - local: ci/nested.yml\nvariables:\n  ON_DEVELOP: 'true'\n")
	write("ci/nested.yml", "# -- Nested on develop.\n.nested_develop:\n  script: [echo]\n")
	git("add", ".")
	git("commit", "-q", "-m", "develop")

	// feature (checked out): both files changed, include.yml only in the working tree.
	git("checkout", "-q", "-b", "feature")
	write("ci/nested.yml", ".nested_feature:\n  script: [echo]\n")
	git("commit", "-q", "-am", "feature")
	write("ci/include.yml", "include:\n  - local: ci/nested.yml\nvariables:\n  UNCOMMITTED: 'true'\n")

	resolver := NewIncludeResolver("", "", "", root)
	resolver.LocalProjects = []string{"m13tLabs/glab-docs"}
	resolver.CurrentRefs = []string{"HEAD", "feature"}
	include := func(ref string) IncludeItem {
		return IncludeItem{Kind: "project", Location: "m13tlabs/glab-docs", File: "/ci/include.yml", Ref: ref}
	}

	t.Run("other ref is read from git, nested includes at the same ref", func(t *testing.T) {
		summary := resolver.Summarize(include("develop"))
		require.NotNil(t, summary)
		assert.Equal(t, []string{"ON_DEVELOP"}, variableNames(summary))
		assert.Equal(t, []string{".nested_develop"}, jobNames(summary))
		assert.Equal(t, "Nested on develop.", summary.Files[1].Jobs[0].Description)
	})

	t.Run("checked-out ref is read from the working tree", func(t *testing.T) {
		summary := resolver.Summarize(include("feature"))
		require.NotNil(t, summary)
		assert.Equal(t, []string{"UNCOMMITTED"}, variableNames(summary))
		assert.Equal(t, []string{".nested_feature"}, jobNames(summary))
	})

	t.Run("unknown ref without a server URL can't be resolved", func(t *testing.T) {
		assert.Nil(t, resolver.Summarize(include("no-such-branch")))
	})

	t.Run("unknown ref falls back to the API", func(t *testing.T) {
		var token string
		server := fakeGitLab(t, map[string]string{"ci/include.yml": "variables:\n  FROM_API: 'true'\n"}, &token)
		defer server.Close()
		apiResolver := NewIncludeResolver(server.URL, "", "", root)
		apiResolver.LocalProjects = []string{"infra/jobs/helpers"}
		summary := apiResolver.Summarize(IncludeItem{Kind: "project", Location: "infra/jobs/helpers", File: "ci/include.yml", Ref: "main"})
		require.NotNil(t, summary)
		assert.Equal(t, []string{"FROM_API"}, variableNames(summary))
	})
}

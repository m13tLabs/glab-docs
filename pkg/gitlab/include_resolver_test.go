package gitlab

import (
	"net/http"
	"net/http/httptest"
	"os"
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

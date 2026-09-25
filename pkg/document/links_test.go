package document

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"

	"github.com/m13tLabs/glab-docs/pkg/gitlab"
)

func infoAt(dir, name string) gitlab.ComponentDocumentationInfo {
	return gitlab.ComponentDocumentationInfo{
		Name:               name,
		ComponentDirectory: dir,
		SourceFile:         dir + "/" + name + ".yml",
	}
}

func TestGroupComponentsByOutput(t *testing.T) {
	infoByFile := map[string]gitlab.ComponentDocumentationInfo{
		"templates/build.yml":         infoAt("templates", "build"),
		"templates/lint.yml":          infoAt("templates", "lint"),
		"templates/scan/template.yml": infoAt("templates/scan", "scan"),
	}

	groups := GroupComponentsByOutput(infoByFile, "README.md")

	assert.ElementsMatch(t, []string{"templates/build.yml", "templates/lint.yml"}, groups["templates/README.md"])
	assert.Equal(t, []string{"templates/scan/template.yml"}, groups["templates/scan/README.md"])
	// Sorted by component name within a group.
	assert.Equal(t, []string{"templates/build.yml", "templates/lint.yml"}, groups["templates/README.md"])
}

func TestBuildComponentLinks(t *testing.T) {
	infoByFile := map[string]gitlab.ComponentDocumentationInfo{
		"templates/build.yml":         infoAt("templates", "build"),
		"templates/lint.yml":          infoAt("templates", "lint"),
		"templates/scan/template.yml": infoAt("templates/scan", "scan"),
	}
	groups := GroupComponentsByOutput(infoByFile, "README.md")

	links := BuildComponentLinks(infoByFile, groups)

	// Combined group: siblings get an anchor, since they share the doc with something else.
	assert.Equal(t, ComponentLink{OutputPath: "templates/README.md", Anchor: "build"}, links["templates/build.yml"])
	assert.Equal(t, ComponentLink{OutputPath: "templates/README.md", Anchor: "lint"}, links["templates/lint.yml"])
	// Standalone component: no anchor needed, it has the whole file to itself.
	assert.Equal(t, ComponentLink{OutputPath: "templates/scan/README.md", Anchor: ""}, links["templates/scan/template.yml"])
}

func TestResolveIncludeLink(t *testing.T) {
	links := map[string]ComponentLink{
		"templates/build.yml":         {OutputPath: "templates/README.md", Anchor: "build"},
		"templates/lint.yml":          {OutputPath: "templates/README.md", Anchor: "lint"},
		"templates/scan/template.yml": {OutputPath: "templates/scan/README.md", Anchor: ""},
	}

	t.Run("same combined doc links to an anchor", func(t *testing.T) {
		link := resolveIncludeLink("templates/build.yml", gitlab.IncludeItem{Kind: "local", Location: "templates/lint.yml"}, links)
		assert.Equal(t, "#lint", link)
	})

	t.Run("different doc links relatively, with anchor when the target is combined", func(t *testing.T) {
		link := resolveIncludeLink("templates/scan/template.yml", gitlab.IncludeItem{Kind: "local", Location: "templates/build.yml"}, links)
		assert.Equal(t, "../README.md#build", link)
	})

	t.Run("different doc with a standalone target has no anchor", func(t *testing.T) {
		otherLinks := map[string]ComponentLink{
			"templates/build.yml":         {OutputPath: "templates/README.md", Anchor: ""},
			"templates/scan/template.yml": {OutputPath: "templates/scan/README.md", Anchor: ""},
		}
		link := resolveIncludeLink("templates/scan/template.yml", gitlab.IncludeItem{Kind: "local", Location: "templates/build.yml"}, otherLinks)
		assert.Equal(t, "../README.md", link)
	})

	t.Run("leading slash on the local path is tolerated", func(t *testing.T) {
		link := resolveIncludeLink("templates/build.yml", gitlab.IncludeItem{Kind: "local", Location: "/templates/lint.yml"}, links)
		assert.Equal(t, "#lint", link)
	})

	t.Run("non-local include kinds are left alone", func(t *testing.T) {
		link := resolveIncludeLink("templates/build.yml", gitlab.IncludeItem{Kind: "component", Location: "gitlab.com/group/proj/lint@1.0.0"}, links)
		assert.Equal(t, "", link)
	})

	t.Run("local include of an undiscovered file is left alone", func(t *testing.T) {
		link := resolveIncludeLink("templates/build.yml", gitlab.IncludeItem{Kind: "local", Location: "templates/unknown.yml"}, links)
		assert.Equal(t, "", link)
	})

	t.Run("self-include is left alone", func(t *testing.T) {
		link := resolveIncludeLink("templates/build.yml", gitlab.IncludeItem{Kind: "local", Location: "templates/build.yml"}, links)
		assert.Equal(t, "", link)
	})
}

func TestMarkdownAnchor(t *testing.T) {
	assert.Equal(t, "build-image", markdownAnchor("Build Image"))
	assert.Equal(t, "scan_security-fast", markdownAnchor("scan_security (fast)!"))
}

func TestResolveComponentLocation(t *testing.T) {
	t.Run("no server url configured leaves the address as-is, unlinked", func(t *testing.T) {
		location, link := resolveComponentLocation(gitlab.IncludeItem{
			Kind:     "component",
			Location: "$CI_SERVER_FQDN/infra/jobs/gitlab-components/glab-docs/update-docs",
			Ref:      "v0.5.0",
		}, "")
		assert.Equal(t, "$CI_SERVER_FQDN/infra/jobs/gitlab-components/glab-docs/update-docs", location)
		assert.Equal(t, "", link)
	})

	t.Run("address without the $CI_SERVER_FQDN placeholder is left alone even with a server url", func(t *testing.T) {
		location, link := resolveComponentLocation(gitlab.IncludeItem{
			Kind:     "component",
			Location: "gitlab.com/my-group/my-project/my-component",
			Ref:      "1.0.0",
		}, "https://gitlab.example.com")
		assert.Equal(t, "gitlab.com/my-group/my-project/my-component", location)
		assert.Equal(t, "", link)
	})

	t.Run("resolves to the component's source file at its ref, not the raw address", func(t *testing.T) {
		// Regression test: linking to the raw address (.../glab-docs/update-docs) 404s, since
		// that's not a real browsable path - only .../glab-docs (the project) is, and the
		// component's file lives under its templates/ directory.
		location, link := resolveComponentLocation(gitlab.IncludeItem{
			Kind:     "component",
			Location: "$CI_SERVER_FQDN/infra/jobs/gitlab-components/glab-docs/update-docs",
			Ref:      "v0.5.0",
		}, "https://gitlab.example.com")
		assert.Equal(t, "infra/jobs/gitlab-components/glab-docs/update-docs", location)
		assert.Equal(t, "https://gitlab.example.com/infra/jobs/gitlab-components/glab-docs/-/blob/v0.5.0/templates/update-docs.yml", link)
	})

	t.Run("falls back to HEAD when the include has no ref", func(t *testing.T) {
		_, link := resolveComponentLocation(gitlab.IncludeItem{
			Kind:     "component",
			Location: "$CI_SERVER_FQDN/infra/jobs/gitlab-components/glab-docs/update-docs",
		}, "https://gitlab.example.com")
		assert.Equal(t, "https://gitlab.example.com/infra/jobs/gitlab-components/glab-docs/-/blob/HEAD/templates/update-docs.yml", link)
	})

	t.Run("only the last path segment is treated as the component name, however deep the project group nesting", func(t *testing.T) {
		_, link := resolveComponentLocation(gitlab.IncludeItem{
			Kind:     "component",
			Location: "$CI_SERVER_FQDN/a/b/c/d/component-name",
			Ref:      "1.0.0",
		}, "https://gitlab.example.com")
		assert.Equal(t, "https://gitlab.example.com/a/b/c/d/-/blob/1.0.0/templates/component-name.yml", link)
	})

	t.Run("single-segment address can't be split into a project and component - linked as-is", func(t *testing.T) {
		location, link := resolveComponentLocation(gitlab.IncludeItem{
			Kind:     "component",
			Location: "$CI_SERVER_FQDN/only-one-segment",
			Ref:      "1.0.0",
		}, "https://gitlab.example.com")
		assert.Equal(t, "only-one-segment", location)
		assert.Equal(t, "https://gitlab.example.com/only-one-segment", link)
	})
}

func TestResolveProjectLocation(t *testing.T) {
	t.Run("no server url configured leaves the location as-is, unlinked", func(t *testing.T) {
		location, link := resolveProjectLocation(gitlab.IncludeItem{
			Kind:     "project",
			Location: "infra/jobs/gitlab-components/helpers",
			File:     "gitlab-ci/include.yml",
			Ref:      "v0.6.1",
		}, "")
		assert.Equal(t, "infra/jobs/gitlab-components/helpers (file: gitlab-ci/include.yml)", location)
		assert.Equal(t, "", link)
	})

	t.Run("without a file: links to the project root", func(t *testing.T) {
		location, link := resolveProjectLocation(gitlab.IncludeItem{
			Kind:     "project",
			Location: "infra/jobs/gitlab-components/helpers",
			Ref:      "v0.6.1",
		}, "https://gitlab.example.com")
		assert.Equal(t, "infra/jobs/gitlab-components/helpers", location)
		assert.Equal(t, "https://gitlab.example.com/infra/jobs/gitlab-components/helpers", link)
	})

	t.Run("with a file: appends it to the display text and links straight to its blob at ref", func(t *testing.T) {
		location, link := resolveProjectLocation(gitlab.IncludeItem{
			Kind:     "project",
			Location: "infra/jobs/gitlab-components/helpers",
			File:     "gitlab-ci/include.yml",
			Ref:      "v0.6.1",
		}, "https://gitlab.example.com")
		assert.Equal(t, "infra/jobs/gitlab-components/helpers (file: gitlab-ci/include.yml)", location)
		assert.Equal(t, "https://gitlab.example.com/infra/jobs/gitlab-components/helpers/-/blob/v0.6.1/gitlab-ci/include.yml", link)
	})

	t.Run("file: blob link falls back to HEAD when the include has no ref", func(t *testing.T) {
		_, link := resolveProjectLocation(gitlab.IncludeItem{
			Kind:     "project",
			Location: "infra/jobs/gitlab-components/helpers",
			File:     "gitlab-ci/include.yml",
		}, "https://gitlab.example.com")
		assert.Equal(t, "https://gitlab.example.com/infra/jobs/gitlab-components/helpers/-/blob/HEAD/gitlab-ci/include.yml", link)
	})

	t.Run("empty location is left unlinked even with a server url", func(t *testing.T) {
		location, link := resolveProjectLocation(gitlab.IncludeItem{Kind: "project"}, "https://gitlab.example.com")
		assert.Equal(t, "", location)
		assert.Equal(t, "", link)
	})
}

// TestGetIncludeRows exercises getIncludeRows itself, not just the per-kind resolvers it calls -
// this is what actually reads --gitlab-server-url (via viper) and dispatches on item.Kind, so a
// typo'd viper key or a swapped case label wouldn't be caught by testing the resolvers alone.
func TestGetIncludeRows(t *testing.T) {
	setGitlabServerURL := func(t *testing.T, url string) {
		t.Helper()
		viper.Set("gitlab-server-url", url)
		t.Cleanup(func() { viper.Set("gitlab-server-url", "") })
	}

	items := []gitlab.IncludeItem{
		{Kind: "local", Location: "templates/lint.yml"},
		{Kind: "project", Location: "infra/jobs/gitlab-components/helpers", File: "gitlab-ci/include.yml", Ref: "v0.6.1"},
		{Kind: "component", Location: "$CI_SERVER_FQDN/infra/jobs/gitlab-components/glab-docs/update-docs", Ref: "v0.5.0"},
		{Kind: "remote", Location: "https://example.com/some.yml"},
	}
	links := map[string]ComponentLink{
		"templates/build.yml": {OutputPath: "templates/README.md", Anchor: "build"},
		"templates/lint.yml":  {OutputPath: "templates/README.md", Anchor: "lint"},
	}

	t.Run("without a server url, only local includes are linked", func(t *testing.T) {
		setGitlabServerURL(t, "")
		rows := getIncludeRows("templates/build.yml", items, links)

		assert.Equal(t, []includeRow{
			{Kind: "local", Location: "templates/lint.yml", Link: "#lint"},
			{Kind: "project", Location: "infra/jobs/gitlab-components/helpers (file: gitlab-ci/include.yml)", Ref: "v0.6.1"},
			{Kind: "component", Location: "$CI_SERVER_FQDN/infra/jobs/gitlab-components/glab-docs/update-docs", Ref: "v0.5.0"},
			{Kind: "remote", Location: "https://example.com/some.yml"},
		}, rows)
	})

	t.Run("with a server url, project and component includes are also linked", func(t *testing.T) {
		setGitlabServerURL(t, "https://gitlab.example.com")
		rows := getIncludeRows("templates/build.yml", items, links)

		assert.Equal(t, []includeRow{
			{Kind: "local", Location: "templates/lint.yml", Link: "#lint"},
			{
				Kind:     "project",
				Location: "infra/jobs/gitlab-components/helpers (file: gitlab-ci/include.yml)",
				Ref:      "v0.6.1",
				Link:     "https://gitlab.example.com/infra/jobs/gitlab-components/helpers/-/blob/v0.6.1/gitlab-ci/include.yml",
			},
			{
				Kind:     "component",
				Location: "infra/jobs/gitlab-components/glab-docs/update-docs",
				Ref:      "v0.5.0",
				Link:     "https://gitlab.example.com/infra/jobs/gitlab-components/glab-docs/-/blob/v0.5.0/templates/update-docs.yml",
			},
			{Kind: "remote", Location: "https://example.com/some.yml"},
		}, rows)
	})

	t.Run("a trailing slash on the server url doesn't produce a double slash in links", func(t *testing.T) {
		setGitlabServerURL(t, "https://gitlab.example.com/")
		rows := getIncludeRows("templates/build.yml", items, links)

		assert.Equal(t, "https://gitlab.example.com/infra/jobs/gitlab-components/helpers/-/blob/v0.6.1/gitlab-ci/include.yml", rows[1].Link)
		assert.Equal(t, "https://gitlab.example.com/infra/jobs/gitlab-components/glab-docs/-/blob/v0.5.0/templates/update-docs.yml", rows[2].Link)
	})
}

func TestIncludeDescription(t *testing.T) {
	variables := func(src string) *yaml.Node {
		var doc yaml.Node
		if err := yaml.Unmarshal([]byte(src), &doc); err != nil {
			t.Fatal(err)
		}
		return doc.Content[0]
	}

	assert.Equal(t, "", includeDescription(nil))
	assert.Equal(t,
		"Shared shell helpers.<br>**Variables:**<ul><li>`CI_DEBUG` = `true` - Enable debug logging</li><li>`LOG_LEVEL` = `info`</li></ul>"+
			"**Jobs:**<ul><li>`.logging` - Logging helpers</li><li>`.git` - Git helpers</li></ul>",
		includeDescription(&gitlab.IncludeSummary{
			Description: "Shared shell helpers.",
			Files: []gitlab.ComponentDocumentationInfo{
				{
					Variables: variables("CI_DEBUG:\n  value: 'true'\n  description: Enable debug logging\n"),
					Jobs:      []gitlab.Job{{Name: ".logging", Description: "Logging helpers"}},
				},
				{
					// A nested include: its jobs are listed with their descriptions too, and CI_DEBUG /
					// .logging redefined down the chain are listed once, as first seen.
					Variables: variables("LOG_LEVEL: info\nCI_DEBUG: 'false'\n"),
					Jobs:      []gitlab.Job{{Name: ".git", Description: "Git helpers"}, {Name: ".logging", Description: "overridden"}},
				},
			},
		}))
	assert.Equal(t, "**Jobs:**<ul><li>`lint`</li></ul>", includeDescription(&gitlab.IncludeSummary{
		Files: []gitlab.ComponentDocumentationInfo{{Jobs: []gitlab.Job{{Name: "lint"}}}},
	}))
}

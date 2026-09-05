package document

import (
	"testing"

	"github.com/stretchr/testify/assert"

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

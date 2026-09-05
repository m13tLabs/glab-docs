package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"

	"github.com/m13tLabs/glab-docs/pkg/document"
	"github.com/m13tLabs/glab-docs/pkg/gitlab"
)

var _ viper.FlagValueSet = &testFlagSet{}

type testFlagSet map[string]interface{}

func (s testFlagSet) VisitAll(fn func(viper.FlagValue)) {
	for k, v := range s {
		flagVal := &testFlagValue{name: k, value: fmt.Sprintf("%v", v)}
		switch v.(type) {
		case bool:
			flagVal.typ = "bool"
		case []string:
			flagVal.typ = "stringSlice"
		default:
			flagVal.typ = "string"
		}
		fn(flagVal)
	}
}

var _ viper.FlagValue = &testFlagValue{}

type testFlagValue struct {
	name  string
	value string
	typ   string
}

func (v *testFlagValue) HasChanged() bool    { return false }
func (v *testFlagValue) Name() string        { return v.name }
func (v *testFlagValue) ValueString() string { return v.value }
func (v *testFlagValue) ValueType() string   { return v.typ }

func baseFlags(searchRoot string) testFlagSet {
	return testFlagSet{
		"search-root":       searchRoot,
		"search-pattern":    []string{"*.yml"},
		"template-files":    []string{"README.md.gotmpl"},
		"output-file":       "README.md",
		"ignore-file":       ".glabdocsignore",
		"log-level":         "warn",
		"sort-values-order": document.AlphaNumSortOrder,
		"component-prefix":  "",
	}
}

func generateAndRead(t *testing.T, flags testFlagSet, readmePath string) string {
	t.Helper()
	if err := viper.BindFlagValues(flags); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(readmePath) })

	version = "1.2.3"
	glabDocs(nil, nil)

	data, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestSkipsVersionFooter(t *testing.T) {
	flags := baseFlags(filepath.Join("testdata", "skip-version-footer"))
	flags["skip-version-footer"] = true
	doc := generateAndRead(t, flags, filepath.Join("testdata", "skip-version-footer", "README.md"))

	if strings.Contains(doc, "glab-docs v1.2.3") {
		t.Errorf("generated documentation should not contain the version footer, got:\n%s", doc)
	}
}

func TestIncludesVersionFooter(t *testing.T) {
	flags := baseFlags(filepath.Join("testdata", "skip-version-footer"))
	flags["skip-version-footer"] = false
	doc := generateAndRead(t, flags, filepath.Join("testdata", "skip-version-footer", "README.md"))

	if !strings.Contains(doc, "glab-docs v1.2.3") {
		t.Errorf("generated documentation must contain the version footer, got:\n%s", doc)
	}
}

func TestCombinesFlatSiblingsIntoOneReadmeWithLinkedIncludes(t *testing.T) {
	flags := baseFlags(filepath.Join("testdata", "combined"))
	doc := generateAndRead(t, flags, filepath.Join("testdata", "combined", "README.md"))

	// One section per template, each one heading level deeper than a standalone README.
	if strings.Count(doc, "## build\n") != 1 || strings.Count(doc, "## lint\n") != 1 {
		t.Errorf("expected one `## build` and one `## lint` section, got:\n%s", doc)
	}
	// build's `local: lint.yml` include resolves to an in-page anchor, since both templates
	// share this one combined README.
	if !strings.Contains(doc, "[`lint.yml`](#lint)") {
		t.Errorf("expected build's Includes table to link lint.yml to its #lint section, got:\n%s", doc)
	}
	// Only one version footer for the whole combined file, not one per section.
	if strings.Count(doc, "glab-docs v1.2.3") != 1 {
		t.Errorf("expected exactly one version footer, got:\n%s", doc)
	}
}

func TestCombinedTitle(t *testing.T) {
	flags := baseFlags(filepath.Join("testdata", "combined"))
	flags["combined-title"] = "Glab Pipeline Docs"
	doc := generateAndRead(t, flags, filepath.Join("testdata", "combined", "README.md"))

	if !strings.HasPrefix(doc, "# Glab Pipeline Docs\n") {
		t.Errorf("expected the combined README to start with the H1 title, got:\n%s", doc)
	}
}

func TestCombinedTitleOmittedWhenEmpty(t *testing.T) {
	flags := baseFlags(filepath.Join("testdata", "combined"))
	flags["combined-title"] = ""
	doc := generateAndRead(t, flags, filepath.Join("testdata", "combined", "README.md"))

	if strings.HasPrefix(doc, "#\n") || strings.Contains(doc, "\n# \n") {
		t.Errorf("expected no empty H1 when combined-title is empty, got:\n%s", doc)
	}
	if !strings.HasPrefix(doc, "## build\n") {
		t.Errorf("expected the combined README to start directly with the first template's section, got:\n%s", doc)
	}
}

func TestComponentToGenerateExpandsToFullCombinedGroup(t *testing.T) {
	flags := baseFlags(filepath.Join("testdata", "combined"))
	flags["component-to-generate"] = []string{"lint.yml"}
	doc := generateAndRead(t, flags, filepath.Join("testdata", "combined", "README.md"))

	if !strings.Contains(doc, "## build\n") || !strings.Contains(doc, "## lint\n") {
		t.Errorf("requesting only lint.yml should still (re)write the whole shared README, got:\n%s", doc)
	}
}

func TestParsesComponentInfo(t *testing.T) {
	info, err := gitlab.ParseComponentInformation(
		filepath.Join("testdata", "skip-version-footer", "template.yml"),
		gitlab.DocumentationParsingConfig{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if info.Name != "skip-version-footer" {
		t.Errorf("expected component name derived from dir, got %q", info.Name)
	}
}

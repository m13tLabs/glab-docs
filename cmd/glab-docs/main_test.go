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

package util

import (
	"os"
	"strings"
	"testing"
	"time"
)

type fakeFileInfo struct{ dir bool }

func (f fakeFileInfo) Name() string       { return "" }
func (f fakeFileInfo) Size() int64        { return 0 }
func (f fakeFileInfo) Mode() os.FileMode  { return 0 }
func (f fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeFileInfo) IsDir() bool        { return f.dir }
func (f fakeFileInfo) Sys() any           { return nil }

func TestIgnoreRules(t *testing.T) {
	rules, err := ParseIgnoreRules(strings.NewReader(`
# a comment

/.gitlab-ci.yml
node_modules/
*.tmp
`))
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		path string
		dir  bool
		want bool
	}{
		{".gitlab-ci.yml", false, true},           // anchored to root
		{"sub/.gitlab-ci.yml", false, false},      // anchored - not matched deeper
		{"node_modules", true, true},              // dir-only rule, is a dir
		{"node_modules", false, false},            // dir-only rule, not a dir
		{"build.tmp", false, true},                // basename glob
		{"a/b/c.tmp", false, true},                // basename glob, nested
		{"templates/glab-docs.yml", false, false}, // unmatched
	}
	for _, c := range cases {
		got := rules.Ignore(c.path, fakeFileInfo{dir: c.dir})
		if got != c.want {
			t.Errorf("Ignore(%q, dir=%v) = %v, want %v", c.path, c.dir, got, c.want)
		}
	}

	if EmptyIgnoreRules().Ignore("anything", fakeFileInfo{}) {
		t.Error("empty ruleset should ignore nothing")
	}
	if _, err := ParseIgnoreRules(strings.NewReader("a/**/b")); err == nil {
		t.Error("expected ** to be rejected")
	}
}

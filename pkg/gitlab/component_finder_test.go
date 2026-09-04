package gitlab_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"github.com/m13tLabs/glab-docs/pkg/gitlab"
)

func TestFindComponentFiles(t *testing.T) {
	root := t.TempDir()
	write := func(rel string) {
		p := filepath.Join(root, rel)
		assert.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		assert.NoError(t, os.WriteFile(p, []byte("spec:\n  inputs: {}\n"), 0o644))
	}
	write("templates/build.yml")
	write("templates/scan/template.yml")
	write("comp/templates/deploy.yml")
	write(".gitlab-ci.yml")
	write("nested/service.gitlab-ci.yml")
	write("docs/notes.yml") // must NOT match

	viper.Reset()
	defer viper.Reset()

	found, err := gitlab.FindComponentFiles(root)
	assert.NoError(t, err)
	sort.Strings(found)
	assert.Equal(t, []string{
		".gitlab-ci.yml",
		"comp/templates/deploy.yml",
		"nested/service.gitlab-ci.yml",
		"templates/build.yml",
		"templates/scan/template.yml",
	}, found)

	// Pointing the search root straight at a templates dir still matches templates/*.yml.
	found, err = gitlab.FindComponentFiles(filepath.Join(root, "templates"))
	assert.NoError(t, err)
	sort.Strings(found)
	assert.Equal(t, []string{"build.yml", "scan/template.yml"}, found)
}

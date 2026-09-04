package gitlab

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gobwas/glob"
	"github.com/m13tLabs/glab-docs/pkg/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// DefaultSearchPatterns are the globs used to locate documentable GitLab CI YAML files when the
// user does not override --search-pattern. They cover CI/CD component files laid out per the
// GitLab component spec (templates/<name>.yml or templates/<name>/template.yml) as well as plain
// pipeline files.
var DefaultSearchPatterns = []string{
	"templates/*.yml",
	"templates/*.yaml",
	"templates/*/template.yml",
	"templates/*/template.yaml",
	"*.gitlab-ci.yml",
	"*.gitlab-ci.yaml",
	".gitlab-ci.yml",
	".gitlab-ci.yaml",
}

func compileSearchPatterns(patterns []string) []glob.Glob {
	compiled := make([]glob.Glob, 0, len(patterns))
	for _, pattern := range patterns {
		g, err := glob.Compile(pattern, '/')
		if err != nil {
			log.Warnf("Ignoring invalid search pattern %q: %s", pattern, err)
			continue
		}
		compiled = append(compiled, g)
	}
	return compiled
}

// matchesAnyPattern reports whether relPath, or any of its trailing path segments, matches one of
// the compiled patterns. rootName (the base name of the search root) is prepended first, so that
// pointing --search-root straight at a "templates" directory still lets a "templates/*.yml"
// pattern match, and checking suffixes lets it match "some-component/templates/foo.yml" too.
func matchesAnyPattern(relPath, rootName string, matchers []glob.Glob) bool {
	segments := strings.Split(relPath, "/")
	if rootName != "" && rootName != "." && rootName != "/" {
		segments = append([]string{rootName}, segments...)
	}
	for start := range segments {
		candidate := strings.Join(segments[start:], "/")
		for _, matcher := range matchers {
			if matcher.Match(candidate) {
				return true
			}
		}
	}
	return false
}

// FindComponentFiles walks componentSearchRoot and returns the paths (relative to the root) of
// every YAML file that matches one of the configured search patterns and is not excluded by the
// ignore file.
func FindComponentFiles(componentSearchRoot string) ([]string, error) {
	ignoreFilename := viper.GetString("ignore-file")
	ignoreContext := util.NewIgnoreContext(ignoreFilename)

	patterns := viper.GetStringSlice("search-pattern")
	if len(patterns) == 0 {
		patterns = DefaultSearchPatterns
	}
	matchers := compileSearchPatterns(patterns)

	rootName := filepath.Base(componentSearchRoot)
	componentFiles := make([]string, 0)

	err := filepath.Walk(componentSearchRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		absolutePath, _ := filepath.Abs(path)

		if info.IsDir() {
			if ignoreContext.ShouldIgnore(absolutePath, info) {
				log.Debugf("Ignoring directory %s", path)
				return filepath.SkipDir
			}
			return nil
		}

		if ignoreContext.ShouldIgnore(absolutePath, info) {
			log.Debugf("Ignoring file %s", path)
			return nil
		}

		relativePath, err := filepath.Rel(componentSearchRoot, path)
		if err != nil {
			return err
		}

		matchPath := filepath.ToSlash(relativePath)
		if matchesAnyPattern(matchPath, rootName, matchers) {
			componentFiles = append(componentFiles, relativePath)
		}

		return nil
	})

	return componentFiles, err
}

package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/m13tLabs/glab-docs/pkg/document"
	"github.com/m13tLabs/glab-docs/pkg/gitlab"
	"github.com/m13tLabs/glab-docs/pkg/util"
)

// parallelProcessIterable runs visitFn on each element of the iterable (slice or map key) using
// parallelism worker goroutines.
func parallelProcessIterable(iterable interface{}, parallelism int, visitFn func(elem interface{})) {
	workChan := make(chan interface{})

	wg := &sync.WaitGroup{}
	wg.Add(parallelism)

	for i := 0; i < parallelism; i++ {
		go func() {
			defer wg.Done()
			for elem := range workChan {
				visitFn(elem)
			}
		}()
	}

	iterableValue := reflect.ValueOf(iterable)

	if iterableValue.Kind() == reflect.Map {
		for _, key := range iterableValue.MapKeys() {
			workChan <- key.Interface()
		}
	} else {
		sliceLen := iterableValue.Len()
		for i := 0; i < sliceLen; i++ {
			workChan <- iterableValue.Index(i).Interface()
		}
	}

	close(workChan)
	wg.Wait()
}

func getDocumentationParsingConfigFromArgs() (gitlab.DocumentationParsingConfig, error) {
	var regexps []*regexp.Regexp
	for _, item := range viper.GetStringSlice("documentation-strict-ignore-absent-regex") {
		regex, err := regexp.Compile(item)
		if err != nil {
			return gitlab.DocumentationParsingConfig{}, err
		}
		regexps = append(regexps, regex)
	}
	return gitlab.DocumentationParsingConfig{
		StrictMode:                 viper.GetBool("documentation-strict-mode"),
		AllowedMissingValuePaths:   viper.GetStringSlice("documentation-strict-ignore-absent"),
		AllowedMissingValueRegexps: regexps,
	}, nil
}

func readDocumentationInfoByComponentFile(componentSearchRoot string, parallelism int) (map[string]gitlab.ComponentDocumentationInfo, error) {
	var fullComponentSearchRoot string

	if path.IsAbs(componentSearchRoot) {
		fullComponentSearchRoot = componentSearchRoot
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("error getting working directory: %w", err)
		}
		fullComponentSearchRoot = filepath.Join(cwd, componentSearchRoot)
	}

	componentFiles, err := gitlab.FindComponentFiles(fullComponentSearchRoot)
	if err != nil {
		return nil, fmt.Errorf("error finding component files: %w", err)
	}

	log.Infof("Found component files [%s]", strings.Join(componentFiles, ", "))

	parsingConfig, err := getDocumentationParsingConfigFromArgs()
	if err != nil {
		return nil, fmt.Errorf("error parsing the linting config: %w", err)
	}

	infoByFile := make(map[string]gitlab.ComponentDocumentationInfo, len(componentFiles))
	mu := &sync.Mutex{}

	parallelProcessIterable(componentFiles, parallelism, func(elem interface{}) {
		relFile := elem.(string)
		info, err := gitlab.ParseComponentInformation(filepath.Join(componentSearchRoot, relFile), parsingConfig)
		if err != nil {
			log.Warnf("Error parsing information for %s, skipping: %s", relFile, err)
			return
		}
		mu.Lock()
		infoByFile[relFile] = info
		mu.Unlock()
	})

	return infoByFile, nil
}

func getComponentsToGenerate(infoByFile map[string]gitlab.ComponentDocumentationInfo) map[string]gitlab.ComponentDocumentationInfo {
	requested := viper.GetStringSlice("component-to-generate")
	if len(requested) == 0 {
		return infoByFile
	}

	toGenerate := make(map[string]gitlab.ComponentDocumentationInfo, len(requested))
	skipped := false
	for _, file := range requested {
		if info, ok := infoByFile[file]; ok {
			toGenerate[file] = info
		} else {
			log.Warnf("Couldn't find documentation info for <%s> - skipping", file)
			skipped = true
		}
	}
	if skipped {
		possible := make([]string, 0, len(infoByFile))
		for file := range infoByFile {
			possible = append(possible, file)
		}
		log.Warnf("Some files listed in `component-to-generate` weren't found. Available: [%s]", strings.Join(possible, ", "))
	}
	return toGenerate
}

func resolveComponentPrefix(info gitlab.ComponentDocumentationInfo) string {
	prefix := viper.GetString("component-prefix")
	if prefix != "" {
		return fmt.Sprintf("%s/%s@<version>", strings.TrimRight(prefix, "/"), info.Name)
	}
	return fmt.Sprintf("$CI_SERVER_FQDN/%s/%s@<version>", resolveProjectPath(), info.Name)
}

// resolveProjectPath finds this repository's own group/project path, for the fallback usage
// snippet address - $CI_PROJECT_PATH when running as a real GitLab CI job, otherwise the local
// git remote (so a local `glab-docs` run, e.g. to preview docs before committing per the `check`
// mode's own advice, produces the same address CI would). A literal "<path-to-project>"
// placeholder is used when neither is available.
func resolveProjectPath() string {
	if projectPath := os.Getenv("CI_PROJECT_PATH"); projectPath != "" {
		return projectPath
	}
	if projectPath, err := util.FindGitProjectPath(); err == nil && projectPath != "" {
		return projectPath
	}
	return "<path-to-project>"
}

// expandToFullOutputGroups grows toGenerate so that, whenever `--component-to-generate` selects
// only some of several components that share an output file (e.g. one of several flat
// `templates/*.yml` siblings), the whole group is (re)written together rather than clobbering the
// shared file with a partial combined doc.
func expandToFullOutputGroups(
	toGenerate map[string]gitlab.ComponentDocumentationInfo,
	infoByFile map[string]gitlab.ComponentDocumentationInfo,
	allGroups map[string][]string,
) map[string]gitlab.ComponentDocumentationInfo {
	expanded := make(map[string]gitlab.ComponentDocumentationInfo, len(toGenerate))
	for relFile, info := range toGenerate {
		expanded[relFile] = info
	}

	for _, group := range allGroups {
		if len(group) < 2 {
			continue
		}
		anyRequested := false
		for _, relFile := range group {
			if _, ok := toGenerate[relFile]; ok {
				anyRequested = true
				break
			}
		}
		if !anyRequested {
			continue
		}
		for _, relFile := range group {
			expanded[relFile] = infoByFile[relFile]
		}
	}

	return expanded
}

func writeDocumentation(componentSearchRoot string, infoByFile map[string]gitlab.ComponentDocumentationInfo, dryRun bool, parallelism int) {
	templateFiles := viper.GetStringSlice("template-files")
	skipVersionFooter := viper.GetBool("skip-version-footer")
	outputFileName := viper.GetString("output-file")
	combinedTitle := viper.GetString("combined-title")

	log.Debugf("Rendering from optional template files [%s]", strings.Join(templateFiles, ", "))

	allGroups := document.GroupComponentsByOutput(infoByFile, outputFileName)
	links := document.BuildComponentLinks(infoByFile, allGroups)

	toGenerate := getComponentsToGenerate(infoByFile)
	toGenerate = expandToFullOutputGroups(toGenerate, infoByFile, allGroups)

	writeGroups := document.GroupComponentsByOutput(toGenerate, outputFileName)
	outputPaths := make([]string, 0, len(writeGroups))
	for outputPath := range writeGroups {
		outputPaths = append(outputPaths, outputPath)
	}

	parallelProcessIterable(outputPaths, parallelism, func(elem interface{}) {
		outputPath := elem.(string)
		relFiles := writeGroups[outputPath]

		if len(relFiles) == 1 {
			info := infoByFile[relFiles[0]]
			document.PrintDocumentation(
				info,
				relFiles[0],
				componentSearchRoot,
				templateFiles,
				dryRun,
				version,
				resolveComponentPrefix(info),
				skipVersionFooter,
				links,
			)
			return
		}

		document.PrintCombinedDocumentation(
			relFiles,
			infoByFile,
			filepath.Dir(outputPath),
			componentSearchRoot,
			templateFiles,
			dryRun,
			version,
			resolveComponentPrefix,
			skipVersionFooter,
			links,
			combinedTitle,
		)
	})
}

func glabDocs(_ *cobra.Command, _ []string) {
	initializeCli()

	componentSearchRoot := viper.GetString("search-root")
	dryRun := viper.GetBool("dry-run")

	parallelism := runtime.NumCPU() * 2
	if dryRun {
		parallelism = 1
	}

	infoByFile, err := readDocumentationInfoByComponentFile(componentSearchRoot, parallelism)
	if err != nil {
		log.Fatal(err)
	}

	if viper.GetBool("include-details") {
		resolver := gitlab.NewIncludeResolver(
			viper.GetString("gitlab-server-url"),
			viper.GetString("gitlab-token"),
			os.Getenv("CI_JOB_TOKEN"),
			resolveLocalIncludeRoot(componentSearchRoot),
		)
		resolver.LocalProjects = resolveLocalProjectPaths()
		resolver.LocalComponents = localComponentFiles(componentSearchRoot, infoByFile)
		resolver.ResolveAll(infoByFile)
	}

	writeDocumentation(componentSearchRoot, infoByFile, dryRun, parallelism)
}

// resolveLocalProjectPaths lists every path the documented repository may be addressed by in a
// `component:` include of one of its own components: $CI_PROJECT_PATH, the project path of
// --component-prefix (host stripped) and the local git remote's.
func resolveLocalProjectPaths() []string {
	paths := make([]string, 0, 3)
	if projectPath := os.Getenv("CI_PROJECT_PATH"); projectPath != "" {
		paths = append(paths, projectPath)
	}
	if _, projectPath, found := strings.Cut(strings.Trim(viper.GetString("component-prefix"), "/"), "/"); found {
		paths = append(paths, projectPath)
	}
	if projectPath, err := util.FindGitProjectPath(); err == nil && projectPath != "" {
		paths = append(paths, projectPath)
	}
	return paths
}

// localComponentFiles maps the name of every discovered file that lives under a `templates/`
// directory - i.e. is a CI/CD component, not a plain pipeline - to its absolute path, for
// `component:` includes of the repository's own components to resolve against.
func localComponentFiles(componentSearchRoot string, infoByFile map[string]gitlab.ComponentDocumentationInfo) map[string]string {
	absRoot, err := filepath.Abs(componentSearchRoot)
	if err != nil {
		absRoot = componentSearchRoot
	}
	components := make(map[string]string)
	for relFile, info := range infoByFile {
		if !slices.Contains(strings.Split(filepath.ToSlash(filepath.Join(filepath.Base(absRoot), relFile)), "/"), "templates") {
			continue
		}
		components[info.Name] = filepath.Join(absRoot, relFile)
	}
	return components
}

// resolveLocalIncludeRoot finds the directory the documented repository's `local:` includes are
// relative to - its root: $CI_PROJECT_DIR in a GitLab CI job, otherwise the git toplevel, falling
// back to the search root when neither is available (the runtime image ships without git).
func resolveLocalIncludeRoot(componentSearchRoot string) string {
	if projectDir := os.Getenv("CI_PROJECT_DIR"); projectDir != "" {
		return projectDir
	}
	if root, err := util.FindGitRepositoryRoot(); err == nil && root != "" {
		return root
	}
	return componentSearchRoot
}

func main() {
	command, err := newGlabDocsCommand(glabDocs)
	if err != nil {
		log.Errorf("Failed to create the CLI commander: %s", err)
		os.Exit(1)
	}

	if err := command.Execute(); err != nil {
		log.Errorf("Failed to start the CLI: %s", err)
		os.Exit(1)
	}
}

package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/m13tLabs/glab-docs/pkg/document"
	"github.com/m13tLabs/glab-docs/pkg/gitlab"
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
	return fmt.Sprintf("$CI_SERVER_FQDN/<path-to-project>/%s@<version>", info.Name)
}

func writeDocumentation(componentSearchRoot string, infoByFile map[string]gitlab.ComponentDocumentationInfo, dryRun bool, parallelism int) {
	templateFiles := viper.GetStringSlice("template-files")
	skipVersionFooter := viper.GetBool("skip-version-footer")

	log.Debugf("Rendering from optional template files [%s]", strings.Join(templateFiles, ", "))

	toGenerate := getComponentsToGenerate(infoByFile)

	parallelProcessIterable(toGenerate, parallelism, func(elem interface{}) {
		info := infoByFile[elem.(string)]
		document.PrintDocumentation(
			info,
			componentSearchRoot,
			templateFiles,
			dryRun,
			version,
			resolveComponentPrefix(info),
			skipVersionFooter,
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

	writeDocumentation(componentSearchRoot, infoByFile, dryRun, parallelism)
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

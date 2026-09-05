package document

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/m13tLabs/glab-docs/pkg/gitlab"
)

func getOutputFile(componentDirectory string, dryRun bool) (*os.File, error) {
	if dryRun {
		return os.Stdout, nil
	}

	outputFile := viper.GetString("output-file")
	return os.Create(filepath.Join(componentDirectory, outputFile))
}

// renderComponentMarkdown renders a single component/pipeline file's documentation to a string,
// without writing it anywhere - the shared step behind both PrintDocumentation (one component per
// output file) and PrintCombinedDocumentation (several components sharing one output file).
func renderComponentMarkdown(
	info gitlab.ComponentDocumentationInfo,
	relFile string,
	componentSearchRoot string,
	templateFiles []string,
	glabDocsVersion string,
	componentPrefix string,
	skipVersionFooter bool,
	links map[string]ComponentLink,
	headingLevel int,
) (string, error) {
	documentationTemplate, err := newComponentDocumentationTemplate(info, componentSearchRoot, templateFiles)
	if err != nil {
		return "", fmt.Errorf("generating gotemplates for %s: %w", info.SourceFile, err)
	}

	templateDataObject, err := getComponentTemplateData(info, relFile, glabDocsVersion, componentPrefix, skipVersionFooter, links, headingLevel)
	if err != nil {
		return "", fmt.Errorf("generating template data for %s: %w", info.SourceFile, err)
	}

	var output bytes.Buffer
	if err := documentationTemplate.Execute(&output, templateDataObject); err != nil {
		return "", fmt.Errorf("generating documentation for %s: %w", info.SourceFile, err)
	}
	return output.String(), nil
}

// PrintDocumentation renders the README for a single GitLab CI component/pipeline file that has
// its output file (componentDirectory/output-file) to itself.
func PrintDocumentation(
	info gitlab.ComponentDocumentationInfo,
	relFile string,
	componentSearchRoot string,
	templateFiles []string,
	dryRun bool,
	glabDocsVersion string,
	componentPrefix string,
	skipVersionFooter bool,
	links map[string]ComponentLink,
) {
	log.Infof("Generating README documentation for %s", info.SourceFile)

	rendered, err := renderComponentMarkdown(info, relFile, componentSearchRoot, templateFiles, glabDocsVersion, componentPrefix, skipVersionFooter, links, 1)
	if err != nil {
		log.Warnf("%s", err)
		return
	}

	outputFile, err := getOutputFile(info.ComponentDirectory, dryRun)
	if err != nil {
		log.Warnf("Could not open README file for %s, skipping", info.SourceFile)
		return
	}
	if !dryRun {
		defer outputFile.Close()
	}

	output := applyMarkDownFormat(*bytes.NewBufferString(rendered))
	if _, err := output.WriteTo(outputFile); err != nil {
		log.Warnf("Error writing documentation file for %s: %s", info.SourceFile, err)
	}
}

// PrintCombinedDocumentation renders one README covering several component/pipeline files that
// all resolve to the same output file (e.g. sibling `templates/*.yml` files, which share the
// "templates" directory) - one section per template, in place of each clobbering the others'
// output in turn. An H1 title (skipped when empty) sits above them, each member keeps its own
// Usage/Inputs/Variables/Jobs/Includes sub-sections nested one heading level deeper than a
// standalone README, and only one version footer is written for the whole file.
func PrintCombinedDocumentation(
	relFiles []string,
	infoByFile map[string]gitlab.ComponentDocumentationInfo,
	componentDirectory string,
	componentSearchRoot string,
	templateFiles []string,
	dryRun bool,
	glabDocsVersion string,
	componentPrefixFor func(gitlab.ComponentDocumentationInfo) string,
	skipVersionFooter bool,
	links map[string]ComponentLink,
	title string,
) {
	log.Infof("Generating combined README documentation for [%s] in %s", strings.Join(relFiles, ", "), componentDirectory)

	var combined bytes.Buffer
	if title != "" {
		combined.WriteString("# ")
		combined.WriteString(title)
		combined.WriteString("\n\n")
	}
	for i, relFile := range relFiles {
		info := infoByFile[relFile]

		rendered, err := renderComponentMarkdown(info, relFile, componentSearchRoot, templateFiles, glabDocsVersion, componentPrefixFor(info), true, links, 2)
		if err != nil {
			log.Warnf("%s", err)
			continue
		}

		if i > 0 {
			combined.WriteString("\n\n")
		}
		combined.WriteString(rendered)
	}

	if !skipVersionFooter {
		footer, err := renderVersionFooter(glabDocsVersion)
		if err != nil {
			log.Warnf("Error rendering version footer for %s: %s", componentDirectory, err)
		} else {
			combined.WriteString("\n\n")
			combined.WriteString(footer)
		}
	}

	outputFile, err := getOutputFile(componentDirectory, dryRun)
	if err != nil {
		log.Warnf("Could not open README file for %s, skipping", componentDirectory)
		return
	}
	if !dryRun {
		defer outputFile.Close()
	}

	output := applyMarkDownFormat(combined)
	if _, err := output.WriteTo(outputFile); err != nil {
		log.Warnf("Error writing documentation file for %s: %s", componentDirectory, err)
	}
}

func applyMarkDownFormat(output bytes.Buffer) bytes.Buffer {
	outputString := output.String()
	outputString = regexp.MustCompile(` \n`).ReplaceAllString(outputString, "\n")
	outputString = regexp.MustCompile(`\n{3,}`).ReplaceAllString(outputString, "\n\n")

	output.Reset()
	output.WriteString(outputString)
	return output
}

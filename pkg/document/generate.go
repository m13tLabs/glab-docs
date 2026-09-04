package document

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"

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

// PrintDocumentation renders the README for a single GitLab CI component/pipeline file.
func PrintDocumentation(
	info gitlab.ComponentDocumentationInfo,
	componentSearchRoot string,
	templateFiles []string,
	dryRun bool,
	glabDocsVersion string,
	componentPrefix string,
	skipVersionFooter bool,
) {
	log.Infof("Generating README documentation for %s", info.SourceFile)

	documentationTemplate, err := newComponentDocumentationTemplate(info, componentSearchRoot, templateFiles)
	if err != nil {
		log.Warnf("Error generating gotemplates for %s: %s", info.SourceFile, err)
		return
	}

	templateDataObject, err := getComponentTemplateData(info, glabDocsVersion, componentPrefix, skipVersionFooter)
	if err != nil {
		log.Warnf("Error generating template data for %s: %s", info.SourceFile, err)
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

	var output bytes.Buffer
	if err := documentationTemplate.Execute(&output, templateDataObject); err != nil {
		log.Warnf("Error generating documentation for %s: %s", info.SourceFile, err)
	}

	output = applyMarkDownFormat(output)
	if _, err := output.WriteTo(outputFile); err != nil {
		log.Warnf("Error writing documentation file for %s: %s", info.SourceFile, err)
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

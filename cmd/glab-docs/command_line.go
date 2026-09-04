package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/m13tLabs/glab-docs/pkg/document"
	"github.com/m13tLabs/glab-docs/pkg/gitlab"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var version string

func possibleLogLevels() []string {
	levels := make([]string, 0)

	for _, l := range log.AllLevels {
		levels = append(levels, l.String())
	}

	return levels
}

func initializeCli() {
	logLevelName := viper.GetString("log-level")
	logLevel, err := log.ParseLevel(logLevelName)
	if err != nil {
		log.Errorf("Failed to parse provided log level %s: %s", logLevelName, err)
		os.Exit(1)
	}

	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	log.SetLevel(logLevel)
}

func newGlabDocsCommand(run func(cmd *cobra.Command, args []string)) (*cobra.Command, error) {
	command := &cobra.Command{
		Use:     "glab-docs",
		Short:   "glab-docs automatically generates markdown documentation for GitLab CI components and pipelines from their spec:inputs and variables",
		Version: version,
		Run:     run,
	}

	logLevelUsage := fmt.Sprintf("Level of logs that should printed, one of (%s)", strings.Join(possibleLogLevels(), ", "))
	command.PersistentFlags().StringP("search-root", "c", ".", "directory to search recursively within for GitLab CI YAML files")
	command.PersistentFlags().StringSliceP("search-pattern", "p", gitlab.DefaultSearchPatterns, "glob patterns (relative to search-root) that identify documentable CI YAML files")
	command.PersistentFlags().BoolP("dry-run", "d", false, "don't actually render any markdown files just print to stdout")
	command.PersistentFlags().StringP("ignore-file", "i", ".glabdocsignore", "the filename to use as an ignore file to exclude directories/files")
	command.PersistentFlags().StringP("log-level", "l", "info", logLevelUsage)
	command.PersistentFlags().StringP("output-file", "o", "README.md", "markdown file path relative to each component directory to which rendered documentation will be written")
	command.PersistentFlags().StringP("sort-values-order", "s", document.AlphaNumSortOrder, fmt.Sprintf("order in which to sort the inputs/variables tables (\"%s\" or \"%s\")", document.AlphaNumSortOrder, document.FileSortOrder))
	command.PersistentFlags().StringSliceP("template-files", "t", []string{"README.md.gotmpl"}, "gotemplate file paths relative to each component directory from which documentation will be generated")
	command.PersistentFlags().String("component-prefix", "", "component address used in the generated usage snippet, e.g. gitlab.com/my-group/my-project. When empty a $CI_SERVER_FQDN placeholder is used")
	command.PersistentFlags().StringSliceP("component-to-generate", "g", []string{}, "list of component files that will have documentation generated. Comma separated. Empty - generate for all matches")
	command.PersistentFlags().BoolP("documentation-strict-mode", "x", false, "fail the generation of docs if there are undocumented inputs, variables or jobs")
	command.PersistentFlags().StringSliceP("documentation-strict-ignore-absent", "y", []string{}, "comma separated inputs.<name> / variables.<name> / jobs.<name> paths allowed not to be documented in strict mode")
	command.PersistentFlags().StringSliceP("documentation-strict-ignore-absent-regex", "z", []string{}, "comma separated regexps of inputs./variables./jobs. paths allowed not to be documented in strict mode")
	command.PersistentFlags().Bool("skip-version-footer", false, "if true the glab-docs version footer will not be shown in the default README template")

	viper.AutomaticEnv()
	viper.SetEnvPrefix("GLAB_DOCS")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	err := viper.BindPFlags(command.PersistentFlags())

	return command, err
}

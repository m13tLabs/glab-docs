package gitlab_test

import (
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/m13tLabs/glab-docs/pkg/gitlab"
)

type ComponentParsingTestSuite struct {
	suite.Suite
}

func TestComponentParsingTestSuite(t *testing.T) {
	suite.Run(t, new(ComponentParsingTestSuite))
}

func (suite *ComponentParsingTestSuite) fixture(name string) string {
	return filepath.Join("test-fixtures", name, "template.yml")
}

func (suite *ComponentParsingTestSuite) TestParsesInputsVariablesAndJobs() {
	info, err := gitlab.ParseComponentInformation(suite.fixture("full"), gitlab.DocumentationParsingConfig{})
	suite.NoError(err)
	suite.Equal("full", info.Name)
	suite.NotNil(info.SpecInputs)
	suite.Nil(info.Variables) // variables here are nested inside a job, not top-level
	suite.Len(info.Jobs, 1)
	suite.Equal("build image", info.Jobs[0].Name)
}

func (suite *ComponentParsingTestSuite) TestNotFullyDocumentedStrictModeOn() {
	_, err := gitlab.ParseComponentInformation(suite.fixture("full"), gitlab.DocumentationParsingConfig{StrictMode: true})
	suite.EqualError(err, "values without documentation: \ninputs.stage\ninputs.push")
}

func (suite *ComponentParsingTestSuite) TestUndocumentedJobFailsStrictMode() {
	_, err := gitlab.ParseComponentInformation(suite.fixture("full"), gitlab.DocumentationParsingConfig{
		StrictMode:               true,
		AllowedMissingValuePaths: []string{"inputs.stage", "inputs.push"},
	})
	suite.EqualError(err, "values without documentation: \njobs.build image")
}

func (suite *ComponentParsingTestSuite) TestStrictModeIgnoresByPath() {
	_, err := gitlab.ParseComponentInformation(suite.fixture("full"), gitlab.DocumentationParsingConfig{
		StrictMode:               true,
		AllowedMissingValuePaths: []string{"inputs.stage", "push", "jobs.build image"},
	})
	suite.NoError(err)
}

func (suite *ComponentParsingTestSuite) TestStrictModeIgnoresByRegexp() {
	_, err := gitlab.ParseComponentInformation(suite.fixture("full"), gitlab.DocumentationParsingConfig{
		StrictMode: true,
		AllowedMissingValueRegexps: []*regexp.Regexp{
			regexp.MustCompile(`^inputs\.`),
			regexp.MustCompile(`^jobs\.`),
		},
	})
	suite.NoError(err)
}

func (suite *ComponentParsingTestSuite) TestFullyDocumentedStrictModeOn() {
	_, err := gitlab.ParseComponentInformation(suite.fixture("fully-documented"), gitlab.DocumentationParsingConfig{StrictMode: true})
	suite.NoError(err)
}

func (suite *ComponentParsingTestSuite) TestJobParsing() {
	info, err := gitlab.ParseComponentInformation(suite.fixture("jobs"), gitlab.DocumentationParsingConfig{})
	suite.NoError(err)

	suite.Equal([]string{"build", "test", "deploy"}, info.Stages)
	suite.NotNil(info.Variables)

	// `.base` is hidden and jobs come back ordered by the declared stages.
	names := make([]string, 0, len(info.Jobs))
	for _, j := range info.Jobs {
		names = append(names, j.Name)
	}
	suite.Equal([]string{".base", "compile", "test", "ship"}, names)

	byName := map[string]gitlab.Job{}
	for _, j := range info.Jobs {
		byName[j.Name] = j
	}
	suite.True(byName[".base"].Hidden)
	suite.Equal("Compiles the project.", byName["compile"].Description)
	suite.Equal([]string{"compile"}, byName["test"].Needs)
	suite.Equal("manual", byName["ship"].When)
	suite.Equal([]string{"test"}, byName["ship"].Needs)
	suite.Equal([]string{".base"}, byName["ship"].Extends)
}

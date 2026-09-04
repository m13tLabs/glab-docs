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

func (suite *ComponentParsingTestSuite) TestParsesInputsAndVariables() {
	info, err := gitlab.ParseComponentInformation(suite.fixture("full"), gitlab.DocumentationParsingConfig{})
	suite.NoError(err)
	suite.Equal("full", info.Name)
	suite.NotNil(info.SpecInputs)
	suite.Nil(info.Variables) // variables here are nested inside a job, not top-level
}

func (suite *ComponentParsingTestSuite) TestNotFullyDocumentedStrictModeOff() {
	_, err := gitlab.ParseComponentInformation(suite.fixture("full"), gitlab.DocumentationParsingConfig{StrictMode: false})
	suite.NoError(err)
}

func (suite *ComponentParsingTestSuite) TestNotFullyDocumentedStrictModeOn() {
	_, err := gitlab.ParseComponentInformation(suite.fixture("full"), gitlab.DocumentationParsingConfig{StrictMode: true})
	suite.EqualError(err, "values without documentation: \ninputs.stage\ninputs.push")
}

func (suite *ComponentParsingTestSuite) TestStrictModeIgnoresByPath() {
	_, err := gitlab.ParseComponentInformation(suite.fixture("full"), gitlab.DocumentationParsingConfig{
		StrictMode:               true,
		AllowedMissingValuePaths: []string{"inputs.stage", "push"},
	})
	suite.NoError(err)
}

func (suite *ComponentParsingTestSuite) TestStrictModeIgnoresByRegexp() {
	_, err := gitlab.ParseComponentInformation(suite.fixture("full"), gitlab.DocumentationParsingConfig{
		StrictMode:                 true,
		AllowedMissingValueRegexps: []*regexp.Regexp{regexp.MustCompile(`^inputs\..*`)},
	})
	suite.NoError(err)
}

func (suite *ComponentParsingTestSuite) TestFullyDocumentedStrictModeOn() {
	_, err := gitlab.ParseComponentInformation(suite.fixture("fully-documented"), gitlab.DocumentationParsingConfig{StrictMode: true})
	suite.NoError(err)
}

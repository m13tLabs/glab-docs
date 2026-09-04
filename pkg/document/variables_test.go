package document

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"

	"github.com/m13tLabs/glab-docs/pkg/gitlab"
)

func variablesNode(t *testing.T, src string) *yaml.Node {
	t.Helper()
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(strings.TrimSpace(src)), &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return doc.Content[0].Content[1] // variables value
}

func TestGetVariableRows(t *testing.T) {
	node := variablesNode(t, `
variables:
  SIMPLE: "foo"
  DOCKER_DRIVER: overlay2
  DEPLOY_ENV:
    value: staging
    description: Target environment.
    options:
      - staging
      - production
`)

	rows, err := getVariableRows(node, map[string]gitlab.ValueDescription{})
	assert.NoError(t, err)
	sortVariableRows(rows)

	byName := make(map[string]variableRow)
	for _, r := range rows {
		byName[r.Name] = r
	}

	assert.Equal(t, "`foo`", byName["SIMPLE"].Default)
	assert.Equal(t, "`staging`", byName["DEPLOY_ENV"].Default)
	assert.Equal(t, "Target environment.", byName["DEPLOY_ENV"].Description)
	assert.Equal(t, []string{"staging", "production"}, byName["DEPLOY_ENV"].Options)
}

func TestGetVariableRowsCommentOverride(t *testing.T) {
	node := variablesNode(t, `
variables:
  REGISTRY: $CI_REGISTRY_IMAGE
`)

	rows, err := getVariableRows(node, map[string]gitlab.ValueDescription{
		"REGISTRY": {Description: "Registry the image is pushed to."},
	})
	assert.NoError(t, err)
	assert.Equal(t, "Registry the image is pushed to.", rows[0].Description)
}

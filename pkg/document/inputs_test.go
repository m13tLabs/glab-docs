package document

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"

	"github.com/m13tLabs/glab-docs/pkg/gitlab"
)

func inputsNode(t *testing.T, src string) *yaml.Node {
	t.Helper()
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(strings.TrimSpace(src)), &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	inputs := doc.Content[0].Content[1].Content[1] // spec -> inputs value
	return inputs
}

func rowsByName(rows []inputRow) map[string]inputRow {
	m := make(map[string]inputRow, len(rows))
	for _, r := range rows {
		m[r.Name] = r
	}
	return m
}

func TestGetInputRows(t *testing.T) {
	node := inputsNode(t, `
spec:
  inputs:
    website:
      description: URL to scan.
    scanner:
      default: zaproxy
      options:
        - zaproxy
        - other
    parallel:
      type: number
      default: 2
    enabled:
      type: boolean
      default: true
    paths:
      type: array
      default: ["src"]
    tag:
      default: ""
      regex: '^[\w.-]+$'
`)

	rows, err := getInputRows(node, map[string]gitlab.ValueDescription{})
	assert.NoError(t, err)
	sortInputRows(rows)
	byName := rowsByName(rows)

	assert.True(t, byName["website"].Required)
	assert.Equal(t, "string", byName["website"].Type)
	assert.Equal(t, "URL to scan.", byName["website"].Description)

	assert.False(t, byName["scanner"].Required)
	assert.Equal(t, "`zaproxy`", byName["scanner"].Default)
	assert.Equal(t, []string{"zaproxy", "other"}, byName["scanner"].Options)

	assert.Equal(t, "number", byName["parallel"].Type)
	assert.Equal(t, "`2`", byName["parallel"].Default)

	assert.Equal(t, "boolean", byName["enabled"].Type)
	assert.Equal(t, "array", byName["paths"].Type)
	assert.Equal(t, "`[\"src\"]`", byName["paths"].Default)

	assert.Equal(t, `^[\w.-]+$`, byName["tag"].Regex)
}

func TestGetInputRowsCommentOverride(t *testing.T) {
	node := inputsNode(t, `
spec:
  inputs:
    stage:
      default: test
      description: Native description.
`)

	rows, err := getInputRows(node, map[string]gitlab.ValueDescription{
		"inputs.stage": {Description: "Overridden description.", Section: "Core"},
	})
	assert.NoError(t, err)
	assert.Equal(t, "Overridden description.", rows[0].Description)
	assert.Equal(t, "Core", rows[0].Section)
}

func TestGetInputRowsNil(t *testing.T) {
	rows, err := getInputRows(nil, nil)
	assert.NoError(t, err)
	assert.Empty(t, rows)
}

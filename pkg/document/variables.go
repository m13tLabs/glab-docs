package document

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/m13tLabs/glab-docs/pkg/gitlab"
)

// getVariableRows builds the table rows for a pipeline's top-level `variables:` block. Each entry
// is either a scalar (the value) or the extended form `{value, description, options, expand}`.
func getVariableRows(variables *yaml.Node, descriptions map[string]gitlab.ValueDescription) ([]variableRow, error) {
	if variables == nil {
		return nil, nil
	}
	if variables.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("variables must be a mapping (was kind %d)", variables.Kind)
	}

	rows := make([]variableRow, 0, len(variables.Content)/2)

	for i := 0; i+1 < len(variables.Content); i += 2 {
		keyNode := variables.Content[i]
		valueNode := variables.Content[i+1]

		row := variableRow{
			Name:       keyNode.Value,
			LineNumber: keyNode.Line,
			Column:     keyNode.Column,
		}

		switch valueNode.Kind {
		case yaml.MappingNode:
			for j := 0; j+1 < len(valueNode.Content); j += 2 {
				fieldKey := valueNode.Content[j].Value
				fieldVal := valueNode.Content[j+1]
				switch fieldKey {
				case "value":
					row.Default = renderNodeValue(fieldVal)
				case "description":
					row.Description = fieldVal.Value
				case "options":
					row.Options = stringSliceFromNode(fieldVal)
				}
			}
		default:
			row.Default = renderNodeValue(valueNode)
		}

		override := commentOverride(row.Name, "variables", keyNode, descriptions)
		if override.Description != "" {
			row.Description = override.Description
		}
		if override.Default != "" {
			row.Default = override.Default
		}
		if override.Section != "" {
			row.Section = override.Section
		}

		rows = append(rows, row)
	}

	return rows, nil
}

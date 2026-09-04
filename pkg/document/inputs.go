package document

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/m13tLabs/glab-docs/pkg/gitlab"
)

const (
	typeString  = "string"
	typeNumber  = "number"
	typeBoolean = "boolean"
	typeArray   = "array"
)

// renderNodeValue produces a Markdown-safe, backtick-wrapped rendering of a YAML node's value.
func renderNodeValue(node *yaml.Node) string {
	if node == nil {
		return ""
	}

	switch node.Kind {
	case yaml.ScalarNode:
		if node.Tag == "!!null" || node.Value == "" {
			return ""
		}
		return fmt.Sprintf("`%s`", node.Value)
	case yaml.SequenceNode, yaml.MappingNode:
		var decoded interface{}
		if err := node.Decode(&decoded); err != nil {
			return ""
		}
		encoded, err := json.Marshal(decoded)
		if err != nil {
			return ""
		}
		return fmt.Sprintf("`%s`", string(encoded))
	default:
		return ""
	}
}

func inferInputType(explicit string, defaultNode *yaml.Node) string {
	switch explicit {
	case typeString, typeNumber, typeBoolean, typeArray:
		return explicit
	}
	if defaultNode == nil {
		return typeString
	}
	switch defaultNode.Kind {
	case yaml.SequenceNode:
		return typeArray
	case yaml.MappingNode:
		return typeString
	case yaml.ScalarNode:
		switch defaultNode.Tag {
		case "!!bool":
			return typeBoolean
		case "!!int", "!!float":
			return typeNumber
		}
	}
	return typeString
}

func stringSliceFromNode(node *yaml.Node) []string {
	if node == nil || node.Kind != yaml.SequenceNode {
		return nil
	}
	out := make([]string, 0, len(node.Content))
	for _, item := range node.Content {
		out = append(out, item.Value)
	}
	return out
}

// commentOverride returns the `# --` / `# key --` annotation for a named input or variable,
// checking the bare name, the prefixed path, and the key node's HeadComment.
func commentOverride(name, prefix string, keyNode *yaml.Node, descriptions map[string]gitlab.ValueDescription) gitlab.ValueDescription {
	if d, ok := descriptions[prefix+"."+name]; ok {
		return d
	}
	if d, ok := descriptions[name]; ok {
		return d
	}
	if keyNode != nil && keyNode.HeadComment != "" && strings.Contains(keyNode.HeadComment, gitlab.PrefixComment) {
		key, d := gitlab.ParseComment(strings.Split(keyNode.HeadComment, "\n"))
		if key == "" {
			return d
		}
	}
	return gitlab.ValueDescription{}
}

func getInputRows(specInputs *yaml.Node, descriptions map[string]gitlab.ValueDescription) ([]inputRow, error) {
	if specInputs == nil {
		return nil, nil
	}
	if specInputs.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("spec.inputs must be a mapping (was kind %d)", specInputs.Kind)
	}

	rows := make([]inputRow, 0, len(specInputs.Content)/2)

	for i := 0; i+1 < len(specInputs.Content); i += 2 {
		keyNode := specInputs.Content[i]
		valueNode := specInputs.Content[i+1]
		name := keyNode.Value

		row := inputRow{
			Name:       name,
			LineNumber: keyNode.Line,
			Column:     keyNode.Column,
			Required:   true,
		}

		var explicitType string
		var defaultNode *yaml.Node

		if valueNode.Kind == yaml.MappingNode {
			for j := 0; j+1 < len(valueNode.Content); j += 2 {
				fieldKey := valueNode.Content[j].Value
				fieldVal := valueNode.Content[j+1]
				switch fieldKey {
				case "default":
					row.Required = false
					defaultNode = fieldVal
				case "description":
					row.Description = fieldVal.Value
				case "type":
					explicitType = fieldVal.Value
				case "regex":
					row.Regex = fieldVal.Value
				case "options":
					row.Options = stringSliceFromNode(fieldVal)
				}
			}
		}

		row.Type = inferInputType(explicitType, defaultNode)
		if !row.Required {
			row.Default = renderNodeValue(defaultNode)
		}

		override := commentOverride(name, "inputs", keyNode, descriptions)
		if override.Description != "" {
			row.Description = override.Description
		}
		if override.Default != "" {
			row.Default = override.Default
		}
		if override.ValueType != "" {
			row.Type = override.ValueType
		}
		if override.Section != "" {
			row.Section = override.Section
		}

		rows = append(rows, row)
	}

	return rows, nil
}

package gitlab

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

var (
	valuesDescriptionRegex   = regexp.MustCompile(`^\s*#\s*(.*)\s+--\s*(.*)$`)
	rawDescriptionRegex      = regexp.MustCompile(`^\s*#\s+@raw`)
	commentContinuationRegex = regexp.MustCompile(`^\s*#(\s?)(.*)$`)
	defaultValueRegex        = regexp.MustCompile(`^\s*# @default -- (.*)$`)
	valueTypeRegex           = regexp.MustCompile(`^\((.*?)\)\s*(.*)$`)
	valueNotationTypeRegex   = regexp.MustCompile(`^\s*#\s+@notationType\s+--\s+(.*)$`)
	sectionRegex             = regexp.MustCompile(`^\s*# @section -- (.*)$`)
)

// ValueDescription holds the documentation parsed for a single input or variable, either from its
// native `description:` field or from a `# --` comment annotation.
type ValueDescription struct {
	Description  string
	Default      string
	Section      string
	ValueType    string
	NotationType string
}

// IncludeItem is one entry of a pipeline's top-level `include:` list.
type IncludeItem struct {
	// Kind is one of component, local, project, remote, template.
	Kind string
	// Location is the component address / file path / URL / template name.
	Location string
	// Ref is the git ref for project includes (empty otherwise).
	Ref string
}

// ComponentDocumentationInfo is everything parsed from a single GitLab CI YAML file that the
// templates need in order to render its README.
type ComponentDocumentationInfo struct {
	Name               string
	Description        string
	ComponentDirectory string
	SourceFile         string

	SpecInputs           *yaml.Node
	InputDescriptions    map[string]ValueDescription
	Variables            *yaml.Node
	VariableDescriptions map[string]ValueDescription
	Includes             []IncludeItem
}

// DocumentationParsingConfig controls strict-mode linting of undocumented inputs/variables.
type DocumentationParsingConfig struct {
	StrictMode                 bool
	AllowedMissingValuePaths   []string
	AllowedMissingValueRegexps []*regexp.Regexp
}

func getYamlFileContents(filename string) ([]byte, error) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, err
	}

	yamlFileContents, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return []byte(strings.ReplaceAll(string(yamlFileContents), "\r\n", "\n")), nil
}

func findMapValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil {
		return nil
	}
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		node = node.Content[0]
	}
	if node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

// removeIgnored strips mapping entries (and their children) whose key carries a `# @ignore`
// comment. Copied from the original helm-docs behaviour.
func removeIgnored(rootNode *yaml.Node, parentKind yaml.Kind) {
	if rootNode == nil {
		return
	}
	newContent := make([]*yaml.Node, 0, len(rootNode.Content))
	for i := 0; i < len(rootNode.Content); i++ {
		node := rootNode.Content[i]
		if !strings.Contains(node.HeadComment, "@ignore") {
			removeIgnored(node, node.Kind)
			newContent = append(newContent, node)
		} else if parentKind == yaml.MappingNode {
			// in a mapping each key is represented by two nodes; drop the value too
			i++
		}
	}
	rootNode.Content = newContent
}

func parseIncludes(includeNode *yaml.Node) []IncludeItem {
	if includeNode == nil {
		return nil
	}

	// `include:` may be a single string, a single map, or a sequence of either.
	var entries []*yaml.Node
	switch includeNode.Kind {
	case yaml.SequenceNode:
		entries = includeNode.Content
	default:
		entries = []*yaml.Node{includeNode}
	}

	includes := make([]IncludeItem, 0, len(entries))
	for _, entry := range entries {
		switch entry.Kind {
		case yaml.ScalarNode:
			includes = append(includes, IncludeItem{Kind: "local", Location: entry.Value})
		case yaml.MappingNode:
			item := IncludeItem{}
			for i := 0; i+1 < len(entry.Content); i += 2 {
				k := entry.Content[i].Value
				v := entry.Content[i+1].Value
				switch k {
				case "component":
					item.Kind, item.Location = "component", v
				case "local":
					item.Kind, item.Location = "local", v
				case "remote":
					item.Kind, item.Location = "remote", v
				case "template":
					item.Kind, item.Location = "template", v
				case "project":
					if item.Kind == "" {
						item.Kind = "project"
					}
					item.Location = v
				case "file":
					if item.Kind == "" || item.Kind == "project" {
						item.Kind = "project"
					}
					if item.Location != "" {
						item.Location += " :: " + v
					} else {
						item.Location = v
					}
				case "ref":
					item.Ref = v
				}
			}
			if item.Kind != "" {
				includes = append(includes, item)
			}
		}
	}
	return includes
}

func deriveComponentName(sourceFile string) string {
	dir := filepath.Dir(sourceFile)
	base := filepath.Base(sourceFile)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	stem = strings.TrimSuffix(stem, ".gitlab-ci")

	if stem == "" || stem == "template" || strings.HasPrefix(base, ".gitlab-ci") {
		parent := filepath.Base(dir)
		if parent == "templates" {
			parent = filepath.Base(filepath.Dir(dir))
		}
		if parent != "" && parent != "." && parent != string(filepath.Separator) {
			return parent
		}
		return "pipeline"
	}
	return stem
}

func fileLeadingDescription(contents []byte, docs []*yaml.Node) string {
	for _, doc := range docs {
		candidates := []string{doc.HeadComment}
		if len(doc.Content) > 0 {
			root := doc.Content[0]
			candidates = append(candidates, root.HeadComment)
			if len(root.Content) > 0 {
				candidates = append(candidates, root.Content[0].HeadComment)
			}
		}
		for _, headComment := range candidates {
			if headComment == "" || !strings.Contains(headComment, PrefixComment) {
				continue
			}
			key, desc := ParseComment(strings.Split(headComment, "\n"))
			if key == "" && desc.Description != "" {
				return desc.Description
			}
		}
	}

	// Fall back to a scan of the leading comment block of the raw file.
	commentLines := make([]string, 0)
	for _, line := range strings.Split(string(contents), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if !strings.HasPrefix(trimmed, "#") {
			break
		}
		commentLines = append(commentLines, line)
	}
	if len(commentLines) > 0 {
		if key, desc := ParseComment(commentLines); key == "" && desc.Description != "" {
			return desc.Description
		}
	}
	return ""
}

// parseFileComments scans the raw file for old-style `# key -- description` annotations. New-style
// `# --` comments attached to a specific input/variable are picked up from the YAML node's
// HeadComment in the document package instead.
func parseFileComments(sourceFile string) (map[string]ValueDescription, error) {
	f, err := os.Open(sourceFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	keyToDescriptions := make(map[string]ValueDescription)
	scanner := bufio.NewScanner(f)
	foundComment := false
	commentLines := make([]string, 0)

	flush := func() {
		if len(commentLines) == 0 {
			return
		}
		key, description := ParseComment(commentLines)
		if key != "" {
			keyToDescriptions[key] = description
		}
		commentLines = commentLines[:0]
		foundComment = false
	}

	for scanner.Scan() {
		line := scanner.Text()

		if !foundComment {
			match := valuesDescriptionRegex.FindStringSubmatch(line)
			if len(match) < 3 || match[1] == "" {
				continue
			}
			foundComment = true
			commentLines = append(commentLines, line)
			continue
		}

		if len(defaultValueRegex.FindStringSubmatch(line)) > 1 ||
			len(sectionRegex.FindStringSubmatch(line)) > 1 ||
			len(commentContinuationRegex.FindStringSubmatch(line)) > 1 {
			commentLines = append(commentLines, line)
			continue
		}

		flush()
	}
	flush()

	return keyToDescriptions, scanner.Err()
}

func decodeAllDocuments(contents []byte) ([]*yaml.Node, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(contents))
	docs := make([]*yaml.Node, 0, 2)
	for {
		var node yaml.Node
		err := decoder.Decode(&node)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if node.Kind == 0 {
			continue
		}
		docs = append(docs, &node)
	}
	return docs, nil
}

func checkDocumentation(
	names []string,
	comments map[string]ValueDescription,
	headComments map[string]bool,
	config DocumentationParsingConfig,
	prefix string,
) error {
	undocumented := make([]string, 0)
	for _, name := range names {
		if headComments[name] {
			continue
		}
		if _, ok := comments[name]; ok {
			continue
		}
		if _, ok := comments[prefix+"."+name]; ok {
			continue
		}

		path := prefix + "." + name
		ignored := false
		for _, allowed := range config.AllowedMissingValuePaths {
			if allowed == path || allowed == name {
				ignored = true
			}
		}
		for _, re := range config.AllowedMissingValueRegexps {
			if re.MatchString(path) {
				ignored = true
			}
		}
		if !ignored {
			undocumented = append(undocumented, path)
		}
	}
	if len(undocumented) > 0 {
		return fmt.Errorf("values without documentation: \n%s", strings.Join(undocumented, "\n"))
	}
	return nil
}

func mappingKeyNames(node *yaml.Node) []string {
	names := make([]string, 0)
	if node == nil || node.Kind != yaml.MappingNode {
		return names
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		names = append(names, node.Content[i].Value)
	}
	return names
}

// documentedNames returns the set of input/variable names that are already documented, either via
// a `# --` HeadComment on the key or via a native `description:` child field.
func documentedNames(node *yaml.Node) map[string]bool {
	documented := make(map[string]bool)
	if node == nil || node.Kind != yaml.MappingNode {
		return documented
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		if keyNode.HeadComment != "" && strings.Contains(keyNode.HeadComment, PrefixComment) {
			documented[keyNode.Value] = true
			continue
		}
		if valueNode.Kind == yaml.MappingNode {
			for j := 0; j+1 < len(valueNode.Content); j += 2 {
				if valueNode.Content[j].Value == "description" && valueNode.Content[j+1].Value != "" {
					documented[keyNode.Value] = true
				}
			}
		}
	}
	return documented
}

// ParseComponentInformation reads a single GitLab CI YAML file and extracts everything needed to
// document it.
func ParseComponentInformation(sourceFile string, config DocumentationParsingConfig) (ComponentDocumentationInfo, error) {
	info := ComponentDocumentationInfo{
		SourceFile:           sourceFile,
		ComponentDirectory:   filepath.Dir(sourceFile),
		Name:                 deriveComponentName(sourceFile),
		InputDescriptions:    map[string]ValueDescription{},
		VariableDescriptions: map[string]ValueDescription{},
	}

	contents, err := getYamlFileContents(sourceFile)
	if err != nil {
		if os.IsNotExist(err) {
			log.Warnf("Component file %s missing, skipping", sourceFile)
		}
		return info, err
	}

	docs, err := decodeAllDocuments(contents)
	if err != nil {
		return info, err
	}
	if len(docs) == 0 {
		return info, fmt.Errorf("%s contains no YAML documents", sourceFile)
	}

	var specRoot, bodyRoot *yaml.Node
	for _, doc := range docs {
		if findMapValue(doc, "spec") != nil {
			specRoot = doc
		}
		if findMapValue(doc, "variables") != nil || findMapValue(doc, "include") != nil {
			bodyRoot = doc
		}
	}
	if bodyRoot == nil {
		bodyRoot = docs[len(docs)-1]
	}

	if specRoot != nil {
		info.SpecInputs = findMapValue(findMapValue(specRoot, "spec"), "inputs")
		removeIgnored(info.SpecInputs, yaml.MappingNode)
	}

	info.Variables = findMapValue(bodyRoot, "variables")
	removeIgnored(info.Variables, yaml.MappingNode)

	info.Includes = parseIncludes(findMapValue(bodyRoot, "include"))
	info.Description = fileLeadingDescription(contents, docs)

	fileComments, err := parseFileComments(sourceFile)
	if err != nil {
		return info, err
	}
	info.InputDescriptions = fileComments
	info.VariableDescriptions = fileComments

	if config.StrictMode {
		if err := checkDocumentation(
			mappingKeyNames(info.SpecInputs), fileComments, documentedNames(info.SpecInputs), config, "inputs",
		); err != nil {
			return info, err
		}
		if err := checkDocumentation(
			mappingKeyNames(info.Variables), fileComments, documentedNames(info.Variables), config, "variables",
		); err != nil {
			return info, err
		}
	}

	return info, nil
}

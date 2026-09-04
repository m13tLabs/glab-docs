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
	"sort"
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

// Job is a single job defined in a pipeline body.
type Job struct {
	Name         string
	Description  string
	Stage        string
	When         string   // explicit `when:`, or "rules" / "only/except" when those gate the job
	Needs        []string // upstream job names from `needs:`
	Extends      []string // templates pulled in via `extends:`
	Image        string
	AllowFailure bool
	Hidden       bool // name starts with "." - a template/anchor job, not scheduled
	LineNumber   int
}

// reservedTopLevelKeys are the global keywords that can appear at the top level of a pipeline
// body but are not jobs. https://docs.gitlab.com/ee/ci/yaml/#keywords
var reservedTopLevelKeys = map[string]bool{
	"stages": true, "variables": true, "include": true, "default": true,
	"workflow": true, "image": true, "services": true, "cache": true,
	"before_script": true, "after_script": true, "spec": true, "sast": true,
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
	Jobs                 []Job
	Stages               []string // declared `stages:` order, if any
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

// scalarOrNameList flattens a node that is a scalar, a sequence of scalars, or a sequence of
// mappings with a `job:`/`name:` key (as GitLab's `needs:` allows) into a list of strings.
func scalarOrNameList(node *yaml.Node) []string {
	if node == nil {
		return nil
	}
	switch node.Kind {
	case yaml.ScalarNode:
		if node.Value == "" {
			return nil
		}
		return []string{node.Value}
	case yaml.SequenceNode:
		out := make([]string, 0, len(node.Content))
		for _, item := range node.Content {
			switch item.Kind {
			case yaml.ScalarNode:
				out = append(out, item.Value)
			case yaml.MappingNode:
				if v := findMapValue(item, "job"); v != nil {
					out = append(out, v.Value)
				} else if v := findMapValue(item, "name"); v != nil {
					out = append(out, v.Value)
				}
			}
		}
		return out
	}
	return nil
}

func imageName(node *yaml.Node) string {
	if node == nil {
		return ""
	}
	if node.Kind == yaml.ScalarNode {
		return node.Value
	}
	if v := findMapValue(node, "name"); v != nil {
		return v.Value
	}
	return ""
}

func parseStages(stagesNode *yaml.Node) []string {
	if stagesNode == nil || stagesNode.Kind != yaml.SequenceNode {
		return nil
	}
	out := make([]string, 0, len(stagesNode.Content))
	for _, s := range stagesNode.Content {
		out = append(out, s.Value)
	}
	return out
}

// parseJobs walks the top-level keys of a pipeline body and returns one Job per non-reserved
// mapping entry.
func parseJobs(bodyRoot *yaml.Node, fileComments map[string]ValueDescription) []Job {
	if bodyRoot != nil && bodyRoot.Kind == yaml.DocumentNode && len(bodyRoot.Content) > 0 {
		bodyRoot = bodyRoot.Content[0]
	}
	if bodyRoot == nil || bodyRoot.Kind != yaml.MappingNode {
		return nil
	}

	jobs := make([]Job, 0)
	for i := 0; i+1 < len(bodyRoot.Content); i += 2 {
		keyNode := bodyRoot.Content[i]
		valueNode := bodyRoot.Content[i+1]
		name := keyNode.Value

		if reservedTopLevelKeys[name] || valueNode.Kind != yaml.MappingNode {
			continue
		}
		if strings.Contains(keyNode.HeadComment, "@ignore") {
			continue
		}

		job := Job{
			Name:       name,
			Hidden:     strings.HasPrefix(name, "."),
			Stage:      "",
			LineNumber: keyNode.Line,
		}

		if s := findMapValue(valueNode, "stage"); s != nil {
			job.Stage = s.Value
		}
		if w := findMapValue(valueNode, "when"); w != nil {
			job.When = w.Value
		} else if findMapValue(valueNode, "rules") != nil {
			job.When = "rules"
		} else if findMapValue(valueNode, "only") != nil || findMapValue(valueNode, "except") != nil {
			job.When = "only/except"
		}
		job.Needs = scalarOrNameList(findMapValue(valueNode, "needs"))
		job.Extends = scalarOrNameList(findMapValue(valueNode, "extends"))
		job.Image = imageName(findMapValue(valueNode, "image"))
		if af := findMapValue(valueNode, "allow_failure"); af != nil {
			job.AllowFailure = af.Value == "true"
		}

		if keyNode.HeadComment != "" && strings.Contains(keyNode.HeadComment, PrefixComment) {
			if key, d := ParseComment(strings.Split(keyNode.HeadComment, "\n")); key == "" {
				job.Description = d.Description
			}
		}
		if job.Description == "" {
			if d, ok := fileComments["jobs."+name]; ok {
				job.Description = d.Description
			} else if d, ok := fileComments[name]; ok {
				job.Description = d.Description
			}
		}

		jobs = append(jobs, job)
	}
	return jobs
}

// docLooksLikeBody reports whether a YAML document is the pipeline body (as opposed to the
// `spec:` header) - it carries variables/include/stages/workflow or at least one job.
func docLooksLikeBody(doc *yaml.Node) bool {
	root := doc
	if root != nil && root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = root.Content[0]
	}
	if root == nil || root.Kind != yaml.MappingNode {
		return false
	}
	for i := 0; i+1 < len(root.Content); i += 2 {
		key := root.Content[i].Value
		if key == "spec" {
			continue
		}
		if key == "variables" || key == "include" || key == "stages" || key == "workflow" {
			return true
		}
		if !reservedTopLevelKeys[key] && root.Content[i+1].Kind == yaml.MappingNode {
			return true // a job
		}
	}
	return false
}

func stageOrder(stages []string) map[string]int {
	order := make(map[string]int, len(stages))
	for i, s := range stages {
		order[s] = i
	}
	return order
}

// sortJobsByStage orders jobs by the declared `stages:` sequence (falling back to GitLab's
// implicit .pre / build / test / deploy / .post), then by declaration order.
func sortJobsByStage(jobs []Job, stages []string) {
	effective := stages
	if len(effective) == 0 {
		effective = []string{".pre", "build", "test", "deploy", ".post"}
	}
	order := stageOrder(effective)
	rank := func(j Job) int {
		if j.Hidden {
			return -1 // templates/anchors first
		}
		s := j.Stage
		if s == "" {
			s = "test"
		}
		if r, ok := order[s]; ok {
			return r
		}
		return len(order) + 1
	}
	sort.SliceStable(jobs, func(i, j int) bool {
		ri, rj := rank(jobs[i]), rank(jobs[j])
		if ri != rj {
			return ri < rj
		}
		return jobs[i].LineNumber < jobs[j].LineNumber
	})
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
		if docLooksLikeBody(doc) {
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
	info.Stages = parseStages(findMapValue(bodyRoot, "stages"))
	info.Description = fileLeadingDescription(contents, docs)

	fileComments, err := parseFileComments(sourceFile)
	if err != nil {
		return info, err
	}
	info.InputDescriptions = fileComments
	info.VariableDescriptions = fileComments
	info.Jobs = parseJobs(bodyRoot, fileComments)
	sortJobsByStage(info.Jobs, info.Stages)

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

		jobNames := make([]string, 0, len(info.Jobs))
		documentedJobs := make(map[string]bool, len(info.Jobs))
		for _, j := range info.Jobs {
			if j.Hidden {
				continue
			}
			jobNames = append(jobNames, j.Name)
			if j.Description != "" {
				documentedJobs[j.Name] = true
			}
		}
		if err := checkDocumentation(jobNames, fileComments, documentedJobs, config, "jobs"); err != nil {
			return info, err
		}
	}

	return info, nil
}

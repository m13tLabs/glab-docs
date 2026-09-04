package document

import (
	"os"
	"path/filepath"
	"strings"
	"text/template"

	log "github.com/sirupsen/logrus"

	"github.com/m13tLabs/glab-docs/pkg/gitlab"
	"github.com/m13tLabs/glab-docs/pkg/util"
)

const defaultDocumentationTemplate = `{{ template "pipeline.header" . }}

{{ template "pipeline.description" . }}

{{ template "pipeline.usageSection" . }}

{{ template "pipeline.inputsSection" . }}

{{ template "pipeline.variablesSection" . }}

{{ template "pipeline.jobsSection" . }}

{{ template "pipeline.includesSection" . }}

{{- if not .SkipVersionFooter }}
{{ template "glab-docs.versionFooter" . }}
{{- end }}
`

func getNameTemplate() string {
	return `{{ define "pipeline.name" }}{{ .Name }}{{ end }}`
}

func getHeaderTemplate() string {
	b := strings.Builder{}
	b.WriteString(`{{ define "pipeline.header" }}`)
	b.WriteString("# {{ .Name }}")
	b.WriteString("{{ end }}")
	return b.String()
}

func getDescriptionTemplate() string {
	b := strings.Builder{}
	b.WriteString(`{{ define "pipeline.description" }}`)
	b.WriteString("{{ if .Description }}{{ .Description }}{{ end }}")
	b.WriteString("{{ end }}")
	return b.String()
}

func getUsageTemplate() string {
	b := strings.Builder{}
	b.WriteString(`{{ define "pipeline.usageHeader" }}## Usage{{ end }}`)

	b.WriteString(`{{ define "pipeline.usageSnippet" }}`)
	b.WriteString("```yaml\n")
	b.WriteString("include:\n")
	b.WriteString("  - component: {{ .ComponentPrefix }}")
	b.WriteString("{{- if .Inputs }}\n    inputs:")
	b.WriteString("{{- range .Inputs }}\n")
	b.WriteString(`      {{ .Name }}: {{ if .Required }}"" # required{{ else if .Default }}{{ trimAll "` + "`" + `" .Default }}{{ else }}""{{ end }}`)
	b.WriteString("{{- end }}")
	b.WriteString("{{- end }}\n")
	b.WriteString("```")
	b.WriteString("{{ end }}")

	b.WriteString(`{{ define "pipeline.usageSection" }}`)
	b.WriteString("{{ if and .ComponentPrefix .Inputs }}")
	b.WriteString(`{{ template "pipeline.usageHeader" . }}`)
	b.WriteString("\n\n")
	b.WriteString(`{{ template "pipeline.usageSnippet" . }}`)
	b.WriteString("{{ end }}")
	b.WriteString("{{ end }}")

	return b.String()
}

func getInputsTemplate() string {
	b := strings.Builder{}
	b.WriteString(`{{ define "pipeline.inputsHeader" }}## Inputs{{ end }}`)

	const row = "\n| {{ .Name }} | {{ .Type }} | {{ if .Required }}_required_{{ else if .Default }}{{ .Default }}{{ else }}_none_{{ end }} " +
		"| {{ range $i, $o := .Options }}{{ if $i }}, {{ end }}`{{ $o }}`{{ end }} | {{ .Description }}{{ if .Regex }}{{ if .Description }}<br>{{ end }}Pattern: `{{ .Regex }}`{{ end }} |"
	const header = "| Input | Type | Default | Options | Description |\n|-------|------|---------|---------|-------------|"

	b.WriteString(`{{ define "pipeline.inputsTable" }}`)
	b.WriteString("{{ if .InputSections.Sections }}")
	b.WriteString("{{ range .InputSections.Sections }}")
	b.WriteString("\n\n### {{ .SectionName }}\n\n")
	b.WriteString(header)
	b.WriteString("  {{- range .SectionItems }}")
	b.WriteString(row)
	b.WriteString("  {{- end }}")
	b.WriteString("{{- end }}")
	b.WriteString("{{ if .InputSections.DefaultSection.SectionItems }}")
	b.WriteString("\n\n### {{ .InputSections.DefaultSection.SectionName }}\n\n")
	b.WriteString(header)
	b.WriteString("  {{- range .InputSections.DefaultSection.SectionItems }}")
	b.WriteString(row)
	b.WriteString("  {{- end }}")
	b.WriteString("{{ end }}")
	b.WriteString("{{ else }}")
	b.WriteString(header)
	b.WriteString("  {{- range .Inputs }}")
	b.WriteString(row)
	b.WriteString("  {{- end }}")
	b.WriteString("{{ end }}")
	b.WriteString("{{ end }}")

	b.WriteString(`{{ define "pipeline.inputsSection" }}`)
	b.WriteString("{{ if .Inputs }}")
	b.WriteString(`{{ template "pipeline.inputsHeader" . }}`)
	b.WriteString("\n\n")
	b.WriteString(`{{ template "pipeline.inputsTable" . }}`)
	b.WriteString("{{ end }}")
	b.WriteString("{{ end }}")

	return b.String()
}

func getVariablesTemplate() string {
	b := strings.Builder{}
	b.WriteString(`{{ define "pipeline.variablesHeader" }}## Variables{{ end }}`)

	b.WriteString(`{{ define "pipeline.variablesTable" }}`)
	b.WriteString("| Variable | Default | Options | Description |\n|----------|---------|---------|-------------|")
	b.WriteString("  {{- range .Variables }}")
	b.WriteString("\n| {{ .Name }} | {{ if .Default }}{{ .Default }}{{ else }}_none_{{ end }} | {{ range $i, $o := .Options }}{{ if $i }}, {{ end }}`{{ $o }}`{{ end }} | {{ .Description }} |")
	b.WriteString("  {{- end }}")
	b.WriteString("{{ end }}")

	b.WriteString(`{{ define "pipeline.variablesSection" }}`)
	b.WriteString("{{ if .Variables }}")
	b.WriteString(`{{ template "pipeline.variablesHeader" . }}`)
	b.WriteString("\n\n")
	b.WriteString(`{{ template "pipeline.variablesTable" . }}`)
	b.WriteString("{{ end }}")
	b.WriteString("{{ end }}")

	return b.String()
}

func getJobsTemplate() string {
	b := strings.Builder{}
	b.WriteString(`{{ define "pipeline.jobsHeader" }}## Jobs{{ end }}`)

	b.WriteString(`{{ define "pipeline.jobsTable" }}`)
	b.WriteString("| Job | Stage | When | Needs | Description |\n|-----|-------|------|-------|-------------|")
	b.WriteString("  {{- range .JobRows }}")
	b.WriteString("\n| `{{ .Name }}` | {{ if .Stage }}`{{ .Stage }}`{{ else }}`test`{{ end }} | {{ if .When }}`{{ .When }}`{{ end }} " +
		"| {{ range $i, $n := .Needs }}{{ if $i }}, {{ end }}`{{ $n }}`{{ end }} | {{ .Description }}{{ if .Extends }}{{ if .Description }}<br>{{ end }}Extends: {{ range $i, $e := .Extends }}{{ if $i }}, {{ end }}`{{ $e }}`{{ end }}{{ end }} |")
	b.WriteString("  {{- end }}")
	b.WriteString("{{ end }}")

	b.WriteString(`{{ define "pipeline.jobsSection" }}`)
	b.WriteString("{{ if .JobRows }}")
	b.WriteString(`{{ template "pipeline.jobsHeader" . }}`)
	b.WriteString("\n\n")
	b.WriteString(`{{ template "pipeline.jobsTable" . }}`)
	b.WriteString("{{ end }}")
	b.WriteString("{{ end }}")

	return b.String()
}

func getIncludesTemplate() string {
	b := strings.Builder{}
	b.WriteString(`{{ define "pipeline.includesHeader" }}## Includes{{ end }}`)

	b.WriteString(`{{ define "pipeline.includesTable" }}`)
	b.WriteString("| Type | Location | Ref |\n|------|----------|-----|")
	b.WriteString("  {{- range .IncludeItems }}")
	b.WriteString("\n| {{ .Kind }} | `{{ .Location }}` | {{ if .Ref }}`{{ .Ref }}`{{ end }} |")
	b.WriteString("  {{- end }}")
	b.WriteString("{{ end }}")

	b.WriteString(`{{ define "pipeline.includesSection" }}`)
	b.WriteString("{{ if .IncludeItems }}")
	b.WriteString(`{{ template "pipeline.includesHeader" . }}`)
	b.WriteString("\n\n")
	b.WriteString(`{{ template "pipeline.includesTable" . }}`)
	b.WriteString("{{ end }}")
	b.WriteString("{{ end }}")

	return b.String()
}

func getVersionFooterTemplate() string {
	b := strings.Builder{}
	b.WriteString(`{{ define "glab-docs.version" }}{{ if .GlabDocsVersion }}{{ .GlabDocsVersion }}{{ end }}{{ end }}`)
	b.WriteString(`{{ define "glab-docs.versionFooter" }}`)
	b.WriteString("{{ if .GlabDocsVersion }}\n")
	b.WriteString("----------------------------------------------\n")
	b.WriteString("Autogenerated from the pipeline definition using [glab-docs v{{ .GlabDocsVersion }}](https://github.com/m13tLabs/glab-docs/releases/v{{ .GlabDocsVersion }})")
	b.WriteString("{{ end }}")
	b.WriteString("{{ end }}")
	return b.String()
}

func getDocumentationTemplate(componentDirectory string, componentSearchRoot string, templateFiles []string) (string, error) {
	templateFilesForComponent := make([]string, 0)
	var templateNotFound bool

	for _, templateFile := range templateFiles {
		var fullTemplatePath string

		if util.IsRelativePath(templateFile) {
			fullTemplatePath = filepath.Join(componentSearchRoot, templateFile)
		} else if util.IsBaseFilename(templateFile) {
			fullTemplatePath = filepath.Join(componentDirectory, templateFile)
		} else {
			fullTemplatePath = templateFile
		}

		if _, err := os.Stat(fullTemplatePath); os.IsNotExist(err) {
			log.Debugf("Did not find template file %s for %s, using default template", templateFile, componentDirectory)
			templateNotFound = true
			continue
		}

		templateFilesForComponent = append(templateFilesForComponent, fullTemplatePath)
	}

	allTemplateContents := make([]byte, 0)
	for _, templateFileForComponent := range templateFilesForComponent {
		templateContents, err := os.ReadFile(templateFileForComponent)
		if err != nil {
			return "", err
		}
		allTemplateContents = append(allTemplateContents, templateContents...)
	}

	if templateNotFound {
		allTemplateContents = append(allTemplateContents, []byte(defaultDocumentationTemplate)...)
	}

	return string(allTemplateContents), nil
}

func getDocumentationTemplates(componentDirectory string, componentSearchRoot string, templateFiles []string) ([]string, error) {
	documentationTemplate, err := getDocumentationTemplate(componentDirectory, componentSearchRoot, templateFiles)
	if err != nil {
		log.Errorf("Failed to read documentation template for %s: %s", componentDirectory, err)
		return nil, err
	}

	return []string{
		getNameTemplate(),
		getHeaderTemplate(),
		getDescriptionTemplate(),
		getUsageTemplate(),
		getInputsTemplate(),
		getVariablesTemplate(),
		getJobsTemplate(),
		getIncludesTemplate(),
		getVersionFooterTemplate(),
		documentationTemplate,
	}, nil
}

func newComponentDocumentationTemplate(info gitlab.ComponentDocumentationInfo, componentSearchRoot string, templateFiles []string) (*template.Template, error) {
	documentationTemplate := template.New(info.SourceFile)
	documentationTemplate.Funcs(util.FuncMap())

	goTemplateList, err := getDocumentationTemplates(info.ComponentDirectory, componentSearchRoot, templateFiles)
	if err != nil {
		return nil, err
	}

	for _, t := range goTemplateList {
		if _, err := documentationTemplate.Parse(t); err != nil {
			return nil, err
		}
	}

	return documentationTemplate, nil
}

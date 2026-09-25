package document

import (
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/m13tLabs/glab-docs/pkg/gitlab"
)

// inputRow is one row of the component's `spec:inputs:` table.
type inputRow struct {
	Name        string
	Type        string
	Default     string
	Description string
	Regex       string
	Options     []string
	Required    bool
	Section     string
	LineNumber  int
	Column      int
}

// variableRow is one row of the pipeline's `variables:` table.
type variableRow struct {
	Name        string
	Default     string
	Description string
	Options     []string
	Section     string
	LineNumber  int
	Column      int
}

type section struct {
	SectionName  string
	SectionItems []inputRow
}

type sections struct {
	DefaultSection section
	Sections       []section
}

type componentTemplateData struct {
	gitlab.ComponentDocumentationInfo

	GlabDocsVersion   string
	ComponentPrefix   string
	Inputs            []inputRow
	InputSections     sections
	Variables         []variableRow
	IncludeItems      []includeRow
	JobRows           []jobRow
	Files             files
	SkipVersionFooter bool
	// HeadingLevel is the Markdown level (1 = `#`) of this component's own name heading, one
	// level below that of its section headings (Usage/Inputs/...), two below sub-section
	// headings (grouped `@section` inputs). It's 1 for a component rendered into its own
	// standalone README, and higher when several components are combined into one README with
	// a section apiece - see PrintCombinedDocumentation.
	HeadingLevel int
}

// includeRow is one row of the pipeline's `include:` table - a gitlab.IncludeItem plus, when the
// included file is another documented component, a Link to that component's own documentation.
type includeRow struct {
	Kind     string
	Location string
	Ref      string
	Link     string
	// Description summarizes what the include adds (its own description, variables and jobs),
	// already Markdown-formatted for a table cell; empty when it couldn't be resolved.
	Description string
}

// jobRow is one row of the pipeline's Jobs table.
type jobRow struct {
	Name        string
	Stage       string
	When        string
	Needs       []string
	Extends     []string
	Image       string
	Description string
	Hidden      bool
}

func resolveSortOrder() string {
	sortOrder := viper.GetString("sort-values-order")
	if sortOrder != FileSortOrder && sortOrder != AlphaNumSortOrder {
		log.Warnf("Invalid sort order provided %s, defaulting to %s", sortOrder, AlphaNumSortOrder)
		sortOrder = AlphaNumSortOrder
	}
	return sortOrder
}

func sortInputRows(rows []inputRow) {
	sortOrder := resolveSortOrder()
	sort.SliceStable(rows, func(i, j int) bool {
		if sortOrder == FileSortOrder {
			if rows[i].LineNumber == rows[j].LineNumber {
				return rows[i].Column < rows[j].Column
			}
			return rows[i].LineNumber < rows[j].LineNumber
		}
		return rows[i].Name < rows[j].Name
	})
}

func sortVariableRows(rows []variableRow) {
	sortOrder := resolveSortOrder()
	sort.SliceStable(rows, func(i, j int) bool {
		if sortOrder == FileSortOrder {
			if rows[i].LineNumber == rows[j].LineNumber {
				return rows[i].Column < rows[j].Column
			}
			return rows[i].LineNumber < rows[j].LineNumber
		}
		return rows[i].Name < rows[j].Name
	})
}

func groupInputSections(rows []inputRow) sections {
	grouped := sections{
		DefaultSection: section{SectionName: "Inputs", SectionItems: []inputRow{}},
	}

	for _, row := range rows {
		if row.Section == "" {
			grouped.DefaultSection.SectionItems = append(grouped.DefaultSection.SectionItems, row)
			continue
		}

		found := false
		for i := range grouped.Sections {
			if grouped.Sections[i].SectionName == row.Section {
				grouped.Sections[i].SectionItems = append(grouped.Sections[i].SectionItems, row)
				found = true
				break
			}
		}
		if !found {
			grouped.Sections = append(grouped.Sections, section{
				SectionName:  row.Section,
				SectionItems: []inputRow{row},
			})
		}
	}

	return grouped
}

func getComponentTemplateData(
	info gitlab.ComponentDocumentationInfo,
	relFile string,
	glabDocsVersion, componentPrefix string,
	skipVersionFooter bool,
	links map[string]ComponentLink,
	headingLevel int,
) (componentTemplateData, error) {
	inputRows, err := getInputRows(info.SpecInputs, info.InputDescriptions)
	if err != nil {
		return componentTemplateData{}, err
	}
	sortInputRows(inputRows)

	variableRows, err := getVariableRows(info.Variables, info.VariableDescriptions)
	if err != nil {
		return componentTemplateData{}, err
	}
	sortVariableRows(variableRows)

	componentFiles, err := getFiles(info.ComponentDirectory)
	if err != nil {
		return componentTemplateData{}, err
	}

	return componentTemplateData{
		ComponentDocumentationInfo: info,
		GlabDocsVersion:            glabDocsVersion,
		ComponentPrefix:            componentPrefix,
		Inputs:                     inputRows,
		InputSections:              groupInputSections(inputRows),
		Variables:                  variableRows,
		IncludeItems:               getIncludeRows(relFile, info.Includes, links),
		JobRows:                    getJobRows(info.Jobs),
		Files:                      componentFiles,
		SkipVersionFooter:          skipVersionFooter,
		HeadingLevel:               headingLevel,
	}, nil
}

func getIncludeRows(relFile string, items []gitlab.IncludeItem, links map[string]ComponentLink) []includeRow {
	serverURL := strings.TrimRight(viper.GetString("gitlab-server-url"), "/")

	rows := make([]includeRow, 0, len(items))
	for _, item := range items {
		location, link := item.Location, resolveIncludeLink(relFile, item, links)
		switch item.Kind {
		case "component":
			location, link = resolveComponentLocation(item, serverURL)
		case "project":
			location, link = resolveProjectLocation(item, serverURL)
		}
		rows = append(rows, includeRow{
			Kind:        item.Kind,
			Location:    location,
			Ref:         item.Ref,
			Link:        link,
			Description: includeDescription(item.Summary),
		})
	}
	return rows
}

// includeDescription renders an include's summary for its Description cell - the included file's
// own description, then a bullet list of the variables it (transitively) adds, with their default
// and description, and one of its jobs with their `# --` description - both gathered across the
// included file and everything it includes. Names defined by several of the included files are
// listed once, as first defined. The lists are inline HTML, since Markdown list syntax doesn't
// work inside a table cell; Markdown within the list items (code spans) still renders.
func includeDescription(summary *gitlab.IncludeSummary) string {
	if summary == nil {
		return ""
	}

	seenVariables := map[string]bool{}
	seenJobs := map[string]bool{}
	variableItems := make([]string, 0)
	jobItems := make([]string, 0)
	for _, file := range summary.Files {
		variableRows, err := getVariableRows(file.Variables, file.VariableDescriptions)
		if err != nil {
			log.Warnf("Skipping variables of included %s: %s", file.SourceFile, err)
		}
		for _, v := range variableRows {
			if seenVariables[v.Name] {
				continue
			}
			seenVariables[v.Name] = true
			item := "`" + v.Name + "`"
			if v.Default != "" {
				item += " = " + v.Default
			}
			if v.Description != "" {
				item += " - " + v.Description
			}
			variableItems = append(variableItems, item)
		}

		for _, j := range file.Jobs {
			if seenJobs[j.Name] {
				continue
			}
			seenJobs[j.Name] = true
			item := "`" + j.Name + "`"
			if j.Description != "" {
				item += " - " + j.Description
			}
			jobItems = append(jobItems, item)
		}
	}

	var b strings.Builder
	b.WriteString(summary.Description)
	for _, list := range []struct {
		title string
		items []string
	}{{"Variables", variableItems}, {"Jobs", jobItems}} {
		if len(list.items) == 0 {
			continue
		}
		if b.Len() > 0 && !strings.HasSuffix(b.String(), "</ul>") {
			b.WriteString("<br>") // a </ul> already ends its line
		}
		b.WriteString("**" + list.title + ":**<ul><li>" + strings.Join(list.items, "</li><li>") + "</li></ul>")
	}
	return b.String()
}

func getJobRows(jobs []gitlab.Job) []jobRow {
	rows := make([]jobRow, 0, len(jobs))
	for _, j := range jobs {
		if j.Hidden && j.Description == "" {
			continue // undocumented `.hidden` jobs are templates/anchors, not part of the pipeline
		}
		rows = append(rows, jobRow{
			Name:        j.Name,
			Stage:       j.Stage,
			When:        j.When,
			Needs:       j.Needs,
			Extends:     j.Extends,
			Image:       j.Image,
			Description: j.Description,
			Hidden:      j.Hidden,
		})
	}
	return rows
}

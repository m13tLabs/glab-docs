package document

import (
	"sort"

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
	IncludeItems      []gitlab.IncludeItem
	JobRows           []jobRow
	Files             files
	SkipVersionFooter bool
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

func getComponentTemplateData(info gitlab.ComponentDocumentationInfo, glabDocsVersion, componentPrefix string, skipVersionFooter bool) (componentTemplateData, error) {
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
		IncludeItems:               info.Includes,
		JobRows:                    getJobRows(info.Jobs),
		Files:                      componentFiles,
		SkipVersionFooter:          skipVersionFooter,
	}, nil
}

func getJobRows(jobs []gitlab.Job) []jobRow {
	rows := make([]jobRow, 0, len(jobs))
	for _, j := range jobs {
		if j.Hidden {
			continue // `.hidden` jobs are templates/anchors, not part of the pipeline
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

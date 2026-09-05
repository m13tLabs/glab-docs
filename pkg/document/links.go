package document

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/m13tLabs/glab-docs/pkg/gitlab"
)

// ComponentLink is where a discovered component's documentation ends up, keyed by the same
// search-root-relative path used as the key of the infoByFile map (e.g. "templates/build.yml").
// It lets other components' Includes tables link to it.
type ComponentLink struct {
	// OutputPath is the rendered doc's path, relative to the component search root.
	OutputPath string
	// Anchor is the Markdown heading anchor for this component's section within OutputPath, or
	// "" when the component has that whole output file to itself.
	Anchor string
}

// GroupComponentsByOutput buckets components by the output file their ComponentDirectory
// resolves to given outputFileName. Several sibling component files (e.g. flat `templates/*.yml`
// files, which all share the "templates" directory) can resolve to the same output path; each
// group is returned sorted by component name so callers render/link them in a stable order.
func GroupComponentsByOutput(infoByFile map[string]gitlab.ComponentDocumentationInfo, outputFileName string) map[string][]string {
	groups := make(map[string][]string)
	for relFile, info := range infoByFile {
		outputPath := filepath.Join(info.ComponentDirectory, outputFileName)
		groups[outputPath] = append(groups[outputPath], relFile)
	}
	for outputPath := range groups {
		group := groups[outputPath]
		sort.Slice(group, func(i, j int) bool {
			return infoByFile[group[i]].Name < infoByFile[group[j]].Name
		})
	}
	return groups
}

// BuildComponentLinks resolves where every component in groups will end up, for Includes tables
// to link to.
func BuildComponentLinks(infoByFile map[string]gitlab.ComponentDocumentationInfo, groups map[string][]string) map[string]ComponentLink {
	links := make(map[string]ComponentLink, len(infoByFile))
	for outputPath, relFiles := range groups {
		combined := len(relFiles) > 1
		for _, relFile := range relFiles {
			anchor := ""
			if combined {
				anchor = markdownAnchor(infoByFile[relFile].Name)
			}
			links[relFile] = ComponentLink{OutputPath: outputPath, Anchor: anchor}
		}
	}
	return links
}

var anchorStripRegex = regexp.MustCompile(`[^\w\- ]`)

// markdownAnchor approximates GitHub/GitLab's heading-to-anchor slugification, close enough for
// linking between sections of a document this tool itself generated.
func markdownAnchor(name string) string {
	s := strings.ToLower(name)
	s = anchorStripRegex.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, " ", "-")
	return s
}

// resolveIncludeLink returns the relative link (with a "#anchor" suffix when the target shares a
// combined doc with something else) that fromRelFile's Includes table should use to point at a
// `local:` include item, or "" when the item isn't a local include or doesn't match a discovered
// component (an include of a file this tool doesn't document, or a `component:`/`project:`/
// `remote:` include, which can't be resolved to a local path).
func resolveIncludeLink(fromRelFile string, item gitlab.IncludeItem, links map[string]ComponentLink) string {
	if item.Kind != "local" {
		return ""
	}

	targetRelFile := filepath.ToSlash(strings.TrimPrefix(item.Location, "/"))
	if targetRelFile == fromRelFile {
		return ""
	}

	target, ok := links[targetRelFile]
	if !ok {
		return ""
	}
	from, ok := links[fromRelFile]
	if !ok {
		return ""
	}

	if from.OutputPath == target.OutputPath {
		if target.Anchor == "" {
			return ""
		}
		return "#" + target.Anchor
	}

	rel, err := filepath.Rel(filepath.Dir(from.OutputPath), target.OutputPath)
	if err != nil {
		return ""
	}
	rel = filepath.ToSlash(rel)
	if target.Anchor != "" {
		rel += "#" + target.Anchor
	}
	return rel
}

package diagram

import (
	"fmt"
	"sort"
	"strings"

	"github.com/olehluchkiv/goifaces/internal/analyzer"
	"github.com/olehluchkiv/goifaces/internal/diagram/split"
)

// Slide represents one navigable page in the slide deck.
type Slide struct {
	Title   string
	Mermaid string
}

// SlideOptions controls slide deck generation.
type SlideOptions struct {
	Threshold int // node count above which slides activate; 0 = always single
}

// DefaultSlideOptions returns sensible defaults.
func DefaultSlideOptions() SlideOptions {
	return SlideOptions{Threshold: 20}
}

// BuildSlides converts analysis result into slides using the provided Splitter.
// Splitting activates when node count >= threshold OR relation count >= threshold
// (a dense graph with many relations benefits from splitting even with fewer nodes).
// Otherwise returns a single slide with the full diagram.
func BuildSlides(result *analyzer.Result, diagOpts DiagramOptions, splitter split.Splitter, opts SlideOptions) []Slide {
	totalNodes := len(result.Interfaces) + len(result.Types)
	totalRelations := len(result.Relations)
	if opts.Threshold > 0 && totalNodes < opts.Threshold && totalRelations < opts.Threshold {
		return []Slide{{
			Title:   "Full Diagram",
			Mermaid: GenerateMermaid(result, diagOpts),
		}}
	}

	var slides []Slide

	// Slide 0: overview — all nodes, no methods
	slides = append(slides, Slide{
		Title:   "Overview",
		Mermaid: generateOverviewMermaid(result, diagOpts),
	})

	// Detail slides from splitter groups
	groups := splitter.Split(result)
	for _, g := range groups {
		sub := subResultForSplitGroup(result, g)
		slides = append(slides, Slide{
			Title:   g.Title,
			Mermaid: GenerateMermaid(sub, diagOpts),
		})
	}

	return slides
}

// subResultForSplitGroup filters a Result to only nodes in a split.Group,
// plus matching relations.
func subResultForSplitGroup(full *analyzer.Result, g split.Group) *analyzer.Result {
	// Build lookup sets from group keys.
	included := make(map[string]bool, len(g.HubKeys)+len(g.SpokeKeys))
	ifaceKeys := make(map[string]bool)
	typeKeys := make(map[string]bool)

	for _, k := range g.HubKeys {
		included[k] = true
		ifaceKeys[k] = true
	}
	for _, k := range g.SpokeKeys {
		included[k] = true
		typeKeys[k] = true
	}

	sub := &analyzer.Result{}

	for i := range full.Interfaces {
		ik := typeKey(full.Interfaces[i].PkgPath, full.Interfaces[i].Name)
		if ifaceKeys[ik] {
			sub.Interfaces = append(sub.Interfaces, full.Interfaces[i])
		}
	}

	for i := range full.Types {
		tk := typeKey(full.Types[i].PkgPath, full.Types[i].Name)
		if typeKeys[tk] {
			sub.Types = append(sub.Types, full.Types[i])
		}
	}

	for _, rel := range full.Relations {
		ik := typeKey(rel.Interface.PkgPath, rel.Interface.Name)
		tk := typeKey(rel.Type.PkgPath, rel.Type.Name)
		if included[ik] && included[tk] {
			sub.Relations = append(sub.Relations, rel)
		}
	}

	return sub
}

// generateOverviewMermaid produces a Mermaid classDiagram with all nodes but empty class bodies
// (only <<interface>> tag, no methods). Includes init directive, classDef styles, cssClass
// assignments, and all relation lines.
func generateOverviewMermaid(result *analyzer.Result, opts DiagramOptions) string {
	var b strings.Builder

	// Sort interfaces deterministically
	ifaces := make([]analyzer.InterfaceDef, len(result.Interfaces))
	copy(ifaces, result.Interfaces)
	sort.Slice(ifaces, func(i, j int) bool {
		if ifaces[i].PkgName != ifaces[j].PkgName {
			return ifaces[i].PkgName < ifaces[j].PkgName
		}
		return ifaces[i].Name < ifaces[j].Name
	})

	// Sort types deterministically
	typs := make([]analyzer.TypeDef, len(result.Types))
	copy(typs, result.Types)
	sort.Slice(typs, func(i, j int) bool {
		if typs[i].PkgName != typs[j].PkgName {
			return typs[i].PkgName < typs[j].PkgName
		}
		return typs[i].Name < typs[j].Name
	})

	// Sort relations deterministically
	rels := make([]analyzer.Relation, len(result.Relations))
	copy(rels, result.Relations)
	sort.Slice(rels, func(i, j int) bool {
		typeKeyI := rels[i].Type.PkgName + "_" + rels[i].Type.Name
		typeKeyJ := rels[j].Type.PkgName + "_" + rels[j].Type.Name
		if typeKeyI != typeKeyJ {
			return typeKeyI < typeKeyJ
		}
		ifaceKeyI := rels[i].Interface.PkgName + "_" + rels[i].Interface.Name
		ifaceKeyJ := rels[j].Interface.PkgName + "_" + rels[j].Interface.Name
		return ifaceKeyI < ifaceKeyJ
	})

	// Header + style definitions
	if opts.IncludeInit {
		b.WriteString("%%{init: {'theme': 'base', 'themeVariables': {'primaryColor': '#ffffff', 'primaryBorderColor': '#cccccc', 'primaryTextColor': '#000000', 'lineColor': '#555555'}}%%\n")
	}
	b.WriteString("classDiagram")
	if len(ifaces) > 0 || len(typs) > 0 {
		b.WriteString("\n")
		b.WriteString("    direction LR\n")
		b.WriteString("    classDef interfaceStyle fill:#2374ab,stroke:#1a5a8a,color:#fff,stroke-width:2px,font-weight:bold\n")
		b.WriteString("    classDef implStyle fill:#4a9c6d,stroke:#357a50,color:#fff,stroke-width:2px")
	}

	// Interface blocks — empty bodies with only <<interface>> tag
	for _, iface := range ifaces {
		id := nodeID(iface.PkgName, iface.Name)
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("    class %s {\n", id))
		b.WriteString("        <<interface>>\n")
		b.WriteString("    }")
	}

	// Type blocks — empty bodies
	if len(ifaces) > 0 && len(typs) > 0 {
		b.WriteString("\n")
	}
	for _, typ := range typs {
		id := nodeID(typ.PkgName, typ.Name)
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("    class %s {\n", id))
		b.WriteString("    }")
	}

	// Relations
	if (len(ifaces) > 0 || len(typs) > 0) && len(rels) > 0 {
		b.WriteString("\n")
	}
	for _, rel := range rels {
		b.WriteString("\n")
		writeRelation(&b, rel)
	}

	// Style assignments
	if len(ifaces) > 0 || len(typs) > 0 {
		b.WriteString("\n")
		for _, iface := range ifaces {
			id := nodeID(iface.PkgName, iface.Name)
			b.WriteString(fmt.Sprintf("\n    cssClass \"%s\" interfaceStyle", id))
		}
		for _, typ := range typs {
			id := nodeID(typ.PkgName, typ.Name)
			b.WriteString(fmt.Sprintf("\n    cssClass \"%s\" implStyle", id))
		}
	}

	return b.String()
}

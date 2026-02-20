package diagram

import (
	"fmt"
	"go/types"
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

// generateOverviewMermaid produces a Mermaid classDiagram showing only interface nodes
// and interface embedding/extension arrows (--|>). Implementation blocks and implementation
// arrows (..|>) are omitted — they appear only on detail slides.
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

	// Build lookup of interfaces by (pkgPath, name) for embedding detection
	type ifaceKey struct{ pkgPath, name string }
	ifaceLookup := make(map[ifaceKey]analyzer.InterfaceDef, len(ifaces))
	for _, iface := range ifaces {
		ifaceLookup[ifaceKey{iface.PkgPath, iface.Name}] = iface
	}

	// Header + style definitions
	if opts.IncludeInit {
		b.WriteString("%%{init: {'theme': 'base', 'themeVariables': {'primaryColor': '#ffffff', 'primaryBorderColor': '#cccccc', 'primaryTextColor': '#000000', 'lineColor': '#555555'}}%%\n")
	}
	b.WriteString("classDiagram")
	if len(ifaces) > 0 {
		b.WriteString("\n")
		b.WriteString("    direction LR\n")
		b.WriteString("    classDef interfaceStyle fill:#2374ab,stroke:#1a5a8a,color:#fff,stroke-width:2px,font-weight:bold")
	}

	// Interface blocks — empty bodies with only <<interface>> tag
	for _, iface := range ifaces {
		id := nodeID(iface.PkgName, iface.Name)
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("    class %s {\n", id))
		b.WriteString("        <<interface>>\n")
		b.WriteString("    }")
	}

	// Embedding relations: interface --|> interface (solid + triangle for inheritance)
	var embeddings []string
	for _, iface := range ifaces {
		if iface.TypeObj == nil {
			continue
		}
		childID := nodeID(iface.PkgName, iface.Name)
		for i := 0; i < iface.TypeObj.NumEmbeddeds(); i++ {
			embedded := iface.TypeObj.EmbeddedType(i)
			named, ok := embedded.(*types.Named)
			if !ok {
				continue
			}
			obj := named.Obj()
			if obj.Pkg() == nil {
				continue // universe-scope type like error
			}
			parent, exists := ifaceLookup[ifaceKey{obj.Pkg().Path(), obj.Name()}]
			if !exists {
				continue // embedded interface not in our result set
			}
			parentID := nodeID(parent.PkgName, parent.Name)
			if childID == parentID {
				continue // guard against self-embedding
			}
			embeddings = append(embeddings, fmt.Sprintf("    %s --|> %s", childID, parentID))
		}
	}
	sort.Strings(embeddings)

	if len(ifaces) > 0 && len(embeddings) > 0 {
		b.WriteString("\n")
	}
	for _, emb := range embeddings {
		b.WriteString("\n")
		b.WriteString(emb)
	}

	// Style assignments
	if len(ifaces) > 0 {
		b.WriteString("\n")
		for _, iface := range ifaces {
			id := nodeID(iface.PkgName, iface.Name)
			b.WriteString(fmt.Sprintf("\n    cssClass \"%s\" interfaceStyle", id))
		}
	}

	return b.String()
}

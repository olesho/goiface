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

	// Slide 0: package map — shows repository package hierarchy
	slides = append(slides, Slide{
		Title:   "Package Map",
		Mermaid: generatePackageMapMermaid(result, diagOpts),
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

	// Post-filter: remove interfaces with no relations on this slide.
	// Hub interfaces are replicated onto every group but may have no
	// implementing type present, leaving orphaned nodes.
	usedIfaces := make(map[string]bool, len(sub.Relations))
	for _, rel := range sub.Relations {
		ik := typeKey(rel.Interface.PkgPath, rel.Interface.Name)
		usedIfaces[ik] = true
	}
	filtered := sub.Interfaces[:0]
	for _, iface := range sub.Interfaces {
		ik := typeKey(iface.PkgPath, iface.Name)
		if usedIfaces[ik] {
			filtered = append(filtered, iface)
		}
	}
	sub.Interfaces = filtered

	return sub
}

// pkgStats holds per-package counts for the package map.
type pkgStats struct {
	Interfaces int
	Types      int
}

// generatePackageMapMermaid produces a Mermaid flowchart showing the repository's
// package hierarchy. Each package is a node displaying its name and counts of
// interfaces and types. Packages with subpackages are rendered as subgraphs.
func generatePackageMapMermaid(result *analyzer.Result, opts DiagramOptions) string {
	// Collect stats per package path
	stats := make(map[string]*pkgStats)
	for _, iface := range result.Interfaces {
		s, ok := stats[iface.PkgPath]
		if !ok {
			s = &pkgStats{}
			stats[iface.PkgPath] = s
		}
		s.Interfaces++
	}
	for _, typ := range result.Types {
		s, ok := stats[typ.PkgPath]
		if !ok {
			s = &pkgStats{}
			stats[typ.PkgPath] = s
		}
		s.Types++
	}

	if len(stats) == 0 {
		return "flowchart LR"
	}

	// Collect and sort package paths
	var paths []string
	for p := range stats {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	// Find common prefix to strip (module path)
	prefix := longestCommonPrefix(paths)
	// Trim to last slash to get module root
	if idx := strings.LastIndex(prefix, "/"); idx >= 0 {
		prefix = prefix[:idx+1]
	}

	// Build tree
	root := &pkgNode{children: make(map[string]*pkgNode)}
	for _, p := range paths {
		rel := strings.TrimPrefix(p, prefix)
		if rel == "" {
			rel = lastSegment(p)
		}
		parts := strings.Split(rel, "/")
		insertNode(root, parts, p, stats[p])
	}

	var b strings.Builder
	if opts.IncludeInit {
		b.WriteString("%%{init: {'theme': 'base', 'themeVariables': {'primaryColor': '#ffffff', 'primaryBorderColor': '#cccccc', 'primaryTextColor': '#000000', 'lineColor': '#555555'}}}%%\n")
	}
	b.WriteString("flowchart LR")

	renderTree(&b, root, 1)

	return b.String()
}

// pkgNode represents a node in the package hierarchy tree.
type pkgNode struct {
	name     string    // segment name (e.g. "api")
	pkgPath  string    // full package path (only set for leaf/actual packages)
	stats    *pkgStats // non-nil for actual packages
	children map[string]*pkgNode
}

func insertNode(parent *pkgNode, parts []string, fullPath string, s *pkgStats) {
	if len(parts) == 0 {
		return
	}
	name := parts[0]
	child, ok := parent.children[name]
	if !ok {
		child = &pkgNode{name: name, children: make(map[string]*pkgNode)}
		parent.children[name] = child
	}
	if len(parts) == 1 {
		child.pkgPath = fullPath
		child.stats = s
	} else {
		insertNode(child, parts[1:], fullPath, s)
	}
}

func renderTree(b *strings.Builder, node *pkgNode, depth int) {
	// Sort children for deterministic output
	var names []string
	for name := range node.children {
		names = append(names, name)
	}
	sort.Strings(names)

	indent := strings.Repeat("    ", depth)

	for _, name := range names {
		child := node.children[name]
		id := sanitizeID(child.pkgPath)
		if id == "" {
			id = "pkg_" + sanitizeID(name)
		}

		hasChildren := len(child.children) > 0

		if hasChildren {
			// Render as subgraph with nested children
			b.WriteString(fmt.Sprintf("\n%ssubgraph %s[\"%s\"]", indent, id, name))

			// If this node itself is a package (has stats), add a summary node inside
			if child.stats != nil {
				innerID := id + "__self"
				label := formatPkgLabel(name, child.stats)
				b.WriteString(fmt.Sprintf("\n%s    %s[\"%s\"]", indent, innerID, label))
			}

			renderTree(b, child, depth+1)
			b.WriteString(fmt.Sprintf("\n%send", indent))
		} else {
			// Leaf node
			label := formatPkgLabel(name, child.stats)
			b.WriteString(fmt.Sprintf("\n%s%s[\"%s\"]", indent, id, label))
		}
	}
}

func formatPkgLabel(name string, s *pkgStats) string {
	if s == nil {
		return name
	}
	var parts []string
	if s.Interfaces > 0 {
		parts = append(parts, fmt.Sprintf("%d ifaces", s.Interfaces))
	}
	if s.Types > 0 {
		parts = append(parts, fmt.Sprintf("%d types", s.Types))
	}
	if len(parts) == 0 {
		return name
	}
	return fmt.Sprintf("%s<br/>%s", name, strings.Join(parts, ", "))
}

func longestCommonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	prefix := strs[0]
	for _, s := range strs[1:] {
		for !strings.HasPrefix(s, prefix) {
			prefix = prefix[:len(prefix)-1]
			if prefix == "" {
				return ""
			}
		}
	}
	return prefix
}

func lastSegment(path string) string {
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		return path[idx+1:]
	}
	return path
}

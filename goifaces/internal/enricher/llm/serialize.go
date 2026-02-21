package llm

import (
	"fmt"
	"sort"
	"strings"

	"github.com/olehluchkiv/goifaces/internal/analyzer"
)

// maxMethodsPerNode is the max methods to include per interface/type in prompts.
const maxMethodsPerNode = 10

// SerializeResult converts an analyzer.Result into a compact text representation
// suitable for LLM prompts. It keeps prompt size manageable by truncating methods
// and focusing on the most connected nodes for large projects.
func SerializeResult(result *analyzer.Result) string {
	var b strings.Builder

	// For large projects, pre-filter to most-connected nodes
	ifaces := result.Interfaces
	types := result.Types
	relations := result.Relations

	if len(ifaces)+len(types) > 100 {
		ifaces, types, relations = filterTopNodes(result, 100)
	}

	// Interfaces
	b.WriteString("INTERFACES:\n")
	for _, iface := range ifaces {
		key := nodeKey(iface.PkgPath, iface.Name)
		b.WriteString(fmt.Sprintf("  %s (package: %s)\n", key, iface.PkgName))
		methods := iface.Methods
		if len(methods) > maxMethodsPerNode {
			methods = methods[:maxMethodsPerNode]
			b.WriteString(fmt.Sprintf("    methods (%d shown of %d):\n", maxMethodsPerNode, len(iface.Methods)))
		} else if len(methods) > 0 {
			b.WriteString("    methods:\n")
		}
		for _, m := range methods {
			b.WriteString(fmt.Sprintf("      - %s%s\n", m.Name, m.Signature))
		}
	}

	// Types
	b.WriteString("\nTYPES:\n")
	for _, typ := range types {
		key := nodeKey(typ.PkgPath, typ.Name)
		kind := "type"
		if typ.IsStruct {
			kind = "struct"
		}
		b.WriteString(fmt.Sprintf("  %s (%s, package: %s)\n", key, kind, typ.PkgName))
		methods := typ.Methods
		if len(methods) > maxMethodsPerNode {
			methods = methods[:maxMethodsPerNode]
			b.WriteString(fmt.Sprintf("    methods (%d shown of %d):\n", maxMethodsPerNode, len(typ.Methods)))
		} else if len(methods) > 0 {
			b.WriteString("    methods:\n")
		}
		for _, m := range methods {
			b.WriteString(fmt.Sprintf("      - %s%s\n", m.Name, m.Signature))
		}
	}

	// Relations
	b.WriteString("\nRELATIONSHIPS:\n")
	for i, rel := range relations {
		tKey := nodeKey(rel.Type.PkgPath, rel.Type.Name)
		iKey := nodeKey(rel.Interface.PkgPath, rel.Interface.Name)
		ptr := ""
		if rel.ViaPointer {
			ptr = " (pointer receiver)"
		}
		b.WriteString(fmt.Sprintf("  [%d] %s implements %s%s\n", i, tKey, iKey, ptr))
	}

	return b.String()
}

// SerializeRelations converts relations into a compact indexed list for scoring prompts.
func SerializeRelations(relations []analyzer.Relation) string {
	var b strings.Builder
	for i, rel := range relations {
		tKey := nodeKey(rel.Type.PkgPath, rel.Type.Name)
		iKey := nodeKey(rel.Interface.PkgPath, rel.Interface.Name)
		b.WriteString(fmt.Sprintf("[%d] %s implements %s\n", i, tKey, iKey))
	}
	return b.String()
}

// SerializeNodeList converts interfaces and types into a compact keyed list.
func SerializeNodeList(result *analyzer.Result) string {
	var b strings.Builder
	for _, iface := range result.Interfaces {
		b.WriteString(fmt.Sprintf("  IFACE %s\n", nodeKey(iface.PkgPath, iface.Name)))
	}
	for _, typ := range result.Types {
		b.WriteString(fmt.Sprintf("  TYPE  %s\n", nodeKey(typ.PkgPath, typ.Name)))
	}
	return b.String()
}

func nodeKey(pkgPath, name string) string {
	return pkgPath + "." + name
}

// filterTopNodes returns the top N most-connected nodes and their relations.
func filterTopNodes(result *analyzer.Result, maxNodes int) ([]analyzer.InterfaceDef, []analyzer.TypeDef, []analyzer.Relation) {
	edgeCount := make(map[string]int)
	for _, rel := range result.Relations {
		edgeCount[nodeKey(rel.Type.PkgPath, rel.Type.Name)]++
		edgeCount[nodeKey(rel.Interface.PkgPath, rel.Interface.Name)]++
	}

	type ranked struct {
		key   string
		count int
	}
	var ranks []ranked
	for k, c := range edgeCount {
		ranks = append(ranks, ranked{k, c})
	}
	sort.Slice(ranks, func(i, j int) bool {
		return ranks[i].count > ranks[j].count
	})

	keep := make(map[string]bool)
	for i, r := range ranks {
		if i >= maxNodes {
			break
		}
		keep[r.key] = true
	}

	var ifaces []analyzer.InterfaceDef
	for _, iface := range result.Interfaces {
		if keep[nodeKey(iface.PkgPath, iface.Name)] {
			ifaces = append(ifaces, iface)
		}
	}

	var types []analyzer.TypeDef
	for _, typ := range result.Types {
		if keep[nodeKey(typ.PkgPath, typ.Name)] {
			types = append(types, typ)
		}
	}

	var rels []analyzer.Relation
	for _, rel := range result.Relations {
		tKey := nodeKey(rel.Type.PkgPath, rel.Type.Name)
		iKey := nodeKey(rel.Interface.PkgPath, rel.Interface.Name)
		if keep[tKey] && keep[iKey] {
			rels = append(rels, rel)
		}
	}

	return ifaces, types, rels
}

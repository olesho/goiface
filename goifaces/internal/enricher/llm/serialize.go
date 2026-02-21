package llm

import (
	"fmt"
	"sort"
	"strings"

	"github.com/olehluchkiv/goifaces/internal/analyzer"
)

// SerializeResult converts an analyzer.Result into a compact text representation
// suitable for LLM prompts. It keeps the output concise to stay within token budgets.
func SerializeResult(result *analyzer.Result) string {
	var b strings.Builder

	b.WriteString("## Interfaces\n")
	for _, iface := range result.Interfaces {
		key := iface.PkgPath + "." + iface.Name
		fmt.Fprintf(&b, "- %s", key)
		if len(iface.Methods) > 0 {
			methods := formatMethods(iface.Methods, 10)
			fmt.Fprintf(&b, " [%s]", methods)
		}
		b.WriteByte('\n')
	}

	b.WriteString("\n## Types\n")
	for _, typ := range result.Types {
		key := typ.PkgPath + "." + typ.Name
		kind := "type"
		if typ.IsStruct {
			kind = "struct"
		}
		fmt.Fprintf(&b, "- %s (%s)", key, kind)
		if len(typ.Methods) > 0 {
			methods := formatMethods(typ.Methods, 10)
			fmt.Fprintf(&b, " [%s]", methods)
		}
		b.WriteByte('\n')
	}

	b.WriteString("\n## Relationships\n")
	for i, rel := range result.Relations {
		tKey := rel.Type.PkgPath + "." + rel.Type.Name
		iKey := rel.Interface.PkgPath + "." + rel.Interface.Name
		ptr := ""
		if rel.ViaPointer {
			ptr = " (via pointer)"
		}
		fmt.Fprintf(&b, "%d: %s implements %s%s\n", i, tKey, iKey, ptr)
	}

	return b.String()
}

// SerializeInterfaces returns a compact list of interfaces for prompts.
func SerializeInterfaces(result *analyzer.Result) string {
	var b strings.Builder
	for _, iface := range result.Interfaces {
		key := iface.PkgPath + "." + iface.Name
		fmt.Fprintf(&b, "- %s", key)
		if len(iface.Methods) > 0 {
			methods := formatMethods(iface.Methods, 10)
			fmt.Fprintf(&b, " [%s]", methods)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// SerializeRelations returns a compact list of relationships for prompts.
func SerializeRelations(result *analyzer.Result) string {
	var b strings.Builder
	for i, rel := range result.Relations {
		tKey := rel.Type.PkgPath + "." + rel.Type.Name
		iKey := rel.Interface.PkgPath + "." + rel.Interface.Name
		fmt.Fprintf(&b, "%d: %s implements %s\n", i, tKey, iKey)
	}
	return b.String()
}

// SerializeNodeList returns a compact list of all node keys for prompts.
func SerializeNodeList(result *analyzer.Result) string {
	var b strings.Builder
	for _, iface := range result.Interfaces {
		fmt.Fprintf(&b, "- %s.%s (interface)\n", iface.PkgPath, iface.Name)
	}
	for _, typ := range result.Types {
		fmt.Fprintf(&b, "- %s.%s (type)\n", typ.PkgPath, typ.Name)
	}
	return b.String()
}

// PreFilterByEdgeCount returns a result with only the top N most-connected nodes.
// Used to keep prompts within token budgets for large projects.
func PreFilterByEdgeCount(result *analyzer.Result, maxNodes int) *analyzer.Result {
	totalNodes := len(result.Interfaces) + len(result.Types)
	if totalNodes <= maxNodes {
		return result
	}

	edgeCount := make(map[string]int)
	for _, iface := range result.Interfaces {
		edgeCount[iface.PkgPath+"."+iface.Name] = 0
	}
	for _, typ := range result.Types {
		edgeCount[typ.PkgPath+"."+typ.Name] = 0
	}
	for _, rel := range result.Relations {
		tKey := rel.Type.PkgPath + "." + rel.Type.Name
		iKey := rel.Interface.PkgPath + "." + rel.Interface.Name
		edgeCount[tKey]++
		edgeCount[iKey]++
	}

	type nodeRank struct {
		key   string
		count int
	}
	var ranks []nodeRank
	for k, c := range edgeCount {
		ranks = append(ranks, nodeRank{k, c})
	}
	sort.Slice(ranks, func(i, j int) bool {
		if ranks[i].count != ranks[j].count {
			return ranks[i].count > ranks[j].count
		}
		return ranks[i].key < ranks[j].key
	})

	keep := make(map[string]bool)
	for i, r := range ranks {
		if i >= maxNodes {
			break
		}
		keep[r.key] = true
	}

	out := &analyzer.Result{}
	for _, iface := range result.Interfaces {
		if keep[iface.PkgPath+"."+iface.Name] {
			out.Interfaces = append(out.Interfaces, iface)
		}
	}
	for _, typ := range result.Types {
		if keep[typ.PkgPath+"."+typ.Name] {
			out.Types = append(out.Types, typ)
		}
	}
	for _, rel := range result.Relations {
		tKey := rel.Type.PkgPath + "." + rel.Type.Name
		iKey := rel.Interface.PkgPath + "." + rel.Interface.Name
		if keep[tKey] && keep[iKey] {
			out.Relations = append(out.Relations, rel)
		}
	}
	return out
}

func formatMethods(methods []analyzer.MethodSig, max int) string {
	names := make([]string, 0, len(methods))
	for i, m := range methods {
		if i >= max {
			names = append(names, fmt.Sprintf("...+%d more", len(methods)-max))
			break
		}
		names = append(names, m.Name+m.Signature)
	}
	return strings.Join(names, ", ")
}

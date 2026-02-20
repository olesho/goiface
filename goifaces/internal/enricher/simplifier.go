package enricher

import (
	"sort"

	"github.com/olehluchkiv/goifaces/internal/analyzer"
)

// Simplifier reduces a large diagram to its most important elements.
type Simplifier interface {
	Simplify(result *analyzer.Result, maxNodes int) *analyzer.Result
}

// DefaultSimplifier prunes orphans and caps at MaxNodes by edge count.
type DefaultSimplifier struct {
	MaxNodes int // 0 means no cap
}

func NewDefaultSimplifier() *DefaultSimplifier {
	return &DefaultSimplifier{MaxNodes: 0}
}

func (s *DefaultSimplifier) Enrich(result *analyzer.Result) *analyzer.Result {
	if s.MaxNodes <= 0 {
		return result
	}
	return s.Simplify(result, s.MaxNodes)
}

func (s *DefaultSimplifier) Simplify(result *analyzer.Result, maxNodes int) *analyzer.Result {
	// Count edges per node
	edgeCount := make(map[string]int)
	for _, rel := range result.Relations {
		tKey := rel.Type.PkgPath + "." + rel.Type.Name
		iKey := rel.Interface.PkgPath + "." + rel.Interface.Name
		edgeCount[tKey]++
		edgeCount[iKey]++
	}

	// If total nodes already within cap, return as-is
	totalNodes := len(result.Interfaces) + len(result.Types)
	if totalNodes <= maxNodes {
		return result
	}

	// Rank nodes by edge count, keep top N
	type nodeRank struct {
		key   string
		count int
	}
	var ranks []nodeRank
	for k, c := range edgeCount {
		ranks = append(ranks, nodeRank{k, c})
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

	// Filter
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

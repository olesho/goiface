package enricher

import "github.com/olehluchkiv/goifaces/internal/analyzer"

// Grouper assigns architectural roles to clusters of interfaces/types.
type Grouper interface {
	Group(result *analyzer.Result) []SemanticGroup
}

// DefaultGrouper groups by package name (mechanical).
type DefaultGrouper struct{}

func NewDefaultGrouper() *DefaultGrouper { return &DefaultGrouper{} }

func (g *DefaultGrouper) Enrich(result *analyzer.Result) *analyzer.Result {
	// Code default: no transformation, grouping is handled in diagram generation by package
	return result
}

func (g *DefaultGrouper) Group(result *analyzer.Result) []SemanticGroup {
	groups := make(map[string]*SemanticGroup)
	for _, iface := range result.Interfaces {
		sg, ok := groups[iface.PkgName]
		if !ok {
			sg = &SemanticGroup{Name: iface.PkgName}
			groups[iface.PkgName] = sg
		}
		sg.Interfaces = append(sg.Interfaces, iface.PkgPath+"."+iface.Name)
	}
	for _, typ := range result.Types {
		sg, ok := groups[typ.PkgName]
		if !ok {
			sg = &SemanticGroup{Name: typ.PkgName}
			groups[typ.PkgName] = sg
		}
		sg.Types = append(sg.Types, typ.PkgPath+"."+typ.Name)
	}
	var out []SemanticGroup
	for _, sg := range groups {
		out = append(out, *sg)
	}
	return out
}

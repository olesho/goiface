package diagram

import (
	"sort"

	"github.com/olehluchkiv/goifaces/internal/analyzer"
)

// InteractiveInterface holds pre-computed data for a single interface in the interactive UI.
type InteractiveInterface struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	PkgName    string   `json:"pkgName"`
	Methods    []string `json:"methods"`
	SourceFile string   `json:"sourceFile,omitempty"`
}

// InteractiveType holds pre-computed data for a single implementation type in the interactive UI.
type InteractiveType struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	PkgName    string `json:"pkgName"`
	SourceFile string `json:"sourceFile,omitempty"`
}

// InteractiveRelation maps a type to an interface it implements.
type InteractiveRelation struct {
	TypeID      string `json:"typeId"`
	InterfaceID string `json:"interfaceId"`
}

// PackageMapNode represents a node in the package hierarchy for the HTML treemap.
type PackageMapNode struct {
	Name       string            `json:"name"`
	RelPath    string            `json:"relPath"`
	PkgPath    string            `json:"pkgPath"`
	Interfaces int               `json:"interfaces"`
	Types      int               `json:"types"`
	Value      int               `json:"value"`
	Children   []*PackageMapNode `json:"children,omitempty"`
}

// InteractiveData holds all data needed for the interactive tabbed UI.
type InteractiveData struct {
	PackageMapNodes []*PackageMapNode      `json:"packageMapNodes,omitempty"`
	Interfaces      []InteractiveInterface `json:"interfaces"`
	Types           []InteractiveType      `json:"types"`
	Relations       []InteractiveRelation  `json:"relations"`
	RepoAddress     string                 `json:"repoAddress"`
}

// PrepareInteractiveData converts an analyzer.Result into the data structure
// needed by the interactive server template. It computes sanitized node IDs
// and method signatures.
func PrepareInteractiveData(result *analyzer.Result, opts DiagramOptions) InteractiveData {
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

	// Build interactive interfaces
	interactiveIfaces := make([]InteractiveInterface, len(ifaces))
	for i, iface := range ifaces {
		limit := len(iface.Methods)
		if opts.MaxMethodsPerBox > 0 && limit > opts.MaxMethodsPerBox {
			limit = opts.MaxMethodsPerBox
		}
		methods := make([]string, limit)
		for j := 0; j < limit; j++ {
			methods[j] = SanitizeSignature(iface.Methods[j].Signature)
		}
		interactiveIfaces[i] = InteractiveInterface{
			ID:         NodeID(iface.PkgName, iface.Name),
			Name:       iface.PkgName + "." + iface.Name,
			PkgName:    iface.PkgName,
			Methods:    methods,
			SourceFile: iface.SourceFile,
		}
	}

	// Build interactive types
	interactiveTypes := make([]InteractiveType, len(typs))
	for i, typ := range typs {
		interactiveTypes[i] = InteractiveType{
			ID:         NodeID(typ.PkgName, typ.Name),
			Name:       typ.PkgName + "." + typ.Name,
			PkgName:    typ.PkgName,
			SourceFile: typ.SourceFile,
		}
	}

	// Build interactive relations
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

	interactiveRels := make([]InteractiveRelation, len(rels))
	for i, rel := range rels {
		interactiveRels[i] = InteractiveRelation{
			TypeID:      NodeID(rel.Type.PkgName, rel.Type.Name),
			InterfaceID: NodeID(rel.Interface.PkgName, rel.Interface.Name),
		}
	}

	return InteractiveData{
		Interfaces: interactiveIfaces,
		Types:      interactiveTypes,
		Relations:  interactiveRels,
	}
}

// FilterBySelection filters an analyzer.Result to include only the selected
// types and interfaces, plus any items directly related to them via
// implementation relations. This mirrors the client-side JS filtering logic
// and is used for testing.
func FilterBySelection(result *analyzer.Result, selectedTypeIDs, selectedIfaceIDs []string) *analyzer.Result {
	// Build lookup sets for selected IDs
	selTypes := make(map[string]bool, len(selectedTypeIDs))
	for _, id := range selectedTypeIDs {
		selTypes[id] = true
	}
	selIfaces := make(map[string]bool, len(selectedIfaceIDs))
	for _, id := range selectedIfaceIDs {
		selIfaces[id] = true
	}

	// Find all relations involving selected items, and collect related IDs
	relatedTypes := make(map[string]bool)
	relatedIfaces := make(map[string]bool)
	var filteredRels []analyzer.Relation

	for _, rel := range result.Relations {
		typeID := NodeID(rel.Type.PkgName, rel.Type.Name)
		ifaceID := NodeID(rel.Interface.PkgName, rel.Interface.Name)

		// Include relation if either side is selected
		if selTypes[typeID] || selIfaces[ifaceID] {
			filteredRels = append(filteredRels, rel)
			relatedTypes[typeID] = true
			relatedIfaces[ifaceID] = true
		}
	}

	// Collect all IDs that should be in the result:
	// selected items + items related to them via surviving relations
	includeTypes := make(map[string]bool)
	includeIfaces := make(map[string]bool)
	for id := range selTypes {
		includeTypes[id] = true
	}
	for id := range selIfaces {
		includeIfaces[id] = true
	}
	for id := range relatedTypes {
		includeTypes[id] = true
	}
	for id := range relatedIfaces {
		includeIfaces[id] = true
	}

	// Filter interfaces
	var filteredIfaces []analyzer.InterfaceDef
	for _, iface := range result.Interfaces {
		if includeIfaces[NodeID(iface.PkgName, iface.Name)] {
			filteredIfaces = append(filteredIfaces, iface)
		}
	}

	// Filter types
	var filteredTypes []analyzer.TypeDef
	for _, typ := range result.Types {
		if includeTypes[NodeID(typ.PkgName, typ.Name)] {
			filteredTypes = append(filteredTypes, typ)
		}
	}

	return &analyzer.Result{
		Interfaces: filteredIfaces,
		Types:      filteredTypes,
		Relations:  filteredRels,
		ModulePath: result.ModulePath,
	}
}

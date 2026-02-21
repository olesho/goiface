package diagram

import (
	"sort"

	"github.com/olehluchkiv/goifaces/internal/analyzer"
)

// InteractiveInterface holds prepared data for an interface in the interactive UI.
type InteractiveInterface struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	PkgName    string   `json:"pkgName"`
	Methods    []string `json:"methods"`
	SourceFile string   `json:"sourceFile"`
}

// InteractiveType holds prepared data for an implementation type in the interactive UI.
type InteractiveType struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	PkgName    string `json:"pkgName"`
	SourceFile string `json:"sourceFile"`
}

// InteractiveRelation holds a type→interface implementation relation.
type InteractiveRelation struct {
	TypeID      string `json:"typeId"`
	InterfaceID string `json:"interfaceId"`
}

// InteractiveData holds all data needed for the interactive tabbed UI.
type InteractiveData struct {
	PackageMapMermaid string                 `json:"packageMapMermaid"`
	Interfaces        []InteractiveInterface `json:"interfaces"`
	Types             []InteractiveType      `json:"types"`
	Relations         []InteractiveRelation  `json:"relations"`
	RepoAddress       string                 `json:"repoAddress"`
}

// PrepareInteractiveData converts an analyzer.Result into the data structure
// the interactive server needs, computing sanitized node IDs and method signatures.
func PrepareInteractiveData(result *analyzer.Result, opts DiagramOptions) InteractiveData {
	// Sort interfaces deterministically by (pkgName, name).
	ifaces := make([]analyzer.InterfaceDef, len(result.Interfaces))
	copy(ifaces, result.Interfaces)
	sort.Slice(ifaces, func(i, j int) bool {
		if ifaces[i].PkgName != ifaces[j].PkgName {
			return ifaces[i].PkgName < ifaces[j].PkgName
		}
		return ifaces[i].Name < ifaces[j].Name
	})

	// Sort types deterministically by (pkgName, name).
	typs := make([]analyzer.TypeDef, len(result.Types))
	copy(typs, result.Types)
	sort.Slice(typs, func(i, j int) bool {
		if typs[i].PkgName != typs[j].PkgName {
			return typs[i].PkgName < typs[j].PkgName
		}
		return typs[i].Name < typs[j].Name
	})

	var interactiveIfaces []InteractiveInterface
	for _, iface := range ifaces {
		var methods []string
		limit := len(iface.Methods)
		if opts.MaxMethodsPerBox > 0 && limit > opts.MaxMethodsPerBox {
			limit = opts.MaxMethodsPerBox
		}
		for i := 0; i < limit; i++ {
			methods = append(methods, SanitizeSignature(iface.Methods[i].Signature))
		}
		interactiveIfaces = append(interactiveIfaces, InteractiveInterface{
			ID:         NodeID(iface.PkgName, iface.Name),
			Name:       iface.Name,
			PkgName:    iface.PkgName,
			Methods:    methods,
			SourceFile: iface.SourceFile,
		})
	}

	var interactiveTypes []InteractiveType
	for _, typ := range typs {
		interactiveTypes = append(interactiveTypes, InteractiveType{
			ID:         NodeID(typ.PkgName, typ.Name),
			Name:       typ.Name,
			PkgName:    typ.PkgName,
			SourceFile: typ.SourceFile,
		})
	}

	// Sort relations deterministically.
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

	var interactiveRels []InteractiveRelation
	for _, rel := range rels {
		interactiveRels = append(interactiveRels, InteractiveRelation{
			TypeID:      NodeID(rel.Type.PkgName, rel.Type.Name),
			InterfaceID: NodeID(rel.Interface.PkgName, rel.Interface.Name),
		})
	}

	return InteractiveData{
		Interfaces: interactiveIfaces,
		Types:      interactiveTypes,
		Relations:  interactiveRels,
	}
}

// FilterBySelection filters a Result to only include selected items and their
// direct relations. This is the Go-side equivalent of the client-side JS filtering,
// used for testing.
func FilterBySelection(result *analyzer.Result, selectedTypeIDs, selectedIfaceIDs []string) *analyzer.Result {
	typeIDSet := make(map[string]bool, len(selectedTypeIDs))
	for _, id := range selectedTypeIDs {
		typeIDSet[id] = true
	}
	ifaceIDSet := make(map[string]bool, len(selectedIfaceIDs))
	for _, id := range selectedIfaceIDs {
		ifaceIDSet[id] = true
	}

	// Find all relations involving selected items.
	// A relation is included if its type OR its interface is selected.
	relatedTypeIDs := make(map[string]bool)
	relatedIfaceIDs := make(map[string]bool)
	var filteredRels []analyzer.Relation

	for _, rel := range result.Relations {
		tID := NodeID(rel.Type.PkgName, rel.Type.Name)
		iID := NodeID(rel.Interface.PkgName, rel.Interface.Name)

		if typeIDSet[tID] || ifaceIDSet[iID] {
			filteredRels = append(filteredRels, rel)
			relatedTypeIDs[tID] = true
			relatedIfaceIDs[iID] = true
		}
	}

	// Include selected items + items brought in through relations.
	var filteredIfaces []analyzer.InterfaceDef
	for _, iface := range result.Interfaces {
		id := NodeID(iface.PkgName, iface.Name)
		if ifaceIDSet[id] || relatedIfaceIDs[id] {
			filteredIfaces = append(filteredIfaces, iface)
		}
	}

	var filteredTypes []analyzer.TypeDef
	for _, typ := range result.Types {
		id := NodeID(typ.PkgName, typ.Name)
		if typeIDSet[id] || relatedTypeIDs[id] {
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

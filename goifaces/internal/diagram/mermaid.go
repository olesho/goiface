package diagram

import (
	"fmt"
	"sort"
	"strings"

	"github.com/olehluchkiv/goifaces/internal/analyzer"
)

// DiagramOptions controls Mermaid diagram generation.
type DiagramOptions struct {
	MaxMethodsPerBox int  // default 5, 0 means unlimited
	IncludeInit      bool // include %%{init:}%% directive (for standalone .mmd files)
}

// DefaultDiagramOptions returns sensible defaults for diagram generation.
func DefaultDiagramOptions() DiagramOptions {
	return DiagramOptions{MaxMethodsPerBox: 5}
}

// GenerateMermaid produces a Mermaid classDiagram string from analysis results.
func GenerateMermaid(result *analyzer.Result, opts DiagramOptions) string {
	var b strings.Builder

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

	// Sort relations deterministically by (type name, interface name).
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

	// Header + style definitions.
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

	// Interfaces section.
	for _, iface := range ifaces {
		b.WriteString("\n")
		writeInterfaceBlock(&b, iface, opts)
	}

	// Types section (separated by blank line from interfaces if both exist).
	if len(ifaces) > 0 && len(typs) > 0 {
		b.WriteString("\n")
	}
	for _, typ := range typs {
		b.WriteString("\n")
		writeTypeBlock(&b, typ)
	}

	// Relations section (separated by blank line from types if both exist).
	if (len(ifaces) > 0 || len(typs) > 0) && len(rels) > 0 {
		b.WriteString("\n")
	}
	for _, rel := range rels {
		b.WriteString("\n")
		writeRelation(&b, rel)
	}

	// Style assignments section.
	if len(ifaces) > 0 || len(typs) > 0 {
		b.WriteString("\n")
		for _, iface := range ifaces {
			id := NodeID(iface.PkgName, iface.Name)
			b.WriteString(fmt.Sprintf("\n    cssClass \"%s\" interfaceStyle", id))
		}
		for _, typ := range typs {
			id := NodeID(typ.PkgName, typ.Name)
			b.WriteString(fmt.Sprintf("\n    cssClass \"%s\" implStyle", id))
		}
	}

	return b.String()
}

// SanitizeSignature removes characters in method signatures that break Mermaid syntax.
// Mermaid treats {}, <>, and ~ as special in class diagram labels.
// Uses only ASCII-safe replacements that work in both mmdc CLI and browser Mermaid.js.
func SanitizeSignature(sig string) string {
	// Replace <-chan with chan (drop direction indicator — Mermaid can't handle <).
	sig = strings.ReplaceAll(sig, "<-chan", "chan")
	// Replace interface{} with "any" BEFORE stripping braces — bare "interface"
	// is a reserved keyword in browser Mermaid.js (<<interface>> tag parsing).
	sig = strings.ReplaceAll(sig, "interface{}", "any")
	// Strip remaining empty braces — in Go signatures these are empty type literals
	// like struct{}, map[K]struct{}.
	sig = strings.ReplaceAll(sig, "{}", "")
	return sig
}

// sanitizeID replaces /, ., - with _ in node identifiers.
func sanitizeID(s string) string {
	r := strings.NewReplacer("/", "_", ".", "_", "-", "_")
	return r.Replace(s)
}

// NodeID builds a sanitized node ID from pkgName and type/interface name.
func NodeID(pkgName, name string) string {
	return sanitizeID(pkgName + "_" + name)
}

// typeKey builds a unique key for a type from its package path and name.
func typeKey(pkgPath, name string) string {
	return pkgPath + "." + name
}

// writeInterfaceBlock writes a Mermaid class block for an interface.
func writeInterfaceBlock(b *strings.Builder, iface analyzer.InterfaceDef, opts DiagramOptions) {
	id := NodeID(iface.PkgName, iface.Name)
	b.WriteString(fmt.Sprintf("    class %s {\n", id))
	b.WriteString("        <<interface>>\n")
	if iface.SourceFile != "" {
		b.WriteString("        %% file: " + iface.SourceFile + "\n")
	}
	writeMethodLines(b, iface.Methods, opts)
	b.WriteString("    }")
}

// writeTypeBlock writes a Mermaid class block for a concrete type.
// Only the type name is shown — methods are omitted because they're
// already listed in the interface blocks this type implements.
func writeTypeBlock(b *strings.Builder, typ analyzer.TypeDef) {
	id := NodeID(typ.PkgName, typ.Name)
	b.WriteString(fmt.Sprintf("    class %s {\n", id))
	if typ.SourceFile != "" {
		b.WriteString("        %% file: " + typ.SourceFile + "\n")
	}
	b.WriteString("    }")
}

// writeMethodLines writes method lines with optional truncation.
func writeMethodLines(b *strings.Builder, methods []MethodSig, opts DiagramOptions) {
	limit := len(methods)
	truncated := false
	if opts.MaxMethodsPerBox > 0 && limit > opts.MaxMethodsPerBox {
		limit = opts.MaxMethodsPerBox
		truncated = true
	}

	for i := 0; i < limit; i++ {
		b.WriteString(fmt.Sprintf("        +%s\n", SanitizeSignature(methods[i].Signature)))
	}
	if truncated {
		b.WriteString("        ...\n")
	}
}

// writeRelation writes a single Mermaid relation line.
func writeRelation(b *strings.Builder, rel analyzer.Relation) {
	typeID := NodeID(rel.Type.PkgName, rel.Type.Name)
	ifaceID := NodeID(rel.Interface.PkgName, rel.Interface.Name)
	line := fmt.Sprintf("    %s --|> %s", typeID, ifaceID)
	b.WriteString(line)
}

// MethodSig is a local alias to avoid repeating the package prefix.
type MethodSig = analyzer.MethodSig

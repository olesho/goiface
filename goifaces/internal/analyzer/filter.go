package analyzer

import (
	"strings"
	"unicode"
)

// Filter applies filtering options to the analysis result.
func Filter(result *Result, opts AnalyzeOptions) *Result {
	filtered := &Result{}

	// Build sets of interfaces and types that participate in relations
	ifaceSet := make(map[string]bool)
	typeSet := make(map[string]bool)

	for _, rel := range result.Relations {
		iface := rel.Interface
		typ := rel.Type

		// Filter stdlib
		if !opts.IncludeStdlib {
			if isStdlib(iface.PkgPath) {
				continue
			}
		}

		// Filter unexported
		if !opts.IncludeUnexported {
			if isUnexported(iface.Name) || isUnexported(typ.Name) {
				continue
			}
		}

		// Filter by package prefix
		if opts.Filter != "" {
			ifaceMatch := strings.HasPrefix(iface.PkgPath, opts.Filter)
			typeMatch := strings.HasPrefix(typ.PkgPath, opts.Filter)
			if !ifaceMatch && !typeMatch {
				continue
			}
		}

		filtered.Relations = append(filtered.Relations, rel)
		ifaceSet[ifaceKey(iface)] = true
		typeSet[typeKey(typ)] = true
	}

	// Include only interfaces and types that participate in relations (prune orphans)
	for i := range result.Interfaces {
		iface := &result.Interfaces[i]
		if ifaceSet[ifaceKey(iface)] {
			filtered.Interfaces = append(filtered.Interfaces, *iface)
		}
	}

	for i := range result.Types {
		typ := &result.Types[i]
		if typeSet[typeKey(typ)] {
			filtered.Types = append(filtered.Types, *typ)
		}
	}

	return filtered
}

func isStdlib(pkgPath string) bool {
	// Stdlib packages have no dot in the first path element
	firstSlash := strings.IndexByte(pkgPath, '/')
	firstPart := pkgPath
	if firstSlash >= 0 {
		firstPart = pkgPath[:firstSlash]
	}
	return !strings.Contains(firstPart, ".")
}

func isUnexported(name string) bool {
	if name == "" {
		return true
	}
	// Built-in types like 'error' are lowercase but considered exported
	if name == "error" {
		return false
	}
	return unicode.IsLower(rune(name[0]))
}

func ifaceKey(iface *InterfaceDef) string {
	return iface.PkgPath + "." + iface.Name
}

func typeKey(typ *TypeDef) string {
	return typ.PkgPath + "." + typ.Name
}

package analyzer

import (
	"context"
	"fmt"
	"go/token"
	"go/types"
	"log/slog"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/types/typeutil"
)

// Analyze loads Go packages from dir and finds all interface-implementation relationships.
func Analyze(ctx context.Context, dir string, opts AnalyzeOptions, logger *slog.Logger) (*Result, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedSyntax |
			packages.NeedTypesInfo | packages.NeedImports,
		Dir:     dir,
		Context: ctx,
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("loading packages: %w", err)
	}

	// When including stdlib, also load common stdlib packages that define interfaces
	if opts.IncludeStdlib {
		stdlibPatterns := []string{"fmt", "io", "io/fs", "encoding", "encoding/json", "sort", "hash", "context"}
		stdPkgs, stdErr := packages.Load(cfg, stdlibPatterns...)
		if stdErr != nil {
			logger.Warn("failed to load stdlib packages", "error", stdErr)
		} else {
			pkgs = append(pkgs, stdPkgs...)
		}
	}

	logger.Info("packages loaded", "packages_count", len(pkgs))

	// Log packages with errors but continue
	for _, pkg := range pkgs {
		for _, e := range pkg.Errors {
			logger.Warn("package load error", "package", pkg.PkgPath, "error", e.Msg)
		}
	}

	// Phase 2: Collect interfaces and named types
	var ifaces []InterfaceDef
	var namedTypes []TypeDef
	seenIfaces := make(map[string]bool) // pkgPath.Name dedup

	collectFromScope := func(scope *types.Scope, pkgPath, pkgName string, fset *token.FileSet, moduleRoot string) {
		for _, name := range scope.Names() {
			obj := scope.Lookup(name)
			tn, ok := obj.(*types.TypeName)
			if !ok {
				continue
			}

			named, ok := tn.Type().(*types.Named)
			if !ok {
				continue
			}

			if iface, ok := named.Underlying().(*types.Interface); ok {
				key := pkgPath + "." + tn.Name()
				if seenIfaces[key] {
					continue
				}
				seenIfaces[key] = true
				ifaceDef := InterfaceDef{
					Name:       tn.Name(),
					PkgPath:    pkgPath,
					PkgName:    pkgName,
					Methods:    extractIfaceMethods(iface),
					TypeObj:    iface,
					SourceFile: resolveSourceFile(fset, tn.Pos(), moduleRoot),
				}
				ifaces = append(ifaces, ifaceDef)
				logger.Debug("found interface", "name", tn.Name(), "package", pkgPath, "methods", iface.NumMethods())
			}
		}
	}

	for _, pkg := range pkgs {
		if pkg.Types == nil {
			continue
		}

		// Collect types from direct packages
		scope := pkg.Types.Scope()
		collectFromScope(scope, pkg.PkgPath, pkg.Name, pkg.Fset, dir)

		for _, name := range scope.Names() {
			obj := scope.Lookup(name)
			tn, ok := obj.(*types.TypeName)
			if !ok {
				continue
			}
			named, ok := tn.Type().(*types.Named)
			if !ok {
				continue
			}
			if _, ok := named.Underlying().(*types.Interface); !ok {
				methods := extractTypeMethods(named)
				typeDef := TypeDef{
					Name:       tn.Name(),
					PkgPath:    pkg.PkgPath,
					PkgName:    pkg.Name,
					IsStruct:   isStruct(named),
					Methods:    methods,
					TypeObj:    named,
					SourceFile: resolveSourceFile(pkg.Fset, tn.Pos(), dir),
				}
				namedTypes = append(namedTypes, typeDef)
				logger.Debug("found type", "name", tn.Name(), "package", pkg.PkgPath, "methods", len(methods))
			}
		}

		// Also collect interfaces from imported packages (for stdlib matching)
		for _, imp := range pkg.Imports {
			if imp.Types == nil {
				continue
			}
			collectFromScope(imp.Types.Scope(), imp.PkgPath, imp.Name, imp.Fset, dir)
		}
	}

	// Also add the built-in 'error' interface from the universe scope
	errorObj := types.Universe.Lookup("error")
	if errorObj != nil {
		if tn, ok := errorObj.(*types.TypeName); ok {
			if iface, ok := tn.Type().Underlying().(*types.Interface); ok {
				key := "builtin.error"
				if !seenIfaces[key] {
					seenIfaces[key] = true
					ifaces = append(ifaces, InterfaceDef{
						Name:    "error",
						PkgPath: "builtin",
						PkgName: "builtin",
						Methods: extractIfaceMethods(iface),
						TypeObj: iface,
					})
				}
			}
		}
	}

	logger.Info("types collected", "interfaces", len(ifaces), "types", len(namedTypes))

	// Phase 3: Match implementations
	var methodSetCache typeutil.MethodSetCache
	var relations []Relation

	for i := range namedTypes {
		t := &namedTypes[i]
		for j := range ifaces {
			iface := &ifaces[j]

			// Skip empty interfaces
			if iface.TypeObj.NumMethods() == 0 {
				continue
			}

			valType := t.TypeObj
			valMethodSet := methodSetCache.MethodSet(valType)
			ptrMethodSet := methodSetCache.MethodSet(types.NewPointer(valType))

			if types.Implements(valType, iface.TypeObj) || matchesMethodSet(valMethodSet, iface.TypeObj) {
				relations = append(relations, Relation{
					Type:       t,
					Interface:  iface,
					ViaPointer: false,
				})
				logger.Debug("match found", "type", t.Name, "interface", iface.Name, "via_pointer", false)
			} else if types.Implements(types.NewPointer(valType), iface.TypeObj) || matchesMethodSet(ptrMethodSet, iface.TypeObj) {
				relations = append(relations, Relation{
					Type:       t,
					Interface:  iface,
					ViaPointer: true,
				})
				logger.Debug("match found", "type", t.Name, "interface", iface.Name, "via_pointer", true)
			}
		}
	}

	logger.Info("analysis complete", "relations", len(relations))

	return &Result{
		Interfaces: ifaces,
		Types:      namedTypes,
		Relations:  relations,
	}, nil
}

func extractIfaceMethods(iface *types.Interface) []MethodSig {
	methods := make([]MethodSig, iface.NumMethods())
	for i := 0; i < iface.NumMethods(); i++ {
		m := iface.Method(i)
		methods[i] = MethodSig{
			Name:      m.Name(),
			Signature: formatSignature(m),
		}
	}
	return methods
}

func extractTypeMethods(named *types.Named) []MethodSig {
	var methods []MethodSig
	// Value receiver methods
	for i := 0; i < named.NumMethods(); i++ {
		m := named.Method(i)
		methods = append(methods, MethodSig{
			Name:      m.Name(),
			Signature: formatSignature(m),
		})
	}
	return methods
}

func formatSignature(fn *types.Func) string {
	sig := fn.Type().(*types.Signature)
	var b strings.Builder
	b.WriteString(fn.Name())
	b.WriteString("(")
	params := sig.Params()
	for i := 0; i < params.Len(); i++ {
		if i > 0 {
			b.WriteString(", ")
		}
		p := params.At(i)
		b.WriteString(shortType(p.Type()))
	}
	b.WriteString(")")
	results := sig.Results()
	if results.Len() > 0 {
		b.WriteString(" ")
		if results.Len() == 1 {
			b.WriteString(shortType(results.At(0).Type()))
		} else {
			b.WriteString("(")
			for i := 0; i < results.Len(); i++ {
				if i > 0 {
					b.WriteString(", ")
				}
				b.WriteString(shortType(results.At(i).Type()))
			}
			b.WriteString(")")
		}
	}
	return b.String()
}

func shortType(t types.Type) string {
	return types.TypeString(t, func(pkg *types.Package) string {
		return pkg.Name()
	})
}

func isStruct(named *types.Named) bool {
	_, ok := named.Underlying().(*types.Struct)
	return ok
}

func matchesMethodSet(mset *types.MethodSet, iface *types.Interface) bool {
	for i := 0; i < iface.NumMethods(); i++ {
		m := iface.Method(i)
		sel := mset.Lookup(m.Pkg(), m.Name())
		if sel == nil {
			return false
		}
	}
	return true
}

// resolveSourceFile resolves a token position to a file path relative to moduleRoot.
func resolveSourceFile(fset *token.FileSet, pos token.Pos, moduleRoot string) string {
	if fset == nil || !pos.IsValid() {
		return ""
	}
	position := fset.Position(pos)
	if !position.IsValid() || position.Filename == "" {
		return ""
	}
	rel, err := filepath.Rel(moduleRoot, position.Filename)
	if err != nil {
		return position.Filename
	}
	return rel
}

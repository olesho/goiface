package analyzer

import "go/types"

// InterfaceDef represents a discovered Go interface.
type InterfaceDef struct {
	Name       string
	PkgPath    string
	PkgName    string
	Methods    []MethodSig
	TypeObj    *types.Interface
	SourceFile string
}

// TypeDef represents a discovered named Go type.
type TypeDef struct {
	Name       string
	PkgPath    string
	PkgName    string
	IsStruct   bool
	Methods    []MethodSig
	TypeObj    *types.Named
	SourceFile string
}

// MethodSig captures a method name and its signature string.
type MethodSig struct {
	Name      string
	Signature string
}

// Relation captures that a concrete type implements an interface.
type Relation struct {
	Type       *TypeDef
	Interface  *InterfaceDef
	ViaPointer bool // true if only *T (not T) satisfies the interface
}

// Result holds the complete analysis output.
type Result struct {
	Interfaces []InterfaceDef
	Types      []TypeDef
	Relations  []Relation
	ModulePath string // module path from go.mod (e.g. "github.com/user/repo")
}

// AnalyzeOptions controls analysis behavior.
type AnalyzeOptions struct {
	Filter            string // package path prefix filter
	IncludeStdlib     bool
	IncludeUnexported bool
}

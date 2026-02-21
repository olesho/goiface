package llm

import (
	"strings"
	"testing"

	"github.com/olehluchkiv/goifaces/internal/analyzer"
	"github.com/stretchr/testify/assert"
)

func TestSerializeResult(t *testing.T) {
	result := &analyzer.Result{
		Interfaces: []analyzer.InterfaceDef{
			{
				Name:    "Reader",
				PkgPath: "io",
				PkgName: "io",
				Methods: []analyzer.MethodSig{
					{Name: "Read", Signature: "(p []byte) (n int, err error)"},
				},
			},
		},
		Types: []analyzer.TypeDef{
			{
				Name:     "File",
				PkgPath:  "os",
				PkgName:  "os",
				IsStruct: true,
				Methods: []analyzer.MethodSig{
					{Name: "Read", Signature: "(p []byte) (n int, err error)"},
					{Name: "Close", Signature: "() error"},
				},
			},
		},
		Relations: []analyzer.Relation{
			{
				Type:      &analyzer.TypeDef{Name: "File", PkgPath: "os"},
				Interface: &analyzer.InterfaceDef{Name: "Reader", PkgPath: "io"},
			},
		},
	}

	text := SerializeResult(result)

	assert.Contains(t, text, "INTERFACES:")
	assert.Contains(t, text, "io.Reader")
	assert.Contains(t, text, "Read(p []byte) (n int, err error)")
	assert.Contains(t, text, "TYPES:")
	assert.Contains(t, text, "os.File (struct, package: os)")
	assert.Contains(t, text, "RELATIONSHIPS:")
	assert.Contains(t, text, "[0] os.File implements io.Reader")
}

func TestSerializeRelations(t *testing.T) {
	relations := []analyzer.Relation{
		{
			Type:      &analyzer.TypeDef{Name: "Conn", PkgPath: "net"},
			Interface: &analyzer.InterfaceDef{Name: "Reader", PkgPath: "io"},
		},
		{
			Type:       &analyzer.TypeDef{Name: "Conn", PkgPath: "net"},
			Interface:  &analyzer.InterfaceDef{Name: "Writer", PkgPath: "io"},
			ViaPointer: true,
		},
	}

	text := SerializeRelations(relations)
	assert.Contains(t, text, "[0] net.Conn implements io.Reader")
	assert.Contains(t, text, "[1] net.Conn implements io.Writer")
}

func TestSerializeNodeList(t *testing.T) {
	result := &analyzer.Result{
		Interfaces: []analyzer.InterfaceDef{
			{Name: "A", PkgPath: "pkg"},
		},
		Types: []analyzer.TypeDef{
			{Name: "B", PkgPath: "pkg"},
		},
	}

	text := SerializeNodeList(result)
	assert.Contains(t, text, "IFACE pkg.A")
	assert.Contains(t, text, "TYPE  pkg.B")
}

func TestSerializeResult_MethodTruncation(t *testing.T) {
	var methods []analyzer.MethodSig
	for i := range 15 {
		methods = append(methods, analyzer.MethodSig{
			Name:      "Method" + strings.Repeat("X", i),
			Signature: "()",
		})
	}

	result := &analyzer.Result{
		Interfaces: []analyzer.InterfaceDef{
			{Name: "Big", PkgPath: "pkg", PkgName: "pkg", Methods: methods},
		},
	}

	text := SerializeResult(result)
	assert.Contains(t, text, "10 shown of 15")
}

func TestFilterTopNodes(t *testing.T) {
	ifaceA := analyzer.InterfaceDef{Name: "A", PkgPath: "pkg"}
	ifaceB := analyzer.InterfaceDef{Name: "B", PkgPath: "pkg"}
	typeC := analyzer.TypeDef{Name: "C", PkgPath: "pkg"}
	typeD := analyzer.TypeDef{Name: "D", PkgPath: "pkg"}

	result := &analyzer.Result{
		Interfaces: []analyzer.InterfaceDef{ifaceA, ifaceB},
		Types:      []analyzer.TypeDef{typeC, typeD},
		Relations: []analyzer.Relation{
			{Type: &typeC, Interface: &ifaceA},
			{Type: &typeC, Interface: &ifaceB},
			{Type: &typeD, Interface: &ifaceA},
		},
	}

	// Keep only 2 nodes — should prefer most-connected
	ifaces, types, rels := filterTopNodes(result, 2)
	total := len(ifaces) + len(types)
	assert.LessOrEqual(t, total, 2)
	assert.NotEmpty(t, rels)
}

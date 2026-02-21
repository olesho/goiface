package llm_test

import (
	"strings"
	"testing"

	"github.com/olehluchkiv/goifaces/internal/analyzer"
	"github.com/olehluchkiv/goifaces/internal/enricher/llm"
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

	text := llm.SerializeResult(result)

	assert.Contains(t, text, "## Interfaces")
	assert.Contains(t, text, "io.Reader")
	assert.Contains(t, text, "Read(p []byte) (n int, err error)")
	assert.Contains(t, text, "## Types")
	assert.Contains(t, text, "os.File")
	assert.Contains(t, text, "## Relationships")
	assert.Contains(t, text, "os.File implements io.Reader")
}

func TestSerializeRelations(t *testing.T) {
	result := &analyzer.Result{
		Relations: []analyzer.Relation{
			{
				Type:      &analyzer.TypeDef{Name: "Conn", PkgPath: "net"},
				Interface: &analyzer.InterfaceDef{Name: "Reader", PkgPath: "io"},
			},
			{
				Type:       &analyzer.TypeDef{Name: "Conn", PkgPath: "net"},
				Interface:  &analyzer.InterfaceDef{Name: "Writer", PkgPath: "io"},
				ViaPointer: true,
			},
		},
	}

	text := llm.SerializeRelations(result)
	assert.Contains(t, text, "net.Conn implements io.Reader")
	assert.Contains(t, text, "net.Conn implements io.Writer")
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

	text := llm.SerializeNodeList(result)
	assert.Contains(t, text, "pkg.A (interface)")
	assert.Contains(t, text, "pkg.B (type)")
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

	text := llm.SerializeResult(result)
	// HEAD's formatMethods truncates at 10 and adds "...+N more"
	assert.Contains(t, text, "...+5 more")
}

func TestPreFilterByEdgeCount(t *testing.T) {
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
	filtered := llm.PreFilterByEdgeCount(result, 2)
	total := len(filtered.Interfaces) + len(filtered.Types)
	assert.LessOrEqual(t, total, 2)
	assert.NotEmpty(t, filtered.Relations)
}

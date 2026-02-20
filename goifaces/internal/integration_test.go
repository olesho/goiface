package internal_test

import (
	"context"
	"go/token"
	"go/types"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/olehluchkiv/goifaces/internal/analyzer"
	"github.com/olehluchkiv/goifaces/internal/diagram"
	"github.com/olehluchkiv/goifaces/internal/diagram/split"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// normalizeOutput sorts class definitions and relations alphabetically
// to make comparison deterministic regardless of map iteration order.
// Designed for full-diagram output from GenerateMermaid only. For overview
// slide output (generateOverviewMermaid), cssClass lines appear after class
// blocks and will be silently dropped by this function.
func normalizeOutput(s string) string {
	s = strings.TrimSpace(s)
	lines := strings.Split(s, "\n")
	if len(lines) == 0 {
		return s
	}

	// Parse into sections: header, class blocks, relation lines
	var header string
	var blocks []string
	var relations []string

	i := 0
	// Collect header: optional init directive + "classDiagram"
	var headerLines []string
	for i < len(lines) {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "%%{init:") || trimmed == "classDiagram" || strings.HasPrefix(trimmed, "classDef ") || strings.HasPrefix(trimmed, "cssClass ") {
			headerLines = append(headerLines, trimmed)
			i++
			if trimmed == "classDiagram" {
				break
			}
		} else {
			break
		}
	}
	// Also consume classDef/cssClass lines that follow classDiagram as part of header
	for i < len(lines) {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "classDef ") || strings.HasPrefix(trimmed, "cssClass ") || strings.HasPrefix(trimmed, "direction ") {
			headerLines = append(headerLines, trimmed)
			i++
		} else {
			break
		}
	}
	header = strings.Join(headerLines, "\n")

	// Parse class blocks and relations
	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			i++
			continue
		}

		if strings.HasPrefix(trimmed, "class ") {
			// Start of a class block - collect until closing brace
			var block []string
			for i < len(lines) {
				block = append(block, lines[i])
				if strings.TrimSpace(lines[i]) == "}" {
					i++
					break
				}
				i++
			}
			blocks = append(blocks, strings.Join(block, "\n"))
		} else if strings.Contains(trimmed, "..|>") || strings.Contains(trimmed, "--|>") {
			relations = append(relations, line)
			i++
		} else {
			i++
		}
	}

	sort.Strings(blocks)
	sort.Strings(relations)

	var parts []string
	parts = append(parts, header)
	parts = append(parts, blocks...)
	if len(relations) > 0 {
		parts = append(parts, "") // blank line before relations
		parts = append(parts, relations...)
	}

	return strings.Join(parts, "\n")
}

func testdataDir(name string) string {
	// Find the project root by looking for go.mod
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	// We're in internal/, go up one level
	root := filepath.Dir(wd)
	return filepath.Join(root, "testdata", name)
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
}

func TestEndToEnd(t *testing.T) {
	ctx := context.Background()
	logger := testLogger()

	// Use unlimited methods for test clarity
	diagramOpts := diagram.DiagramOptions{MaxMethodsPerBox: 0}

	tests := []struct {
		name     string
		dir      string
		opts     analyzer.AnalyzeOptions
		validate func(t *testing.T, got string)
	}{
		{
			name: "01_single_iface",
			dir:  testdataDir("01_single_iface"),
			opts: analyzer.AnalyzeOptions{},
			validate: func(t *testing.T, got string) {
				assert.Contains(t, got, "classDiagram")
				assert.Contains(t, got, "<<interface>>")
				assert.Contains(t, got, "shapes_Shape")
				assert.Contains(t, got, "shapes_Circle")
				assert.Contains(t, got, "shapes_Circle --|> shapes_Shape")
				assert.Contains(t, got, "Area()")
			},
		},
		{
			name: "02_multi_impl",
			dir:  testdataDir("02_multi_impl"),
			opts: analyzer.AnalyzeOptions{},
			validate: func(t *testing.T, got string) {
				assert.Contains(t, got, "animals_Speaker")
				assert.Contains(t, got, "animals_Dog")
				assert.Contains(t, got, "animals_Cat")
				assert.Contains(t, got, "animals_Dog --|> animals_Speaker")
				assert.Contains(t, got, "animals_Cat --|> animals_Speaker")
				// Fish has no Speak() — should not appear
				assert.NotContains(t, got, "animals_Fish")
			},
		},
		{
			name: "03_multi_iface",
			dir:  testdataDir("03_multi_iface"),
			opts: analyzer.AnalyzeOptions{},
			validate: func(t *testing.T, got string) {
				assert.Contains(t, got, "store_Reader")
				assert.Contains(t, got, "store_Writer")
				assert.Contains(t, got, "store_ReadWriter")
				assert.Contains(t, got, "store_MemStore")
				assert.Contains(t, got, "store_ReadOnlyCache")
				// MemStore implements all three
				assert.Contains(t, got, "store_MemStore --|> store_Reader")
				assert.Contains(t, got, "store_MemStore --|> store_Writer")
				assert.Contains(t, got, "store_MemStore --|> store_ReadWriter")
				// ReadOnlyCache implements only Reader
				assert.Contains(t, got, "store_ReadOnlyCache --|> store_Reader")
				assert.NotContains(t, got, "store_ReadOnlyCache --|> store_Writer")
				assert.NotContains(t, got, "store_ReadOnlyCache --|> store_ReadWriter")
			},
		},
		{
			name: "04_pointer_receiver",
			dir:  testdataDir("04_pointer_receiver"),
			opts: analyzer.AnalyzeOptions{},
			validate: func(t *testing.T, got string) {
				assert.Contains(t, got, "db_Closer")
				assert.Contains(t, got, "db_Connection")
				assert.Contains(t, got, "db_Connection --|> db_Closer")
			},
		},
		{
			name: "05_embedded_iface",
			dir:  testdataDir("05_embedded_iface"),
			opts: analyzer.AnalyzeOptions{},
			validate: func(t *testing.T, got string) {
				assert.Contains(t, got, "io2_Reader")
				assert.Contains(t, got, "io2_Closer")
				assert.Contains(t, got, "io2_ReadCloser")
				assert.Contains(t, got, "io2_MyFile")
				// MyFile implements all three
				assert.Contains(t, got, "io2_MyFile --|> io2_Reader")
				assert.Contains(t, got, "io2_MyFile --|> io2_Closer")
				assert.Contains(t, got, "io2_MyFile --|> io2_ReadCloser")
			},
		},
		{
			name: "06_cross_package",
			dir:  testdataDir("06_cross_package"),
			opts: analyzer.AnalyzeOptions{},
			validate: func(t *testing.T, got string) {
				assert.Contains(t, got, "ifaces_Logger")
				assert.Contains(t, got, "impl_ConsoleLogger")
				assert.Contains(t, got, "impl_ConsoleLogger --|> ifaces_Logger")
			},
		},
		{
			name: "07_stdlib_ifaces_with_stdlib",
			dir:  testdataDir("07_stdlib_ifaces"),
			opts: analyzer.AnalyzeOptions{IncludeStdlib: true},
			validate: func(t *testing.T, got string) {
				// With stdlib included, we should see implementations of stdlib interfaces
				assert.Contains(t, got, "mylib_MyError")
				assert.Contains(t, got, "mylib_Pretty")
				assert.Contains(t, got, "mylib_Bytes")
			},
		},
		{
			name: "07_stdlib_ifaces_without_stdlib",
			dir:  testdataDir("07_stdlib_ifaces"),
			opts: analyzer.AnalyzeOptions{IncludeStdlib: false},
			validate: func(t *testing.T, got string) {
				// Without stdlib, no repo-defined interfaces → empty diagram
				// The types only implement stdlib interfaces, so nothing to show
				normalized := normalizeOutput(got)
				assert.Contains(t, normalized, "classDiagram")
				// Should not contain any class blocks or relations
				assert.NotContains(t, normalized, "class ")
				assert.NotContains(t, normalized, "--|>")
			},
		},
		{
			name: "08_empty_iface",
			dir:  testdataDir("08_empty_iface"),
			opts: analyzer.AnalyzeOptions{},
			validate: func(t *testing.T, got string) {
				// Empty interfaces should be skipped — no relations
				normalized := normalizeOutput(got)
				assert.Contains(t, normalized, "classDiagram")
				// Should not contain any class blocks or relations
				assert.NotContains(t, normalized, "class ")
				assert.NotContains(t, normalized, "--|>")
			},
		},
		{
			name: "09_unexported_default",
			dir:  testdataDir("09_unexported"),
			opts: analyzer.AnalyzeOptions{IncludeUnexported: false},
			validate: func(t *testing.T, got string) {
				// Only exported: Runner and Cat
				assert.Contains(t, got, "internal_Runner")
				assert.Contains(t, got, "internal_Cat")
				assert.Contains(t, got, "internal_Cat --|> internal_Runner")
				// Unexported should NOT appear
				assert.NotContains(t, got, "internal_walker")
				assert.NotContains(t, got, "internal_dog")
			},
		},
		{
			name: "09_unexported_included",
			dir:  testdataDir("09_unexported"),
			opts: analyzer.AnalyzeOptions{IncludeUnexported: true},
			validate: func(t *testing.T, got string) {
				// All should appear
				assert.Contains(t, got, "internal_walker")
				assert.Contains(t, got, "internal_Runner")
				assert.Contains(t, got, "internal_dog")
				assert.Contains(t, got, "internal_Cat")
				assert.Contains(t, got, "internal_dog --|> internal_walker")
				assert.Contains(t, got, "internal_dog --|> internal_Runner")
				assert.Contains(t, got, "internal_Cat --|> internal_Runner")
			},
		},
		{
			name: "10_diamond",
			dir:  testdataDir("10_diamond"),
			opts: analyzer.AnalyzeOptions{},
			validate: func(t *testing.T, got string) {
				assert.Contains(t, got, "diamond_Saver")
				assert.Contains(t, got, "diamond_Loader")
				assert.Contains(t, got, "diamond_Persister")
				assert.Contains(t, got, "diamond_DB")
				// DB implements all three
				assert.Contains(t, got, "diamond_DB --|> diamond_Saver")
				assert.Contains(t, got, "diamond_DB --|> diamond_Loader")
				assert.Contains(t, got, "diamond_DB --|> diamond_Persister")
			},
		},
		{
			name: "11_source_file_path",
			dir:  testdataDir("01_single_iface"),
			opts: analyzer.AnalyzeOptions{},
			validate: func(t *testing.T, got string) {
				assert.Contains(t, got, "%% file:")
				assert.Contains(t, got, "shapes.go")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := analyzer.Analyze(ctx, tt.dir, tt.opts, logger)
			require.NoError(t, err)

			filtered := analyzer.Filter(result, tt.opts)
			got := diagram.GenerateMermaid(filtered, diagramOpts)

			tt.validate(t, got)
		})
	}
}

func TestHubAndSpokeSlides(t *testing.T) {
	// Build synthetic go-memdb-like data: 4 hub interfaces, 12 types, 38 relations
	pkg := "memdb"
	makeIface := func(name string) analyzer.InterfaceDef {
		return analyzer.InterfaceDef{Name: name, PkgPath: pkg, PkgName: pkg}
	}
	makeType := func(name string) analyzer.TypeDef {
		return analyzer.TypeDef{Name: name, PkgPath: pkg, PkgName: pkg}
	}

	ifaces := []analyzer.InterfaceDef{
		makeIface("Indexer"),
		makeIface("MultiIndexer"),
		makeIface("PrefixIndexer"),
		makeIface("ResultIterator"),
		makeIface("SingleIndexer"),
	}

	fieldIndexTypes := []string{
		"BoolFieldIndex", "CompoundIndex", "CompoundMultiIndex",
		"ConditionalIndex", "FieldSetIndex", "IntFieldIndex",
		"StringFieldIndex", "StringMapFieldIndex", "StringSliceFieldIndex",
		"UUIDFieldIndex", "UintFieldIndex",
	}
	var types []analyzer.TypeDef
	for _, name := range fieldIndexTypes {
		types = append(types, makeType(name))
	}
	types = append(types, makeType("FilterIterator"))

	// Build relations: each field index → Indexer, MultiIndexer, SingleIndexer
	ifaceMap := make(map[string]*analyzer.InterfaceDef)
	for i := range ifaces {
		ifaceMap[ifaces[i].Name] = &ifaces[i]
	}
	typeMap := make(map[string]*analyzer.TypeDef)
	for i := range types {
		typeMap[types[i].Name] = &types[i]
	}

	var rels []analyzer.Relation
	for _, name := range fieldIndexTypes {
		for _, ifaceName := range []string{"Indexer", "MultiIndexer", "SingleIndexer"} {
			rels = append(rels, analyzer.Relation{
				Type:      typeMap[name],
				Interface: ifaceMap[ifaceName],
			})
		}
	}
	// PrefixIndexer connections
	for _, name := range []string{"StringFieldIndex", "StringMapFieldIndex", "StringSliceFieldIndex", "CompoundIndex"} {
		rels = append(rels, analyzer.Relation{
			Type:      typeMap[name],
			Interface: ifaceMap["PrefixIndexer"],
		})
	}
	// FilterIterator → ResultIterator
	rels = append(rels, analyzer.Relation{
		Type:      typeMap["FilterIterator"],
		Interface: ifaceMap["ResultIterator"],
	})

	result := &analyzer.Result{
		Interfaces: ifaces,
		Types:      types,
		Relations:  rels,
	}

	diagOpts := diagram.DiagramOptions{MaxMethodsPerBox: 5}
	splitter := split.NewHubAndSpoke(split.Options{HubThreshold: 3, ChunkSize: 3})

	// Default threshold=20: 17 nodes but 38 relations — relation count triggers splitting
	slideOpts := diagram.SlideOptions{Threshold: 20}
	slides := diagram.BuildSlides(result, diagOpts, splitter, slideOpts)

	// Slide 0 = overview + 4 detail slides = 5 total
	require.Equal(t, 5, len(slides), "expected 5 slides (1 overview + 4 detail): 17 nodes < 20 but 38 relations >= 20")
	assert.Equal(t, "Overview", slides[0].Title)

	// Each detail slide should contain hub interfaces
	hubNames := []string{"memdb_Indexer", "memdb_MultiIndexer", "memdb_SingleIndexer", "memdb_PrefixIndexer"}
	for i := 1; i < len(slides); i++ {
		mermaid := slides[i].Mermaid
		for _, hub := range hubNames {
			assert.Contains(t, mermaid, hub,
				"slide %d should contain hub %s", i, hub)
		}
	}

	// FilterIterator and ResultIterator should be on the same slide
	var filterSlide string
	for i := 1; i < len(slides); i++ {
		if strings.Contains(slides[i].Mermaid, "memdb_FilterIterator") {
			filterSlide = slides[i].Mermaid
		}
	}
	require.NotEmpty(t, filterSlide, "FilterIterator should appear on some slide")
	assert.Contains(t, filterSlide, "memdb_ResultIterator",
		"ResultIterator should be on same slide as FilterIterator")

	// Overview should show only interfaces, no implementation blocks or arrows
	overview := slides[0].Mermaid
	assert.NotContains(t, overview, "+", "overview should have no method lines")

	// Should NOT contain implementation node IDs
	for _, implName := range fieldIndexTypes {
		implID := "memdb_" + implName
		assert.NotContains(t, overview, implID,
			"overview should not contain implementation node %s", implID)
	}
	assert.NotContains(t, overview, "memdb_FilterIterator",
		"overview should not contain implementation node memdb_FilterIterator")

	// Hub-and-spoke has no interface embedding, so no arrows in overview
	assert.NotContains(t, overview, "--|>", "overview should not contain arrows (no embedding in this dataset)")

	// Should NOT contain implStyle
	assert.NotContains(t, overview, "implStyle", "overview should not contain implStyle")

	// SHOULD contain interface node IDs
	for _, ifaceName := range []string{"memdb_Indexer", "memdb_MultiIndexer", "memdb_PrefixIndexer", "memdb_ResultIterator", "memdb_SingleIndexer"} {
		assert.Contains(t, overview, ifaceName,
			"overview should contain interface node %s", ifaceName)
	}
}

func TestOverviewInterfaceEmbedding(t *testing.T) {
	// Synthetic interfaces for testing collectEmbeddingArrows in isolation.
	// Interfaces are intentionally empty (no methods) — in production the analyzer
	// skips empty interfaces, but this test passes a hand-crafted Result directly
	// to BuildSlides to bypass that guard.
	pkg := types.NewPackage("example.com/mylib", "mylib")

	readerIface := types.NewInterfaceType(nil, nil)
	readerIface.Complete()
	readerNamed := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Reader", nil),
		readerIface, nil,
	)

	closerIface := types.NewInterfaceType(nil, nil)
	closerIface.Complete()
	closerNamed := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Closer", nil),
		closerIface, nil,
	)

	// ReadCloser embeds Reader and Closer
	readCloserIface := types.NewInterfaceType(nil, []types.Type{readerNamed, closerNamed})
	readCloserIface.Complete()

	ifaces := []analyzer.InterfaceDef{
		{Name: "Reader", PkgPath: "example.com/mylib", PkgName: "mylib", TypeObj: readerIface},
		{Name: "Closer", PkgPath: "example.com/mylib", PkgName: "mylib", TypeObj: closerIface},
		{Name: "ReadCloser", PkgPath: "example.com/mylib", PkgName: "mylib", TypeObj: readCloserIface},
	}

	// Add a concrete type to have a result with relations for slide splitting
	typs := []analyzer.TypeDef{
		{Name: "MyFile", PkgPath: "example.com/mylib", PkgName: "mylib"},
	}

	ifaceRefs := make(map[string]*analyzer.InterfaceDef)
	for i := range ifaces {
		ifaceRefs[ifaces[i].Name] = &ifaces[i]
	}
	typeRef := &typs[0]

	rels := []analyzer.Relation{
		{Type: typeRef, Interface: ifaceRefs["Reader"]},
		{Type: typeRef, Interface: ifaceRefs["Closer"]},
		{Type: typeRef, Interface: ifaceRefs["ReadCloser"]},
	}

	result := &analyzer.Result{
		Interfaces: ifaces,
		Types:      typs,
		Relations:  rels,
	}

	diagOpts := diagram.DiagramOptions{MaxMethodsPerBox: 5}
	splitter := split.NewHubAndSpoke(split.Options{HubThreshold: 1, ChunkSize: 3})
	slideOpts := diagram.SlideOptions{Threshold: 1} // force splitting

	slides := diagram.BuildSlides(result, diagOpts, splitter, slideOpts)
	require.GreaterOrEqual(t, len(slides), 2, "should have at least overview + 1 detail slide")

	overview := slides[0].Mermaid

	// Overview SHOULD contain embedding arrows
	assert.Contains(t, overview, "mylib_ReadCloser --|> mylib_Reader",
		"overview should show ReadCloser embeds Reader")
	assert.Contains(t, overview, "mylib_ReadCloser --|> mylib_Closer",
		"overview should show ReadCloser embeds Closer")

	// Overview should NOT contain implementation blocks or arrows
	assert.NotContains(t, overview, "mylib_MyFile",
		"overview should not contain implementation node")
	// Note: overview contains --|> for embedding arrows (ReadCloser --|> Reader),
	// which is correct. Implementation arrows are excluded by the overview generator.
	assert.NotContains(t, overview, "implStyle",
		"overview should not contain implStyle")

	// Overview SHOULD contain all three interface nodes
	assert.Contains(t, overview, "mylib_Reader")
	assert.Contains(t, overview, "mylib_Closer")
	assert.Contains(t, overview, "mylib_ReadCloser")
}

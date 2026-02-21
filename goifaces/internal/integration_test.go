package internal_test

import (
	"context"
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
// Designed for full-diagram output from GenerateMermaid only.
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
		} else if strings.HasPrefix(trimmed, "cssClass ") || strings.HasPrefix(trimmed, "classDef ") {
			headerLines = append(headerLines, trimmed)
			header = strings.Join(headerLines, "\n")
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

	// Slide 0 = package map + 4 detail slides = 5 total
	require.Equal(t, 5, len(slides), "expected 5 slides (1 package map + 4 detail): 17 nodes < 20 but 38 relations >= 20")
	assert.Equal(t, "Package Map", slides[0].Title)

	// Indexer, MultiIndexer, SingleIndexer connect to all 11 field-index types,
	// so they should appear on every detail slide that has field-index types.
	// PrefixIndexer connects only to String*, StringMap*, StringSlice*, CompoundIndex.
	// ResultIterator connects only to FilterIterator.
	coreHubs := []string{"memdb_Indexer", "memdb_MultiIndexer", "memdb_SingleIndexer"}
	for i := 1; i < len(slides); i++ {
		mermaid := slides[i].Mermaid
		// Every detail slide has at least one field-index type, so core hubs appear everywhere
		for _, hub := range coreHubs {
			assert.Contains(t, mermaid, hub,
				"slide %d should contain hub %s", i, hub)
		}
	}

	// PrefixIndexer should appear ONLY on slides containing its implementing types
	prefixTypes := []string{"memdb_StringFieldIndex", "memdb_StringMapFieldIndex", "memdb_StringSliceFieldIndex", "memdb_CompoundIndex"}
	for i := 1; i < len(slides); i++ {
		mermaid := slides[i].Mermaid
		hasPrefixType := false
		for _, pt := range prefixTypes {
			if strings.Contains(mermaid, pt) {
				hasPrefixType = true
				break
			}
		}
		if hasPrefixType {
			assert.Contains(t, mermaid, "memdb_PrefixIndexer",
				"slide %d has PrefixIndexer types, so PrefixIndexer hub should be present", i)
		} else {
			assert.NotContains(t, mermaid, "memdb_PrefixIndexer",
				"slide %d has no PrefixIndexer types, so PrefixIndexer hub should be absent (orphan)", i)
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

	// ResultIterator should NOT appear on slides without FilterIterator (it would be orphaned)
	for i := 1; i < len(slides); i++ {
		mermaid := slides[i].Mermaid
		if !strings.Contains(mermaid, "memdb_FilterIterator") {
			assert.NotContains(t, mermaid, "memdb_ResultIterator",
				"slide %d has no FilterIterator, so ResultIterator should be absent (orphan)", i)
		}
	}

	// Package map should be a flowchart showing package hierarchy
	pkgMap := slides[0].Mermaid
	assert.Contains(t, pkgMap, "flowchart LR", "package map should be a flowchart")
	assert.Contains(t, pkgMap, "memdb", "package map should show memdb package")
	assert.Contains(t, pkgMap, "ifaces", "package map should show interface count")
	assert.Contains(t, pkgMap, "types", "package map should show type count")
}

func TestPackageMapMultiPackage(t *testing.T) {
	// Create a result with types from multiple packages
	ifaces := []analyzer.InterfaceDef{
		{Name: "Reader", PkgPath: "example.com/mylib/io", PkgName: "io"},
		{Name: "Writer", PkgPath: "example.com/mylib/io", PkgName: "io"},
		{Name: "Handler", PkgPath: "example.com/mylib/http", PkgName: "http"},
	}
	typs := []analyzer.TypeDef{
		{Name: "FileReader", PkgPath: "example.com/mylib/io", PkgName: "io"},
		{Name: "Server", PkgPath: "example.com/mylib/http", PkgName: "http"},
		{Name: "Router", PkgPath: "example.com/mylib/http/router", PkgName: "router"},
	}
	var rels []analyzer.Relation
	for i := range ifaces {
		rels = append(rels, analyzer.Relation{
			Type:      &typs[0],
			Interface: &ifaces[i],
		})
	}

	result := &analyzer.Result{
		Interfaces: ifaces,
		Types:      typs,
		Relations:  rels,
	}

	diagOpts := diagram.DiagramOptions{MaxMethodsPerBox: 5}
	splitter := split.NewHubAndSpoke(split.Options{HubThreshold: 3, ChunkSize: 3})
	slideOpts := diagram.SlideOptions{Threshold: 1}
	slides := diagram.BuildSlides(result, diagOpts, splitter, slideOpts)

	require.GreaterOrEqual(t, len(slides), 2, "should have package map + detail slides")
	pkgMap := slides[0].Mermaid

	assert.Equal(t, "Package Map", slides[0].Title)
	assert.Contains(t, pkgMap, "flowchart LR", "package map should be a flowchart")

	// Should show packages
	assert.Contains(t, pkgMap, "io", "should contain io package")
	assert.Contains(t, pkgMap, "http", "should contain http package")
	assert.Contains(t, pkgMap, "router", "should contain router subpackage")

	// Should show counts
	assert.Contains(t, pkgMap, "2 ifaces", "io should show 2 interfaces")
	assert.Contains(t, pkgMap, "1 ifaces", "http should show 1 interface")

	// Labels must use <br/> for line breaks, not literal \n
	assert.NotContains(t, pkgMap, `\n`, "package map labels should not contain literal backslash-n")
}

func TestFormatPkgLabel(t *testing.T) {
	// formatPkgLabel is unexported, so we test it indirectly via BuildSlides
	// which calls generatePackageMapMermaid → renderTree → formatPkgLabel.
	pkg := "example.com/proj/mypkg"
	diagOpts := diagram.DiagramOptions{MaxMethodsPerBox: 5}
	splitter := split.NewHubAndSpoke(split.Options{HubThreshold: 3, ChunkSize: 3})
	slideOpts := diagram.SlideOptions{Threshold: 0} // force splitting to get package map

	t.Run("both_ifaces_and_types", func(t *testing.T) {
		ifaces := []analyzer.InterfaceDef{
			{Name: "A", PkgPath: pkg, PkgName: "mypkg"},
			{Name: "B", PkgPath: pkg, PkgName: "mypkg"},
		}
		typs := []analyzer.TypeDef{
			{Name: "X", PkgPath: pkg, PkgName: "mypkg"},
			{Name: "Y", PkgPath: pkg, PkgName: "mypkg"},
			{Name: "Z", PkgPath: pkg, PkgName: "mypkg"},
		}
		rels := []analyzer.Relation{
			{Type: &typs[0], Interface: &ifaces[0]},
			{Type: &typs[1], Interface: &ifaces[0]},
			{Type: &typs[2], Interface: &ifaces[1]},
		}
		result := &analyzer.Result{
			Interfaces: ifaces,
			Types:      typs,
			Relations:  rels,
		}

		slides := diagram.BuildSlides(result, diagOpts, splitter, slideOpts)
		require.GreaterOrEqual(t, len(slides), 1)
		pkgMap := slides[0].Mermaid

		assert.Contains(t, pkgMap, "mypkg<br/>2 ifaces, 3 types",
			"label should use <br/> for line break with both ifaces and types")
		assert.NotContains(t, pkgMap, `\n`,
			"package map labels should not contain literal backslash-n")
	})

	t.Run("empty_result", func(t *testing.T) {
		// An empty result (no interfaces, no types) produces a bare flowchart
		// with no nodes — formatPkgLabel is never called.
		result := &analyzer.Result{}

		slides := diagram.BuildSlides(result, diagOpts, splitter, slideOpts)
		require.GreaterOrEqual(t, len(slides), 1)
		pkgMap := slides[0].Mermaid

		// With no packages, the label is just "flowchart LR" with no stats
		assert.Contains(t, pkgMap, "flowchart LR")
		assert.NotContains(t, pkgMap, "<br/>",
			"empty result should not produce any label with line break")
	})

	t.Run("only_interfaces", func(t *testing.T) {
		ifaces := []analyzer.InterfaceDef{
			{Name: "A", PkgPath: pkg, PkgName: "mypkg"},
			{Name: "B", PkgPath: pkg, PkgName: "mypkg"},
			{Name: "C", PkgPath: pkg, PkgName: "mypkg"},
		}
		// Need at least one relation and type to trigger splitting
		typs := []analyzer.TypeDef{
			{Name: "X", PkgPath: pkg, PkgName: "mypkg"},
		}
		rels := []analyzer.Relation{
			{Type: &typs[0], Interface: &ifaces[0]},
		}
		// Use a second package that has only interfaces (no types)
		ifacePkg := "example.com/proj/ifonly"
		ifaceOnlyIfaces := []analyzer.InterfaceDef{
			{Name: "P", PkgPath: ifacePkg, PkgName: "ifonly"},
			{Name: "Q", PkgPath: ifacePkg, PkgName: "ifonly"},
		}
		allIfaces := make([]analyzer.InterfaceDef, 0, len(ifaces)+len(ifaceOnlyIfaces))
		allIfaces = append(allIfaces, ifaces...)
		allIfaces = append(allIfaces, ifaceOnlyIfaces...)
		result := &analyzer.Result{
			Interfaces: allIfaces,
			Types:      typs,
			Relations:  rels,
		}

		slides := diagram.BuildSlides(result, diagOpts, splitter, slideOpts)
		require.GreaterOrEqual(t, len(slides), 1)
		pkgMap := slides[0].Mermaid

		assert.Contains(t, pkgMap, "ifonly<br/>2 ifaces",
			"package with only interfaces should show iface count without types")
		assert.NotContains(t, pkgMap, "ifonly<br/>2 ifaces,",
			"should not have trailing comma when there are no types")
	})

	t.Run("only_types", func(t *testing.T) {
		// A package that has only types (no interfaces)
		typePkg := "example.com/proj/typonly"
		ifaces := []analyzer.InterfaceDef{
			{Name: "A", PkgPath: pkg, PkgName: "mypkg"},
		}
		typs := []analyzer.TypeDef{
			{Name: "X", PkgPath: pkg, PkgName: "mypkg"},
			{Name: "T1", PkgPath: typePkg, PkgName: "typonly"},
			{Name: "T2", PkgPath: typePkg, PkgName: "typonly"},
			{Name: "T3", PkgPath: typePkg, PkgName: "typonly"},
		}
		rels := []analyzer.Relation{
			{Type: &typs[0], Interface: &ifaces[0]},
		}
		result := &analyzer.Result{
			Interfaces: ifaces,
			Types:      typs,
			Relations:  rels,
		}

		slides := diagram.BuildSlides(result, diagOpts, splitter, slideOpts)
		require.GreaterOrEqual(t, len(slides), 1)
		pkgMap := slides[0].Mermaid

		assert.Contains(t, pkgMap, "typonly<br/>3 types",
			"package with only types should show type count without ifaces")
		assert.NotContains(t, pkgMap, `\n`,
			"package map labels should not contain literal backslash-n")
	})
}

func TestOrphanedInterfacesRemovedFromSlides(t *testing.T) {
	// Synthetic data: 2 hub interfaces, 3 types split across 2 slides.
	// Interface A connects to X, Y, Z. Interface B connects only to Z.
	// Slide 1 has X, Y → A should appear, B should NOT (orphaned).
	// Slide 2 has Z → both A and B should appear.
	pkg := "test"
	makeIface := func(name string) analyzer.InterfaceDef {
		return analyzer.InterfaceDef{Name: name, PkgPath: pkg, PkgName: pkg}
	}
	makeType := func(name string) analyzer.TypeDef {
		return analyzer.TypeDef{Name: name, PkgPath: pkg, PkgName: pkg}
	}

	ifaceA := makeIface("A")
	ifaceB := makeIface("B")
	typeX := makeType("X")
	typeY := makeType("Y")
	typeZ := makeType("Z")

	rels := []analyzer.Relation{
		{Type: &typeX, Interface: &ifaceA},
		{Type: &typeY, Interface: &ifaceA},
		{Type: &typeZ, Interface: &ifaceA},
		{Type: &typeZ, Interface: &ifaceB},
	}

	result := &analyzer.Result{
		Interfaces: []analyzer.InterfaceDef{ifaceA, ifaceB},
		Types:      []analyzer.TypeDef{typeX, typeY, typeZ},
		Relations:  rels,
	}

	// With HubThreshold=1, both A (3 connections) and B (1 connection) qualify as
	// hubs (>= 1), so both appear in HubKeys on every detail slide.
	// The post-filter in subResultForSplitGroup removes B from slides where none of
	// B's implementing types (only Z) are present.
	diagOpts := diagram.DiagramOptions{MaxMethodsPerBox: 5}
	splitter := split.NewHubAndSpoke(split.Options{HubThreshold: 1, ChunkSize: 2})
	slideOpts := diagram.SlideOptions{Threshold: 0} // force splitting

	slides := diagram.BuildSlides(result, diagOpts, splitter, slideOpts)

	// Should have package map + at least 2 detail slides
	require.GreaterOrEqual(t, len(slides), 3,
		"expected package map + at least 2 detail slides")
	assert.Equal(t, "Package Map", slides[0].Title)

	// For each detail slide, verify orphaned interfaces are removed
	for i := 1; i < len(slides); i++ {
		mermaid := slides[i].Mermaid

		// If B's only type (Z) is not on this slide, B should be absent
		if !strings.Contains(mermaid, "test_Z") {
			assert.NotContains(t, mermaid, "test_B",
				"slide %d: B has no implementing types, should be absent", i)
		} else {
			assert.Contains(t, mermaid, "test_B",
				"slide %d: Z is present, so B should be present", i)
		}

		// A connects to X, Y, Z — if none are present, A should be absent
		hasAType := strings.Contains(mermaid, "test_X") ||
			strings.Contains(mermaid, "test_Y") ||
			strings.Contains(mermaid, "test_Z")
		if hasAType {
			assert.Contains(t, mermaid, "test_A",
				"slide %d: has A's implementing types, so A should be present", i)
		} else {
			assert.NotContains(t, mermaid, "test_A",
				"slide %d: no types for A, so A should be absent", i)
		}
	}
}

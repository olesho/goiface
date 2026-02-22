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

	// Package map should have pastel color definitions
	assert.Contains(t, pkgMap, "classDef pkgColor0", "package map should define color classes")
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

	// Should show packages with relative paths
	assert.Contains(t, pkgMap, "io", "should contain io package")
	assert.Contains(t, pkgMap, "http", "should contain http package")
	assert.Contains(t, pkgMap, "http/router", "leaf node inside subgraph should show full relative path")

	// Should show counts
	assert.Contains(t, pkgMap, "2 ifaces", "io should show 2 interfaces")
	assert.Contains(t, pkgMap, "1 ifaces", "http should show 1 interface")

	// Should have color definitions and style assignments
	assert.Contains(t, pkgMap, "classDef pkgColor0", "should define color class 0")
	assert.Contains(t, pkgMap, "classDef pkgColor1", "should define color class 1")
	assert.True(t,
		strings.Contains(pkgMap, "style ") || strings.Contains(pkgMap, "class "),
		"should apply colors via style or class statements")

	// Labels must use real newline for line breaks (not <br/> which gets HTML-escaped)
	assert.NotContains(t, pkgMap, `\n`, "package map labels should not contain literal backslash-n")
	assert.NotContains(t, pkgMap, "<br/>", "package map labels should use newline, not <br/>")
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

		assert.Contains(t, pkgMap, "mypkg\n2 ifaces, 3 types",
			"label should use newline for line break with both ifaces and types")
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

		assert.Contains(t, pkgMap, "ifonly\n2 ifaces",
			"package with only interfaces should show iface count without types")
		assert.NotContains(t, pkgMap, "ifonly\n2 ifaces,",
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

		assert.Contains(t, pkgMap, "typonly\n3 types",
			"package with only types should show type count without ifaces")
		assert.NotContains(t, pkgMap, `\n`,
			"package map labels should not contain literal backslash-n")
	})
}

func TestPackageMapRelativePaths(t *testing.T) {
	// Deeply nested packages should show full relative paths in node labels.
	ifaces := []analyzer.InterfaceDef{
		{Name: "Handler", PkgPath: "example.com/app/internal/http/middleware/auth", PkgName: "auth"},
		{Name: "Store", PkgPath: "example.com/app/internal/db", PkgName: "db"},
	}
	typs := []analyzer.TypeDef{
		{Name: "JWTAuth", PkgPath: "example.com/app/internal/http/middleware/auth", PkgName: "auth"},
		{Name: "PGStore", PkgPath: "example.com/app/internal/db", PkgName: "db"},
	}
	rels := []analyzer.Relation{
		{Type: &typs[0], Interface: &ifaces[0]},
		{Type: &typs[1], Interface: &ifaces[1]},
	}
	result := &analyzer.Result{
		Interfaces: ifaces,
		Types:      typs,
		Relations:  rels,
	}

	diagOpts := diagram.DiagramOptions{MaxMethodsPerBox: 5}
	splitter := split.NewHubAndSpoke(split.Options{HubThreshold: 3, ChunkSize: 3})
	slideOpts := diagram.SlideOptions{Threshold: 0}
	slides := diagram.BuildSlides(result, diagOpts, splitter, slideOpts)

	require.GreaterOrEqual(t, len(slides), 1)
	pkgMap := slides[0].Mermaid

	// Deeply nested package should show full relative path
	assert.Contains(t, pkgMap, "http/middleware/auth",
		"deeply nested package should show full relative path")
	assert.Contains(t, pkgMap, "db",
		"single-segment package should show its name")

	// Subgraph titles must use only the short segment name, not the full relative path
	for line := range strings.SplitSeq(pkgMap, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "subgraph ") {
			assert.NotContains(t, trimmed, "/",
				"subgraph titles should be short segments, not full relative paths: %q", trimmed)
		}
	}
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

func TestFilterBySelection(t *testing.T) {
	// Build synthetic data: 2 types (A, B), 2 interfaces (I, J).
	// A implements I and J. B implements J.
	pkg := "test"
	ifaceI := analyzer.InterfaceDef{Name: "I", PkgPath: pkg, PkgName: pkg}
	ifaceJ := analyzer.InterfaceDef{Name: "J", PkgPath: pkg, PkgName: pkg}
	typeA := analyzer.TypeDef{Name: "A", PkgPath: pkg, PkgName: pkg}
	typeB := analyzer.TypeDef{Name: "B", PkgPath: pkg, PkgName: pkg}
	typeC := analyzer.TypeDef{Name: "C", PkgPath: pkg, PkgName: pkg} // orphan: no relations

	rels := []analyzer.Relation{
		{Type: &typeA, Interface: &ifaceI},
		{Type: &typeA, Interface: &ifaceJ},
		{Type: &typeB, Interface: &ifaceJ},
	}

	result := &analyzer.Result{
		Interfaces: []analyzer.InterfaceDef{ifaceI, ifaceJ},
		Types:      []analyzer.TypeDef{typeA, typeB, typeC},
		Relations:  rels,
	}

	t.Run("single_implementation_selected", func(t *testing.T) {
		// Select only A → result has A, I, J and relations A→I, A→J. B excluded.
		filtered := diagram.FilterBySelection(result, []string{"test_A"}, nil)
		assert.Len(t, filtered.Types, 1)
		assert.Equal(t, "A", filtered.Types[0].Name)
		assert.Len(t, filtered.Interfaces, 2) // I and J
		assert.Len(t, filtered.Relations, 2)  // A→I, A→J
	})

	t.Run("single_interface_selected", func(t *testing.T) {
		// Select only I → result has I, A and relation A→I. B and J excluded.
		filtered := diagram.FilterBySelection(result, nil, []string{"test_I"})
		assert.Len(t, filtered.Interfaces, 1)
		assert.Equal(t, "I", filtered.Interfaces[0].Name)
		assert.Len(t, filtered.Types, 1)
		assert.Equal(t, "A", filtered.Types[0].Name)
		assert.Len(t, filtered.Relations, 1)
	})

	t.Run("multiple_selections_union", func(t *testing.T) {
		// Select A and J → union: A→I, A→J, B→J. Result has A, B, I, J.
		filtered := diagram.FilterBySelection(result, []string{"test_A"}, []string{"test_J"})
		assert.Len(t, filtered.Types, 2)      // A and B
		assert.Len(t, filtered.Interfaces, 2) // I and J
		assert.Len(t, filtered.Relations, 3)  // A→I, A→J, B→J
	})

	t.Run("empty_selection", func(t *testing.T) {
		// Select nothing → empty result.
		filtered := diagram.FilterBySelection(result, nil, nil)
		assert.Empty(t, filtered.Interfaces)
		assert.Empty(t, filtered.Types)
		assert.Empty(t, filtered.Relations)
	})

	t.Run("orphan_selection", func(t *testing.T) {
		// Select C (no relations) → result has only C, no interfaces, no relations.
		filtered := diagram.FilterBySelection(result, []string{"test_C"}, nil)
		assert.Len(t, filtered.Types, 1)
		assert.Equal(t, "C", filtered.Types[0].Name)
		assert.Empty(t, filtered.Interfaces)
		assert.Empty(t, filtered.Relations)
	})

	t.Run("cross_tab_selections", func(t *testing.T) {
		// Select type B + interface I → union: A→I (from I), B→J (from B)
		filtered := diagram.FilterBySelection(result, []string{"test_B"}, []string{"test_I"})
		assert.Len(t, filtered.Relations, 2) // A→I, B→J
		// Types: A (via I), B (selected)
		assert.Len(t, filtered.Types, 2)
		// Interfaces: I (selected), J (via B)
		assert.Len(t, filtered.Interfaces, 2)
	})
}

func TestPrepareInteractiveData(t *testing.T) {
	pkg := "test"
	iface := analyzer.InterfaceDef{
		Name: "MyIface", PkgPath: pkg, PkgName: pkg,
		Methods:    []analyzer.MethodSig{{Name: "Do", Signature: "Do(ctx context.Context) error"}},
		SourceFile: "iface.go",
	}
	typ := analyzer.TypeDef{
		Name: "MyType", PkgPath: pkg, PkgName: pkg,
		SourceFile: "type.go",
	}
	result := &analyzer.Result{
		Interfaces: []analyzer.InterfaceDef{iface},
		Types:      []analyzer.TypeDef{typ},
		Relations:  []analyzer.Relation{{Type: &typ, Interface: &iface}},
	}

	data := diagram.PrepareInteractiveData(result, diagram.DiagramOptions{MaxMethodsPerBox: 5})

	require.Len(t, data.Interfaces, 1)
	assert.Equal(t, "test_MyIface", data.Interfaces[0].ID)
	assert.Equal(t, "test.MyIface", data.Interfaces[0].Name)
	assert.Equal(t, "test", data.Interfaces[0].PkgName)
	require.Len(t, data.Interfaces[0].Methods, 1)
	assert.Equal(t, "Do(ctx context.Context) error", data.Interfaces[0].Methods[0])

	require.Len(t, data.Types, 1)
	assert.Equal(t, "test_MyType", data.Types[0].ID)

	require.Len(t, data.Relations, 1)
	assert.Equal(t, "test_MyType", data.Relations[0].TypeID)
	assert.Equal(t, "test_MyIface", data.Relations[0].InterfaceID)
}

func TestNodeIDExported(t *testing.T) {
	assert.Equal(t, "pkg_MyType", diagram.NodeID("pkg", "MyType"))
	assert.Equal(t, "my_pkg_MyType", diagram.NodeID("my-pkg", "MyType"))
}

func TestSanitizeSignatureExported(t *testing.T) {
	assert.Equal(t, "Do(x any) error", diagram.SanitizeSignature("Do(x interface{}) error"))
	assert.Equal(t, "Chan(ch chan int)", diagram.SanitizeSignature("Chan(ch <-chan int)"))
}

func TestOrphanedTypesRemovedFromSlides(t *testing.T) {
	// Scenario: type W only implements non-hub interface C. C is attached to
	// a different chunk (the one containing Y, which also implements C). W is
	// chunked separately and ends up on a slide where C is absent, leaving W
	// with no relations — it should be pruned.
	//
	// Setup:
	//   Hub A (3 connections: X, Y, Z — meets threshold=3)
	//   Non-hub C (2 connections: Y, W — below threshold)
	//   Types sorted: W, X, Y, Z → chunk1=[W,X], chunk2=[Y,Z]
	//   C attached to chunk2 (Y is there, first match)
	//   chunk1: HubKeys=[A], SpokeKeys=[W,X]
	//     W→C: C not in HubKeys → relation dropped → W is orphaned type
	pkg := "test"
	makeIface := func(name string) analyzer.InterfaceDef {
		return analyzer.InterfaceDef{Name: name, PkgPath: pkg, PkgName: pkg}
	}
	makeType := func(name string) analyzer.TypeDef {
		return analyzer.TypeDef{Name: name, PkgPath: pkg, PkgName: pkg}
	}

	ifaceA := makeIface("A")
	ifaceC := makeIface("C")
	typeW := makeType("W")
	typeX := makeType("X")
	typeY := makeType("Y")
	typeZ := makeType("Z")

	rels := []analyzer.Relation{
		{Type: &typeX, Interface: &ifaceA},
		{Type: &typeY, Interface: &ifaceA},
		{Type: &typeZ, Interface: &ifaceA},
		{Type: &typeY, Interface: &ifaceC}, // C connects to Y (chunk2)
		{Type: &typeW, Interface: &ifaceC}, // W only implements C
	}

	result := &analyzer.Result{
		Interfaces: []analyzer.InterfaceDef{ifaceA, ifaceC},
		Types:      []analyzer.TypeDef{typeW, typeX, typeY, typeZ},
		Relations:  rels,
	}

	diagOpts := diagram.DiagramOptions{MaxMethodsPerBox: 5}
	splitter := split.NewHubAndSpoke(split.Options{HubThreshold: 3, ChunkSize: 2})
	slideOpts := diagram.SlideOptions{Threshold: 0}

	slides := diagram.BuildSlides(result, diagOpts, splitter, slideOpts)

	require.GreaterOrEqual(t, len(slides), 3,
		"expected package map + at least 2 detail slides")

	for i := 1; i < len(slides); i++ {
		mermaid := slides[i].Mermaid

		// Every type on a slide should have at least one relation on that slide.
		// Check W specifically: if C is not on this slide, W should be absent.
		if !strings.Contains(mermaid, "test_C") && !strings.Contains(mermaid, "test_A") {
			// No interfaces at all → no types should appear
			assert.NotContains(t, mermaid, "test_W",
				"slide %d: no interfaces present, W should be absent", i)
		}

		// More general check: any type that appears should have a relation arrow
		for _, typeName := range []string{"W", "X", "Y", "Z"} {
			nodeID := "test_" + typeName
			if strings.Contains(mermaid, "class "+nodeID) {
				assert.Contains(t, mermaid, nodeID+" --|>",
					"slide %d: type %s appears but has no outgoing relation (orphaned)", i, typeName)
			}
		}
	}
}

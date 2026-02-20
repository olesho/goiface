package split

import (
	"sort"
	"testing"

	"github.com/olehluchkiv/goifaces/internal/analyzer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeIface creates a minimal InterfaceDef for testing.
func makeIface(name, pkg string) analyzer.InterfaceDef {
	return analyzer.InterfaceDef{Name: name, PkgPath: pkg, PkgName: pkg}
}

// makeType creates a minimal TypeDef for testing.
func makeType(name, pkg string) analyzer.TypeDef {
	return analyzer.TypeDef{Name: name, PkgPath: pkg, PkgName: pkg}
}

// buildResult creates a Result from interface/type names and relation pairs.
// Relations are specified as "TypeName->IfaceName".
func buildResult(ifaces []analyzer.InterfaceDef, types []analyzer.TypeDef, rels [][2]string) *analyzer.Result {
	ifaceMap := make(map[string]*analyzer.InterfaceDef)
	for i := range ifaces {
		ifaceMap[ifaces[i].PkgPath+"."+ifaces[i].Name] = &ifaces[i]
	}
	typeMap := make(map[string]*analyzer.TypeDef)
	for i := range types {
		typeMap[types[i].PkgPath+"."+types[i].Name] = &types[i]
	}

	var relations []analyzer.Relation
	for _, pair := range rels {
		t := typeMap[pair[0]]
		iface := ifaceMap[pair[1]]
		if t != nil && iface != nil {
			relations = append(relations, analyzer.Relation{Type: t, Interface: iface})
		}
	}

	return &analyzer.Result{
		Interfaces: ifaces,
		Types:      types,
		Relations:  relations,
	}
}

func TestHubSpoke_GoMemDBLike(t *testing.T) {
	// Simulate go-memdb: 4 hub interfaces, 11 field index types + 1 FilterIterator
	pkg := "memdb"
	ifaces := []analyzer.InterfaceDef{
		makeIface("Indexer", pkg),
		makeIface("MultiIndexer", pkg),
		makeIface("PrefixIndexer", pkg),
		makeIface("ResultIterator", pkg),
		makeIface("SingleIndexer", pkg),
	}

	typeNames := []string{
		"BoolFieldIndex", "CompoundIndex", "CompoundMultiIndex",
		"ConditionalIndex", "FieldSetIndex", "FilterIterator",
		"IntFieldIndex", "StringFieldIndex", "StringMapFieldIndex",
		"StringSliceFieldIndex", "UUIDFieldIndex", "UintFieldIndex",
	}
	var types []analyzer.TypeDef
	for _, name := range typeNames {
		types = append(types, makeType(name, pkg))
	}

	// Each field index type connects to Indexer, MultiIndexer, SingleIndexer (3 hubs)
	// Some also connect to PrefixIndexer
	var rels [][2]string
	fieldIndexTypes := []string{
		"BoolFieldIndex", "CompoundIndex", "CompoundMultiIndex",
		"ConditionalIndex", "FieldSetIndex", "IntFieldIndex",
		"StringFieldIndex", "StringMapFieldIndex", "StringSliceFieldIndex",
		"UUIDFieldIndex", "UintFieldIndex",
	}
	for _, name := range fieldIndexTypes {
		rels = append(rels,
			[2]string{pkg + "." + name, pkg + ".Indexer"},
			[2]string{pkg + "." + name, pkg + ".MultiIndexer"},
			[2]string{pkg + "." + name, pkg + ".SingleIndexer"},
		)
	}
	// PrefixIndexer has 4 connections (String* types)
	for _, name := range []string{"StringFieldIndex", "StringMapFieldIndex", "StringSliceFieldIndex", "CompoundIndex"} {
		rels = append(rels, [2]string{pkg + "." + name, pkg + ".PrefixIndexer"})
	}
	// FilterIterator only connects to ResultIterator
	rels = append(rels, [2]string{pkg + ".FilterIterator", pkg + ".ResultIterator"})

	result := buildResult(ifaces, types, rels)
	splitter := NewHubAndSpoke(Options{HubThreshold: 3, ChunkSize: 3})
	groups := splitter.Split(result)

	// Should produce 4 groups (12 spokes / 3 chunk size)
	require.Equal(t, 4, len(groups))

	// Verify all hub interfaces appear on every slide
	hubKey := func(name string) string { return pkg + "." + name }
	expectedHubs := []string{
		hubKey("Indexer"), hubKey("MultiIndexer"),
		hubKey("PrefixIndexer"), hubKey("SingleIndexer"),
	}

	for i, g := range groups {
		for _, hub := range expectedHubs {
			assert.Contains(t, g.HubKeys, hub, "group %d missing hub %s", i, hub)
		}
		assert.LessOrEqual(t, len(g.SpokeKeys), 3, "group %d has too many spokes", i)
		assert.Greater(t, len(g.SpokeKeys), 0, "group %d has no spokes", i)
	}

	// Verify FilterIterator and ResultIterator are on the same slide
	var filterSlide int
	for i, g := range groups {
		for _, k := range g.SpokeKeys {
			if k == pkg+".FilterIterator" {
				filterSlide = i
			}
		}
	}
	// ResultIterator is a non-hub interface, should be in HubKeys of FilterIterator's slide
	assert.Contains(t, groups[filterSlide].HubKeys, pkg+".ResultIterator",
		"ResultIterator should be on the same slide as FilterIterator")

	// Verify each spoke appears exactly once across all groups
	allSpokes := make(map[string]int)
	for _, g := range groups {
		for _, k := range g.SpokeKeys {
			allSpokes[k]++
		}
	}
	assert.Equal(t, 12, len(allSpokes), "all 12 spokes should be present")
	for k, count := range allSpokes {
		assert.Equal(t, 1, count, "spoke %s should appear exactly once", k)
	}
}

func TestHubSpoke_SmallGraph(t *testing.T) {
	// 2 interfaces, 2 types, each type implements 1 interface
	// All have connection count 1 — no hubs, single group
	pkg := "small"
	ifaces := []analyzer.InterfaceDef{
		makeIface("Reader", pkg),
		makeIface("Writer", pkg),
	}
	types := []analyzer.TypeDef{
		makeType("FileReader", pkg),
		makeType("FileWriter", pkg),
	}
	rels := [][2]string{
		{pkg + ".FileReader", pkg + ".Reader"},
		{pkg + ".FileWriter", pkg + ".Writer"},
	}

	result := buildResult(ifaces, types, rels)
	splitter := NewHubAndSpoke(Options{HubThreshold: 3, ChunkSize: 3})
	groups := splitter.Split(result)

	// Only 2 spokes, chunk size 3 → 1 group
	require.Equal(t, 1, len(groups))
	assert.Equal(t, 2, len(groups[0].SpokeKeys))

	// No hubs (all connections < 3), but non-hub interfaces should be attached
	// Both Reader and Writer should be in hub keys of the single chunk
	assert.Contains(t, groups[0].HubKeys, pkg+".Reader")
	assert.Contains(t, groups[0].HubKeys, pkg+".Writer")
}

func TestHubSpoke_IsolatedCluster(t *testing.T) {
	// One hub interface with 3 types, plus an isolated pair (ResultIterator + FilterIterator)
	pkg := "iso"
	ifaces := []analyzer.InterfaceDef{
		makeIface("Indexer", pkg),
		makeIface("ResultIterator", pkg),
	}
	types := []analyzer.TypeDef{
		makeType("Alpha", pkg),
		makeType("Beta", pkg),
		makeType("FilterIterator", pkg),
		makeType("Gamma", pkg),
	}
	rels := [][2]string{
		{pkg + ".Alpha", pkg + ".Indexer"},
		{pkg + ".Beta", pkg + ".Indexer"},
		{pkg + ".Gamma", pkg + ".Indexer"},
		{pkg + ".FilterIterator", pkg + ".ResultIterator"},
	}

	result := buildResult(ifaces, types, rels)
	splitter := NewHubAndSpoke(Options{HubThreshold: 3, ChunkSize: 3})
	groups := splitter.Split(result)

	// 4 spokes / 3 = 2 groups
	require.Equal(t, 2, len(groups))

	// Find FilterIterator's group
	var filterGroup *Group
	for i := range groups {
		for _, k := range groups[i].SpokeKeys {
			if k == pkg+".FilterIterator" {
				filterGroup = &groups[i]
			}
		}
	}
	require.NotNil(t, filterGroup, "FilterIterator should be in some group")
	assert.Contains(t, filterGroup.HubKeys, pkg+".ResultIterator",
		"ResultIterator should be attached to FilterIterator's group")

	// Indexer is a hub (3 connections), should be in all groups
	for i, g := range groups {
		assert.Contains(t, g.HubKeys, pkg+".Indexer",
			"group %d should have hub Indexer", i)
	}
}

func TestHubSpoke_AllHubs(t *testing.T) {
	// All interfaces are hubs (each connected to 3+ types)
	// All types are spokes
	pkg := "allhub"
	ifaces := []analyzer.InterfaceDef{
		makeIface("A", pkg),
		makeIface("B", pkg),
	}
	types := []analyzer.TypeDef{
		makeType("X", pkg),
		makeType("Y", pkg),
		makeType("Z", pkg),
	}
	// Each type implements both interfaces → A(3), B(3) → both are hubs
	rels := [][2]string{
		{pkg + ".X", pkg + ".A"}, {pkg + ".X", pkg + ".B"},
		{pkg + ".Y", pkg + ".A"}, {pkg + ".Y", pkg + ".B"},
		{pkg + ".Z", pkg + ".A"}, {pkg + ".Z", pkg + ".B"},
	}

	result := buildResult(ifaces, types, rels)
	splitter := NewHubAndSpoke(Options{HubThreshold: 3, ChunkSize: 3})
	groups := splitter.Split(result)

	// 3 spokes / 3 = 1 group
	require.Equal(t, 1, len(groups))
	assert.Equal(t, 3, len(groups[0].SpokeKeys))

	// Both interfaces are hubs
	hubKeys := groups[0].HubKeys
	sort.Strings(hubKeys)
	assert.Equal(t, []string{pkg + ".A", pkg + ".B"}, hubKeys)
}

func TestHubSpoke_NoTypes(t *testing.T) {
	// Only interfaces, no types → single group with all as hubs
	pkg := "notypes"
	ifaces := []analyzer.InterfaceDef{
		makeIface("A", pkg),
		makeIface("B", pkg),
	}

	result := buildResult(ifaces, nil, nil)
	splitter := NewHubAndSpoke(Options{HubThreshold: 3, ChunkSize: 3})
	groups := splitter.Split(result)

	require.Equal(t, 1, len(groups))
	assert.Equal(t, 0, len(groups[0].SpokeKeys))
	assert.Equal(t, 2, len(groups[0].HubKeys))
}

func TestHubSpoke_EmptyResult(t *testing.T) {
	result := &analyzer.Result{}
	splitter := NewHubAndSpoke(Options{HubThreshold: 3, ChunkSize: 3})
	groups := splitter.Split(result)

	assert.Nil(t, groups)
}

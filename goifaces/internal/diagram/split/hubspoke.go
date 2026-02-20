package split

import (
	"sort"
	"strings"

	"github.com/olehluchkiv/goifaces/internal/analyzer"
)

// HubAndSpoke implements the hub-and-spoke splitting strategy.
// High-connectivity interfaces (hubs) repeat on every detail slide,
// while implementations (spokes) are chunked into groups of ChunkSize.
type HubAndSpoke struct {
	opts Options
}

// NewHubAndSpoke creates a hub-and-spoke splitter with the given options.
func NewHubAndSpoke(opts Options) *HubAndSpoke {
	if opts.HubThreshold <= 0 {
		opts.HubThreshold = DefaultOptions().HubThreshold
	}
	if opts.ChunkSize <= 0 {
		opts.ChunkSize = DefaultOptions().ChunkSize
	}
	return &HubAndSpoke{opts: opts}
}

// Split implements Splitter. It identifies hub interfaces (those with
// connections >= HubThreshold), then chunks the remaining spokes into
// groups of ChunkSize. Non-hub interfaces are attached to whichever
// chunk contains their connected types.
func (h *HubAndSpoke) Split(result *analyzer.Result) []Group {
	connCount := connectionCount(result)

	// Classify interfaces as hubs or non-hubs.
	hubIfaceKeys := make(map[string]bool)
	nonHubIfaceKeys := make(map[string]bool)
	for _, iface := range result.Interfaces {
		key := typeKey(iface.PkgPath, iface.Name)
		if connCount[key] >= h.opts.HubThreshold {
			hubIfaceKeys[key] = true
		} else {
			nonHubIfaceKeys[key] = true
		}
	}

	// Spokes are types (not interfaces). Collect and sort alphabetically.
	var spokeKeys []string
	for _, typ := range result.Types {
		key := typeKey(typ.PkgPath, typ.Name)
		spokeKeys = append(spokeKeys, key)
	}
	sort.Strings(spokeKeys)

	// If there are no spokes, return a single group with all hubs.
	if len(spokeKeys) == 0 {
		var allKeys []string
		for k := range hubIfaceKeys {
			allKeys = append(allKeys, k)
		}
		for k := range nonHubIfaceKeys {
			allKeys = append(allKeys, k)
		}
		sort.Strings(allKeys)
		if len(allKeys) == 0 {
			return nil
		}
		return []Group{{
			Title:   "All Interfaces",
			HubKeys: allKeys,
		}}
	}

	// Build sorted hub key list for consistent ordering.
	sortedHubKeys := sortedKeys(hubIfaceKeys)

	// Build index: which types does each non-hub interface connect to?
	nonHubIfaceToTypes := make(map[string]map[string]bool)
	for _, rel := range result.Relations {
		ik := typeKey(rel.Interface.PkgPath, rel.Interface.Name)
		tk := typeKey(rel.Type.PkgPath, rel.Type.Name)
		if nonHubIfaceKeys[ik] {
			if nonHubIfaceToTypes[ik] == nil {
				nonHubIfaceToTypes[ik] = make(map[string]bool)
			}
			nonHubIfaceToTypes[ik][tk] = true
		}
	}

	// Chunk spokes into groups of ChunkSize.
	chunks := chunkSlice(spokeKeys, h.opts.ChunkSize)

	var groups []Group
	attachedNonHubIfaces := make(map[string]bool)

	for _, chunk := range chunks {
		chunkSet := make(map[string]bool, len(chunk))
		for _, k := range chunk {
			chunkSet[k] = true
		}

		// Find non-hub interfaces connected to types in this chunk.
		var extraIfaceKeys []string
		for ik, connectedTypes := range nonHubIfaceToTypes {
			if attachedNonHubIfaces[ik] {
				continue
			}
			for tk := range connectedTypes {
				if chunkSet[tk] {
					extraIfaceKeys = append(extraIfaceKeys, ik)
					attachedNonHubIfaces[ik] = true
					break
				}
			}
		}
		sort.Strings(extraIfaceKeys)

		// Hub keys = all hubs + any non-hub interfaces attached to this chunk.
		hubKeys := make([]string, len(sortedHubKeys))
		copy(hubKeys, sortedHubKeys)
		hubKeys = append(hubKeys, extraIfaceKeys...)

		// Build title from spoke names (just the Name part, not full key).
		title := buildTitle(chunk)

		groups = append(groups, Group{
			Title:     title,
			HubKeys:   hubKeys,
			SpokeKeys: chunk,
		})
	}

	// If any non-hub interfaces were not attached to any chunk (no connected types),
	// add them as hubs to the first group.
	for ik := range nonHubIfaceKeys {
		if !attachedNonHubIfaces[ik] && len(groups) > 0 {
			groups[0].HubKeys = append(groups[0].HubKeys, ik)
		}
	}

	return groups
}

// connectionCount counts connections (relations) per node key.
func connectionCount(result *analyzer.Result) map[string]int {
	counts := make(map[string]int)
	for _, rel := range result.Relations {
		ik := typeKey(rel.Interface.PkgPath, rel.Interface.Name)
		tk := typeKey(rel.Type.PkgPath, rel.Type.Name)
		counts[ik]++
		counts[tk]++
	}
	return counts
}

// typeKey builds a unique key from package path and name.
func typeKey(pkgPath, name string) string {
	return pkgPath + "." + name
}

// chunkSlice splits a slice into chunks of at most size n.
func chunkSlice(items []string, n int) [][]string {
	var chunks [][]string
	for i := 0; i < len(items); i += n {
		end := i + n
		if end > len(items) {
			end = len(items)
		}
		chunks = append(chunks, items[i:end])
	}
	return chunks
}

// sortedKeys returns sorted keys from a bool map.
func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// buildTitle extracts the Name part from keys (pkgPath.Name) and joins them.
func buildTitle(keys []string) string {
	names := make([]string, len(keys))
	for i, k := range keys {
		if idx := strings.LastIndex(k, "."); idx >= 0 {
			names[i] = k[idx+1:]
		} else {
			names[i] = k
		}
	}
	return strings.Join(names, ", ")
}

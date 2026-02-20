package split

import "github.com/olehluchkiv/goifaces/internal/analyzer"

// Group represents one slide's content: hub nodes (repeated on every slide)
// plus spoke nodes (unique to this slide).
type Group struct {
	Title     string
	HubKeys   []string // node keys (pkgPath.Name) for hub nodes
	SpokeKeys []string // node keys for spoke nodes unique to this slide
}

// Splitter splits an analysis result into groups for slide generation.
type Splitter interface {
	Split(result *analyzer.Result) []Group
}

// Options controls splitting behavior.
type Options struct {
	HubThreshold int // min connections to be a hub; default 3
	ChunkSize    int // max spokes per slide; default 3
}

// DefaultOptions returns sensible defaults.
func DefaultOptions() Options {
	return Options{HubThreshold: 3, ChunkSize: 3}
}

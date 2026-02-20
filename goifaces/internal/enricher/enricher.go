package enricher

import "github.com/olehluchkiv/goifaces/internal/analyzer"

// Enricher transforms an analysis result, adding semantic information.
type Enricher interface {
	Enrich(result *analyzer.Result) *analyzer.Result
}

// SemanticGroup represents a logical grouping of interfaces/types.
type SemanticGroup struct {
	Name       string
	Interfaces []string // interface keys (pkg.Name)
	Types      []string // type keys (pkg.Name)
}

// DetectedPattern represents a recognized design pattern.
type DetectedPattern struct {
	Name         string
	Description  string
	Participants []string // type/interface keys involved
}

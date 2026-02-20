package enricher

import "github.com/olehluchkiv/goifaces/internal/analyzer"

// Scorer assigns importance weights to implementation relationships.
type Scorer interface {
	Score(relations []analyzer.Relation) map[int]float64
}

// DefaultScorer gives all relationships equal weight (1.0).
type DefaultScorer struct{}

func NewDefaultScorer() *DefaultScorer { return &DefaultScorer{} }

func (s *DefaultScorer) Enrich(result *analyzer.Result) *analyzer.Result {
	return result
}

func (s *DefaultScorer) Score(relations []analyzer.Relation) map[int]float64 {
	m := make(map[int]float64, len(relations))
	for i := range relations {
		m[i] = 1.0
	}
	return m
}

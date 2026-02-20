package enricher

import "github.com/olehluchkiv/goifaces/internal/analyzer"

// PatternDetector identifies design patterns in the interface graph.
type PatternDetector interface {
	Detect(result *analyzer.Result) []DetectedPattern
}

// DefaultPatternDetector is a no-op (returns empty).
type DefaultPatternDetector struct{}

func NewDefaultPatternDetector() *DefaultPatternDetector { return &DefaultPatternDetector{} }

func (d *DefaultPatternDetector) Enrich(result *analyzer.Result) *analyzer.Result {
	return result
}

func (d *DefaultPatternDetector) Detect(result *analyzer.Result) []DetectedPattern {
	return nil
}

package enricher

import "github.com/olehluchkiv/goifaces/internal/analyzer"

// Annotator adds human-readable descriptions to interfaces and types.
type Annotator interface {
	Annotate(result *analyzer.Result) map[string]string
}

// DefaultAnnotator is a no-op (no annotations).
type DefaultAnnotator struct{}

func NewDefaultAnnotator() *DefaultAnnotator { return &DefaultAnnotator{} }

func (a *DefaultAnnotator) Enrich(result *analyzer.Result) *analyzer.Result {
	return result
}

func (a *DefaultAnnotator) Annotate(_ *analyzer.Result) map[string]string {
	return nil
}

package enricher

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/olehluchkiv/goifaces/internal/analyzer"
	"github.com/olehluchkiv/goifaces/internal/enricher/llm"
)

// LLMAnnotator uses an LLM to generate human-readable descriptions for
// interfaces and types.
type LLMAnnotator struct {
	client   *llm.Client
	fallback *DefaultAnnotator
	logger   *slog.Logger
}

// NewLLMAnnotator creates a new LLM-backed annotator.
func NewLLMAnnotator(client *llm.Client, fallback *DefaultAnnotator, logger *slog.Logger) *LLMAnnotator {
	return &LLMAnnotator{
		client:   client,
		fallback: fallback,
		logger:   logger.With("component", "enricher.llm-annotator"),
	}
}

func (a *LLMAnnotator) Enrich(result *analyzer.Result) *analyzer.Result {
	return result
}

const annotatorSystemPrompt = `You are a Go code documentation specialist. Given Go interfaces, types, and their relationships, generate concise human-readable descriptions explaining each node's architectural role/purpose.

Respond with JSON only:
{
  "annotations": {
    "pkg.Name": "Brief description of purpose (< 80 chars)"
  }
}

Rules:
- Every interface and type should get an annotation
- Keep descriptions under 80 characters
- Focus on the architectural role, not implementation details
- Use the exact node keys from the input`

func (a *LLMAnnotator) Annotate(result *analyzer.Result) map[string]string {
	if len(result.Interfaces) == 0 && len(result.Types) == 0 {
		return a.fallback.Annotate(result)
	}

	prompt := llm.SerializeResult(result)
	resp, err := a.client.Complete(context.Background(), annotatorSystemPrompt, prompt)
	if err != nil {
		a.logger.Warn("LLM annotator failed, using default", "error", err)
		return a.fallback.Annotate(result)
	}

	var parsed struct {
		Annotations map[string]string `json:"annotations"`
	}
	if err := json.Unmarshal([]byte(resp), &parsed); err != nil {
		a.logger.Warn("LLM annotator returned invalid JSON", "error", err)
		return a.fallback.Annotate(result)
	}

	if len(parsed.Annotations) == 0 {
		a.logger.Warn("LLM annotator returned empty annotations")
		return a.fallback.Annotate(result)
	}

	// Validate keys exist in result
	validNodes := make(map[string]bool)
	for _, iface := range result.Interfaces {
		validNodes[iface.PkgPath+"."+iface.Name] = true
	}
	for _, typ := range result.Types {
		validNodes[typ.PkgPath+"."+typ.Name] = true
	}

	annotations := make(map[string]string)
	for key, desc := range parsed.Annotations {
		if validNodes[key] {
			annotations[key] = desc
		}
	}

	if len(annotations) == 0 {
		a.logger.Warn("LLM annotator returned no valid annotations")
		return a.fallback.Annotate(result)
	}

	return annotations
}

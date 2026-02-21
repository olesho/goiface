package enricher

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/olehluchkiv/goifaces/internal/analyzer"
	"github.com/olehluchkiv/goifaces/internal/enricher/llm"
)

const annotatorSystemPrompt = `You are an expert Go software architect. Given a list of Go interfaces and types with their methods and relationships, generate concise human-readable descriptions for each.

Each description should explain the role or purpose of the interface/type in the system architecture.

Respond with JSON only.`

const annotatorUserPrompt = `Generate brief descriptions for these Go interfaces and types:

%s

Return JSON with this schema:
{"annotations": {"pkg.Name": "Brief description of purpose"}}

Rules:
- Each description must be under 80 characters
- Focus on architectural role, not implementation details
- Cover all interfaces and types from the input`

// LLMAnnotator uses an LLM to generate human-readable descriptions.
type LLMAnnotator struct {
	ctx      context.Context
	client   *llm.Client
	fallback *DefaultAnnotator
	logger   *slog.Logger
}

// NewLLMAnnotator creates an LLM-backed annotator.
func NewLLMAnnotator(ctx context.Context, client *llm.Client, fallback *DefaultAnnotator, logger *slog.Logger) *LLMAnnotator {
	return &LLMAnnotator{
		ctx:      ctx,
		client:   client,
		fallback: fallback,
		logger:   logger.With("component", "llm-annotator"),
	}
}

func (a *LLMAnnotator) Enrich(result *analyzer.Result) *analyzer.Result {
	return result
}

func (a *LLMAnnotator) Annotate(result *analyzer.Result) map[string]string {
	if len(result.Interfaces) == 0 && len(result.Types) == 0 {
		return a.fallback.Annotate(result)
	}

	filtered := llm.PreFilterByEdgeCount(result, 100)
	prompt := llm.SerializeResult(filtered)

	raw, err := a.client.Complete(a.ctx, annotatorSystemPrompt, fmt.Sprintf(annotatorUserPrompt, prompt))
	if err != nil {
		a.logger.Warn("LLM annotator failed, using default", "error", err)
		return a.fallback.Annotate(result)
	}

	var resp struct {
		Annotations map[string]string `json:"annotations"`
	}
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		a.logger.Warn("LLM annotator returned invalid JSON, using default", "error", err)
		return a.fallback.Annotate(result)
	}

	if len(resp.Annotations) == 0 {
		a.logger.Warn("LLM annotator returned no annotations, using default")
		return a.fallback.Annotate(result)
	}

	// Validate keys against actual nodes
	known := make(map[string]bool)
	for _, iface := range result.Interfaces {
		known[iface.PkgPath+"."+iface.Name] = true
	}
	for _, typ := range result.Types {
		known[typ.PkgPath+"."+typ.Name] = true
	}

	annotations := make(map[string]string)
	for key, desc := range resp.Annotations {
		if known[key] {
			annotations[key] = desc
		}
	}

	if len(annotations) == 0 {
		a.logger.Warn("LLM annotator returned no valid annotations, using default")
		return a.fallback.Annotate(result)
	}

	return annotations
}

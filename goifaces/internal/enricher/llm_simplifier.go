package enricher

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"

	"github.com/olehluchkiv/goifaces/internal/analyzer"
	"github.com/olehluchkiv/goifaces/internal/enricher/llm"
)

// LLMSimplifier uses an LLM to intelligently select the most architecturally
// significant nodes to keep when simplifying a diagram.
type LLMSimplifier struct {
	client   *llm.Client
	fallback *DefaultSimplifier
	logger   *slog.Logger
}

// NewLLMSimplifier creates a new LLM-backed simplifier.
func NewLLMSimplifier(client *llm.Client, fallback *DefaultSimplifier, logger *slog.Logger) *LLMSimplifier {
	return &LLMSimplifier{
		client:   client,
		fallback: fallback,
		logger:   logger.With("component", "enricher.llm-simplifier"),
	}
}

func (s *LLMSimplifier) Enrich(result *analyzer.Result) *analyzer.Result {
	if s.fallback.MaxNodes <= 0 {
		return result
	}
	return s.Simplify(result, s.fallback.MaxNodes)
}

const simplifierSystemPrompt = `You are a Go architecture analyst. Given a graph of Go interfaces and types, select the most architecturally significant nodes to keep for a simplified diagram.

Respond with JSON only:
{
  "keep": ["pkg.Name1", "pkg.Name2"]
}

Rules:
- Keep exactly the number of nodes requested (or fewer if the total is smaller)
- Prefer hub interfaces that many types implement
- Prefer types that bridge different architectural domains
- Keep nodes that preserve the most informative relationships
- Use the exact node keys from the input`

func (s *LLMSimplifier) Simplify(result *analyzer.Result, maxNodes int) *analyzer.Result {
	totalNodes := len(result.Interfaces) + len(result.Types)
	if totalNodes <= maxNodes {
		return result
	}

	if len(result.Relations) == 0 {
		return s.fallback.Simplify(result, maxNodes)
	}

	prompt := llm.SerializeResult(result) +
		"\n\nSelect the " + itoa(maxNodes) + " most architecturally significant nodes to keep."

	resp, err := s.client.Complete(context.Background(), simplifierSystemPrompt, prompt)
	if err != nil {
		s.logger.Warn("LLM simplifier failed, using default", "error", err)
		return s.fallback.Simplify(result, maxNodes)
	}

	var parsed struct {
		Keep []string `json:"keep"`
	}
	if err := json.Unmarshal([]byte(resp), &parsed); err != nil {
		s.logger.Warn("LLM simplifier returned invalid JSON", "error", err)
		return s.fallback.Simplify(result, maxNodes)
	}

	if len(parsed.Keep) == 0 {
		s.logger.Warn("LLM simplifier returned empty keep list")
		return s.fallback.Simplify(result, maxNodes)
	}

	keep := make(map[string]bool)
	for _, key := range parsed.Keep {
		keep[key] = true
	}

	out := &analyzer.Result{}
	for _, iface := range result.Interfaces {
		if keep[iface.PkgPath+"."+iface.Name] {
			out.Interfaces = append(out.Interfaces, iface)
		}
	}
	for _, typ := range result.Types {
		if keep[typ.PkgPath+"."+typ.Name] {
			out.Types = append(out.Types, typ)
		}
	}
	for _, rel := range result.Relations {
		tKey := rel.Type.PkgPath + "." + rel.Type.Name
		iKey := rel.Interface.PkgPath + "." + rel.Interface.Name
		if keep[tKey] && keep[iKey] {
			out.Relations = append(out.Relations, rel)
		}
	}

	return out
}

func itoa(n int) string {
	return strconv.Itoa(n)
}

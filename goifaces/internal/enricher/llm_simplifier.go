package enricher

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/olehluchkiv/goifaces/internal/analyzer"
	"github.com/olehluchkiv/goifaces/internal/enricher/llm"
)

const simplifierSystemPrompt = `You are an expert Go software architect. Given a list of interfaces and types in a Go project, select the most architecturally significant nodes to keep in a simplified diagram.

Prefer keeping:
- Hub interfaces with many implementations
- Types that bridge different domains/packages
- Core domain types over utility types

Respond with JSON only.`

const simplifierUserPrompt = `Select the %d most important nodes to keep from this Go project:

%s

Return JSON with this schema:
{"keep": ["pkg.Name1", "pkg.Name2"]}

Rules:
- Return exactly %d node keys (or fewer if there aren't enough)
- Keys must match exactly from the input list
- Prefer interfaces and hub types over leaf implementations`

// LLMSimplifier uses an LLM to intelligently select which nodes to keep.
type LLMSimplifier struct {
	ctx      context.Context
	client   *llm.Client
	fallback *DefaultSimplifier
	logger   *slog.Logger
	MaxNodes int
}

// NewLLMSimplifier creates an LLM-backed intelligent simplifier.
func NewLLMSimplifier(ctx context.Context, client *llm.Client, fallback *DefaultSimplifier, logger *slog.Logger) *LLMSimplifier {
	return &LLMSimplifier{
		ctx:      ctx,
		client:   client,
		fallback: fallback,
		logger:   logger.With("component", "llm-simplifier"),
		MaxNodes: fallback.MaxNodes,
	}
}

func (s *LLMSimplifier) Enrich(result *analyzer.Result) *analyzer.Result {
	if s.MaxNodes <= 0 {
		return result
	}
	return s.Simplify(result, s.MaxNodes)
}

func (s *LLMSimplifier) Simplify(result *analyzer.Result, maxNodes int) *analyzer.Result {
	totalNodes := len(result.Interfaces) + len(result.Types)
	if totalNodes <= maxNodes {
		return result
	}

	nodeList := llm.SerializeNodeList(result)
	userPrompt := fmt.Sprintf(simplifierUserPrompt, maxNodes, nodeList, maxNodes)

	raw, err := s.client.Complete(s.ctx, simplifierSystemPrompt, userPrompt)
	if err != nil {
		s.logger.Warn("LLM simplifier failed, using default", "error", err)
		return s.fallback.Simplify(result, maxNodes)
	}

	var resp struct {
		Keep []string `json:"keep"`
	}
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		s.logger.Warn("LLM simplifier returned invalid JSON, using default", "error", err)
		return s.fallback.Simplify(result, maxNodes)
	}

	// Build keep set, validating against actual nodes
	allNodes := make(map[string]bool)
	for _, iface := range result.Interfaces {
		allNodes[iface.PkgPath+"."+iface.Name] = true
	}
	for _, typ := range result.Types {
		allNodes[typ.PkgPath+"."+typ.Name] = true
	}

	keep := make(map[string]bool)
	for _, key := range resp.Keep {
		if allNodes[key] {
			keep[key] = true
		}
	}

	if len(keep) == 0 {
		s.logger.Warn("LLM simplifier returned no valid nodes, using default")
		return s.fallback.Simplify(result, maxNodes)
	}

	// Filter result to only kept nodes
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

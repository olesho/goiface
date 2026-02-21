package enricher

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/olehluchkiv/goifaces/internal/analyzer"
	"github.com/olehluchkiv/goifaces/internal/enricher/llm"
)

const grouperSystemPrompt = `You are an expert Go software architect. Given a list of Go interfaces, types, their methods, and implementation relationships, identify the architectural layers or domains they belong to.

Group them into semantic categories like "Data Access Layer", "Business Logic", "HTTP Transport", "Domain Models", "Configuration", etc. based on their names, methods, and relationships.

Respond with JSON only.`

const grouperUserPrompt = `Analyze these Go interfaces and types and group them into architectural layers:

%s

Return JSON with this schema:
{"groups": [{"name": "Layer Name", "interfaces": ["pkg.Name"], "types": ["pkg.Name"]}]}

Rules:
- Every interface and type must appear in exactly one group
- Use descriptive architectural layer names
- Group by semantic purpose, not by package`

// LLMGrouper uses an LLM to identify architectural layers.
type LLMGrouper struct {
	ctx      context.Context
	client   *llm.Client
	fallback *DefaultGrouper
	logger   *slog.Logger
}

// NewLLMGrouper creates an LLM-backed semantic grouper.
func NewLLMGrouper(ctx context.Context, client *llm.Client, fallback *DefaultGrouper, logger *slog.Logger) *LLMGrouper {
	return &LLMGrouper{
		ctx:      ctx,
		client:   client,
		fallback: fallback,
		logger:   logger.With("component", "llm-grouper"),
	}
}

func (g *LLMGrouper) Enrich(result *analyzer.Result) *analyzer.Result {
	return result
}

func (g *LLMGrouper) Group(result *analyzer.Result) []SemanticGroup {
	if len(result.Interfaces) == 0 && len(result.Types) == 0 {
		return g.fallback.Group(result)
	}

	// Pre-filter for large projects
	filtered := llm.PreFilterByEdgeCount(result, 100)
	prompt := llm.SerializeResult(filtered)

	raw, err := g.client.Complete(g.ctx, grouperSystemPrompt, fmt.Sprintf(grouperUserPrompt, prompt))
	if err != nil {
		g.logger.Warn("LLM grouper failed, using default", "error", err)
		return g.fallback.Group(result)
	}

	var resp struct {
		Groups []struct {
			Name       string   `json:"name"`
			Interfaces []string `json:"interfaces"`
			Types      []string `json:"types"`
		} `json:"groups"`
	}
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		g.logger.Warn("LLM grouper returned invalid JSON, using default", "error", err)
		return g.fallback.Group(result)
	}

	// Validate: collect known keys
	knownIfaces := make(map[string]bool)
	for _, iface := range result.Interfaces {
		knownIfaces[iface.PkgPath+"."+iface.Name] = true
	}
	knownTypes := make(map[string]bool)
	for _, typ := range result.Types {
		knownTypes[typ.PkgPath+"."+typ.Name] = true
	}

	groups := make([]SemanticGroup, 0, len(resp.Groups))
	for _, rg := range resp.Groups {
		sg := SemanticGroup{Name: rg.Name}
		for _, key := range rg.Interfaces {
			if knownIfaces[key] {
				sg.Interfaces = append(sg.Interfaces, key)
			}
		}
		for _, key := range rg.Types {
			if knownTypes[key] {
				sg.Types = append(sg.Types, key)
			}
		}
		if len(sg.Interfaces) > 0 || len(sg.Types) > 0 {
			groups = append(groups, sg)
		}
	}

	if len(groups) == 0 {
		g.logger.Warn("LLM grouper returned no valid groups, using default")
		return g.fallback.Group(result)
	}

	return groups
}

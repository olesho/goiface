package enricher

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/olehluchkiv/goifaces/internal/analyzer"
	"github.com/olehluchkiv/goifaces/internal/enricher/llm"
)

// LLMGrouper uses an LLM to identify semantic/architectural groups.
type LLMGrouper struct {
	client   *llm.Client
	fallback *DefaultGrouper
	logger   *slog.Logger
}

// NewLLMGrouper creates a new LLM-backed grouper.
func NewLLMGrouper(client *llm.Client, fallback *DefaultGrouper, logger *slog.Logger) *LLMGrouper {
	return &LLMGrouper{
		client:   client,
		fallback: fallback,
		logger:   logger.With("component", "enricher.llm-grouper"),
	}
}

func (g *LLMGrouper) Enrich(result *analyzer.Result) *analyzer.Result {
	return result
}

const grouperSystemPrompt = `You are a Go architecture analyst. Given a list of Go interfaces, types, and their relationships, identify logical architectural layers or domains they belong to (e.g., "Data Access Layer", "Business Logic", "HTTP Transport", "Domain Models").

Respond with JSON only:
{
  "groups": [
    {
      "name": "Layer/Domain Name",
      "interfaces": ["pkg.InterfaceName"],
      "types": ["pkg.TypeName"]
    }
  ]
}

Rules:
- Every interface and type must appear in exactly one group
- Group names should be descriptive architectural labels
- If unsure, group by functional similarity rather than package name`

func (g *LLMGrouper) Group(result *analyzer.Result) []SemanticGroup {
	if len(result.Interfaces) == 0 && len(result.Types) == 0 {
		return g.fallback.Group(result)
	}

	prompt := llm.SerializeResult(result)
	resp, err := g.client.Complete(context.Background(), grouperSystemPrompt, prompt)
	if err != nil {
		g.logger.Warn("LLM grouper failed, using default", "error", err)
		return g.fallback.Group(result)
	}

	var parsed struct {
		Groups []struct {
			Name       string   `json:"name"`
			Interfaces []string `json:"interfaces"`
			Types      []string `json:"types"`
		} `json:"groups"`
	}
	if err := json.Unmarshal([]byte(resp), &parsed); err != nil {
		g.logger.Warn("LLM grouper returned invalid JSON", "error", err)
		return g.fallback.Group(result)
	}

	if len(parsed.Groups) == 0 {
		g.logger.Warn("LLM grouper returned empty groups")
		return g.fallback.Group(result)
	}

	// Build valid node key sets for validation
	validNodes := make(map[string]bool)
	for _, iface := range result.Interfaces {
		validNodes[iface.PkgPath+"."+iface.Name] = true
	}
	for _, typ := range result.Types {
		validNodes[typ.PkgPath+"."+typ.Name] = true
	}

	var groups []SemanticGroup
	for _, grp := range parsed.Groups {
		sg := SemanticGroup{Name: grp.Name}
		for _, key := range grp.Interfaces {
			if validNodes[key] {
				sg.Interfaces = append(sg.Interfaces, key)
			}
		}
		for _, key := range grp.Types {
			if validNodes[key] {
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

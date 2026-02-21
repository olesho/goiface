package enricher

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/olehluchkiv/goifaces/internal/analyzer"
	"github.com/olehluchkiv/goifaces/internal/enricher/llm"
)

// LLMPatternDetector uses an LLM to identify design patterns.
type LLMPatternDetector struct {
	client   *llm.Client
	fallback *DefaultPatternDetector
	logger   *slog.Logger
}

// NewLLMPatternDetector creates a new LLM-backed pattern detector.
func NewLLMPatternDetector(client *llm.Client, fallback *DefaultPatternDetector, logger *slog.Logger) *LLMPatternDetector {
	return &LLMPatternDetector{
		client:   client,
		fallback: fallback,
		logger:   logger.With("component", "enricher.llm-patterns"),
	}
}

func (d *LLMPatternDetector) Enrich(result *analyzer.Result) *analyzer.Result {
	return result
}

const patternSystemPrompt = `You are a Go design pattern analyst. Given Go interfaces, types, and their implementation relationships, identify GoF and Go-specific design patterns present in the code (Strategy, Factory, Repository, Observer, Decorator, Adapter, Builder, etc.).

Respond with JSON only:
{
  "patterns": [
    {
      "name": "Pattern Name",
      "description": "Brief description of how this pattern manifests",
      "participants": ["pkg.InterfaceName", "pkg.TypeName"]
    }
  ]
}

Rules:
- Only report patterns you are confident about
- Use the exact node keys from the input
- Prefer Go-idiomatic pattern names (e.g., "Interface Segregation" over just "ISP")`

func (d *LLMPatternDetector) Detect(result *analyzer.Result) []DetectedPattern {
	if len(result.Relations) == 0 {
		return d.fallback.Detect(result)
	}

	prompt := llm.SerializeResult(result)
	resp, err := d.client.Complete(context.Background(), patternSystemPrompt, prompt)
	if err != nil {
		d.logger.Warn("LLM pattern detector failed, using default", "error", err)
		return d.fallback.Detect(result)
	}

	var parsed struct {
		Patterns []struct {
			Name         string   `json:"name"`
			Description  string   `json:"description"`
			Participants []string `json:"participants"`
		} `json:"patterns"`
	}
	if err := json.Unmarshal([]byte(resp), &parsed); err != nil {
		d.logger.Warn("LLM pattern detector returned invalid JSON", "error", err)
		return d.fallback.Detect(result)
	}

	// Build valid node key set
	validNodes := make(map[string]bool)
	for _, iface := range result.Interfaces {
		validNodes[iface.PkgPath+"."+iface.Name] = true
	}
	for _, typ := range result.Types {
		validNodes[typ.PkgPath+"."+typ.Name] = true
	}

	var patterns []DetectedPattern
	for _, p := range parsed.Patterns {
		dp := DetectedPattern{
			Name:        p.Name,
			Description: p.Description,
		}
		for _, key := range p.Participants {
			if validNodes[key] {
				dp.Participants = append(dp.Participants, key)
			}
		}
		if len(dp.Participants) > 0 {
			patterns = append(patterns, dp)
		}
	}

	return patterns
}

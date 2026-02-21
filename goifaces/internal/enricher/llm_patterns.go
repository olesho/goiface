package enricher

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/olehluchkiv/goifaces/internal/analyzer"
	"github.com/olehluchkiv/goifaces/internal/enricher/llm"
)

const patternsSystemPrompt = `You are an expert Go software architect. Given a list of Go interfaces, types, their methods, and implementation relationships, identify design patterns present in the code.

Look for GoF patterns (Strategy, Factory, Repository, Observer, Decorator, Adapter, etc.) and Go-specific patterns (functional options, middleware chains, etc.).

Respond with JSON only.`

const patternsUserPrompt = `Analyze these Go interfaces and types for design patterns:

%s

Return JSON with this schema:
{"patterns": [{"name": "Pattern Name", "description": "Brief explanation", "participants": ["pkg.Name"]}]}

Rules:
- Only report patterns you are confident about
- Participants must be actual keys from the input
- Description should be one sentence`

// LLMPatternDetector uses an LLM to recognize design patterns.
type LLMPatternDetector struct {
	ctx      context.Context
	client   *llm.Client
	fallback *DefaultPatternDetector
	logger   *slog.Logger
}

// NewLLMPatternDetector creates an LLM-backed pattern detector.
func NewLLMPatternDetector(ctx context.Context, client *llm.Client, fallback *DefaultPatternDetector, logger *slog.Logger) *LLMPatternDetector {
	return &LLMPatternDetector{
		ctx:      ctx,
		client:   client,
		fallback: fallback,
		logger:   logger.With("component", "llm-patterns"),
	}
}

func (d *LLMPatternDetector) Enrich(result *analyzer.Result) *analyzer.Result {
	return result
}

func (d *LLMPatternDetector) Detect(result *analyzer.Result) []DetectedPattern {
	if len(result.Interfaces) == 0 && len(result.Types) == 0 {
		return d.fallback.Detect(result)
	}

	filtered := llm.PreFilterByEdgeCount(result, 100)
	prompt := llm.SerializeResult(filtered)

	raw, err := d.client.Complete(d.ctx, patternsSystemPrompt, fmt.Sprintf(patternsUserPrompt, prompt))
	if err != nil {
		d.logger.Warn("LLM pattern detector failed, using default", "error", err)
		return d.fallback.Detect(result)
	}

	var resp struct {
		Patterns []struct {
			Name         string   `json:"name"`
			Description  string   `json:"description"`
			Participants []string `json:"participants"`
		} `json:"patterns"`
	}
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		d.logger.Warn("LLM pattern detector returned invalid JSON, using default", "error", err)
		return d.fallback.Detect(result)
	}

	// Collect known keys for validation
	known := make(map[string]bool)
	for _, iface := range result.Interfaces {
		known[iface.PkgPath+"."+iface.Name] = true
	}
	for _, typ := range result.Types {
		known[typ.PkgPath+"."+typ.Name] = true
	}

	patterns := make([]DetectedPattern, 0, len(resp.Patterns))
	for _, rp := range resp.Patterns {
		dp := DetectedPattern{
			Name:        rp.Name,
			Description: rp.Description,
		}
		for _, key := range rp.Participants {
			if known[key] {
				dp.Participants = append(dp.Participants, key)
			}
		}
		if len(dp.Participants) > 0 {
			patterns = append(patterns, dp)
		}
	}

	return patterns
}

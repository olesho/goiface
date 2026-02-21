package enricher

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/olehluchkiv/goifaces/internal/analyzer"
	"github.com/olehluchkiv/goifaces/internal/enricher/llm"
)

const scorerSystemPrompt = `You are an expert Go software architect. Given a list of implementation relationships (concrete type implements interface), score each by architectural importance on a scale of 0.0 to 1.0.

High scores (0.7-1.0): Core domain relationships, key abstractions
Medium scores (0.4-0.6): Important but not central relationships
Low scores (0.0-0.3): Incidental implementations (e.g., implementing error, fmt.Stringer, or other utility interfaces)

Respond with JSON only.`

const scorerUserPrompt = `Score these Go implementation relationships by architectural importance:

%s

Return JSON with this schema:
{"scores": {"0": 0.9, "1": 0.3}}

Rules:
- Keys are string-encoded relation indices matching the input order
- Values are floats between 0.0 and 1.0
- Score every relationship from the input`

// LLMScorer uses an LLM to rank relationships by importance.
type LLMScorer struct {
	ctx      context.Context
	client   *llm.Client
	fallback *DefaultScorer
	logger   *slog.Logger
}

// NewLLMScorer creates an LLM-backed relationship scorer.
func NewLLMScorer(ctx context.Context, client *llm.Client, fallback *DefaultScorer, logger *slog.Logger) *LLMScorer {
	return &LLMScorer{
		ctx:      ctx,
		client:   client,
		fallback: fallback,
		logger:   logger.With("component", "llm-scorer"),
	}
}

func (s *LLMScorer) Enrich(result *analyzer.Result) *analyzer.Result {
	return result
}

func (s *LLMScorer) Score(relations []analyzer.Relation) map[int]float64 {
	if len(relations) == 0 {
		return s.fallback.Score(relations)
	}

	// Build a temporary result for serialization
	tempResult := &analyzer.Result{Relations: relations}
	prompt := llm.SerializeRelations(tempResult)

	raw, err := s.client.Complete(s.ctx, scorerSystemPrompt, fmt.Sprintf(scorerUserPrompt, prompt))
	if err != nil {
		s.logger.Warn("LLM scorer failed, using default", "error", err)
		return s.fallback.Score(relations)
	}

	var resp struct {
		Scores map[string]float64 `json:"scores"`
	}
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		s.logger.Warn("LLM scorer returned invalid JSON, using default", "error", err)
		return s.fallback.Score(relations)
	}

	if len(resp.Scores) == 0 {
		s.logger.Warn("LLM scorer returned no scores, using default")
		return s.fallback.Score(relations)
	}

	scores := make(map[int]float64, len(relations))
	for key, val := range resp.Scores {
		idx, err := strconv.Atoi(key)
		if err != nil || idx < 0 || idx >= len(relations) {
			continue
		}
		// Clamp to [0.0, 1.0]
		if val < 0 {
			val = 0
		}
		if val > 1 {
			val = 1
		}
		scores[idx] = val
	}

	// Fill in any missing scores with default (1.0)
	for i := range relations {
		if _, ok := scores[i]; !ok {
			scores[i] = 1.0
		}
	}

	return scores
}

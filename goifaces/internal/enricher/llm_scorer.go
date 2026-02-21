package enricher

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"

	"github.com/olehluchkiv/goifaces/internal/analyzer"
	"github.com/olehluchkiv/goifaces/internal/enricher/llm"
)

// LLMScorer uses an LLM to score relationships by architectural importance.
type LLMScorer struct {
	client   *llm.Client
	fallback *DefaultScorer
	logger   *slog.Logger
}

// NewLLMScorer creates a new LLM-backed relationship scorer.
func NewLLMScorer(client *llm.Client, fallback *DefaultScorer, logger *slog.Logger) *LLMScorer {
	return &LLMScorer{
		client:   client,
		fallback: fallback,
		logger:   logger.With("component", "enricher.llm-scorer"),
	}
}

func (s *LLMScorer) Enrich(result *analyzer.Result) *analyzer.Result {
	return result
}

const scorerSystemPrompt = `You are a Go architecture analyst. Given a list of type-implements-interface relationships, score each by architectural importance from 0.0 to 1.0.

Respond with JSON only:
{
  "scores": {
    "0": 0.9,
    "1": 0.3
  }
}

Rules:
- Keys are string-encoded indices matching the input order
- Core domain relationships score high (0.7-1.0)
- Incidental implementations (error, fmt.Stringer) score low (0.1-0.3)
- Every relationship should get a score
- Default to 0.5 if unsure`

func (s *LLMScorer) Score(relations []analyzer.Relation) map[int]float64 {
	if len(relations) == 0 {
		return s.fallback.Score(relations)
	}

	prompt := llm.SerializeRelations(relations)
	resp, err := s.client.Complete(context.Background(), scorerSystemPrompt, prompt)
	if err != nil {
		s.logger.Warn("LLM scorer failed, using default", "error", err)
		return s.fallback.Score(relations)
	}

	var parsed struct {
		Scores map[string]float64 `json:"scores"`
	}
	if err := json.Unmarshal([]byte(resp), &parsed); err != nil {
		s.logger.Warn("LLM scorer returned invalid JSON", "error", err)
		return s.fallback.Score(relations)
	}

	if len(parsed.Scores) == 0 {
		s.logger.Warn("LLM scorer returned empty scores")
		return s.fallback.Score(relations)
	}

	scores := make(map[int]float64, len(relations))
	for key, score := range parsed.Scores {
		idx, err := strconv.Atoi(key)
		if err != nil || idx < 0 || idx >= len(relations) {
			continue
		}
		// Clamp score to [0.0, 1.0]
		if score < 0 {
			score = 0
		}
		if score > 1 {
			score = 1
		}
		scores[idx] = score
	}

	if len(scores) == 0 {
		s.logger.Warn("LLM scorer returned no valid score indices, using default")
		return s.fallback.Score(relations)
	}

	// Fill in missing indices with default score
	for i := range relations {
		if _, ok := scores[i]; !ok {
			scores[i] = 1.0
		}
	}

	return scores
}

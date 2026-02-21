package enricher_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/olehluchkiv/goifaces/internal/analyzer"
	"github.com/olehluchkiv/goifaces/internal/enricher"
	"github.com/olehluchkiv/goifaces/internal/enricher/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
}

func chatResponse(content string) []byte {
	resp := map[string]any{
		"choices": []map[string]any{
			{"message": map[string]string{"content": content}},
		},
	}
	data, _ := json.Marshal(resp)
	return data
}

func mockLLMServer(response string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(chatResponse(response))
	}))
}

func failingLLMServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("server error"))
	}))
}

func newTestClient(serverURL string) *llm.Client {
	return llm.NewClient(llm.Config{
		Endpoint: serverURL,
		APIKey:   "test",
		Model:    "test",
	}, testLogger())
}

func sampleResult() *analyzer.Result {
	iface := analyzer.InterfaceDef{
		Name:    "Repository",
		PkgPath: "example.com/app/store",
		PkgName: "store",
		Methods: []analyzer.MethodSig{{Name: "Get", Signature: "(id string) (Item, error)"}},
	}
	typ := analyzer.TypeDef{
		Name:     "PostgresRepo",
		PkgPath:  "example.com/app/store",
		PkgName:  "store",
		IsStruct: true,
		Methods:  []analyzer.MethodSig{{Name: "Get", Signature: "(id string) (Item, error)"}},
	}
	return &analyzer.Result{
		Interfaces: []analyzer.InterfaceDef{iface},
		Types:      []analyzer.TypeDef{typ},
		Relations: []analyzer.Relation{
			{Type: &typ, Interface: &iface},
		},
	}
}

func bgCtx() context.Context {
	return context.Background()
}

// --- LLM Grouper Tests ---

func TestLLMGrouper_Success(t *testing.T) {
	server := mockLLMServer(`{"groups": [{"name": "Data Access Layer", "interfaces": ["example.com/app/store.Repository"], "types": ["example.com/app/store.PostgresRepo"]}]}`)
	defer server.Close()

	client := newTestClient(server.URL)
	g := enricher.NewLLMGrouper(bgCtx(), client, enricher.NewDefaultGrouper(), testLogger())

	result := sampleResult()
	groups := g.Group(result)

	require.Len(t, groups, 1)
	assert.Equal(t, "Data Access Layer", groups[0].Name)
	assert.Contains(t, groups[0].Interfaces, "example.com/app/store.Repository")
	assert.Contains(t, groups[0].Types, "example.com/app/store.PostgresRepo")
}

func TestLLMGrouper_Fallback(t *testing.T) {
	server := failingLLMServer()
	defer server.Close()

	client := newTestClient(server.URL)
	g := enricher.NewLLMGrouper(bgCtx(), client, enricher.NewDefaultGrouper(), testLogger())

	result := sampleResult()
	groups := g.Group(result)

	// Should fall back to default grouper (by package)
	require.NotEmpty(t, groups)
	assert.Equal(t, "store", groups[0].Name)
}

func TestLLMGrouper_InvalidJSON(t *testing.T) {
	server := mockLLMServer(`not json`)
	defer server.Close()

	client := newTestClient(server.URL)
	g := enricher.NewLLMGrouper(bgCtx(), client, enricher.NewDefaultGrouper(), testLogger())

	result := sampleResult()
	groups := g.Group(result)

	require.NotEmpty(t, groups)
	assert.Equal(t, "store", groups[0].Name)
}

func TestLLMGrouper_UnknownKeys(t *testing.T) {
	server := mockLLMServer(`{"groups": [{"name": "Layer", "interfaces": ["nonexistent.Foo"], "types": ["nonexistent.Bar"]}]}`)
	defer server.Close()

	client := newTestClient(server.URL)
	g := enricher.NewLLMGrouper(bgCtx(), client, enricher.NewDefaultGrouper(), testLogger())

	result := sampleResult()
	groups := g.Group(result)

	// Unknown keys filtered out -> falls back to default
	require.NotEmpty(t, groups)
	assert.Equal(t, "store", groups[0].Name)
}

func TestLLMGrouper_EmptyResult(t *testing.T) {
	client := newTestClient("http://unused")
	g := enricher.NewLLMGrouper(bgCtx(), client, enricher.NewDefaultGrouper(), testLogger())

	result := &analyzer.Result{}
	groups := g.Group(result)

	assert.Empty(t, groups)
}

func TestLLMGrouper_Enrich(t *testing.T) {
	client := newTestClient("http://unused")
	g := enricher.NewLLMGrouper(bgCtx(), client, enricher.NewDefaultGrouper(), testLogger())

	result := sampleResult()
	enriched := g.Enrich(result)

	assert.Equal(t, result, enriched, "Enrich should return result unchanged")
}

// --- LLM Pattern Detector Tests ---

func TestLLMPatternDetector_Success(t *testing.T) {
	server := mockLLMServer(`{"patterns": [{"name": "Repository", "description": "Data access abstraction", "participants": ["example.com/app/store.Repository", "example.com/app/store.PostgresRepo"]}]}`)
	defer server.Close()

	client := newTestClient(server.URL)
	d := enricher.NewLLMPatternDetector(bgCtx(), client, enricher.NewDefaultPatternDetector(), testLogger())

	result := sampleResult()
	patterns := d.Detect(result)

	require.Len(t, patterns, 1)
	assert.Equal(t, "Repository", patterns[0].Name)
	assert.Len(t, patterns[0].Participants, 2)
}

func TestLLMPatternDetector_Fallback(t *testing.T) {
	server := failingLLMServer()
	defer server.Close()

	client := newTestClient(server.URL)
	d := enricher.NewLLMPatternDetector(bgCtx(), client, enricher.NewDefaultPatternDetector(), testLogger())

	result := sampleResult()
	patterns := d.Detect(result)

	assert.Empty(t, patterns)
}

func TestLLMPatternDetector_Enrich(t *testing.T) {
	client := newTestClient("http://unused")
	d := enricher.NewLLMPatternDetector(bgCtx(), client, enricher.NewDefaultPatternDetector(), testLogger())

	result := sampleResult()
	enriched := d.Enrich(result)

	assert.Equal(t, result, enriched)
}

// --- LLM Simplifier Tests ---

func TestLLMSimplifier_Success(t *testing.T) {
	server := mockLLMServer(`{"keep": ["example.com/app/store.Repository"]}`)
	defer server.Close()

	client := newTestClient(server.URL)
	s := enricher.NewLLMSimplifier(bgCtx(), client, enricher.NewDefaultSimplifier(), testLogger())

	result := sampleResult()
	simplified := s.Simplify(result, 1)

	assert.Len(t, simplified.Interfaces, 1)
	assert.Empty(t, simplified.Types, "type not in keep list should be removed")
}

func TestLLMSimplifier_NoCapNeeded(t *testing.T) {
	client := newTestClient("http://unused")
	s := enricher.NewLLMSimplifier(bgCtx(), client, enricher.NewDefaultSimplifier(), testLogger())

	result := sampleResult()
	simplified := s.Simplify(result, 100)

	assert.Equal(t, result, simplified, "should return as-is when under cap")
}

func TestLLMSimplifier_Fallback(t *testing.T) {
	server := failingLLMServer()
	defer server.Close()

	client := newTestClient(server.URL)
	s := enricher.NewLLMSimplifier(bgCtx(), client, enricher.NewDefaultSimplifier(), testLogger())

	result := sampleResult()
	simplified := s.Simplify(result, 1)

	// Should fall back to default simplifier
	assert.NotNil(t, simplified)
}

func TestLLMSimplifier_Enrich_NoMaxNodes(t *testing.T) {
	client := newTestClient("http://unused")
	s := enricher.NewLLMSimplifier(bgCtx(), client, enricher.NewDefaultSimplifier(), testLogger())

	result := sampleResult()
	enriched := s.Enrich(result)

	assert.Equal(t, result, enriched, "Enrich with 0 MaxNodes should pass through")
}

// --- LLM Annotator Tests ---

func TestLLMAnnotator_Success(t *testing.T) {
	server := mockLLMServer(`{"annotations": {"example.com/app/store.Repository": "Data access interface for items", "example.com/app/store.PostgresRepo": "PostgreSQL implementation"}}`)
	defer server.Close()

	client := newTestClient(server.URL)
	a := enricher.NewLLMAnnotator(bgCtx(), client, enricher.NewDefaultAnnotator(), testLogger())

	result := sampleResult()
	annotations := a.Annotate(result)

	require.Len(t, annotations, 2)
	assert.Equal(t, "Data access interface for items", annotations["example.com/app/store.Repository"])
	assert.Equal(t, "PostgreSQL implementation", annotations["example.com/app/store.PostgresRepo"])
}

func TestLLMAnnotator_Fallback(t *testing.T) {
	server := failingLLMServer()
	defer server.Close()

	client := newTestClient(server.URL)
	a := enricher.NewLLMAnnotator(bgCtx(), client, enricher.NewDefaultAnnotator(), testLogger())

	result := sampleResult()
	annotations := a.Annotate(result)

	assert.Nil(t, annotations)
}

func TestLLMAnnotator_UnknownKeys(t *testing.T) {
	server := mockLLMServer(`{"annotations": {"nonexistent.Foo": "description"}}`)
	defer server.Close()

	client := newTestClient(server.URL)
	a := enricher.NewLLMAnnotator(bgCtx(), client, enricher.NewDefaultAnnotator(), testLogger())

	result := sampleResult()
	annotations := a.Annotate(result)

	// Unknown keys filtered -> falls back to default
	assert.Nil(t, annotations)
}

func TestLLMAnnotator_Enrich(t *testing.T) {
	client := newTestClient("http://unused")
	a := enricher.NewLLMAnnotator(bgCtx(), client, enricher.NewDefaultAnnotator(), testLogger())

	result := sampleResult()
	enriched := a.Enrich(result)

	assert.Equal(t, result, enriched)
}

// --- LLM Scorer Tests ---

func TestLLMScorer_Success(t *testing.T) {
	server := mockLLMServer(`{"scores": {"0": 0.85}}`)
	defer server.Close()

	client := newTestClient(server.URL)
	s := enricher.NewLLMScorer(bgCtx(), client, enricher.NewDefaultScorer(), testLogger())

	result := sampleResult()
	scores := s.Score(result.Relations)

	require.Len(t, scores, 1)
	assert.InDelta(t, 0.85, scores[0], 0.001)
}

func TestLLMScorer_Fallback(t *testing.T) {
	server := failingLLMServer()
	defer server.Close()

	client := newTestClient(server.URL)
	s := enricher.NewLLMScorer(bgCtx(), client, enricher.NewDefaultScorer(), testLogger())

	result := sampleResult()
	scores := s.Score(result.Relations)

	require.Len(t, scores, 1)
	assert.Equal(t, 1.0, scores[0])
}

func TestLLMScorer_ClampValues(t *testing.T) {
	server := mockLLMServer(`{"scores": {"0": 1.5}}`)
	defer server.Close()

	client := newTestClient(server.URL)
	s := enricher.NewLLMScorer(bgCtx(), client, enricher.NewDefaultScorer(), testLogger())

	result := sampleResult()
	scores := s.Score(result.Relations)

	assert.InDelta(t, 1.0, scores[0], 0.001, "should clamp to 1.0")
}

func TestLLMScorer_MissingIndices(t *testing.T) {
	server := mockLLMServer(`{"scores": {}}`)
	defer server.Close()

	client := newTestClient(server.URL)
	s := enricher.NewLLMScorer(bgCtx(), client, enricher.NewDefaultScorer(), testLogger())

	result := sampleResult()
	scores := s.Score(result.Relations)

	// LLM returned empty -> falls back to default
	require.Len(t, scores, 1)
	assert.Equal(t, 1.0, scores[0])
}

func TestLLMScorer_Enrich(t *testing.T) {
	client := newTestClient("http://unused")
	s := enricher.NewLLMScorer(bgCtx(), client, enricher.NewDefaultScorer(), testLogger())

	result := sampleResult()
	enriched := s.Enrich(result)

	assert.Equal(t, result, enriched)
}

// --- Integration-style test: full pipeline with mock LLM ---

func TestLLMEnricherPipeline_WithMockServer(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		messages := req["messages"].([]any)
		userMsg := messages[1].(map[string]any)["content"].(string)

		var response string
		if strings.Contains(userMsg, "architectural layers") {
			response = `{"groups": [{"name": "Data Access", "interfaces": ["example.com/app/store.Repository"], "types": ["example.com/app/store.PostgresRepo"]}]}`
		} else {
			response = `{}`
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(chatResponse(response))
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	logger := testLogger()

	// Build enricher pipeline
	enrichers := []enricher.Enricher{
		enricher.NewLLMGrouper(ctx, client, enricher.NewDefaultGrouper(), logger),
		enricher.NewLLMSimplifier(ctx, client, enricher.NewDefaultSimplifier(), logger),
	}

	result := sampleResult()
	for _, e := range enrichers {
		result = e.Enrich(result)
	}

	// Result should be unchanged (enrichers don't modify via Enrich)
	assert.Len(t, result.Interfaces, 1)
	assert.Len(t, result.Types, 1)
	assert.Len(t, result.Relations, 1)

	// Test that Grouper actually works via Group()
	grouper := enrichers[0].(*enricher.LLMGrouper)
	groups := grouper.Group(sampleResult())
	require.NotEmpty(t, groups)
	assert.Equal(t, "Data Access", groups[0].Name)
}

// --- Fallback pipeline test: all enrichers fail gracefully ---

func TestLLMEnricherPipeline_AllFallback(t *testing.T) {
	ctx := context.Background()
	server := failingLLMServer()
	defer server.Close()

	client := newTestClient(server.URL)
	logger := testLogger()

	result := sampleResult()

	// All enrichers should fall back gracefully
	grouper := enricher.NewLLMGrouper(ctx, client, enricher.NewDefaultGrouper(), logger)
	groups := grouper.Group(result)
	require.NotEmpty(t, groups)

	detector := enricher.NewLLMPatternDetector(ctx, client, enricher.NewDefaultPatternDetector(), logger)
	patterns := detector.Detect(result)
	assert.Empty(t, patterns)

	annotator := enricher.NewLLMAnnotator(ctx, client, enricher.NewDefaultAnnotator(), logger)
	annotations := annotator.Annotate(result)
	assert.Nil(t, annotations)

	scorer := enricher.NewLLMScorer(ctx, client, enricher.NewDefaultScorer(), logger)
	scores := scorer.Score(result.Relations)
	assert.Equal(t, 1.0, scores[0])

	simplifier := enricher.NewLLMSimplifier(ctx, client, enricher.NewDefaultSimplifier(), logger)
	simplified := simplifier.Simplify(result, 1)
	assert.NotNil(t, simplified)
}

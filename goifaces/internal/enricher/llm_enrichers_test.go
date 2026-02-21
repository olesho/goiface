package enricher

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/olehluchkiv/goifaces/internal/analyzer"
	"github.com/olehluchkiv/goifaces/internal/enricher/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
}

func mockLLMServer(t *testing.T, response string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": response}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Logf("failed to encode response: %v", err)
		}
	}))
}

func mockLLMServerError(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write([]byte("internal error")); err != nil {
			t.Logf("failed to write error response: %v", err)
		}
	}))
}

func testResult() *analyzer.Result {
	ifaceReader := analyzer.InterfaceDef{
		Name: "Reader", PkgPath: "io", PkgName: "io",
		Methods: []analyzer.MethodSig{{Name: "Read", Signature: "(p []byte) (int, error)"}},
	}
	ifaceWriter := analyzer.InterfaceDef{
		Name: "Writer", PkgPath: "io", PkgName: "io",
		Methods: []analyzer.MethodSig{{Name: "Write", Signature: "(p []byte) (int, error)"}},
	}
	typeFile := analyzer.TypeDef{
		Name: "File", PkgPath: "os", PkgName: "os", IsStruct: true,
	}
	typeBuffer := analyzer.TypeDef{
		Name: "Buffer", PkgPath: "bytes", PkgName: "bytes", IsStruct: true,
	}
	return &analyzer.Result{
		Interfaces: []analyzer.InterfaceDef{ifaceReader, ifaceWriter},
		Types:      []analyzer.TypeDef{typeFile, typeBuffer},
		Relations: []analyzer.Relation{
			{Type: &typeFile, Interface: &ifaceReader},
			{Type: &typeFile, Interface: &ifaceWriter},
			{Type: &typeBuffer, Interface: &ifaceReader},
			{Type: &typeBuffer, Interface: &ifaceWriter},
		},
	}
}

func newTestClient(url string) *llm.Client {
	return llm.NewClient(llm.Config{
		Endpoint: url,
		APIKey:   "test",
		Model:    "test",
	}, testLogger())
}

// --- Grouper Tests ---

func TestLLMGrouper_Success(t *testing.T) {
	resp := `{
		"groups": [
			{"name": "I/O Layer", "interfaces": ["io.Reader", "io.Writer"], "types": []},
			{"name": "Storage", "interfaces": [], "types": ["os.File"]},
			{"name": "Buffers", "interfaces": [], "types": ["bytes.Buffer"]}
		]
	}`
	srv := mockLLMServer(t, resp)
	defer srv.Close()

	grouper := NewLLMGrouper(newTestClient(srv.URL), NewDefaultGrouper(), testLogger())
	groups := grouper.Group(testResult())

	require.Len(t, groups, 3)
	assert.Equal(t, "I/O Layer", groups[0].Name)
	assert.Equal(t, []string{"io.Reader", "io.Writer"}, groups[0].Interfaces)
}

func TestLLMGrouper_FallbackOnError(t *testing.T) {
	srv := mockLLMServerError(t)
	defer srv.Close()

	grouper := NewLLMGrouper(newTestClient(srv.URL), NewDefaultGrouper(), testLogger())
	groups := grouper.Group(testResult())

	// Should fall back to default grouper which groups by package
	require.NotEmpty(t, groups)
}

func TestLLMGrouper_FallbackOnBadJSON(t *testing.T) {
	srv := mockLLMServer(t, `not json at all`)
	defer srv.Close()

	grouper := NewLLMGrouper(newTestClient(srv.URL), NewDefaultGrouper(), testLogger())
	groups := grouper.Group(testResult())
	require.NotEmpty(t, groups)
}

func TestLLMGrouper_IgnoresInvalidKeys(t *testing.T) {
	resp := `{
		"groups": [
			{"name": "Valid", "interfaces": ["io.Reader"], "types": ["nonexistent.Type"]},
			{"name": "AllInvalid", "interfaces": ["fake.Iface"], "types": []}
		]
	}`
	srv := mockLLMServer(t, resp)
	defer srv.Close()

	grouper := NewLLMGrouper(newTestClient(srv.URL), NewDefaultGrouper(), testLogger())
	groups := grouper.Group(testResult())

	// Only "Valid" group should remain (AllInvalid has no valid nodes)
	require.Len(t, groups, 1)
	assert.Equal(t, "Valid", groups[0].Name)
	assert.Equal(t, []string{"io.Reader"}, groups[0].Interfaces)
}

func TestLLMGrouper_Enrich(t *testing.T) {
	srv := mockLLMServerError(t)
	defer srv.Close()

	grouper := NewLLMGrouper(newTestClient(srv.URL), NewDefaultGrouper(), testLogger())
	result := testResult()
	enriched := grouper.Enrich(result)
	assert.Equal(t, result, enriched, "Enrich should return result unchanged")
}

func TestLLMGrouper_EmptyResult(t *testing.T) {
	srv := mockLLMServerError(t)
	defer srv.Close()

	grouper := NewLLMGrouper(newTestClient(srv.URL), NewDefaultGrouper(), testLogger())
	groups := grouper.Group(&analyzer.Result{})
	assert.Empty(t, groups)
}

// --- Pattern Detector Tests ---

func TestLLMPatternDetector_Success(t *testing.T) {
	resp := `{
		"patterns": [
			{
				"name": "Strategy",
				"description": "Multiple implementations of I/O interfaces",
				"participants": ["io.Reader", "os.File", "bytes.Buffer"]
			}
		]
	}`
	srv := mockLLMServer(t, resp)
	defer srv.Close()

	detector := NewLLMPatternDetector(newTestClient(srv.URL), NewDefaultPatternDetector(), testLogger())
	patterns := detector.Detect(testResult())

	require.Len(t, patterns, 1)
	assert.Equal(t, "Strategy", patterns[0].Name)
	assert.Len(t, patterns[0].Participants, 3)
}

func TestLLMPatternDetector_FallbackOnError(t *testing.T) {
	srv := mockLLMServerError(t)
	defer srv.Close()

	detector := NewLLMPatternDetector(newTestClient(srv.URL), NewDefaultPatternDetector(), testLogger())
	patterns := detector.Detect(testResult())
	assert.Nil(t, patterns)
}

func TestLLMPatternDetector_Enrich(t *testing.T) {
	srv := mockLLMServerError(t)
	defer srv.Close()

	detector := NewLLMPatternDetector(newTestClient(srv.URL), NewDefaultPatternDetector(), testLogger())
	result := testResult()
	assert.Equal(t, result, detector.Enrich(result))
}

// --- Simplifier Tests ---

func TestLLMSimplifier_Success(t *testing.T) {
	resp := `{"keep": ["io.Reader", "os.File"]}`
	srv := mockLLMServer(t, resp)
	defer srv.Close()

	fallback := &DefaultSimplifier{MaxNodes: 2}
	simplifier := NewLLMSimplifier(newTestClient(srv.URL), fallback, testLogger())
	result := simplifier.Simplify(testResult(), 2)

	assert.Len(t, result.Interfaces, 1)
	assert.Equal(t, "Reader", result.Interfaces[0].Name)
	assert.Len(t, result.Types, 1)
	assert.Equal(t, "File", result.Types[0].Name)
}

func TestLLMSimplifier_FallbackOnError(t *testing.T) {
	srv := mockLLMServerError(t)
	defer srv.Close()

	fallback := &DefaultSimplifier{MaxNodes: 2}
	simplifier := NewLLMSimplifier(newTestClient(srv.URL), fallback, testLogger())
	result := simplifier.Simplify(testResult(), 2)

	// Should have at most 2 nodes from default simplifier
	total := len(result.Interfaces) + len(result.Types)
	assert.LessOrEqual(t, total, 2)
}

func TestLLMSimplifier_NoCapNeeded(t *testing.T) {
	srv := mockLLMServerError(t) // should not be called
	defer srv.Close()

	fallback := &DefaultSimplifier{MaxNodes: 100}
	simplifier := NewLLMSimplifier(newTestClient(srv.URL), fallback, testLogger())
	result := testResult()
	simplified := simplifier.Simplify(result, 100)
	assert.Equal(t, result, simplified, "should return unchanged when under cap")
}

func TestLLMSimplifier_Enrich(t *testing.T) {
	srv := mockLLMServerError(t)
	defer srv.Close()

	fallback := &DefaultSimplifier{MaxNodes: 0} // no cap
	simplifier := NewLLMSimplifier(newTestClient(srv.URL), fallback, testLogger())
	result := testResult()
	assert.Equal(t, result, simplifier.Enrich(result), "Enrich with MaxNodes=0 returns unchanged")
}

// --- Annotator Tests ---

func TestLLMAnnotator_Success(t *testing.T) {
	resp := `{
		"annotations": {
			"io.Reader": "Reads bytes from a source",
			"io.Writer": "Writes bytes to a destination",
			"os.File": "OS file handle for I/O operations",
			"bytes.Buffer": "In-memory byte buffer"
		}
	}`
	srv := mockLLMServer(t, resp)
	defer srv.Close()

	annotator := NewLLMAnnotator(newTestClient(srv.URL), NewDefaultAnnotator(), testLogger())
	annotations := annotator.Annotate(testResult())

	require.Len(t, annotations, 4)
	assert.Equal(t, "Reads bytes from a source", annotations["io.Reader"])
}

func TestLLMAnnotator_FallbackOnError(t *testing.T) {
	srv := mockLLMServerError(t)
	defer srv.Close()

	annotator := NewLLMAnnotator(newTestClient(srv.URL), NewDefaultAnnotator(), testLogger())
	annotations := annotator.Annotate(testResult())
	assert.Nil(t, annotations, "default annotator returns nil")
}

func TestLLMAnnotator_IgnoresInvalidKeys(t *testing.T) {
	resp := `{
		"annotations": {
			"io.Reader": "valid",
			"fake.Thing": "invalid key"
		}
	}`
	srv := mockLLMServer(t, resp)
	defer srv.Close()

	annotator := NewLLMAnnotator(newTestClient(srv.URL), NewDefaultAnnotator(), testLogger())
	annotations := annotator.Annotate(testResult())

	assert.Len(t, annotations, 1)
	assert.Equal(t, "valid", annotations["io.Reader"])
}

func TestLLMAnnotator_Enrich(t *testing.T) {
	srv := mockLLMServerError(t)
	defer srv.Close()

	annotator := NewLLMAnnotator(newTestClient(srv.URL), NewDefaultAnnotator(), testLogger())
	result := testResult()
	assert.Equal(t, result, annotator.Enrich(result))
}

// --- Scorer Tests ---

func TestLLMScorer_Success(t *testing.T) {
	resp := `{"scores": {"0": 0.9, "1": 0.8, "2": 0.5, "3": 0.4}}`
	srv := mockLLMServer(t, resp)
	defer srv.Close()

	scorer := NewLLMScorer(newTestClient(srv.URL), NewDefaultScorer(), testLogger())
	scores := scorer.Score(testResult().Relations)

	require.Len(t, scores, 4)
	assert.Equal(t, 0.9, scores[0])
	assert.Equal(t, 0.8, scores[1])
	assert.Equal(t, 0.5, scores[2])
	assert.Equal(t, 0.4, scores[3])
}

func TestLLMScorer_FallbackOnError(t *testing.T) {
	srv := mockLLMServerError(t)
	defer srv.Close()

	scorer := NewLLMScorer(newTestClient(srv.URL), NewDefaultScorer(), testLogger())
	scores := scorer.Score(testResult().Relations)

	require.Len(t, scores, 4)
	for _, s := range scores {
		assert.Equal(t, 1.0, s, "default scorer gives all 1.0")
	}
}

func TestLLMScorer_ClampsScores(t *testing.T) {
	resp := `{"scores": {"0": 1.5, "1": -0.5}}`
	srv := mockLLMServer(t, resp)
	defer srv.Close()

	scorer := NewLLMScorer(newTestClient(srv.URL), NewDefaultScorer(), testLogger())
	result := testResult()
	scores := scorer.Score(result.Relations)

	assert.Equal(t, 1.0, scores[0], "should clamp to 1.0")
	assert.Equal(t, 0.0, scores[1], "should clamp to 0.0")
	// Missing indices filled with default 1.0
	assert.Equal(t, 1.0, scores[2])
	assert.Equal(t, 1.0, scores[3])
}

func TestLLMScorer_IgnoresInvalidIndices(t *testing.T) {
	resp := `{"scores": {"0": 0.5, "99": 0.1, "abc": 0.2}}`
	srv := mockLLMServer(t, resp)
	defer srv.Close()

	scorer := NewLLMScorer(newTestClient(srv.URL), NewDefaultScorer(), testLogger())
	scores := scorer.Score(testResult().Relations)

	assert.Equal(t, 0.5, scores[0])
	assert.Equal(t, 1.0, scores[1], "missing index filled with default")
}

func TestLLMScorer_Enrich(t *testing.T) {
	srv := mockLLMServerError(t)
	defer srv.Close()

	scorer := NewLLMScorer(newTestClient(srv.URL), NewDefaultScorer(), testLogger())
	result := testResult()
	assert.Equal(t, result, scorer.Enrich(result))
}

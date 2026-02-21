package llm_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/olehluchkiv/goifaces/internal/enricher/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
}

func mockServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

func chatResponse(content string) []byte {
	resp := map[string]any{
		"choices": []map[string]any{
			{
				"message": map[string]string{
					"content": content,
				},
			},
		},
	}
	data, _ := json.Marshal(resp)
	return data
}

func TestComplete_Success(t *testing.T) {
	server := mockServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/chat/completions", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		var req map[string]any
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "test-model", req["model"])

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(chatResponse(`{"result": "ok"}`))
	})
	defer server.Close()

	client := llm.NewClient(llm.Config{
		Endpoint: server.URL,
		APIKey:   "test-key",
		Model:    "test-model",
	}, testLogger())

	result, err := client.Complete(context.Background(), "system", "user")
	require.NoError(t, err)
	assert.Equal(t, `{"result": "ok"}`, result)
}

func TestComplete_JSONMode(t *testing.T) {
	server := mockServer(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))

		rf, ok := req["response_format"].(map[string]any)
		require.True(t, ok, "response_format should be present")
		assert.Equal(t, "json_object", rf["type"])

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(chatResponse(`{}`))
	})
	defer server.Close()

	client := llm.NewClient(llm.Config{
		Endpoint: server.URL,
		APIKey:   "key",
		Model:    "model",
	}, testLogger())

	_, err := client.Complete(context.Background(), "sys", "usr")
	require.NoError(t, err)
}

func TestComplete_ServerError_Retries(t *testing.T) {
	var calls atomic.Int32
	server := mockServer(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("internal error"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(chatResponse(`{"retried": true}`))
	})
	defer server.Close()

	client := llm.NewClient(llm.Config{
		Endpoint: server.URL,
		APIKey:   "key",
		Model:    "model",
	}, testLogger())

	result, err := client.Complete(context.Background(), "sys", "usr")
	require.NoError(t, err)
	assert.Equal(t, `{"retried": true}`, result)
	assert.Equal(t, int32(2), calls.Load())
}

func TestComplete_ServerError_ExhaustedRetries(t *testing.T) {
	var calls atomic.Int32
	server := mockServer(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("always failing"))
	})
	defer server.Close()

	client := llm.NewClient(llm.Config{
		Endpoint: server.URL,
		APIKey:   "key",
		Model:    "model",
	}, testLogger())

	_, err := client.Complete(context.Background(), "sys", "usr")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed after retries")
	assert.Equal(t, int32(2), calls.Load())
}

func TestComplete_ClientError_NoRetry(t *testing.T) {
	var calls atomic.Int32
	server := mockServer(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad request"))
	})
	defer server.Close()

	client := llm.NewClient(llm.Config{
		Endpoint: server.URL,
		APIKey:   "key",
		Model:    "model",
	}, testLogger())

	_, err := client.Complete(context.Background(), "sys", "usr")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 400")
	assert.Equal(t, int32(1), calls.Load(), "should not retry on 4xx")
}

func TestComplete_Timeout(t *testing.T) {
	server := mockServer(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(chatResponse(`{}`))
	})
	defer server.Close()

	client := llm.NewClient(llm.Config{
		Endpoint: server.URL,
		APIKey:   "key",
		Model:    "model",
		Timeout:  100 * time.Millisecond,
	}, testLogger())

	_, err := client.Complete(context.Background(), "sys", "usr")
	require.Error(t, err)
}

func TestComplete_MalformedResponse(t *testing.T) {
	server := mockServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not json"))
	})
	defer server.Close()

	client := llm.NewClient(llm.Config{
		Endpoint: server.URL,
		APIKey:   "key",
		Model:    "model",
	}, testLogger())

	_, err := client.Complete(context.Background(), "sys", "usr")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestComplete_NoChoices(t *testing.T) {
	server := mockServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		resp, _ := json.Marshal(map[string]any{"choices": []any{}})
		_, _ = w.Write(resp)
	})
	defer server.Close()

	client := llm.NewClient(llm.Config{
		Endpoint: server.URL,
		APIKey:   "key",
		Model:    "model",
	}, testLogger())

	_, err := client.Complete(context.Background(), "sys", "usr")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no choices")
}

func TestComplete_APIError(t *testing.T) {
	server := mockServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		resp, _ := json.Marshal(map[string]any{
			"error": map[string]string{"message": "quota exceeded"},
		})
		_, _ = w.Write(resp)
	})
	defer server.Close()

	client := llm.NewClient(llm.Config{
		Endpoint: server.URL,
		APIKey:   "key",
		Model:    "model",
	}, testLogger())

	_, err := client.Complete(context.Background(), "sys", "usr")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "quota exceeded")
}

func TestComplete_ContextCanceled(t *testing.T) {
	server := mockServer(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(chatResponse(`{}`))
	})
	defer server.Close()

	client := llm.NewClient(llm.Config{
		Endpoint: server.URL,
		APIKey:   "key",
		Model:    "model",
		Timeout:  10 * time.Second,
	}, testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.Complete(ctx, "sys", "usr")
	require.Error(t, err)
}

func TestComplete_RateLimitWithRetryAfter(t *testing.T) {
	var calls atomic.Int32
	server := mockServer(func(w http.ResponseWriter, _ *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte("rate limited"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(chatResponse(`{"ok": true}`))
	})
	defer server.Close()

	client := llm.NewClient(llm.Config{
		Endpoint: server.URL,
		APIKey:   "key",
		Model:    "model",
	}, testLogger())

	result, err := client.Complete(context.Background(), "sys", "usr")
	require.NoError(t, err)
	assert.Equal(t, `{"ok": true}`, result)
	assert.Equal(t, int32(2), calls.Load())
}

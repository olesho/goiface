package llm

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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
}

func writeJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	require.NoError(t, json.NewEncoder(w).Encode(v))
}

func TestComplete_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/chat/completions", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		var req chatRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "test-model", req.Model)
		assert.Equal(t, "json_object", req.ResponseFormat.Type)
		assert.Len(t, req.Messages, 2)

		resp := chatResponse{
			Choices: []chatChoice{
				{Message: chatMessage{Content: `{"result": "ok"}`}},
			},
		}
		writeJSON(t, w, resp)
	}))
	defer srv.Close()

	client := NewClient(Config{
		Endpoint: srv.URL,
		APIKey:   "test-key",
		Model:    "test-model",
	}, testLogger())

	result, err := client.Complete(context.Background(), "system", "user")
	require.NoError(t, err)
	assert.Equal(t, `{"result": "ok"}`, result)
}

func TestComplete_RetryOn5xx(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := attempts.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			_, err := w.Write([]byte("internal error"))
			require.NoError(t, err)
			return
		}
		resp := chatResponse{
			Choices: []chatChoice{
				{Message: chatMessage{Content: `{"retry": "ok"}`}},
			},
		}
		writeJSON(t, w, resp)
	}))
	defer srv.Close()

	client := NewClient(Config{
		Endpoint: srv.URL,
		APIKey:   "key",
		Model:    "model",
	}, testLogger())

	result, err := client.Complete(context.Background(), "sys", "usr")
	require.NoError(t, err)
	assert.Equal(t, `{"retry": "ok"}`, result)
	assert.Equal(t, int32(2), attempts.Load())
}

func TestComplete_NoRetryOn4xx(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte("bad request"))
		require.NoError(t, err)
	}))
	defer srv.Close()

	client := NewClient(Config{
		Endpoint: srv.URL,
		APIKey:   "key",
		Model:    "model",
	}, testLogger())

	_, err := client.Complete(context.Background(), "sys", "usr")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "400")
	assert.Equal(t, int32(1), attempts.Load(), "should not retry on 4xx")
}

func TestComplete_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewClient(Config{
		Endpoint: srv.URL,
		APIKey:   "key",
		Model:    "model",
		Timeout:  100 * time.Millisecond,
	}, testLogger())

	_, err := client.Complete(context.Background(), "sys", "usr")
	require.Error(t, err)
}

func TestComplete_MalformedResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte("not json"))
		require.NoError(t, err)
	}))
	defer srv.Close()

	client := NewClient(Config{
		Endpoint: srv.URL,
		APIKey:   "key",
		Model:    "model",
	}, testLogger())

	_, err := client.Complete(context.Background(), "sys", "usr")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestComplete_EmptyChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := chatResponse{Choices: []chatChoice{}}
		writeJSON(t, w, resp)
	}))
	defer srv.Close()

	client := NewClient(Config{
		Endpoint: srv.URL,
		APIKey:   "key",
		Model:    "model",
	}, testLogger())

	_, err := client.Complete(context.Background(), "sys", "usr")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no choices")
}

func TestComplete_NoAPIKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.Header.Get("Authorization"))
		resp := chatResponse{
			Choices: []chatChoice{
				{Message: chatMessage{Content: `{}`}},
			},
		}
		writeJSON(t, w, resp)
	}))
	defer srv.Close()

	client := NewClient(Config{
		Endpoint: srv.URL,
		Model:    "model",
	}, testLogger())

	_, err := client.Complete(context.Background(), "sys", "usr")
	require.NoError(t, err)
}

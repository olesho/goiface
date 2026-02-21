package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

// Client is a lightweight HTTP client for OpenAI-compatible chat completions API.
type Client struct {
	endpoint   string
	apiKey     string
	model      string
	httpClient *http.Client
	logger     *slog.Logger
}

// Config holds configuration for the LLM client.
type Config struct {
	Endpoint string // API base URL (e.g., "https://api.openai.com/v1")
	APIKey   string
	Model    string
	Timeout  time.Duration // Per-request timeout (default: 30s)
}

// NewClient creates an LLM client with the given configuration.
func NewClient(cfg Config, logger *slog.Logger) *Client {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	return &Client{
		endpoint: cfg.Endpoint,
		apiKey:   cfg.APIKey,
		model:    cfg.Model,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		logger: logger.With("component", "llm-client"),
	}
}

// chatRequest is the request body for the chat completions API.
type chatRequest struct {
	Model          string          `json:"model"`
	Messages       []chatMessage   `json:"messages"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
	Temperature    float64         `json:"temperature"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseFormat struct {
	Type string `json:"type"`
}

// chatResponse is the response body from the chat completions API.
type chatResponse struct {
	Choices []chatChoice `json:"choices"`
	Error   *apiError    `json:"error,omitempty"`
}

type chatChoice struct {
	Message chatMessage `json:"message"`
}

type apiError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// Complete sends a chat completion request and returns the raw JSON response content.
// It uses JSON mode and includes one retry on 5xx errors.
func (c *Client) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	reqBody := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		ResponseFormat: &responseFormat{Type: "json_object"},
		Temperature:    0.2,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := c.endpoint + "/chat/completions"

	// Try up to 2 times (initial + 1 retry on 5xx)
	var lastErr error
	for attempt := range 2 {
		if attempt > 0 {
			c.logger.Warn("retrying LLM request", "attempt", attempt+1)
		}

		result, retryable, err := c.doRequest(ctx, url, body)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if !retryable {
			return "", lastErr
		}
	}
	return "", lastErr
}

// doRequest performs a single HTTP request. Returns (result, retryable, error).
func (c *Client) doRequest(ctx context.Context, url string, body []byte) (string, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", false, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", false, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false, fmt.Errorf("read response: %w", err)
	}

	// Handle rate limiting
	if resp.StatusCode == http.StatusTooManyRequests {
		delay := 2 * time.Second
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
			if secs, parseErr := strconv.Atoi(retryAfter); parseErr == nil && secs > 0 && secs <= 60 {
				delay = time.Duration(secs) * time.Second
			}
		}
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-ctx.Done():
			return "", false, ctx.Err()
		}
		return "", true, fmt.Errorf("rate limited (429)")
	}

	// 5xx = retryable
	if resp.StatusCode >= 500 {
		return "", true, fmt.Errorf("server error: %d %s", resp.StatusCode, string(respBody))
	}

	// Other non-2xx = not retryable
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", false, fmt.Errorf("api error: %d %s", resp.StatusCode, string(respBody))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", false, fmt.Errorf("unmarshal response: %w", err)
	}

	if chatResp.Error != nil {
		return "", false, fmt.Errorf("api error: %s: %s", chatResp.Error.Type, chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", false, fmt.Errorf("no choices in response")
	}

	return chatResp.Choices[0].Message.Content, false, nil
}

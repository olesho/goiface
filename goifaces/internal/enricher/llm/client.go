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

// Config holds LLM client configuration.
type Config struct {
	Endpoint string // API base URL (e.g., https://api.openai.com/v1)
	APIKey   string
	Model    string
	Timeout  time.Duration
}

// LogValue masks the API key when the config is logged via slog.
func (c Config) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("endpoint", c.Endpoint),
		slog.String("model", c.Model),
		slog.String("api_key", "[REDACTED]"),
	)
}

// Client speaks the OpenAI-compatible chat completions API.
type Client struct {
	cfg    Config
	http   *http.Client
	logger *slog.Logger
}

// NewClient creates an LLM client with the given configuration.
func NewClient(cfg Config, logger *slog.Logger) *Client {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	return &Client{
		cfg:    cfg,
		http:   &http.Client{Timeout: cfg.Timeout},
		logger: logger.With("component", "llm-client"),
	}
}

// chatRequest is the OpenAI chat completions request body.
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

// chatResponse is the OpenAI chat completions response body.
type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Complete sends a chat completion request and returns the raw JSON response content.
func (c *Client) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	reqBody := chatRequest{
		Model: c.cfg.Model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		ResponseFormat: &responseFormat{Type: "json_object"},
		Temperature:    0.2,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	endpoint := c.cfg.Endpoint + "/chat/completions"

	// Try up to 2 times (initial + 1 retry on 5xx)
	var lastErr error
	for attempt := range 2 {
		if attempt > 0 {
			c.logger.Debug("retrying LLM request", "attempt", attempt+1)
		}

		result, err := c.doRequest(ctx, endpoint, data)
		if err == nil {
			return result, nil
		}
		lastErr = err

		// Only retry on server errors
		if !isRetryable(err) {
			return "", err
		}

		// Respect Retry-After header if present
		if re, ok := err.(*serverError); ok && re.retryAfter > 0 {
			select {
			case <-time.After(re.retryAfter):
			case <-ctx.Done():
				return "", ctx.Err()
			}
		} else {
			select {
			case <-time.After(time.Second):
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}
	}

	return "", fmt.Errorf("LLM request failed after retries: %w", lastErr)
}

func (c *Client) doRequest(ctx context.Context, endpoint string, data []byte) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)

	c.logger.Debug("sending LLM request", "endpoint", endpoint, "model", c.cfg.Model)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	const maxBodyBytes = 10 * 1024 * 1024 // 10 MB
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	// Handle rate limiting
	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		return "", &serverError{statusCode: resp.StatusCode, retryAfter: retryAfter}
	}

	// Handle server errors (retryable)
	if resp.StatusCode >= 500 {
		return "", &serverError{statusCode: resp.StatusCode}
	}

	// Handle client errors (non-retryable)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("LLM API error (status %d): %s", resp.StatusCode, string(body))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("LLM API error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("LLM returned no choices")
	}

	content := chatResp.Choices[0].Message.Content
	c.logger.Debug("received LLM response", "length", len(content))
	return content, nil
}

type serverError struct {
	statusCode int
	retryAfter time.Duration
}

func (e *serverError) Error() string {
	return fmt.Sprintf("server error: status %d", e.statusCode)
}

func isRetryable(err error) bool {
	_, ok := err.(*serverError)
	return ok
}

func parseRetryAfter(val string) time.Duration {
	if val == "" {
		return 0
	}
	// Try parsing as seconds
	if seconds, err := strconv.Atoi(val); err == nil {
		return time.Duration(seconds) * time.Second
	}
	return 0
}

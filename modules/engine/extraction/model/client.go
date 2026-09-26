// Byline: Claude Code · Opus 5.5 · 2026-09-25
//
// Package model is the one place entity/event extraction talks to a language
// model. The provider is configuration, not code: any OpenAI-compatible
// chat-completions endpoint. The owner's 2026-09-25 choice is
// moonshotai/kimi-k3 on NVIDIA NIM with response_format json_object
// (NIM guided_json returns HTTP 400 for this model). GLM models are refused
// outright (owner rule 2026-09-25: "Stop using GLM 5.1 in general").
package model

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Environment names read once at worker boot.
const (
	EnvBaseURL    = "ENTITY_MODEL_BASE_URL"
	EnvModelID    = "ENTITY_MODEL_ID"
	EnvAPIKeyFile = "ENTITY_MODEL_API_KEY_FILE"
	EnvMaxTokens  = "ENTITY_MODEL_MAX_TOKENS"
)

// Defaults: the owner's 2026-09-25 choice.
const (
	DefaultBaseURL   = "https://integrate.api.nvidia.com/v1"
	DefaultModelID   = "moonshotai/kimi-k3"
	DefaultMaxTokens = 6000
	// MinMaxTokens: kimi-k3 is a reasoning model; below ~1500 the answer
	// content comes back empty (parent probe 2026-09-25).
	MinMaxTokens = 1500
)

// Config selects the model endpoint.
type Config struct {
	BaseURL   string
	ModelID   string
	APIKey    string
	MaxTokens int
}

// ErrDisabled means no model is configured; model extraction is skipped
// with a visible flag, and rule-based proposals still work.
var ErrDisabled = errors.New("entity model is not configured")

// ConfigFromEnv reads the model configuration. A missing key file disables
// the model (ErrDisabled); a present but invalid configuration is an error.
func ConfigFromEnv() (Config, error) {
	keyFile := strings.TrimSpace(os.Getenv(EnvAPIKeyFile))
	if keyFile == "" {
		return Config{}, ErrDisabled
	}
	cfg := Config{
		BaseURL:   strings.TrimRight(strings.TrimSpace(os.Getenv(EnvBaseURL)), "/"),
		ModelID:   strings.TrimSpace(os.Getenv(EnvModelID)),
		MaxTokens: DefaultMaxTokens,
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	if cfg.ModelID == "" {
		cfg.ModelID = DefaultModelID
	}
	if raw := strings.TrimSpace(os.Getenv(EnvMaxTokens)); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			return Config{}, fmt.Errorf("%s must be an integer", EnvMaxTokens)
		}
		cfg.MaxTokens = value
	}
	key, err := readKeyFile(keyFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, ErrDisabled
		}
		return Config{}, err
	}
	cfg.APIKey = key
	return cfg, cfg.Validate()
}

// Validate rejects configurations that must never run.
func (c Config) Validate() error {
	if strings.Contains(strings.ToLower(c.ModelID), "glm") {
		return fmt.Errorf("model %q refused: GLM models are not used (owner rule 2026-09-25)", c.ModelID)
	}
	if c.ModelID == "" || c.BaseURL == "" || c.APIKey == "" {
		return errors.New("entity model configuration is incomplete")
	}
	if !strings.HasPrefix(c.BaseURL, "https://") && !strings.HasPrefix(c.BaseURL, "http://") {
		return errors.New("entity model base URL must be http(s)")
	}
	if c.MaxTokens < MinMaxTokens {
		return fmt.Errorf("max tokens %d is below %d; a reasoning model returns empty content", c.MaxTokens, MinMaxTokens)
	}
	return nil
}

func readKeyFile(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("%s must be an absolute path", EnvAPIKeyFile)
	}
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Size() < 8 || info.Size() > 4096 {
		return "", fmt.Errorf("%s must be a regular file of 8-4096 bytes", EnvAPIKeyFile)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	key := strings.TrimSpace(string(raw))
	// Tolerate a KEY=value line so an env-style secret file can be mounted.
	if name, value, ok := strings.Cut(key, "="); ok && !strings.ContainsAny(name, " \t") && strings.Contains(name, "KEY") {
		key = strings.Trim(strings.TrimSpace(value), `"'`)
	}
	if key == "" || strings.ContainsAny(key, " \t\r\n") {
		return "", fmt.Errorf("%s does not hold a single key", EnvAPIKeyFile)
	}
	return key, nil
}

// Message is one chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Completion is the part of a chat completion extraction uses.
type Completion struct {
	Content      string
	FinishReason string
	Model        string
	// Diagnostics: how much the model reasoned and how many tokens it used.
	ReasoningChars   int
	CompletionTokens int
	PromptTokens     int
}

// Client calls the chat-completions endpoint.
type Client struct {
	Config Config
	HTTP   *http.Client
	// Sleep waits between retries; tests replace it.
	Sleep func(context.Context, time.Duration) error
	// MaxAttempts bounds transport retries (429, 5xx, network) per call.
	MaxAttempts int
	// InitialBackoff doubles after each retry unless Retry-After says more.
	InitialBackoff time.Duration
}

// NewClient builds a client for a validated configuration.
func NewClient(cfg Config) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &Client{Config: cfg, HTTP: &http.Client{Timeout: 240 * time.Second}, Sleep: sleepContext, MaxAttempts: 6, InitialBackoff: 5 * time.Second}, nil
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// CallOptions tunes one call. Thinking nil leaves the model's default; set,
// it is sent as chat_template_kwargs.thinking (the switch kimi-k3 honours on
// NIM: false answered in 0.6 s with no reasoning, probe 2026-09-25).
type CallOptions struct {
	Thinking *bool
}

type chatRequest struct {
	Model              string            `json:"model"`
	Messages           []Message         `json:"messages"`
	ResponseFormat     map[string]string `json:"response_format"`
	MaxTokens          int               `json:"max_tokens"`
	Temperature        float64           `json:"temperature"`
	ChatTemplateKwargs map[string]any    `json:"chat_template_kwargs,omitempty"`
}

type chatResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content          *string `json:"content"`
			ReasoningContent *string `json:"reasoning_content"`
			Reasoning        *string `json:"reasoning"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

// TransportError is a non-retryable HTTP failure.
type TransportError struct {
	Status int
	Body   string
}

func (e *TransportError) Error() string {
	return fmt.Sprintf("model endpoint returned HTTP %d: %s", e.Status, e.Body)
}

// Complete sends one JSON-object chat completion, retrying rate limits and
// server errors with exponential backoff (honouring Retry-After).
func (c *Client) Complete(ctx context.Context, messages []Message, options CallOptions) (Completion, error) {
	request := chatRequest{
		Model: c.Config.ModelID, Messages: messages,
		ResponseFormat: map[string]string{"type": "json_object"},
		MaxTokens:      c.Config.MaxTokens, Temperature: 0,
	}
	if options.Thinking != nil {
		request.ChatTemplateKwargs = map[string]any{"thinking": *options.Thinking}
	}
	body, err := json.Marshal(request)
	if err != nil {
		return Completion{}, err
	}
	attempts := c.MaxAttempts
	if attempts < 1 {
		attempts = 1
	}
	backoff := c.InitialBackoff
	if backoff <= 0 {
		backoff = 2 * time.Second
	}
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		completion, retryAfter, err := c.once(ctx, body)
		if err == nil {
			return completion, nil
		}
		lastErr = err
		var transport *TransportError
		if errors.As(err, &transport) && transport.Status != http.StatusTooManyRequests && transport.Status < 500 {
			return Completion{}, err
		}
		if ctx.Err() != nil {
			return Completion{}, ctx.Err()
		}
		if attempt == attempts {
			break
		}
		wait := backoff
		if retryAfter > wait {
			wait = retryAfter
		}
		if err := c.Sleep(ctx, wait); err != nil {
			return Completion{}, err
		}
		backoff *= 2
	}
	return Completion{}, fmt.Errorf("model endpoint failed after %d attempts: %w", attempts, lastErr)
}

func (c *Client) once(ctx context.Context, body []byte) (Completion, time.Duration, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Config.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Completion{}, 0, err
	}
	request.Header.Set("Authorization", "Bearer "+c.Config.APIKey)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	response, err := c.HTTP.Do(request)
	if err != nil {
		return Completion{}, 0, err
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, 8<<20))
	if err != nil {
		return Completion{}, 0, err
	}
	if response.StatusCode != http.StatusOK {
		retryAfter := time.Duration(0)
		if seconds, convErr := strconv.Atoi(strings.TrimSpace(response.Header.Get("Retry-After"))); convErr == nil && seconds > 0 && seconds <= 120 {
			retryAfter = time.Duration(seconds) * time.Second
		}
		snippet := strings.TrimSpace(string(raw))
		if len(snippet) > 300 {
			snippet = snippet[:300]
		}
		return Completion{}, retryAfter, &TransportError{Status: response.StatusCode, Body: snippet}
	}
	var decoded chatResponse
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return Completion{}, 0, fmt.Errorf("model endpoint returned malformed JSON: %w", err)
	}
	if len(decoded.Choices) == 0 {
		return Completion{}, 0, errors.New("model endpoint returned no choices")
	}
	choice := decoded.Choices[0]
	content := ""
	if choice.Message.Content != nil {
		content = *choice.Message.Content
	}
	reasoning := 0
	for _, value := range []*string{choice.Message.ReasoningContent, choice.Message.Reasoning} {
		if value != nil {
			reasoning += len(*value)
		}
	}
	return Completion{
		Content: content, FinishReason: choice.FinishReason, Model: decoded.Model, ReasoningChars: reasoning,
		CompletionTokens: decoded.Usage.CompletionTokens, PromptTokens: decoded.Usage.PromptTokens,
	}, 0, nil
}

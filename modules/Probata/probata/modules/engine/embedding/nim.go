// Package embedding is the text-embedding boundary for the pre-approval search
// surface. It owns one HTTP call shape, the OpenAI-compatible /embeddings
// endpoint NVIDIA NIM serves, and nothing else.
//
// The model is the one every writer of MsgEvents20260918 already uses:
// nvidia/nemotron-3-embed-1b, 2048 dimensions, input_type=passage, batched
// (one request embeds a whole page; probed with a 4-text call 2026-09-18 and
// 2026-09-26). The collection's text_nim vector has vectorizer "none", so the
// writer must supply the vector; a vector from any other model would be
// silently incomparable with the 366k vectors already there.
//
// Two live NIM failure shapes are handled before the request is sent (owner
// global notes, 2026-09-26): the whole request fails when any input contains
// the lowercase text "data:image/", or when an input is blank. Both are
// rewritten rather than dropped, so the batch keeps its length and order.
//
// Byline: Claude Code · Opus 5.5 · 2026-10-01
package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	// DefaultBaseURL is NVIDIA's hosted NIM endpoint.
	DefaultBaseURL = "https://integrate.api.nvidia.com/v1"
	// DefaultModel is the collection's embedder.
	DefaultModel = "nvidia/nemotron-3-embed-1b"
	// DefaultDimensions is that model's vector length.
	DefaultDimensions = 2048
	// maxInputChars mirrors the Case Bible writers' 8000-character cut.
	maxInputChars = 8000
	// blankPlaceholder replaces an empty input.
	blankPlaceholder = "(empty)"
)

// NIM embeds text through an OpenAI-compatible /embeddings endpoint.
type NIM struct {
	BaseURL    string
	Model      string
	Dimensions int
	// APIKey is sent as a bearer token. It is never logged and never appears
	// in an error.
	APIKey string
	HTTP   *http.Client
	// Retries bounds retries on HTTP 429 and 5xx. Zero means 4.
	Retries int
	// Sleep is injectable for tests; nil uses a context-aware timer.
	Sleep func(context.Context, time.Duration) error
}

func (n NIM) baseURL() string {
	if strings.TrimSpace(n.BaseURL) == "" {
		return DefaultBaseURL
	}
	return strings.TrimRight(n.BaseURL, "/")
}

func (n NIM) model() string {
	if strings.TrimSpace(n.Model) == "" {
		return DefaultModel
	}
	return n.Model
}

// ModelName is the model id recorded on every object (embed_model).
func (n NIM) ModelName() string { return n.model() }

func (n NIM) dimensions() int {
	if n.Dimensions <= 0 {
		return DefaultDimensions
	}
	return n.Dimensions
}

func (n NIM) client() *http.Client {
	if n.HTTP != nil {
		return n.HTTP
	}
	return &http.Client{Timeout: 3 * time.Minute}
}

func (n NIM) sleep(ctx context.Context, d time.Duration) error {
	if n.Sleep != nil {
		return n.Sleep(ctx, d)
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// PrepareInput applies the NIM input guards and the length cut to one text.
func PrepareInput(text string) string {
	text = strings.ReplaceAll(text, "data:image/", "data: image/")
	if len(text) > maxInputChars {
		cut := maxInputChars
		for cut > 0 && !utf8Start(text[cut]) {
			cut--
		}
		text = text[:cut]
	}
	if strings.TrimSpace(text) == "" {
		return blankPlaceholder
	}
	return text
}

func utf8Start(b byte) bool { return b&0xC0 != 0x80 }

type embedRequest struct {
	Model          string   `json:"model"`
	Input          []string `json:"input"`
	InputType      string   `json:"input_type"`
	EncodingFormat string   `json:"encoding_format"`
	Truncate       string   `json:"truncate"`
}

type embedResponse struct {
	Data []struct {
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

// Embed returns one vector per text, in input order, or an error. It never
// returns a partial batch.
func (n NIM) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if strings.TrimSpace(n.APIKey) == "" {
		return nil, errors.New("embedding: NIM api key is not configured")
	}
	if len(texts) == 0 {
		return nil, errors.New("embedding: at least one text is required")
	}
	inputs := make([]string, len(texts))
	for i, text := range texts {
		inputs[i] = PrepareInput(text)
	}
	body, err := json.Marshal(embedRequest{
		Model: n.model(), Input: inputs, InputType: "passage", EncodingFormat: "float", Truncate: "END",
	})
	if err != nil {
		return nil, fmt.Errorf("embedding: encode request: %w", err)
	}
	retries := n.Retries
	if retries <= 0 {
		retries = 4
	}
	var lastErr error
	for attempt := 0; attempt <= retries; attempt++ {
		if attempt > 0 {
			if err := n.sleep(ctx, time.Duration(5*attempt)*time.Second); err != nil {
				return nil, err
			}
		}
		vectors, retryable, err := n.post(ctx, body, len(texts))
		if err == nil {
			return vectors, nil
		}
		lastErr = err
		if !retryable {
			break
		}
	}
	return nil, lastErr
}

func (n NIM) post(ctx context.Context, body []byte, want int) ([][]float32, bool, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, n.baseURL()+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, false, fmt.Errorf("embedding: build request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+n.APIKey)
	response, err := n.client().Do(request)
	if err != nil {
		return nil, true, fmt.Errorf("embedding: POST /embeddings: %w", err)
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(response.Body, 256<<20))
	if err != nil {
		return nil, true, fmt.Errorf("embedding: read response: %w", err)
	}
	if response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500 {
		return nil, true, fmt.Errorf("embedding: NIM returned HTTP %d: %s", response.StatusCode, truncate(payload))
	}
	if response.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("embedding: NIM returned HTTP %d: %s", response.StatusCode, truncate(payload))
	}
	var decoded embedResponse
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, false, fmt.Errorf("embedding: decode response: %w", err)
	}
	if len(decoded.Data) != want {
		return nil, false, fmt.Errorf("embedding: NIM returned %d vectors for %d texts", len(decoded.Data), want)
	}
	vectors := make([][]float32, want)
	for _, item := range decoded.Data {
		if item.Index < 0 || item.Index >= want || vectors[item.Index] != nil {
			return nil, false, fmt.Errorf("embedding: NIM returned an invalid or repeated index %d", item.Index)
		}
		if len(item.Embedding) != n.dimensions() {
			return nil, false, fmt.Errorf("embedding: NIM returned a %d-dimension vector, want %d", len(item.Embedding), n.dimensions())
		}
		vectors[item.Index] = item.Embedding
	}
	return vectors, false, nil
}

func truncate(payload []byte) string {
	const limit = 300
	text := strings.TrimSpace(string(payload))
	if len(text) > limit {
		return text[:limit] + "..."
	}
	return text
}

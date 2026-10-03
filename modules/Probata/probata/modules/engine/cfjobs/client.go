// Byline: Claude Code · Sonnet 5.5 · 2026-10-02

package cfjobs

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go.temporal.io/sdk/temporal"
)

// ErrorTypeBadRequest marks a Worker answer that no retry can fix (401, 400, 404): wrong token, wrong bucket, wrong path.
const ErrorTypeBadRequest = "cf_worker_bad_request"

// WorkerClient calls one Cloudflare Worker over HTTPS with its bearer token.
type WorkerClient struct {
	// BaseURL is the Worker's origin, e.g. https://casebible-format-sniffer.<account>.workers.dev (no trailing slash).
	BaseURL string
	// Token is the Worker's secret, sent as `Authorization: Bearer`.
	Token string
	// HTTP is the client to use; nil means a client with no overall timeout (the Activity's own timeouts bound the call).
	HTTP *http.Client
}

func (c *WorkerClient) client() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{}
}

func (c *WorkerClient) post(ctx context.Context, path string, body any) (*http.Response, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, temporal.NewNonRetryableApplicationError("encode Worker request", ErrorTypeBadRequest, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.BaseURL, "/")+path, bytes.NewReader(payload))
	if err != nil {
		return nil, temporal.NewNonRetryableApplicationError("build Worker request", ErrorTypeBadRequest, err)
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "propria-cfjobs/1")
	resp, err := c.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("call Worker %s: %w", path, err) // network failure: retryable
	}
	if resp.StatusCode == http.StatusOK {
		return resp, nil
	}
	defer resp.Body.Close()
	detail, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	message := fmt.Sprintf("Worker %s answered HTTP %d: %s", path, resp.StatusCode, strings.TrimSpace(string(detail)))
	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound:
		return nil, temporal.NewNonRetryableApplicationError(message, ErrorTypeBadRequest, nil)
	}
	return nil, errors.New(message) // 5xx, 429 and the rest: retryable
}

// PostJSON sends `body` to `path` and decodes the single JSON answer into `out`.
//
// A 400, 401, 403 or 404 comes back as a non-retryable error; anything else that fails is retryable.
func (c *WorkerClient) PostJSON(ctx context.Context, path string, body, out any) error {
	resp, err := c.post(ctx, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode Worker %s answer: %w", path, err)
	}
	return nil
}

// PostNDJSON sends `body` to `path` and calls `onLine` with each line of the NDJSON answer as it arrives.
//
// The answer must end with a `{"type":"done"}` line; a body that stops without it is a cut call and an error, so the
// caller retries. A `{"type":"fatal"}` line is also an error. `onLine` returning an error stops the read.
func (c *WorkerClient) PostNDJSON(ctx context.Context, path string, body any, onLine func(line []byte) error) error {
	resp, err := c.post(ctx, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	reader := bufio.NewReaderSize(resp.Body, 1<<20)
	done := false
	for {
		line, readErr := reader.ReadBytes('\n')
		line = bytes.TrimSpace(line)
		if len(line) > 0 {
			var head struct {
				Type  string `json:"type"`
				Error string `json:"error"`
			}
			if err := json.Unmarshal(line, &head); err != nil {
				return fmt.Errorf("Worker %s sent a line that is not JSON: %w", path, err)
			}
			switch head.Type {
			case "fatal":
				return fmt.Errorf("Worker %s failed: %s", path, head.Error)
			case "done":
				done = true
			}
			if err := onLine(line); err != nil {
				return err
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			return fmt.Errorf("read Worker %s answer: %w", path, readErr)
		}
	}
	if !done {
		return fmt.Errorf("Worker %s answer ended without its done line (call was cut)", path)
	}
	return nil
}

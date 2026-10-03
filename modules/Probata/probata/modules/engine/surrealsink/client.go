// Byline: Claude Code · Sonnet 5.5 · 2026-10-02
//
// Package surrealsink is the one place the engine writes to surreal-case
// (SurrealDB 3.2.4 on ovh-files, tailnet only). Conversations land in their own
// conv_* tables, linked to the case, and never touch the Family Court Toolkit's
// tables. Every write is an UPSERT under a deterministic record id, so sending a
// conversation twice leaves one copy.
package surrealsink

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Environment names read once at worker boot.
const (
	EnvURL          = "SURREAL_CASE_URL"
	EnvNamespace    = "SURREAL_CASE_NAMESPACE"
	EnvDatabase     = "SURREAL_CASE_DATABASE"
	EnvUserFile     = "SURREAL_CASE_USER_FILE"
	EnvPasswordFile = "SURREAL_CASE_PASSWORD_FILE"
	EnvAuthLevel    = "SURREAL_CASE_AUTH_LEVEL"
	EnvCaseRef      = "SURREAL_CASE_STATUS_ID"
)

// Defaults: the toolkit's case database. The conv_ tables are new and separate.
const (
	DefaultNamespace = "fct"
	DefaultDatabase  = "case"
	DefaultCaseRef   = "current"
)

// ErrNotConfigured means no Surreal connection is configured on this worker.
var ErrNotConfigured = errors.New("surreal-case is not configured on this worker (SURREAL_CASE_URL and the credential files are not set)")

// Config selects the Surreal connection.
type Config struct {
	URL       string
	Namespace string
	Database  string
	User      string
	Password  string
	// AuthLevel is "database" (a database user, the intended login) or "root".
	AuthLevel string
	// CaseRef is the id in the toolkit's case_status table that threads link to.
	CaseRef string
}

// ConfigFromEnv reads the connection. An unset URL is ErrNotConfigured; a set URL with
// unreadable credentials is an error.
func ConfigFromEnv() (Config, error) {
	address := strings.TrimRight(strings.TrimSpace(os.Getenv(EnvURL)), "/")
	if address == "" {
		return Config{}, ErrNotConfigured
	}
	cfg := Config{
		URL: address, Namespace: envOr(EnvNamespace, DefaultNamespace), Database: envOr(EnvDatabase, DefaultDatabase),
		AuthLevel: envOr(EnvAuthLevel, "database"), CaseRef: envOr(EnvCaseRef, DefaultCaseRef),
	}
	var err error
	if cfg.User, err = readSecret(os.Getenv(EnvUserFile)); err != nil {
		return Config{}, fmt.Errorf("%s: %w", EnvUserFile, err)
	}
	if cfg.Password, err = readSecret(os.Getenv(EnvPasswordFile)); err != nil {
		return Config{}, fmt.Errorf("%s: %w", EnvPasswordFile, err)
	}
	return cfg, cfg.Validate()
}

// Validate rejects configurations that must never run.
func (c Config) Validate() error {
	if !strings.HasPrefix(c.URL, "http://") && !strings.HasPrefix(c.URL, "https://") {
		return errors.New("surreal URL must be http(s)")
	}
	if c.Namespace == "" || c.Database == "" || c.User == "" || c.Password == "" {
		return errors.New("surreal configuration is incomplete")
	}
	if c.AuthLevel != "database" && c.AuthLevel != "root" {
		return errors.New("surreal auth level must be database or root")
	}
	return nil
}

func envOr(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func readSecret(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" || !filepath.IsAbs(path) {
		return "", errors.New("must be an absolute path")
	}
	// A credential file that is not mounted (or that Docker made into a directory because the host file was
	// absent) disables the send with a reason; it must not stop the worker from importing.
	if info, statErr := os.Stat(path); statErr != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("%w: %s is not a mounted file", ErrNotConfigured, path)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	value := strings.TrimSpace(string(raw))
	if value == "" || strings.ContainsAny(value, "\r\n") {
		return "", errors.New("must hold one single-line value")
	}
	return value, nil
}

// Client talks to SurrealDB over HTTP (/rpc, method "query", with bound variables).
type Client struct {
	cfg  Config
	http *http.Client
}

// New validates the configuration.
func New(cfg Config, httpClient *http.Client) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	return &Client{cfg: cfg, http: httpClient}, nil
}

// Config returns the connection settings (without secrets in logs: callers print only Namespace/Database).
func (c *Client) Config() Config { return c.cfg }

type rpcRequest struct {
	ID     int    `json:"id"`
	Method string `json:"method"`
	Params []any  `json:"params"`
}

type rpcResponse struct {
	Result []statementResult `json:"result"`
	Error  *rpcError         `json:"error"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type statementResult struct {
	Status string          `json:"status"`
	Time   string          `json:"time"`
	Result json.RawMessage `json:"result"`
}

// maxBody mirrors the instance's request limit: stay far below it.
const maxBody = 600 * 1024

// Query runs SurrealQL with bound variables and returns one raw result per statement.
// Any failed statement is an error (SurrealDB reports them inside a 200 response).
func (c *Client) Query(ctx context.Context, sql string, vars map[string]any) ([]json.RawMessage, error) {
	payload, err := json.Marshal(rpcRequest{ID: 1, Method: "query", Params: []any{sql, vars}})
	if err != nil {
		return nil, err
	}
	if len(payload) > maxBody {
		return nil, fmt.Errorf("surreal request is %d bytes; the limit here is %d", len(payload), maxBody)
	}
	endpoint, err := url.JoinPath(c.cfg.URL, "rpc")
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("surreal-ns", c.cfg.Namespace)
	request.Header.Set("surreal-db", c.cfg.Database)
	if c.cfg.AuthLevel == "database" {
		request.Header.Set("surreal-auth-ns", c.cfg.Namespace)
		request.Header.Set("surreal-auth-db", c.cfg.Database)
	}
	request.SetBasicAuth(c.cfg.User, c.cfg.Password)
	response, err := c.http.Do(request)
	if err != nil {
		return nil, fmt.Errorf("surreal request failed: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("surreal answered HTTP %d: %s", response.StatusCode, clip(string(body), 300))
	}
	var decoded rpcResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, fmt.Errorf("surreal reply is not JSON: %w", err)
	}
	if decoded.Error != nil {
		return nil, fmt.Errorf("surreal error %d: %s", decoded.Error.Code, clip(decoded.Error.Message, 300))
	}
	out := make([]json.RawMessage, len(decoded.Result))
	for i, statement := range decoded.Result {
		if statement.Status != "OK" {
			return nil, fmt.Errorf("surreal statement %d failed: %s", i, clip(string(statement.Result), 300))
		}
		out[i] = statement.Result
	}
	return out, nil
}

func clip(value string, limit int) string {
	if len(value) > limit {
		return value[:limit] + "..."
	}
	return value
}

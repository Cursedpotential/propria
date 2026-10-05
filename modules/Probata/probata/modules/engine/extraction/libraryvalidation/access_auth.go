// Byline: Codex · GPT-6.1 · 2026-10-04.
package libraryvalidation

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Cursedpotential/probata/engine/surrealsink"
)

const EnvDBAccess = "TOOLKIT_VALIDATION_DB_ACCESS"
const ValidatorAccess = "toolkit_validator"
const CurrencyAccess = "toolkit_currency"
const ValidatorPrincipal = "toolkit_service:validator"
const CurrencyPrincipal = "toolkit_service:currency"

// AccessConfig selects a record principal rather than an unrestricted database EDITOR user.
// Inputs: shared connection, mounted username/password values, fixed access method and expected principal.
// Outputs: scoped HTTP authentication configuration; effects: none. Choose for all receipt/currency writers.
type AccessConfig struct {
	DB        surrealsink.Config
	Access    string
	Principal string
}

// NewAccessSQLClient adds bounded record signin and bearer authentication without changing shared surrealsink.
// Inputs: explicit record-access configuration and optional trusted transport. Outputs: existing SQLClient with scoped HTTP.
// Effects: no I/O until Query; signin is cached/refreshed on demand. No database/root Basic Auth fallback is permitted.
func NewAccessSQLClient(cfg AccessConfig, transport http.RoundTripper) (SQLClient, error) {
	u, err := url.Parse(cfg.DB.URL)
	validRole := (cfg.Access == ValidatorAccess && cfg.Principal == ValidatorPrincipal) || (cfg.Access == CurrencyAccess && cfg.Principal == CurrencyPrincipal)
	if cfg.DB.Validate() != nil || cfg.DB.AuthLevel != "database" || err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || !validRole || len(cfg.DB.User) > 128 || len(cfg.DB.Password) > 256 {
		return SQLClient{}, errors.New("library writer requires explicit valid record ACCESS configuration")
	}
	if transport == nil {
		base := http.DefaultTransport.(*http.Transport).Clone()
		base.Proxy = nil
		transport = base
	}
	signin, _ := url.JoinPath(cfg.DB.URL, "signin")
	rpc, _ := url.JoinPath(cfg.DB.URL, "rpc")
	login := &http.Client{Transport: transport, Timeout: FetchTimeout, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return errors.New("record signin redirect refused") }}
	auth := &accessTransport{cfg: cfg, base: transport, login: login, signin: signin, rpc: rpc, now: time.Now}
	client := &http.Client{Transport: auth, Timeout: FetchTimeout, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return errors.New("record query redirect refused") }}
	return SQLClient{Config: cfg.DB, HTTP: client}, nil
}

type accessTransport struct {
	cfg         AccessConfig
	base        http.RoundTripper
	login       *http.Client
	signin, rpc string
	mu          sync.Mutex
	token       string
	expires     time.Time
	now         func() time.Time
}

// RoundTrip replaces SQLClient's legacy Basic header only for its configured exact RPC endpoint.
// Inputs: bounded Query request; outputs: server response. Effects: scoped signin and query; never forwards passwords or tokens elsewhere.
func (a *accessTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method != http.MethodPost || req.URL.String() != a.rpc {
		return nil, errors.New("record access transport refused a different endpoint")
	}
	token, err := a.session(req.Context())
	if err != nil {
		return nil, err
	}
	copy := req.Clone(req.Context())
	copy.Header.Set("Authorization", "Bearer "+token)
	copy.Header.Del("surreal-auth-ns")
	copy.Header.Del("surreal-auth-db")
	prefix, err := bindLiteralRPC(copy)
	if err != nil {
		return nil, err
	}
	response, err := a.base.RoundTrip(copy)
	if err != nil || prefix == 0 || response.StatusCode != http.StatusOK {
		return response, err
	}
	return trimBindingResults(response, prefix)
}

var rpcVariableName = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{0,63}$`)

// bindLiteralRPC preserves bound strings against SurrealDB 3.2.4 JSON RPC's legacy record/date inference.
// Inputs: fixed Query JSON request; outputs: rewritten bounded request using encoding::json::decode on one opaque JSON string.
// Effects: local request replacement only; SQL remains fixed, variable names are validated, and values never become SQL text.
func bindLiteralRPC(req *http.Request) (int, error) {
	raw, err := io.ReadAll(io.LimitReader(req.Body, MaxArtifactBytes+1))
	_ = req.Body.Close()
	if err != nil || int64(len(raw)) > MaxArtifactBytes {
		return 0, errors.New("record query body exceeds budget")
	}
	var rpc struct {
		ID     json.RawMessage   `json:"id"`
		Method string            `json:"method"`
		Params []json.RawMessage `json:"params"`
	}
	if json.Unmarshal(raw, &rpc) != nil || rpc.Method != "query" || len(rpc.Params) != 2 {
		return 0, errors.New("record transport requires fixed Query contract")
	}
	var sql string
	var variables map[string]json.RawMessage
	if json.Unmarshal(rpc.Params[0], &sql) != nil || json.Unmarshal(rpc.Params[1], &variables) != nil {
		return 0, errors.New("record query variables invalid")
	}
	if len(variables) != 0 {
		keys := make([]string, 0, len(variables))
		for key := range variables {
			if !rpcVariableName.MatchString(key) || key == "toolkit_bound_json" {
				return 0, errors.New("record query variable name refused")
			}
			keys = append(keys, key)
		}
		sort.Strings(keys)
		var prefix strings.Builder
		for _, key := range keys {
			prefix.WriteString("LET $" + key + " = encoding::json::decode($toolkit_bound_json)." + key + ";\n")
		}
		sql = prefix.String() + sql
		packed, _ := json.Marshal(variables)
		raw, err = json.Marshal(map[string]any{"id": rpc.ID, "method": "query", "params": []any{sql, map[string]string{"toolkit_bound_json": string(packed)}}})
		if err != nil || int64(len(raw)) > MaxArtifactBytes {
			return 0, errors.New("record query encoded body exceeds budget")
		}
	}
	req.Body = io.NopCloser(bytes.NewReader(raw))
	req.ContentLength = int64(len(raw))
	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(raw)), nil }
	return len(variables), nil
}

// trimBindingResults removes successful local LET prefix results while retaining substantive server errors.
// Inputs: bounded RPC reply/prefix count; outputs: original SQLClient statement contract. Effects: local response replacement only.
func trimBindingResults(response *http.Response, prefix int) (*http.Response, error) {
	raw, err := io.ReadAll(io.LimitReader(response.Body, MaxArtifactBytes+1))
	_ = response.Body.Close()
	if err != nil || int64(len(raw)) > MaxArtifactBytes {
		return nil, errors.New("record query reply exceeds budget")
	}
	var rpc map[string]json.RawMessage
	var results []json.RawMessage
	if json.Unmarshal(raw, &rpc) != nil {
		return nil, errors.New("record query reply invalid")
	}
	if json.Unmarshal(rpc["result"], &results) == nil && len(results) >= prefix {
		prefixOK := true
		for _, result := range results[:prefix] {
			var entry struct {
				Status string `json:"status"`
			}
			if json.Unmarshal(result, &entry) != nil || entry.Status != "OK" {
				prefixOK = false
				break
			}
		}
		if prefixOK {
			rpc["result"], _ = json.Marshal(results[prefix:])
			raw, _ = json.Marshal(rpc)
		}
	}
	response.Body = io.NopCloser(bytes.NewReader(raw))
	response.ContentLength = int64(len(raw))
	response.Header.Del("Content-Length")
	return response, nil
}

// session obtains only a server-issued record token bound to the selected namespace/database/access/principal.
// Inputs: request context; outputs: private token. Effects: bounded signin; raw response/credentials never enter errors or logs.
func (a *accessTransport) session(ctx context.Context) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.token != "" && a.expires.After(a.now().Add(15*time.Second)) {
		return a.token, nil
	}
	body, _ := json.Marshal(map[string]string{"NS": a.cfg.DB.Namespace, "DB": a.cfg.DB.Database, "AC": a.cfg.Access, "username": a.cfg.DB.User, "password": a.cfg.DB.Password})
	if len(body) > 1024 {
		return "", errors.New("record signin exceeds body budget")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.signin, bytes.NewReader(body))
	if err != nil {
		return "", errors.New("record signin request invalid")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := a.login.Do(req)
	if err != nil {
		return "", errors.New("record signin transport failed")
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 16385))
	if err != nil || len(raw) > 16384 || resp.StatusCode != http.StatusOK {
		return "", errors.New("record signin refused or exceeded budget")
	}
	var reply struct {
		Code  int    `json:"code"`
		Token string `json:"token"`
	}
	if json.Unmarshal(raw, &reply) != nil || reply.Code != 200 || len(reply.Token) > 8192 {
		return "", errors.New("record signin returned no usable token")
	}
	parts := strings.Split(reply.Token, ".")
	if len(parts) != 3 || parts[0] == "" || parts[2] == "" {
		return "", errors.New("record token invalid")
	}
	claimsRaw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", errors.New("record token invalid")
	}
	var claims struct {
		Namespace string `json:"NS"`
		Database  string `json:"DB"`
		Access    string `json:"AC"`
		ID        string `json:"ID"`
		Expires   int64  `json:"exp"`
	}
	if json.Unmarshal(claimsRaw, &claims) != nil || claims.Namespace != a.cfg.DB.Namespace || claims.Database != a.cfg.DB.Database || claims.Access != a.cfg.Access || claims.ID != a.cfg.Principal || claims.Expires <= a.now().Add(15*time.Second).Unix() {
		return "", errors.New("record token does not bind required writer principal")
	}
	// The configured server issued this token over the admitted connection; it verifies the JWT signature on Query.
	a.token, a.expires = reply.Token, time.Unix(claims.Expires, 0)
	return a.token, nil
}

// NewScopedServiceFromEnv upgrades the existing service composition to mandatory validator record-access authentication.
// Inputs: existing versioned store, existing mounted config and TOOLKIT_VALIDATION_DB_ACCESS=toolkit_validator.
// Outputs: Service with scoped SQL writer. Effects: existing constructor's local preflight only; no DB writes or signin at boot.
// Choose over NewServiceFromEnv in parent worker composition; database EDITOR users bypass table permissions.
func NewScopedServiceFromEnv(store VersionStore) (*Service, error) {
	if strings.TrimSpace(os.Getenv(EnvDBAccess)) != ValidatorAccess {
		return nil, errors.New(EnvDBAccess + " must select toolkit_validator record access")
	}
	s, err := NewServiceFromEnv(store)
	if err != nil {
		return nil, err
	}
	repo, ok := s.Repository.(HTTPRepository)
	if !ok {
		return nil, errors.New("shared validation repository composition mismatch")
	}
	repo.DB, err = NewAccessSQLClient(AccessConfig{DB: repo.DB.Config, Access: ValidatorAccess, Principal: ValidatorPrincipal}, nil)
	if err != nil {
		return nil, err
	}
	s.Repository = repo
	return s, nil
}

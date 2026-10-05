// Byline: Codex · GPT-6.1 · 2026-10-04.
package librarysync

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const metadataBudget int64 = 2 << 20

func boundedHTTPClient() *http.Client {
	return &http.Client{Timeout: IOTimeout, Transport: &http.Transport{Proxy: nil, DialContext: (&net.Dialer{Timeout: 10 * time.Second}).DialContext, TLSHandshakeTimeout: 10 * time.Second, ResponseHeaderTimeout: 15 * time.Second, MaxConnsPerHost: 4, MaxIdleConnsPerHost: 4}, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("service redirects forbidden") }}
}

// HTTPBackend calls only the parent-owned protected sync routes with a server-held token.
// Inputs: fixed private HTTPS origin/token; outputs: Backend results. Effects: bounded HTTP; no DB credentials or MCP calls.
type HTTPBackend struct {
	origin string
	token  string
	client *http.Client
}

// NewHTTPBackend fixes the origin and refuses redirects, credentials in URLs and oversized tokens.
// Inputs: service origin (no path), dedicated token and optional trusted transport; outputs: backend or configuration error.
// Effects: none until called. The ordinary MCP bearer token must not be supplied as this dedicated service credential.
func NewHTTPBackend(origin, token string, transport http.RoundTripper) (*HTTPBackend, error) {
	u, err := url.Parse(origin)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || len(token) < 32 || len(token) > 4096 || strings.ContainsAny(token, "\r\n\x00") {
		return nil, errors.New("sync requires fixed private HTTPS origin and strong mounted service token")
	}
	c := boundedHTTPClient()
	if transport != nil {
		c.Transport = transport
	}
	return &HTTPBackend{origin: strings.TrimSuffix(origin, "/"), token: token, client: c}, nil
}

func (b *HTTPBackend) request(ctx context.Context, method, route string, value any, claim *Claim, max int64) ([]byte, error) {
	if b == nil || b.client == nil {
		return nil, errors.New("sync backend not configured")
	}
	var body []byte
	var err error
	if value != nil {
		body, err = json.Marshal(value)
		if err != nil || int64(len(body)) > metadataBudget {
			return nil, errors.New("sync request exceeds metadata budget")
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, b.origin+APIBase+route, bytes.NewReader(body))
	if err != nil {
		return nil, errors.New("sync request invalid")
	}
	req.Header.Set("Authorization", "Bearer "+b.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if claim != nil {
		req.Header.Set("X-Toolkit-Sync-Lease", claim.LeaseID)
	}
	resp, err := b.client.Do(req)
	if err != nil {
		return nil, errors.New("sync backend request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New("sync backend refused operation")
	}
	if resp.ContentLength > max {
		return nil, errors.New("sync response body budget exceeded")
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, max+1))
	if err != nil || int64(len(raw)) > max {
		return nil, errors.New("sync response incomplete or exceeded body budget")
	}
	return raw, nil
}

func (b *HTTPBackend) json(ctx context.Context, method, route string, in, out any) error {
	raw, err := b.request(ctx, method, route, in, nil, metadataBudget)
	if err != nil {
		return err
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(out) != nil {
		return errors.New("sync backend response does not match contract")
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return errors.New("sync backend response trailing payload")
	}
	return nil
}

func operationRoute(id, suffix string) (string, error) {
	if !uuidID.MatchString(id) {
		return "", errors.New("invalid sync operation ID")
	}
	return "/outbox/" + id + suffix, nil
}

// Claim calls POST /outbox/claim with operation_id/attempt_id; outputs: descriptor and binding lease; effects: backend transaction.
func (b *HTTPBackend) Claim(ctx context.Context, in OperationInput) (Claim, error) {
	var out Claim
	if !uuidID.MatchString(in.OperationID) || in.AttemptID == "" || len(in.AttemptID) > 200 {
		return out, errors.New("invalid sync operation attempt")
	}
	err := b.json(ctx, http.MethodPost, "/outbox/claim", in, &out)
	return out, err
}

// Payload calls GET /outbox/{uuid}/payload with X-Toolkit-Sync-Lease and service auth.
// Inputs: active lease; outputs: exact opaque bytes, never returned through Temporal. Effects: one bounded HTTP read.
func (b *HTTPBackend) Payload(ctx context.Context, c Claim) ([]byte, error) {
	r, err := operationRoute(c.Operation.OperationID, "/payload")
	if err != nil {
		return nil, err
	}
	if c.LeaseID == "" || c.Operation.PayloadSize <= 0 || c.Operation.PayloadSize > MaxPayloadBytes {
		return nil, errors.New("invalid leased payload descriptor")
	}
	raw, err := b.request(ctx, http.MethodGet, r, nil, &c, c.Operation.PayloadSize)
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) != c.Operation.PayloadSize || digest(raw) != c.Operation.PayloadSHA256 {
		return nil, errors.New("immutable outbox payload hash/size mismatch")
	}
	return raw, nil
}

// BeginWrite calls POST /outbox/{uuid}/write-intent with lease_id/fence/attempt_id.
// Outputs: one-shot permission and intent_id; effects: writing transition. Repeated/uncertain attempts may never issue a blind PUT.
func (b *HTTPBackend) BeginWrite(ctx context.Context, c Claim, attempt string) (Intent, error) {
	var out Intent
	r, err := operationRoute(c.Operation.OperationID, "/write-intent")
	if err != nil {
		return out, err
	}
	err = b.json(ctx, http.MethodPost, r, map[string]any{"lease_id": c.LeaseID, "fence": c.Fence, "attempt_id": attempt}, &out)
	if err == nil && out.MayWrite && out.IntentID == "" {
		err = errors.New("backend write intent missing identity")
	}
	return out, err
}

// Observe calls POST /observations with exact object/artifact references; outputs: pending/blocked IDs; effects: guarded import/proposal retention.
func (b *HTTPBackend) Observe(ctx context.Context, in Observation) (Outcome, error) {
	var out Outcome
	err := b.json(ctx, http.MethodPost, "/observations", in, &out)
	return out, err
}

// Seen calls GET /observations/{raw-hex-id}/status before source reads.
// Inputs: deterministic observation identity; outputs: seen only for a durable completed or explicitly blocked observation.
// Effects: backend metadata read; parent never marks a merely reserved/unprocessed version seen.
func (b *HTTPBackend) Seen(ctx context.Context, id string) (bool, error) {
	if !rawHash.MatchString(id) {
		return false, errors.New("invalid observation ID")
	}
	var out struct {
		Seen bool `json:"seen"`
	}
	err := b.json(ctx, http.MethodGet, "/observations/"+id+"/status", nil, &out)
	return out.Seen, err
}

// Complete calls POST /outbox/{uuid}/complete with reconciliation evidence and expected pointer revision.
// Outputs: transactional CAS outcome; effects: outbox/binding/conflict update. Parent rejects failed claims/currency when publishing separately.
func (b *HTTPBackend) Complete(ctx context.Context, c Claim, in Completion) (Outcome, error) {
	var out Outcome
	r, err := operationRoute(c.Operation.OperationID, "/complete")
	if err != nil {
		return out, err
	}
	err = b.json(ctx, http.MethodPost, r, in, &out)
	return out, err
}

// Failure persists explicit error state through POST /outbox/{uuid}/failure.
// Inputs: active lease/status/safe code; outputs: durable outcome. Effects: failure transaction, never a source overwrite.
func (b *HTTPBackend) Failure(ctx context.Context, c Claim, status, code string) (Outcome, error) {
	var out Outcome
	r, err := operationRoute(c.Operation.OperationID, "/failure")
	if err != nil {
		return out, err
	}
	err = b.json(ctx, http.MethodPost, r, map[string]any{"lease_id": c.LeaseID, "fence": c.Fence, "status": status, "error_code": code}, &out)
	return out, err
}

// Original resolves GET /bindings/{library_file-id}/original?version_id=<exact-version> under server auth.
// Inputs: stable binding and requested retained version; outputs: an authorized Binding whose original_pointer equals that version.
// Effects: metadata read. Parent authenticates the end user before proxying; no arbitrary bucket/key parameters are accepted.
func (b *HTTPBackend) Original(ctx context.Context, id, version string) (Binding, error) {
	var out Binding
	if !strings.HasPrefix(id, "library_file:") || !rawHash.MatchString(strings.TrimPrefix(id, "library_file:")) || !validVersion(version) {
		return out, errors.New("original requires stable binding and exact version")
	}
	err := b.json(ctx, http.MethodGet, "/bindings/"+url.PathEscape(id)+"/original?version_id="+url.QueryEscape(version), nil, &out)
	return out, err
}

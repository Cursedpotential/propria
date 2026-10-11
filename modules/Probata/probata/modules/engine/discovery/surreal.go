// Byline: Codex · GPT-6.1-sol · 2026-10-11
package discovery

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// ProjectionRecord is a source-linked selection row whose canonical identity and freshness are rechecked in PG.
type ProjectionRecord struct {
	RecordID        string `json:"record_id"`
	SourceVersionID string `json:"source_version_id"`
	GenerationID    string `json:"generation_id"`
	RecordRevision  string `json:"record_revision"`
	InputRevision   string `json:"input_revision"`
}

// SurrealReader reads only the server-configured source-linked projection; all persistence remains PG.
type SurrealReader interface {
	ReadSelection(context.Context, Dataset, RunRequest, int) ([]ProjectionRecord, error)
}

// SurrealConfig binds credentials and an allowlisted table per dataset outside caller request JSON.
type SurrealConfig struct {
	URL, Namespace, Database, User, Password, Token string
	Tables                                          map[string]string
	HTTPClient                                      *http.Client
}

// SurrealHTTPReader is the concrete bounded analytical input reader; it never creates tables or falls back to another store.
type SurrealHTTPReader struct {
	cfg  SurrealConfig
	http *http.Client
}

var identifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,99}$`)

// NewSurrealHTTPReader validates a managed endpoint, namespace/database and dataset table allowlist without network I/O.
// Inputs: server-only config. Outputs: reader. Effects: none; choose only for explicitly selected Surreal datasets.
func NewSurrealHTTPReader(cfg SurrealConfig) (*SurrealHTTPReader, error) {
	u, e := url.Parse(cfg.URL)
	if e != nil || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.Host == "" || u.RawQuery != "" || u.Fragment != "" || !identifier.MatchString(cfg.Namespace) || !identifier.MatchString(cfg.Database) || (cfg.Token == "" && (cfg.User == "" || cfg.Password == "")) || len(cfg.Tables) == 0 {
		return nil, ErrInvalid
	}
	for _, table := range cfg.Tables {
		if !identifier.MatchString(table) {
			return nil, ErrInvalid
		}
	}
	c := cfg.HTTPClient
	if c == nil {
		c = &http.Client{Timeout: 45 * time.Second}
	}
	return &SurrealHTTPReader{cfg, c}, nil
}

// ReadSelection selects canonical PG record pins from one allowlisted table and bounded server selection.
// Inputs: matched dataset/run and finite limit. Outputs: projection identity rows. Effects: one read-only SQL HTTP request.
// Choose for Surreal inputs; missing/stale/malformed projections fail visibly without PG fallback.
func (r *SurrealHTTPReader) ReadSelection(ctx context.Context, d Dataset, in RunRequest, limit int) ([]ProjectionRecord, error) {
	if r == nil || limit < 1 || limit > MaxRecords+1 {
		return nil, ErrInvalid
	}
	table := r.cfg.Tables[d.DatasetID]
	if !identifier.MatchString(table) || in.Scope != d.Scope || in.SelectionRevision != d.SelectionRevision {
		return nil, ErrScope
	}
	ids := make([]string, len(in.GenerationIDs))
	for i, id := range in.GenerationIDs {
		if !ID(id) {
			return nil, ErrInvalid
		}
		ids[i] = "'" + id + "'"
	}
	// All interpolated identifiers are server allowlisted; values are validated canonical UUIDs or JSON literals.
	revision, _ := json.Marshal(d.SelectionRevision)
	sql := "SELECT record_id, source_version_id, generation_id, record_revision, input_revision FROM " + table + " WHERE matter_id = '" + in.Scope.MatterID + "' AND court_case_id = '" + in.Scope.CourtCaseID + "' AND selection_revision = " + string(revision) + " AND generation_id IN [" + strings.Join(ids, ",") + "]" + surrealFilters(in.Filters) + " ORDER BY record_id LIMIT " + jsonNumber(limit) + ";"
	req, e := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(r.cfg.URL, "/")+"/sql", bytes.NewBufferString(sql))
	if e != nil {
		return nil, ErrInvalid
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("surreal-ns", r.cfg.Namespace)
	req.Header.Set("surreal-db", r.cfg.Database)
	if r.cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+r.cfg.Token)
	} else {
		req.SetBasicAuth(r.cfg.User, r.cfg.Password)
	}
	response, e := r.http.Do(req)
	if e != nil {
		return nil, ErrUnavailable
	}
	defer response.Body.Close()
	raw, e := io.ReadAll(io.LimitReader(response.Body, MaxBytes+1))
	if e != nil || len(raw) > MaxBytes || response.StatusCode != 200 {
		return nil, ErrUnavailable
	}
	var envelope []struct {
		Status string             `json:"status"`
		Result []ProjectionRecord `json:"result"`
	}
	if json.Unmarshal(raw, &envelope) != nil || len(envelope) != 1 || envelope[0].Status != "OK" || len(envelope[0].Result) > limit {
		return nil, ErrUnavailable
	}
	return envelope[0].Result, nil
}
func jsonNumber(n int) string { raw, _ := json.Marshal(n); return string(raw) }

func surrealFilters(f Filters) string {
	out := ""
	stamp := func(t *time.Time) string {
		raw, _ := json.Marshal(t.UTC().Format(time.RFC3339Nano))
		return "type::datetime(" + string(raw) + ")"
	}
	if f.StartAt != nil {
		out += " AND occurred_at >= " + stamp(f.StartAt)
	}
	if f.EndAt != nil {
		out += " AND occurred_at <= " + stamp(f.EndAt)
	}
	if f.Perspective == "as_lived" && f.Cutoff != nil {
		out += " AND occurred_at != NONE AND occurred_at != NULL AND occurred_at <= " + stamp(f.Cutoff) + " AND source_available_from != NONE AND source_available_from != NULL AND source_available_from <= " + stamp(f.Cutoff)
	}
	if len(f.Platforms) > 0 {
		raw, _ := json.Marshal(f.Platforms)
		out += " AND platform IN " + string(raw)
	}
	if len(f.Participants) > 0 {
		raw, _ := json.Marshal(f.Participants)
		out += " AND array::len(array::intersect(participants.identifier," + string(raw) + ")) > 0"
	}
	return out
}

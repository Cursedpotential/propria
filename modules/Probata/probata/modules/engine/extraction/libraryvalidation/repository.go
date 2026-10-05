// Byline: Codex · GPT-6.1 · 2026-10-04.
package libraryvalidation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Cursedpotential/probata/engine/surrealsink"
)

// RecordReader is the existing case_record read-only contract; inputs: exact record ID; outputs: shared fields/version.
type RecordReader interface {
	Read(context.Context, string) (Record, error)
}

// MCPRecords reads case_record over FCT's existing stateless HTTP transport using a server-mounted bearer token.
// Inputs: configured endpoint/token; outputs: unchanged shared records. Effects: read-only MCP calls, never receipt writes.
type MCPRecords struct {
	Endpoint, Token string
	HTTP            *http.Client
}

// Read invokes only case_record and unwraps the existing legal-record envelope without logging personal bodies.
// Inputs: exact shared record ID; outputs: record/version or missing/error. Effects: one bounded authenticated HTTP read.
func (m MCPRecords) Read(ctx context.Context, id string) (Record, error) {
	u, err := url.Parse(m.Endpoint)
	if err != nil || u.Host == "" || u.User != nil || (u.Scheme != "https" && u.Scheme != "http") || m.Token == "" || m.HTTP == nil {
		return Record{}, errors.New("shared FCT case_record reader is not configured")
	}
	if len(id) > 256 || !strings.Contains(id, ":") || strings.ContainsAny(id, "\r\n\x00") {
		return Record{}, errors.New("invalid shared record ID")
	}
	payload, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "case_record", "arguments": map[string]any{"id": id}}})
	ctx, cancel := context.WithTimeout(ctx, FetchTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.Endpoint, bytes.NewReader(payload))
	if err != nil {
		return Record{}, errors.New("FCT request construction failed")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", "2025-06-18")
	req.Header.Set("Authorization", "Bearer "+m.Token)
	resp, err := m.HTTP.Do(req)
	if err != nil {
		return Record{}, errors.New("FCT case_record transport failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Record{}, fmt.Errorf("FCT case_record HTTP %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, MaxArtifactBytes+1))
	if err != nil || int64(len(raw)) > MaxArtifactBytes {
		return Record{}, errors.New("FCT case_record response exceeds budget")
	}
	if strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		var message []byte
		for _, line := range bytes.Split(raw, []byte("\n")) {
			if bytes.HasPrefix(line, []byte("data:")) {
				data := bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
				var probe map[string]json.RawMessage
				if json.Unmarshal(data, &probe) == nil && probe["id"] != nil {
					if message != nil {
						return Record{}, errors.New("ambiguous FCT response")
					}
					message = data
				}
			}
		}
		raw = message
	}
	var rpc struct {
		Error  json.RawMessage `json:"error"`
		Result struct {
			IsError    bool            `json:"isError"`
			Structured json.RawMessage `json:"structuredContent"`
			Content    []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	if json.Unmarshal(raw, &rpc) != nil || (len(rpc.Error) > 0 && string(rpc.Error) != "null") || rpc.Result.IsError {
		return Record{}, errors.New("FCT case_record returned an error")
	}
	body := rpc.Result.Structured
	if len(body) == 0 {
		for _, content := range rpc.Result.Content {
			if content.Type == "text" {
				body = []byte(content.Text)
				break
			}
		}
	}
	var envelope struct {
		Available bool `json:"available"`
		Found     bool `json:"found"`
		Record
	}
	if json.Unmarshal(body, &envelope) != nil || !envelope.Available {
		return Record{}, errors.New("FCT shared store unavailable or malformed")
	}
	if !envelope.Found {
		return Record{}, nil
	}
	if envelope.ID != id || !digestPattern.MatchString(envelope.Version) || envelope.Fields == nil {
		return Record{}, errors.New("FCT record identity/version mismatch")
	}
	return envelope.Record, nil
}

// SQLClient commits using the shared DB's server credentials and bound RPC variables.
// Inputs: existing surreal-case configuration; outputs: statement results. Effects: explicit server-only DB queries.
type SQLClient struct {
	Config surrealsink.Config
	HTTP   *http.Client
}

// Query returns the substantive per-statement error rather than a transaction's trailing NotExecuted marker.
// Inputs: fixed SQL and bound variables; outputs: successful statement JSON. Effects: one bounded authenticated RPC query.
func (c SQLClient) Query(ctx context.Context, sql string, vars map[string]any) ([]json.RawMessage, error) {
	if err := c.Config.Validate(); err != nil || c.HTTP == nil {
		return nil, errors.New("trusted validation DB client is not configured")
	}
	payload, err := json.Marshal(map[string]any{"id": 1, "method": "query", "params": []any{sql, vars}})
	if err != nil || int64(len(payload)) > MaxArtifactBytes {
		return nil, errors.New("validation DB request exceeds budget")
	}
	endpoint, err := url.JoinPath(c.Config.URL, "rpc")
	if err != nil {
		return nil, errors.New("invalid validation DB endpoint")
	}
	ctx, cancel := context.WithTimeout(ctx, FetchTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, errors.New("validation DB request failed")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("surreal-ns", c.Config.Namespace)
	req.Header.Set("surreal-db", c.Config.Database)
	if c.Config.AuthLevel == "database" {
		req.Header.Set("surreal-auth-ns", c.Config.Namespace)
		req.Header.Set("surreal-auth-db", c.Config.Database)
	}
	req.SetBasicAuth(c.Config.User, c.Config.Password)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, errors.New("validation DB transport failed")
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, MaxArtifactBytes+1))
	if err != nil || int64(len(raw)) > MaxArtifactBytes {
		return nil, errors.New("validation DB reply exceeds budget")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("validation DB HTTP %d", resp.StatusCode)
	}
	var reply struct {
		Error  json.RawMessage `json:"error"`
		Result []struct {
			Status string          `json:"status"`
			Result json.RawMessage `json:"result"`
		} `json:"result"`
	}
	if json.Unmarshal(raw, &reply) != nil || (len(reply.Error) > 0 && string(reply.Error) != "null") || len(reply.Result) == 0 {
		return nil, errors.New("validation DB RPC failed")
	}
	var first error
	for i, statement := range reply.Result {
		if statement.Status != "OK" {
			var message string
			_ = json.Unmarshal(statement.Result, &message)
			lower := strings.ToLower(message)
			failure := fmt.Errorf("validation DB statement %d: %s", i, safeSQLError(message))
			if first == nil {
				first = failure
			}
			if !strings.Contains(lower, "not executed") && !strings.Contains(lower, "notexecuted") && !strings.Contains(lower, "failed transaction") && !strings.Contains(lower, "cancelled transaction") {
				return nil, failure
			}
		}
	}
	if first != nil {
		return nil, first
	}
	out := make([]json.RawMessage, len(reply.Result))
	for i, statement := range reply.Result {
		out[i] = statement.Result
	}
	return out, nil
}

// safeSQLError exposes controlled gate failures without arbitrary DB-returned data; inputs: error text; outputs: diagnostic.
func safeSQLError(message string) string {
	for _, known := range []string{"Library proposal missing", "Library proposal changed", "Library proposal stale", "Library source missing", "Library source changed", "Library validation receipt conflict", "Library validation evidence incomplete", "Library currency review changed", "Library validation expired"} {
		if strings.Contains(message, known) {
			return known
		}
	}
	return "database statement failed"
}

// HTTPRepository combines existing shared reads with trusted receipt commits; no ordinary MCP tool can write validations.
// Inputs: reader, DB client and server signing key; outputs: proposal/source/receipt identities. Effects: explicitly bounded I/O.
type HTTPRepository struct {
	Reader     RecordReader
	DB         SQLClient
	SigningKey []byte
}

// Record reads shared source/currency records; inputs: ID; outputs: unchanged record/version; effects: existing case_record call.
func (r HTTPRepository) Record(ctx context.Context, id string) (Record, error) {
	if r.Reader == nil {
		return Record{}, errors.New("case_record reader missing")
	}
	return r.Reader.Read(ctx, id)
}

// Proposal loads the full proposed record without redaction and validates only its bounded contract.
// Inputs: proposal ID; outputs: in-memory projection. Effects: shared read only; no personal bodies enter history.
func (r HTTPRepository) Proposal(ctx context.Context, id string) (Proposal, error) {
	if !proposalPattern.MatchString(id) {
		return Proposal{}, errors.New("invalid library proposal ID")
	}
	var record Record
	var immutableVersion string
	// Match the case_record read to a single server observation; dispatch may race either read.
	for attempt := 0; attempt < 3; attempt++ {
		var err error
		record, err = r.Record(ctx, id)
		if err != nil {
			return Proposal{}, err
		}
		if record.ID == "" {
			return Proposal{}, errors.New("library proposal missing")
		}
		results, err := r.DB.Query(ctx, proposalVersionSQL, map[string]any{"key": proposalPattern.FindStringSubmatch(id)[1]})
		if err != nil {
			return Proposal{}, err
		}
		var observed struct {
			Full      string `json:"full"`
			Immutable string `json:"immutable"`
		}
		for _, result := range results {
			_ = json.Unmarshal(result, &observed)
		}
		if !digestPattern.MatchString(observed.Full) || !digestPattern.MatchString(observed.Immutable) {
			return Proposal{}, errors.New("proposal version readback invalid")
		}
		if observed.Full == record.Version {
			immutableVersion = observed.Immutable
			break
		}
	}
	if immutableVersion == "" {
		return Proposal{}, errors.New("proposal changed during bounded preparation reads; retry validation")
	}
	if record.ID == "" {
		return Proposal{}, errors.New("library proposal missing")
	}
	raw, _ := json.Marshal(record.Fields)
	var p Proposal
	if json.Unmarshal(raw, &p) != nil {
		return Proposal{}, errors.New("malformed library proposal")
	}
	p.ID = id
	p.Version = immutableVersion
	if err := ValidateProposal(p); err != nil {
		return Proposal{}, err
	}
	return p, nil
}

// proposalVersionSQL observes full and immutable hashes from one shared record without changing its contents.
// Inputs: bound proposal UUID; outputs: case_record-compatible full hash and stable validation hash. Effects: read only.
// Excludes only parent-owned dispatch/dispatch_at; all substantive fields and status remain in the trust gate.
const proposalVersionSQL = `
LET $p = (SELECT * OMIT embedding FROM ONLY type::record('library_proposal', $key));
IF $p = NONE { THROW 'Library proposal missing'; };
RETURN { full: 'sha256:' + crypto::sha256(<string> $p),
 immutable: 'sha256:' + crypto::sha256(<string> object::remove($p, ['dispatch', 'dispatch_at'])) };
`

// Commit checks every shared version atomically, writes datetime/record types, and confirms the exact signed receipt.
// Inputs: complete authenticated receipt; outputs: validation record ID. Effects: only library_validation UPSERT, never publication.
func (r HTTPRepository) Commit(ctx context.Context, receipt Receipt) (string, error) {
	if !proposalPattern.MatchString(receipt.ProposalID) {
		return "", errors.New("invalid receipt proposal ID")
	}
	sig := receipt.Signature
	unsigned := receipt
	unsigned.Signature = ""
	if !matchesSignature(unsigned, sig, r.SigningKey) {
		return "", errors.New("receipt signature mismatch")
	}
	raw, err := json.Marshal(receipt)
	if err != nil {
		return "", err
	}
	var data map[string]any
	if json.Unmarshal(raw, &data) != nil {
		return "", errors.New("receipt encoding failed")
	}
	key := proposalPattern.FindStringSubmatch(receipt.ProposalID)[1]
	results, err := r.DB.Query(ctx, commitSQL, map[string]any{"key": key, "receipt": data})
	if err != nil {
		return "", err
	}
	want := "library_validation:" + key
	for _, raw := range results {
		var result struct {
			ID        string `json:"id"`
			Signature string `json:"signature"`
		}
		if json.Unmarshal(raw, &result) == nil && result.ID == want && result.Signature == sig {
			return want, nil
		}
	}
	return "", errors.New("trusted receipt commit readback mismatch")
}

// commitSQL preserves proposed personal data and rechecks source, target, proposal and currency evidence within the write transaction.
// Inputs: bound receipt/key; outputs: receipt identity/signature. Effects: one validation row only, retry-safe for identical content.
const commitSQL = `
BEGIN TRANSACTION;
LET $p = (SELECT * OMIT embedding FROM ONLY type::record('library_proposal', $key));
IF $p = NONE { THROW 'Library proposal missing'; };
IF 'sha256:' + crypto::sha256(<string> object::remove($p, ['dispatch', 'dispatch_at'])) != $receipt.proposal_version
 OR $p.proposed_hash != $receipt.proposed_hash
 OR 'sha256:' + crypto::sha256(<string> $p.proposed_record) != $receipt.proposed_hash
 OR $p.citations != $receipt.claims { THROW 'Library proposal changed'; };
IF $p.status != 'pending_validation' { THROW 'Library proposal stale'; };
LET $target = (SELECT * OMIT embedding FROM ONLY $p.target);
LET $target_version = IF $target = NONE { 'absent' } ELSE { 'sha256:' + crypto::sha256(<string> $target) };
IF $target_version != $p.expected_version { THROW 'Library proposal stale'; };
IF array::len($receipt.claim_checks) != array::len($p.citations) { THROW 'Library validation evidence incomplete'; };
FOR $c IN $p.citations {
 LET $source = (SELECT * OMIT embedding FROM ONLY type::record('source', string::split($c.source_id, ':')[1]));
 LET $self = $c.source_id = record::tb($p.target) + ':' + <string> record::id($p.target)
  AND $c.source_version = 'absent' AND $p.expected_version = 'absent' AND record::tb($p.target) = 'source';
 IF !$self AND $source = NONE { THROW 'Library source missing'; };
 IF $self AND $source != NONE { THROW 'Library source changed'; };
 IF !$self AND 'sha256:' + crypto::sha256(<string> $source) != $c.source_version { THROW 'Library source changed'; };
};
FOR $check IN $receipt.claim_checks {
 IF $check.currency_status = 'cleared' {
  LET $currency = (SELECT * FROM ONLY type::record('library_currency', string::split($check.source_id, ':')[1]));
  IF $currency = NONE OR $currency.signature != $check.currency_evidence.signature { THROW 'Library currency review changed'; };
 };
};
LET $completed = <datetime> $receipt.completed_at;
LET $expires = <datetime> $receipt.expires_at;
IF $completed > time::now() OR $expires <= time::now() { THROW 'Library validation expired'; };
LET $typed_checks = array::map($receipt.claim_checks, |$check| object::extend($check, { evidence_time: <datetime> $check.evidence_time }));
LET $typed = object::extend($receipt, {
 proposal_id: $p.id, completed_at: $completed, expires_at: $expires, claim_checks: $typed_checks
});
LET $old = (SELECT * FROM ONLY type::record('library_validation', $key));
IF $old != NONE AND $old.signature != $receipt.signature
 AND ($old.status = 'VERIFIED_PRIMARY' OR $old.signature != $receipt.previous_signature
  OR $old.attempt_id = $receipt.attempt_id OR $receipt.attempt_id = NONE OR $receipt.attempt_id = '') { THROW 'Library validation receipt conflict'; };
UPSERT type::record('library_validation', $key) CONTENT $typed;
LET $saved = (SELECT * FROM ONLY type::record('library_validation', $key));
-- Build the canonical ID from its bound key; provider record-string quoting is not an API identity.
RETURN { id: 'library_validation:' + $key, signature: $saved.signature };
COMMIT TRANSACTION;
`

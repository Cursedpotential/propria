// Package libraryvalidation validates shared library proposals against retained official source snapshots.
// Inputs are shared proposal IDs and immutable references; outputs are signed, bounded receipts.
// Side effects are explicit HTTP reads, derivative B2 writes and trusted receipt commits; publication belongs to FCT.
// Choose this lane for every library revision rather than treating link availability or model agreement as validation.
// Byline: Codex · GPT-6.1 · 2026-10-04.
package libraryvalidation

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"time"
)

const (
	ValidatorVersion       = "library-validation/1.1.0"
	CheckVersion           = "primary-quote-pin/1.0.0"
	Verified               = "VERIFIED_PRIMARY"
	Blocked                = "BLOCKED"
	Conflicted             = "CONFLICTED"
	Stale                  = "STALE"
	Partial                = "PARTIAL"
	Fetched                = "FETCHED"
	Cleared                = "cleared"
	Provisional            = "provisional"
	MaxClaims              = 64
	MaxSourceBytes   int64 = 8 << 20
	MaxArtifactBytes int64 = 2 << 20
	MaxTextBytes           = 1 << 20
	MaxPassageBytes        = 48000
	MaxPages               = 512
	FetchTimeout           = 30 * time.Second
	ExtractTimeout         = 90 * time.Second
	ModelTimeout           = 180 * time.Second
	ReceiptLifetime        = 24 * time.Hour
)

var proposalPattern = regexp.MustCompile(`^library_proposal:([a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12})$`)
var sourcePattern = regexp.MustCompile(`^source:[^\s:]{1,200}$`)
var digestPattern = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)
var ErrStale = errors.New("library validation proposal or source changed")

// Citation preserves the proposal's exact claim and source coordinates, including personal text.
// Inputs: stored proposal citations; outputs: unchanged citation fields. Effects: none; choose over history payloads.
type Citation struct {
	SourceID      string `json:"source_id"`
	SourceVersion string `json:"source_version"`
	Pinpoint      string `json:"pinpoint"`
	Claim         string `json:"claim"`
}

// Proposal is an in-Activity projection of the shared pending revision; it never enters workflow history.
// Inputs: case_record result; outputs: version/hash, citations and full proposed record. Effects: none.
type Proposal struct {
	ID              string                     `json:"id"`
	Version         string                     `json:"version"`
	Target          string                     `json:"target"`
	ExpectedVersion string                     `json:"expected_version"`
	ProposedHash    string                     `json:"proposed_hash"`
	Status          string                     `json:"status"`
	Citations       []Citation                 `json:"citations"`
	ProposedRecord  map[string]json.RawMessage `json:"proposed_record"`
}

// Record carries case_record's server-computed version and unchanged shared fields.
// Inputs: HTTP read; outputs: record identity and fields. Effects: none; choose instead of recomputing Surreal hashes in Go.
type Record struct {
	ID      string                     `json:"id"`
	Version string                     `json:"version"`
	Fields  map[string]json.RawMessage `json:"record"`
}

// ArtifactRef pins a retained derivative to both its provider version and its full SHA-256.
// Inputs: B2 write/readback; outputs: compact reference only. Effects: none; choose for every Temporal boundary.
type ArtifactRef struct {
	URI       string `json:"uri"`
	VersionID string `json:"version_id"`
	SHA256    string `json:"sha256"`
	Bytes     int64  `json:"bytes"`
}

// Artifacts writes derivatives and reads exact versions; implementations must verify bytes before returning.
// Inputs: bounded key/body or pinned reference; outputs: references/bytes. Effects: object storage only.
type Artifacts interface {
	Put(context.Context, string, []byte, string) (ArtifactRef, error)
	Read(context.Context, ArtifactRef, int64) ([]byte, error)
}

// Repository separates existing case_record reads from privileged validation receipt writes.
// Inputs: identities/receipt; outputs: shared records or acknowledged receipt ID. Effects: documented per method.
type Repository interface {
	Proposal(context.Context, string) (Proposal, error)
	Record(context.Context, string) (Record, error)
	Commit(context.Context, Receipt) (string, error)
}

// CurrencyEvidence is an independently signed current-release attestation, not a model verdict or editable source flag.
// Inputs: trusted reviewer record library_currency:<source-key>; outputs: exact source/snapshot/release bindings.
// Effects: none. Choose when an authorized reviewer has checked current amendments/releases; absence blocks publication.
type CurrencyEvidence struct {
	SourceID           string      `json:"source_id"`
	SourceVersion      string      `json:"source_version"`
	SnapshotSHA256     string      `json:"snapshot_sha256"`
	PrimaryURL         string      `json:"primary_url"`
	ReleaseURL         string      `json:"release_url"`
	ReleaseVersion     string      `json:"release_version"`
	ReleaseSHA256      string      `json:"release_sha256"`
	ReviewID           string      `json:"review_id"`
	ReviewerID         string      `json:"reviewer_id"`
	DecisionID         string      `json:"decision_id"`
	ReviewedThrough    time.Time   `json:"reviewed_through"`
	ApprovalRef        ArtifactRef `json:"approval_ref"`
	SourceSnapshotRef  ArtifactRef `json:"source_snapshot_ref"`
	ReleaseSnapshotRef ArtifactRef `json:"release_snapshot_ref"`
	ReleasePinpoint    string      `json:"release_pinpoint"`
	ReleaseQuoteSHA256 string      `json:"release_quote_sha256"`
	EffectiveAt        time.Time   `json:"effective_at"`
	CheckedAt          time.Time   `json:"checked_at"`
	ExpiresAt          time.Time   `json:"expires_at"`
	Status             string      `json:"status"`
	Signature          string      `json:"signature"`
}

// Snapshot records one fetch result and the exact proposal/source state used to obtain it.
// Inputs: source fetch; outputs: a signed artifact outside history. Effects: none in this type.
type Snapshot struct {
	ProposalID      string            `json:"proposal_id"`
	ProposedHash    string            `json:"proposed_hash"`
	ProposalVersion string            `json:"proposal_version"`
	Index           int               `json:"index"`
	Citation        Citation          `json:"citation"`
	PrimaryURL      string            `json:"primary_url"`
	FinalURL        string            `json:"final_url"`
	MediaType       string            `json:"media_type"`
	Raw             ArtifactRef       `json:"raw"`
	FetchedAt       time.Time         `json:"fetched_at"`
	ETag            string            `json:"etag,omitempty"`
	LastModified    string            `json:"last_modified,omitempty"`
	Status          string            `json:"status"`
	FailureCode     string            `json:"failure_code,omitempty"`
	Currency        *CurrencyEvidence `json:"currency,omitempty"`
}

// ClaimCheck retains each deterministic quote check, semantic verdict and independently checked currency state.
// Inputs: one citation and snapshot; outputs: trusted ordered receipt evidence. Effects: none; never return through history.
type ClaimCheck struct {
	Citation
	ProposalID        string            `json:"proposal_id"`
	ProposedHash      string            `json:"proposed_hash"`
	ProposalVersion   string            `json:"proposal_version"`
	Index             int               `json:"index"`
	Status            string            `json:"status"`
	CurrencyStatus    string            `json:"currency_status"`
	SnapshotRef       string            `json:"snapshot_ref"`
	SnapshotSHA256    string            `json:"snapshot_sha256"`
	SnapshotVersionID string            `json:"snapshot_version_id"`
	VersionID         string            `json:"version_id"`
	QuoteSHA256       string            `json:"quote_sha256"`
	Quote             string            `json:"quote,omitempty"`
	EvidenceTime      time.Time         `json:"evidence_time"`
	PrimaryURL        string            `json:"primary_url"`
	FinalURL          string            `json:"final_url"`
	Model             string            `json:"model"`
	CheckVersion      string            `json:"check_version"`
	Extractor         string            `json:"extractor,omitempty"`
	ExtractorVersion  string            `json:"extractor_version,omitempty"`
	FailureCode       string            `json:"failure_code,omitempty"`
	Currency          *CurrencyEvidence `json:"currency_evidence,omitempty"`
}

// Receipt binds every citation/check and snapshot version to one exact proposal and expiry.
// Inputs: complete signed check artifacts; outputs: trusted server-side validation row. Effects: none in this type.
type Receipt struct {
	ProposalID        string       `json:"proposal_id"`
	ProposalVersion   string       `json:"proposal_version"`
	AttemptID         string       `json:"attempt_id,omitempty"`
	PreviousSignature string       `json:"previous_signature,omitempty"`
	ProposedHash      string       `json:"proposed_hash"`
	Status            string       `json:"status"`
	CurrencyStatus    string       `json:"currency_status"`
	ValidatorVersion  string       `json:"validator_version"`
	CompletedAt       time.Time    `json:"completed_at"`
	ExpiresAt         time.Time    `json:"expires_at"`
	Claims            []Citation   `json:"claims"`
	ClaimChecks       []ClaimCheck `json:"claim_checks"`
	Signature         string       `json:"signature"`
}

// Hash names a full byte digest; inputs: exact bytes; outputs: sha256-prefixed hash; effects: none.
func Hash(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// signJSON authenticates exact JSON with a server-held key; inputs: object/key; outputs: HMAC; effects: none.
func signJSON(value any, key []byte) (string, error) {
	if len(key) < 32 {
		return "", errors.New("validation signing key must have at least 32 bytes")
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(raw)
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// matchesSignature compares an independently produced HMAC without exposing keys; inputs: signed value; outputs: validity.
func matchesSignature(value any, signature string, key []byte) bool {
	expected, err := signJSON(value, key)
	return err == nil && hmac.Equal([]byte(expected), []byte(signature))
}

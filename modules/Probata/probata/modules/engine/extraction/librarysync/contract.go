// Package librarysync observes exact B2 legal-file versions and exports guarded app revisions through a durable outbox.
// Inputs: configured legal scope, saved operation IDs and pinned references; outputs: retained evidence and explicit CAS outcomes.
// Effects: bounded B2/HTTP operations only. Choose alongside libraryvalidation; this package never clears claims or currency.
// Byline: Codex · GPT-6.1 · 2026-10-04.
package librarysync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/extraction/libraryvalidation"
)

const (
	ContractVersion        = "toolkit-library-sync/1"
	APIBase                = "/api/internal/library-sync"
	MaxPage                = 200
	MaxCyclePages          = 10
	MaxHistoryPages        = 5
	MaxOriginalBytes int64 = 20 << 20
	MaxPayloadBytes  int64 = 8 << 20
	IOTimeout              = 90 * time.Second
	Pending                = "pending"
	Synced                 = "synced"
	Blocked                = "blocked"
	Conflicted             = "conflicted"
	WriteUnknown           = "write_unknown"
	RetryWait              = "retry_wait"
	CitationRequired       = "citation_required"
)

var rawHash = regexp.MustCompile(`^[a-f0-9]{64}$`)
var uuidID = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)

// Scope bounds every provider operation to one configured current B2 legal collection.
// Inputs: account identity, bucket and physical legal root ending KnowledgeBase/legal/; outputs: admission checks.
// Effects: none. Configure the actual prefix explicitly; never search historical copies as a fallback.
type Scope struct {
	AccountScope string `json:"account_scope"`
	Bucket       string `json:"bucket"`
	LegalRoot    string `json:"legal_root"`
}

// Validate rejects missing identity and ambiguous prefixes; inputs: scope; outputs: error; effects: none.
func (s Scope) Validate() error {
	if s.AccountScope == "" || len(s.AccountScope) > 200 || strings.ContainsAny(s.AccountScope, "\r\n\x00") || s.Bucket == "" || strings.ContainsAny(s.Bucket, "/\\?#@:\r\n\x00") || !safeKey(strings.TrimSuffix(s.LegalRoot, "/")) || !strings.HasSuffix(s.LegalRoot, "KnowledgeBase/legal/") {
		return errors.New("library sync requires explicit B2 account/bucket/legal root")
	}
	return nil
}

// Admit restricts exact keys to the three owner-approved children; inputs: bucket/key; outputs: error; effects: none.
func (s Scope) Admit(bucket, key string) error {
	if err := s.Validate(); err != nil {
		return err
	}
	if bucket != s.Bucket || !safeKey(key) {
		return errors.New("object outside configured B2 legal scope")
	}
	for _, child := range []string{"case-law/", "benchbooks/", "reference-data/"} {
		if strings.HasPrefix(key, s.LegalRoot+child) && len(key) > len(s.LegalRoot+child) {
			return nil
		}
	}
	return errors.New("object outside configured B2 legal scope")
}

// BindingID derives a stable record identity independent of object version; inputs: admitted key; outputs: library_file ID.
// Effects: none. Use this mapping rather than creating a new source identity on every B2 upload.
func (s Scope) BindingID(key string) (string, error) {
	if err := s.Admit(s.Bucket, key); err != nil {
		return "", err
	}
	b, _ := json.Marshal([]string{"b2", s.AccountScope, s.Bucket, key})
	return "library_file:" + digest(b), nil
}

func safeKey(k string) bool {
	return k != "" && len(k) <= 1024 && !strings.HasPrefix(k, "/") && path.Clean(k) == k && !strings.ContainsAny(k, "\\\r\n\x00") && !strings.Contains(k, "//") && !strings.HasSuffix(k, "/")
}
func validVersion(v string) bool {
	return v != "" && v != "null" && len(v) <= 2048 && !strings.ContainsAny(v, "\r\n\x00")
}
func digest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

// Pointer identifies complete bytes at one provider-retained version, never an ETag or latest-object alias.
// Inputs: exact GET/hash evidence; outputs: stored original/accepted pointer. Effects: none; hashes are raw lower hex.
type Pointer struct {
	VersionID   string    `json:"version_id"`
	SHA256      string    `json:"sha256"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type"`
	ObservedAt  time.Time `json:"observed_at"`
}

func (p Pointer) validate() error {
	if !validVersion(p.VersionID) || !rawHash.MatchString(p.SHA256) || p.Size < 0 || p.Size > MaxOriginalBytes || p.ContentType == "" || p.ObservedAt.IsZero() {
		return errors.New("invalid pinned object pointer")
	}
	return nil
}

// Object is metadata for one upload or hide marker, with optional complete-byte evidence.
// Inputs: exact version listing/GET; outputs: bounded observation coordinates. Effects: none; never contains source text.
type Object struct {
	Bucket      string    `json:"bucket"`
	Key         string    `json:"key"`
	VersionID   string    `json:"version_id"`
	Size        int64     `json:"size"`
	Latest      bool      `json:"latest"`
	Hidden      bool      `json:"hidden"`
	UploadedAt  time.Time `json:"uploaded_at"`
	SHA256      string    `json:"sha256,omitempty"`
	ContentType string    `json:"content_type,omitempty"`
	OperationID string    `json:"operation_id,omitempty"`
	IntentID    string    `json:"intent_id,omitempty"`
}

// Cursor preserves both S3 version-list continuation coordinates; inputs/outputs: metadata only; effects: none.
type Cursor struct {
	Key       string `json:"key"`
	VersionID string `json:"version_id"`
}

// ListInput requests one bounded legal-prefix page; inputs: child and cursor; outputs: Page; effects: listing only.
type ListInput struct {
	Child  string `json:"child"`
	Cursor Cursor `json:"cursor"`
}

// Page retains explicit continuation and hide markers; inputs: provider page; outputs: references only; effects: none.
type Page struct {
	Objects  []Object `json:"objects"`
	Next     Cursor   `json:"next"`
	Complete bool     `json:"complete"`
}

// Binding stores stable source mapping and independent original/export pointers.
// Inputs: backend row; outputs: admitted original read coordinates. Effects: none. App-owned pointer_revision is CAS protected.
type Binding struct {
	ID       string `json:"id"`
	Provider string `json:"provider"`
	Scope
	Key                 string   `json:"key"`
	RecordID            string   `json:"record_id"`
	RecordVersion       string   `json:"record_version"`
	PointerRevision     string   `json:"pointer_revision"`
	Format              string   `json:"format"`
	AcceptedPointer     *Pointer `json:"accepted_pointer"`
	OriginalPointer     *Pointer `json:"original_pointer"`
	RecordExportKey     string   `json:"record_export_key,omitempty"`
	RecordExportPointer *Pointer `json:"record_export_pointer"`
}

// Operation is the immutable full-payload outbox descriptor, captured in the app edit/publication transaction.
// Inputs: backend claim; outputs: export coordinates. Effects: none. Store exact payload_utf8 separately or in the same row.
// Parent schema: library_sync_outbox:<operation_id>; immutable fields below, payload_utf8/payload_ref and created_at.
// Mutable fields are status, lease_id/fence/lease_expires_at, attempts, retry_at, written_pointer and safe error_code.
// Preserve typed values through the declared codec; record_version is not the encoded payload SHA-256.
type Operation struct {
	Contract            string   `json:"contract_version"`
	OperationID         string   `json:"operation_id"`
	BindingID           string   `json:"binding_id"`
	RecordID            string   `json:"record_id"`
	RecordVersion       string   `json:"record_version"`
	RevisionRef         string   `json:"revision_ref"`
	Bucket              string   `json:"bucket"`
	Key                 string   `json:"key"`
	PayloadSHA256       string   `json:"payload_sha256"`
	PayloadSize         int64    `json:"payload_size"`
	ContentType         string   `json:"content_type"`
	CodecVersion        string   `json:"codec_version"`
	BasePointerRevision string   `json:"base_pointer_revision"`
	BaseB2Pointer       *Pointer `json:"base_b2_pointer"`
}

func (o Operation) validate(s Scope) error {
	if err := s.Admit(o.Bucket, o.Key); err != nil {
		return err
	}
	if o.Contract != ContractVersion || !uuidID.MatchString(o.OperationID) || !strings.HasPrefix(o.BindingID, "library_file:") || !rawHash.MatchString(strings.TrimPrefix(o.BindingID, "library_file:")) || o.RecordID == "" || o.RecordVersion == "" || o.RevisionRef == "" || o.CodecVersion == "" || o.BasePointerRevision == "" || !rawHash.MatchString(o.PayloadSHA256) || o.PayloadSize <= 0 || o.PayloadSize > MaxPayloadBytes || o.ContentType == "" {
		return errors.New("outbox immutable descriptor invalid")
	}
	if o.BaseB2Pointer != nil {
		return o.BaseB2Pointer.validate()
	}
	return nil
}

// Claim leases one binding's operation through private service auth; inputs: operation/attempt; outputs: immutable descriptor.
// Effects: backend transaction only. Fence must change on reassignment; completion checks record and binding versions too.
type Claim struct {
	Operation     Operation `json:"operation"`
	LeaseID       string    `json:"lease_id"`
	Fence         int64     `json:"fence"`
	ExpiresAt     time.Time `json:"lease_expires_at"`
	Status        string    `json:"status"`
	WriteIntentID string    `json:"write_intent_id"`
}

// OperationInput carries saved identity and attempt through Temporal, never personal payloads.
type OperationInput struct {
	OperationID string `json:"operation_id"`
	AttemptID   string `json:"attempt_id"`
}

// Handle identifies an authenticated immutable derivative descriptor outside workflow history.
type Handle struct {
	OperationID string                        `json:"operation_id,omitempty"`
	Ref         libraryvalidation.ArtifactRef `json:"ref"`
	Status      string                        `json:"status"`
	Code        string                        `json:"error_code,omitempty"`
}

// Intent grants at most one backend-authorized PUT for an attempt; inputs: leased operation; outputs: permission/ref.
// Effects: backend writing transition. A repeated/lost-response request returns may_write:false, requiring reconciliation.
type Intent struct {
	MayWrite bool   `json:"may_write"`
	IntentID string `json:"intent_id"`
}

// Observation stages one exact object and extractor-bound artifact without making legal verification claims.
// Inputs: hash/retention results; outputs: backend IDs/status. Effects: existing proposal/import gates only.
// Parent persists library_sync_observation:<observation_id>; preserve every version, including hide events.
type Observation struct {
	Contract      string                         `json:"contract_version"`
	ObservationID string                         `json:"observation_id"`
	BindingID     string                         `json:"binding_id"`
	Object        Object                         `json:"object"`
	RawRef        *libraryvalidation.ArtifactRef `json:"raw_ref,omitempty"`
	ExtractionRef *libraryvalidation.ArtifactRef `json:"extraction_ref,omitempty"`
	Status        string                         `json:"status"`
	Code          string                         `json:"error_code,omitempty"`
}

// Completion records reconciliation before backend pointer CAS; inputs: signed worker artifacts; outputs: Outcome.
// Effects: app-owned outbox/pointer/conflict transaction. Partial coverage or any competing version must retain conflict/unknown.
// Parent retains library_sync_conflict rows containing all pointers/revision refs; ordinary MCP writers cannot create these rows.
type Completion struct {
	IntentID            string   `json:"intent_id"`
	LeaseID             string   `json:"lease_id"`
	Fence               int64    `json:"fence"`
	BasePointerRevision string   `json:"base_pointer_revision"`
	Written             *Pointer `json:"written_pointer"`
	Observed            []Object `json:"observed_versions"`
	Coverage            string   `json:"coverage"`
	Status              string   `json:"status"`
	Code                string   `json:"error_code,omitempty"`
}

// Outcome is the bounded durable backend result; inputs: atomic gate result; outputs: UI/history projection; effects: none.
type Outcome struct {
	Status      string `json:"status"`
	BindingID   string `json:"binding_id,omitempty"`
	ProposalID  string `json:"proposal_id,omitempty"`
	OperationID string `json:"operation_id,omitempty"`
	Code        string `json:"error_code,omitempty"`
}

// Backend is the parent-owned private HTTP service boundary, not an ordinary MCP writer or broadened DB login.
// Inputs: admitted identities, pinned refs and leased operations; outputs: durable outcomes and streamed payload bytes.
// Effects: guarded app transactions. Observe never auto-clears citations/currency; Complete independently enforces DB CAS.
type Backend interface {
	UploadIncomingPayload(context.Context, string, []byte, IncomingMetadata) (Outcome, error)
	Seen(context.Context, string) (bool, error)
	Claim(context.Context, OperationInput) (Claim, error)
	Payload(context.Context, Claim) ([]byte, error)
	BeginWrite(context.Context, Claim, string) (Intent, error)
	Observe(context.Context, Observation) (Outcome, error)
	Complete(context.Context, Claim, Completion) (Outcome, error)
	Failure(context.Context, Claim, string, string) (Outcome, error)
	Original(context.Context, string, string) (Binding, error)
}

// IncomingMetadata binds a complete private upload to its original B2 bytes without conflating derived text and source hashes.
// Inputs: exact raw/derived payload bytes and original hash; outputs: binary HTTP headers. Effects: none; payloads stay outside Temporal history.
type IncomingMetadata struct {
	ContentType  string `json:"content_type"`
	SHA256       string `json:"sha256"`
	SourceSHA256 string `json:"source_sha256"`
}

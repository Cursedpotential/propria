// Package sourcecontext defines the framework-neutral, reference-producing
// contract for actor-bound intake metadata submissions.
package sourcecontext

import (
	"context"
	"crypto/sha256"
	"time"
)

type ObservedSource struct {
	Key               string `json:"key"`
	Name              string `json:"name"`
	ByteLength        int64  `json:"byte_length"`
	ETag              string `json:"etag"`
	PreviewSHA256     string `json:"preview_sha256"`
	VerificationState string `json:"verification_state"`
}

type HumanAssertions struct {
	SourceClass          string `json:"source_class"`
	SourcePrincipal      string `json:"source_principal,omitempty"`
	OtherParty           string `json:"other_party,omitempty"`
	AcquiredAt           string `json:"acquired_at,omitempty"`
	AcquisitionMethod    string `json:"acquisition_method,omitempty"`
	AcquisitionAuthority string `json:"acquisition_authority,omitempty"`
	SourceDevice         string `json:"source_device,omitempty"`
	DeviceCustodian      string `json:"device_custodian,omitempty"`
	OccurredStart        string `json:"occurred_start,omitempty"`
	OccurredEnd          string `json:"occurred_end,omitempty"`
	DateCertainty        string `json:"date_certainty,omitempty"`
	Context              string `json:"context,omitempty"`
	Notes                string `json:"notes,omitempty"`
}

type Spec struct {
	RequestID, MatterID, CourtCaseID, SourceRef  string
	SupersedesRef                                string
	ObservedSource                               ObservedSource
	Assertions                                   HumanAssertions
	ChangeReason, ActorSubjectUID, ActorUsername string
	IdempotencyKey                               string
	ContentDigest                                [sha256.Size]byte
}

type Receipt struct {
	SourceContextRef string    `json:"source_context_ref"`
	ReceiptRef       string    `json:"receipt_ref"`
	ContentDigest    string    `json:"content_digest"`
	Revision         int       `json:"revision"`
	RecordedAt       time.Time `json:"recorded_at"`
}

type Writer interface {
	PersistSourceContext(context.Context, Spec) (Receipt, error)
}

type Validator interface {
	ValidateSourceContext(context.Context, string, string, string, string, string) error
}

// Revision is the newest actor-bound source-context revision for one request,
// returned with the exact immutable observation it is bound to. A supersession
// must name this revision and echo this observation unchanged; the store
// rejects anything else. Byline: Claude Code · Opus 5.5 · 2026-09-25
type Revision struct {
	SourceContextRef string          `json:"source_context_ref"`
	Revision         int             `json:"revision"`
	ObservedSource   ObservedSource  `json:"observed_source"`
	Assertions       HumanAssertions `json:"assertions"`
	ChangeReason     string          `json:"change_reason"`
	ActorUsername    string          `json:"actor_username"`
	ReceiptRef       string          `json:"receipt_ref"`
	RecordedAt       time.Time       `json:"recorded_at"`
}

// Registration is what register_source recorded for one request: the declared
// format the run started with, the source context it was validated against,
// and the retained original's identity once retain_original has run. Every
// field is a reference or a scalar; no source bytes. Byline: Claude Code ·
// Opus 5.5 · 2026-09-25
type Registration struct {
	SourceVersionRef string  `json:"source_version_ref"`
	DeclaredFormat   string  `json:"declared_format"`
	SourceContextRef *string `json:"source_context_ref"`
	OriginalFilename *string `json:"original_filename"`
	OriginalSHA256   *string `json:"original_sha256"`
	OriginalBytes    *int64  `json:"original_bytes"`
}

// Reader resolves the operator context and registration facts of one
// request. found=false is an ordinary answer: batch items and engine
// successor runs start without operator context, and a run that failed
// before register_source has no registration.
type Reader interface {
	CurrentSourceContext(ctx context.Context, requestID, sourceRef string) (Revision, bool, error)
	SourceRegistration(ctx context.Context, requestID string) (Registration, bool, error)
}

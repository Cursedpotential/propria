// Byline: Claude Code · Opus 5.5 · 2026-09-26
//
// Package sourcemeta is the read model behind the Review metadata screen and
// the contract for the owner's metadata corrections.
//
// The screen shows every metadata fact the platform durably holds for one
// file of a run: the source registration, the retained original, every
// context.source_metadata row the source-observation Activities recorded (all
// classes, native JSON verbatim, with extractor provenance), retained members,
// projected attachments, hash receipts — and the owner's corrections.
//
// Observed values are never written. A correction is an append-only,
// attributed overlay (context.source_metadata_correction) that supersedes
// exactly the newest revision of the same field on the same file.
//
// This package reads nothing itself and extracts nothing: extraction belongs to
// the source-observation Activities (ExtractEmbeddedMetadata and friends).
package sourcemeta

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	MaxValueBytes   = 64 << 10
	MaxReasonBytes  = 4000
	MaxFieldKeyLen  = 512
	MaxKeyBytes     = 512
	MaxMetadataRows = 100
	MaxMembers      = 200
	MaxCorrections  = 500
	MaxRowBytes     = 1 << 20

	ActionCorrect = "correct"
	ActionRetract = "retract"

	SubjectSource     = "source"
	SubjectMember     = "member"
	SubjectAttachment = "attachment"
)

var (
	ErrNotFound            = errors.New("this run has no registered source")
	ErrSubjectNotInRun     = errors.New("the file is not part of this run")
	ErrScopeMissing        = errors.New("this run has no matter and court case scope, so it cannot carry corrections")
	ErrStaleRevision       = errors.New("a newer correction exists for this field; reload and correct again")
	ErrIdempotencyConflict = errors.New("idempotency key is already bound to a different correction")
	ErrNotInstalled        = errors.New("the metadata correction overlay table is not installed yet")

	previewHandlePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{32,128}$`)
	sha256Pattern        = regexp.MustCompile(`^[0-9a-f]{64}$`)
	fieldKeyPattern      = regexp.MustCompile(`^[a-z][a-z_]{0,31}:.`)
)

// ObjectFacts is one retained object's identity.
type ObjectFacts struct {
	ObjectRef    string    `json:"object_ref"`
	StorageClass string    `json:"storage_class"`
	SHA256       string    `json:"sha256"`
	ByteLength   int64     `json:"byte_length"`
	ImmutableAt  time.Time `json:"immutable_at"`
}

// SourceFacts is what register_source and retain_original recorded.
type SourceFacts struct {
	SourceVersionRef string       `json:"source_version_ref"`
	SourceKey        string       `json:"source_key"`
	ProvenanceClass  string       `json:"provenance_class"`
	DeclaredFormat   string       `json:"declared_format"`
	OriginalFilename *string      `json:"original_filename"`
	AcquiredAt       time.Time    `json:"acquired_at"`
	Status           string       `json:"status"`
	MatterID         *string      `json:"matter_id"`
	CourtCaseID      *string      `json:"court_case_id"`
	SourceContextRef *string      `json:"source_context_ref"`
	Original         *ObjectFacts `json:"original"`
}

// MetadataRow is one context.source_metadata row, native JSON verbatim.
// Fields is null only when the row is larger than the screen reads; then
// FieldsBytes says how large it is.
type MetadataRow struct {
	MetadataRef      string          `json:"metadata_ref"`
	MetadataClass    string          `json:"metadata_class"`
	ExtractorID      string          `json:"extractor_id"`
	ExtractorVersion *string         `json:"extractor_version"`
	GeneratedAt      time.Time       `json:"generated_at"`
	ReceiptRef       string          `json:"receipt_ref"`
	Fields           json.RawMessage `json:"fields"`
	FieldsBytes      int64           `json:"fields_bytes"`
}

// Member is one retained object that belongs to the source version besides
// its original (a container member, an attachment, a derived reference).
type Member struct {
	ObjectRef       string          `json:"object_ref"`
	Role            string          `json:"role"`
	ParentObjectRef *string         `json:"parent_object_ref"`
	SHA256          string          `json:"sha256"`
	ByteLength      int64           `json:"byte_length"`
	StorageClass    string          `json:"storage_class"`
	MemberLocator   json.RawMessage `json:"member_locator"`
}

// AttachmentFacts is what the run's preview projected for one attachment
// digest, with every message that carries it.
type AttachmentFacts struct {
	SHA256           string   `json:"sha256"`
	Filename         *string  `json:"filename"`
	MediaType        *string  `json:"media_type"`
	ByteLength       *int64   `json:"byte_length"`
	SourceLocatorRef string   `json:"source_locator_ref"`
	MessageIDs       []string `json:"message_ids"`
}

// HashReceipt is one custody hash the platform recorded for the source or its
// generations.
type HashReceipt struct {
	HashKind     string    `json:"hash_kind"`
	Construction string    `json:"construction"`
	Digest       string    `json:"digest"`
	ComputedAt   time.Time `json:"computed_at"`
	ComputedBy   string    `json:"computed_by"`
}

// Correction is one revision of an owner correction.
type Correction struct {
	CorrectionRef  string          `json:"correction_ref"`
	SubjectSHA256  string          `json:"subject_sha256"`
	FieldKey       string          `json:"field_key"`
	Revision       int             `json:"revision"`
	SupersedesRef  *string         `json:"supersedes_ref"`
	Action         string          `json:"action"`
	SourceValue    json.RawMessage `json:"source_value"`
	CorrectedValue json.RawMessage `json:"corrected_value"`
	ChangeReason   string          `json:"change_reason"`
	ActorUsername  string          `json:"actor_username"`
	ReceiptRef     string          `json:"receipt_ref"`
	RecordedAt     time.Time       `json:"recorded_at"`
}

// View is the metadata screen for one file of one run. Lists are never null.
type View struct {
	PreviewHandle        string           `json:"preview_handle"`
	RequestID            string           `json:"request_id"`
	SourceRef            string           `json:"source_ref"`
	SubjectKind          string           `json:"subject_kind"`
	SubjectSHA256        string           `json:"subject_sha256"`
	Source               *SourceFacts     `json:"source"`
	Metadata             []MetadataRow    `json:"metadata"`
	MetadataTruncated    bool             `json:"metadata_truncated"`
	RecordMetadataCount  int64            `json:"record_metadata_count"`
	Members              []Member         `json:"members"`
	MembersTruncated     bool             `json:"members_truncated"`
	Attachment           *AttachmentFacts `json:"attachment"`
	AttachmentCount      int64            `json:"attachment_count"`
	Hashes               []HashReceipt    `json:"hashes"`
	Corrections          []Correction     `json:"corrections"`
	CorrectionsAvailable bool             `json:"corrections_available"`
}

// CorrectionSpec is one owner correction as the store persists it.
type CorrectionSpec struct {
	PreviewHandle   string
	SubjectSHA256   string
	FieldKey        string
	SupersedesRef   string
	Action          string
	SourceValue     json.RawMessage
	CorrectedValue  json.RawMessage
	ChangeReason    string
	ActorSubjectUID string
	ActorUsername   string
	IdempotencyKey  string
	ContentDigest   [sha256.Size]byte
}

// Receipt identifies one persisted correction revision.
type Receipt struct {
	CorrectionRef string    `json:"correction_ref"`
	ReceiptRef    string    `json:"receipt_ref"`
	ContentDigest string    `json:"content_digest"`
	Revision      int       `json:"revision"`
	RecordedAt    time.Time `json:"recorded_at"`
}

// Store reads the screen and persists corrections.
type Store interface {
	Read(ctx context.Context, previewHandle, subjectSHA256 string) (View, error)
	PersistCorrection(context.Context, CorrectionSpec) (Receipt, error)
}

// ValidatePreviewHandle checks the opaque run handle.
func ValidatePreviewHandle(handle string) error {
	if !previewHandlePattern.MatchString(handle) {
		return errors.New("preview_handle is not a valid review handle")
	}
	return nil
}

// ValidateSHA256 accepts "" (the run's source) or 64 lowercase hex digits.
func ValidateSHA256(value string, required bool) error {
	if value == "" && !required {
		return nil
	}
	if !sha256Pattern.MatchString(value) {
		return errors.New("subject_sha256 must be 64 lowercase hex characters")
	}
	return nil
}

// ValidateCorrection bounds one correction before it reaches the store.
func ValidateCorrection(spec CorrectionSpec) error {
	if err := ValidatePreviewHandle(spec.PreviewHandle); err != nil {
		return err
	}
	if err := ValidateSHA256(spec.SubjectSHA256, true); err != nil {
		return err
	}
	if len(spec.FieldKey) < 3 || len(spec.FieldKey) > MaxFieldKeyLen || !utf8.ValidString(spec.FieldKey) || !fieldKeyPattern.MatchString(spec.FieldKey) {
		return errors.New(`field_key must be "<origin>:<path>", for example embedded:EXIF:DateTimeOriginal`)
	}
	for _, r := range spec.FieldKey {
		if r < 0x20 || r == 0x7f {
			return errors.New("field_key cannot hold control characters")
		}
	}
	if spec.SupersedesRef != "" {
		if _, err := uuid.Parse(spec.SupersedesRef); err != nil {
			return errors.New("supersedes_ref must be a UUID")
		}
	}
	switch spec.Action {
	case ActionCorrect:
		if isAbsentJSON(spec.CorrectedValue) {
			return errors.New("a correction needs a corrected_value; use retract to withdraw one")
		}
	case ActionRetract:
		if !isAbsentJSON(spec.CorrectedValue) {
			return errors.New("a retract carries no corrected_value")
		}
		if spec.SupersedesRef == "" {
			return errors.New("a retract must name the correction it withdraws")
		}
	default:
		return errors.New(`action must be "correct" or "retract"`)
	}
	for name, value := range map[string]json.RawMessage{"source_value": spec.SourceValue, "corrected_value": spec.CorrectedValue} {
		if isAbsentJSON(value) {
			continue
		}
		if len(value) > MaxValueBytes || !json.Valid(value) {
			return fmt.Errorf("%s must be valid JSON of at most %d bytes", name, MaxValueBytes)
		}
	}
	reason := spec.ChangeReason
	if strings.TrimSpace(reason) == "" || len(reason) > MaxReasonBytes || strings.ContainsRune(reason, 0) || !utf8.ValidString(reason) {
		return fmt.Errorf("change_reason is required and must be at most %d bytes", MaxReasonBytes)
	}
	for _, value := range []string{spec.ActorSubjectUID, spec.ActorUsername} {
		if strings.TrimSpace(value) == "" || len(value) > 512 || strings.ContainsAny(value, "\x00\r\n") {
			return errors.New("Authentik actor identity is required")
		}
	}
	if strings.TrimSpace(spec.IdempotencyKey) == "" || len(spec.IdempotencyKey) > MaxKeyBytes || strings.ContainsAny(spec.IdempotencyKey, "\x00\r\n") {
		return fmt.Errorf("a bounded Idempotency-Key of at most %d bytes is required", MaxKeyBytes)
	}
	return nil
}

// CorrectionDigest binds one correction to its idempotency key.
func CorrectionDigest(spec CorrectionSpec) [sha256.Size]byte {
	return sha256.Sum256([]byte(strings.Join([]string{
		"metadata-correction.v1", spec.PreviewHandle, spec.SubjectSHA256, spec.FieldKey, spec.SupersedesRef,
		spec.Action, compactJSON(spec.SourceValue), compactJSON(spec.CorrectedValue), spec.ChangeReason,
		spec.ActorSubjectUID, spec.IdempotencyKey,
	}, "\x00")))
}

// DecodeSHA256 turns a validated hex digest into bytes.
func DecodeSHA256(value string) ([]byte, error) {
	if err := ValidateSHA256(value, true); err != nil {
		return nil, err
	}
	return hex.DecodeString(value)
}

func isAbsentJSON(value json.RawMessage) bool {
	trimmed := strings.TrimSpace(string(value))
	return trimmed == "" || trimmed == "null"
}

func compactJSON(value json.RawMessage) string {
	if isAbsentJSON(value) {
		return ""
	}
	var out bytes.Buffer
	if err := json.Compact(&out, value); err != nil {
		return string(value)
	}
	return out.String()
}

// Byline: Claude Code · Opus 5.5 · 2026-09-25
// Byline: Claude Code · Opus 5.5 · 2026-09-26 (finished: server-resolved scope, digests, stable errors)
//
// Package contextreview defines the append-only, actor-bound context review
// overlays for one normalized record: who it is TO and ABOUT, whether it is
// about the child, whether it is relevant — and, separately, the hindsight-only
// foreshadowing flag.
//
// Corrections are overlays. The record they describe is never written. Every
// revision is actor-bound, receipt-addressed and supersedes exactly the newest
// revision of the same record and matter.
//
// The foreshadowing flag records that the owner now knows, in hindsight, a
// record was significant. It lives in its own rows (horizon = "hindsight") and
// is read only on the hindsight path: Read with HorizonAsLived never queries
// it. AGENTS.md "WHY THIS EXISTS": one leaked future fact silently spoils the
// ignorant walk.
package contextreview

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
)

// Horizon selects the read path. The zero value parses to HorizonAsLived so an
// unspecified caller can never be handed a hindsight-only fact by default.
type Horizon string

const (
	HorizonAsLived   Horizon = "as_lived"
	HorizonHindsight Horizon = "hindsight"
)

const (
	MaxParties      = 32
	MaxLabelRunes   = 200
	MaxReasonBytes  = 4000
	MaxNoteBytes    = 4000
	MaxHistoryItems = 50
	MaxKeyBytes     = 512
)

var (
	ErrSubjectNotFound     = errors.New("the message is not part of this review")
	ErrScopeMissing        = errors.New("this run has no matter and court case scope, so it cannot carry review overlays")
	ErrStaleRevision       = errors.New("a newer revision exists; reload and review again")
	ErrIdempotencyConflict = errors.New("idempotency key is already bound to a different submission")
	ErrNotInstalled        = errors.New("the context review overlay tables are not installed yet")

	previewHandlePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{32,128}$`)
	aboutChildValues     = map[string]bool{"yes": true, "no": true, "unsure": true}
)

// ParseHorizon accepts "", "as_lived" or "hindsight". The empty string is the
// fail-safe as-lived horizon.
func ParseHorizon(value string) (Horizon, error) {
	switch Horizon(strings.TrimSpace(value)) {
	case "", HorizonAsLived:
		return HorizonAsLived, nil
	case HorizonHindsight:
		return HorizonHindsight, nil
	}
	return "", fmt.Errorf("horizon must be %q or %q", HorizonAsLived, HorizonHindsight)
}

// Party is one person or organisation a record is to or about. EntityID links
// registry.entity once entity extraction lands; until then it is nil.
type Party struct {
	Label    string  `json:"label"`
	EntityID *string `json:"entity_id"`
}

// Assertions are the as-lived-safe review fields. Nil means "not reviewed".
type Assertions struct {
	AddressedTo []Party `json:"addressed_to"`
	About       []Party `json:"about"`
	AboutChild  *string `json:"about_child"`
	Relevant    *bool   `json:"relevant"`
}

// Normalized returns the assertions with nil party lists as empty lists, so
// the persisted JSON and the digest never distinguish "absent" from "none".
func (a Assertions) Normalized() Assertions {
	if a.AddressedTo == nil {
		a.AddressedTo = []Party{}
	}
	if a.About == nil {
		a.About = []Party{}
	}
	return a
}

// Subject names one record as the Review surface showed it. MessageID is the
// normalized record identity the preview message projects
// (context.normalized_record_identity.id). The matter and court case are
// resolved by the store from the run itself, never taken from the caller.
type Subject struct {
	PreviewHandle string
	MessageID     string
}

type ReviewSpec struct {
	Subject
	SupersedesRef   string
	Assertions      Assertions
	ChangeReason    string
	ActorSubjectUID string
	ActorUsername   string
	IdempotencyKey  string
	ContentDigest   [sha256.Size]byte
}

type ForeshadowingSpec struct {
	Subject
	SupersedesRef   string
	Foreshadowing   bool
	Note            string
	ChangeReason    string
	ActorSubjectUID string
	ActorUsername   string
	IdempotencyKey  string
	ContentDigest   [sha256.Size]byte
}

// Receipt identifies one persisted revision. Horizon is "as_lived" for a
// review revision and "hindsight" for a foreshadowing revision.
type Receipt struct {
	Ref           string    `json:"ref"`
	ReceiptRef    string    `json:"receipt_ref"`
	ContentDigest string    `json:"content_digest"`
	Revision      int       `json:"revision"`
	RecordedAt    time.Time `json:"recorded_at"`
	Horizon       Horizon   `json:"horizon"`
}

type ReviewRevision struct {
	ReviewRef     string     `json:"review_ref"`
	Revision      int        `json:"revision"`
	Assertions    Assertions `json:"assertions"`
	ChangeReason  string     `json:"change_reason"`
	ActorUsername string     `json:"actor_username"`
	ReceiptRef    string     `json:"receipt_ref"`
	RecordedAt    time.Time  `json:"recorded_at"`
}

type ForeshadowingRevision struct {
	FlagRef       string    `json:"flag_ref"`
	Revision      int       `json:"revision"`
	Foreshadowing bool      `json:"foreshadowing"`
	Note          string    `json:"note"`
	Horizon       Horizon   `json:"horizon"`
	KnowledgeTime time.Time `json:"knowledge_time"`
	ChangeReason  string    `json:"change_reason"`
	ActorUsername string    `json:"actor_username"`
	ReceiptRef    string    `json:"receipt_ref"`
}

// View is one record's review history, newest first. Foreshadowing is nil on
// the as-lived path, so its key is absent from the JSON; on the hindsight path
// it is present, possibly empty.
type View struct {
	PreviewHandle string                   `json:"preview_handle"`
	MessageID     string                   `json:"message_id"`
	Horizon       Horizon                  `json:"horizon"`
	Reviews       []ReviewRevision         `json:"reviews"`
	Foreshadowing *[]ForeshadowingRevision `json:"foreshadowing,omitempty"`
}

type Store interface {
	PersistReview(context.Context, ReviewSpec) (Receipt, error)
	PersistForeshadowing(context.Context, ForeshadowingSpec) (Receipt, error)
	Read(context.Context, Subject, Horizon) (View, error)
}

// ValidateSubject checks the identity of the reviewed record.
func ValidateSubject(subject Subject) error {
	if !previewHandlePattern.MatchString(subject.PreviewHandle) {
		return errors.New("preview_handle is not a valid review handle")
	}
	if _, err := uuid.Parse(subject.MessageID); err != nil {
		return errors.New("message_id must be a UUID")
	}
	return nil
}

// ValidateAssertions bounds the review fields the store will persist.
func ValidateAssertions(assertions Assertions) error {
	for _, list := range []struct {
		name    string
		parties []Party
	}{{"addressed_to", assertions.AddressedTo}, {"about", assertions.About}} {
		if len(list.parties) > MaxParties {
			return fmt.Errorf("%s holds more than %d names", list.name, MaxParties)
		}
		seen := make(map[string]bool, len(list.parties))
		for _, party := range list.parties {
			if err := validateParty(party); err != nil {
				return fmt.Errorf("%s: %w", list.name, err)
			}
			key := strings.ToLower(party.Label)
			if seen[key] {
				return fmt.Errorf("%s names %q twice", list.name, party.Label)
			}
			seen[key] = true
		}
	}
	if assertions.AboutChild != nil && !aboutChildValues[*assertions.AboutChild] {
		return errors.New(`about_child must be "yes", "no", "unsure" or null`)
	}
	return nil
}

// ValidateChange bounds the reason and note every revision carries.
func ValidateChange(reason, note string) error {
	if strings.TrimSpace(reason) == "" || len(reason) > MaxReasonBytes || strings.ContainsRune(reason, 0) || !utf8.ValidString(reason) {
		return fmt.Errorf("change_reason is required and must be at most %d bytes", MaxReasonBytes)
	}
	if len(note) > MaxNoteBytes || strings.ContainsRune(note, 0) || !utf8.ValidString(note) {
		return fmt.Errorf("note must be at most %d bytes", MaxNoteBytes)
	}
	return nil
}

// ValidateSupersedes accepts "" (first revision) or a UUID.
func ValidateSupersedes(ref string) error {
	if ref == "" {
		return nil
	}
	if _, err := uuid.Parse(ref); err != nil {
		return errors.New("supersedes_ref must be a UUID")
	}
	return nil
}

// ValidateActor bounds the Authentik actor and the idempotency key.
func ValidateActor(subjectUID, username, idempotencyKey string) error {
	for _, value := range []string{subjectUID, username} {
		if strings.TrimSpace(value) == "" || len(value) > 512 || strings.ContainsAny(value, "\x00\r\n") {
			return errors.New("Authentik actor identity is required")
		}
	}
	if strings.TrimSpace(idempotencyKey) == "" || len(idempotencyKey) > MaxKeyBytes || strings.ContainsAny(idempotencyKey, "\x00\r\n") {
		return fmt.Errorf("a bounded Idempotency-Key of at most %d bytes is required", MaxKeyBytes)
	}
	return nil
}

func validateParty(party Party) error {
	label := strings.TrimSpace(party.Label)
	if label == "" || label != party.Label || utf8.RuneCountInString(label) > MaxLabelRunes || !utf8.ValidString(label) {
		return fmt.Errorf("every name needs 1-%d characters without surrounding spaces", MaxLabelRunes)
	}
	for _, r := range label {
		if unicode.IsControl(r) {
			return errors.New("names cannot hold control characters")
		}
	}
	if party.EntityID != nil {
		if _, err := uuid.Parse(*party.EntityID); err != nil {
			return errors.New("entity_id must be a UUID or null")
		}
	}
	return nil
}

// ReviewDigest binds one review submission to its idempotency key: the same
// key replayed with the same content answers the stored receipt, and the same
// key with different content is a conflict.
func ReviewDigest(spec ReviewSpec) [sha256.Size]byte {
	return digest("context-review.v1", spec.PreviewHandle, spec.MessageID, spec.SupersedesRef,
		spec.Assertions.Normalized(), spec.ChangeReason, spec.ActorSubjectUID, spec.IdempotencyKey)
}

// ForeshadowingDigest is ReviewDigest for the hindsight-only flag.
func ForeshadowingDigest(spec ForeshadowingSpec) [sha256.Size]byte {
	return digest("foreshadowing.v1", spec.PreviewHandle, spec.MessageID, spec.SupersedesRef,
		struct {
			Foreshadowing bool   `json:"foreshadowing"`
			Note          string `json:"note"`
		}{spec.Foreshadowing, spec.Note}, spec.ChangeReason, spec.ActorSubjectUID, spec.IdempotencyKey)
}

func digest(kind, handle, messageID, supersedes string, body any, reason, actor, key string) [sha256.Size]byte {
	canonical, _ := json.Marshal(body)
	return sha256.Sum256([]byte(strings.Join([]string{
		kind, handle, messageID, supersedes, string(canonical), reason, actor, key,
	}, "\x00")))
}

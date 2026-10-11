package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/extraction/entities"
	"github.com/google/uuid"
)

// AISourcePin identifies the retained original and its provider version without substituting a normalized generation.
// Inputs: custody UUIDs, original SHA256 and optional verified provider version. Outputs: a validated pin.
// Effects: none. Choose for AI conversation candidates whose source is context.source_version.
type AISourcePin struct {
	SourceVersionID string  `json:"source_version_id"`
	SourceObjectID  string  `json:"source_object_id"`
	VersionID       *string `json:"version_id"`
	SourceSHA256    string  `json:"source_sha256"`
	SourceRef       string  `json:"source_ref,omitempty"`
	PreparedRef     string  `json:"prepared_ref,omitempty"`
}

// AISourceSpan locates a bounded source excerpt in the retained native export.
// Inputs: native JSON pointer or whole-text source and codepoint interval with exact excerpt hash. Outputs: a validated locator.
// Effects: none. Choose for review evidence instead of storing a conversation body.
type AISourceSpan struct {
	Start  int    `json:"start"`
	End    int    `json:"end"`
	SHA256 string `json:"sha256"`
}

// AICandidate is one bounded entity, dated event, or factual account proposed from a retained AI source.
// Inputs: source pin, native pointer, span and a short grounded quote. Outputs: a pending review row.
// Effects: none until StageAICandidates persists it. Choose fact for accounts without a known event time.
type AICandidate struct {
	AISourcePin
	NativeJSONPointer   string       `json:"native_json_pointer"`
	SourceSpan          AISourceSpan `json:"source_span"`
	SpanUnit            string       `json:"span_unit"`
	Kind                string       `json:"kind"`
	ReportedKind        string       `json:"reported_kind"`
	ReviewDomain        string       `json:"review_domain"`
	Name                string       `json:"name,omitempty"`
	EntityType          string       `json:"entity_type,omitempty"`
	EventType           string       `json:"event_type,omitempty"`
	Predicate           string       `json:"predicate,omitempty"`
	Statement           string       `json:"statement,omitempty"`
	EvidenceQuote       string       `json:"evidence_quote,omitempty"`
	OccurredAt          *time.Time   `json:"occurred_at"`
	SourceAvailableFrom *time.Time   `json:"source_available_from,omitempty"`
	Confidence          float64      `json:"confidence"`
}

// AIReviewRow is the bounded row exposed to the owner for an explicit decision.
// Inputs: staged candidate identity and provenance. Outputs: review state and candidate metadata.
// Effects: none. Choose for AI review reads; it never exposes full conversation text.
type AIReviewRow struct {
	ID            string          `json:"candidate_id"`
	Kind          string          `json:"kind"`
	ReportedKind  string          `json:"reported_kind"`
	ReviewDomain  string          `json:"review_domain"`
	ReviewState   string          `json:"review_state"`
	ContentSHA256 string          `json:"content_sha256"`
	DecisionID    string          `json:"decision_id,omitempty"`
	Candidate     json.RawMessage `json:"candidate"`
}

// AIReviewStore is the retained-source review seam, separate from normalized-message extraction.
// Inputs: verified source pins and bounded candidates. Outputs: pending rows and review decisions.
// Effects: writes only working extraction/candidate tables; it has no promotion method.
// Choose for AI originals; the SMS Store retains its existing generation contract.
type AIReviewStore interface {
	VerifyAISource(context.Context, string, AISourcePin) (string, error)
	ResolveAIPreview(context.Context, string, AISourcePin) (string, error)
	StageAICandidates(context.Context, string, string, string, []AICandidate) ([]string, error)
	ListAICandidates(context.Context, string, string, int) ([]AIReviewRow, error)
	DecideAICandidate(context.Context, AISourcePin, string, string, string, entities.Actor, string, time.Time) (string, error)
}

// ValidateAICandidate rejects unpinned, ungrounded or unbounded candidate metadata before any database write.
// Inputs: candidate from a retained bundle. Outputs: user-correctable validation error.
// Effects: none. Choose at both API and direct service entry points.
func ValidateAICandidate(c AICandidate) error {
	if _, err := uuid.Parse(c.SourceVersionID); err != nil {
		return errors.New("source_version_id must be a UUID")
	}
	if c.SourceRef == "" {
		if _, err := uuid.Parse(c.SourceObjectID); err != nil {
			return errors.New("source_object_id must be a UUID")
		}
	} else if c.SourceObjectID != "" || c.PreparedRef == "" || len(c.SourceRef) > 2048 || len(c.PreparedRef) > 4096 || strings.ContainsAny(c.SourceRef+c.PreparedRef, "\x00\r\n") {
		return errors.New("native context requires its exact source and prepared references without a retained object ID")
	}
	if !isSHA256(c.SourceSHA256) || !isSHA256(c.SourceSpan.SHA256) {
		return errors.New("source and span SHA256 must be full lowercase hex digests")
	}
	if c.VersionID != nil && (strings.TrimSpace(*c.VersionID) == "" || len(*c.VersionID) > 512) {
		return errors.New("version_id must be a bounded verified value or null")
	}
	if (c.NativeJSONPointer != "" && !strings.HasPrefix(c.NativeJSONPointer, "/")) || len(c.NativeJSONPointer) > 2048 || c.SourceSpan.Start < 0 || c.SourceSpan.End <= c.SourceSpan.Start {
		return errors.New("native JSON pointer or whole-text source span is required")
	}
	if c.SpanUnit != "unicode_codepoint" {
		return errors.New("span_unit must be unicode_codepoint")
	}
	if len(c.EvidenceQuote) == 0 || len(c.EvidenceQuote) > 1000 {
		return errors.New("evidence_quote must be 1-1000 bytes")
	}
	if math.IsNaN(c.Confidence) || math.IsInf(c.Confidence, 0) || c.Confidence < 0 || c.Confidence > 1 {
		return errors.New("confidence must be between zero and one")
	}
	if c.SourceAvailableFrom != nil && c.SourceAvailableFrom.IsZero() {
		return errors.New("source_available_from must be a verified nonzero native source clock")
	}
	if len(c.Name) > 200 || len(c.Statement) > 2000 || len(c.Predicate) > 100 || len(c.EventType) > 100 {
		return errors.New("candidate text exceeds review bounds")
	}
	if c.ReportedKind != c.Kind {
		return errors.New("reported_kind must preserve the original candidate kind")
	}
	if c.ReviewDomain != "ai_chat_content" && c.ReviewDomain != "ai_chat_account" {
		return errors.New("review_domain must identify AI content or account")
	}
	switch c.Kind {
	case "entity":
		if strings.TrimSpace(c.Name) == "" || !entities.ValidRegistryType(entities.RegistryType(c.EntityType)) {
			return errors.New("entity needs a name and valid entity_type")
		}
	case "event":
		if strings.TrimSpace(c.Statement) == "" || strings.TrimSpace(c.EventType) == "" {
			return errors.New("event needs event_type and statement")
		}
	case "fact":
		if strings.TrimSpace(c.Predicate) == "" || strings.TrimSpace(c.Statement) == "" {
			return errors.New("fact needs predicate and statement")
		}
	case "strategy", "history":
		if c.ReviewDomain != "ai_chat_account" {
			return errors.New("strategy/history require ai_chat_account review domain")
		}
		if strings.TrimSpace(c.Statement) == "" {
			return errors.New("strategy/history account needs a bounded statement")
		}
	case "artifact", "document", "work_product":
		return errors.New("artifact/document/work_product requires retained file-manifest placement review, not candidate fact staging")
	default:
		return fmt.Errorf("unsupported AI candidate kind %q", c.Kind)
	}
	return nil
}

func isSHA256(value string) bool {
	if len(value) != 64 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

// AICandidateDigest binds a candidate's content and source pin for retry-safe identity.
// Inputs: a validated candidate. Outputs: SHA256 bytes. Effects: none.
// Choose for retained-source deduplication instead of the SMS generation digest.
func AICandidateDigest(c AICandidate) [32]byte {
	raw, _ := json.Marshal(c)
	return sha256.Sum256(raw)
}

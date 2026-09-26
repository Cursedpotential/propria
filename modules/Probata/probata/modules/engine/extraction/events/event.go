// Byline: Claude Code · Opus 5.5 · 2026-09-25
//
// Package events proposes and corrects timeline events drawn from a run's
// normalized messages. Like entities it is pure: activities do the I/O.
//
// Knowledge horizon (AGENTS.md "WHY THIS EXISTS"): an event is extraction
// output, not a belief. Every proposal carries occurred_at (event time) and
// the source records' source_available_from; an as-lived reader may see the
// event no earlier than the latest availability among its sources. "Worth
// recalling" is an owner provenance marker (detected_by = owner). It is not a
// hindsight or foreshadowing flag, and this package has none.
package events

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/extraction/entities"
)

// Event types offered to the model and the owner. timeline.event_candidate
// stores free text; this closed list keeps proposals comparable.
var EventTypes = []string{
	"appointment", "court", "medical", "school", "custody_exchange", "travel",
	"incident", "communication", "financial", "residence", "work", "other",
}

// ValidEventType reports whether value is one of EventTypes.
func ValidEventType(value string) bool {
	for _, known := range EventTypes {
		if known == value {
			return true
		}
	}
	return false
}

// Temporal precision values (timeline.event_candidate CHECK).
const (
	PrecisionPoint     = "point"
	PrecisionInterval  = "interval"
	PrecisionUncertain = "uncertain"
)

// OwnerMarkExtractor is the extraction_run identity of owner-marked events.
const (
	OwnerMarkExtractor        = "owner.mark"
	OwnerMarkExtractorVersion = "1"
)

// SourceRecord is one message an event is drawn from.
type SourceRecord struct {
	RecordID            string     `json:"record_id"`
	Ordinal             int64      `json:"ordinal"`
	OccurredAt          *time.Time `json:"occurred_at,omitempty"`
	SourceAvailableFrom *time.Time `json:"source_available_from,omitempty"`
	Start               *int       `json:"start,omitempty"`
	End                 *int       `json:"end,omitempty"`
	Snippet             string     `json:"snippet,omitempty"`
}

// Proposal is one proposed event.
type Proposal struct {
	CandidateID        string               `json:"candidate_id,omitempty"`
	ExtractionRunID    string               `json:"extraction_run_id,omitempty"`
	Title              string               `json:"title"`
	Description        string               `json:"description,omitempty"`
	EventType          string               `json:"event_type"`
	OccurredAt         *time.Time           `json:"occurred_at,omitempty"`
	TemporalPrecision  string               `json:"temporal_precision"`
	TemporalConfidence float64              `json:"temporal_confidence"`
	WhenStated         string               `json:"when_stated,omitempty"`
	SourceRecords      []SourceRecord       `json:"source_records"`
	EntityKeys         []string             `json:"entity_keys,omitempty"`
	EntityNames        []string             `json:"entity_names,omitempty"`
	DetectedBy         string               `json:"detected_by"`
	Extractors         []string             `json:"extractors"`
	Confidence         float64              `json:"confidence"`
	Flags              []entities.Flag      `json:"flags,omitempty"`
	Supersedes         []string             `json:"supersedes,omitempty"`
	Correction         *entities.Correction `json:"correction,omitempty"`
	ReviewState        string               `json:"review_state,omitempty"`
	PromotedToID       string               `json:"promoted_to_id,omitempty"`
	GenerationID       string               `json:"normalized_generation_id"`
	SourceVersionID    string               `json:"source_version_id,omitempty"`
	PreviewHandle      string               `json:"preview_handle,omitempty"`
}

// Bounds.
const (
	MaxTitleRunes       = 200
	MaxDescriptionRunes = 2000
	MaxSourceRecords    = 50
	MaxEntityKeys       = 32
)

// SourceAvailableFrom is the latest availability among the event's sources:
// the earliest moment every source it rests on had been acquired. An
// ignorant reader may not see the event before it. Nil when any source
// lacks a clock — which fails closed at validation.
func (p Proposal) SourceAvailableFrom() *time.Time {
	var latest *time.Time
	for _, record := range p.SourceRecords {
		if record.SourceAvailableFrom == nil {
			return nil
		}
		if latest == nil || record.SourceAvailableFrom.After(*latest) {
			value := *record.SourceAvailableFrom
			latest = &value
		}
	}
	return latest
}

// Included reports whether the event takes part in the next commit.
func (p Proposal) Included() bool {
	return p.ReviewState == "" || p.ReviewState == entities.StatePending
}

// PrimaryRecordID is the first (earliest ordinal) source record.
func (p Proposal) PrimaryRecordID() string {
	if len(p.SourceRecords) == 0 {
		return ""
	}
	return p.SourceRecords[0].RecordID
}

// Normalize puts the proposal in canonical, bounded shape.
func (p *Proposal) Normalize() {
	p.Title = clip(strings.TrimSpace(p.Title), MaxTitleRunes)
	p.Description = clip(strings.TrimSpace(p.Description), MaxDescriptionRunes)
	if !ValidEventType(p.EventType) {
		p.EventType = "other"
	}
	switch p.TemporalPrecision {
	case PrecisionPoint, PrecisionInterval, PrecisionUncertain:
	default:
		p.TemporalPrecision = PrecisionUncertain
	}
	if p.TemporalConfidence < 0 {
		p.TemporalConfidence = 0
	}
	if p.TemporalConfidence > 1 {
		p.TemporalConfidence = 1
	}
	if p.Confidence < 0 {
		p.Confidence = 0
	}
	if p.Confidence > 1 {
		p.Confidence = 1
	}
	sort.SliceStable(p.SourceRecords, func(i, j int) bool { return p.SourceRecords[i].Ordinal < p.SourceRecords[j].Ordinal })
	seen := map[string]bool{}
	records := p.SourceRecords[:0]
	for _, record := range p.SourceRecords {
		if record.RecordID == "" || seen[record.RecordID] {
			continue
		}
		seen[record.RecordID] = true
		records = append(records, record)
	}
	p.SourceRecords = records
	if len(p.SourceRecords) > MaxSourceRecords {
		p.SourceRecords = p.SourceRecords[:MaxSourceRecords]
	}
	p.EntityKeys = uniqueSorted(p.EntityKeys)
	if len(p.EntityKeys) > MaxEntityKeys {
		p.EntityKeys = p.EntityKeys[:MaxEntityKeys]
	}
	p.EntityNames = uniqueSorted(p.EntityNames)
	p.Extractors = uniqueSorted(p.Extractors)
	if p.DetectedBy == "" {
		p.DetectedBy = entities.DetectedAuto
	}
}

func clip(value string, limit int) string {
	runes := []rune(value)
	if len(runes) > limit {
		return string(runes[:limit])
	}
	return value
}

func uniqueSorted(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}

type contentView struct {
	Title       string               `json:"title"`
	Description string               `json:"description,omitempty"`
	EventType   string               `json:"event_type"`
	OccurredAt  *time.Time           `json:"occurred_at,omitempty"`
	Precision   string               `json:"temporal_precision"`
	WhenStated  string               `json:"when_stated,omitempty"`
	Records     []string             `json:"records"`
	EntityKeys  []string             `json:"entity_keys,omitempty"`
	EntityNames []string             `json:"entity_names,omitempty"`
	DetectedBy  string               `json:"detected_by"`
	Extractors  []string             `json:"extractors"`
	Supersedes  []string             `json:"supersedes,omitempty"`
	Correction  *entities.Correction `json:"correction,omitempty"`
	Generation  string               `json:"generation"`
	ReviewState string               `json:"review_state,omitempty"`
}

// ContentSHA256 is the dedup digest stored in candidate_event.content_sha256.
func (p Proposal) ContentSHA256() [32]byte {
	records := make([]string, 0, len(p.SourceRecords))
	for _, record := range p.SourceRecords {
		records = append(records, record.RecordID)
	}
	supersedes := append([]string(nil), p.Supersedes...)
	sort.Strings(supersedes)
	raw, _ := json.Marshal(contentView{
		Title: p.Title, Description: p.Description, EventType: p.EventType, OccurredAt: p.OccurredAt,
		Precision: p.TemporalPrecision, WhenStated: p.WhenStated, Records: records,
		EntityKeys: p.EntityKeys, EntityNames: p.EntityNames, DetectedBy: p.DetectedBy,
		Extractors: p.Extractors, Supersedes: supersedes, Correction: p.Correction,
		Generation: p.GenerationID, ReviewState: p.ReviewState,
	})
	return sha256.Sum256(raw)
}

// ContentHex is ContentSHA256 in hex.
func (p Proposal) ContentHex() string {
	digest := p.ContentSHA256()
	return hex.EncodeToString(digest[:])
}

// MarkRequest is the owner's "event worth recalling" on one record.
type MarkRequest struct {
	RecordID    string `json:"record_id"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	EventType   string `json:"event_type,omitempty"`
}

// FromRecord builds an owner-marked event from one message. The event is
// dated by the SOURCE record's own time, never by when the owner clicked.
func FromRecord(record entities.MessageView, request MarkRequest, scope entities.RunScope, actor entities.Actor, at time.Time) (Proposal, error) {
	if actor.SubjectUID == "" || actor.Username == "" {
		return Proposal{}, errors.New("an authenticated actor is required")
	}
	if record.RecordID == "" || record.RecordID != strings.TrimSpace(request.RecordID) {
		return Proposal{}, errors.New("the record does not match the request")
	}
	if record.OccurredAt == nil {
		return Proposal{}, errors.New("this record has no time of its own, so it cannot date an event; add the event by hand with a stated time instead")
	}
	if record.SourceAvailableFrom == nil {
		return Proposal{}, errors.New("this record has no source availability clock; it cannot enter the timeline")
	}
	title := strings.TrimSpace(request.Title)
	if title == "" {
		title = defaultTitle(record)
	}
	eventType := strings.TrimSpace(request.EventType)
	if eventType == "" {
		eventType = "other"
	}
	if !ValidEventType(eventType) {
		return Proposal{}, fmt.Errorf("event_type %q is not one of %v", eventType, EventTypes)
	}
	occurred := *record.OccurredAt
	available := *record.SourceAvailableFrom
	proposal := Proposal{
		Title: title, Description: request.Description, EventType: eventType,
		OccurredAt: &occurred, TemporalPrecision: PrecisionPoint, TemporalConfidence: 1,
		SourceRecords: []SourceRecord{{
			RecordID: record.RecordID, Ordinal: record.Ordinal, OccurredAt: &occurred,
			SourceAvailableFrom: &available, Snippet: clip(record.Body, 200),
		}},
		DetectedBy: entities.DetectedOwner, Extractors: []string{OwnerMarkExtractor + "@" + OwnerMarkExtractorVersion},
		Confidence: 1, GenerationID: scope.GenerationID, SourceVersionID: scope.SourceVersionID, PreviewHandle: scope.PreviewHandle,
		Correction: &entities.Correction{Op: "mark_event", Actor: actor, At: at.UTC()},
	}
	proposal.Normalize()
	return proposal, nil
}

func defaultTitle(record entities.MessageView) string {
	body := strings.Join(strings.Fields(record.Body), " ")
	if body == "" {
		return "Event from message #" + fmt.Sprint(record.Ordinal)
	}
	return clip(body, 80)
}

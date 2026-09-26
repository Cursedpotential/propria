// Byline: Claude Code · Opus 5.5 · 2026-09-25

package events

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/extraction/entities"
)

// Event correction operations.
const (
	OpReject  = "reject"
	OpRestore = "restore"
	OpEdit    = "edit"
	OpMerge   = "merge"
)

// CorrectionRequest is one owner edit to event proposals. Nil fields of an
// edit are left unchanged; an empty EntityKeys slice (not nil) clears them.
type CorrectionRequest struct {
	Op           string     `json:"op"`
	CandidateIDs []string   `json:"candidate_ids"`
	Title        *string    `json:"title,omitempty"`
	Description  *string    `json:"description,omitempty"`
	EventType    *string    `json:"event_type,omitempty"`
	OccurredAt   *time.Time `json:"occurred_at,omitempty"`
	EntityKeys   []string   `json:"entity_keys,omitempty"`
	SetEntities  bool       `json:"set_entities,omitempty"`
	Note         string     `json:"note,omitempty"`
}

// CorrectionResult mirrors entities.CorrectionResult for events.
type CorrectionResult struct {
	Insert    []Proposal `json:"insert"`
	Supersede []string   `json:"supersede"`
}

// ApplyCorrection applies one owner edit to current event proposals.
func ApplyCorrection(current map[string]Proposal, request CorrectionRequest, actor entities.Actor, at time.Time, digest string) (CorrectionResult, error) {
	if actor.SubjectUID == "" || actor.Username == "" {
		return CorrectionResult{}, errors.New("an authenticated actor is required")
	}
	if len(request.CandidateIDs) == 0 {
		return CorrectionResult{}, errors.New("candidate_ids is required")
	}
	var bases []Proposal
	for _, id := range request.CandidateIDs {
		base, ok := current[id]
		if !ok {
			return CorrectionResult{}, fmt.Errorf("event %s is not current: %w", id, entities.ErrConflict)
		}
		bases = append(bases, base)
	}
	stamp := func(p Proposal, supersedes ...string) Proposal {
		p.CandidateID, p.PromotedToID = "", ""
		p.Supersedes = append([]string(nil), supersedes...)
		p.Correction = &entities.Correction{Op: request.Op, Actor: actor, At: at.UTC(), Note: strings.TrimSpace(request.Note), RequestDigest: digest}
		p.Normalize()
		return p
	}
	requirePending := func(p Proposal) error {
		switch p.ReviewState {
		case entities.StatePending, "":
			return nil
		case entities.StateApproved:
			return fmt.Errorf("%q is already on the timeline", p.Title)
		case entities.StateRejected:
			return fmt.Errorf("%q is rejected; restore it first", p.Title)
		default:
			return fmt.Errorf("%q is %s: %w", p.Title, p.ReviewState, entities.ErrConflict)
		}
	}
	switch request.Op {
	case OpReject, OpRestore:
		if len(bases) != 1 {
			return CorrectionResult{}, fmt.Errorf("%s takes exactly one event", request.Op)
		}
		base := bases[0]
		next := cloneEvent(base)
		if request.Op == OpReject {
			if err := requirePending(base); err != nil {
				return CorrectionResult{}, err
			}
			next.ReviewState = entities.StateRejected
		} else {
			if base.ReviewState != entities.StateRejected {
				return CorrectionResult{}, fmt.Errorf("%q is not rejected", base.Title)
			}
			next.ReviewState = entities.StatePending
		}
		next = stamp(next, base.CandidateID)
		return CorrectionResult{Insert: []Proposal{next}, Supersede: []string{base.CandidateID}}, nil
	case OpEdit:
		if len(bases) != 1 {
			return CorrectionResult{}, errors.New("edit takes exactly one event")
		}
		base := bases[0]
		if err := requirePending(base); err != nil {
			return CorrectionResult{}, err
		}
		next := cloneEvent(base)
		if request.Title != nil {
			if strings.TrimSpace(*request.Title) == "" {
				return CorrectionResult{}, errors.New("an event needs a title")
			}
			next.Title = *request.Title
		}
		if request.Description != nil {
			next.Description = *request.Description
		}
		if request.EventType != nil {
			if !ValidEventType(*request.EventType) {
				return CorrectionResult{}, fmt.Errorf("event_type %q is not one of %v", *request.EventType, EventTypes)
			}
			next.EventType = *request.EventType
		}
		if request.OccurredAt != nil {
			value := request.OccurredAt.UTC()
			next.OccurredAt = &value
			// The owner stated the time; it is exact as stated.
			next.TemporalPrecision, next.TemporalConfidence = PrecisionPoint, 1
		}
		if request.SetEntities {
			next.EntityKeys = append([]string(nil), request.EntityKeys...)
		}
		next = stamp(next, base.CandidateID)
		next.ReviewState = entities.StatePending
		return CorrectionResult{Insert: []Proposal{next}, Supersede: []string{base.CandidateID}}, nil
	case OpMerge:
		if len(bases) < 2 {
			return CorrectionResult{}, errors.New("merge takes two or more events")
		}
		merged := cloneEvent(bases[0])
		ids := []string{bases[0].CandidateID}
		for _, base := range bases {
			if err := requirePending(base); err != nil {
				return CorrectionResult{}, err
			}
		}
		for _, base := range bases[1:] {
			ids = append(ids, base.CandidateID)
			merged.SourceRecords = append(merged.SourceRecords, base.SourceRecords...)
			merged.EntityKeys = append(merged.EntityKeys, base.EntityKeys...)
			merged.EntityNames = append(merged.EntityNames, base.EntityNames...)
			merged.Extractors = append(merged.Extractors, base.Extractors...)
			if merged.Description == "" {
				merged.Description = base.Description
			}
			if base.DetectedBy == entities.DetectedOwner {
				merged.DetectedBy = entities.DetectedOwner
			}
			if merged.OccurredAt == nil || (base.OccurredAt != nil && base.OccurredAt.Before(*merged.OccurredAt)) {
				merged.OccurredAt = base.OccurredAt
			}
		}
		if request.Title != nil && strings.TrimSpace(*request.Title) != "" {
			merged.Title = *request.Title
		}
		merged = stamp(merged, ids...)
		merged.ReviewState = entities.StatePending
		return CorrectionResult{Insert: []Proposal{merged}, Supersede: ids}, nil
	default:
		return CorrectionResult{}, fmt.Errorf("unknown event correction op %q", request.Op)
	}
}

func cloneEvent(p Proposal) Proposal {
	out := p
	out.SourceRecords = append([]SourceRecord(nil), p.SourceRecords...)
	out.EntityKeys = append([]string(nil), p.EntityKeys...)
	out.EntityNames = append([]string(nil), p.EntityNames...)
	out.Extractors = append([]string(nil), p.Extractors...)
	out.Flags = append([]entities.Flag(nil), p.Flags...)
	return out
}

// ResolveEntityKeys maps an event's entity keys to the included entity
// proposals that currently cover them. Unresolved keys are returned so
// validation can name them.
func ResolveEntityKeys(event Proposal, proposals []entities.Proposal) (resolved map[string]string, unresolved []string, ambiguous []string) {
	resolved = map[string]string{}
	for _, key := range event.EntityKeys {
		var owners []string
		for _, proposal := range proposals {
			if !proposal.Included() && proposal.ReviewState != entities.StateApproved {
				continue
			}
			if proposal.Covers(key) {
				owners = append(owners, proposal.CandidateID)
			}
		}
		switch len(owners) {
		case 0:
			unresolved = append(unresolved, key)
		case 1:
			resolved[key] = owners[0]
		default:
			ambiguous = append(ambiguous, key)
		}
	}
	return resolved, unresolved, ambiguous
}

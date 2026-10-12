package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Cursedpotential/probata/engine/extraction/entities"
	"github.com/google/uuid"
)

// AIAvailabilityCorrection carries an owner assertion about when cited material became available.
// Inputs: original source clock, human corrected clock/reason and current decision reference. Outputs: bounded overlay.
// Effects: none; choose inside the existing approval decision, never to mutate native candidate metadata.
type AIAvailabilityCorrection struct {
	FieldKey       string          `json:"field_key"`
	SourceValue    json.RawMessage `json:"source_value"`
	CorrectedValue json.RawMessage `json:"corrected_value"`
	ChangeReason   string          `json:"change_reason"`
	SupersedesRef  string          `json:"supersedes_ref,omitempty"`
	Action         string          `json:"action"`
}

// AIAvailabilityReviewStore extends the existing owner decision without changing legacy callers.
// Inputs: immutable candidate pin/hash and optional availability correction. Outputs: append-only decision ID.
// Effects: existing review ledger/receipt writes only; choose when owner supplies discovery information.
type AIAvailabilityReviewStore interface {
	DecideAICandidateWithCorrection(context.Context, AISourcePin, string, string, string, entities.Actor, string, time.Time, *AIAvailabilityCorrection) (string, error)
}

// Validate checks the bounded existing correction vocabulary without inferring a clock.
// Inputs: optional correction. Outputs: validation error. Effects: none; pick before hashing or ledger access.
func (c *AIAvailabilityCorrection) Validate() error {
	if c == nil {
		return nil
	}
	if c.FieldKey != "source:source_available_from" || strings.TrimSpace(c.ChangeReason) == "" || len(c.ChangeReason) > 4000 || !utf8.ValidString(c.ChangeReason) || strings.ContainsRune(c.ChangeReason, 0) {
		return errors.New("availability correction requires source:source_available_from and a bounded human reason")
	}
	if _, err := availabilityClock(c.SourceValue); err != nil {
		return err
	}
	if c.SupersedesRef != "" {
		if id, err := uuid.Parse(c.SupersedesRef); err != nil || id.String() != c.SupersedesRef {
			return errors.New("supersedes_ref must be the current canonical decision UUID")
		}
	}
	switch c.Action {
	case "correct":
		value, err := availabilityClock(c.CorrectedValue)
		if err != nil || value == nil {
			return errors.New("corrected_value must be a nonzero RFC3339 timestamp")
		}
	case "retract":
		if string(c.CorrectedValue) != "null" || c.SupersedesRef == "" {
			return errors.New("retract requires explicit null and the current correction decision")
		}
	default:
		return errors.New("availability action must be correct or retract")
	}
	return nil
}

// Effective validates the original clock and returns the attributed overlay without changing the candidate.
// Inputs: original candidate availability and owner decision time. Outputs: effective clock or error.
// Effects: none; choose only after receipt and ledger equality have been established.
func (c *AIAvailabilityCorrection) Effective(original *time.Time, at time.Time) (*time.Time, error) {
	if c == nil {
		return original, nil
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	source, err := availabilityClock(c.SourceValue)
	if err != nil {
		return nil, err
	}
	if (source == nil) != (original == nil) || (source != nil && !source.Equal(*original)) {
		return nil, errors.New("source_value differs from original candidate availability")
	}
	if c.Action == "retract" {
		return original, nil
	}
	value, _ := availabilityClock(c.CorrectedValue)
	if at.IsZero() || value.UnixMicro() > at.UnixMicro() || (original != nil && value.Before(*original)) {
		return nil, errors.New("human discovery must be at or after native availability and no later than its decision")
	}
	return value, nil
}

// availabilityClock accepts only an explicit unknown or a bounded timezone-qualified timestamp.
// Inputs: raw JSON. Outputs: optional UTC clock. Effects: none; choose for availability assertions only.
func availabilityClock(raw json.RawMessage) (*time.Time, error) {
	if len(raw) == 0 || len(raw) > 128 {
		return nil, errors.New("availability value must be explicit null or RFC3339 timestamp")
	}
	if string(raw) == "null" {
		return nil, nil
	}
	var text string
	if json.Unmarshal(raw, &text) != nil {
		return nil, errors.New("availability value must be explicit null or RFC3339 timestamp")
	}
	clock, err := time.Parse(time.RFC3339Nano, text)
	if err != nil || clock.IsZero() {
		return nil, errors.New("availability timestamp must be nonzero RFC3339 with timezone")
	}
	clock = clock.UTC()
	return &clock, nil
}

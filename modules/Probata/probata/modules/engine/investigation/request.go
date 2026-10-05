// Byline: Codex · GPT-5 · 2026-10-05 (single-case operating contract)
// Package investigation describes durable requests awaiting a future executor.
package investigation

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/google/uuid"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrScope    = errors.New("investigation scope does not match the selected case")
	ErrConflict = errors.New("investigation request conflicts with an accepted request")
	ErrSource   = errors.New("investigation source is missing or its version has changed")
	ErrNotFound = errors.New("investigation request was not found")
)

const MaxSources = 30

type Scope struct {
	Mode        caseidentity.Mode `json:"mode"`
	MatterID    string            `json:"matter_id"`
	CourtCaseID string            `json:"court_case_id"`
}
type Source struct {
	Kind          string `json:"kind"`
	RecordID      string `json:"record_id"`
	RecordVersion string `json:"record_version"`
}
type Request struct {
	Scope
	LegalMatterID string   `json:"legal_matter_id"`
	ClaimID       string   `json:"claim_id"`
	FollowupID    string   `json:"followup_id"`
	Question      string   `json:"question"`
	Sources       []Source `json:"sources"`
}
type Actor struct{ UID, Username, Key string }
type Result struct {
	Summary string   `json:"summary"`
	Sources []Source `json:"sources"`
	Tool    string   `json:"tool"`
	RunID   string   `json:"run_id"`
}
type Receipt struct {
	Request
	RequestID string    `json:"request_id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Results   []Result  `json:"results"`
}
type Store interface {
	Create(context.Context, Request, Actor) (Receipt, error)
	Read(context.Context, string, Scope) (Receipt, error)
}

func ValidID(s string) bool {
	id, e := uuid.Parse(s)
	return e == nil && id != uuid.Nil && id.String() == s
}
func ValidateScope(s Scope) error {
	if _, e := caseidentity.ParseMode(string(s.Mode)); e != nil {
		return e
	}
	if !ValidID(s.MatterID) || !ValidID(s.CourtCaseID) {
		return errors.New("scope requires canonical non-nil UUIDs")
	}
	return nil
}
func Validate(r Request) error {
	if e := ValidateScope(r.Scope); e != nil {
		return e
	}
	if !ValidID(r.LegalMatterID) || !ValidID(r.ClaimID) || !ValidID(r.FollowupID) {
		return errors.New("legal correlation requires canonical non-nil UUIDs")
	}
	if strings.TrimSpace(r.Question) == "" || !utf8.ValidString(r.Question) || utf8.RuneCountInString(r.Question) > 5000 {
		return errors.New("question must contain 1 through 5000 characters")
	}
	if len(r.Sources) > MaxSources {
		return errors.New("too many investigation sources")
	}
	seen := map[string]bool{}
	for _, s := range r.Sources {
		if (s.Kind != "entity" && s.Kind != "event") || !ValidID(s.RecordID) || strings.TrimSpace(s.RecordVersion) == "" || len(s.RecordVersion) > 1024 {
			return errors.New("invalid investigation source")
		}
		key := s.Kind + ":" + s.RecordID
		if seen[key] {
			return errors.New("duplicate investigation source")
		}
		seen[key] = true
	}
	return nil
}
func ValidateActor(a Actor) error {
	if strings.TrimSpace(a.UID) == "" || len(a.UID) > 200 || strings.TrimSpace(a.Username) == "" || len(a.Username) > 200 || !ValidID(a.Key) {
		return errors.New("authenticated actor and UUID Idempotency-Key required")
	}
	return nil
}
func CanonicalPayload(r Request) []byte {
	if r.Sources == nil {
		r.Sources = []Source{}
	}
	raw, _ := json.Marshal(r)
	return raw
}

// ValidateTransition bounds future executor receipts. No executor is mounted by
// this package; callers must supply the expected durable state for atomic CAS.
func ValidateTransition(from, to string, results []Result) error {
	allowed := from == "received" && (to == "running" || to == "failed" || to == "cancelled") || from == "running" && (to == "completed" || to == "failed" || to == "cancelled")
	if !allowed {
		return ErrConflict
	}
	if len(results) > 100 {
		return errors.New("too many investigation results")
	}
	for _, r := range results {
		if utf8.RuneCountInString(r.Summary) > 5000 || len(r.Tool) > 250 || len(r.RunID) > 250 || len(r.Sources) > MaxSources {
			return errors.New("investigation result exceeds bounds")
		}
		for _, s := range r.Sources {
			if (s.Kind != "entity" && s.Kind != "event") || !ValidID(s.RecordID) || strings.TrimSpace(s.RecordVersion) == "" || len(s.RecordVersion) > 1024 {
				return errors.New("invalid result source")
			}
		}
	}
	raw, _ := json.Marshal(results)
	if len(raw) > 262144 {
		return errors.New("investigation results exceed byte limit")
	}
	return nil
}

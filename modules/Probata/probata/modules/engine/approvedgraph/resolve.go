// Byline: Codex · GPT-6 · 2026-10-07
// Package approvedgraph resolves an exact committed candidate revision before projection.
package approvedgraph

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/extraction/entities"
	"github.com/Cursedpotential/probata/engine/extraction/events"
)

// Scope names the existing owner-reviewed commit and its pinned source.
type Scope struct {
	ReceiptID       string
	MatterID        string
	CourtCaseID     string
	MatterMode      string
	PreviewHandle   string
	GenerationID    string
	SourceVersionID string
}

// Receipt is the immutable completed extraction commit read from working.extraction_run.
type Receipt struct {
	ID         string
	Status     string
	Outcome    string
	Summary    string
	Digest     string
	MatterMode string
	Actor      entities.Actor
	FinishedAt time.Time
}

// Entity is an approved candidate row with its persisted content hash and promotion.
type Entity struct {
	Proposal   entities.Proposal
	ContentHex string
	PromotedAt time.Time
}

// Event is an approved candidate event row with its persisted content hash and promotion.
type Event struct {
	Proposal   events.Proposal
	ContentHex string
	PromotedAt time.Time
}

// Revision contains exactly the candidates promoted by one completed commit.
type Revision struct {
	Scope    Scope
	Receipt  Receipt
	Source   SourcePin
	Records  map[string]RecordPin
	Entities []Entity
	Events   []Event
}

// SourcePin identifies the retained original object behind the reviewed source version.
type SourcePin struct {
	SourceID  string
	ObjectID  string
	ObjectURI string
	SHA256    string
}

// RecordPin is canonical metadata for one record of the pinned source generation.
type RecordPin struct {
	ID                  string
	SourceVersionID     string
	SHA256              string
	OccurredAt          *time.Time
	SourceAvailableFrom *time.Time
}

// Reader reads the existing commit receipt and its promoted candidate rows.
// Inputs: a receipt ID and its exact promotion timestamp. Outputs: persisted rows.
// Effects: read-only. Pick this over a current-candidates listing for historical revisions.
type Reader interface {
	ReadReceipt(context.Context, string) (Receipt, error)
	ReadSourcePin(context.Context, Scope) (SourcePin, error)
	ReadRecordPin(context.Context, Scope, string) (RecordPin, error)
	ReadPromotedEntities(context.Context, time.Time) ([]Entity, error)
	ReadPromotedEvents(context.Context, time.Time) ([]Event, error)
}

// Resolve verifies the stored candidate hashes and the whole committed-set digest.
// Inputs: pinned scope and a read-only reader. Outputs: a verified revision.
// Effects: read-only. Pick this before any graph projection or approved-context query.
func Resolve(ctx context.Context, reader Reader, scope Scope) (Revision, error) {
	if reader == nil || scope.ReceiptID == "" || scope.MatterID == "" || scope.CourtCaseID == "" || scope.MatterMode != "LIVE" || scope.PreviewHandle == "" || scope.GenerationID == "" || scope.SourceVersionID == "" {
		return Revision{}, errors.New("approved graph: complete receipt and source scope required")
	}
	receipt, err := reader.ReadReceipt(ctx, scope.ReceiptID)
	if err != nil {
		return Revision{}, err
	}
	if receipt.ID != scope.ReceiptID || receipt.Status != "completed" || receipt.Outcome != "committed" || receipt.MatterMode != scope.MatterMode || receipt.FinishedAt.IsZero() || receipt.Actor.Username == "" || receipt.Actor.SubjectUID == "" || !validDigest(receipt.Digest) {
		return Revision{}, errors.New("approved graph: receipt is not a completed actor-bound commit")
	}
	wantSummary := "preview:" + scope.PreviewHandle + " generation:" + scope.GenerationID + " source:" + scope.SourceVersionID
	if receipt.Summary != wantSummary {
		return Revision{}, errors.New("approved graph: receipt source scope differs")
	}
	source, err := reader.ReadSourcePin(ctx, scope)
	if err != nil {
		return Revision{}, err
	}
	if source.SourceID == "" || source.ObjectID == "" || source.ObjectURI == "" || !validDigest(source.SHA256) {
		return Revision{}, errors.New("approved graph: retained original source hash and locator required")
	}
	entitiesRows, err := reader.ReadPromotedEntities(ctx, receipt.FinishedAt)
	if err != nil {
		return Revision{}, err
	}
	eventRows, err := reader.ReadPromotedEvents(ctx, receipt.FinishedAt)
	if err != nil {
		return Revision{}, err
	}
	parts := make([]string, 0, len(entitiesRows)+len(eventRows))
	for _, row := range entitiesRows {
		p := row.Proposal
		if !row.PromotedAt.Equal(receipt.FinishedAt) || p.CandidateID == "" || p.PromotedToID == "" || p.ReviewState != "approved" || p.GenerationID != scope.GenerationID || p.SourceVersionID != scope.SourceVersionID || p.PreviewHandle != scope.PreviewHandle || !validDigest(row.ContentHex) || p.ContentHex() != row.ContentHex {
			return Revision{}, fmt.Errorf("approved graph: entity %s has an invalid promotion, source pin, or content hash", p.CandidateID)
		}
		parts = append(parts, "e:"+p.CandidateID+":"+row.ContentHex)
	}
	for _, row := range eventRows {
		p := row.Proposal
		if !row.PromotedAt.Equal(receipt.FinishedAt) || p.CandidateID == "" || p.PromotedToID == "" || p.ReviewState != "approved" || p.GenerationID != scope.GenerationID || p.SourceVersionID != scope.SourceVersionID || p.PreviewHandle != scope.PreviewHandle || !validDigest(row.ContentHex) {
			return Revision{}, fmt.Errorf("approved graph: event %s has an invalid promotion or source pin", p.CandidateID)
		}
		p.ReviewState = "pending"
		pending := p.ContentHex() == row.ContentHex
		p.ReviewState = ""
		if !pending && p.ContentHex() != row.ContentHex {
			return Revision{}, fmt.Errorf("approved graph: event %s cannot reproduce its preapproval content hash", p.CandidateID)
		}
		parts = append(parts, "v:"+p.CandidateID+":"+row.ContentHex)
	}
	sort.Strings(parts)
	sum := sha256.Sum256([]byte(scope.PreviewHandle + "\n" + scope.GenerationID + "\n" + strings.Join(parts, "\n")))
	if hex.EncodeToString(sum[:]) != receipt.Digest {
		return Revision{}, errors.New("approved graph: promoted set differs from commit digest")
	}
	records := make(map[string]RecordPin)
	readRecord := func(id string) error {
		if id == "" {
			return errors.New("approved graph: empty record locator")
		}
		if _, exists := records[id]; exists {
			return nil
		}
		pin, err := reader.ReadRecordPin(ctx, scope, id)
		if err != nil {
			return err
		}
		if pin.ID != id || pin.SourceVersionID != scope.SourceVersionID || !validDigest(pin.SHA256) || pin.SourceAvailableFrom == nil {
			return fmt.Errorf("approved graph: record %s has no canonical source or availability pin", id)
		}
		records[id] = pin
		return nil
	}
	for _, row := range entitiesRows {
		if len(row.Proposal.ModelMentions) == 0 {
			return Revision{}, fmt.Errorf("approved graph: entity %s has no hash-bound model mention", row.Proposal.CandidateID)
		}
		for _, mention := range row.Proposal.ModelMentions {
			if err := readRecord(mention.RecordID); err != nil {
				return Revision{}, err
			}
		}
	}
	for _, row := range eventRows {
		if len(row.Proposal.SourceRecords) == 0 {
			return Revision{}, fmt.Errorf("approved graph: event %s has no hash-bound source record", row.Proposal.CandidateID)
		}
		for _, record := range row.Proposal.SourceRecords {
			if err := readRecord(record.RecordID); err != nil {
				return Revision{}, err
			}
		}
	}
	return Revision{Scope: scope, Receipt: receipt, Source: source, Records: records, Entities: entitiesRows, Events: eventRows}, nil
}

// validDigest checks the existing SHA-256 hex encoding used by commitcheck.
func validDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

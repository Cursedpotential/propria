// Byline: Codex · GPT-6 · 2026-10-07
package approvedgraph

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Cursedpotential/probata/engine/extraction/entities"
	"github.com/Cursedpotential/probata/engine/extraction/events"
)

type recordFixture struct {
	receipt Receipt
	source  SourcePin
	events  []Event
	record  RecordPin
}

func (f recordFixture) ReadReceipt(context.Context, string) (Receipt, error)    { return f.receipt, nil }
func (f recordFixture) ReadSourcePin(context.Context, Scope) (SourcePin, error) { return f.source, nil }
func (f recordFixture) ReadPromotedEntities(context.Context, time.Time) ([]Entity, error) {
	return nil, nil
}
func (f recordFixture) ReadPromotedEvents(context.Context, time.Time) ([]Event, error) {
	return f.events, nil
}
func (f recordFixture) ReadRecordPin(context.Context, Scope, string) (RecordPin, error) {
	return f.record, nil
}

func testRevisionFixture() (Scope, recordFixture) {
	old := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)
	approved := time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC)
	scope := Scope{ReceiptID: "receipt", MatterID: "matter", CourtCaseID: "case", MatterMode: "LIVE", PreviewHandle: "preview", GenerationID: "generation", SourceVersionID: "source-version"}
	p := events.Proposal{CandidateID: "candidate", ReviewState: "", Title: "Bounded account", EventType: "other",
		GenerationID: scope.GenerationID, SourceVersionID: scope.SourceVersionID, PreviewHandle: scope.PreviewHandle,
		OccurredAt: &old, SourceRecords: []events.SourceRecord{{RecordID: "record"}}}
	hash := p.ContentHex()
	p.ReviewState, p.PromotedToID = "approved", "promoted"
	parts := "v:" + p.CandidateID + ":" + hash
	digest := sha256.Sum256([]byte(scope.PreviewHandle + "\n" + scope.GenerationID + "\n" + parts))
	fixture := recordFixture{receipt: Receipt{ID: scope.ReceiptID, Status: "completed", Outcome: "committed", MatterMode: "LIVE",
		Summary: "preview:" + scope.PreviewHandle + " generation:" + scope.GenerationID + " source:" + scope.SourceVersionID,
		Digest:  hex.EncodeToString(digest[:]), Actor: entities.Actor{SubjectUID: "owner-uid", Username: "owner"}, FinishedAt: approved},
		source: SourcePin{SourceID: "source", ObjectID: "object", ObjectURI: "r2://original", SHA256: strings.Repeat("a", 64)},
		events: []Event{{Proposal: p, ContentHex: hash, PromotedAt: approved}},
		record: RecordPin{ID: "record", SourceVersionID: scope.SourceVersionID, SHA256: strings.Repeat("b", 64), OccurredAt: &old, SourceAvailableFrom: &old}}
	return scope, fixture
}

func TestResolveRequiresExactReceiptSetAndCanonicalRecord(t *testing.T) {
	scope, fixture := testRevisionFixture()
	revision, err := Resolve(context.Background(), fixture, scope)
	if err != nil {
		t.Fatalf("exact fixture rejected: %v", err)
	}
	claims, err := BuildClaims(revision)
	if err != nil || len(claims) != 1 {
		t.Fatalf("cited claim: count=%d err=%v", len(claims), err)
	}
	if claims[0].SourceAvailableFrom == nil || claims[0].SourceAvailableFrom.Year() != 2020 || claims[0].ApprovedAt.Year() != 2026 || claims[0].RecordSHA256 != fixture.record.SHA256 {
		t.Fatal("historical source clock or canonical record hash was lost")
	}
	fixture.receipt.Digest = strings.Repeat("0", 64)
	if _, err := Resolve(context.Background(), fixture, scope); err == nil {
		t.Fatal("changed commit digest accepted")
	}
	scope, fixture = testRevisionFixture()
	fixture.events[0].Proposal.Title = "Altered account"
	if _, err := Resolve(context.Background(), fixture, scope); err == nil {
		t.Fatal("changed candidate content accepted")
	}
	scope, fixture = testRevisionFixture()
	fixture.source.SHA256 = "unbound"
	if _, err := Resolve(context.Background(), fixture, scope); err == nil {
		t.Fatal("invalid root source hash accepted")
	}
	scope, fixture = testRevisionFixture()
	fixture.record.SourceVersionID = "other-source"
	if _, err := Resolve(context.Background(), fixture, scope); err == nil {
		t.Fatal("foreign record pin accepted")
	}
	scope, fixture = testRevisionFixture()
	fixture.record.SourceAvailableFrom = nil
	revision, err = Resolve(context.Background(), fixture, scope)
	if err != nil {
		t.Fatalf("unknown availability lost exact approved source pin: %v", err)
	}
	claims, err = BuildClaims(revision)
	if err != nil || len(claims) != 1 || claims[0].SourceAvailableFrom != nil {
		t.Fatal("unknown availability must remain explicit in approved claim")
	}
	raw, err := json.Marshal(claims[0])
	if err != nil || !strings.Contains(string(raw), `"source_available_from":null`) {
		t.Fatal("unknown availability must be encoded explicitly without a guessed timestamp")
	}
}

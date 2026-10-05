// Byline: Codex · GPT-5 · 2026-10-05 (single-case operating contract)
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/investigation"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"strings"
	"testing"
	"time"
)

const investigationTestID = "11111111-1111-4111-8111-111111111111"

type investigationRow struct{ scan func(...any) error }

func (r investigationRow) Scan(d ...any) error { return r.scan(d...) }

type investigationTx struct {
	pgx.Tx
	stored                    *investigation.Receipt
	hash                      string
	commits, inserts, sources int
	eventCount                int
}

func (x *investigationTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag("UPDATE 1"), nil
}
func (x *investigationTx) Commit(context.Context) error   { x.commits++; return nil }
func (x *investigationTx) Rollback(context.Context) error { return nil }
func (x *investigationTx) Query(_ context.Context, q string, _ ...any) (pgx.Rows, error) {
	x.sources++
	return &legalEventRows{count: x.eventCount}, nil
}
func (x *investigationTx) QueryRow(_ context.Context, q string, args ...any) pgx.Row {
	return investigationRow{func(d ...any) error {
		if q == caseCourtCaseSQL {
			*(d[0].(*string)) = authoritativeCourtCaseID
			*(d[1].(*string)) = authoritativeMatterID
			return nil
		}
		if strings.HasPrefix(q, "INSERT INTO") {
			x.inserts++
			var request investigation.Request
			_ = json.Unmarshal(args[11].([]byte), &request)
			x.hash = args[10].(string)
			x.stored = &investigation.Receipt{Request: request, RequestID: args[0].(string), Status: "received", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(), Results: []investigation.Result{}}
		}
		if x.stored == nil {
			return pgx.ErrNoRows
		}
		raw, _ := json.Marshal(x.stored.Request)
		*(d[0].(*string)) = x.stored.RequestID
		*(d[1].(*[]byte)) = raw
		*(d[2].(*string)) = x.stored.Status
		*(d[3].(*time.Time)) = x.stored.CreatedAt
		*(d[4].(*time.Time)) = x.stored.UpdatedAt
		*(d[5].(*[]byte)) = []byte("[]")
		*(d[6].(*string)) = x.hash
		return nil
	}}
}

type investigationDB struct{ *investigationTx }

func (d investigationDB) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	return d.investigationTx, nil
}
func investigationInput() investigation.Request {
	return investigation.Request{Scope: investigation.Scope{Mode: caseidentity.ModeLive, MatterID: authoritativeMatterID, CourtCaseID: authoritativeCourtCaseID}, LegalMatterID: investigationTestID, ClaimID: investigationTestID, FollowupID: investigationTestID, Question: "Investigate", Sources: []investigation.Source{}}
}
func TestInvestigationStoreReceiptReplayConflictAndScope(t *testing.T) {
	tx := &investigationTx{}
	s, e := NewInvestigationRequestStore(investigationDB{tx})
	if e != nil {
		t.Fatal(e)
	}
	r := investigationInput()
	a := investigation.Actor{UID: "a", Username: "actor", Key: investigationTestID}
	first, e := s.Create(context.Background(), r, a)
	if e != nil || first.Status != "received" || tx.inserts != 1 || tx.commits != 1 {
		t.Fatalf("admission: %v %+v", e, first)
	}
	replay, e := s.Create(context.Background(), r, a)
	if e != nil || replay.RequestID != first.RequestID || tx.inserts != 1 {
		t.Fatal("replay lost durable receipt", e)
	}
	r.Question = "altered"
	if _, e = s.Create(context.Background(), r, a); !errors.Is(e, investigation.ErrConflict) {
		t.Fatal("changed replay accepted", e)
	}
	r = investigationInput()
	r.MatterID = investigationTestID
	if _, e = s.Create(context.Background(), r, a); !errors.Is(e, investigation.ErrScope) {
		t.Fatal("cross-scope accepted", e)
	}
	scope := first.Scope
	scope.CourtCaseID = "22222222-2222-4222-8222-222222222222"
	if _, e = s.Read(context.Background(), first.RequestID, scope); !errors.Is(e, investigation.ErrScope) {
		t.Fatal("cross-scope read accepted", e)
	}
}
func TestInvestigationStoreRejectsMissingNativeSource(t *testing.T) {
	tx := &investigationTx{}
	s, _ := NewInvestigationRequestStore(investigationDB{tx})
	r := investigationInput()
	r.Sources = []investigation.Source{{Kind: "event", RecordID: investigationTestID, RecordVersion: "native-version"}}
	_, e := s.Create(context.Background(), r, investigation.Actor{UID: "a", Username: "a", Key: investigationTestID})
	if !errors.Is(e, investigation.ErrSource) || tx.inserts != 0 || tx.sources != 1 {
		t.Fatalf("missing source admitted: %v", e)
	}
}

func TestInvestigationNativeVersionAndReplayAfterSourceChanges(t *testing.T) {
	tx := &investigationTx{eventCount: 1}
	store, _ := NewInvestigationRequestStore(investigationDB{tx})
	r := investigationInput()
	r.Sources = []investigation.Source{{Kind: "event", RecordID: investigationTestID, RecordVersion: "wrong-version"}}
	a := investigation.Actor{UID: "a", Username: "a", Key: investigationTestID}
	if _, e := store.Create(context.Background(), r, a); !errors.Is(e, investigation.ErrSource) {
		t.Fatal("stale native version admitted", e)
	}
	r.Sources[0].RecordVersion = "generation#native-version"
	first, e := store.Create(context.Background(), r, a)
	if e != nil {
		t.Fatal(e)
	}
	queries := tx.sources
	tx.eventCount = 0
	replay, e := store.Create(context.Background(), r, a)
	if e != nil || replay.RequestID != first.RequestID || queries != tx.sources {
		t.Fatal("accepted replay incorrectly rechecked source freshness", e)
	}
}

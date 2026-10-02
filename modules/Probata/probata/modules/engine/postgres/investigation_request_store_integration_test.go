package postgres

import (
	"context"
	"errors"
	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/investigation"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestInvestigationPostgresIsolated is opt-in and deliberately retains synthetic
// receipts in the named isolated database for independent proof/readback.
// Bootstrap canonical schema and TEST case seed before running this test.
func TestInvestigationPostgresIsolated(t *testing.T) {
	dsn := os.Getenv("PROBATA_INVESTIGATION_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set PROBATA_INVESTIGATION_TEST_DATABASE_URL for isolated PostgreSQL proof")
	}
	parsed, e := url.Parse(dsn)
	if e != nil || os.Getenv("PROBATA_INVESTIGATION_TEST_ALLOW_ISOLATED") != "1" || !strings.HasPrefix(strings.TrimPrefix(parsed.Path, "/"), "investigation_test_") {
		t.Fatal("explicit isolated database authorization required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	pool, e := pgxpool.New(ctx, dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer pool.Close()
	identity, _ := NewCaseIdentityStore(pool)
	view, e := identity.Read(ctx, caseidentity.ModeTest)
	if e != nil || view.Matter == nil || view.CourtCase == nil {
		t.Fatalf("canonical TEST case fixture required: %v", e)
	}
	store, _ := NewInvestigationRequestStore(pool)
	r := investigation.Request{Scope: investigation.Scope{Mode: caseidentity.ModeTest, MatterID: view.Matter.ID, CourtCaseID: view.CourtCase.ID}, LegalMatterID: uuid.NewString(), ClaimID: uuid.NewString(), FollowupID: uuid.NewString(), Question: "Synthetic isolated investigation receipt proof", Sources: []investigation.Source{}}
	actor := investigation.Actor{UID: "synthetic-isolated-proof", Username: "synthetic-isolated-proof", Key: uuid.NewString()}
	first, e := store.Create(ctx, r, actor)
	if e != nil {
		t.Fatal(e)
	}
	if first.Status != "received" || len(first.Results) != 0 {
		t.Fatal("receipt incorrectly implies execution")
	}
	var wg sync.WaitGroup
	failures := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, e := store.Create(ctx, r, actor)
			if e == nil && got.RequestID != first.RequestID {
				e = errors.New("retry changed receipt")
			}
			failures <- e
		}()
	}
	wg.Wait()
	close(failures)
	for e := range failures {
		if e != nil {
			t.Fatal(e)
		}
	}
	another := actor
	another.Key = uuid.NewString()
	another.UID = "another-synthetic-actor"
	again, e := store.Create(ctx, r, another)
	if e != nil || again.RequestID != first.RequestID {
		t.Fatalf("logical duplicate: %v", e)
	}
	changed := r
	changed.Question = "changed"
	if _, e = store.Create(ctx, changed, another); !errors.Is(e, investigation.ErrConflict) {
		t.Fatal("changed alias replay accepted", e)
	}
	changed = r
	changed.ClaimID = uuid.NewString()
	if _, e = store.Create(ctx, changed, another); !errors.Is(e, investigation.ErrConflict) {
		t.Fatal("alias key reused for different correlation", e)
	}
	read, e := store.Read(ctx, first.RequestID, r.Scope)
	if e != nil || read.RequestID != first.RequestID {
		t.Fatal("readback failed", e)
	}
	badScope := r.Scope
	badScope.MatterID = uuid.NewString()
	if _, e = store.Read(ctx, first.RequestID, badScope); !errors.Is(e, investigation.ErrScope) {
		t.Fatal("cross scope read accepted", e)
	}
	missing := r
	missing.FollowupID = uuid.NewString()
	missing.Sources = []investigation.Source{{Kind: "event", RecordID: uuid.NewString(), RecordVersion: "missing-native-version"}}
	actor.Key = uuid.NewString()
	if _, e = store.Create(ctx, missing, actor); !errors.Is(e, investigation.ErrSource) {
		t.Fatal("missing source admitted", e)
	}
	var count int
	e = pool.QueryRow(ctx, `SELECT count(*) FROM ops.legal_investigation_request WHERE mode=$1 AND matter_id=$2::uuid AND court_case_id=$3::uuid AND legal_matter_id=$4::uuid AND claim_id=$5::uuid AND followup_id=$6::uuid`, r.Mode, r.MatterID, r.CourtCaseID, r.LegalMatterID, r.ClaimID, r.FollowupID).Scan(&count)
	if e != nil || count != 1 {
		t.Fatalf("actual relation count %d: %v", count, e)
	}
	t.Logf("retained synthetic receipt request_id=%s mode=%s matter_id=%s court_case_id=%s concurrent_retries=8 canonical_relation_count=%d status=%s", first.RequestID, r.Mode, r.MatterID, r.CourtCaseID, count, first.Status)
}

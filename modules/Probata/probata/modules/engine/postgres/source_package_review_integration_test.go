package postgres

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/sourcemeta"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestPackageReviewDurableTransitions exercises real PostgreSQL correction transactions in an isolated database.
// Inputs: PACKAGE_REVIEW_TEST_DSN pointing at a new backend_package_review_* database on the existing VPS.
// Outputs: cumulative edits, idempotent receipts, stale refusal, accepted-only commit and immutable history assertions.
// Effects: synthetic fixtures only in that named disposable database; never connects to the production database by default.
func TestPackageReviewDurableTransitions(t *testing.T) {
	dsn := os.Getenv("PACKAGE_REVIEW_TEST_DSN")
	if dsn == "" {
		t.Skip("isolated VPS database DSN not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var db string
	if err = pool.QueryRow(ctx, "SELECT current_database()").Scan(&db); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(db, "backend_package_review_") {
		t.Fatal("test refuses any non-disposable database")
	}
	fixture, err := os.ReadFile("source_package_review_test.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, string(fixture)); err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile("source_package_review.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, string(migration)); err != nil {
		t.Fatal(err)
	}
	store, err := NewSourceMetadataStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	spec := packageReviewFixture()
	first, err := store.PersistPackageReview(ctx, spec)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := store.PersistPackageReview(ctx, spec)
	if err != nil || replay.CorrectionRef != first.CorrectionRef {
		t.Fatalf("replay %#v %v", replay, err)
	}
	spec.Revision = first.CorrectionRef
	spec.IdempotencyKey = "click-2"
	spec.Fields = map[string]any{"confidence": 0.8}
	second, err := store.PersistPackageReview(ctx, spec)
	if err != nil {
		t.Fatal(err)
	}
	stale := spec
	stale.IdempotencyKey = "stale-click"
	if _, err = store.PersistPackageReview(ctx, stale); !errors.Is(err, sourcemeta.ErrStaleRevision) {
		t.Fatalf("stale write = %v", err)
	}
	spec.Revision = second.CorrectionRef
	spec.IdempotencyKey = "early-commit"
	spec.Operation = "commit"
	spec.Fields = nil
	if _, err = store.PersistPackageReview(ctx, spec); !errors.Is(err, sourcemeta.ErrStaleRevision) {
		t.Fatalf("unaccepted commit = %v", err)
	}
	spec.Operation = "accept"
	spec.IdempotencyKey = "accept-click"
	accepted, err := store.PersistPackageReview(ctx, spec)
	if err != nil {
		t.Fatal(err)
	}
	spec.Revision = accepted.CorrectionRef
	spec.Operation = "commit"
	spec.IdempotencyKey = "commit-click"
	committed, err := store.PersistPackageReview(ctx, spec)
	if err != nil {
		t.Fatal(err)
	}
	if committed.Revision != 4 {
		t.Fatalf("revision = %d", committed.Revision)
	}
	var count int
	var state, label string
	var confidence float64
	err = pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM context.source_metadata_correction),corrected_value->>'review_state',corrected_value->'fields'->>'label',(corrected_value->'fields'->>'confidence')::float8 FROM context.source_metadata_correction WHERE correction_ref=$1::uuid`, committed.CorrectionRef).Scan(&count, &state, &label, &confidence)
	if err != nil || count != 4 || state != "committed" || label != "Corrected" || confidence != 0.8 {
		t.Fatalf("durable result %d %s %s %f %v", count, state, label, confidence, err)
	}
	if _, err = pool.Exec(ctx, "UPDATE context.source_metadata_correction SET change_reason='overwrite'"); err == nil {
		t.Fatal("append-only correction mutated")
	}
	t.Run("native_canonical_candidate", func(t *testing.T) { testNativePackageCandidate(t, pool) })
}

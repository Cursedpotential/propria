// Byline: Codex · GPT-6.1 · 2026-10-06. Selective port from held 7a1db624; independent byte-only operation.
package postgres

// Byline: Codex · 2026-10-04. Live tests guard the dedicated synthetic database and retain all fixtures.
import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/sourceintegrity"
	"github.com/Cursedpotential/probata/engine/stagegraph"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.temporal.io/sdk/temporal"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// integrityFixtureOpener exposes synthetic bytes or controlled failure to the production adapter.
// Input: retained reference. Output: test stream; side effects: source-open count only.
type integrityFixtureOpener struct {
	body  []byte
	err   error
	opens int
}

// OpenOriginal supplies only the configured synthetic stream for SQL receipt tests.
// Input: ref. Output: read-only stream/error; side effects: test counter only.
func (o *integrityFixtureOpener) OpenOriginal(context.Context, proffer.Ref) (io.ReadCloser, error) {
	o.opens++
	if o.err != nil {
		return nil, o.err
	}
	return io.NopCloser(bytes.NewReader(o.body)), nil
}

// TestSourceIntegrityStoreRejectsUnsupportedEvidence rejects format/canonical conclusions from byte coverage.
// Inputs: synthetic assessments. Outputs: validation assertions. Effects: memory only;
// choose to enforce the persisted evidence contract without a database.
func TestSourceIntegrityStoreRejectsUnsupportedEvidence(t *testing.T) {
	if _, err := NewSourceIntegrityStore(nil, &integrityFixtureOpener{}); err == nil {
		t.Fatal("nil database accepted")
	}
	a, _ := sourceintegrity.InspectStream(context.Background(), bytes.NewReader([]byte{1}), 1)
	if err := validateIntegrityAssessment(a, 1); err != nil {
		t.Fatal(err)
	}
	a.Status = "verified_good"
	if validateIntegrityAssessment(a, 1) == nil {
		t.Fatal("byte positivity promoted to good")
	}
	a.Status = "all_zero"
	if validateIntegrityAssessment(a, 1) == nil {
		t.Fatal("nonzero all_zero accepted")
	}
	a = sourceintegrity.Unchecked(1)
	a.Reason = "source body text"
	if validateIntegrityAssessment(a, 1) == nil {
		t.Fatal("arbitrary error body accepted")
	}
	a = sourceintegrity.Unchecked(1)
	a.Complete = true
	if validateIntegrityAssessment(a, 1) == nil {
		t.Fatal("missing EOF accepted")
	}
}

// TestSourceIntegrityCoordinateExactIdentity preserves distinct operation IDs and rejects irrelevant legacy slots.
// Inputs: synthetic UUID references and operation IDs. Outputs: exact key and rejection assertions.
// Effects: memory only; choose to prevent normalization from merging distinct immutable operations.
func TestSourceIntegrityCoordinateExactIdentity(t *testing.T) {
	req := proffer.StageRequest{RequestID: "source-job", SourceVersionRef: proffer.Ref(uuid.NewString()), Refs: map[string]proffer.Ref{"original": proffer.Ref(uuid.NewString()), "integrity_operation": "operation"}}
	_, _, first, err := integrityCoordinate(req)
	if err != nil {
		t.Fatal(err)
	}
	req.Refs["integrity_operation"] = " operation "
	_, _, second, err := integrityCoordinate(req)
	if err != nil || first == second || second != "source-integrity: operation :"+string(req.Refs["original"])+":"+sourceintegrity.CheckVersion {
		t.Fatal("distinct exact operation identity was normalized")
	}
	for _, field := range []string{"matter", "court-case"} {
		invalid := req
		if field == "matter" {
			invalid.MatterID = "unsupported"
		} else {
			invalid.CourtCaseID = "unsupported"
		}
		if _, _, _, err := integrityCoordinate(invalid); err == nil {
			t.Fatal("unsupported legacy slot accepted")
		}
	}
}

// TestSourceIntegrityStoreLiveReceiptsAndIdempotency verifies append-only receipts in the existing dedicated fixture database.
// Inputs: explicit test DSN and synthetic originals. Outputs: retained receipt IDs and replay assertions.
// Effects: synthetic inserts only; choose after separate principal admission, never against production or to create schema.
func TestSourceIntegrityStoreLiveReceiptsAndIdempotency(t *testing.T) {
	dsn := os.Getenv("SOURCE_INTEGRITY_TEST_DATABASE_URL")
	if file := os.Getenv("SOURCE_INTEGRITY_TEST_DATABASE_URL_FILE"); file != "" {
		secret, err := os.ReadFile(file)
		if err != nil {
			t.Fatal("private synthetic database URL file unavailable")
		}
		dsn = strings.TrimSpace(string(secret))
	}
	if dsn == "" {
		t.Skip("dedicated synthetic database URL not supplied")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var db string
	if err = pool.QueryRow(ctx, "SELECT current_database()").Scan(&db); err != nil || db != "cb_integrity_temporal_20261004" {
		t.Fatalf("refusing fixture writes outside dedicated test database: %q %v", db, err)
	}
	// Existing retained fixture only: never create, replace, grant or alter its objects.
	var admitted bool
	if err = pool.QueryRow(ctx, `SELECT has_schema_privilege(current_user,'context','USAGE') AND
	 has_any_column_privilege(current_user,'context.source_version','UPDATE') AND
	 (SELECT bool_and(has_table_privilege(current_user,'context.'||name,'SELECT') AND
	 has_table_privilege(current_user,'context.'||name,'INSERT')) FROM unnest(
	 ARRAY['retained_object','source_version','activity_execution','activity_receipt']) AS name)`).Scan(&admitted); err != nil || !admitted {
		t.Fatalf("existing synthetic fixture admission failed: %v", err)
	}
	var guarded bool
	if err = pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='integrity_fixture_receipt_append_only' AND tgrelid='context.activity_receipt'::regclass AND NOT tgisinternal)`).Scan(&guarded); err != nil || !guarded {
		t.Fatalf("existing synthetic append-only guard absent: %v", err)
	}
	for _, tc := range []struct {
		name     string
		body     []byte
		expected int64
		failure  bool
		cancel   bool
	}{{"zero-byte", nil, 0, false, false}, {"all-zero", make([]byte, 65539), 65539, false, false}, {"nonzero-unassessed", []byte{0, 1}, 2, false, false}, {"short-read", []byte{0}, 4, false, false}, {"open-failure", nil, 4, true, false}, {"canceled", nil, 4, false, true}} {
		t.Run(tc.name, func(t *testing.T) {
			source, original := uuid.New(), uuid.New()
			origin := "synthetic-integrity-source-" + source.String()
			op := "synthetic-integrity-job-" + uuid.NewString()
			_, err := pool.Exec(ctx, `INSERT INTO context.retained_object VALUES($1,'inline','synthetic:source-integrity',$2,$3)`, original, tc.body, tc.expected)
			if err != nil {
				t.Fatal(err)
			}
			_, err = pool.Exec(ctx, `INSERT INTO context.source_version VALUES($1,$2,'retained','synthetic',$3)`, source, origin, original)
			if err != nil {
				t.Fatal(err)
			}
			req := proffer.StageRequest{RequestID: origin, SourceVersionRef: proffer.Ref(source.String()), DeclaredFormat: "synthetic", Refs: map[string]proffer.Ref{"original": proffer.Ref(original.String()), "integrity_operation": proffer.Ref(op)}}
			opener := &integrityFixtureOpener{body: tc.body}
			if tc.failure {
				opener.err = errors.New("synthetic source access failure")
			}
			store, err := NewSourceIntegrityStore(pool, opener)
			if err != nil {
				t.Fatal(err)
			}
			observed := time.Now().UTC().Add(-5 * time.Second).Truncate(time.Microsecond)
			a := activities.SourceIntegrityActivities{Store: store, Execution: func(context.Context) (string, string) { return op, "synthetic-run" }, RuntimeMetadata: func(context.Context) (time.Time, string, string) {
				return observed, "synthetic-activity", "synthetic-vps-host"
			}}
			runCtx := ctx
			if tc.cancel {
				var cancel context.CancelFunc
				runCtx, cancel = context.WithCancel(ctx)
				cancel()
			}
			result, err := a.AssessSourceIntegrity(runCtx, req)
			if tc.cancel {
				if !temporal.IsCanceledError(err) {
					t.Fatal(err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if result.ReceiptRef == "" {
				t.Fatal("missing durable receipt")
			}
			var payload []byte
			var status string
			var started, completed time.Time
			err = pool.QueryRow(ctx, `SELECT status,COALESCE(result_ref,error_detail),started_at,completed_at FROM context.activity_receipt WHERE id=$1::uuid`, string(result.ReceiptRef)).Scan(&status, &payload, &started, &completed)
			if err != nil {
				t.Fatal(err)
			}
			var evidence integrityEvidence
			if json.Unmarshal(payload, &evidence) != nil {
				t.Fatal("invalid receipt JSON")
			}
			if !started.Equal(observed) || !completed.After(started) || evidence.ActivityID != "synthetic-activity" || evidence.ExecutionHost != "synthetic-vps-host" {
				t.Fatalf("time/runtime correlation lost: %v %v %+v", started, completed, evidence)
			}
			wantSuccess := !tc.failure && !tc.cancel && int64(len(tc.body)) == tc.expected
			if (status == "success") != wantSuccess || evidence.Assessment.Complete != wantSuccess || evidence.OperationID != op || evidence.OperationRunID != "synthetic-run" || evidence.Assessment.FormatValidation != "unassessed" {
				t.Fatalf("receipt %s %+v", status, evidence)
			}
			if tc.name == "nonzero-unassessed" && evidence.Assessment.Status != "unknown" {
				t.Fatal("nonzero promoted")
			}
			if !wantSuccess && (result.Status != proffer.StatusFailed || result.Ref != "" || evidence.Assessment.Status != "unknown") {
				t.Fatal("failed read promoted")
			}
			before := opener.opens
			again, err := a.AssessSourceIntegrity(ctx, req)
			if err != nil || again != result || opener.opens != before {
				t.Fatalf("terminal recovery %+v %v opens %d/%d", again, err, opener.opens, before)
			}
			spec := activities.SourceIntegrityReceiptSpec{Request: req, Assessment: evidence.Assessment, Attempt: 2, OperationRunID: "repeated-run"}
			again, err = store.PersistSourceIntegrity(ctx, spec)
			if err != nil || again != result {
				t.Fatalf("repeat completion %+v %v", again, err)
			}
			var count int
			if err = pool.QueryRow(ctx, `SELECT count(*) FROM context.activity_receipt r JOIN context.activity_execution e ON e.id=r.activity_execution_id WHERE e.source_version_id=$1 AND e.activity_name=$2`, source, string(stagegraph.AssessSourceIntegrity)).Scan(&count); err != nil || count != 1 {
				t.Fatalf("receipt count %d %v", count, err)
			}
			if tc.name == "nonzero-unassessed" {
				// A fresh operation with simultaneous completions must retain exactly one receipt.
				concurrent := req
				raceOperation := "synthetic-integrity-race-" + uuid.NewString()
				concurrent.Refs = map[string]proffer.Ref{"original": req.Refs["original"], "integrity_operation": proffer.Ref(raceOperation)}
				var results [2]proffer.StageResult
				var failures [2]error
				var workers sync.WaitGroup
				for i := range results {
					workers.Add(1)
					go func(i int) {
						defer workers.Done()
						results[i], failures[i] = store.PersistSourceIntegrity(ctx, activities.SourceIntegrityReceiptSpec{Request: concurrent, Assessment: evidence.Assessment, Attempt: 1, StartedAt: observed, OperationRunID: "synthetic-race-run"})
					}(i)
				}
				workers.Wait()
				if failures[0] != nil || failures[1] != nil || results[0] != results[1] || results[0].ReceiptRef == "" {
					t.Fatal("concurrent terminal receipt differs", failures)
				}
				if err = pool.QueryRow(ctx, `SELECT count(*) FROM context.activity_receipt r JOIN context.activity_execution e ON e.id=r.activity_execution_id WHERE e.source_version_id=$1 AND e.activity_name=$2`, source, string(stagegraph.AssessSourceIntegrity)).Scan(&count); err != nil || count != 2 {
					t.Fatalf("two independent operations require two immutable receipts: %d %v", count, err)
				}
				t.Logf("retained synthetic concurrent receipt %s", results[0].ReceiptRef)
			}
			wrong := req
			if _, err = pool.Exec(ctx, `UPDATE context.activity_receipt SET attempt=99 WHERE id=$1::uuid`, string(result.ReceiptRef)); err == nil {
				t.Fatal("synthetic receipt append-only guard did not refuse update")
			}
			wrong.RequestID = "wrong-source-workflow"
			if _, _, err = store.LoadSourceIntegrity(ctx, wrong); err == nil {
				t.Fatal("wrong source binding accepted")
			}
			t.Logf("retained synthetic receipt %s: status=%s assessment=%s checked=%d complete=%t", result.ReceiptRef, status, evidence.Assessment.Status, evidence.Assessment.CheckedBytes, evidence.Assessment.Complete)
		})
	}
}

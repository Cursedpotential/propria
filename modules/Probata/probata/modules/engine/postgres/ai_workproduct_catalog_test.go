// Byline: Codex · GPT-6.1 · 2026-10-05.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// aiOccurrenceTestBatch builds a pinned original occurrence without fake provider or package identifiers.
// Inputs: none. Outputs: valid synthetic batch. Effects: none; choose for repository transaction failures.
func aiOccurrenceTestBatch() activities.AIWorkproductCatalogBatch {
	p := strings.Repeat("a", 64)
	req := activities.AIWorkproductCatalogInput{Operation: "synthetic-db", ReadbackRef: "file:///synthetic/readback.json", ReadbackSHA256: p, MetadataRef: "file:///synthetic/metadata.json"}
	key := "consignatio/casevault/KnowledgeBase/ai-chats/_derived/misc/synthetic/Fixture.md"
	m := activities.AIWorkproductOccurrenceMetadata{Schema: activities.AIWorkproductCatalogSchema, Operation: req.Operation, OriginalPath: activities.AIWorkproductOccurrenceScope + "/Fixture.md", ObservedMtimeNS: 1759681000123456789, TransportObservedAt: "2026-10-05T12:00:00Z", OriginalUnchangedSizeMtime: true, EventDateStatus: "unknown", Provider: "unknown", SourceUnit: "synthetic", SourceRef: "file:///synthetic/source-01", TransportServerPath: "/synthetic/source-01", ProvenanceRef: "file:///synthetic/transport.json", ProvenanceSHA256: p, PlacementManifestRef: "file:///synthetic/manifest.json", PlacementManifestSHA256: p, ReadbackRef: req.ReadbackRef, ReadbackSHA256: p, ObjectRef: proffer.Ref("b2://salem-data/" + key + "?versionId=fixture-v1"), VersionID: "fixture-v1", SHA256: p}
	return activities.AIWorkproductCatalogBatch{Schema: activities.AIWorkproductCatalogSchema, Request: req, Rows: []activities.AIWorkproductOccurrence{{Source: activities.AIWorkproductOccurrenceSource, Scope: activities.AIWorkproductOccurrenceScope, Path: "Fixture.md", SourceID: "", Size: 100, Disposition: "copied", B2Key: key, Metadata: m}}}
}

// aiOccurrenceDB admits only narrow capability queries and one function call per repository transaction.
// Inputs: test switches. Outputs: transaction observations. Effects: memory only; choose for cancellation, isolation and uncertain commit proof.
type aiOccurrenceDB struct {
	opts      []pgx.TxOptions
	admitted  bool
	callErr   error
	commitErr error
	raw       []byte
	count     int
	calls     int
	commits   int
}

// BeginTx records independently opened isolation boundaries and respects cancellation.
// Inputs: context/options. Outputs: fixture transaction. Effects: memory observations only.
func (d *aiOccurrenceDB) BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	d.opts = append(d.opts, opts)
	return &aiOccurrenceTx{db: d, opts: opts}, nil
}

// aiOccurrenceTx rejects SQL outside explicit admission and two restricted API calls.
// Inputs: repository queries. Outputs: pgx-compatible fixture responses. Effects: observations only.
type aiOccurrenceTx struct {
	pgx.Tx
	db   *aiOccurrenceDB
	opts pgx.TxOptions
	call bool
}

// QueryRow returns only declared restricted-call results without emulating the raw table.
// Inputs: SQL/args. Outputs: scalar fixture result. Effects: function-call counter only.
func (t *aiOccurrenceTx) QueryRow(ctx context.Context, q string, args ...any) pgx.Row {
	if strings.Contains(q, "FROM pg_catalog.pg_roles") {
		return aiOccurrenceRow{value: t.db.admitted}
	}
	if q == "SELECT source_occurrence_api.register_ai_workproduct($1::text,$2::text)" {
		t.call = true
		t.db.calls++
		return aiOccurrenceRow{value: t.db.count, err: t.db.callErr}
	}
	if q == "SELECT source_occurrence_api.read_ai_workproduct($1::text,$2::text)" {
		t.call = true
		t.db.calls++
		return aiOccurrenceRow{value: t.db.raw, err: t.db.callErr}
	}
	return aiOccurrenceRow{err: errors.New("unexpected SQL")}
}

// Commit can expose an uncertain mutation outcome without pretending a successful readback.
// Inputs: context. Outputs: fixture error. Effects: commit counter only; choose for retry contract proof.
func (t *aiOccurrenceTx) Commit(ctx context.Context) error {
	t.db.commits++
	if t.call {
		return t.db.commitErr
	}
	return ctx.Err()
}

// Rollback retains all synthetic observations and performs no filesystem cleanup.
// Inputs: context. Outputs: nil. Effects: none; choose for deferred transaction termination.
func (t *aiOccurrenceTx) Rollback(context.Context) error { return nil }

// aiOccurrenceRow models the only scalar result types of the narrow API.
// Inputs: value/error. Outputs: Scan result. Effects: destination assignment only.
type aiOccurrenceRow struct {
	value any
	err   error
}

// Scan copies admission, count or bounded JSON responses without logging payloads.
// Inputs: one destination. Outputs: error. Effects: fixture destination only.
func (r aiOccurrenceRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != 1 {
		return errors.New("unexpected scan arity")
	}
	switch p := dest[0].(type) {
	case *bool:
		*p = r.value.(bool)
	case *int:
		*p = r.value.(int)
	case *[]byte:
		*p = append([]byte(nil), r.value.([]byte)...)
	default:
		return errors.New("unexpected scan type")
	}
	return nil
}

// TestAIWorkproductCatalogRepositoryTransactions proves admission, serializable registration and independent read-only readback.
// Inputs: fixture pool. Outputs: bounded API and isolation assertions. Effects: memory only.
func TestAIWorkproductCatalogRepositoryTransactions(t *testing.T) {
	b := aiOccurrenceTestBatch()
	_, pin, err := activities.AIWorkproductCatalogCanonical(b)
	if err != nil {
		t.Fatal(err)
	}
	b.Rows[0].Metadata.CatalogBatchSHA256 = pin
	raw, _ := json.Marshal(b.Rows)
	b.Rows[0].Metadata.CatalogBatchSHA256 = ""
	db := &aiOccurrenceDB{admitted: true, count: 1, raw: raw}
	repo, constructorErr := NewToolkitRecoveryCatalog(db)
	if constructorErr != nil {
		t.Fatal(constructorErr)
	}
	if err = repo.RegisterAIWorkproductOccurrences(context.Background(), b); err != nil {
		t.Fatal(err)
	}
	rows, err := repo.ReadAIWorkproductOccurrences(context.Background(), b.Request.Operation, pin)
	if err != nil || len(rows) != 1 {
		t.Fatal(err)
	}
	if len(db.opts) != 4 || db.opts[0].AccessMode != pgx.ReadOnly || db.opts[1].IsoLevel != pgx.Serializable || db.opts[2].AccessMode != pgx.ReadOnly || db.opts[3].IsoLevel != pgx.RepeatableRead || db.opts[3].AccessMode != pgx.ReadOnly || db.calls != 2 {
		t.Fatal("repository mixed admission, mutation and independent readback")
	}
}

// TestAIWorkproductCatalogRepositoryFailures proves no mutation on admission/cancellation and visible uncertain commit/collision failures.
// Inputs: fixture failure modes. Outputs: error and call-count assertions. Effects: memory only.
func TestAIWorkproductCatalogRepositoryFailures(t *testing.T) {
	b := aiOccurrenceTestBatch()
	for _, which := range []string{"admission", "cancel", "collision", "count", "uncertain-commit", "oversize-read", "unknown-field-read"} {
		t.Run(which, func(t *testing.T) {
			db := &aiOccurrenceDB{admitted: true, count: 1, raw: []byte("[]")}
			repo, constructorErr := NewToolkitRecoveryCatalog(db)
			if constructorErr != nil {
				t.Fatal(constructorErr)
			}
			ctx := context.Background()
			read := false
			switch which {
			case "admission":
				db.admitted = false
			case "cancel":
				var c context.CancelFunc
				ctx, c = context.WithCancel(ctx)
				c()
			case "collision":
				db.callErr = &pgconn.PgError{Code: "23505"}
			case "count":
				db.count = 0
			case "uncertain-commit":
				db.commitErr = errors.New("lost commit response")
			case "oversize-read":
				read = true
				db.raw = []byte(strings.Repeat(" ", 256<<10+1))
			case "unknown-field-read":
				read = true
				db.raw = []byte(`[{"unexpected":true}]`)
			}
			var err error
			if read {
				_, err = repo.ReadAIWorkproductOccurrences(ctx, b.Request.Operation, strings.Repeat("a", 64))
			} else {
				err = repo.RegisterAIWorkproductOccurrences(ctx, b)
			}
			if err == nil {
				t.Fatal("failure mode accepted")
			}
			if which == "collision" && !errors.Is(err, ErrAIWorkproductCatalogCollision) {
				t.Fatal("collision classification lost")
			}
			if (which == "admission" || which == "cancel") && db.calls != 0 {
				t.Fatal("unadmitted mutation called")
			}
		})
	}
}

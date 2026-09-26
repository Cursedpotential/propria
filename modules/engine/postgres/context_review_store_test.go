// Byline: Claude Code · Opus 5.5 · 2026-09-26
package postgres

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Cursedpotential/probata/engine/contextreview"
)

const foreshadowingTable = "record_foreshadowing_flag"

func TestAsLivedReadPlanNeverNamesTheForeshadowingTable(t *testing.T) {
	asLived := planContextReviewRead(contextreview.HorizonAsLived)
	if asLived.foreshadowing != "" {
		t.Fatal("the as-lived read plan holds a foreshadowing statement")
	}
	if strings.Contains(asLived.reviews, foreshadowingTable) || strings.Contains(contextReviewSubjectSQL, foreshadowingTable) {
		t.Fatal("an as-lived statement names the foreshadowing table")
	}
	// An unknown or zero horizon is as-lived: it never widens to hindsight.
	if planContextReviewRead("").foreshadowing != "" || planContextReviewRead("anything").foreshadowing != "" {
		t.Fatal("an unspecified horizon read the foreshadowing table")
	}
	hindsight := planContextReviewRead(contextreview.HorizonHindsight)
	if !strings.Contains(hindsight.foreshadowing, foreshadowingTable) || !strings.Contains(hindsight.foreshadowing, "horizon = 'hindsight'") {
		t.Fatal("the hindsight plan does not read the flag with its horizon predicate")
	}
}

func TestReadExecutesNoForeshadowingStatementOnTheAsLivedPath(t *testing.T) {
	subject := contextreview.Subject{PreviewHandle: strings.Repeat("h", 32), MessageID: "0190a000-0000-7000-8000-0000000000e1"}
	for _, test := range []struct {
		horizon       contextreview.Horizon
		wantFlagQuery bool
	}{{contextreview.HorizonAsLived, false}, {contextreview.HorizonHindsight, true}} {
		db := newReviewFakeDB()
		store, err := NewContextReviewStore(db)
		if err != nil {
			t.Fatal(err)
		}
		view, err := store.Read(context.Background(), subject, test.horizon)
		if err != nil {
			t.Fatalf("%s: %v", test.horizon, err)
		}
		named := false
		for _, statement := range db.statements {
			if strings.Contains(statement, foreshadowingTable) {
				named = true
			}
		}
		if named != test.wantFlagQuery {
			t.Fatalf("%s: foreshadowing statement executed = %v, want %v", test.horizon, named, test.wantFlagQuery)
		}
		if (view.Foreshadowing != nil) != test.wantFlagQuery {
			t.Fatalf("%s: foreshadowing member present = %v", test.horizon, view.Foreshadowing != nil)
		}
		if len(view.Reviews) != 1 || view.Reviews[0].Assertions.AboutChild == nil || *view.Reviews[0].Assertions.AboutChild != "yes" {
			t.Fatalf("%s: review not read back: %+v", test.horizon, view.Reviews)
		}
		if test.wantFlagQuery && (len(*view.Foreshadowing) != 1 || (*view.Foreshadowing)[0].Horizon != contextreview.HorizonHindsight) {
			t.Fatalf("hindsight flag not read back: %+v", *view.Foreshadowing)
		}
	}
}

func TestReadMapsUnscopedRunAndMissingTables(t *testing.T) {
	subject := contextreview.Subject{PreviewHandle: strings.Repeat("h", 32), MessageID: "0190a000-0000-7000-8000-0000000000e1"}
	unscoped := newReviewFakeDB()
	unscoped.subject = []any{"0190a000-0000-7000-8000-0000000000e1", (*string)(nil), (*string)(nil)}
	store, _ := NewContextReviewStore(unscoped)
	if _, err := store.Read(context.Background(), subject, contextreview.HorizonAsLived); !errors.Is(err, contextreview.ErrScopeMissing) {
		t.Fatalf("unscoped run: %v", err)
	}
	missing := newReviewFakeDB()
	missing.queryErr = &pgconn.PgError{Code: "42P01"}
	store, _ = NewContextReviewStore(missing)
	if _, err := store.Read(context.Background(), subject, contextreview.HorizonAsLived); !errors.Is(err, contextreview.ErrNotInstalled) {
		t.Fatalf("missing tables: %v", err)
	}
	gone := newReviewFakeDB()
	gone.subject = nil
	store, _ = NewContextReviewStore(gone)
	if _, err := store.Read(context.Background(), subject, contextreview.HorizonAsLived); !errors.Is(err, contextreview.ErrSubjectNotFound) {
		t.Fatalf("unknown message: %v", err)
	}
}

// reviewFakeDB records every statement and answers the three read statements
// with fixed rows. It is not a SQL engine: the SQL itself is proven against a
// real PostgreSQL by sql/validation/2026-09-25-context-review-horizon-test.sql.
type reviewFakeDB struct {
	statements []string
	subject    []any
	queryErr   error
}

func newReviewFakeDB() *reviewFakeDB {
	matter, courtCase := "0190a000-0000-7000-8000-00000000aa01", "0190a000-0000-7000-8000-00000000cc01"
	return &reviewFakeDB{subject: []any{"0190a000-0000-7000-8000-0000000000e1", &matter, &courtCase}}
}

func (d *reviewFakeDB) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	return nil, errors.New("reviewFakeDB does not open transactions")
}

func (d *reviewFakeDB) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	d.statements = append(d.statements, sql)
	if d.subject == nil {
		return reviewFakeRow{err: pgx.ErrNoRows}
	}
	return reviewFakeRow{values: d.subject}
}

func (d *reviewFakeDB) Query(_ context.Context, sql string, _ ...any) (pgx.Rows, error) {
	d.statements = append(d.statements, sql)
	if d.queryErr != nil {
		return nil, d.queryErr
	}
	at := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	if strings.Contains(sql, foreshadowingTable) {
		return &reviewFakeRows{rows: [][]any{{"0190a000-0000-7000-8000-0000000000b1", 1, true, "known later", "hindsight", at, "flag", "owner", "foreshadowing://1"}}}, nil
	}
	yes, relevant := "yes", true
	return &reviewFakeRows{rows: [][]any{{"0190a000-0000-7000-8000-0000000000a1", 1, []byte(`[{"label":"Recipient A","entity_id":null}]`),
		[]byte(`[]`), &yes, &relevant, "review", "owner", "context-review://1", at}}}, nil
}

type reviewFakeRow struct {
	values []any
	err    error
}

func (r reviewFakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	return reviewAssignAll(dest, r.values)
}

type reviewFakeRows struct {
	rows [][]any
	at   int
}

func (r *reviewFakeRows) Close()                                       {}
func (r *reviewFakeRows) Err() error                                   { return nil }
func (r *reviewFakeRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *reviewFakeRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *reviewFakeRows) Values() ([]any, error)                       { return r.rows[r.at-1], nil }
func (r *reviewFakeRows) RawValues() [][]byte                          { return nil }
func (r *reviewFakeRows) Conn() *pgx.Conn                              { return nil }
func (r *reviewFakeRows) Next() bool {
	if r.at >= len(r.rows) {
		return false
	}
	r.at++
	return true
}
func (r *reviewFakeRows) Scan(dest ...any) error { return reviewAssignAll(dest, r.rows[r.at-1]) }

func reviewAssignAll(dest []any, values []any) error {
	if len(dest) != len(values) {
		return fmt.Errorf("fake scan: %d destinations for %d values", len(dest), len(values))
	}
	for i := range dest {
		target := reflect.ValueOf(dest[i]).Elem()
		if values[i] == nil {
			target.Set(reflect.Zero(target.Type()))
			continue
		}
		value := reflect.ValueOf(values[i])
		if !value.Type().AssignableTo(target.Type()) {
			return fmt.Errorf("fake scan: column %d is %s, destination is %s", i, value.Type(), target.Type())
		}
		target.Set(value)
	}
	return nil
}

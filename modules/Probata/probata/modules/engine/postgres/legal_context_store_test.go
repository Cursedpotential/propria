package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/jackc/pgx/v5"
	"reflect"
	"strings"
	"testing"
)

func TestLegalPersonFingerprintPreservesNativeIdentityAndChanges(t *testing.T) {
	person := caseidentity.Person{ID: "native-person", DisplayName: "Person", Identifiers: []caseidentity.Identifier{{ID: "native-alias", RawValue: "original", Status: "confirmed"}}}
	first, err := legalPersonRecord(person)
	if err != nil {
		t.Fatal(err)
	}
	again, _ := legalPersonRecord(person)
	if first.Origin.RecordID != person.ID || first.Origin.RecordVersion != again.Origin.RecordVersion || !strings.HasPrefix(first.Origin.RecordVersion, "view-sha256:") {
		t.Fatal("native identity or deterministic fingerprint lost")
	}
	person.Identifiers[0].RawValue = "corrected"
	changed, _ := legalPersonRecord(person)
	if first.Origin.RecordVersion == changed.Origin.RecordVersion {
		t.Fatal("changed identifier did not change view fingerprint")
	}
	var descriptor map[string]any
	if err = json.Unmarshal(first.Record, &descriptor); err != nil || descriptor["scope"] != "registry" {
		t.Fatal("registry identity must not imply case participation")
	}
}

type legalEventRows struct {
	pgx.Rows
	index int
	count int
	err   error
}

func (r *legalEventRows) Next() bool { r.index++; return r.index <= r.count }
func (r *legalEventRows) Close()     {}
func (r *legalEventRows) Err() error { return r.err }
func (r *legalEventRows) Scan(dest ...any) error {
	*(dest[0].(*string)) = "native-event"
	*(dest[1].(*string)) = "generation#native-version"
	*(dest[2].(*string)) = "Native title"
	*(dest[3].(*json.RawMessage)) = json.RawMessage(`{"scope":"case","authority":"candidate_context"}`)
	return nil
}

type legalEventQuery struct {
	sql  string
	args []any
	rows *legalEventRows
	err  error
}

func (q *legalEventQuery) Query(_ context.Context, sql string, args ...any) (pgx.Rows, error) {
	q.sql = sql
	q.args = args
	return q.rows, q.err
}
func (q *legalEventQuery) QueryRow(context.Context, string, ...any) pgx.Row {
	panic("unexpected row query")
}

func TestLegalEventQueryBindsCaseGenerationMembershipAndBoundedFilters(t *testing.T) {
	db := &legalEventQuery{rows: &legalEventRows{count: 3}}
	query := LegalContextQuery{Kind: "event", Search: "owner's statement", RecordID: "chosen-id", Limit: 2}
	records, truncated, err := fetchLegalEvents(context.Background(), db, "selected-matter", "selected-court", query)
	if err != nil || !truncated || len(records) != 2 {
		t.Fatalf("bounded result: %v %v %d", err, truncated, len(records))
	}
	want := []any{"selected-matter", "selected-court", "owner's statement", "%owner's statement%", "chosen-id", 3}
	if !reflect.DeepEqual(db.args, want) {
		t.Fatalf("case/filter parameters %v", db.args)
	}
	for _, guard := range []string{"sv.matter_id=$1::uuid", "sv.court_case_id=$2::uuid", "sn.preview_handle=e.source_locator->>'preview_handle'", "sn.normalized_generation_id::text=e.source_locator->>'normalized_generation_id'", "m.included", "m.member_authority='candidate_context'", "LIMIT $6"} {
		if !strings.Contains(db.sql, guard) {
			t.Errorf("missing scope guard %s", guard)
		}
	}
	if strings.Contains(db.sql, query.Search) {
		t.Fatal("search was interpolated into SQL")
	}
	if records[0].Origin.RecordID != "native-event" || records[0].Origin.RecordVersion != "generation#native-version" {
		t.Fatal("native event identity/version changed")
	}
	db.rows = &legalEventRows{}
	absent, truncated, err := fetchLegalEvents(context.Background(), db, "selected-matter", "selected-court", query)
	if err != nil || truncated || len(absent) != 0 {
		t.Fatal("absent scoped record must be empty")
	}
	db.err = errors.New("database unavailable")
	if _, _, err = fetchLegalEvents(context.Background(), db, "selected-matter", "selected-court", query); err == nil {
		t.Fatal("database failure used fallback")
	}
}

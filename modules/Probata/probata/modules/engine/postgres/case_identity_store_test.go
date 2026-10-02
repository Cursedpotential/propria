// Byline: Claude Code · Opus 5.5 · 2026-10-01
package postgres

import (
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Cursedpotential/probata/engine/caseidentity"
)

func caseAlias(id, entity, raw, status string, supersedes *string, at time.Time) aliasRow {
	return aliasRow{
		IdentifierVersion: caseidentity.IdentifierVersion{ID: id, Status: status, RecordedBy: "matt", RecordedAt: at, SupersedesID: supersedes},
		entityID:          entity, raw: raw, kind: "phone", normalized: caseidentity.NormIdentifier(raw),
	}
}

func TestChainIdentifiersShowsTheNewestRowWithItsHistory(t *testing.T) {
	t0 := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	first, second := "a1", "a2"
	rows := []aliasRow{
		caseAlias(first, "matt", "810-252-2779", "candidate", nil, t0),
		caseAlias(second, "matt", "810-252-2779", "confirmed", &first, t0.Add(time.Minute)),
		caseAlias("a3", "matt", "810-252-2779", "retired", &second, t0.Add(2*time.Minute)),
		caseAlias("b1", "katrina", "810-268-9630", "confirmed", nil, t0),
	}
	chains := chainIdentifiers(rows)
	if len(chains["matt"]) != 1 {
		t.Fatalf("matt has %d chains, want 1: %+v", len(chains["matt"]), chains["matt"])
	}
	current := chains["matt"][0]
	if current.ID != "a3" || current.Status != "retired" || len(current.History) != 2 ||
		current.History[0].ID != "a2" || current.History[1].ID != "a1" {
		t.Fatalf("chain = %+v", current)
	}
	if len(chains["katrina"]) != 1 || len(chains["katrina"][0].History) != 0 {
		t.Fatalf("katrina chains = %+v", chains["katrina"])
	}
}

func TestChainIdentifiersNeverShowsACycle(t *testing.T) {
	a, b := "a", "b"
	rows := []aliasRow{caseAlias(a, "p", "x", "confirmed", &b, time.Now()), caseAlias(b, "p", "x", "confirmed", &a, time.Now())}
	// Both rows are superseded, so neither is current. UNIQUE(supersedes_id)
	// and insert order make a cycle unwritable; the reader still cannot loop.
	if chains := chainIdentifiers(rows); len(chains["p"]) != 0 {
		t.Fatalf("cycle produced chains %+v", chains)
	}
}

func TestCaseIdentityErrorsMapToThePageErrors(t *testing.T) {
	for code, want := range map[string]error{
		"23505": caseidentity.ErrStale,
		"23514": caseidentity.ErrRejected,
		"23503": caseidentity.ErrRejected,
		"42P01": caseidentity.ErrNotInstalled,
		"42501": caseidentity.ErrNotInstalled,
	} {
		if got := caseIdentityWriteError(&pgconn.PgError{Code: code, Message: "m"}); !errors.Is(got, want) {
			t.Errorf("%s mapped to %v, want %v", code, got, want)
		}
	}
	if caseIdentityError(nil) != nil {
		t.Fatal("nil error mapped to an error")
	}
}

func TestHeaderAndPersonColumnsAreTheOnlySQLIdentifiers(t *testing.T) {
	// Every column that reaches fmt.Sprintf in EditHeader / EditPerson comes
	// from these maps; pin them so a new entry is a reviewed change that also
	// updates the platform_runtime column grants.
	if len(caseidentity.HeaderColumns["matter"]) != 4 || len(caseidentity.HeaderColumns["court_case"]) != 10 || len(caseidentity.PersonColumns) != 9 {
		t.Fatal("editable column sets changed; review the UPDATE builders and the platform_runtime column grants")
	}
}

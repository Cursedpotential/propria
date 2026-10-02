// Byline: Claude Code · Opus 5.5 · 2026-10-01
package postgres

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Cursedpotential/probata/engine/caseidentity"
)

func TestSortIdentifiersPutsPhonesFirstAndSpellingsTogether(t *testing.T) {
	list := []caseidentity.Identifier{
		{ID: "n", Kind: "name", Normalized: "matthew"},
		{ID: "p2", Kind: "phone", Normalized: "8102751930"},
		{ID: "o", Kind: "other", Normalized: "x"},
		{ID: "p1", Kind: "phone", Normalized: "8102522779"},
		{ID: "p3", Kind: "phone", Normalized: "8102751930"},
	}
	sortIdentifiers(list)
	var got []string
	for _, identifier := range list {
		got = append(got, identifier.ID)
	}
	if want := []string{"p1", "p2", "p3", "n", "o"}; !equal(got, want) {
		t.Fatalf("order %v, want %v", got, want)
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
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
	// Every column that reaches fmt.Sprintf in EditHeader / EditPerson / EditIdentifier comes
	// from these maps; pin them so a new entry is a reviewed change that also
	// updates the platform_runtime column grants.
	if len(caseidentity.HeaderColumns["matter"]) != 4 || len(caseidentity.HeaderColumns["court_case"]) != 10 ||
		len(caseidentity.PersonColumns) != 9 || len(caseidentity.IdentifierColumns) != 6 {
		t.Fatal("editable column sets changed; review the UPDATE builders and the platform_runtime column grants")
	}
}

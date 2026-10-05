// Byline: Codex · GPT-5 · 2026-10-05
package postgres

import (
	"context"
	"errors"
	"github.com/Cursedpotential/probata/engine/caseidentity"
	"strings"
	"testing"
)

func TestCaseResolverNeverEnumeratesOrSelectsAnotherCase(t *testing.T) {
	store := &CaseIdentityStore{}
	// A nil queryer proves modes do not query for a sole unrelated matter.
	for _, mode := range []caseidentity.Mode{"", caseidentity.ModeDev, caseidentity.ModeLive, "TEST", "REAL"} {
		id, err := store.matterID(context.Background(), nil, mode)
		if err != nil || id != authoritativeMatterID {
			t.Fatalf("%s: %s %v", mode, id, err)
		}
	}
	if !strings.Contains(caseCourtCaseSQL, authoritativeCourtCaseID) {
		t.Fatal("court query does not bind exact approved identity")
	}
}

func TestDurableModeRequiresAnExplicitCanonicalReceipt(t *testing.T) {
	for detail, want := range map[string]string{"": "", "starting": "", `{}`: "", `{"operating_mode":"REAL"}`: "", `{"operating_mode":"DEV"}`: "DEV", `{"operating_mode":"LIVE"}`: "LIVE", `{"operating_mode":"invalid"}`: ""} {
		if got := recordedOperatingMode(detail); got != want {
			t.Fatalf("%s -> %q", detail, got)
		}
	}
}

func TestDirectCaseStoreDevPolicyDeniesBeforeTransaction(t *testing.T) {
	// nil DB would panic if the shared write seam reached persistence.
	store := &CaseIdentityStore{operatingMode: caseidentity.ModeDev}
	tx, rollback, err := store.begin(context.Background(), "unused")
	if tx != nil || !errors.Is(err, caseidentity.ErrDevWrite) {
		t.Fatal("Dev reached transaction", err)
	}
	rollback()
	_, err = store.EditHeader(context.Background(), caseidentity.ModeDev, caseidentity.HeaderSpec{}, caseidentity.Actor{})
	if !errors.Is(err, caseidentity.ErrDevWrite) {
		t.Fatal("header mode guard lost", err)
	}
}

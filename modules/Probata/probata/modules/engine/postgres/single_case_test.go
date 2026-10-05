// Byline: Codex · GPT-5 · 2026-10-05
package postgres

import (
	"context"
	"errors"
	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/runtimeapi/previewmodel"
	"github.com/google/uuid"
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

func TestAdmissionReceiptRecoversScopeBeforeRegistrationAndRejectsConflict(t *testing.T) {
	detail := `{"operating_mode":"LIVE","matter_id":"01a0f751-e07b-75cc-9ad5-63ad9449a8ba","court_case_id":"01a0f751-e07b-76a1-a738-eb3e3aa3e68c"}`
	binding := previewmodel.Binding{OperatingMode: recordedOperatingMode(detail)}
	applyBindingAdmission(&binding, detail)
	if binding.MatterID == nil || binding.MatterID.String() != authoritativeMatterID || binding.CourtCaseID == nil || binding.CourtCaseID.String() != authoritativeCourtCaseID || binding.OperatingMode != "LIVE" {
		t.Fatal(binding)
	}
	other := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	binding.MatterID = &other
	applyBindingAdmission(&binding, detail)
	if binding.OperatingMode != "" {
		t.Fatal("conflicting source scope admitted")
	}
	binding = previewmodel.Binding{OperatingMode: "LIVE"}
	applyBindingAdmission(&binding, `{"operating_mode":"LIVE"}`)
	if binding.OperatingMode != "" {
		t.Fatal("missing scope silently admitted")
	}
}

func TestWrongCourtHeaderRejectedBeforeTransaction(t *testing.T) {
	store := &CaseIdentityStore{operatingMode: caseidentity.ModeLive}
	_, err := store.EditHeader(context.Background(), caseidentity.ModeLive, caseidentity.HeaderSpec{Target: "court_case", ID: "11111111-1111-1111-1111-111111111111"}, caseidentity.Actor{})
	if !errors.Is(err, caseidentity.ErrRejected) {
		t.Fatal(err)
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

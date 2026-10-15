// Byline: Codex · GPT-6.1-sol · 2026-10-08.
package postgres

import (
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/activities"
)

func TestNewContextSourceRegistrationStoreRequiresExistingDatabase(t *testing.T) {
	if _, err := NewContextSourceRegistrationStore(nil); err == nil {
		t.Fatal("accepted nil database")
	}
	if _, err := NewContextSourceRegistrationStore(testDB{}); err != nil {
		t.Fatal(err)
	}
}

func TestContextRegistrationAllowsUnscopedUnknownFormat(t *testing.T) {
	req := activities.ContextSourceRegistrationRequest{
		RequestID: "request-1", WorkflowID: "temporal-workflow-1", SourceRef: "native://export/first/1",
		SourceKind: "first_party_ai_turn", ProviderVersionID: "provider-v2", PackageRef: "export-1",
	}
	got, err := normalizeContextRegistration(req)
	if err != nil {
		t.Fatal(err)
	}
	if got.DeclaredFormat != "unknown" || got.MatterID != "" || got.CourtCaseID != "" ||
		got.ProviderVersionID != "provider-v2" || got.PackageRef != "export-1" {
		t.Fatalf("unscoped native source coordinates changed: %+v", got)
	}
}

func TestContextRegistrationRejectsMissingIdentityAndOversizedMetadata(t *testing.T) {
	base := activities.ContextSourceRegistrationRequest{RequestID: "request-1", WorkflowID: "temporal-workflow-1",
		SourceRef: "native://export/first/1", SourceKind: "first_party_ai_turn"}
	for name, change := range map[string]func(*activities.ContextSourceRegistrationRequest){
		"source pointer":     func(req *activities.ContextSourceRegistrationRequest) { req.SourceRef = "" },
		"workflow":           func(req *activities.ContextSourceRegistrationRequest) { req.WorkflowID = "" },
		"body-sized pointer": func(req *activities.ContextSourceRegistrationRequest) { req.SourceRef = strings.Repeat("x", 2049) },
	} {
		t.Run(name, func(t *testing.T) {
			req := base
			change(&req)
			if _, err := normalizeContextRegistration(req); err == nil {
				t.Fatal("accepted invalid context source coordinates")
			}
		})
	}
}

func TestContextRegistrationPreservesOptionalScopeWithoutInventingCanonicalIDs(t *testing.T) {
	base := activities.ContextSourceRegistrationRequest{RequestID: "request-1", WorkflowID: "temporal-workflow-1",
		SourceRef: "native://export/first/1"}
	for _, scope := range []struct{ matter, courtCase string }{
		{"11111111-1111-4111-8111-111111111111", ""},
		{"not-yet-a-uuid", "22222222-2222-4222-8222-222222222222"},
	} {
		req := base
		req.MatterID, req.CourtCaseID = scope.matter, scope.courtCase
		got, err := normalizeContextRegistration(req)
		if err != nil || got.MatterID != scope.matter || got.CourtCaseID != scope.courtCase || got.SourceKind != "unknown" {
			t.Fatalf("optional native metadata was not retained: %+v, %v", got, err)
		}
		matter, courtCase := canonicalContextScope(got)
		if matter != "" || courtCase != "" {
			t.Fatalf("incomplete or invalid scope acquired canonical IDs: %q %q", matter, courtCase)
		}
	}
	base.MatterID = "11111111-1111-4111-8111-111111111111"
	base.CourtCaseID = "22222222-2222-4222-8222-222222222222"
	matter, courtCase := canonicalContextScope(base)
	if matter != base.MatterID || courtCase != base.CourtCaseID {
		t.Fatalf("valid paired scope did not reach canonical columns: %q %q", matter, courtCase)
	}
}

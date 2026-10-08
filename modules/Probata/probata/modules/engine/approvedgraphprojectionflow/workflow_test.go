package approvedgraphprojectionflow

import (
	"testing"

	"github.com/Cursedpotential/probata/engine/approvedgraph"
	"github.com/Cursedpotential/probata/engine/caseidentity"
)

func TestValidateScopeRequiresExactCanonicalCommitAndSourcePins(t *testing.T) {
	scope := approvedgraph.Scope{ReceiptID: "receipt", MatterID: caseidentity.AuthoritativeMatterID,
		CourtCaseID: caseidentity.AuthoritativeCourtCaseID, MatterMode: string(caseidentity.ModeLive),
		PreviewHandle: "preview", GenerationID: "generation", SourceVersionID: "version"}
	if err := ValidateScope(scope); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*approvedgraph.Scope){
		func(s *approvedgraph.Scope) { s.MatterMode = "DEV" },
		func(s *approvedgraph.Scope) { s.CourtCaseID = "different" },
		func(s *approvedgraph.Scope) { s.ReceiptID = "" },
		func(s *approvedgraph.Scope) { s.PreviewHandle = "preview\nother" },
		func(s *approvedgraph.Scope) { s.GenerationID = "" },
		func(s *approvedgraph.Scope) { s.SourceVersionID = "" },
	} {
		invalid := scope
		mutate(&invalid)
		if ValidateScope(invalid) == nil {
			t.Fatalf("accepted unbound scope: %+v", invalid)
		}
	}
}

package investigation

import (
	"github.com/Cursedpotential/probata/engine/caseidentity"
	"strings"
	"testing"
)

func TestValidationBoundsAndOpaqueCorrelation(t *testing.T) {
	id := "11111111-1111-4111-8111-111111111111"
	r := Request{Scope: Scope{Mode: caseidentity.ModeReal, MatterID: id, CourtCaseID: id}, LegalMatterID: id, ClaimID: id, FollowupID: id, Question: "Investigate", Sources: []Source{}}
	if e := Validate(r); e != nil {
		t.Fatal(e)
	}
	for _, change := range []func(*Request){func(r *Request) { r.Mode = "real" }, func(r *Request) { r.ClaimID = "bad" }, func(r *Request) { r.Question = strings.Repeat("x", 5001) }, func(r *Request) { r.Question = " " }, func(r *Request) { r.Sources = []Source{{Kind: "event", RecordID: id, RecordVersion: ""}} }, func(r *Request) { r.Sources = make([]Source, 101) }} {
		copy := r
		change(&copy)
		if Validate(copy) == nil {
			t.Fatal("invalid request accepted")
		}
	}
	r.Sources = nil
	one := string(CanonicalPayload(r))
	r.Sources = []Source{}
	if one != string(CanonicalPayload(r)) {
		t.Fatal("nil/empty source replay diverged")
	}
	if ValidateActor(Actor{UID: "uid", Username: "name", Key: id}) != nil {
		t.Fatal("actor rejected")
	}
}

func TestInvestigationLifecycleBounds(t *testing.T) {
	if ValidateTransition("received", "running", nil) != nil || ValidateTransition("running", "completed", []Result{{Summary: "result", Sources: []Source{}, Tool: "tool", RunID: "run"}}) != nil {
		t.Fatal("valid transition rejected")
	}
	for _, states := range [][2]string{{"received", "completed"}, {"completed", "running"}, {"running", "received"}} {
		if ValidateTransition(states[0], states[1], nil) == nil {
			t.Fatal("invalid transition accepted")
		}
	}
	if ValidateTransition("running", "completed", []Result{{Summary: strings.Repeat("x", 5001)}}) == nil {
		t.Fatal("unbounded results accepted")
	}
}

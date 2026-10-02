// Byline: Claude Code · Opus 5.5 · 2026-10-02
package postgres

import (
	"reflect"
	"testing"

	"github.com/Cursedpotential/probata/engine/disclosure"
	"github.com/Cursedpotential/probata/engine/firstparty"
)

// The same message seen from two phones names the same people: Matt's phone
// states "self" (the perspective, Matt) and Katrina's number; Katrina's phone
// states Matt's number and "self" (Katrina). Both reduce to the same party keys
// and the same sender key, so the match key the SQL function computes agrees.
func TestMessageKeyPartsAgreeAcrossDevices(t *testing.T) {
	const matt, katrina = "01a0f751-e07b-76b6-afcb-63acfbba373e", "01a0f751-e07b-76c7-8c0f-65692ad656b8"
	fromMattsPhone := firstparty.Plan{Resolution: disclosure.Resolution{Identifiers: []disclosure.ResolvedIdentifier{
		{Raw: "+18102959303", Normalized: "8102959303", EntityID: katrina},
	}}}
	fromKatrinasPhone := firstparty.Plan{Resolution: disclosure.Resolution{Identifiers: []disclosure.ResolvedIdentifier{
		{Raw: "8102959302", Normalized: "8102959302", EntityID: matt},
	}}}
	a, aSender := messageKeyParts(fromMattsPhone, firstparty.Message{Participants: []firstparty.Participant{
		{Raw: "self", Role: "from", EntityID: matt}, {Raw: "+18102959303", Role: "to", EntityID: katrina},
	}})
	b, bSender := messageKeyParts(fromKatrinasPhone, firstparty.Message{Participants: []firstparty.Participant{
		{Raw: "8102959302", Role: "from", EntityID: upperASCII(matt)}, {Raw: "self", Role: "to", EntityID: katrina},
	}})
	if !reflect.DeepEqual(a, b) || aSender != bSender || aSender != matt {
		t.Fatalf("parties %v / %v, senders %q / %q", a, b, aSender, bSender)
	}

	// An unresolved party is named by its registry-normalized identifier, so two
	// spellings of one number still agree; with no normalization, the raw text.
	plan := firstparty.Plan{Resolution: disclosure.Resolution{Identifiers: []disclosure.ResolvedIdentifier{
		{Raw: "+18108360123", Normalized: "8108360123"}, {Raw: "(810) 836-0123", Normalized: "8108360123"},
	}}}
	if partyKey(plan, firstparty.Participant{Raw: "+18108360123"}) != partyKey(plan, firstparty.Participant{Raw: "(810) 836-0123"}) {
		t.Fatal("two spellings of one unresolved number produced different party keys")
	}
	if got := partyKey(plan, firstparty.Participant{Raw: " Some Name "}); got != "raw:some name" {
		t.Fatalf("unnormalized party key = %q", got)
	}
}

func upperASCII(value string) string {
	out := []byte(value)
	for i, c := range out {
		if c >= 'a' && c <= 'z' {
			out[i] = c - 32
		}
	}
	return string(out)
}

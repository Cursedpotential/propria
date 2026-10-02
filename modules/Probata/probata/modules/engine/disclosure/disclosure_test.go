// Byline: Claude Code · Opus 5.5 · 2026-10-02
package disclosure

import "testing"

func TestOwnerParticipantRule(t *testing.T) {
	owner := Owner{PersonID: "owner", Identifiers: map[string]bool{"8105550100": true}}
	identifiers := []ResolvedIdentifier{
		{Raw: "+1 810 555 0100", Normalized: "8105550100", EntityID: "owner", IsOwner: true},
		{Raw: "+18105550199", Normalized: "8105550199", EntityID: "partner"},
		{Raw: "+18105550123", Normalized: "8105550123"},
	}
	cases := []struct {
		name        string
		perspective string
		sender      string
		recipients  []string
		want        string
		wantErr     bool
	}{
		{"owner is the recipient", "", "+18105550199", []string{"+1 810 555 0100"}, Contemporaneous, false},
		{"owner absent", "", "+18105550199", []string{"+18105550123"}, Discovered, false},
		{"self on the owner's phone", "owner", "self", []string{"+18105550199"}, Contemporaneous, false},
		{"self on someone else's phone", "partner", "self", []string{"+18105550123"}, Discovered, false},
		{"self with no perspective", "", "self", nil, "", true},
		{"identifier outside the resolution", "", "+18105550000", nil, "", true},
		{"no participants", "", "", nil, Discovered, false},
	}
	for _, c := range cases {
		o := owner
		o.PerspectivePersonID = c.perspective
		resolution, err := NewResolution(o, identifiers)
		if err != nil {
			t.Fatalf("%s: NewResolution: %v", c.name, err)
		}
		_, got, basis, err := resolution.ForMessage(c.sender, c.recipients)
		if (err != nil) != c.wantErr || got != c.want || (err == nil && basis != Basis) {
			t.Errorf("%s: got %q %q, %v; want %q, error %v", c.name, got, basis, err, c.want, c.wantErr)
		}
	}
	if _, err := NewResolution(Owner{}, nil); err == nil {
		t.Error("an unregistered owner must fail closed")
	}
}

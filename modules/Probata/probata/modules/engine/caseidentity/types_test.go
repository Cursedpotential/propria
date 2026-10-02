// Byline: Claude Code · Opus 5.5 · 2026-10-01
package caseidentity

import (
	"strings"
	"testing"
)

func ptr(s string) *string { return &s }

// The cases are the ones registry.norm_identifier was read back with live on
// 2026-10-01: 8102689630 x3, 8103533592, 34428, katrina kinzel.
func TestNormIdentifierMatchesTheSQLRule(t *testing.T) {
	for in, want := range map[string]string{
		"+1 (810) 268-9630":   "8102689630",
		"18102689630":         "8102689630",
		"810.268.9630":        "8102689630",
		"8102689630":          "8102689630",
		"*678103533592":       "8103533592",
		"34428":               "34428",
		"Katrina Kinzel":      "katrina kinzel",
		" Me ":                "me",
		"someone@example.com": "someone@example.com",
		"":                    "",
		"   ":                 "",
	} {
		if got := NormIdentifier(in); got != want {
			t.Errorf("NormIdentifier(%q) = %q, want %q", in, got, want)
		}
	}
}

func validIdentifier() IdentifierSpec {
	return IdentifierSpec{
		EntityID: "01a0f751-e07b-76b6-afcb-63acfbba373e", RawValue: "810-295-9302", Kind: "phone",
		Status: "confirmed", Period: ptr("2021-2024"), Basis: "owner 2026-09-23 14:23: 9302 was his",
	}
}

func TestValidateIdentifierKeepsTheRawSpellingAndRequiresABasis(t *testing.T) {
	if err := ValidateIdentifier(validIdentifier()); err != nil {
		t.Fatalf("valid identifier rejected: %v", err)
	}
	cases := map[string]func(*IdentifierSpec){
		"no entity":          func(s *IdentifierSpec) { s.EntityID = "matt" },
		"padded raw":         func(s *IdentifierSpec) { s.RawValue = " 810-295-9302" },
		"empty raw":          func(s *IdentifierSpec) { s.RawValue = "" },
		"line break":         func(s *IdentifierSpec) { s.RawValue = "810\n295" },
		"unknown kind":       func(s *IdentifierSpec) { s.Kind = "fax" },
		"unknown status":     func(s *IdentifierSpec) { s.Status = "maybe" },
		"no basis":           func(s *IdentifierSpec) { s.Basis = " " },
		"long raw":           func(s *IdentifierSpec) { s.RawValue = strings.Repeat("9", MaxValueBytes+1) },
		"version, no reason": func(s *IdentifierSpec) { s.SupersedesID = "01a0f751-e07b-76b6-afcb-63acfbba373e" },
		"bad supersedes":     func(s *IdentifierSpec) { s.SupersedesID = "previous"; s.ChangeReason = "x" },
		"control char basis": func(s *IdentifierSpec) { s.Basis = "a\x07b" },
	}
	for name, mutate := range cases {
		spec := validIdentifier()
		mutate(&spec)
		if err := ValidateIdentifier(spec); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	version := validIdentifier()
	version.SupersedesID, version.ChangeReason = "01a0f751-e07b-76b6-afcb-63acfbba373e", "owner confirmed on the Case page"
	if err := ValidateIdentifier(version); err != nil {
		t.Fatalf("valid version rejected: %v", err)
	}
}

func TestValidateHeaderAllowsOnlyEditableColumns(t *testing.T) {
	ok := HeaderSpec{Target: "court_case", ID: "01a0f751-e07b-76a1-a738-eb3e3aa3e68c",
		Fields:       map[string]*string{"docket_number": ptr("2025-53985-DC"), "filed_on": ptr("2025-06-01"), "jurisdiction": nil},
		ChangeReason: "caption from the complaint"}
	if err := ValidateHeader(ok); err != nil {
		t.Fatalf("valid header edit rejected: %v", err)
	}
	for name, spec := range map[string]HeaderSpec{
		"unknown target":   {Target: "person", ID: ok.ID, Fields: ok.Fields, ChangeReason: "x"},
		"unknown column":   {Target: "court_case", ID: ok.ID, Fields: map[string]*string{"created_by": ptr("me")}, ChangeReason: "x"},
		"injection column": {Target: "matter", ID: ok.ID, Fields: map[string]*string{"title = 'x'; --": ptr("x")}, ChangeReason: "x"},
		"clear caption":    {Target: "court_case", ID: ok.ID, Fields: map[string]*string{"caption": nil}, ChangeReason: "x"},
		"bad date":         {Target: "court_case", ID: ok.ID, Fields: map[string]*string{"filed_on": ptr("June 2025")}, ChangeReason: "x"},
		"no reason":        {Target: "court_case", ID: ok.ID, Fields: ok.Fields},
		"no fields":        {Target: "matter", ID: ok.ID, ChangeReason: "x"},
	} {
		if err := ValidateHeader(spec); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestValidatePersonAndNewPerson(t *testing.T) {
	ok := PersonSpec{ID: "01a0f751-e07b-76c7-8c0f-65692ad656b8",
		Fields: map[string]*string{"short_name": ptr("Katrina"), "is_minor": ptr("false")}, ChangeReason: "label the catalog tools use"}
	if err := ValidatePerson(ok); err != nil {
		t.Fatalf("valid person edit rejected: %v", err)
	}
	for name, spec := range map[string]PersonSpec{
		"unknown column":  {ID: ok.ID, Fields: map[string]*string{"merged_into_id": ptr("x")}, ChangeReason: "x"},
		"bad bool":        {ID: ok.ID, Fields: map[string]*string{"is_minor": ptr("yes")}, ChangeReason: "x"},
		"clear name":      {ID: ok.ID, Fields: map[string]*string{"display_name": nil}, ChangeReason: "x"},
		"long short name": {ID: ok.ID, Fields: map[string]*string{"short_name": ptr(strings.Repeat("k", 65))}, ChangeReason: "x"},
		"no reason":       {ID: ok.ID, Fields: ok.Fields},
	} {
		if err := ValidatePerson(spec); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if err := ValidateNewPerson(NewPersonSpec{DisplayName: "A Witness", RoleInCase: "witness", ConnectionTo: "third_party", ChangeReason: "named in a text"}); err != nil {
		t.Fatalf("valid new person rejected: %v", err)
	}
	if err := ValidateNewPerson(NewPersonSpec{DisplayName: " padded", RoleInCase: "witness", ConnectionTo: "third_party", ChangeReason: "x"}); err == nil {
		t.Fatal("padded display name accepted")
	}
}

func TestActorKeysAreScopedPerActorAndOperation(t *testing.T) {
	a := Actor{SubjectUID: "uid-1", Username: "matt", IdempotencyKey: "k1"}
	b := Actor{SubjectUID: "uid-2", Username: "other", IdempotencyKey: "k1"}
	if a.StoredKey("identifier") == b.StoredKey("identifier") {
		t.Fatal("two actors share a stored key")
	}
	if a.StoredKey("identifier") == a.StoredKey("header") {
		t.Fatal("two operations share a stored key")
	}
	if err := ValidateActor(Actor{SubjectUID: "uid", Username: "matt"}); err == nil {
		t.Fatal("missing Idempotency-Key accepted")
	}
}

func TestValidateTriageLookupAndMode(t *testing.T) {
	if err := ValidateTriage(TriageSpec{RawValue: "34428", Decision: "dismissed", Basis: "carrier short code"}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateTriage(TriageSpec{RawValue: "34428", Decision: "deleted", Basis: "x"}); err == nil {
		t.Fatal("unknown decision accepted")
	}
	if err := ValidateLookup(nil); err == nil {
		t.Fatal("empty lookup accepted")
	}
	if _, err := ParseMode("PROD"); err == nil {
		t.Fatal("unknown mode accepted")
	}
}

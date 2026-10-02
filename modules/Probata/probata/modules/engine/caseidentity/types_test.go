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
		"control char basis": func(s *IdentifierSpec) { s.Basis = "a\x07b" },
	}
	for name, mutate := range cases {
		spec := validIdentifier()
		mutate(&spec)
		if err := ValidateIdentifier(spec); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestValidateIdentifierEditAndDelete(t *testing.T) {
	id := "01a0f751-e07b-76b6-afcb-63acfbba373e"
	ok := IdentifierEditSpec{ID: id, Fields: map[string]*string{"kind": ptr("legal"), "period": nil, "basis": ptr("the caption names her")}, ChangeReason: "her legal name"}
	if err := ValidateIdentifierEdit(ok); err != nil {
		t.Fatalf("valid edit rejected: %v", err)
	}
	for name, spec := range map[string]IdentifierEditSpec{
		"bad id":          {ID: "x", Fields: ok.Fields, ChangeReason: "x"},
		"no fields":       {ID: id, ChangeReason: "x"},
		"no reason":       {ID: id, Fields: ok.Fields},
		"unknown field":   {ID: id, Fields: map[string]*string{"normalized": ptr("x")}, ChangeReason: "x"},
		"injection field": {ID: id, Fields: map[string]*string{"status = 'x' --": ptr("x")}, ChangeReason: "x"},
		"clear raw":       {ID: id, Fields: map[string]*string{"raw_value": nil}, ChangeReason: "x"},
		"padded raw":      {ID: id, Fields: map[string]*string{"raw_value": ptr(" 810")}, ChangeReason: "x"},
		"bad kind":        {ID: id, Fields: map[string]*string{"kind": ptr("fax")}, ChangeReason: "x"},
		"bad status":      {ID: id, Fields: map[string]*string{"status": ptr("sure")}, ChangeReason: "x"},
		"bad person":      {ID: id, Fields: map[string]*string{"entity_id": ptr("Matt")}, ChangeReason: "x"},
		"empty basis":     {ID: id, Fields: map[string]*string{"basis": ptr(" ")}, ChangeReason: "x"},
	} {
		if err := ValidateIdentifierEdit(spec); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if err := ValidateIdentifierDelete(IdentifierDeleteSpec{ID: id, ChangeReason: "typed into the wrong person"}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateIdentifierDelete(IdentifierDeleteSpec{ID: id}); err == nil {
		t.Fatal("delete without a reason accepted")
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

// Byline: Claude Code · Sonnet · 2026-10-02 (placeholders, merge, rename fields)
func TestPlaceholderNumbersAndNames(t *testing.T) {
	for in, want := range map[string]string{"+1 (810) 555-0142": "8105550142", "18105550142": "8105550142", "810.555.0142": "8105550142"} {
		got, ok := NormalizePhone(in)
		if !ok || got != want {
			t.Fatalf("NormalizePhone(%q) = %q, %v", in, got, ok)
		}
	}
	for _, bad := range []string{"34428", "1115", "someone@example.com", "Katrina", "", "0105550142", "+44 20 7946 0958"} {
		if _, ok := NormalizePhone(bad); ok {
			t.Fatalf("NormalizePhone(%q) accepted a non-US-phone value", bad)
		}
	}
	if got := PlaceholderName("8105550142"); got != "Unknown 810-555-0142" {
		t.Fatalf("PlaceholderName = %q", got)
	}
}

func TestPlaceholderAndMergeValidation(t *testing.T) {
	if err := ValidatePlaceholders(PlaceholderSpec{Numbers: []string{"8105550142"}, ChangeReason: "seen in calls"}); err != nil {
		t.Fatal(err)
	}
	if ValidatePlaceholders(PlaceholderSpec{ChangeReason: "x"}) == nil {
		t.Fatal("an empty batch must be refused")
	}
	if ValidatePlaceholders(PlaceholderSpec{Numbers: make([]string, MaxPlaceholderNumbers+1), ChangeReason: "x"}) == nil {
		t.Fatal("an oversized batch must be refused")
	}
	if ValidatePlaceholders(PlaceholderSpec{Numbers: []string{"8105550142"}}) == nil {
		t.Fatal("a reason is required")
	}
	a, b := "01a0f751-e07b-76b6-afcb-63acfbba373e", "01a0f751-e07b-76c7-8c0f-65692ad656b8"
	if err := ValidateMerge(MergeSpec{FromID: a, IntoID: b, ChangeReason: "her other phone"}); err != nil {
		t.Fatal(err)
	}
	if ValidateMerge(MergeSpec{FromID: a, IntoID: a, ChangeReason: "x"}) == nil {
		t.Fatal("merging a person into itself must be refused")
	}
	if ValidateMerge(MergeSpec{FromID: a, IntoID: "nope", ChangeReason: "x"}) == nil {
		t.Fatal("a bad uuid must be refused")
	}
}

func TestNamingAPlaceholderIsAPersonEdit(t *testing.T) {
	yes, no := "true", "false"
	name, state, status := "Jordan Reyes", "confirmed", "approved"
	if err := ValidatePerson(PersonSpec{ID: "01a0f751-e07b-76b6-afcb-63acfbba373e", ChangeReason: "named by the owner", Fields: map[string]*string{
		"display_name": &name, "verification_state": &state, "requires_human_review": &no, "review_status": &status}}); err != nil {
		t.Fatal(err)
	}
	badStatus := "maybe"
	if ValidatePerson(PersonSpec{ID: "01a0f751-e07b-76b6-afcb-63acfbba373e", ChangeReason: "x", Fields: map[string]*string{"review_status": &badStatus}}) == nil {
		t.Fatal("an unknown review status must be refused")
	}
	if ValidatePerson(PersonSpec{ID: "01a0f751-e07b-76b6-afcb-63acfbba373e", ChangeReason: "x", Fields: map[string]*string{"requires_human_review": &yes, "verification_state": &badStatus}}) == nil {
		t.Fatal("an unknown verification state must be refused")
	}
}

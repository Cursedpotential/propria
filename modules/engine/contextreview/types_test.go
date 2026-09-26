// Byline: Claude Code · Opus 5.5 · 2026-09-26
package contextreview

import (
	"strings"
	"testing"
)

func ptr[T any](value T) *T { return &value }

func TestParseHorizonDefaultsToAsLived(t *testing.T) {
	for _, raw := range []string{"", "  ", "as_lived"} {
		got, err := ParseHorizon(raw)
		if err != nil || got != HorizonAsLived {
			t.Fatalf("ParseHorizon(%q) = %q, %v; want as_lived", raw, got, err)
		}
	}
	if got, err := ParseHorizon("hindsight"); err != nil || got != HorizonHindsight {
		t.Fatalf("ParseHorizon(hindsight) = %q, %v", got, err)
	}
	if _, err := ParseHorizon("HINDSIGHT"); err == nil {
		t.Fatal("an unknown horizon spelling was accepted")
	}
}

func TestValidateSubject(t *testing.T) {
	good := Subject{PreviewHandle: strings.Repeat("a", 32), MessageID: "0190a000-0000-7000-8000-0000000000e1"}
	if err := ValidateSubject(good); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSubject(Subject{PreviewHandle: "short", MessageID: good.MessageID}); err == nil {
		t.Fatal("short preview handle accepted")
	}
	if err := ValidateSubject(Subject{PreviewHandle: good.PreviewHandle, MessageID: "not-a-uuid"}); err == nil {
		t.Fatal("non-UUID message id accepted")
	}
}

func TestValidateAssertions(t *testing.T) {
	valid := Assertions{
		AddressedTo: []Party{{Label: "Recipient A"}},
		About:       []Party{{Label: "The child", EntityID: ptr("0190a000-0000-7000-8000-0000000000e1")}},
		AboutChild:  ptr("yes"),
		Relevant:    ptr(true),
	}
	if err := ValidateAssertions(valid); err != nil {
		t.Fatal(err)
	}
	cases := map[string]Assertions{
		"bad about_child":    {AboutChild: ptr("maybe")},
		"padded label":       {About: []Party{{Label: " padded "}}},
		"empty label":        {About: []Party{{Label: ""}}},
		"control character":  {About: []Party{{Label: "a\u0007b"}}},
		"bad entity id":      {About: []Party{{Label: "Someone", EntityID: ptr("x")}}},
		"duplicate name":     {AddressedTo: []Party{{Label: "Same"}, {Label: "same"}}},
		"label too long":     {About: []Party{{Label: strings.Repeat("x", MaxLabelRunes+1)}}},
		"too many addressed": {AddressedTo: manyParties(MaxParties + 1)},
	}
	for name, assertions := range cases {
		if err := ValidateAssertions(assertions); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func manyParties(n int) []Party {
	parties := make([]Party, n)
	for i := range parties {
		parties[i] = Party{Label: "Person " + strings.Repeat("x", i+1)}
	}
	return parties
}

func TestValidateChangeAndActor(t *testing.T) {
	if err := ValidateChange("", ""); err == nil {
		t.Fatal("empty reason accepted")
	}
	if err := ValidateChange("reason", strings.Repeat("n", MaxNoteBytes+1)); err == nil {
		t.Fatal("oversized note accepted")
	}
	if err := ValidateChange("reason", "note"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateActor("uid", "owner", ""); err == nil {
		t.Fatal("missing idempotency key accepted")
	}
	if err := ValidateActor("", "owner", "key"); err == nil {
		t.Fatal("missing actor accepted")
	}
	if err := ValidateActor("uid", "owner", "key"); err != nil {
		t.Fatal(err)
	}
}

func TestDigestsBindContentAndKey(t *testing.T) {
	spec := ReviewSpec{
		Subject:    Subject{PreviewHandle: strings.Repeat("a", 32), MessageID: "0190a000-0000-7000-8000-0000000000e1"},
		Assertions: Assertions{Relevant: ptr(true)}, ChangeReason: "why", ActorSubjectUID: "uid", IdempotencyKey: "key",
	}
	first := ReviewDigest(spec)
	if first != ReviewDigest(spec) {
		t.Fatal("review digest is not stable")
	}
	// nil and empty party lists are the same submission.
	spec.Assertions.About = []Party{}
	if first != ReviewDigest(spec) {
		t.Fatal("empty and absent party lists digest differently")
	}
	changed := spec
	changed.Assertions.Relevant = ptr(false)
	if first == ReviewDigest(changed) {
		t.Fatal("changed content kept the same digest")
	}
	flag := ForeshadowingSpec{Subject: spec.Subject, Foreshadowing: true, ChangeReason: "why", ActorSubjectUID: "uid", IdempotencyKey: "key"}
	cleared := flag
	cleared.Foreshadowing = false
	if ForeshadowingDigest(flag) == ForeshadowingDigest(cleared) {
		t.Fatal("setting and clearing the flag share a digest")
	}
}

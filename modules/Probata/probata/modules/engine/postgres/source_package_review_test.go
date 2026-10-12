package postgres

import (
	"reflect"
	"testing"
)

// packageReviewFixture supplies native source pins without retained objects or preview bindings.
// Inputs: none. Output: a minimal attributed derived assertion; effects: none.
func packageReviewFixture() PackageReviewSpec {
	return PackageReviewSpec{SourceVersionRef: "11111111-1111-4111-8111-111111111111", Operation: "edit", ItemID: "candidate:native:7", Reason: "Owner corrected assertion", ActorSubjectUID: "owner", ActorUsername: "owner", IdempotencyKey: "click-1", Fields: map[string]any{"label": "Corrected"}, SourceRefs: []map[string]any{{"source_ref": "b2://case/SourceCorpus/sms/unit.xml", "source_version_ref": "11111111-1111-4111-8111-111111111111", "locator": map[string]any{"record": 7}}}}
}

// TestNativePackageReviewCoordinates verifies source-first writes require no fake custody identities.
// Inputs: native request and invalid mutations. Output: acceptance/refusal assertions; effects: none.
func TestNativePackageReviewCoordinates(t *testing.T) {
	spec := packageReviewFixture()
	if err := ValidatePackageReview(spec); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"body", "messages", "turns", "raw_body", "full_text", "message_body", "content", "text", "payload", "raw_reply", "reply", "replies"} {
		bad := packageReviewFixture()
		bad.Fields = map[string]any{"nested": []any{map[string]any{key: "native source content"}}}
		if err := ValidatePackageReview(bad); err == nil {
			t.Errorf("source field %q admitted to PostgreSQL", key)
		}
	}
	bad := packageReviewFixture()
	bad.SourceRefs[0]["source_version_ref"] = "22222222-2222-4222-8222-222222222222"
	if err := ValidatePackageReview(bad); err == nil {
		t.Fatal("cross-version source pin accepted")
	}
	bad = packageReviewFixture()
	delete(bad.SourceRefs[0], "locator")
	if err := ValidatePackageReview(bad); err == nil {
		t.Fatal("missing locator accepted")
	}
}

// TestPackageCorrectionsAccumulate verifies successive partial edits retain earlier corrections.
// Inputs: nested prior overrides and changes. Output: fresh cumulative map; effects: none.
func TestPackageCorrectionsAccumulate(t *testing.T) {
	prior := map[string]any{"label": "first", "native": map[string]any{"category": "sms", "keep": true}}
	changes := map[string]any{"native": map[string]any{"category": "calls"}, "removed": nil}
	got := mergePackageFields(prior, changes)
	want := map[string]any{"label": "first", "native": map[string]any{"category": "calls", "keep": true}, "removed": nil}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("cumulative corrections = %#v", got)
	}
	if prior["native"].(map[string]any)["category"] != "sms" {
		t.Fatal("original correction mutated")
	}
}

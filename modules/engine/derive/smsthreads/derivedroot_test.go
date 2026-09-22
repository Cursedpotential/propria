// Byline: Claude Code · Opus 5 · 2026-09-21
//
// Unit tests for derived placement. In-memory only; nothing here touches an
// object store, and none of it is evidence the deployed worker works.

package smsthreads

import (
	"context"
	"testing"
)

func TestLocateMapsTheInnerPathUnderTheDerivedRoot(t *testing.T) {
	roots, err := ParseDerivedRoots(
		`[{"source":"b2://bucket/source-root/","derived":"b2://bucket/derived-root/"}]`)
	if err != nil {
		t.Fatalf("ParseDerivedRoots() error = %v", err)
	}

	for name, testCase := range map[string]struct {
		key        string
		wantPrefix string
	}{
		"plain":             {"source-root/a/b.xml", "derived-root/a/b.xml/"},
		"prefix with space": {"source-root/sms /file.xml", "derived-root/sms /file.xml/"},
		"root-level file":   {"source-root/file.xml", "derived-root/file.xml/"},
		"deep":              {"source-root/x/y/z/file.xml", "derived-root/x/y/z/file.xml/"},
	} {
		got, err := roots.Locate("b2", "bucket", testCase.key)
		if err != nil {
			t.Fatalf("%s: Locate() error = %v", name, err)
		}
		if !got.Mapped || got.Prefix != testCase.wantPrefix || got.Bucket != "bucket" || got.Scheme != "b2" {
			t.Errorf("%s: Locate() = %+v, want prefix %q", name, got, testCase.wantPrefix)
		}
		if got.ManifestKey() != testCase.wantPrefix+ManifestName {
			t.Errorf("%s: ManifestKey() = %q", name, got.ManifestKey())
		}
	}
}

// A configured source prefix is a directory boundary: "v1" must never match
// "v10", with or without a trailing slash in the configuration.
func TestLocateNormalisesTrailingSlashes(t *testing.T) {
	roots, err := ParseDerivedRoots(
		`[{"source":"b2://bucket/vault/v1","derived":"b2://bucket/derived/v1"}]`)
	if err != nil {
		t.Fatalf("ParseDerivedRoots() error = %v", err)
	}
	mapped, err := roots.Locate("b2", "bucket", "vault/v1/a.xml")
	if err != nil {
		t.Fatal(err)
	}
	if mapped.Prefix != "derived/v1/a.xml/" {
		t.Errorf("Locate() prefix = %q, want derived/v1/a.xml/", mapped.Prefix)
	}
	sibling, err := roots.Locate("b2", "bucket", "vault/v10/a.xml")
	if err != nil {
		t.Fatal(err)
	}
	if sibling.Mapped {
		t.Errorf("vault/v10 must not match the vault/v1 root: %+v", sibling)
	}
}

// No match, an empty configuration, and a different bucket or scheme all fall
// back to beside-the-original so nothing breaks before the pair is set.
func TestLocateFallsBackBesideTheOriginal(t *testing.T) {
	roots, err := ParseDerivedRoots(`[{"source":"b2://bucket/vault/","derived":"b2://bucket/derived/"}]`)
	if err != nil {
		t.Fatal(err)
	}
	for name, probe := range map[string]struct{ scheme, bucket, key string }{
		"other prefix": {"b2", "bucket", "elsewhere/a.xml"},
		"other bucket": {"b2", "other", "vault/a.xml"},
		"other scheme": {"r2", "bucket", "vault/a.xml"},
	} {
		got, err := roots.Locate(probe.scheme, probe.bucket, probe.key)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got.Mapped || got.Prefix != probe.key+DerivedSuffix+"/" || got.Bucket != probe.bucket {
			t.Errorf("%s: Locate() = %+v, want the beside-the-original fallback", name, got)
		}
	}

	var unset DerivedRoots
	got, err := unset.Locate("b2", "bucket", "vault/a.xml")
	if err != nil {
		t.Fatal(err)
	}
	if got.Mapped || got.Prefix != "vault/a.xml"+DerivedSuffix+"/" {
		t.Errorf("unset roots must fall back, got %+v", got)
	}
}

func TestLocatePrefersTheLongestMatchingSourceRoot(t *testing.T) {
	roots, err := ParseDerivedRoots(`[
		{"source":"b2://bucket/vault/","derived":"b2://bucket/derived/broad/"},
		{"source":"b2://bucket/vault/messages/","derived":"b2://bucket/derived/narrow/"}
	]`)
	if err != nil {
		t.Fatal(err)
	}
	got, err := roots.Locate("b2", "bucket", "vault/messages/a.xml")
	if err != nil {
		t.Fatal(err)
	}
	if got.Prefix != "derived/narrow/a.xml/" {
		t.Errorf("Locate() prefix = %q, want the more specific rule to win", got.Prefix)
	}
}

func TestParseDerivedRootsRejectsBadConfiguration(t *testing.T) {
	for name, raw := range map[string]string{
		"not json":              `{"source":"b2://a/b/"}`,
		"no scheme":             `[{"source":"bucket/vault/","derived":"b2://bucket/derived/"}]`,
		"no bucket":             `[{"source":"b2://","derived":"b2://bucket/derived/"}]`,
		"cross scheme":          `[{"source":"b2://bucket/vault/","derived":"r2://bucket/derived/"}]`,
		"derived inside source": `[{"source":"b2://bucket/vault/","derived":"b2://bucket/vault/derived/"}]`,
	} {
		if _, err := ParseDerivedRoots(raw); err == nil {
			t.Errorf("%s: ParseDerivedRoots() must fail", name)
		}
	}
	empty, err := ParseDerivedRoots("   ")
	if err != nil || empty != nil {
		t.Errorf("an unset variable must be accepted as no rules, got %v / %v", empty, err)
	}
}

func TestRequireConfiguredSchemes(t *testing.T) {
	roots, err := ParseDerivedRoots(`[{"source":"b2://bucket/vault/","derived":"b2://bucket/derived/"}]`)
	if err != nil {
		t.Fatal(err)
	}
	if err := roots.RequireConfiguredSchemes([]string{"b2", "r2"}); err != nil {
		t.Errorf("configured scheme must pass: %v", err)
	}
	if err := roots.RequireConfiguredSchemes([]string{"r2"}); err == nil {
		t.Error("an unconfigured scheme must fail worker startup")
	}
}

// Switching the configuration on must not orphan or re-derive a derivation
// already published beside its original.
func TestResolvePublishedFindsTheLegacyLocation(t *testing.T) {
	roots, err := ParseDerivedRoots(`[{"source":"b2://bucket/vault/","derived":"b2://bucket/derived/"}]`)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	legacy := &memoryStore{objects: map[string][]byte{
		"bucket/vault/a.xml" + DerivedSuffix + "/" + ManifestName: []byte("{}"),
	}}
	got, published, err := ResolvePublished(ctx, legacy, roots, "b2", "bucket", "vault/a.xml")
	if err != nil {
		t.Fatal(err)
	}
	if !published || got.Mapped || got.Prefix != "vault/a.xml"+DerivedSuffix+"/" {
		t.Errorf("an existing legacy derivation must be found in place, got %+v published=%v", got, published)
	}

	// The mapped location wins when both exist.
	both := &memoryStore{objects: map[string][]byte{
		"bucket/vault/a.xml" + DerivedSuffix + "/" + ManifestName: []byte("{}"),
		"bucket/derived/a.xml/" + ManifestName:                    []byte("{}"),
	}}
	got, published, err = ResolvePublished(ctx, both, roots, "b2", "bucket", "vault/a.xml")
	if err != nil {
		t.Fatal(err)
	}
	if !published || !got.Mapped || got.Prefix != "derived/a.xml/" {
		t.Errorf("the mapped location must win, got %+v", got)
	}

	// Nothing published: the mapped location is where a new derivation goes.
	got, published, err = ResolvePublished(ctx, &memoryStore{objects: map[string][]byte{}}, roots, "b2", "bucket", "vault/a.xml")
	if err != nil {
		t.Fatal(err)
	}
	if published || !got.Mapped || got.Prefix != "derived/a.xml/" {
		t.Errorf("an underived source must resolve to the mapped location, got %+v published=%v", got, published)
	}
}

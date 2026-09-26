// Byline: Claude Code · Fable 5.1 · 2026-09-20
package objectstores

import "testing"

func TestParseStoresAcceptsAnyS3CompatibleSchemeAndRejectsUnsafeOnes(t *testing.T) {
	stores, err := ParseStores(`{"b2": "/run/secrets/casebible-b2.json", "wasabi": "/run/secrets/wasabi.json"}`)
	if err != nil {
		t.Fatal(err)
	}
	if got := stores.Schemes(); len(got) != 2 || got[0] != "b2" || got[1] != "wasabi" {
		t.Fatalf("schemes = %v", got)
	}
	if !stores.Has("B2") || stores.Has("r2") {
		t.Fatalf("Has is wrong: %v", stores)
	}
	if empty, err := ParseStores("  "); err != nil || len(empty) != 0 {
		t.Fatalf("empty input: %v %v", empty, err)
	}
	for _, bad := range []string{
		`{"upload": "/x.json"}`, `{"s3": "/x.json"}`, `{"B2": "/x.json"}`, `{"b2": "relative.json"}`,
		`{"b2": " /padded.json"}`, `["b2"]`, `{"b-2": "/x.json"}`,
	} {
		if _, err := ParseStores(bad); err == nil {
			t.Errorf("accepted %s", bad)
		}
	}
}

func TestRootsMatchOnlyObjectsInsideAConfiguredPrefix(t *testing.T) {
	roots, err := ParseRoots(`[
		{"id": "b2-vault", "label": "B2 / Vault", "url": "b2://salem-data/consignatio/vault/v1/"},
		{"id": "r2-sorted", "label": "R2 / Sorted", "url": "r2://casebible-sorted/", "temporary": true}
	]`)
	if err != nil {
		t.Fatal(err)
	}
	for _, ok := range [][3]string{
		{"b2", "salem-data", "consignatio/vault/v1/Takeout/a b.xml"},
		{"B2", "salem-data", "consignatio/vault/v1/x"},
		{"r2", "casebible-sorted", "anything/at/all.pdf"},
	} {
		if _, matched := roots.Match(ok[0], ok[1], ok[2]); !matched {
			t.Errorf("did not match %v", ok)
		}
	}
	for _, no := range [][3]string{
		{"b2", "salem-data", "consignatio/vault/v1/"},         // the prefix itself is not an object
		{"b2", "salem-data", "consignatio/intake/secret.xml"}, // sibling prefix
		{"b2", "salem-data", "consignatio/vault/v10/x"},       // prefix must end at a slash
		{"b2", "other-bucket", "consignatio/vault/v1/x"},
		{"r2", "salem-data", "consignatio/vault/v1/x"}, // right bucket, wrong store
		{"r2", "casebible-raw", "x"},
	} {
		if _, matched := roots.Match(no[0], no[1], no[2]); matched {
			t.Errorf("matched %v", no)
		}
	}
	if root, _ := roots.Match("b2", "salem-data", "consignatio/vault/v1/x"); root.ID != "b2-vault" || root.Prefix() != "consignatio/vault/v1/" || root.Temporary {
		t.Fatalf("root = %+v", root)
	}
}

func TestParseRootsRejectsMalformedRoots(t *testing.T) {
	for _, bad := range []string{
		`[{"id": "", "label": "x", "url": "b2://b/p/"}]`,
		`[{"id": "a", "label": "x", "url": "b2://b/no-trailing-slash"}]`,
		`[{"id": "a", "label": "x", "url": "b2://b/../p/"}]`,
		`[{"id": "a", "label": "x", "url": "b2:///p/"}]`,
		`[{"id": "a", "label": "x", "url": "b2://user@b/p/"}]`,
		`[{"id": "a", "label": "x", "url": "b2://b/p/?q=1"}]`,
		`[{"id": "a", "label": "x", "url": "b2://b/"}, {"id": "a", "label": "y", "url": "r2://c/"}]`,
		`{"id": "a"}`,
	} {
		if _, err := ParseRoots(bad); err == nil {
			t.Errorf("accepted %s", bad)
		}
	}
}

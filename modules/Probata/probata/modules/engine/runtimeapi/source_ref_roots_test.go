// Byline: Claude Code · Fable 5.1 · 2026-09-20
package runtimeapi

import (
	"testing"

	"github.com/Cursedpotential/probata/engine/objectstores"
)

// Source authority is the configured root list, whatever provider it names.
func TestValidateAuthorizedSourceRefAdmitsConfiguredRootsOnly(t *testing.T) {
	t.Setenv("PLATFORM_DEV_AUTH_BYPASS", "")
	t.Setenv(objectstores.RootsEnv, `[{"id": "b2-vault", "label": "B2 / Vault", "url": "b2://salem-data/consignatio/vault/v1/"}]`)

	kind, key, err := validateAuthorizedSourceRef("b2://salem-data/consignatio/vault/v1/Takeout/My%20Activity/sms.xml")
	if err != nil || kind != "b2" || key != "consignatio/vault/v1/Takeout/My Activity/sms.xml" {
		t.Fatalf("configured root rejected: kind=%q key=%q err=%v", kind, key, err)
	}
	for _, ref := range []string{
		"b2://salem-data/consignatio/intake/x.xml",        // sibling prefix
		"b2://salem-data/consignatio/vault/v1/../../x",    // traversal
		"b2://salem-data/consignatio/vault/v1/",           // the prefix is not an object
		"b2://other-bucket/consignatio/vault/v1/x.xml",    // other bucket
		"b2://salem-data/consignatio/vault/v1/x.xml?v=1",  // query
		"b2://user@salem-data/consignatio/vault/v1/x.xml", // userinfo
	} {
		if _, _, err := validateAuthorizedSourceRef(ref); err == nil {
			t.Errorf("admitted %q", ref)
		}
	}

	t.Setenv(objectstores.RootsEnv, "")
	if _, _, err := validateAuthorizedSourceRef("b2://salem-data/consignatio/vault/v1/x.xml"); err == nil {
		t.Error("admitted a b2 ref with no configured roots")
	}
	if _, _, err := validateAuthorizedSourceRef("r2://casebible-sorted/x.xml"); err != nil {
		t.Errorf("existing Case Bible Sorted authority regressed: %v", err)
	}
}

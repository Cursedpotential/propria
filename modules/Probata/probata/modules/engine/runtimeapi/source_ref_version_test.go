// Byline: Codex / 2026-10-06.
package runtimeapi

import (
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/objectstores"
)

// TestSourceVersionAdmissionPreservesAuthority exercises real root checks with exact version selectors.
// Inputs: allowed and denied object URIs. Outputs: test assertions. Effects: scoped test environment.
// Choose to prevent removing a provenance pin merely to pass HTTP admission.
func TestSourceVersionAdmissionPreservesAuthority(t *testing.T) {
	t.Setenv("PLATFORM_DEV_AUTH_BYPASS", "")
	t.Setenv(objectstores.RootsEnv, `[{"id":"b2-vault","label":"B2","url":"b2://salem-data/consignatio/vault/v1/"}]`)
	base := "b2://salem-data/consignatio/vault/v1/conversations.json"
	for _, ref := range []string{base, base + "?versionId=retained%2B%2Fversion"} {
		kind, key, err := validateAuthorizedSourceRef(ref)
		if err != nil || kind != "b2" || key != "consignatio/vault/v1/conversations.json" {
			t.Fatalf("valid exact source rejected: kind=%q key=%q err=%v", kind, key, err)
		}
	}
	for _, ref := range []string{
		base + "?", base + "?versionId=", base + "?versionId=null",
		base + "?versionId=a&versionId=b", base + "?versionId=a&other=b",
		base + "?other=a", base + "?versionId=%ZZ", base + "?versionId=%0A",
		base + "?versionId=%00", base + "?versionId=" + strings.Repeat("x", 2049),
		base + "?versionId=valid#fragment",
		"b2://other-bucket/consignatio/vault/v1/x?versionId=valid",
		"b2://salem-data/outside/x?versionId=valid",
		"b2://salem-data/consignatio/vault/v1/../../x?versionId=valid",
		"b2://user@salem-data/consignatio/vault/v1/x?versionId=valid",
		"upload://" + strings.Repeat("a", 64) + "?versionId=valid",
	} {
		if _, _, err := validateAuthorizedSourceRef(ref); err == nil {
			t.Errorf("invalid or unauthorized versioned URI admitted: %q", ref)
		}
	}
	t.Setenv(objectstores.RootsEnv, "")
	if _, _, err := validateAuthorizedSourceRef(base + "?versionId=valid"); err == nil {
		t.Fatal("version selector granted authority without an admitted B2 root")
	}
}

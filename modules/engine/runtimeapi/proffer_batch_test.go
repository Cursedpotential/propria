// Byline: Claude Code · Opus 5 · 2026-09-21
//
// Folder-locator authority for batch-by-folder intake. A batch may not name a
// bucket outside the configured source roots, exactly as a single start may
// not. Unit test: SOURCE_ROOTS_JSON is set in-process, nothing is listed.

package runtimeapi

import (
	"testing"

	"github.com/Cursedpotential/probata/engine/objectstores"
)

func TestValidateAuthorizedFolderRef(t *testing.T) {
	t.Setenv(objectstores.RootsEnv,
		`[{"id":"b2-vault","label":"B2 / Vault","url":"b2://salem-data/consignatio/vault/v1/"}]`)

	for name, probe := range map[string]struct {
		ref        string
		wantPrefix string
	}{
		"folder inside the root":     {"b2://salem-data/consignatio/vault/v1/sms/", "consignatio/vault/v1/sms/"},
		"trailing slash is added":    {"b2://salem-data/consignatio/vault/v1/sms", "consignatio/vault/v1/sms/"},
		"a space in the folder name": {"b2://salem-data/consignatio/vault/v1/sms%20/", "consignatio/vault/v1/sms /"},
		"the root itself":            {"b2://salem-data/consignatio/vault/v1/", "consignatio/vault/v1/"},
	} {
		scheme, bucket, prefix, err := validateAuthorizedFolderRef(probe.ref)
		if err != nil {
			t.Fatalf("%s: validateAuthorizedFolderRef(%q) error = %v", name, probe.ref, err)
		}
		if scheme != "b2" || bucket != "salem-data" || prefix != probe.wantPrefix {
			t.Errorf("%s: got %s/%s/%s, want prefix %q", name, scheme, bucket, prefix, probe.wantPrefix)
		}
	}

	for name, ref := range map[string]string{
		"unconfigured bucket": "b2://someone-elses-bucket/vault/",
		"unconfigured scheme": "r2://salem-data/consignatio/vault/v1/sms/",
		"outside the root":    "b2://salem-data/elsewhere/",
		"path traversal":      "b2://salem-data/consignatio/vault/v1/../../",
		"no bucket":           "b2:///vault/",
		"not a locator":       "consignatio/vault/v1/",
		"query string":        "b2://salem-data/consignatio/vault/v1/?list=1",
		"empty":               "",
	} {
		if _, _, _, err := validateAuthorizedFolderRef(ref); err == nil {
			t.Errorf("%s: validateAuthorizedFolderRef(%q) must be refused", name, ref)
		}
	}
}

// One at a time is the owner's instruction and the fallback for every
// unreadable or out-of-range configured value.
func TestBatchMaxInFlightFromEnv(t *testing.T) {
	for raw, want := range map[string]int{
		"": 1, "  ": 1, "0": 1, "-3": 1, "many": 1, "17": 1,
		"1": 1, "4": 4, "16": 16,
	} {
		if got := batchMaxInFlightFromEnv(raw); got != want {
			t.Errorf("batchMaxInFlightFromEnv(%q) = %d, want %d", raw, got, want)
		}
	}
}

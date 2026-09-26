// Byline: Claude Code · Fable 5.1 · 2026-09-20
package postgres

import (
	"testing"

	"github.com/Cursedpotential/probata/engine/objectstores"
)

// A store becomes DuckDB-readable by being configured, not by being named in code.
func TestDuckDBSourceURLTranslatesAnyConfiguredStoreToS3(t *testing.T) {
	t.Setenv(objectstores.StoresEnv, `{"b2": "/run/secrets/casebible-b2.json", "wasabi": "/run/secrets/wasabi.json"}`)
	for locator, want := range map[string]string{
		"b2://salem-data/consignatio/vault/v1/sms.xml": "s3://salem-data/consignatio/vault/v1/sms.xml",
		"wasabi://bucket/key.csv":                      "s3://bucket/key.csv",
		"r2://nexus/x.xml":                             "s3://nexus/x.xml",
	} {
		got, err := duckDBSourceURL(locator)
		if err != nil || got != want {
			t.Errorf("duckDBSourceURL(%q) = %q, %v; want %q", locator, got, err, want)
		}
	}
	for _, locator := range []string{"upload://abc", "file:///sealed/source.xml", "gcs://bucket/key"} {
		if _, err := duckDBSourceURL(locator); err == nil {
			t.Errorf("expected %q to fail closed", locator)
		}
	}
}

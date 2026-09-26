// Byline: Claude Code · Opus 5.5 · 2026-09-25

package profferworker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func clearCatalogEnvironment(t *testing.T) {
	t.Helper()
	for _, name := range []string{"INTAKE_DISCOVERY_PG_PASSWORD_FILE", "INTAKE_DISCOVERY_PG_HOST", "INTAKE_DISCOVERY_PG_PORT",
		"INTAKE_DISCOVERY_PG_DATABASE", "INTAKE_DISCOVERY_PG_USER", "INTAKE_DISCOVERY_OBJECT_STORE"} {
		t.Setenv(name, "")
	}
}

// The catalog is optional: without its password file the worker starts and
// only repair.find_other_version reports that it is not configured.
func TestLoadConfigLeavesTheCatalogOffWhenUnconfigured(t *testing.T) {
	setWorkerEnvironment(t)
	clearCatalogEnvironment(t)
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.Catalog.Enabled {
		t.Fatal("the catalog was enabled without a password file")
	}
}

func TestLoadConfigReadsTheCatalogInTheWorkbenchConvention(t *testing.T) {
	setWorkerEnvironment(t)
	clearCatalogEnvironment(t)
	password := filepath.Join(t.TempDir(), "metabase-ro")
	if err := os.WriteFile(password, []byte("catalog-password\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("INTAKE_DISCOVERY_PG_PASSWORD_FILE", password)
	t.Setenv("INTAKE_DISCOVERY_PG_HOST", "fgz1n7useplhk0t91uk7k1aw")
	t.Setenv("INTAKE_DISCOVERY_PG_PORT", "5432")
	t.Setenv("INTAKE_DISCOVERY_OBJECT_STORE", "b2://salem-data/")
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	catalog := cfg.Catalog
	if !catalog.Enabled || catalog.Password != "catalog-password" || catalog.Database != "casebible" || catalog.User != "metabase_ro" ||
		catalog.Port != 5432 || catalog.ObjectScheme != "b2" || catalog.ObjectBucket != "salem-data" {
		t.Fatalf("catalog config = %+v", catalog)
	}
}

func TestLoadConfigRejectsAHalfConfiguredCatalog(t *testing.T) {
	password := filepath.Join(t.TempDir(), "metabase-ro")
	if err := os.WriteFile(password, []byte("catalog-password\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for name, env := range map[string]map[string]string{
		"no host":         {"INTAKE_DISCOVERY_PG_PORT": "5432", "INTAKE_DISCOVERY_OBJECT_STORE": "b2://salem-data"},
		"no port":         {"INTAKE_DISCOVERY_PG_HOST": "db", "INTAKE_DISCOVERY_OBJECT_STORE": "b2://salem-data"},
		"no object store": {"INTAKE_DISCOVERY_PG_HOST": "db", "INTAKE_DISCOVERY_PG_PORT": "5432"},
		"store with path": {"INTAKE_DISCOVERY_PG_HOST": "db", "INTAKE_DISCOVERY_PG_PORT": "5432", "INTAKE_DISCOVERY_OBJECT_STORE": "b2://salem-data/vault"},
	} {
		setWorkerEnvironment(t)
		clearCatalogEnvironment(t)
		t.Setenv("INTAKE_DISCOVERY_PG_PASSWORD_FILE", password)
		for key, value := range env {
			t.Setenv(key, value)
		}
		if _, err := LoadConfig(); err == nil || !strings.Contains(err.Error(), "INTAKE_DISCOVERY") {
			t.Fatalf("%s: LoadConfig() error = %v", name, err)
		}
	}
}

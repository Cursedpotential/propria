// Byline: Codex · GPT-6 · 2026-10-11
package discoveryconfig

import "testing"

// TestDisabledDiscoveryIgnoresOptionalDependencies verifies independent feature admission.
func TestDisabledDiscoveryIgnoresOptionalDependencies(t *testing.T) {
	t.Setenv("ADAPTIVE_DISCOVERY_ENABLED", "false")
	t.Setenv("ADAPTIVE_DISCOVERY_DATASETS_JSON", "malformed")
	t.Setenv("ADAPTIVE_DISCOVERY_MODEL_CONFIG_JSON", "malformed")
	cfg, err := FromEnvironment()
	if err != nil || cfg.Enabled || len(cfg.Datasets) != 0 {
		t.Fatalf("disabled discovery consulted optional setup: %+v %v", cfg, err)
	}
}

// TestEnabledDiscoveryRequiresExplicitSetup verifies it cannot silently select sources or models.
func TestEnabledDiscoveryRequiresExplicitSetup(t *testing.T) {
	t.Setenv("ADAPTIVE_DISCOVERY_ENABLED", "true")
	t.Setenv("ADAPTIVE_DISCOVERY_ROOT", "relative")
	if _, err := FromEnvironment(); err == nil {
		t.Fatal("enabled discovery accepted implicit storage")
	}
}

// Byline: Codex · GPT-6.1-sol · 2026-10-08.
package main

import (
	"bytes"
	"strings"
	"testing"
)

// TestApprovedSchemaNeedsNoEnvironment verifies the operator's static SQL emission.
// Inputs: no bundle and deliberately absent analytical credentials. Output: SQL assertions.
// Effects: environment isolated to this test and memory-only stdout; no database or network.
// Choose for the CLI boundary rather than duplicating the schema generator's table contract.
func TestApprovedSchemaNeedsNoEnvironment(t *testing.T) {
	for _, key := range []string{"ANALYSIS_SURREAL_URL", "ANALYSIS_SURREAL_USER", "ANALYSIS_SURREAL_PASSWORD_FILE", "ANALYSIS_SURREAL_NAMESPACE", "ANALYSIS_SURREAL_DATABASE"} {
		t.Setenv(key, "")
	}
	var out bytes.Buffer
	if err := run([]string{"approved-schema"}, &out); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.String()) == "" || !strings.Contains(out.String(), "DEFINE TABLE") {
		t.Fatal("approved-schema must emit nonempty schema SQL without mounted inputs")
	}
}

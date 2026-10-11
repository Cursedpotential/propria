package proffer

import (
	"strings"
	"testing"
)

// TestNativeAIResultNeedsPersistedBundleHash guards the native source-only Activity boundary.
// Inputs: a synthetic prepared result with exact source pins. Outputs: hash acceptance or rejection.
// Effects: none. Choose to catch missing or malformed receipts before the next Activity is scheduled.
func TestNativeAIResultNeedsPersistedBundleHash(t *testing.T) {
	req := AIContentRequest{RequestID: "req", OperatingMode: "REAL", MatterID: "matter", CourtCaseID: "case",
		SourceVersionID: "source", OriginalRef: "original", NativeSourceOnly: true, MaxRecords: 1, MaxChunks: 1, MaxModelCalls: 1}
	out := AIContentResult{RequestID: req.RequestID, OperatingMode: req.OperatingMode, MatterID: req.MatterID,
		CourtCaseID: req.CourtCaseID, SourceVersionID: req.SourceVersionID, Stage: "prepared", BundleRef: "bundle",
		BundleSHA256: strings.Repeat("a", 64), SourceObjectID: "original", SourceSHA256: strings.Repeat("b", 64)}
	if err := out.validate(req, "prepared", true); err != nil {
		t.Fatalf("valid persisted hash rejected: %v", err)
	}
	out.BundleSHA256 = ""
	if err := out.validate(req, "prepared", true); err == nil {
		t.Fatal("native result without persisted bundle hash admitted")
	}
	out.BundleSHA256 = strings.Repeat("A", 64)
	if err := out.validate(req, "prepared", true); err == nil {
		t.Fatal("noncanonical bundle hash admitted")
	}
	out.BundleSHA256 = strings.Repeat("a", 64)
	out.SourceSHA256 = strings.Repeat("B", 64)
	if err := out.validate(req, "prepared", true); err == nil {
		t.Fatal("noncanonical retained original hash admitted")
	}
	out.SourceSHA256 = strings.Repeat("b", 64)
	out.SourceVersionID = "other"
	if err := out.validate(req, "prepared", true); err == nil {
		t.Fatal("changed source version admitted")
	}
}

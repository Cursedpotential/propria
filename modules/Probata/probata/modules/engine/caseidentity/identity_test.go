// Byline: Codex · GPT-5 · 2026-10-05
package caseidentity

import (
	"errors"
	"testing"
)

func TestOperatingModesDoNotSelectDifferentCases(t *testing.T) {
	for raw, want := range map[string]Mode{"": ModeLive, "LIVE": ModeLive, "REAL": ModeLive, "DEV": ModeDev, "TEST": ModeDev} {
		mode, err := ParseMode(raw)
		if err != nil || mode != want {
			t.Fatalf("%q: %s %v", raw, mode, err)
		}
		if !AdmittedIdentity(AuthoritativeMatterID, AuthoritativeCourtCaseID) {
			t.Fatal("approved pair not admitted")
		}
	}
	for _, pair := range [][2]string{{"deadbeef-dead-beef-dead-beefdeadbeef", "cafebabe-cafe-babe-cafe-babecafebabe"}, {AuthoritativeMatterID, "cafebabe-cafe-babe-cafe-babecafebabe"}, {"11111111-1111-1111-1111-111111111111", AuthoritativeCourtCaseID}} {
		if AdmittedIdentity(pair[0], pair[1]) {
			t.Fatal("unapproved pair admitted", pair)
		}
	}
	if RequireCanonicalWrite(ModeLive) != nil || !errors.Is(RequireCanonicalWrite(ModeDev), ErrDevWrite) || !errors.Is(RequireCanonicalWrite(""), ErrOperatingModeUnknown) {
		t.Fatal("write policy is not fail closed")
	}
}

func TestStorageCompatibilityLabelsAreNotOperatingModes(t *testing.T) {
	for mode, want := range map[Mode]string{ModeLive: "REAL", ModeDev: "TEST"} {
		got, err := StoredMode(mode)
		if err != nil || got != want {
			t.Fatalf("%s: %s %v", mode, got, err)
		}
	}
	if _, err := StoredMode(""); err == nil {
		t.Fatal("unknown storage mode admitted")
	}
}

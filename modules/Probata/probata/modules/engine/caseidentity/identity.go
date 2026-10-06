// Byline: Codex · GPT-5 · 2026-10-05
package caseidentity

import (
	"errors"
	"strings"
)

// Authoritative IDs preserve the owner's 2026-10-01 approved registry identity.
// Provenance: sql/bootstrap/case_registry_live_identity_20261001.json and its
// approved import receipt. Both operating contexts keep these exact IDs.
const (
	AuthoritativeMatterID    = "01a0f751-e07b-75cc-9ad5-63ad9449a8ba"
	AuthoritativeCourtCaseID = "01a0f751-e07b-76a1-a738-eb3e3aa3e68c"
)

var ErrDevWrite = errors.New("DEV writes require a verified isolated workspace; canonical writes are unavailable")
var ErrOperatingModeUnknown = errors.New("the operation has no verified operating_mode; canonical writes are unavailable")

// AdmittedIdentity verifies the one approved matter and court-case pair.
//
// Inputs: registry IDs. Outputs: admission only, never mode. Side effects: none.
// Use before accepting client or persisted case scope.
func AdmittedIdentity(matterID, courtCaseID string) bool {
	return strings.EqualFold(strings.TrimSpace(matterID), AuthoritativeMatterID) &&
		strings.EqualFold(strings.TrimSpace(courtCaseID), AuthoritativeCourtCaseID)
}

// RequireCanonicalWrite rejects Dev and unverifiable operating contexts.
//
// Inputs: canonical mode. Outputs: nil only for LIVE. Side effects: none.
// Use before persistence or dispatch; no auth/feature flag grants isolation.
func RequireCanonicalWrite(mode Mode) error {
	switch mode {
	case ModeLive:
		return nil
	case ModeDev:
		return ErrDevWrite
	default:
		return ErrOperatingModeUnknown
	}
}

// StoredMode preserves legacy investigation SQL labels without selecting a case.
//
// Inputs: canonical mode. Outputs: SQL value. Side effects: none.
// Compatibility debt: ops.legal_investigation_request still CHECKs TEST/REAL;
// remove this adapter only after a separately approved schema transition.
func StoredMode(mode Mode) (string, error) {
	switch mode {
	case ModeLive:
		return "REAL", nil
	case ModeDev:
		return "TEST", nil
	default:
		return "", ErrOperatingModeUnknown
	}
}

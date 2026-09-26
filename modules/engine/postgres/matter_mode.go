// Byline: Claude Code · Opus 5.5 · 2026-09-25

package postgres

import "strings"

// MatterModeForIdentity names the Workbench mode a matter/court-case identity
// belongs to, from the engine's own two admitted identities — the same
// constants ProbeProfferSchema admits (D-125, D-126): the pre-launch DEV
// sentinel is TEST (the Workbench's TEST matter), the authoritative go-live
// identity is REAL. Any other identity is neither, and callers must fail
// closed rather than guess.
func MatterModeForIdentity(matterID, courtCaseID string) (string, bool) {
	matter, court := strings.ToLower(strings.TrimSpace(matterID)), strings.ToLower(strings.TrimSpace(courtCaseID))
	switch {
	case matter == devMatterID && court == devCourtCaseID:
		return "TEST", true
	case matter == authoritativeMatterID && court == authoritativeCourtCaseID:
		return "REAL", true
	}
	return "", false
}

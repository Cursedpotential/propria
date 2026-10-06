// Byline: Codex · GPT-5 · 2026-10-05 (single-case operating contract)
// Byline: Claude Code · Opus 5.5 · 2026-09-25

package postgres

import "github.com/Cursedpotential/probata/engine/caseidentity"

// AdmittedCaseIdentity validates the authoritative pair without deriving mode.
// Inputs: matter/court-case IDs. Outputs: admission only. Side effects: none.
// Use for scope checks; the Oct 5 owner ruling supersedes D-126 case selection.
func AdmittedCaseIdentity(matterID, courtCaseID string) bool {
	return caseidentity.AdmittedIdentity(matterID, courtCaseID)
}

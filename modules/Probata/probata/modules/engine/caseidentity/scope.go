// Byline: Codex · GPT-6.1-sol · 2026-10-06
package caseidentity

import (
	"context"
	"time"
)

// ScopeReadTimeout bounds fresh identity approval before a caller's five-second timeout.
const ScopeReadTimeout = 2 * time.Second

// ScopeView contains only the registry identity required to approve case scope.
// Inputs: authoritative registry rows. Outputs: the existing nested identity JSON shape.
// Side effects: none. Choose over View when people, history and counts are not needed.
type ScopeView struct {
	Mode      Mode           `json:"mode"`
	Matter    ScopeMatter    `json:"matter"`
	CourtCase ScopeCourtCase `json:"court_case"`
}

// ScopeMatter carries the approved matter's registry ID without descriptive fields.
type ScopeMatter struct {
	ID string `json:"id"`
}

// ScopeCourtCase carries the approved court registry ID and its actual matter link.
type ScopeCourtCase struct {
	ID       string `json:"id"`
	MatterID string `json:"matter_id"`
}

// ScopeReader reads a fresh bounded registry identity without loading the whole Case page.
// Inputs: request context and operating mode. Outputs: authoritative pair or an error.
// Side effects: read-only database I/O; never mint or cache an approval.
// Implement separately from Store so existing full-page consumers remain compatible.
type ScopeReader interface {
	ReadScope(context.Context, Mode) (ScopeView, error)
}

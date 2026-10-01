// Byline: Claude Code · Opus 5.5 · 2026-10-01

package runtimeapi

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// validateOptionalPersonIDs checks the explicit first-party context identity
// (D04) a start request may carry. Both are optional here: whether a run
// needs them is decided by propose_first_party_context_activity, which fails
// loudly when a generation holds messages and either is absent. Present
// values must be UUIDs; they are never derived or defaulted.
func validateOptionalPersonIDs(ownerPersonID, perspectivePersonID string) error {
	for name, value := range map[string]string{"owner_person_id": ownerPersonID, "perspective_person_id": perspectivePersonID} {
		if strings.TrimSpace(value) == "" {
			continue
		}
		if _, err := uuid.Parse(strings.TrimSpace(value)); err != nil {
			return fmt.Errorf("%s must be a UUID", name)
		}
	}
	return nil
}

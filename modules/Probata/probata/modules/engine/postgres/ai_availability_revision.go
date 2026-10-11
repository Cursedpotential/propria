package postgres

import (
	"encoding/json"

	"github.com/Cursedpotential/probata/engine/extraction/entities"
	"github.com/Cursedpotential/probata/engine/extraction/service"
)

// validateAIAvailabilityRevision admits only a new owner assertion against the locked current receipt.
// Inputs: locked state, current decision and receipt, requested correction. Outputs: conflict or success.
// Effects: none; choose after exact replay checks and before appending the review ledger.
func validateAIAvailabilityRevision(state, current string, receipt []byte, c *service.AIAvailabilityCorrection) error {
	if c == nil {
		if state != "pending" && state != "needs_info" {
			return entities.ErrConflict
		}
		return nil
	}
	if current == "" {
		if state != "pending" || c.SupersedesRef != "" || c.Action != "correct" {
			return entities.ErrConflict
		}
		return nil
	}
	if c.SupersedesRef != current {
		return entities.ErrConflict
	}
	if c.Action == "retract" {
		var prior struct {
			MetadataCorrection *service.AIAvailabilityCorrection `json:"metadata_correction"`
		}
		if json.Unmarshal(receipt, &prior) != nil || prior.MetadataCorrection == nil || prior.MetadataCorrection.Action != "correct" {
			return entities.ErrConflict
		}
	}
	return nil
}

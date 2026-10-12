// Byline: Codex · GPT-5 · 2026-10-05
package postgres

import (
	"context"
	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/runtimeapi/previewmodel"
	"github.com/google/uuid"
)

// requirePreviewWrite admits only an explicit LIVE initial receipt whose exact
// approved pair agrees with the registered source. Inputs: durable handle.
// Outputs: admission error. Effects: one read, no transaction or mutation.
// Overlay stores call this before beginning any persistence, including retries.
func requirePreviewWrite(ctx context.Context, q rowQuerier, handle string) error {
	var binding previewmodel.Binding
	var detail string
	var versionID *uuid.UUID
	if err := q.QueryRow(ctx, `
		SELECT COALESCE((SELECT detail FROM context.proffer_preview_event
		    WHERE preview_handle=b.preview_handle AND (event_id=0 OR (event_type='receipt_recorded' AND phase='operating_mode_repair')) ORDER BY event_id DESC LIMIT 1), ''), v.matter_id, v.court_case_id,v.id
		FROM context.proffer_preview_binding b
		LEFT JOIN LATERAL (SELECT matter_id, court_case_id,id FROM context.source_version
		    WHERE workflow_id=b.workflow_id ORDER BY version_ordinal DESC, created_at DESC LIMIT 1) v ON true
		WHERE b.preview_handle=$1`, handle).Scan(&detail, &binding.MatterID, &binding.CourtCaseID, &versionID); err != nil {
		return err
	}
	if versionID != nil {
		binding.SourceVersionID = *versionID
	}
	binding.OperatingMode = recordedOperatingMode(detail)
	applyBindingAdmission(&binding, detail)
	return caseidentity.RequireCanonicalWrite(caseidentity.Mode(binding.OperatingMode))
}

package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/Cursedpotential/probata/engine/humancontext"
)

// HumanSourceView exposes genuine source identity and native stage refs without legacy generation IDs.
// It is metadata only; native messages and attachments remain in external immutable bundles.
type HumanSourceView struct {
	Handle           string                `json:"preview_handle"`
	SourceVersionRef string                `json:"source_version_ref"`
	SourceRef        string                `json:"source_ref"`
	WorkflowID       string                `json:"workflow_id"`
	CreatedAt        time.Time             `json:"created_at"`
	Stages           []humancontext.Result `json:"stages"`
	ManifestRef      string                `json:"manifest_ref,omitempty"`
}

// View resolves a real registered human source and its durable extraction/index/readback stage references.
// Inputs: actual source-version UUID. Outputs: native handle and metadata; effects: read-only SQL.
// Pick for the additive context viewer, not ProfferPreviewStore's normalized-generation projection.
func (s *HumanSourceMetadata) View(ctx context.Context, version string) (HumanSourceView, error) {
	view := HumanSourceView{Handle: "human-" + version, SourceVersionRef: version, Stages: []humancontext.Result{}}
	if err := s.DB.QueryRow(ctx, `SELECT source.source_key,version.workflow_id,version.acquired_at FROM context.source_version version JOIN context.source source ON source.id=version.source_id WHERE version.id=$1::uuid AND version.declared_format='smsbackuprestore_xml'`, version).Scan(&view.SourceRef, &view.WorkflowID, &view.CreatedAt); err != nil {
		return view, err
	}
	rows, err := s.DB.Query(ctx, `SELECT metadata->'result' FROM context.source_metadata WHERE source_version_id=$1::uuid AND extractor_id LIKE 'context-human-%' ORDER BY generated_at`, version)
	if err != nil {
		return view, err
	}
	defer rows.Close()
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return view, err
		}
		var result humancontext.Result
		if err := json.Unmarshal(raw, &result); err != nil {
			return view, err
		}
		view.Stages = append(view.Stages, result)
		if result.Stage == "extracted" || result.Stage == "published" || result.Stage == "verified" {
			view.ManifestRef = result.ManifestRef
		}
	}
	if err := rows.Err(); err != nil {
		return view, err
	}
	if view.SourceRef == "" {
		return view, errors.New("human source unavailable")
	}
	return view, nil
}

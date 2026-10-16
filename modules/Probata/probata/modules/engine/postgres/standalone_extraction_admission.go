package postgres

import (
	"context"
	"fmt"
	"github.com/Cursedpotential/probata/engine/extraction/flow"
)

// StandaloneWorkerPreview checks the published projection required by the current extraction worker.
// Inputs: exact already-resolved source/run pins. Outputs: worker compatibility or read failure.
// Effects: SELECT only; use before starting standalone extraction/commit while saved artifact reads remain independent.
func (s *EntityExtractionStore) StandaloneWorkerPreview(ctx context.Context, run flow.RunRef) (bool, error) {
	var compatible bool
	err := s.db.QueryRow(ctx, `SELECT EXISTS (
  SELECT 1 FROM (
   SELECT source_version_id,normalized_generation_id FROM context.proffer_preview_snapshot
   WHERE preview_handle=$1 ORDER BY snapshot_seq DESC LIMIT 1
  ) snapshot WHERE snapshot.source_version_id=$2::uuid AND snapshot.normalized_generation_id=$3::uuid
 )`, run.PreviewHandle, run.SourceVersionID, run.GenerationID).Scan(&compatible)
	if err != nil {
		return false, fmt.Errorf("read extraction worker preview prerequisite: %w", err)
	}
	return compatible, nil
}

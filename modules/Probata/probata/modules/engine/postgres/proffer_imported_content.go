// Byline: Claude Code · Opus 5.5 · 2026-10-16
//
// FindImportedContent backs the duplicate-import guard on /reference-import/start
// (owner spec 2026-10-15): content that is already imported -- or whose import is
// still running -- in this matter is blocked, and the operator is pointed at the
// existing import. Identity is exact: the same source reference (a B2 key carries
// its versionId, so the same provider version is the same bytes) or, for uploads,
// the same SHA-256 of the retained original.
package postgres

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/Cursedpotential/probata/engine/runtimeapi/previewmodel"
)

// uploadContentDigest returns the SHA-256 an upload:// reference names, or nil.
func uploadContentDigest(sourceRef string) []byte {
	hexDigest, ok := strings.CutPrefix(strings.TrimSpace(sourceRef), "upload://")
	if !ok || len(hexDigest) != 64 {
		return nil
	}
	digest, err := hex.DecodeString(hexDigest)
	if err != nil {
		return nil
	}
	return digest
}

// FindImportedContent lists earlier imports of the same content: source versions
// in this matter (or recorded with no matter: the 2026-10-02 overnight imports
// predate matter stamping, and this platform holds one approved case) with the
// same source key or retained SHA-256, plus start records
// for the identical source reference whose run has not registered a version yet.
// Committed means the version published or was approved (the Workbench's "done").
func (s *ProfferPreviewStore) FindImportedContent(ctx context.Context, matterID uuid.UUID, sourceRef string) ([]previewmodel.ImportedContent, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("imported-content lookup requires a database")
	}
	sourceRef = strings.TrimSpace(sourceRef)
	if sourceRef == "" || matterID == uuid.Nil {
		return nil, errors.New("imported-content lookup requires a source reference and matter")
	}
	rows, err := s.db.Query(ctx, `
		SELECT version.workflow_id, source.source_key, version.acquired_at,
		       (EXISTS (SELECT 1 FROM context.activity_execution ae
		                JOIN context.activity_receipt ar ON ar.activity_execution_id = ae.id
		                WHERE ae.source_version_id = version.id
		                  AND ae.activity_name = 'publish_generation_activity' AND ar.status = 'success')
		        OR EXISTS (SELECT 1 FROM context.proffer_preview_snapshot sn
		                   JOIN context.proffer_preview_decision d ON d.preview_handle = sn.preview_handle AND d.approved
		                   WHERE sn.source_version_id = version.id)) AS committed
		FROM context.source_version version
		JOIN context.source source ON source.id = version.source_id
		JOIN context.retained_object object ON object.id = version.original_object_id
		WHERE (version.matter_id = $1::uuid OR version.matter_id IS NULL)
		  AND (source.source_key = $2 OR ($3::bytea IS NOT NULL AND object.content_sha256 = $3::bytea))
		UNION ALL
		SELECT binding.workflow_id, binding.source_ref, binding.created_at, false
		FROM context.proffer_preview_binding binding
		WHERE binding.source_ref = $2
		  AND NOT EXISTS (SELECT 1 FROM context.source_version v WHERE v.workflow_id = binding.workflow_id)
		ORDER BY 3
		LIMIT 50`, matterID, sourceRef, uploadContentDigest(sourceRef))
	if err != nil {
		return nil, fmt.Errorf("find imported content: %w", err)
	}
	defer rows.Close()
	var found []previewmodel.ImportedContent
	for rows.Next() {
		var item previewmodel.ImportedContent
		if err := rows.Scan(&item.WorkflowID, &item.SourceKey, &item.ImportedAt, &item.Committed); err != nil {
			return nil, fmt.Errorf("read imported content: %w", err)
		}
		found = append(found, item)
	}
	return found, rows.Err()
}

package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// UseBatchUnitManifestRoot binds the existing external derive scratch directory for package provenance validation.
// Inputs: configured absolute scratch root. Outputs: none. Effects: configures this repository before serving calls.
// Choose when enabling explicit package batching; ordinary source registrations need no manifest reader.
func (r *SourceLifecycleRepository) UseBatchUnitManifestRoot(root string) { r.unitManifestRoot = root }

// batchUnitRegistrationMetadata verifies original membership identity before any registration writes.
// Inputs: frozen manifest references and source registration scope. Outputs: compact pointer-only metadata.
// Effects: reads external inventory metadata; attachment paths and payloads remain in the external manifest.
func (r *SourceLifecycleRepository) batchUnitRegistrationMetadata(spec activities.SourceRegistrationSpec) ([]byte, error) {
	if spec.UnitRootRef == "" && spec.UnitManifestRef == "" && spec.UnitManifestSHA256 == "" && spec.UnitMemberCount == 0 {
		return nil, nil
	}
	if r.unitManifestRoot == "" || spec.UnitRootRef == "" || spec.UnitManifestRef == "" || len(spec.UnitManifestSHA256) != 64 || spec.UnitMemberCount < 1 {
		return nil, errors.New("unit registration requires configured manifest root and complete frozen inventory identity")
	}
	manifest, _, err := activities.ReadBatchUnitManifest(r.unitManifestRoot, spec.UnitManifestRef, spec.UnitManifestSHA256)
	if err != nil {
		return nil, err
	}
	if manifest.Request.UnitRoot != string(spec.UnitRootRef) || manifest.PrimaryRef != spec.AcquisitionRef || manifest.SelectedFormat != spec.DeclaredFormat || len(manifest.Members) != spec.UnitMemberCount || manifest.Request.MatterID != spec.MatterID || manifest.Request.CourtCaseID != spec.CourtCaseID || !strings.HasPrefix(spec.RequestID, manifest.Request.BatchID+"-") {
		return nil, errors.New("frozen package inventory does not own the registered primary or intake scope")
	}
	return json.Marshal(map[string]any{"contract": manifest.Contract, "unit_root": spec.UnitRootRef, "unit_manifest_ref": spec.UnitManifestRef, "unit_manifest_sha256": spec.UnitManifestSHA256, "member_count": spec.UnitMemberCount, "primary_ref": manifest.PrimaryRef, "detected_format": manifest.DetectedFormat, "selected_format": manifest.SelectedFormat, "signature": manifest.Signature})
}

// verifyBatchUnitRegistrationMetadata refuses retry ownership changes without rewriting preserved metadata.
// Inputs: successful source identity and expected frozen pointer metadata. Outputs: exact equality or error.
// Effects: SQL metadata read only; choose before returning an existing registration receipt.
func verifyBatchUnitRegistrationMetadata(ctx context.Context, tx pgx.Tx, id uuid.UUID, expected []byte) error {
	if len(expected) == 0 {
		return nil
	}
	var equal bool
	if err := tx.QueryRow(ctx, `SELECT count(*)=1 AND bool_and(metadata=$2::jsonb) FROM context.source_metadata WHERE source_version_id=$1::uuid AND metadata_class='record_native' AND extractor_id='batch-unit-inventory'`, id, expected).Scan(&equal); err != nil {
		return fmt.Errorf("verify frozen package provenance: %w", err)
	}
	if !equal {
		return errors.New("existing registration belongs to a different frozen package inventory")
	}
	return nil
}

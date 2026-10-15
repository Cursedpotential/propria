// Byline: Codex · GPT-6.1-sol · 2026-10-08.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var _ activities.ContextPackageMetadataStore = (*ContextSourceRegistrationStore)(nil)

// RecordContextPackage pins the verified native package without storing native AI records.
// Inputs: registered source identity and exact readback result. Outputs: error or idempotent metadata registration.
// Effects: one transaction in existing context/lifecycle tables; no DDL, custody-status change or catalog claim.
// Choose after physical package verification; an explicitly pending catalog remains a separate repair task.
func (s *ContextSourceRegistrationStore) RecordContextPackage(ctx context.Context, in proffer.ContextPackageRequest, result proffer.ContextPackageResult) error {
	return s.recordContextPackageMetadata(ctx, in, result, activities.AIContextPackageActivityName, "ai-context-package", "pending")
}

// RecordContextPackageCatalog appends independent catalog completion without replacing the physical pending receipt.
// Inputs: registered source and complete catalog/readback receipt pins. Outputs: idempotent compact metadata.
// Effects: existing metadata/lifecycle writes only; choose after narrow catalog row verification.
func (s *ContextSourceRegistrationStore) RecordContextPackageCatalog(ctx context.Context, in proffer.ContextPackageRequest, result proffer.ContextPackageResult) error {
	if result.CatalogReceiptRef == "" || len(result.CatalogReceiptSHA256) != 64 {
		return errors.New("catalog completion requires independent receipt pins")
	}
	return s.recordContextPackageMetadata(ctx, in, result, activities.AIContextCatalogActivityName, "ai-context-package-catalog", "complete")
}

// recordContextPackageMetadata appends one exact metadata phase using the existing lifecycle transaction.
// Inputs: phase identity and verified package pins. Outputs: idempotent registration or specific mismatch.
// Effects: existing context rows only; choose to preserve both physical and catalog receipt history.
func (s *ContextSourceRegistrationStore) recordContextPackageMetadata(ctx context.Context, in proffer.ContextPackageRequest, result proffer.ContextPackageResult, activityName, extractor, status string) error {
	versionID, err := uuid.Parse(in.SourceVersionRef)
	if err != nil || s == nil || s.db == nil || !result.Complete || result.Files < 4 || result.Files > 65 || result.Bytes <= 0 || result.Bytes > 129<<20 || result.CatalogStatus != status || len(result.ManifestSHA256) != 64 || len(result.ReceiptSHA256) != 64 || !strings.HasPrefix(result.ManifestRef, "b2://salem-data/consignatio/casevault/KnowledgeBase/ai-chats/_Incoming/") {
		return errors.New("native package metadata requires a registered source and complete exact readback pins")
	}
	metadata, err := json.Marshal(struct {
		Contract          string                       `json:"contract"`
		SourceRef         string                       `json:"source_ref"`
		ProviderVersionID string                       `json:"provider_version_id,omitempty"`
		PackageRef        string                       `json:"package_ref,omitempty"`
		Physical          proffer.ContextPackageResult `json:"physical"`
	}{"ai-context-package-v1", in.SourceRef, in.ProviderVersionID, in.PackageRef, result})
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			cleanup, cancel := lifecycleCleanup(ctx)
			defer cancel()
			_ = tx.Rollback(cleanup)
		}
	}()
	var sourceRef, workflowID string
	if err = tx.QueryRow(ctx, `SELECT s.source_key,v.workflow_id FROM context.source_version v JOIN context.source s ON s.id=v.source_id WHERE v.id=$1::uuid FOR UPDATE OF v`, versionID).Scan(&sourceRef, &workflowID); err != nil {
		return err
	}
	if sourceRef != in.SourceRef || workflowID != in.WorkflowID {
		return errors.New("native package metadata differs from actual registered source/workflow")
	}
	var registrationMatches bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM context.source_metadata WHERE source_version_id=$1::uuid AND extraction_activity_receipt_id=$2::uuid AND extractor_id='context-source-registration' AND metadata->>'provider_version_id'=$3 AND metadata->>'package_ref'=$4 AND metadata->>'declared_format'=$5)`, versionID, in.RegistrationReceiptRef, in.ProviderVersionID, in.PackageRef, in.SourceFormat).Scan(&registrationMatches); err != nil {
		return err
	}
	if !registrationMatches {
		return errors.New("native package lineage does not match registration receipt")
	}
	executionID, err := lifecycleEnsureExecution(ctx, tx, versionID, in.WorkflowID, activityName, in.RequestID, nil)
	if err != nil {
		return err
	}
	_, receiptRef, found, err := successfulReceipt(ctx, tx, executionID, "source_version", versionID.String())
	if err != nil {
		return err
	}
	if found {
		var count int
		var same bool
		if err = tx.QueryRow(ctx, `SELECT count(*),COALESCE(bool_and(metadata=$3::jsonb),false) FROM context.source_metadata WHERE source_version_id=$1::uuid AND extraction_activity_receipt_id=$2::uuid AND extractor_id=$4`, versionID, string(receiptRef), metadata, extractor).Scan(&count, &same); err != nil {
			return err
		}
		if count != 1 || !same {
			return errors.New("native package retry changed exact package pins")
		}
	} else {
		now := s.clock().UTC()
		receiptID, e := insertSuccessReceipt(ctx, tx, executionID, 1, "source_version", versionID.String(), now)
		if e != nil {
			return e
		}
		if _, err = tx.Exec(ctx, `INSERT INTO context.source_metadata (source_version_id,metadata_class,metadata,extractor_id,extractor_version,extraction_activity_receipt_id,generated_at) VALUES ($1::uuid,'container',$2::jsonb,$3,'1.0.0',$4::uuid,$5)`, versionID, metadata, extractor, receiptID, now); err != nil {
			return fmt.Errorf("persist native package pins: %w", err)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	committed = true
	return nil
}

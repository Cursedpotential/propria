// Byline: Codex · GPT-6.1-sol · 2026-10-08.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const contextRegisterSourceActivityName = "context_register_source_activity"

var _ activities.ContextSourceRegistrationStore = (*ContextSourceRegistrationStore)(nil)

// ContextSourceRegistrationStore persists an unretained source and its native locator metadata.
// Inputs: the existing platform DB. Outputs: a registration adapter. Effects: SQL transactions only.
// Choose for context imports before parsing; SourceLifecycleRepository retains the separate custody path.
type ContextSourceRegistrationStore struct {
	db    DB
	clock func() time.Time
}

// NewContextSourceRegistrationStore constructs the no-hash context registration adapter.
// Inputs: existing platform DB. Outputs: a ready store or validation error. Effects: none.
// Choose for context source identity; it never resolves or retains source bytes.
func NewContextSourceRegistrationStore(db DB) (*ContextSourceRegistrationStore, error) {
	if db == nil {
		return nil, errors.New("context source registration requires a database")
	}
	return &ContextSourceRegistrationStore{db: db, clock: func() time.Time { return time.Now().UTC() }}, nil
}

type contextRegistrationMetadata struct {
	RequestID         string `json:"request_id"`
	WorkflowID        string `json:"workflow_id"`
	SourceRef         string `json:"source_ref"`
	ProviderVersionID string `json:"provider_version_id"`
	PackageRef        string `json:"package_ref"`
	SourceKind        string `json:"source_kind"`
	DeclaredFormat    string `json:"declared_format"`
	MatterID          string `json:"matter_id"`
	CourtCaseID       string `json:"court_case_id"`
	ClockBasis        string `json:"clock_basis"`
}

// RegisterContextSource creates or recovers one source version, success receipt and native coordinates.
// Inputs: actual workflow/request IDs, source pointer, native version/package/kind and optional case scope.
// Outputs: source-version and receipt references. Effects: one atomic SQL transaction, no source bytes or hashes.
// Choose before a context parser; RetainOriginal is neither called nor required here.
func (s *ContextSourceRegistrationStore) RegisterContextSource(ctx context.Context, req activities.ContextSourceRegistrationRequest) (activities.ContextSourceRegistrationResult, error) {
	if s == nil || s.db == nil {
		return activities.ContextSourceRegistrationResult{}, errors.New("context source registration store is unavailable")
	}
	var err error
	req, err = normalizeContextRegistration(req)
	if err != nil {
		return activities.ContextSourceRegistrationResult{}, err
	}
	metadata, err := json.Marshal(contextRegistrationMetadata{
		RequestID: req.RequestID, WorkflowID: req.WorkflowID, SourceRef: req.SourceRef,
		ProviderVersionID: req.ProviderVersionID, PackageRef: req.PackageRef,
		SourceKind: req.SourceKind, DeclaredFormat: req.DeclaredFormat,
		MatterID: req.MatterID, CourtCaseID: req.CourtCaseID,
		ClockBasis: "technical_registration_only_not_owner_knowledge",
	})
	if err != nil {
		return activities.ContextSourceRegistrationResult{}, fmt.Errorf("encode context registration metadata: %w", err)
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return activities.ContextSourceRegistrationResult{}, fmt.Errorf("begin context registration: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			cleanupCtx, cancel := lifecycleCleanup(ctx)
			defer cancel()
			_ = tx.Rollback(cleanupCtx)
		}
	}()
	if _, err := tx.Exec(ctx, `
		INSERT INTO context.source (source_key, provenance_class)
		VALUES ($1, 'unknown') ON CONFLICT (source_key) DO NOTHING`, req.SourceRef); err != nil {
		return activities.ContextSourceRegistrationResult{}, fmt.Errorf("register context source identity: %w", err)
	}
	var sourceID uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT id FROM context.source WHERE source_key=$1 FOR UPDATE`, req.SourceRef).Scan(&sourceID); err != nil {
		return activities.ContextSourceRegistrationResult{}, fmt.Errorf("lock context source identity: %w", err)
	}
	now := s.clock().UTC()
	canonicalMatterID, canonicalCourtCaseID := canonicalContextScope(req)
	if _, err := tx.Exec(ctx, `
		INSERT INTO context.source_version
			(source_id, version_ordinal, workflow_id, submission_idempotency_key,
			 declared_format, acquired_at, matter_id, court_case_id)
		SELECT $1::uuid, COALESCE(MAX(version_ordinal), 0) + 1, $2, $3, $4, $5, $6::uuid, $7::uuid
		FROM context.source_version WHERE source_id=$1::uuid
		ON CONFLICT DO NOTHING`, sourceID, req.WorkflowID, req.RequestID, req.DeclaredFormat,
		now, contextOptionalUUID(canonicalMatterID), contextOptionalUUID(canonicalCourtCaseID)); err != nil {
		return activities.ContextSourceRegistrationResult{}, fmt.Errorf("register context source version: %w", err)
	}
	var versionID uuid.UUID
	var sourceKey, workflowID, requestID, declaredFormat string
	var matterID, courtCaseID *string
	err = tx.QueryRow(ctx, `
		SELECT version.id, source.source_key, version.workflow_id,
		       version.submission_idempotency_key, version.declared_format,
		       version.matter_id::text, version.court_case_id::text
		FROM context.source_version version
		JOIN context.source source ON source.id=version.source_id
		WHERE version.workflow_id=$1 FOR UPDATE`, req.WorkflowID).Scan(
		&versionID, &sourceKey, &workflowID, &requestID, &declaredFormat, &matterID, &courtCaseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return activities.ContextSourceRegistrationResult{}, errors.New("context registration request/source is already owned by another workflow")
	}
	if err != nil {
		return activities.ContextSourceRegistrationResult{}, fmt.Errorf("verify context source version: %w", err)
	}
	if sourceKey != req.SourceRef || workflowID != req.WorkflowID || requestID != req.RequestID ||
		declaredFormat != req.DeclaredFormat || contextOptionalValue(matterID) != canonicalMatterID ||
		contextOptionalValue(courtCaseID) != canonicalCourtCaseID {
		return activities.ContextSourceRegistrationResult{}, errors.New("context registration workflow/request is owned by different source coordinates")
	}
	executionID, err := lifecycleEnsureExecution(ctx, tx, versionID, req.WorkflowID,
		contextRegisterSourceActivityName, req.RequestID, nil)
	if err != nil {
		return activities.ContextSourceRegistrationResult{}, err
	}
	resultRef, receiptRef, found, err := successfulReceipt(ctx, tx, executionID, "source_version", versionID.String())
	if err != nil {
		return activities.ContextSourceRegistrationResult{}, err
	}
	if found {
		var count int
		var matches bool
		if err := tx.QueryRow(ctx, `
			SELECT count(*), COALESCE(bool_and(metadata=$3::jsonb), false)
			FROM context.source_metadata
			WHERE source_version_id=$1::uuid AND extraction_activity_receipt_id=$2::uuid
			  AND metadata_class='record_native' AND extractor_id='context-source-registration'`,
			versionID, string(receiptRef), metadata).Scan(&count, &matches); err != nil {
			return activities.ContextSourceRegistrationResult{}, fmt.Errorf("verify context registration metadata: %w", err)
		}
		if count != 1 || !matches {
			return activities.ContextSourceRegistrationResult{}, errors.New("context registration retry changed native source coordinates")
		}
		if err := tx.Commit(ctx); err != nil {
			return activities.ContextSourceRegistrationResult{}, fmt.Errorf("commit recovered context registration: %w", err)
		}
		committed = true
		return activities.ContextSourceRegistrationResult{SourceVersionRef: string(resultRef), ReceiptRef: string(receiptRef), Status: "registered"}, nil
	}
	receiptID, err := insertSuccessReceipt(ctx, tx, executionID, 1, "source_version", versionID.String(), now)
	if err != nil {
		return activities.ContextSourceRegistrationResult{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO context.source_metadata
			(source_version_id, metadata_class, metadata, extractor_id, extractor_version,
			 extraction_activity_receipt_id, generated_at)
		VALUES ($1::uuid, 'record_native', $2::jsonb, 'context-source-registration', '1.0.0', $3::uuid, $4)`,
		versionID, metadata, receiptID, now); err != nil {
		return activities.ContextSourceRegistrationResult{}, fmt.Errorf("persist context native source coordinates: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return activities.ContextSourceRegistrationResult{}, fmt.Errorf("commit context source registration: %w", err)
	}
	committed = true
	return activities.ContextSourceRegistrationResult{SourceVersionRef: versionID.String(), ReceiptRef: receiptID.String(), Status: "registered"}, nil
}

// normalizeContextRegistration checks pointer and metadata bounds without parsing source bytes.
// Inputs: existing contract values. Output: validated value or error. Effects: none.
// Choose at this adapter boundary rather than reading or changing source content.
func normalizeContextRegistration(req activities.ContextSourceRegistrationRequest) (activities.ContextSourceRegistrationRequest, error) {
	for name, value := range map[string]string{
		"request_id": req.RequestID, "workflow_id": req.WorkflowID,
		"source_ref": req.SourceRef,
	} {
		if strings.TrimSpace(value) == "" || strings.ContainsAny(value, "\r\n\x00") {
			return req, fmt.Errorf("context registration requires bounded %s", name)
		}
	}
	if len(req.RequestID) > 512 || len(req.WorkflowID) > 512 || len(req.SourceRef) > 2048 ||
		len(req.SourceKind) > 128 || len(req.ProviderVersionID) > 512 || len(req.PackageRef) > 2048 ||
		len(req.MatterID) > 512 || len(req.CourtCaseID) > 512 ||
		strings.ContainsAny(req.ProviderVersionID+req.PackageRef+req.MatterID+req.CourtCaseID, "\r\n\x00") {
		return req, errors.New("context registration source coordinates exceed bounds")
	}
	if strings.TrimSpace(req.SourceKind) == "" {
		req.SourceKind = "unknown"
	}
	if strings.TrimSpace(req.DeclaredFormat) == "" {
		req.DeclaredFormat = "unknown"
	}
	if len(req.DeclaredFormat) > 128 || strings.ContainsAny(req.DeclaredFormat, "\r\n\x00") {
		return req, errors.New("context registration format exceeds bounds")
	}
	return req, nil
}

// canonicalContextScope supplies paired existing UUID coordinates or leaves native metadata unscoped.
// Inputs: existing contract values. Output: validated value or error. Effects: none.
// Choose at this adapter boundary rather than reading or changing source content.
func canonicalContextScope(req activities.ContextSourceRegistrationRequest) (string, string) {
	if req.MatterID == "" || req.CourtCaseID == "" {
		return "", ""
	}
	matter, matterErr := uuid.Parse(req.MatterID)
	courtCase, caseErr := uuid.Parse(req.CourtCaseID)
	if matterErr != nil || caseErr != nil {
		return "", ""
	}
	return matter.String(), courtCase.String()
}

// contextOptionalUUID maps absent optional scope to SQL NULL.
// Inputs: existing contract values. Output: validated value or error. Effects: none.
// Choose at this adapter boundary rather than reading or changing source content.
func contextOptionalUUID(value string) any {
	if value == "" {
		return nil
	}
	return value
}

// contextOptionalValue reads nullable scope without inventing an identity.
// Inputs: existing contract values. Output: validated value or error. Effects: none.
// Choose at this adapter boundary rather than reading or changing source content.
func contextOptionalValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// Byline: Claude Code · Opus 5 · 2026-09-20
//
// Durable side of derive_structured_text_activity: resolve the retained
// source's own object-store locator, and record one append-only receipt for
// the derivation. No schema change is required — context.activity_execution
// and context.activity_receipt already accept a new activity name, and the
// engine role holds INSERT+SELECT on both (verified read-only against the
// live database on 2026-09-20).

package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/stagegraph"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// nonObjectStoreSchemes are locators that name a worker-local or ingress
// copy. Derived objects are published BESIDE the original in its own store,
// so a source reachable only through one of these is refused rather than
// derived into some other location.
var nonObjectStoreSchemes = map[string]struct{}{
	"file": {}, "upload": {}, "http": {}, "https": {},
}

// DeriveStore implements both halves of the derive Activity's durable seam.
type DeriveStore struct {
	db    DB
	clock func() time.Time
}

func NewDeriveStore(db DB) (*DeriveStore, error) {
	if db == nil {
		return nil, errors.New("postgres derive store: database is required")
	}
	return &DeriveStore{db: db, clock: func() time.Time { return time.Now().UTC() }}, nil
}

// ResolveDeriveSource returns the object-store coordinate of the retained
// source. context.source.source_key carries the acquisition locator
// (b2://bucket/key on every live SMS source, checked 2026-09-20);
// context.retained_object.object_uri is the worker-host sealed copy and is
// used only when it is itself an object-store locator.
func (s *DeriveStore) ResolveDeriveSource(ctx context.Context, req proffer.StageRequest) (activities.DeriveSourceLocator, error) {
	sourceID, err := uuid.Parse(string(req.SourceVersionRef))
	if err != nil {
		return activities.DeriveSourceLocator{}, fmt.Errorf("derive source version reference: %w", err)
	}
	originalID, err := uuid.Parse(string(req.Refs["original"]))
	if err != nil {
		return activities.DeriveSourceLocator{}, fmt.Errorf("derive original reference: %w", err)
	}
	var sourceKey, objectURI, workflowID, status, declared string
	if err := s.db.QueryRow(ctx, `
		SELECT source.source_key, object.object_uri, version.workflow_id, version.status, version.declared_format
		FROM context.source_version version
		JOIN context.source source ON source.id = version.source_id
		JOIN context.retained_object object ON object.id = version.original_object_id
		WHERE version.id = $1::uuid AND version.original_object_id = $2::uuid`,
		sourceID, originalID,
	).Scan(&sourceKey, &objectURI, &workflowID, &status, &declared); err != nil {
		return activities.DeriveSourceLocator{}, fmt.Errorf("resolve derive source locator: %w", err)
	}
	if workflowID != req.RequestID || status != "retained" {
		return activities.DeriveSourceLocator{}, errors.New("derive source is not retained by this workflow")
	}
	if declared != req.DeclaredFormat {
		return activities.DeriveSourceLocator{}, errors.New("derive declared format does not match retained source")
	}
	for _, candidate := range []string{sourceKey, objectURI} {
		locator, ok := parseObjectStoreLocator(candidate)
		if ok {
			return locator, nil
		}
	}
	return activities.DeriveSourceLocator{}, errors.New(
		"derive requires an object-store source locator; this source is reachable only through a worker-local copy")
}

// parseObjectStoreLocator splits <scheme>://<bucket>/<key> and rejects the
// schemes that do not name a publishable object store.
func parseObjectStoreLocator(value string) (activities.DeriveSourceLocator, bool) {
	scheme, rest, found := strings.Cut(strings.TrimSpace(value), "://")
	if !found {
		return activities.DeriveSourceLocator{}, false
	}
	scheme = strings.ToLower(scheme)
	if _, blocked := nonObjectStoreSchemes[scheme]; blocked {
		return activities.DeriveSourceLocator{}, false
	}
	bucket, key, hasKey := strings.Cut(rest, "/")
	if scheme == "" || bucket == "" || !hasKey || key == "" {
		return activities.DeriveSourceLocator{}, false
	}
	return activities.DeriveSourceLocator{Scheme: scheme, Bucket: bucket, Key: key}, true
}

// PersistDerivedGeneration records one append-only derivation receipt and
// returns its compact result reference. It is idempotent on
// (source_version, activity_name, idempotency_key): a retried Activity
// returns the first receipt rather than writing a second.
func (s *DeriveStore) PersistDerivedGeneration(ctx context.Context, spec activities.DeriveReceiptSpec) (proffer.Ref, proffer.Ref, error) {
	sourceID, err := uuid.Parse(string(spec.SourceVersionRef))
	if err != nil {
		return "", "", fmt.Errorf("derive receipt source version reference: %w", err)
	}
	if strings.TrimSpace(spec.RequestID) == "" || strings.TrimSpace(spec.ManifestURI) == "" ||
		strings.TrimSpace(spec.ManifestSHA256) == "" || spec.Attempt < 1 {
		return "", "", errors.New("derive receipt requires request, manifest reference, digest and attempt")
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", "", err
	}
	rollback := true
	defer func() {
		if rollback {
			cleanup, cancel := boundedCleanup(ctx)
			defer cancel()
			_ = tx.Rollback(cleanup)
		}
	}()
	// The manifest object identifies the derivation exactly: same source, same
	// content, same key. Its digest is the idempotency coordinate.
	key := "derive-structured-text:" + spec.ManifestURI + ":" + spec.ManifestSHA256
	executionID, err := parserEnsureExecution(ctx, tx, sourceID, spec.RequestID, string(stagegraph.DeriveSMSThreads), key)
	if err != nil {
		return "", "", err
	}
	var priorReceipt uuid.UUID
	var priorResult []byte
	err = tx.QueryRow(ctx, `
		SELECT id, result_ref FROM context.activity_receipt
		WHERE activity_execution_id=$1 AND status='success' ORDER BY attempt LIMIT 1`, executionID).
		Scan(&priorReceipt, &priorResult)
	if err == nil {
		var prior struct {
			RefID string `json:"ref_id"`
		}
		if json.Unmarshal(priorResult, &prior) != nil || strings.TrimSpace(prior.RefID) == "" {
			return "", "", errors.New("stored derivation receipt lacks a result reference")
		}
		if err = tx.Commit(ctx); err != nil {
			return "", "", err
		}
		rollback = false
		return proffer.Ref(prior.RefID), proffer.Ref(priorReceipt.String()), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", "", err
	}

	receiptID, resultID := uuid.New(), uuid.New()
	now := s.clock()
	resultJSON, err := json.Marshal(map[string]any{
		"ref_kind": "derived_structured_text", "ref_id": resultID.String(),
		"source_locator": spec.SourceLocator, "manifest_uri": spec.ManifestURI,
		"manifest_sha256": spec.ManifestSHA256, "derived_prefix": spec.DerivedPrefix,
		"schema": spec.Schema, "records": spec.Records, "rejected": spec.Rejected,
		"media_objects": spec.MediaObjects, "chunk_count": spec.ChunkCount,
		"thread_count": spec.ThreadCount, "reused": spec.Reused,
	})
	if err != nil {
		return "", "", err
	}
	if _, err = tx.Exec(ctx, `
		INSERT INTO context.activity_receipt(id,activity_execution_id,attempt,status,started_at,completed_at,result_ref)
		VALUES($1,$2,$3::integer,'success',$4,$4,$5)`, receiptID, executionID, spec.Attempt, now, resultJSON); err != nil {
		return "", "", fmt.Errorf("persist derivation receipt: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return "", "", err
	}
	rollback = false
	return proffer.Ref(resultID.String()), proffer.Ref(receiptID.String()), nil
}

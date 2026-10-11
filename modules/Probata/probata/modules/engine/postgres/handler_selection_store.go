package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/parser"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/sourceformat"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const handlerSignatureReadLimit int64 = sourceformat.ReadLimit

// HandlerSelectionStore is the durable, content-backed implementation of the
// recommendation and validation Activities and the authenticated HTTP
// decision writer. Source bytes are inspected only through the retained
// original reference; declared format and filename are not detection inputs.
type HandlerSelectionStore struct {
	db      DB
	open    *Repository
	parsers *parser.Registry
	clock   func() time.Time
}

func NewHandlerSelectionStore(db DB, open ObjectOpener) (*HandlerSelectionStore, error) {
	return newHandlerSelectionStore(db, open, nil)
}

// NewHandlerSelectionStoreWithRegistry enables content recommendation. The
// exact decoder capability selected here is persisted as the candidate that
// parser selection and execution must later honor.
func NewHandlerSelectionStoreWithRegistry(db DB, open ObjectOpener, registry *parser.Registry) (*HandlerSelectionStore, error) {
	if registry == nil {
		return nil, errors.New("postgres handler selection store: parser registry is required")
	}
	return newHandlerSelectionStore(db, open, registry)
}

func newHandlerSelectionStore(db DB, open ObjectOpener, registry *parser.Registry) (*HandlerSelectionStore, error) {
	if db == nil {
		return nil, errors.New("postgres handler selection store: database is required")
	}
	repository, err := NewRepository(db, open)
	if err != nil {
		return nil, err
	}
	return &HandlerSelectionStore{db: db, open: repository, parsers: registry, clock: func() time.Time { return time.Now().UTC() }}, nil
}

// RecommendHandler derives one bounded candidate set from retained content and
// persists every reference returned to Temporal in one transaction.
// Inputs: durable source references and attempt. Output: recommendation or fixed credential exclusion error.
// Side effects: 8 MiB format read, independent whole-stream credential scan, persistence only after admission.
// Pick for handler selection; EOF scan completion covers this credential signature policy rather than every possible secret.
func (s *HandlerSelectionStore) RecommendHandler(ctx context.Context, req proffer.StageRequest, attempt int32) (proffer.HandlerRecommendationResult, error) {
	sourceID, originalID, err := handlerRequestIDs(req)
	if err != nil || attempt < 1 {
		return proffer.HandlerRecommendationResult{}, errors.New("handler recommendation requires valid source/original references and attempt")
	}
	var contentDigest []byte
	err = s.db.QueryRow(ctx, `
		SELECT object.content_sha256
		FROM context.source_version source
		JOIN context.source_version_object link ON link.source_version_id=source.id
		JOIN context.retained_object object ON object.id=link.object_id
		WHERE source.id=$1::uuid AND source.status='retained'
		  AND object.id=$2::uuid AND link.object_role IN ('original','derived_reference')`, sourceID, originalID).Scan(&contentDigest)
	if err != nil {
		return proffer.HandlerRecommendationResult{}, fmt.Errorf("load retained content identity for recommendation: %w", err)
	}
	if len(contentDigest) != sha256.Size {
		return proffer.HandlerRecommendationResult{}, errors.New("retained content identity has an invalid sha256")
	}
	reader, err := s.open.OpenOriginal(ctx, req.Refs["original"])
	if err != nil {
		return proffer.HandlerRecommendationResult{}, err
	}
	defer reader.Close()
	rawHead, err := io.ReadAll(io.LimitReader(reader, handlerSignatureReadLimit+1))
	if err != nil {
		return proffer.HandlerRecommendationResult{}, fmt.Errorf("read retained content signature: %w", err)
	}
	head := signatureHead(rawHead, handlerSignatureReadLimit)
	detected, signatureKind, err := admittedHandlerContent(head)
	if err != nil {
		return proffer.HandlerRecommendationResult{}, err
	}
	scan, scanErr := sourceformat.ScanCredentials(io.MultiReader(bytes.NewReader(rawHead), reader))
	if scan.Signature != "" {
		return proffer.HandlerRecommendationResult{}, fmt.Errorf("retained source excluded by %s: %s", sourceformat.PolicyID, scan.Signature)
	}
	if scanErr != nil || !scan.Complete {
		return proffer.HandlerRecommendationResult{}, errors.New("retained source credential scan incomplete")
	}
	var decoderCapability parser.Capability
	if _, templateErr := activities.StructuredELTFormatForDeclaredFormat(detected); templateErr != nil && !activities.DeriveEligibleFormat(detected) {
		if s.parsers == nil {
			return proffer.HandlerRecommendationResult{}, errors.New("handler recommendation requires the registered parser capability set")
		}
		decoderCapability, err = s.parsers.SelectCapability(parser.FormatID(detected))
		if err != nil {
			return proffer.HandlerRecommendationResult{}, fmt.Errorf("resolve registered handler for detected signature %q: %w", detected, err)
		}
	}
	result, err := s.persistRecommendation(ctx, req, sourceID, originalID, contentDigest, detected, signatureKind, decoderCapability, attempt)
	if err != nil {
		return proffer.HandlerRecommendationResult{}, err
	}
	// A signature registers one processing unit. This is an engine routing
	// decision, not an operator approval, custody action, or UI preference.
	if len(result.Alternatives) != 0 {
		return proffer.HandlerRecommendationResult{}, errors.New("persisted handler recommendation predates single-handler routing; start a fresh run")
	}
	result.EngineDecisionRef, err = s.PersistHandlerSelectionDecision(ctx, req.SourceVersionRef,
		result.RecommendationRef, "engine:signature-registry/v1", result.Recommended.CompatibilityRef,
		"engine-handler:"+string(result.RecommendationRef))
	return result, err
}

// signatureHead delegates bounded retained-prefix preparation to sourceformat.
// Inputs: prefix and limit. Output: bounded byte view. Side effects: none.
// Pick this compatibility wrapper before retained-content detection.
func signatureHead(head []byte, limit int64) []byte { return sourceformat.SignatureHead(head, limit) }
func handlerRequestIDs(req proffer.StageRequest) (uuid.UUID, uuid.UUID, error) {
	sourceID, sourceErr := uuid.Parse(string(req.SourceVersionRef))
	originalID, originalErr := uuid.Parse(string(req.Refs["original"]))
	if sourceErr != nil || originalErr != nil || strings.TrimSpace(req.RequestID) == "" {
		return uuid.Nil, uuid.Nil, errors.New("invalid handler recommendation identity")
	}
	return sourceID, originalID, nil
}

// detectHandlerContent delegates retained-byte detection to the shared sourceformat unit.
// Inputs: retained source prefix. Outputs: format and signature IDs, or an empty-source error.
// Side effects: none. Pick this compatibility wrapper for existing postgres callers and tests.
func detectHandlerContent(head []byte) (format, signatureKind string, err error) {
	return sourceformat.Detect(head)
}

// admittedHandlerContent rejects secret-bearing prefixes before any parser capability is recommended.
// Input: bounded retained bytes. Output: format/signature or a fixed policy/credential reason, never secret values.
// Side effects: none. Pick from RecommendHandler; Detect remains available for format-only callers.
func admittedHandlerContent(head []byte) (format, signature string, err error) {
	detection, err := sourceformat.Inspect(head)
	if detection.Credential != "" {
		return "", "", fmt.Errorf("retained source excluded by %s: %s", sourceformat.PolicyID, detection.Credential)
	}
	return detection.Format, detection.Signature, err
}
func handlerCandidatesForDetectedFormat(detected string, decoder parser.Capability) []proffer.HandlerCandidate {
	candidates := make([]proffer.HandlerCandidate, 0, 2)
	// Signature-based routing: one registry entry per signature, no ladder.
	// A signature that must be derived before anything can read it resolves
	// to the derive route and to nothing else — the DuckDB sms_xml_v1
	// template is NOT offered as an alternative, because it fails on every
	// real backup (no streaming XML reader, 16 MB cap).
	// Byline: Claude Code · Opus 5 · 2026-09-20
	if activities.DeriveEligibleFormat(detected) {
		return append(candidates, proffer.HandlerCandidate{
			HandlerID: activities.DeriveHandlerID, HandlerVersion: activities.DeriveHandlerVersion,
			ExecutionPath: proffer.HandlerPathDerive, CompatibilityRef: proffer.Ref(uuid.NewString()),
			Reason: "retained source must be streamed and republished as structured text beside the original before any extractor can read it",
		})
	}
	if _, formatErr := activities.StructuredELTFormatForDeclaredFormat(detected); formatErr == nil {
		return append(candidates, proffer.HandlerCandidate{
			HandlerID: activities.StructuredELTParserID, HandlerVersion: activities.StructuredELTParserVersion,
			ExecutionPath: proffer.HandlerPathDuckDB, CompatibilityRef: proffer.Ref(uuid.NewString()),
			Reason: "retained source content matches the pinned DuckDB structured extraction signature",
		})
	}
	return append(candidates, proffer.HandlerCandidate{
		HandlerID: decoder.ParserID, HandlerVersion: decoder.ParserVersion,
		ExecutionPath: proffer.HandlerPathDecoder, CompatibilityRef: proffer.Ref(uuid.NewString()),
		Reason: "retained content signature maps to this exact registered decoder; no DuckDB template covers this signature",
	})
}

func (s *HandlerSelectionStore) persistRecommendation(ctx context.Context, req proffer.StageRequest, sourceID, originalID uuid.UUID, digest []byte, detected, signatureKind string, decoder parser.Capability, attempt int32) (proffer.HandlerRecommendationResult, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return proffer.HandlerRecommendationResult{}, err
	}
	rollback := true
	defer func() {
		if rollback {
			cleanup, cancel := boundedCleanup(ctx)
			defer cancel()
			_ = tx.Rollback(cleanup)
		}
	}()
	key := "handler-recommendation:" + originalID.String() + ":" + fmt.Sprintf("%x", digest)
	executionID, err := parserEnsureExecution(ctx, tx, sourceID, req.RequestID, proffer.RecommendHandlerActivityName, key)
	if err != nil {
		return proffer.HandlerRecommendationResult{}, err
	}
	if prior, ok, err := loadRecommendationTx(ctx, tx, executionID); err != nil {
		return proffer.HandlerRecommendationResult{}, err
	} else if ok {
		if err = tx.Commit(ctx); err != nil {
			return proffer.HandlerRecommendationResult{}, err
		}
		rollback = false
		return prior, nil
	}
	signatureID, formatID, recommendationID, receiptID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	now := s.clock()
	candidates := handlerCandidatesForDetectedFormat(detected, decoder)
	candidatesJSON, _ := json.Marshal(candidates)
	result := proffer.HandlerRecommendationResult{
		RecommendationRef: proffer.Ref(recommendationID.String()), ReceiptRef: proffer.Ref(receiptID.String()),
		DetectedFormat: detected, DetectedFormatRef: proffer.Ref(formatID.String()), SignatureRef: proffer.Ref(signatureID.String()),
		Recommended: candidates[0], Alternatives: candidates[1:],
	}
	resultJSON, _ := json.Marshal(map[string]any{
		"ref_kind": "handler_recommendation", "ref_id": recommendationID.String(), "detected_format": detected,
		"detected_format_ref": formatID.String(), "signature_ref": signatureID.String(), "candidates": candidates,
	})
	if _, err = tx.Exec(ctx, `INSERT INTO context.handler_content_signature(id,source_version_id,original_object_id,signature_kind,content_sha256,created_at) VALUES($1,$2,$3,$4,$5,$6)`, signatureID, sourceID, originalID, signatureKind, digest, now); err != nil {
		return proffer.HandlerRecommendationResult{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO context.handler_detected_format(id,source_version_id,content_signature_id,format_id,created_at) VALUES($1,$2,$3,$4,$5)`, formatID, sourceID, signatureID, detected, now); err != nil {
		return proffer.HandlerRecommendationResult{}, err
	}
	for _, candidate := range candidates {
		if _, err = tx.Exec(ctx, `INSERT INTO context.handler_compatibility(id,detected_format_id,handler_id,handler_version,execution_path,reason,created_at) VALUES($1,$2,$3,$4,$5,$6,$7)`, candidate.CompatibilityRef, formatID, candidate.HandlerID, candidate.HandlerVersion, string(candidate.ExecutionPath), candidate.Reason, now); err != nil {
			return proffer.HandlerRecommendationResult{}, err
		}
	}
	if _, err = tx.Exec(ctx, `INSERT INTO context.activity_receipt(id,activity_execution_id,attempt,status,started_at,completed_at,result_ref) VALUES($1,$2,$3,'success',$4,$4,$5)`, receiptID, executionID, attempt, now, resultJSON); err != nil {
		return proffer.HandlerRecommendationResult{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO context.handler_recommendation(id,source_version_id,original_object_id,content_signature_id,detected_format_id,activity_receipt_id,candidates,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, recommendationID, sourceID, originalID, signatureID, formatID, receiptID, candidatesJSON, now); err != nil {
		return proffer.HandlerRecommendationResult{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return proffer.HandlerRecommendationResult{}, err
	}
	rollback = false
	return result, nil
}

func loadRecommendationTx(ctx context.Context, tx pgx.Tx, executionID uuid.UUID) (proffer.HandlerRecommendationResult, bool, error) {
	var recommendationID, receiptID, formatID, signatureID uuid.UUID
	var detected, failureReceipt string
	var candidatesJSON []byte
	err := tx.QueryRow(ctx, `
		SELECT recommendation.id,receipt.id,format.id,signature.id,format.format_id,recommendation.candidates,
		       COALESCE(receipt.result_ref->>'failure_receipt_ref','')
		FROM context.activity_receipt receipt
		JOIN context.handler_recommendation recommendation ON recommendation.activity_receipt_id=receipt.id
		JOIN context.handler_detected_format format ON format.id=recommendation.detected_format_id
		JOIN context.handler_content_signature signature ON signature.id=recommendation.content_signature_id
		WHERE receipt.activity_execution_id=$1 AND receipt.status='success' ORDER BY receipt.attempt LIMIT 1`, executionID).
		Scan(&recommendationID, &receiptID, &formatID, &signatureID, &detected, &candidatesJSON, &failureReceipt)
	if errors.Is(err, pgx.ErrNoRows) {
		return proffer.HandlerRecommendationResult{}, false, nil
	}
	if err != nil {
		return proffer.HandlerRecommendationResult{}, false, err
	}
	var candidates []proffer.HandlerCandidate
	if json.Unmarshal(candidatesJSON, &candidates) != nil || len(candidates) == 0 || len(candidates) > 4 {
		return proffer.HandlerRecommendationResult{}, false, errors.New("stored handler recommendation candidate set is invalid")
	}
	return proffer.HandlerRecommendationResult{
		FailureReceiptRef: proffer.Ref(failureReceipt),
		RecommendationRef: proffer.Ref(recommendationID.String()), ReceiptRef: proffer.Ref(receiptID.String()),
		DetectedFormat: detected, DetectedFormatRef: proffer.Ref(formatID.String()), SignatureRef: proffer.Ref(signatureID.String()),
		Recommended: candidates[0], Alternatives: candidates[1:],
	}, true, nil
}

// PersistHandlerSelectionDecision binds an authenticated actor to one exact
// candidate from one exact recommendation before Temporal is signaled.
func (s *HandlerSelectionStore) PersistHandlerSelectionDecision(ctx context.Context, sourceRef, recommendationRef, actorRef, compatibilityRef proffer.Ref, idempotencyKey string) (proffer.Ref, error) {
	sourceID, sourceErr := uuid.Parse(string(sourceRef))
	recommendationID, recommendationErr := uuid.Parse(string(recommendationRef))
	compatibilityID, compatibilityErr := uuid.Parse(string(compatibilityRef))
	if sourceErr != nil || recommendationErr != nil || compatibilityErr != nil || strings.TrimSpace(string(actorRef)) == "" || strings.TrimSpace(idempotencyKey) == "" {
		return "", errors.New("handler selection decision requires valid durable references, actor, and idempotency key")
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", err
	}
	rollback := true
	defer func() {
		if rollback {
			cleanup, cancel := boundedCleanup(ctx)
			defer cancel()
			_ = tx.Rollback(cleanup)
		}
	}()
	var id uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO context.handler_selection_decision(source_version_id,recommendation_id,compatibility_id,actor_ref,decision_idempotency_key)
		SELECT $1,$2,$3,$4,$5
		FROM context.handler_recommendation recommendation
		JOIN context.handler_compatibility compatibility ON compatibility.detected_format_id=recommendation.detected_format_id
		WHERE recommendation.id=$2 AND recommendation.source_version_id=$1 AND compatibility.id=$3
		  AND recommendation.candidates @> jsonb_build_array(jsonb_build_object(
		    'handler_id',compatibility.handler_id,'handler_version',compatibility.handler_version,
		    'execution_path',compatibility.execution_path,'compatibility_ref',compatibility.id::text))
		ON CONFLICT(decision_idempotency_key) DO NOTHING RETURNING id`, sourceID, recommendationID, compatibilityID, string(actorRef), idempotencyKey).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		var storedSource, storedRecommendation, storedCompatibility uuid.UUID
		var storedActor string
		err = tx.QueryRow(ctx, `SELECT id,source_version_id,recommendation_id,compatibility_id,actor_ref FROM context.handler_selection_decision WHERE decision_idempotency_key=$1`, idempotencyKey).
			Scan(&id, &storedSource, &storedRecommendation, &storedCompatibility, &storedActor)
		if err == nil && (storedSource != sourceID || storedRecommendation != recommendationID || storedCompatibility != compatibilityID || storedActor != string(actorRef)) {
			err = errors.New("handler selection idempotency key is bound to different content")
		}
	}
	if err != nil {
		return "", fmt.Errorf("persist exact handler selection decision: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return "", err
	}
	rollback = false
	return proffer.Ref(id.String()), nil
}

// ValidateHandlerSelection reloads every asserted reference, proves the
// actor-bound decision is a member of the durable bounded set, and persists
// the validation Activity receipt before returning the chosen route.
func (s *HandlerSelectionStore) ValidateHandlerSelection(ctx context.Context, req proffer.StageRequest, attempt int32) (proffer.HandlerSelectionValidationResult, error) {
	sourceID, sourceErr := uuid.Parse(string(req.SourceVersionRef))
	recommendationID, recommendationErr := uuid.Parse(string(req.Refs["handler_recommendation"]))
	decisionID, decisionErr := uuid.Parse(string(req.Refs["handler_decision"]))
	if sourceErr != nil || recommendationErr != nil || decisionErr != nil || strings.TrimSpace(req.RequestID) == "" || attempt < 1 {
		return proffer.HandlerSelectionValidationResult{}, errors.New("handler selection validation requires source, recommendation, decision, request, and attempt")
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return proffer.HandlerSelectionValidationResult{}, err
	}
	rollback := true
	defer func() {
		if rollback {
			cleanup, cancel := boundedCleanup(ctx)
			defer cancel()
			_ = tx.Rollback(cleanup)
		}
	}()
	key := "handler-validation:" + recommendationID.String() + ":" + decisionID.String()
	executionID, err := parserEnsureExecution(ctx, tx, sourceID, req.RequestID, proffer.ValidateHandlerSelectionActivityName, key)
	if err != nil {
		return proffer.HandlerSelectionValidationResult{}, err
	}
	var actor, declared, detected, handlerID, handlerVersion, path, reason string
	var formatID, signatureID, compatibilityID uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT decision.actor_ref,source.declared_format,format.format_id,format.id,signature.id,compatibility.id,
		       compatibility.handler_id,compatibility.handler_version,compatibility.execution_path,compatibility.reason
		FROM context.handler_selection_decision decision
		JOIN context.source_version source ON source.id=decision.source_version_id
		JOIN context.handler_recommendation recommendation ON recommendation.id=decision.recommendation_id
		JOIN context.handler_detected_format format ON format.id=recommendation.detected_format_id
		JOIN context.handler_content_signature signature ON signature.id=recommendation.content_signature_id
		JOIN context.handler_compatibility compatibility ON compatibility.id=decision.compatibility_id
		WHERE decision.id=$1 AND decision.source_version_id=$2 AND recommendation.id=$3
		  AND recommendation.source_version_id=$2 AND format.source_version_id=$2
		  AND signature.source_version_id=$2 AND signature.original_object_id=recommendation.original_object_id
		  AND compatibility.detected_format_id=format.id
		  AND recommendation.candidates @> jsonb_build_array(jsonb_build_object(
		    'handler_id',compatibility.handler_id,'handler_version',compatibility.handler_version,
		    'execution_path',compatibility.execution_path,'compatibility_ref',compatibility.id::text))`, decisionID, sourceID, recommendationID).
		Scan(&actor, &declared, &detected, &formatID, &signatureID, &compatibilityID, &handlerID, &handlerVersion, &path, &reason)
	if err != nil {
		return proffer.HandlerSelectionValidationResult{}, fmt.Errorf("validate exact actor-bound handler selection: %w", err)
	}
	if req.DeclaredFormat != declared || req.Refs["detected_format"] != proffer.Ref(formatID.String()) || req.Refs["content_signature"] != proffer.Ref(signatureID.String()) {
		return proffer.HandlerSelectionValidationResult{}, errors.New("handler selection validation references do not match the durable recommendation")
	}
	chosen := proffer.HandlerCandidate{HandlerID: handlerID, HandlerVersion: handlerVersion, ExecutionPath: proffer.HandlerExecutionPath(path), CompatibilityRef: proffer.Ref(compatibilityID.String()), Reason: reason}
	var priorReceipt uuid.UUID
	if err = tx.QueryRow(ctx, `SELECT id FROM context.activity_receipt WHERE activity_execution_id=$1 AND status='success' ORDER BY attempt LIMIT 1`, executionID).Scan(&priorReceipt); err == nil {
		if err = tx.Commit(ctx); err != nil {
			return proffer.HandlerSelectionValidationResult{}, err
		}
		rollback = false
		return validationResult(decisionID, actor, priorReceipt, recommendationID, detected, formatID, signatureID, chosen), nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return proffer.HandlerSelectionValidationResult{}, err
	}
	receiptID, validationID := uuid.New(), uuid.New()
	now := s.clock()
	resultJSON, _ := json.Marshal(map[string]any{"ref_kind": "handler_selection_validation", "ref_id": validationID.String(), "decision_ref": decisionID.String(), "chosen": chosen})
	if _, err = tx.Exec(ctx, `INSERT INTO context.activity_receipt(id,activity_execution_id,attempt,status,started_at,completed_at,result_ref) VALUES($1,$2,$3,'success',$4,$4,$5)`, receiptID, executionID, attempt, now, resultJSON); err != nil {
		return proffer.HandlerSelectionValidationResult{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO context.handler_selection_validation(id,source_version_id,recommendation_id,decision_id,compatibility_id,actor_ref,activity_receipt_id,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, validationID, sourceID, recommendationID, decisionID, compatibilityID, actor, receiptID, now); err != nil {
		return proffer.HandlerSelectionValidationResult{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return proffer.HandlerSelectionValidationResult{}, err
	}
	rollback = false
	return validationResult(decisionID, actor, receiptID, recommendationID, detected, formatID, signatureID, chosen), nil
}

// LoadHandlerExecutionAuthorization resolves the durable, validated handler
// decision needed by either exact decoder execution or the DuckDB pair. The
// operator's declared format remains immutable; the separate detected format
// is reloaded through the chosen compatibility and validation records.
func (s *HandlerSelectionStore) LoadHandlerExecutionAuthorization(ctx context.Context, req proffer.StageRequest) (activities.HandlerExecutionAuthorization, error) {
	sourceID, sourceErr := uuid.Parse(string(req.SourceVersionRef))
	recommendationID, recommendationErr := uuid.Parse(string(req.Refs["handler_recommendation"]))
	decisionID, decisionErr := uuid.Parse(string(req.Refs["handler_decision"]))
	// The handler_validation ref is the validation's activity receipt (ValidationReceipt),
	// not its row id: the validation writer mints both and returns the receipt. Matching
	// validation.id found no rows for every source (found live 2026-09-20);
	// activity_receipt_id is UNIQUE on handler_selection_validation.
	validationReceiptID, validationErr := uuid.Parse(string(req.Refs["handler_validation"]))
	formatID, formatErr := uuid.Parse(string(req.Refs["detected_format"]))
	signatureID, signatureErr := uuid.Parse(string(req.Refs["content_signature"]))
	compatibilityID, compatibilityErr := uuid.Parse(string(req.Refs["handler_compatibility"]))
	if sourceErr != nil || recommendationErr != nil || decisionErr != nil || validationErr != nil || formatErr != nil || signatureErr != nil || compatibilityErr != nil {
		return activities.HandlerExecutionAuthorization{}, errors.New("handler execution authorization requires valid durable handler references")
	}
	var authorization activities.HandlerExecutionAuthorization
	var declared, path string
	err := s.db.QueryRow(ctx, `
		SELECT source.declared_format,format.format_id,compatibility.handler_id,
		       compatibility.handler_version,compatibility.execution_path
		FROM context.handler_selection_validation validation
		JOIN context.source_version source ON source.id=validation.source_version_id
		JOIN context.handler_recommendation recommendation ON recommendation.id=validation.recommendation_id
		JOIN context.handler_selection_decision decision ON decision.id=validation.decision_id
		JOIN context.handler_detected_format format ON format.id=recommendation.detected_format_id
		JOIN context.handler_content_signature signature ON signature.id=recommendation.content_signature_id
		JOIN context.handler_compatibility compatibility ON compatibility.id=validation.compatibility_id
		WHERE validation.activity_receipt_id=$1 AND validation.source_version_id=$2
		  AND recommendation.id=$3 AND recommendation.source_version_id=$2
		  AND decision.id=$4 AND decision.source_version_id=$2
		  AND decision.recommendation_id=recommendation.id
		  AND decision.compatibility_id=compatibility.id
		  AND format.id=$5 AND format.source_version_id=$2
		  AND signature.id=$6 AND signature.source_version_id=$2
		  AND signature.original_object_id=recommendation.original_object_id
		  AND compatibility.id=$7 AND compatibility.detected_format_id=format.id`,
		validationReceiptID, sourceID, recommendationID, decisionID, formatID, signatureID, compatibilityID).
		Scan(&declared, &authorization.DetectedFormat, &authorization.HandlerID, &authorization.HandlerVersion, &path)
	if err != nil {
		return activities.HandlerExecutionAuthorization{}, fmt.Errorf("load durable handler execution authorization: %w", err)
	}
	if req.DeclaredFormat != declared {
		return activities.HandlerExecutionAuthorization{}, errors.New("handler execution declared format does not match the durable source declaration")
	}
	authorization.ExecutionPath = proffer.HandlerExecutionPath(path)
	return authorization, nil
}

func validationResult(decisionID uuid.UUID, actor string, receiptID, recommendationID uuid.UUID, detected string, formatID, signatureID uuid.UUID, chosen proffer.HandlerCandidate) proffer.HandlerSelectionValidationResult {
	return proffer.HandlerSelectionValidationResult{
		DecisionRef: proffer.Ref(decisionID.String()), ActorRef: proffer.Ref(actor), ValidationReceipt: proffer.Ref(receiptID.String()),
		RecommendationRef: proffer.Ref(recommendationID.String()), DetectedFormat: detected,
		DetectedFormatRef: proffer.Ref(formatID.String()), SignatureRef: proffer.Ref(signatureID.String()), Chosen: chosen,
	}
}

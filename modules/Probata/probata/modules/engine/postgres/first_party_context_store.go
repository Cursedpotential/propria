// Byline: Claude Code · Opus 5.5 · 2026-10-01
// Byline: Claude Code · Opus 5.5 · 2026-10-02 (participant resolution; third-party projection)
//
// The PostgreSQL boundary of the first-party context import (D04,
// activities/first_party_context.go). It owns every transaction and
// idempotency coordinate of the four Activities:
//
//   - reads: the verified generation's message records
//     (context.normalized_record_identity), the derivation that produced the
//     source (a derive_sms_threads_activity receipt), and registry.person;
//   - receipts: one context.activity_execution / activity_receipt per
//     Activity, through the same retry-recovery helpers the other stores use;
//   - the spine commit: working.normalized_record (cited by
//     source_version_id, no custody claim: artifact_id stays NULL until
//     promotion), message_projection_route, message and message_participant,
//     one transaction per conversation, with the derived-write guard armed and
//     this writer declared the deriver;
//   - the thread commit: working.first_party_context_thread and its version,
//     membership and source assertion rows, one transaction per conversation.
//     A conversation first seen here gets version 1 through
//     FirstPartyThreadStore; a later chunk of the same conversation extends
//     the current version in place, whatever its review state (OD-07 granted
//     UPDATE for this; owner 2026-10-02: nothing is immutable until it is
//     promoted to evidence).
//
// Ids are copied, never minted for records: working.normalized_record.id and
// working.message.id are the context.normalized_record_identity id (DF-04).
// The only id minted here is a new conversation's thread id, a UUIDv7.
package postgres

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/contextthread"
	"github.com/Cursedpotential/probata/engine/disclosure"
	"github.com/Cursedpotential/probata/engine/firstparty"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/stagegraph"
)

// firstPartyRecordSource is working.normalized_record.source for every row
// this import writes; the (source, conversation_id) index serves the
// conversation lookup.
const firstPartyRecordSource = "proffer"

// FirstPartyContextStore implements activities.FirstPartyContextStore.
type FirstPartyContextStore struct {
	db      DB
	threads *FirstPartyThreadStore
	now     func() time.Time
}

// NewFirstPartyContextStore requires a database.
func NewFirstPartyContextStore(db DB) (*FirstPartyContextStore, error) {
	threads, err := NewFirstPartyThreadStore(db)
	if err != nil {
		return nil, err
	}
	return &FirstPartyContextStore{db: db, threads: threads, now: func() time.Time { return time.Now().UTC() }}, nil
}

// LoadFirstPartyContext implements activities.FirstPartyContextStore.
func (s *FirstPartyContextStore) LoadFirstPartyContext(
	ctx context.Context, req proffer.StageRequest, generationRef, verificationRef proffer.Ref,
) (activities.FirstPartyContextInput, error) {
	sourceVersionID, err := uuid.Parse(string(req.SourceVersionRef))
	if err != nil {
		return activities.FirstPartyContextInput{}, fmt.Errorf("source version reference %q: %w", req.SourceVersionRef, err)
	}
	generationID, err := uuid.Parse(string(generationRef))
	if err != nil {
		return activities.FirstPartyContextInput{}, fmt.Errorf("normalized generation reference %q: %w", generationRef, err)
	}
	verificationID, err := uuid.Parse(string(verificationRef))
	if err != nil {
		return activities.FirstPartyContextInput{}, fmt.Errorf("normalized verification reference %q: %w", verificationRef, err)
	}

	var verifiedGenerationID uuid.UUID
	var verifyStatus string
	if err := s.db.QueryRow(ctx, `
		SELECT normalized_generation_id, status FROM context.reconciliation_receipt
		WHERE id = $1::uuid AND reconciliation_kind = 'normalized_generation_verification'`, verificationID).
		Scan(&verifiedGenerationID, &verifyStatus); err != nil {
		return activities.FirstPartyContextInput{}, fmt.Errorf("read normalized generation verification receipt: %w", err)
	}
	if verifyStatus != "success" || verifiedGenerationID != generationID {
		return activities.FirstPartyContextInput{}, errors.New("first-party context import requires a successful verification of this exact normalized generation")
	}

	var generationSource uuid.UUID
	var workflowID, status, declaredFormat, sourceKey string
	var matterID, courtCaseID uuid.NullUUID
	if err := s.db.QueryRow(ctx, `
		SELECT generation.source_version_id, version.workflow_id, version.status, version.declared_format,
		       source.source_key, version.matter_id, version.court_case_id
		FROM context.normalized_generation generation
		JOIN context.source_version version ON version.id = generation.source_version_id
		JOIN context.source source ON source.id = version.source_id
		WHERE generation.id = $1::uuid`, generationID).
		Scan(&generationSource, &workflowID, &status, &declaredFormat, &sourceKey, &matterID, &courtCaseID); err != nil {
		return activities.FirstPartyContextInput{}, fmt.Errorf("resolve first-party context source: %w", err)
	}
	if generationSource != sourceVersionID || workflowID != req.RequestID || status != "retained" {
		return activities.FirstPartyContextInput{}, errors.New("normalized generation is not a retained source version owned by this request")
	}
	if !matterID.Valid || !courtCaseID.Valid ||
		!strings.EqualFold(matterID.UUID.String(), strings.TrimSpace(req.MatterID)) ||
		!strings.EqualFold(courtCaseID.UUID.String(), strings.TrimSpace(req.CourtCaseID)) {
		return activities.FirstPartyContextInput{}, errors.New("source version's matter and court case differ from the run's")
	}

	messages, stated, err := s.readMessages(ctx, generationID)
	if err != nil {
		return activities.FirstPartyContextInput{}, err
	}
	input := activities.FirstPartyContextInput{
		Source: firstparty.Source{
			SourceVersionID: sourceVersionID.String(), NormalizedGenerationID: generationID.String(),
			DeclaredFormat: declaredFormat, SourceKey: sourceKey,
		},
		Messages: messages, StatedIdentifiers: stated,
	}
	if len(messages) == 0 {
		return input, nil
	}

	// The platform comes from the registered derivation that published this
	// source, never from its name or content.
	var derivedFrom string
	err = s.db.QueryRow(ctx, `
		SELECT execution.source_version_id::text
		FROM context.activity_receipt receipt
		JOIN context.activity_execution execution ON execution.id = receipt.activity_execution_id
		WHERE execution.activity_name = $2 AND receipt.status = 'success'
		  AND coalesce(receipt.result_ref->>'derived_prefix', '') <> ''
		  AND starts_with($1, receipt.result_ref->>'derived_prefix')
		ORDER BY length(receipt.result_ref->>'derived_prefix') DESC, receipt.created_at DESC
		LIMIT 1`, sourceKey, string(stagegraph.DeriveSMSThreads)).Scan(&derivedFrom)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		input.Reason = fmt.Sprintf("source %s (declared %s) traces to no registered derivation, so its messaging platform is unknown", sourceKey, declaredFormat)
		return input, nil
	case err != nil:
		return activities.FirstPartyContextInput{}, fmt.Errorf("resolve the derivation that published the source: %w", err)
	}
	platform, capture, representation, ok := firstparty.PlatformForDerivation(activities.DeriveHandlerID, declaredFormat)
	if !ok {
		input.Reason = fmt.Sprintf("derivation %s has no first-party platform registered for declared format %s", activities.DeriveHandlerID, declaredFormat)
		return input, nil
	}
	input.Source.Platform, input.Source.CaptureKind, input.Source.RepresentationKind = platform, capture, representation
	input.Source.DerivedFromSourceVersionID = derivedFrom
	input.PlatformResolved = true
	return input, nil
}

type normalizedMessagePayload struct {
	Content struct {
		Body string `json:"body"`
	} `json:"content"`
	Participants []struct {
		Role       string `json:"role"`
		Identifier string `json:"identifier"`
	} `json:"participants"`
}

// readMessages reads the generation's records in ordinal order. Message
// records become SourceMessages; every record of any kind contributes its
// stated identifiers, because the one participant resolution serves the
// Weaviate-first stage too, which publishes calls as well as messages.
//
// A participant of role "unknown" counts as a recipient, matching the
// Weaviate-first stage (2026-10-02), unless it repeats the sender.
func (s *FirstPartyContextStore) readMessages(ctx context.Context, generationID uuid.UUID) ([]firstparty.SourceMessage, []string, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id::text, record_type, record_ordinal, occurred_at, normalized_payload
		FROM context.normalized_record_identity
		WHERE normalized_generation_id = $1::uuid
		ORDER BY record_ordinal`, generationID)
	if err != nil {
		return nil, nil, fmt.Errorf("read normalized records: %w", err)
	}
	defer rows.Close()
	var messages []firstparty.SourceMessage
	stated := map[string]bool{}
	for rows.Next() {
		if len(messages) >= firstparty.MaxMessages {
			return nil, nil, fmt.Errorf("generation holds more than %d message records", firstparty.MaxMessages)
		}
		var id, recordType string
		var ordinal int64
		var occurred pgtype.Timestamptz
		var raw []byte
		if err := rows.Scan(&id, &recordType, &ordinal, &occurred, &raw); err != nil {
			return nil, nil, err
		}
		var payload normalizedMessagePayload
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, nil, fmt.Errorf("decode normalized record %s: %w", id, err)
		}
		message := firstparty.SourceMessage{RecordID: id, Ordinal: ordinal, Body: payload.Content.Body}
		if occurred.Valid {
			at := occurred.Time.UTC()
			message.OccurredAt = &at
		}
		var unknown []string
		for _, party := range payload.Participants {
			identifier := strings.TrimSpace(party.Identifier)
			if identifier == "" {
				continue
			}
			if !strings.EqualFold(identifier, firstparty.SelfMarker) {
				stated[identifier] = true
			}
			message.Parties = append(message.Parties, identifier)
			switch party.Role {
			case "sender":
				if message.Sender == "" {
					message.Sender = identifier
				}
			case "recipient":
				message.Recipients = append(message.Recipients, identifier)
			default:
				unknown = append(unknown, identifier)
			}
		}
		for _, identifier := range unknown {
			if !strings.EqualFold(identifier, message.Sender) {
				message.Recipients = append(message.Recipients, identifier)
			}
		}
		if recordType == "message" {
			messages = append(messages, message)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	identifiers := make([]string, 0, len(stated))
	for identifier := range stated {
		identifiers = append(identifiers, identifier)
	}
	sort.Strings(identifiers)
	return messages, identifiers, nil
}

// ResolveFirstPartyIdentity implements activities.FirstPartyContextStore.
func (s *FirstPartyContextStore) ResolveFirstPartyIdentity(
	ctx context.Context, req proffer.StageRequest, ownerPersonID, perspectivePersonID string,
) (contextthread.Identity, error) {
	owner, err := uuid.Parse(strings.TrimSpace(ownerPersonID))
	if err != nil {
		return contextthread.Identity{}, fmt.Errorf("owner_person_id %q is not a uuid", ownerPersonID)
	}
	perspective, err := uuid.Parse(strings.TrimSpace(perspectivePersonID))
	if err != nil {
		return contextthread.Identity{}, fmt.Errorf("perspective_person_id %q is not a uuid", perspectivePersonID)
	}
	matter, err := uuid.Parse(strings.TrimSpace(req.MatterID))
	if err != nil {
		return contextthread.Identity{}, fmt.Errorf("run matter %q is not a uuid", req.MatterID)
	}
	courtCase, err := uuid.Parse(strings.TrimSpace(req.CourtCaseID))
	if err != nil {
		return contextthread.Identity{}, fmt.Errorf("run court case %q is not a uuid", req.CourtCaseID)
	}
	if _, admitted := MatterModeForIdentity(matter.String(), courtCase.String()); !admitted {
		return contextthread.Identity{}, fmt.Errorf("matter %s with court case %s is not an admitted platform identity", matter, courtCase)
	}
	var ownerRole pgtype.Text
	var owners int
	var ownerExists, perspectiveExists bool
	if err := s.db.QueryRow(ctx, `
		SELECT (SELECT role_in_case FROM registry.person WHERE id = $1::uuid),
		       (SELECT count(*) FROM registry.person WHERE role_in_case = 'user'),
		       EXISTS (SELECT 1 FROM registry.person WHERE id = $1::uuid),
		       EXISTS (SELECT 1 FROM registry.person WHERE id = $2::uuid)`,
		owner, perspective).Scan(&ownerRole, &owners, &ownerExists, &perspectiveExists); err != nil {
		return contextthread.Identity{}, fmt.Errorf("read registry persons: %w", err)
	}
	if !ownerExists {
		return contextthread.Identity{}, fmt.Errorf("owner person %s is not in registry.person", owner)
	}
	if !perspectiveExists {
		return contextthread.Identity{}, fmt.Errorf("perspective person %s is not in registry.person", perspective)
	}
	// The registry's own definition of the owner (the one the projection
	// validator also uses): exactly one person whose role in the case is 'user'.
	if owners != 1 || !ownerRole.Valid || ownerRole.String != "user" {
		return contextthread.Identity{}, fmt.Errorf("owner person %s is not the registry's one case owner (role_in_case 'user')", owner)
	}
	return contextthread.Identity{
		OwnerPersonID: owner.String(), MatterID: matter.String(), CourtCaseID: courtCase.String(),
		PerspectivePersonID: perspective.String(),
	}, nil
}

// ResolveParticipants implements activities.FirstPartyContextStore through
// the one registry resolution (owner_identity.go): the owner is loaded and
// checked (role, not merged, at least one confirmed identifier), then every
// stated identifier is resolved against confirmed identifiers only.
func (s *FirstPartyContextStore) ResolveParticipants(ctx context.Context, identity contextthread.Identity, raws []string) (disclosure.Resolution, error) {
	owner, err := LoadOwner(ctx, s.db, identity.OwnerPersonID, identity.PerspectivePersonID)
	if err != nil {
		return disclosure.Resolution{}, err
	}
	return ResolveParticipants(ctx, s.db, owner, raws)
}

// LoadParticipantResolution implements activities.FirstPartyContextStore with
// the shared reader the Weaviate-first stage uses.
func (s *FirstPartyContextStore) LoadParticipantResolution(ctx context.Context, ref proffer.Ref) (disclosure.Resolution, error) {
	return LoadParticipantResolution(ctx, s.db, string(ref))
}

// PersistParticipantResolution records the resolve stage's receipt in the
// shape LoadParticipantResolution reads: result_ref =
// {"ref_kind":"participant_resolution","ref_id":<uuid>,"resolution":<Resolution>},
// where ref_id is the receipt id and the stage's result Ref.
func (s *FirstPartyContextStore) PersistParticipantResolution(ctx context.Context, spec activities.ParticipantResolutionSpec) (proffer.Ref, proffer.Ref, error) {
	sourceVersionID, err := uuid.Parse(string(spec.SourceVersionRef))
	if err != nil {
		return "", "", fmt.Errorf("source version reference %q: %w", spec.SourceVersionRef, err)
	}
	if strings.TrimSpace(spec.RequestID) == "" || spec.Attempt < 1 || spec.NormalizedGenerationRef == "" {
		return "", "", errors.New("participant resolution receipt requires request, generation and attempt")
	}
	notApplicable := strings.TrimSpace(spec.NotApplicable) != ""
	key := strings.Join([]string{activities.ParticipantResolutionKind, string(spec.NormalizedGenerationRef), "not_applicable"}, ":")
	if !notApplicable {
		if err := spec.Resolution.Validate(); err != nil {
			return "", "", err
		}
		key = strings.Join([]string{activities.ParticipantResolutionKind, string(spec.NormalizedGenerationRef), firstparty.ResolutionDigest(spec.Resolution)}, ":")
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", "", err
	}
	defer func() { cleanup, cancel := boundedCleanup(ctx); defer cancel(); _ = tx.Rollback(cleanup) }()
	executionID, err := parserEnsureExecution(ctx, tx, sourceVersionID, spec.RequestID, ResolveParticipantsActivityName, key)
	if err != nil {
		return "", "", err
	}
	var priorID uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT id FROM context.activity_receipt
		WHERE activity_execution_id = $1::uuid AND status IN ('success', 'not_applicable')
		ORDER BY attempt LIMIT 1`, executionID).Scan(&priorID)
	if err == nil {
		if err := tx.Commit(ctx); err != nil {
			return "", "", err
		}
		return proffer.Ref(priorID.String()), proffer.Ref(priorID.String()), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", "", fmt.Errorf("inspect prior participant resolution receipt: %w", err)
	}
	receiptID, err := uuid.NewV7()
	if err != nil {
		return "", "", err
	}
	now := s.now()
	if notApplicable {
		if _, err := tx.Exec(ctx, `
			INSERT INTO context.activity_receipt (id, activity_execution_id, attempt, status, started_at, completed_at, not_applicable_reason)
			VALUES ($1::uuid, $2::uuid, $3, 'not_applicable', $4, $4, $5)`,
			receiptID, executionID, spec.Attempt, now, spec.NotApplicable); err != nil {
			return "", "", fmt.Errorf("write participant resolution not-applicable receipt: %w", err)
		}
	} else {
		result, err := json.Marshal(map[string]any{
			"ref_kind": activities.ParticipantResolutionKind, "ref_id": receiptID.String(),
			"normalized_generation": string(spec.NormalizedGenerationRef), "resolution": spec.Resolution,
		})
		if err != nil {
			return "", "", err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO context.activity_receipt (id, activity_execution_id, attempt, status, started_at, completed_at, result_ref)
			VALUES ($1::uuid, $2::uuid, $3, 'success', $4, $4, $5::jsonb)`,
			receiptID, executionID, spec.Attempt, now, result); err != nil {
			return "", "", fmt.Errorf("write participant resolution receipt: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return "", "", err
	}
	return proffer.Ref(receiptID.String()), proffer.Ref(receiptID.String()), nil
}

// firstPartyReceipt is the result_ref JSON of a proposal or confirmation.
type firstPartyReceipt struct {
	RefKind               string `json:"ref_kind"`
	RefID                 string `json:"ref_id"`
	NormalizedGeneration  string `json:"normalized_generation"`
	Parent                string `json:"parent,omitempty"`
	ParticipantResolution string `json:"participant_resolution,omitempty"`
	PlanDigest            string `json:"plan_digest"`
	OwnerPerson           string `json:"owner_person"`
	PerspectivePerson     string `json:"perspective_person"`
	Matter                string `json:"matter"`
	CourtCase             string `json:"court_case"`
	Messages              int    `json:"messages"`
	Threads               int    `json:"threads"`
}

// PersistFirstPartyReceipt implements activities.FirstPartyContextStore.
func (s *FirstPartyContextStore) PersistFirstPartyReceipt(ctx context.Context, spec activities.FirstPartyReceiptSpec) (proffer.Ref, proffer.Ref, error) {
	sourceVersionID, err := uuid.Parse(string(spec.SourceVersionRef))
	if err != nil {
		return "", "", fmt.Errorf("source version reference %q: %w", spec.SourceVersionRef, err)
	}
	if strings.TrimSpace(spec.RequestID) == "" || spec.Attempt < 1 || spec.NormalizedGenerationRef == "" {
		return "", "", errors.New("first-party receipt requires request, generation and attempt")
	}
	notApplicable := strings.TrimSpace(spec.NotApplicable) != ""
	if !notApplicable && (len(spec.PlanDigest) != 64 || spec.Messages <= 0) {
		return "", "", errors.New("first-party receipt requires a plan digest and a message count")
	}
	key := strings.Join([]string{spec.Kind, string(spec.NormalizedGenerationRef), string(spec.ParentRef), string(spec.ResolutionRef), spec.PlanDigest}, ":")
	if notApplicable {
		key = strings.Join([]string{spec.Kind, string(spec.NormalizedGenerationRef), "not_applicable"}, ":")
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", "", err
	}
	defer func() { cleanup, cancel := boundedCleanup(ctx); defer cancel(); _ = tx.Rollback(cleanup) }()
	executionID, err := parserEnsureExecution(ctx, tx, sourceVersionID, spec.RequestID, string(spec.Stage), key)
	if err != nil {
		return "", "", err
	}
	var priorID uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT id FROM context.activity_receipt
		WHERE activity_execution_id = $1::uuid AND status IN ('success', 'not_applicable')
		ORDER BY attempt LIMIT 1`, executionID).Scan(&priorID)
	if err == nil {
		if err := tx.Commit(ctx); err != nil {
			return "", "", err
		}
		return proffer.Ref(priorID.String()), proffer.Ref(priorID.String()), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", "", fmt.Errorf("inspect prior first-party receipt: %w", err)
	}
	receiptID, err := uuid.NewV7()
	if err != nil {
		return "", "", err
	}
	now := s.now()
	if notApplicable {
		if _, err := tx.Exec(ctx, `
			INSERT INTO context.activity_receipt (id, activity_execution_id, attempt, status, started_at, completed_at, not_applicable_reason)
			VALUES ($1::uuid, $2::uuid, $3, 'not_applicable', $4, $4, $5)`,
			receiptID, executionID, spec.Attempt, now, spec.NotApplicable); err != nil {
			return "", "", fmt.Errorf("write first-party not-applicable receipt: %w", err)
		}
	} else {
		result, err := json.Marshal(firstPartyReceipt{
			RefKind: spec.Kind, RefID: receiptID.String(), NormalizedGeneration: string(spec.NormalizedGenerationRef),
			Parent: string(spec.ParentRef), ParticipantResolution: string(spec.ResolutionRef), PlanDigest: spec.PlanDigest,
			OwnerPerson: spec.Identity.OwnerPersonID, PerspectivePerson: spec.Identity.PerspectivePersonID,
			Matter: spec.Identity.MatterID, CourtCase: spec.Identity.CourtCaseID,
			Messages: spec.Messages, Threads: spec.Threads,
		})
		if err != nil {
			return "", "", err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO context.activity_receipt (id, activity_execution_id, attempt, status, started_at, completed_at, result_ref)
			VALUES ($1::uuid, $2::uuid, $3, 'success', $4, $4, $5::jsonb)`,
			receiptID, executionID, spec.Attempt, now, result); err != nil {
			return "", "", fmt.Errorf("write first-party receipt: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return "", "", err
	}
	return proffer.Ref(receiptID.String()), proffer.Ref(receiptID.String()), nil
}

// LoadFirstPartyReceipt implements activities.FirstPartyContextStore.
func (s *FirstPartyContextStore) LoadFirstPartyReceipt(ctx context.Context, kind string, ref proffer.Ref) (activities.FirstPartyReceipt, error) {
	receiptID, err := uuid.Parse(string(ref))
	if err != nil {
		return activities.FirstPartyReceipt{}, fmt.Errorf("first-party receipt reference %q: %w", ref, err)
	}
	var sourceVersionID uuid.UUID
	var raw []byte
	if err := s.db.QueryRow(ctx, `
		SELECT execution.source_version_id, receipt.result_ref
		FROM context.activity_receipt receipt
		JOIN context.activity_execution execution ON execution.id = receipt.activity_execution_id
		WHERE receipt.id = $1::uuid AND receipt.status = 'success'`, receiptID).Scan(&sourceVersionID, &raw); err != nil {
		return activities.FirstPartyReceipt{}, fmt.Errorf("read first-party receipt %s: %w", receiptID, err)
	}
	var decoded firstPartyReceipt
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return activities.FirstPartyReceipt{}, fmt.Errorf("decode first-party receipt %s: %w", receiptID, err)
	}
	if decoded.RefKind != kind {
		return activities.FirstPartyReceipt{}, fmt.Errorf("receipt %s is a %q, want %q", receiptID, decoded.RefKind, kind)
	}
	return activities.FirstPartyReceipt{
		Kind: decoded.RefKind, ReceiptRef: ref, SourceVersionRef: proffer.Ref(sourceVersionID.String()),
		NormalizedGenerationRef: proffer.Ref(decoded.NormalizedGeneration), ParentRef: proffer.Ref(decoded.Parent),
		ResolutionRef: proffer.Ref(decoded.ParticipantResolution),
		PlanDigest:    decoded.PlanDigest,
		Identity: contextthread.Identity{
			OwnerPersonID: decoded.OwnerPerson, MatterID: decoded.Matter, CourtCaseID: decoded.CourtCase,
			PerspectivePersonID: decoded.PerspectivePerson,
		},
	}, nil
}

// commitReceipt is the result_ref JSON of a commit.
type commitReceipt struct {
	RefKind    string           `json:"ref_kind"`
	RefID      string           `json:"ref_id"`
	Gate       string           `json:"gate"`
	PlanDigest string           `json:"plan_digest"`
	Messages   int              `json:"messages"`
	Threads    []map[string]any `json:"threads"`
}

// beginCommit checks the gate receipt, then returns the prior receipt when
// this exact commit already succeeded.
func (s *FirstPartyContextStore) beginCommit(
	ctx context.Context, spec activities.FirstPartyCommitSpec, stage stagegraph.StageID, gateKind string,
) (executionID uuid.UUID, prior proffer.Ref, found bool, err error) {
	sourceVersionID, err := uuid.Parse(string(spec.SourceVersionRef))
	if err != nil {
		return uuid.Nil, "", false, fmt.Errorf("source version reference %q: %w", spec.SourceVersionRef, err)
	}
	gateID, err := uuid.Parse(string(spec.GateRef))
	if err != nil {
		return uuid.Nil, "", false, fmt.Errorf("commit gate reference %q: %w", spec.GateRef, err)
	}
	var gateSource uuid.UUID
	var raw []byte
	if err := s.db.QueryRow(ctx, `
		SELECT execution.source_version_id, receipt.result_ref
		FROM context.activity_receipt receipt
		JOIN context.activity_execution execution ON execution.id = receipt.activity_execution_id
		WHERE receipt.id = $1::uuid AND receipt.status = 'success'`, gateID).Scan(&gateSource, &raw); err != nil {
		return uuid.Nil, "", false, fmt.Errorf("read commit gate receipt %s: %w", gateID, err)
	}
	var gate struct {
		RefKind    string `json:"ref_kind"`
		PlanDigest string `json:"plan_digest"`
	}
	if err := json.Unmarshal(raw, &gate); err != nil {
		return uuid.Nil, "", false, fmt.Errorf("decode commit gate receipt: %w", err)
	}
	if gateSource != sourceVersionID || gate.RefKind != gateKind || gate.PlanDigest != spec.Plan.Digest {
		return uuid.Nil, "", false, fmt.Errorf("commit gate %s is not a %s of this plan for this source version", gateID, gateKind)
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return uuid.Nil, "", false, err
	}
	defer func() { cleanup, cancel := boundedCleanup(ctx); defer cancel(); _ = tx.Rollback(cleanup) }()
	executionID, err = parserEnsureExecution(ctx, tx, sourceVersionID, spec.RequestID, string(stage), string(stage)+":"+gateID.String())
	if err != nil {
		return uuid.Nil, "", false, err
	}
	receiptID, _, has, err := normalizeLatestReceipt(ctx, tx, executionID)
	if err != nil {
		return uuid.Nil, "", false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, "", false, err
	}
	if has {
		return executionID, proffer.Ref(receiptID.String()), true, nil
	}
	return executionID, "", false, nil
}

func (s *FirstPartyContextStore) finishCommit(ctx context.Context, executionID uuid.UUID, attempt int32, result commitReceipt) (proffer.Ref, error) {
	receiptID, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	result.RefID = receiptID.String()
	encoded, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", err
	}
	defer func() { cleanup, cancel := boundedCleanup(ctx); defer cancel(); _ = tx.Rollback(cleanup) }()
	now := s.now()
	if _, err := tx.Exec(ctx, `
		INSERT INTO context.activity_receipt (id, activity_execution_id, attempt, status, started_at, completed_at, result_ref)
		VALUES ($1::uuid, $2::uuid, $3, 'success', $4, $4, $5::jsonb)`,
		receiptID, executionID, attempt, now, encoded); err != nil {
		return "", fmt.Errorf("write %s receipt: %w", result.RefKind, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return proffer.Ref(receiptID.String()), nil
}

// declareDeriver arms the derived-write guard and declares this transaction
// the deriver (working.derived_write_guard), for this transaction only.
func declareDeriver(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `SELECT set_config('app.enforce_derived_guard', 'on', true), set_config('app.deriving', 'on', true)`)
	return err
}

func lockConversation(ctx context.Context, tx pgx.Tx, scopedKey string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, scopedKey)
	return err
}

// CommitFirstPartyMessages implements activities.FirstPartyContextStore. It is
// the one writer of the generation's message rows, both corpora: first-party
// conversations get the working.message spine, third-party conversations get
// the working.third_party_* projection. Each conversation is one transaction.
func (s *FirstPartyContextStore) CommitFirstPartyMessages(ctx context.Context, spec activities.FirstPartyCommitSpec) (proffer.Ref, proffer.Ref, error) {
	executionID, prior, found, err := s.beginCommit(ctx, spec, stagegraph.CommitFirstPartyMessages, activities.FirstPartyConfirmationKind)
	if err != nil {
		return "", "", err
	}
	if found {
		return prior, prior, nil
	}
	threads := make([]map[string]any, 0, len(spec.Plan.Conversations))
	for _, conversation := range spec.Plan.Conversations {
		var conversationID string
		var written int64
		if conversation.Corpus == firstparty.CorpusFirstParty {
			conversationID, written, err = s.commitConversationSpine(ctx, spec.Plan, conversation)
		} else {
			conversationID, written, err = s.commitThirdPartyConversation(ctx, spec.Plan, conversation)
		}
		if err != nil {
			return "", "", fmt.Errorf("commit %s conversation %s: %w", conversation.Corpus, conversation.Key, err)
		}
		threads = append(threads, map[string]any{
			"corpus": conversation.Corpus, "conversation": conversationID, "key": conversation.Key,
			"messages": len(conversation.Messages), "inserted": written,
		})
	}
	receipt, err := s.finishCommit(ctx, executionID, spec.Attempt, commitReceipt{
		RefKind: activities.FirstPartyMessagesKind, Gate: string(spec.GateRef), PlanDigest: spec.Plan.Digest,
		Messages: spec.Plan.MessageCount, Threads: threads,
	})
	if err != nil {
		return "", "", err
	}
	return receipt, receipt, nil
}

const insertSpineRecordsSQL = `
	INSERT INTO working.normalized_record
	    (id, artifact_id, source_version_id, record_type, source, conversation_id, participants, content,
	     occurred_at, disclosure_tier, attrs, derived_from_raw_table, derived_from_raw_id, deriver_version,
	     derived_at, source_record_key, sender, recipients, message_corpus)
	SELECT u.id, NULL, $2::uuid, 'message', $3, $4, u.participants::jsonb, u.content,
	       u.occurred_at, u.tier, u.attrs::jsonb, 'context.normalized_record_identity', u.id, $5,
	       now(), u.record_key, NULLIF(u.sender, ''), u.recipients::jsonb, $6
	FROM unnest($1::uuid[], $7::text[], $8::text[], $9::timestamptz[], $10::text[], $11::text[], $12::text[], $13::text[], $14::text[])
	     AS u(id, participants, content, occurred_at, attrs, record_key, sender, recipients, tier)
	ON CONFLICT (id) DO NOTHING`

const insertSpineRoutesSQL = `
	INSERT INTO working.message_projection_route
	    (normalized_record_id, projection_kind, decision_state, basis, proposed_by, approved_by, approved_at, deriver_version)
	SELECT u.id, $5, CASE WHEN u.approved THEN 'approved' ELSE 'proposed' END, u.basis::jsonb, $2,
	       CASE WHEN u.approved THEN $2 END, CASE WHEN u.approved THEN now() END, $2
	FROM unnest($1::uuid[], $3::boolean[], $4::text[]) AS u(id, approved, basis)
	ON CONFLICT (normalized_record_id) DO NOTHING`

const insertSpineMessagesSQL = `
	INSERT INTO working.message
	    (id, conversation_id, ts_utc, platform, sender_raw, recipient_raw, direction, message_type,
	     content_sha256, char_count, derived_from_record_id, deriver_version, derived_at, projection_kind)
	SELECT u.id, $2::uuid, u.occurred_at, $3, NULLIF(u.sender, ''), NULLIF(u.recipient_raw, ''), u.direction, 'text',
	       u.sha, u.char_count, u.id, $4, now(), 'first_party'
	FROM unnest($1::uuid[], $5::timestamptz[], $6::text[], $7::text[], $8::text[], $9::bytea[], $10::integer[])
	     AS u(id, occurred_at, sender, recipient_raw, direction, sha, char_count)
	ON CONFLICT (id) DO NOTHING`

const insertSpineParticipantsSQL = `
	INSERT INTO working.message_participant (message_id, participant_raw, role, entity_id, deriver_version)
	SELECT u.message_id, u.participant_raw, u.role, NULLIF(u.entity_id, '')::uuid, $5
	FROM unnest($1::uuid[], $2::text[], $3::text[], $4::text[]) AS u(message_id, participant_raw, role, entity_id)
	ON CONFLICT ON CONSTRAINT uq_msg_part DO NOTHING`

// spineRows is one conversation's rows in column-array form.
type spineRows struct {
	ids, participants, contents, attrs, recordKeys, senders, recipients, tiers []string
	occurred                                                                   []pgtype.Timestamptz
	approved                                                                   []bool
	bases, recipientRaw, directions                                            []string
	shas                                                                       [][]byte
	charCounts                                                                 []int32
	partMessages, partRaw, partRoles, partEntities                             []string
}

func buildSpineRows(plan firstparty.Plan, conversation firstparty.Conversation) (spineRows, error) {
	var rows spineRows
	for _, message := range conversation.Messages {
		rows.ids = append(rows.ids, message.RecordID)
		parties, _ := json.Marshal(append([]string{message.Sender}, message.Recipients...))
		rows.participants = append(rows.participants, string(parties))
		rows.contents = append(rows.contents, message.Body)
		occurred := pgtype.Timestamptz{}
		if message.OccurredAt != nil {
			occurred = pgtype.Timestamptz{Time: *message.OccurredAt, Valid: true}
		}
		rows.occurred = append(rows.occurred, occurred)
		attr, _ := json.Marshal(map[string]any{
			"platform": plan.Source.Platform, "conversation_key": conversation.Key, "corpus": conversation.Corpus,
			"normalized_generation_id": plan.Source.NormalizedGenerationID, "record_ordinal": message.Ordinal,
			"perspective_person_id": plan.Identity.PerspectivePersonID, "owner_took_part": message.OwnerTookPart,
			"disclosure_tier_basis": message.DisclosureBasis, "direction": message.Direction, "plan_digest": plan.Digest,
		})
		rows.attrs = append(rows.attrs, string(attr))
		rows.recordKeys = append(rows.recordKeys, fmt.Sprintf("%s#%d", plan.Source.SourceVersionID, message.Ordinal))
		rows.senders = append(rows.senders, message.Sender)
		recipientList := make([]map[string]string, 0, len(message.Recipients))
		for _, recipient := range message.Recipients {
			recipientList = append(recipientList, map[string]string{"identity": recipient, "role": "to"})
		}
		encodedRecipients, _ := json.Marshal(recipientList)
		rows.recipients = append(rows.recipients, string(encodedRecipients))
		rows.tiers = append(rows.tiers, message.DisclosureTier)
		rows.approved = append(rows.approved, message.RouteApproved)
		routeBasis, _ := json.Marshal(map[string]any{
			"source_parties_present":       message.Sender != "" && len(message.Recipients) > 0,
			"source_party_review_required": !message.RouteApproved, "owner_took_part": message.OwnerTookPart,
			"plan_digest": plan.Digest,
		})
		rows.bases = append(rows.bases, string(routeBasis))
		rows.recipientRaw = append(rows.recipientRaw, strings.Join(message.Recipients, ", "))
		rows.directions = append(rows.directions, message.Direction)
		sha, err := hex.DecodeString(message.ContentSHA256)
		if err != nil || len(sha) != 32 {
			return spineRows{}, fmt.Errorf("message %s has a malformed content digest", message.RecordID)
		}
		rows.shas = append(rows.shas, sha)
		rows.charCounts = append(rows.charCounts, int32(len([]rune(message.Body))))
		for _, participant := range message.Participants {
			rows.partMessages = append(rows.partMessages, message.RecordID)
			rows.partRaw = append(rows.partRaw, participant.Raw)
			rows.partRoles = append(rows.partRoles, participant.Role)
			rows.partEntities = append(rows.partEntities, participant.EntityID)
		}
	}
	return rows, nil
}

// insertRecordsAndRoutes writes the conversation's working.normalized_record
// rows and their projection routes, in the caller's transaction.
func insertRecordsAndRoutes(ctx context.Context, tx pgx.Tx, plan firstparty.Plan, conversation firstparty.Conversation, rows spineRows) (int64, error) {
	inserted, err := tx.Exec(ctx, insertSpineRecordsSQL,
		rows.ids, plan.Source.SourceVersionID, firstPartyRecordSource, conversation.ScopedKey,
		firstparty.DeriverVersion, conversation.Corpus, rows.participants, rows.contents, rows.occurred, rows.attrs,
		rows.recordKeys, rows.senders, rows.recipients, rows.tiers)
	if err != nil {
		return 0, fmt.Errorf("insert normalized records: %w", err)
	}
	if _, err := tx.Exec(ctx, insertSpineRoutesSQL, rows.ids, firstparty.DeriverVersion, rows.approved, rows.bases, conversation.Corpus); err != nil {
		return 0, fmt.Errorf("insert message projection routes: %w", err)
	}
	return inserted.RowsAffected(), nil
}

func beginConversation(ctx context.Context, db DB, conversation firstparty.Conversation) (pgx.Tx, error) {
	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	if err := declareDeriver(ctx, tx); err != nil {
		_ = tx.Rollback(ctx)
		return nil, fmt.Errorf("declare the deriver: %w", err)
	}
	if err := lockConversation(ctx, tx, conversation.ScopedKey); err != nil {
		_ = tx.Rollback(ctx)
		return nil, fmt.Errorf("lock conversation: %w", err)
	}
	return tx, nil
}

// commitConversationSpine writes one first-party conversation's spine rows in
// one transaction and returns its thread id and how many records were new.
func (s *FirstPartyContextStore) commitConversationSpine(ctx context.Context, plan firstparty.Plan, conversation firstparty.Conversation) (string, int64, error) {
	tx, err := beginConversation(ctx, s.db, conversation)
	if err != nil {
		return "", 0, err
	}
	defer func() { cleanup, cancel := boundedCleanup(ctx); defer cancel(); _ = tx.Rollback(cleanup) }()
	// One thread per scoped conversation: reuse the id an earlier run of the
	// same conversation committed, else mint a UUIDv7.
	var threadID string
	err = tx.QueryRow(ctx, `
		SELECT message.conversation_id::text
		FROM working.normalized_record record
		JOIN working.message message ON message.id = record.id
		WHERE record.source = $1 AND record.conversation_id = $2
		LIMIT 1`, firstPartyRecordSource, conversation.ScopedKey).Scan(&threadID)
	if errors.Is(err, pgx.ErrNoRows) {
		minted, mintErr := uuid.NewV7()
		if mintErr != nil {
			return "", 0, mintErr
		}
		threadID = minted.String()
	} else if err != nil {
		return "", 0, fmt.Errorf("find the conversation's thread: %w", err)
	}
	rows, err := buildSpineRows(plan, conversation)
	if err != nil {
		return "", 0, err
	}
	inserted, err := insertRecordsAndRoutes(ctx, tx, plan, conversation, rows)
	if err != nil {
		return "", 0, err
	}
	if _, err := tx.Exec(ctx, insertSpineMessagesSQL,
		rows.ids, threadID, plan.Source.Platform, firstparty.DeriverVersion,
		rows.occurred, rows.senders, rows.recipientRaw, rows.directions, rows.shas, rows.charCounts); err != nil {
		return "", 0, fmt.Errorf("insert messages: %w", err)
	}
	if len(rows.partMessages) > 0 {
		if _, err := tx.Exec(ctx, insertSpineParticipantsSQL, rows.partMessages, rows.partRaw, rows.partRoles, rows.partEntities, firstparty.DeriverVersion); err != nil {
			return "", 0, fmt.Errorf("insert message participants: %w", err)
		}
	}

	// Every planned record must now be exactly this plan's row: an id that
	// already existed from another source, conversation or content is refused.
	var matching int
	if err := tx.QueryRow(ctx, `
		SELECT count(*)
		FROM working.normalized_record record
		JOIN working.message message ON message.id = record.id
		WHERE record.id = ANY($1::uuid[]) AND record.source_version_id = $2::uuid
		  AND record.conversation_id = $3 AND message.conversation_id = $4::uuid
		  AND message.derived_from_record_id = record.id AND record.message_corpus = 'first_party'`,
		rows.ids, plan.Source.SourceVersionID, conversation.ScopedKey, threadID).Scan(&matching); err != nil {
		return "", 0, fmt.Errorf("verify the spine rows: %w", err)
	}
	if matching != len(rows.ids) {
		return "", 0, fmt.Errorf("%d of %d records are this plan's spine rows; an existing row disagrees", matching, len(rows.ids))
	}
	if _, err := tx.Exec(ctx, "SET CONSTRAINTS ALL IMMEDIATE"); err != nil {
		return "", 0, fmt.Errorf("message projection validation: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return "", 0, err
	}
	return threadID, inserted, nil
}

// upsertThirdPartyConversationSQL files the conversation under the acquired
// backup's source version (no custody claim; promotion supplies
// source_artifact_id and the acquisition link). Its span widens as later
// chunks arrive; its message count is recomputed from its rows below.
const upsertThirdPartyConversationSQL = `
	INSERT INTO working.third_party_conversation
	    (source_artifact_id, source_version_id, platform, external_thread_key, started_at, ended_at,
	     message_count, review_status, platform_attrs, deriver_version)
	VALUES (NULL, $1::uuid, $2, $3, $4, $5, 0, 'pending', $6::jsonb, $7)
	ON CONFLICT (source_version_id, platform, external_thread_key) WHERE source_version_id IS NOT NULL
	DO UPDATE SET started_at = LEAST(working.third_party_conversation.started_at, EXCLUDED.started_at),
	              ended_at = GREATEST(working.third_party_conversation.ended_at, EXCLUDED.ended_at)
	RETURNING id::text`

// insertThirdPartyMessagesSQL writes the messages and, only for messages new
// in this statement, their participants (third_party_message_participant's
// unique key cannot dedupe a NULL entity, so a retry must not re-add them).
const insertThirdPartyMessagesSQL = `
	WITH inserted AS (
	    INSERT INTO working.third_party_message
	        (id, conversation_id, normalized_record_id, occurred_at, platform, sender_raw, sender_entity_id,
	         content_sha256, platform_attrs, deriver_version)
	    SELECT u.id, $2::uuid, u.id, u.occurred_at, $3, NULLIF(u.sender, ''), NULLIF(u.sender_entity, '')::uuid,
	           u.sha, '{}'::jsonb, $4
	    FROM unnest($1::uuid[], $5::timestamptz[], $6::text[], $7::text[], $8::bytea[])
	         AS u(id, occurred_at, sender, sender_entity, sha)
	    ON CONFLICT (normalized_record_id) DO NOTHING
	    RETURNING id
	)
	INSERT INTO working.third_party_message_participant (message_id, entity_id, participant_raw, role, deriver_version)
	SELECT p.message_id, NULLIF(p.entity_id, '')::uuid, p.participant_raw, p.role, $4
	FROM unnest($9::uuid[], $10::text[], $11::text[], $12::text[]) AS p(message_id, participant_raw, role, entity_id)
	JOIN inserted ON inserted.id = p.message_id`

// commitThirdPartyConversation writes one third-party conversation's rows in
// one transaction: normalized records (acquired_third_party), PROPOSED routes,
// the conversation, its messages (id = normalized record id) and participants.
func (s *FirstPartyContextStore) commitThirdPartyConversation(ctx context.Context, plan firstparty.Plan, conversation firstparty.Conversation) (string, int64, error) {
	tx, err := beginConversation(ctx, s.db, conversation)
	if err != nil {
		return "", 0, err
	}
	defer func() { cleanup, cancel := boundedCleanup(ctx); defer cancel(); _ = tx.Rollback(cleanup) }()
	rows, err := buildSpineRows(plan, conversation)
	if err != nil {
		return "", 0, err
	}
	for _, approved := range rows.approved {
		if approved {
			return "", 0, errors.New("a third-party route may never start approved")
		}
	}
	inserted, err := insertRecordsAndRoutes(ctx, tx, plan, conversation, rows)
	if err != nil {
		return "", 0, err
	}
	first, last := conversation.Coverage()
	attrs, _ := json.Marshal(map[string]any{
		"perspective_person_id": plan.Identity.PerspectivePersonID, "scoped_key": conversation.ScopedKey,
		"conversation_parties": conversation.Parties, "projection_deriver": firstparty.DeriverVersion,
	})
	var conversationID string
	if err := tx.QueryRow(ctx, upsertThirdPartyConversationSQL,
		plan.Source.AcquiredSourceVersionID(), plan.Source.Platform, conversation.Key, first, last,
		string(attrs), firstparty.DeriverVersion).Scan(&conversationID); err != nil {
		return "", 0, fmt.Errorf("file the third-party conversation: %w", err)
	}
	senderEntities := make([]string, len(conversation.Messages))
	for index, message := range conversation.Messages {
		for _, participant := range message.Participants {
			if participant.Role == "from" {
				senderEntities[index] = participant.EntityID
			}
		}
	}
	if _, err := tx.Exec(ctx, insertThirdPartyMessagesSQL,
		rows.ids, conversationID, plan.Source.Platform, firstparty.DeriverVersion,
		rows.occurred, rows.senders, senderEntities, rows.shas,
		rows.partMessages, rows.partRaw, rows.partRoles, rows.partEntities); err != nil {
		return "", 0, fmt.Errorf("insert third-party messages: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE working.third_party_conversation
		SET message_count = (SELECT count(*) FROM working.third_party_message WHERE conversation_id = $1::uuid)
		WHERE id = $1::uuid`, conversationID); err != nil {
		return "", 0, fmt.Errorf("recount the third-party conversation: %w", err)
	}
	var matching int
	if err := tx.QueryRow(ctx, `
		SELECT count(*)
		FROM working.normalized_record record
		JOIN working.third_party_message message ON message.normalized_record_id = record.id
		WHERE record.id = ANY($1::uuid[]) AND record.source_version_id = $2::uuid
		  AND record.conversation_id = $3 AND message.conversation_id = $4::uuid
		  AND message.id = record.id AND record.message_corpus = 'acquired_third_party'`,
		rows.ids, plan.Source.SourceVersionID, conversation.ScopedKey, conversationID).Scan(&matching); err != nil {
		return "", 0, fmt.Errorf("verify the third-party rows: %w", err)
	}
	if matching != len(rows.ids) {
		return "", 0, fmt.Errorf("%d of %d records are this plan's third-party rows; an existing row disagrees", matching, len(rows.ids))
	}
	if _, err := tx.Exec(ctx, "SET CONSTRAINTS ALL IMMEDIATE"); err != nil {
		return "", 0, fmt.Errorf("message projection validation: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return "", 0, err
	}
	return conversationID, inserted, nil
}

// CommitFirstPartyContextThreads implements activities.FirstPartyContextStore.
func (s *FirstPartyContextStore) CommitFirstPartyContextThreads(ctx context.Context, spec activities.FirstPartyCommitSpec) (proffer.Ref, proffer.Ref, error) {
	executionID, prior, found, err := s.beginCommit(ctx, spec, stagegraph.CommitFirstPartyContextThreads, activities.FirstPartyMessagesKind)
	if err != nil {
		return "", "", err
	}
	if found {
		return prior, prior, nil
	}
	mode, admitted := MatterModeForIdentity(spec.Plan.Identity.MatterID, spec.Plan.Identity.CourtCaseID)
	if !admitted {
		return "", "", fmt.Errorf("matter %s with court case %s is not an admitted platform identity", spec.Plan.Identity.MatterID, spec.Plan.Identity.CourtCaseID)
	}
	// Only first-party conversations become first-party context threads.
	// Third-party context threads need an evidence.acquisition, which a
	// Proffer run never creates; promotion supplies it.
	threads := make([]map[string]any, 0, len(spec.Plan.Conversations))
	for _, conversation := range spec.Plan.FirstParty() {
		outcome, err := s.commitConversationThread(ctx, spec.Plan, conversation, mode)
		if err != nil {
			return "", "", fmt.Errorf("commit thread for conversation %s: %w", conversation.Key, err)
		}
		threads = append(threads, outcome)
	}
	receipt, err := s.finishCommit(ctx, executionID, spec.Attempt, commitReceipt{
		RefKind: activities.FirstPartyThreadsKind, Gate: string(spec.GateRef), PlanDigest: spec.Plan.Digest,
		Messages: spec.Plan.MessageCount, Threads: threads,
	})
	if err != nil {
		return "", "", err
	}
	return receipt, receipt, nil
}

func (s *FirstPartyContextStore) commitConversationThread(ctx context.Context, plan firstparty.Plan, conversation firstparty.Conversation, mode string) (map[string]any, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { cleanup, cancel := boundedCleanup(ctx); defer cancel(); _ = tx.Rollback(cleanup) }()
	if err := lockConversation(ctx, tx, conversation.ScopedKey); err != nil {
		return nil, fmt.Errorf("lock conversation: %w", err)
	}
	ids := make([]string, 0, len(conversation.Messages))
	for _, message := range conversation.Messages {
		ids = append(ids, message.RecordID)
	}
	// The thread is the one the spine commit wrote into working.message.
	rows, err := tx.Query(ctx, `
		SELECT conversation_id::text, count(*) FROM working.message WHERE id = ANY($1::uuid[]) GROUP BY 1`, ids)
	if err != nil {
		return nil, fmt.Errorf("read the spine's thread: %w", err)
	}
	var threadID string
	var threadsSeen, messagesSeen int64
	for rows.Next() {
		var id string
		var n int64
		if err := rows.Scan(&id, &n); err != nil {
			rows.Close()
			return nil, err
		}
		threadID, threadsSeen, messagesSeen = id, threadsSeen+1, messagesSeen+n
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if threadsSeen != 1 || messagesSeen != int64(len(ids)) {
		return nil, fmt.Errorf("the conversation's %d messages sit in %d threads with %d spine rows; the spine commit must precede the thread commit", len(ids), threadsSeen, messagesSeen)
	}

	var owner, matter, courtCase string
	err = tx.QueryRow(ctx, selectThreadIdentitySQL, threadID).Scan(&owner, &matter, &courtCase)
	if errors.Is(err, pgx.ErrNoRows) {
		result, err := s.threads.commitVersionTx(ctx, tx, plan.NewThreadVersion(threadID, conversation), mode)
		if err != nil {
			return nil, err
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}
		return map[string]any{
			"thread": result.ContextThreadID, "version": result.ThreadVersionID, "created": true,
			"members_added": result.MembersWritten, "sources_added": result.SourcesWritten,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read thread %s: %w", threadID, err)
	}
	identity := plan.Identity
	if owner != identity.OwnerPersonID || matter != identity.MatterID || courtCase != identity.CourtCaseID {
		return nil, fmt.Errorf("thread %s belongs to owner %s in matter %s case %s, not to this import's identity", threadID, owner, matter, courtCase)
	}
	return s.extendThread(ctx, tx, plan, conversation, threadID)
}

// extendThread adds this generation's messages and source assertion to the
// thread's current version, in any review state (owner 2026-10-02: nothing
// is immutable until promoted to evidence), and recomputes the version's
// bounds, horizon and digest from its rows.
// Byline: Claude Code · Opus 5.5 · 2026-10-02
func (s *FirstPartyContextStore) extendThread(ctx context.Context, tx pgx.Tx, plan firstparty.Plan, conversation firstparty.Conversation, threadID string) (map[string]any, error) {
	var versionID string
	var versionOrdinal int
	if err := tx.QueryRow(ctx, `
		SELECT version.id::text, version.version_ordinal
		FROM working.first_party_context_thread_version version
		WHERE version.context_thread_id = $1::uuid
		  AND NOT EXISTS (SELECT 1 FROM working.first_party_context_thread_version later WHERE later.supersedes_id = version.id)
		ORDER BY version.version_ordinal DESC LIMIT 1
		FOR UPDATE`, threadID).Scan(&versionID, &versionOrdinal); err != nil {
		return nil, fmt.Errorf("read thread %s's current version: %w", threadID, err)
	}

	rows, err := tx.Query(ctx, `
		SELECT message_id::text, thread_ordinal FROM working.first_party_context_thread_message
		WHERE thread_version_id = $1::uuid ORDER BY thread_ordinal`, versionID)
	if err != nil {
		return nil, fmt.Errorf("read thread membership: %w", err)
	}
	members := map[string]bool{}
	ordered := []string{}
	next := int64(0)
	for rows.Next() {
		var id string
		var ordinal int64
		if err := rows.Scan(&id, &ordinal); err != nil {
			rows.Close()
			return nil, err
		}
		members[id] = true
		ordered = append(ordered, id)
		if ordinal >= next {
			next = ordinal + 1
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	added := 0
	for _, message := range conversation.Messages {
		if members[message.RecordID] {
			continue
		}
		member := firstparty.Member(message, next)
		if _, err := tx.Exec(ctx, insertMembershipSQL,
			versionID, threadID, member.MessageID, member.Ordinal, member.OccurredAt,
			member.RequiredForHorizon, member.MembershipConfidence); err != nil {
			return nil, fmt.Errorf("add message %s to thread %s: %w", message.RecordID, threadID, err)
		}
		ordered = append(ordered, message.RecordID)
		next++
		added++
	}

	var hasSource bool
	var nextAnchor int64
	if err := tx.QueryRow(ctx, `
		SELECT coalesce(bool_or(source_version_id = $2::uuid AND assertion_version = 1), false),
		       coalesce(max(source_anchor_ordinal), -1) + 1
		FROM working.first_party_context_thread_source WHERE thread_version_id = $1::uuid`,
		versionID, plan.Source.SourceVersionID).Scan(&hasSource, &nextAnchor); err != nil {
		return nil, fmt.Errorf("read thread source assertions: %w", err)
	}
	sourcesAdded := 0
	if !hasSource {
		if err := insertThreadSource(ctx, tx, versionID, threadID, plan.Identity.PerspectivePersonID,
			plan.SourceAssertion(conversation, nextAnchor)); err != nil {
			return nil, err
		}
		sourcesAdded = 1
	}

	// Bounds and horizon are recomputed from the rows, exactly as the deferred
	// validator recomputes them; GREATEST ignores a NULL side.
	if _, err := tx.Exec(ctx, `
		UPDATE working.first_party_context_thread_version version
		SET first_occurred_at = membership.first_at,
		    last_occurred_at = membership.last_at,
		    knowledge_available_from = GREATEST(membership.required_at, sources.required_at),
		    assertion_digest = $2::bytea
		FROM (SELECT min(occurred_at) AS first_at, max(occurred_at) AS last_at,
		             max(source_available_from) FILTER (WHERE required_for_horizon) AS required_at
		      FROM working.first_party_context_thread_message WHERE thread_version_id = $1::uuid) membership,
		     (SELECT max(source_available_from) FILTER (WHERE required_for_horizon) AS required_at
		      FROM working.first_party_context_thread_source WHERE thread_version_id = $1::uuid) sources
		WHERE version.id = $1::uuid`, versionID, firstparty.MembershipDigest(ordered)); err != nil {
		return nil, fmt.Errorf("extend thread %s version %d: %w", threadID, versionOrdinal, err)
	}
	if _, err := tx.Exec(ctx, "SET CONSTRAINTS ALL IMMEDIATE"); err != nil {
		return nil, fmt.Errorf("thread %s version %d failed its completeness validation: %w", threadID, versionOrdinal, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return map[string]any{
		"thread": threadID, "version": versionID, "created": false,
		"members_added": added, "sources_added": sourcesAdded,
	}, nil
}

package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"

	"github.com/Cursedpotential/probata/engine/approvedgraph"
	"github.com/Cursedpotential/probata/engine/approvedgraphai"
	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/extraction/flow"
	"github.com/Cursedpotential/probata/engine/extraction/service"
	"github.com/Cursedpotential/probata/engine/surrealsink"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ErrSupersededApprovedDecision marks a historical native projection that cannot serve current analysis.
var ErrSupersededApprovedDecision = approvedgraphai.ErrSupersededDecision

// ApprovedCurrentReader binds analytical reads to the existing PostgreSQL owner-decision ledger.
// Inputs: existing platform DB and graph sink. Outputs: current-decision reader and lister.
// Effects: read-only when called; choose for approved queries, walk preparation/finalization and display freshness.
type ApprovedCurrentReader struct {
	db     DB
	Graph  ApprovedGraphQueries
	native approvedgraphai.Reader
}

// ApprovedGraphQueries is the existing bounded analytical read and descriptor seam.
// Inputs: pinned graph scope. Outputs: bounded claims/descriptors. Effects: analytical reads only.
// Choose inside ApprovedCurrentReader; it does not modify projections or original artifacts.
type ApprovedGraphQueries interface {
	QueryApprovedClaims(context.Context, surrealsink.ApprovedQueryScope) (surrealsink.ApprovedQueryResult, error)
	ListApprovedProjections(context.Context, surrealsink.ApprovedProjectionScope) (surrealsink.ApprovedProjectionPage, error)
}

// NewApprovedCurrentReader constructs a current-decision guard on existing DB and graph clients.
// Inputs: platform DB and graph query/lister. Outputs: configured wrapper. Effects: no I/O.
// Choose in worker and starter composition; it preserves the retained-message commit receipt route.
func NewApprovedCurrentReader(db DB, graph ApprovedGraphQueries) (*ApprovedCurrentReader, error) {
	if db == nil || graph == nil {
		return nil, errors.New("approved current reader requires existing DB and graph")
	}
	reader, err := NewApprovedDecisionChecker(db)
	if err != nil {
		return nil, err
	}
	reader.Graph = graph
	return reader, nil
}

// currentClaim resolves native approvals from the ledger or validates an existing retained-message commit.
// Inputs: canonical case and selected revision/digest/generation. Outputs: native claim or nil for message commit.
// Effects: read-only PostgreSQL/source-pin validation; choose before and after analytical reads.
func (r *ApprovedCurrentReader) currentClaim(ctx context.Context, q surrealsink.ApprovedQueryScope) (*approvedgraph.Claim, error) {
	if r == nil || r.db == nil || !caseidentity.AdmittedIdentity(q.MatterID, q.CourtCaseID) {
		return nil, errors.New("current approval requires canonical scope")
	}
	if id, err := uuid.Parse(q.ApprovedRevisionID); err != nil || id.String() != q.ApprovedRevisionID {
		return nil, errors.New("current approval requires canonical revision UUID")
	}
	var target, table string
	var raw []byte
	err := r.db.QueryRow(ctx, `SELECT target_id::text,target_kind,rationale FROM analysis.review_decision WHERE decision_id=$1::uuid`, q.ApprovedRevisionID).Scan(&target, &table, &raw)
	if errors.Is(err, pgx.ErrNoRows) {
		reader, _ := NewApprovedGraphReader(r.db)
		receipt, err := reader.ReadReceipt(ctx, q.ApprovedRevisionID)
		if err != nil {
			return nil, err
		}
		if receipt.Status != "completed" || receipt.Outcome != "committed" || receipt.MatterMode != "LIVE" || receipt.Digest != q.ApprovalDigest || receipt.Actor.SubjectUID == "" || receipt.Actor.Username == "" || receipt.FinishedAt.IsZero() || q.ProjectionGenerationID != flow.DeterministicID("approved_graph_generation", receipt.ID, receipt.Digest) {
			return nil, errors.New("retained-message approved receipt differs")
		}
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var rationale struct {
		SourceVersionID string `json:"source_version_id"`
		RequestDigest   string `json:"request_digest"`
	}
	if json.Unmarshal(raw, &rationale) != nil || (table != "working.candidate_entity" && table != "working.candidate_event" && table != "working.candidate_fact") || rationale.SourceVersionID == "" || rationale.RequestDigest != q.ApprovalDigest || flow.DeterministicID("ai_review_decision", target, q.ApprovalDigest) != q.ApprovedRevisionID {
		return nil, errors.New("native approval ledger scope differs")
	}
	scope := approvedgraphai.Scope{MatterID: q.MatterID, CourtCaseID: q.CourtCaseID, SourceVersionID: rationale.SourceVersionID, CandidateID: target, DecisionID: q.ApprovedRevisionID, RequestDigest: q.ApprovalDigest}
	row, err := r.native.ReadApprovedAI(ctx, scope)
	if err != nil {
		return nil, err
	}
	var attrs struct {
		ReviewDecision struct {
			ID string `json:"decision_id"`
		} `json:"review_decision"`
	}
	if json.Unmarshal(row.Attrs, &attrs) != nil {
		return nil, errors.New("current native receipt malformed")
	}
	if attrs.ReviewDecision.ID != q.ApprovedRevisionID || row.ReviewState != "approved" {
		return nil, ErrSupersededApprovedDecision
	}
	claim, err := approvedgraphai.Resolve(ctx, currentAISnapshot{row}, scope)
	if err != nil {
		return nil, err
	}
	if claim.ControlGenerationID != q.ProjectionGenerationID {
		return nil, errors.New("native approval generation differs")
	}
	return &claim, nil
}

// CheckCurrentApprovedProjection validates the selected owner approval without reading analytical claim bodies.
// Inputs: fixed selected revision metadata. Outputs: nil when still current. Effects: PostgreSQL/source-pin reads.
// Choose immediately before displaying a stored query/walk artifact; old artifacts remain preserved.
func (r *ApprovedCurrentReader) CheckCurrentApprovedProjection(ctx context.Context, q surrealsink.ApprovedQueryScope) error {
	_, err := r.currentClaim(ctx, q)
	return err
}

// QueryApprovedClaims checks current approval before retrieval and rechecks it before releasing any claims.
// Inputs: bounded selected historical/hindsight graph scope. Outputs: cited current claims or error.
// Effects: PostgreSQL and analytical reads only; choose for every walk step and final source-pin revalidation.
func (r *ApprovedCurrentReader) QueryApprovedClaims(ctx context.Context, q surrealsink.ApprovedQueryScope) (surrealsink.ApprovedQueryResult, error) {
	before, err := r.currentClaim(ctx, q)
	if err != nil {
		return surrealsink.ApprovedQueryResult{}, err
	}
	got, err := r.Graph.QueryApprovedClaims(ctx, q)
	if err != nil {
		return surrealsink.ApprovedQueryResult{}, err
	}
	after, err := r.currentClaim(ctx, q)
	if err != nil {
		return surrealsink.ApprovedQueryResult{}, err
	}
	if !reflect.DeepEqual(before, after) {
		return surrealsink.ApprovedQueryResult{}, ErrSupersededApprovedDecision
	}
	if before != nil {
		for _, c := range got.Claims {
			if !reflect.DeepEqual(c, *before) {
				return surrealsink.ApprovedQueryResult{}, errors.New("native graph claim differs from current owner availability/source pins")
			}
		}
	}
	return got, nil
}

// ListApprovedProjections omits superseded native revisions from each bounded descriptor page.
// Inputs: existing paged case selection. Outputs: current descriptors with unchanged continuation.
// Effects: graph metadata and PostgreSQL checks; choose for projection selection without erasing historical rows.
func (r *ApprovedCurrentReader) ListApprovedProjections(ctx context.Context, s surrealsink.ApprovedProjectionScope) (surrealsink.ApprovedProjectionPage, error) {
	page, err := r.Graph.ListApprovedProjections(ctx, s)
	if err != nil {
		return page, err
	}
	items := make([]surrealsink.ApprovedProjectionDescriptor, 0, len(page.Items))
	for _, item := range page.Items {
		q := surrealsink.ApprovedQueryScope{MatterID: item.MatterID, CourtCaseID: item.CourtCaseID, ApprovedRevisionID: item.ApprovedRevisionID, ApprovalDigest: item.ApprovalDigest, ProjectionGenerationID: item.ProjectionGenerationID}
		if err := r.CheckCurrentApprovedProjection(ctx, q); err != nil {
			if errors.Is(err, ErrSupersededApprovedDecision) {
				continue
			}
			return surrealsink.ApprovedProjectionPage{}, err
		}
		items = append(items, item)
	}
	page.Items = items
	return page, nil
}

var _ service.AIAvailabilityReviewStore = (*EntityExtractionStore)(nil)

// currentAISnapshot supplies the already-read current receipt to the existing strict resolver.
// Inputs: exact persisted snapshot. Outputs: same snapshot. Effects: none; choose to avoid a second unbound read.
type currentAISnapshot struct{ row approvedgraphai.Snapshot }

// ReadApprovedAI returns the selected persisted snapshot for strict ledger/receipt verification.
// Inputs: selected scope. Outputs: snapshot. Effects: none; choose inside the current-decision adapter only.
func (r currentAISnapshot) ReadApprovedAI(context.Context, approvedgraphai.Scope) (approvedgraphai.Snapshot, error) {
	return r.row, nil
}

// NewApprovedDecisionChecker constructs the metadata-only guard for existing workflow status routes.
// Inputs: existing platform DB. Outputs: live decision checker. Effects: no I/O.
// Choose for stored artifact display checks; analytical query methods require NewApprovedCurrentReader.
func NewApprovedDecisionChecker(db DB) (*ApprovedCurrentReader, error) {
	if db == nil {
		return nil, errors.New("current decision checker requires existing DB")
	}
	native, err := NewApprovedAIGraphReader(db)
	if err != nil {
		return nil, err
	}
	return &ApprovedCurrentReader{db: db, native: native}, nil
}

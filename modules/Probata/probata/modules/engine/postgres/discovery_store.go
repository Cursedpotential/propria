package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/discovery"
	"github.com/jackc/pgx/v5"
	"strconv"
	"time"
)

// DiscoveryStore is the canonical analysis-only PG repository; normalized source inputs are never writable through it.
type DiscoveryStore struct {
	db       DB
	datasets []discovery.Dataset
}

// NewDiscoveryStore validates server-bound datasets and injects the existing PG connection without I/O.
func NewDiscoveryStore(db DB, ds []discovery.Dataset) (*DiscoveryStore, error) {
	if db == nil || discovery.ValidateDatasets(ds) != nil {
		return nil, discovery.ErrInvalid
	}
	return &DiscoveryStore{db, append([]discovery.Dataset{}, ds...)}, nil
}

var _ discovery.Store = (*DiscoveryStore)(nil)

// MarkFailed records an exhausted finite execution only if its durable feedback revision remains current.
// Inputs: exact scope/revision and protected stage diagnostic. Outputs: error. Effects: PG run metadata only.
func (s *DiscoveryStore) MarkFailed(ctx context.Context, id string, scope discovery.Scope, revision int64, reason string) error {
	if reason != "assembly" && reason != "scoring" && reason != "publication" {
		return discovery.ErrInvalid
	}
	tx, e := s.begin(ctx)
	if e != nil {
		return e
	}
	defer discoveryRollback(ctx, tx)
	run, e := lockDiscoveryRun(ctx, tx, id, scope)
	if e != nil {
		return e
	}
	if run.FeedbackRevision != revision {
		return discovery.ErrConflict
	}
	if run.Status == "failed" || run.Status == "ready" || run.Status == "ready_partial" {
		return tx.Commit(ctx)
	}
	prior := run.Revision
	run.Revision++
	run.Status = "failed"
	run.DispatchPending = false
	run.FailureStage = reason
	run.Error = "discovery " + reason + " failed; inspect its protected Temporal execution"
	run.UpdatedAt = time.Now().UTC()
	if e = writeDiscoveryRun(ctx, tx, run, prior); e != nil {
		return e
	}
	return tx.Commit(ctx)
}

// LoadManifest reads an immutable scoped input snapshot for replay or later feedback rescores.
// Inputs: run/scope and exact feedback revision. Outputs: external ref and stored batch. Effects: SELECTs only.
// Choose before re-reading inputs; a recorded run never silently changes its source membership.
func (s *DiscoveryStore) LoadManifest(ctx context.Context, id string, scope discovery.Scope, revision int64) (discovery.Ref, discovery.Batch, error) {
	if _, e := s.ReadRun(ctx, id, scope); e != nil {
		return discovery.Ref{}, discovery.Batch{}, e
	}
	var reference, raw []byte
	e := s.db.QueryRow(ctx, `SELECT manifest_ref,payload FROM analysis.discovery_input WHERE run_id=$1::uuid AND feedback_revision=$2`, id, revision).Scan(&reference, &raw)
	if errors.Is(e, pgx.ErrNoRows) {
		return discovery.Ref{}, discovery.Batch{}, discovery.ErrNotFound
	}
	if e != nil {
		return discovery.Ref{}, discovery.Batch{}, e
	}
	var ref discovery.Ref
	var b discovery.Batch
	if json.Unmarshal(reference, &ref) != nil || json.Unmarshal(raw, &b) != nil || b.RunID != id || b.Scope != scope || b.FeedbackRevision != revision {
		return ref, b, discovery.ErrInvalid
	}
	return ref, b, nil
}

// Datasets returns matching server definitions; inputs scope, outputs descriptors, effects none.
func (s *DiscoveryStore) Datasets(_ context.Context, scope discovery.Scope) ([]discovery.Dataset, error) {
	if scope.Validate() != nil {
		return nil, discovery.ErrScope
	}
	out := []discovery.Dataset{}
	for _, d := range s.datasets {
		if d.Scope == scope {
			out = append(out, d)
		}
	}
	return out, nil
}
func readDiscoveryRun(row pgx.Row) (discovery.Run, error) {
	var raw []byte
	var r discovery.Run
	if e := row.Scan(&raw); e != nil {
		if errors.Is(e, pgx.ErrNoRows) {
			return r, discovery.ErrNotFound
		}
		return r, e
	}
	e := json.Unmarshal(raw, &r)
	return r, e
}

const discoveryRunRead = `SELECT payload FROM analysis.discovery_run WHERE run_id=$1::uuid AND matter_id=$2::uuid AND court_case_id=$3::uuid AND operating_mode=$4`

func (s *DiscoveryStore) begin(ctx context.Context) (pgx.Tx, error) {
	return s.db.BeginTx(ctx, pgx.TxOptions{})
}
func discoveryRollback(ctx context.Context, tx pgx.Tx) {
	cleanup, cancel := boundedCleanup(ctx)
	defer cancel()
	_ = tx.Rollback(cleanup)
}
func lockDiscoveryRun(ctx context.Context, tx pgx.Tx, id string, scope discovery.Scope) (discovery.Run, error) {
	return readDiscoveryRun(tx.QueryRow(ctx, discoveryRunRead+` FOR UPDATE`, id, scope.MatterID, scope.CourtCaseID, scope.Mode))
}
func writeDiscoveryRun(ctx context.Context, tx pgx.Tx, r discovery.Run, prior int64) error {
	raw, e := json.Marshal(r)
	if e != nil {
		return e
	}
	tag, e := tx.Exec(ctx, `UPDATE analysis.discovery_run SET revision=$2,feedback_revision=$3,score_revision=$4,status=$5,dispatch_pending=$6,payload=$7::jsonb,updated_at=$8 WHERE run_id=$1::uuid AND revision=$9`, r.Request.RunID, r.Revision, r.FeedbackRevision, r.ScoreRevision, r.Status, r.DispatchPending, raw, r.UpdatedAt, prior)
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return discovery.ErrConflict
	}
	return nil
}

// CreateRun persists one actor/scoped idempotent run and pending dispatch without contacting Temporal.
// Inputs: validated server-bound request. Outputs: canonical Run. Effects: analysis row insert only; DEV writes are fenced.
func (s *DiscoveryStore) CreateRun(ctx context.Context, in discovery.RunRequest) (discovery.Run, error) {
	in.Normalize()
	if e := in.Validate(); e != nil {
		return discovery.Run{}, e
	}
	if e := caseidentity.RequireCanonicalWrite(caseidentity.Mode(in.Scope.Mode)); e != nil {
		return discovery.Run{}, e
	}
	if _, e := discovery.Selection(in, s.datasets); e != nil {
		return discovery.Run{}, e
	}
	now := time.Now().UTC()
	r := discovery.Run{Request: in, Revision: 1, Status: "pending", DispatchPending: true, CreatedAt: now, UpdatedAt: now}
	raw, _ := json.Marshal(r)
	tx, e := s.begin(ctx)
	if e != nil {
		return r, e
	}
	defer discoveryRollback(ctx, tx)
	_, e = tx.Exec(ctx, `INSERT INTO analysis.discovery_run(run_id,matter_id,court_case_id,operating_mode,dataset_id,actor_uid,revision,feedback_revision,score_revision,status,dispatch_pending,payload,created_at,updated_at) VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,$6,1,0,0,'pending',true,$7::jsonb,$8,$8) ON CONFLICT(run_id) DO NOTHING`, in.RunID, in.Scope.MatterID, in.Scope.CourtCaseID, in.Scope.Mode, in.DatasetID, in.Actor.SubjectUID, raw, now)
	if e != nil {
		return r, e
	}
	stored, e := lockDiscoveryRun(ctx, tx, in.RunID, in.Scope)
	if e != nil {
		return r, e
	}
	if discovery.Fingerprint(stored.Request) != discovery.Fingerprint(in) {
		return r, discovery.ErrConflict
	}
	if e = tx.Commit(ctx); e != nil {
		return r, e
	}
	return stored, nil
}

// ReadRun reads one exact case/mode-scoped run; inputs identity/scope, outputs canonical state, effects SELECT only.
func (s *DiscoveryStore) ReadRun(ctx context.Context, id string, scope discovery.Scope) (discovery.Run, error) {
	if !discovery.ID(id) || scope.Validate() != nil {
		return discovery.Run{}, discovery.ErrInvalid
	}
	return readDiscoveryRun(s.db.QueryRow(ctx, discoveryRunRead, id, scope.MatterID, scope.CourtCaseID, scope.Mode))
}

// ListCandidates reads the bounded current ranking queue with explicit continuation and current owner review state.
// Inputs: scoped run, offset cursor and <=100 limit. Outputs: HTTP-only candidate bodies/CAS revision. Effects: SELECT only.
func (s *DiscoveryStore) ListCandidates(ctx context.Context, id string, scope discovery.Scope, cursor string, limit int) (discovery.CandidatePage, error) {
	r, e := s.ReadRun(ctx, id, scope)
	if e != nil {
		return discovery.CandidatePage{}, e
	}
	if limit < 1 || limit > discovery.MaxCandidates {
		return discovery.CandidatePage{}, discovery.ErrInvalid
	}
	offset := 0
	if cursor != "" {
		offset, e = strconv.Atoi(cursor)
		if e != nil || offset < 0 || offset > discovery.MaxCandidates {
			return discovery.CandidatePage{}, discovery.ErrInvalid
		}
	}
	rows, e := s.db.Query(ctx, `SELECT c.payload,COALESCE((SELECT f.payload->>'kind' FROM analysis.discovery_feedback f WHERE f.run_id=c.run_id AND f.candidate_id=c.candidate_id ORDER BY f.revision DESC LIMIT 1),'unreviewed') FROM analysis.discovery_candidate c WHERE c.run_id=$1::uuid AND c.score_revision=$2 ORDER BY c.score DESC,c.candidate_id LIMIT $3 OFFSET $4`, id, r.ScoreRevision, limit+1, offset)
	if e != nil {
		return discovery.CandidatePage{}, e
	}
	defer rows.Close()
	page := discovery.CandidatePage{Items: []discovery.Candidate{}, CurrentRevision: r.Revision, Revision: r.Revision}
	for rows.Next() {
		var raw []byte
		var state string
		if e = rows.Scan(&raw, &state); e != nil {
			return page, e
		}
		var c discovery.Candidate
		if e = json.Unmarshal(raw, &c); e != nil {
			return page, e
		}
		c.ReviewState = state
		page.Items = append(page.Items, c)
	}
	if e = rows.Err(); e != nil {
		return page, e
	}
	if len(page.Items) > limit {
		page.Items = page.Items[:limit]
		page.NextCursor = strconv.Itoa(offset + limit)
	}
	return page, nil
}

// ReadContext returns the selected candidate's complete bounded native episode and genuine source locators.
// Inputs: scoped run and candidate ID. Outputs: HTTP-only source bodies/current revision. Effects: analysis snapshot SELECTs only.
func (s *DiscoveryStore) ReadContext(ctx context.Context, id string, scope discovery.Scope, candidateID string) (discovery.Context, error) {
	r, e := s.ReadRun(ctx, id, scope)
	if e != nil {
		return discovery.Context{}, e
	}
	var raw, batchRaw []byte
	var state string
	e = s.db.QueryRow(ctx, `SELECT c.payload,i.payload,COALESCE((SELECT f.payload->>'kind' FROM analysis.discovery_feedback f WHERE f.run_id=c.run_id AND f.candidate_id=c.candidate_id ORDER BY f.revision DESC LIMIT 1),'unreviewed') FROM analysis.discovery_candidate c JOIN analysis.discovery_score score ON score.run_id=c.run_id AND score.score_revision=c.score_revision JOIN analysis.discovery_input i ON i.run_id=score.run_id AND i.feedback_revision=score.feedback_revision WHERE c.run_id=$1::uuid AND c.score_revision=$2 AND c.candidate_id=$3`, id, r.ScoreRevision, candidateID).Scan(&raw, &batchRaw, &state)
	if errors.Is(e, pgx.ErrNoRows) {
		return discovery.Context{}, discovery.ErrNotFound
	}
	if e != nil {
		return discovery.Context{}, e
	}
	var c discovery.Candidate
	var b discovery.Batch
	if json.Unmarshal(raw, &c) != nil || json.Unmarshal(batchRaw, &b) != nil {
		return discovery.Context{}, discovery.ErrInvalid
	}
	out := discovery.Context{Candidate: c, Records: []discovery.SourceRecord{}, CurrentRevision: r.Revision, Revision: r.Revision, ReviewState: state}
	out.Candidate.ReviewState = state
	ids := map[string]bool{}
	for _, ep := range b.Episodes {
		if ep.EpisodeID == c.EpisodeID {
			for _, recordID := range ep.RecordIDs {
				ids[recordID] = true
			}
		}
	}
	for _, record := range b.Records {
		if ids[record.RecordID] {
			out.Records = append(out.Records, record)
		}
	}
	return out, nil
}

// SubmitFeedback saves an append-only idempotent correction, increments CAS/ranking input revisions and marks durable rescore dispatch.
// Inputs: scoped run, feedback key/revision and authenticated actor. Outputs: saved decision/run. Effects: analysis-only transaction.
// Choose for review corrections; wrong_person and skip retain distinct taxonomy rather than becoming all-label negatives.
func (s *DiscoveryStore) SubmitFeedback(ctx context.Context, id string, scope discovery.Scope, in discovery.FeedbackRequest, actor discovery.Actor) (discovery.FeedbackReceipt, error) {
	if in.Validate() != nil || actor.Validate() != nil {
		return discovery.FeedbackReceipt{}, discovery.ErrInvalid
	}
	if e := caseidentity.RequireCanonicalWrite(caseidentity.Mode(scope.Mode)); e != nil {
		return discovery.FeedbackReceipt{}, e
	}
	tx, e := s.begin(ctx)
	if e != nil {
		return discovery.FeedbackReceipt{}, e
	}
	defer discoveryRollback(ctx, tx)
	run, e := lockDiscoveryRun(ctx, tx, id, scope)
	if e != nil {
		return discovery.FeedbackReceipt{}, e
	}
	var priorRaw []byte
	e = tx.QueryRow(ctx, `SELECT payload FROM analysis.discovery_feedback WHERE feedback_id=$1::uuid`, in.FeedbackID).Scan(&priorRaw)
	if e == nil {
		var f discovery.Feedback
		if json.Unmarshal(priorRaw, &f) != nil || f.Actor != actor || discovery.Fingerprint(f.FeedbackRequest) != discovery.Fingerprint(in) {
			return discovery.FeedbackReceipt{}, discovery.ErrConflict
		}
		var belongs bool
		if e = tx.QueryRow(ctx, `SELECT run_id=$2::uuid FROM analysis.discovery_feedback WHERE feedback_id=$1::uuid`, in.FeedbackID, id).Scan(&belongs); e != nil || !belongs {
			return discovery.FeedbackReceipt{}, discovery.ErrConflict
		}
		if e = tx.Commit(ctx); e != nil {
			return discovery.FeedbackReceipt{}, e
		}
		return discovery.FeedbackReceipt{Feedback: f, Run: run}, nil
	}
	if !errors.Is(e, pgx.ErrNoRows) {
		return discovery.FeedbackReceipt{}, e
	}
	if run.Revision != in.ExpectedRevision {
		return discovery.FeedbackReceipt{}, discovery.ErrConflict
	}
	var candidateRaw, batchRaw []byte
	e = tx.QueryRow(ctx, `SELECT c.payload,i.payload FROM analysis.discovery_candidate c JOIN analysis.discovery_score score ON score.run_id=c.run_id AND score.score_revision=c.score_revision JOIN analysis.discovery_input i ON i.run_id=score.run_id AND i.feedback_revision=score.feedback_revision WHERE c.run_id=$1::uuid AND c.score_revision=$2 AND c.candidate_id=$3`, id, run.ScoreRevision, in.CandidateID).Scan(&candidateRaw, &batchRaw)
	if errors.Is(e, pgx.ErrNoRows) {
		return discovery.FeedbackReceipt{}, discovery.ErrNotFound
	}
	if e != nil {
		return discovery.FeedbackReceipt{}, e
	}
	var candidate discovery.Candidate
	var batch discovery.Batch
	if json.Unmarshal(candidateRaw, &candidate) != nil || json.Unmarshal(batchRaw, &batch) != nil {
		return discovery.FeedbackReceipt{}, discovery.ErrInvalid
	}
	allowed := map[string]bool{}
	example := ""
	inputPins := map[string]string{}
	for _, ep := range batch.Episodes {
		if ep.EpisodeID == candidate.EpisodeID {
			for _, rid := range ep.RecordIDs {
				allowed[rid] = true
			}
		}
	}
	for _, record := range batch.Records {
		if allowed[record.RecordID] {
			inputPins[record.RecordID] = discovery.SourcePin(record)
			example += record.Text + "\n"
		}
	}
	for _, rid := range in.RecordIDs {
		if !allowed[rid] {
			return discovery.FeedbackReceipt{}, discovery.ErrScope
		}
	}
	all := map[string]bool{}
	for _, record := range batch.Records {
		all[record.RecordID] = true
	}
	for _, group := range in.EpisodeGroups {
		for _, rid := range group {
			if !all[rid] {
				return discovery.FeedbackReceipt{}, discovery.ErrScope
			}
		}
	}
	runes := []rune(example)
	if len(runes) > 4000 {
		example = string(runes[:4000])
	}
	now := time.Now().UTC()
	f := discovery.Feedback{FeedbackRequest: in, RunID: id, Revision: run.FeedbackRevision + 1, Actor: actor, CreatedAt: now, ExampleText: example, PreviousLabels: append([]string{}, candidate.Labels...), InputPins: inputPins}
	raw, _ := json.Marshal(f)
	_, e = tx.Exec(ctx, `INSERT INTO analysis.discovery_feedback(feedback_id,run_id,candidate_id,revision,payload,created_at) VALUES($1::uuid,$2::uuid,$3,$4,$5::jsonb,$6)`, in.FeedbackID, id, in.CandidateID, f.Revision, raw, now)
	if e != nil {
		return discovery.FeedbackReceipt{}, e
	}
	prior := run.Revision
	run.Revision++
	run.FeedbackRevision = f.Revision
	run.DispatchPending = true
	run.Status = "rescore_pending"
	run.UpdatedAt = now
	if e = writeDiscoveryRun(ctx, tx, run, prior); e != nil {
		return discovery.FeedbackReceipt{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return discovery.FeedbackReceipt{}, e
	}
	return discovery.FeedbackReceipt{Feedback: f, Run: run}, nil
}

// FeedbackForRun reads finite same-run and compatible historical training decisions for source-linked re-ranking.
// Inputs: scoped run. Outputs: immutable decisions ordered deterministically. Effects: analysis SELECT only.
func (s *DiscoveryStore) FeedbackForRun(ctx context.Context, id string, scope discovery.Scope) ([]discovery.Feedback, error) {
	run, e := s.ReadRun(ctx, id, scope)
	if e != nil {
		return nil, e
	}
	hypotheses, _ := json.Marshal(run.Request.Hypotheses)
	generations, _ := json.Marshal(run.Request.GenerationIDs)
	rows, e := s.db.Query(ctx, `SELECT f.payload FROM analysis.discovery_feedback f JOIN analysis.discovery_run r ON r.run_id=f.run_id WHERE r.matter_id=$1::uuid AND r.court_case_id=$2::uuid AND r.operating_mode=$3 AND r.dataset_id=$4 AND r.payload->'request'->>'selection_revision'=$5 AND (f.run_id=$6::uuid OR (f.created_at <= $7 AND r.payload->'request'->'hypotheses'=$8::jsonb AND r.payload->'request'->'generation_ids'=$9::jsonb AND r.payload->'request'->>'input_backend'=$10)) ORDER BY f.created_at,f.feedback_id LIMIT 257`, scope.MatterID, scope.CourtCaseID, scope.Mode, run.Request.DatasetID, run.Request.SelectionRevision, id, run.CreatedAt, hypotheses, generations, run.Request.InputBackend)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []discovery.Feedback{}
	for rows.Next() {
		var raw []byte
		if e = rows.Scan(&raw); e != nil {
			return nil, e
		}
		var f discovery.Feedback
		if e = json.Unmarshal(raw, &f); e != nil {
			return nil, e
		}
		if len(out) == discovery.MaxFeedback {
			return nil, fmt.Errorf("%w: feedback history exceeds finite scoring budget", discovery.ErrInvalid)
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// SaveManifest retains one immutable finite source/feedback snapshot without advancing owner review revision.
// Inputs: exact run/feedback revision and external batch ref. Outputs: error. Effects: analysis input insert and run ref only.
func (s *DiscoveryStore) SaveManifest(ctx context.Context, id string, scope discovery.Scope, feedbackRevision int64, ref discovery.Ref, batch discovery.Batch) error {
	tx, e := s.begin(ctx)
	if e != nil {
		return e
	}
	defer discoveryRollback(ctx, tx)
	run, e := lockDiscoveryRun(ctx, tx, id, scope)
	if e != nil {
		return e
	}
	if run.FeedbackRevision != feedbackRevision || batch.RunID != id || batch.Scope != scope {
		return discovery.ErrConflict
	}
	raw, _ := json.Marshal(batch)
	reference, _ := json.Marshal(ref)
	_, e = tx.Exec(ctx, `INSERT INTO analysis.discovery_input(run_id,feedback_revision,manifest_ref,payload,created_at) VALUES($1::uuid,$2,$3::jsonb,$4::jsonb,now()) ON CONFLICT(run_id,feedback_revision) DO NOTHING`, id, feedbackRevision, reference, raw)
	if e != nil {
		return e
	}
	var stored []byte
	if e = tx.QueryRow(ctx, `SELECT manifest_ref FROM analysis.discovery_input WHERE run_id=$1::uuid AND feedback_revision=$2`, id, feedbackRevision).Scan(&stored); e != nil {
		return e
	}
	var priorRef discovery.Ref
	if json.Unmarshal(stored, &priorRef) != nil || priorRef != ref {
		return discovery.ErrConflict
	}
	run.ManifestRef = &ref
	run.InputRecordCount = len(batch.Records)
	run.InputTruncated = batch.Truncated
	run.Status = "scoring"
	run.UpdatedAt = time.Now().UTC()
	if e = writeDiscoveryRun(ctx, tx, run, run.Revision); e != nil {
		return e
	}
	return tx.Commit(ctx)
}

// PersistResult atomically publishes one cited score revision and indexed candidate/citation rows in PG.
// Inputs: exact score receipt, immutable result and manifest. Outputs: updated run. Effects: analysis rows only; stale CAS fails.
func (s *DiscoveryStore) PersistResult(ctx context.Context, id string, scope discovery.Scope, receipt discovery.ScoreReceipt, result discovery.Result, batch discovery.Batch) (discovery.Run, error) {
	if e := discovery.ValidateResult(result, batch); e != nil {
		return discovery.Run{}, e
	}
	tx, e := s.begin(ctx)
	if e != nil {
		return discovery.Run{}, e
	}
	defer discoveryRollback(ctx, tx)
	run, e := lockDiscoveryRun(ctx, tx, id, scope)
	if e != nil {
		return run, e
	}
	if run.FeedbackRevision != receipt.FeedbackRevision {
		return run, discovery.ErrConflict
	}
	var existing []byte
	e = tx.QueryRow(ctx, `SELECT result_ref FROM analysis.discovery_score WHERE run_id=$1::uuid AND feedback_revision=$2`, id, receipt.FeedbackRevision).Scan(&existing)
	if e == nil {
		var ref discovery.Ref
		if json.Unmarshal(existing, &ref) != nil || ref != receipt.ResultRef {
			return run, discovery.ErrConflict
		}
		if e = tx.Commit(ctx); e != nil {
			return run, e
		}
		return run, nil
	}
	if !errors.Is(e, pgx.ErrNoRows) {
		return run, e
	}
	next := run.ScoreRevision + 1
	resultRef, _ := json.Marshal(receipt.ResultRef)
	metadata, _ := json.Marshal(result)
	_, e = tx.Exec(ctx, `INSERT INTO analysis.discovery_score(run_id,score_revision,feedback_revision,model_version,result_ref,payload,created_at) VALUES($1::uuid,$2,$3,$4,$5::jsonb,$6::jsonb,now())`, id, next, receipt.FeedbackRevision, result.ModelVersion, resultRef, metadata)
	if e != nil {
		return run, e
	}
	for _, c := range result.Candidates {
		raw, _ := json.Marshal(c)
		_, e = tx.Exec(ctx, `INSERT INTO analysis.discovery_candidate(run_id,score_revision,candidate_id,episode_id,score,payload) VALUES($1::uuid,$2,$3,$4,$5,$6::jsonb)`, id, next, c.CandidateID, c.EpisodeID, c.Score, raw)
		if e != nil {
			return run, e
		}
		for ordinal, cite := range c.Citations {
			raw, _ := json.Marshal(cite)
			_, e = tx.Exec(ctx, `INSERT INTO analysis.discovery_citation(run_id,score_revision,candidate_id,ordinal,record_id,locator_id,payload) VALUES($1::uuid,$2,$3,$4,$5::uuid,$6::uuid,$7::jsonb)`, id, next, c.CandidateID, ordinal, cite.RecordID, cite.LocatorID, raw)
			if e != nil {
				return run, e
			}
		}
	}
	prior := run.Revision
	run.Revision++
	run.ScoreRevision = next
	run.Status = "ready"
	if batch.Truncated {
		run.Status = "ready_partial"
	}
	run.DispatchPending = false
	run.UpdatedAt = time.Now().UTC()
	if e = writeDiscoveryRun(ctx, tx, run, prior); e != nil {
		return run, e
	}
	if e = tx.Commit(ctx); e != nil {
		return run, e
	}
	return run, nil
}

// RecordDispatch acknowledges a real Temporal execution only if its feedback revision remains current.
// Inputs: scoped run/revision and actual execution IDs. Outputs: error. Effects: durable dispatch receipt and run metadata.
func (s *DiscoveryStore) RecordDispatch(ctx context.Context, id string, scope discovery.Scope, revision int64, workflowID, temporalRunID string) error {
	if workflowID == "" || temporalRunID == "" || len(workflowID) > 200 || len(temporalRunID) > 200 {
		return discovery.ErrInvalid
	}
	tx, e := s.begin(ctx)
	if e != nil {
		return e
	}
	defer discoveryRollback(ctx, tx)
	run, e := lockDiscoveryRun(ctx, tx, id, scope)
	if e != nil {
		return e
	}
	if run.FeedbackRevision != revision {
		return discovery.ErrConflict
	}
	tag, e := tx.Exec(ctx, `INSERT INTO analysis.discovery_dispatch(run_id,feedback_revision,workflow_id,temporal_run_id,created_at) VALUES($1::uuid,$2,$3,$4,now()) ON CONFLICT(run_id,feedback_revision) DO UPDATE SET temporal_run_id=EXCLUDED.temporal_run_id WHERE analysis.discovery_dispatch.workflow_id=EXCLUDED.workflow_id`, id, revision, workflowID, temporalRunID)
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return discovery.ErrConflict
	}
	run.WorkflowID = workflowID
	run.TemporalRunID = temporalRunID
	run.DispatchPending = false
	run.UpdatedAt = time.Now().UTC()
	if e = writeDiscoveryRun(ctx, tx, run, run.Revision); e != nil {
		return e
	}
	return tx.Commit(ctx)
}

// PendingDispatch reads a bounded restart-safe outbox of owner-authorized discovery starts/rescores.
// Inputs: <=100 limit. Outputs: saved Runs. Effects: SELECT only; external dispatch belongs to the root worker.
func (s *DiscoveryStore) PendingDispatch(ctx context.Context, limit int) ([]discovery.Run, error) {
	if limit < 1 || limit > 100 {
		return nil, discovery.ErrInvalid
	}
	rows, e := s.db.Query(ctx, `SELECT payload FROM analysis.discovery_run WHERE dispatch_pending ORDER BY updated_at,run_id LIMIT $1`, limit)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []discovery.Run{}
	for rows.Next() {
		r, e := readDiscoveryRun(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

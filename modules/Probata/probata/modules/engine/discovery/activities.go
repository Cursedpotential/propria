// Byline: Codex · GPT-6.1-sol · 2026-10-11
package discovery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
)

// WorkRequest identifies one finite score revision; no source or candidate bodies cross history.
type WorkRequest struct {
	RunID            string `json:"run_id"`
	Scope            Scope  `json:"scope"`
	FeedbackRevision int64  `json:"feedback_revision"`
}

// PersistRequest identifies one Python receipt at one immutable input batch reference.
type PersistRequest struct {
	WorkRequest
	BatchRef Ref          `json:"batch_ref"`
	Receipt  ScoreReceipt `json:"receipt"`
}

// Completion is the finite canonical publication result retained in history.
type Completion struct {
	RunID            string `json:"run_id"`
	Revision         int64  `json:"revision"`
	ScoreRevision    int64  `json:"score_revision"`
	FeedbackRevision int64  `json:"feedback_revision"`
	CandidateCount   int    `json:"candidate_count"`
}

// FailureRequest contains only protected failure metadata for a durable finite run.
type FailureRequest struct {
	WorkRequest
	Reason string `json:"reason"`
}

// Fail records a current-revision failure in PG so failed work does not remain indefinitely shown as scoring.
// Inputs: run/scope/revision and safe stage reason. Outputs: none. Effects: analysis run metadata only.
// Choose after an exhausted Activity; an older execution cannot fail a newer owner revision.
func (a Activities) Fail(ctx context.Context, in FailureRequest) error {
	if a.Store == nil {
		return ErrUnavailable
	}
	return a.Store.MarkFailed(ctx, in.RunID, in.Scope, in.FeedbackRevision, in.Reason)
}

// Activities composes three atomic analysis operations using injected PG/read-only input/storage ports.
type Activities struct {
	Store       Store
	Reader      SourceReader
	Files       *Files
	ModelConfig json.RawMessage
}

func (a Activities) valid() error {
	if a.Store == nil || a.Reader == nil || a.Files == nil {
		return ErrUnavailable
	}
	return nil
}

// Assemble snapshots one server-bound, horizon-prefiltered input selection and current feedback outside history.
// Inputs: exact run/scope/feedback revision. Outputs: Python batch ref. Effects: input SELECTs, derivative write and PG manifest only.
// Choose before scoring; it never extracts, hashes custody, promotes evidence or mutates normalized inputs.
func (a Activities) Assemble(ctx context.Context, in WorkRequest) (ScoreRequest, error) {
	if e := a.valid(); e != nil {
		return ScoreRequest{}, e
	}
	run, e := a.Store.ReadRun(ctx, in.RunID, in.Scope)
	if e != nil {
		return ScoreRequest{}, e
	}
	if run.FeedbackRevision != in.FeedbackRevision {
		return ScoreRequest{}, ErrConflict
	}
	if ref, _, loadErr := a.Store.LoadManifest(ctx, in.RunID, in.Scope, in.FeedbackRevision); loadErr == nil {
		return ScoreRequest{RunID: in.RunID, BatchRef: ref, FeedbackRevision: in.FeedbackRevision}, nil
	} else if !errors.Is(loadErr, ErrNotFound) {
		return ScoreRequest{}, loadErr
	}
	var records []SourceRecord
	var truncated bool
	if in.FeedbackRevision > 0 {
		_, original, loadErr := a.Store.LoadManifest(ctx, in.RunID, in.Scope, 0)
		if loadErr != nil {
			return ScoreRequest{}, loadErr
		}
		records, truncated = original.Records, original.Truncated
	} else {
		records, truncated, e = a.Reader.ReadSources(ctx, run.Request)
		if e != nil {
			return ScoreRequest{}, e
		}
	}
	feedback, e := a.Store.FeedbackForRun(ctx, in.RunID, in.Scope)
	if e != nil {
		return ScoreRequest{}, e
	}
	currentPins := map[string]string{}
	for _, r := range records {
		currentPins[r.RecordID] = SourcePin(r)
	}
	eligible := feedback[:0]
	for _, f := range feedback {
		if f.RunID != in.RunID {
			compatible := len(f.InputPins) > 0
			for id, revision := range f.InputPins {
				if currentPins[id] != revision {
					compatible = false
				}
			}
			if !compatible {
				continue
			}
		}
		if f.RunID == in.RunID && f.Revision > in.FeedbackRevision {
			continue
		}
		if run.Request.Filters.Perspective == "as_lived" && f.CreatedAt.After(*run.Request.Filters.Cutoff) {
			continue
		}
		eligible = append(eligible, f)
	}
	batch := Batch{Schema: BatchSchema, RunID: in.RunID, Scope: in.Scope, DatasetID: run.Request.DatasetID, SelectionRevision: run.Request.SelectionRevision, FeedbackRevision: in.FeedbackRevision, Records: records, Episodes: Episodes(records), Hypotheses: run.Request.Hypotheses, Feedback: eligible, Filters: run.Request.Filters, ModelVersion: ModelVersion, MaxCandidates: run.Request.MaxCandidates, Truncated: truncated, Config: a.ModelConfig}
	applyEpisodeFeedback(&batch)
	ref, e := a.Files.Put(in.RunID, fmt.Sprintf("batch-f%d", in.FeedbackRevision), batch)
	if e != nil {
		return ScoreRequest{}, e
	}
	if e = a.Store.SaveManifest(ctx, in.RunID, in.Scope, in.FeedbackRevision, ref, batch); e != nil {
		return ScoreRequest{}, e
	}
	return ScoreRequest{RunID: in.RunID, BatchRef: ref, FeedbackRevision: in.FeedbackRevision}, nil
}
func applyEpisodeFeedback(b *Batch) {
	for _, f := range b.Feedback {
		if f.Kind != "correct_episode" {
			continue
		}
		known := map[string]bool{}
		for _, r := range b.Records {
			known[r.RecordID] = true
		}
		moved := map[string]bool{}
		groups := []Episode{}
		valid := true
		for _, g := range f.EpisodeGroups {
			for _, id := range g {
				if !known[id] || moved[id] {
					valid = false
				}
				moved[id] = true
			}
			groups = append(groups, Episode{EpisodeID: "episode:" + Fingerprint(g)[:32], RecordIDs: g})
		}
		if !valid {
			continue
		}
		kept := []Episode{}
		for _, ep := range b.Episodes {
			ids := []string{}
			for _, id := range ep.RecordIDs {
				if !moved[id] {
					ids = append(ids, id)
				}
			}
			if len(ids) > 0 {
				ep.RecordIDs = ids
				kept = append(kept, ep)
			}
		}
		b.Episodes = append(kept, groups...)
	}
	sort.Slice(b.Episodes, func(i, j int) bool { return b.Episodes[i].EpisodeID < b.Episodes[j].EpisodeID })
}

// Persist validates a bounded Python score generation against its immutable cited inputs and publishes PG candidates.
// Inputs: exact batch and Python result refs. Outputs: counts/revisions. Effects: derivative reads and one analysis-only transaction.
// Choose after scoring; stale feedback revisions never overwrite a newer ranking.
func (a Activities) Persist(ctx context.Context, in PersistRequest) (Completion, error) {
	if e := a.valid(); e != nil {
		return Completion{}, e
	}
	var batch Batch
	var result Result
	if e := a.Files.Read(in.BatchRef, &batch); e != nil {
		return Completion{}, e
	}
	if e := a.Files.Read(in.Receipt.ResultRef, &result); e != nil {
		return Completion{}, e
	}
	if in.RunID != batch.RunID || in.Scope != batch.Scope || in.FeedbackRevision != batch.FeedbackRevision || in.Receipt.RunID != in.RunID || in.Receipt.FeedbackRevision != in.FeedbackRevision || in.Receipt.CandidateCount != len(result.Candidates) || in.Receipt.ModelVersion != result.ModelVersion {
		return Completion{}, ErrInvalid
	}
	if e := ValidateResult(result, batch); e != nil {
		return Completion{}, e
	}
	run, e := a.Store.PersistResult(ctx, in.RunID, in.Scope, in.Receipt, result, batch)
	if e != nil {
		return Completion{}, e
	}
	return Completion{RunID: in.RunID, Revision: run.Revision, ScoreRevision: run.ScoreRevision, FeedbackRevision: run.FeedbackRevision, CandidateCount: len(result.Candidates)}, nil
}

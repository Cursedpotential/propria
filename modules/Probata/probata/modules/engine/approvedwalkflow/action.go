// Package approvedwalkflow durably compares bounded as-lived analytical steps with separate hindsight.
package approvedwalkflow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/approvedgraph"
	"github.com/Cursedpotential/probata/engine/approvedgraphqueryflow"
	"github.com/Cursedpotential/probata/engine/extraction/libraryvalidation"
)

const (
	WorkflowName     = "approved_context_walk_workflow"
	WorkflowIDPrefix = "approved-context-walk:"
	StatusQuery      = "status"
	PrepareActivity  = "prepare_approved_walk_step_activity"
	ReasonActivity   = "reason_approved_walk_step_activity"
	FinalizeActivity = "finalize_approved_walk_activity"
	PythonQueue      = "evidence-pipeline"
	PrefixRoot       = "consignatio/casevault/DerivedKnowledge/analysis/walks/"
	MaxSteps         = 20
	MaxPromptBytes   = 96 << 10
)

// Request selects exact approved source/control pins and explicitly ordered historical cutoffs.
// Inputs: existing query binding and <=20 cutoffs. Outputs: durable walk request. Effects: none; use for one selected window.
type Request struct {
	Query   approvedgraphqueryflow.Request `json:"query"`
	Cutoffs []time.Time                    `json:"cutoffs"`
}

// Validate rejects ambiguous perspectives, paging and unbounded historical requests.
// Inputs: request. Outputs: error. Effects: none; choose before scheduling any query or model Activity.
func (r Request) Validate() error {
	if err := r.Query.Validate(); err != nil {
		return err
	}
	if r.Query.Scope.Perspective != "as_lived" || r.Query.Scope.Cursor != "" || len(r.Query.RequestID) > 96 || len(r.Cutoffs) < 1 || len(r.Cutoffs) > MaxSteps {
		return errors.New("walk requires one to twenty ordered cutoffs and an unpaged as-lived scope")
	}
	for i, cutoff := range r.Cutoffs {
		if cutoff.IsZero() || cutoff.After(r.Query.Scope.Horizon) || (i > 0 && !cutoff.After(r.Cutoffs[i-1])) {
			return errors.New("walk cutoffs must increase and remain within the requested horizon")
		}
	}
	if !r.Cutoffs[len(r.Cutoffs)-1].Equal(r.Query.Scope.Horizon) {
		return errors.New("last walk cutoff must equal the requested horizon")
	}
	return nil
}

// StepRequest carries refs/control metadata only between Go orchestration and atomic preparation.
type StepRequest struct {
	Request  Request                        `json:"request"`
	Index    int                            `json:"index"`
	Previous *libraryvalidation.ArtifactRef `json:"previous,omitempty"`
}

// Reference keeps source/model bodies outside Temporal history.
type Reference struct {
	Artifact libraryvalidation.ArtifactRef `json:"artifact"`
}

// Prepared returns one immutable eligible-input reference and explicit retrieval coverage.
type Prepared struct {
	Reference
	HasMore    bool `json:"has_more"`
	ClaimCount int  `json:"claim_count"`
}

// Citation retains the exact approved claim and native source locator beside its quotation.
type Citation struct {
	ClaimID   string              `json:"claim_id"`
	Quote     string              `json:"quote"`
	SourcePin approvedgraph.Claim `json:"source_pin"`
}

// Finding is an analytical candidate, never a truth, evidence, clinical or legal determination.
type Finding struct {
	Kind      string     `json:"kind"`
	Text      string     `json:"text"`
	Citations []Citation `json:"citations"`
}

// Checkpoint is the versioned one-step reasoning result whose citations are re-resolved before release.
type Checkpoint struct {
	InputArtifact libraryvalidation.ArtifactRef `json:"input_artifact"`
	Schema        string                        `json:"schema"`
	Perspective   string                        `json:"perspective"`
	Horizon       *time.Time                    `json:"horizon"`
	ScopeDigest   string                        `json:"scope_digest"`
	StepIndex     int                           `json:"step_index"`
	InputDigest   string                        `json:"input_digest"`
	ClaimIDs      []string                      `json:"claim_ids"`
	Findings      []Finding                     `json:"findings"`
	Model         json.RawMessage               `json:"model"`
	PromptVersion string                        `json:"prompt_version"`
	PromptSHA256  string                        `json:"prompt_sha256"`
	Usage         json.RawMessage               `json:"usage"`
	ReviewStatus  string                        `json:"review_status"`
	Limitations   []string                      `json:"limitations"`
}

type stepBody struct {
	Schema      string                `json:"schema"`
	Perspective string                `json:"perspective"`
	Horizon     *time.Time            `json:"horizon"`
	ScopeDigest string                `json:"scope_digest"`
	StepIndex   int                   `json:"step_index"`
	Claims      []approvedgraph.Claim `json:"claims"`
	Previous    *Checkpoint           `json:"previous,omitempty"`
}

// FinalizeRequest passes only step/checkpoint references to final citation revalidation.
type FinalizeRequest struct {
	Request Request     `json:"request"`
	Steps   []Reference `json:"steps"`
	Inputs  []Prepared  `json:"inputs"`
}

// Result exposes immutable final-result and checkpoint references without analytical content.
type Result struct {
	Artifact     libraryvalidation.ArtifactRef `json:"artifact"`
	Checkpoints  []Reference                   `json:"checkpoints"`
	Partial      bool                          `json:"partial"`
	ReviewStatus string                        `json:"review_status"`
}

// Activities composes existing approved retrieval and versioned derivatives without an ingestion path.
type Activities struct {
	Query     approvedgraphqueryflow.Activities
	Artifacts libraryvalidation.B2Artifacts
}

func scopeDigest(in Request) string {
	scope := in.Query.Scope
	scope.Perspective = ""
	scope.Horizon = time.Time{}
	raw, _ := json.Marshal(struct {
		Actor string
		Scope any
	}{in.Query.Actor.SubjectUID, scope})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func (a Activities) read(ctx context.Context, ref libraryvalidation.ArtifactRef, value any) error {
	raw, err := a.Artifacts.Read(ctx, ref, approvedgraphqueryflow.MaxResultBytes)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, value)
}

func (a Activities) valid() error {
	if a.Query.Reader == nil || a.Query.Artifacts.Validate() != nil || a.Artifacts.Validate() != nil || a.Artifacts.Bucket != approvedgraphqueryflow.ApprovedQueryBucket || !strings.HasPrefix(a.Artifacts.Prefix, PrefixRoot) || len(a.Artifacts.Prefix) <= len(PrefixRoot) {
		return errors.New("walk requires a separate versioned analysis derivative prefix")
	}
	return nil
}

// Prepare retrieves a horizon-prefiltered page and constructs one chronological eligible model input.
// Inputs: exact query/cutoff and optional earlier same-perspective checkpoint ref. Outputs: input ref/coverage only.
// Effects: existing approved query, exact artifact reads and derivative write; choose independently per cutoff.
func (a Activities) Prepare(ctx context.Context, in StepRequest) (Prepared, error) {
	if err := in.Request.Validate(); err != nil {
		return Prepared{}, err
	}
	if err := a.valid(); err != nil {
		return Prepared{}, err
	}
	if in.Index < 0 || in.Index > len(in.Request.Cutoffs) || (in.Index == len(in.Request.Cutoffs) && in.Previous != nil) || (in.Index == 0 && in.Previous != nil) || (in.Index > 0 && in.Index < len(in.Request.Cutoffs) && in.Previous == nil) {
		return Prepared{}, errors.New("walk step index or prior-state boundary invalid")
	}
	query := in.Request.Query
	query.RequestID = fmt.Sprintf("%s-step-%d", query.RequestID, in.Index)
	var horizon *time.Time
	if in.Index == len(in.Request.Cutoffs) {
		query.Scope.Perspective = "hindsight"
		query.Scope.Horizon = time.Time{}
	} else {
		h := in.Request.Cutoffs[in.Index]
		horizon = &h
		query.Scope.Horizon = h
	}
	// Query.Run calls the existing WHERE horizon filter before retrieval/limit; no hindsight page seeds as-lived.
	result, err := a.Query.Run(ctx, query)
	if err != nil {
		return Prepared{}, err
	}
	var snapshot struct {
		Claims []approvedgraph.Claim `json:"claims"`
	}
	raw, err := a.Query.Artifacts.Read(ctx, result.Artifact, approvedgraphqueryflow.MaxResultBytes)
	if err != nil {
		return Prepared{}, err
	}
	if err = json.Unmarshal(raw, &snapshot); err != nil {
		return Prepared{}, err
	}
	claims := snapshot.Claims
	if claims == nil {
		claims = []approvedgraph.Claim{}
	}
	sort.SliceStable(claims, func(i, j int) bool {
		x, y := claims[i].SourceAvailableFrom, claims[j].SourceAvailableFrom
		if x == nil {
			if y == nil {
				return claims[i].ID < claims[j].ID
			}
			return false
		}
		if y == nil {
			return true
		}
		if x.Equal(*y) {
			return claims[i].ID < claims[j].ID
		}
		return x.Before(*y)
	})
	body := stepBody{Schema: "approved-walk-step-v1", Perspective: query.Scope.Perspective, Horizon: horizon, ScopeDigest: scopeDigest(in.Request), StepIndex: in.Index, Claims: claims}
	if in.Previous != nil {
		var previous Checkpoint
		if err := a.read(ctx, *in.Previous, &previous); err != nil {
			return Prepared{}, err
		}
		if previous.Perspective != "as_lived" || previous.Horizon == nil || previous.Horizon.After(*horizon) || previous.ScopeDigest != body.ScopeDigest || previous.StepIndex != in.Index-1 {
			return Prepared{}, errors.New("prior checkpoint crossed knowledge horizon")
		}
		// A desc-limited page may lose prior claims. Never feed unresolvable old summaries into the new step.
		if err := validateCitations(previous, claims); err != nil {
			return Prepared{}, fmt.Errorf("prior state is outside this bounded page: %w", err)
		}
		body.Previous = &previous
	}
	encoded, err := json.Marshal(body)
	if err != nil || len(encoded) > MaxPromptBytes {
		return Prepared{}, errors.New("walk step exceeds finite model input byte budget")
	}
	ref, err := a.Artifacts.Put(ctx, fmt.Sprintf("%s/step-%d.json", query.RequestID, in.Index), encoded, "application/json")
	if err != nil {
		return Prepared{}, err
	}
	return Prepared{Reference: Reference{Artifact: ref}, HasMore: result.HasMore, ClaimCount: len(claims)}, nil
}

func validateCitations(check Checkpoint, claims []approvedgraph.Claim) error {
	allowed := map[string]approvedgraph.Claim{}
	for _, c := range claims {
		allowed[c.ID] = c
	}
	if len(check.Findings) > 16 {
		return errors.New("finding limit exceeded")
	}
	kinds := map[string]bool{"supporting": true, "contrary": true, "positive": true, "missing_context": true, "alternative": true, "uncertainty": true}
	for _, f := range check.Findings {
		if !kinds[f.Kind] || len([]rune(f.Text)) < 1 || len([]rune(f.Text)) > 1000 || len(f.Citations) < 1 || len(f.Citations) > 8 {
			return errors.New("invalid bounded finding")
		}
		for _, cite := range f.Citations {
			claim, ok := allowed[cite.ClaimID]
			if !ok || strings.TrimSpace(cite.Quote) == "" || !strings.Contains(claim.Text, cite.Quote) || !reflect.DeepEqual(claim, cite.SourcePin) {
				return errors.New("citation no longer resolves to exact eligible source pin")
			}
		}
	}
	return nil
}

// Finalize re-queries every horizon and re-resolves all citations before writing the deterministic comparison.
// Inputs: pinned inputs/checkpoints and original scope. Outputs: final derivative ref/coverage. Effects: graph reads and B2 write/readback.
// Pick after separate as-lived/hindsight steps; review remains pending and no source or evidence state changes.
func (a Activities) Finalize(ctx context.Context, in FinalizeRequest) (Result, error) {
	if err := in.Request.Validate(); err != nil {
		return Result{}, err
	}
	if err := a.valid(); err != nil {
		return Result{}, err
	}
	if len(in.Steps) != len(in.Request.Cutoffs)+1 || len(in.Inputs) != len(in.Steps) {
		return Result{}, errors.New("walk checkpoint count invalid")
	}
	checkpoints := make([]Checkpoint, len(in.Steps))
	partial := false
	for i, ref := range in.Steps {
		var input stepBody
		if err := a.read(ctx, in.Inputs[i].Artifact, &input); err != nil {
			return Result{}, err
		}
		if err := a.read(ctx, ref.Artifact, &checkpoints[i]); err != nil {
			return Result{}, err
		}
		check := checkpoints[i]
		q := in.Request.Query.Scope
		var horizon *time.Time
		if i == len(in.Request.Cutoffs) {
			q.Perspective = "hindsight"
			q.Horizon = time.Time{}
		} else {
			h := in.Request.Cutoffs[i]
			q.Horizon = h
			horizon = &h
		}
		if input.Schema != "approved-walk-step-v1" || input.ScopeDigest != scopeDigest(in.Request) || input.Perspective != q.Perspective || input.StepIndex != i || !reflect.DeepEqual(input.Horizon, horizon) || (i == len(in.Request.Cutoffs) && input.Previous != nil) {
			return Result{}, errors.New("walk input reference binding mismatch")
		}
		if !reflect.DeepEqual(check.InputArtifact, in.Inputs[i].Artifact) || (i == 0 && input.Previous != nil) || (i > 0 && i < len(in.Request.Cutoffs) && (input.Previous == nil || !reflect.DeepEqual(*input.Previous, checkpoints[i-1]))) {
			return Result{}, errors.New("checkpoint exact input reference or prior chain mismatch")
		}
		if check.Schema != "approved-walk-checkpoint-v1" || check.Perspective != q.Perspective || !reflect.DeepEqual(check.Horizon, horizon) || check.ScopeDigest != scopeDigest(in.Request) || check.StepIndex != i || check.ReviewStatus != "pending" || check.PromptVersion != "approved-walk-balanced-v1" || len(check.Model) == 0 || len(check.PromptSHA256) != 64 {
			return Result{}, errors.New("reasoning checkpoint pins mismatch")
		}
		fresh, err := a.Query.Reader.QueryApprovedClaims(ctx, q)
		if err != nil {
			return Result{}, err
		}
		current := map[string]approvedgraph.Claim{}
		for _, c := range fresh.Claims {
			current[c.ID] = c
		}
		if len(current) != len(input.Claims) || len(check.ClaimIDs) != len(input.Claims) {
			return Result{}, errors.New("approved input set changed during walk")
		}
		ids := map[string]bool{}
		for _, id := range check.ClaimIDs {
			if ids[id] {
				return Result{}, errors.New("duplicate checkpoint claim")
			}
			ids[id] = true
		}
		for _, c := range input.Claims {
			if !reflect.DeepEqual(current[c.ID], c) || !ids[c.ID] || (horizon != nil && (c.SourceAvailableFrom == nil || c.SourceAvailableFrom.After(*horizon))) {
				return Result{}, errors.New("approved source pin or horizon changed during walk")
			}
		}
		if err := validateCitations(check, fresh.Claims); err != nil {
			return Result{}, err
		}
		partial = partial || in.Inputs[i].HasMore || fresh.HasMore
	}
	before, after := checkpoints[len(checkpoints)-2], checkpoints[len(checkpoints)-1]
	known := map[string]bool{}
	for _, id := range before.ClaimIDs {
		known[id] = true
	}
	added := []string{}
	unknown := []string{}
	later := []string{}
	families := map[string][]string{}
	for _, id := range after.ClaimIDs {
		if !known[id] {
			added = append(added, id)
		}
	}
	var finalInput stepBody
	if err := a.read(ctx, in.Inputs[len(in.Inputs)-1].Artifact, &finalInput); err != nil {
		return Result{}, err
	}
	for _, c := range finalInput.Claims {
		families[c.SourceSHA256] = append(families[c.SourceSHA256], c.ID)
		if c.SourceAvailableFrom == nil {
			unknown = append(unknown, c.ID)
		} else if c.SourceAvailableFrom.After(in.Request.Query.Scope.Horizon) {
			later = append(later, c.ID)
		}
	}
	sort.Strings(later)
	for hash := range families {
		sort.Strings(families[hash])
	}
	sort.Strings(added)
	sort.Strings(unknown)
	// Exact wording comparison is mechanical, not a truth/deception classification.
	key := func(f Finding) string {
		b, _ := json.Marshal(f)
		sum := sha256.Sum256(b)
		return hex.EncodeToString(sum[:])
	}
	old := map[string]bool{}
	for _, f := range before.Findings {
		old[key(f)] = true
	}
	changed := []Finding{}
	for _, f := range after.Findings {
		if !old[key(f)] {
			changed = append(changed, f)
		}
	}
	output := map[string]any{"schema": "approved-walk-result-v1", "scope_digest": scopeDigest(in.Request), "query": in.Request.Query, "cutoffs": in.Request.Cutoffs, "checkpoints": in.Steps, "inputs": in.Inputs, "partial": partial, "review_status": "pending", "comparison": map[string]any{"hindsight_only_claim_ids": added, "available_after_last_cutoff_claim_ids": later, "unknown_availability_claim_ids": unknown, "hindsight_findings_not_identical_to_last_as_lived": changed}, "exact_duplicate_source_families": families, "family_independence": "Unresolved beyond exact source hashes; multiple copies and AI assertions are not independent corroboration.", "limits": map[string]any{"max_steps": MaxSteps, "claim_page_limit": in.Request.Query.Scope.Limit, "max_prompt_bytes": MaxPromptBytes, "max_input_token_upper_bound_per_step": MaxPromptBytes + 4096, "max_output_tokens_per_step": 3000, "max_model_calls_including_activity_retries": 2 * len(in.Steps), "max_workflow_minutes": 60}, "coverage": "Selected cutoff pages only; descending retrieval is sorted chronologically inside each page. A limited page is not complete case history.", "validation": "Exact approved source pins, native locators and approved claim-text quotations re-resolved at every cutoff before release; original-text entailment and source independence require human review."}
	raw, err := json.Marshal(output)
	if err != nil || len(raw) > approvedgraphqueryflow.MaxResultBytes {
		return Result{}, errors.New("final walk artifact exceeds byte budget")
	}
	artifact, err := a.Artifacts.Put(ctx, in.Request.Query.RequestID+"/result.json", raw, "application/json")
	if err != nil {
		return Result{}, err
	}
	return Result{Artifact: artifact, Checkpoints: in.Steps, Partial: partial, ReviewStatus: "pending"}, nil
}

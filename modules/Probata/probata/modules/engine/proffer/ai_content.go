// Byline: Codex · GPT-6 · 2026-10-06.
package proffer

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/stagegraph"
	"go.temporal.io/sdk/workflow"
)

const (
	aiContentChangeID                      = "proffer-ai-conversation-content-v1"
	aiContentVersion                       = workflow.Version(1)
	aiContentBoundsChangeID                = "proffer-ai-content-bounds-v2"
	aiContentProviderRetryChangeID         = "proffer-ai-provider-cooldown-v1"
	aiContentBundleHashChangeID            = "proffer-ai-content-bundle-sha256-v1"
	aiContentNativeSourceChangeID          = "proffer-ai-native-source-only-v1"
	aiContentMaxRecords                    = 1024
	aiContentMaxChunks                     = 256
	aiContentMaxModelCalls                 = 512
	AIPrepareContentActivityName           = "ai_prepare_content_activity"
	AIExtractWorkProductsActivityName      = "ai_extract_work_products_activity"
	AIExtractCandidatesActivityName        = "ai_extract_candidates_activity"
	AIEmbedContentActivityName             = "ai_embed_content_activity"
	AIPublishContentActivityName           = "ai_publish_content_activity"
	AIVerifyContentPublicationActivityName = "ai_verify_content_publication_activity"
)

// AIContentRequest pins one bounded AI-content operation to its verified source generation.
// Inputs: source/case identities, immutable predecessor bundle references and resource limits.
// Outputs: each Python Activity returns AIContentResult; no conversation text enters history.
// Effects: defined by the named Activity. Choose for verified AI exports, never human messaging.
type AIContentRequest struct {
	RequestID              string  `json:"request_id"`
	SourceVersionID        string  `json:"source_version_id"`
	NormalizedGenerationID string  `json:"normalized_generation_id"`
	VerificationID         string  `json:"verification_id"`
	OriginalRef            Ref     `json:"original_ref,omitempty"`
	NativeSourceOnly       bool    `json:"native_source_only,omitempty"`
	SourceObjectID         string  `json:"source_object_id,omitempty"`
	SourceSHA256           string  `json:"source_sha256,omitempty"`
	VersionID              *string `json:"version_id,omitempty"`
	OperatingMode          string  `json:"operating_mode"`
	MatterID               string  `json:"matter_id"`
	CourtCaseID            string  `json:"court_case_id"`
	PreparedRef            Ref     `json:"prepared_ref,omitempty"`
	WorkProductsRef        Ref     `json:"work_products_ref,omitempty"`
	CandidatesRef          Ref     `json:"candidates_ref,omitempty"`
	EmbeddingsRef          Ref     `json:"embeddings_ref,omitempty"`
	PublicationRef         Ref     `json:"publication_ref,omitempty"`
	MaxRecords             int     `json:"max_records,omitempty"`
	MaxTextBytes           int     `json:"max_text_bytes,omitempty"`
	MaxChunks              int     `json:"max_chunks,omitempty"`
	MaxModelCalls          int     `json:"max_model_calls,omitempty"`
}

// AIContentResult reports one persisted content operation using references and counts only.
// Inputs: the Activity's exact source pins. Outputs: a retained bundle, method and counters.
// Effects: none in Go. Choose instead of transmitting chunks, candidates or vectors in history.
type AIContentResult struct {
	RequestID              string  `json:"request_id"`
	OperatingMode          string  `json:"operating_mode"`
	MatterID               string  `json:"matter_id"`
	CourtCaseID            string  `json:"court_case_id"`
	Stage                  string  `json:"stage"`
	BundleRef              Ref     `json:"bundle_ref"`
	BundleSHA256           string  `json:"bundle_sha256,omitempty"`
	SourceObjectID         string  `json:"source_object_id,omitempty"`
	SourceSHA256           string  `json:"source_sha256,omitempty"`
	VersionID              *string `json:"version_id,omitempty"`
	SourceVersionID        string  `json:"source_version_id"`
	NormalizedGenerationID string  `json:"normalized_generation_id"`
	VerificationID         string  `json:"verification_id"`
	Conversations          int     `json:"conversations"`
	Records                int     `json:"records"`
	Chunks                 int     `json:"chunks"`
	Candidates             int     `json:"candidates"`
	WorkProducts           int     `json:"work_products"`
	ModelCalls             int     `json:"model_calls"`
	ObjectsWritten         int     `json:"objects_written"`
	ObjectsVerified        int     `json:"objects_verified"`
	ModelID                string  `json:"model_id"`
	Method                 string  `json:"method"`
}

// AIContentSummary exposes retained output references for the workflow's completed AI path.
// Inputs: independently settled content results. Outputs: references and aggregate counts.
// Effects: retained in the workflow result. Choose to locate content without embedding payloads.
type AIContentSummary struct {
	PreparedRef         Ref     `json:"prepared_ref"`
	WorkProductsRef     Ref     `json:"work_products_ref"`
	CandidatesRef       Ref     `json:"candidates_ref"`
	EmbeddingsRef       Ref     `json:"embeddings_ref"`
	PublicationRef      Ref     `json:"publication_ref"`
	VerificationRef     Ref     `json:"verification_ref"`
	PreparedSHA256      string  `json:"prepared_sha256,omitempty"`
	WorkProductsSHA256  string  `json:"work_products_sha256,omitempty"`
	CandidatesSHA256    string  `json:"candidates_sha256,omitempty"`
	EmbeddingsSHA256    string  `json:"embeddings_sha256,omitempty"`
	PublicationSHA256   string  `json:"publication_sha256,omitempty"`
	VerificationSHA256  string  `json:"verification_sha256,omitempty"`
	SourceObjectID      string  `json:"source_object_id,omitempty"`
	SourceSHA256        string  `json:"source_sha256,omitempty"`
	VersionID           *string `json:"version_id,omitempty"`
	CandidateStageRunID string  `json:"candidate_stage_run_id,omitempty"`
	StagedOccurrences   int     `json:"staged_occurrences,omitempty"`
	HeldUnclassified    int     `json:"held_unclassified,omitempty"`
	HeldManifest        int     `json:"held_manifest,omitempty"`
	HeldDuplicate       int     `json:"held_duplicate,omitempty"`
	Conversations       int     `json:"conversations"`
	Chunks              int     `json:"chunks"`
	Candidates          int     `json:"candidates"`
	WorkProducts        int     `json:"work_products"`
}

// validate checks exact provenance and bounded counts before another Activity can be scheduled.
// Inputs: original pins and expected terminal stage. Outputs: an error on missing or drifting evidence.
// Effects: none. Choose at every Activity boundary, including readback, rather than trusting success text.
func (out AIContentResult) validate(req AIContentRequest, stage string, requireBundleHash bool) error {
	if out.WorkProducts < 0 {
		return fmt.Errorf("AI content %s returned a negative work-product count", stage)
	}
	if out.Stage != stage || strings.TrimSpace(string(out.BundleRef)) == "" || len(out.BundleRef) > 4096 {
		return fmt.Errorf("AI content %s returned an invalid stage or bundle reference", stage)
	}
	if (requireBundleHash || out.BundleSHA256 != "") && !aiBundleSHA256Pattern.MatchString(out.BundleSHA256) {
		return fmt.Errorf("AI content %s returned an invalid persisted bundle SHA-256", stage)
	}
	if out.RequestID != req.RequestID || out.OperatingMode != req.OperatingMode || out.MatterID != req.MatterID || out.CourtCaseID != req.CourtCaseID || out.SourceVersionID != req.SourceVersionID || out.NormalizedGenerationID != req.NormalizedGenerationID || out.VerificationID != req.VerificationID {
		return fmt.Errorf("AI content %s changed verified source/generation pins", stage)
	}
	if req.NativeSourceOnly {
		if out.SourceObjectID != string(req.OriginalRef) || !aiBundleSHA256Pattern.MatchString(out.SourceSHA256) {
			return fmt.Errorf("AI content %s changed the retained original pin", stage)
		}
		if req.SourceSHA256 != "" && (out.SourceSHA256 != req.SourceSHA256 || !sameAIVersionID(out.VersionID, req.VersionID)) {
			return fmt.Errorf("AI content %s changed the prepared source pin", stage)
		}
	}
	if out.Conversations < 0 || out.Records < 0 || out.Records > req.MaxRecords || out.Chunks < 0 || out.Chunks > req.MaxChunks || out.Candidates < 0 || out.ModelCalls < 0 || out.ModelCalls > req.MaxModelCalls || out.ObjectsWritten < 0 || out.ObjectsWritten > req.MaxChunks || out.ObjectsVerified < 0 || out.ObjectsVerified > req.MaxChunks {
		return fmt.Errorf("AI content %s returned out-of-bounds counts", stage)
	}
	return nil
}

var aiBundleSHA256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// sameAIVersionID compares optional provider version pins without collapsing unknown into empty.
// Inputs: optional version strings. Outputs: exact equality. Effects: none.
// Choose at native Activity boundaries when provider version is known.
func sameAIVersionID(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// execAIContent schedules separate prepare, extraction, embedding, publication and readback Activities.
// Inputs: this run's exact normalized generation and verification receipt. Outputs: retained result references.
// Effects: Python workers persist scoped derived bundles and additive conversation-chunk search objects.
// Choose after verified AI applicability; it replaces the per-record context-search publisher entirely.
func (r *run) execAIContent(ctx workflow.Context, generation, verification Ref) (*AIContentSummary, error) {
	return r.execAIContentWithSource(ctx, generation, verification, "")
}

// execAIContentWithSource runs the six bounded Activities directly on a retained native export.
// Inputs: either verified generation pins or the retained original object ID. Outputs: six bundle references.
// Effects: Python workers persist derived bundles and chunk search projections. Choose the original form
// only for the version-gated native AI route before parser selection.
func (r *run) execAIContentWithSource(ctx workflow.Context, generation, verification, original Ref) (*AIContentSummary, error) {
	req := AIContentRequest{
		RequestID: r.requestID, SourceVersionID: string(r.sourceVersionRef),
		NormalizedGenerationID: string(generation), VerificationID: string(verification),
		OperatingMode: r.operatingMode, MatterID: r.matterID, CourtCaseID: r.courtCaseID,
		OriginalRef: original, NativeSourceOnly: original != "", SourceObjectID: string(original),
		MaxRecords: 256, MaxTextBytes: 2097152, MaxChunks: 128, MaxModelCalls: 128,
	}
	// Preserve already scheduled limits when replaying the initial bounded rollout.
	if workflow.GetVersion(ctx, aiContentBoundsChangeID, workflow.DefaultVersion, 1) != workflow.DefaultVersion {
		req.MaxRecords, req.MaxChunks, req.MaxModelCalls = aiContentMaxRecords, aiContentMaxChunks, aiContentMaxModelCalls
	}
	if req.SourceVersionID == "" || (original == "" && (req.NormalizedGenerationID == "" || req.VerificationID == "")) ||
		(original != "" && (generation != "" || verification != "")) {
		return nil, fmt.Errorf("AI content needs either a retained original or the run's verified generation")
	}
	summary := &AIContentSummary{}
	// Retain the version marker for recorded histories while requiring new bundle
	// receipts only on the native source route. Earlier normalized-generation
	// Activities returned no bundle hash and must replay with their original shape.
	_ = workflow.GetVersion(ctx, aiContentBundleHashChangeID, workflow.DefaultVersion, 1)
	requireBundleHash := req.NativeSourceOnly
	steps := []struct{ name, stage string }{
		{AIPrepareContentActivityName, "prepared"},
		{AIExtractWorkProductsActivityName, "work_products"},
		{AIExtractCandidatesActivityName, "candidates"},
		{AIEmbedContentActivityName, "embedded"},
		{AIPublishContentActivityName, "published"},
		{AIVerifyContentPublicationActivityName, "verified"},
	}
	for _, step := range steps {
		id := stagegraph.StageID(step.name)
		r.markStageStarted(id)
		var out AIContentResult
		options := chunkActivityOptions(2 * time.Hour)
		if step.name == AIExtractCandidatesActivityName && workflow.GetVersion(ctx, aiContentProviderRetryChangeID, workflow.DefaultVersion, 1) != workflow.DefaultVersion {
			options = aiExtractionActivityOptions()
		}
		err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, options), step.name, req).Get(ctx, &out)
		r.markStageSettled(id)
		if err == nil {
			err = out.validate(req, step.stage, requireBundleHash)
		}
		if err == nil && step.stage == "prepared" && (out.Conversations < 1 || out.Chunks < 1) {
			err = fmt.Errorf("AI content preparation returned no conversation chunks")
		}
		if err == nil && (step.stage == "embedded" || step.stage == "published" || step.stage == "verified") && out.Chunks != summary.Chunks {
			err = fmt.Errorf("AI content %s changed the prepared chunk count", step.stage)
		}
		if err == nil && step.stage == "published" && out.ObjectsWritten != summary.Chunks {
			err = fmt.Errorf("AI content publication did not account for every chunk")
		}
		if err == nil && step.stage == "verified" && out.ObjectsVerified != summary.Chunks {
			err = fmt.Errorf("AI content readback did not verify every chunk")
		}
		if err != nil {
			r.results = append(r.results, StageResult{Stage: id, Status: StatusFailed, Reason: err.Error()})
			return nil, fmt.Errorf("proffer: %s: %w", step.name, err)
		}
		r.results = append(r.results, StageResult{Stage: id, Status: StatusSuccess, Ref: out.BundleRef, ReceiptRef: out.BundleRef})
		switch step.stage {
		case "work_products":
			req.WorkProductsRef = out.BundleRef
			summary.WorkProductsRef = out.BundleRef
			summary.WorkProductsSHA256 = out.BundleSHA256
			summary.WorkProducts = out.WorkProducts
		case "prepared":
			req.PreparedRef = out.BundleRef
			if req.NativeSourceOnly {
				req.SourceSHA256, req.VersionID = out.SourceSHA256, out.VersionID
				summary.SourceObjectID, summary.SourceSHA256, summary.VersionID = out.SourceObjectID, out.SourceSHA256, out.VersionID
			}
			summary.PreparedRef = out.BundleRef
			summary.PreparedSHA256 = out.BundleSHA256
			summary.Conversations = out.Conversations
			summary.Chunks = out.Chunks
		case "candidates":
			req.CandidatesRef = out.BundleRef
			summary.CandidatesRef = out.BundleRef
			summary.CandidatesSHA256 = out.BundleSHA256
			summary.Candidates = out.Candidates
			if req.NativeSourceOnly {
				staged, stageErr := r.stageNativeAICandidates(ctx, req, out)
				if stageErr != nil {
					return nil, stageErr
				}
				summary.CandidateStageRunID = staged.RunID
				summary.StagedOccurrences = staged.StagedOccurrences
				summary.HeldUnclassified = staged.HeldUnclassified
				summary.HeldManifest = staged.HeldManifest
				summary.HeldDuplicate = staged.HeldDuplicate
			}
		case "embedded":
			req.EmbeddingsRef = out.BundleRef
			summary.EmbeddingsRef = out.BundleRef
			summary.EmbeddingsSHA256 = out.BundleSHA256
		case "published":
			req.PublicationRef = out.BundleRef
			summary.PublicationRef = out.BundleRef
			summary.PublicationSHA256 = out.BundleSHA256
		case "verified":
			summary.VerificationRef = out.BundleRef
			summary.VerificationSHA256 = out.BundleSHA256
		}
	}
	return summary, nil
}

// AICandidateStageSource is the exact retained-object pin required by review staging.
// Inputs: source version/object identity, full hash and optional provider version. Outputs: a JSON pin.
// Effects: none. Choose to carry verified prepare pins without importing the Activity package.
type AICandidateStageSource struct {
	SourceVersionID string  `json:"source_version_id"`
	SourceObjectID  string  `json:"source_object_id"`
	SourceSHA256    string  `json:"source_sha256"`
	VersionID       *string `json:"version_id"`
}

// AICandidateStageRequest passes a persisted candidate bundle by reference to the native staging Activity.
// Inputs: original admission request, exact source pin and persisted bundle hash. Outputs: a staging receipt.
// Effects: none in workflow history. Choose after native candidate extraction succeeds.
type AICandidateStageRequest struct {
	PreviewHandle string                 `json:"preview_handle,omitempty"`
	RequestID     string                 `json:"request_id"`
	Source        AICandidateStageSource `json:"source"`
	BundleRef     Ref                    `json:"bundle_ref"`
	BundleSHA256  string                 `json:"bundle_sha256"`
}

// AICandidateStageResult is a bounded receipt for classified and held proposals.
// Inputs: the Activity's exact bundle and source. Outputs: staging counts and run ID.
// Effects: none in workflow history. Choose to prove staging without sending candidate bodies to Temporal.
type AICandidateStageResult struct {
	RunID             string `json:"run_id"`
	RequestDigest     string `json:"request_digest"`
	BundleRef         Ref    `json:"bundle_ref"`
	BundleSHA256      string `json:"bundle_sha256"`
	WorkProductsRef   Ref    `json:"work_products_ref"`
	Candidates        int    `json:"candidates"`
	StagedOccurrences int    `json:"staged_occurrences"`
	HeldUnclassified  int    `json:"held_unclassified"`
	HeldManifest      int    `json:"held_manifest"`
	HeldDuplicate     int    `json:"held_duplicate"`
}

// stageNativeAICandidates schedules source-bound staging without putting candidate text into workflow history.
// Inputs: verified prepare pin and persisted candidates bundle. Outputs: bounded staging receipt.
// Effects: pending working rows and held manifests in the Activity. Choose only on native AI route.
func (r *run) stageNativeAICandidates(ctx workflow.Context, req AIContentRequest, candidates AIContentResult) (AICandidateStageResult, error) {
	id := stagegraph.StageAICandidateBundle
	r.markStageStarted(id)
	input := AICandidateStageRequest{RequestID: req.RequestID,
		Source:    AICandidateStageSource{SourceVersionID: req.SourceVersionID, SourceObjectID: req.SourceObjectID, SourceSHA256: req.SourceSHA256, VersionID: req.VersionID},
		BundleRef: candidates.BundleRef, BundleSHA256: candidates.BundleSHA256}
	var out AICandidateStageResult
	err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, chunkActivityOptions(2*time.Hour)), string(id), input).Get(ctx, &out)
	r.markStageSettled(id)
	if err == nil && (out.BundleRef != input.BundleRef || out.BundleSHA256 != input.BundleSHA256 ||
		out.WorkProductsRef != req.WorkProductsRef || out.RunID == "" || !aiBundleSHA256Pattern.MatchString(out.RequestDigest) ||
		out.Candidates != candidates.Candidates || out.StagedOccurrences < 0 || out.HeldUnclassified < 0 || out.HeldManifest < 0 || out.HeldDuplicate < 0) {
		err = fmt.Errorf("AI candidate staging receipt disagrees with persisted source bundle")
	}
	if err != nil {
		r.results = append(r.results, StageResult{Stage: id, Status: StatusFailed, Reason: err.Error()})
		return AICandidateStageResult{}, fmt.Errorf("proffer: %s: %w", id, err)
	}
	r.results = append(r.results, StageResult{Stage: id, Status: StatusSuccess, Ref: Ref(out.RunID), ReceiptRef: Ref(out.RunID)})
	return out, nil
}

// aiExtractionActivityOptions lets provider cooldowns resume within a bounded extraction lifecycle.
// Inputs: none. Outputs: extraction-only options with a 24-hour total deadline and capped backoff.
// Effects: scheduler retries do not increase the Python ledger's two requests per chunk or source budget.
// Choose only for the version-gated AI extraction stage; other stages retain their existing retry policy.
// Byline: Codex · GPT-6 · 2026-10-07.
func aiExtractionActivityOptions() workflow.ActivityOptions {
	options := chunkActivityOptions(2 * time.Hour)
	options.ScheduleToCloseTimeout = 24 * time.Hour
	options.RetryPolicy = retryPolicy(60*time.Second, 0)
	options.RetryPolicy.MaximumInterval = 15 * time.Minute
	return options
}

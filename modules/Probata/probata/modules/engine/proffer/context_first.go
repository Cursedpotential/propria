package proffer

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	ContextContractVersion          = "context-v1"
	ContextStatusQueryName          = "context_status"
	ContextWorkflowIDPrefix         = "context-source:"
	contextFirstChangeID            = "proffer-context-first-source-v1"
	contextRegisterActivityName     = "context_register_source_activity"
	contextPrepareActivityName      = "ai_context_prepare_activity"
	contextWorkProductsActivityName = "ai_context_extract_work_products_activity"
	contextCandidatesActivityName   = "ai_context_extract_candidates_activity"
	contextEmbedActivityName        = "ai_context_embed_activity"
	contextPublishActivityName      = "ai_context_publish_activity"
	contextVerifyActivityName       = "ai_context_verify_publication_activity"
)

// ContextSummary reports a registered context source and compact derivative references.
// Inputs: registration and bounded Activity receipts. Outputs: reference-only status.
// Effects: none; choose for context-first imports, not legacy custody results.
type ContextSummary struct {
	ReviewSource     *ContextVerifiedReviewSource `json:"review_source,omitempty"`
	CandidateStage   *ContextCandidateStage       `json:"candidate_stage,omitempty"`
	Review           *ContextReviewResult         `json:"candidate_review,omitempty"`
	Package          *ContextPackageResult        `json:"package,omitempty"`
	ActorSubjectUID  string                       `json:"actor_subject_uid"`
	Status           string                       `json:"status"`
	Reason           string                       `json:"reason,omitempty"`
	SourceVersionRef string                       `json:"source_version_ref,omitempty"`
	ReceiptRef       string                       `json:"receipt_ref,omitempty"`
	DecodedRef       string                       `json:"decoded_ref,omitempty"`
	PreparedRef      string                       `json:"prepared_ref,omitempty"`
	WorkProductsRef  string                       `json:"work_products_ref,omitempty"`
	CandidatesRef    string                       `json:"candidates_ref,omitempty"`
	EmbeddingsRef    string                       `json:"embeddings_ref,omitempty"`
	PublicationRef   string                       `json:"publication_ref,omitempty"`
	VerificationRef  string                       `json:"verification_ref,omitempty"`
	Conversations    int                          `json:"conversations,omitempty"`
	Records          int                          `json:"records,omitempty"`
	Chunks           int                          `json:"chunks,omitempty"`
	WorkProducts     int                          `json:"work_products,omitempty"`
	Candidates       int                          `json:"candidates,omitempty"`
	ObjectsWritten   int                          `json:"objects_written,omitempty"`
	ObjectsVerified  int                          `json:"objects_verified,omitempty"`
}

// ContextProgress exposes the actor-bound source state while its Proffer run is open.
// Inputs: current context summary. Outputs: compact Temporal query value.
// Effects: none; choose for polling, not source-content retrieval.
type ContextProgress struct{ ContextSummary }

// ContextStarted identifies the existing Temporal execution for one context import.
// Inputs: a successful starter call. Outputs: actual workflow and run IDs.
// Effects: none; choose for the context HTTP start response.
type ContextStarted struct {
	WorkflowID string `json:"workflow_id"`
	RunID      string `json:"run_id"`
}

// ContextSourceRegistrationRequest identifies one source before parsing or custody work.
// Inputs: actual request and workflow IDs, source pointer, native version/package/kind, format and optional case scope.
// Outputs: a compact request for durable registration. Effects: none. Choose for context imports, not retained originals.
type ContextSourceRegistrationRequest struct {
	RequestID         string `json:"request_id"`
	WorkflowID        string `json:"workflow_id"`
	SourceRef         string `json:"source_ref"`
	ProviderVersionID string `json:"provider_version_id"`
	PackageRef        string `json:"package_ref"`
	SourceKind        string `json:"source_kind"`
	DeclaredFormat    string `json:"declared_format"`
	MatterID          string `json:"matter_id"`
	CourtCaseID       string `json:"court_case_id"`
}

// ContextSourceRegistrationResult returns only the discoverable identity and its receipt.
// Inputs: the persisted registration. Outputs: source-version and receipt references with registration status.
// Effects: none. Choose as the Activity result; source content and native turns stay out of Temporal history.
type ContextSourceRegistrationResult struct {
	SourceVersionRef string `json:"source_version_ref"`
	ReceiptRef       string `json:"receipt_ref"`
	Status           string `json:"status"`
}

// ContextResourceBounds carries the existing Python operation resource limits.
// Inputs: optional positive per-request ceilings. Outputs: unchanged JSON fields.
// Effects: none; choose for explicit small runs, never source slicing or policy admission.
type ContextResourceBounds struct {
	MaxSourceBytes int `json:"max_source_bytes,omitempty"`
	MaxRecords     int `json:"max_records,omitempty"`
	MaxTextBytes   int `json:"max_text_bytes,omitempty"`
	MaxChunks      int `json:"max_chunks,omitempty"`
	MaxModelCalls  int `json:"max_model_calls,omitempty"`
}

// Validate rejects invalid resource bounds without processing or truncating source content.
// Inputs: requested Python-compatible ceilings. Output: error or nil. Effects: none.
// Choose at HTTP and workflow boundaries before registration or provider calls.
func (b ContextResourceBounds) Validate() error {
	for _, bound := range []struct{ value, ceiling int }{
		{b.MaxSourceBytes, 32 << 20}, {b.MaxRecords, 1024}, {b.MaxTextBytes, 2 << 20},
		{b.MaxChunks, 256}, {b.MaxModelCalls, 512},
	} {
		if bound.value < 0 || bound.value > bound.ceiling {
			return errors.New("context resource bound is outside the existing operation ceiling")
		}
	}
	return nil
}

// defaults resolves omitted bounds while keeping model calls small by default.
// Inputs: validated request limits. Output: complete Python limits. Effects: none.
// Choose before scheduling; exceeding any bound is a partial result, not silent truncation.
func (b ContextResourceBounds) defaults() ContextResourceBounds {
	if b.MaxSourceBytes == 0 {
		b.MaxSourceBytes = 32 << 20
	}
	if b.MaxRecords == 0 {
		b.MaxRecords = 1024
	}
	if b.MaxTextBytes == 0 {
		b.MaxTextBytes = 2 << 20
	}
	if b.MaxChunks == 0 {
		b.MaxChunks = 256
	}
	if b.MaxModelCalls == 0 {
		b.MaxModelCalls = 8
	}
	return b
}

type aiContextRequest struct {
	ContractVersion    string `json:"contract_version"`
	SourceRef          string `json:"source_ref"`
	ProviderVersionID  string `json:"provider_version_id,omitempty"`
	PackageRef         string `json:"package_ref,omitempty"`
	SourceFormat       string `json:"source_format,omitempty"`
	RequestID          string `json:"request_id"`
	TemporalRunID      string `json:"temporal_run_id"`
	TemporalActivityID string `json:"temporal_activity_id"`
	DecodedRef         string `json:"decoded_ref,omitempty"`
	PreparedRef        string `json:"prepared_ref,omitempty"`
	WorkProductsRef    string `json:"work_products_ref,omitempty"`
	CandidatesRef      string `json:"candidates_ref,omitempty"`
	EmbeddingsRef      string `json:"embeddings_ref,omitempty"`
	PublicationRef     string `json:"publication_ref,omitempty"`
	ContextResourceBounds
}

type aiContextReceipt struct {
	ContractVersion string `json:"contract_version"`
	Stage           string `json:"stage"`
	BundleRef       string `json:"bundle_ref"`
	Status          string `json:"status"`
	Conversations   int    `json:"conversations"`
	Records         int    `json:"records"`
	Chunks          int    `json:"chunks"`
	WorkProducts    int    `json:"work_products"`
	Candidates      int    `json:"candidates"`
	ObjectsWritten  int    `json:"objects_written"`
	ObjectsVerified int    `json:"objects_verified"`
}

// validate checks the actual stage receipt against this run's resource ceilings.
// Inputs: stage and resolved bounds. Output: error or nil. Effects: none.
// Choose after every Python Activity rather than trusting count-only completion.
func (r aiContextReceipt) validate(stage string, bounds ContextResourceBounds) error {
	if r.ContractVersion != "ai-context-v1" || r.Stage != stage || r.BundleRef == "" || len(r.BundleRef) > 4096 ||
		r.Records < 0 || r.Records > bounds.MaxRecords || r.Chunks < 0 || r.Chunks > bounds.MaxChunks ||
		r.WorkProducts < 0 || r.Candidates < 0 || r.ObjectsWritten < 0 || r.ObjectsVerified < 0 {
		return fmt.Errorf("context %s returned an invalid bounded receipt", stage)
	}
	return nil
}

// contextFirstWorkflow registers a source before trying its parser or topic chunker.
// Inputs: an explicit context-v1 Proffer input with source and optional native version/case metadata.
// Outputs: durable registration plus compact partial or complete derivative references.
// Effects: one registration write, then six sequential existing Python Activities on their queue;
// choose before legacy RetainOriginal, source fingerprints, approval and case admission.
func contextFirstWorkflow(ctx workflow.Context, in WorkflowInput) (WorkflowResult, error) {
	if in.ContextContract != ContextContractVersion || !strings.HasPrefix(in.RequestID, ContextWorkflowIDPrefix) ||
		strings.TrimSpace(in.ActorSubjectUID) == "" || strings.TrimSpace(string(in.SourceRef)) == "" ||
		len(in.SourceRef) > 2048 || strings.TrimSpace(in.SourceKind) == "" || len(in.SourceKind) > 128 ||
		len(in.DeclaredFormat) > 128 || len(in.ProviderVersionID) > 512 || len(in.PackageRef) > 2048 {
		return WorkflowResult{}, errors.New("context-first import requires bounded actor and source coordinates")
	}
	if err := in.ContextResourceBounds.Validate(); err != nil {
		return WorkflowResult{}, err
	}
	limits := in.ContextResourceBounds.defaults()
	summary := ContextSummary{ActorSubjectUID: in.ActorSubjectUID, Status: "registering"}
	if err := workflow.SetQueryHandler(ctx, ContextStatusQueryName, func() (ContextProgress, error) {
		return ContextProgress{summary}, nil
	}); err != nil {
		return WorkflowResult{}, fmt.Errorf("register context status query: %w", err)
	}
	finish := func(status, reason string) (WorkflowResult, error) {
		// A partial enrichment/publication still retains all complete native outputs available.
		if summary.PreparedRef != "" && summary.WorkProductsRef != "" {
			info := workflow.GetInfo(ctx)
			request := ContextPackageRequest{RequestID: in.RequestID, WorkflowID: info.WorkflowExecution.ID, RunID: info.WorkflowExecution.RunID, SourceRef: string(in.SourceRef), ProviderVersionID: in.ProviderVersionID, PackageRef: string(in.PackageRef), SourceFormat: in.DeclaredFormat, SourceVersionRef: summary.SourceVersionRef, RegistrationReceiptRef: summary.ReceiptRef, PreparedRef: summary.PreparedRef, WorkProductsRef: summary.WorkProductsRef, CandidatesRef: summary.CandidatesRef, EmbeddingsRef: summary.EmbeddingsRef, PublicationRef: summary.PublicationRef, VerificationRef: summary.VerificationRef}
			options := workflow.ActivityOptions{StartToCloseTimeout: 20 * time.Minute, ScheduleToCloseTimeout: 30 * time.Minute, HeartbeatTimeout: time.Minute, WaitForCancellation: true, RetryPolicy: &temporal.RetryPolicy{MaximumAttempts: 1}}
			var packageResult ContextPackageResult
			err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, options), "retain_ai_context_package_activity", request).Get(ctx, &packageResult)
			if err != nil || !packageResult.Complete || packageResult.ManifestRef == "" || packageResult.ManifestSHA256 == "" {
				if reason == "" {
					reason = "package_retention_failed"
				} else {
					reason += ";package_retention_failed"
				}
				status = "partial"
			} else {
				summary.Package = &packageResult
				catalogRequest := ContextCatalogRequest{Request: request, Package: packageResult}
				catalogOptions := workflow.ActivityOptions{StartToCloseTimeout: 20 * time.Minute, ScheduleToCloseTimeout: 30 * time.Minute, RetryPolicy: &temporal.RetryPolicy{MaximumAttempts: 1}}
				var catalogResult ContextPackageResult
				catalogErr := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, catalogOptions), "catalog_ai_context_package_activity", catalogRequest).Get(ctx, &catalogResult)
				if catalogErr == nil && catalogResult.CatalogStatus == "complete" && catalogResult.ManifestRef == packageResult.ManifestRef && catalogResult.ManifestSHA256 == packageResult.ManifestSHA256 && catalogResult.CatalogReceiptRef != "" && catalogResult.CatalogReceiptSHA256 != "" {
					packageResult = catalogResult
					summary.Package = &packageResult
				}
				if packageResult.CatalogStatus != "complete" {
					if reason == "" {
						reason = "catalog_pending"
					} else {
						reason += ";catalog_pending"
					}
					status = "partial"
				}
			}
		}
		summary.Status, summary.Reason = status, reason
		resultStatus := StatusSuccess
		if status != "complete" {
			resultStatus = StatusNotApplicable
		}
		return WorkflowResult{SourceVersionRef: Ref(summary.SourceVersionRef), Status: resultStatus, Context: &summary}, nil
	}
	info := workflow.GetInfo(ctx)
	registration := ContextSourceRegistrationRequest{
		RequestID: in.RequestID, WorkflowID: info.WorkflowExecution.ID,
		SourceRef: string(in.SourceRef), ProviderVersionID: in.ProviderVersionID,
		PackageRef: string(in.PackageRef), SourceKind: in.SourceKind,
		DeclaredFormat: in.DeclaredFormat, MatterID: in.MatterID, CourtCaseID: in.CourtCaseID,
	}
	var registered ContextSourceRegistrationResult
	registerOptions := workflow.ActivityOptions{StartToCloseTimeout: 2 * time.Minute, ScheduleToCloseTimeout: 15 * time.Minute}
	if err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, registerOptions), contextRegisterActivityName, registration).Get(ctx, &registered); err != nil {
		summary.Status, summary.Reason = "failed", "registration_failed"
		return WorkflowResult{Context: &summary}, fmt.Errorf("context registration: %w", err)
	}
	if registered.Status != "registered" || registered.SourceVersionRef == "" || registered.ReceiptRef == "" {
		return finish("partial", "registration_receipt_invalid")
	}
	summary.SourceVersionRef, summary.ReceiptRef = registered.SourceVersionRef, registered.ReceiptRef
	if in.DeclaredFormat != "chatgpt" && in.DeclaredFormat != "claude" && in.DeclaredFormat != "gemini_markdown" {
		return finish("needs_parser", "parser_not_available")
	}
	request := aiContextRequest{
		ContractVersion: "ai-context-v1", SourceRef: string(in.SourceRef),
		ProviderVersionID: in.ProviderVersionID, PackageRef: string(in.PackageRef),
		SourceFormat: in.DeclaredFormat, RequestID: in.RequestID,
		TemporalRunID:         info.WorkflowExecution.RunID,
		ContextResourceBounds: limits,
	}
	step := func(name, stage string, input aiContextRequest) (aiContextReceipt, error) {
		input.TemporalActivityID = name
		options := chunkActivityOptions(2 * time.Hour)
		options.ActivityID = name
		var receipt aiContextReceipt
		if err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, options), name, input).Get(ctx, &receipt); err != nil {
			return aiContextReceipt{}, err
		}
		return receipt, receipt.validate(stage, limits)
	}
	summary.Status = "preparing"
	prepared, err := step(contextPrepareActivityName, "prepared", request)
	if err != nil {
		return finish("partial", "preparation_failed")
	}
	request.PreparedRef, summary.PreparedRef = prepared.BundleRef, prepared.BundleRef
	summary.Records, summary.Conversations, summary.Chunks = prepared.Records, prepared.Conversations, prepared.Chunks
	summary.Status = "extracting"
	works, err := step(contextWorkProductsActivityName, "work_products", request)
	if err != nil {
		return finish("partial", "work_product_failed")
	}
	request.WorkProductsRef, summary.WorkProductsRef = works.BundleRef, works.BundleRef
	summary.WorkProducts = works.WorkProducts
	summary.Status = "enriching"
	candidates, candidateErr := step(contextCandidatesActivityName, "candidates", request)
	var reviewErr error
	if candidateErr == nil {
		request.CandidatesRef, summary.CandidatesRef = candidates.BundleRef, candidates.BundleRef
		summary.Candidates = candidates.Candidates
		var providerVersion *string
		if in.ProviderVersionID != "" {
			value := in.ProviderVersionID
			providerVersion = &value
		}
		reviewRequest := ContextReviewRequest{ContractVersion: "ai-context-v1", OperatingMode: "LIVE", MatterID: in.MatterID, CourtCaseID: in.CourtCaseID, RequestID: in.RequestID, Source: ContextReviewSource{SourceVersionID: registered.SourceVersionRef, SourceRef: string(in.SourceRef), VersionID: providerVersion}, PreparedRef: request.PreparedRef, BundleRef: request.CandidatesRef}
		reviewOptions := workflow.ActivityOptions{StartToCloseTimeout: 2 * time.Minute, ScheduleToCloseTimeout: 10 * time.Minute, RetryPolicy: &temporal.RetryPolicy{MaximumAttempts: 3}}
		var reviewResult ContextReviewResult
		reviewErr = workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, reviewOptions), "stage_ai_candidate_bundle_activity", reviewRequest).Get(ctx, &reviewResult)
		if reviewErr == nil {
			summary.Review = &reviewResult
			summary.ReviewSource = reviewResult.Source
			summary.CandidateStage = &ContextCandidateStage{Status: "staged", Candidates: reviewResult.Candidates, Staged: reviewResult.Staged}
			if reviewResult.Source == nil {
				summary.CandidateStage.Status = "verified_source_unavailable"
				reviewErr = errors.New("native staging omitted independently verified source")
			}
		} else {
			summary.CandidateStage = &ContextCandidateStage{Status: "failed"}
		}
	}
	embedded, embedErr := step(contextEmbedActivityName, "embedded", request)
	if embedErr == nil {
		request.EmbeddingsRef, summary.EmbeddingsRef = embedded.BundleRef, embedded.BundleRef
	}
	if candidateErr != nil || embedErr != nil {
		return finish("partial", "candidate_or_embedding_failed")
	}
	summary.Status = "publishing"
	published, err := step(contextPublishActivityName, "published", request)
	if err != nil {
		return finish("partial", "publication_failed")
	}
	request.PublicationRef, summary.PublicationRef = published.BundleRef, published.BundleRef
	summary.ObjectsWritten = published.ObjectsWritten
	summary.Status = "verifying"
	verified, err := step(contextVerifyActivityName, "verified", request)
	if err != nil {
		return finish("partial", "publication_readback_failed")
	}
	summary.VerificationRef, summary.ObjectsVerified = verified.BundleRef, verified.ObjectsVerified
	if summary.ObjectsWritten != summary.Chunks || summary.ObjectsVerified != summary.Chunks {
		return finish("partial", "publication_count_mismatch")
	}
	if reviewErr != nil {
		return finish("partial", "candidate_review_failed")
	}
	if published.Status == "partial_enrichment" || verified.Status == "partial_enrichment" {
		return finish("partial_enrichment", "optional_enrichment_unavailable")
	}
	return finish("complete", "")
}

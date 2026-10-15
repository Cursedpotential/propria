package proffer

import (
	"errors"
	"strings"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// contextAtomicAIWorkflow preserves decoded records and search publication before optional candidate enrichment.
// Inputs: registered immutable source, explicit bounds and enclosing package finisher. Outputs: actual refs and separate stage outcomes.
// Effects: sequential named Activities only; choose through the version marker so recorded legacy histories keep their commands.
func contextAtomicAIWorkflow(ctx workflow.Context, in WorkflowInput, registered ContextSourceRegistrationResult, summary *ContextSummary, limits ContextResourceBounds, finish func(string, string) (WorkflowResult, error)) (WorkflowResult, error) {
	info := workflow.GetInfo(ctx)
	request := aiContextRequest{ContractVersion: "ai-context-v1", SourceRef: string(in.SourceRef), ProviderVersionID: in.ProviderVersionID, PackageRef: string(in.PackageRef), SourceFormat: in.DeclaredFormat, SourceVersionID: registered.SourceVersionRef, HistoricalSourceVersionID: registered.HistoricalSourceVersionID, SourceSHA256: registered.SourceSHA256, RequestID: in.RequestID, WorkflowID: info.WorkflowExecution.ID, TemporalRunID: info.WorkflowExecution.RunID, ContextResourceBounds: limits}
	step := func(name, stage string) (aiContextReceipt, error) {
		request.TemporalActivityID = name
		options := chunkActivityOptions(2 * time.Hour)
		options.ActivityID = name
		var receipt aiContextReceipt
		if err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, options), name, request).Get(ctx, &receipt); err != nil {
			return receipt, err
		}
		return receipt, receipt.validate(stage, limits)
	}
	summary.Status = "decoding"
	decoded, err := step(contextDecodeActivityName, "decoded")
	if err != nil {
		return finish("partial", "decode_failed")
	}
	request.DecodedRef, summary.DecodedRef = decoded.BundleRef, decoded.BundleRef
	summary.Records, summary.Conversations = decoded.Records, decoded.Conversations
	summary.Status = "extracting_work_products"
	works, err := step(contextWorkProductsActivityName, "work_products")
	if err != nil {
		return finish("partial", "work_product_failed")
	}
	request.WorkProductsRef, summary.WorkProductsRef = works.BundleRef, works.BundleRef
	summary.WorkProducts = works.WorkProducts
	summary.Status = "chunking"
	prepared, err := step(contextPrepareActivityName, "prepared")
	if err != nil {
		return finish("partial", "chunking_failed")
	}
	request.PreparedRef, summary.PreparedRef = prepared.BundleRef, prepared.BundleRef
	summary.Chunks = prepared.Chunks
	if prepared.Records != decoded.Records || prepared.Conversations != decoded.Conversations {
		return finish("partial", "decoded_chunk_coverage_mismatch")
	}
	failures := []string{}
	summary.Status = "embedding"
	embedded, embedErr := step(contextEmbedActivityName, "embedded")
	if embedErr != nil {
		failures = append(failures, "embedding_failed")
	} else {
		request.EmbeddingsRef, summary.EmbeddingsRef = embedded.BundleRef, embedded.BundleRef
		summary.EmbeddingModelID, summary.EmbeddingCalls = embedded.ModelID, embedded.ModelCalls
	}
	if embedErr == nil {
		summary.Status = "publishing"
		published, publishErr := step(contextPublishActivityName, "published")
		if publishErr != nil {
			failures = append(failures, "publication_failed")
		} else {
			request.PublicationRef, summary.PublicationRef = published.BundleRef, published.BundleRef
			summary.ObjectsWritten = published.ObjectsWritten
			summary.Status = "verifying"
			verified, verifyErr := step(contextVerifyActivityName, "verified")
			if verifyErr != nil {
				failures = append(failures, "publication_readback_failed")
			} else {
				summary.VerificationRef, summary.ObjectsVerified = verified.BundleRef, verified.ObjectsVerified
				if published.ObjectsWritten != summary.Chunks || verified.ObjectsVerified != summary.Chunks {
					failures = append(failures, "publication_count_mismatch")
				}
			}
		}
	}
	// Enrichment never supplies prerequisites to the already-published immutable search objects.
	summary.Status = "enriching"
	candidates, candidateErr := step(contextCandidatesActivityName, "candidates")
	if candidateErr != nil {
		summary.EnrichmentStatus = "failed"
		summary.CandidateStage = &ContextCandidateStage{Status: "not_run"}
		failures = append(failures, "candidate_extraction_failed")
	} else {
		request.CandidatesRef, summary.CandidatesRef = candidates.BundleRef, candidates.BundleRef
		summary.Candidates, summary.ModelID, summary.ModelCalls = candidates.Candidates, candidates.ModelID, candidates.ModelCalls
		summary.GroundedCandidates, summary.HeldCandidates, summary.RejectedCandidates = candidates.GroundedCandidates, candidates.HeldCandidates, candidates.RejectedCandidates
		summary.TypedCandidates = candidates.TypedCandidates
		summary.EnrichmentStatus = candidates.Status
		if candidates.Status != "complete" {
			failures = append(failures, "candidate_enrichment_incomplete")
		}
		var provider *string
		if in.ProviderVersionID != "" {
			v := in.ProviderVersionID
			provider = &v
		}
		review := ContextReviewRequest{ContractVersion: "ai-context-v1", OperatingMode: "LIVE", MatterID: in.MatterID, CourtCaseID: in.CourtCaseID, RequestID: in.RequestID, Source: ContextReviewSource{SourceVersionID: registered.SourceVersionRef, SourceRef: string(in.SourceRef), VersionID: provider}, PreparedRef: request.PreparedRef, BundleRef: request.CandidatesRef}
		options := workflow.ActivityOptions{StartToCloseTimeout: 2 * time.Minute, ScheduleToCloseTimeout: 10 * time.Minute, RetryPolicy: &temporal.RetryPolicy{MaximumAttempts: 3}}
		var staged ContextReviewResult
		stageErr := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, options), "stage_ai_candidate_bundle_activity", review).Get(ctx, &staged)
		if stageErr == nil && staged.Source == nil {
			stageErr = errors.New("staging omitted verified source")
		}
		if stageErr != nil {
			summary.CandidateStage = &ContextCandidateStage{Status: "failed"}
			failures = append(failures, "candidate_staging_failed")
		} else {
			summary.Review, summary.ReviewSource = &staged, staged.Source
			held := staged.HeldUnclassified + staged.HeldManifest + staged.HeldExternal
			status := "staged"
			if held > 0 {
				status = "partially_staged"
				failures = append(failures, "candidate_occurrences_held")
			}
			summary.CandidateStage = &ContextCandidateStage{Status: status, Candidates: staged.Candidates, Staged: staged.Staged, Held: held}
		}
	}
	if len(failures) > 0 {
		return finish("partial", strings.Join(failures, ";"))
	}
	if embedded.Status != "complete" {
		return finish("partial_enrichment", "embedding_unavailable")
	}
	return finish("complete", "")
}

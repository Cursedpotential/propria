package activities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/aicontextsource"
	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/extraction/flow"
	"github.com/Cursedpotential/probata/engine/extraction/service"
)

type aiContextCandidateBundle struct {
	ContractVersion string                   `json:"contract_version"`
	Stage           string                   `json:"stage"`
	Source          aicontextsource.Identity `json:"source"`
	PreparedRef     string                   `json:"prepared_ref"`
	Candidates      []struct {
		Kind        string     `json:"kind"`
		Name        string     `json:"name"`
		EntityType  string     `json:"entity_type"`
		EventType   string     `json:"event_type"`
		Predicate   string     `json:"predicate"`
		Statement   string     `json:"statement"`
		Quote       string     `json:"quote"`
		Status      string     `json:"status"`
		Confidence  *float64   `json:"confidence"`
		OccurredAt  *time.Time `json:"occurred_at"`
		Occurrences []struct {
			NativeJSONPointer   string     `json:"native_json_pointer"`
			Start               int        `json:"start"`
			End                 int        `json:"end"`
			Unit                string     `json:"unit"`
			SourceAvailableFrom *time.Time `json:"source_available_from"`
		} `json:"occurrences"`
	} `json:"candidates"`
}

// stageAIContextCandidateBundle verifies native context files and stages only complete typed assertions.
// Inputs: actual registered source UUID, provider/version, prepared and candidate bundle URIs.
// Outputs: derived original/candidate hashes, pending candidate IDs and counts.
// Effects: bounded local reads and pending PostgreSQL proposals; pick for ai-context-v1 only.
func (a AICandidateStagingActivities) stageAIContextCandidateBundle(ctx context.Context, in AICandidateStageInput) (AICandidateStageResult, error) {
	if a.Store == nil || in.RequestID == "" || in.PreviewHandle != "" || in.PreparedRef == "" || in.BundleRef == "" || in.Source.SourceVersionID == "" || in.Source.SourceRef == "" || in.Source.SourceObjectID != "" {
		return AICandidateStageResult{}, errors.New("native context staging requires registered source, request and exact prepared/candidate references")
	}
	if !caseidentity.AdmittedIdentity(in.MatterID, in.CourtCaseID) || in.OperatingMode != string(caseidentity.ModeLive) {
		return AICandidateStageResult{}, errors.New("native context staging requires canonical LIVE operating context")
	}
	if in.Source.PreparedRef != "" && in.Source.PreparedRef != in.PreparedRef {
		return AICandidateStageResult{}, errors.New("native context prepared reference differs from source pin")
	}
	prepared, original, err := aicontextsource.ReadPrepared(in.PreparedRef)
	if err != nil {
		return AICandidateStageResult{}, err
	}
	if prepared.Source.SourceRef != in.Source.SourceRef || !sameAIVersion(prepared.Source.ProviderVersionID, in.Source.VersionID) || (in.Source.SourceSHA256 != "" && in.Source.SourceSHA256 != prepared.OriginalSHA256) {
		return AICandidateStageResult{}, errors.New("native context prepared original differs from registered source request")
	}
	in.Source.SourceSHA256 = prepared.OriginalSHA256
	in.Source.PreparedRef = in.PreparedRef
	raw, candidatePath, err := aicontextsource.ReadFile(in.BundleRef, "candidates.json")
	if err != nil {
		return AICandidateStageResult{}, err
	}
	_, preparedPath, err := aicontextsource.ReadFile(in.PreparedRef, "prepared.json")
	if err != nil {
		return AICandidateStageResult{}, err
	}
	if a.DerivedRoot == "" {
		return AICandidateStageResult{}, errors.New("native context staging requires the configured derived root")
	}
	for _, path := range []string{candidatePath, preparedPath} {
		rel, err := filepath.Rel(filepath.Clean(a.DerivedRoot), path)
		if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return AICandidateStageResult{}, errors.New("native context bundle escaped configured derived root")
		}
	}
	if filepath.Dir(candidatePath) != filepath.Dir(preparedPath) {
		return AICandidateStageResult{}, errors.New("native context candidate bundle differs from prepared source directory")
	}
	if mode, err := a.Store.VerifyAISource(ctx, "", in.Source); err != nil {
		return AICandidateStageResult{}, err
	} else if mode != in.OperatingMode {
		return AICandidateStageResult{}, errors.New("native context operating mode differs from registration")
	}
	bundleHash := aicontextsource.Hash(raw)
	if in.BundleSHA256 != "" && in.BundleSHA256 != bundleHash {
		return AICandidateStageResult{}, errors.New("native context candidate bundle SHA256 differs")
	}
	var bundle aiContextCandidateBundle
	if err := json.Unmarshal(raw, &bundle); err != nil {
		return AICandidateStageResult{}, fmt.Errorf("native context candidates: %w", err)
	}
	if bundle.ContractVersion != "ai-context-v1" || bundle.Stage != "candidates" || bundle.PreparedRef != in.PreparedRef || bundle.Source.SourceRef != prepared.Source.SourceRef || !sameAIVersion(bundle.Source.ProviderVersionID, prepared.Source.ProviderVersionID) || !sameAIVersion(bundle.Source.PackageRef, prepared.Source.PackageRef) || bundle.Source.SourceFormat != prepared.Source.SourceFormat || bundle.Source.SourceSHA256 != prepared.Source.SourceSHA256 || len(bundle.Candidates) > 500 {
		return AICandidateStageResult{}, errors.New("native context candidate bundle does not match prepared source")
	}
	result := AICandidateStageResult{Source: &in.Source, BundleRef: in.BundleRef, BundleSHA256: bundleHash, Candidates: len(bundle.Candidates)}
	rows := make([]service.AICandidate, 0, len(bundle.Candidates))
	seen := map[[32]byte]bool{}
	for _, item := range bundle.Candidates {
		if item.Status != "unreviewed_context_candidate" || item.Confidence == nil || len(item.Occurrences) == 0 {
			result.HeldUnclassified++
			continue
		}
		domain := "ai_chat_content"
		if item.Kind == "strategy" || item.Kind == "history" {
			domain = "ai_chat_account"
		}
		if item.Kind != "entity" && item.Kind != "event" && item.Kind != "fact" && item.Kind != "strategy" && item.Kind != "history" {
			result.HeldUnclassified += len(item.Occurrences)
			continue
		}
		for _, occ := range item.Occurrences {
			if occ.Unit != "unicode_codepoint" {
				return AICandidateStageResult{}, errors.New("native context occurrence unit differs")
			}
			spanSHA, err := aicontextsource.VerifyQuote(prepared.Source.SourceFormat, original, occ.NativeJSONPointer, occ.Start, occ.End, item.Quote)
			if err != nil {
				return AICandidateStageResult{}, err
			}
			row := service.AICandidate{AISourcePin: in.Source, NativeJSONPointer: occ.NativeJSONPointer,
				SourceSpan: service.AISourceSpan{Start: occ.Start, End: occ.End, SHA256: spanSHA}, SpanUnit: occ.Unit,
				Kind: item.Kind, ReportedKind: item.Kind, ReviewDomain: domain, Name: item.Name, EntityType: item.EntityType, EventType: item.EventType,
				Predicate: item.Predicate, Statement: item.Statement, EvidenceQuote: item.Quote, OccurredAt: item.OccurredAt, SourceAvailableFrom: occ.SourceAvailableFrom, Confidence: *item.Confidence}
			if err := service.ValidateAICandidate(row); err != nil {
				result.HeldUnclassified++
				continue
			}
			digest := service.AICandidateDigest(row)
			if seen[digest] {
				result.HeldDuplicate++
				continue
			}
			seen[digest] = true
			rows = append(rows, row)
			if len(rows) > 500 {
				return AICandidateStageResult{}, errors.New("native context candidate occurrence count exceeds bound")
			}
		}
	}
	if len(rows) == 0 {
		return result, nil
	}
	// The real registered source and bounded candidate file hash make retries deterministic.
	requestHash := aicontextsource.Hash([]byte(strings.Join([]string{in.RequestID, in.Source.SourceVersionID, bundleHash}, "\n")))
	result.RequestDigest = requestHash
	result.RunID = flow.DeterministicID("ai_context_candidate_stage", in.Source.SourceVersionID, requestHash)
	ids, err := a.Store.StageAICandidates(ctx, "", result.RunID, result.RequestDigest, rows)
	if err != nil {
		return AICandidateStageResult{}, err
	}
	if len(ids) != len(rows) {
		return AICandidateStageResult{}, errors.New("native context review store returned incomplete staged identities")
	}
	result.Staged, result.CandidateIDs = len(ids), ids
	return result, nil
}

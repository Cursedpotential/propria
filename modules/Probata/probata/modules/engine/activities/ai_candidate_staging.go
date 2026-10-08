// Byline: Codex · GPT-6.1-sol · 2026-10-07.
package activities

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/extraction/flow"
	"github.com/Cursedpotential/probata/engine/extraction/service"
	"github.com/Cursedpotential/probata/engine/proffer"
)

const maxAICandidateBundleBytes = 32 << 20

// AICandidateStageInput pins one persisted producer bundle and its retained original.
// Inputs: canonical operating context, preview handle, exact source pin, file URI and persisted byte hash; outputs: Activity coordinates.
// Effects: none. Choose after AI extraction, before owner review or separate work-product placement.
type AICandidateStageInput struct {
	OperatingMode string              `json:"operating_mode"`
	MatterID      string              `json:"matter_id"`
	CourtCaseID   string              `json:"court_case_id"`
	RequestID     string              `json:"request_id"`
	PreviewHandle string              `json:"preview_handle"`
	Source        service.AISourcePin `json:"source"`
	BundleRef     string              `json:"bundle_ref"`
	BundleSHA256  string              `json:"bundle_sha256"`
}

// AICandidateStageResult reports pending rows and held source material without carrying conversation text.
// Inputs: staged review IDs and manifest reference; outputs: bounded counts, pins and deterministic run identity.
// Effects: none. Choose as the Temporal result for staging, never as an owner decision.
type AICandidateStageResult struct {
	RunID            string `json:"run_id,omitempty"`
	RequestDigest    string `json:"request_digest,omitempty"`
	BundleRef        string `json:"bundle_ref"`
	BundleSHA256     string `json:"bundle_sha256"`
	WorkProductsRef  string `json:"work_products_ref,omitempty"`
	Candidates       int    `json:"candidates"`
	Staged           int    `json:"staged_occurrences"`
	HeldUnclassified int    `json:"held_unclassified"`
	HeldManifest     int    `json:"held_manifest"`
	HeldDuplicate    int    `json:"held_duplicate"`
}

// AICandidateStagingActivities binds a server-owned derived root and pending-only review store.
// Inputs: existing derived root and AIReviewStore; outputs: one staging Activity.
// Effects: filesystem reads and pending proposal writes only. Choose instead of publishing or placing files.
type AICandidateStagingActivities struct {
	DerivedRoot string
	Store       service.AIReviewStore
}

type aiCandidateBundle struct {
	Version string `json:"version"`
	Stage   string `json:"stage"`
	Pins    struct {
		SourceVersionID string `json:"source_version_id"`
	} `json:"pins"`
	Source struct {
		SourceVersionID  string  `json:"source_version_id"`
		OriginalObjectID string  `json:"original_object_id"`
		OriginalSHA256   string  `json:"original_sha256"`
		VersionID        *string `json:"version_id"`
	} `json:"source"`
	WorkProductsRef string                `json:"work_products_ref"`
	Candidates      []aiProducerCandidate `json:"candidates"`
}

type aiProducerCandidate struct {
	Kind                 string                 `json:"kind"`
	ReportedKind         string                 `json:"reported_kind"`
	ReviewDomain         string                 `json:"review_domain"`
	BridgeDisposition    string                 `json:"bridge_disposition"`
	ClassificationStatus string                 `json:"classification_status"`
	Name                 string                 `json:"name"`
	EntityType           string                 `json:"entity_type"`
	EventType            string                 `json:"event_type"`
	Predicate            string                 `json:"predicate"`
	Statement            string                 `json:"statement"`
	OccurredAt           *time.Time             `json:"occurred_at"`
	Confidence           *float64               `json:"confidence"`
	Occurrences          []aiProducerOccurrence `json:"occurrences"`
}

type aiProducerOccurrence struct {
	SourceVersionID   string  `json:"source_version_id"`
	SourceObjectID    string  `json:"source_object_id"`
	VersionID         *string `json:"version_id"`
	SourceSHA256      string  `json:"source_sha256"`
	NativeJSONPointer string  `json:"native_json_pointer"`
	SpanUnit          string  `json:"span_unit"`
	SourceSpan        struct {
		Start  int    `json:"start"`
		End    int    `json:"end"`
		SHA256 string `json:"sha256"`
		Unit   string `json:"unit"`
	} `json:"source_span"`
	Quote               string     `json:"quote"`
	EvidenceQuote       string     `json:"evidence_quote"`
	SourceAvailableFrom *time.Time `json:"source_available_from,omitempty"`
}

// StageAICandidateBundle verifies persisted bytes and stages each eligible exact occurrence for review.
// Inputs: source and byte pins for one candidate bundle. Outputs: metadata, counts and references only.
// Effects: reads the bounded derived file, verifies custody via Store and writes pending candidates.
// Choose after the producer's candidates stage; work-product placement remains a separate Activity.
func (a AICandidateStagingActivities) StageAICandidateBundle(ctx context.Context, in AICandidateStageInput) (AICandidateStageResult, error) {
	result := AICandidateStageResult{BundleRef: in.BundleRef, BundleSHA256: in.BundleSHA256}
	if a.Store == nil || (in.RequestID == "" && in.PreviewHandle == "") || len(in.PreviewHandle) > 300 || !validSHA256(in.BundleSHA256) {
		return AICandidateStageResult{}, errors.New("AI staging requires a store, request or bounded preview, and bundle SHA256")
	}
	if in.RequestID != "" {
		resolved, err := a.Store.ResolveAIPreview(ctx, in.RequestID, in.Source)
		if err != nil {
			return AICandidateStageResult{}, err
		}
		if resolved == "" || len(resolved) > 300 || (in.PreviewHandle != "" && in.PreviewHandle != resolved) {
			return AICandidateStageResult{}, errors.New("AI staging preview differs from durable request binding")
		}
		in.PreviewHandle = resolved
	}
	mode, err := a.Store.VerifyAISource(ctx, in.PreviewHandle, in.Source)
	if err != nil {
		return AICandidateStageResult{}, err
	}
	if mode != "LIVE" {
		return AICandidateStageResult{}, errors.New("AI staging requires durable LIVE admission")
	}
	raw, err := aiReadCandidateBundle(a.DerivedRoot, in.BundleRef, in.Source.SourceVersionID, in.BundleSHA256)
	if err != nil {
		return AICandidateStageResult{}, err
	}
	var bundle aiCandidateBundle
	if err := json.Unmarshal(raw, &bundle); err != nil {
		return AICandidateStageResult{}, fmt.Errorf("decode AI candidate bundle: %w", err)
	}
	if bundle.Version != "ai-content-native-v3" || bundle.Stage != "candidates" || bundle.Pins.SourceVersionID != in.Source.SourceVersionID ||
		bundle.Source.SourceVersionID != in.Source.SourceVersionID || bundle.Source.OriginalObjectID != in.Source.SourceObjectID ||
		bundle.Source.OriginalSHA256 != in.Source.SourceSHA256 || !sameAIVersion(bundle.Source.VersionID, in.Source.VersionID) {
		return AICandidateStageResult{}, errors.New("AI candidate bundle source, version or stage differs from retained pin")
	}
	if len(bundle.Candidates) > 500 {
		return AICandidateStageResult{}, errors.New("AI candidate bundle exceeds review batch bound")
	}
	result.Candidates = len(bundle.Candidates)
	result.WorkProductsRef = bundle.WorkProductsRef
	rows := make([]service.AICandidate, 0, len(bundle.Candidates))
	seen := make(map[[32]byte]struct{})
	for _, item := range bundle.Candidates {
		if len(item.Occurrences) == 0 {
			return AICandidateStageResult{}, errors.New("AI candidate has no exact source occurrence")
		}
		if item.ReportedKind != item.Kind {
			return AICandidateStageResult{}, errors.New("AI reported kind changed")
		}
		for _, occurrence := range item.Occurrences {
			quoteHash := sha256.Sum256([]byte(occurrence.EvidenceQuote))
			if occurrence.SourceVersionID != in.Source.SourceVersionID || occurrence.SourceObjectID != in.Source.SourceObjectID ||
				occurrence.SourceSHA256 != in.Source.SourceSHA256 || !sameAIVersion(occurrence.VersionID, in.Source.VersionID) ||
				occurrence.SpanUnit != "unicode_codepoint" || occurrence.SourceSpan.Unit != "unicode_codepoint" || occurrence.Quote != occurrence.EvidenceQuote ||
				occurrence.SourceSpan.End <= occurrence.SourceSpan.Start ||
				hex.EncodeToString(quoteHash[:]) != occurrence.SourceSpan.SHA256 {
				return AICandidateStageResult{}, errors.New("AI occurrence has a mismatched source pin, span or quote")
			}
		}
		switch item.Kind {
		case "artifact", "document", "work_product":
			if bundle.WorkProductsRef == "" {
				return AICandidateStageResult{}, errors.New("created work requires a retained manifest reference")
			}
			result.HeldManifest += len(item.Occurrences)
			continue
		case "entity", "event", "fact", "strategy", "history":
		default:
			return AICandidateStageResult{}, fmt.Errorf("unsupported AI kind %q", item.Kind)
		}
		if item.BridgeDisposition != "pending_go_candidate" || item.ClassificationStatus != "typed" || item.Confidence == nil {
			result.HeldUnclassified += len(item.Occurrences)
			continue
		}
		for _, occurrence := range item.Occurrences {
			row := service.AICandidate{AISourcePin: in.Source, NativeJSONPointer: occurrence.NativeJSONPointer,
				SourceSpan: service.AISourceSpan{Start: occurrence.SourceSpan.Start, End: occurrence.SourceSpan.End, SHA256: occurrence.SourceSpan.SHA256},
				SpanUnit:   occurrence.SpanUnit, Kind: item.Kind, ReportedKind: item.ReportedKind, ReviewDomain: item.ReviewDomain,
				Name: item.Name, EntityType: item.EntityType, EventType: item.EventType, Predicate: item.Predicate,
				Statement: item.Statement, EvidenceQuote: occurrence.EvidenceQuote, OccurredAt: item.OccurredAt,
				SourceAvailableFrom: occurrence.SourceAvailableFrom, Confidence: *item.Confidence}
			if err := service.ValidateAICandidate(row); err != nil {
				return AICandidateStageResult{}, err
			}
			digest := service.AICandidateDigest(row)
			if _, duplicate := seen[digest]; duplicate {
				result.HeldDuplicate++
				continue
			}
			seen[digest] = struct{}{}
			rows = append(rows, row)
			if len(rows) > 500 {
				return AICandidateStageResult{}, errors.New("AI occurrence count exceeds review batch bound")
			}
		}
	}
	if len(rows) == 0 {
		return result, nil
	}
	requestHash := sha256.Sum256([]byte(in.PreviewHandle + "\n" + in.Source.SourceVersionID + "\n" + in.BundleSHA256))
	result.RequestDigest = hex.EncodeToString(requestHash[:])
	result.RunID = flow.DeterministicID("ai_content_candidate_stage", in.Source.SourceVersionID, result.RequestDigest)
	ids, err := a.Store.StageAICandidates(ctx, in.PreviewHandle, result.RunID, result.RequestDigest, rows)
	if err != nil {
		return AICandidateStageResult{}, err
	}
	if len(ids) != len(rows) {
		return AICandidateStageResult{}, errors.New("AI review store returned incomplete staged identities")
	}
	result.Staged = len(ids)
	return result, nil
}

// aiReadCandidateBundle reads a SHA-pinned file beneath the exact source directory.
// Inputs: managed root, file URI, source version and expected SHA256. Outputs: original bytes.
// Effects: bounded filesystem read. Choose for retained producer bundles, not source originals.
func aiReadCandidateBundle(root, ref, sourceVersionID, expectedHash string) ([]byte, error) {
	rootInfo, err := os.Lstat(root)
	if err != nil || rootInfo.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("AI derived root must be an existing nonlinked directory")
	}
	canonical, err := canonicalDirectory(root)
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(ref)
	if err != nil || u.Scheme != "file" || u.Host != "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" {
		return nil, errors.New("AI bundle requires a local file URI")
	}
	lexical := filepath.FromSlash(u.Path)
	if len(lexical) > 1 && lexical[0] == os.PathSeparator && filepath.VolumeName(lexical[1:]) != "" {
		lexical = lexical[1:]
	}
	if !filepath.IsAbs(lexical) {
		return nil, errors.New("AI bundle path must be absolute")
	}
	lexical = filepath.Clean(lexical)
	lexicalRel, err := filepath.Rel(canonical, lexical)
	if err != nil || lexicalRel == "." || lexicalRel == ".." || strings.HasPrefix(lexicalRel, ".."+string(os.PathSeparator)) {
		return nil, errors.New("AI bundle escaped derived root")
	}
	current := canonical
	for _, part := range strings.Split(filepath.ToSlash(lexicalRel), "/") {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("AI bundle path must not contain symlinks")
		}
	}
	path, err := resolveFileRef(proffer.Ref(ref), canonical, false)
	if err != nil {
		return nil, err
	}
	rel, err := filepath.Rel(canonical, path)
	if err != nil || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || rel == ".." {
		return nil, errors.New("AI bundle escaped derived root")
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) != 3 || parts[0] != sourceVersionID || parts[1] != "candidates" {
		return nil, errors.New("AI bundle must be in exact source candidates directory")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > maxAICandidateBundleBytes {
		return nil, errors.New("AI candidate bundle must be a bounded regular file")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, maxAICandidateBundleBytes+1))
	if err != nil {
		return nil, err
	}
	if len(raw) < 1 || len(raw) > maxAICandidateBundleBytes {
		return nil, errors.New("AI candidate bundle exceeds byte bound")
	}
	hash := sha256.Sum256(raw)
	if hex.EncodeToString(hash[:]) != expectedHash {
		return nil, errors.New("AI candidate bundle persisted byte SHA256 differs")
	}
	if !json.Valid(raw) || !bytes.Equal(bytes.TrimSpace(raw), raw) {
		return nil, errors.New("AI candidate bundle must contain one exact JSON object")
	}
	return raw, nil
}

func sameAIVersion(a, b *string) bool {
	return a == nil && b == nil || a != nil && b != nil && *a == *b
}

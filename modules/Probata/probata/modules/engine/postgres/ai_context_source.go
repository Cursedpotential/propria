package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Cursedpotential/probata/engine/aicontextsource"
	"github.com/Cursedpotential/probata/engine/extraction/service"
	"github.com/Cursedpotential/probata/engine/sourceformat"
	"github.com/jackc/pgx/v5"
)

// verifyAIContextSource binds a copied native original to the actual registered source and provider version.
// Inputs: platform DB and caller's source/version/hash/prepared pin. Outputs: declared native format.
// Effects: read-only PostgreSQL and bounded copied-original reads.
// Pick only for context.source_version without a retained object or preview binding.
func verifyAIContextSource(ctx context.Context, db DB, pin service.AISourcePin) (string, error) {
	if pin.SourceRef == "" || pin.SourceObjectID != "" || pin.PreparedRef == "" || !aicontextsource.ValidDigest(pin.SourceSHA256) {
		return "", service.ErrInvalid{Err: errors.New("native context needs actual registered source, prepared original and hash")}
	}
	prepared, _, err := aicontextsource.ReadPrepared(pin.PreparedRef)
	if err != nil {
		return "", service.ErrInvalid{Err: err}
	}
	if prepared.Source.SourceRef != pin.SourceRef || prepared.OriginalSHA256 != pin.SourceSHA256 || !sameVersionID(prepared.Source.ProviderVersionID, pin.VersionID) {
		return "", service.ErrInvalid{Err: errors.New("native context prepared source pin differs")}
	}
	var sourceRef, format, provider, metadataRef, packageRef, metadataFormat string
	var noObject bool
	var registrations int
	err = db.QueryRow(ctx, `SELECT source.source_key,version.declared_format,version.original_object_id IS NULL,
 coalesce(metadata.metadata->>'provider_version_id',''),coalesce(metadata.metadata->>'source_ref',''),
 coalesce(metadata.metadata->>'package_ref',''),coalesce(metadata.metadata->>'declared_format',''),count(*) OVER()
FROM context.source_version version
JOIN context.source source ON source.id=version.source_id
JOIN context.source_metadata metadata ON metadata.source_version_id=version.id
 AND metadata.metadata_class='record_native' AND metadata.extractor_id='context-source-registration'
WHERE version.id=$1::uuid AND version.matter_id=$2::uuid AND version.court_case_id=$3::uuid
ORDER BY metadata.generated_at DESC LIMIT 2`, pin.SourceVersionID, authoritativeMatterID, authoritativeCourtCaseID).
		Scan(&sourceRef, &format, &noObject, &provider, &metadataRef, &packageRef, &metadataFormat, &registrations)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", service.ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("native context registered source: %w", err)
	}
	expectedProvider := ""
	if pin.VersionID != nil {
		expectedProvider = *pin.VersionID
	}
	expectedPackage := ""
	if prepared.Source.PackageRef != nil {
		expectedPackage = *prepared.Source.PackageRef
	}
	if registrations != 1 || !noObject || sourceRef != pin.SourceRef || metadataRef != sourceRef || provider != expectedProvider || packageRef != expectedPackage || format != prepared.Source.SourceFormat || metadataFormat != format {
		return "", service.ErrInvalid{Err: errors.New("native context source/version/format differs from registration")}
	}
	switch format {
	case sourceformat.ChatGPTMarkdown, sourceformat.ClaudeMarkdown, sourceformat.GeminiMarkdown, "chatgpt", "claude", sourceformat.ClaudeAIExportJSON, "chatgpt_official_json", "chatgpt_json_array", "chatgpt_conversations_json", "claude_conversations_json":
	default:
		return "", service.ErrInvalid{Err: errors.New("registered native context format is unsupported")}
	}
	return format, nil
}

// verifyAIContextEvidence rechecks every staged native locator against the copied exact original.
// Inputs: registered context pin and bounded candidates. Outputs: validation error or source proof.
// Effects: read-only registered-source and local original reads; pick instead of retained-object opening.
func (s *EntityExtractionStore) verifyAIContextEvidence(ctx context.Context, pin service.AISourcePin, candidates []service.AICandidate) error {
	format, err := verifyAIContextSource(ctx, s.db, pin)
	if err != nil {
		return err
	}
	_, original, err := aicontextsource.ReadPrepared(pin.PreparedRef)
	if err != nil {
		return service.ErrInvalid{Err: err}
	}
	var document any
	switch format {
	case sourceformat.ChatGPTMarkdown, sourceformat.ClaudeMarkdown, sourceformat.GeminiMarkdown:
		document = string(original)
		for _, candidate := range candidates {
			if candidate.SourceAvailableFrom != nil {
				return service.ErrInvalid{Err: errors.New("native Markdown has no verified per-turn source clock")}
			}
		}
	case "chatgpt", "claude", sourceformat.ClaudeAIExportJSON, "chatgpt_official_json", "chatgpt_json_array", "chatgpt_conversations_json", "claude_conversations_json":
		if err := json.Unmarshal(original, &document); err != nil {
			return service.ErrInvalid{Err: fmt.Errorf("native context JSON is malformed: %w", err)}
		}
	default:
		return service.ErrInvalid{Err: errors.New("registered native context format is unsupported")}
	}
	if err := verifyAISpans(document, candidates); err != nil {
		return err
	}
	clockFormat := format
	if format == "chatgpt" {
		clockFormat = "chatgpt_official_json"
	}
	if format == "claude" {
		clockFormat = sourceformat.ClaudeAIExportJSON
	}
	for _, candidate := range candidates {
		if candidate.SourceAvailableFrom != nil {
			if err := verifyAINativeClock(document, clockFormat, candidate); err != nil {
				return service.ErrInvalid{Err: err}
			}
		}
	}
	return nil
}

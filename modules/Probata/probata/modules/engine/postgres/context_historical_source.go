package postgres

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/jackc/pgx/v5"
)

// verifyHistoricalContextSource checks an explicit predecessor before any native registration writes.
// Inputs: registration transaction and optional historical immutable coordinates. Outputs: exact match or refusal.
// Effects: SELECT only; choose for historical imports without recovering or normalizing chat generations.
func verifyHistoricalContextSource(ctx context.Context, tx pgx.Tx, req activities.ContextSourceRegistrationRequest) error {
	if req.HistoricalSourceVersionID == "" {
		return nil
	}
	var sourceRef, format, digest, provider string
	err := tx.QueryRow(ctx, `SELECT source.source_key,version.declared_format,
 COALESCE(encode(original.content_sha256,'hex'),metadata.metadata->>'source_sha256',''),
 COALESCE(metadata.metadata->>'provider_version_id','')
 FROM context.source_version version JOIN context.source source ON source.id=version.source_id
 LEFT JOIN context.retained_object original ON original.id=version.original_object_id
 LEFT JOIN LATERAL (SELECT candidate.metadata FROM context.source_metadata candidate
 WHERE candidate.source_version_id=version.id AND candidate.extractor_id='context-source-registration'
 ORDER BY candidate.generated_at DESC LIMIT 1) metadata ON true
 WHERE version.id=$1::uuid`, req.HistoricalSourceVersionID).Scan(&sourceRef, &format, &digest, &provider)
	if errors.Is(err, pgx.ErrNoRows) {
		return errors.New("historical source version does not exist")
	}
	if err != nil {
		return fmt.Errorf("verify historical source identity: %w", err)
	}
	return historicalContextCoordinates(req, sourceRef, provider, format, digest)
}

// historicalContextCoordinates compares the registered locator without inferring identity from a filename or text.
// Inputs: immutable registration fields and requested native coordinates. Outputs: equality or a specific refusal.
// Effects: none; choose after exact-version registration lookup, never latest-source selection.
func historicalContextCoordinates(req activities.ContextSourceRegistrationRequest, sourceRef, provider, format, digest string) error {
	parsed, err := url.Parse(sourceRef)
	if err != nil || parsed.Fragment != "" {
		return errors.New("historical registered source locator is invalid")
	}
	query := parsed.Query()
	if len(query) > 0 {
		values := query["versionId"]
		if len(query) != 1 || len(values) != 1 || values[0] == "" || (provider != "" && provider != values[0]) {
			return errors.New("historical provider version is ambiguous")
		}
		provider = values[0]
		parsed.RawQuery = ""
	}
	if parsed.String() != req.SourceRef || provider == "" || provider != req.ProviderVersionID || digest == "" || digest != req.SourceSHA256 {
		return errors.New("historical source locator, provider version or original digest differs")
	}
	supported := format == req.DeclaredFormat
	switch req.DeclaredFormat {
	case "chatgpt":
		supported = format == "chatgpt" || format == "chatgpt_official_json" || format == "chatgpt_json_array" || format == "chatgpt_conversations_json"
	case "claude":
		supported = format == "claude" || format == "claude_ai_export_json" || format == "claude_conversations_json"
	case "gemini_markdown":
		supported = format == "gemini_markdown"
	}
	if !supported {
		return errors.New("historical registered format differs from native provider format")
	}
	return nil
}

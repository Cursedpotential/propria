package postgres

import (
	"context"
	"fmt"
	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/contextsearch"
	"github.com/Cursedpotential/probata/engine/extraction/flow"
	"github.com/Cursedpotential/probata/engine/extraction/model"
	"strings"
)

// PreflightConversations resolves durable admission and counts exact generation text.
// Inputs: approved matter, conversation refs and registered extractor selection.
// Outputs: exact pins, workload and explicit blockers/unknown estimates. Effects:
// SELECT only; use before dispatch and repeat immediately before starting Temporal.
func (s *EntityExtractionStore) PreflightConversations(ctx context.Context, matter string, refs []flow.ConversationRef, extractorIDs []string) (flow.ExtractionPreflight, error) {
	out := flow.ExtractionPreflight{Eligible: true, Blockers: []string{}, Targets: []flow.ExtractionPreflightTarget{},
		CostEstimate: "Unknown: no provider pricing estimate is recorded", RuntimeEstimate: "Unknown: no calibrated runtime estimate is recorded",
		Bounds: []string{fmt.Sprintf("Default model: batches <=%d messages / %d excerpt characters; each message excerpt <=%d characters (full text remains retained)", model.MaxBatchMessages, model.MaxBatchChars, model.MaxBodyCharsShown),
			fmt.Sprintf("External extractors: <=%d messages per generation; %d per window", flow.MaxExternalMessages, flow.ExternalPageMessages), "External character/byte admission ceiling is not published; observed full-text sizes are shown"}}
	specs, err := flow.NormalizeExtractors(extractorIDs)
	if err != nil {
		return out, err
	}
	out.Extractors = specs
	if len(refs) == 0 || len(refs) > flow.MaxConversationsPerRequest {
		return out, fmt.Errorf("select 1 to %d conversations", flow.MaxConversationsPerRequest)
	}
	for _, ref := range refs {
		target := flow.ExtractionPreflightTarget{Ref: ref, WorkloadStatus: "not_queried", Runs: []flow.RunRef{}, AcquisitionRelationships: []string{}}
		formats, err := s.db.Query(ctx, `SELECT DISTINCT v.declared_format FROM context.source s
   JOIN context.source_version v ON v.source_id=s.id
   WHERE v.matter_id=$1::uuid AND regexp_replace(s.source_key,'\.derived/.*$','')=$2
   AND COALESCE(substring(s.source_key from '\.derived/threads/(.+?)(?:\.[0-9]{4})?\.ndjson$'),regexp_replace(s.source_key,'^.*/',''))=$3 LIMIT 20`, matter, ref.ExportKey, ref.Conv)
		if err != nil {
			return out, err
		}
		ai := false
		for formats.Next() {
			var format string
			if err := formats.Scan(&format); err != nil {
				formats.Close()
				return out, err
			}
			ai = ai || aiConversationDeclaredFormat(format)
		}
		formats.Close()
		if err := formats.Err(); err != nil {
			return out, err
		}
		if ai {
			out.Blockers = append(out.Blockers, ref.Label()+": AI chat content belongs to the Weaviate context/results lane; the human conversation extractor does not admit it and no PostgreSQL normalization is required")
			out.Targets = append(out.Targets, target)
			continue
		}
		files, err := s.conversationFiles(ctx, matter, ref)
		if err != nil {
			return out, err
		}
		if len(files) == 0 {
			out.Blockers = append(out.Blockers, ref.Label()+": no current normalized generation")
		}
		target.ResolvedGenerations = len(files)
		for _, file := range files {
			if file.PreviewHandle == "" {
				out.Blockers = append(out.Blockers, ref.Label()+": generation has no durable source operation")
				continue
			}
			run, err := s.ResolveRun(ctx, file.PreviewHandle)
			if err != nil {
				out.Blockers = append(out.Blockers, ref.Label()+": durable operation references unavailable")
				continue
			}
			target.Runs = append(target.Runs, run)
			if run.SourceVersionID != file.SourceVersionID || run.GenerationID != file.GenerationID {
				out.Blockers = append(out.Blockers, ref.Label()+": source/generation disagrees with durable operation")
				continue
			}
			if err := caseidentity.RequireCanonicalWrite(caseidentity.Mode(run.MatterMode)); err != nil {
				out.Blockers = append(out.Blockers, ref.Label()+": "+err.Error())
			}
			var messages, chars, bytes, maxChars, maxBytes int64
			err = s.db.QueryRow(ctx, `SELECT count(*), coalesce(sum(char_length(coalesce(n.normalized_payload->'content'->>'body',''))),0),
    coalesce(sum(octet_length(coalesce(n.normalized_payload->'content'->>'body',''))),0),
    coalesce(max(char_length(coalesce(n.normalized_payload->'content'->>'body',''))),0),
    coalesce(max(octet_length(coalesce(n.normalized_payload->'content'->>'body',''))),0)
    FROM context.normalized_record_identity n WHERE n.normalized_generation_id=$1::uuid AND n.record_type='message'`, file.GenerationID).Scan(&messages, &chars, &bytes, &maxChars, &maxBytes)
			if err != nil {
				return out, err
			}
			target.Messages += messages
			target.QueriedGenerations++
			target.Characters += chars
			target.UTF8Bytes += bytes
			if maxChars > target.MaxMessageCharacters {
				target.MaxMessageCharacters = maxChars
			}
			if maxBytes > target.MaxMessageUTF8Bytes {
				target.MaxMessageUTF8Bytes = maxBytes
			}
			for _, spec := range specs {
				if spec.Kind == flow.KindExternal && messages > flow.MaxExternalMessages {
					out.Blockers = append(out.Blockers, ref.Label()+": exceeds external extractor message bound")
				}
			}
			rows, err := s.db.Query(ctx, `SELECT DISTINCT coalesce(r.projection_kind,'unclassified') FROM context.normalized_record_identity n LEFT JOIN working.message_projection_route r ON r.normalized_record_id=n.id WHERE n.normalized_generation_id=$1::uuid AND n.record_type='message' ORDER BY 1 LIMIT 10`, file.GenerationID)
			if err != nil {
				return out, err
			}
			for rows.Next() {
				var relation string
				if err := rows.Scan(&relation); err != nil {
					rows.Close()
					return out, err
				}
				target.AcquisitionRelationships = append(target.AcquisitionRelationships, relation)
			}
			rows.Close()
			if err := rows.Err(); err != nil {
				return out, err
			}
		}
		if target.QueriedGenerations > 0 {
			target.WorkloadStatus = "partial"
			if target.QueriedGenerations == target.ResolvedGenerations {
				target.WorkloadStatus = "available"
			}
		}
		if target.WorkloadStatus == "available" && target.Messages == 0 {
			out.Blockers = append(out.Blockers, ref.Label()+": no message records available")
		}
		out.Targets = append(out.Targets, target)
	}
	out.Eligible = len(out.Blockers) == 0
	return out, nil
}

// aiConversationDeclaredFormat recognizes retained declared AI formats without guessing JSON.
// Input: source_version.declared_format. Output: route discriminator; no effects.
// Pick before generation lookup so AI content stays in its existing Weaviate lane.
func aiConversationDeclaredFormat(format string) bool {
	format = strings.ToLower(strings.TrimSpace(format))
	return contextsearch.IsAIChatFormat(format) || format == "chatgpt" || format == "claude" || format == "gemini_markdown"
}

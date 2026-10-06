// Byline: Claude Code · Sonnet 5.5 · 2026-10-02
//
// Conversation reads for the extraction request and the Surreal send. A
// conversation is the Workbench's unit (an export file plus a conversation key);
// the SQL below resolves it exactly as the Workbench's read-only imported views
// do (workbench/api/app/repo/imported_pg.py, the sv CTE), then reads the current
// normalized generation of each of its source versions. Reads only.

// Byline: Codex · GPT-5 · 2026-10-05 (durable operation receipt resolution).
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Cursedpotential/probata/engine/extraction/flow"
	"github.com/Cursedpotential/probata/engine/extraction/service"
)

var _ service.ConversationStore = (*EntityExtractionStore)(nil)

// conversationFilesSQL lists the source versions of one conversation with each one's
// current (highest-ordinal) normalized generation and the preview it was shown in.
const conversationFilesSQL = `
WITH sv AS (
  SELECT v.id, v.version_ordinal, s.source_key,
         regexp_replace(s.source_key, '\.derived/.*$', '') AS export_key,
         COALESCE(substring(s.source_key from '\.derived/threads/(.+?)(?:\.[0-9]{4})?\.ndjson$'),
                  regexp_replace(s.source_key, '^.*/', '')) AS conv
  FROM context.source_version v
  JOIN context.source s ON s.id = v.source_id
  WHERE v.matter_id = $1::uuid
)
SELECT sv.id::text, sv.source_key, g.id::text, coalesce(p.preview_handle, '')
FROM sv
JOIN LATERAL (
  SELECT id FROM context.normalized_generation g
  WHERE g.source_version_id = sv.id ORDER BY g.generation_ordinal DESC LIMIT 1
) g ON true
LEFT JOIN LATERAL (
  SELECT preview_handle FROM context.proffer_preview_snapshot sn
  WHERE sn.normalized_generation_id = g.id ORDER BY sn.snapshot_seq DESC LIMIT 1
) p ON true
WHERE sv.export_key = $2 AND sv.conv = $3
ORDER BY sv.version_ordinal, sv.id`

type conversationFile struct {
	service.SourceFile
	PreviewHandle string
}

func (s *EntityExtractionStore) conversationFiles(ctx context.Context, matterID string, ref flow.ConversationRef) ([]conversationFile, error) {
	if _, err := uuid.Parse(matterID); err != nil {
		return nil, fmt.Errorf("matter id is not a uuid: %w", service.ErrNotFound)
	}
	rows, err := s.db.Query(ctx, conversationFilesSQL, matterID, ref.ExportKey, ref.Conv)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []conversationFile
	for rows.Next() {
		var file conversationFile
		if err := rows.Scan(&file.SourceVersionID, &file.SourceKey, &file.GenerationID, &file.PreviewHandle); err != nil {
			return nil, err
		}
		out = append(out, file)
	}
	return out, rows.Err()
}

// ResolveConversation returns one RunRef per source version of the conversation.
func (s *EntityExtractionStore) ResolveConversation(ctx context.Context, matterID string, ref flow.ConversationRef) ([]flow.RunRef, error) {
	files, err := s.conversationFiles(ctx, matterID, ref)
	if err != nil {
		return nil, err
	}
	var runs []flow.RunRef
	for _, file := range files {
		if file.PreviewHandle == "" {
			// A generation nobody previewed has no handle to attribute proposals to.
			continue
		}
		run, readErr := s.ResolveRun(ctx, file.PreviewHandle)
		if readErr != nil {
			return nil, readErr
		}
		if run.GenerationID != file.GenerationID || run.SourceVersionID != file.SourceVersionID {
			return nil, errors.New("conversation source disagrees with its durable operation")
		}
		runs = append(runs, run)
	}
	return runs, nil
}

// ConversationInfo counts what the conversation holds.
func (s *EntityExtractionStore) ConversationInfo(ctx context.Context, matterID string, ref flow.ConversationRef) (service.ConversationInfo, error) {
	files, err := s.conversationFiles(ctx, matterID, ref)
	if err != nil {
		return service.ConversationInfo{}, err
	}
	info := service.ConversationInfo{}
	generations := make([]string, 0, len(files))
	for _, file := range files {
		info.SourceFiles = append(info.SourceFiles, file.SourceFile)
		generations = append(generations, file.GenerationID)
	}
	if len(generations) == 0 {
		return info, service.ErrNotFound
	}
	var first, last *time.Time
	if err := s.db.QueryRow(ctx, `SELECT count(*), min(occurred_at), max(occurred_at)
FROM context.normalized_record_identity
WHERE normalized_generation_id = ANY($1::uuid[]) AND record_type = 'message'`, generations).Scan(&info.Messages, &first, &last); err != nil {
		return info, err
	}
	info.FirstAt, info.LastAt = first, last
	rows, err := s.db.Query(ctx, `SELECT p.value->>'identifier', count(DISTINCT n.id)::int
FROM context.normalized_record_identity n
CROSS JOIN LATERAL jsonb_array_elements(n.normalized_payload->'participants') AS p(value)
WHERE n.normalized_generation_id = ANY($1::uuid[]) AND n.record_type = 'message' AND p.value->>'identifier' IS NOT NULL
GROUP BY 1 ORDER BY 2 DESC, 1 LIMIT 50`, generations)
	if err != nil {
		return info, err
	}
	defer rows.Close()
	for rows.Next() {
		var participant service.ParticipantCount
		if err := rows.Scan(&participant.Identifier, &participant.Messages); err != nil {
			return info, err
		}
		info.Participants = append(info.Participants, participant)
	}
	return info, rows.Err()
}

// ConversationMessages reads one keyset page, oldest first. A message without a time
// sorts first, so the cursor never meets a NULL comparison.
func (s *EntityExtractionStore) ConversationMessages(ctx context.Context, generationIDs []string, after service.MessageCursor, limit int) ([]service.ConversationMessage, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	hasCursor := after.ID != ""
	cursorID := after.ID
	if !hasCursor {
		cursorID = uuid.Nil.String()
	}
	rows, err := s.db.Query(ctx, `SELECT n.id::text, n.occurred_at,
       coalesce(n.normalized_payload->'content'->>'body', ''),
       coalesce(n.normalized_payload->'participants', '[]'::jsonb),
       coalesce(n.normalized_payload->>'timestamp_certainty', ''),
       coalesce(r.projection_kind, ''),
       coalesce(m.attachment_count, 0),
       n.source_version_id::text, n.record_ordinal
FROM context.normalized_record_identity n
LEFT JOIN working.message_projection_route r ON r.normalized_record_id = n.id
LEFT JOIN working.message m ON m.id = n.id
WHERE n.normalized_generation_id = ANY($1::uuid[]) AND n.record_type = 'message'
  AND (NOT $2::bool OR (coalesce(n.occurred_at, '-infinity'::timestamptz), n.id) > (coalesce($3::timestamptz, '-infinity'::timestamptz), $4::uuid))
ORDER BY coalesce(n.occurred_at, '-infinity'::timestamptz), n.id
LIMIT $5`, generationIDs, hasCursor, after.At, cursorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []service.ConversationMessage
	for rows.Next() {
		var message service.ConversationMessage
		var participants []byte
		if err := rows.Scan(&message.ID, &message.OccurredAt, &message.Body, &participants, &message.Certainty,
			&message.ProjectionKind, &message.AttachmentCount, &message.SourceVersionID, &message.Ordinal); err != nil {
			return nil, err
		}
		message.Participants = json.RawMessage(participants)
		if message.OccurredAt != nil {
			utc := message.OccurredAt.UTC()
			message.OccurredAt = &utc
		}
		out = append(out, message)
	}
	return out, rows.Err()
}

// ConversationExtractions reads every run, entity and event the generations hold.
func (s *EntityExtractionStore) ConversationExtractions(ctx context.Context, generationIDs []string) (service.ExtractionExport, error) {
	export := service.ExtractionExport{}
	if len(generationIDs) == 0 {
		return export, nil
	}
	runRows, err := s.db.Query(ctx, `SELECT r.id::text, g.gen, r.extractor, r.extractor_version, coalesce(r.model_id, ''), r.status,
       coalesce((r.stats->>'compare_only') = 'true', false), r.stats, r.started_at, r.finished_at
FROM working.extraction_run r
JOIN unnest($1::text[]) AS g(gen) ON r.source_summary LIKE '%generation:' || g.gen || '%'
ORDER BY r.started_at, r.id`, generationIDs)
	if err != nil {
		return export, err
	}
	for runRows.Next() {
		var run service.ExportedRun
		var stats []byte
		if err := runRows.Scan(&run.ID, &run.GenerationID, &run.Extractor, &run.Version, &run.ModelID, &run.Status, &run.CompareOnly, &stats, &run.StartedAt, &run.FinishedAt); err != nil {
			runRows.Close()
			return export, err
		}
		run.Stats = json.RawMessage(stats)
		export.Runs = append(export.Runs, run)
	}
	runRows.Close()
	if err := runRows.Err(); err != nil {
		return export, err
	}
	entityRows, err := s.db.Query(ctx, `SELECT id::text, extraction_run_id::text, source_raw_id, name, entity_type, coalesce(confidence, 0), review_state, attrs
FROM working.candidate_entity
WHERE source_raw_table = $1 AND source_raw_id = ANY($2::text[]) AND review_state <> 'superseded'
ORDER BY created_at, id`, generationScopeTable, generationIDs)
	if err != nil {
		return export, err
	}
	for entityRows.Next() {
		var entity service.ExportedEntity
		var attrs []byte
		if err := entityRows.Scan(&entity.ID, &entity.RunID, &entity.GenerationID, &entity.Name, &entity.EntityType, &entity.Confidence, &entity.ReviewState, &attrs); err != nil {
			entityRows.Close()
			return export, err
		}
		entity.Attrs = json.RawMessage(attrs)
		export.Entities = append(export.Entities, entity)
	}
	entityRows.Close()
	if err := entityRows.Err(); err != nil {
		return export, err
	}
	eventRows, err := s.db.Query(ctx, `SELECT id::text, extraction_run_id::text, source_raw_id, summary, event_type, occurred_at, coalesce(confidence, 0), review_state, attrs
FROM working.candidate_event
WHERE source_raw_table = $1 AND source_raw_id = ANY($2::text[]) AND review_state <> 'superseded'
ORDER BY occurred_at, id`, generationScopeTable, generationIDs)
	if err != nil {
		return export, err
	}
	defer eventRows.Close()
	for eventRows.Next() {
		var event service.ExportedEvent
		var attrs []byte
		if err := eventRows.Scan(&event.ID, &event.RunID, &event.GenerationID, &event.Title, &event.EventType, &event.OccurredAt, &event.Confidence, &event.ReviewState, &attrs); err != nil {
			return export, err
		}
		event.Attrs = json.RawMessage(attrs)
		export.Events = append(export.Events, event)
	}
	return export, eventRows.Err()
}

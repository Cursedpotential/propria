// Byline: Claude Code · Opus 5.5 · 2026-09-25
//
// EntityExtractionStore implements extraction/service.Store over the platform
// database. Staging uses the existing working.extraction_run,
// working.candidate_entity and working.candidate_event tables; commits write
// registry.entity / entity_alias, working.entity_mention / entity_resolution,
// timeline.event_candidate (+ its typed source_available_from anchor in
// context.relative_time_anchor) and timeline.timeline_member. Every write
// carries a deterministic id and ON CONFLICT DO NOTHING, so a retried
// Activity never writes a row twice. Grants for the engine role:
// scripts/2026-09-25-entity-proposals.sql.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Cursedpotential/probata/engine/extraction/entities"
	"github.com/Cursedpotential/probata/engine/extraction/events"
	"github.com/Cursedpotential/probata/engine/extraction/flow"
	"github.com/Cursedpotential/probata/engine/extraction/service"
)

// EntityExtractionStore is the durable extraction seam.
type EntityExtractionStore struct{ db DB }

// NewEntityExtractionStore validates the database seam.
func NewEntityExtractionStore(db DB) (*EntityExtractionStore, error) {
	if db == nil {
		return nil, errors.New("postgres entity extraction store: database is required")
	}
	return &EntityExtractionStore{db: db}, nil
}

var _ service.Store = (*EntityExtractionStore)(nil)

const (
	generationScopeTable = "context.normalized_generation"
	maxCurrentProposals  = 5000
	entityDuckDBQuoteTag = "entity_elt"
	generationToken      = "{{GENERATION}}"
)

// participantAggregateSQL is the DuckDB ELT for the deterministic proposal
// step: it unnests every normalized message's participants, normalizes each
// identifier (E.164 phones, lower-case e-mail and handles, the source-relative
// "self"), tokenizes display names, and aggregates role counts, the time
// window and five supporting records per address. The Go mirror is
// entities.NormalizeAddress; a disagreement is flagged on the proposal.
const participantAggregateSQL = `
WITH msg AS (
  SELECT CAST(id AS VARCHAR) AS record_id, record_ordinal AS ordinal, occurred_at,
         CAST(normalized_payload AS JSON) AS payload
  FROM pgduckdb.context.normalized_record_identity
  WHERE normalized_generation_id = '{{GENERATION}}'::UUID AND record_type = 'message'
),
total AS (SELECT count(*) AS total_messages FROM msg),
part AS (
  SELECT record_id, ordinal, occurred_at,
         json_extract_string(payload, '$.source_available_from') AS source_available_from,
         unnest(CAST(json_extract(payload, '$.participants') AS JSON[])) AS p
  FROM msg
),
fields AS (
  SELECT record_id, ordinal, occurred_at, source_available_from,
         lower(coalesce(json_extract_string(p, '$.role'), '')) AS role,
         trim(coalesce(json_extract_string(p, '$.identifier'), '')) AS identifier,
         nullif(trim(coalesce(json_extract_string(p, '$.display_name'), '')), '') AS display_name
  FROM part
),
kinds AS (
  SELECT *,
    CASE
      WHEN lower(identifier) IN ('self', 'me') THEN 'self'
      WHEN regexp_full_match(lower(identifier), '[^@\s]+@[^@\s]+\.[^@\s]+') THEN 'email'
      WHEN regexp_full_match(identifier, '@\S+') THEN 'handle'
      WHEN regexp_full_match(identifier, '\+?[0-9 ().\-]+')
           AND length(regexp_replace(identifier, '[^0-9]', '', 'g')) BETWEEN 7 AND 15 THEN 'phone'
      ELSE 'other'
    END AS address_kind,
    regexp_replace(identifier, '[^0-9]', '', 'g') AS digits
  FROM fields WHERE identifier <> ''
),
addr AS (
  SELECT *,
    CASE address_kind
      WHEN 'self' THEN 'self'
      WHEN 'email' THEN lower(identifier)
      WHEN 'handle' THEN lower(identifier)
      WHEN 'phone' THEN CASE
        WHEN starts_with(identifier, '+') THEN '+' || digits
        WHEN length(digits) = 10 THEN '+1' || digits
        WHEN length(digits) = 11 AND starts_with(digits, '1') THEN '+' || digits
        ELSE digits END
      ELSE lower(regexp_replace(identifier, '\s+', ' ', 'g'))
    END AS normalized_address
  FROM kinds
),
per_record AS (
  SELECT address_kind, normalized_address, record_id, ordinal,
         min(occurred_at) AS occurred_at, min(source_available_from) AS source_available_from,
         min(CASE WHEN role IN ('sender', 'from') THEN 0 WHEN role IN ('recipient', 'to', 'cc', 'bcc') THEN 1 ELSE 2 END) AS role_rank
  FROM addr GROUP BY ALL
),
agg AS (
  SELECT address_kind, normalized_address,
         list(DISTINCT identifier) AS identifiers,
         list(DISTINCT display_name) FILTER (WHERE display_name IS NOT NULL) AS display_names,
         epoch_ms(min(occurred_at)) AS first_ms, epoch_ms(max(occurred_at)) AS last_ms
  FROM addr GROUP BY ALL
),
counts AS (
  SELECT address_kind, normalized_address,
         count(*) FILTER (WHERE role_rank = 0) AS sender_count,
         count(*) FILTER (WHERE role_rank = 1) AS recipient_count,
         count(*) AS message_count
  FROM per_record GROUP BY ALL
),
samples AS (
  SELECT address_kind, normalized_address,
         list({'record_id': record_id, 'ordinal': ordinal,
               'role': CASE role_rank WHEN 0 THEN 'sender' WHEN 1 THEN 'recipient' ELSE 'participant' END,
               'occurred_ms': epoch_ms(occurred_at), 'source_available_from': source_available_from} ORDER BY ordinal) AS samples
  FROM (SELECT *, row_number() OVER (PARTITION BY address_kind, normalized_address ORDER BY ordinal) AS rn FROM per_record)
  WHERE rn <= 5
  GROUP BY ALL
),
participant_rows AS (
  SELECT c.address_kind, c.normalized_address,
         CAST(to_json(list_sort(g.identifiers)) AS VARCHAR) AS identifiers,
         CAST(to_json(list_sort(coalesce(g.display_names, []))) AS VARCHAR) AS display_names,
         CAST(to_json(list_transform(list_sort(coalesce(g.display_names, [])),
           n -> list_filter(string_split(regexp_replace(lower(regexp_replace(n, '[''’.]', '', 'g')), '[^\pL\pN]+', ' ', 'g'), ' '), t -> t <> ''))) AS VARCHAR) AS name_tokens,
         c.sender_count, c.recipient_count, c.message_count, t.total_messages,
         g.first_ms, g.last_ms,
         CAST(to_json(s.samples) AS VARCHAR) AS samples
  FROM counts c
  JOIN agg g USING (address_kind, normalized_address)
  LEFT JOIN samples s USING (address_kind, normalized_address)
  CROSS JOIN total t
)
SELECT participant_rows.*,
       CAST(to_json({'address_kind': address_kind, 'normalized_address': normalized_address, 'identifiers': identifiers,
                     'display_names': display_names, 'name_tokens': name_tokens, 'sender_count': sender_count,
                     'recipient_count': recipient_count, 'message_count': message_count, 'total_messages': total_messages,
                     'first_ms': first_ms, 'last_ms': last_ms, 'samples': samples}) AS VARCHAR) AS json_line
FROM participant_rows
ORDER BY message_count DESC, normalized_address
`

func participantInnerSQL(generationID string) (string, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(generationID))
	if err != nil {
		return "", fmt.Errorf("generation id must be a UUID: %w", err)
	}
	inner := strings.ReplaceAll(participantAggregateSQL, generationToken, parsed.String())
	if strings.Contains(inner, "$"+entityDuckDBQuoteTag+"$") {
		return "", errors.New("participant query contains the reserved DuckDB quote tag")
	}
	return inner, nil
}

// ParticipantAggregateQuery renders the full PostgreSQL statement for one
// generation.
func ParticipantAggregateQuery(generationID string) (string, error) {
	inner, err := participantInnerSQL(generationID)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`SELECT (r['address_kind'])::text AS address_kind, (r['normalized_address'])::text AS normalized_address,
       (r['identifiers'])::text AS identifiers, (r['display_names'])::text AS display_names, (r['name_tokens'])::text AS name_tokens,
       (r['sender_count'])::bigint AS sender_count, (r['recipient_count'])::bigint AS recipient_count,
       (r['message_count'])::bigint AS message_count, (r['total_messages'])::bigint AS total_messages,
       (r['first_ms'])::bigint AS first_ms, (r['last_ms'])::bigint AS last_ms, (r['samples'])::text AS samples
FROM duckdb.query($%[1]s$%[2]s$%[1]s$) AS r`, entityDuckDBQuoteTag, inner), nil
}

// DryRunQueries renders read-only statements that print one JSON object per
// line: the participant ELT rows, then the run's messages. psql output of
// these feeds cmd/entity-dryrun, which runs the production proposal code
// without any database connection or write.
func DryRunQueries(generationID string, messageLimit int) (string, error) {
	inner, err := participantInnerSQL(generationID)
	if err != nil {
		return "", err
	}
	if messageLimit <= 0 {
		messageLimit = 100000
	}
	participants := fmt.Sprintf(`SELECT (r['json_line'])::text FROM duckdb.query($%[1]s$%[2]s$%[1]s$) AS r`, entityDuckDBQuoteTag, inner)
	return fmt.Sprintf(`SET default_transaction_read_only = on;
BEGIN READ ONLY;
\echo #participants
%s;
\echo #messages
SELECT json_build_object('record_id', id::text, 'source_version_id', source_version_id::text, 'ordinal', record_ordinal, 'occurred_at', occurred_at,
       'source_available_from', normalized_payload->>'source_available_from',
       'body', coalesce(normalized_payload->'content'->>'body', ''),
       'participants', coalesce(normalized_payload->'participants', '[]'::jsonb))::text
FROM context.normalized_record_identity
WHERE normalized_generation_id = '%s'::uuid AND record_type = 'message'
ORDER BY record_ordinal LIMIT %d;
ROLLBACK;
`, participants, uuid.MustParse(generationID).String(), messageLimit), nil
}

type participantSample struct {
	RecordID            string  `json:"record_id"`
	Ordinal             int64   `json:"ordinal"`
	Role                string  `json:"role"`
	OccurredMS          *int64  `json:"occurred_ms"`
	SourceAvailableFrom *string `json:"source_available_from"`
}

// ParticipantAggregates runs the DuckDB ELT for one generation.
func (s *EntityExtractionStore) ParticipantAggregates(ctx context.Context, generationID string) ([]entities.ParticipantAggregate, error) {
	query, err := ParticipantAggregateQuery(generationID)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("participant ELT: %w", err)
	}
	defer rows.Close()
	var out []entities.ParticipantAggregate
	for rows.Next() {
		var raw ParticipantRow
		if err := rows.Scan(&raw.AddressKind, &raw.NormalizedAddress, &raw.Identifiers, &raw.DisplayNames, &raw.NameTokens,
			&raw.SenderCount, &raw.RecipientCount, &raw.MessageCount, &raw.TotalMessages, &raw.FirstMS, &raw.LastMS, &raw.Samples); err != nil {
			return nil, err
		}
		row, err := raw.Aggregate()
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// ParticipantRow is one participant ELT row as the query returns it (JSON
// arrays as text). The store and the dry run decode it the same way.
type ParticipantRow struct {
	AddressKind       string  `json:"address_kind"`
	NormalizedAddress string  `json:"normalized_address"`
	Identifiers       string  `json:"identifiers"`
	DisplayNames      string  `json:"display_names"`
	NameTokens        string  `json:"name_tokens"`
	SenderCount       int64   `json:"sender_count"`
	RecipientCount    int64   `json:"recipient_count"`
	MessageCount      int64   `json:"message_count"`
	TotalMessages     int64   `json:"total_messages"`
	FirstMS           *int64  `json:"first_ms"`
	LastMS            *int64  `json:"last_ms"`
	Samples           *string `json:"samples"`
}

// Aggregate decodes the row into the extraction input.
func (raw ParticipantRow) Aggregate() (entities.ParticipantAggregate, error) {
	row := entities.ParticipantAggregate{
		AddressKind: raw.AddressKind, NormalizedAddress: raw.NormalizedAddress,
		SenderCount: int(raw.SenderCount), RecipientCount: int(raw.RecipientCount), MessageCount: int(raw.MessageCount),
		TotalMessages: int(raw.TotalMessages), FirstAt: millis(raw.FirstMS), LastAt: millis(raw.LastMS),
	}
	if err := json.Unmarshal([]byte(raw.Identifiers), &row.Identifiers); err != nil {
		return row, fmt.Errorf("participant identifiers: %w", err)
	}
	if len(row.Identifiers) > 0 {
		row.Identifier = row.Identifiers[0]
	}
	if err := json.Unmarshal([]byte(raw.DisplayNames), &row.DisplayNames); err != nil {
		return row, fmt.Errorf("participant display names: %w", err)
	}
	if err := json.Unmarshal([]byte(raw.NameTokens), &row.NameTokens); err != nil {
		return row, fmt.Errorf("participant name tokens: %w", err)
	}
	if raw.Samples != nil && *raw.Samples != "" && *raw.Samples != "null" {
		var samples []participantSample
		if err := json.Unmarshal([]byte(*raw.Samples), &samples); err != nil {
			return row, fmt.Errorf("participant samples: %w", err)
		}
		for _, sample := range samples {
			seen := entities.ParticipantSeen{RecordID: sample.RecordID, Ordinal: sample.Ordinal, Role: sample.Role, OccurredAt: millis(sample.OccurredMS)}
			if sample.SourceAvailableFrom != nil {
				seen.SourceAvailableFrom = parseTime(*sample.SourceAvailableFrom)
			}
			row.Samples = append(row.Samples, seen)
		}
	}
	return row, nil
}

func millis(value *int64) *time.Time {
	if value == nil {
		return nil
	}
	at := time.UnixMilli(*value).UTC()
	return &at
}

func parseTime(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05.999999999Z07", "2006-01-02 15:04:05Z07"} {
		if at, err := time.Parse(layout, value); err == nil {
			utc := at.UTC()
			return &utc
		}
	}
	return nil
}

const messageColumns = `id::text, record_ordinal, occurred_at, normalized_payload->>'source_available_from',
       coalesce(normalized_payload->'content'->>'body', ''), coalesce(normalized_payload->'participants', '[]'::jsonb)`

func scanMessage(row pgx.Row) (entities.MessageView, error) {
	var message entities.MessageView
	var available *string
	var participants []byte
	if err := row.Scan(&message.RecordID, &message.Ordinal, &message.OccurredAt, &available, &message.Body, &participants); err != nil {
		return entities.MessageView{}, err
	}
	if available != nil {
		message.SourceAvailableFrom = parseTime(*available)
	}
	if message.OccurredAt != nil {
		utc := message.OccurredAt.UTC()
		message.OccurredAt = &utc
	}
	if err := json.Unmarshal(participants, &message.Participants); err != nil {
		return entities.MessageView{}, fmt.Errorf("message %s participants: %w", message.RecordID, err)
	}
	return message, nil
}

// MessagePage streams messages in ordinal order (keyset pagination).
func (s *EntityExtractionStore) MessagePage(ctx context.Context, generationID string, afterOrdinal int64, limit int) ([]entities.MessageView, error) {
	if limit <= 0 || limit > 1000 {
		limit = 400
	}
	rows, err := s.db.Query(ctx, `SELECT `+messageColumns+`
FROM context.normalized_record_identity
WHERE normalized_generation_id = $1::uuid AND record_type = 'message' AND record_ordinal > $2
ORDER BY record_ordinal LIMIT $3`, generationID, afterOrdinal, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []entities.MessageView
	for rows.Next() {
		message, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, message)
	}
	return out, rows.Err()
}

// MessageByID reads one record of the generation (any record type).
func (s *EntityExtractionStore) MessageByID(ctx context.Context, generationID, recordID string) (entities.MessageView, error) {
	if _, err := uuid.Parse(recordID); err != nil {
		return entities.MessageView{}, service.ErrNotFound
	}
	message, err := scanMessage(s.db.QueryRow(ctx, `SELECT `+messageColumns+`
FROM context.normalized_record_identity WHERE id = $1::uuid AND normalized_generation_id = $2::uuid`, recordID, generationID))
	if errors.Is(err, pgx.ErrNoRows) {
		return entities.MessageView{}, service.ErrNotFound
	}
	return message, err
}

// RecordsMissing returns the ids that are not records of the generation.
func (s *EntityExtractionStore) RecordsMissing(ctx context.Context, generationID string, recordIDs []string) ([]string, error) {
	var valid, missing []string
	for _, id := range recordIDs {
		if _, err := uuid.Parse(id); err != nil {
			missing = append(missing, id)
			continue
		}
		valid = append(valid, id)
	}
	if len(valid) == 0 {
		return missing, nil
	}
	rows, err := s.db.Query(ctx, `SELECT r.id::text FROM unnest($1::uuid[]) AS r(id)
WHERE NOT EXISTS (SELECT 1 FROM context.normalized_record_identity n WHERE n.id = r.id AND n.normalized_generation_id = $2::uuid)`, valid, generationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		missing = append(missing, id)
	}
	return missing, rows.Err()
}

// ResolveRun maps a preview handle to its current normalized generation.
func (s *EntityExtractionStore) ResolveRun(ctx context.Context, previewHandle string) (flow.RunRef, error) {
	var run flow.RunRef
	err := s.db.QueryRow(ctx, `SELECT b.preview_handle, s.normalized_generation_id::text, s.source_version_id::text
FROM context.proffer_preview_binding b
JOIN LATERAL (
  SELECT normalized_generation_id, source_version_id FROM context.proffer_preview_snapshot sn
  WHERE sn.preview_handle = b.preview_handle ORDER BY snapshot_seq DESC LIMIT 1
) s ON true
WHERE b.preview_handle = $1`, previewHandle).Scan(&run.PreviewHandle, &run.GenerationID, &run.SourceVersionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return flow.RunRef{}, fmt.Errorf("run %s has no normalized generation yet: %w", previewHandle, service.ErrNotFound)
	}
	return run, err
}

// BeginRun inserts an extraction_run row once.
func (s *EntityExtractionStore) BeginRun(ctx context.Context, run service.RunRow) error {
	stats, err := json.Marshal(nonNilStats(run.Stats))
	if err != nil {
		return err
	}
	_, err = s.exec(ctx, `INSERT INTO working.extraction_run (id, extractor, extractor_version, model_id, source_summary, status, stats, prompt_version)
VALUES ($1::uuid, $2, $3, NULLIF($4, ''), $5, 'running', $6::jsonb, NULLIF($7, ''))
ON CONFLICT (id) DO NOTHING`, run.ID, run.Extractor, run.Version, run.ModelID, run.Summary, string(stats), run.PromptVersion)
	return err
}

// FinishRun closes a running extraction_run row.
func (s *EntityExtractionStore) FinishRun(ctx context.Context, runID, status string, stats map[string]any, errText string) error {
	raw, err := json.Marshal(nonNilStats(stats))
	if err != nil {
		return err
	}
	if status == "failed" && strings.TrimSpace(errText) == "" {
		errText = "failed"
	}
	_, err = s.exec(ctx, `UPDATE working.extraction_run SET status = $2, finished_at = now(), error = NULLIF($3, ''), stats = stats || $4::jsonb
WHERE id = $1::uuid AND status = 'running'`, runID, status, errText, string(raw))
	return err
}

func nonNilStats(stats map[string]any) map[string]any {
	if stats == nil {
		return map[string]any{}
	}
	return stats
}

// ExtractionRuns lists the runs of a generation for the Review panel.
func (s *EntityExtractionStore) ExtractionRuns(ctx context.Context, generationID string) ([]service.RunSummary, error) {
	rows, err := s.db.Query(ctx, `SELECT id::text, extractor, extractor_version, coalesce(model_id, ''), status, coalesce(error, ''),
       started_at, finished_at, stats
FROM working.extraction_run WHERE source_summary LIKE $1 ORDER BY started_at DESC, id LIMIT 50`, "%generation:"+generationID+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []service.RunSummary
	for rows.Next() {
		var run service.RunSummary
		var stats []byte
		if err := rows.Scan(&run.ID, &run.Extractor, &run.Version, &run.ModelID, &run.Status, &run.Error, &run.StartedAt, &run.FinishedAt, &stats); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(stats, &run.Stats); err != nil {
			return nil, err
		}
		out = append(out, run)
	}
	return out, rows.Err()
}

func (s *EntityExtractionStore) exec(ctx context.Context, sql string, args ...any) (int64, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, sql, args...)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// ---- entity staging --------------------------------------------------------

const insertCandidateEntitySQL = `INSERT INTO working.candidate_entity
  (id, extraction_run_id, source_raw_table, source_raw_id, entity_type, name, normalized_name, confidence, attrs,
   content_sha256, review_state, domain, ontology_version, knowledge_time)
VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9::jsonb, $10, $11, 'context', 'probata.entities/v1', now())
ON CONFLICT DO NOTHING`

func entityInsertArgs(id, runID string, proposal entities.Proposal) ([]any, error) {
	proposal.Normalize()
	state := proposal.ReviewState
	if state == "" {
		state = entities.StatePending
	}
	attrs := proposal
	attrs.CandidateID, attrs.ExtractionRunID, attrs.ReviewState, attrs.PromotedToID = "", "", "", ""
	raw, err := json.Marshal(attrs)
	if err != nil {
		return nil, err
	}
	normalized := entities.RegistryNormalizedName(proposal.Name)
	if normalized == "" {
		normalized = "unnamed"
	}
	name := proposal.Name
	if strings.TrimSpace(name) == "" {
		name = "unnamed"
	}
	digest := proposal.ContentSHA256()
	return []any{id, runID, generationScopeTable, proposal.GenerationID, string(proposal.Coarse()), name, normalized,
		proposal.Confidence, string(raw), digest[:], state}, nil
}

// StageEntities inserts proposals under a run; ids are deterministic from
// run and content, so a retried Activity finds its rows instead of adding.
func (s *EntityExtractionStore) StageEntities(ctx context.Context, runID string, proposals []entities.Proposal) ([]string, error) {
	ids := make([]string, len(proposals))
	for i, proposal := range proposals {
		ids[i] = flow.DeterministicID("candidate", runID, proposal.ContentHex())
	}
	if err := s.ReplaceEntities(ctx, runID, ids, proposals, nil); err != nil {
		return nil, err
	}
	return ids, nil
}

// ReplaceEntities inserts new proposal rows and marks the rows they replace
// superseded, in one transaction. A replayed request (all rows already
// present) is a no-op; a supersede that finds a row no longer current
// rolls back with ErrConflict.
func (s *EntityExtractionStore) ReplaceEntities(ctx context.Context, runID string, ids []string, insert []entities.Proposal, supersede []string) error {
	if len(ids) != len(insert) {
		return errors.New("replace entities: one id per inserted proposal is required")
	}
	return s.replace(ctx, "working.candidate_entity", ids, len(insert), func(tx pgx.Tx, i int) (int64, error) {
		args, err := entityInsertArgs(ids[i], runID, insert[i])
		if err != nil {
			return 0, err
		}
		tag, err := tx.Exec(ctx, insertCandidateEntitySQL, args...)
		return tag.RowsAffected(), err
	}, supersede)
}

func (s *EntityExtractionStore) replace(ctx context.Context, table string, ids []string, count int, insert func(pgx.Tx, int) (int64, error), supersede []string) error {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var inserted int64
	for i := 0; i < count; i++ {
		affected, err := insert(tx, i)
		if err != nil {
			return err
		}
		inserted += affected
	}
	if count > 0 && inserted == 0 {
		var present int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM `+table+` WHERE id = ANY($1::uuid[])`, ids).Scan(&present); err != nil {
			return err
		}
		if present == count {
			return tx.Commit(ctx) // replay: every row already written
		}
		if len(supersede) > 0 {
			return fmt.Errorf("an identical proposal already exists: %w", entities.ErrConflict)
		}
	}
	if len(supersede) > 0 {
		tag, err := tx.Exec(ctx, `UPDATE `+table+` SET review_state = 'superseded'
WHERE id = ANY($1::uuid[]) AND review_state IN ('pending', 'rejected')`, supersede)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != int64(len(uniqueStrings(supersede))) {
			return entities.ErrConflict
		}
	}
	return tx.Commit(ctx)
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}

const selectCandidateEntitySQL = `SELECT id::text, extraction_run_id::text, name, coalesce(confidence, 0), attrs, review_state, coalesce(promoted_to_id, '')
FROM working.candidate_entity`

func scanEntities(rows pgx.Rows) ([]entities.Proposal, error) {
	defer rows.Close()
	var out []entities.Proposal
	for rows.Next() {
		var id, runID, name, state, promoted string
		var confidence float64
		var attrs []byte
		if err := rows.Scan(&id, &runID, &name, &confidence, &attrs, &state, &promoted); err != nil {
			return nil, err
		}
		var proposal entities.Proposal
		if err := json.Unmarshal(attrs, &proposal); err != nil {
			return nil, fmt.Errorf("candidate_entity %s attrs: %w", id, err)
		}
		proposal.CandidateID, proposal.ExtractionRunID, proposal.ReviewState, proposal.PromotedToID = id, runID, state, promoted
		proposal.Name, proposal.Confidence = name, confidence
		out = append(out, proposal)
	}
	return out, rows.Err()
}

// CurrentEntities returns the generation's non-superseded proposals.
func (s *EntityExtractionStore) CurrentEntities(ctx context.Context, generationID string) ([]entities.Proposal, error) {
	rows, err := s.db.Query(ctx, selectCandidateEntitySQL+`
WHERE source_raw_table = $1 AND source_raw_id = $2 AND review_state IN ('pending', 'rejected', 'approved')
ORDER BY created_at, id LIMIT $3`, generationScopeTable, generationID, maxCurrentProposals)
	if err != nil {
		return nil, err
	}
	return scanEntities(rows)
}

// PendingEntitiesOfRuns returns the still-pending proposals of given runs.
func (s *EntityExtractionStore) PendingEntitiesOfRuns(ctx context.Context, runIDs []string) ([]entities.Proposal, error) {
	if len(runIDs) == 0 {
		return nil, nil
	}
	rows, err := s.db.Query(ctx, selectCandidateEntitySQL+`
WHERE extraction_run_id = ANY($1::uuid[]) AND review_state = 'pending' ORDER BY created_at, id LIMIT $2`, runIDs, maxCurrentProposals)
	if err != nil {
		return nil, err
	}
	return scanEntities(rows)
}

// ---- event staging ---------------------------------------------------------

const insertCandidateEventSQL = `INSERT INTO working.candidate_event
  (id, extraction_run_id, source_raw_table, source_raw_id, event_type, summary, occurred_at, temporal_confidence,
   confidence, attrs, content_sha256, review_state, domain, ontology_version, knowledge_time)
VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11, $12, 'context', 'probata.events/v1', now())
ON CONFLICT DO NOTHING`

func eventInsertArgs(id, runID string, event events.Proposal) ([]any, error) {
	event.Normalize()
	if event.OccurredAt == nil {
		return nil, fmt.Errorf("event %q has no time and cannot be staged", event.Title)
	}
	state := event.ReviewState
	if state == "" {
		state = entities.StatePending
	}
	attrs := event
	attrs.CandidateID, attrs.ExtractionRunID, attrs.ReviewState, attrs.PromotedToID = "", "", "", ""
	raw, err := json.Marshal(attrs)
	if err != nil {
		return nil, err
	}
	digest := event.ContentSHA256()
	return []any{id, runID, generationScopeTable, event.GenerationID, event.EventType, event.Title, event.OccurredAt,
		event.TemporalConfidence, event.Confidence, string(raw), digest[:], state}, nil
}

// StageEvents inserts event proposals under a run.
func (s *EntityExtractionStore) StageEvents(ctx context.Context, runID string, proposals []events.Proposal) ([]string, error) {
	ids := make([]string, len(proposals))
	for i, proposal := range proposals {
		ids[i] = flow.DeterministicID("candidate_event", runID, proposal.ContentHex())
	}
	if err := s.ReplaceEvents(ctx, runID, ids, proposals, nil); err != nil {
		return nil, err
	}
	return ids, nil
}

// ReplaceEvents is ReplaceEntities for working.candidate_event.
func (s *EntityExtractionStore) ReplaceEvents(ctx context.Context, runID string, ids []string, insert []events.Proposal, supersede []string) error {
	if len(ids) != len(insert) {
		return errors.New("replace events: one id per inserted proposal is required")
	}
	return s.replace(ctx, "working.candidate_event", ids, len(insert), func(tx pgx.Tx, i int) (int64, error) {
		args, err := eventInsertArgs(ids[i], runID, insert[i])
		if err != nil {
			return 0, err
		}
		tag, err := tx.Exec(ctx, insertCandidateEventSQL, args...)
		return tag.RowsAffected(), err
	}, supersede)
}

// CurrentEvents returns the generation's non-superseded event proposals.
func (s *EntityExtractionStore) CurrentEvents(ctx context.Context, generationID string) ([]events.Proposal, error) {
	rows, err := s.db.Query(ctx, `SELECT id::text, extraction_run_id::text, summary, attrs, review_state, coalesce(promoted_to_id, '')
FROM working.candidate_event
WHERE source_raw_table = $1 AND source_raw_id = $2 AND review_state IN ('pending', 'rejected', 'approved')
ORDER BY occurred_at, id LIMIT $3`, generationScopeTable, generationID, maxCurrentProposals)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []events.Proposal
	for rows.Next() {
		var id, runID, title, state, promoted string
		var attrs []byte
		if err := rows.Scan(&id, &runID, &title, &attrs, &state, &promoted); err != nil {
			return nil, err
		}
		var event events.Proposal
		if err := json.Unmarshal(attrs, &event); err != nil {
			return nil, fmt.Errorf("candidate_event %s attrs: %w", id, err)
		}
		event.CandidateID, event.ExtractionRunID, event.ReviewState, event.PromotedToID, event.Title = id, runID, state, promoted, title
		out = append(out, event)
	}
	return out, rows.Err()
}

// ---- registry reads --------------------------------------------------------
//
// Aliases are read from registry.entity_alias_current (the newest row of each
// alias chain the Workbench Case page writes) and retired ones are left out.
// Claude Code · Opus 5.5 · 2026-10-01

const registrySelectSQL = `SELECT e.id::text, coalesce(e.display_name::text, e.canonical_name::text, ''), e.entity_type::text,
       coalesce(e.normalized_name::text, ''),
       coalesce((SELECT json_agg(json_build_object('text', a.alias_text::text, 'kind', coalesce(a.alias_kind, 'other')) ORDER BY a.alias_text::text)
                 FROM registry.entity_alias_current a WHERE a.entity_id = e.id AND a.status <> 'retired'), '[]'::json)
FROM registry.entity e`

func scanRegistry(rows pgx.Rows) ([]entities.RegistryEntity, error) {
	defer rows.Close()
	var out []entities.RegistryEntity
	for rows.Next() {
		var entity entities.RegistryEntity
		var registryType string
		var aliases []byte
		if err := rows.Scan(&entity.ID, &entity.DisplayName, &registryType, &entity.NormalizedName, &aliases); err != nil {
			return nil, err
		}
		entity.RegistryType = entities.RegistryType(registryType)
		if err := json.Unmarshal(aliases, &entity.Aliases); err != nil {
			return nil, err
		}
		out = append(out, entity)
	}
	return out, rows.Err()
}

// Registry returns the live committed entities a lookup names.
func (s *EntityExtractionStore) Registry(ctx context.Context, lookup service.RegistryLookup) ([]entities.RegistryEntity, error) {
	var ids []string
	for _, id := range lookup.IDs {
		if _, err := uuid.Parse(id); err == nil {
			ids = append(ids, id)
		}
	}
	if len(ids)+len(lookup.NormalizedNames)+len(lookup.AliasTexts) == 0 {
		return nil, nil
	}
	rows, err := s.db.Query(ctx, registrySelectSQL+`
WHERE e.merged_into_id IS NULL
  AND (e.id = ANY($1::uuid[]) OR e.normalized_name::text = ANY($2::text[])
       OR EXISTS (SELECT 1 FROM registry.entity_alias_current a WHERE a.entity_id = e.id AND a.status <> 'retired' AND lower(a.alias_text::text) = ANY($3::text[])))
ORDER BY e.id LIMIT 5000`, ids, uniqueStrings(lookup.NormalizedNames), uniqueStrings(lookup.AliasTexts))
	if err != nil {
		return nil, err
	}
	return scanRegistry(rows)
}

// SearchRegistry finds live committed entities by name or alias.
func (s *EntityExtractionStore) SearchRegistry(ctx context.Context, query string, limit int) ([]entities.RegistryEntity, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	pattern := "%" + strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(strings.TrimSpace(query)) + "%"
	rows, err := s.db.Query(ctx, registrySelectSQL+`
WHERE e.merged_into_id IS NULL
  AND ($1 = '%%' OR coalesce(e.display_name::text, e.canonical_name::text, '') ILIKE $1
       OR EXISTS (SELECT 1 FROM registry.entity_alias_current a WHERE a.entity_id = e.id AND a.status <> 'retired' AND a.alias_text::text ILIKE $1))
ORDER BY coalesce(e.display_name::text, e.canonical_name::text, ''), e.id LIMIT $2`, pattern, limit)
	if err != nil {
		return nil, err
	}
	return scanRegistry(rows)
}

// ---- commit writes ---------------------------------------------------------

// WriteEntities inserts new registry.entity rows. Enum and domain columns
// take their values by assignment coercion from text (verified live with
// PREPARE); provenance is an ai.source_ref[] literal (sourceRefs) cast
// explicitly, which needs USAGE on schema ai (scripts/2026-09-25-entity-proposals.sql).
func (s *EntityExtractionStore) WriteEntities(ctx context.Context, rows []service.EntityRow) (int, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	var ids, types, names, normalized, tiers, provenance []string
	var confidence []float64
	for _, row := range rows {
		ids, types, names, normalized = append(ids, row.ID), append(types, row.RegistryType), append(names, row.Name), append(normalized, row.NormalizedName)
		tiers, confidence = append(tiers, row.DataTier), append(confidence, row.Confidence)
		provenance = append(provenance, sourceRefs(
			[3]string{"postgres", row.CandidateID, "working.candidate_entity/" + row.CandidateID},
			[3]string{"postgres", row.ReceiptID, "working.extraction_run/" + row.ReceiptID},
		))
	}
	affected, err := s.exec(ctx, insertRegistryEntitiesSQL, ids, types, names, normalized, tiers, confidence, provenance)
	return int(affected), err
}

const insertRegistryEntitiesSQL = `INSERT INTO registry.entity
  (id, entity_type, display_name, canonical_name, normalized_name, data_tier, evidence_confidence, provenance,
   requires_human_review, review_status, safe_for_legal_use)
SELECT r.id, r.entity_type, r.name, r.name, r.normalized, r.tier, r.confidence, r.provenance::ai.source_ref[], false, 'approved', false
FROM unnest($1::uuid[], $2::text[], $3::text[], $4::text[], $5::text[], $6::numeric[], $7::text[])
     AS r(id, entity_type, name, normalized, tier, confidence, provenance)
ON CONFLICT (id) DO NOTHING`

// literalEscaper backslash-escapes the two characters a quoted array element
// or composite field cannot hold bare.
var literalEscaper = strings.NewReplacer(`\`, `\\`, `"`, `\"`)

// sourceRefs renders an ai.source_ref[] array literal: {"(system,id,locator)",...}.
func sourceRefs(refs ...[3]string) string {
	parts := make([]string, 0, len(refs))
	for _, ref := range refs {
		composite := "(" + compositeField(ref[0]) + "," + compositeField(ref[1]) + "," + compositeField(ref[2]) + ")"
		parts = append(parts, `"`+literalEscaper.Replace(composite)+`"`)
	}
	return "{" + strings.Join(parts, ",") + "}"
}

// compositeField quotes a composite (row) field when it holds a character
// the row literal syntax gives meaning to.
func compositeField(value string) string {
	if value != "" && !strings.ContainsAny(value, "(),\" \t\r\n\\") {
		return value
	}
	return `"` + literalEscaper.Replace(value) + `"`
}

// WriteAliases inserts registry.entity_alias rows an entity does not have.
func (s *EntityExtractionStore) WriteAliases(ctx context.Context, rows []service.AliasRow) (int, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	var ids, entityIDs, texts, kinds, provenance []string
	var confidence []float64
	for _, row := range rows {
		ids, entityIDs, texts, kinds = append(ids, row.ID), append(entityIDs, row.EntityID), append(texts, row.Text), append(kinds, row.Kind)
		confidence = append(confidence, row.Confidence)
		provenance = append(provenance, sourceRefs([3]string{"postgres", row.CandidateID, "working.candidate_entity/" + row.CandidateID + "#alias"}))
	}
	affected, err := s.exec(ctx, insertRegistryAliasesSQL, ids, entityIDs, texts, kinds, confidence, provenance)
	return int(affected), err
}

const insertRegistryAliasesSQL = `INSERT INTO registry.entity_alias (id, entity_id, alias_text, alias_kind, confidence, provenance)
SELECT r.id, r.entity_id, r.alias_text, r.kind, r.confidence, r.provenance::ai.source_ref[]
FROM unnest($1::uuid[], $2::uuid[], $3::text[], $4::text[], $5::numeric[], $6::text[])
     AS r(id, entity_id, alias_text, kind, confidence, provenance)
WHERE NOT EXISTS (SELECT 1 FROM registry.entity_alias a WHERE a.entity_id = r.entity_id AND lower(a.alias_text::text) = lower(r.alias_text))
ON CONFLICT (id) DO NOTHING`

// WriteMentions inserts append-only mentions and their current resolutions.
// A mention that currently resolves elsewhere has that resolution closed
// (sys_period upper bound) before the new one is written.
func (s *EntityExtractionStore) WriteMentions(ctx context.Context, mentions []service.MentionRow, resolutions []service.ResolutionRow, reviewer service.Reviewer, receiptID string) (int, int, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var ids, surfaces, kinds, records, snippets, methods, provenance []string
	var starts, ends []*int32
	var confidence []float64
	for _, mention := range mentions {
		ids, surfaces, kinds, records = append(ids, mention.ID), append(surfaces, mention.Surface), append(kinds, mention.Kind), append(records, mention.RecordID)
		snippets, methods, confidence = append(snippets, mention.Snippet), append(methods, mention.Method), append(confidence, mention.Confidence)
		starts, ends = append(starts, int32Pointer(mention.Start)), append(ends, int32Pointer(mention.End))
		provenance = append(provenance, sourceRefs([3]string{"postgres", mention.RecordID, "context.normalized_record_identity/" + mention.RecordID}))
	}
	tag, err := tx.Exec(ctx, insertMentionsSQL, ids, surfaces, kinds, records, starts, ends, snippets, methods, confidence, provenance)
	if err != nil {
		return 0, 0, fmt.Errorf("write mentions: %w", err)
	}
	mentionCount := int(tag.RowsAffected())
	var resolutionIDs, mentionIDs, entityIDs, matchMethods, resolvedBy, metrics, resolutionProvenance []string
	var scores []float64
	for _, resolution := range resolutions {
		raw, err := json.Marshal(resolution.Metrics)
		if err != nil {
			return 0, 0, err
		}
		resolutionIDs, mentionIDs, entityIDs = append(resolutionIDs, resolution.ID), append(mentionIDs, resolution.MentionID), append(entityIDs, resolution.EntityID)
		matchMethods, resolvedBy, metrics = append(matchMethods, resolution.MatchMethod), append(resolvedBy, resolution.ResolvedBy), append(metrics, string(raw))
		scores = append(scores, resolution.Score)
		resolutionProvenance = append(resolutionProvenance, sourceRefs(
			[3]string{"postgres", resolution.CandidateID, "working.candidate_entity/" + resolution.CandidateID},
			[3]string{"postgres", receiptID, "working.extraction_run/" + receiptID},
		))
	}
	if _, err := tx.Exec(ctx, `UPDATE working.entity_resolution r SET sys_period = tstzrange(lower(r.sys_period), now())
FROM unnest($1::uuid[], $2::uuid[]) AS i(mention_id, entity_id)
WHERE r.mention_id = i.mention_id AND upper_inf(r.sys_period) AND r.canonical_entity_id <> i.entity_id`, mentionIDs, entityIDs); err != nil {
		return 0, 0, fmt.Errorf("close superseded resolutions: %w", err)
	}
	tag, err = tx.Exec(ctx, insertResolutionsSQL, resolutionIDs, mentionIDs, entityIDs, matchMethods, resolvedBy, scores, metrics,
		resolutionProvenance, reviewer.Username, reviewer.At, "committed by extraction commit "+receiptID)
	if err != nil {
		return 0, 0, fmt.Errorf("write resolutions: %w", err)
	}
	resolutionCount := int(tag.RowsAffected())
	if err := tx.Commit(ctx); err != nil {
		return 0, 0, err
	}
	return mentionCount, resolutionCount, nil
}

// VerificationStatements are the writes whose target columns use ai.* enum,
// domain and composite types. The dry-run tool PREPAREs them inside a
// READ ONLY transaction: parse analysis proves name lookup and type
// coercion for the engine role without executing a write.
func VerificationStatements() map[string]string {
	return map[string]string{
		"stage_candidate_entity": insertCandidateEntitySQL,
		"stage_candidate_event":  insertCandidateEventSQL,
		"registry_entity":        insertRegistryEntitiesSQL,
		"registry_entity_alias":  insertRegistryAliasesSQL,
		"entity_mention":         insertMentionsSQL,
		"entity_resolution":      insertResolutionsSQL,
	}
}

// insertMentionsSQL writes append-only mentions; data_tier and provenance
// reach their ai.* columns by assignment coercion from text.
const insertMentionsSQL = `INSERT INTO working.entity_mention
  (id, surface_text, mention_kind, subject_type, subject_id, start_char, end_char, context_snippet, extraction_method, confidence, data_tier, provenance)
SELECT m.id, m.surface, m.kind, 'context.normalized_record_identity', m.record_id, m.start_char, m.end_char, NULLIF(m.snippet, ''),
       m.method, m.confidence, 'extracted', m.provenance::ai.source_ref[]
FROM unnest($1::uuid[], $2::text[], $3::text[], $4::uuid[], $5::int4[], $6::int4[], $7::text[], $8::text[], $9::numeric[], $10::text[])
     AS m(id, surface, kind, record_id, start_char, end_char, snippet, method, confidence, provenance)
ON CONFLICT (id) DO NOTHING`

// insertResolutionsSQL writes the current resolution of each mention unless
// it already resolves to the same entity.
const insertResolutionsSQL = `INSERT INTO working.entity_resolution
  (id, mention_id, canonical_entity_id, match_method, resolved_by, match_score, similarity_metrics, requires_human_review,
   review_status, reviewed_by, reviewed_at, review_notes, safe_for_legal_use, provenance)
SELECT i.id, i.mention_id, i.entity_id, i.method, i.resolved_by, i.score, i.metrics::jsonb, false,
       'approved', $9, $10, $11, false, i.provenance::ai.source_ref[]
FROM unnest($1::uuid[], $2::uuid[], $3::uuid[], $4::text[], $5::text[], $6::numeric[], $7::text[], $8::text[])
     AS i(id, mention_id, entity_id, method, resolved_by, score, metrics, provenance)
WHERE NOT EXISTS (SELECT 1 FROM working.entity_resolution r
                  WHERE r.mention_id = i.mention_id AND upper_inf(r.sys_period) AND r.canonical_entity_id = i.entity_id)
ON CONFLICT DO NOTHING`

func int32Pointer(value *int) *int32 {
	if value == nil {
		return nil
	}
	converted := int32(*value)
	return &converted
}

// WriteEventCandidates inserts committed events and their typed
// source_available_from anchors.
func (s *EntityExtractionStore) WriteEventCandidates(ctx context.Context, rows []service.EventCandidateRow, reviewer service.Reviewer) (int, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	written := 0
	for _, row := range rows {
		locator, err := json.Marshal(row.SourceLocator)
		if err != nil {
			return 0, err
		}
		refs := row.EntityRefs
		if refs == nil {
			refs = []string{}
		}
		tag, err := tx.Exec(ctx, `INSERT INTO timeline.event_candidate
  (id, source_system, source_record_id, source_record_version, source_locator, extraction_run_id, temporal_precision,
   occurred_at, temporal_confidence, display_summary, event_type, entity_refs)
VALUES ($1::uuid, $2, $3, $4, $5::jsonb, $6, $7, $8, $9, $10, $11, $12::text[])
ON CONFLICT DO NOTHING`, row.ID, service.SourceSystem, row.SourceRecordID, row.SourceRecordVersion, string(locator),
			row.ExtractionRunID, row.Precision, row.OccurredAt, row.TemporalConfidence, row.Summary, row.EventType, refs)
		if err != nil {
			return 0, fmt.Errorf("write event candidate %s: %w", row.ID, err)
		}
		written += int(tag.RowsAffected())
		metadata, err := json.Marshal(row.AnchorMetadata)
		if err != nil {
			return 0, err
		}
		presentation, _ := json.Marshal(map[string]any{"label": "Source available from", "event_candidate_id": row.ID})
		if _, err := tx.Exec(ctx, `INSERT INTO context.relative_time_anchor
  (id, anchor_key, version_ordinal, placement_kind, lower_bound_at, metadata_basis, raw_metadata, confidence,
   review_state, reviewed_by, reviewed_at, provenance_digest, presentation_payload)
VALUES ($1::uuid, $2::uuid, 1, 'after', $3, 'normalized_record.source_available_from', $4::jsonb, 1.0,
        'approved', $5, $6, $7, $8::jsonb)
ON CONFLICT DO NOTHING`, row.AnchorID, row.AnchorKey, row.AvailableFrom, string(metadata), reviewer.Username, reviewer.At, row.ProvenanceDigest, string(presentation)); err != nil {
			return 0, fmt.Errorf("write source availability anchor: %w", err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO timeline.event_candidate_relative_time_anchor (event_candidate_id, anchor_id, anchor_role)
VALUES ($1::uuid, $2::uuid, 'source_available_from') ON CONFLICT DO NOTHING`, row.ID, row.AnchorID); err != nil {
			return 0, fmt.Errorf("link source availability anchor: %w", err)
		}
	}
	return written, tx.Commit(ctx)
}

// EnsureCollection creates the timeline collection once and returns its id.
func (s *EntityExtractionStore) EnsureCollection(ctx context.Context, slug, title string) (string, error) {
	if _, err := s.exec(ctx, `INSERT INTO timeline.timeline_collection (id, slug, title) VALUES ($1::uuid, $2, $3) ON CONFLICT (slug) DO NOTHING`,
		flow.DeterministicID("timeline_collection", slug), slug, title); err != nil {
		return "", err
	}
	var id string
	err := s.db.QueryRow(ctx, `SELECT id::text FROM timeline.timeline_collection WHERE slug = $1`, slug).Scan(&id)
	return id, err
}

// WriteTimelineMembers adds candidate-authority members to a collection.
func (s *EntityExtractionStore) WriteTimelineMembers(ctx context.Context, rows []service.TimelineMemberRow) (int, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	var ids, collections, candidates []string
	for _, row := range rows {
		ids, collections, candidates = append(ids, row.ID), append(collections, row.CollectionID), append(candidates, row.CandidateID)
	}
	affected, err := s.exec(ctx, `INSERT INTO timeline.timeline_member (id, collection_id, member_authority, candidate_id, included)
SELECT r.id, r.collection_id, 'candidate_context', r.candidate_id, true
FROM unnest($1::uuid[], $2::uuid[], $3::uuid[]) AS r(id, collection_id, candidate_id)
ON CONFLICT DO NOTHING`, ids, collections, candidates)
	return int(affected), err
}

// FinalizeCommit writes the commit receipt and promotes committed proposals.
func (s *EntityExtractionStore) FinalizeCommit(ctx context.Context, receipt service.CommitReceipt, entityPromotions, eventPromotions []service.Promotion) error {
	stats, err := json.Marshal(nonNilStats(receipt.Stats))
	if err != nil {
		return err
	}
	started := receipt.StartedAt
	if started.IsZero() || started.After(time.Now()) {
		started = time.Now()
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `INSERT INTO working.extraction_run
  (id, started_at, finished_at, extractor, extractor_version, source_summary, status, error, stats)
VALUES ($1::uuid, $2, now(), $3, $4, $5, $6, NULLIF($7, ''), $8::jsonb)
ON CONFLICT (id) DO NOTHING`, receipt.ID, started, service.CommitExtractor, service.CommitVersion, receipt.Summary, receipt.Status, receipt.Error, string(stats)); err != nil {
		return fmt.Errorf("write commit receipt: %w", err)
	}
	for _, target := range []struct {
		table      string
		promotions []service.Promotion
	}{{"working.candidate_entity", entityPromotions}, {"working.candidate_event", eventPromotions}} {
		if len(target.promotions) == 0 {
			continue
		}
		var ids, tables, targets []string
		for _, promotion := range target.promotions {
			ids, tables, targets = append(ids, promotion.CandidateID), append(tables, promotion.Table), append(targets, promotion.TargetID)
		}
		if _, err := tx.Exec(ctx, `UPDATE `+target.table+` c
SET review_state = 'approved', promoted_to_table = p.tbl, promoted_to_id = p.target, promoted_at = now()
FROM unnest($1::uuid[], $2::text[], $3::text[]) AS p(id, tbl, target)
WHERE c.id = p.id AND c.review_state = 'pending'`, ids, tables, targets); err != nil {
			return fmt.Errorf("promote %s: %w", target.table, err)
		}
	}
	return tx.Commit(ctx)
}

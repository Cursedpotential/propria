// Byline: Claude Code · Opus 5.5 · 2026-10-02
//
// Message match-up across sources (owner 2026-10-02: "both Facebook exports,
// deduped"; decision C). A message whose platform, parties, sender, sent time
// to the second and body hash already exist from a different source version is
// recorded in working.message_occurrence as a further occurrence of the
// existing working row instead of a second row. The key is computed by one SQL
// function, working.message_match_key, for new messages here and for the
// one-time back-fill of committed rows, so both always agree.
package postgres

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/firstparty"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/stagegraph"
)

// MessageOccurrenceDeriverVersion stamps every working.message_occurrence row.
const MessageOccurrenceDeriverVersion = "engine:message_occurrence@1.0.0"

const partySeparator = "\x1f"

type rowQueryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

// partyKey is one party as the match key names it: the registry entity when
// resolved, else the registry-normalized identifier, else the raw identifier.
// The back-fill SQL builds the same string from committed participant rows.
func partyKey(plan firstparty.Plan, participant firstparty.Participant) string {
	if entity := strings.ToLower(strings.TrimSpace(participant.EntityID)); entity != "" {
		return entity
	}
	if resolved, ok := plan.Resolution.Lookup(participant.Raw); ok && strings.TrimSpace(resolved.Normalized) != "" {
		return "norm:" + strings.TrimSpace(resolved.Normalized)
	}
	return "raw:" + strings.ToLower(strings.TrimSpace(participant.Raw))
}

// messageKeyParts returns the message's distinct party keys (sorted) and its
// sender's key.
func messageKeyParts(plan firstparty.Plan, message firstparty.Message) ([]string, string) {
	seen := map[string]bool{}
	parties := []string{}
	sender := ""
	for _, participant := range message.Participants {
		key := partyKey(plan, participant)
		if participant.Role == "from" {
			sender = key
		}
		if !seen[key] {
			seen[key] = true
			parties = append(parties, key)
		}
	}
	sort.Strings(parties)
	return parties, sender
}

// pairLockKey serializes match-up for one set of parties, so two sources of the
// same conversation (two backups, two exports, two phones) cannot both commit a
// message as new at the same moment.
func pairLockKey(plan firstparty.Plan, conversation firstparty.Conversation) string {
	seen := map[string]bool{}
	parties := []string{}
	for _, message := range conversation.Messages {
		keys, _ := messageKeyParts(plan, message)
		for _, key := range keys {
			if !seen[key] {
				seen[key] = true
				parties = append(parties, key)
			}
		}
	}
	sort.Strings(parties)
	return "working.message_occurrence:" + plan.Source.Platform + ":" + strings.Join(parties, ",")
}

// matchKeys computes working.message_match_key for every message of one
// conversation, keyed by record id.
func matchKeys(ctx context.Context, q rowQueryer, plan firstparty.Plan, messages []firstparty.Message) (map[string]string, error) {
	if len(messages) == 0 {
		return map[string]string{}, nil
	}
	ids := make([]string, len(messages))
	parties := make([]string, len(messages))
	senders := make([]string, len(messages))
	occurred := make([]pgtype.Timestamptz, len(messages))
	shas := make([][]byte, len(messages))
	for index, message := range messages {
		keys, sender := messageKeyParts(plan, message)
		ids[index], parties[index], senders[index] = message.RecordID, strings.Join(keys, partySeparator), sender
		if message.OccurredAt != nil {
			occurred[index] = pgtype.Timestamptz{Time: *message.OccurredAt, Valid: true}
		}
		sum, err := hex.DecodeString(message.ContentSHA256)
		if err != nil {
			return nil, fmt.Errorf("message %s content hash: %w", message.RecordID, err)
		}
		shas[index] = sum
	}
	rows, err := q.Query(ctx, `
		SELECT u.id::text, working.message_match_key($1, string_to_array(u.parties, chr(31)), NULLIF(u.sender, ''), u.occurred_at, u.sha)
		FROM unnest($2::uuid[], $3::text[], $4::text[], $5::timestamptz[], $6::bytea[]) AS u(id, parties, sender, occurred_at, sha)`,
		plan.Source.Platform, ids, parties, senders, occurred, shas)
	if err != nil {
		return nil, fmt.Errorf("compute message match keys: %w", err)
	}
	defer rows.Close()
	out := make(map[string]string, len(messages))
	for rows.Next() {
		var id, key string
		if err := rows.Scan(&id, &key); err != nil {
			return nil, err
		}
		out[id] = key
	}
	return out, rows.Err()
}

type primaryOccurrence struct {
	RecordID    string
	Perspective string
}

// existingPrimaries returns, per match key, the earliest committed working row
// of that message from a source version other than this one.
func existingPrimaries(ctx context.Context, q rowQueryer, keys []string, sourceVersionID string) (map[string]primaryOccurrence, error) {
	out := map[string]primaryOccurrence{}
	if len(keys) == 0 {
		return out, nil
	}
	rows, err := q.Query(ctx, `
		SELECT DISTINCT ON (match_key) match_key, primary_record_id::text, coalesce(perspective_person_id::text, '')
		FROM working.message_occurrence
		WHERE match_key = ANY($1::text[]) AND normalized_record_id = primary_record_id
		  AND source_version_id <> $2::uuid
		ORDER BY match_key, recorded_at, normalized_record_id`, keys, sourceVersionID)
	if err != nil {
		return nil, fmt.Errorf("find matching committed messages: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		var primary primaryOccurrence
		if err := rows.Scan(&key, &primary.RecordID, &primary.Perspective); err != nil {
			return nil, err
		}
		out[key] = primary
	}
	return out, rows.Err()
}

// conversationMatch is one conversation split by the match-up rule.
type conversationMatch struct {
	Kept    firstparty.Conversation
	Keys    map[string]string
	Matched []activities.MessageMatch
}

// matchConversation splits one conversation into the messages this source
// commits and the ones an earlier source already committed.
func matchConversation(ctx context.Context, q rowQueryer, plan firstparty.Plan, conversation firstparty.Conversation) (conversationMatch, error) {
	keys, err := matchKeys(ctx, q, plan, conversation.Messages)
	if err != nil {
		return conversationMatch{}, err
	}
	distinct := make([]string, 0, len(keys))
	seen := map[string]bool{}
	for _, key := range keys {
		if !seen[key] {
			seen[key] = true
			distinct = append(distinct, key)
		}
	}
	sort.Strings(distinct)
	primaries, err := existingPrimaries(ctx, q, distinct, plan.Source.SourceVersionID)
	if err != nil {
		return conversationMatch{}, err
	}
	out := conversationMatch{Kept: conversation, Keys: keys}
	out.Kept.Messages = make([]firstparty.Message, 0, len(conversation.Messages))
	perspective := strings.ToLower(strings.TrimSpace(plan.Identity.PerspectivePersonID))
	for _, message := range conversation.Messages {
		key := keys[message.RecordID]
		primary, ok := primaries[key]
		if !ok || primary.RecordID == message.RecordID {
			out.Kept.Messages = append(out.Kept.Messages, message)
			continue
		}
		out.Matched = append(out.Matched, activities.MessageMatch{
			RecordID: message.RecordID, PrimaryRecordID: primary.RecordID, MatchKey: key,
			CrossDevice: primary.Perspective != "" && perspective != "" && primary.Perspective != perspective,
		})
	}
	return out, nil
}

const insertOccurrencesSQL = `
	INSERT INTO working.message_occurrence
	    (normalized_record_id, match_key, source_version_id, primary_record_id, projection_kind,
	     perspective_person_id, cross_device, deriver_version)
	SELECT u.id, u.match_key, $1::uuid, u.primary_id, $2, NULLIF($3, '')::uuid, u.cross_device, $4
	FROM unnest($5::uuid[], $6::text[], $7::uuid[], $8::boolean[]) AS u(id, match_key, primary_id, cross_device)
	ON CONFLICT (normalized_record_id) DO NOTHING`

// recordOccurrences writes one conversation's occurrence rows: each kept
// message as its own primary, each matched one pointing at the earlier row.
func recordOccurrences(ctx context.Context, tx pgx.Tx, plan firstparty.Plan, kind string, match conversationMatch) error {
	ids, keys, primaries := []string{}, []string{}, []string{}
	cross := []bool{}
	for _, message := range match.Kept.Messages {
		ids, keys, primaries = append(ids, message.RecordID), append(keys, match.Keys[message.RecordID]), append(primaries, message.RecordID)
		cross = append(cross, false)
	}
	for _, matched := range match.Matched {
		ids, keys, primaries = append(ids, matched.RecordID), append(keys, matched.MatchKey), append(primaries, matched.PrimaryRecordID)
		cross = append(cross, matched.CrossDevice)
	}
	if len(ids) == 0 {
		return nil
	}
	if _, err := tx.Exec(ctx, insertOccurrencesSQL, plan.Source.SourceVersionID, kind, plan.Identity.PerspectivePersonID,
		MessageOccurrenceDeriverVersion, ids, keys, primaries, cross); err != nil {
		return fmt.Errorf("record message occurrences: %w", err)
	}
	return nil
}

func lockParties(ctx context.Context, tx pgx.Tx, plan firstparty.Plan, conversation firstparty.Conversation) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, pairLockKey(plan, conversation))
	return err
}

// MessageMatchStore implements activities.MessageMatchStore: the pre-publish
// look-up that tells the Weaviate-first stage which messages already have a
// working row (and search object) from another source.
type MessageMatchStore struct {
	db  DB
	now func() time.Time
}

// NewMessageMatchStore validates its pool.
func NewMessageMatchStore(db DB) (*MessageMatchStore, error) {
	if db == nil {
		return nil, errors.New("message match store requires a database")
	}
	return &MessageMatchStore{db: db, now: func() time.Time { return time.Now().UTC() }}, nil
}

// FindMessageMatches implements activities.MessageMatchStore.
func (s *MessageMatchStore) FindMessageMatches(ctx context.Context, plan firstparty.Plan) ([]activities.MessageMatch, error) {
	out := []activities.MessageMatch{}
	for _, conversation := range plan.Conversations {
		match, err := matchConversation(ctx, s.db, plan, conversation)
		if err != nil {
			return nil, err
		}
		out = append(out, match.Matched...)
	}
	return out, nil
}

type messageMatchReceipt struct {
	RefKind     string                    `json:"ref_kind"`
	RefID       string                    `json:"ref_id"`
	Generation  string                    `json:"normalized_generation"`
	Messages    int                       `json:"messages"`
	Matched     int                       `json:"matched"`
	CrossDevice int                       `json:"cross_device"`
	Matches     []activities.MessageMatch `json:"matches"`
}

// PersistMessageMatches records the look-up's receipt; its id is the result Ref.
func (s *MessageMatchStore) PersistMessageMatches(ctx context.Context, spec activities.MessageMatchSpec) (proffer.Ref, proffer.Ref, error) {
	sourceVersionID, err := uuid.Parse(string(spec.SourceVersionRef))
	if err != nil {
		return "", "", fmt.Errorf("source version reference %q: %w", spec.SourceVersionRef, err)
	}
	stage := string(stagegraph.MatchMessageOccurrences)
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", "", err
	}
	defer func() { cleanup, cancel := boundedCleanup(ctx); defer cancel(); _ = tx.Rollback(cleanup) }()
	executionID, err := parserEnsureExecution(ctx, tx, sourceVersionID, spec.RequestID, stage, stage+":"+string(spec.NormalizedGenerationRef))
	if err != nil {
		return "", "", err
	}
	if receiptID, _, has, err := normalizeLatestReceipt(ctx, tx, executionID); err != nil {
		return "", "", err
	} else if has {
		return proffer.Ref(receiptID.String()), proffer.Ref(receiptID.String()), tx.Commit(ctx)
	}
	receiptID, err := uuid.NewV7()
	if err != nil {
		return "", "", err
	}
	cross := 0
	for _, match := range spec.Matches {
		if match.CrossDevice {
			cross++
		}
	}
	matches := spec.Matches
	if matches == nil {
		matches = []activities.MessageMatch{}
	}
	encoded, err := json.Marshal(messageMatchReceipt{
		RefKind: "message_matches", RefID: receiptID.String(), Generation: string(spec.NormalizedGenerationRef),
		Messages: spec.Messages, Matched: len(matches), CrossDevice: cross, Matches: matches,
	})
	if err != nil {
		return "", "", err
	}
	now := s.now()
	if _, err := tx.Exec(ctx, `
		INSERT INTO context.activity_receipt (id, activity_execution_id, attempt, status, started_at, completed_at, result_ref)
		VALUES ($1::uuid, $2::uuid, $3, 'success', $4, $4, $5::jsonb)`,
		receiptID, executionID, spec.Attempt, now, encoded); err != nil {
		return "", "", fmt.Errorf("write message match receipt: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return "", "", err
	}
	return proffer.Ref(receiptID.String()), proffer.Ref(receiptID.String()), nil
}

// LoadMatchedRecords returns the record ids a message-match receipt names.
func (s *MessageMatchStore) LoadMatchedRecords(ctx context.Context, ref proffer.Ref) (map[string]bool, error) {
	var raw []byte
	if err := s.db.QueryRow(ctx, `
		SELECT receipt.result_ref FROM context.activity_receipt receipt
		JOIN context.activity_execution execution ON execution.id = receipt.activity_execution_id
		WHERE receipt.id = $1::uuid AND receipt.status = 'success' AND execution.activity_name = $2`,
		string(ref), string(stagegraph.MatchMessageOccurrences)).Scan(&raw); err != nil {
		return nil, fmt.Errorf("read message match receipt %s: %w", ref, err)
	}
	var receipt messageMatchReceipt
	if err := json.Unmarshal(raw, &receipt); err != nil {
		return nil, fmt.Errorf("decode message match receipt: %w", err)
	}
	out := make(map[string]bool, len(receipt.Matches))
	for _, match := range receipt.Matches {
		out[strings.ToLower(match.RecordID)] = true
	}
	return out, nil
}

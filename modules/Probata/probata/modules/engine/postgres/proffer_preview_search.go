// Byline: Claude Code · Opus 5 · 2026-09-20 (server-side preview message search)
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/Cursedpotential/probata/engine/runtimeapi/previewmodel"
)

// messagePredicate is the single source of truth for the filter. It appears
// verbatim in both the page read and the count so a match can never be counted
// differently from how it is listed.
//
// Parameters: $1 preview_handle, $2 snapshot_seq, $3 body ILIKE pattern (NULL =
// no text filter), $4 has_attachments, $5 sender participant id (NULL = any),
// $6 from bound (NULL = open), $7 to bound (NULL = open).
//
// Every user value is bound, never concatenated. The ILIKE pattern arrives
// pre-escaped by previewmodel.EscapeLikePattern and is read with ESCAPE '\' so
// a literal "100%" or "a_b" searches for itself instead of acting as wildcards.
const messagePredicate = `
	m.preview_handle = $1 AND m.snapshot_seq = $2::bigint
	AND ($3::text IS NULL OR m.body ILIKE $3::text ESCAPE '\')
	AND (NOT $4::boolean OR EXISTS (
	        SELECT 1 FROM context.proffer_preview_attachment a
	        WHERE a.preview_handle = $1 AND a.snapshot_seq = $2::bigint
	          AND a.message_id = m.message_id))
	AND ($5::text IS NULL OR m.sender_participant_id = $5::text)
	AND ($6::timestamptz IS NULL OR m.sent_at >= $6::timestamptz)
	AND ($7::timestamptz IS NULL OR m.sent_at <= $7::timestamptz)`

// filterArgs renders the filter as the seven bound parameters. A disabled
// predicate is passed as a typed NULL rather than omitted, so one statement
// text serves every combination and PostgreSQL can keep one plan.
func filterArgs(handle string, seq int64, filter previewmodel.MessageFilter) []any {
	var pattern, sender any
	var from, to any
	if filter.Query != "" {
		pattern = previewmodel.EscapeLikePattern(filter.Query)
	}
	if filter.Sender != "" {
		sender = filter.Sender
	}
	if filter.From != nil {
		from = filter.From.UTC()
	}
	if filter.To != nil {
		to = filter.To.UTC()
	}
	return []any{handle, seq, pattern, filter.HasAttachments, sender, from, to}
}

// latestSnapshotSeq resolves the newest projection generation for a handle,
// distinguishing "no such preview" from "not published yet".
func (s *ProfferPreviewStore) latestSnapshotSeq(ctx context.Context, handle string) (int64, error) {
	var seq int64
	err := s.db.QueryRow(ctx, `SELECT snapshot_seq FROM context.proffer_preview_snapshot WHERE preview_handle = $1 ORDER BY snapshot_seq DESC LIMIT 1`, handle).Scan(&seq)
	if errors.Is(err, pgx.ErrNoRows) {
		if _, bindingErr := s.Binding(ctx, handle); errors.Is(bindingErr, previewmodel.ErrNotFound) {
			return 0, bindingErr
		}
		return 0, previewmodel.ErrNotReady
	}
	return seq, err
}

// SearchPage serves one ordinal-ordered window of a preview's messages under a
// bounded server-side filter, plus the filtered and unfiltered totals. The
// owner has eight years of messages; narrowing has to happen in PostgreSQL, not
// over the rows a browser happens to have loaded.
func (s *ProfferPreviewStore) SearchPage(ctx context.Context, handle string, filter previewmodel.MessageFilter, offset, limit int) (previewmodel.Page, error) {
	if offset < 0 || limit < 1 || limit > 250 {
		return previewmodel.Page{}, errors.New("preview page bounds are invalid")
	}
	if err := filter.Validate(); err != nil {
		return previewmodel.Page{}, err
	}
	seq, err := s.latestSnapshotSeq(ctx, handle)
	if err != nil {
		return previewmodel.Page{}, err
	}
	page := previewmodel.Page{TotalMatches: 0, TotalMessages: 0}

	participantRows, err := s.db.Query(ctx, `SELECT participant_id, display_name, canonical_address FROM context.proffer_preview_participant WHERE preview_handle = $1 AND snapshot_seq = $2 ORDER BY participant_id`, handle, seq)
	if err != nil {
		return page, err
	}
	for participantRows.Next() {
		var participant previewmodel.Participant
		if err := participantRows.Scan(&participant.ParticipantID, &participant.DisplayName, &participant.CanonicalAddress); err != nil {
			participantRows.Close()
			return page, err
		}
		page.Participants = append(page.Participants, participant)
	}
	if err := participantRows.Err(); err != nil {
		participantRows.Close()
		return page, err
	}
	participantRows.Close()

	args := filterArgs(handle, seq, filter)
	// One pass yields both totals: the filtered count and the whole thread.
	if err := s.db.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE`+messagePredicate+`), count(*)
		FROM context.proffer_preview_message m
		WHERE m.preview_handle = $1 AND m.snapshot_seq = $2::bigint`,
		args...).Scan(&page.TotalMatches, &page.TotalMessages); err != nil {
		return page, fmt.Errorf("count preview messages: %w", err)
	}

	messageRows, err := s.db.Query(ctx, `
		SELECT m.message_id, m.ordinal, m.sent_at, m.sender_participant_id, m.body,
		       m.participant_ids, m.source_locator_ref
		FROM context.proffer_preview_message m
		WHERE`+messagePredicate+`
		ORDER BY m.ordinal, m.message_id OFFSET $8 LIMIT $9`,
		append(args, offset, limit+1)...)
	if err != nil {
		return page, fmt.Errorf("read preview messages: %w", err)
	}
	for messageRows.Next() {
		var message previewmodel.Message
		if err := messageRows.Scan(&message.MessageID, &message.Ordinal, &message.SentAt,
			&message.SenderParticipantID, &message.Body, &message.ParticipantIDs,
			&message.SourceLocatorRef); err != nil {
			messageRows.Close()
			return page, err
		}
		page.Messages = append(page.Messages, message)
	}
	if err := messageRows.Err(); err != nil {
		messageRows.Close()
		return page, err
	}
	messageRows.Close()
	if len(page.Messages) > limit {
		next := offset + limit
		page.NextOffset = &next
		page.Messages = page.Messages[:limit]
	}
	if len(page.Messages) == 0 {
		return page, nil
	}
	return page, s.hydrateAttachments(ctx, handle, seq, &page)
}

// hydrateAttachments attaches the projected attachment rows for exactly the
// messages on this page.
func (s *ProfferPreviewStore) hydrateAttachments(ctx context.Context, handle string, seq int64, page *previewmodel.Page) error {
	messageIDs := make([]string, len(page.Messages))
	byID := make(map[string]*previewmodel.Message, len(page.Messages))
	for index := range page.Messages {
		messageIDs[index] = page.Messages[index].MessageID
		byID[messageIDs[index]] = &page.Messages[index]
	}
	attachmentRows, err := s.db.Query(ctx, `
		SELECT message_id, attachment_id, filename, media_type, byte_length,
		       CASE WHEN sha256 IS NULL THEN NULL ELSE encode(sha256, 'hex') END,
		       source_locator_ref
		FROM context.proffer_preview_attachment
		WHERE preview_handle = $1 AND snapshot_seq = $2 AND message_id = ANY($3::text[])
		ORDER BY message_id, attachment_id`, handle, seq, messageIDs)
	if err != nil {
		return err
	}
	defer attachmentRows.Close()
	for attachmentRows.Next() {
		var messageID string
		var attachment previewmodel.Attachment
		if err := attachmentRows.Scan(&messageID, &attachment.AttachmentID, &attachment.Filename,
			&attachment.MediaType, &attachment.ByteLength, &attachment.SHA256,
			&attachment.SourceLocatorRef); err != nil {
			return err
		}
		attachment.MarkPayload()
		if message := byID[messageID]; message != nil {
			message.Attachments = append(message.Attachments, attachment)
		}
	}
	return attachmentRows.Err()
}

var _ previewmodel.MessageSearchStore = (*ProfferPreviewStore)(nil)

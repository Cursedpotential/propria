// Byline: Claude Code · Opus 5 · 2026-09-20 (preview message search tests)
package runtimeapi

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Cursedpotential/probata/engine/runtimeapi/previewmodel"
)

type searchMessagesResponse struct {
	Messages []struct {
		MessageID string `json:"message_id"`
		Ordinal   int64  `json:"ordinal"`
		Body      string `json:"body"`
	} `json:"messages"`
	Participants  []PreviewParticipant `json:"participants"`
	TotalMatches  int64                `json:"total_matches"`
	TotalMessages int64                `json:"total_messages"`
	NextCursor    *string              `json:"next_cursor"`
}

// searchFixture publishes a small projection whose bodies exercise every
// filter, including the LIKE metacharacters that must be treated literally.
func searchFixture(t *testing.T) (*PreviewHTTPHandler, string) {
	t.Helper()
	handler, store, _ := previewTestHandler(t)
	handle := startPreview(t, handler)

	snapshot := PreviewSnapshot{PreviewHandle: handle, Phase: "awaiting_decision", PreviewDigest: strings.Repeat("a", 64)}
	snapshot.Correlation.RequestID = "request-1"
	snapshot.Correlation.SourceVersionID = uuid.MustParse("33333333-3333-3333-3333-333333333333")
	snapshot.Correlation.RawGenerationID = uuid.MustParse("44444444-4444-4444-4444-444444444444")
	snapshot.Correlation.NormalizedGenerationID = uuid.MustParse("55555555-5555-5555-5555-555555555555")
	snapshot.Parser = &PreviewParser{ParserID: "sbv", ParserVersion: "1.2.3", ConfigDigest: strings.Repeat("b", 64)}
	for i, kind := range receiptTypes {
		snapshot.Receipts = append(snapshot.Receipts, PreviewReceipt{
			ReceiptType: kind, ReceiptRef: "receipt-" + kind, Status: "completed",
			RecordedAt: time.Unix(int64(i+1), 0).UTC(),
		})
	}
	alice := PreviewParticipant{ParticipantID: "p-1", DisplayName: "Alice"}
	bob := PreviewParticipant{ParticipantID: "p-2", DisplayName: "Bob"}

	at := func(day int) *time.Time {
		value := time.Date(2025, 6, day, 12, 0, 0, 0, time.UTC)
		return &value
	}
	sender := func(id string) *string { return &id }
	messages := []PreviewMessage{
		{MessageID: "m-1", Ordinal: 0, Body: "Happy anniversary my dude", SentAt: at(1), SenderParticipantID: sender("p-1"), ParticipantIDs: []string{"p-1"}, SourceLocatorRef: "ref/m-1"},
		{MessageID: "m-2", Ordinal: 1, Body: "battery is at 100% today", SentAt: at(2), SenderParticipantID: sender("p-2"), ParticipantIDs: []string{"p-2"}, SourceLocatorRef: "ref/m-2"},
		{MessageID: "m-3", Ordinal: 2, Body: "the file is named a_b.txt", SentAt: at(3), SenderParticipantID: sender("p-1"), ParticipantIDs: []string{"p-1"}, SourceLocatorRef: "ref/m-3"},
		{MessageID: "m-4", Ordinal: 3, Body: "axb is not the same thing", SentAt: at(4), SenderParticipantID: sender("p-2"), ParticipantIDs: []string{"p-2"}, SourceLocatorRef: "ref/m-4"},
		{MessageID: "m-5", Ordinal: 4, Body: "photo attached", SentAt: at(5), SenderParticipantID: sender("p-1"), ParticipantIDs: []string{"p-1"}, SourceLocatorRef: "ref/m-5",
			Attachments: []PreviewAttachment{{AttachmentID: "a-1", SourceLocatorRef: "ref/a-1"}}},
	}
	require.NoError(t, store.PutProjection(handle, snapshot, []PreviewParticipant{alice, bob}, messages, nil))
	return handler, handle
}

func searchMessages(t *testing.T, handler *PreviewHTTPHandler, handle, query string) (*searchMessagesResponse, int) {
	t.Helper()
	target := "/reference-import/previews/" + handle + "/messages"
	if query != "" {
		target += "?" + query
	}
	recorder := servePreview(handler.Routes(), http.MethodGet, target, nil)
	if recorder.Code != http.StatusOK {
		return nil, recorder.Code
	}
	var response searchMessagesResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	return &response, recorder.Code
}

func TestPreviewMessageSearchFilters(t *testing.T) {
	handler, handle := searchFixture(t)

	for _, testCase := range []struct {
		name  string
		query string
		want  []string
	}{
		{"unfiltered", "", []string{"m-1", "m-2", "m-3", "m-4", "m-5"}},
		{"body substring", "q=anniversary", []string{"m-1"}},
		{"body substring is case insensitive", "q=ANNIVERSARY", []string{"m-1"}},
		{"sender", "sender=p-2", []string{"m-2", "m-4"}},
		{"has attachments", "has_attachments=true", []string{"m-5"}},
		{"date window", "from=2025-06-02T00:00:00Z&to=2025-06-03T23:59:59Z", []string{"m-2", "m-3"}},
		{"combined", "q=is&sender=p-2", []string{"m-2", "m-4"}},
		// A percent sign and an underscore in the needle are literals, not
		// wildcards. "a_b" must not match "axb".
		{"percent is literal", "q=" + url.QueryEscape("100%"), []string{"m-2"}},
		{"underscore is literal", "q=" + url.QueryEscape("a_b"), []string{"m-3"}},
		{"no match is an empty list", "q=nothingmatchesthis", nil},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			response, code := searchMessages(t, handler, handle, testCase.query)
			require.Equal(t, http.StatusOK, code)
			got := make([]string, 0, len(response.Messages))
			for _, message := range response.Messages {
				got = append(got, message.MessageID)
			}
			require.Equal(t, testCase.want, nilIfEmpty(got))
			require.Equal(t, int64(len(testCase.want)), response.TotalMatches)
			require.Equal(t, int64(5), response.TotalMessages, "the unfiltered thread total never changes")
		})
	}
}

func nilIfEmpty(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	return values
}

// An empty result must serialize as [] rather than null: the BFF rejects a
// whole page on a null list.
func TestPreviewMessageSearchEmptyResultIsEmptyList(t *testing.T) {
	handler, handle := searchFixture(t)
	recorder := servePreview(handler.Routes(), http.MethodGet,
		"/reference-import/previews/"+handle+"/messages?q=nothingmatchesthis", nil)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"messages":[]`)
	require.NotContains(t, recorder.Body.String(), `"messages":null`)
}

func TestPreviewMessageSearchRejectsInvalidFilters(t *testing.T) {
	handler, handle := searchFixture(t)
	for _, query := range []string{
		"q=" + url.QueryEscape(strings.Repeat("x", previewmodel.MaxMessageQueryBytes+1)),
		"has_attachments=maybe",
		"from=not-a-timestamp",
		"from=2025-06-05T00:00:00Z&to=2025-06-01T00:00:00Z",
	} {
		_, code := searchMessages(t, handler, handle, query)
		require.Equal(t, http.StatusUnprocessableEntity, code, "query %q must be refused", query)
	}
}

// A cursor is bound to the filter that minted it. Replaying it under a
// different filter must be refused rather than silently paging another set.
func TestPreviewMessageSearchCursorIsBoundToItsFilter(t *testing.T) {
	handler, handle := searchFixture(t)

	first, code := searchMessages(t, handler, handle, "sender=p-1&limit=1")
	require.Equal(t, http.StatusOK, code)
	require.NotNil(t, first.NextCursor, "a narrowed page of 3 with limit 1 must offer a cursor")
	cursor := url.QueryEscape(*first.NextCursor)

	// Same filter: the cursor pages forward.
	second, code := searchMessages(t, handler, handle, "sender=p-1&limit=1&cursor="+cursor)
	require.Equal(t, http.StatusOK, code)
	require.Len(t, second.Messages, 1)
	require.NotEqual(t, first.Messages[0].MessageID, second.Messages[0].MessageID)

	// Any other filter: refused.
	for _, query := range []string{
		"limit=1&cursor=" + cursor,
		"sender=p-2&limit=1&cursor=" + cursor,
		"sender=p-1&q=photo&limit=1&cursor=" + cursor,
	} {
		_, code := searchMessages(t, handler, handle, query)
		require.Equal(t, http.StatusUnprocessableEntity, code, "cursor must not cross filter %q", query)
	}
}

func TestEscapeLikePattern(t *testing.T) {
	require.Equal(t, `%plain%`, previewmodel.EscapeLikePattern("plain"))
	require.Equal(t, `%100\%%`, previewmodel.EscapeLikePattern("100%"))
	require.Equal(t, `%a\_b%`, previewmodel.EscapeLikePattern("a_b"))
	require.Equal(t, `%c:\\path%`, previewmodel.EscapeLikePattern(`c:\path`))
}

// The zero filter keeps the legacy scope so an unfiltered cursor stays stable,
// and every distinct filter gets its own scope.
func TestMessageFilterCursorScope(t *testing.T) {
	require.Equal(t, "messages", previewmodel.MessageFilter{}.CursorScope())
	withQuery := previewmodel.MessageFilter{Query: "a"}.CursorScope()
	withOther := previewmodel.MessageFilter{Query: "b"}.CursorScope()
	withSender := previewmodel.MessageFilter{Sender: "a"}.CursorScope()
	require.NotEqual(t, "messages", withQuery)
	require.NotEqual(t, withQuery, withOther)
	require.NotEqual(t, withQuery, withSender, "the same value in a different field is a different filter")
}

// Publication is gated on normalized records of ANY kind, so a calls-only
// generation publishes while a generation that normalized nothing is refused.
func TestValidateAllowsRecordsWithoutMessages(t *testing.T) {
	handle := strings.Repeat("h", 32)
	snapshot := PreviewSnapshot{PreviewHandle: handle, Phase: "awaiting_decision", PreviewDigest: strings.Repeat("a", 64)}
	for _, kind := range receiptTypes {
		snapshot.Receipts = append(snapshot.Receipts, PreviewReceipt{
			ReceiptType: kind, ReceiptRef: "receipt-" + kind, Status: "completed",
		})
	}

	t.Run("calls only publishes", func(t *testing.T) {
		callsOnly := snapshot
		callsOnly.NormalizedRecordCount = 609
		require.NoError(t, ValidatePreviewProjection(handle, callsOnly, nil, nil))
	})

	t.Run("zero records of any kind is refused", func(t *testing.T) {
		err := ValidatePreviewProjection(handle, snapshot, nil, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "at least one normalized record")
	})

	t.Run("mixed generation still validates its messages", func(t *testing.T) {
		mixed := snapshot
		mixed.NormalizedRecordCount = 10
		participants := []PreviewParticipant{{ParticipantID: "p-1", DisplayName: "Alice"}}
		messages := []PreviewMessage{{MessageID: "m-1", Ordinal: 0, Body: "hi", ParticipantIDs: []string{"p-1"}, SourceLocatorRef: "ref/m-1"}}
		require.NoError(t, ValidatePreviewProjection(handle, mixed, participants, messages))

		messages[0].ParticipantIDs = []string{"unknown"}
		require.Error(t, ValidatePreviewProjection(handle, mixed, participants, messages))
	})
}

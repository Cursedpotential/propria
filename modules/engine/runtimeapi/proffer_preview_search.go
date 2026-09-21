// Byline: Claude Code · Opus 5 · 2026-09-20 (preview message search: memory store + request parsing)
package runtimeapi

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/runtimeapi/previewmodel"
)

// SearchPage is the in-memory twin of the durable search. It applies exactly
// the same predicate (previewmodel.MessageFilter.Matches) so handler tests
// exercise real filter semantics rather than a stub that always matches.
func (s *MemoryPreviewStore) SearchPage(_ context.Context, handle string, filter previewmodel.MessageFilter, offset, limit int) (PreviewPage, error) {
	if offset < 0 || limit < 1 || limit > maxPreviewPage {
		return PreviewPage{}, errors.New("preview page bounds are invalid")
	}
	if err := filter.Validate(); err != nil {
		return PreviewPage{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry := s.entries[handle]
	if entry == nil {
		return PreviewPage{}, ErrPreviewNotFound
	}
	if len(entry.projections) == 0 {
		return PreviewPage{}, ErrPreviewNotReady
	}
	projection := entry.projections[len(entry.projections)-1]
	matched := make([]PreviewMessage, 0, len(projection.messages))
	for _, message := range projection.messages {
		if filter.Matches(message) {
			matched = append(matched, message)
		}
	}
	page := PreviewPage{
		Participants:  clonePreviewParticipants(projection.participants),
		TotalMatches:  int64(len(matched)),
		TotalMessages: int64(len(projection.messages)),
	}
	if offset > len(matched) {
		// Past the end of the filtered set is an empty window, not a gap: the
		// result must be [] rather than an error the browser cannot recover from.
		page.Messages = []PreviewMessage{}
		return page, nil
	}
	end := offset + limit
	if end > len(matched) {
		end = len(matched)
	}
	page.Messages = clonePreviewMessages(matched[offset:end])
	if page.Messages == nil {
		page.Messages = []PreviewMessage{}
	}
	if end < len(matched) {
		page.NextOffset = &end
	}
	return page, nil
}

// parseMessageFilter reads the bounded search parameters off the query string.
// It fails closed: an unparsable bound is a 422, never a silently dropped
// predicate that would widen the result set behind the operator's back.
func parseMessageFilter(query url.Values) (previewmodel.MessageFilter, error) {
	filter := previewmodel.MessageFilter{
		Query:  strings.TrimSpace(query.Get("q")),
		Sender: strings.TrimSpace(query.Get("sender")),
	}
	switch raw := strings.TrimSpace(query.Get("has_attachments")); raw {
	case "", "false":
	case "true":
		filter.HasAttachments = true
	default:
		return filter, errors.New("has_attachments must be true or false")
	}
	for _, bound := range []struct {
		name  string
		field **time.Time
	}{{"from", &filter.From}, {"to", &filter.To}} {
		raw := strings.TrimSpace(query.Get(bound.name))
		if raw == "" {
			continue
		}
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return filter, errors.New(bound.name + " must be an RFC3339 timestamp")
		}
		utc := parsed.UTC()
		*bound.field = &utc
	}
	if err := filter.Validate(); err != nil {
		return filter, err
	}
	return filter, nil
}

var _ previewmodel.MessageSearchStore = (*MemoryPreviewStore)(nil)

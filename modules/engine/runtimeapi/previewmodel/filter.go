// Byline: Claude Code · Opus 5 · 2026-09-20 (server-side preview message search filter)
package previewmodel

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// MaxMessageQueryBytes bounds the free-text needle. The owner's threads run to
// years of messages; the needle stays small so one COUNT plus one page scan is
// the entire cost of a search.
const MaxMessageQueryBytes = 200

// MessageFilter is the bounded server-side narrowing applied to one preview's
// message projection. The zero value means "the whole thread" and keeps the
// pre-existing unfiltered paging behaviour byte-for-byte.
//
// Every field is a predicate over columns that already exist on
// context.proffer_preview_message (plus an EXISTS over the attachment table).
// It defines no new storage and never reaches raw or normalized bytes.
type MessageFilter struct {
	// Query is a case-insensitive substring of the message body.
	Query string
	// HasAttachments keeps only messages carrying at least one attachment.
	HasAttachments bool
	// Sender is an exact preview participant id (the projection's own opaque id).
	Sender string
	// From and To bound sent_at inclusively.
	From *time.Time
	To   *time.Time
}

// IsZero reports whether the filter narrows nothing.
func (f MessageFilter) IsZero() bool {
	return f.Query == "" && !f.HasAttachments && f.Sender == "" && f.From == nil && f.To == nil
}

// Validate fails closed on an out-of-bounds or self-contradictory filter.
func (f MessageFilter) Validate() error {
	if len(f.Query) > MaxMessageQueryBytes {
		return fmt.Errorf("q must be at most %d bytes", MaxMessageQueryBytes)
	}
	if f.Query != "" && !utf8.ValidString(f.Query) {
		return errors.New("q must be valid UTF-8")
	}
	if strings.ContainsAny(f.Query, "\x00") || strings.ContainsAny(f.Sender, "\x00") {
		return errors.New("filter values must not contain NUL")
	}
	if len(f.Sender) > 128 {
		return errors.New("sender must be at most 128 bytes")
	}
	if f.From != nil && f.To != nil && f.To.Before(*f.From) {
		return errors.New("to must not precede from")
	}
	return nil
}

// CursorScope binds a paging cursor to the exact filter that minted it. A
// cursor issued for one filter therefore fails signature/scope validation under
// any other filter instead of silently paging the wrong result set.
func (f MessageFilter) CursorScope() string {
	if f.IsZero() {
		return "messages"
	}
	var builder strings.Builder
	builder.WriteString(f.Query)
	builder.WriteByte(0)
	if f.HasAttachments {
		builder.WriteString("1")
	}
	builder.WriteByte(0)
	builder.WriteString(f.Sender)
	builder.WriteByte(0)
	if f.From != nil {
		builder.WriteString(f.From.UTC().Format(time.RFC3339Nano))
	}
	builder.WriteByte(0)
	if f.To != nil {
		builder.WriteString(f.To.UTC().Format(time.RFC3339Nano))
	}
	digest := sha256.Sum256([]byte(builder.String()))
	return "messages-" + hex.EncodeToString(digest[:8])
}

// likeEscaper neutralises the LIKE metacharacters. The value itself is always
// bound as a parameter; this only stops a literal "100%" or "a_b" from being
// read as a wildcard. Pair it with ESCAPE '\' in the statement.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// EscapeLikePattern turns a user needle into a bounded contains-pattern.
func EscapeLikePattern(value string) string {
	return "%" + likeEscaper.Replace(value) + "%"
}

// Matches is the in-memory twin of the SQL predicate. The memory store uses it
// so test coverage exercises the same semantics the durable store implements.
func (f MessageFilter) Matches(message Message) bool {
	if f.Query != "" && !strings.Contains(strings.ToLower(message.Body), strings.ToLower(f.Query)) {
		return false
	}
	if f.HasAttachments && len(message.Attachments) == 0 {
		return false
	}
	if f.Sender != "" && (message.SenderParticipantID == nil || *message.SenderParticipantID != f.Sender) {
		return false
	}
	if f.From != nil && (message.SentAt == nil || message.SentAt.Before(*f.From)) {
		return false
	}
	if f.To != nil && (message.SentAt == nil || message.SentAt.After(*f.To)) {
		return false
	}
	return true
}

// MessageSearchStore is optional so existing preview stores and test doubles
// stay source-compatible, mirroring ContentStore. A store that does not
// implement it simply serves unfiltered pages with no totals.
type MessageSearchStore interface {
	SearchPage(ctx context.Context, handle string, filter MessageFilter, offset, limit int) (Page, error)
}

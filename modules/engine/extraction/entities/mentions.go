// Byline: Claude Code · Opus 5.5 · 2026-09-25

package entities

import (
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Span is a half-open rune range [Start, End) inside a message body.
type Span struct {
	Start int
	End   int
}

// MinBodyAliasRunes is the shortest name alias searched for in bodies.
// "Al", "Jo" or an initial would match noise.
const MinBodyAliasRunes = 3

// FindNameOccurrences returns every word-bounded occurrence of name in body.
// Matching ignores case except for the first letter, which must be upper
// case in the body: people capitalize names, and requiring it keeps "will",
// "may" or "rose" from resolving to Will, May or Rose.
func FindNameOccurrences(body, name string) []Span {
	needle := []rune(strings.TrimSpace(name))
	if len(needle) < MinBodyAliasRunes {
		return nil
	}
	hay := []rune(body)
	var out []Span
	for i := 0; i+len(needle) <= len(hay); i++ {
		if !unicode.IsUpper(hay[i]) || unicode.ToLower(hay[i]) != unicode.ToLower(needle[0]) {
			continue
		}
		if i > 0 && isWordRune(hay[i-1]) {
			continue
		}
		matched := true
		for j := 1; j < len(needle); j++ {
			if unicode.ToLower(hay[i+j]) != unicode.ToLower(needle[j]) {
				matched = false
				break
			}
		}
		if !matched {
			continue
		}
		end := i + len(needle)
		if end < len(hay) && isWordRune(hay[end]) {
			continue
		}
		out = append(out, Span{Start: i, End: end})
		i = end - 1
	}
	return out
}

// FindTextOccurrence locates a model-reported mention verbatim (case-
// insensitive, word-bounded) in body. It is how model output is grounded:
// a mention the text does not contain is dropped, never trusted.
func FindTextOccurrence(body, text string) (Span, bool) {
	needle := []rune(strings.TrimSpace(text))
	if len(needle) == 0 {
		return Span{}, false
	}
	hay := []rune(body)
	for i := 0; i+len(needle) <= len(hay); i++ {
		if i > 0 && isWordRune(hay[i-1]) && isWordRune(needle[0]) {
			continue
		}
		matched := true
		for j := range needle {
			if unicode.ToLower(hay[i+j]) != unicode.ToLower(needle[j]) {
				matched = false
				break
			}
		}
		if !matched {
			continue
		}
		end := i + len(needle)
		if end < len(hay) && isWordRune(hay[end]) && isWordRune(needle[len(needle)-1]) {
			continue
		}
		return Span{Start: i, End: end}, true
	}
	return Span{}, false
}

// FindDigitsOccurrences finds a phone number typed inside a body, ignoring
// separators ("810-555-0101", "(810) 555 0101"). digits is the national
// number (NationalDigits).
func FindDigitsOccurrences(body, digits string) []Span {
	if len(digits) < 7 {
		return nil
	}
	hay := []rune(body)
	var out []Span
	for i := 0; i < len(hay); i++ {
		if !unicode.IsDigit(hay[i]) || (i > 0 && unicode.IsDigit(hay[i-1])) {
			continue
		}
		matched, j, k := 0, i, 0
		for j < len(hay) && k < len(digits) {
			r := hay[j]
			switch {
			case unicode.IsDigit(r):
				if byte(r) != digits[k] {
					k = len(digits) + 1
					continue
				}
				k++
				matched = j + 1
			case r == ' ' || r == '-' || r == '.' || r == '(' || r == ')':
			default:
				k = len(digits) + 1
				continue
			}
			j++
		}
		if k == len(digits) && (matched >= len(hay) || !unicode.IsDigit(hay[matched])) {
			out = append(out, Span{Start: i, End: matched})
			i = matched - 1
		}
	}
	return out
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

// Snippet returns up to radius runes of context either side of the span,
// with ellipses where it was cut.
func Snippet(body string, span Span, radius int) string {
	runes := []rune(body)
	start, end := span.Start-radius, span.End+radius
	prefix, suffix := "…", "…"
	if start <= 0 {
		start, prefix = 0, ""
	}
	if end >= len(runes) {
		end, suffix = len(runes), ""
	}
	if start > len(runes) {
		return ""
	}
	return prefix + strings.TrimSpace(string(runes[start:end])) + suffix
}

// MessageView is the part of a normalized message record extraction reads.
type MessageView struct {
	RecordID            string             `json:"record_id"`
	Ordinal             int64              `json:"ordinal"`
	OccurredAt          *time.Time         `json:"occurred_at,omitempty"`
	SourceAvailableFrom *time.Time         `json:"source_available_from,omitempty"`
	Body                string             `json:"body"`
	Participants        []MessageAddressee `json:"participants"`
}

// MessageAddressee is one participant entry of a normalized message.
type MessageAddressee struct {
	Role        string `json:"role"`
	Identifier  string `json:"identifier"`
	DisplayName string `json:"display_name,omitempty"`
}

// BodyMentions returns the body occurrences of the proposal's name aliases
// and typed phone numbers in one message.
func BodyMentions(message MessageView, proposal Proposal) []Mention {
	var out []Mention
	seen := map[Span]bool{}
	add := func(span Span, surface, kind string, confidence float64) {
		if seen[span] {
			return
		}
		seen[span] = true
		start, end := span.Start, span.End
		out = append(out, Mention{
			RecordID: message.RecordID, Ordinal: message.Ordinal, OccurredAt: message.OccurredAt,
			SourceAvailableFrom: message.SourceAvailableFrom, Kind: kind, Role: RoleBody,
			Surface: surface, Start: &start, End: &end, Snippet: Snippet(message.Body, span, 40),
			Method: MethodBodyAlias, Confidence: confidence,
		})
	}
	if message.Body == "" {
		return nil
	}
	for _, alias := range proposal.NameAliases() {
		tokens := Tokens(alias.Text)
		kind := "name"
		if len(tokens) == 1 && len(Tokens(proposal.Name)) > 1 {
			kind = "partial"
		}
		for _, span := range FindNameOccurrences(message.Body, alias.Text) {
			add(span, string([]rune(message.Body)[span.Start:span.End]), kind, 0.8)
		}
	}
	for _, alias := range proposal.AddressAliases() {
		address := Address{Kind: alias.AddressKind, Normalized: alias.Normalized}
		if digits := NationalDigits(address); digits != "" {
			for _, span := range FindDigitsOccurrences(message.Body, digits) {
				add(span, string([]rune(message.Body)[span.Start:span.End]), "phone", 0.9)
			}
		}
		if address.Kind == AddressEmail {
			if span, ok := FindTextOccurrence(message.Body, address.Normalized); ok {
				add(span, string([]rune(message.Body)[span.Start:span.End]), "email", 0.9)
			}
		}
	}
	return out
}

// ParticipantMentions returns the header mentions a message makes of the
// proposal: one per matching participant address, strongest role first.
// Scoped ("self") aliases match only inside their own source version.
func ParticipantMentions(message MessageView, proposal Proposal, sourceVersionID string) []Mention {
	addresses := map[string]Alias{}
	for _, alias := range proposal.AddressAliases() {
		if alias.Scope != "" && alias.Scope != "source:"+sourceVersionID {
			continue
		}
		addresses[string(alias.AddressKind)+":"+alias.Normalized] = alias
	}
	if len(addresses) == 0 {
		return nil
	}
	best := map[string]MessageAddressee{}
	rank := func(role string) int {
		switch roleOf(role) {
		case RoleSender:
			return 3
		case RoleRecipient:
			return 2
		default:
			return 1
		}
	}
	for _, participant := range message.Participants {
		address := NormalizeAddress(participant.Identifier)
		key := string(address.Kind) + ":" + address.Normalized
		if _, ok := addresses[key]; !ok {
			continue
		}
		if current, ok := best[key]; !ok || rank(participant.Role) > rank(current.Role) {
			best[key] = participant
		}
	}
	out := make([]Mention, 0, len(best))
	for _, participant := range best {
		address := NormalizeAddress(participant.Identifier)
		out = append(out, Mention{
			RecordID: message.RecordID, Ordinal: message.Ordinal, OccurredAt: message.OccurredAt,
			SourceAvailableFrom: message.SourceAvailableFrom, Kind: MentionKindFor(address),
			Role: roleOf(participant.Role), Surface: participant.Identifier,
			Method: MethodParticipant, Confidence: 1,
		})
	}
	sortMentions(out)
	return out
}

func sortMentions(mentions []Mention) {
	for i := 1; i < len(mentions); i++ {
		for j := i; j > 0 && mentions[j-1].SpanKey() > mentions[j].SpanKey(); j-- {
			mentions[j-1], mentions[j] = mentions[j], mentions[j-1]
		}
	}
}

// RuneLen is the body length in runes (the unit of start/end offsets).
func RuneLen(body string) int { return utf8.RuneCountInString(body) }

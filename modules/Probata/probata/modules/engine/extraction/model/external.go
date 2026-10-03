// Byline: Claude Code · Sonnet 5.5 · 2026-10-02

package model

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Cursedpotential/probata/engine/extraction/entities"
	"github.com/Cursedpotential/probata/engine/extraction/events"
)

// ExternalMention cites one message by its record id and the exact text used.
type ExternalMention struct {
	RecordID string `json:"record_id"`
	Text     string `json:"text"`
}

// ExternalEntity is one person, place or organization an external extractor found.
type ExternalEntity struct {
	Name     string            `json:"name"`
	Aliases  []string          `json:"aliases"`
	Mentions []ExternalMention `json:"mentions"`
}

// ExternalEvent is one event an external extractor found, tied to one message.
type ExternalEvent struct {
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	RecordID      string   `json:"record_id"`
	Date          *string  `json:"date"`
	When          *string  `json:"when"`
	Type          string   `json:"type"`
	People        []string `json:"people"`
	Places        []string `json:"places"`
	Organizations []string `json:"organizations"`
}

// ExternalPage is what an external (Python) extractor returns for one window of a
// conversation's messages. It is the default extractor's reply schema with the
// message labels replaced by record ids, so the window never has to travel
// through Temporal history: the Go side re-reads the same window by ordinal.
type ExternalPage struct {
	Extractor        string           `json:"extractor"`
	ExtractorVersion string           `json:"extractor_version"`
	ModelID          string           `json:"model_id,omitempty"`
	AfterOrdinal     int64            `json:"after_ordinal"`
	LastOrdinal      int64            `json:"last_ordinal"`
	Messages         int              `json:"messages"`
	Done             bool             `json:"done"`
	Skipped          bool             `json:"skipped,omitempty"`
	Reason           string           `json:"reason,omitempty"`
	People           []ExternalEntity `json:"people"`
	Places           []ExternalEntity `json:"places"`
	Organizations    []ExternalEntity `json:"organizations"`
	Events           []ExternalEvent  `json:"events"`
}

// ExternalTag is the extractor tag every proposal of an external extractor carries.
func ExternalTag(extractor, version string) string { return extractor + "@" + version }

// ExternalMethod is the mention method recorded for an external extractor.
func ExternalMethod(extractor, version string) string {
	return "extract:" + extractor + "@" + version
}

// GroundExternal validates an external extractor's page against the same schema
// the default extractor's replies must meet, then grounds it in the page's own
// messages: a mention whose text is not in its message, or an entity that appears
// nowhere in the window, is dropped and counted as ungrounded. Output that is
// malformed beyond repair yields an Invalid outcome and writes nothing.
// The messages are the window the extractor was given (ordinal order).
func GroundExternal(page ExternalPage, messages []entities.MessageView, scope entities.RunScope) BatchOutcome {
	outcome := BatchOutcome{Attempts: 1}
	if len(messages) > 0 {
		outcome.FirstOrd, outcome.LastOrd = messages[0].Ordinal, messages[len(messages)-1].Ordinal
	}
	if strings.TrimSpace(page.Extractor) == "" || strings.TrimSpace(page.ExtractorVersion) == "" {
		outcome.Invalid, outcome.Reason = true, "the extractor did not name itself and its version"
		return outcome
	}
	labelOf := make(map[string]string, len(messages))
	for i, message := range messages {
		labelOf[message.RecordID] = messageLabel(i)
	}
	response := Response{People: []EntityOut{}, Places: []EntityOut{}, Organizations: []EntityOut{}, Events: []EventOut{}}
	convert := func(list []ExternalEntity) []EntityOut {
		out := []EntityOut{}
		for _, item := range list {
			name := strings.TrimSpace(item.Name)
			if name == "" || len([]rune(name)) > entities.MaxNameRunes {
				outcome.Ungrounded++
				continue
			}
			entity := EntityOut{Name: name, Aliases: []string{}, Mentions: []MentionOut{}}
			for _, alias := range item.Aliases {
				if alias = strings.TrimSpace(alias); alias != "" && len([]rune(alias)) <= entities.MaxNameRunes {
					entity.Aliases = append(entity.Aliases, alias)
				}
			}
			for _, mention := range item.Mentions {
				label, ok := labelOf[mention.RecordID]
				if !ok || strings.TrimSpace(mention.Text) == "" {
					outcome.Ungrounded++
					continue
				}
				entity.Mentions = append(entity.Mentions, MentionOut{Message: label, Text: mention.Text})
			}
			out = append(out, entity)
		}
		return out
	}
	response.People, response.Places, response.Organizations = convert(page.People), convert(page.Places), convert(page.Organizations)
	for _, item := range page.Events {
		label, ok := labelOf[item.RecordID]
		title := strings.TrimSpace(item.Title)
		if !ok || title == "" || len([]rune(title)) > events.MaxTitleRunes {
			outcome.Ungrounded++
			continue
		}
		eventType := item.Type
		if !events.ValidEventType(eventType) {
			eventType = "other"
		}
		date := item.Date
		if date != nil && !dateTimePattern.MatchString(*date) {
			date = nil
		}
		response.Events = append(response.Events, EventOut{
			Title: title, Description: item.Description, Message: label, Date: date, When: item.When, Type: eventType,
			People: nonNilStrings(item.People), Places: nonNilStrings(item.Places), Organizations: nonNilStrings(item.Organizations),
		})
	}
	// One window, one batch: labels m1..mN cover every message, so the shared
	// validator and grounding run unchanged.
	batch := Batch{Index: 0, Messages: messages}
	encoded, err := json.Marshal(response)
	if err != nil {
		outcome.Invalid, outcome.Reason = true, err.Error()
		return outcome
	}
	if _, err := Decode(string(encoded), batch); err != nil {
		outcome.Invalid, outcome.Reason = true, fmt.Sprintf("external output failed validation: %v", err)
		return outcome
	}
	groundWith(&outcome, response, batch, ExternalTag(page.Extractor, page.ExtractorVersion),
		ExternalMethod(page.Extractor, page.ExtractorVersion), scope)
	return outcome
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

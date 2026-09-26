// Byline: Claude Code · Opus 5.5 · 2026-09-25

package model

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/extraction/entities"
	"github.com/Cursedpotential/probata/engine/extraction/events"
)

// Extractor identity recorded on working.extraction_run.
const (
	Extractor        = "probata.extract.model"
	ExtractorVersion = "1"
	PromptVersion    = "entities-events/v1"
)

// Batch bounds: one call covers at most this many messages or characters.
const (
	MaxBatchMessages  = 40
	MaxBatchChars     = 12000
	MaxBodyCharsShown = 1500
)

// Participant is one legend entry shown to the model.
type Participant struct {
	Label   string           `json:"label"`
	Address entities.Address `json:"address"`
	Name    string           `json:"name,omitempty"`
}

// Batch is one bounded window of messages.
type Batch struct {
	Index        int                    `json:"index"`
	Messages     []entities.MessageView `json:"messages"`
	Participants []Participant          `json:"participants"`
}

// Batches splits messages (in ordinal order) into bounded windows.
func Batches(messages []entities.MessageView, participants []Participant, startIndex int) []Batch {
	var out []Batch
	current := Batch{Index: startIndex, Participants: participants}
	chars := 0
	for _, message := range messages {
		length := len([]rune(message.Body))
		if length > MaxBodyCharsShown {
			length = MaxBodyCharsShown
		}
		if len(current.Messages) > 0 && (len(current.Messages) >= MaxBatchMessages || chars+length > MaxBatchChars) {
			out = append(out, current)
			current = Batch{Index: current.Index + 1, Participants: participants}
			chars = 0
		}
		current.Messages = append(current.Messages, message)
		chars += length
	}
	if len(current.Messages) > 0 {
		out = append(out, current)
	}
	return out
}

// Response is the exact JSON object the model must return.
type Response struct {
	People        []EntityOut `json:"people"`
	Places        []EntityOut `json:"places"`
	Organizations []EntityOut `json:"organizations"`
	Events        []EventOut  `json:"events"`
}

// EntityOut is one person, place or organization.
type EntityOut struct {
	Name        string       `json:"name"`
	Aliases     []string     `json:"aliases"`
	Participant *string      `json:"participant"`
	Mentions    []MentionOut `json:"mentions"`
}

// MentionOut cites one message and the exact text used.
type MentionOut struct {
	Message string `json:"message"`
	Text    string `json:"text"`
}

// EventOut is one event worth putting on a timeline.
type EventOut struct {
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	Message       string   `json:"message"`
	Date          *string  `json:"date"`
	When          *string  `json:"when"`
	Type          string   `json:"type"`
	People        []string `json:"people"`
	Places        []string `json:"places"`
	Organizations []string `json:"organizations"`
}

// SchemaText is the schema given to the model and enforced by Decode.
const SchemaText = `{
  "people":        [ {"name": string, "aliases": [string], "participant": "P1".."Pn" or null, "mentions": [ {"message": "m1".., "text": string} ]} ],
  "places":        [ {"name": string, "aliases": [string], "participant": null, "mentions": [ {"message": "m1".., "text": string} ]} ],
  "organizations": [ {"name": string, "aliases": [string], "participant": null, "mentions": [ {"message": "m1".., "text": string} ]} ],
  "events":        [ {"title": string, "description": string, "message": "m1"..,
                      "date": "YYYY-MM-DD" or "YYYY-MM-DDTHH:MM" or null, "when": string or null,
                      "type": one of [` + `"appointment","court","medical","school","custody_exchange","travel","incident","communication","financial","residence","work","other"` + `],
                      "people": [string], "places": [string], "organizations": [string]} ]
}
Every key shown is required; use [] and null when there is nothing. No other keys.`

const systemPrompt = `You read text messages and list (1) the people, places and organizations they mention and (2) the events worth putting on a timeline.
You extract. You do not interpret, judge, diagnose, or infer motives, feelings or relationships.
Rules:
- Only list names that appear in the messages. Every mentions[].text must be copied exactly from the message it cites.
- aliases: other names, nicknames or spellings the messages use for the same entity (copied exactly). Pet names like "babe" are not aliases.
- participant: when a person is one of the conversation participants and the messages make it plain (they are addressed by name in a message sent to them, or sign their own message), give that participant's label (P1, P2, ...). Otherwise null. Places and organizations always use null.
- events: things that happened or were arranged (appointments, court dates, hand-offs of a child, trips, moves, incidents). title and description restate what the message says, nothing more. In title and description call participants by name when the messages give one, otherwise "the device owner" or "the other participant" — never by their P-label.
- A possessive ("Jane's") is not an alias; list the name once.
- events[].date: only when the message states a complete calendar date (with year); otherwise null and put the phrase as written (for example "next Tuesday at 3") in when.
- Reply with ONE JSON object and nothing else, exactly this schema:
` + SchemaText

var dateTimePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}(T\d{2}:\d{2})?$`)

// Decode parses and validates a model reply against the schema: unknown
// keys, missing keys, wrong types, unknown labels and out-of-range values
// are all errors. Nothing that fails here is ever written.
func Decode(content string, batch Batch) (Response, error) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return Response{}, errors.New("empty reply")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(trimmed), &raw); err != nil {
		return Response{}, fmt.Errorf("reply is not one JSON object: %w", err)
	}
	for _, key := range []string{"people", "places", "organizations", "events"} {
		if _, ok := raw[key]; !ok {
			return Response{}, fmt.Errorf("missing key %q", key)
		}
	}
	decoder := json.NewDecoder(bytes.NewReader([]byte(trimmed)))
	decoder.DisallowUnknownFields()
	var response Response
	if err := decoder.Decode(&response); err != nil {
		return Response{}, fmt.Errorf("reply does not match the schema: %w", err)
	}
	labels := map[string]bool{}
	for i := range batch.Messages {
		labels[messageLabel(i)] = true
	}
	participants := map[string]bool{}
	for _, participant := range batch.Participants {
		participants[participant.Label] = true
	}
	checkEntities := func(kind string, list []EntityOut, allowParticipant bool) error {
		if list == nil {
			return fmt.Errorf("%s must be an array", kind)
		}
		for i, item := range list {
			if strings.TrimSpace(item.Name) == "" || len([]rune(item.Name)) > entities.MaxNameRunes {
				return fmt.Errorf("%s[%d].name must be 1-200 characters", kind, i)
			}
			if item.Aliases == nil || item.Mentions == nil {
				return fmt.Errorf("%s[%d] must carry aliases and mentions arrays", kind, i)
			}
			if item.Participant != nil {
				if !allowParticipant {
					return fmt.Errorf("%s[%d].participant must be null", kind, i)
				}
				if !participants[*item.Participant] {
					return fmt.Errorf("%s[%d].participant %q is not a participant label", kind, i, *item.Participant)
				}
			}
			for j, mention := range item.Mentions {
				if !labels[mention.Message] {
					return fmt.Errorf("%s[%d].mentions[%d].message %q is not a message label", kind, i, j, mention.Message)
				}
				if strings.TrimSpace(mention.Text) == "" {
					return fmt.Errorf("%s[%d].mentions[%d].text is empty", kind, i, j)
				}
			}
		}
		return nil
	}
	if err := checkEntities("people", response.People, true); err != nil {
		return Response{}, err
	}
	if err := checkEntities("places", response.Places, false); err != nil {
		return Response{}, err
	}
	if err := checkEntities("organizations", response.Organizations, false); err != nil {
		return Response{}, err
	}
	if response.Events == nil {
		return Response{}, errors.New("events must be an array")
	}
	for i, event := range response.Events {
		if strings.TrimSpace(event.Title) == "" || len([]rune(event.Title)) > events.MaxTitleRunes {
			return Response{}, fmt.Errorf("events[%d].title must be 1-200 characters", i)
		}
		if !labels[event.Message] {
			return Response{}, fmt.Errorf("events[%d].message %q is not a message label", i, event.Message)
		}
		if !events.ValidEventType(event.Type) {
			return Response{}, fmt.Errorf("events[%d].type %q is not an allowed type", i, event.Type)
		}
		if event.Date != nil && !dateTimePattern.MatchString(*event.Date) {
			return Response{}, fmt.Errorf("events[%d].date %q is not YYYY-MM-DD or YYYY-MM-DDTHH:MM", i, *event.Date)
		}
		if event.People == nil || event.Places == nil || event.Organizations == nil {
			return Response{}, fmt.Errorf("events[%d] must carry people, places and organizations arrays", i)
		}
	}
	return response, nil
}

func messageLabel(i int) string { return fmt.Sprintf("m%d", i+1) }

// Prompt renders the chat messages for one batch.
func Prompt(batch Batch) []Message {
	var builder strings.Builder
	builder.WriteString("Participants:\n")
	for _, participant := range batch.Participants {
		fmt.Fprintf(&builder, "%s = ", participant.Label)
		switch {
		case participant.Address.Kind == entities.AddressSelf && participant.Name != "":
			fmt.Fprintf(&builder, "%s (the owner of the device this conversation was saved from)\n", participant.Name)
		case participant.Address.Kind == entities.AddressSelf:
			builder.WriteString("the owner of the device this conversation was saved from (name not given)\n")
		case participant.Name != "":
			fmt.Fprintf(&builder, "%s, %s\n", participant.Name, entities.DisplayAddress(participant.Address))
		default:
			fmt.Fprintf(&builder, "%s (name not given)\n", entities.DisplayAddress(participant.Address))
		}
	}
	labelOf := map[string]string{}
	for _, participant := range batch.Participants {
		labelOf[string(participant.Address.Kind)+":"+participant.Address.Normalized] = participant.Label
	}
	builder.WriteString("\nMessages (label, time UTC, sender -> recipients: text):\n")
	for i, message := range batch.Messages {
		sender, recipients := "?", []string{}
		for _, addressee := range message.Participants {
			address := entities.NormalizeAddress(addressee.Identifier)
			label := labelOf[string(address.Kind)+":"+address.Normalized]
			if label == "" {
				continue
			}
			switch strings.ToLower(addressee.Role) {
			case "sender", "from":
				sender = label
			case "recipient", "to", "cc", "bcc":
				recipients = append(recipients, label)
			}
		}
		when := "unknown time"
		if message.OccurredAt != nil {
			when = message.OccurredAt.UTC().Format("2006-01-02 15:04")
		}
		body := message.Body
		if runes := []rune(body); len(runes) > MaxBodyCharsShown {
			body = string(runes[:MaxBodyCharsShown]) + " [cut]"
		}
		body = strings.ReplaceAll(body, "\n", " / ")
		fmt.Fprintf(&builder, "[%s] %s %s -> %s: %s\n", messageLabel(i), when, sender, strings.Join(recipients, ","), body)
	}
	return []Message{{Role: "system", Content: systemPrompt}, {Role: "user", Content: builder.String()}}
}

// BatchOutcome is one batch's grounded result.
type BatchOutcome struct {
	Index    int                 `json:"index"`
	Entities []entities.Proposal `json:"entities"`
	Events   []events.Proposal   `json:"events"`
	Invalid  bool                `json:"invalid"`
	Reason   string              `json:"reason,omitempty"`
	Attempts int                 `json:"attempts"`
	// RetryReasons are the validation errors of attempts that were retried.
	RetryReasons []string `json:"retry_reasons,omitempty"`
	// Modes are the thinking modes of each attempt, in order ("off", "on").
	Modes      []string `json:"modes,omitempty"`
	Ungrounded int      `json:"ungrounded"`
	FirstOrd   int64    `json:"first_ordinal"`
	LastOrd    int64    `json:"last_ordinal"`
}

// Completer is the narrow seam the extractor needs from Client.
type Completer interface {
	Complete(context.Context, []Message, CallOptions) (Completion, error)
}

// LongPromptTokens is where kimi-k3 must think: with thinking off, prompts of
// roughly 10k-43k tokens came back as a run of "!" (3 of 3); with thinking on,
// about 1 in 8 short calls came back empty (parent session probes 2026-09-25).
const LongPromptTokens = 8000

// EstimateTokens is a conservative prompt size (about 3.5 characters a token).
func EstimateTokens(messages []Message) int {
	chars := 0
	for _, message := range messages {
		chars += len([]rune(message.Content))
	}
	return chars*2/7 + 1
}

// junkReply reports a reply that is only one character repeated ("!!!!").
func junkReply(content string) bool {
	runes := []rune(strings.TrimSpace(content))
	if len(runes) < 4 {
		return false
	}
	for _, r := range runes[1:] {
		if r != runes[0] {
			return false
		}
	}
	return true
}

// ExtractBatch asks the model in the mode that suits the prompt's length,
// validates the reply, and on failure retries once: a short prompt retries
// with thinking on, a long prompt retries with thinking on again, because
// thinking off answers long prompts with junk every time (6 of 6, 2026-09-25;
// owner decision 2026-09-26, option B). A reply that fails twice yields an
// Invalid outcome and no proposals: output that failed validation is never
// written. Byline: Claude Code · Opus 5.5 · 2026-09-26
func ExtractBatch(ctx context.Context, completer Completer, modelID string, batch Batch, scope entities.RunScope) (BatchOutcome, error) {
	outcome := BatchOutcome{Index: batch.Index}
	if len(batch.Messages) > 0 {
		outcome.FirstOrd = batch.Messages[0].Ordinal
		outcome.LastOrd = batch.Messages[len(batch.Messages)-1].Ordinal
	}
	messages := Prompt(batch)
	primary := EstimateTokens(messages) > LongPromptTokens
	var failure error
	for attempt, thinking := range []bool{primary, true} {
		outcome.Attempts = attempt + 1
		mode := thinking
		outcome.Modes = append(outcome.Modes, thinkingLabel(mode))
		completion, err := completer.Complete(ctx, messages, CallOptions{Thinking: &mode})
		if err != nil {
			return outcome, err
		}
		content := completion.Content
		schemaFailure := false
		switch {
		case strings.TrimSpace(content) == "":
			failure = fmt.Errorf("empty reply (thinking %s)", thinkingLabel(mode))
		case junkReply(content):
			failure = fmt.Errorf("junk reply: one character repeated (thinking %s)", thinkingLabel(mode))
		case completion.FinishReason == "length":
			failure = fmt.Errorf("reply was cut off at the token limit (thinking %s)", thinkingLabel(mode))
		default:
			var response Response
			response, failure = Decode(content, batch)
			if failure == nil {
				ground(&outcome, response, batch, modelID, scope)
				return outcome, nil
			}
			schemaFailure = true
		}
		outcome.RetryReasons = append(outcome.RetryReasons, failure.Error())
		if schemaFailure && attempt == 0 {
			messages = append(messages,
				Message{Role: "assistant", Content: truncate(content, 2000)},
				Message{Role: "user", Content: "That reply did not match the schema: " + failure.Error() + ". Reply again with ONE JSON object exactly matching the schema, and nothing else."},
			)
		}
	}
	outcome.Invalid = true
	outcome.Reason = failure.Error()
	return outcome, nil
}

func thinkingLabel(on bool) string {
	if on {
		return "on"
	}
	return "off"
}

func truncate(value string, limit int) string {
	if len(value) > limit {
		return value[:limit]
	}
	return value
}

// ground converts a validated reply into proposals, keeping only what the
// messages actually contain.
func ground(outcome *BatchOutcome, response Response, batch Batch, modelID string, scope entities.RunScope) {
	byLabel := map[string]entities.MessageView{}
	for i, message := range batch.Messages {
		byLabel[messageLabel(i)] = message
	}
	participantByLabel := map[string]Participant{}
	for _, participant := range batch.Participants {
		participantByLabel[participant.Label] = participant
	}
	appearsInBatch := func(text string) bool {
		for _, message := range batch.Messages {
			if _, ok := entities.FindTextOccurrence(message.Body, text); ok {
				return true
			}
		}
		return false
	}
	method := entities.ModelMethod(modelID)
	extractor := "model:" + modelID + "@" + ExtractorVersion
	convert := func(item EntityOut, registryType entities.RegistryType) (entities.Proposal, bool) {
		proposal := entities.Proposal{
			Name: strings.TrimSpace(item.Name), RegistryType: registryType, DetectedBy: entities.DetectedAuto,
			Extractors: []string{extractor}, Confidence: 0.7, GenerationID: scope.GenerationID,
			SourceVersionID: scope.SourceVersionID, PreviewHandle: scope.PreviewHandle,
			ModelBatchOrigin: []int{batch.Index},
		}
		for _, mention := range item.Mentions {
			message, ok := byLabel[mention.Message]
			if !ok {
				outcome.Ungrounded++
				continue
			}
			span, found := entities.FindTextOccurrence(message.Body, mention.Text)
			if !found {
				outcome.Ungrounded++
				continue
			}
			start, end := span.Start, span.End
			kind := "name"
			if len(entities.Tokens(mention.Text)) == 1 && len(entities.Tokens(proposal.Name)) > 1 {
				kind = "partial"
			}
			proposal.ModelMentions = append(proposal.ModelMentions, entities.Mention{
				RecordID: message.RecordID, Ordinal: message.Ordinal, OccurredAt: message.OccurredAt,
				SourceAvailableFrom: message.SourceAvailableFrom, Kind: kind, Role: entities.RoleBody,
				Surface: string([]rune(message.Body)[span.Start:span.End]), Start: &start, End: &end,
				Snippet: entities.Snippet(message.Body, span, 40), Method: method, Confidence: 0.7,
			})
			proposal.ObserveTime(message.OccurredAt)
		}
		if len(proposal.ModelMentions) == 0 && !appearsInBatch(proposal.Name) {
			outcome.Ungrounded++
			return entities.Proposal{}, false
		}
		// Prefer the spelling the messages themselves use when the model
		// lower-cased a name ("jane" -> "Jane").
		if proposal.Name == strings.ToLower(proposal.Name) {
			for _, mention := range proposal.ModelMentions {
				if entities.NameKey(mention.Surface) == entities.NameKey(proposal.Name) && mention.Surface != strings.ToLower(mention.Surface) {
					proposal.Name = mention.Surface
					break
				}
			}
		}
		for _, alias := range item.Aliases {
			alias = strings.TrimSpace(alias)
			if alias == "" || entities.NameKey(alias) == entities.NameKey(proposal.Name) || possessiveOf(alias, proposal.Name) {
				continue
			}
			if !appearsInBatch(alias) {
				outcome.Ungrounded++
				continue
			}
			kind := entities.AliasNickname
			if entities.RelateNames(alias, proposal.Name) == entities.NameSpelling {
				kind = entities.AliasMisspelling
			} else if entities.RelateNames(alias, proposal.Name) == entities.NameContained {
				kind = entities.AliasOther
			}
			proposal.AddAlias(entities.NewNameAlias(alias, kind, entities.SourceModel, 0.7))
		}
		if item.Participant != nil {
			if participant, ok := participantByLabel[*item.Participant]; ok {
				proposal.AddAlias(entities.NewAddressAlias(participant.Address, entities.SourceModel, "source:"+scope.SourceVersionID))
				if participant.Address.Kind == entities.AddressSelf {
					proposal.SourceOwner = true
				}
			}
		}
		proposal.MentionCount = len(proposal.ModelMentions)
		proposal.Normalize()
		return proposal, true
	}
	for _, item := range response.People {
		if proposal, ok := convert(item, "person"); ok {
			outcome.Entities = append(outcome.Entities, proposal)
		}
	}
	for _, item := range response.Places {
		if proposal, ok := convert(item, "location"); ok {
			outcome.Entities = append(outcome.Entities, proposal)
		}
	}
	for _, item := range response.Organizations {
		if proposal, ok := convert(item, "org"); ok {
			outcome.Entities = append(outcome.Entities, proposal)
		}
	}
	for _, item := range response.Events {
		message, ok := byLabel[item.Message]
		if !ok {
			outcome.Ungrounded++
			continue
		}
		proposal := events.Proposal{
			Title: relabel(item.Title, participantByLabel), Description: relabel(item.Description, participantByLabel), EventType: item.Type,
			DetectedBy: entities.DetectedAuto, Extractors: []string{extractor}, Confidence: 0.6,
			GenerationID: scope.GenerationID, SourceVersionID: scope.SourceVersionID, PreviewHandle: scope.PreviewHandle,
			SourceRecords: []events.SourceRecord{{
				RecordID: message.RecordID, Ordinal: message.Ordinal, OccurredAt: message.OccurredAt,
				SourceAvailableFrom: message.SourceAvailableFrom, Snippet: truncateRunes(message.Body, 200),
			}},
		}
		if item.When != nil {
			proposal.WhenStated = strings.TrimSpace(*item.When)
		}
		if at, precision, ok := statedTime(item.Date); ok {
			proposal.OccurredAt, proposal.TemporalPrecision, proposal.TemporalConfidence = &at, precision, 0.8
		} else if message.OccurredAt != nil {
			// No stated calendar date: the event is dated by the message that
			// discusses it, and marked uncertain so the owner sees it.
			value := *message.OccurredAt
			proposal.OccurredAt, proposal.TemporalPrecision, proposal.TemporalConfidence = &value, events.PrecisionUncertain, 0.3
		}
		// Models file places under people and the reverse, so an event names
		// its entities by type-agnostic name keys; resolution finds whichever
		// proposal carries that name.
		for _, list := range [][]string{item.People, item.Places, item.Organizations} {
			for _, name := range list {
				if entities.NameKey(name) == "" {
					continue
				}
				proposal.EntityNames = append(proposal.EntityNames, name)
				proposal.EntityKeys = append(proposal.EntityKeys, entities.AnyNameKey(name))
			}
		}
		proposal.Normalize()
		outcome.Events = append(outcome.Events, proposal)
	}
	sort.SliceStable(outcome.Entities, func(i, j int) bool { return outcome.Entities[i].Name < outcome.Entities[j].Name })
}

var participantLabel = regexp.MustCompile(`\bP([0-9]+)\b`)

// relabel replaces participant labels the model leaked into prose with the
// participant's name, or a plain description when no name is known.
func relabel(text string, legend map[string]Participant) string {
	return participantLabel.ReplaceAllStringFunc(text, func(label string) string {
		participant, ok := legend[label]
		switch {
		case !ok:
			return label
		case participant.Name != "":
			return participant.Name
		case participant.Address.Kind == entities.AddressSelf:
			return "the device owner"
		default:
			return entities.DisplayAddress(participant.Address)
		}
	})
}

// possessiveOf reports whether alias is just name + "'s".
func possessiveOf(alias, name string) bool {
	lower := strings.ToLower(strings.TrimSpace(alias))
	for _, suffix := range []string{"'s", "’s", "s'"} {
		if trimmed, ok := strings.CutSuffix(lower, suffix); ok && entities.NameKey(trimmed) == entities.NameKey(name) {
			return true
		}
	}
	return false
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) > limit {
		return string(runes[:limit]) + "…"
	}
	return value
}

// statedTime parses a stated calendar date. The conversation's time zone is
// not known, so the stated wall time is recorded as given (UTC) and the
// precision says how much was stated.
func statedTime(value *string) (time.Time, string, bool) {
	if value == nil {
		return time.Time{}, "", false
	}
	if at, err := time.Parse("2006-01-02T15:04", *value); err == nil {
		return at.UTC(), events.PrecisionPoint, true
	}
	if at, err := time.Parse("2006-01-02", *value); err == nil {
		return at.UTC(), events.PrecisionUncertain, true
	}
	return time.Time{}, "", false
}

// Byline: Claude Code · Opus 5.5 · 2026-10-01
//
// Package firstparty plans the first-party context import (D04): how one
// verified normalized generation of message records becomes working.* spine
// rows (normalized_record, message_projection_route, message,
// message_participant) and first-party context thread rows
// (working.first_party_context_thread and its version / membership / source
// assertion rows).
//
// It is pure: no SQL, no clock, no identity constants. The Activities in
// engine/activities compute a Plan from Store-provided records, record its
// digest when they PROPOSE it (before the owner's preview decision), rebuild
// and compare it when they CONFIRM it (after the decision) and again before
// each COMMIT, so the rows written are provably the rows the owner was shown.
// The PostgreSQL Store in engine/postgres owns every transaction.
//
// Ids are copied, never minted here: a working.normalized_record id IS the
// context.normalized_record_identity id it projects, and the working.message
// id IS that same id (DF-04: message_id_fkey requires it). Nothing is derived
// from a path, a platform key or a default owner: identity arrives explicitly
// and is checked against the registry by the Store.
package firstparty

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Cursedpotential/probata/engine/contextthread"
	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
)

const (
	// DeriverVersion stamps every working.* row this import writes.
	DeriverVersion = "engine:first_party_context@1.0.0"
	// ClassifierID / ClassifierVersion name how messages were grouped into a
	// thread: by the platform's own conversation identity (the normalized
	// participant set), which is deterministic, hence confidence 1.
	ClassifierID      = "first_party_platform_conversation"
	ClassifierVersion = "1.0.0"
	// MetadataExtractorID / Version name the producer of a source assertion.
	MetadataExtractorID      = "first_party_context_projector"
	MetadataExtractorVersion = "1.0.0"
	// AssertedBy is the asserter recorded on every source assertion.
	AssertedBy = "engine:first_party_context_projector@1.0.0"
	// SelfMarker is the decoder's name for the device owner -- the perspective
	// person. It is kept exactly as the source states it.
	SelfMarker = "self"
	// MaxMessages bounds one generation's plan, which is held in memory. A
	// derived SMS chunk is size-capped far below this.
	MaxMessages = 200000
)

// Disclosure tiers written to working.normalized_record.disclosure_tier.
const (
	DisclosureContemporaneous = "contemporaneous"
	DisclosureDiscovered      = "discovered"
)

// SourceMessage is one normalized message record as the Store reads it from
// context.normalized_record_identity.
type SourceMessage struct {
	RecordID   string
	Ordinal    int64
	OccurredAt *time.Time
	Sender     string
	Recipients []string
	// Parties is every identifier the record names, any role; it is the input
	// to the conversation key.
	Parties []string
	Body    string
}

// Source is the provenance of the generation being imported. Platform,
// CaptureKind and RepresentationKind come from a registered derivation
// (see PlatformForDerivation), never from a guess.
type Source struct {
	SourceVersionID        string
	NormalizedGenerationID string
	DeclaredFormat         string
	SourceKey              string
	Platform               string
	CaptureKind            string
	RepresentationKind     string
	// DerivedFromSourceVersionID is the source version whose derivation
	// produced this one, when there is one.
	DerivedFromSourceVersionID string
}

// Message is one planned spine row set.
type Message struct {
	RecordID      string     `json:"record_id"`
	Ordinal       int64      `json:"ordinal"`
	OccurredAt    *time.Time `json:"occurred_at,omitempty"`
	Sender        string     `json:"sender"`
	Recipients    []string   `json:"recipients"`
	Body          string     `json:"-"`
	ContentSHA256 string     `json:"content_sha256"`
	Direction     string     `json:"direction"`
	// RouteApproved mirrors the governed projection rule: a first-party route
	// is approved when the source itself names a sender and at least one
	// recipient; otherwise it stays proposed for source-party review.
	RouteApproved bool `json:"route_approved"`
}

// Conversation is one platform conversation within the generation.
type Conversation struct {
	Key       string    `json:"key"`
	ScopedKey string    `json:"scoped_key"`
	Parties   []string  `json:"parties"`
	Messages  []Message `json:"messages"`
}

// Plan is the whole import of one generation.
type Plan struct {
	Identity       contextthread.Identity `json:"identity"`
	Source         Source                 `json:"source"`
	DisclosureTier string                 `json:"disclosure_tier"`
	Conversations  []Conversation         `json:"conversations"`
	MessageCount   int                    `json:"message_count"`
	// Digest is sha256 over the canonical JSON of everything above, plus each
	// message body's sha256 -- the value the owner's decision is bound to.
	Digest string `json:"-"`
}

// Build plans one generation. It refuses an incomplete identity, an
// unregistered platform, and a record that cannot be projected.
func Build(identity contextthread.Identity, source Source, records []SourceMessage) (Plan, error) {
	if err := identity.Validate(); err != nil {
		return Plan{}, err
	}
	for name, value := range map[string]string{
		"source version": source.SourceVersionID, "normalized generation": source.NormalizedGenerationID,
	} {
		if _, err := uuid.Parse(strings.TrimSpace(value)); err != nil {
			return Plan{}, fmt.Errorf("first-party context plan requires a %s id: %q", name, value)
		}
	}
	for name, value := range map[string]string{
		"platform": source.Platform, "capture kind": source.CaptureKind,
		"representation kind": source.RepresentationKind, "declared format": source.DeclaredFormat,
	} {
		if strings.TrimSpace(value) == "" {
			return Plan{}, fmt.Errorf("first-party context plan requires a %s", name)
		}
	}
	if len(records) == 0 {
		return Plan{}, errors.New("first-party context plan requires at least one message record")
	}
	if len(records) > MaxMessages {
		return Plan{}, fmt.Errorf("generation holds %d message records, above the %d bound of one plan", len(records), MaxMessages)
	}

	ordered := make([]SourceMessage, len(records))
	copy(ordered, records)
	sort.SliceStable(ordered, func(a, b int) bool { return ordered[a].Ordinal < ordered[b].Ordinal })

	byKey := map[string]*Conversation{}
	keys := []string{}
	seen := map[string]bool{}
	for _, record := range ordered {
		id := strings.TrimSpace(record.RecordID)
		if _, err := uuid.Parse(id); err != nil {
			return Plan{}, fmt.Errorf("message record id %q is not a uuid", record.RecordID)
		}
		if seen[id] {
			return Plan{}, fmt.Errorf("message record %s appears twice in one generation", id)
		}
		seen[id] = true
		key, parties := smsthreads.ConversationKey(record.Parties)
		if key == "unknown" {
			key, parties = smsthreads.ConversationKey(append([]string{record.Sender}, record.Recipients...))
		}
		if key == "unknown" {
			return Plan{}, fmt.Errorf("message record %s names no party other than the device owner; its conversation cannot be identified", id)
		}
		conversation, ok := byKey[key]
		if !ok {
			conversation = &Conversation{
				Key: key, ScopedKey: ScopedKey(identity, source.Platform, key), Parties: parties,
			}
			byKey[key] = conversation
			keys = append(keys, key)
		}
		conversation.Messages = append(conversation.Messages, planMessage(id, record))
	}
	sort.Strings(keys)

	plan := Plan{
		Identity: identity, Source: source, DisclosureTier: DisclosureTierFor(identity),
		MessageCount: len(ordered), Conversations: make([]Conversation, 0, len(keys)),
	}
	for _, key := range keys {
		plan.Conversations = append(plan.Conversations, *byKey[key])
	}
	digest, err := plan.computeDigest()
	if err != nil {
		return Plan{}, err
	}
	plan.Digest = digest
	return plan, nil
}

func planMessage(id string, record SourceMessage) Message {
	sender := strings.TrimSpace(record.Sender)
	recipients := uniqueTrimmed(record.Recipients)
	sum := sha256.Sum256([]byte(record.Body))
	direction := "unknown"
	switch {
	case strings.EqualFold(sender, SelfMarker):
		direction = "outbound"
	case containsFold(recipients, SelfMarker):
		direction = "inbound"
	}
	var occurred *time.Time
	if record.OccurredAt != nil {
		at := record.OccurredAt.UTC()
		occurred = &at
	}
	return Message{
		RecordID: id, Ordinal: record.Ordinal, OccurredAt: occurred, Sender: sender, Recipients: recipients,
		Body: record.Body, ContentSHA256: hex.EncodeToString(sum[:]), Direction: direction,
		RouteApproved: sender != "" && len(recipients) > 0,
	}
}

// ScopedKey is the conversation's identity across runs: one thread per
// (matter, court case, owner, perspective, platform, conversation key). It is
// stored in working.normalized_record.conversation_id so a later chunk of the
// same conversation finds the thread an earlier chunk created.
func ScopedKey(identity contextthread.Identity, platform, key string) string {
	return strings.Join([]string{
		"first_party", identity.MatterID, identity.CourtCaseID, identity.OwnerPersonID,
		identity.PerspectivePersonID, platform, key,
	}, "/")
}

// DisclosureTierFor is the rule this import writes, recorded so it can be
// vetoed: a record from the owner's own device was available to him as it
// happened (contemporaneous); a record from anyone else's device reached him
// only when the source was acquired (discovered).
func DisclosureTierFor(identity contextthread.Identity) string {
	if strings.EqualFold(identity.PerspectivePersonID, identity.OwnerPersonID) {
		return DisclosureContemporaneous
	}
	return DisclosureDiscovered
}

// DisclosureBasis explains DisclosureTierFor on the row it was applied to.
func DisclosureBasis(identity contextthread.Identity) string {
	if DisclosureTierFor(identity) == DisclosureContemporaneous {
		return "source is the owner's own device or export"
	}
	return "source is another person's device or export; known to the owner from acquisition"
}

// MembershipDigest is a thread version's assertion digest: sha256 over the
// classifier identity and the ordered membership. Creating a version and
// extending one in place compute it the same way.
func MembershipDigest(orderedMessageIDs []string) []byte {
	hash := sha256.New()
	_, _ = hash.Write([]byte(ClassifierID + "\n" + ClassifierVersion))
	for _, id := range orderedMessageIDs {
		_, _ = hash.Write([]byte("\n" + strings.ToLower(strings.TrimSpace(id))))
	}
	return hash.Sum(nil)
}

// ProvenanceDigest is a source assertion's provenance digest: sha256 over the
// source version, the generation and the conversation's ordered message ids.
func ProvenanceDigest(source Source, conversation Conversation) []byte {
	hash := sha256.New()
	_, _ = hash.Write([]byte(source.SourceVersionID + "\n" + source.NormalizedGenerationID + "\n" + conversation.ScopedKey))
	for _, message := range conversation.Messages {
		_, _ = hash.Write([]byte("\n" + message.RecordID))
	}
	return hash.Sum(nil)
}

// Coverage is the occurred_at span of a conversation's messages in this
// generation; unknown clocks are ignored as SQL min/max ignore NULL.
func (c Conversation) Coverage() (*time.Time, *time.Time) {
	var first, last *time.Time
	for _, message := range c.Messages {
		if message.OccurredAt == nil {
			continue
		}
		at := *message.OccurredAt
		if first == nil || at.Before(*first) {
			first = &at
		}
		if last == nil || at.After(*last) {
			last = &at
		}
	}
	return first, last
}

// SourceAssertion is this generation's assertion about one conversation.
func (p Plan) SourceAssertion(conversation Conversation, anchorOrdinal int64) contextthread.SourceAssertion {
	first, last := conversation.Coverage()
	count := int64(len(conversation.Messages))
	raw, _ := json.Marshal(map[string]any{
		"source_key": p.Source.SourceKey, "declared_format": p.Source.DeclaredFormat,
		"normalized_generation_id":       p.Source.NormalizedGenerationID,
		"derived_from_source_version_id": p.Source.DerivedFromSourceVersionID,
		"conversation_parties":           conversation.Parties, "plan_digest": p.Digest,
	})
	return contextthread.SourceAssertion{
		SourceVersionID: p.Source.SourceVersionID, AnchorOrdinal: anchorOrdinal,
		Platform: p.Source.Platform, PlatformConversationKey: conversation.Key,
		RepresentationKind: p.Source.RepresentationKind, CaptureKind: p.Source.CaptureKind,
		DeclaredFormat:          p.Source.DeclaredFormat,
		CoverageFirstOccurredAt: first, CoverageLastOccurredAt: last, CoverageMessageCount: &count,
		// Required only when its coverage end is known; a clockless source
		// would need a primary_fallback anchor, which this import never invents.
		RequiredForHorizon: last != nil,
		MetadataClockKind:  "other", MetadataClockBasis: "message_occurred_at_bounds",
		MetadataReviewState: "unreviewed", RawMetadata: raw,
		MetadataExtractorID: MetadataExtractorID, MetadataExtractorVersion: MetadataExtractorVersion,
		AssertionVersion: 1, Confidence: 1, ReviewState: contextthread.ReviewProposed,
		ProvenanceDigest: ProvenanceDigest(p.Source, conversation), AssertedBy: AssertedBy,
	}
}

// Member is a message's membership row at a thread ordinal. A message with no
// known clock is a member but is not required for the horizon.
func Member(message Message, ordinal int64) contextthread.Member {
	return contextthread.Member{
		MessageID: message.RecordID, Ordinal: ordinal, OccurredAt: message.OccurredAt,
		MembershipConfidence: 1, RequiredForHorizon: message.OccurredAt != nil,
	}
}

// NewThreadVersion is version 1 of a thread first seen in this generation.
func (p Plan) NewThreadVersion(threadID string, conversation Conversation) contextthread.VersionCommit {
	members := make([]contextthread.Member, 0, len(conversation.Messages))
	ids := make([]string, 0, len(conversation.Messages))
	for index, message := range conversation.Messages {
		members = append(members, Member(message, int64(index)))
		ids = append(ids, message.RecordID)
	}
	return contextthread.VersionCommit{
		ContextThreadID: threadID, Identity: p.Identity, VersionOrdinal: 1,
		ClassifierID: ClassifierID, ClassifierVersion: ClassifierVersion,
		AssertionDigest: MembershipDigest(ids), Confidence: 1, ReviewState: contextthread.ReviewProposed,
		Members: members, Sources: []contextthread.SourceAssertion{p.SourceAssertion(conversation, 0)},
	}
}

// Rebuilt reports whether an independently rebuilt plan is the plan whose
// digest was recorded.
func (p Plan) Rebuilt(recordedDigest string) error {
	if strings.TrimSpace(recordedDigest) == "" {
		return errors.New("no recorded first-party context plan digest to compare against")
	}
	if p.Digest != recordedDigest {
		return fmt.Errorf("first-party context plan changed since it was recorded: digest %s, recorded %s", p.Digest, recordedDigest)
	}
	return nil
}

type digestMessage struct {
	Message
	BodySHA256 string `json:"body_sha256"`
}

type digestConversation struct {
	Key       string          `json:"key"`
	ScopedKey string          `json:"scoped_key"`
	Parties   []string        `json:"parties"`
	Messages  []digestMessage `json:"messages"`
}

func (p Plan) computeDigest() (string, error) {
	conversations := make([]digestConversation, 0, len(p.Conversations))
	for _, conversation := range p.Conversations {
		messages := make([]digestMessage, 0, len(conversation.Messages))
		for _, message := range conversation.Messages {
			messages = append(messages, digestMessage{Message: message, BodySHA256: message.ContentSHA256})
		}
		conversations = append(conversations, digestConversation{
			Key: conversation.Key, ScopedKey: conversation.ScopedKey, Parties: conversation.Parties, Messages: messages,
		})
	}
	encoded, err := json.Marshal(struct {
		Contract       string                 `json:"contract"`
		Identity       contextthread.Identity `json:"identity"`
		Source         Source                 `json:"source"`
		DisclosureTier string                 `json:"disclosure_tier"`
		Conversations  []digestConversation   `json:"conversations"`
	}{DeriverVersion, p.Identity, p.Source, p.DisclosureTier, conversations})
	if err != nil {
		return "", fmt.Errorf("encode first-party context plan: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

// PlatformForDerivation is the registry of derivations whose output this
// import may project, keyed by the derive Activity's handler. A generation
// that holds messages but traces to no registered derivation is refused: its
// platform would otherwise have to be guessed.
func PlatformForDerivation(handlerID, declaredFormat string) (platform, captureKind, representationKind string, ok bool) {
	if handlerID == "smsthreads_derive" && strings.TrimSpace(declaredFormat) == "ndjson" {
		return "sms", "sms_backup_restore", "json", true
	}
	return "", "", "", false
}

func uniqueTrimmed(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || seen[strings.ToLower(trimmed)] {
			continue
		}
		seen[strings.ToLower(trimmed)] = true
		out = append(out, trimmed)
	}
	return out
}

func containsFold(values []string, want string) bool {
	for _, value := range values {
		if strings.EqualFold(value, want) {
			return true
		}
	}
	return false
}

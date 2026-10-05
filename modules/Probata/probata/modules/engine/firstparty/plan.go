// Byline: Claude Code · Opus 5.5 · 2026-10-01
// Byline: Claude Code · Opus 5.5 · 2026-10-02 (split by owner participation; one disclosure rule)
//
// Package firstparty plans the context import (D04): how one verified
// normalized generation of message records becomes working.* rows.
//
// Every message is split by OWNER PARTICIPATION (owner, 2026-10-02), decided
// against the recorded participant resolution:
//
//   - the owner is a stated sender or recipient: first-party context --
//     working.normalized_record, message_projection_route, message and
//     message_participant, plus the working.first_party_context_thread family;
//   - he is not: acquired third-party material -- working.normalized_record
//     (message_corpus acquired_third_party), a PROPOSED acquired_third_party
//     route, working.third_party_conversation / third_party_message /
//     third_party_message_participant. Third-party context threads need an
//     evidence.acquisition and are left to promotion.
//
// Every message's disclosure tier comes from the one disclosure rule.
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

	"github.com/Cursedpotential/probata/engine/contextsearch"
	"github.com/Cursedpotential/probata/engine/contextthread"
	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
	"github.com/Cursedpotential/probata/engine/disclosure"
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
	SelfMarker = disclosure.SelfIdentifier
	// MaxMessages bounds one generation's plan, which is held in memory. A
	// derived SMS chunk is size-capped far below this.
	MaxMessages = 200000
)

// Corpora a message is split into (working.normalized_record.message_corpus).
const (
	CorpusFirstParty = "first_party"
	CorpusThirdParty = "acquired_third_party"
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

// AcquiredSourceVersionID is the source version a third-party conversation is
// filed under: the backup that was acquired, so every chunk derived from one
// backup files its conversations in one place.
func (s Source) AcquiredSourceVersionID() string {
	if strings.TrimSpace(s.DerivedFromSourceVersionID) != "" {
		return s.DerivedFromSourceVersionID
	}
	return s.SourceVersionID
}

// Participant is one stated party of a message, as the registry resolved it.
type Participant struct {
	Raw  string `json:"raw"`
	Role string `json:"role"`
	// EntityID is the registry entity this identifier is confirmed for; empty
	// when unresolved (the Case identity page's unknowns queue reads those).
	EntityID string `json:"entity_id,omitempty"`
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
	// OwnerTookPart, DisclosureTier and DisclosureBasis are the disclosure
	// rule's answer for this message.
	OwnerTookPart   bool          `json:"owner_took_part"`
	DisclosureTier  string        `json:"disclosure_tier"`
	DisclosureBasis string        `json:"disclosure_basis"`
	Participants    []Participant `json:"participants"`
	// RouteApproved mirrors the governed projection rule: a first-party route
	// is approved when the source itself names a sender and at least one
	// recipient; otherwise it stays proposed for source-party review.
	RouteApproved bool `json:"route_approved"`
}

// Conversation is one platform conversation within the generation, in one corpus.
type Conversation struct {
	Corpus    string    `json:"corpus"`
	Key       string    `json:"key"`
	ScopedKey string    `json:"scoped_key"`
	Parties   []string  `json:"parties"`
	Messages  []Message `json:"messages"`
}

// Plan is the whole import of one generation.
type Plan struct {
	Identity      contextthread.Identity `json:"identity"`
	Source        Source                 `json:"source"`
	Resolution    disclosure.Resolution  `json:"-"`
	Conversations []Conversation         `json:"conversations"`
	MessageCount  int                    `json:"message_count"`
	// Digest is sha256 over the canonical JSON of everything above, plus each
	// message body's sha256 -- the value the owner's decision is bound to.
	Digest string `json:"-"`
}

// Build plans one generation against its recorded participant resolution.
// It refuses an incomplete identity, a resolution made for other people, an
// unregistered platform, AI conversation provenance, and a record that cannot be projected.
// Inputs: verified source, normalized messages, explicit identity and recorded participant resolution.
// Outputs: a deterministic messaging plan or validation error. Side effects: none.
// Pick for human messaging projections; AI conversations retain their separate context-search route.
func Build(identity contextthread.Identity, source Source, records []SourceMessage, resolution disclosure.Resolution) (Plan, error) {
	if contextsearch.IsAIChatFormat(source.DeclaredFormat) {
		return Plan{}, errors.New("AI chat sources remain AI context and are not first-party messaging")
	}
	if err := identity.Validate(); err != nil {
		return Plan{}, err
	}
	if err := resolution.Validate(); err != nil {
		return Plan{}, err
	}
	if !strings.EqualFold(resolution.OwnerPersonID, identity.OwnerPersonID) ||
		!strings.EqualFold(resolution.PerspectivePersonID, identity.PerspectivePersonID) {
		return Plan{}, errors.New("the participant resolution was made for a different owner or perspective person")
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
		message, err := planMessage(id, record, &resolution)
		if err != nil {
			return Plan{}, fmt.Errorf("message record %s: %w", id, err)
		}
		corpus := CorpusThirdParty
		if message.OwnerTookPart {
			corpus = CorpusFirstParty
		}
		grouping := corpus + "/" + key
		conversation, ok := byKey[grouping]
		if !ok {
			conversation = &Conversation{
				Corpus: corpus, Key: key, ScopedKey: ScopedKey(corpus, identity, source.Platform, key), Parties: parties,
			}
			byKey[grouping] = conversation
			keys = append(keys, grouping)
		}
		conversation.Messages = append(conversation.Messages, message)
	}
	sort.Strings(keys)

	plan := Plan{
		Identity: identity, Source: source, Resolution: resolution,
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

func planMessage(id string, record SourceMessage, resolution *disclosure.Resolution) (Message, error) {
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
	took, tier, basis, err := resolution.ForMessage(sender, recipients)
	if err != nil {
		return Message{}, err
	}
	participants := make([]Participant, 0, len(recipients)+1)
	add := func(raw, role string) error {
		// The device owner's own marker is the perspective person.
		if strings.EqualFold(strings.TrimSpace(raw), SelfMarker) {
			participants = append(participants, Participant{Raw: raw, Role: role, EntityID: resolution.PerspectivePersonID})
			return nil
		}
		identifier, ok := resolution.Lookup(raw)
		if !ok {
			return fmt.Errorf("identifier %q is not in the participant resolution", raw)
		}
		participants = append(participants, Participant{Raw: raw, Role: role, EntityID: identifier.EntityID})
		return nil
	}
	if sender != "" {
		if err := add(sender, "from"); err != nil {
			return Message{}, err
		}
	}
	for _, recipient := range recipients {
		if err := add(recipient, "to"); err != nil {
			return Message{}, err
		}
	}
	return Message{
		RecordID: id, Ordinal: record.Ordinal, OccurredAt: occurred, Sender: sender, Recipients: recipients,
		Body: record.Body, ContentSHA256: hex.EncodeToString(sum[:]), Direction: direction,
		OwnerTookPart: took, DisclosureTier: tier, DisclosureBasis: basis, Participants: participants,
		RouteApproved: took && sender != "" && len(recipients) > 0,
	}, nil
}

// ScopedKey is the conversation's identity across runs: one per (corpus,
// matter, court case, owner, perspective, platform, conversation key). It is
// stored in working.normalized_record.conversation_id so a later chunk of the
// same conversation finds the thread or conversation an earlier chunk created.
func ScopedKey(corpus string, identity contextthread.Identity, platform, key string) string {
	return strings.Join([]string{
		corpus, identity.MatterID, identity.CourtCaseID, identity.OwnerPersonID,
		identity.PerspectivePersonID, platform, key,
	}, "/")
}

// FirstParty and ThirdParty return the plan's conversations of one corpus.
func (p Plan) FirstParty() []Conversation { return p.corpus(CorpusFirstParty) }
func (p Plan) ThirdParty() []Conversation { return p.corpus(CorpusThirdParty) }

func (p Plan) corpus(corpus string) []Conversation {
	out := []Conversation{}
	for _, conversation := range p.Conversations {
		if conversation.Corpus == corpus {
			out = append(out, conversation)
		}
	}
	return out
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
	Corpus    string          `json:"corpus"`
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
			Corpus: conversation.Corpus, Key: conversation.Key, ScopedKey: conversation.ScopedKey, Parties: conversation.Parties, Messages: messages,
		})
	}
	encoded, err := json.Marshal(struct {
		Contract      string                 `json:"contract"`
		Identity      contextthread.Identity `json:"identity"`
		Source        Source                 `json:"source"`
		Resolution    string                 `json:"resolution_digest"`
		Conversations []digestConversation   `json:"conversations"`
	}{DeriverVersion, p.Identity, p.Source, ResolutionDigest(p.Resolution), conversations})
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

// PlatformForDetectedFormat registers the platforms a source carries in its
// own content signature, for sources that are not derived: a Facebook
// Messenger thread file (message_N.json of a Facebook export) is detected by
// the engine's signature registry (facebook_messenger_thread_json_v1) and
// stored in context.handler_detected_format. The platform comes from that
// persisted engine decision, never from the file name.
// Byline: Claude Code · Opus 5.5 · 2026-10-02
func PlatformForDetectedFormat(detectedFormat string) (platform, captureKind, representationKind string, ok bool) {
	switch strings.TrimSpace(detectedFormat) {
	case "facebook_messenger_json":
		return "facebook_messenger", "facebook_export", "json", true
	case "facebook_messenger_html":
		// Same platform and capture as the JSON flavor; only the representation differs.
		return "facebook_messenger", "facebook_export", "html", true
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

// ResolutionDigest binds a plan to the exact recorded participant resolution.
func ResolutionDigest(resolution disclosure.Resolution) string {
	identifiers := append([]disclosure.ResolvedIdentifier(nil), resolution.Identifiers...)
	sort.Slice(identifiers, func(a, b int) bool { return identifiers[a].Raw < identifiers[b].Raw })
	encoded, _ := json.Marshal(struct {
		Basis, Owner, Perspective string
		Identifiers               []disclosure.ResolvedIdentifier
	}{resolution.Basis, strings.ToLower(resolution.OwnerPersonID), strings.ToLower(resolution.PerspectivePersonID), identifiers})
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

// Byline: Claude Code · Opus 5.5 · 2026-09-25

package entities

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// RegistryType is a value of the ai.entity_type enum (registry.entity).
type RegistryType string

// CandidateType is a value allowed by working.candidate_entity's CHECK.
type CandidateType string

const (
	CandidatePerson       CandidateType = "person"
	CandidateOrganization CandidateType = "organization"
	CandidateLocation     CandidateType = "location"
	CandidateDevice       CandidateType = "device"
	CandidateAccount      CandidateType = "account"
	CandidateOther        CandidateType = "other"
)

// registryTypes maps every ai.entity_type value to the coarse class the
// staging table accepts. The precise registry type rides in attrs.
var registryTypes = map[RegistryType]CandidateType{
	"person": CandidatePerson, "attorney": CandidatePerson, "doctor": CandidatePerson,
	"org": CandidateOrganization, "school": CandidateOrganization, "court": CandidateOrganization,
	"institution": CandidateOrganization, "platform": CandidateOrganization, "ai_system": CandidateOrganization,
	"location": CandidateLocation, "address": CandidateLocation,
	"device": CandidateDevice, "phone": CandidateDevice,
	"account": CandidateAccount, "email": CandidateAccount, "handle": CandidateAccount,
	"project": CandidateOther, "tech": CandidateOther, "concept": CandidateOther, "vehicle": CandidateOther,
}

// RegistryTypes lists every accepted registry type in a stable order.
func RegistryTypes() []RegistryType {
	out := make([]RegistryType, 0, len(registryTypes))
	for value := range registryTypes {
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// ValidRegistryType reports whether value is an ai.entity_type member.
func ValidRegistryType(value RegistryType) bool {
	_, ok := registryTypes[value]
	return ok
}

// CandidateTypeFor returns the staging class for a registry type.
func CandidateTypeFor(value RegistryType) CandidateType {
	if coarse, ok := registryTypes[value]; ok {
		return coarse
	}
	return CandidateOther
}

// AliasKind is a registry.entity_alias.alias_kind value.
type AliasKind string

const (
	AliasNickname    AliasKind = "nickname"
	AliasLegal       AliasKind = "legal"
	AliasMaiden      AliasKind = "maiden"
	AliasHandle      AliasKind = "handle"
	AliasMisspelling AliasKind = "misspelling"
	AliasPhonetic    AliasKind = "phonetic"
	AliasInitials    AliasKind = "initials"
	AliasOther       AliasKind = "other"
)

// ValidAliasKind reports whether value satisfies entity_alias_alias_kind_check.
func ValidAliasKind(value AliasKind) bool {
	switch value {
	case AliasNickname, AliasLegal, AliasMaiden, AliasHandle, AliasMisspelling, AliasPhonetic, AliasInitials, AliasOther:
		return true
	}
	return false
}

// Alias sources.
const (
	SourceParticipant = "participant"
	SourceModel       = "model"
	SourceOwner       = "owner"
	SourceRules       = "rules"
)

// Alias is one surface that names the entity.
type Alias struct {
	Text        string      `json:"text"`
	Kind        AliasKind   `json:"kind"`
	AddressKind AddressKind `json:"address_kind,omitempty"`
	Normalized  string      `json:"normalized"`
	Source      string      `json:"source"`
	Confidence  float64     `json:"confidence"`
	// Scope is set only for source-relative identities ("self"). A scoped
	// alias groups mentions inside its source and is never written to
	// registry.entity_alias.
	Scope string `json:"scope,omitempty"`
}

// IsAddress reports whether the alias is a participant identifier.
func (a Alias) IsAddress() bool { return a.AddressKind != "" }

// Key is the stable identity an alias contributes to its entity. Events
// reference entities by these keys, so a merge, split or rename never
// strands an event on a superseded proposal row.
func (a Alias) Key(coarse CandidateType) string {
	if a.IsAddress() {
		if a.Scope != "" {
			return "participant:" + a.Normalized + "@" + a.Scope
		}
		return "participant:" + a.Normalized
	}
	return "name:" + string(coarse) + ":" + NameKey(a.Text)
}

// AnyNameKey references an entity by name regardless of its type.
func AnyNameKey(name string) string { return "name:any:" + NameKey(name) }

// Covers reports whether the proposal answers to a key. A "name:any:" key
// matches a name of any type; every other key must match exactly.
func (p Proposal) Covers(key string) bool {
	if rest, ok := strings.CutPrefix(key, "name:any:"); ok {
		for _, own := range p.Keys() {
			if strings.HasPrefix(own, "name:") && strings.HasSuffix(own, ":"+rest) {
				return true
			}
		}
		return false
	}
	for _, own := range p.Keys() {
		if own == key {
			return true
		}
	}
	return false
}

// NewAddressAlias builds the alias for a participant identifier.
func NewAddressAlias(address Address, source string, scope string) Alias {
	text := address.Normalized
	if text == "" {
		text = strings.TrimSpace(address.Raw)
	}
	alias := Alias{Text: text, Kind: AliasHandle, AddressKind: address.Kind, Normalized: address.Normalized, Source: source, Confidence: 1}
	if address.Kind == AddressSelf {
		alias.Scope = scope
	}
	return alias
}

// NewNameAlias builds a name alias.
func NewNameAlias(text string, kind AliasKind, source string, confidence float64) Alias {
	return Alias{Text: strings.TrimSpace(text), Kind: kind, Normalized: NameKey(text), Source: source, Confidence: confidence}
}

// Mention roles.
const (
	RoleSender    = "sender"
	RoleRecipient = "recipient"
	RoleUnknown   = "participant"
	RoleBody      = "body"
)

// Extraction methods recorded on working.entity_mention.extraction_method.
const (
	MethodParticipant = "rules:participant@1"
	MethodBodyAlias   = "rules:body-alias@1"
)

// ModelMethod names a model extraction for the mention/extractor record.
func ModelMethod(modelID string) string { return "model:" + modelID + "@1" }

// Mention is one place a record names the entity. Start/End are rune
// offsets into the message body; header (participant) mentions have none.
type Mention struct {
	RecordID            string     `json:"record_id"`
	Ordinal             int64      `json:"ordinal"`
	OccurredAt          *time.Time `json:"occurred_at,omitempty"`
	SourceAvailableFrom *time.Time `json:"source_available_from,omitempty"`
	Kind                string     `json:"kind"`
	Role                string     `json:"role"`
	Surface             string     `json:"surface"`
	Start               *int       `json:"start,omitempty"`
	End                 *int       `json:"end,omitempty"`
	Snippet             string     `json:"snippet,omitempty"`
	Method              string     `json:"method"`
	Confidence          float64    `json:"confidence"`
}

// SpanKey identifies a mention independently of the entity it resolves to.
func (m Mention) SpanKey() string {
	start, end := -1, -1
	if m.Start != nil {
		start = *m.Start
	}
	if m.End != nil {
		end = *m.End
	}
	return fmt.Sprintf("%s|%d|%d|%s|%s|%s", m.RecordID, start, end, strings.ToLower(m.Surface), m.Kind, m.Role)
}

// Match proposes merging the entity into one already in the registry.
type Match struct {
	EntityID    string  `json:"entity_id"`
	DisplayName string  `json:"display_name"`
	Via         string  `json:"via"`
	Alias       string  `json:"alias,omitempty"`
	Score       float64 `json:"score"`
}

// Suggestion is a weaker possible match the owner may accept by hand.
type Suggestion struct {
	EntityID    string  `json:"entity_id,omitempty"`
	CandidateID string  `json:"candidate_id,omitempty"`
	DisplayName string  `json:"display_name"`
	Reason      string  `json:"reason"`
	Score       float64 `json:"score"`
}

// Flag is one small, owner-visible note on a proposal.
type Flag struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

// Actor is the authenticated person behind a correction or commit.
type Actor struct {
	SubjectUID string `json:"subject_uid"`
	Username   string `json:"username"`
}

// Correction records who changed a proposal, when, and how. It is written
// with the new proposal row; the superseded row is never edited beyond its
// review_state.
type Correction struct {
	Op            string    `json:"op"`
	Actor         Actor     `json:"actor"`
	At            time.Time `json:"at"`
	Note          string    `json:"note,omitempty"`
	RequestDigest string    `json:"request_digest,omitempty"`
}

// ParticipantStat summarizes one participant address inside the run.
type ParticipantStat struct {
	Identifier     string     `json:"identifier"`
	Address        Address    `json:"address"`
	SenderCount    int        `json:"sender_count"`
	RecipientCount int        `json:"recipient_count"`
	MessageCount   int        `json:"message_count"`
	FirstAt        *time.Time `json:"first_at,omitempty"`
	LastAt         *time.Time `json:"last_at,omitempty"`
}

// Review states of working.candidate_entity / candidate_event.
const (
	StatePending    = "pending"
	StateApproved   = "approved"
	StateRejected   = "rejected"
	StateSuperseded = "superseded"
)

// Detection provenance.
const (
	DetectedAuto  = "auto"
	DetectedOwner = "owner"
)

// Bounds that keep proposals, Temporal payloads and the UI small.
const (
	MaxMentionSample = 25
	MaxModelMentions = 500
	MaxAliases       = 64
	MaxNameRunes     = 200
)

// Proposal is one proposed entity with its aliases and supporting mentions.
type Proposal struct {
	CandidateID      string            `json:"candidate_id,omitempty"`
	ExtractionRunID  string            `json:"extraction_run_id,omitempty"`
	Name             string            `json:"name"`
	RegistryType     RegistryType      `json:"registry_type"`
	Aliases          []Alias           `json:"aliases"`
	Participants     []ParticipantStat `json:"participants,omitempty"`
	MentionCount     int               `json:"mention_count"`
	MentionSample    []Mention         `json:"mention_sample,omitempty"`
	ModelMentions    []Mention         `json:"model_mentions,omitempty"`
	Match            *Match            `json:"match,omitempty"`
	Suggestions      []Suggestion      `json:"suggestions,omitempty"`
	Confidence       float64           `json:"confidence"`
	DetectedBy       string            `json:"detected_by"`
	Extractors       []string          `json:"extractors"`
	SourceOwner      bool              `json:"source_owner,omitempty"`
	Flags            []Flag            `json:"flags,omitempty"`
	Supersedes       []string          `json:"supersedes,omitempty"`
	Correction       *Correction       `json:"correction,omitempty"`
	ReviewState      string            `json:"review_state,omitempty"`
	PromotedToID     string            `json:"promoted_to_id,omitempty"`
	FirstOccurredAt  *time.Time        `json:"first_occurred_at,omitempty"`
	LastOccurredAt   *time.Time        `json:"last_occurred_at,omitempty"`
	GenerationID     string            `json:"normalized_generation_id"`
	SourceVersionID  string            `json:"source_version_id,omitempty"`
	PreviewHandle    string            `json:"preview_handle,omitempty"`
	ModelBatchOrigin []int             `json:"model_batches,omitempty"`
}

// Coarse returns the staging class of the proposal.
func (p Proposal) Coarse() CandidateType { return CandidateTypeFor(p.RegistryType) }

// Keys is the set of identities this proposal covers, derived from its name
// and aliases.
func (p Proposal) Keys() []string {
	seen := map[string]bool{}
	var keys []string
	add := func(key string) {
		if key != "" && !seen[key] {
			seen[key] = true
			keys = append(keys, key)
		}
	}
	coarse := p.Coarse()
	if !LooksLikeAddress(p.Name) && NameKey(p.Name) != "" {
		add("name:" + string(coarse) + ":" + NameKey(p.Name))
	}
	for _, alias := range p.Aliases {
		if alias.IsAddress() || NameKey(alias.Text) != "" {
			add(alias.Key(coarse))
		}
	}
	sort.Strings(keys)
	return keys
}

// NameAliases returns the aliases that are names (not addresses), plus the
// proposal's own name, deduplicated by comparison key.
func (p Proposal) NameAliases() []Alias {
	seen := map[string]bool{}
	var out []Alias
	if !LooksLikeAddress(p.Name) && NameKey(p.Name) != "" {
		seen[NameKey(p.Name)] = true
		out = append(out, NewNameAlias(p.Name, AliasOther, SourceRules, p.Confidence))
	}
	for _, alias := range p.Aliases {
		key := NameKey(alias.Text)
		if alias.IsAddress() || key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, alias)
	}
	return out
}

// AddressAliases returns the participant identifiers of the proposal.
func (p Proposal) AddressAliases() []Alias {
	var out []Alias
	for _, alias := range p.Aliases {
		if alias.IsAddress() {
			out = append(out, alias)
		}
	}
	return out
}

// Included reports whether a proposal takes part in the next commit: every
// current proposal is included unless the owner rejected it. Review is
// pulled, not pushed (ADR-0062 §2): nothing waits for a per-row approval.
func (p Proposal) Included() bool {
	return p.ReviewState == "" || p.ReviewState == StatePending
}

// contentView is the part of a proposal that defines its content hash.
type contentView struct {
	Name          string       `json:"name"`
	RegistryType  RegistryType `json:"registry_type"`
	Aliases       []Alias      `json:"aliases"`
	Match         *Match       `json:"match,omitempty"`
	ModelMentions []Mention    `json:"model_mentions,omitempty"`
	DetectedBy    string       `json:"detected_by"`
	Extractors    []string     `json:"extractors"`
	SourceOwner   bool         `json:"source_owner,omitempty"`
	Supersedes    []string     `json:"supersedes,omitempty"`
	Correction    *Correction  `json:"correction,omitempty"`
	Generation    string       `json:"generation"`
	Flags         []Flag       `json:"flags,omitempty"`
}

// ContentSHA256 is the dedup digest stored in candidate_entity.content_sha256.
// Re-running the same extractor over the same generation reproduces it, so a
// retried Activity inserts nothing new.
func (p Proposal) ContentSHA256() [32]byte {
	aliases := append([]Alias(nil), p.Aliases...)
	sort.Slice(aliases, func(i, j int) bool {
		return aliases[i].Key(p.Coarse())+"|"+aliases[i].Text < aliases[j].Key(p.Coarse())+"|"+aliases[j].Text
	})
	extractors := append([]string(nil), p.Extractors...)
	sort.Strings(extractors)
	supersedes := append([]string(nil), p.Supersedes...)
	sort.Strings(supersedes)
	raw, _ := json.Marshal(contentView{
		Name: p.Name, RegistryType: p.RegistryType, Aliases: aliases, Match: p.Match,
		ModelMentions: p.ModelMentions, DetectedBy: p.DetectedBy, Extractors: extractors,
		SourceOwner: p.SourceOwner, Supersedes: supersedes, Correction: p.Correction,
		Generation: p.GenerationID, Flags: p.Flags,
	})
	return sha256.Sum256(raw)
}

// ContentHex is ContentSHA256 in hex.
func (p Proposal) ContentHex() string {
	digest := p.ContentSHA256()
	return hex.EncodeToString(digest[:])
}

// AddAlias appends an alias unless an alias with the same key already exists;
// a stronger kind or source replaces a weaker duplicate's metadata.
func (p *Proposal) AddAlias(alias Alias) bool {
	if strings.TrimSpace(alias.Text) == "" {
		return false
	}
	key := alias.Key(p.Coarse())
	if !alias.IsAddress() && NameKey(alias.Text) == NameKey(p.Name) {
		return false
	}
	for i, existing := range p.Aliases {
		if existing.Key(p.Coarse()) == key {
			if alias.Source == SourceOwner && existing.Source != SourceOwner {
				p.Aliases[i] = alias
				return true
			}
			if alias.Confidence > existing.Confidence {
				p.Aliases[i].Confidence = alias.Confidence
			}
			return false
		}
	}
	if len(p.Aliases) >= MaxAliases {
		p.addFlag(Flag{Code: "aliases_truncated", Detail: fmt.Sprintf("more than %d aliases; the rest were not kept", MaxAliases)})
		return false
	}
	p.Aliases = append(p.Aliases, alias)
	return true
}

// RemoveAlias drops the alias whose text matches (by comparison key).
func (p *Proposal) RemoveAlias(text string) bool {
	target := NameKey(text)
	address := NormalizeAddress(text)
	for i, alias := range p.Aliases {
		matches := (!alias.IsAddress() && NameKey(alias.Text) == target && target != "") ||
			(alias.IsAddress() && (alias.Normalized == address.Normalized || alias.Text == strings.TrimSpace(text)))
		if matches {
			p.Aliases = append(p.Aliases[:i], p.Aliases[i+1:]...)
			return true
		}
	}
	return false
}

func (p *Proposal) addFlag(flag Flag) {
	for _, existing := range p.Flags {
		if existing.Code == flag.Code {
			return
		}
	}
	p.Flags = append(p.Flags, flag)
}

// AddMentionSample keeps a bounded, ordered sample of supporting mentions.
func (p *Proposal) AddMentionSample(mentions ...Mention) {
	seen := map[string]bool{}
	for _, existing := range p.MentionSample {
		seen[existing.SpanKey()] = true
	}
	for _, mention := range mentions {
		if seen[mention.SpanKey()] {
			continue
		}
		seen[mention.SpanKey()] = true
		p.MentionSample = append(p.MentionSample, mention)
	}
	sort.SliceStable(p.MentionSample, func(i, j int) bool { return p.MentionSample[i].Ordinal < p.MentionSample[j].Ordinal })
	if len(p.MentionSample) > MaxMentionSample {
		p.MentionSample = p.MentionSample[:MaxMentionSample]
	}
}

// ObserveTime widens the proposal's first/last occurrence window. These stay
// on the staging row for the reviewer only; they are never written to the
// registry entity, where an unfiltered last-seen date would leak hindsight.
func (p *Proposal) ObserveTime(at *time.Time) {
	if at == nil {
		return
	}
	if p.FirstOccurredAt == nil || at.Before(*p.FirstOccurredAt) {
		value := *at
		p.FirstOccurredAt = &value
	}
	if p.LastOccurredAt == nil || at.After(*p.LastOccurredAt) {
		value := *at
		p.LastOccurredAt = &value
	}
}

// Normalize trims the proposal into its canonical, bounded shape.
func (p *Proposal) Normalize() {
	p.Name = strings.TrimSpace(p.Name)
	if runes := []rune(p.Name); len(runes) > MaxNameRunes {
		p.Name = string(runes[:MaxNameRunes])
	}
	if !ValidRegistryType(p.RegistryType) {
		p.RegistryType = "person"
	}
	sort.SliceStable(p.Aliases, func(i, j int) bool {
		if p.Aliases[i].IsAddress() != p.Aliases[j].IsAddress() {
			return p.Aliases[i].IsAddress()
		}
		return strings.ToLower(p.Aliases[i].Text) < strings.ToLower(p.Aliases[j].Text)
	})
	sort.Strings(p.Extractors)
	p.Extractors = compactStrings(p.Extractors)
	if len(p.ModelMentions) > MaxModelMentions {
		p.ModelMentions = p.ModelMentions[:MaxModelMentions]
		p.addFlag(Flag{Code: "model_mentions_truncated", Detail: fmt.Sprintf("more than %d model mentions; the rest were not kept", MaxModelMentions)})
	}
	if p.Confidence < 0 {
		p.Confidence = 0
	}
	if p.Confidence > 1 {
		p.Confidence = 1
	}
	if p.DetectedBy == "" {
		p.DetectedBy = DetectedAuto
	}
}

func compactStrings(values []string) []string {
	out := values[:0]
	var previous string
	for i, value := range values {
		if value == "" || (i > 0 && value == previous) {
			continue
		}
		out = append(out, value)
		previous = value
	}
	return out
}

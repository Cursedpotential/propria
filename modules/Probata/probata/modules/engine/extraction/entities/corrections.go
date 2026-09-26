// Byline: Claude Code · Opus 5.5 · 2026-09-25

package entities

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Correction operations the Review panel sends.
const (
	OpReject      = "reject"
	OpRestore     = "restore"
	OpRename      = "rename"
	OpRetype      = "retype"
	OpAddAlias    = "add_alias"
	OpRemoveAlias = "remove_alias"
	OpMerge       = "merge"
	OpSplit       = "split"
	OpSetMatch    = "set_match"
	OpClearMatch  = "clear_match"
)

// ErrConflict means the proposal changed since the owner loaded it.
var ErrConflict = errors.New("proposal changed since it was loaded; reload and try again")

// CorrectionRequest is one owner edit.
type CorrectionRequest struct {
	Op           string       `json:"op"`
	CandidateIDs []string     `json:"candidate_ids"`
	Name         string       `json:"name,omitempty"`
	RegistryType RegistryType `json:"registry_type,omitempty"`
	AliasText    string       `json:"alias_text,omitempty"`
	AliasKind    AliasKind    `json:"alias_kind,omitempty"`
	SplitAliases []string     `json:"split_aliases,omitempty"`
	MatchEntity  *Match       `json:"match,omitempty"`
	Note         string       `json:"note,omitempty"`
}

// CorrectionResult is what a correction writes: new rows (each superseding
// the proposals named in its Supersedes) and the ids to mark superseded.
type CorrectionResult struct {
	Insert    []Proposal `json:"insert"`
	Supersede []string   `json:"supersede"`
}

// ApplyCorrection applies one owner edit to the current proposals it names.
// It is pure: current carries the rows as loaded, actor/at/digest stamp the
// new rows. A proposal already committed cannot be edited here; a rejected
// one can only be restored.
func ApplyCorrection(current map[string]Proposal, request CorrectionRequest, actor Actor, at time.Time, digest string) (CorrectionResult, error) {
	if actor.SubjectUID == "" || actor.Username == "" {
		return CorrectionResult{}, errors.New("an authenticated actor is required")
	}
	if len(request.CandidateIDs) == 0 {
		return CorrectionResult{}, errors.New("candidate_ids is required")
	}
	var bases []Proposal
	for _, id := range request.CandidateIDs {
		base, ok := current[id]
		if !ok {
			return CorrectionResult{}, fmt.Errorf("proposal %s is not current: %w", id, ErrConflict)
		}
		bases = append(bases, base)
	}
	stamp := func(p Proposal, supersedes ...string) Proposal {
		p.CandidateID = ""
		p.PromotedToID = ""
		p.Supersedes = append([]string(nil), supersedes...)
		p.Correction = &Correction{Op: request.Op, Actor: actor, At: at.UTC(), Note: strings.TrimSpace(request.Note), RequestDigest: digest}
		p.Normalize()
		return p
	}
	requirePending := func(p Proposal) error {
		switch p.ReviewState {
		case StatePending, "":
			return nil
		case StateApproved:
			return fmt.Errorf("%q is already committed; correct the committed entity instead", p.Name)
		case StateRejected:
			return fmt.Errorf("%q is rejected; restore it first", p.Name)
		default:
			return fmt.Errorf("%q is %s: %w", p.Name, p.ReviewState, ErrConflict)
		}
	}
	one := func() (Proposal, error) {
		if len(bases) != 1 {
			return Proposal{}, fmt.Errorf("%s takes exactly one proposal", request.Op)
		}
		return bases[0], nil
	}
	switch request.Op {
	case OpReject:
		base, err := one()
		if err != nil {
			return CorrectionResult{}, err
		}
		if err := requirePending(base); err != nil {
			return CorrectionResult{}, err
		}
		next := stamp(clone(base), base.CandidateID)
		next.ReviewState = StateRejected
		return CorrectionResult{Insert: []Proposal{next}, Supersede: []string{base.CandidateID}}, nil
	case OpRestore:
		base, err := one()
		if err != nil {
			return CorrectionResult{}, err
		}
		if base.ReviewState != StateRejected {
			return CorrectionResult{}, fmt.Errorf("%q is not rejected", base.Name)
		}
		next := stamp(clone(base), base.CandidateID)
		next.ReviewState = StatePending
		return CorrectionResult{Insert: []Proposal{next}, Supersede: []string{base.CandidateID}}, nil
	case OpRename:
		base, err := one()
		if err != nil {
			return CorrectionResult{}, err
		}
		if err := requirePending(base); err != nil {
			return CorrectionResult{}, err
		}
		name := strings.TrimSpace(request.Name)
		if name == "" || len([]rune(name)) > MaxNameRunes {
			return CorrectionResult{}, errors.New("a name of 1-200 characters is required")
		}
		next := clone(base)
		old := next.Name
		next.Name = name
		next.RemoveAlias(name)
		if !LooksLikeAddress(old) && !(next.SourceOwner && strings.HasPrefix(old, "Device owner")) && NameKey(old) != NameKey(name) {
			next.AddAlias(NewNameAlias(old, AliasOther, SourceOwner, 1))
		}
		next = stamp(next, base.CandidateID)
		next.ReviewState = StatePending
		return CorrectionResult{Insert: []Proposal{next}, Supersede: []string{base.CandidateID}}, nil
	case OpRetype:
		base, err := one()
		if err != nil {
			return CorrectionResult{}, err
		}
		if err := requirePending(base); err != nil {
			return CorrectionResult{}, err
		}
		if !ValidRegistryType(request.RegistryType) {
			return CorrectionResult{}, fmt.Errorf("registry_type %q is not an entity type", request.RegistryType)
		}
		next := clone(base)
		next.RegistryType = request.RegistryType
		next = stamp(next, base.CandidateID)
		next.ReviewState = StatePending
		return CorrectionResult{Insert: []Proposal{next}, Supersede: []string{base.CandidateID}}, nil
	case OpAddAlias:
		base, err := one()
		if err != nil {
			return CorrectionResult{}, err
		}
		if err := requirePending(base); err != nil {
			return CorrectionResult{}, err
		}
		text := strings.TrimSpace(request.AliasText)
		if text == "" || len([]rune(text)) > MaxNameRunes {
			return CorrectionResult{}, errors.New("alias_text of 1-200 characters is required")
		}
		next := clone(base)
		var alias Alias
		if LooksLikeAddress(text) {
			address := NormalizeAddress(text)
			if address.Kind == AddressSelf {
				return CorrectionResult{}, errors.New(`"self" is relative to each source and cannot be added as an alias`)
			}
			alias = NewAddressAlias(address, SourceOwner, "")
		} else {
			kind := request.AliasKind
			if kind == "" {
				kind = AliasNickname
			}
			if !ValidAliasKind(kind) || kind == AliasHandle {
				return CorrectionResult{}, fmt.Errorf("alias_kind %q is not valid for a name", kind)
			}
			alias = NewNameAlias(text, kind, SourceOwner, 1)
		}
		if !next.AddAlias(alias) {
			return CorrectionResult{}, fmt.Errorf("%q is already an alias of %q", text, next.Name)
		}
		next = stamp(next, base.CandidateID)
		next.ReviewState = StatePending
		return CorrectionResult{Insert: []Proposal{next}, Supersede: []string{base.CandidateID}}, nil
	case OpRemoveAlias:
		base, err := one()
		if err != nil {
			return CorrectionResult{}, err
		}
		if err := requirePending(base); err != nil {
			return CorrectionResult{}, err
		}
		next := clone(base)
		if !next.RemoveAlias(request.AliasText) {
			return CorrectionResult{}, fmt.Errorf("%q is not an alias of %q", request.AliasText, base.Name)
		}
		next.MentionSample = withoutSurface(next.MentionSample, request.AliasText)
		next.ModelMentions = withoutSurface(next.ModelMentions, request.AliasText)
		next = stamp(next, base.CandidateID)
		next.ReviewState = StatePending
		return CorrectionResult{Insert: []Proposal{next}, Supersede: []string{base.CandidateID}}, nil
	case OpMerge:
		if len(bases) < 2 {
			return CorrectionResult{}, errors.New("merge takes two or more proposals")
		}
		var match *Match
		for _, base := range bases {
			if err := requirePending(base); err != nil {
				return CorrectionResult{}, err
			}
			if base.Match != nil {
				if match != nil && match.EntityID != base.Match.EntityID {
					return CorrectionResult{}, errors.New("these proposals match different committed entities; clear one match first")
				}
				match = base.Match
			}
		}
		merged := mergeProposals(bases, RunScope{PreviewHandle: bases[0].PreviewHandle, GenerationID: bases[0].GenerationID, SourceVersionID: bases[0].SourceVersionID})
		if name := strings.TrimSpace(request.Name); name != "" {
			old := merged.Name
			merged.Name = name
			merged.RemoveAlias(name)
			if !LooksLikeAddress(old) && NameKey(old) != NameKey(name) {
				merged.AddAlias(NewNameAlias(old, AliasOther, SourceOwner, 1))
			}
		}
		merged.Match = match
		merged.DetectedBy = bases[0].DetectedBy
		ids := make([]string, 0, len(bases))
		for _, base := range bases {
			ids = append(ids, base.CandidateID)
		}
		merged = stamp(merged, ids...)
		merged.ReviewState = StatePending
		return CorrectionResult{Insert: []Proposal{merged}, Supersede: ids}, nil
	case OpSplit:
		base, err := one()
		if err != nil {
			return CorrectionResult{}, err
		}
		if err := requirePending(base); err != nil {
			return CorrectionResult{}, err
		}
		name := strings.TrimSpace(request.Name)
		if name == "" {
			return CorrectionResult{}, errors.New("split needs a name for the new entity")
		}
		if len(request.SplitAliases) == 0 {
			return CorrectionResult{}, errors.New("split needs at least one alias to move")
		}
		stay, moved := clone(base), clone(base)
		moved.Aliases, moved.Participants, moved.MentionSample, moved.ModelMentions = nil, nil, nil, nil
		moved.Match, moved.Suggestions, moved.MentionCount, moved.SourceOwner = nil, nil, 0, false
		moved.Name = name
		for _, text := range request.SplitAliases {
			var found *Alias
			for i := range stay.Aliases {
				alias := stay.Aliases[i]
				if (alias.IsAddress() && (alias.Normalized == NormalizeAddress(text).Normalized || alias.Text == strings.TrimSpace(text))) ||
					(!alias.IsAddress() && NameKey(alias.Text) == NameKey(text)) {
					found = &alias
					break
				}
			}
			if found == nil {
				return CorrectionResult{}, fmt.Errorf("%q is not an alias of %q", text, base.Name)
			}
			stay.RemoveAlias(text)
			moved.AddAlias(*found)
			if found.IsAddress() {
				for i := 0; i < len(stay.Participants); i++ {
					if NormalizeAddress(stay.Participants[i].Identifier).Normalized == found.Normalized {
						moved.Participants = append(moved.Participants, stay.Participants[i])
						moved.MentionCount += stay.Participants[i].MessageCount
						stay.MentionCount -= stay.Participants[i].MessageCount
						stay.Participants = append(stay.Participants[:i], stay.Participants[i+1:]...)
						i--
					}
				}
				if found.AddressKind == AddressSelf {
					moved.SourceOwner, stay.SourceOwner = true, false
				}
			}
			moved.MentionSample = append(moved.MentionSample, withSurface(stay.MentionSample, text)...)
			stay.MentionSample = withoutSurface(stay.MentionSample, text)
			moved.ModelMentions = append(moved.ModelMentions, withSurface(stay.ModelMentions, text)...)
			stay.ModelMentions = withoutSurface(stay.ModelMentions, text)
		}
		if len(stay.Keys()) == 0 {
			return CorrectionResult{}, errors.New("split would leave the original entity with nothing; rename it instead")
		}
		stay = stamp(stay, base.CandidateID)
		moved = stamp(moved, base.CandidateID)
		stay.ReviewState, moved.ReviewState = StatePending, StatePending
		return CorrectionResult{Insert: []Proposal{stay, moved}, Supersede: []string{base.CandidateID}}, nil
	case OpSetMatch:
		base, err := one()
		if err != nil {
			return CorrectionResult{}, err
		}
		if err := requirePending(base); err != nil {
			return CorrectionResult{}, err
		}
		if request.MatchEntity == nil || strings.TrimSpace(request.MatchEntity.EntityID) == "" {
			return CorrectionResult{}, errors.New("match.entity_id is required")
		}
		next := clone(base)
		match := *request.MatchEntity
		match.Via, match.Score = "owner", 1
		next.Match = &match
		next = stamp(next, base.CandidateID)
		next.ReviewState = StatePending
		return CorrectionResult{Insert: []Proposal{next}, Supersede: []string{base.CandidateID}}, nil
	case OpClearMatch:
		base, err := one()
		if err != nil {
			return CorrectionResult{}, err
		}
		if err := requirePending(base); err != nil {
			return CorrectionResult{}, err
		}
		if base.Match == nil {
			return CorrectionResult{}, fmt.Errorf("%q has no match to clear", base.Name)
		}
		next := clone(base)
		next.Match = nil
		next = stamp(next, base.CandidateID)
		next.ReviewState = StatePending
		return CorrectionResult{Insert: []Proposal{next}, Supersede: []string{base.CandidateID}}, nil
	default:
		return CorrectionResult{}, fmt.Errorf("unknown correction op %q", request.Op)
	}
}

func clone(p Proposal) Proposal {
	out := p
	out.Aliases = append([]Alias(nil), p.Aliases...)
	out.Participants = append([]ParticipantStat(nil), p.Participants...)
	out.MentionSample = append([]Mention(nil), p.MentionSample...)
	out.ModelMentions = append([]Mention(nil), p.ModelMentions...)
	out.Suggestions = append([]Suggestion(nil), p.Suggestions...)
	out.Extractors = append([]string(nil), p.Extractors...)
	out.Flags = append([]Flag(nil), p.Flags...)
	out.ModelBatchOrigin = append([]int(nil), p.ModelBatchOrigin...)
	if p.Match != nil {
		match := *p.Match
		out.Match = &match
	}
	return out
}

func surfaceMatches(mention Mention, text string) bool {
	if NameKey(mention.Surface) == NameKey(text) && NameKey(text) != "" {
		return true
	}
	address := NormalizeAddress(text)
	return address.Normalized != "" && NormalizeAddress(mention.Surface).Normalized == address.Normalized
}

func withSurface(mentions []Mention, text string) []Mention {
	var out []Mention
	for _, mention := range mentions {
		if surfaceMatches(mention, text) {
			out = append(out, mention)
		}
	}
	return out
}

func withoutSurface(mentions []Mention, text string) []Mention {
	var out []Mention
	for _, mention := range mentions {
		if !surfaceMatches(mention, text) {
			out = append(out, mention)
		}
	}
	return out
}

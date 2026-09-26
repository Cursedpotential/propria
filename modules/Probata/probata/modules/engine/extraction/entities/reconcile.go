// Byline: Claude Code · Opus 5.5 · 2026-09-25

package entities

import (
	"fmt"
	"sort"
	"strings"
)

// ReconcileExtractor is the identity of the grouping step.
const (
	ReconcileExtractor        = "probata.entities.reconcile"
	ReconcileExtractorVersion = "1"
)

// RegistryAlias is one committed alias.
type RegistryAlias struct {
	Text string    `json:"text"`
	Kind AliasKind `json:"kind"`
}

// RegistryEntity is one live committed entity (merged_into_id IS NULL).
type RegistryEntity struct {
	ID             string          `json:"id"`
	DisplayName    string          `json:"display_name"`
	RegistryType   RegistryType    `json:"registry_type"`
	NormalizedName string          `json:"normalized_name"`
	Aliases        []RegistryAlias `json:"aliases,omitempty"`
}

// ReconcileInput is everything the grouping step decides over.
type ReconcileInput struct {
	Scope RunScope
	// Partials are this extraction's raw proposals (rules and model), each
	// already staged with a CandidateID.
	Partials []Proposal
	// Existing are the generation's current proposals from earlier
	// extractions and corrections: pending, rejected, or approved.
	Existing []Proposal
	// Registry holds the live committed entities relevant to the keys.
	Registry []RegistryEntity
}

// DroppedProposal records a proposal withheld because the owner already
// rejected what it covers.
type DroppedProposal struct {
	Name       string   `json:"name"`
	Keys       []string `json:"keys"`
	RejectedBy string   `json:"rejected_candidate_id"`
}

// ReconcilePlan is what the grouping step writes.
type ReconcilePlan struct {
	Insert    []Proposal        `json:"insert"`
	Supersede []string          `json:"supersede"`
	Dropped   []DroppedProposal `json:"dropped,omitempty"`
}

// Reconcile groups partial proposals into one proposal per entity, folds
// them into the owner's current proposals, and matches them to committed
// entities. It never overwrites an owner correction: an existing pending
// proposal keeps its name and type and only gains aliases.
func Reconcile(input ReconcileInput) ReconcilePlan {
	groups := groupPartials(input.Partials)
	var plan ReconcilePlan
	supersede := map[string]bool{}
	for _, group := range groups {
		merged := mergeProposals(group, input.Scope)
		for _, part := range group {
			if part.CandidateID != "" {
				supersede[part.CandidateID] = true
			}
		}
		keys := keySet(merged.Keys())
		var rejected, pending, approved []Proposal
		for _, existing := range input.Existing {
			if !overlaps(keys, existing.Keys()) {
				continue
			}
			switch existing.ReviewState {
			case StateRejected:
				rejected = append(rejected, existing)
			case StateApproved:
				approved = append(approved, existing)
			case StatePending, "":
				pending = append(pending, existing)
			}
		}
		for _, reject := range rejected {
			rejectedKeys := keySet(reject.Keys())
			kept := merged.Aliases[:0]
			for _, alias := range merged.Aliases {
				if !rejectedKeys[alias.Key(merged.Coarse())] {
					kept = append(kept, alias)
				}
			}
			merged.Aliases = kept
			if rejectedKeys["name:"+string(merged.Coarse())+":"+NameKey(merged.Name)] {
				if replacement := firstNameAlias(merged); replacement != "" {
					merged.RemoveAlias(replacement)
					merged.Name = replacement
				} else if len(merged.AddressAliases()) > 0 {
					merged.Name = DisplayAddress(Address{Kind: merged.AddressAliases()[0].AddressKind, Normalized: merged.AddressAliases()[0].Normalized})
				} else {
					merged.Name = ""
				}
			}
			if len(merged.Keys()) == 0 || merged.Name == "" {
				plan.Dropped = append(plan.Dropped, DroppedProposal{Name: reject.Name, Keys: reject.Keys(), RejectedBy: reject.CandidateID})
				merged = Proposal{}
				break
			}
			merged.addFlag(Flag{Code: "partly_rejected_before", Detail: "part of this entity was rejected earlier; those aliases were left out"})
		}
		if merged.Name == "" {
			continue
		}
		if len(pending) > 0 {
			target := pending[0]
			for _, other := range pending[1:] {
				// Two current proposals now cover one entity; surface it rather
				// than silently merging the owner's separate rows.
				target.addFlag(Flag{Code: "possible_duplicate", Detail: fmt.Sprintf("also covered by proposal %q", other.Name)})
			}
			result, changed := foldInto(target, merged)
			if changed {
				result.Supersedes = append(append([]string{}, target.CandidateID), sortedKeys(supersedeFor(group))...)
				plan.Insert = append(plan.Insert, result)
				supersede[target.CandidateID] = true
			}
			continue
		}
		if len(approved) > 0 {
			target := approved[0]
			addition := newAliasesOnly(target, merged)
			if len(addition.Aliases) == 0 && len(addition.ModelMentions) == 0 {
				continue
			}
			addition.Match = &Match{EntityID: target.PromotedToID, DisplayName: target.Name, Via: "committed_proposal", Score: 1}
			if len(approved) > 1 {
				addition.addFlag(Flag{Code: "overlaps_committed", Detail: "overlaps more than one committed entity; choose the right one before committing"})
			}
			addition.Supersedes = sortedKeys(supersedeFor(group))
			plan.Insert = append(plan.Insert, addition)
			continue
		}
		merged.Supersedes = sortedKeys(supersedeFor(group))
		plan.Insert = append(plan.Insert, merged)
	}
	for i := range plan.Insert {
		if plan.Insert[i].Match == nil {
			matchRegistry(&plan.Insert[i], input.Registry)
		}
		plan.Insert[i].Normalize()
	}
	sort.SliceStable(plan.Insert, func(i, j int) bool {
		return strings.ToLower(plan.Insert[i].Name) < strings.ToLower(plan.Insert[j].Name)
	})
	plan.Supersede = sortedKeys(supersede)
	return plan
}

func supersedeFor(group []Proposal) map[string]bool {
	out := map[string]bool{}
	for _, part := range group {
		if part.CandidateID != "" {
			out[part.CandidateID] = true
		}
	}
	return out
}

// groupPartials unions partial proposals that share a key or whose names are
// the same or a spelling variant; a shorter name contained in exactly one
// longer name of the same type joins it too.
func groupPartials(partials []Proposal) [][]Proposal {
	parent := make([]int, len(partials))
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(i int) int {
		if parent[i] != i {
			parent[i] = find(parent[i])
		}
		return parent[i]
	}
	union := func(a, b int) {
		ra, rb := find(a), find(b)
		if ra == rb {
			return
		}
		if ra < rb {
			parent[rb] = ra
		} else {
			parent[ra] = rb
		}
	}
	keyOwner := map[string]int{}
	for i, part := range partials {
		for _, key := range part.Keys() {
			if owner, ok := keyOwner[key]; ok {
				union(owner, i)
			} else {
				keyOwner[key] = i
			}
		}
	}
	for i := range partials {
		for j := i + 1; j < len(partials); j++ {
			relation := bestRelation(partials[i], partials[j])
			switch {
			case partials[i].Coarse() == partials[j].Coarse():
				if relation == NameSame || relation == NameSpelling {
					union(i, j)
				}
			case placeOrOrganization(partials[i].Coarse()) && placeOrOrganization(partials[j].Coarse()):
				// "Oakland County court" filed once as a place and once as an
				// organization is one thing; only an identical name crosses.
				if relation == NameSame {
					union(i, j)
				}
			}
		}
	}
	for i := range partials {
		var containers []int
		for j := range partials {
			if i == j || partials[i].Coarse() != partials[j].Coarse() {
				continue
			}
			if len(Tokens(partials[i].Name)) < len(Tokens(partials[j].Name)) && RelateNames(partials[i].Name, partials[j].Name) == NameContained {
				containers = append(containers, find(j))
			}
		}
		if distinct := uniqueInts(containers); len(distinct) == 1 {
			union(i, distinct[0])
		}
	}
	byRoot := map[int][]Proposal{}
	var roots []int
	for i, part := range partials {
		root := find(i)
		if _, ok := byRoot[root]; !ok {
			roots = append(roots, root)
		}
		byRoot[root] = append(byRoot[root], part)
	}
	sort.Ints(roots)
	out := make([][]Proposal, 0, len(roots))
	for _, root := range roots {
		out = append(out, byRoot[root])
	}
	return out
}

func placeOrOrganization(value CandidateType) bool {
	return value == CandidateLocation || value == CandidateOrganization
}

func bestRelation(a, b Proposal) NameRelation {
	best := NameUnrelated
	for _, left := range a.NameAliases() {
		for _, right := range b.NameAliases() {
			switch RelateNames(left.Text, right.Text) {
			case NameSame:
				return NameSame
			case NameSpelling:
				best = NameSpelling
			}
		}
	}
	return best
}

func uniqueInts(values []int) []int {
	seen := map[int]bool{}
	var out []int
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}

// namePriority ranks where a display name came from.
func namePriority(p Proposal) int {
	switch {
	case p.Correction != nil:
		return 4
	case LooksLikeAddress(p.Name) || p.SourceOwner && strings.HasPrefix(p.Name, "Device owner"):
		return 0
	case hasExtractor(p, "model:"):
		return 2
	default:
		return 3
	}
}

func hasExtractor(p Proposal, prefix string) bool {
	for _, extractor := range p.Extractors {
		if strings.HasPrefix(extractor, prefix) {
			return true
		}
	}
	return false
}

// mergeProposals folds a group into one proposal.
func mergeProposals(group []Proposal, scope RunScope) Proposal {
	ordered := append([]Proposal(nil), group...)
	sort.SliceStable(ordered, func(i, j int) bool {
		pi, pj := namePriority(ordered[i]), namePriority(ordered[j])
		if pi != pj {
			return pi > pj
		}
		if ordered[i].MentionCount != ordered[j].MentionCount {
			return ordered[i].MentionCount > ordered[j].MentionCount
		}
		if len(ordered[i].Aliases) != len(ordered[j].Aliases) {
			return len(ordered[i].Aliases) > len(ordered[j].Aliases)
		}
		return ordered[i].Name < ordered[j].Name
	})
	base := ordered[0]
	name := base.Name
	if name == strings.ToLower(name) && base.Correction == nil {
		for _, part := range ordered {
			if NameKey(part.Name) == NameKey(name) && part.Name != strings.ToLower(part.Name) {
				name = part.Name
				break
			}
		}
	}
	merged := Proposal{
		Name: name, RegistryType: base.RegistryType, DetectedBy: base.DetectedBy,
		GenerationID: scope.GenerationID, SourceVersionID: scope.SourceVersionID, PreviewHandle: scope.PreviewHandle,
		Confidence: base.Confidence, Match: base.Match,
	}
	for _, part := range ordered {
		for _, alias := range part.Aliases {
			merged.AddAlias(alias)
		}
		if part.Name != merged.Name && !LooksLikeAddress(part.Name) && !(part.SourceOwner && strings.HasPrefix(part.Name, "Device owner")) {
			kind := AliasOther
			if RelateNames(part.Name, merged.Name) == NameSpelling {
				kind = AliasMisspelling
			}
			source := SourceRules
			if hasExtractor(part, "model:") {
				source = SourceModel
			}
			merged.AddAlias(NewNameAlias(part.Name, kind, source, part.Confidence))
		}
		merged.Participants = mergeParticipants(merged.Participants, part.Participants)
		merged.MentionCount += part.MentionCount
		merged.AddMentionSample(part.MentionSample...)
		merged.ModelMentions = mergeMentions(merged.ModelMentions, part.ModelMentions)
		merged.Extractors = append(merged.Extractors, part.Extractors...)
		merged.SourceOwner = merged.SourceOwner || part.SourceOwner
		merged.ObserveTime(part.FirstOccurredAt)
		merged.ObserveTime(part.LastOccurredAt)
		merged.ModelBatchOrigin = append(merged.ModelBatchOrigin, part.ModelBatchOrigin...)
		if part.Confidence > merged.Confidence {
			merged.Confidence = part.Confidence
		}
		for _, flag := range part.Flags {
			merged.addFlag(flag)
		}
		if merged.Match == nil && part.Match != nil {
			merged.Match = part.Match
		}
	}
	// A name alias that is only a spelling variant of the chosen name keeps
	// that meaning even when it arrived as a plain display name.
	for i, alias := range merged.Aliases {
		if !alias.IsAddress() && alias.Kind == AliasOther && RelateNames(alias.Text, merged.Name) == NameSpelling {
			merged.Aliases[i].Kind = AliasMisspelling
		}
	}
	merged.Extractors = append(merged.Extractors, ReconcileExtractor+"@"+ReconcileExtractorVersion)
	merged.ModelBatchOrigin = uniqueInts(merged.ModelBatchOrigin)
	sort.Ints(merged.ModelBatchOrigin)
	merged.Normalize()
	return merged
}

func mergeParticipants(into, from []ParticipantStat) []ParticipantStat {
	index := map[string]int{}
	for i, stat := range into {
		index[stat.Identifier] = i
	}
	for _, stat := range from {
		if i, ok := index[stat.Identifier]; ok {
			if stat.MessageCount > into[i].MessageCount {
				into[i] = stat
			}
			continue
		}
		index[stat.Identifier] = len(into)
		into = append(into, stat)
	}
	sort.Slice(into, func(i, j int) bool { return into[i].Identifier < into[j].Identifier })
	return into
}

func mergeMentions(into, from []Mention) []Mention {
	seen := map[string]bool{}
	for _, mention := range into {
		seen[mention.SpanKey()] = true
	}
	for _, mention := range from {
		if !seen[mention.SpanKey()] {
			seen[mention.SpanKey()] = true
			into = append(into, mention)
		}
	}
	sort.SliceStable(into, func(i, j int) bool {
		if into[i].Ordinal != into[j].Ordinal {
			return into[i].Ordinal < into[j].Ordinal
		}
		return into[i].SpanKey() < into[j].SpanKey()
	})
	return into
}

// foldInto adds merged's new aliases and mentions to an existing pending
// proposal, keeping the owner's name, type and match.
func foldInto(target, merged Proposal) (Proposal, bool) {
	result := target
	result.CandidateID = ""
	result.Aliases = append([]Alias(nil), target.Aliases...)
	result.MentionSample = append([]Mention(nil), target.MentionSample...)
	result.ModelMentions = append([]Mention(nil), target.ModelMentions...)
	result.Participants = append([]ParticipantStat(nil), target.Participants...)
	result.Extractors = append([]string(nil), target.Extractors...)
	result.Flags = append([]Flag(nil), target.Flags...)
	result.Correction = nil
	changed := false
	for _, alias := range merged.Aliases {
		if result.AddAlias(alias) {
			changed = true
		}
	}
	if NameKey(merged.Name) != NameKey(result.Name) && !LooksLikeAddress(merged.Name) && !merged.SourceOwner {
		if result.AddAlias(NewNameAlias(merged.Name, AliasOther, SourceModel, merged.Confidence)) {
			changed = true
		}
	}
	before := len(result.ModelMentions)
	result.ModelMentions = mergeMentions(result.ModelMentions, merged.ModelMentions)
	changed = changed || len(result.ModelMentions) != before
	result.Participants = mergeParticipants(result.Participants, merged.Participants)
	result.AddMentionSample(merged.MentionSample...)
	result.Extractors = append(result.Extractors, merged.Extractors...)
	result.ObserveTime(merged.FirstOccurredAt)
	result.ObserveTime(merged.LastOccurredAt)
	if merged.MentionCount > result.MentionCount {
		result.MentionCount = merged.MentionCount
	}
	result.ReviewState = StatePending
	return result, changed
}

// newAliasesOnly returns the part of merged that a committed proposal does
// not already carry, for attaching to its registry entity.
func newAliasesOnly(committed, merged Proposal) Proposal {
	known := keySet(committed.Keys())
	addition := Proposal{
		Name: committed.Name, RegistryType: committed.RegistryType, DetectedBy: DetectedAuto,
		GenerationID: merged.GenerationID, SourceVersionID: merged.SourceVersionID, PreviewHandle: merged.PreviewHandle,
		Confidence: merged.Confidence, Extractors: merged.Extractors, SourceOwner: merged.SourceOwner,
		Participants: merged.Participants, MentionCount: merged.MentionCount, MentionSample: merged.MentionSample,
		FirstOccurredAt: merged.FirstOccurredAt, LastOccurredAt: merged.LastOccurredAt,
	}
	for _, alias := range merged.Aliases {
		if !known[alias.Key(merged.Coarse())] {
			addition.AddAlias(alias)
		}
	}
	if NameKey(merged.Name) != NameKey(committed.Name) && !LooksLikeAddress(merged.Name) && !known["name:"+string(merged.Coarse())+":"+NameKey(merged.Name)] {
		addition.AddAlias(NewNameAlias(merged.Name, AliasOther, SourceModel, merged.Confidence))
	}
	addition.ModelMentions = append(addition.ModelMentions, merged.ModelMentions...)
	return addition
}

// matchRegistry proposes merging into a committed entity: an exact address
// alias is a match, an identical normalized name of the same kind is a
// match, a spelling variant is only a suggestion.
func matchRegistry(proposal *Proposal, registry []RegistryEntity) {
	for _, alias := range proposal.AddressAliases() {
		if alias.Scope != "" {
			continue
		}
		for _, entity := range registry {
			for _, committed := range entity.Aliases {
				address := NormalizeAddress(committed.Text)
				if address.Kind == alias.AddressKind && address.Normalized == alias.Normalized {
					proposal.Match = &Match{EntityID: entity.ID, DisplayName: entity.DisplayName, Via: "alias", Alias: alias.Normalized, Score: 1}
					return
				}
			}
		}
	}
	if !LooksLikeAddress(proposal.Name) && !proposal.SourceOwner {
		for _, entity := range registry {
			if CandidateTypeFor(entity.RegistryType) == proposal.Coarse() && entity.NormalizedName == RegistryNormalizedName(proposal.Name) {
				proposal.Match = &Match{EntityID: entity.ID, DisplayName: entity.DisplayName, Via: "name", Score: 0.9}
				return
			}
		}
	}
	for _, entity := range registry {
		if CandidateTypeFor(entity.RegistryType) != proposal.Coarse() {
			continue
		}
		names := []string{entity.DisplayName}
		for _, committed := range entity.Aliases {
			if !LooksLikeAddress(committed.Text) {
				names = append(names, committed.Text)
			}
		}
		for _, alias := range proposal.NameAliases() {
			for _, name := range names {
				switch RelateNames(alias.Text, name) {
				case NameSame:
					proposal.Suggestions = appendSuggestion(proposal.Suggestions, Suggestion{EntityID: entity.ID, DisplayName: entity.DisplayName, Reason: "same name as a committed alias", Score: 0.8})
				case NameSpelling:
					proposal.Suggestions = appendSuggestion(proposal.Suggestions, Suggestion{EntityID: entity.ID, DisplayName: entity.DisplayName, Reason: "spelling variant of a committed name", Score: 0.6})
				}
			}
		}
	}
}

func appendSuggestion(list []Suggestion, suggestion Suggestion) []Suggestion {
	for _, existing := range list {
		if existing.EntityID == suggestion.EntityID {
			return list
		}
	}
	return append(list, suggestion)
}

func firstNameAlias(p Proposal) string {
	for _, alias := range p.Aliases {
		if !alias.IsAddress() && NameKey(alias.Text) != "" {
			return alias.Text
		}
	}
	return ""
}

func keySet(keys []string) map[string]bool {
	out := make(map[string]bool, len(keys))
	for _, key := range keys {
		out[key] = true
	}
	return out
}

func overlaps(keys map[string]bool, other []string) bool {
	for _, key := range other {
		if keys[key] {
			return true
		}
	}
	return false
}

func sortedKeys(values map[string]bool) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

// Byline: Claude Code · Opus 5.5 · 2026-09-25

package entities

import (
	"sort"
	"strings"
	"time"
)

// ParticipantAggregate is one row of the DuckDB participant ELT query
// (postgres/entity_extraction_store.go participantAggregateSQL): every
// distinct normalized participant address of the run with the raw spellings
// that produced it, display names and their name tokens, role counts, time
// window and a few supporting records. When NormalizedAddress is set it came
// from the DuckDB query and is authoritative; NormalizeAddress is the Go
// mirror used for identifiers that arrive any other way.
type ParticipantAggregate struct {
	Identifier        string            `json:"identifier"`
	Identifiers       []string          `json:"identifiers,omitempty"`
	AddressKind       string            `json:"address_kind,omitempty"`
	NormalizedAddress string            `json:"normalized_address,omitempty"`
	DisplayNames      []string          `json:"display_names"`
	NameTokens        [][]string        `json:"name_tokens,omitempty"`
	SenderCount       int               `json:"sender_count"`
	RecipientCount    int               `json:"recipient_count"`
	MessageCount      int               `json:"message_count"`
	TotalMessages     int               `json:"total_messages"`
	FirstAt           *time.Time        `json:"first_at,omitempty"`
	LastAt            *time.Time        `json:"last_at,omitempty"`
	Samples           []ParticipantSeen `json:"samples,omitempty"`
}

// AddressOf returns the row's address: the DuckDB normalization when
// present, else the Go mirror. mismatch reports a disagreement between the
// two so drift is visible instead of silent.
func AddressOf(row ParticipantAggregate) (address Address, mismatch bool) {
	mirror := NormalizeAddress(row.Identifier)
	if row.NormalizedAddress == "" {
		return mirror, false
	}
	address = Address{Kind: AddressKind(row.AddressKind), Raw: row.Identifier, Normalized: row.NormalizedAddress}
	return address, mirror.Kind != address.Kind || mirror.Normalized != address.Normalized
}

// ParticipantSeen is one record a participant appears in.
type ParticipantSeen struct {
	RecordID            string     `json:"record_id"`
	Ordinal             int64      `json:"ordinal"`
	Role                string     `json:"role"`
	OccurredAt          *time.Time `json:"occurred_at,omitempty"`
	SourceAvailableFrom *time.Time `json:"source_available_from,omitempty"`
}

// RulesExtractor is the extractor identity recorded on working.extraction_run.
const (
	RulesExtractor        = "probata.entities.rules"
	RulesExtractorVersion = "1"
)

// RunScope identifies what a proposal set was computed against.
type RunScope struct {
	PreviewHandle   string `json:"preview_handle"`
	GenerationID    string `json:"normalized_generation_id"`
	SourceVersionID string `json:"source_version_id"`
}

// ProposeFromParticipants groups a run's participant identifiers into
// proposed people. Identifiers that normalize to one address are one
// person; addresses whose contact cards carry the same display name are one
// person. Nothing is merged on a weaker signal here — sound-alike names are
// the reconciler's job and only ever a proposal.
func ProposeFromParticipants(scope RunScope, rows []ParticipantAggregate) []Proposal {
	type group struct {
		addresses map[string]Address
		rows      []ParticipantAggregate
		names     map[string]int
		display   map[string]string
		spellings map[string]map[string]int
	}
	parent := map[string]string{}
	var find func(string) string
	find = func(key string) string {
		if parent[key] == key {
			return key
		}
		parent[key] = find(parent[key])
		return parent[key]
	}
	union := func(a, b string) {
		ra, rb := find(a), find(b)
		if ra != rb {
			if ra < rb {
				parent[rb] = ra
			} else {
				parent[ra] = rb
			}
		}
	}
	byAddress := map[string][]ParticipantAggregate{}
	addresses := map[string]Address{}
	mismatched := map[string]bool{}
	for _, row := range rows {
		address, mismatch := AddressOf(row)
		if address.Normalized == "" {
			continue
		}
		if mismatch {
			mismatched[string(address.Kind)+":"+address.Normalized] = true
		}
		key := string(address.Kind) + ":" + address.Normalized
		if _, ok := parent[key]; !ok {
			parent[key] = key
		}
		byAddress[key] = append(byAddress[key], row)
		addresses[key] = address
	}
	// A display name shared by two non-self addresses joins them: one contact
	// card, two numbers. "self" never joins anyone by name.
	nameOwner := map[string]string{}
	for key, list := range byAddress {
		if addresses[key].Kind == AddressSelf {
			continue
		}
		for _, row := range list {
			for _, name := range row.DisplayNames {
				if LooksLikeAddress(name) || NameKey(name) == "" {
					continue
				}
				nameKey := NameKey(name)
				if owner, ok := nameOwner[nameKey]; ok {
					union(owner, key)
				} else {
					nameOwner[nameKey] = key
				}
			}
		}
	}
	groups := map[string]*group{}
	for key, list := range byAddress {
		root := find(key)
		g := groups[root]
		if g == nil {
			g = &group{addresses: map[string]Address{}, names: map[string]int{}, display: map[string]string{}, spellings: map[string]map[string]int{}}
			groups[root] = g
		}
		g.addresses[key] = addresses[key]
		g.rows = append(g.rows, list...)
		for _, row := range list {
			for _, name := range row.DisplayNames {
				if LooksLikeAddress(name) || NameKey(name) == "" {
					continue
				}
				nameKey := NameKey(name)
				g.names[nameKey] += row.MessageCount
				if g.spellings[nameKey] == nil {
					g.spellings[nameKey] = map[string]int{}
				}
				g.spellings[nameKey][strings.TrimSpace(name)] += row.MessageCount + 1
			}
		}
	}
	for _, g := range groups {
		for nameKey, spellings := range g.spellings {
			g.display[nameKey] = bestSpelling(spellings)
		}
	}
	roots := make([]string, 0, len(groups))
	for root := range groups {
		roots = append(roots, root)
	}
	sort.Strings(roots)
	out := make([]Proposal, 0, len(roots))
	for _, root := range roots {
		g := groups[root]
		proposal := Proposal{
			RegistryType:    "person",
			DetectedBy:      DetectedAuto,
			Extractors:      []string{RulesExtractor + "@" + RulesExtractorVersion},
			Confidence:      1,
			GenerationID:    scope.GenerationID,
			SourceVersionID: scope.SourceVersionID,
			PreviewHandle:   scope.PreviewHandle,
		}
		addressKeys := make([]string, 0, len(g.addresses))
		for key := range g.addresses {
			addressKeys = append(addressKeys, key)
		}
		sort.Strings(addressKeys)
		for _, key := range addressKeys {
			address := g.addresses[key]
			if address.Kind == AddressSelf {
				proposal.SourceOwner = true
			}
			proposal.AddAlias(NewAddressAlias(address, SourceParticipant, "source:"+scope.SourceVersionID))
			if mismatched[key] {
				proposal.addFlag(Flag{Code: "normalization_mismatch", Detail: "the DuckDB and Go address normalizers disagree for " + address.Normalized + "; DuckDB's form was used"})
			}
		}
		proposal.Name = preferredName(g.names, g.display)
		if proposal.Name == "" {
			proposal.Name = DisplayAddress(g.addresses[addressKeys[0]])
			if proposal.SourceOwner {
				proposal.Name = "Device owner (this source)"
			}
		}
		for nameKey, text := range g.display {
			if nameKey != NameKey(proposal.Name) {
				proposal.AddAlias(NewNameAlias(text, AliasOther, SourceParticipant, 0.9))
			}
		}
		stats := map[string]*ParticipantStat{}
		for _, row := range g.rows {
			address, _ := AddressOf(row)
			stat := stats[row.Identifier]
			if stat == nil {
				stat = &ParticipantStat{Identifier: row.Identifier, Address: address}
				stats[row.Identifier] = stat
			}
			stat.SenderCount += row.SenderCount
			stat.RecipientCount += row.RecipientCount
			stat.MessageCount += row.MessageCount
			stat.FirstAt = earlier(stat.FirstAt, row.FirstAt)
			stat.LastAt = later(stat.LastAt, row.LastAt)
			proposal.MentionCount += row.MessageCount
			proposal.ObserveTime(row.FirstAt)
			proposal.ObserveTime(row.LastAt)
			for _, seen := range row.Samples {
				proposal.AddMentionSample(Mention{
					RecordID: seen.RecordID, Ordinal: seen.Ordinal, OccurredAt: seen.OccurredAt,
					SourceAvailableFrom: seen.SourceAvailableFrom, Kind: MentionKindFor(address), Role: roleOf(seen.Role),
					Surface: row.Identifier, Method: MethodParticipant, Confidence: 1,
				})
			}
		}
		identifiers := make([]string, 0, len(stats))
		for identifier := range stats {
			identifiers = append(identifiers, identifier)
		}
		sort.Strings(identifiers)
		for _, identifier := range identifiers {
			proposal.Participants = append(proposal.Participants, *stats[identifier])
		}
		proposal.Normalize()
		out = append(out, proposal)
	}
	return out
}

// bestSpelling picks the display spelling used most, then the one with more
// capitals ("Katherine" over "katherine"), then the lexically first — never
// map order.
func bestSpelling(spellings map[string]int) string {
	best, bestCount, bestCaps := "", -1, -1
	for spelling, count := range spellings {
		caps := 0
		for _, r := range spelling {
			if r >= 'A' && r <= 'Z' || r > 127 && strings.ToUpper(string(r)) == string(r) && strings.ToLower(string(r)) != string(r) {
				caps++
			}
		}
		switch {
		case count > bestCount,
			count == bestCount && caps > bestCaps,
			count == bestCount && caps == bestCaps && spelling < best:
			best, bestCount, bestCaps = spelling, count, caps
		}
	}
	return best
}

func preferredName(counts map[string]int, display map[string]string) string {
	best, bestCount := "", -1
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if counts[key] > bestCount {
			best, bestCount = key, counts[key]
		}
	}
	if best == "" {
		return ""
	}
	return display[best]
}

func roleOf(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "sender", "from":
		return RoleSender
	case "recipient", "to", "cc", "bcc":
		return RoleRecipient
	default:
		return RoleUnknown
	}
}

// MentionKindFor maps an address to working.entity_mention.mention_kind.
func MentionKindFor(address Address) string {
	switch address.Kind {
	case AddressPhone:
		return "phone"
	case AddressEmail:
		return "email"
	case AddressHandle:
		return "handle"
	default:
		return "other"
	}
}

func earlier(a, b *time.Time) *time.Time {
	if a == nil {
		return b
	}
	if b == nil || a.Before(*b) {
		return a
	}
	return b
}

func later(a, b *time.Time) *time.Time {
	if a == nil {
		return b
	}
	if b == nil || a.After(*b) {
		return a
	}
	return b
}

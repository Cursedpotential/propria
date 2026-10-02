// Package disclosure is the ONE disclosure-tier rule for the whole application
// (owner, 2026-10-02):
//
//	contemporaneous when the owner (Matthew S. Salem, the registry.person
//	whose role_in_case is 'user') was a participant in the record, as sender
//	or recipient; discovered when he was not.
//
// Every stage that writes a disclosure tier calls this package: the
// Weaviate-first publish (activities/publish_context_search.go) and the
// first-party context import. Participants are resolved ONCE per run, by
// resolve_context_participants_activity, into a Resolution that both stages
// read by reference, so an alias added between the two stages can never make
// the Weaviate tier and the PostgreSQL tier disagree (parent ruling
// 2026-10-02).
//
// The package is pure. The registry reads and registry.norm_identifier live in
// the PostgreSQL boundary (postgres.LoadOwner, postgres.ResolveParticipants),
// so identifier normalization is never re-implemented in Go.
//
// Hindsight is never assigned here: it is a review-time overlay
// (engine/contextreview), not a property of a record.
//
// Byline: Claude Code · Opus 5.5 · 2026-10-02
package disclosure

import (
	"errors"
	"fmt"
	"strings"
)

// Tiers, closed to the values the disclosure_tier CHECK constraints allow.
const (
	Contemporaneous = "contemporaneous"
	Discovered      = "discovered"
)

// Basis names this rule on every row or object it is applied to.
const Basis = "owner_participant:v1"

// SelfIdentifier is the device marker SMS Backup & Restore (and the SBV
// normalizer) writes for the phone's own owner. It names whoever owns the
// device the source came from, so it is the owner only when the run's
// perspective person is the owner.
const SelfIdentifier = "self"

// Owner is the case owner as the registry knows him.
type Owner struct {
	// PersonID is the registry.person id whose role_in_case is 'user'.
	PersonID string
	// PerspectivePersonID is whose device or export the source is; empty when
	// the run did not say.
	PerspectivePersonID string
	// Identifiers are his confirmed aliases, normalized by
	// registry.norm_identifier.
	Identifiers map[string]bool
}

// Validate fails closed on an owner the rule cannot be applied with.
func (o Owner) Validate() error {
	if strings.TrimSpace(o.PersonID) == "" {
		return errors.New("disclosure: the case owner (registry.person role_in_case='user') is not registered")
	}
	if len(o.Identifiers) == 0 {
		return errors.New("disclosure: the case owner has no confirmed identifiers in registry.entity_alias_current")
	}
	return nil
}

// ResolvedIdentifier is one stated identifier resolved against the registry.
type ResolvedIdentifier struct {
	// Raw is the identifier exactly as the source states it.
	Raw string `json:"raw"`
	// Normalized is registry.norm_identifier(Raw); empty when it normalizes
	// to nothing.
	Normalized string `json:"normalized"`
	// EntityID is the registry.entity it is a confirmed alias of, or empty.
	EntityID string `json:"entity_id,omitempty"`
	// IsOwner is true when EntityID is the case owner.
	IsOwner bool `json:"is_owner"`
}

// Resolution is every stated identifier of one run, resolved once.
type Resolution struct {
	Basis               string               `json:"basis"`
	OwnerPersonID       string               `json:"owner_person_id"`
	PerspectivePersonID string               `json:"perspective_person_id,omitempty"`
	Identifiers         []ResolvedIdentifier `json:"identifiers"`

	index map[string]ResolvedIdentifier
}

// NewResolution builds a Resolution for an owner and his run's identifiers.
func NewResolution(owner Owner, identifiers []ResolvedIdentifier) (Resolution, error) {
	if err := owner.Validate(); err != nil {
		return Resolution{}, err
	}
	resolution := Resolution{
		Basis: Basis, OwnerPersonID: owner.PersonID, PerspectivePersonID: strings.TrimSpace(owner.PerspectivePersonID),
		Identifiers: identifiers,
	}
	if err := resolution.Validate(); err != nil {
		return Resolution{}, err
	}
	return resolution, nil
}

// Validate fails closed on a resolution made under another rule or without an
// owner, or one that states the same identifier twice with different answers.
func (r Resolution) Validate() error {
	if r.Basis != Basis {
		return fmt.Errorf("disclosure: resolution basis %q is not %q", r.Basis, Basis)
	}
	if strings.TrimSpace(r.OwnerPersonID) == "" {
		return errors.New("disclosure: resolution names no case owner")
	}
	seen := map[string]ResolvedIdentifier{}
	for _, identifier := range r.Identifiers {
		if previous, ok := seen[identifier.Raw]; ok && previous != identifier {
			return fmt.Errorf("disclosure: resolution states identifier %q twice with different answers", identifier.Raw)
		}
		seen[identifier.Raw] = identifier
	}
	return nil
}

func (r *Resolution) lookupIndex() map[string]ResolvedIdentifier {
	if r.index == nil {
		r.index = make(map[string]ResolvedIdentifier, len(r.Identifiers))
		for _, identifier := range r.Identifiers {
			r.index[identifier.Raw] = identifier
		}
	}
	return r.index
}

// Lookup returns the resolution of one identifier exactly as stated.
func (r *Resolution) Lookup(raw string) (ResolvedIdentifier, bool) {
	identifier, ok := r.lookupIndex()[raw]
	return identifier, ok
}

// ForMessage applies the rule to one record, given its sender and recipients
// exactly as the source states them. It returns whether the owner took part,
// the tier and the basis. A stated identifier the resolution does not hold
// fails closed: it means the record was not part of the resolved run.
func (r *Resolution) ForMessage(sender string, recipients []string) (bool, string, string, error) {
	if err := r.Validate(); err != nil {
		return false, "", "", err
	}
	stated := make([]string, 0, len(recipients)+1)
	if strings.TrimSpace(sender) != "" {
		stated = append(stated, sender)
	}
	for _, recipient := range recipients {
		if strings.TrimSpace(recipient) != "" {
			stated = append(stated, recipient)
		}
	}
	tookPart := false
	for _, raw := range stated {
		if strings.EqualFold(strings.TrimSpace(raw), SelfIdentifier) {
			perspective := strings.TrimSpace(r.PerspectivePersonID)
			if perspective == "" {
				return false, "", "", errors.New(`disclosure: a participant is the device marker "self" but the run names no perspective person, so whose device it is is unknown`)
			}
			if strings.EqualFold(perspective, strings.TrimSpace(r.OwnerPersonID)) {
				tookPart = true
			}
			continue
		}
		identifier, ok := r.Lookup(raw)
		if !ok {
			return false, "", "", fmt.Errorf("disclosure: identifier %q is not in this run's participant resolution", raw)
		}
		if identifier.IsOwner {
			tookPart = true
		}
	}
	return tookPart, Tier(tookPart), Basis, nil
}

// Tier is the disclosure tier for a record.
func Tier(ownerTookPart bool) string {
	if ownerTookPart {
		return Contemporaneous
	}
	return Discovered
}

// Byline: Claude Code · Opus 5.5 · 2026-09-25
//
// Package commitcheck validates a run's current entity and event proposals
// before they are committed. Every rule is named, fails closed, and says why
// — the same pass/fail list shape as the repair plan validator
// (docs/pending-review/2026-09-25-repair-workflow-builder.md).
package commitcheck

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/Cursedpotential/probata/engine/extraction/entities"
	"github.com/Cursedpotential/probata/engine/extraction/events"
)

// Check statuses.
const (
	Pass = "pass"
	Fail = "fail"
)

// Bounds on one commit.
const (
	MaxEntitiesPerCommit = 2000
	MaxEventsPerCommit   = 5000
)

// Check is one named rule result.
type Check struct {
	Rule   string `json:"rule"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}

// Counts summarizes what a commit would write.
type Counts struct {
	Entities          int `json:"entities"`
	NewEntities       int `json:"new_entities"`
	MergeIntoExisting int `json:"merge_into_existing"`
	Aliases           int `json:"aliases"`
	Events            int `json:"events"`
	Rejected          int `json:"rejected"`
}

// Report is the validation result. Digest binds a commit to exactly the
// proposal set that was validated.
type Report struct {
	OK     bool    `json:"ok"`
	Checks []Check `json:"checks"`
	Digest string  `json:"digest"`
	Counts Counts  `json:"counts"`
}

// Snapshot is everything validation reads.
type Snapshot struct {
	MatterMode          string                    `json:"matter_mode"`
	PreviewHandle       string                    `json:"preview_handle"`
	CurrentGenerationID string                    `json:"current_generation_id"`
	Entities            []entities.Proposal       `json:"entities"`
	Events              []events.Proposal         `json:"events"`
	Registry            []entities.RegistryEntity `json:"registry"`
	// MissingRecords are supporting record ids that are not rows of the
	// run's normalized generation.
	MissingRecords []string `json:"missing_records"`
}

// Digest is the stable identity of the included proposal set.
func Digest(snapshot Snapshot) string {
	var parts []string
	for _, proposal := range snapshot.Entities {
		if proposal.Included() {
			parts = append(parts, "e:"+proposal.CandidateID+":"+proposal.ContentHex())
		}
	}
	for _, event := range snapshot.Events {
		if event.Included() {
			parts = append(parts, "v:"+event.CandidateID+":"+event.ContentHex())
		}
	}
	sort.Strings(parts)
	sum := sha256.Sum256([]byte(snapshot.PreviewHandle + "\n" + snapshot.CurrentGenerationID + "\n" + strings.Join(parts, "\n")))
	return hex.EncodeToString(sum[:])
}

// Validate runs every rule. It never short-circuits: the owner sees every
// problem at once.
func Validate(snapshot Snapshot) Report {
	var checks []Check
	add := func(rule string, problems []string, passReason string) {
		if len(problems) == 0 {
			checks = append(checks, Check{Rule: rule, Status: Pass, Reason: passReason})
			return
		}
		sort.Strings(problems)
		const shown = 5
		reason := strings.Join(firstN(problems, shown), "; ")
		if len(problems) > shown {
			reason += fmt.Sprintf("; and %d more", len(problems)-shown)
		}
		checks = append(checks, Check{Rule: rule, Status: Fail, Reason: reason})
	}
	var included []entities.Proposal
	var counts Counts
	for _, proposal := range snapshot.Entities {
		switch {
		case proposal.Included():
			included = append(included, proposal)
		case proposal.ReviewState == entities.StateRejected:
			counts.Rejected++
		}
	}
	var includedEvents []events.Proposal
	for _, event := range snapshot.Events {
		if event.Included() {
			includedEvents = append(includedEvents, event)
		} else if event.ReviewState == entities.StateRejected {
			counts.Rejected++
		}
	}
	registryByID := map[string]entities.RegistryEntity{}
	for _, entity := range snapshot.Registry {
		registryByID[entity.ID] = entity
	}

	// 1. Test data never becomes canonical.
	if snapshot.MatterMode == "REAL" {
		add("live_mode", nil, "Live (REAL) run: the registry and timeline are valid destinations")
	} else {
		add("live_mode", []string{fmt.Sprintf("this is a %s-mode run; proposals can be reviewed but only a Live-mode run commits to the registry and timeline", nonEmpty(snapshot.MatterMode, "unknown"))}, "")
	}
	// 2. Something to commit.
	if len(included)+len(includedEvents) == 0 {
		add("has_proposals", []string{"no current proposals are included; extract or restore some first"}, "")
	} else {
		add("has_proposals", nil, fmt.Sprintf("%d entities and %d events are included", len(included), len(includedEvents)))
	}
	// 3. Staleness guard: proposals must be computed against the run's
	// current normalized generation (DUAL-GRAPH-IDENTITY §3).
	var stale []string
	if snapshot.CurrentGenerationID == "" {
		stale = append(stale, "the run has no normalized generation")
	}
	for _, proposal := range included {
		if proposal.GenerationID != snapshot.CurrentGenerationID {
			stale = append(stale, fmt.Sprintf("%q was extracted from generation %s", proposal.Name, proposal.GenerationID))
		}
	}
	for _, event := range includedEvents {
		if event.GenerationID != snapshot.CurrentGenerationID {
			stale = append(stale, fmt.Sprintf("event %q was extracted from generation %s", event.Title, event.GenerationID))
		}
	}
	add("current_generation", stale, "every proposal was computed against the run's current normalized generation")
	// 4. Names and types.
	var naming []string
	for _, proposal := range included {
		switch {
		case strings.TrimSpace(proposal.Name) == "" || entities.RegistryNormalizedName(proposal.Name) == "":
			naming = append(naming, fmt.Sprintf("proposal %s has no name", proposal.CandidateID))
		case !entities.ValidRegistryType(proposal.RegistryType):
			naming = append(naming, fmt.Sprintf("%q has unknown type %q", proposal.Name, proposal.RegistryType))
		case len(proposal.Aliases) > entities.MaxAliases:
			naming = append(naming, fmt.Sprintf("%q has more than %d aliases", proposal.Name, entities.MaxAliases))
		}
	}
	add("names_and_types", naming, "every entity has a name and a registry type")
	// 5. Aliases belong to exactly one entity.
	type claimant struct{ id, name string }
	owner := map[string]claimant{}
	var aliasClashes []string
	for _, proposal := range included {
		for _, key := range proposal.Keys() {
			if other, ok := owner[key]; ok && other.id != proposal.CandidateID {
				aliasClashes = append(aliasClashes, fmt.Sprintf("%s is claimed by both %q and %q; merge them or remove the alias", displayKey(key), other.name, proposal.Name))
				continue
			}
			owner[key] = claimant{id: proposal.CandidateID, name: proposal.Name}
		}
		counts.Aliases += len(writableAliases(proposal))
	}
	add("aliases_unique", aliasClashes, "every alias belongs to exactly one entity")
	// 6. Merge targets are live committed entities (else aliases orphan).
	var deadTargets []string
	for _, proposal := range included {
		if proposal.Match == nil {
			counts.NewEntities++
			continue
		}
		if _, ok := registryByID[proposal.Match.EntityID]; !ok {
			deadTargets = append(deadTargets, fmt.Sprintf("%q would merge into %s, which is not a live committed entity", proposal.Name, proposal.Match.EntityID))
			continue
		}
		counts.MergeIntoExisting++
	}
	counts.Entities = len(included)
	add("match_targets_live", deadTargets, "every merge target is a live committed entity")
	// 7. No duplicate of an already-committed entity.
	var duplicates []string
	newNames := map[string]string{}
	for _, proposal := range included {
		target := ""
		if proposal.Match != nil {
			target = proposal.Match.EntityID
		}
		if target == "" {
			key := string(proposal.RegistryType) + "|" + entities.RegistryNormalizedName(proposal.Name)
			if other, ok := newNames[key]; ok {
				duplicates = append(duplicates, fmt.Sprintf("%q and %q would create two %s entities with the same name; merge or rename one", other, proposal.Name, proposal.RegistryType))
			}
			newNames[key] = proposal.Name
			for _, entity := range snapshot.Registry {
				if entity.RegistryType == proposal.RegistryType && entity.NormalizedName == entities.RegistryNormalizedName(proposal.Name) {
					duplicates = append(duplicates, fmt.Sprintf("%q already exists as committed entity %q; merge into it", proposal.Name, entity.DisplayName))
				}
			}
		}
		for _, alias := range proposal.AddressAliases() {
			if alias.Scope != "" {
				continue
			}
			for _, entity := range snapshot.Registry {
				if entity.ID == target {
					continue
				}
				for _, committed := range entity.Aliases {
					address := entities.NormalizeAddress(committed.Text)
					if address.Kind == alias.AddressKind && address.Normalized == alias.Normalized {
						duplicates = append(duplicates, fmt.Sprintf("%s already belongs to committed entity %q; merge %q into it", alias.Normalized, entity.DisplayName, proposal.Name))
					}
				}
			}
		}
	}
	add("no_duplicate_committed", duplicates, "no proposal duplicates a committed entity or alias")
	// 8. Every supporting record is a row of this run.
	if len(snapshot.MissingRecords) > 0 {
		var missing []string
		for _, id := range snapshot.MissingRecords {
			missing = append(missing, "record "+id+" is not in this run")
		}
		add("mentions_resolve", missing, "")
	} else {
		add("mentions_resolve", nil, "every mention and event source points at a record of this run")
	}
	// 9-12. Events.
	var untimed, unclocked, unsourced, unresolved []string
	eventKeys := map[string]string{}
	var duplicateEvents []string
	for _, event := range includedEvents {
		if event.OccurredAt == nil {
			untimed = append(untimed, fmt.Sprintf("event %q has no time", event.Title))
		}
		if len(event.SourceRecords) == 0 {
			unsourced = append(unsourced, fmt.Sprintf("event %q has no source message", event.Title))
		} else if event.SourceAvailableFrom() == nil {
			unclocked = append(unclocked, fmt.Sprintf("event %q rests on a record without a source availability clock", event.Title))
		}
		_, missing, ambiguous := events.ResolveEntityKeys(event, snapshot.Entities)
		for _, key := range missing {
			unresolved = append(unresolved, fmt.Sprintf("event %q names %s, which no current entity covers", event.Title, displayKey(key)))
		}
		for _, key := range ambiguous {
			unresolved = append(unresolved, fmt.Sprintf("event %q names %s, which two entities cover", event.Title, displayKey(key)))
		}
		if event.OccurredAt != nil {
			key := event.PrimaryRecordID() + "|" + event.OccurredAt.UTC().Format("2006-01-02T15:04") + "|" + entities.NameKey(event.Title)
			if other, ok := eventKeys[key]; ok {
				duplicateEvents = append(duplicateEvents, fmt.Sprintf("%q and %q are the same event from the same message; merge them", other, event.Title))
			}
			eventKeys[key] = event.Title
		}
	}
	counts.Events = len(includedEvents)
	add("events_have_time", untimed, "every event has a time")
	add("events_have_sources", unsourced, "every event rests on at least one source message")
	add("events_source_clock", unclocked, "every event carries its sources' availability, so an as-lived reader cannot see it early")
	add("event_entities_resolve", unresolved, "every entity an event names resolves to exactly one entity")
	add("events_unique", duplicateEvents, "no two events duplicate each other")
	// 13. Bounded.
	var bounds []string
	if len(included) > MaxEntitiesPerCommit {
		bounds = append(bounds, fmt.Sprintf("%d entities exceed the %d-per-commit bound", len(included), MaxEntitiesPerCommit))
	}
	if len(includedEvents) > MaxEventsPerCommit {
		bounds = append(bounds, fmt.Sprintf("%d events exceed the %d-per-commit bound", len(includedEvents), MaxEventsPerCommit))
	}
	add("bounded", bounds, "the commit fits its bounds")

	report := Report{OK: true, Checks: checks, Digest: Digest(snapshot), Counts: counts}
	for _, check := range checks {
		if check.Status != Pass {
			report.OK = false
		}
	}
	return report
}

// writableAliases are the aliases a commit writes to registry.entity_alias:
// never a source-scoped "self", never the entity's own name.
func writableAliases(proposal entities.Proposal) []entities.Alias {
	var out []entities.Alias
	for _, alias := range proposal.Aliases {
		if alias.Scope != "" {
			continue
		}
		if !alias.IsAddress() && entities.NameKey(alias.Text) == entities.NameKey(proposal.Name) {
			continue
		}
		out = append(out, alias)
	}
	return out
}

// WritableAliases is exported for the commit Activity so validation and the
// writer agree on exactly which aliases are written.
func WritableAliases(proposal entities.Proposal) []entities.Alias { return writableAliases(proposal) }

func displayKey(key string) string {
	switch {
	case strings.HasPrefix(key, "participant:"):
		value := strings.TrimPrefix(key, "participant:")
		if at := strings.Index(value, "@"); at >= 0 && strings.HasPrefix(value, "self") {
			return "the device owner of this source"
		}
		return value
	case strings.HasPrefix(key, "name:"):
		parts := strings.SplitN(key, ":", 3)
		if len(parts) == 3 {
			return fmt.Sprintf("%q", parts[2])
		}
	}
	return key
}

func firstN(values []string, n int) []string {
	if len(values) <= n {
		return values
	}
	return values[:n]
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

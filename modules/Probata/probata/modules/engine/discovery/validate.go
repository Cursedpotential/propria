// Byline: Codex · GPT-6.1-sol · 2026-10-11
package discovery

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"math"
	"sort"
	"strings"
	"unicode/utf8"
)

// ID admits a canonical nonnil UUID; input string, output boolean, no effects.
func ID(s string) bool { u, e := uuid.Parse(s); return e == nil && u != uuid.Nil && u.String() == s }
func bounded(s string, n int, required bool) bool {
	return utf8.ValidString(s) && len(s) <= n && (!required || strings.TrimSpace(s) != "") && !strings.ContainsRune(s, 0)
}

// Validate checks one explicit case identity and operating mode without persistence.
func (s Scope) Validate() error {
	if (s.Mode != "LIVE" && s.Mode != "DEV") || !ID(s.MatterID) || !ID(s.CourtCaseID) {
		return ErrScope
	}
	return nil
}

// Validate checks authenticated operator identity, separate from source attribution.
func (a Actor) Validate() error {
	if !bounded(a.SubjectUID, 200, true) || !bounded(a.Username, 200, true) {
		return ErrInvalid
	}
	return nil
}

// Validate bounds source filters and requires an explicit horizon before historical retrieval.
func (f Filters) Validate() error {
	if f.Perspective != "as_lived" && f.Perspective != "hindsight" {
		return fmt.Errorf("%w: explicit perspective required", ErrInvalid)
	}
	if f.Perspective == "as_lived" && (f.Cutoff == nil || f.Cutoff.IsZero()) {
		return fmt.Errorf("%w: historical cutoff required", ErrInvalid)
	}
	if f.StartAt != nil && f.EndAt != nil && f.StartAt.After(*f.EndAt) {
		return ErrInvalid
	}
	return stringList(append(append([]string{}, f.Participants...), f.Platforms...), 40, 200)
}
func stringList(ss []string, max, n int) error {
	if len(ss) > max {
		return ErrInvalid
	}
	for _, s := range ss {
		if !bounded(s, n, true) {
			return ErrInvalid
		}
	}
	return nil
}

// Validate rejects unbounded requests and caller-controlled input services before any write.
func (r RunRequest) Validate() error {
	if !ID(r.RunID) || r.Scope.Validate() != nil || r.Actor.Validate() != nil || r.Filters.Validate() != nil || !bounded(r.DatasetID, 200, true) || !bounded(r.SelectionRevision, 200, true) || (r.InputBackend != "postgresql" && r.InputBackend != "surrealdb") || len(r.GenerationIDs) < 1 || len(r.GenerationIDs) > 32 || r.MaxRecords < 1 || r.MaxRecords > MaxRecords || r.MaxCandidates < 1 || r.MaxCandidates > MaxCandidates || len(r.Hypotheses) > 40 {
		return ErrInvalid
	}
	seen := map[string]bool{}
	for _, id := range r.GenerationIDs {
		if !ID(id) || seen[id] {
			return ErrInvalid
		}
		seen[id] = true
	}
	seen = map[string]bool{}
	for _, h := range r.Hypotheses {
		if !bounded(h.HypothesisID, 200, true) || !bounded(h.Text, 4000, true) || !bounded(h.Category, 200, false) || seen[h.HypothesisID] || stringList(h.Prototypes, 20, 2000) != nil {
			return ErrInvalid
		}
		seen[h.HypothesisID] = true
	}
	raw, e := json.Marshal(r)
	if e != nil || len(raw) > 49152 {
		return ErrInvalid
	}
	return nil
}

// Normalize fills finite defaults only at the create boundary; existing durable modes are never inferred.
func (r *RunRequest) Normalize() {
	if r.MaxRecords == 0 {
		r.MaxRecords = MaxRecords
	}
	if r.MaxCandidates == 0 {
		r.MaxCandidates = MaxCandidates
	}
	if r.Filters.Perspective == "" {
		r.Filters.Perspective = "hindsight"
	}
	sort.Strings(r.GenerationIDs)
}

// Selection matches a request to an immutable server definition and an explicit generation subset.
func Selection(r RunRequest, datasets []Dataset) (Dataset, error) {
	for _, d := range datasets {
		if d.DatasetID != r.DatasetID {
			continue
		}
		if d.Scope != r.Scope || d.InputBackend != r.InputBackend || d.SelectionRevision != r.SelectionRevision {
			return Dataset{}, ErrScope
		}
		allow := map[string]bool{}
		for _, id := range d.GenerationIDs {
			allow[id] = true
		}
		for _, id := range r.GenerationIDs {
			if !allow[id] {
				return Dataset{}, ErrScope
			}
		}
		return d, nil
	}
	return Dataset{}, ErrScope
}

// ValidateDatasets admits only server-bound, scoped definitions with bounded IDs and no ambiguous duplicates.
func ValidateDatasets(ds []Dataset) error {
	if len(ds) > 100 {
		return ErrInvalid
	}
	seen := map[string]bool{}
	for _, d := range ds {
		if !bounded(d.DatasetID, 200, true) || !bounded(d.Label, 400, true) || !bounded(d.SelectionRevision, 200, true) || d.Scope.Validate() != nil || seen[d.DatasetID] || len(d.GenerationIDs) < 1 || len(d.GenerationIDs) > 32 || (d.InputBackend != "postgresql" && d.InputBackend != "surrealdb") {
			return ErrInvalid
		}
		seen[d.DatasetID] = true
		for _, id := range d.GenerationIDs {
			if !ID(id) {
				return ErrInvalid
			}
		}
	}
	return nil
}

// Fingerprint identifies a derived JSON value for idempotency; it is never a custody or source-admission hash.
func Fingerprint(v any) string { b, _ := json.Marshal(v); return DigestBytes(b) }

// DigestBytes fingerprints exact derivative bytes across languages; it never admits or hashes source custody.
func DigestBytes(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }

// Episodes uses explicit links or a source-scoped native conversation; people/time similarity never creates membership.
func Episodes(rs []SourceRecord) []Episode {
	out := []Episode{}
	index := map[string]int{}
	for _, r := range rs {
		key := "record:" + r.RecordID
		if r.ConversationID != "" {
			key = "conversation:" + r.SourceVersionID + ":" + r.ConversationID
		}
		if r.ExplicitEpisodeID != "" {
			key = "link:" + r.ExplicitEpisodeID
		}
		id := "episode:" + Fingerprint(key)[:32]
		i, ok := index[id]
		if !ok {
			i = len(out)
			index[id] = i
			out = append(out, Episode{EpisodeID: id, RecordIDs: []string{}})
		}
		out[i].RecordIDs = append(out[i].RecordIDs, r.RecordID)
	}
	return out
}

// Validate checks one append-only feedback act; skip and attribution corrections are distinct from label negatives.
func (f FeedbackRequest) Validate() error {
	switch f.Kind {
	case "confirm", "reject", "correct_labels", "wrong_person", "skip", "reported_speech", "missing_context", "partial_match", "correct_episode", "free_text", "relevant", "irrelevant", "wrong_behavior":
	default:
		return ErrInvalid
	}
	if !ID(f.FeedbackID) || !bounded(f.CandidateID, 200, true) || f.ExpectedRevision < 1 || !bounded(f.Note, 4000, false) || stringList(f.Labels, 40, 200) != nil || stringList(f.AttributedActorIDs, 40, 200) != nil || len(f.RecordIDs) > MaxRecords || len(f.EpisodeGroups) > 100 {
		return ErrInvalid
	}
	for _, id := range f.RecordIDs {
		if !ID(id) {
			return ErrInvalid
		}
	}
	seen := map[string]bool{}
	for _, g := range f.EpisodeGroups {
		if len(g) < 1 || len(g) > MaxRecords {
			return ErrInvalid
		}
		for _, id := range g {
			if !ID(id) || seen[id] {
				return ErrInvalid
			}
			seen[id] = true
		}
	}
	if f.Kind == "correct_episode" && len(f.EpisodeGroups) == 0 {
		return ErrInvalid
	}
	return nil
}

// ValidateResult verifies cited spans and episode membership against the immutable input before canonical score persistence.
func ValidateResult(result Result, b Batch) error {
	if result.Schema != ResultSchema || result.RunID != b.RunID || result.FeedbackRevision != b.FeedbackRevision || !bounded(result.ModelVersion, 200, true) || len(result.Candidates) > b.MaxCandidates {
		return ErrInvalid
	}
	records := map[string]SourceRecord{}
	for _, r := range b.Records {
		records[r.RecordID] = r
	}
	episodes := map[string]map[string]bool{}
	for _, e := range b.Episodes {
		m := map[string]bool{}
		for _, id := range e.RecordIDs {
			m[id] = true
		}
		episodes[e.EpisodeID] = m
	}
	seen := map[string]bool{}
	for _, c := range result.Candidates {
		if !bounded(c.CandidateID, 200, true) || seen[c.CandidateID] || episodes[c.EpisodeID] == nil || !bounded(c.Title, 500, true) || !bounded(c.Summary, 8000, false) || math.IsNaN(c.Score) || math.IsInf(c.Score, 0) || c.Score < 0 || c.Score > 1 || stringList(c.Labels, 40, 200) != nil || stringList(c.SpeakerIDs, 40, 200) != nil || stringList(c.AttributedActorIDs, 40, 200) != nil || len(c.Citations) < 1 || len(c.Citations) > 100 || !bounded(c.AttributionBasis, 4000, false) || !bounded(c.Rationale, 8000, false) || stringList(c.Uncertainty, 40, 2000) != nil {
			return ErrInvalid
		}
		seen[c.CandidateID] = true
		for _, scores := range []map[string]float64{c.LabelScores, c.DetectorScores} {
			if len(scores) > 40 {
				return ErrInvalid
			}
			for k, v := range scores {
				if !bounded(k, 200, true) || math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > 1 {
					return ErrInvalid
				}
			}
		}
		for _, cite := range c.Citations {
			r, ok := records[cite.RecordID]
			if !ok || !episodes[c.EpisodeID][cite.RecordID] || cite.StartChar < 0 || cite.EndChar <= cite.StartChar || cite.EndChar > utf8.RuneCountInString(r.Text) {
				return ErrInvalid
			}
			runes := []rune(r.Text)
			if string(runes[cite.StartChar:cite.EndChar]) != cite.Quote {
				return ErrInvalid
			}
			found := false
			for _, l := range r.Locators {
				if l.LocatorID == cite.LocatorID && l.SourceVersionID == r.SourceVersionID {
					found = true
				}
			}
			if !found {
				return ErrInvalid
			}
		}
	}
	return nil
}

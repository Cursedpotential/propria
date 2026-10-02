// Byline: Claude Code · Opus 5.5 · 2026-10-02
//
// TEMPORARY, NOT TO BE MERGED: a local stand-in with the exact API that the
// weaviate-first change publishes in engine/disclosure (owner 2026-10-02: one
// implementation both stages call). It lets this branch compile and be tested
// until that commit lands; then this file goes and every use switches to
// disclosure.Resolution / disclosure.ResolvedIdentifier.
package firstparty

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

const (
	selfMarker        = "self"
	tierBasis         = "owner_participant:v1"
	tierContemporary  = "contemporaneous"
	tierDiscoveredVal = "discovered"
)

// ResolvedIdentifier mirrors disclosure.ResolvedIdentifier.
type ResolvedIdentifier struct {
	Raw        string `json:"raw"`
	Normalized string `json:"normalized"`
	EntityID   string `json:"entity_id"`
	IsOwner    bool   `json:"is_owner"`
}

// Resolution mirrors disclosure.Resolution.
type Resolution struct {
	Basis               string               `json:"basis"`
	OwnerPersonID       string               `json:"owner_person_id"`
	PerspectivePersonID string               `json:"perspective_person_id"`
	Identifiers         []ResolvedIdentifier `json:"identifiers"`
}

// Validate mirrors disclosure.Resolution.Validate.
func (r Resolution) Validate() error {
	if strings.TrimSpace(r.OwnerPersonID) == "" || strings.TrimSpace(r.PerspectivePersonID) == "" {
		return errors.New("participant resolution requires owner and perspective person ids")
	}
	return nil
}

// Lookup mirrors disclosure.Resolution.Lookup.
func (r Resolution) Lookup(raw string) (ResolvedIdentifier, bool) {
	trimmed := strings.TrimSpace(raw)
	if strings.EqualFold(trimmed, selfMarker) {
		return ResolvedIdentifier{Raw: trimmed, EntityID: r.PerspectivePersonID,
			IsOwner: strings.EqualFold(r.OwnerPersonID, r.PerspectivePersonID)}, true
	}
	for _, identifier := range r.Identifiers {
		if identifier.Raw == trimmed {
			return identifier, true
		}
	}
	return ResolvedIdentifier{}, false
}

// ForMessage mirrors disclosure.Resolution.ForMessage.
func (r Resolution) ForMessage(sender string, recipients []string) (bool, string, string, error) {
	took := false
	for _, raw := range append([]string{sender}, recipients...) {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		identifier, ok := r.Lookup(raw)
		if !ok {
			return false, "", "", fmt.Errorf("identifier %q is not in the participant resolution", raw)
		}
		took = took || identifier.IsOwner
	}
	if took {
		return true, tierContemporary, tierBasis, nil
	}
	return false, tierDiscoveredVal, tierBasis, nil
}

// resolutionDigest binds a plan to the exact recorded resolution.
func resolutionDigest(r Resolution) string {
	identifiers := append([]ResolvedIdentifier(nil), r.Identifiers...)
	sort.Slice(identifiers, func(a, b int) bool { return identifiers[a].Raw < identifiers[b].Raw })
	copyOf := r
	copyOf.Identifiers = identifiers
	encoded, _ := json.Marshal(copyOf)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

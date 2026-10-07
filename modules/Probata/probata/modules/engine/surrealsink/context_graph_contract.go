// Byline: Codex | GPT-6.1-sol | 2026-10-07
package surrealsink

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"sort"
	"strings"
	"time"
)

// MaxContextGraphBundleBytes bounds each batch; callers must batch without truncating sources.
const MaxContextGraphBundleBytes = 384 << 10

var contextNodeKinds = []string{"ctx_source", "ctx_source_version", "ctx_content_unit", "ctx_entity", "ctx_statement", "ctx_proposition", "ctx_event_account"}
var contextEdgeKinds = []string{"has_version", "contains", "derived_from", "authored_by_asserted", "spoken_by_asserted", "mentions", "asserts", "describes", "depends_on", "before"}

// ContextGraphScope identifies the server-admitted case/policy/service scope.
// Inputs are existing identifiers; output is a scoped value, with no I/O or grant.
// Pick after admission rather than treating bundle JSON as authority.
type ContextGraphScope struct {
	MatterID         string `json:"matter_id"`
	CaseID           string `json:"case_id"`
	AccessPolicyID   string `json:"access_policy_id"`
	CreatedByService string `json:"created_by_service"`
}

// ContextSourcePin preserves root source/version/hash/locator without copying original bytes.
// Inputs are retained ingestion coordinates; output is an immutable citation value.
// No I/O occurs; pick for every node and relation, including uncited dependencies.
type ContextSourcePin struct {
	SourceID        string `json:"source_id"`
	SourceVersionID string `json:"source_version_id"`
	SourceHash      string `json:"source_hash"`
	Locator         string `json:"locator"`
	ValidationRef   string `json:"validation_ref,omitempty"`
}

// ContextGraphNode carries one existing source, turn, speaker or extracted account.
// Inputs are bounded derivative content or immutable full-body references and root pins.
// Output is a typed projection value with no I/O. Pick for source-asserted accounts,
// never synthesized truth, canonical person identity or formal completed artifacts.
type ContextGraphNode struct {
	NodeID              string             `json:"node_id"`
	Kind                string             `json:"kind"`
	DerivativeKind      string             `json:"derivative_kind,omitempty"`
	SourceOrigin        string             `json:"source_origin,omitempty"`
	SourcePins          []ContextSourcePin `json:"source_pins"`
	Body                string             `json:"body,omitempty"`
	BodyRef             string             `json:"body_ref,omitempty"`
	BodyHash            string             `json:"body_hash,omitempty"`
	TransformationRefs  []string           `json:"transformation_refs,omitempty"`
	InputNodeIDs        []string           `json:"input_node_ids,omitempty"`
	SourceAvailableFrom *time.Time         `json:"source_available_from,omitempty"`
	OccurredAt          *time.Time         `json:"occurred_at,omitempty"`
	TimePrecision       string             `json:"time_precision,omitempty"`
	RelativeAnchorRef   string             `json:"relative_anchor_ref,omitempty"`
}

// ContextGraphEdge asserts one named source-cited relation between bundle nodes.
// Inputs are endpoints and root pins; output is a typed relation, with no I/O.
// Pick for retained extraction assertions and complete created-work dependencies.
type ContextGraphEdge struct {
	EdgeID     string             `json:"edge_id"`
	Kind       string             `json:"kind"`
	FromNodeID string             `json:"from_node_id"`
	ToNodeID   string             `json:"to_node_id"`
	SourcePins []ContextSourcePin `json:"source_pins"`
}

// ContextGraphBundle is one immutable bounded downstream projection generation.
// Inputs are already ingested/extracted real output, admitted scope and complete
// dependencies. Output is a typed packet; no model call, ingestion or I/O occurs.
// Pick for ProjectContextGraph; send its external locator through Temporal history.
type ContextGraphBundle struct {
	Scope            ContextGraphScope  `json:"scope"`
	GenerationID     string             `json:"generation_id"`
	ExtractionRunRef string             `json:"extraction_run_ref"`
	Nodes            []ContextGraphNode `json:"nodes"`
	Edges            []ContextGraphEdge `json:"edges"`
}

func graphIdentifier(s string) bool {
	return len(s) > 0 && len(s) <= 240 && strings.TrimSpace(s) == s && !strings.ContainsAny(s, "\x00\r\n")
}
func graphExternalRef(s string) bool {
	return len(s) > 0 && len(s) <= 2000 && strings.TrimSpace(s) == s && !strings.ContainsAny(s, "\x00\r\n")
}
func graphDigest(s string) bool {
	b, e := hex.DecodeString(s)
	return e == nil && len(b) == 32 && s == strings.ToLower(s)
}
func graphAllowed(s string, kinds []string) bool {
	for _, k := range kinds {
		if s == k {
			return true
		}
	}
	return false
}
func graphHash(v any) string {
	b, _ := json.Marshal(v)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
func validateGraphPins(pins []ContextSourcePin) error {
	if len(pins) == 0 || len(pins) > 64 {
		return errors.New("bounded root source pins required")
	}
	for _, p := range pins {
		if !graphIdentifier(p.SourceID) || !graphIdentifier(p.SourceVersionID) || !graphDigest(p.SourceHash) || len(p.Locator) == 0 || len(p.Locator) > 2000 || strings.TrimSpace(p.Locator) == "" || (p.ValidationRef != "" && !graphIdentifier(p.ValidationRef)) {
			return errors.New("invalid root source pin")
		}
	}
	return nil
}

// Validate checks the exact functional packet before schema inspection or persistence.
// Inputs are this bundle; output is a static error or nil, with no I/O. Pick before
// projection and schema generation; completeness outside supplied inputs remains adapter-owned.
func (b ContextGraphBundle) Validate() error {
	for _, s := range []string{b.Scope.MatterID, b.Scope.CaseID, b.Scope.AccessPolicyID, b.Scope.CreatedByService, b.GenerationID} {
		if !graphIdentifier(s) {
			return errors.New("explicit graph scope/generation/extraction identity required")
		}
	}
	if !graphExternalRef(b.ExtractionRunRef) {
		return errors.New("bounded external extraction reference required")
	}
	if len(b.Nodes) == 0 || len(b.Nodes) > 128 || len(b.Edges) == 0 || len(b.Edges) > 256 {
		return errors.New("graph node/relation count outside bound")
	}
	raw, e := json.Marshal(b)
	if e != nil || len(raw) > MaxContextGraphBundleBytes {
		return errors.New("graph bundle exceeds byte bound")
	}
	nodes := map[string]ContextGraphNode{}
	pins := map[string]bool{}
	for _, n := range b.Nodes {
		if !graphIdentifier(n.NodeID) || !graphAllowed(n.Kind, contextNodeKinds) || nodes[n.NodeID].NodeID != "" {
			return errors.New("invalid or duplicate graph node")
		}
		if e := validateGraphPins(n.SourcePins); e != nil {
			return e
		}
		nodeJSON, _ := json.Marshal(n)
		if len(nodeJSON) > 65536 {
			return errors.New("graph node serialized payload exceeds bound")
		}
		if len(n.Body) > 32<<10 || len(n.BodyRef) > 2000 || len(n.DerivativeKind) > 80 || len(n.SourceOrigin) > 80 || len(n.TimePrecision) > 80 || len(n.TransformationRefs) > 64 || len(n.InputNodeIDs) > 128 {
			return errors.New("graph node field exceeds bound")
		}
		if n.BodyRef != "" && !graphDigest(n.BodyHash) {
			return errors.New("immutable external body reference requires hash")
		}
		if n.BodyHash != "" && !graphDigest(n.BodyHash) {
			return errors.New("invalid derivative body hash")
		}
		if n.Body != "" && n.BodyHash != "" {
			sum := sha256.Sum256([]byte(n.Body))
			if hex.EncodeToString(sum[:]) != n.BodyHash {
				return errors.New("derivative body hash mismatch")
			}
		}
		for _, ref := range n.TransformationRefs {
			if !graphExternalRef(ref) {
				return errors.New("invalid transformation reference")
			}
		}
		if n.RelativeAnchorRef != "" && !graphIdentifier(n.RelativeAnchorRef) {
			return errors.New("invalid relative anchor reference")
		}
		if n.DerivativeKind == "created_work" && (n.Kind != "ctx_content_unit" || (n.Body == "" && n.BodyRef == "") || len(n.InputNodeIDs) == 0) {
			return errors.New("created work requires full body or pinned external body and complete input IDs")
		}
		for _, p := range n.SourcePins {
			pins[graphHash(p)] = true
		}
		nodes[n.NodeID] = n
	}
	edges := map[string]bool{}
	deps := map[string]map[string]bool{}
	for _, e := range b.Edges {
		if !graphIdentifier(e.EdgeID) || edges[e.EdgeID] || !graphAllowed(e.Kind, contextEdgeKinds) || nodes[e.FromNodeID].NodeID == "" || nodes[e.ToNodeID].NodeID == "" || e.FromNodeID == e.ToNodeID {
			return errors.New("invalid graph relation or endpoint")
		}
		if err := validateGraphPins(e.SourcePins); err != nil {
			return err
		}
		for _, p := range e.SourcePins {
			if !pins[graphHash(p)] {
				return errors.New("relation root pin is absent from graph nodes")
			}
		}
		edges[e.EdgeID] = true
		if e.Kind == "has_version" && (nodes[e.FromNodeID].Kind != "ctx_source" || nodes[e.ToNodeID].Kind != "ctx_source_version") {
			return errors.New("has_version endpoint kinds differ")
		}
		if (e.Kind == "authored_by_asserted" || e.Kind == "spoken_by_asserted") && nodes[e.ToNodeID].Kind != "ctx_entity" {
			return errors.New("speaker relation requires source-asserted entity")
		}
		if e.Kind == "before" {
			from, to := nodes[e.FromNodeID], nodes[e.ToNodeID]
			fromTurn := from.DerivativeKind == "source_turn" || from.DerivativeKind == "ai_source_turn"
			toTurn := to.DerivativeKind == "source_turn" || to.DerivativeKind == "ai_source_turn"
			if from.Kind != "ctx_content_unit" || to.Kind != "ctx_content_unit" || !fromTurn || !toTurn || from.OccurredAt == nil || to.OccurredAt == nil || !from.OccurredAt.Before(*to.OccurredAt) {
				return errors.New("before requires strictly ordered native source-turn timestamps")
			}
		}
		if e.Kind == "depends_on" {
			if deps[e.FromNodeID] == nil {
				deps[e.FromNodeID] = map[string]bool{}
			}
			if deps[e.FromNodeID][e.ToNodeID] {
				return errors.New("duplicate dependency relation")
			}
			deps[e.FromNodeID][e.ToNodeID] = true
		}
	}
	for _, n := range b.Nodes {
		if n.DerivativeKind != "created_work" {
			continue
		}
		expected := map[string]bool{}
		for _, id := range n.InputNodeIDs {
			if nodes[id].NodeID == "" || id == n.NodeID || expected[id] {
				return errors.New("invalid complete created-work input IDs")
			}
			expected[id] = true
		}
		if len(expected) != len(deps[n.NodeID]) {
			return errors.New("created-work dependency manifest differs from edges")
		}
		for id := range expected {
			if !deps[n.NodeID][id] {
				return errors.New("missing created-work dependency edge")
			}
		}
	}
	return nil
}

// DecodeContextGraphBundle reads one strict bounded private bundle without other effects.
// Input is a file/reader; output is a validated complete typed bundle. Pick for the
// CLI and retained-artifact adapters; unknown fields and trailing JSON are rejected.
func DecodeContextGraphBundle(r io.Reader) (ContextGraphBundle, error) {
	var b ContextGraphBundle
	raw, e := io.ReadAll(io.LimitReader(r, MaxContextGraphBundleBytes+1))
	if e != nil || len(raw) > MaxContextGraphBundleBytes {
		return b, errors.New("cannot read bounded graph bundle")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&b) != nil {
		return b, errors.New("invalid graph bundle JSON")
	}
	var trailing any
	if d.Decode(&trailing) != io.EOF {
		return b, errors.New("graph bundle has trailing JSON")
	}
	return b, b.Validate()
}

func graphCanonical(b ContextGraphBundle) ContextGraphBundle {
	b.Nodes = append([]ContextGraphNode(nil), b.Nodes...)
	b.Edges = append([]ContextGraphEdge(nil), b.Edges...)
	sort.Slice(b.Nodes, func(i, j int) bool { return b.Nodes[i].NodeID < b.Nodes[j].NodeID })
	sort.Slice(b.Edges, func(i, j int) bool { return b.Edges[i].EdgeID < b.Edges[j].EdgeID })
	return b
}

package flow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// ExtractionPreflight reports exact current targets and admission without dispatch.
// Inputs: stored source/generation/mode and workload counts. Output: bounded browser
// metadata. Effects: none; choose before starting a conversation extraction.
type ExtractionPreflight struct {
	Token           string                      `json:"token"`
	Eligible        bool                        `json:"eligible"`
	Blockers        []string                    `json:"blockers"`
	Targets         []ExtractionPreflightTarget `json:"targets"`
	Extractors      []ExtractorSpec             `json:"extractors"`
	CostEstimate    string                      `json:"cost_estimate"`
	RuntimeEstimate string                      `json:"runtime_estimate"`
	Bounds          []string                    `json:"bounds"`
}

// PreflightToken binds confirmation to the exact inspected pins, workload and selection.
// Inputs: read-only preflight. Output: deterministic SHA-256 token. Effects: none;
// pick at preview and dispatch so an enlarged or replaced target requires refresh.
func PreflightToken(check ExtractionPreflight) string {
	raw, _ := json.Marshal(struct {
		Targets    []ExtractionPreflightTarget
		Extractors []ExtractorSpec
		Bounds     []string
	}{check.Targets, check.Extractors, check.Bounds})
	hash := sha256.Sum256(raw)
	return hex.EncodeToString(hash[:])
}

// ExtractionPreflightTarget retains exact resolved source coordinates and text size.
// Inputs: one conversation resolution. Output: counts and durable RunRefs; no bodies.
// Effects: none; pick for an explicit selection, never infer mode from a browser flag.
type ExtractionPreflightTarget struct {
	WorkloadStatus           string          `json:"workload_status"`
	QueriedGenerations       int             `json:"queried_generations"`
	ResolvedGenerations      int             `json:"resolved_generations"`
	Ref                      ConversationRef `json:"ref"`
	Runs                     []RunRef        `json:"runs"`
	Messages                 int64           `json:"messages"`
	Characters               int64           `json:"characters"`
	UTF8Bytes                int64           `json:"utf8_bytes"`
	MaxMessageCharacters     int64           `json:"max_message_characters"`
	MaxMessageUTF8Bytes      int64           `json:"max_message_utf8_bytes"`
	AcquisitionRelationships []string        `json:"acquisition_relationships"`
}

// Byline: Claude Code · Opus 5.5 · 2026-09-25

// Byline: Codex · GPT-5 · 2026-10-05 (case-connected flow classification).
package repairplan

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Cursedpotential/probata/engine/stagegraph"
)

// Write classes. "original" does not exist on purpose: no registered tool may
// write the source it repairs.
const (
	WritesDerived = "derived"
	WritesNone    = "none"
)

// Output kinds decide how a plan re-enters Proffer.
const (
	// OutputDerivedObject is one new derived object; re-entry is one run.
	OutputDerivedObject = "derived_object"
	// OutputExistingObject is another existing object (a re-pointed source);
	// re-entry is one run on it.
	OutputExistingObject = "existing_object"
	// OutputDerivedChunkFolder is a manifest plus a folder of derived chunks;
	// re-entry is one batch over the folder.
	OutputDerivedChunkFolder = "derived_chunk_folder"
)

// ToolSpec is a registered repair Activity: the public Tool plus what the
// validator and the workflow need to schedule it safely.
type ToolSpec struct {
	Tool
	// OutputKind is one of the Output* constants.
	OutputKind string
	// PreservesType means the output has the input's type (TypeSameAsInput).
	PreservesType bool
	// RepointsSource means the step replaces the plan's source with another
	// existing object instead of deriving one.
	RepointsSource bool
	// Streaming means the Activity streams its input with bounded memory.
	Streaming bool
	// FlowName is the declared n8n flow binding a NeedsN8N step runs through
	// run_n8n_flow_activity.
	FlowName string
	// Activity options. Every step is bounded; none retries forever.
	StartToClose time.Duration
	Heartbeat    time.Duration
	MaxAttempts  int32
}

// Registry is the ordered, validated set of repair tools.
type Registry struct {
	specs []ToolSpec
	byID  map[string]ToolSpec
}

// CaseFlowNames returns only registered repair tools' n8n bindings.
// Inputs: validated registry. Outputs: classification names. Effects: none.
func (r *Registry) CaseFlowNames() []string {
	var names []string
	if r == nil {
		return names
	}
	for _, spec := range r.specs {
		if spec.NeedsN8N {
			names = append(names, spec.FlowName)
		}
	}
	return names
}

// NewRegistry validates specs: unique ids, known write classes and output
// kinds, a params schema in the supported subset, bounded options, and a flow
// binding for every n8n tool.
func NewRegistry(specs ...ToolSpec) (*Registry, error) {
	registry := &Registry{byID: make(map[string]ToolSpec, len(specs))}
	for _, spec := range specs {
		if spec.ID == "" || spec.Description == "" {
			return nil, fmt.Errorf("repair tool %q needs an id and a description", spec.ID)
		}
		if _, dup := registry.byID[spec.ID]; dup {
			return nil, fmt.Errorf("repair tool %q is registered twice", spec.ID)
		}
		if spec.Writes != WritesDerived && spec.Writes != WritesNone {
			return nil, fmt.Errorf("repair tool %q has write class %q; only derived or none exist", spec.ID, spec.Writes)
		}
		switch spec.OutputKind {
		case OutputDerivedObject, OutputDerivedChunkFolder:
			if spec.Writes != WritesDerived {
				return nil, fmt.Errorf("repair tool %q derives output but declares writes %q", spec.ID, spec.Writes)
			}
		case OutputExistingObject:
			if spec.Writes != WritesNone || !spec.RepointsSource {
				return nil, fmt.Errorf("repair tool %q names an existing object, so it must write nothing and re-point", spec.ID)
			}
		default:
			return nil, fmt.Errorf("repair tool %q has unknown output kind %q", spec.ID, spec.OutputKind)
		}
		if len(spec.InputTypes) == 0 || len(spec.OutputTypes) == 0 {
			return nil, fmt.Errorf("repair tool %q must declare input and output types", spec.ID)
		}
		if spec.PreservesType != (spec.OutputTypes[0] == TypeSameAsInput) {
			return nil, fmt.Errorf("repair tool %q: same-as-input output and PreservesType must agree", spec.ID)
		}
		if _, err := parseParamsSchema(spec.ParamsSchema); err != nil {
			return nil, fmt.Errorf("repair tool %q: %w", spec.ID, err)
		}
		if spec.StartToClose <= 0 || spec.MaxAttempts <= 0 {
			return nil, fmt.Errorf("repair tool %q must declare a bounded timeout and attempt count", spec.ID)
		}
		if spec.NeedsN8N && spec.FlowName == "" {
			return nil, fmt.Errorf("repair tool %q needs n8n but names no flow binding", spec.ID)
		}
		registry.specs = append(registry.specs, spec)
		registry.byID[spec.ID] = spec
	}
	return registry, nil
}

// Tools returns the public descriptions in registry order.
func (r *Registry) Tools() []Tool {
	tools := make([]Tool, 0, len(r.specs))
	for _, spec := range r.specs {
		tool := spec.Tool
		tool.InputTypes = append([]string(nil), spec.InputTypes...)
		tool.OutputTypes = append([]string(nil), spec.OutputTypes...)
		tool.ParamsSchema = append(json.RawMessage(nil), spec.ParamsSchema...)
		tools = append(tools, tool)
	}
	return tools
}

// Lookup resolves one tool by registry id.
func (r *Registry) Lookup(id string) (ToolSpec, bool) {
	spec, ok := r.byID[id]
	return spec, ok
}

const (
	findOtherVersionSchema = `{"type":"object","additionalProperties":false,"properties":{` +
		`"max_candidates":{"type":"integer","minimum":1,"maximum":20,"default":5,"description":"How many verified copies to report; the largest becomes the plan's source."},` +
		`"require_larger":{"type":"boolean","default":false,"description":"Only accept a copy larger than the source, never merely a different hash."},` +
		`"include_quarantine":{"type":"boolean","default":false,"description":"Also accept copies the owner set aside under a _quarantine/ folder."}}}`
	noParamsSchema = `{"type":"object","additionalProperties":false,"properties":{}}`
)

// DefaultRegistry is the production registry: the three repair Activities of
// the ratified first increment. Every one runs directly on the proffer worker
// (needs_n8n false); an n8n-backed tool is a new entry with FlowName set.
func DefaultRegistry() *Registry {
	registry, err := NewRegistry(
		ToolSpec{
			Tool: Tool{
				ID: string(stagegraph.RepairFindOtherVersion),
				Description: "Find another copy of this file by name in the Case Bible catalog — same file name, a different hash or a larger size — " +
					"confirm it still exists, and use it as the source for the following steps. Reads only; writes nothing.",
				InputTypes: []string{TypeAny}, OutputTypes: []string{TypeSameAsInput},
				ParamsSchema: json.RawMessage(findOtherVersionSchema), Writes: WritesNone,
			},
			OutputKind: OutputExistingObject, PreservesType: true, RepointsSource: true, Streaming: true,
			StartToClose: 2 * time.Minute, MaxAttempts: 3,
		},
		ToolSpec{
			Tool: Tool{
				ID: string(stagegraph.RepairSalvageTruncatedXML),
				Description: "Stream a cut-off XML backup, keep every record up to the last complete one, close the document, and publish " +
					"it as a new, hashed derived copy (an exact byte prefix of the original plus the closing tag). The original is never written.",
				InputTypes: []string{TypeSMSBackupXML, TypeXML}, OutputTypes: []string{TypeSameAsInput},
				ParamsSchema: json.RawMessage(noParamsSchema), Writes: WritesDerived,
			},
			OutputKind: OutputDerivedObject, PreservesType: true, Streaming: true,
			StartToClose: 4 * time.Hour, Heartbeat: 2 * time.Minute, MaxAttempts: 3,
		},
		ToolSpec{
			Tool: Tool{
				ID: string(stagegraph.RepairLenientDecode),
				Description: "Decode the SMS backup with the SBV decoder in lenient mode: records it cannot read are set aside as rejects, " +
					"and if decoding stops on damaged bytes everything decoded before the stop is kept. Publishes derived NDJSON threads " +
					"with a hashed manifest. The original is never written.",
				InputTypes: []string{TypeSMSBackupXML}, OutputTypes: []string{TypeDerivedThreads},
				ParamsSchema: json.RawMessage(noParamsSchema), Writes: WritesDerived,
			},
			OutputKind: OutputDerivedChunkFolder, Streaming: true,
			StartToClose: 4 * time.Hour, Heartbeat: 2 * time.Minute, MaxAttempts: 3,
		},
	)
	if err != nil {
		panic("repairplan: default registry is invalid: " + err.Error())
	}
	return registry
}

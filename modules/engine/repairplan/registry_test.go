// Byline: Claude Code · Opus 5.5 · 2026-09-25

package repairplan

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Cursedpotential/probata/engine/stagegraph"
)

// GET tools must list every repair-capable Activity in exactly the contract's
// shape; all three first-increment tools run on the worker (needs_n8n false).
func TestDefaultRegistryListsTheThreeRepairActivitiesInContractShape(t *testing.T) {
	tools := DefaultRegistry().Tools()
	encoded, err := json.Marshal(ToolsResponse{Tools: tools})
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Tools []map[string]json.RawMessage `json:"tools"`
	}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Tools) != 3 {
		t.Fatalf("tools = %d, want 3", len(decoded.Tools))
	}
	wantKeys := []string{"id", "description", "input_types", "output_types", "params_schema", "writes", "needs_n8n"}
	for _, tool := range decoded.Tools {
		if len(tool) != len(wantKeys) {
			t.Fatalf("tool has keys %v, want exactly %v", keys(tool), wantKeys)
		}
		for _, key := range wantKeys {
			if _, ok := tool[key]; !ok {
				t.Fatalf("tool is missing %q: %s", key, encoded)
			}
		}
		if string(tool["needs_n8n"]) != "false" {
			t.Fatalf("first-increment tool needs n8n: %s", tool["id"])
		}
	}
	byID := map[string]Tool{}
	for _, tool := range tools {
		byID[tool.ID] = tool
	}
	if byID[string(stagegraph.RepairFindOtherVersion)].Writes != WritesNone ||
		byID[string(stagegraph.RepairSalvageTruncatedXML)].Writes != WritesDerived ||
		byID[string(stagegraph.RepairLenientDecode)].Writes != WritesDerived {
		t.Fatalf("write classes wrong: %+v", byID)
	}
	if got := byID[string(stagegraph.RepairLenientDecode)].OutputTypes; len(got) != 1 || got[0] != TypeDerivedThreads {
		t.Fatalf("lenient decode output types = %v", got)
	}
}

func keys(m map[string]json.RawMessage) []string {
	out := make([]string, 0, len(m))
	for key := range m {
		out = append(out, key)
	}
	return out
}

func validSpec(id string) ToolSpec {
	return ToolSpec{
		Tool: Tool{ID: id, Description: "d", InputTypes: []string{TypeAny}, OutputTypes: []string{TypeSameAsInput},
			ParamsSchema: json.RawMessage(noParamsSchema), Writes: WritesNone},
		OutputKind: OutputExistingObject, PreservesType: true, RepointsSource: true, Streaming: true,
		StartToClose: time.Minute, MaxAttempts: 1,
	}
}

func TestNewRegistryRefusesUnsafeTools(t *testing.T) {
	for name, mutate := range map[string]func(*ToolSpec){
		"writes the original":     func(s *ToolSpec) { s.Writes = "original" },
		"derives but writes none": func(s *ToolSpec) { s.OutputKind = OutputDerivedObject },
		"n8n without a flow":      func(s *ToolSpec) { s.NeedsN8N = true },
		"unbounded":               func(s *ToolSpec) { s.MaxAttempts = 0 },
		"no timeout":              func(s *ToolSpec) { s.StartToClose = 0 },
		"schema not an object":    func(s *ToolSpec) { s.ParamsSchema = json.RawMessage(`{"type":"string"}`) },
		"schema outside subset":   func(s *ToolSpec) { s.ParamsSchema = json.RawMessage(`{"type":"object","oneOf":[]}`) },
		"no types":                func(s *ToolSpec) { s.InputTypes = nil },
		"type claim disagrees":    func(s *ToolSpec) { s.PreservesType = false },
		"unknown output kind":     func(s *ToolSpec) { s.OutputKind = "somewhere" },
	} {
		spec := validSpec("x.tool")
		mutate(&spec)
		if _, err := NewRegistry(spec); err == nil {
			t.Fatalf("%s: registry accepted the tool", name)
		}
	}
	if _, err := NewRegistry(validSpec("a"), validSpec("a")); err == nil || !strings.Contains(err.Error(), "twice") {
		t.Fatalf("duplicate id accepted: %v", err)
	}
	flow := validSpec("n8n.tool")
	flow.NeedsN8N, flow.FlowName = true, "ocr_page"
	if _, err := NewRegistry(flow); err != nil {
		t.Fatalf("an n8n tool with a flow binding must register: %v", err)
	}
}

func TestParamsSchemaValidation(t *testing.T) {
	schema := json.RawMessage(findOtherVersionSchema)
	for _, ok := range []string{``, `null`, `{}`, `{"max_candidates":1}`, `{"max_candidates":20,"require_larger":true}`} {
		if err := validateParams(schema, json.RawMessage(ok)); err != nil {
			t.Fatalf("%q rejected: %v", ok, err)
		}
	}
	for _, bad := range []string{`[]`, `"x"`, `{"max_candidates":0}`, `{"max_candidates":21}`, `{"max_candidates":2.5}`,
		`{"max_candidates":"5"}`, `{"require_larger":"yes"}`, `{"path":"/etc/passwd"}`, `{bad json`} {
		if err := validateParams(schema, json.RawMessage(bad)); err == nil {
			t.Fatalf("%q accepted", bad)
		}
	}
	enum := json.RawMessage(`{"type":"object","required":["mode"],"properties":{"mode":{"type":"string","enum":["a","b"]}}}`)
	if err := validateParams(enum, json.RawMessage(`{"mode":"a"}`)); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{`{}`, `{"mode":"c"}`} {
		if err := validateParams(enum, json.RawMessage(bad)); err == nil {
			t.Fatalf("%q accepted", bad)
		}
	}
	if canonicalParams(json.RawMessage(`{ "b":1, "a" :2 }`)) != `{"a":2,"b":1}` || canonicalParams(nil) != "{}" {
		t.Fatal("canonical params are not canonical")
	}
}

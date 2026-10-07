package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/surrealsink"
)

// privateOutputTestDir retains tiny format-only files for owner cleanup, without deletion.
// Inputs are the test handle; output is an isolated temporary directory. Effects
// create that directory only; pick for preservation tests without live graph/corpus data.
func privateOutputTestDir(t *testing.T) string {
	t.Helper()
	dir, e := os.MkdirTemp("", "context-graph-private-format-")
	if e != nil {
		t.Fatal(e)
	}
	return dir
}

// TestPrivateTraversalOutputPreservesDifferentBytesAndReplaysIdentical checks file safety.
// Inputs are a small typed format value; output is assertions. Effects create only
// temporary format files, with no live database, source corpus or delivery proof.
func TestPrivateTraversalOutputPreservesDifferentBytesAndReplaysIdentical(t *testing.T) {
	path := filepath.Join(privateOutputTestDir(t), "private-traversal.json")
	view := surrealsink.ContextGraphReadback{Receipt: surrealsink.ContextGraphReceipt{GenerationRef: "format-generation-ref", CheckpointRef: "format-checkpoint"}, Bundle: surrealsink.ContextGraphBundle{Nodes: []surrealsink.ContextGraphNode{{NodeID: "format-node", Body: "private format body", SourcePins: []surrealsink.ContextSourcePin{{SourceID: "format-source", Locator: "format:span"}}}}}}
	receipt, e := writePrivateTraversal(path, view)
	if e != nil {
		t.Fatal(e)
	}
	again, e := writePrivateTraversal(path, view)
	if e != nil || again != receipt {
		t.Fatal("identical output did not replay")
	}
	saved, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	var parsed surrealsink.ContextGraphReadback
	if json.Unmarshal(saved, &parsed) != nil || parsed.Bundle.Nodes[0].Body != view.Bundle.Nodes[0].Body || parsed.Bundle.Nodes[0].SourcePins[0].Locator != "format:span" {
		t.Fatal("actual typed payload/citations missing")
	}
	printed, _ := json.Marshal(receipt)
	if strings.Contains(string(printed), "private format body") || strings.Contains(string(printed), "format:span") {
		t.Fatal("private body/citation leaked to stdout receipt")
	}
	if runtime.GOOS != "windows" {
		info, _ := os.Stat(path)
		if info.Mode().Perm() != 0600 {
			t.Fatal("output permissions are not 0600")
		}
	}
	view.Bundle.Nodes[0].Body = "changed private format body"
	if _, e = writePrivateTraversal(path, view); e == nil {
		t.Fatal("different existing output overwritten")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(saved, after) {
		t.Fatal("existing output bytes changed")
	}
	if _, e = writePrivateTraversal("relative-output.json", view); e == nil {
		t.Fatal("relative output admitted")
	}
	if _, e = writePrivateTraversal(filepath.Dir(path), view); e == nil {
		t.Fatal("directory output admitted")
	}
}

// TestReadOutputFlagRejectsOtherOperationsBeforeIO checks CLI admission only.
// Inputs are argument vectors; output is errors and empty stdout, with no effects.
func TestReadOutputFlagRejectsOtherOperationsBeforeIO(t *testing.T) {
	for _, args := range [][]string{{"read", "--bundle", "unread", "--output", "relative.json"}, {"project", "--bundle", "unread", "--output", filepath.Join(os.TempDir(), "unused.json")}} {
		var out bytes.Buffer
		if run(args, &out) == nil || out.Len() != 0 {
			t.Fatal("invalid output flag admitted")
		}
	}
}

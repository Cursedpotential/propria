package humancontext

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestNativeChunkPagePreservesMembersAndContinuation verifies actual chunk files are paged losslessly.
// Inputs: synthetic native manifest/chunks. Outputs: full field and continuation assertions.
// Effects: temporary fixture files only; choose before exposing native chunks through the BFF.
func TestNativeChunkPagePreservesMembersAndContinuation(t *testing.T) {
	root := t.TempDir()
	chunks := filepath.Join(root, "chunks.jsonl")
	manifest := filepath.Join(root, "manifest.json")
	raw := []byte("{\"chunk_id\":\"a\",\"members\":[{\"ordinal\":1}],\"unrecognized\":true}\n{\"chunk_id\":\"b\",\"members\":[{\"ordinal\":2}]}\n")
	if err := os.WriteFile(chunks, raw, 0600); err != nil {
		t.Fatal(err)
	}
	m := Manifest{ContractVersion: Contract, Source: Source{SourceVersionRef: "actual-native-version"}, ChunksRef: fileRef(chunks), Chunks: 2}
	encoded, _ := json.Marshal(m)
	if err := os.WriteFile(manifest, encoded, 0600); err != nil {
		t.Fatal(err)
	}
	b := &Bundles{Root: root}
	first, err := b.ReadChunkPage(context.Background(), fileRef(manifest), 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if first["next_offset"] != int64(1) {
		t.Fatalf("continuation %#v", first)
	}
	var row map[string]any
	json.Unmarshal(first["items"].([]json.RawMessage)[0], &row)
	if row["unrecognized"] != true || len(row["members"].([]any)) != 1 {
		t.Fatal("native fields lost")
	}
	last, err := b.ReadChunkPage(context.Background(), fileRef(manifest), 1, 1)
	if err != nil || last["next_offset"] != nil {
		t.Fatalf("final page %#v %v", last, err)
	}
	if _, err = b.ReadChunkPage(context.Background(), fileRef(manifest), -1, 1); err == nil {
		t.Fatal("negative offset accepted")
	}
}

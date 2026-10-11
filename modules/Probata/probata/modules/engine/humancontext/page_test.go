package humancontext

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNativePageSerializedBudgetContinuesWithoutSkipping exercises real escaped JSON response sizes.
// Inputs: seven large synthetic rows. Outputs: bounded pages in exact order including final page.
// Effects: disposable test bundle files only, no source/metadata/search writes.
func TestNativePageSerializedBudgetContinuesWithoutSkipping(t *testing.T) {
	b := &Bundles{Root: t.TempDir()}
	recordsPath := filepath.Join(b.Root, "records.jsonl")
	offsetsPath := filepath.Join(b.Root, "offsets.bin")
	records, err := os.Create(recordsPath)
	if err != nil {
		t.Fatal(err)
	}
	offsets, err := os.Create(offsetsPath)
	if err != nil {
		t.Fatal(err)
	}
	position := uint64(0)
	for ordinal := int64(1); ordinal <= 7; ordinal++ {
		native, _ := json.Marshal(map[string]string{"body": strings.Repeat("<", 130000)})
		raw, err := json.Marshal(Record{Ordinal: ordinal, Status: "complete", Native: native})
		if err != nil {
			t.Fatal(err)
		}
		if err := binary.Write(offsets, binary.LittleEndian, position); err != nil {
			t.Fatal(err)
		}
		n, err := records.Write(append(raw, '\n'))
		if err != nil {
			t.Fatal(err)
		}
		position += uint64(n)
	}
	records.Close()
	offsets.Close()
	m := Manifest{ContractVersion: Contract, Source: Source{SourceVersionRef: "synthetic"}, RecordsRef: fileRef(recordsPath), OffsetsRef: fileRef(offsetsPath), Records: 7}
	raw, _ := json.Marshal(m)
	manifestPath := filepath.Join(b.Root, "manifest.json")
	os.WriteFile(manifestPath, raw, 0640)
	offset := int64(0)
	ordinals := []int64{}
	pages := 0
	for {
		page, err := b.ReadPage(context.Background(), fileRef(manifestPath), offset, MaxPageRecords)
		if err != nil {
			t.Fatal(err)
		}
		wire, err := json.Marshal(page)
		if err != nil {
			t.Fatal(err)
		}
		if len(wire)+1 > MaxPageBytes {
			t.Fatalf("serialized response exceeds budget: %d", len(wire))
		}
		for _, row := range page.Records {
			ordinals = append(ordinals, row.Ordinal)
		}
		pages++
		if page.NextOffset == nil {
			break
		}
		if *page.NextOffset <= offset {
			t.Fatal("nonadvancing continuation")
		}
		offset = *page.NextOffset
	}
	if pages < 2 || len(ordinals) != 7 {
		t.Fatalf("coverage %v pages %d", ordinals, pages)
	}
	for j, ordinal := range ordinals {
		if ordinal != int64(j+1) {
			t.Fatalf("skipped/duplicate row %v", ordinals)
		}
	}
	oversized := `{"ordinal":1,"status":"complete","native":{"body":"` + strings.Repeat("<", 800000) + `"}}` + "\n"
	if err := os.WriteFile(recordsPath, []byte(oversized), 0640); err != nil {
		t.Fatal(err)
	}
	m.Records = 1
	raw, _ = json.Marshal(m)
	if err := os.WriteFile(manifestPath, raw, 0640); err != nil {
		t.Fatal(err)
	}
	if _, err := b.ReadPage(context.Background(), fileRef(manifestPath), 0, 1); err == nil || !strings.Contains(err.Error(), "serialized page byte budget") {
		t.Fatalf("oversized escaped native row not bounded: %v", err)
	}
	if _, err := b.ReadPage(context.Background(), fileRef(manifestPath), 0, 51); err == nil {
		t.Fatal("oversized page count accepted")
	}
}

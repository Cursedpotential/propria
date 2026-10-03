// Byline: Claude Code · Sonnet · 2026-10-02
package activities

import (
	"archive/zip"
	"bytes"
	"context"
	"testing"
)

// TestBlockReaderAtReadsAZipDirectoryInFewRangedRequests builds a ZIP larger than one block and checks that
// archive/zip over blockReaderAt lists and opens its members with a handful of ranged reads, not one per field.
func TestBlockReaderAtReadsAZipDirectoryInFewRangedRequests(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	pad, _ := w.CreateHeader(&zip.FileHeader{Name: "pad.bin", Method: zip.Store})
	_, _ = pad.Write(make([]byte, zipBlockSize+1024))
	card, _ := w.Create("Takeout/Contacts/All.vcf")
	_, _ = card.Write([]byte("BEGIN:VCARD\r\nFN:A\r\nEND:VCARD\r\n"))
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	data := buf.Bytes()
	requests := 0
	reader := &blockReaderAt{ctx: context.Background(), size: int64(len(data)), fetch: func(_ context.Context, start, end int64) ([]byte, error) {
		requests++
		return data[start : end+1], nil
	}}
	archive, err := zip.NewReader(reader, int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, file := range archive.File {
		if file.Name == "Takeout/Contacts/All.vcf" {
			rc, err := file.Open()
			if err != nil {
				t.Fatal(err)
			}
			body := new(bytes.Buffer)
			_, _ = body.ReadFrom(rc)
			rc.Close()
			found = bytes.Contains(body.Bytes(), []byte("FN:A"))
		}
	}
	if !found || requests > 4 {
		t.Fatalf("found = %v requests = %d (a directory read plus one member must be a few block fetches)", found, requests)
	}
}

package humancontext

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"os"
)

const MaxPageBytes = 4 * 1024 * 1024
const MaxPageRecords = 50

// Page is a bounded native bundle projection, independent of legacy normalized-generation receipts.
type Page struct {
	Records    []Record `json:"records"`
	NextOffset *int64   `json:"next_offset,omitempty"`
	Total      int64    `json:"total"`
}

// ReadPage seeks the immutable fixed-width offset index and returns only the requested native records.
// Inputs: extraction/publication/verification manifest, offset and limit. Outputs: bounded page and next offset.
// Effects: read-only shared bundle files; pick for SBV pagination without rescanning a huge original.
func (b *Bundles) ReadPage(ctx context.Context, ref string, offset int64, limit int) (Page, error) {
	if offset < 0 || limit < 1 || limit > MaxPageRecords {
		return Page{}, errors.New("invalid human page bounds")
	}
	m, err := b.ReadManifest(ref)
	if err != nil {
		return Page{}, err
	}
	if m.RecordsRef == "" || m.OffsetsRef == "" {
		return Page{}, errors.New("human extraction not ready")
	}
	page := Page{Records: []Record{}, Total: m.Records}
	if offset >= m.Records {
		return page, nil
	}
	indexPath, err := b.path(m.OffsetsRef)
	if err != nil {
		return page, err
	}
	index, err := os.Open(indexPath)
	if err != nil {
		return page, err
	}
	defer index.Close()
	if _, err := index.Seek(offset*8, io.SeekStart); err != nil {
		return page, err
	}
	var position uint64
	if err := binary.Read(index, binary.LittleEndian, &position); err != nil {
		return page, err
	}
	recordPath, err := b.path(m.RecordsRef)
	if err != nil {
		return page, err
	}
	file, err := os.Open(recordPath)
	if err != nil {
		return page, err
	}
	defer file.Close()
	if _, err := file.Seek(int64(position), io.SeekStart); err != nil {
		return page, err
	}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 65536), MaxMetadataBytes+1)
	// Reserve the exact bounded envelope including maximum continuation digits and encoder newline.
	envelope := page
	maxNext := m.Records
	envelope.NextOffset = &maxNext
	header, err := json.Marshal(envelope)
	if err != nil {
		return page, err
	}
	encodedBytes := len(header) + 1
	budgetStopped := false
	for len(page.Records) < limit && scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return page, err
		}
		var row Record
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
			return page, err
		}
		// encoding/json uses the same HTML escaping as the HTTP encoder, including RawMessage content.
		encoded, err := json.Marshal(row)
		if err != nil {
			return page, err
		}
		extra := len(encoded)
		if len(page.Records) > 0 {
			extra++
		}
		if encodedBytes+extra > MaxPageBytes {
			if len(page.Records) == 0 {
				return page, errors.New("native record exceeds serialized page byte budget")
			}
			budgetStopped = true
			break
		}
		page.Records = append(page.Records, row)
		encodedBytes += extra
	}
	if err := scanner.Err(); err != nil {
		return page, err
	}
	next := offset + int64(len(page.Records))
	if !budgetStopped && len(page.Records) < limit && next < m.Records {
		return page, errors.New("human bundle ended before recorded count")
	}
	if next < m.Records {
		page.NextOffset = &next
	}
	return page, nil
}

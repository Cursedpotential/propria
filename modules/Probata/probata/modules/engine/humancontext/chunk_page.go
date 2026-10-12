package humancontext

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os"
)

// ReadChunkPage exposes the actual native chunk plan with all member locators intact.
// Inputs: trusted manifest, offset and bounded limit. Output: complete chunk objects and continuation.
// Effects: read-only derivative files; choose over rebuilding chunks from normalized records.
func (b *Bundles) ReadChunkPage(ctx context.Context, ref string, offset int64, limit int) (map[string]any, error) {
	if offset < 0 || limit < 1 || limit > 50 {
		return nil, errors.New("invalid native chunk page bounds")
	}
	m, e := b.ReadManifest(ref)
	if e != nil {
		return nil, e
	}
	if m.ChunksRef == "" {
		return nil, errors.New("native chunk plan unavailable")
	}
	p, e := b.path(m.ChunksRef)
	if e != nil {
		return nil, e
	}
	f, e := os.Open(p)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 65536), MaxMetadataBytes+1)
	rows := []json.RawMessage{}
	var index int64
	size := 0
	for s.Scan() {
		if e = ctx.Err(); e != nil {
			return nil, e
		}
		if index < offset {
			index++
			continue
		}
		raw := append([]byte(nil), s.Bytes()...)
		if !json.Valid(raw) {
			return nil, errors.New("invalid native chunk JSON")
		}
		if len(rows) >= limit || size+len(raw) > 3<<20 {
			break
		}
		rows = append(rows, raw)
		size += len(raw)
		index++
	}
	if e = s.Err(); e != nil {
		return nil, e
	}
	var next any
	if index < m.Chunks {
		next = index
	}
	return map[string]any{"items": rows, "total": m.Chunks, "next_offset": next, "bundle_ref": m.ChunksRef, "source": m.Source}, nil
}

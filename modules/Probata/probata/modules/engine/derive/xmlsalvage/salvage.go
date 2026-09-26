// Byline: Claude Code · Opus 5.5 · 2026-09-25

package xmlsalvage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
)

// ErrNothingToSalvage reports a source with no complete record before the
// cut: there is nothing a derived copy could keep. It is a property of the
// bytes, so a retry cannot change it.
var ErrNothingToSalvage = errors.New("xmlsalvage: no complete record precedes the cut-off")

const (
	headCaptureBytes = 64 << 10
	progressEvery    = 16 << 20
)

// claimedCountPattern reads the backup's own record claim from its root tag
// (SMS Backup & Restore writes <smses count="N"> / <calls count="N">).
var claimedCountPattern = regexp.MustCompile(`(?i)<[A-Za-z_][\w.:-]*\s[^>]*\bcount=["']([0-9]{1,12})["']`)

// Outcome describes one salvage.
type Outcome struct {
	Result
	// SourceBytes and SourceSHA256 cover every byte the source served.
	SourceBytes  int64  `json:"source_bytes"`
	SourceSHA256 string `json:"source_sha256"`
	// DerivedBytes and DerivedSHA256 cover the salvaged document exactly as
	// it was left in the output file.
	DerivedBytes  int64  `json:"derived_bytes"`
	DerivedSHA256 string `json:"derived_sha256"`
	// BytesDropped is how many source bytes after the last complete record
	// the salvage did not keep.
	BytesDropped int64 `json:"bytes_dropped"`
	// ClaimedCount is the record count the backup's root tag claims, or -1
	// when it names none. The salvaged copy keeps the root tag unchanged, so
	// the claim still describes the original backup.
	ClaimedCount int64 `json:"claimed_count"`
	// ClosingTag is what the salvage appended after the last kept record.
	ClosingTag string `json:"closing_tag"`
}

// Salvage streams source once into out, keeps bytes [0, LastCompleteEnd) and
// closes the document element. The result is therefore provable against the
// original: it is an exact byte prefix of the source followed by ClosingTag.
//
// out must be an empty, writable scratch file on a data volume; the caller
// owns its lifetime. progress, when set, receives the source bytes read so
// far at bounded intervals (a Temporal heartbeat, a CLI line).
func Salvage(ctx context.Context, source io.Reader, out *os.File, progress func(int64)) (Outcome, error) {
	if source == nil || out == nil {
		return Outcome{}, errors.New("xmlsalvage: source and output file are required")
	}
	scanner := &Scanner{}
	sourceHash := sha256.New()
	head := &headCapture{limit: headCaptureBytes}
	reader := &progressReader{ctx: ctx, source: source, progress: progress}
	written, err := io.Copy(io.MultiWriter(out, scanner, sourceHash, head), reader)
	if err != nil {
		return Outcome{}, fmt.Errorf("xmlsalvage: stream source: %w", err)
	}
	result := scanner.Result()
	outcome := Outcome{
		Result: result, SourceBytes: written, SourceSHA256: hex.EncodeToString(sourceHash.Sum(nil)),
		ClaimedCount: claimedCount(head.bytes),
	}
	if result.Root == "" || result.Records == 0 || result.LastCompleteEnd <= 0 {
		return outcome, ErrNothingToSalvage
	}
	outcome.BytesDropped = written - result.LastCompleteEnd
	outcome.ClosingTag = "\n</" + result.Root + ">\n"
	if err := out.Truncate(result.LastCompleteEnd); err != nil {
		return Outcome{}, fmt.Errorf("xmlsalvage: cut at the last complete record: %w", err)
	}
	if _, err := out.Seek(result.LastCompleteEnd, io.SeekStart); err != nil {
		return Outcome{}, err
	}
	if _, err := io.WriteString(out, outcome.ClosingTag); err != nil {
		return Outcome{}, fmt.Errorf("xmlsalvage: close the document: %w", err)
	}
	if err := out.Sync(); err != nil {
		return Outcome{}, err
	}
	if _, err := out.Seek(0, io.SeekStart); err != nil {
		return Outcome{}, err
	}
	derivedHash := sha256.New()
	derivedBytes, err := io.Copy(derivedHash, &progressReader{ctx: ctx, source: out})
	if err != nil {
		return Outcome{}, fmt.Errorf("xmlsalvage: hash the salvaged copy: %w", err)
	}
	outcome.DerivedBytes = derivedBytes
	outcome.DerivedSHA256 = hex.EncodeToString(derivedHash.Sum(nil))
	if _, err := out.Seek(0, io.SeekStart); err != nil {
		return Outcome{}, err
	}
	return outcome, nil
}

func claimedCount(head []byte) int64 {
	match := claimedCountPattern.FindSubmatch(head)
	if len(match) != 2 {
		return -1
	}
	value, err := strconv.ParseInt(string(match[1]), 10, 64)
	if err != nil {
		return -1
	}
	return value
}

// headCapture keeps the first limit bytes written to it.
type headCapture struct {
	limit int
	bytes []byte
}

func (h *headCapture) Write(p []byte) (int, error) {
	if room := h.limit - len(h.bytes); room > 0 {
		if len(p) < room {
			room = len(p)
		}
		h.bytes = append(h.bytes, p[:room]...)
	}
	return len(p), nil
}

// progressReader honours cancellation between reads and reports progress at
// bounded intervals.
type progressReader struct {
	ctx      context.Context
	source   io.Reader
	progress func(int64)
	read     int64
	reported int64
}

func (r *progressReader) Read(p []byte) (int, error) {
	if r.ctx != nil {
		if err := r.ctx.Err(); err != nil {
			return 0, err
		}
	}
	n, err := r.source.Read(p)
	r.read += int64(n)
	if r.progress != nil && r.read-r.reported >= progressEvery {
		r.reported = r.read
		r.progress(r.read)
	}
	return n, err
}

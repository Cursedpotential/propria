// Package smsthreads derives memory-safe structured text from one large SMS
// Backup & Restore XML and publishes it BESIDE the original object.
//
// Byline: Claude Code · Fable 5.1 · 2026-09-20
//
// Owner ruling 2026-09-20 18:47-18:51: "have sbv extract it and split out media
// and create structured text, save it back where it was, then use duckdb to
// extract the text" · "save in chunks, likely by thread" · "next to original".
//
// Why: valid backups are 500 MB to multi-GB, mostly base64 MMS media. The
// DuckDB XML reader available to pg_duckdb 1.1.1 (DuckDB 1.4.3) refuses files
// over 16 MB and has no streaming mode. SBV's parse-only importer streams one
// <sms>/<mms>/<call> at a time, so this unit reads the XML once and writes:
//
//	<key>.derived/threads/<thread>.ndjson   one JSON line per record, by thread
//	<key>.derived/media/<sha256><ext>       each decoded MMS attachment, once
//	<key>.derived/rejects.ndjson            records the decoder refused
//	<key>.derived/manifest.json             hashes, counts, provenance (written LAST)
//
// DuckDB then extracts from the NDJSON with its native streaming JSON reader.
// The original object is never modified. One unit, one job: no parsing into
// the platform bundle, no custody hashing, no orchestration (AGENTS.md
// ATOMICITY). Safe to retry: a finished derivation is detected by its manifest
// and refused; a partial one is simply overwritten object by object, because
// every object name is derived from content or thread identity.
package smsthreads

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/lowcarbdev/sbv/pkg/parseonly"
)

const (
	DerivedSuffix   = ".derived"
	ManifestName    = "manifest.json"
	SchemaVersion   = "smsthreads/v1"
	defaultMaxOpen  = 64
	contentTypeJSON = "application/json"
	contentTypeND   = "application/x-ndjson"
)

// Options names one source object and where scratch files may live. Scratch
// must be on a data volume, never the system temp dir.
type Options struct {
	Store       ObjectStore
	Scheme      string // "b2", "r2": only used to spell object URIs
	Bucket, Key string
	ScratchRoot string
	MaxOpen     int
	Now         func() time.Time
}

type ThreadFile struct {
	Thread       string   `json:"thread"`
	Participants []string `json:"participants"`
	Key          string   `json:"key"`
	Records      uint64   `json:"records"`
	Bytes        int64    `json:"bytes"`
	SHA256       string   `json:"sha256"`
	First        string   `json:"first_occurred_at,omitempty"`
	Last         string   `json:"last_occurred_at,omitempty"`
}

type Manifest struct {
	Schema        string       `json:"schema"`
	Source        string       `json:"source"`
	SourceSHA256  string       `json:"source_sha256"`
	SourceBytes   int64        `json:"source_bytes"`
	DerivedPrefix string       `json:"derived_prefix"`
	DerivedAt     string       `json:"derived_at"`
	Decoder       string       `json:"decoder"`
	Records       uint64       `json:"records"`
	Rejected      uint64       `json:"rejected"`
	MediaObjects  uint64       `json:"media_objects"`
	MediaBytes    int64        `json:"media_bytes"`
	MediaRefs     uint64       `json:"media_references"`
	Threads       []ThreadFile `json:"threads"`
	Rejects       *ThreadFile  `json:"rejects,omitempty"`
}

// Line is one NDJSON record. Attachment URIs point at the media objects beside
// the source; raw bytes of accepted records stay in the original.
type Line struct {
	Thread               string                          `json:"thread"`
	SourcePos            string                          `json:"source_pos"`
	Kind                 string                          `json:"kind"`
	Status               string                          `json:"status"`
	StatusReason         string                          `json:"status_reason,omitempty"`
	OccurredAt           string                          `json:"occurred_at,omitempty"`
	Sender               string                          `json:"sender,omitempty"`
	Recipients           []parseonly.Recipient           `json:"recipients,omitempty"`
	Participants         []string                        `json:"participants,omitempty"`
	Content              string                          `json:"content"`
	Metadata             map[string]any                  `json:"metadata,omitempty"`
	Attachments          []LineAttachment                `json:"attachments,omitempty"`
	AttachmentReferences []parseonly.AttachmentReference `json:"attachment_references,omitempty"`
	AttachmentFailures   []parseonly.AttachmentFailure   `json:"attachment_failures,omitempty"`
	Raw                  string                          `json:"raw,omitempty"` // rejected records only
}

type LineAttachment struct {
	Ordinal uint64 `json:"ordinal"`
	Name    string `json:"name,omitempty"`
	MIME    string `json:"mime,omitempty"`
	SHA256  string `json:"sha256"`
	Bytes   int64  `json:"bytes"`
	URI     string `json:"uri"`
}

// Derive streams the source once and publishes the derived objects.
func Derive(ctx context.Context, opts Options) (Manifest, error) {
	if opts.Store == nil || opts.Bucket == "" || opts.Key == "" || opts.Scheme == "" {
		return Manifest{}, errors.New("smsthreads: store, scheme, bucket and key are required")
	}
	if !filepath.IsAbs(opts.ScratchRoot) && !strings.HasPrefix(opts.ScratchRoot, "/") {
		return Manifest{}, errors.New("smsthreads: scratch root must be an absolute path on a data volume")
	}
	if opts.MaxOpen <= 0 {
		opts.MaxOpen = defaultMaxOpen
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	prefix := opts.Key + DerivedSuffix + "/"
	if done, err := opts.Store.Exists(ctx, opts.Bucket, prefix+ManifestName); err != nil {
		return Manifest{}, fmt.Errorf("smsthreads: check existing manifest: %w", err)
	} else if done {
		return Manifest{}, fmt.Errorf("smsthreads: %s%s already exists; a finished derivation is never overwritten", prefix, ManifestName)
	}
	scratch, err := os.MkdirTemp(opts.ScratchRoot, "smsthreads-")
	if err != nil {
		return Manifest{}, fmt.Errorf("smsthreads: create scratch: %w", err)
	}
	defer os.RemoveAll(scratch)

	source, err := opts.Store.Open(ctx, opts.Bucket, opts.Key)
	if err != nil {
		return Manifest{}, fmt.Errorf("smsthreads: open source: %w", err)
	}
	defer source.Close()
	sourceHash := sha256.New()
	counted := &countingReader{reader: io.TeeReader(source, sourceHash)}

	importer, err := parseonly.New(parseonly.FormatSMSBackupXML)
	if err != nil {
		return Manifest{}, err
	}
	sourceURI := fmt.Sprintf("%s://%s/%s", opts.Scheme, opts.Bucket, opts.Key)
	media := &mediaSink{ctx: ctx, opts: opts, prefix: prefix, staging: filepath.Join(scratch, "staging"), sourceURI: sourceURI, seen: map[string]bool{}}
	threads := newThreadWriter(filepath.Join(scratch, "threads"), opts.MaxOpen)
	defer threads.closeAll()

	var records, rejected uint64
	emit := func(emitCtx context.Context, record parseonly.Record) error {
		if err := emitCtx.Err(); err != nil {
			return err
		}
		line := toLine(record)
		if record.Status == parseonly.StatusRejected {
			rejected++
			line.Thread = "rejects"
			line.Raw = string(record.Raw)
			return threads.write("rejects", nil, line, record.OccurredAt)
		}
		records++
		name, participants := threadIdentity(record)
		line.Thread = name
		return threads.write(name, participants, line, record.OccurredAt)
	}
	if err := importer.ParseWithArtifacts(ctx, counted, sourceURI, media, emit); err != nil {
		return Manifest{}, fmt.Errorf("smsthreads: %w", err)
	}
	// The decoder may stop at the closing tag; drain so the digest covers the whole object.
	if _, err := io.Copy(io.Discard, counted); err != nil {
		return Manifest{}, fmt.Errorf("smsthreads: drain source: %w", err)
	}
	if records == 0 && rejected == 0 {
		return Manifest{}, errors.New("smsthreads: decoder emitted no records")
	}
	if err := threads.closeAll(); err != nil {
		return Manifest{}, err
	}

	manifest := Manifest{
		Schema: SchemaVersion, Source: sourceURI, SourceSHA256: hex.EncodeToString(sourceHash.Sum(nil)),
		SourceBytes: counted.count, DerivedPrefix: fmt.Sprintf("%s://%s/%s", opts.Scheme, opts.Bucket, prefix),
		DerivedAt: opts.Now().UTC().Format(time.RFC3339), Decoder: "sbv/parseonly " + parseonly.FormatSMSBackupXML,
		Records: records, Rejected: rejected,
		MediaObjects: uint64(len(media.seen)), MediaBytes: media.bytes, MediaRefs: media.refs,
	}
	for _, name := range threads.names() {
		entry := threads.files[name]
		key := prefix + "threads/" + name + ".ndjson"
		if name == "rejects" {
			key = prefix + "rejects.ndjson"
		}
		file, err := publishFile(ctx, opts, entry.path, key, contentTypeND)
		if err != nil {
			return Manifest{}, err
		}
		file.Thread, file.Participants, file.Records = name, entry.participants, entry.records
		file.First, file.Last = formatTime(entry.first), formatTime(entry.last)
		if name == "rejects" {
			manifest.Rejects = &file
			continue
		}
		manifest.Threads = append(manifest.Threads, file)
	}
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return Manifest{}, err
	}
	manifestPath := filepath.Join(scratch, ManifestName)
	if err := os.WriteFile(manifestPath, body, 0o600); err != nil {
		return Manifest{}, err
	}
	if _, err := publishFile(ctx, opts, manifestPath, prefix+ManifestName, contentTypeJSON); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func toLine(record parseonly.Record) Line {
	line := Line{
		SourcePos: record.SourcePos, Kind: record.Kind, Status: string(record.Status), StatusReason: record.StatusReason,
		OccurredAt: formatTime(record.OccurredAt), Sender: record.Sender, Recipients: record.Recipients,
		Participants: record.Participants, Content: record.Content, Metadata: record.Metadata,
		AttachmentReferences: record.AttachmentReferences, AttachmentFailures: record.AttachmentFailures,
	}
	for _, attachment := range record.Attachments {
		line.Attachments = append(line.Attachments, LineAttachment{
			Ordinal: attachment.AttachmentOrdinal, Name: attachment.OriginalName, MIME: attachment.MIME,
			SHA256: attachment.DigestSHA256, Bytes: attachment.ByteCount, URI: attachment.Locator.URI,
		})
	}
	return line
}

func formatTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func publishFile(ctx context.Context, opts Options, path, key, contentType string) (ThreadFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return ThreadFile{}, err
	}
	defer file.Close()
	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return ThreadFile{}, err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return ThreadFile{}, err
	}
	if err := opts.Store.Put(ctx, opts.Bucket, key, file, size, contentType); err != nil {
		return ThreadFile{}, fmt.Errorf("smsthreads: publish %s: %w", key, err)
	}
	return ThreadFile{Key: key, Bytes: size, SHA256: hex.EncodeToString(hash.Sum(nil))}, nil
}

type countingReader struct {
	reader io.Reader
	count  int64
}

func (r *countingReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.count += int64(n)
	return n, err
}

// threadIdentity names a thread by its normalized participant set, so a group
// MMS is one thread per distinct set. Calls are their own thread.
func threadIdentity(record parseonly.Record) (string, []string) {
	if record.Kind == "call" {
		return "calls", nil
	}
	set := map[string]bool{}
	for _, value := range record.Participants {
		if normalized := normalizeParty(value); normalized != "" {
			set[normalized] = true
		}
	}
	if len(set) == 0 {
		for _, value := range append([]string{record.Sender}, recipientIdentities(record.Recipients)...) {
			if normalized := normalizeParty(value); normalized != "" {
				set[normalized] = true
			}
		}
	}
	participants := make([]string, 0, len(set))
	for value := range set {
		participants = append(participants, value)
	}
	sort.Strings(participants)
	if len(participants) == 0 {
		return "unknown", nil
	}
	joined := strings.Join(participants, "_")
	if len(joined) > 80 {
		digest := sha256.Sum256([]byte(joined))
		joined = fmt.Sprintf("group-%d-%s", len(participants), hex.EncodeToString(digest[:6]))
	}
	return joined, participants
}

func recipientIdentities(values []parseonly.Recipient) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value.Identity)
	}
	return out
}

// normalizeParty keeps digits for phone numbers (dropping a NANP leading 1) and
// a file-name-safe lower-case form for short codes, e-mail senders and names.
func normalizeParty(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" || value == "null" || value == "insert-address-token" {
		return ""
	}
	digits, other := 0, 0
	for _, r := range value {
		switch {
		case r >= '0' && r <= '9':
			digits++
		case r == '+' || r == '-' || r == ' ' || r == '(' || r == ')' || r == '.':
		default:
			other++
		}
	}
	var b strings.Builder
	if other == 0 && digits > 0 {
		for _, r := range value {
			if r >= '0' && r <= '9' {
				b.WriteRune(r)
			}
		}
		number := b.String()
		if len(number) == 11 && number[0] == '1' {
			number = number[1:]
		}
		return number
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '@' || r == '.' || r == '-' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Byline: Claude Code · Fable 5.1 · 2026-09-20

package smsthreads

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Report is the result of checking a published derivation against its own
// manifest (owner 2026-09-20: "a validation script that checks output json and
// chunks"). It reads; it never writes or repairs.
type Report struct {
	Manifest       string   `json:"manifest"`
	Schema         string   `json:"schema"`
	Files          int      `json:"files"`
	Records        uint64   `json:"records"`
	Rejected       uint64   `json:"rejected"`
	MediaRefs      uint64   `json:"media_references"`
	MediaObjects   int      `json:"media_objects"`
	LargestChunk   int64    `json:"largest_chunk_bytes"`
	OversizeChunks []string `json:"oversize_chunks,omitempty"`
	Problems       []string `json:"problems,omitempty"`
	OK             bool     `json:"ok"`
}

const maxProblems = 50

// Validate re-reads every chunk a manifest names and proves: the object exists
// with the recorded size and SHA-256; every line is one JSON object carrying
// the required fields and the thread its file claims; per-file and total record
// counts match; every attachment URI names an existing media object; no chunk
// exceeds maxChunk (0 = the writer's default).
// The derived objects are read from the SAME mapped location the writer
// publishes to (derivedroot.go), so a validation run and a derive run can
// never disagree about where the derivation lives.
func Validate(ctx context.Context, store ObjectStore, roots DerivedRoots, scheme, bucket, key string, maxChunk int64) (Report, error) {
	if maxChunk <= 0 {
		maxChunk = defaultMaxChunk
	}
	location, _, err := ResolvePublished(ctx, store, roots, scheme, bucket, key)
	if err != nil {
		return Report{}, err
	}
	prefix := location.Prefix
	bucket = location.Bucket
	report := Report{Manifest: location.URI() + ManifestName}
	body, err := readAll(ctx, store, bucket, location.ManifestKey())
	if err != nil {
		return report, fmt.Errorf("smsthreads: read manifest: %w", err)
	}
	var manifest Manifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return report, fmt.Errorf("smsthreads: manifest is not valid JSON: %w", err)
	}
	report.Schema = manifest.Schema
	problem := func(format string, args ...any) {
		if len(report.Problems) < maxProblems {
			report.Problems = append(report.Problems, fmt.Sprintf(format, args...))
		}
	}
	mediaSeen := map[string]bool{}
	uriPrefix := fmt.Sprintf("%s://%s/", location.Scheme, location.Bucket)
	check := func(file ThreadFile, rejected bool) {
		report.Files++
		if file.Bytes > report.LargestChunk {
			report.LargestChunk = file.Bytes
		}
		if file.Bytes > maxChunk {
			report.OversizeChunks = append(report.OversizeChunks, file.Key)
		}
		reader, err := store.Open(ctx, bucket, file.Key)
		if err != nil {
			problem("%s: cannot open: %v", file.Key, err)
			return
		}
		defer reader.Close()
		hash := sha256.New()
		scanner := bufio.NewScanner(io.TeeReader(reader, hash))
		scanner.Buffer(make([]byte, 0, 1<<20), 256<<20)
		var lines uint64
		var size int64
		for scanner.Scan() {
			raw := scanner.Bytes()
			size += int64(len(raw)) + 1
			lines++
			var line Line
			if err := json.Unmarshal(raw, &line); err != nil {
				problem("%s line %d: invalid JSON: %v", file.Key, lines, err)
				continue
			}
			if line.SourcePos == "" || line.Status == "" || line.Thread == "" {
				problem("%s line %d: missing source_pos/status/thread", file.Key, lines)
			}
			if line.Thread != file.Thread {
				problem("%s line %d: thread %q in a %q file", file.Key, lines, line.Thread, file.Thread)
			}
			if !rejected && line.OccurredAt == "" {
				problem("%s line %d: accepted record has no occurred_at", file.Key, lines)
			}
			for _, attachment := range line.Attachments {
				report.MediaRefs++
				if !strings.HasPrefix(attachment.URI, uriPrefix+prefix+"media/") || len(attachment.SHA256) != 64 {
					problem("%s line %d: attachment %d has a bad uri or digest", file.Key, lines, attachment.Ordinal)
					continue
				}
				mediaSeen[strings.TrimPrefix(attachment.URI, uriPrefix)] = true
			}
		}
		if err := scanner.Err(); err != nil {
			problem("%s: read error: %v", file.Key, err)
			return
		}
		if digest := hex.EncodeToString(hash.Sum(nil)); digest != file.SHA256 {
			problem("%s: sha256 %s does not match manifest %s", file.Key, digest, file.SHA256)
		}
		if size != file.Bytes {
			problem("%s: %d bytes read, manifest says %d", file.Key, size, file.Bytes)
		}
		if lines != file.Records {
			problem("%s: %d lines, manifest says %d records", file.Key, lines, file.Records)
		}
		if rejected {
			report.Rejected += lines
		} else {
			report.Records += lines
		}
	}
	for _, file := range manifest.Threads {
		check(file, false)
	}
	for _, file := range manifest.Rejects {
		check(file, true)
	}
	for mediaKey := range mediaSeen {
		exists, err := store.Exists(ctx, bucket, mediaKey)
		if err != nil {
			problem("%s: cannot check media object: %v", mediaKey, err)
		} else if !exists {
			problem("%s: referenced media object is missing", mediaKey)
		}
	}
	report.MediaObjects = len(mediaSeen)
	if report.Records != manifest.Records {
		problem("records: %d in chunks, manifest says %d", report.Records, manifest.Records)
	}
	if report.Rejected != manifest.Rejected {
		problem("rejected: %d in chunks, manifest says %d", report.Rejected, manifest.Rejected)
	}
	if uint64(report.MediaObjects) != manifest.MediaObjects {
		problem("media objects: %d referenced, manifest says %d", report.MediaObjects, manifest.MediaObjects)
	}
	report.OK = len(report.Problems) == 0 && len(report.OversizeChunks) == 0
	return report, nil
}

func readAll(ctx context.Context, store ObjectStore, bucket, key string) ([]byte, error) {
	reader, err := store.Open(ctx, bucket, key)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(io.LimitReader(reader, 256<<20))
}

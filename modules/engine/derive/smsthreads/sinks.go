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
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/lowcarbdev/sbv/pkg/parseonly"
)

// threadWriter appends NDJSON lines to one scratch file per thread. A backup
// can hold thousands of threads, so only MaxOpen files stay open; the least
// recently used one is flushed and closed, then reopened in append mode.
type threadWriter struct {
	root    string
	maxOpen int
	files   map[string]*threadEntry
	tick    uint64
}

type threadEntry struct {
	path         string
	participants []string
	records      uint64
	first, last  *time.Time
	file         *os.File
	buffer       *bufio.Writer
	used         uint64
}

func newThreadWriter(root string, maxOpen int) *threadWriter {
	return &threadWriter{root: root, maxOpen: maxOpen, files: map[string]*threadEntry{}}
}

func (w *threadWriter) names() []string {
	names := make([]string, 0, len(w.files))
	for name := range w.files {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (w *threadWriter) write(name string, participants []string, line Line, occurred *time.Time) error {
	entry, ok := w.files[name]
	if !ok {
		if err := os.MkdirAll(w.root, 0o700); err != nil {
			return err
		}
		entry = &threadEntry{path: filepath.Join(w.root, name+".ndjson"), participants: participants}
		w.files[name] = entry
	}
	if entry.file == nil {
		if err := w.evict(); err != nil {
			return err
		}
		file, err := os.OpenFile(entry.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			return err
		}
		entry.file, entry.buffer = file, bufio.NewWriterSize(file, 64<<10)
	}
	w.tick++
	entry.used = w.tick
	encoded, err := json.Marshal(line)
	if err != nil {
		return fmt.Errorf("smsthreads: encode %s: %w", line.SourcePos, err)
	}
	if _, err := entry.buffer.Write(append(encoded, '\n')); err != nil {
		return err
	}
	entry.records++
	if occurred != nil {
		if entry.first == nil || occurred.Before(*entry.first) {
			value := *occurred
			entry.first = &value
		}
		if entry.last == nil || occurred.After(*entry.last) {
			value := *occurred
			entry.last = &value
		}
	}
	return nil
}

func (w *threadWriter) evict() error {
	open := 0
	var oldest *threadEntry
	for _, entry := range w.files {
		if entry.file == nil {
			continue
		}
		open++
		if oldest == nil || entry.used < oldest.used {
			oldest = entry
		}
	}
	if open < w.maxOpen || oldest == nil {
		return nil
	}
	return oldest.close()
}

func (e *threadEntry) close() error {
	if e.file == nil {
		return nil
	}
	flushErr := e.buffer.Flush()
	closeErr := e.file.Close()
	e.file, e.buffer = nil, nil
	if flushErr != nil {
		return flushErr
	}
	return closeErr
}

func (w *threadWriter) closeAll() error {
	var first error
	for _, entry := range w.files {
		if err := entry.close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

// mediaSink is the parse-only artifact sink. A decoded attachment is published
// once under its content hash beside the source. A streamed raw record (a large
// <mms> element) is NOT copied: its bytes already live in the original, so the
// locator names the source span and carries the digest of the staged bytes.
type mediaSink struct {
	ctx       context.Context
	opts      Options
	prefix    string
	staging   string
	sourceURI string
	seen      map[string]bool
	bytes     int64
	refs      uint64
}

func (s *mediaSink) ArtifactDir(context.Context, string, string) (string, error) {
	if err := os.MkdirAll(s.staging, 0o700); err != nil {
		return "", err
	}
	return s.staging, nil
}

func (s *mediaSink) Store(ctx context.Context, artifact parseonly.Artifact) (parseonly.ArtifactLocator, error) {
	file, err := os.Open(artifact.StagedPath)
	if err != nil {
		return parseonly.ArtifactLocator{}, err
	}
	defer func() {
		file.Close()
		os.Remove(artifact.StagedPath) // scratch stays small: one attachment at a time
	}()
	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return parseonly.ArtifactLocator{}, err
	}
	digest := hex.EncodeToString(hash.Sum(nil))
	if artifact.Kind == parseonly.ArtifactRawRecord {
		return parseonly.ArtifactLocator{StorageClass: "source-span", URI: s.sourceURI + "#" + artifact.ParentSourcePos, ContentHash: digest}, nil
	}
	s.refs++
	key := s.prefix + "media/" + digest + mediaExtension(artifact.OriginalName, artifact.MIME)
	uri := fmt.Sprintf("%s://%s/%s", s.opts.Scheme, s.opts.Bucket, key)
	if !s.seen[key] {
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return parseonly.ArtifactLocator{}, err
		}
		contentType := strings.TrimSpace(artifact.MIME)
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		if err := s.opts.Store.Put(ctx, s.opts.Bucket, key, file, size, contentType); err != nil {
			return parseonly.ArtifactLocator{}, fmt.Errorf("smsthreads: publish media %s: %w", key, err)
		}
		s.seen[key] = true
		s.bytes += size
	}
	return parseonly.ArtifactLocator{StorageClass: s.opts.Scheme, URI: uri, ContentHash: digest}, nil
}

func (s *mediaSink) CompleteAttempt(context.Context, string, string) error { return nil }

// QuarantineAttempt: published media is content-addressed and the manifest is
// written last, so a failed attempt leaves no manifest and a retry reuses the
// same object names. Nothing to move.
func (s *mediaSink) QuarantineAttempt(context.Context, string, string, string) error { return nil }

func mediaExtension(name, mimeType string) string {
	if ext := strings.ToLower(filepath.Ext(strings.TrimSpace(name))); ext != "" && len(ext) <= 8 && !strings.ContainsAny(ext, "/\\ ") {
		return ext
	}
	base, _, _ := mime.ParseMediaType(mimeType)
	known := map[string]string{
		"image/jpeg": ".jpg", "image/png": ".png", "image/gif": ".gif", "image/heic": ".heic", "image/webp": ".webp",
		"video/mp4": ".mp4", "video/3gpp": ".3gp", "audio/amr": ".amr", "audio/mpeg": ".mp3", "audio/mp4": ".m4a",
		"text/x-vcard": ".vcf", "text/vcard": ".vcf", "application/pdf": ".pdf", "text/plain": ".txt",
	}
	return known[base]
}

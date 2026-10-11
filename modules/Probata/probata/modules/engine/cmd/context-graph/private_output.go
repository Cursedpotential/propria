// Byline: Codex | GPT-6.1-sol | 2026-10-07
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/Cursedpotential/probata/engine/surrealsink"
)

// privateTraversalReceipt describes a private verified traversal without exposing its body.
// Inputs are the output artifact's path/hash and verified graph counts/checkpoint.
// Output is safe CLI receipt JSON; no effects occur. Pick after private artifact readback.
type privateTraversalReceipt struct {
	Path          string `json:"path"`
	SHA256        string `json:"sha256"`
	NodeCount     int    `json:"node_count"`
	EdgeCount     int    `json:"edge_count"`
	GenerationRef string `json:"generation_ref"`
	CheckpointRef string `json:"checkpoint_ref"`
}

// readPrivateTraversal reads one bounded regular output file without following a final symlink.
// Input is an absolute output path; output is its exact bytes or a static error.
// Effects are read-only. Pick for identical retries and post-write verification.
func readPrivateTraversal(path string) ([]byte, error) {
	before, e := os.Lstat(path)
	if e != nil || !before.Mode().IsRegular() {
		return nil, errors.New("private traversal output must be a regular file")
	}
	if runtime.GOOS != "windows" && before.Mode().Perm()&0077 != 0 {
		return nil, errors.New("private traversal output permissions must exclude other users")
	}
	file, e := os.Open(path)
	if e != nil {
		return nil, errors.New("private traversal output readback failed")
	}
	defer file.Close()
	opened, e := file.Stat()
	if e != nil || !os.SameFile(before, opened) {
		return nil, errors.New("private traversal output changed during readback")
	}
	raw, e := io.ReadAll(io.LimitReader(file, 2*surrealsink.MaxContextGraphBundleBytes+(64<<10)+1))
	if e != nil || len(raw) > 2*surrealsink.MaxContextGraphBundleBytes+(64<<10) {
		return nil, errors.New("private traversal output exceeds readback bound")
	}
	return raw, nil
}

// writePrivateTraversal persists the exact already-verified typed traversal in a private file.
// Inputs are an absolute path and TraverseContextGraph's response; output is a
// reference/hash/count receipt. Effects exclusively create mode-0600 JSON or read
// back an identical existing file. Pick for private inspection; it never overwrites,
// deletes, re-queries, weakens graph verification or emits a source body to stdout.
func writePrivateTraversal(path string, view surrealsink.ContextGraphReadback) (privateTraversalReceipt, error) {
	result := privateTraversalReceipt{}
	if !filepath.IsAbs(path) {
		return result, errors.New("absolute private traversal output path required")
	}
	raw, e := json.Marshal(view)
	if e != nil || len(raw)+1 > 2*surrealsink.MaxContextGraphBundleBytes+(64<<10) {
		return result, errors.New("private traversal serialization exceeds bound")
	}
	raw = append(raw, '\n')
	file, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e == nil {
		n, writeErr := file.Write(raw)
		syncErr := file.Sync()
		closeErr := file.Close()
		if writeErr != nil || n != len(raw) || syncErr != nil || closeErr != nil {
			return result, errors.New("private traversal write incomplete; existing bytes retained")
		}
	} else if !errors.Is(e, os.ErrExist) {
		return result, errors.New("cannot exclusively create private traversal output")
	}
	saved, e := readPrivateTraversal(path)
	if e != nil {
		return result, e
	}
	if !bytes.Equal(saved, raw) {
		return result, errors.New("private traversal output exists with different bytes; preserved")
	}
	sum := sha256.Sum256(saved)
	return privateTraversalReceipt{Path: path, SHA256: hex.EncodeToString(sum[:]), NodeCount: len(view.Bundle.Nodes), EdgeCount: len(view.Bundle.Edges), GenerationRef: view.Receipt.GenerationRef, CheckpointRef: view.Receipt.CheckpointRef}, nil
}

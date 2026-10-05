// Byline: Codex · GPT-6.1 · 2026-10-04.
package libraryvalidation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"path"
	"strings"
)

// VersionStore is the existing versioned B2 store's exact write/read surface.
// Inputs: object coordinates and bytes; outputs: provider version/stream. Effects: B2 I/O; never delete or overwrite originals.
type VersionStore interface {
	PutRecoveredVersion(context.Context, string, string, io.ReadSeeker, int64, string, string, func(int64)) (string, error)
	OpenVersion(context.Context, string, string, string) (io.ReadCloser, error)
}

// B2Artifacts retains validation derivatives under a distinct configured prefix using the existing versioned store.
// Inputs: provider, bucket/prefix; outputs: pinned artifacts. Effects: derivative writes and exact-version readbacks.
type B2Artifacts struct {
	Store          VersionStore
	Bucket, Prefix string
}

// Validate rejects ambiguous or unrestricted derivative destinations; inputs: configured store; outputs: error; effects: none.
func (b B2Artifacts) Validate() error {
	if b.Store == nil || b.Bucket == "" || strings.ContainsAny(b.Bucket, "/\\?#@:") || b.Prefix == "" || path.Clean(b.Prefix) != b.Prefix || strings.HasPrefix(b.Prefix, "/") || strings.Contains(b.Prefix, "..") || strings.ContainsAny(b.Prefix, "\\?#\r\n") {
		return errors.New("validation artifacts require a distinct bounded B2 derivative prefix")
	}
	return nil
}

// Put writes hash-addressed derivative bytes and independently reads the exact returned version back.
// Inputs: safe relative logical key, bounded bytes/type; outputs: verified ArtifactRef. Effects: versioned derivative storage only.
func (b B2Artifacts) Put(ctx context.Context, key string, raw []byte, media string) (ArtifactRef, error) {
	if err := b.Validate(); err != nil {
		return ArtifactRef{}, err
	}
	if key == "" || path.Clean(key) != key || strings.HasPrefix(key, "/") || strings.Contains(key, "..") || strings.ContainsAny(key, "\\?#\r\n") || len(raw) == 0 || int64(len(raw)) > MaxSourceBytes {
		return ArtifactRef{}, errors.New("invalid or over-budget validation artifact")
	}
	hash := Hash(raw)
	ext := path.Ext(key)
	key = path.Join(b.Prefix, key[:len(key)-len(ext)], strings.TrimPrefix(hash, "sha256:")+ext)
	version, err := b.Store.PutRecoveredVersion(ctx, b.Bucket, key, bytes.NewReader(raw), int64(len(raw)), media, strings.TrimPrefix(hash, "sha256:"), nil)
	if err != nil {
		return ArtifactRef{}, errors.New("validation artifact versioned write/readback failed")
	}
	if version == "" || version == "null" || len(version) > 2048 {
		return ArtifactRef{}, errors.New("validation artifact has no retained provider version")
	}
	uri := url.URL{Scheme: "b2", Host: b.Bucket, Path: "/" + key}
	query := url.Values{"versionId": []string{version}}
	uri.RawQuery = query.Encode()
	ref := ArtifactRef{URI: uri.String(), VersionID: version, SHA256: hash, Bytes: int64(len(raw))}
	if _, err = b.Read(ctx, ref, MaxSourceBytes); err != nil {
		return ArtifactRef{}, err
	}
	return ref, nil
}

// Read opens only an allowed derivative's exact version and validates full byte identity.
// Inputs: pinned artifact and budget; outputs: verified bytes. Effects: bounded B2 GET; choose over latest-object reads.
func (b B2Artifacts) Read(ctx context.Context, ref ArtifactRef, max int64) ([]byte, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}
	u, err := url.Parse(ref.URI)
	if err != nil || u.Scheme != "b2" || u.Host != b.Bucket || u.User != nil || u.Fragment != "" || !strings.HasPrefix(u.Path, "/"+b.Prefix+"/") || path.Clean(u.Path) != u.Path || len(u.Query()) != 1 || u.Query().Get("versionId") != ref.VersionID || len(u.Query()["versionId"]) != 1 || ref.VersionID == "" || ref.VersionID == "null" || !digestPattern.MatchString(ref.SHA256) || ref.Bytes <= 0 || max <= 0 || max > MaxSourceBytes || ref.Bytes > max {
		return nil, errors.New("validation artifact reference or budget is invalid")
	}
	stream, err := b.Store.OpenVersion(ctx, b.Bucket, strings.TrimPrefix(u.Path, "/"), ref.VersionID)
	if err != nil {
		return nil, errors.New("exact validation artifact version unavailable")
	}
	defer stream.Close()
	raw, err := io.ReadAll(io.LimitReader(stream, ref.Bytes+1))
	if err != nil || int64(len(raw)) != ref.Bytes || Hash(raw) != ref.SHA256 {
		return nil, errors.New("validation artifact integrity mismatch")
	}
	return raw, nil
}

type authenticatedArtifact struct {
	Payload   json.RawMessage `json:"payload"`
	Signature string          `json:"signature"`
}

// putSigned retains an authenticated descriptor/check outside history; inputs: value/key; outputs: reference; effects: B2 write.
func putSigned(ctx context.Context, store Artifacts, key string, value any, signing []byte) (ArtifactRef, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return ArtifactRef{}, err
	}
	sig, err := signJSON(json.RawMessage(raw), signing)
	if err != nil {
		return ArtifactRef{}, err
	}
	envelope, err := json.Marshal(authenticatedArtifact{Payload: raw, Signature: sig})
	if err != nil {
		return ArtifactRef{}, err
	}
	if int64(len(envelope)) > MaxArtifactBytes {
		return ArtifactRef{}, errors.New("signed descriptor exceeds budget")
	}
	return store.Put(ctx, key, envelope, "application/json")
}

// readSigned authenticates and decodes a complete immutable descriptor; inputs: reference/key; outputs: typed value; effects: GET.
func readSigned(ctx context.Context, store Artifacts, ref ArtifactRef, dst any, signing []byte) error {
	raw, err := store.Read(ctx, ref, MaxArtifactBytes)
	if err != nil {
		return err
	}
	var envelope authenticatedArtifact
	if err = json.Unmarshal(raw, &envelope); err != nil || !matchesSignature(envelope.Payload, envelope.Signature, signing) {
		return errors.New("validation descriptor signature mismatch")
	}
	return json.Unmarshal(envelope.Payload, dst)
}

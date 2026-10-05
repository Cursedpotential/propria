// Byline: Codex · GPT-6.1 · 2026-10-04.
package librarysync

import (
	"bytes"
	"context"
	"errors"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
)

// OriginalBytes contains verified original bytes only inside the server process, never MCP/Temporal results.
// Inputs: authorized binding and exact pointer; outputs: HTTP body/type/name. Effects: none in this type.
type OriginalBytes struct {
	Bytes       []byte
	ContentType string
	Filename    string
}

// OriginalReader supplies authenticated pinned PDF reads without requiring extraction or validator runtime dependencies.
// Inputs: binding ID/exact version; outputs: verified private bytes. Effects: bounded backend and B2 reads only.
type OriginalReader interface {
	ReadOriginal(context.Context, string, string) (OriginalBytes, error)
}

// PinnedOriginalReader composes only the read-only storage/backend/scope dependencies needed by the existing starter.
// Inputs: server-only B2 storage and authenticated backend; outputs: verified originals. Effects: no extraction, signing or writes.
type PinnedOriginalReader struct {
	Scope   Scope
	Storage Storage
	Backend Backend
}

// NewOriginalReader creates the starter's read-only reader without Python, artifacts or validator signing credentials.
// Inputs: admitted scope/storage/backend; outputs: ready reader or configuration error. Effects: none until ReadOriginal.
func NewOriginalReader(scope Scope, storage Storage, backend Backend) (*PinnedOriginalReader, error) {
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	if storage == nil || backend == nil {
		return nil, errors.New("original reader requires B2 storage and authenticated backend")
	}
	return &PinnedOriginalReader{Scope: scope, Storage: storage, Backend: backend}, nil
}

// ReadOriginal resolves an authorized retained version and verifies its full hash/size before a successful download.
// Inputs: library_file ID/exact VersionId; outputs: at most 20 MiB of original PDF bytes. Effects: backend metadata and pinned B2 GET.
// Choose for app/backend/sidecar streaming proxies; never accept browser bucket/key coordinates or issue browser storage credentials.
func (s *Service) ReadOriginal(ctx context.Context, id, version string) (OriginalBytes, error) {
	if s == nil {
		return OriginalBytes{}, errors.New("original reader not configured")
	}
	reader, err := NewOriginalReader(s.Scope, s.Storage, s.Backend)
	if err != nil {
		return OriginalBytes{}, err
	}
	return reader.ReadOriginal(ctx, id, version)
}

// ReadOriginal verifies an authorized exact-version PDF using only storage, backend and scope.
// Inputs: stable binding ID/exact VersionId; outputs: complete bytes capped at 20 MiB. Effects: metadata read and pinned GET.
// Choose in the existing starter; it never invokes extraction or accesses validation/currency approval credentials.
func (s *PinnedOriginalReader) ReadOriginal(ctx context.Context, id, version string) (OriginalBytes, error) {
	if s == nil || s.Storage == nil || s.Backend == nil || s.Scope.Validate() != nil || !strings.HasPrefix(id, "library_file:") || !rawHash.MatchString(strings.TrimPrefix(id, "library_file:")) || !validVersion(version) {
		return OriginalBytes{}, errors.New("original reader reference or configuration invalid")
	}
	b, err := s.Backend.Original(ctx, id, version)
	if err != nil {
		return OriginalBytes{}, err
	}
	expected, err := s.Scope.BindingID(b.Key)
	if err != nil || b.Provider != "b2" || b.ID != id || expected != id || b.Scope != s.Scope || b.OriginalPointer == nil || b.OriginalPointer.VersionID != version {
		return OriginalBytes{}, errors.New("original binding does not authorize requested version")
	}
	p := *b.OriginalPointer
	if p.validate() != nil || p.Size <= 0 || p.ContentType != "application/pdf" {
		return OriginalBytes{}, errors.New("original PDF identity or body budget invalid")
	}
	raw, actual, err := s.Storage.Read(ctx, Object{Bucket: b.Bucket, Key: b.Key, VersionID: version, Size: p.Size}, MaxOriginalBytes)
	if err != nil || actual.VersionID != version || int64(len(raw)) != p.Size || digest(raw) != p.SHA256 || !bytes.HasPrefix(raw, []byte("%PDF-")) {
		return OriginalBytes{}, errors.New("original PDF integrity check failed")
	}
	return OriginalBytes{Bytes: raw, ContentType: "application/pdf", Filename: path.Base(b.Key)}, nil
}

// OriginalHandler streams verified pinned originals through the existing starter or authenticated server proxy.
// Inputs: read-only reader, mandatory parent authorization callback, GET binding/version. Outputs: PDF bytes or safe explicit errors.
// Effects: bounded metadata/B2 reads, max two simultaneous 20 MiB buffers. Parent owns route mounting and end-user authorization.
// Choose over base64 MCP results; successful response follows complete integrity verification and carries no storage credentials.
type OriginalHandler struct {
	Reader    OriginalReader
	Authorize func(*http.Request) bool
	slots     chan struct{}
}

// NewOriginalHandler requires parent authentication and bounds concurrent downloads; inputs: reader/auth; outputs: HTTP handler.
// Effects: none. Server-to-server token auth belongs to the parent; phone/workdesk must authorize their user before proxying.
func NewOriginalHandler(reader OriginalReader, authorize func(*http.Request) bool) (*OriginalHandler, error) {
	if reader == nil || authorize == nil {
		return nil, errors.New("original route requires configured service and parent authorization")
	}
	return &OriginalHandler{Reader: reader, Authorize: authorize, slots: make(chan struct{}, 2)}, nil
}

// ServeHTTP accepts only GET /toolkit/library/files/{binding_id}/original?version_id=<exact> and streams a verified private PDF.
// Inputs: authenticated request; outputs: original bytes. Effects: bounded reads; integrity failures are visible before success headers.
func (h *OriginalHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if h == nil || h.Reader == nil || h.Authorize == nil {
		http.Error(w, "Original download not configured", http.StatusServiceUnavailable)
		return
	}
	if !h.Authorize(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/toolkit/library/files/"), "/original")
	q := r.URL.Query()
	version := q.Get("version_id")
	if r.URL.Path != "/toolkit/library/files/"+id+"/original" || !strings.HasPrefix(id, "library_file:") || !rawHash.MatchString(strings.TrimPrefix(id, "library_file:")) || len(q) != 1 || len(q["version_id"]) != 1 || !validVersion(version) {
		http.Error(w, "Exact original reference required", http.StatusBadRequest)
		return
	}
	select {
	case h.slots <- struct{}{}:
		defer func() { <-h.slots }()
	default:
		http.Error(w, "Original download busy", http.StatusTooManyRequests)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), IOTimeout)
	defer cancel()
	original, err := h.Reader.ReadOriginal(ctx, id, version)
	if err != nil {
		http.Error(w, "Original unavailable or integrity check failed", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", original.ContentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": original.Filename}))
	w.Header().Set("Content-Length", strconv.Itoa(len(original.Bytes)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(original.Bytes)
}

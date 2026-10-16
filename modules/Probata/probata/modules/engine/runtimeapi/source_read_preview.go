package runtimeapi

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Cursedpotential/probata/engine/sourcepreview"
	"go.temporal.io/sdk/client"
)

const previewCacheBudget = 8 << 20
const previewCacheReservation = sourcepreview.MaxPayloadBytes
const previewCacheEntries = 64
const previewCursorTTL = 15 * time.Minute
const previewBodyTTL = 5 * time.Minute

type rawPreviewEntry struct {
	task      sourcepreview.Task
	actor     string
	page      *sourcepreview.Page
	cursor    string
	created   time.Time
	done      chan struct{}
	finished  bool
	callback  bool
	errorCode string
}

// SourceReadPreviewHandler serves ephemeral raw SMS pages through a single-purpose Temporal workflow.
// Inputs: configured Temporal client/queue and existing service token. Outputs: bounded authenticated routes.
// Effects: memory-only page/cursor cache; choose for source inspection before the persisted normalized Proffer preview.
// Each entry reserves 512 KiB including state and struct overhead; the 8 MiB aggregate cap bounds ready and pending entries.
type SourceReadPreviewHandler struct {
	client  client.Client
	queue   string
	auth    func(http.HandlerFunc) http.HandlerFunc
	mu      sync.Mutex
	entries map[string]*rawPreviewEntry
	cursors map[string]string
}

// NewSourceReadPreviewHandler constructs the bounded preview cache using existing tailnet/service-token authorization.
// Inputs: Temporal client, task queue and mounted service-token path. Outputs: raw-preview handler.
// Effects: validates the token file, allocates empty maps; choose independently of normalized preview persistence.
func NewSourceReadPreviewHandler(c client.Client, queue, tokenFile string) (*SourceReadPreviewHandler, error) {
	if c == nil || strings.TrimSpace(queue) == "" {
		return nil, sourcepreview.ErrInvalid
	}
	if _, err := loadServiceToken(tokenFile); err != nil {
		return nil, err
	}
	guard := &PreviewHTTPHandler{serviceTokenPath: tokenFile}
	return &SourceReadPreviewHandler{client: c, queue: queue, auth: guard.auth, entries: map[string]*rawPreviewEntry{}, cursors: map[string]string{}}, nil
}

// Routes exposes start/poll plus private Activity task/result callbacks with the existing authenticated boundary.
// Inputs: HTTP requests; outputs: bounded JSON, pending receipts or typed errors. Effects: ephemeral cache and workflow starts.
// Choose POST source-preview to view a raw source; callback routes are for the configured Activity only.
func (h *SourceReadPreviewHandler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /reference-import/source-preview", h.auth(h.startRawPreview))
	mux.HandleFunc("GET /reference-import/source-preview/{page_id}", h.auth(h.pollRawPreview))
	mux.HandleFunc("GET /reference-import/source-preview/{page_id}/task", h.auth(h.rawPreviewTask))
	mux.HandleFunc("POST /reference-import/source-preview/{page_id}/result", h.auth(h.rawPreviewResult))
	return mux
}

func opaquePreviewID() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", sourcepreview.ErrUnavailable
	}
	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}
func (h *SourceReadPreviewHandler) expireLocked(now time.Time) {
	for id, entry := range h.entries {
		if now.Sub(entry.created) > previewCursorTTL {
			delete(h.cursors, entry.cursor)
			delete(h.entries, id)
		} else if now.Sub(entry.created) > previewBodyTTL {
			entry.page = nil
		}
	}
}
func (h *SourceReadPreviewHandler) reserveLocked() bool {
	h.expireLocked(time.Now())
	for len(h.entries) >= previewCacheEntries || (len(h.entries)+1)*previewCacheReservation > previewCacheBudget {
		oldest := ""
		var before time.Time
		for id, entry := range h.entries {
			if entry.finished && (oldest == "" || entry.created.Before(before)) {
				oldest = id
				before = entry.created
			}
		}
		if oldest == "" {
			return false
		}
		delete(h.cursors, h.entries[oldest].cursor)
		delete(h.entries, oldest)
	}
	return true
}

// startRawPreview starts one actor-bound raw-source page workflow.
// Inputs: source pins, optional opaque cursor and bounded row limit. Outputs: ready page or 202 poll receipt.
// Effects: reserves ephemeral cache capacity and starts Temporal; choose before extraction to inspect the original source.
func (h *SourceReadPreviewHandler) startRawPreview(w http.ResponseWriter, r *http.Request) {
	actor, _, err := authenticatedActor(r)
	if err != nil {
		previewError(w, 401, errors.New("source_preview_actor_required"))
		return
	}
	var request sourcepreview.Request
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if decoder.Decode(&request) != nil {
		previewError(w, 400, sourcepreview.ErrInvalid)
		return
	}
	if _, _, err := validateAuthorizedSourceRef(request.SourceRef); err != nil || request.ExpectedETag == "" || len(request.ExpectedETag) > 512 || len(request.VersionID) > 512 || len(request.Cursor) > 128 {
		previewError(w, 422, sourcepreview.ErrInvalid)
		return
	}
	if request.Limit == 0 {
		request.Limit = 25
	}
	if request.Limit < 1 || request.Limit > sourcepreview.MaxRows {
		previewError(w, 422, sourcepreview.ErrInvalid)
		return
	}
	id, err := opaquePreviewID()
	if err != nil {
		previewError(w, 503, sourcepreview.ErrUnavailable)
		return
	}
	entry := &rawPreviewEntry{task: sourcepreview.Task{Request: request}, actor: actor, created: time.Now(), done: make(chan struct{})}
	h.mu.Lock()
	h.expireLocked(time.Now())
	if request.Cursor != "" {
		previous := h.entries[h.cursors[request.Cursor]]
		if previous == nil || !previous.finished || previous.errorCode != "" {
			h.mu.Unlock()
			previewError(w, 410, sourcepreview.ErrExpired)
			return
		}
		if previous.actor != actor || previous.task.Request.SourceRef != request.SourceRef || previous.task.Request.ExpectedETag != request.ExpectedETag || previous.task.Request.VersionID != request.VersionID {
			h.mu.Unlock()
			previewError(w, 409, sourcepreview.ErrChanged)
			return
		}
		entry.task.State = previous.task.State
	}
	if !h.reserveLocked() {
		h.mu.Unlock()
		previewError(w, 429, sourcepreview.ErrCapacity)
		return
	}
	h.entries[id] = entry
	h.mu.Unlock()
	run, err := h.client.ExecuteWorkflow(r.Context(), client.StartWorkflowOptions{ID: "source-preview-" + id, TaskQueue: h.queue, WorkflowExecutionTimeout: 25 * time.Second}, sourcepreview.WorkflowName, id)
	if err != nil {
		h.mu.Lock()
		delete(h.entries, id)
		h.mu.Unlock()
		previewError(w, 503, sourcepreview.ErrUnavailable)
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		var receipt sourcepreview.Receipt
		waitErr := run.Get(ctx, &receipt)
		h.mu.Lock()
		defer h.mu.Unlock()
		current := h.entries[id]
		if current == nil {
			return
		}
		if waitErr != nil || !current.callback || receipt.PageID != id {
			current.errorCode = sourcepreview.ErrUnavailable.Error()
			current.page = nil
		}
		current.finished = true
		close(current.done)
	}()
	select {
	case <-entry.done:
		h.renderRawPreview(w, r, id, actor)
	case <-time.After(2 * time.Second):
		h.renderRawPreview(w, r, id, actor)
	case <-r.Context().Done():
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = h.client.CancelWorkflow(ctx, run.GetID(), run.GetRunID())
	}
}

// pollRawPreview reads an actor-bound ephemeral page after its workflow receipt arrives.
// Inputs: opaque page ID and authenticated actor. Outputs: ready page, pending receipt or 410 expiry.
// Effects: expires old cache entries; choose after POST returns 202, without starting another workflow.
func (h *SourceReadPreviewHandler) pollRawPreview(w http.ResponseWriter, r *http.Request) {
	actor, _, err := authenticatedActor(r)
	if err != nil {
		previewError(w, 401, errors.New("source_preview_actor_required"))
		return
	}
	h.renderRawPreview(w, r, r.PathValue("page_id"), actor)
}
func (h *SourceReadPreviewHandler) renderRawPreview(w http.ResponseWriter, r *http.Request, id, actor string) {
	h.mu.Lock()
	h.expireLocked(time.Now())
	entry := h.entries[id]
	if entry == nil || entry.actor != actor {
		h.mu.Unlock()
		previewError(w, 410, sourcepreview.ErrExpired)
		return
	}
	if !entry.finished {
		h.mu.Unlock()
		previewJSON(w, 202, map[string]string{"page_id": id, "status": "pending", "poll_url": "/reference-import/source-preview/" + id})
		return
	}
	if entry.errorCode != "" {
		code := entry.errorCode
		h.mu.Unlock()
		status := 503
		if code == sourcepreview.ErrChanged.Error() {
			status = 409
		}
		if code == sourcepreview.ErrInvalid.Error() {
			status = 422
		}
		previewError(w, status, errors.New(code))
		return
	}
	if entry.page == nil {
		h.mu.Unlock()
		previewError(w, 410, sourcepreview.ErrExpired)
		return
	}
	page := entry.page
	h.mu.Unlock()
	previewJSON(w, 200, page)
}

// rawPreviewTask supplies the Activity with its pending source pins and private scanner continuation.
// Inputs: opaque pending page ID under existing service auth. Outputs: bounded task outside Temporal history.
// Effects: reads the ephemeral cache; choose only from the configured Activity callback client.
func (h *SourceReadPreviewHandler) rawPreviewTask(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	h.expireLocked(time.Now())
	entry := h.entries[r.PathValue("page_id")]
	if entry == nil || entry.finished || entry.callback {
		h.mu.Unlock()
		previewError(w, 410, sourcepreview.ErrExpired)
		return
	}
	task := entry.task
	h.mu.Unlock()
	previewJSON(w, 200, task)
}

// rawPreviewResult accepts one bounded, pinned Activity result into the ephemeral starter cache.
// Inputs: known pending page ID, bounded page and scanner continuation. Outputs: callback acknowledgment.
// Effects: memory-only cache update and opaque cursor allocation; choose only for the configured Activity, never ingestion.
func (h *SourceReadPreviewHandler) rawPreviewResult(w http.ResponseWriter, r *http.Request) {
	var result sourcepreview.Completion
	r.Body = http.MaxBytesReader(w, r.Body, sourcepreview.MaxPayloadBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if decoder.Decode(&result) != nil {
		previewError(w, 400, sourcepreview.ErrInvalid)
		return
	}
	id := r.PathValue("page_id")
	h.mu.Lock()
	defer h.mu.Unlock()
	entry := h.entries[id]
	if entry == nil || entry.finished || entry.callback {
		previewError(w, 409, sourcepreview.ErrExpired)
		return
	}
	if result.ErrorCode != "" {
		if result.ErrorCode != sourcepreview.ErrInvalid.Error() && result.ErrorCode != sourcepreview.ErrChanged.Error() && result.ErrorCode != sourcepreview.ErrUnavailable.Error() {
			previewError(w, 400, sourcepreview.ErrInvalid)
			return
		}
		entry.errorCode = result.ErrorCode
		entry.callback = true
		previewJSON(w, 200, map[string]string{"status": "accepted"})
		return
	}
	if !validRawPreviewResult(id, entry.task, result) {
		previewError(w, 422, sourcepreview.ErrInvalid)
		return
	}
	if !result.State.Done {
		cursor, err := opaquePreviewID()
		if err != nil {
			previewError(w, 503, sourcepreview.ErrUnavailable)
			return
		}
		entry.cursor = cursor
		result.Page.NextCursor = cursor
		h.cursors[cursor] = id
	}
	entry.page = &result.Page
	entry.task.State = result.State
	entry.task.Request.VersionID = result.Page.Source.VersionID
	entry.callback = true
	previewJSON(w, 200, map[string]string{"status": "accepted"})
}

func validRawPreviewResult(id string, task sourcepreview.Task, result sourcepreview.Completion) bool {
	p := result.Page
	pins := p.Source
	if p.PageID != id || p.Status != "ready" || !p.PreviewOnly || pins.SourceRef != task.Request.SourceRef || pins.ETag != task.Request.ExpectedETag || task.Request.VersionID != "" && pins.VersionID != task.Request.VersionID || pins.ByteLength <= 0 || len(pins.VersionID) > 512 || p.NextCursor != "" || len(p.Messages) > task.Request.Limit || p.ScannedBytes < 0 || p.ScannedBytes > sourcepreview.MaxScanBytes || result.State.Offset != task.State.Offset+p.ScannedBytes || result.State.Offset > pins.ByteLength {
		return false
	}
	if p.TruncationReason != "" && p.TruncationReason != "row_limit" && p.TruncationReason != "scan_byte_limit" && p.TruncationReason != "record_byte_limit" {
		return false
	}
	if result.State.SourceLength != pins.ByteLength {
		return false
	}
	stateBytes, err := json.Marshal(result.State)
	if err != nil || len(stateBytes) > sourcepreview.MaxStateBytes {
		return false
	}
	for _, m := range p.Messages {
		if len(m.ID) > 32 || m.DateMS != nil && (*m.DateMS < 0 || *m.DateMS > 8640000000000000) || len(m.Body) > sourcepreview.MaxBodyBytes || len(m.Participants) > sourcepreview.MaxParticipants || len(m.Attachments) > sourcepreview.MaxAttachments || m.Kind != "sms" && m.Kind != "mms" || m.Direction != "incoming" && m.Direction != "outgoing" && m.Direction != "unknown" {
			return false
		}
		for _, v := range m.Participants {
			if len(v) > 128 {
				return false
			}
		}
		for _, v := range m.Attachments {
			if len(v.Name) > 128 || len(v.ContentType) > 128 || v.EncodedBytes < 0 {
				return false
			}
		}
	}
	encoded, err := json.Marshal(result)
	return err == nil && len(encoded) <= sourcepreview.MaxPayloadBytes-1024
}

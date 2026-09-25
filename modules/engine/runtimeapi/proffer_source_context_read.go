// Byline: Claude Code · Opus 5.5 · 2026-09-25 (Review Actions: run source-context read-back)
package runtimeapi

import (
	"errors"
	"net/http"

	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/sourcecontext"
)

// RunSourceContext is the read model behind the Review page's Actions panel:
// what one run was registered with and the operator's newest source-context
// revision for it. It is read-only. Writes stay on POST
// /reference-import/source-contexts, which supersedes append-only and requires
// the exact revision reference and observation returned here.
type RunSourceContext struct {
	PreviewHandle    string                      `json:"preview_handle"`
	RequestID        string                      `json:"request_id"`
	SourceRef        proffer.Ref                 `json:"source_ref"`
	ParserOptionsRef proffer.Ref                 `json:"parser_options_ref"`
	Registration     *sourcecontext.Registration `json:"registration"`
	Current          *sourcecontext.Revision     `json:"current"`
}

// readSourceContext answers GET
// /reference-import/previews/{preview_handle}/source-context. A run without
// operator context (batch items, engine successor runs) or without a
// registration (it failed before register_source) is an ordinary 200 with a
// null member, never an error.
func (h *PreviewHTTPHandler) readSourceContext(w http.ResponseWriter, r *http.Request) {
	reader, ok := h.sourceContext.(sourcecontext.Reader)
	if !ok {
		previewError(w, http.StatusServiceUnavailable, errors.New("source context read-back is unavailable"))
		return
	}
	binding, err := h.store.Binding(r.Context(), r.PathValue("preview_handle"))
	if err != nil {
		h.storeError(w, err)
		return
	}
	response := RunSourceContext{
		PreviewHandle: binding.Handle, RequestID: binding.RequestID,
		SourceRef: binding.SourceRef, ParserOptionsRef: binding.ParserOptionsRef,
	}
	registration, found, err := reader.SourceRegistration(r.Context(), binding.RequestID)
	if err != nil {
		previewError(w, http.StatusServiceUnavailable, err)
		return
	}
	if found {
		response.Registration = &registration
	}
	current, found, err := reader.CurrentSourceContext(r.Context(), binding.RequestID, string(binding.SourceRef))
	if err != nil {
		previewError(w, http.StatusServiceUnavailable, err)
		return
	}
	if found {
		response.Current = &current
	}
	previewJSON(w, http.StatusOK, response)
}

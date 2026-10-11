// Byline: Codex · GPT-5 · 2026-10-05
package runtimeapi

import (
	"github.com/Cursedpotential/probata/engine/caseidentity"
	"net/http"
)

// canonicalRequestWrite checks explicit query/header mode before handler writes.
//
// Inputs: HTTP request context. Outputs: admission or a conflict response.
// Side effects: only an error response. Pick for shared route guards; handlers
// with body modes and durable handles also verify those independent coordinates.
func canonicalRequestWrite(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodPatch && r.Method != http.MethodDelete {
		return true
	}
	for _, raw := range []string{r.URL.Query().Get("mode"), r.URL.Query().Get("operating_mode"), r.Header.Get("X-Propria-Operating-Mode")} {
		mode, err := caseidentity.ParseMode(raw)
		if err != nil {
			previewError(w, http.StatusUnprocessableEntity, err)
			return false
		}
		if err := caseidentity.RequireCanonicalWrite(mode); err != nil {
			previewError(w, http.StatusConflict, err)
			return false
		}
	}
	return true
}

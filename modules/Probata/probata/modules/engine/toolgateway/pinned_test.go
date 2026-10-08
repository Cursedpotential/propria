package toolgateway

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/proffer"
)

// TestPinnedSourceRejectsChangedDigest proves the operator's reviewed digest
// is checked before tool dispatch; no source bytes are changed.
func TestPinnedSourceRejectsChangedDigest(t *testing.T) {
	resolve, _ := sealObject(t, []byte("live-source"))
	runner := &fakeRunner{}
	g := newGateway(t, resolve, runner)
	_, err := g.RunPinned(context.Background(), "repair.detect", proffer.Ref("upload://abc"), strings.Repeat("0", 64), nil)
	if err == nil || !strings.Contains(err.Error(), "does not match") || runner.calls != 0 {
		t.Fatalf("digest mismatch dispatched a tool: err=%v calls=%d", err, runner.calls)
	}
}

// TestPinnedHTTPRejectsUnsafePolicy proves a catalog change fails closed even
// for an initially allowed tool; it leaves the tool runner untouched.
func TestPinnedHTTPRejectsUnsafePolicy(t *testing.T) {
	resolve, _ := sealObject(t, []byte("live-source"))
	runner := &fakeRunner{}
	h := &HTTPHandler{Gateway: newGateway(t, resolve, runner), Index: func() (json.RawMessage, error) {
		return json.RawMessage(`[{"id":"repair.detect","side_effect":"derived_write","execution_policy":"manual_approval_required"}]`), nil
	}}
	sum := sha256.Sum256([]byte("live-source"))
	request := `{"source_ref":"upload://abc","source_sha256":"` + hex.EncodeToString(sum[:]) + `","operation_id":"11111111-1111-1111-1111-111111111111"}`
	r := httptest.NewRequest(http.MethodPost, "/tools/repair.detect/run-pinned", strings.NewReader(request))
	r.RemoteAddr = "100.91.190.107:5555"
	w := httptest.NewRecorder()
	h.Routes().ServeHTTP(w, r)
	if w.Code != http.StatusForbidden || runner.calls != 0 { t.Fatalf("unsafe policy ran: status=%d calls=%d", w.Code, runner.calls) }
}

// TestPinnedHTTPRunsReviewedSource proves a live read-only manifest and exact
// digest dispatch once with reserved audit fields added by the gateway.
func TestPinnedHTTPRunsReviewedSource(t *testing.T) {
	resolve, _ := sealObject(t, []byte("live-source"))
	runner := &fakeRunner{}
	h := &HTTPHandler{Gateway: newGateway(t, resolve, runner), Index: func() (json.RawMessage, error) {
		return json.RawMessage(`[{"id":"repair.detect","side_effect":"read_only","execution_policy":"manual_or_auto"}]`), nil
	}}
	sum := sha256.Sum256([]byte("live-source"))
	request := `{"source_ref":"upload://abc","source_sha256":"` + hex.EncodeToString(sum[:]) + `","operation_id":"11111111-1111-1111-1111-111111111111"}`
	r := httptest.NewRequest(http.MethodPost, "/tools/repair.detect/run-pinned", strings.NewReader(request))
	r.RemoteAddr = "100.91.190.107:5555"
	w := httptest.NewRecorder()
	h.Routes().ServeHTTP(w, r)
	if w.Code != http.StatusOK || runner.calls != 1 { t.Fatalf("reviewed source not run once: status=%d calls=%d body=%s", w.Code, runner.calls, w.Body.String()) }
	if runner.lastArgs["_input_sha256"] != hex.EncodeToString(sum[:]) || runner.lastArgs["_execution_mode"] != "temporal" || runner.lastArgs["_operation_id"] != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("gateway did not attach audited pin: %#v", runner.lastArgs)
	}
}

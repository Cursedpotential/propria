// Byline: Codex | GPT-6.1-sol | 2026-10-07
package surrealsink

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestContextGraphPreflightMatchesActualRPCSerialization verifies exact wire measurement.
// Inputs are small format-only values; output is assertions. Effects are local HTTP
// transport only, with no corpus, SQL execution, graph writes or delivery proof.
func TestContextGraphPreflightMatchesActualRPCSerialization(t *testing.T) {
	b := graphFormatPacket()
	measured, e := PreflightContextGraph(b)
	if e != nil {
		t.Fatal(e)
	}
	if !measured.Fits || measured.LimitBytes != maxBody {
		t.Fatal("small format packet rejected")
	}
	wireBytes := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		wireBytes = len(raw)
		w.WriteHeader(400)
	}))
	defer server.Close()
	client, e := NewAnalysis(Config{URL: server.URL, Namespace: "fct", Database: "analysis", AuthLevel: "database", User: "format-user", Password: "format-password"})
	if e != nil {
		t.Fatal(e)
	}
	sql, vars, e := contextGraphProjectionPlan(b)
	if e != nil {
		t.Fatal(e)
	}
	_, _ = client.Query(context.Background(), sql, vars)
	if wireBytes != measured.RequestBytes {
		t.Fatalf("wire bytes %d differ from measured %d", wireBytes, measured.RequestBytes)
	}
}

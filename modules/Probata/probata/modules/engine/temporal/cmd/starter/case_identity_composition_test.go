// Byline: Codex · GPT-6.1-sol · 2026-10-06 (bounded scope route composition)
// Byline: Claude Code · Opus 5.5 · 2026-10-01
package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMountCaseIdentityRoutesOwnsOnlyItsExactPaths(t *testing.T) {
	marker := func(name string) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.Header().Set("X-Handler", name) })
	}
	routes, err := mountCaseIdentityRoutes(marker("existing"), marker("case-identity"))
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ method, path, want string }{
		{http.MethodGet, "/case-identity?mode=REAL", "case-identity"},
		{http.MethodGet, "/case-identity/scope?mode=LIVE", "case-identity"},
		{http.MethodPost, "/case-identity/scope", "existing"},
		{http.MethodGet, "/case-identity/lookup?value=8102689630", "case-identity"},
		{http.MethodPost, "/case-identity/identifiers", "case-identity"},
		{http.MethodPost, "/case-identity/identifiers/01a0f751-e07b-7000-8000-00000000a001", "case-identity"},
		{http.MethodPost, "/case-identity/identifiers/01a0f751-e07b-7000-8000-00000000a001/delete", "case-identity"},
		{http.MethodPost, "/case-identity/header?mode=REAL", "case-identity"},
		{http.MethodPost, "/case-identity/people", "case-identity"},
		{http.MethodPost, "/case-identity/people/01a0f751-e07b-76b6-afcb-63acfbba373e", "case-identity"},
		{http.MethodPost, "/case-identity/triage", "case-identity"},
		{http.MethodDelete, "/case-identity/identifiers", "existing"},
		{http.MethodGet, "/case-identity/people", "existing"},
		{http.MethodPost, "/reference-import/start", "existing"},
	} {
		recorder := httptest.NewRecorder()
		routes.ServeHTTP(recorder, httptest.NewRequest(test.method, test.path, nil))
		if got := recorder.Header().Get("X-Handler"); got != test.want {
			t.Errorf("%s %s went to %q, want %q", test.method, test.path, got, test.want)
		}
	}
	if _, err := mountCaseIdentityRoutes(nil, marker("c")); err == nil {
		t.Fatal("nil existing handler accepted")
	}
}

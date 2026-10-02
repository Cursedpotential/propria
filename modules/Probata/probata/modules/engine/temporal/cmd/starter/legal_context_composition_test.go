package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLegalContextMountLeavesExistingRoutesAndNonReadMethodsIntact(t *testing.T) {
	existing := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusAccepted) })
	context := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	routes, err := mountLegalContextRoutes(existing, context)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{http.MethodGet, "/legal-context/records?mode=REAL&kind=event", http.StatusOK},
		{http.MethodGet, "/case-identity?mode=REAL", http.StatusAccepted},
		{http.MethodPost, "/legal-context/records", http.StatusAccepted},
		{http.MethodGet, "/legal-context/records/other", http.StatusAccepted},
	} {
		response := httptest.NewRecorder()
		routes.ServeHTTP(response, httptest.NewRequest(tc.method, tc.path, nil))
		if response.Code != tc.status {
			t.Errorf("%s %s: %d want %d", tc.method, tc.path, response.Code, tc.status)
		}
	}
	if _, err = mountLegalContextRoutes(nil, context); err == nil {
		t.Fatal("nil existing accepted")
	}
	if _, err = mountLegalContextRoutes(existing, nil); err == nil {
		t.Fatal("nil reader accepted")
	}
}

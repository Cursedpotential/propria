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

func TestInvestigationMountRoutesReachAdjacentReceiver(t *testing.T) {
	existing := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(202) })
	context := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	routes, e := mountLegalContextRoutes(existing, context)
	if e != nil {
		t.Fatal(e)
	}
	for _, request := range [][2]string{{"POST", "/legal-context/investigations"}, {"GET", "/legal-context/investigations/11111111-1111-4111-8111-111111111111"}} {
		response := httptest.NewRecorder()
		routes.ServeHTTP(response, httptest.NewRequest(request[0], request[1], nil))
		if response.Code != 200 {
			t.Fatal("receiver route not mounted", response.Code)
		}
	}
}

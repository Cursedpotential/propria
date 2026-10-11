package embedding

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestStrictNIMRejectsGuardExpansionBeforeHTTP(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.Error(w, "unexpected embedding request", http.StatusInternalServerError)
	}))
	defer server.Close()
	nim := NIM{BaseURL: server.URL, APIKey: "synthetic-secret", StrictInput: true}
	// The original fits, but the required image guard expands it above 8000 bytes.
	text := strings.Repeat("data:image/", 681)
	if len(text) >= maxInputChars {
		t.Fatal("fixture must fit before guard preparation")
	}
	_, err := nim.Embed(context.Background(), []string{"valid first input", text})
	if err == nil || !strings.Contains(err.Error(), "prepared UTF-8 byte limit") {
		t.Fatalf("expected explicit strict-input error, got %v", err)
	}
	if strings.Contains(err.Error(), nim.APIKey) || strings.Contains(err.Error(), text) {
		t.Fatal("strict-input error disclosed credentials or input")
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("validation sent %d requests before rejecting the batch", got)
	}
}

func TestStrictNIMPreservesGuardedInputAndDisablesProviderTruncation(t *testing.T) {
	for _, text := range []string{"native é data:image/png; exact tail", strings.Repeat("é", maxInputChars/2), " \t\n"} {
		t.Run(fmt.Sprintf("input_bytes_%d", len(text)), func(t *testing.T) {
			requests := make(chan embedRequest, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request embedRequest
				if r.URL.Path != "/embeddings" {
					t.Errorf("unexpected endpoint %q", r.URL.Path)
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Errorf("decode synthetic request: %v", err)
					http.Error(w, "invalid request", http.StatusBadRequest)
					return
				}
				requests <- request
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"data":[{"index":0,"embedding":[1,2]}]}`))
			}))
			defer server.Close()
			nim := NIM{BaseURL: server.URL, APIKey: "synthetic-secret", Dimensions: 2, StrictInput: true}
			vectors, err := nim.Embed(context.Background(), []string{text})
			if err != nil {
				t.Fatal(err)
			}
			request := <-requests
			want := strings.ReplaceAll(text, "data:image/", "data: image/")
			if strings.TrimSpace(want) == "" {
				want = blankPlaceholder
			}
			if len(request.Input) != 1 || request.Input[0] != want || request.Truncate != "NONE" {
				t.Fatalf("strict request changed text or truncation mode: %+v", request)
			}
			if len(vectors) != 1 || len(vectors[0]) != 2 || vectors[0][1] != 2 {
				t.Fatalf("unexpected vectors %v", vectors)
			}
		})
	}
}

func TestStrictNIMOversizedResponseIsNotRetried(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusServiceUnavailable} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var requests, sleeps atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				w.WriteHeader(status)
				_, _ = w.Write([]byte(strings.Repeat("x", strictResponseBytes+1)))
			}))
			defer server.Close()
			nim := NIM{BaseURL: server.URL, APIKey: "synthetic-secret", StrictInput: true, Retries: 3,
				Sleep: func(context.Context, time.Duration) error { sleeps.Add(1); return nil }}
			_, err := nim.Embed(context.Background(), []string{"bounded native input"})
			if err == nil || !strings.Contains(err.Error(), "strict response exceeds 4 MiB limit") {
				t.Fatalf("expected strict response cap error, got %v", err)
			}
			if requests.Load() != 1 || sleeps.Load() != 0 {
				t.Fatalf("oversized response retried: requests=%d sleeps=%d", requests.Load(), sleeps.Load())
			}
			if strings.Contains(err.Error(), nim.APIKey) {
				t.Fatal("response error disclosed credentials")
			}
		})
	}
}

func TestDefaultNIMPreservesLegacyGuardTruncationAndProviderMode(t *testing.T) {
	requests := make(chan embedRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request embedRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode synthetic request: %v", err)
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		requests <- request
		_, _ = w.Write([]byte(`{"data":[{"index":0,"embedding":[1]}]}`))
	}))
	defer server.Close()
	// Leave StrictInput at its zero value to exercise every existing caller's mode.
	nim := NIM{BaseURL: server.URL, APIKey: "synthetic-secret", Dimensions: 1}
	text := strings.Repeat("data:image/", 681)
	if _, err := nim.Embed(context.Background(), []string{text}); err != nil {
		t.Fatal(err)
	}
	request := <-requests
	want := strings.ReplaceAll(text, "data:image/", "data: image/")[:maxInputChars]
	if len(request.Input) != 1 || request.Input[0] != want || request.Truncate != "END" {
		t.Fatalf("legacy request changed: %+v", request)
	}
}

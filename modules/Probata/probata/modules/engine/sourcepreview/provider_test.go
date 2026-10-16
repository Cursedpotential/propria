package sourcepreview

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/acquisition"
	"github.com/Cursedpotential/probata/engine/objectstores"
)

func TestPinnedRangeAgainstSyntheticObjectServer(t *testing.T) {
	fixture := "<smses><sms body='synthetic'/></smses>"
	var ranges []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-Match") != "\"fixture-etag\"" || r.URL.Query().Get("versionId") != "version-one" {
			t.Error("missing exact source pins")
		}
		w.Header().Set("ETag", "\"fixture-etag\"")
		w.Header().Set("X-Amz-Version-Id", "version-one")
		if r.Method == http.MethodHead {
			w.Header().Set("Content-Length", fmt.Sprint(len(fixture)))
			return
		}
		ranges = append(ranges, r.Header.Get("Range"))
		w.Header().Set("Content-Range", fmt.Sprintf("bytes 5-%d/%d", len(fixture)-1, len(fixture)))
		w.Header().Set("Content-Length", fmt.Sprint(len(fixture)-5))
		w.WriteHeader(206)
		io.WriteString(w, fixture[5:])
	}))
	defer server.Close()
	client, err := acquisition.NewS3Client(acquisition.ObjectStorageConfig{Endpoint: server.URL, Region: "auto", AccessKeyID: "synthetic", SecretAccessKey: "synthetic"})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := objectstores.ParseRoots(`[{"id":"synthetic","label":"Synthetic","url":"b2://synthetic/"}]`)
	if err != nil {
		t.Fatal(err)
	}
	p := &Provider{Clients: map[string]ObjectClient{"b2": client}, Roots: roots}
	source, reader, err := p.Open(context.Background(), Request{SourceRef: "b2://synthetic/messages.xml", ExpectedETag: "\"fixture-etag\"", VersionID: "version-one"}, 5)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil || string(data) != fixture[5:] || source.ByteLength != int64(len(fixture)) || len(ranges) != 1 || ranges[0] != fmt.Sprintf("bytes=5-%d", len(fixture)-1) {
		t.Fatal("range mismatch")
	}
	if _, _, err = p.Open(context.Background(), Request{SourceRef: "b2://other/messages.xml", ExpectedETag: "\"fixture-etag\""}, 0); err != ErrInvalid {
		t.Fatal("unauthorized source accepted")
	}
}
func TestProviderRejectsUnpinnedOrIgnoredRange(t *testing.T) {
	for _, mutation := range []string{"etag", "version", "range"} {
		t.Run(mutation, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("ETag", "\"right\"")
				w.Header().Set("X-Amz-Version-Id", "v1")
				w.Header().Set("Content-Length", "10")
				if mutation == "etag" {
					w.Header().Set("ETag", "\"wrong\"")
				}
				if mutation == "version" {
					w.Header().Set("X-Amz-Version-Id", "v2")
				}
				if r.Method != http.MethodHead {
					io.WriteString(w, strings.Repeat("x", 10))
				}
			}))
			defer server.Close()
			client, _ := acquisition.NewS3Client(acquisition.ObjectStorageConfig{Endpoint: server.URL, Region: "auto", AccessKeyID: "synthetic", SecretAccessKey: "synthetic"})
			roots, _ := objectstores.ParseRoots(`[{"id":"synthetic","label":"Synthetic","url":"b2://synthetic/"}]`)
			p := &Provider{Clients: map[string]ObjectClient{"b2": client}, Roots: roots}
			if _, reader, err := p.Open(context.Background(), Request{SourceRef: "b2://synthetic/test.xml", ExpectedETag: "\"right\"", VersionID: "v1"}, 0); err != ErrChanged {
				if reader != nil {
					reader.Close()
				}
				t.Fatal("pin/range mismatch accepted")
			}
		})
	}
}

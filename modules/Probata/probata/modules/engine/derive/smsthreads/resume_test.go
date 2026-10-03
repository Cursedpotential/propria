// Byline: Claude Code · Opus 5.5 · 2026-10-02

package smsthreads

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"syscall"
	"testing"
	"time"
)

// flakyObject serves one object and breaks each stream after cut bytes with
// failure, for as many streams as breaks says.
type flakyObject struct {
	data    []byte
	etag    string
	cut     int
	breaks  int
	failure error
	ranges  []int64
	etags   []string
}

type cutReader struct {
	r       io.Reader
	left    int
	failure error
}

func (c *cutReader) Read(p []byte) (int, error) {
	if c.left <= 0 {
		return 0, c.failure
	}
	if len(p) > c.left {
		p = p[:c.left]
	}
	n, err := c.r.Read(p)
	c.left -= n
	return n, err
}

func (f *flakyObject) get(_ context.Context, from int64, etag string) (io.ReadCloser, string, int64, error) {
	f.ranges, f.etags = append(f.ranges, from), append(f.etags, etag)
	if etag != "" && etag != f.etag {
		return nil, "", -1, errors.New("PreconditionFailed: 412")
	}
	var body io.Reader = bytes.NewReader(f.data[from:])
	if f.breaks > 0 {
		f.breaks--
		body = &cutReader{r: body, left: f.cut, failure: f.failure}
	}
	return io.NopCloser(body), f.etag, int64(len(f.data)), nil
}

func noWait(r *resumingBody) *resumingBody {
	r.sleep = func(context.Context, time.Duration) error { return nil }
	return r
}

func TestABrokenStreamResumesFromItsOffsetPinnedToTheETag(t *testing.T) {
	data := []byte(strings.Repeat("<sms body=\"x\"/>", 4000))
	for _, failure := range []error{io.ErrUnexpectedEOF, syscall.ECONNRESET, io.EOF} {
		object := &flakyObject{data: data, etag: `"v1"`, cut: 7001, breaks: 3, failure: failure}
		body, err := openResuming(context.Background(), object.get)
		if err != nil {
			t.Fatal(err)
		}
		got, err := io.ReadAll(noWait(body))
		if err != nil {
			t.Fatalf("%v: %v", failure, err)
		}
		if !bytes.Equal(got, data) {
			t.Fatalf("%v: read %d bytes, want the whole %d unchanged", failure, len(got), len(data))
		}
		if want := []int64{0, 7001, 14002, 21003}; len(object.ranges) != len(want) {
			t.Fatalf("%v: ranges %v, want %v", failure, object.ranges, want)
		}
		for i, from := range []int64{0, 7001, 14002, 21003} {
			if object.ranges[i] != from {
				t.Fatalf("%v: ranges %v", failure, object.ranges)
			}
		}
		if object.etags[0] != "" || object.etags[1] != `"v1"` || body.Resumes != 3 {
			t.Fatalf("%v: etags %v resumes %d", failure, object.etags, body.Resumes)
		}
	}
}

func TestAResumeRefusesAChangedObject(t *testing.T) {
	object := &flakyObject{data: []byte(strings.Repeat("a", 5000)), etag: `"v1"`, cut: 100, breaks: 1, failure: io.ErrUnexpectedEOF}
	body, err := openResuming(context.Background(), object.get)
	if err != nil {
		t.Fatal(err)
	}
	object.etag = `"v2"`
	if _, err := io.ReadAll(noWait(body)); err == nil || !strings.Contains(err.Error(), "412") {
		t.Fatalf("a changed object must stop the read, got %v", err)
	}
}

func TestReconnectsThatMakeNoProgressAreBounded(t *testing.T) {
	object := &flakyObject{data: []byte(strings.Repeat("a", 5000)), etag: `"v1"`, cut: 0, breaks: 100, failure: io.ErrUnexpectedEOF}
	body, err := openResuming(context.Background(), object.get)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadAll(noWait(body)); err == nil || !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("want the bounded failure, got %v", err)
	}
	if body.Resumes != maxConsecutiveResumes {
		t.Fatalf("resumes %d, want %d", body.Resumes, maxConsecutiveResumes)
	}
}

func TestACancelledReadAndARefusalAreNotResumed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	object := &flakyObject{data: []byte(strings.Repeat("a", 5000)), etag: `"v1"`, cut: 10, breaks: 1, failure: io.ErrUnexpectedEOF}
	body, err := openResuming(ctx, object.get)
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	if _, err := io.ReadAll(noWait(body)); err == nil || body.Resumes != 0 {
		t.Fatalf("a cancelled read resumed: err %v resumes %d", err, body.Resumes)
	}

	refusal := errors.New("AccessDenied")
	object = &flakyObject{data: []byte(strings.Repeat("a", 5000)), etag: `"v1"`, cut: 10, breaks: 1, failure: refusal}
	body, err = openResuming(context.Background(), object.get)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadAll(noWait(body)); !errors.Is(err, refusal) || body.Resumes != 0 {
		t.Fatalf("a refusal resumed: err %v resumes %d", err, body.Resumes)
	}
}

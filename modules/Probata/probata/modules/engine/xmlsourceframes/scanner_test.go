package xmlsourceframes

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func frame(t *testing.T, input string, limit int64) ([]Span, Receipt) {
	t.Helper()
	var spans []Span
	r, err := Scan(context.Background(), io.NopCloser(strings.NewReader(input)), limit, func(s Span) error { spans = append(spans, s); return nil })
	if err != nil {
		t.Fatal(err)
	}
	end := int64(0)
	for _, s := range spans {
		if s.Start != end || s.End <= s.Start {
			t.Fatalf("nonpartition span %+v after %d", s, end)
		}
		end = s.End
	}
	if end != int64(len(input)) || r.Bytes != end {
		t.Fatalf("missing bytes %d %+v", end, r)
	}
	return spans, r
}

func TestOriginalSpansPreserveOrderEntitiesUTF8AndEnvelope(t *testing.T) {
	input := "<?xml version=\"1.0\"?>\n<!DOCTYPE root [<!ENTITY custom 'original'>]><root xmlns:x='urn:original'>\n<!-- </fake> --><x:item a='é &amp; &#10; &custom;'>a<![CDATA[</x:item>]]><child z='>'/></x:item>\n<other/>\n</root>"
	spans, r := frame(t, input, 4096)
	if !r.Complete || r.Elements != 2 {
		t.Fatalf("%+v", r)
	}
	var elements []Span
	for _, s := range spans {
		if s.Kind == "element" {
			elements = append(elements, s)
		}
	}
	if elements[0].Name != "x:item" || elements[1].Name != "other" || elements[0].Ordinal != 1 || elements[1].Ordinal != 2 {
		t.Fatal(elements)
	}
	if !strings.HasPrefix(input[elements[0].Start:elements[0].End], "<x:item") || input[elements[1].Start:elements[1].End] != "<other/>" {
		t.Fatal(elements)
	}
}

type repeatedByte struct{ remaining int64 }

func (r *repeatedByte) Read(p []byte) (int, error) {
	if r.remaining == 0 {
		return 0, io.EOF
	}
	n := len(p)
	if int64(n) > r.remaining {
		n = int(r.remaining)
	}
	for i := range p[:n] {
		p[i] = 'A'
	}
	r.remaining -= int64(n)
	return n, nil
}

func TestOversizedAttributeRetainsExactRangeAndFollowingElement(t *testing.T) {
	const payload = int64(65 * 1024 * 1024)
	prefix := "<root><item><part data='"
	suffix := "'/></item><following body='intact'/></root>"
	reader := io.MultiReader(strings.NewReader(prefix), &repeatedByte{remaining: payload}, strings.NewReader(suffix))
	var spans []Span
	r, err := Scan(context.Background(), io.NopCloser(reader), 1024, func(s Span) error { spans = append(spans, s); return nil })
	if err != nil || !r.Complete || r.Elements != 2 || r.Oversized != 1 {
		t.Fatalf("%+v %v", r, err)
	}
	end := int64(0)
	var elements []Span
	for _, s := range spans {
		if s.Start != end {
			t.Fatal("gap", spans)
		}
		end = s.End
		if s.Kind == "element" {
			elements = append(elements, s)
		}
	}
	if end != int64(len(prefix)+len(suffix))+payload || elements[0].Status != "oversized" || elements[1].Name != "following" || elements[1].Status != "complete" {
		t.Fatal(spans)
	}
	t.Logf("source_bytes=%d spans=%d oversized=%d following_ordinal=%d", end, len(spans), r.Oversized, elements[1].Ordinal)
	if status, err := os.ReadFile("/proc/self/status"); err == nil {
		for _, line := range strings.Split(string(status), "\n") {
			if strings.HasPrefix(line, "VmHWM:") {
				t.Log(line)
			}
		}
	}
}

func TestMalformedAndTruncatedRemaindersAreAccountedWithoutRepair(t *testing.T) {
	for _, input := range []string{"<root><ok/><bad></wrong><next/></root>", "<root><ok/><bad value='cut", "<root><ok/>", "<root/> <second/>"} {
		spans, r := frame(t, input, 4096)
		if r.Complete || (input != "<root><ok/>" && spans[len(spans)-1].Kind != "unparsed") {
			t.Fatalf("%q %+v %+v", input, spans, r)
		}
	}
}

type blockedReader struct{ closed chan struct{} }

func (b *blockedReader) Read([]byte) (int, error) { <-b.closed; return 0, io.ErrClosedPipe }
func (b *blockedReader) Close() error {
	select {
	case <-b.closed:
	default:
		close(b.closed)
	}
	return nil
}
func TestCancellationClosesBlockedRead(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	r := &blockedReader{closed: make(chan struct{})}
	done := make(chan error, 1)
	go func() { _, err := Scan(ctx, r, 1024, func(Span) error { return nil }); done <- err }()
	cancel()
	select {
	case err := <-done:
		if err != context.Canceled {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancellation blocked")
	}
}

func TestRetryProducesIdenticalReferenceOnlySpans(t *testing.T) {
	input := "<root><item body='native'/><item body='following'/></root>"
	a, ar := frame(t, input, 1024)
	b, br := frame(t, input, 1024)
	if ar != br || len(a) != len(b) {
		t.Fatal("retry changed receipt")
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatal("retry changed spans")
		}
	}
}

func TestGigabyteStreamRemainsReferenceOnly(t *testing.T) {
	const payload = int64(1024 * 1024 * 1024)
	prefix, suffix := "<root><item value='", "'/><after/></root>"
	reader := io.MultiReader(strings.NewReader(prefix), &repeatedByte{remaining: payload}, strings.NewReader(suffix))
	count := 0
	end := int64(0)
	r, err := Scan(context.Background(), io.NopCloser(reader), 1024, func(s Span) error {
		if s.Start != end {
			t.Fatal("partition gap")
		}
		end = s.End
		count++
		return nil
	})
	if err != nil || !r.Complete || r.Elements != 2 || r.Oversized != 1 || r.Bytes != payload+int64(len(prefix)+len(suffix)) || end != r.Bytes {
		t.Fatalf("%+v %v", r, err)
	}
	t.Logf("source_bytes=%d spans=%d", r.Bytes, count)
	if status, err := os.ReadFile("/proc/self/status"); err == nil {
		for _, line := range strings.Split(string(status), "\n") {
			if strings.HasPrefix(line, "VmHWM:") {
				t.Log(line)
			}
		}
	}
}

package sourcepreview

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"runtime"
	"strings"
	"testing"
	"unicode/utf8"
)

func scanFixture(t *testing.T, fixture string, rows int) []Message {
	t.Helper()
	state := State{}
	source := Source{ByteLength: int64(len(fixture)), ETag: "fixture", SourceRef: "b2://synthetic/messages.xml"}
	var messages []Message
	for pages := 0; !state.Done; pages++ {
		if pages > 100 {
			t.Fatal("no bounded forward progress")
		}
		result, err := Scan(context.Background(), strings.NewReader(fixture[state.Offset:]), source, state, rows)
		if err != nil {
			t.Fatal(err)
		}
		if result.Page.ScannedBytes <= 0 || result.Page.ScannedBytes > MaxScanBytes {
			t.Fatal("bad read bound")
		}
		encoded, _ := json.Marshal(result)
		if len(encoded) > MaxPayloadBytes {
			t.Fatal("unbounded page")
		}
		messages = append(messages, result.Page.Messages...)
		state = result.State
	}
	return messages
}

func TestPaginationPreservesEveryRecordAndEntities(t *testing.T) {
	var fixture strings.Builder
	fixture.WriteString("<?xml version='1.0'?><smses>")
	for i := 0; i < 251; i++ {
		fmt.Fprintf(&fixture, "<sms date='%d' type='1' address='α' body='你好 &amp; &quot;quoted&quot; &#10;' />", i)
	}
	fixture.WriteString("</smses>")
	messages := scanFixture(t, fixture.String(), 7)
	if len(messages) != 251 {
		t.Fatalf("lost records: %d", len(messages))
	}
	seen := map[string]bool{}
	for i, m := range messages {
		if seen[m.ID] || m.DateMS == nil || *m.DateMS != int64(i) || m.Body != "你好 & \"quoted\" \n" || m.Participants[0] != "α" {
			t.Fatalf("record %d mismatch: %#v", i, m)
		}
		seen[m.ID] = true
	}
}
func TestMassiveMMSAttributeResumesAndNeverReturnsPayload(t *testing.T) {
	fixture := "<smses><sms body='before'/><mms date='123' msg_box='2'><parts><part ct='image/jpeg' cl='photo.jpg' data='" + strings.Repeat("A", 4<<20) + "'/><part ct='text/plain' text='after photo'/></parts><addrs><addr address='+15550000'/></addrs></mms><sms body='after'/></smses>"
	messages := scanFixture(t, fixture, MaxRows)
	if len(messages) != 3 || messages[1].Body != "after photo" || messages[1].Attachments[0].EncodedBytes != 4<<20 || messages[2].Body != "after" {
		t.Fatalf("bad resume: %#v", messages)
	}
	encoded, _ := json.Marshal(messages)
	if len(encoded) > 2048 {
		t.Fatal("attachment payload escaped")
	}
}

type repeatedByteReader struct{ remaining int64 }

func (r *repeatedByteReader) Read(p []byte) (int, error) {
	if r.remaining == 0 {
		return 0, io.EOF
	}
	n := len(p)
	if int64(n) > r.remaining {
		n = int(r.remaining)
	}
	for i := 0; i < n; i++ {
		p[i] = 'A'
	}
	r.remaining -= int64(n)
	return n, nil
}
func TestGiantAttributeAllocationIsIndependentOfSize(t *testing.T) {
	const payload = 16 << 20
	header := "<smses><mms><parts><part ct='image/jpeg' data='"
	footer := "'/></parts></mms></smses>"
	reader := io.MultiReader(strings.NewReader(header), &repeatedByteReader{payload}, strings.NewReader(footer))
	source := Source{ByteLength: int64(len(header) + payload + len(footer))}
	state := State{}
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for !state.Done {
		result, err := Scan(context.Background(), reader, source, state, MaxRows)
		if err != nil {
			t.Fatal(err)
		}
		state = result.State
	}
	runtime.ReadMemStats(&after)
	if allocated := after.TotalAlloc - before.TotalAlloc; allocated > 2<<20 {
		t.Fatalf("allocated %d for streamed giant attribute", allocated)
	}
}
func TestBodyTruncationKeepsUTF8AndHardRows(t *testing.T) {
	body := strings.Repeat("é", 5000)
	messages := scanFixture(t, "<smses><sms body='"+body+"'/><sms body='two'/></smses>", 1)
	if len(messages) != 2 || !messages[0].BodyTruncated || len(messages[0].Body) > MaxBodyBytes || !utf8.ValidString(messages[0].Body) {
		t.Fatal("body cap/UTF8 failed")
	}
}
func TestCancellationAndHostileXML(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Scan(ctx, strings.NewReader("<smses/>"), Source{ByteLength: 8}, State{}, 1); err != context.Canceled {
		t.Fatalf("cancel: %v", err)
	}
	for _, fixture := range []string{"<!DOCTYPE smses [<!ENTITY x SYSTEM 'file:///private'>]><smses></smses>", "<smses><sms body='broken&entity;'/></smses>", "<smses><mms></smses>"} {
		if _, err := Scan(context.Background(), strings.NewReader(fixture), Source{ByteLength: int64(len(fixture))}, State{}, 1); err != ErrInvalid {
			t.Fatalf("hostile XML accepted: %v", err)
		}
	}
}
func TestCommentCannotCreateMessages(t *testing.T) {
	messages := scanFixture(t, "<smses><!-- <sms body='fake'/> --><sms body='real'/></smses>", 5)
	if len(messages) != 1 || messages[0].Body != "real" {
		t.Fatal("comment parsed as record")
	}
}

func TestOversizedRecordSkipsBoundedlyAndFollowingSMSRemainsReachable(t *testing.T) {
	payload := MaxRecordBytes + (2 << 20)
	header := "<smses><mms date='123'><parts><part ct='image/jpeg' name='oversized.jpg' data='"
	footer := "'/></parts></mms><sms date='456' body='following message'/></smses>"
	reader := io.MultiReader(strings.NewReader(header), &repeatedByteReader{payload}, strings.NewReader(footer))
	source := Source{ByteLength: int64(len(header)+len(footer)) + payload}
	state := State{}
	var messages []Message
	pages := 0
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for !state.Done {
		result, err := Scan(context.Background(), reader, source, state, MaxRows)
		if err != nil {
			t.Fatal(err)
		}
		if result.Page.ScannedBytes <= 0 || result.Page.ScannedBytes > MaxScanBytes {
			t.Fatal("bounded forward progress failed")
		}
		state = result.State
		messages = append(messages, result.Page.Messages...)
		pages++
		if pages > 70 {
			t.Fatal("oversized record prevented progress")
		}
	}
	runtime.ReadMemStats(&after)
	if allocated := after.TotalAlloc - before.TotalAlloc; allocated > 4<<20 {
		t.Fatalf("allocated %d while skipping66MiB record", allocated)
	}
	if len(messages) != 2 || messages[0].Kind != "mms" || !messages[0].MetadataTruncated || !messages[0].BodyTruncated || messages[1].Body != "following message" || messages[0].ID == messages[1].ID {
		t.Fatalf("oversized record/later SMS mismatch: %#v", messages)
	}
}

func TestUnknownDateRemainsNullAndCursorLengthIsPinned(t *testing.T) {
	messages := scanFixture(t, "<smses><sms date='null'/><sms date='not-a-clock'/><sms date='123'/></smses>", 1)
	if messages[0].DateMS != nil || messages[1].DateMS != nil || messages[2].DateMS == nil || *messages[2].DateMS != 123 {
		t.Fatal("invented/invalid timestamp")
	}
	if _, err := Scan(context.Background(), strings.NewReader("x"), Source{ByteLength: 100}, State{Offset: 1, SourceLength: 99}, 1); err != ErrChanged {
		t.Fatal("changed length cursor accepted")
	}
}

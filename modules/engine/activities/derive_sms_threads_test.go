// Byline: Claude Code · Fable 5.1 · 2026-09-20
package activities

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
)

type memoryObjectStore struct {
	mu      sync.Mutex
	objects map[string][]byte
	puts    int
}

func (m *memoryObjectStore) Open(_ context.Context, bucket, key string) (io.ReadCloser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	body, ok := m.objects[bucket+"/"+key]
	if !ok {
		return nil, errors.New("no such object")
	}
	return io.NopCloser(bytes.NewReader(body)), nil
}

func (m *memoryObjectStore) Put(_ context.Context, bucket, key string, body io.ReadSeeker, _ int64, _ string) error {
	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.objects[bucket+"/"+key] = data
	m.puts++
	return nil
}

func (m *memoryObjectStore) Exists(_ context.Context, bucket, key string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.objects[bucket+"/"+key]
	return ok, nil
}

func smsBackupFixture() string {
	image := base64.StdEncoding.EncodeToString([]byte("\x89PNG\r\n\x1a\nreal-bytes"))
	return `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<smses count="3">
  <sms protocol="0" address="8105550101" date="1700000000000" type="1" body="hello" read="1" status="-1" contact_name="A" />
  <sms protocol="0" address="8105550202" date="1700000060000" type="2" body="reply" read="1" status="-1" contact_name="B" />
  <mms date="1700000180000" msg_box="1" address="8105550101" m_type="132" contact_name="A">
    <parts>
      <part seq="0" ct="image/png" name="pic.png" cl="pic.png" data="` + image + `" />
      <part seq="0" ct="image/heif" name="null" cl="image000000_11449.jpg" text="null" data="" />
      <part seq="1" ct="text/plain" name="null" text="see picture" />
    </parts>
    <addrs><addr address="8105550101" type="137" charset="106" /></addrs>
  </mms>
</smses>
`
}

func deriveActivity(t *testing.T, store *memoryObjectStore) DeriveSMSThreadsActivities {
	t.Helper()
	return DeriveSMSThreadsActivities{
		ScratchRoot: t.TempDir(),
		Stores: func(scheme string) (smsthreads.ObjectStore, error) {
			if scheme != "b2" {
				return nil, errors.New("not configured")
			}
			return store, nil
		},
	}
}

func TestDeriveSMSThreadsPublishesBesideSourceAndIsSafeToRetry(t *testing.T) {
	store := &memoryObjectStore{objects: map[string][]byte{"bkt/vault/sms-1.xml": []byte(smsBackupFixture())}}
	activity := deriveActivity(t, store)
	request := DeriveSMSThreadsRequest{SourceURI: "b2://bkt/vault/sms-1.xml"}

	first, err := activity.DeriveSMSThreads(context.Background(), request)
	if err != nil {
		t.Fatalf("DeriveSMSThreads() error = %v", err)
	}
	if first.ManifestURI != "b2://bkt/vault/sms-1.xml.derived/manifest.json" || first.AlreadyDerived ||
		first.Records != 3 || first.Rejected != 0 || first.MediaObjects != 1 || first.ThreadChunks < 2 ||
		len(first.SourceSHA256) != 64 || first.SourceBytes != int64(len(smsBackupFixture())) {
		t.Fatalf("unexpected first result: %+v", first)
	}
	if !bytes.Equal(store.objects["bkt/vault/sms-1.xml"], []byte(smsBackupFixture())) {
		t.Fatal("the original object was modified")
	}

	// A Temporal retry after a finished first attempt returns the same manifest
	// and writes nothing.
	putsBefore := store.puts
	second, err := activity.DeriveSMSThreads(context.Background(), request)
	if err != nil {
		t.Fatalf("retry after completion failed: %v", err)
	}
	if !second.AlreadyDerived || store.puts != putsBefore {
		t.Fatalf("retry rewrote objects: already=%v puts %d -> %d", second.AlreadyDerived, putsBefore, store.puts)
	}
	second.AlreadyDerived = false
	if second != first {
		t.Fatalf("retry result differs:\n first=%+v\nsecond=%+v", first, second)
	}
}

func TestDeriveSMSThreadsRejectsBadLocatorsWithoutRetrying(t *testing.T) {
	store := &memoryObjectStore{objects: map[string][]byte{}}
	activity := deriveActivity(t, store)
	for _, uri := range []string{"", "vault/sms.xml", "b2://bkt", "b2:///key.xml", "gs://bkt/key.xml"} {
		_, err := activity.DeriveSMSThreads(context.Background(), DeriveSMSThreadsRequest{SourceURI: uri})
		if !nonRetryable(err) {
			t.Fatalf("source_uri %q must be a permanent failure, got %v", uri, err)
		}
	}
	// A missing object may be a storage hiccup or a late upload: keep retrying.
	_, err := activity.DeriveSMSThreads(context.Background(), DeriveSMSThreadsRequest{SourceURI: "b2://bkt/absent.xml"})
	if err == nil || nonRetryable(err) || !strings.Contains(err.Error(), "derive sms threads") {
		t.Fatalf("missing object: %v", err)
	}
}

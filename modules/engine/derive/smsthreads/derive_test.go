// Byline: Claude Code · Fable 5.1 · 2026-09-20

package smsthreads

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

type memoryStore struct{ objects map[string][]byte }

func (m *memoryStore) Open(_ context.Context, bucket, key string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(m.objects[bucket+"/"+key])), nil
}

func (m *memoryStore) Put(_ context.Context, bucket, key string, body io.ReadSeeker, _ int64, _ string) error {
	data, err := io.ReadAll(body)
	m.objects[bucket+"/"+key] = data
	return err
}

func (m *memoryStore) Exists(_ context.Context, bucket, key string) (bool, error) {
	_, ok := m.objects[bucket+"/"+key]
	return ok, nil
}

func sampleBackup(image []byte) string {
	return `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<smses count="4">
  <sms protocol="0" address="+1 (810) 555-0101" date="1700000000000" type="1" body="hello from A" read="1" status="-1" contact_name="A" />
  <sms protocol="0" address="8105550101" date="1700000060000" type="2" body="reply to A" read="1" status="-1" contact_name="A" />
  <sms protocol="0" address="8105550202" date="1700000120000" type="1" body="hello from B" read="1" status="-1" contact_name="B" />
  <mms date="1700000180000" msg_box="1" address="8105550101" m_type="132" contact_name="A">
    <parts>
      <part seq="0" ct="image/png" name="pic.png" cl="pic.png" data="` + base64.StdEncoding.EncodeToString(image) + `" />
      <part seq="1" ct="text/plain" name="null" text="see picture" />
    </parts>
    <addrs>
      <addr address="8105550101" type="137" charset="106" />
      <addr address="8105559999" type="151" charset="106" />
    </addrs>
  </mms>
</smses>
`
}

func TestDeriveWritesThreadsMediaAndManifestBesideSource(t *testing.T) {
	image := []byte("\x89PNG\r\n\x1a\nnot-a-real-image-but-real-bytes")
	source := sampleBackup(image)
	store := &memoryStore{objects: map[string][]byte{"bkt/vault/sms-test.xml": []byte(source)}}
	manifest, err := Derive(context.Background(), Options{
		Store: store, Scheme: "b2", Bucket: "bkt", Key: "vault/sms-test.xml", ScratchRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	wantSource := sha256.Sum256([]byte(source))
	if manifest.SourceSHA256 != hex.EncodeToString(wantSource[:]) || manifest.SourceBytes != int64(len(source)) {
		t.Fatalf("source digest/size not over the whole object: %+v", manifest)
	}
	if manifest.Records != 4 || manifest.Rejected != 0 {
		t.Fatalf("records=%d rejected=%d", manifest.Records, manifest.Rejected)
	}
	if len(manifest.Threads) < 2 {
		t.Fatalf("expected at least two threads, got %+v", manifest.Threads)
	}
	var lines, withMedia int
	for _, thread := range manifest.Threads {
		if !strings.HasPrefix(thread.Key, "vault/sms-test.xml.derived/threads/") {
			t.Fatalf("thread not beside the source: %s", thread.Key)
		}
		body := store.objects["bkt/"+thread.Key]
		digest := sha256.Sum256(body)
		if hex.EncodeToString(digest[:]) != thread.SHA256 {
			t.Fatalf("thread digest mismatch for %s", thread.Key)
		}
		for _, raw := range bytes.Split(bytes.TrimSpace(body), []byte("\n")) {
			var line Line
			if err := json.Unmarshal(raw, &line); err != nil {
				t.Fatalf("invalid NDJSON line in %s: %v", thread.Key, err)
			}
			lines++
			for _, attachment := range line.Attachments {
				withMedia++
				key := strings.TrimPrefix(attachment.URI, "b2://bkt/")
				if !bytes.Equal(store.objects["bkt/"+key], image) {
					t.Fatalf("media object %s does not hold the decoded bytes", key)
				}
				if !strings.HasPrefix(key, "vault/sms-test.xml.derived/media/") || !strings.HasSuffix(key, ".png") {
					t.Fatalf("media not beside the source: %s", key)
				}
			}
		}
	}
	if lines != 4 || withMedia != 1 || manifest.MediaObjects != 1 {
		t.Fatalf("lines=%d media refs=%d media objects=%d", lines, withMedia, manifest.MediaObjects)
	}
	if _, ok := store.objects["bkt/vault/sms-test.xml.derived/manifest.json"]; !ok {
		t.Fatal("manifest was not published")
	}
	if !bytes.Equal(store.objects["bkt/vault/sms-test.xml"], []byte(source)) {
		t.Fatal("the original object was modified")
	}
	if _, err := Derive(context.Background(), Options{
		Store: store, Scheme: "b2", Bucket: "bkt", Key: "vault/sms-test.xml", ScratchRoot: t.TempDir(),
	}); err == nil {
		t.Fatal("a finished derivation must never be overwritten")
	}
}

func TestNormalizePartyJoinsNumberSpellings(t *testing.T) {
	for _, value := range []string{"+1 (810) 555-0101", "18105550101", "810-555-0101"} {
		if got := normalizeParty(value); got != "8105550101" {
			t.Fatalf("%q -> %q", value, got)
		}
	}
	if got := normalizeParty("Verizon@vtext.com"); got != "verizon@vtext.com" {
		t.Fatalf("got %q", got)
	}
}

// Byline: Claude Code · Opus 5.5 · 2026-09-25

package activities

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"

	"go.temporal.io/sdk/temporal"

	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
	"github.com/Cursedpotential/probata/engine/objectstores"
	"github.com/Cursedpotential/probata/engine/repairplan"
	"github.com/Cursedpotential/probata/engine/stagegraph"
)

// repairStore is an in-memory object store with HEAD, ranged reads and
// large uploads. It records whole-object opens and ranged reads.
type repairStore struct {
	mu      sync.Mutex
	objects map[string][]byte
	puts    []string
	opens   []string
	ranges  []string
}

func (s *repairStore) ReadRange(_ context.Context, bucket, key string, offset, length int64) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ranges = append(s.ranges, fmt.Sprintf("%s@%d+%d", key, offset, length))
	data, ok := s.objects[bucket+"/"+key]
	if !ok {
		return nil, errors.New("no such object: " + key)
	}
	if offset >= int64(len(data)) {
		return []byte{}, nil
	}
	end := offset + length
	if end > int64(len(data)) {
		end = int64(len(data))
	}
	return append([]byte(nil), data[offset:end]...), nil
}

func newRepairStore(objects map[string][]byte) *repairStore {
	return &repairStore{objects: objects}
}

func (s *repairStore) Open(_ context.Context, bucket, key string) (io.ReadCloser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.opens = append(s.opens, key)
	data, ok := s.objects[bucket+"/"+key]
	if !ok {
		return nil, errors.New("no such object: " + key)
	}
	return io.NopCloser(bytes.NewReader(append([]byte(nil), data...))), nil
}

func (s *repairStore) Put(_ context.Context, bucket, key string, body io.ReadSeeker, _ int64, _ string) error {
	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.objects[bucket+"/"+key] = data
	s.puts = append(s.puts, bucket+"/"+key)
	return nil
}

func (s *repairStore) Exists(_ context.Context, bucket, key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.objects[bucket+"/"+key]
	return ok, nil
}

func (s *repairStore) Stat(_ context.Context, bucket, key string) (smsthreads.ObjectInfo, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, ok := s.objects[bucket+"/"+key]
	if !ok {
		return smsthreads.ObjectInfo{}, false, nil
	}
	digest := sha256.Sum256(data)
	return smsthreads.ObjectInfo{Size: int64(len(data)), ETag: hex.EncodeToString(digest[:8])}, true, nil
}

func (s *repairStore) PutLarge(ctx context.Context, bucket, key string, body io.ReaderAt, size int64, contentType string) error {
	return s.Put(ctx, bucket, key, io.NewSectionReader(body, 0, size), size, contentType)
}

func (s *repairStore) putCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.puts)
}

func storesOf(store *repairStore) func(string) (smsthreads.ObjectStore, error) {
	return func(scheme string) (smsthreads.ObjectStore, error) {
		if scheme != "b2" {
			return nil, fmt.Errorf("scheme %q is not configured", scheme)
		}
		return store, nil
	}
}

func testRoots(t *testing.T) objectstores.Roots {
	t.Helper()
	roots, err := objectstores.ParseRoots(`[{"id":"b2-bucket","label":"B2","url":"b2://salem-data/"}]`)
	if err != nil {
		t.Fatal(err)
	}
	return roots
}

func requirePermanent(t *testing.T, err error, contains string) {
	t.Helper()
	var application *temporal.ApplicationError
	if !errors.As(err, &application) || !application.NonRetryable() || !strings.Contains(err.Error(), contains) {
		t.Fatalf("err = %v, want a non-retryable failure containing %q", err, contains)
	}
}

const (
	vaultKey  = "consignatio/vault/v1/sms-20250617122400.xml"
	vaultRef  = "b2://salem-data/" + vaultKey
	dedupeKey = "consignatio/intake/raw-dedupe/v1/source-buckets/gdrive/sms-20250617122400.xml"
)

func smsBackup(image []byte) string {
	return `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<smses count="3">
  <sms protocol="0" address="8105550101" date="1700000000000" type="1" body="one" read="1" status="-1" />
  <sms protocol="0" address="8105550101" date="1700000060000" type="2" body="two" read="1" status="-1" />
  <mms date="1700000180000" msg_box="1" address="8105550101" m_type="132">
    <parts><part seq="0" ct="image/png" name="p.png" data="` + base64.StdEncoding.EncodeToString(image) + `" /></parts>
    <addrs><addr address="8105550101" type="137" charset="106" /></addrs>
  </mms>
</smses>
`
}

// --- repair.find_other_version -------------------------------------------

type fakeCatalog struct {
	rows    []CatalogObject
	byKey   map[string]CatalogObject
	queries []string
}

func (c *fakeCatalog) FindByBasename(_ context.Context, basename string, _ int) ([]CatalogObject, error) {
	c.queries = append(c.queries, basename)
	return c.rows, nil
}

func (c *fakeCatalog) LookupKey(_ context.Context, key string) (CatalogObject, bool, error) {
	row, ok := c.byKey[key]
	return row, ok, nil
}

func findActivity(t *testing.T, store *repairStore, catalog *fakeCatalog) RepairFindOtherVersionActivity {
	return RepairFindOtherVersionActivity{
		Catalog: catalog, Store: CatalogStore{Scheme: "b2", Bucket: "salem-data"},
		Stores: storesOf(store), Roots: testRoots(t),
	}
}

func TestFindOtherVersionChoosesTheLargestVerifiedCopyAndWritesNothing(t *testing.T) {
	truncated := []byte(strings.Repeat("x", 100))
	store := newRepairStore(map[string][]byte{
		"salem-data/" + vaultKey:                              truncated,
		"salem-data/" + dedupeKey:                             bytes.Repeat([]byte("y"), 500),
		"salem-data/consignatio/other/sms-20250617122400.xml": bytes.Repeat([]byte("z"), 300),
	})
	catalog := &fakeCatalog{
		byKey: map[string]CatalogObject{vaultKey: {Key: vaultKey, Size: 100, SHA1: "aaaa"}},
		rows: []CatalogObject{
			{Key: vaultKey, Size: 100, SHA1: "aaaa", Snapshot: "vault_index_source_20260918"},
			{Key: "consignatio/twin/sms-20250617122400.xml", Size: 100, SHA1: "aaaa", Snapshot: "b2_objects"},
			{Key: dedupeKey, Size: 500, Snapshot: "b2_objects"},
			{Key: "consignatio/other/sms-20250617122400.xml", Size: 300, SHA1: "bbbb", Snapshot: "b2_objects"},
			{Key: "consignatio/gone/sms-20250617122400.xml", Size: 900, Snapshot: "b2_objects"},
		},
	}
	result, err := findActivity(t, store, catalog).FindOtherVersion(context.Background(), repairplan.StepRequest{
		SourceRef: vaultRef, SourceType: repairplan.TypeSMSBackupXML, Params: json.RawMessage(`{"max_candidates":3}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.OutputRef != "b2://salem-data/"+dedupeKey || result.OutputKind != repairplan.OutputExistingObject ||
		result.OutputType != repairplan.TypeSMSBackupXML || result.OutputSHA256 != "" {
		t.Fatalf("result = %+v", result)
	}
	var summary struct {
		Candidates []struct {
			SourceRef string `json:"source_ref"`
			Size      int64  `json:"size"`
		} `json:"candidates"`
		Rejected     map[string]int `json:"rejected"`
		OriginalSize int64          `json:"original_size"`
	}
	if err := json.Unmarshal(result.Summary, &summary); err != nil {
		t.Fatal(err)
	}
	if len(summary.Candidates) != 2 || summary.Candidates[1].Size != 300 || summary.OriginalSize != 100 {
		t.Fatalf("summary = %s", result.Summary)
	}
	if summary.Rejected["no longer in the store"] != 1 || summary.Rejected["the source itself"] != 1 ||
		summary.Rejected["the same size and hash, or not larger"] != 1 {
		t.Fatalf("rejections = %v", summary.Rejected)
	}
	if store.putCount() != 0 {
		t.Fatal("find_other_version must write nothing")
	}
	if len(catalog.queries) != 1 || catalog.queries[0] != "sms-20250617122400.xml" {
		t.Fatalf("catalog queried by %v", catalog.queries)
	}
}

func TestFindOtherVersionRequireLargerIgnoresHashOnlyDifferences(t *testing.T) {
	store := newRepairStore(map[string][]byte{
		"salem-data/" + vaultKey:  bytes.Repeat([]byte("x"), 100),
		"salem-data/" + dedupeKey: bytes.Repeat([]byte("y"), 100),
	})
	catalog := &fakeCatalog{
		byKey: map[string]CatalogObject{vaultKey: {Key: vaultKey, Size: 100, SHA1: "aaaa"}},
		rows:  []CatalogObject{{Key: dedupeKey, Size: 100, SHA1: "bbbb"}},
	}
	activity := findActivity(t, store, catalog)
	if _, err := activity.FindOtherVersion(context.Background(), repairplan.StepRequest{
		SourceRef: vaultRef, SourceType: repairplan.TypeSMSBackupXML, Params: json.RawMessage(`{}`),
	}); err != nil {
		t.Fatalf("a different hash qualifies by default: %v", err)
	}
	_, err := activity.FindOtherVersion(context.Background(), repairplan.StepRequest{
		SourceRef: vaultRef, SourceType: repairplan.TypeSMSBackupXML, Params: json.RawMessage(`{"require_larger":true}`),
	})
	requirePermanent(t, err, "no other version of sms-20250617122400.xml was found")
}

// A larger copy can be a zero-filled husk (the 2026-09-13 zero-fill
// quarantine). Every candidate's first 64 KiB is read with one ranged GET —
// never the whole object — and an all-zero head is refused.
// Byline: Claude Code · Opus 5.5 · 2026-09-25
func TestFindOtherVersionRefusesZeroFilledCopiesWithARangedRead(t *testing.T) {
	husk := "consignatio/intake/raw-dedupe/v1/husk/sms-20250617122400.xml"
	good := "consignatio/intake/raw-dedupe/v1/good/sms-20250617122400.xml"
	empty := "consignatio/intake/raw-dedupe/v1/empty/sms-20250617122400.xml"
	lateData := append(make([]byte, 70<<10), []byte("<smses count=\"1\">")...) // zero head, data only past 64 KiB
	store := newRepairStore(map[string][]byte{
		"salem-data/" + vaultKey: bytes.Repeat([]byte("x"), 100),
		"salem-data/" + husk:     make([]byte, 300<<10), // 300 KiB of zeros
		"salem-data/" + good:     append([]byte("<?xml version='1.0'?><smses count=\"5\">"), bytes.Repeat([]byte("s"), 200)...),
		"salem-data/" + empty:    {},
		"salem-data/consignatio/intake/raw-dedupe/v1/late/sms-20250617122400.xml": lateData,
	})
	catalog := &fakeCatalog{rows: []CatalogObject{
		{Key: husk, Size: 300 << 10},
		{Key: "consignatio/intake/raw-dedupe/v1/late/sms-20250617122400.xml", Size: int64(len(lateData))},
		{Key: good, Size: 238},
		{Key: empty, Size: 0, SHA1: "da39a3ee5e6b4b0d3255bfef95601890afd80709"},
	}, byKey: map[string]CatalogObject{vaultKey: {Key: vaultKey, Size: 100, SHA1: "aaaa"}}}

	result, err := findActivity(t, store, catalog).FindOtherVersion(context.Background(), repairplan.StepRequest{
		SourceRef: vaultRef, SourceType: repairplan.TypeSMSBackupXML,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.OutputRef != "b2://salem-data/"+good {
		t.Fatalf("chose %s, want the non-zero copy", result.OutputRef)
	}
	var summary struct {
		Candidates []struct {
			SourceRef        string `json:"source_ref"`
			ZeroCheckedBytes int    `json:"zero_checked_bytes"`
		} `json:"candidates"`
		ZeroFilled     []string       `json:"zero_filled"`
		ZeroCheckBytes int            `json:"zero_check_bytes"`
		Rejected       map[string]int `json:"rejected"`
	}
	if err := json.Unmarshal(result.Summary, &summary); err != nil {
		t.Fatal(err)
	}
	if len(summary.Candidates) != 1 || summary.Candidates[0].ZeroCheckedBytes != 238 || summary.ZeroCheckBytes != 64<<10 {
		t.Fatalf("summary = %s", result.Summary)
	}
	if len(summary.ZeroFilled) != 2 || summary.Rejected[rejectZeroFilled] != 2 || summary.Rejected["empty (no bytes to read)"] != 1 {
		t.Fatalf("zero-filled = %v rejected = %v", summary.ZeroFilled, summary.Rejected)
	}
	for _, read := range store.ranges {
		if read == vaultKey+"@0+4096" {
			continue // the source's own head, read once to tell whether the source is scrambled
		}
		if !strings.HasSuffix(read, "@0+65536") {
			t.Fatalf("head read %q is not one ranged read of the first 64 KiB", read)
		}
	}
	if len(store.opens) != 0 {
		t.Fatalf("candidates were read whole: %v", store.opens)
	}
}

// Every candidate a zero-fill: the step fails and says why.
func TestFindOtherVersionFailsWhenEveryCopyIsZeroFilled(t *testing.T) {
	husk := "consignatio/intake/raw-dedupe/v1/husk/sms-20250617122400.xml"
	store := newRepairStore(map[string][]byte{
		"salem-data/" + vaultKey: bytes.Repeat([]byte("x"), 100),
		"salem-data/" + husk:     make([]byte, 128<<10),
	})
	catalog := &fakeCatalog{rows: []CatalogObject{{Key: husk, Size: 128 << 10}}}
	_, err := findActivity(t, store, catalog).FindOtherVersion(context.Background(), repairplan.StepRequest{
		SourceRef: vaultRef, SourceType: repairplan.TypeSMSBackupXML,
	})
	requirePermanent(t, err, "1 zero-filled (first 64 KiB all zero bytes)")
}

// Copies the owner set aside under _quarantine/ are eligible only on request
// (live catalog 2026-09-25: a vault copy moved to
// consignatio/intake/_quarantine/superseded-sms-backups/...).
func TestFindOtherVersionSkipsQuarantinedCopiesUnlessAsked(t *testing.T) {
	quarantined := "consignatio/intake/_quarantine/superseded-sms-backups/v1/sms-20250617122400.xml"
	store := newRepairStore(map[string][]byte{
		"salem-data/" + vaultKey:    bytes.Repeat([]byte("x"), 100),
		"salem-data/" + quarantined: bytes.Repeat([]byte("y"), 900),
	})
	catalog := &fakeCatalog{rows: []CatalogObject{{Key: quarantined, Size: 900}}}
	activity := findActivity(t, store, catalog)
	_, err := activity.FindOtherVersion(context.Background(), repairplan.StepRequest{SourceRef: vaultRef, SourceType: repairplan.TypeSMSBackupXML})
	requirePermanent(t, err, "set aside under _quarantine/")
	result, err := activity.FindOtherVersion(context.Background(), repairplan.StepRequest{
		SourceRef: vaultRef, SourceType: repairplan.TypeSMSBackupXML, Params: json.RawMessage(`{"include_quarantine":true}`),
	})
	if err != nil || result.OutputRef != "b2://salem-data/"+quarantined {
		t.Fatalf("result = %+v err=%v", result, err)
	}
}

func TestFindOtherVersionFailsClosed(t *testing.T) {
	store := newRepairStore(map[string][]byte{"salem-data/" + vaultKey: []byte("x")})
	_, err := RepairFindOtherVersionActivity{Stores: storesOf(store), Roots: testRoots(t)}.FindOtherVersion(context.Background(),
		repairplan.StepRequest{SourceRef: vaultRef, SourceType: repairplan.TypeSMSBackupXML})
	requirePermanent(t, err, "catalog is not configured")

	activity := findActivity(t, store, &fakeCatalog{})
	_, err = activity.FindOtherVersion(context.Background(), repairplan.StepRequest{SourceRef: "upload://" + strings.Repeat("a", 64), SourceType: "any"})
	requirePermanent(t, err, "not in an object store")
	_, err = activity.FindOtherVersion(context.Background(), repairplan.StepRequest{SourceRef: vaultRef, Params: json.RawMessage(`{"path":"x"}`)})
	requirePermanent(t, err, "unknown field")

	outside := findActivity(t, store, &fakeCatalog{rows: []CatalogObject{{Key: "x/sms-20250617122400.xml", Size: 999}}})
	outside.Roots, _ = objectstores.ParseRoots(`[{"id":"vault","label":"Vault","url":"b2://salem-data/consignatio/vault/v1/"}]`)
	store.objects["salem-data/x/sms-20250617122400.xml"] = bytes.Repeat([]byte("q"), 999)
	_, err = outside.FindOtherVersion(context.Background(), repairplan.StepRequest{SourceRef: vaultRef, SourceType: repairplan.TypeSMSBackupXML})
	requirePermanent(t, err, "outside the configured source roots")
}

// --- repair.salvage_truncated_xml ------------------------------------------

func salvageActivity(t *testing.T, store *repairStore, scratch string) RepairSalvageTruncatedXMLActivity {
	return RepairSalvageTruncatedXMLActivity{Stores: storesOf(store), Roots: testRoots(t), ScratchRoot: scratch}
}

func TestSalvagePublishesAHashedDerivedCopyAndNeverWritesTheOriginal(t *testing.T) {
	full := []byte(smsBackup(bytes.Repeat([]byte("jpeg"), 4000)))
	source := full[:bytes.Index(full, []byte(`data="`))+2000]
	store := newRepairStore(map[string][]byte{"salem-data/" + vaultKey: append([]byte(nil), source...)})
	scratch := t.TempDir()
	var beats int
	activity := salvageActivity(t, store, scratch)
	activity.Heartbeat = func(context.Context, Progress) { beats++ }

	result, err := activity.SalvageTruncatedXML(context.Background(), repairplan.StepRequest{
		SourceRef: vaultRef, SourceType: repairplan.TypeSMSBackupXML, Params: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	derivedKey := vaultKey + ".derived/salvaged/sms-20250617122400.xml"
	if result.OutputRef != "b2://salem-data/"+derivedKey || result.OutputKind != repairplan.OutputDerivedObject || result.Reused {
		t.Fatalf("result = %+v", result)
	}
	derived := store.objects["salem-data/"+derivedKey]
	digest := sha256.Sum256(derived)
	if result.OutputSHA256 != hex.EncodeToString(digest[:]) {
		t.Fatal("the recorded sha256 is not of the published bytes")
	}
	if !bytes.HasSuffix(derived, []byte("\n</smses>\n")) || bytes.Count(derived, []byte("<sms ")) != 2 {
		t.Fatalf("salvaged copy = %q", derived)
	}
	if !bytes.Equal(store.objects["salem-data/"+vaultKey], source) {
		t.Fatal("the original object was modified")
	}
	var manifest SalvageManifest
	if err := json.Unmarshal(store.objects["salem-data/"+derivedKey+".salvage.json"], &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.RecordsKept != 2 || manifest.ClaimedCount != 3 || !manifest.Truncated || manifest.SourceBytes != int64(len(source)) ||
		manifest.DerivedSHA256 != result.OutputSHA256 || manifest.Source != vaultRef {
		t.Fatalf("manifest = %+v", manifest)
	}
	if entries, _ := os.ReadDir(scratch); len(entries) != 0 {
		t.Fatalf("the scratch copy was not removed: %v", entries)
	}
	if store.puts[len(store.puts)-1] != "salem-data/"+derivedKey+".salvage.json" {
		t.Fatal("the manifest must be written last")
	}

	// A retry returns the published salvage without re-streaming or re-putting.
	puts := store.putCount()
	again, err := activity.SalvageTruncatedXML(context.Background(), repairplan.StepRequest{SourceRef: vaultRef, SourceType: repairplan.TypeSMSBackupXML})
	if err != nil || !again.Reused || again.OutputSHA256 != result.OutputSHA256 || store.putCount() != puts {
		t.Fatalf("retry = %+v err=%v puts %d->%d", again, err, puts, store.putCount())
	}

	// A source that changed under a published salvage is refused, never
	// silently re-salvaged over it.
	store.objects["salem-data/"+vaultKey] = append(append([]byte(nil), source...), []byte("more")...)
	_, err = activity.SalvageTruncatedXML(context.Background(), repairplan.StepRequest{SourceRef: vaultRef, SourceType: repairplan.TypeSMSBackupXML})
	requirePermanent(t, err, "changed since its salvage was published")
}

func TestSalvageFailsClosed(t *testing.T) {
	store := newRepairStore(map[string][]byte{"salem-data/" + vaultKey: []byte(`<smses count="3"><sms body="cut`)})
	activity := salvageActivity(t, store, t.TempDir())
	_, err := activity.SalvageTruncatedXML(context.Background(), repairplan.StepRequest{SourceRef: vaultRef, SourceType: repairplan.TypeSMSBackupXML})
	requirePermanent(t, err, "no complete record")

	_, err = activity.SalvageTruncatedXML(context.Background(), repairplan.StepRequest{SourceRef: vaultRef, SourceType: repairplan.TypeDerivedThreads})
	requirePermanent(t, err, "salvage reads XML")

	outside := salvageActivity(t, store, t.TempDir())
	outside.Roots, _ = objectstores.ParseRoots(`[{"id":"vault","label":"Vault","url":"b2://salem-data/consignatio/vault/v1/"}]`)
	roots, err := smsthreads.ParseDerivedRoots(`[{"source":"b2://salem-data/consignatio/vault/v1/","derived":"b2://salem-data/consignatio/derived/"}]`)
	if err != nil {
		t.Fatal(err)
	}
	outside.DerivedRoots = roots
	_, err = outside.SalvageTruncatedXML(context.Background(), repairplan.StepRequest{SourceRef: vaultRef, SourceType: repairplan.TypeSMSBackupXML})
	requirePermanent(t, err, "outside every configured source root")

	_, err = activity.SalvageTruncatedXML(context.Background(), repairplan.StepRequest{SourceRef: "b2://salem-data/missing.xml", SourceType: repairplan.TypeXML})
	requirePermanent(t, err, "does not exist")
	if store.putCount() != 0 {
		t.Fatal("a refused salvage published something")
	}
}

// --- repair.lenient_decode ---------------------------------------------------

func TestLenientDecodePublishesUnderItsOwnVariant(t *testing.T) {
	source := []byte(smsBackup([]byte("png-bytes")))
	store := newRepairStore(map[string][]byte{"salem-data/" + vaultKey: append([]byte(nil), source...)})
	activity := RepairLenientDecodeActivity{Stores: storesOf(store), Roots: testRoots(t), ScratchRoot: t.TempDir()}
	result, err := activity.LenientDecode(context.Background(), repairplan.StepRequest{SourceRef: vaultRef, SourceType: repairplan.TypeSMSBackupXML})
	if err != nil {
		t.Fatal(err)
	}
	prefix := "b2://salem-data/" + vaultKey + ".derived/lenient/"
	if result.OutputRef != prefix+"manifest.json" || result.ReentryRef != prefix+"threads/" ||
		result.OutputKind != repairplan.OutputDerivedChunkFolder || result.OutputType != repairplan.TypeDerivedThreads {
		t.Fatalf("result = %+v", result)
	}
	manifestBytes := store.objects["salem-data/"+vaultKey+".derived/lenient/manifest.json"]
	digest := sha256.Sum256(manifestBytes)
	if result.OutputSHA256 != hex.EncodeToString(digest[:]) {
		t.Fatal("the recorded sha256 is not of the published manifest bytes")
	}
	var summary struct {
		Records     uint64 `json:"records"`
		ChunkCount  int    `json:"chunk_count"`
		ThreadCount int    `json:"thread_count"`
	}
	if err := json.Unmarshal(result.Summary, &summary); err != nil || summary.Records != 3 || summary.ChunkCount == 0 || summary.ThreadCount == 0 {
		t.Fatalf("summary = %s err=%v", result.Summary, err)
	}
	if _, strict := store.objects["salem-data/"+vaultKey+".derived/manifest.json"]; strict {
		t.Fatal("a lenient decode must never occupy the strict derivation's manifest")
	}
	if !bytes.Equal(store.objects["salem-data/"+vaultKey], source) {
		t.Fatal("the original object was modified")
	}
	again, err := activity.LenientDecode(context.Background(), repairplan.StepRequest{SourceRef: vaultRef, SourceType: repairplan.TypeSMSBackupXML})
	if err != nil || !again.Reused || again.OutputSHA256 != result.OutputSHA256 {
		t.Fatalf("retry = %+v err=%v", again, err)
	}
	_, err = activity.LenientDecode(context.Background(), repairplan.StepRequest{SourceRef: vaultRef, SourceType: repairplan.TypeXML})
	requirePermanent(t, err, "reads an SMS backup")
}

// --- repair_record_step_receipt / repair_validate_plan -----------------------

type fakeReceipts struct {
	request repairplan.ReceiptRequest
	attempt int32
}

func (f *fakeReceipts) RecordRepairStepReceipt(_ context.Context, request repairplan.ReceiptRequest, attempt int32) (string, error) {
	f.request, f.attempt = request, attempt
	return "0199bbbb-0000-7000-8000-000000000001", nil
}

func TestRecordStepReceiptValidatesBeforeWriting(t *testing.T) {
	store := &fakeReceipts{}
	activity := RepairStepReceiptActivity{Store: store, Attempt: func(context.Context) int32 { return 2 }}
	good := repairplan.ReceiptRequest{
		WorkflowID: "wf", RunID: "run", StepID: "s1", Activity: string(stagegraph.RepairSalvageTruncatedXML),
		SourceVersionID: "0199aaaa-0000-7000-8000-000000000001", Status: repairplan.ReceiptSuccess,
		Result: &repairplan.StepResult{OutputRef: "b2://x/y"},
	}
	result, err := activity.RecordStepReceipt(context.Background(), good)
	if err != nil || result.ReceiptRef == "" || store.attempt != 2 {
		t.Fatalf("result = %+v err=%v attempt=%d", result, err, store.attempt)
	}
	for name, mutate := range map[string]func(*repairplan.ReceiptRequest){
		"no source version":         func(r *repairplan.ReceiptRequest) { r.SourceVersionID = "" },
		"success without output":    func(r *repairplan.ReceiptRequest) { r.Result = nil },
		"failure without a reason":  func(r *repairplan.ReceiptRequest) { r.Status = repairplan.ReceiptFailed },
		"an unknown receipt status": func(r *repairplan.ReceiptRequest) { r.Status = "maybe" },
	} {
		bad := good
		mutate(&bad)
		if _, err := activity.RecordStepReceipt(context.Background(), bad); err == nil {
			t.Fatalf("%s: accepted", name)
		}
	}
}

// recordingRegistrar is shared with register_test.go.
func TestRepairPlanActivitiesRegisterUnderTheirStageGraphNames(t *testing.T) {
	registrar := &recordingRegistrar{}
	RegisterRepairPlanActivities(registrar, RepairPlanActivities{})
	want := map[string]bool{}
	for _, descriptor := range stagegraph.RepairPlanActivities {
		want[string(descriptor.ID)] = true
	}
	if len(registrar.names) != len(want) {
		t.Fatalf("registered %v", registrar.names)
	}
	for _, name := range registrar.names {
		if !want[name] {
			t.Fatalf("%q is not a registered repair-plan activity", name)
		}
	}
}

func TestValidatePlanActivityFailsClosedWithoutAnAnchorResolver(t *testing.T) {
	if _, err := (RepairPlanValidateActivity{}).ValidatePlan(context.Background(), repairplan.ValidatePlanRequest{}); err == nil {
		t.Fatal("validation without an anchor resolver must fail")
	}
}

// Scrambled-bytes repair (2026-10-02). Byline: Claude Code · Sonnet · 2026-10-02
func scrambledBytes(seed int64, n int) []byte {
	out := make([]byte, n)
	state := uint64(seed)*6364136223846793005 + 1442695040888963407
	for i := range out {
		state = state*6364136223846793005 + 1442695040888963407
		out[i] = byte(state >> 33)
	}
	return out
}

func htmlPage(n int) []byte {
	page := []byte("<html><head><title>Your friends</title></head><body>")
	for len(page) < n {
		page = append(page, []byte("<div class=\"_a6-g\">Matt Salem</div>")...)
	}
	return page[:n]
}

func TestLooksScrambledJudgesTheHeadNotTheName(t *testing.T) {
	random := scrambledBytes(1, 8000)
	if !looksScrambled("your_friends.html", random) {
		t.Fatal("random bytes were not judged scrambled")
	}
	for name, head := range map[string][]byte{
		"html":          htmlPage(8000),
		"jpeg":          append([]byte{0xff, 0xd8, 0xff, 0xe0}, scrambledBytes(2, 8000)...), // compressed data behind its marker
		"zip":           append([]byte("PK\x03\x04"), scrambledBytes(3, 8000)...),
		"mp4":           append([]byte("\x00\x00\x00\x18ftypisom"), scrambledBytes(4, 8000)...),
		"short random":  scrambledBytes(5, 500), // too short to prove anything
		"zeros":         make([]byte, 8000),
		"utf16 text":    append([]byte{0xff, 0xfe}, scrambledBytes(6, 8000)...),
		"plain english": []byte(strings.Repeat("the quick brown fox ", 400)),
	} {
		if looksScrambled("x", head) {
			t.Fatalf("%s was judged scrambled", name)
		}
	}
}

func TestFindOtherVersionPrefersTheSameSizeTwinOfAScrambledSourceAndRefusesScrambledCopies(t *testing.T) {
	const size = 6000
	source := "consignatio/vault/v1/moved/court/fb/facebook-NXPlelIY/connections/friends/your_friends.html"
	twin := "consignatio/vault/v1/social backup/fb/Facebook-2025-08-18/connections/friends/your_friends.html"
	larger := "consignatio/vault/v1/fb/other-export/connections/friends/your_friends.html"
	scrambledToo := "consignatio/vault/v1/moved/court/fb/facebook-local-F-case/connections/friends/your_friends.html"
	store := newRepairStore(map[string][]byte{
		"salem-data/" + source:       scrambledBytes(7, size),
		"salem-data/" + twin:         htmlPage(size),
		"salem-data/" + larger:       htmlPage(size + 900),
		"salem-data/" + scrambledToo: scrambledBytes(8, size),
	})
	catalog := &fakeCatalog{
		byKey: map[string]CatalogObject{source: {Key: source, Size: size, SHA1: "aaaa"}},
		rows: []CatalogObject{
			{Key: larger, Size: size + 900, SHA1: "cccc"},
			{Key: scrambledToo, Size: size, SHA1: "dddd"},
			{Key: twin, Size: size, SHA1: "bbbb"},
		},
	}
	result, err := findActivity(t, store, catalog).FindOtherVersion(context.Background(), repairplan.StepRequest{
		SourceRef: "b2://salem-data/" + source, SourceType: repairplan.TypeAny,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.OutputRef != "b2://salem-data/"+twin {
		t.Fatalf("chose %s, want the same-size intact twin", result.OutputRef)
	}
	var summary struct {
		OriginalScrambled bool           `json:"original_scrambled"`
		Rejected          map[string]int `json:"rejected"`
		Candidates        []struct {
			SourceRef string `json:"source_ref"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(result.Summary, &summary); err != nil {
		t.Fatal(err)
	}
	if !summary.OriginalScrambled || summary.Rejected[rejectScrambled] != 1 || len(summary.Candidates) != 2 ||
		summary.Candidates[1].SourceRef != "b2://salem-data/"+larger {
		t.Fatalf("summary = %s", result.Summary)
	}
}

// An intact source keeps the established rule: the largest verified copy first.
func TestFindOtherVersionKeepsLargestFirstForAnIntactSource(t *testing.T) {
	source := "consignatio/vault/v1/a/your_friends.html"
	twin := "consignatio/vault/v1/b/your_friends.html"
	larger := "consignatio/vault/v1/c/your_friends.html"
	store := newRepairStore(map[string][]byte{
		"salem-data/" + source: htmlPage(5000), "salem-data/" + twin: htmlPage(5000), "salem-data/" + larger: htmlPage(6000),
	})
	catalog := &fakeCatalog{
		byKey: map[string]CatalogObject{source: {Key: source, Size: 5000, SHA1: "aaaa"}},
		rows:  []CatalogObject{{Key: twin, Size: 5000, SHA1: "bbbb"}, {Key: larger, Size: 6000, SHA1: "cccc"}},
	}
	result, err := findActivity(t, store, catalog).FindOtherVersion(context.Background(), repairplan.StepRequest{
		SourceRef: "b2://salem-data/" + source, SourceType: repairplan.TypeAny,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.OutputRef != "b2://salem-data/"+larger {
		t.Fatalf("chose %s, want the larger copy for an intact source", result.OutputRef)
	}
}

// Byline: Claude Code · Opus 5.5 · 2026-09-25

package xmlsalvage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
)

func backup(image []byte) string {
	return `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<!--File Created By SMS Backup & Restore v10.20.002 on 20/09/2026 18:00:00-->
<?xml-stylesheet type="text/xsl" href="sms.xsl"?>
<smses count="4" backup_set="abc" backup_date="1700000000000">
  <sms protocol="0" address="+1 (810) 555-0101" date="1700000000000" type="1" body="a &gt; b, it's 'fine' /> not a close" read="1" status="-1" />
  <sms protocol="0" address="8105550101" date="1700000060000" type="2" body='single "quoted" > still inside' read="1" status="-1" />
  <mms date="1700000180000" msg_box="1" address="8105550101" m_type="132">
    <parts>
      <part seq="0" ct="image/png" name="pic.png" cl="pic.png" data="` + base64.StdEncoding.EncodeToString(image) + `" />
      <part seq="1" ct="text/plain" name="null" text="see &lt;picture&gt;" />
    </parts>
    <addrs>
      <addr address="8105550101" type="137" charset="106" />
    </addrs>
  </mms>
  <sms protocol="0" address="8105550202" date="1700000120000" type="1" body="last one" read="1" status="-1" />
</smses>
`
}

func scanAll(t *testing.T, data []byte, chunk func(int) int) Result {
	t.Helper()
	s := &Scanner{}
	for i := 0; i < len(data); {
		n := chunk(len(data) - i)
		if n > len(data)-i {
			n = len(data) - i
		}
		written, err := s.Write(data[i : i+n])
		if err != nil || written != n {
			t.Fatalf("write = %d, %v", written, err)
		}
		i += n
	}
	return s.Result()
}

// Chunk boundaries must never change what the scanner concludes: the same
// bytes one at a time, in random pieces, or all at once give one answer.
func TestScannerIsIndependentOfWriteBoundaries(t *testing.T) {
	data := []byte(backup(bytes.Repeat([]byte("png-bytes"), 400)))
	whole := scanAll(t, data, func(rest int) int { return rest })
	one := scanAll(t, data, func(int) int { return 1 })
	rng := rand.New(rand.NewSource(7))
	random := scanAll(t, data, func(int) int { return 1 + rng.Intn(97) })
	if whole != one || whole != random {
		t.Fatalf("boundary-dependent result:\nwhole  %+v\none    %+v\nrandom %+v", whole, one, random)
	}
	if whole.Root != "smses" || !whole.RootClosed || whole.Records != 4 || whole.Stopped != "" {
		t.Fatalf("complete backup scanned as %+v", whole)
	}
	lastRecordEnd := int64(bytes.LastIndex(data, []byte(`body="last one" read="1" status="-1" />`)) + len(`body="last one" read="1" status="-1" />`))
	if whole.LastCompleteEnd != lastRecordEnd {
		t.Fatalf("last complete end = %d, want %d", whole.LastCompleteEnd, lastRecordEnd)
	}
	if whole.Truncated() {
		t.Fatal("a closed document is not truncated")
	}
}

func TestScannerFindsTheLastCompleteRecordOfATruncatedBackup(t *testing.T) {
	data := []byte(backup(bytes.Repeat([]byte("jpeg"), 10_000)))
	mmsStart := bytes.Index(data, []byte("<mms "))
	secondSMSEnd := int64(bytes.Index(data, []byte(`still inside' read="1" status="-1" />`)) + len(`still inside' read="1" status="-1" />`))
	for name, cut := range map[string]int{
		"inside the base64 part":     bytes.Index(data, []byte(`data="`)) + 5000,
		"inside the mms start tag":   mmsStart + 20,
		"between parts and addrs":    bytes.Index(data, []byte("<addrs>")),
		"inside the closing mms tag": bytes.Index(data, []byte("</mms>")) + 3,
	} {
		result := scanAll(t, data[:cut], func(rest int) int { return rest })
		if !result.Truncated() || result.Records != 2 || result.LastCompleteEnd != secondSMSEnd {
			t.Fatalf("%s: %+v (want 2 records ending at %d)", name, result, secondSMSEnd)
		}
	}
}

func TestScannerHandlesCommentsCDATAAndDoctype(t *testing.T) {
	doc := `<?xml version="1.0"?>
<!DOCTYPE root [ <!ENTITY e "x>y"> <!-- > --> ]>
<root>
  <!-- <fake/> </root> -->
  <item><![CDATA[ </item> <root> ]]></item>
  <item a="1"/>
  <?pi </root> ?>
</root>`
	result := scanAll(t, []byte(doc), func(int) int { return 3 })
	if result.Root != "root" || !result.RootClosed || result.Records != 2 || result.Stopped != "" {
		t.Fatalf("scanned as %+v", result)
	}
}

// A malformation stops the count before it, so a salvage never keeps bytes
// the scanner could not account for.
func TestScannerStopsAtAMalformedStartTag(t *testing.T) {
	doc := `<smses count="3"><sms body="ok"/><sms body="ok2"/><sms body="broken <sms body="x"/></smses>`
	result := scanAll(t, []byte(doc), func(rest int) int { return rest })
	if result.Records != 2 || result.Stopped == "" {
		t.Fatalf("scanned as %+v", result)
	}
	want := int64(strings.Index(doc, `<sms body="ok2"/>`) + len(`<sms body="ok2"/>`))
	if result.LastCompleteEnd != want {
		t.Fatalf("last complete end = %d, want %d", result.LastCompleteEnd, want)
	}
}

func salvageBytes(t *testing.T, source []byte) (Outcome, []byte, error) {
	t.Helper()
	out, err := os.Create(filepath.Join(t.TempDir(), "salvaged.xml"))
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	outcome, salvageErr := Salvage(context.Background(), bytes.NewReader(source), out, nil)
	if salvageErr != nil {
		return outcome, nil, salvageErr
	}
	derived, err := io.ReadAll(out)
	if err != nil {
		t.Fatal(err)
	}
	return outcome, derived, nil
}

func TestSalvageKeepsAnExactPrefixAndClosesTheDocument(t *testing.T) {
	image := bytes.Repeat([]byte("png!"), 5000)
	full := []byte(backup(image))
	cut := bytes.Index(full, []byte(`data="`)) + 1234
	source := full[:cut]

	outcome, derived, err := salvageBytes(t, source)
	if err != nil {
		t.Fatal(err)
	}
	if !outcome.Truncated() || outcome.Records != 2 || outcome.ClaimedCount != 4 {
		t.Fatalf("outcome = %+v", outcome)
	}
	if !bytes.HasPrefix(derived, source[:outcome.LastCompleteEnd]) || !bytes.HasSuffix(derived, []byte("\n</smses>\n")) {
		t.Fatal("the salvaged copy is not an exact prefix of the source plus the closing tag")
	}
	if int64(len(derived)) != outcome.DerivedBytes || outcome.BytesDropped != int64(len(source))-outcome.LastCompleteEnd {
		t.Fatalf("byte accounting is wrong: %+v (derived %d)", outcome, len(derived))
	}
	sourceDigest, derivedDigest := sha256.Sum256(source), sha256.Sum256(derived)
	if outcome.SourceSHA256 != hex.EncodeToString(sourceDigest[:]) || outcome.DerivedSHA256 != hex.EncodeToString(derivedDigest[:]) {
		t.Fatal("digests do not cover the exact source and derived bytes")
	}
	// The salvaged copy is well-formed XML with exactly the kept records.
	decoder := xml.NewDecoder(bytes.NewReader(derived))
	depth, records := 0, 0
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("salvaged copy is not well-formed XML: %v", err)
		}
		switch token.(type) {
		case xml.StartElement:
			depth++
		case xml.EndElement:
			depth--
			if depth == 1 {
				records++
			}
		}
	}
	if records != 2 || depth != 0 {
		t.Fatalf("salvaged copy has %d records, depth %d", records, depth)
	}
}

// The salvaged copy decodes cleanly through the SBV derive route: every kept
// record parses and nothing is rejected, because the cut-off tail is gone.
func TestSalvagedCopyDecodesThroughTheSBVDeriveRoute(t *testing.T) {
	full := []byte(backup(bytes.Repeat([]byte("jpeg"), 20_000)))
	source := full[:bytes.Index(full, []byte(`data="`))+60_000]
	_, derived, err := salvageBytes(t, source)
	if err != nil {
		t.Fatal(err)
	}
	store := &memoryStore{objects: map[string][]byte{"bkt/v/sms-salvaged.xml": derived}}
	manifest, _, err := smsthreads.Derive(context.Background(), smsthreads.Options{
		Store: store, Scheme: "b2", Bucket: "bkt", Key: "v/sms-salvaged.xml", ScratchRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("the salvaged copy must decode: %v", err)
	}
	if manifest.Records != 2 || manifest.Rejected != 0 {
		t.Fatalf("records=%d rejected=%d", manifest.Records, manifest.Rejected)
	}
}

func TestSalvageRefusesASourceWithNothingComplete(t *testing.T) {
	for name, source := range map[string]string{
		"empty":             "",
		"no root":           `<?xml version="1.0"?><!-- nothing -->`,
		"cut in first item": `<smses count="9"><sms body="cut`,
	} {
		if _, _, err := salvageBytes(t, []byte(source)); !errors.Is(err, ErrNothingToSalvage) {
			t.Fatalf("%s: err = %v, want ErrNothingToSalvage", name, err)
		}
	}
}

func TestSalvageOfACompleteDocumentKeepsEveryRecord(t *testing.T) {
	source := []byte(backup([]byte("img")))
	outcome, derived, err := salvageBytes(t, source)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Truncated() || outcome.Records != 4 {
		t.Fatalf("outcome = %+v", outcome)
	}
	if !bytes.HasPrefix(derived, source[:outcome.LastCompleteEnd]) {
		t.Fatal("complete-document salvage is not a prefix copy")
	}
}

func TestSalvageHonoursCancellation(t *testing.T) {
	out, err := os.Create(filepath.Join(t.TempDir(), "x.xml"))
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Salvage(ctx, strings.NewReader(backup(nil)), out, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

type memoryStore struct{ objects map[string][]byte }

func (m *memoryStore) Open(_ context.Context, bucket, key string) (io.ReadCloser, error) {
	data, ok := m.objects[bucket+"/"+key]
	if !ok {
		return nil, errors.New("no such object: " + key)
	}
	return io.NopCloser(bytes.NewReader(data)), nil
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

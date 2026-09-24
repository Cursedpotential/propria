package runtimeapi

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/acquisition"
	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/proffer"
)

func testZIP(t *testing.T, members ...struct{ name, body string }) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, member := range members {
		entry, err := writer.Create(member.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(entry, member.body); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func zipFixture(t *testing.T, content []byte, format string) (activities.MemberEnumerator, activities.SourceObservationInput) {
	t.Helper()
	filename := filepath.Join(t.TempDir(), "original.zip")
	if err := os.WriteFile(filename, content, 0o600); err != nil {
		t.Fatal(err)
	}
	db := validObservationTestDB()
	db.storageClass = "filesystem"
	db.objectURI = fileURI(filename)
	db.byteLength = int64(len(content))
	digest := sha256.Sum256(content)
	db.contentSHA256 = digest[:]
	enumerator, err := NewZIPMemberEnumerator(db)
	if err != nil {
		t.Fatal(err)
	}
	input := validObservationInput()
	input.DeclaredFormat = format
	return enumerator, input
}

func TestZIPMemberInventoryPreservesOrderAndProvenance(t *testing.T) {
	content := testZIP(t,
		struct{ name, body string }{"a/first.txt", "first"},
		struct{ name, body string }{"b/second.txt", "second"},
	)
	enumerator, input := zipFixture(t, content, "archive")
	stream, err := enumerator.EnumerateMembers(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	for ordinal, expected := range []struct{ name, body string }{{"a/first.txt", "first"}, {"b/second.txt", "second"}} {
		member, err := stream.Next(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256([]byte(expected.body))
		if member.Ordinal != int64(ordinal) || member.Name != expected.name || member.ByteLength != int64(len(expected.body)) ||
			member.SHA256 != hex.EncodeToString(digest[:]) || member.ParentRef != input.OriginalRef ||
			member.SourceByteOffset == nil || member.CompressedByteLength <= 0 || member.ByteOffset != nil {
			t.Fatalf("member %d lost source provenance: %+v", ordinal, member)
		}
		if !strings.HasPrefix(string(member.MemberRef), "zip-member:"+string(input.SourceVersionRef)+":") {
			t.Fatalf("member %d has unstable source identity: %q", ordinal, member.MemberRef)
		}
	}
	if _, err := stream.Next(context.Background()); !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF, got %v", err)
	}
}

func TestZIPMemberInventoryAcceptsUserUploadRetainedObject(t *testing.T) {
	// Exercise the upload:// ingress and resolver that the production worker
	// registers, then pass its exact retained metadata to the same enumerator.
	content := testZIP(t, struct{ name, body string }{"one.txt", "uploaded"})
	root := t.TempDir()
	ingress, err := acquisition.NewUploadIngress(acquisition.UploadIngressConfig{Root: root, MaxBytes: int64(len(content))})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader(content))
	request.RemoteAddr = "100.64.1.2:12345"
	response := httptest.NewRecorder()
	ingress.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("upload failed: status=%d body=%s", response.Code, response.Body.String())
	}
	var accepted struct {
		AcquisitionRef string `json:"acquisition_ref"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &accepted); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(accepted.AcquisitionRef, "upload://") {
		t.Fatalf("unexpected acquisition ref %q", accepted.AcquisitionRef)
	}
	resolver, err := acquisition.NewUploadIngressResolver(root)
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := resolver(context.Background(), proffer.Ref(accepted.AcquisitionRef))
	if err != nil {
		t.Fatal(err)
	}
	if sealed.StorageClass != "immutable_object_store" || !strings.HasPrefix(sealed.ObjectURI, "file://") {
		t.Fatalf("unexpected retained acquisition provenance: class=%q URI=%q", sealed.StorageClass, sealed.ObjectURI)
	}
	db := validObservationTestDB()
	db.storageClass, db.objectURI, db.byteLength, db.contentSHA256 =
		sealed.StorageClass, sealed.ObjectURI, sealed.ByteLength, sealed.ContentSHA256
	enumerator, err := NewZIPMemberEnumerator(db)
	if err != nil {
		t.Fatal(err)
	}
	input := validObservationInput()
	input.DeclaredFormat = "zip"
	stream, err := enumerator.EnumerateMembers(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	member, err := stream.Next(context.Background())
	if err != nil || member.Name != "one.txt" || member.ParentRef != input.OriginalRef {
		t.Fatalf("uploaded ZIP lost source membership: member=%+v err=%v", member, err)
	}
}

func TestZIPMemberInventoryRejectsOtherStorageAndNonFileURI(t *testing.T) {
	content := testZIP(t, struct{ name, body string }{"one.txt", "body"})
	for _, tc := range []struct{ name, class, uri, want string }{
		{"remote class", "r2", "file:///tmp/ignored.zip", "unsupported retained storage class"},
		{"sealed remote URI", "immutable_object_store", "s3://bucket/object", "file URI"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			filename := filepath.Join(t.TempDir(), "original.zip")
			if err := os.WriteFile(filename, content, 0o600); err != nil {
				t.Fatal(err)
			}
			db := validObservationTestDB()
			db.storageClass, db.objectURI, db.byteLength = tc.class, tc.uri, int64(len(content))
			digest := sha256.Sum256(content)
			db.contentSHA256 = digest[:]
			enumerator, err := NewZIPMemberEnumerator(db)
			if err != nil {
				t.Fatal(err)
			}
			input := validObservationInput()
			input.DeclaredFormat = "zip"
			if stream, err := enumerator.EnumerateMembers(context.Background(), input); stream != nil || err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("unsupported retained object accepted: stream=%v err=%v", stream, err)
			}
		})
	}
}

func TestZIPMemberInventoryRejectsMalformedAndUnsupportedInputs(t *testing.T) {
	for _, tc := range []struct {
		name, format string
		data         []byte
		want         string
	}{
		{"truncated", "archive", testZIP(t, struct{ name, body string }{"one", "hello"})[:8], "invalid or unsupported ZIP"},
		{"unsupported archive", "archive", []byte("not a ZIP"), "invalid or unsupported ZIP"},
		{"traversal", "zip", testZIP(t, struct{ name, body string }{"../escape", "x"}), "invalid ZIP member name"},
		{"absolute", "zip", testZIP(t, struct{ name, body string }{"/escape", "x"}), "invalid ZIP member name"},
		{"duplicate", "zip", testZIP(t, struct{ name, body string }{"same", "a"}, struct{ name, body string }{"same", "b"}), "duplicate normalized ZIP member"},
		{"normalized duplicate", "zip", testZIP(t, struct{ name, body string }{"same", "a"}, struct{ name, body string }{"same/", ""}), "duplicate normalized ZIP member"},
		{"drive path", "zip", testZIP(t, struct{ name, body string }{"C:/escape", "x"}), "invalid ZIP member name"},
		{"UNC path", "zip", testZIP(t, struct{ name, body string }{"\\\\server\\share", "x"}), "invalid ZIP member name"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			enumerator, input := zipFixture(t, tc.data, tc.format)
			stream, err := enumerator.EnumerateMembers(context.Background(), input)
			if stream != nil || err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("stream=%v err=%v; want %q", stream, err, tc.want)
			}
		})
	}
}

func TestZIPCentralDirectoryAdmissionCountsPhysicalEntriesBeforeStdlib(t *testing.T) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for i := 0; i <= zipMaxMembers; i++ {
		if _, err := writer.Create(fmt.Sprintf("member-%05d", i)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	content := append([]byte(nil), buffer.Bytes()...)
	end := bytes.LastIndex(content, []byte{'P', 'K', 5, 6})
	if end < 0 {
		t.Fatal("test ZIP has no end record")
	}
	// The claimed count is small, but the physical directory has 10,001
	// entries. archive/zip otherwise parses and allocates them before the
	// caller can inspect len(archive.File).
	binary.LittleEndian.PutUint16(content[end+8:], 1)
	binary.LittleEndian.PutUint16(content[end+10:], 1)
	parserCalled := false
	archive, err := openAdmittedZIP(context.Background(), bytes.NewReader(content), int64(len(content)),
		func(io.ReaderAt, int64) (*zip.Reader, error) {
			parserCalled = true
			return nil, errors.New("stdlib parser must not run")
		})
	if archive != nil || err == nil || !strings.Contains(err.Error(), "member limit") || parserCalled {
		t.Fatalf("oversized physical directory reached parser: archive=%v err=%v called=%t", archive, err, parserCalled)
	}
	enumerator, input := zipFixture(t, content, "zip")
	if stream, err := enumerator.EnumerateMembers(context.Background(), input); stream != nil || err == nil || !strings.Contains(err.Error(), "member limit") {
		t.Fatalf("inventory did not reject oversized directory: stream=%v err=%v", stream, err)
	}
}

func TestZIPCentralDirectoryAdmissionRejectsForgedCountsAndExtents(t *testing.T) {
	original := testZIP(t, struct{ name, body string }{"one.txt", "body"})
	end := bytes.LastIndex(original, []byte{'P', 'K', 5, 6})
	central := bytes.Index(original, []byte{'P', 'K', 1, 2})
	if end < 0 || central < 0 {
		t.Fatal("test ZIP lacks directory records")
	}
	for _, tc := range []struct {
		name, want string
		change     func([]byte)
	}{
		{"forged high count", "member limit", func(content []byte) {
			binary.LittleEndian.PutUint16(content[end+8:], zipMaxMembers+1)
			binary.LittleEndian.PutUint16(content[end+10:], zipMaxMembers+1)
		}},
		{"forged low count", "count mismatch", func(content []byte) {
			binary.LittleEndian.PutUint16(content[end+8:], 0)
			binary.LittleEndian.PutUint16(content[end+10:], 0)
		}},
		{"truncated entry", "truncated ZIP central directory entry", func(content []byte) {
			binary.LittleEndian.PutUint16(content[central+28:], 0xffff)
		}},
		{"false directory extent", "extent", func(content []byte) {
			binary.LittleEndian.PutUint32(content[end+12:], 1)
		}},
		{"oversized directory bytes", "byte limit", func(content []byte) {
			binary.LittleEndian.PutUint32(content[end+12:], uint32(zipMaxDirectoryBytes+1))
		}},
		{"entry disk start", "multi-disk", func(content []byte) {
			binary.LittleEndian.PutUint16(content[central+34:], 1)
		}},
		{"entry compressed ZIP64 sentinel", "ZIP64", func(content []byte) {
			binary.LittleEndian.PutUint32(content[central+20:], 0xffffffff)
		}},
		{"entry uncompressed ZIP64 sentinel", "ZIP64", func(content []byte) {
			binary.LittleEndian.PutUint32(content[central+24:], 0xffffffff)
		}},
		{"entry offset ZIP64 sentinel", "ZIP64", func(content []byte) {
			binary.LittleEndian.PutUint32(content[central+42:], 0xffffffff)
		}},
		{"EOCD disk number", "multi-disk", func(content []byte) {
			binary.LittleEndian.PutUint16(content[end+4:], 1)
		}},
		{"EOCD ZIP64 count", "ZIP64", func(content []byte) {
			binary.LittleEndian.PutUint16(content[end+10:], 0xffff)
		}},
		{"EOCD ZIP64 extent", "ZIP64", func(content []byte) {
			binary.LittleEndian.PutUint32(content[end+12:], 0xffffffff)
		}},
		{"EOCD ZIP64 offset", "ZIP64", func(content []byte) {
			binary.LittleEndian.PutUint32(content[end+16:], 0xffffffff)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			content := append([]byte(nil), original...)
			tc.change(content)
			parserCalled := false
			_, err := openAdmittedZIP(context.Background(), bytes.NewReader(content), int64(len(content)),
				func(io.ReaderAt, int64) (*zip.Reader, error) {
					parserCalled = true
					return nil, nil
				})
			if err == nil || !strings.Contains(err.Error(), tc.want) || parserCalled {
				t.Fatalf("malformed directory reached parser: err=%v called=%t", err, parserCalled)
			}
		})
	}
}

func TestZIPCentralDirectoryAdmissionRejectsZIP64ExtraBeforeStdlib(t *testing.T) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	header := &zip.FileHeader{Name: "one.txt", Method: zip.Store, Extra: []byte{1, 0, 0, 0}}
	entry, err := writer.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(entry, "body"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	parserCalled := false
	_, err = openAdmittedZIP(context.Background(), bytes.NewReader(buffer.Bytes()), int64(buffer.Len()),
		func(io.ReaderAt, int64) (*zip.Reader, error) {
			parserCalled = true
			return nil, nil
		})
	if err == nil || !strings.Contains(err.Error(), "ZIP64") || parserCalled {
		t.Fatalf("ZIP64 extra reached stdlib parser: err=%v called=%t", err, parserCalled)
	}
	malformed := append([]byte(nil), buffer.Bytes()...)
	central := bytes.Index(malformed, []byte{'P', 'K', 1, 2})
	if central < 0 {
		t.Fatal("test ZIP has no central directory")
	}
	extraAt := central + 46 + int(binary.LittleEndian.Uint16(malformed[central+28:]))
	binary.LittleEndian.PutUint16(malformed[extraAt:], 0xcafe)
	binary.LittleEndian.PutUint16(malformed[extraAt+2:], 5)
	parserCalled = false
	_, err = openAdmittedZIP(context.Background(), bytes.NewReader(malformed), int64(len(malformed)),
		func(io.ReaderAt, int64) (*zip.Reader, error) {
			parserCalled = true
			return nil, nil
		})
	if err == nil || !strings.Contains(err.Error(), "extra field extent") || parserCalled {
		t.Fatalf("malformed extra extent reached stdlib parser: err=%v called=%t", err, parserCalled)
	}
}

func TestZIPCentralDirectoryAdmissionMaxCommentAndTrailingBytes(t *testing.T) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	if err := writer.SetComment(strings.Repeat("c", 65535)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	content := buffer.Bytes()
	parserCalled := false
	_, err := openAdmittedZIP(context.Background(), bytes.NewReader(content), int64(len(content)),
		func(source io.ReaderAt, size int64) (*zip.Reader, error) {
			parserCalled = true
			return zip.NewReader(source, size)
		})
	if err != nil || !parserCalled {
		t.Fatalf("valid maximum comment rejected: err=%v called=%t", err, parserCalled)
	}
	trailing := append(append([]byte(nil), content...), 'x')
	parserCalled = false
	_, err = openAdmittedZIP(context.Background(), bytes.NewReader(trailing), int64(len(trailing)),
		func(io.ReaderAt, int64) (*zip.Reader, error) {
			parserCalled = true
			return nil, nil
		})
	if err == nil || parserCalled {
		t.Fatalf("trailing bytes reached parser: err=%v called=%t", err, parserCalled)
	}
}

func TestZIPMemberInventoryRejectsChangedSourceAtSameLength(t *testing.T) {
	content := testZIP(t, struct{ name, body string }{"one.txt", "original"})
	filename := filepath.Join(t.TempDir(), "original.zip")
	changed := append([]byte(nil), content...)
	changed[len(changed)-1] ^= 1
	if err := os.WriteFile(filename, changed, 0o600); err != nil {
		t.Fatal(err)
	}
	db := validObservationTestDB()
	db.storageClass, db.objectURI, db.byteLength = "filesystem", fileURI(filename), int64(len(content))
	digest := sha256.Sum256(content)
	db.contentSHA256 = digest[:]
	enumerator, err := NewZIPMemberEnumerator(db)
	if err != nil {
		t.Fatal(err)
	}
	input := validObservationInput()
	input.DeclaredFormat = "zip"
	if stream, err := enumerator.EnumerateMembers(context.Background(), input); stream != nil || err == nil || !strings.Contains(err.Error(), "content SHA-256 changed") {
		t.Fatalf("same-length source replacement must fail, stream=%v err=%v", stream, err)
	}
}

func TestZIPMemberInventoryRejectsSpecialModeAndEncryption(t *testing.T) {
	for _, tc := range []struct {
		name  string
		mode  os.FileMode
		flags uint16
		want  string
	}{
		{"symlink", os.ModeSymlink | 0o777, 0, "file mode"},
		{"device", os.ModeDevice | 0o600, 0, "file mode"},
		{"encrypted", 0o600, 1, "encrypted"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buffer bytes.Buffer
			writer := zip.NewWriter(&buffer)
			header := &zip.FileHeader{Name: "member", Method: zip.Store}
			header.SetMode(tc.mode)
			entry, err := writer.CreateHeader(header)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := io.WriteString(entry, "body"); err != nil {
				t.Fatal(err)
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			content := buffer.Bytes()
			if tc.flags != 0 {
				central := bytes.Index(content, []byte{'P', 'K', 1, 2})
				if central < 0 {
					t.Fatal("test ZIP has no central directory")
				}
				binary.LittleEndian.PutUint16(content[central+8:], tc.flags)
			}
			enumerator, input := zipFixture(t, content, "zip")
			if stream, err := enumerator.EnumerateMembers(context.Background(), input); stream != nil || err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("unsupported ZIP member accepted, stream=%v err=%v", stream, err)
			}
		})
	}
}

func TestZIPMemberInventoryCancellationClosesStream(t *testing.T) {
	content := testZIP(t, struct{ name, body string }{"one.txt", "body"})
	enumerator, input := zipFixture(t, content, "zip")
	stream, err := enumerator.EnumerateMembers(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := stream.Next(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	if err := stream.Close(); err != nil {
		t.Fatal(err)
	}
	if err := stream.Close(); err != nil {
		t.Fatalf("close must be idempotent: %v", err)
	}
}

func TestZIPMemberInventoryUsesPhysicalOrderWhenCentralDirectoryIsReversed(t *testing.T) {
	content := testZIP(t,
		struct{ name, body string }{"first.txt", "first"},
		struct{ name, body string }{"second.txt", "second"},
	)
	first := bytes.Index(content, []byte{'P', 'K', 1, 2})
	if first < 0 {
		t.Fatal("test ZIP has no central directory")
	}
	second := bytes.Index(content[first+4:], []byte{'P', 'K', 1, 2})
	if second < 0 {
		t.Fatal("test ZIP has no second central entry")
	}
	second += first + 4
	end := bytes.Index(content[second+4:], []byte{'P', 'K', 5, 6})
	if end < 0 {
		t.Fatal("test ZIP has no end record")
	}
	end += second + 4
	reversed := append([]byte(nil), content[:first]...)
	reversed = append(reversed, content[second:end]...)
	reversed = append(reversed, content[first:second]...)
	reversed = append(reversed, content[end:]...)
	enumerator, input := zipFixture(t, reversed, "zip")
	stream, err := enumerator.EnumerateMembers(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	for _, want := range []string{"first.txt", "second.txt"} {
		member, err := stream.Next(context.Background())
		if err != nil || member.Name != want {
			t.Fatalf("physical source order lost: member=%+v err=%v want=%s", member, err, want)
		}
	}
}

func TestZIPMemberInventoryRejectsCRCMismatch(t *testing.T) {
	content := testZIP(t, struct{ name, body string }{"one.txt", "payload"})
	central := bytes.Index(content, []byte{'P', 'K', 1, 2})
	if central < 0 {
		t.Fatal("test ZIP has no central directory")
	}
	content[central+16] ^= 1
	enumerator, input := zipFixture(t, content, "zip")
	stream, err := enumerator.EnumerateMembers(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	if _, err := stream.Next(context.Background()); !errors.Is(err, zip.ErrChecksum) {
		t.Fatalf("CRC mismatch must fail before member is inventoried: %v", err)
	}
}

func TestZIPMemberInventoryDistinguishesOOXMLFromGenericZIP(t *testing.T) {
	plain := testZIP(t, struct{ name, body string }{"ordinary.txt", "body"})
	enumerator, input := zipFixture(t, plain, "xlsx")
	if stream, err := enumerator.EnumerateMembers(context.Background(), input); stream != nil || err == nil || !strings.Contains(err.Error(), "[Content_Types].xml") {
		t.Fatalf("ordinary ZIP mislabeled XLSX must fail, stream=%v err=%v", stream, err)
	}
	for _, tc := range []struct {
		format, mainPart string
	}{
		{"xlsx", "xl/workbook.xml"},
		{"docx", "word/document.xml"},
		{"pptx", "ppt/presentation.xml"},
		{"ooxml", "word/document.xml"},
	} {
		t.Run(tc.format, func(t *testing.T) {
			content := testZIP(t,
				struct{ name, body string }{"[Content_Types].xml", "<Types/>"},
				struct{ name, body string }{"_rels/.rels", "<Relationships/>"},
				struct{ name, body string }{tc.mainPart, "<part/>"},
			)
			enumerator, input := zipFixture(t, content, tc.format)
			stream, err := enumerator.EnumerateMembers(context.Background(), input)
			if err != nil {
				t.Fatal(err)
			}
			defer stream.Close()
			if _, err := stream.Next(context.Background()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestZIPMemberInventoryLimitsAndRetainedIdentity(t *testing.T) {
	content := testZIP(t, struct{ name, body string }{"huge.txt", strings.Repeat("a", 1<<20)})
	enumerator, input := zipFixture(t, content, "xlsx")
	if stream, err := enumerator.EnumerateMembers(context.Background(), input); stream != nil || err == nil || !strings.Contains(err.Error(), "expansion ratio") {
		t.Fatalf("expected compression bomb limit, stream=%v err=%v", stream, err)
	}
	db := validObservationTestDB()
	db.storageClass = "filesystem"
	db.byteLength = zipMaxArchiveBytes + 1
	bounded, err := NewZIPMemberEnumerator(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bounded.EnumerateMembers(context.Background(), input); err == nil || !strings.Contains(err.Error(), "archive byte limit") {
		t.Fatalf("expected archive byte limit, got %v", err)
	}
	db = validObservationTestDB()
	db.resolutionErr = errors.New("source membership denied")
	denied, err := NewZIPMemberEnumerator(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := denied.EnumerateMembers(context.Background(), input); err == nil || !strings.Contains(err.Error(), "source membership denied") {
		t.Fatalf("expected membership failure, got %v", err)
	}
}

func TestZIPMemberInventoryRejectsForgedLengthAndCorruptPayload(t *testing.T) {
	content := append([]byte(nil), testZIP(t, struct{ name, body string }{"one.txt", "payload"})...)
	central := bytes.Index(content, []byte{'P', 'K', 1, 2})
	if central < 0 {
		t.Fatal("test ZIP has no central directory")
	}
	tooLarge := append([]byte(nil), content...)
	binary.LittleEndian.PutUint32(tooLarge[central+24:], uint32(zipMaxMemberBytes+1))
	enumerator, input := zipFixture(t, tooLarge, "zip")
	if stream, err := enumerator.EnumerateMembers(context.Background(), input); stream != nil || err == nil || !strings.Contains(err.Error(), "expanded byte limit") {
		t.Fatalf("expected declared member size limit, stream=%v err=%v", stream, err)
	}
	corrupt := append([]byte(nil), content...)
	archive, err := zip.NewReader(bytes.NewReader(corrupt), int64(len(corrupt)))
	if err != nil || len(archive.File) != 1 {
		t.Fatalf("read test ZIP: %v", err)
	}
	data, err := archive.File[0].DataOffset()
	if err != nil {
		t.Fatal(err)
	}
	corrupt[data] ^= 0xff
	enumerator, input = zipFixture(t, corrupt, "zip")
	stream, err := enumerator.EnumerateMembers(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	if _, err := stream.Next(context.Background()); err == nil {
		t.Fatal("corrupt member must fail before it can be inventoried")
	}
}

func TestZIPMemberInventoryNonContainerIsNotApplicable(t *testing.T) {
	enumerator, input := zipFixture(t, []byte("plain text"), "markdown")
	stream, err := enumerator.EnumerateMembers(context.Background(), input)
	if stream != nil || !errors.Is(err, activities.ErrNotApplicable) {
		t.Fatalf("stream=%v err=%v", stream, err)
	}
}

func TestZIPMemberInventoryDoesNotHydrateNestedArchive(t *testing.T) {
	nested := testZIP(t, struct{ name, body string }{"inside.txt", "inner data"})
	outer := testZIP(t, struct{ name, body string }{"nested.zip", string(nested)})
	enumerator, input := zipFixture(t, outer, "zip")
	stream, err := enumerator.EnumerateMembers(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	member, err := stream.Next(context.Background())
	if err != nil || member.Name != "nested.zip" || member.ByteLength != int64(len(nested)) {
		t.Fatalf("nested archive must be one opaque member: %+v, %v", member, err)
	}
	if _, err := stream.Next(context.Background()); !errors.Is(err, io.EOF) {
		t.Fatalf("nested content was processed unexpectedly: %v", err)
	}
}

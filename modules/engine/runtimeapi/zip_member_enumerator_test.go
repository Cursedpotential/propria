package runtimeapi

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/activities"
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
		{"duplicate", "zip", testZIP(t, struct{ name, body string }{"same", "a"}, struct{ name, body string }{"same", "b"}), "duplicate ZIP member"},
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

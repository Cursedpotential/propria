// Byline: Claude Code · Fable 5.1 · 2026-09-21
package postgres

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

// The structured-text route stores attachments in native_fields with the
// derive/smsthreads.LineAttachment shape. This is the exact JSON read live from
// context.raw_ndjson on 2026-09-21; before the fix no such attachment was ever
// projected into a preview.
func TestStructuredAttachmentProjectsIntoPreview(t *testing.T) {
	const stored = `[{"uri":"b2://bucket/root/sms /file.xml.derived/media/ceb4cae5503deb19bcba31666c14db6ab51c32c51f118f0ff3947520392afb08.png",
		"mime":"image/png","name":"d2665f1c.png","bytes":659746,
		"sha256":"ceb4cae5503deb19bcba31666c14db6ab51c32c51f118f0ff3947520392afb08","ordinal":0}]`
	var attachments []previewStructuredAttachment
	if err := json.Unmarshal([]byte(stored), &attachments); err != nil {
		t.Fatal(err)
	}
	rawID := uuid.MustParse("01a0c181-1e7e-7a3c-a0c6-b982c2a6bf0e")
	got := attachments[0].project(rawID)
	if got.AttachmentID != rawID.String()+":0" {
		t.Fatalf("attachment id = %q", got.AttachmentID)
	}
	if got.SourceLocatorRef != "context.raw_record_identity/"+rawID.String()+"/attachment/0" {
		t.Fatalf("locator = %q", got.SourceLocatorRef)
	}
	if got.Filename == nil || *got.Filename != "d2665f1c.png" || got.MediaType == nil || *got.MediaType != "image/png" {
		t.Fatalf("name/type not projected: %+v", got)
	}
	if got.SHA256 == nil || len(*got.SHA256) != 64 || got.ByteLength == nil || *got.ByteLength != 659746 {
		t.Fatalf("digest/size not projected: %+v", got)
	}
}

// A payload-less part (named in the backup, no bytes) has no digest and no size;
// it must still appear, with those fields absent rather than invented.
func TestStructuredAttachmentWithoutBytesKeepsFieldsAbsent(t *testing.T) {
	var attachments []previewStructuredAttachment
	if err := json.Unmarshal([]byte(`[{"name":"clip.3gp","mime":"video/3gpp","ordinal":2,"sha256":""}]`), &attachments); err != nil {
		t.Fatal(err)
	}
	got := attachments[0].project(uuid.New())
	if got.SHA256 != nil || got.ByteLength != nil {
		t.Fatalf("absent digest/size were invented: %+v", got)
	}
	if got.Filename == nil || *got.Filename != "clip.3gp" {
		t.Fatalf("name lost: %+v", got)
	}
}

// The SBV decoder reports a part the backup names but carries no bytes for as an
// attachment_reference of kind mms_part_without_payload. It reaches the preview
// as an attachment with a reference locator and no size or digest, so the
// surface can flag it; every other reference kind is left alone.
func TestPayloadlessPartReferenceProjectsAsAMissingAttachment(t *testing.T) {
	rawID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	reference := previewStructuredReference{Kind: "mms_part_without_payload", URIOriginal: "image000000_11449.jpg", DisplayText: "ct=image/heif seq=0"}
	projected, ok := reference.project(rawID, 0)
	if !ok {
		t.Fatal("a payload-less MMS part must project")
	}
	if projected.Filename == nil || *projected.Filename != "image000000_11449.jpg" ||
		projected.MediaType == nil || *projected.MediaType != "image/heif" ||
		projected.ByteLength != nil || projected.SHA256 != nil || !projected.PayloadMissing ||
		projected.SourceLocatorRef != "context.raw_record_identity/"+rawID.String()+"/attachment-reference/0" ||
		projected.AttachmentID != rawID.String()+":ref:0" {
		t.Fatalf("unexpected projection: %+v", projected)
	}
	if _, ok := (previewStructuredReference{Kind: "html_link", URIOriginal: "x.jpg"}).project(rawID, 1); ok {
		t.Fatal("other reference kinds are not attachments")
	}
}

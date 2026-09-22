// Byline: Claude Code · Fable 5.1 · 2026-09-21
package previewmodel

import "testing"

func ptr[T any](v T) *T { return &v }

// A backup can name a photo and carry no bytes for it. The preview says so on
// the attachment, whichever way the ingest recorded it (owner requirement
// 2026-09-20: missing-payload checks on ingestion and review).
func TestAttachmentMarksAMissingPayload(t *testing.T) {
	cases := map[string]struct {
		attachment Attachment
		missing    bool
	}{
		"real bytes":                        {Attachment{ByteLength: ptr(int64(2048)), SHA256: ptr("6105d6cc76af400325e94d588ce511be5bfdbb73b437dc51eca43917d7a43e3d"), SourceLocatorRef: "context.raw_record_identity/x/attachment/0"}, false},
		"size unknown, digest known":        {Attachment{SHA256: ptr("6105d6cc76af400325e94d588ce511be5bfdbb73b437dc51eca43917d7a43e3d"), SourceLocatorRef: "context.raw_record_identity/x/attachment/0"}, false},
		"derived before the fix: zero size": {Attachment{ByteLength: ptr(int64(0)), SourceLocatorRef: "context.raw_record_identity/x/attachment/1"}, true},
		"derived before the fix: empty sha": {Attachment{SHA256: ptr(EmptyContentSHA256), SourceLocatorRef: "context.raw_record_identity/x/attachment/1"}, true},
		"source-declared part, no payload":  {Attachment{SourceLocatorRef: "context.raw_record_identity/x" + MissingPayloadLocatorSegment + "0"}, true},
	}
	for name, tc := range cases {
		tc.attachment.MarkPayload()
		if tc.attachment.PayloadMissing != tc.missing {
			t.Fatalf("%s: payload_missing = %v, want %v", name, tc.attachment.PayloadMissing, tc.missing)
		}
	}
}

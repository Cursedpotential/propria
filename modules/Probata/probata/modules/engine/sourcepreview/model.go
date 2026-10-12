// Package sourcepreview reads bounded raw SMS backup pages without ingesting or retaining source content.
package sourcepreview

import "errors"

const (
	MaxRows                 = 50
	MaxScanBytes      int64 = 1 << 20
	MaxRecordBytes    int64 = 64 << 20
	MaxBodyBytes            = 4096
	MaxAttributeBytes       = 4096
	MaxPayloadBytes         = 512 << 10
	MaxStateBytes           = 64 << 10
	MaxParticipants         = 8
	MaxAttachments          = 8
	MaxDepth                = 16
	MaxTags                 = 4096
)

var (
	ErrInvalid     = errors.New("source_preview_invalid")
	ErrChanged     = errors.New("source_preview_source_changed")
	ErrUnavailable = errors.New("source_preview_unavailable")
	ErrExpired     = errors.New("source_preview_expired")
	ErrCapacity    = errors.New("source_preview_capacity")
)

type Request struct {
	SourceRef    string `json:"source_ref"`
	ExpectedETag string `json:"expected_etag"`
	VersionID    string `json:"version_id,omitempty"`
	Cursor       string `json:"cursor,omitempty"`
	Limit        int    `json:"limit,omitempty"`
}
type Source struct {
	SourceRef  string `json:"source_ref"`
	ETag       string `json:"etag"`
	VersionID  string `json:"version_id,omitempty"`
	ByteLength int64  `json:"byte_length"`
}
type Attachment struct {
	ContentType  string `json:"content_type"`
	Name         string `json:"name,omitempty"`
	EncodedBytes int64  `json:"encoded_bytes,omitempty"`
}
type Message struct {
	ID                string       `json:"id"`
	Kind              string       `json:"kind"`
	DateMS            *int64       `json:"date_ms"`
	Direction         string       `json:"direction"`
	Participants      []string     `json:"participants"`
	Body              string       `json:"body"`
	BodyTruncated     bool         `json:"body_truncated"`
	MetadataTruncated bool         `json:"metadata_truncated,omitempty"`
	Attachments       []Attachment `json:"attachments"`
}
type Page struct {
	PageID           string    `json:"page_id"`
	Status           string    `json:"status"`
	Source           Source    `json:"source"`
	Messages         []Message `json:"messages"`
	NextCursor       string    `json:"next_cursor,omitempty"`
	ScannedBytes     int64     `json:"scanned_bytes"`
	TruncationReason string    `json:"truncation_reason,omitempty"`
	PreviewOnly      bool      `json:"preview_only"`
}

// Task is fetched privately by the Activity; partial message state never enters Temporal history.
type Task struct {
	Request Request `json:"request"`
	State   State   `json:"state"`
}

// Completion transfers a bounded page and scanner continuation through the authenticated callback only.
type Completion struct {
	Page      Page   `json:"page"`
	State     State  `json:"state"`
	ErrorCode string `json:"error_code,omitempty"`
}

// Receipt is the only Activity output retained in Temporal history.
type Receipt struct {
	PageID       string `json:"page_id"`
	Rows         int    `json:"rows"`
	ScannedBytes int64  `json:"scanned_bytes"`
}

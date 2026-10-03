// Byline: Claude Code · Sonnet 5.5 · 2026-10-02

package cfjobs

import "context"

// SniffRow is one object's answer from the format sniffer Worker.
type SniffRow struct {
	Key           string  `json:"key"`
	Size          *int64  `json:"size"`
	BytesRead     int     `json:"bytes_read"`
	Format        string  `json:"format"`
	SignatureKind string  `json:"signature_kind"`
	Confidence    float64 `json:"confidence"`
	RuleSource    string  `json:"rule_source"`
	Error         string  `json:"error"`
}

// ZipMember is one central-directory entry from the ZIP lister Worker.
type ZipMember struct {
	Index             int64  `json:"index"`
	Name              string `json:"name"`
	CompSize          int64  `json:"comp_size"`
	Size              int64  `json:"size"`
	CRC32             string `json:"crc32"`
	Method            int    `json:"method"`
	Flags             int    `json:"flags"`
	Encrypted         bool   `json:"encrypted"`
	IsDir             bool   `json:"is_dir"`
	LocalHeaderOffset int64  `json:"local_header_offset"`
	MTime             string `json:"mtime"`
}

// ZipArchive is the summary of one archive from the ZIP lister Worker; Error is set when the archive could not be listed.
type ZipArchive struct {
	Key           string `json:"key"`
	Size          int64  `json:"size"`
	EntriesDecl   int64  `json:"entries_declared"`
	EntriesListed int64  `json:"entries_listed"`
	CDOffset      int64  `json:"cd_offset"`
	CDSize        int64  `json:"cd_size"`
	Zip64         bool   `json:"zip64"`
	CommentLen    int    `json:"comment_len"`
	PrefixBytes   int64  `json:"prefix_bytes"`
	Truncated     bool   `json:"truncated"`
	Requests      int    `json:"requests"`
	BytesRead     int64  `json:"bytes_read"`
	Error         string `json:"error"`
}

// HashResult is one object's digests from the B2 hasher Worker; Error is set when it could not be hashed.
type HashResult struct {
	Key            string `json:"key"`
	Size           int64  `json:"size"`
	SHA1           string `json:"sha1"`
	SHA256         string `json:"sha256"`
	BytesHashed    int64  `json:"bytes_hashed"`
	B2ContentSHA1  string `json:"b2_content_sha1"`
	B2FileID       string `json:"b2_file_id"`
	B2SHA1Mismatch bool   `json:"b2_sha1_mismatch"`
	MS             int64  `json:"ms"`
	Error          string `json:"error"`
}

// Catalog is the Case Bible catalog as the cf jobs use it: work lists out, Worker answers in.
//
// Every write is an idempotent upsert keyed by the object, so an Activity retry or a re-run changes nothing it already
// recorded. The PostgreSQL implementation is PGCatalog; tests use an in-memory one.
type Catalog interface {
	// NextBatch returns the next page of a job's work list after a key, ordered by key.
	NextBatch(ctx context.Context, in NextBatchInput) (NextBatchResult, error)
	// Plan counts a job's whole work list.
	Plan(ctx context.Context, in PlanInput) (objects, bytes int64, err error)
	// UpsertSniff records the sniffer's rows for one scope and ruleset.
	UpsertSniff(ctx context.Context, runID string, scope Scope, ruleset string, rows []SniffRow) error
	// UpsertZipMembers records a chunk of one archive's members.
	UpsertZipMembers(ctx context.Context, runID string, scope Scope, key string, rows []ZipMember) error
	// UpsertZipArchive records an archive's summary or error and drops member rows beyond what it now lists.
	UpsertZipArchive(ctx context.Context, runID string, scope Scope, archive ZipArchive) error
	// UpsertHash records one object's digests or error.
	UpsertHash(ctx context.Context, runID string, scope Scope, result HashResult) error
}

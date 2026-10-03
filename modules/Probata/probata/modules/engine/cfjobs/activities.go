// Byline: Claude Code · Sonnet 5.5 · 2026-10-02

package cfjobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
)

// Activities holds what the cf-jobs Activities need: the catalog and one client per Worker.
type Activities struct {
	Catalog   Catalog
	Sniffer   *WorkerClient
	ZipLister *WorkerClient
	B2Hasher  *WorkerClient
}

// heartbeat reports Activity progress, and does nothing when called outside an Activity (unit tests).
func heartbeat(ctx context.Context, details ...any) {
	defer func() { _ = recover() }()
	activity.RecordHeartbeat(ctx, details...)
}

func badInput(format string, args ...any) error {
	return temporal.NewNonRetryableApplicationError(fmt.Sprintf(format, args...), ErrorTypeBadRequest, nil)
}

func clampBatch(size, def, max int) int {
	if size <= 0 {
		return def
	}
	return min(size, max)
}

// NextBatch returns the next page of a job's work list from the catalog.
//
// Input: the job, its scope, the key to continue after, and the page size. Output: up to that many objects not yet recorded
// cleanly for the job, ordered by key; empty when the list is exhausted. Reads the catalog only. Pick it to page a work list
// out of the database instead of passing the list through Temporal history.
func (a *Activities) NextBatch(ctx context.Context, in NextBatchInput) (NextBatchResult, error) {
	if in.Limit <= 0 || in.Limit > 1000 {
		return NextBatchResult{}, badInput("next batch limit must be 1..1000, got %d", in.Limit)
	}
	return a.Catalog.NextBatch(ctx, in)
}

// Plan sizes a job's remaining work list and the Worker calls it needs.
//
// Input: the job, scope, batch size and optional item cap. Output: object count, bytes, batch count and Worker-request
// count. Reads the catalog only; it is what a dry run reports.
func (a *Activities) Plan(ctx context.Context, in PlanInput) (Plan, error) {
	if in.BatchSize <= 0 {
		return Plan{}, badInput("plan batch size must be positive")
	}
	objects, bytes, err := a.Catalog.Plan(ctx, in)
	if err != nil {
		return Plan{}, err
	}
	batches := (objects + int64(in.BatchSize) - 1) / int64(in.BatchSize)
	return Plan{Job: in.Job, Objects: objects, Bytes: bytes, Batches: batches, BatchSize: in.BatchSize, WorkerRequests: batches}, nil
}

// SniffFormats sends one batch of object keys to the format sniffer Worker and records what it detected.
//
// Input: a batch of up to 200 objects, the scope, the ruleset and head size. Output: counts only; every object's row (format,
// signature kind, confidence, or its error) is upserted into raw_duck.cf_format_sniff_20261002. Side effects: ranged reads of
// the first bytes of each object on Cloudflare's side, one catalog upsert. Safe to retry. Pick it to learn what format an
// object is without downloading it; use ListZipMembers for archives.
func (a *Activities) SniffFormats(ctx context.Context, batch SniffBatch) (Counts, error) {
	if n := len(batch.Items); n == 0 || n > SniffBatchMax {
		return Counts{}, badInput("sniff batch must hold 1..%d objects, got %d", SniffBatchMax, n)
	}
	scope := batch.Scope.withDefaults()
	ruleset := batch.Ruleset
	if ruleset == "" {
		ruleset = DefaultRuleset
	}
	type keyed struct {
		Key  string `json:"key"`
		Size int64  `json:"size"`
	}
	keys := make([]keyed, len(batch.Items))
	sizes := make(map[string]int64, len(batch.Items))
	for i, item := range batch.Items {
		keys[i] = keyed{item.Key, item.Size}
		sizes[item.Key] = item.Size
	}
	request := map[string]any{"provider": scope.Provider, "bucket": scope.Bucket, "keys": keys, "ruleset": ruleset}
	if batch.HeadBytes > 0 {
		request["head_bytes"] = batch.HeadBytes
	}
	var response struct {
		Results []SniffRow `json:"results"`
	}
	if err := a.Sniffer.PostJSON(ctx, "/sniff", request, &response); err != nil {
		return Counts{}, err
	}
	seen := make(map[string]bool, len(response.Results))
	rows := make([]SniffRow, 0, len(batch.Items))
	var counts Counts
	for _, row := range response.Results {
		size, known := sizes[row.Key]
		if !known {
			continue // the Worker answered for a key that was not asked
		}
		seen[row.Key] = true
		if row.Size == nil {
			s := size
			row.Size = &s
		}
		rows = append(rows, row)
		counts.Bytes += int64(row.BytesRead)
	}
	for _, item := range batch.Items {
		if !seen[item.Key] {
			s := item.Size
			rows = append(rows, SniffRow{Key: item.Key, Size: &s, Error: "no answer from the Worker"})
		}
	}
	if err := a.Catalog.UpsertSniff(ctx, batch.RunID, scope, ruleset, rows); err != nil {
		return Counts{}, err
	}
	for _, row := range rows {
		counts.Objects++
		if row.Error != "" {
			counts.Failed++
		}
	}
	return counts, nil
}

// ListZipMembers sends one batch of archive keys to the ZIP lister Worker and records every member it lists.
//
// Input: a batch of up to 50 archives, the scope and an optional per-archive member cap. Output: counts only; archive
// summaries go to raw_duck.cf_zip_archives_20261002 and member rows to cf_zip_members_20261002 as they stream in (upserts by
// archive and member index, stale tail rows dropped), so a retry repeats nothing it already wrote. Side effects: ranged reads
// of each archive's end and central directory on Cloudflare's side; no archive is downloaded. Pick it to see inside a ZIP
// without extracting it.
func (a *Activities) ListZipMembers(ctx context.Context, batch ZipBatch) (Counts, error) {
	if n := len(batch.Items); n == 0 || n > ZipBatchMax {
		return Counts{}, badInput("zip batch must hold 1..%d archives, got %d", ZipBatchMax, n)
	}
	scope := batch.Scope.withDefaults()
	type keyed struct {
		Key  string `json:"key"`
		Size int64  `json:"size"`
	}
	keys := make([]keyed, len(batch.Items))
	sizes := make(map[string]int64, len(batch.Items))
	for i, item := range batch.Items {
		keys[i] = keyed{item.Key, item.Size}
		sizes[item.Key] = item.Size
	}
	request := map[string]any{"provider": scope.Provider, "bucket": scope.Bucket, "keys": keys}
	if batch.MaxMembers > 0 {
		request["max_members"] = batch.MaxMembers
	}
	var counts Counts
	seen := map[string]bool{}
	err := a.ZipLister.PostNDJSON(ctx, "/list", request, func(line []byte) error {
		var head struct {
			Type string `json:"type"`
			Key  string `json:"key"`
		}
		if err := json.Unmarshal(line, &head); err != nil {
			return err
		}
		if head.Type == "members" || head.Type == "archive" || head.Type == "error" {
			if _, asked := sizes[head.Key]; !asked {
				return nil // a key that was not asked
			}
		}
		heartbeat(ctx, head.Key)
		switch head.Type {
		case "members":
			var chunk struct {
				Rows []ZipMember `json:"rows"`
			}
			if err := json.Unmarshal(line, &chunk); err != nil {
				return err
			}
			if err := a.Catalog.UpsertZipMembers(ctx, batch.RunID, scope, head.Key, chunk.Rows); err != nil {
				return err
			}
			counts.Members += len(chunk.Rows)
		case "archive":
			var archive ZipArchive
			if err := json.Unmarshal(line, &archive); err != nil {
				return err
			}
			if err := a.Catalog.UpsertZipArchive(ctx, batch.RunID, scope, archive); err != nil {
				return err
			}
			seen[head.Key] = true
			counts.Objects++
			counts.Bytes += archive.BytesRead
		case "error":
			var failure struct {
				Error string `json:"error"`
			}
			if err := json.Unmarshal(line, &failure); err != nil {
				return err
			}
			if err := a.Catalog.UpsertZipArchive(ctx, batch.RunID, scope, ZipArchive{Key: head.Key, Size: sizes[head.Key], Error: failure.Error}); err != nil {
				return err
			}
			seen[head.Key] = true
			counts.Objects++
			counts.Failed++
		}
		return nil
	})
	if err != nil {
		return Counts{}, err
	}
	for _, item := range batch.Items {
		if !seen[item.Key] {
			if err := a.Catalog.UpsertZipArchive(ctx, batch.RunID, scope, ZipArchive{Key: item.Key, Size: item.Size, Error: "no answer from the Worker"}); err != nil {
				return Counts{}, err
			}
			counts.Objects++
			counts.Failed++
		}
	}
	return counts, nil
}

// HashB2Objects sends one batch of B2 object keys to the B2 hasher Worker and records their SHA-1 and SHA-256.
//
// Input: a batch of up to 4 objects and the scope (B2 only). Output: counts only; each digest pair, or the object's error, is
// upserted into raw_duck.cf_b2_hashes_20261002 as soon as the Worker reports it. Side effects: each object is streamed once
// through the hashes on Cloudflare's side, which takes minutes for a multi-gigabyte object; the Activity heartbeats on every
// progress line. Safe to retry. Pick it for objects whose listing row has no SHA-1; it is the B2 twin of the R2 hasher.
func (a *Activities) HashB2Objects(ctx context.Context, batch HashBatch) (Counts, error) {
	if n := len(batch.Items); n == 0 || n > HashBatchMax {
		return Counts{}, badInput("hash batch must hold 1..%d objects, got %d", HashBatchMax, n)
	}
	scope := batch.Scope.withDefaults()
	if scope.Provider != "b2" {
		return Counts{}, badInput("the B2 hasher only reads B2, got provider %q", scope.Provider)
	}
	keys := make([]string, len(batch.Items))
	sizes := make(map[string]int64, len(batch.Items))
	for i, item := range batch.Items {
		keys[i] = item.Key
		sizes[item.Key] = item.Size
	}
	var counts Counts
	seen := map[string]bool{}
	err := a.B2Hasher.PostNDJSON(ctx, "/hash", map[string]any{"bucket": scope.Bucket, "keys": keys}, func(line []byte) error {
		var head struct {
			Type  string `json:"type"`
			Key   string `json:"key"`
			Bytes int64  `json:"bytes"`
		}
		if err := json.Unmarshal(line, &head); err != nil {
			return err
		}
		switch head.Type {
		case "progress":
			heartbeat(ctx, head.Key, head.Bytes)
		case "result":
			var result HashResult
			if err := json.Unmarshal(line, &result); err != nil {
				return err
			}
			if _, asked := sizes[result.Key]; !asked {
				return nil
			}
			if err := a.Catalog.UpsertHash(ctx, batch.RunID, scope, result); err != nil {
				return err
			}
			heartbeat(ctx, result.Key, result.BytesHashed)
			seen[result.Key] = true
			counts.Objects++
			counts.Bytes += result.BytesHashed
		case "error":
			var failure struct {
				Error string `json:"error"`
			}
			if err := json.Unmarshal(line, &failure); err != nil {
				return err
			}
			if _, asked := sizes[head.Key]; !asked {
				return nil
			}
			if err := a.Catalog.UpsertHash(ctx, batch.RunID, scope, HashResult{Key: head.Key, Size: sizes[head.Key], Error: failure.Error}); err != nil {
				return err
			}
			seen[head.Key] = true
			counts.Objects++
			counts.Failed++
		}
		return nil
	})
	if err != nil {
		return Counts{}, err
	}
	for _, item := range batch.Items {
		if !seen[item.Key] {
			return Counts{}, errors.New("the B2 hasher finished without answering for " + item.Key)
		}
	}
	return counts, nil
}

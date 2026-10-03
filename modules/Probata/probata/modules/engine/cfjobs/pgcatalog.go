// Byline: Claude Code · Sonnet 5.5 · 2026-10-02

package cfjobs

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// membersPerStatement bounds one INSERT of ZIP members (a Takeout archive can hold hundreds of thousands).
const membersPerStatement = 5000

// PGCatalog is the Catalog over the Case Bible PostgreSQL database (`casebible`, schema `raw_duck`).
//
// The tables and candidate views are created by casebible/tools/cf_workers_catalog_20261002.sql.
type PGCatalog struct {
	Pool *pgxpool.Pool
}

// candidateSource is the FROM-subquery that supplies a job's candidate objects: the job's candidate view, or, when the
// caller named keys, those keys straight from the current listing.
func candidateSource(job Job, withKeys bool) (string, error) {
	if withKeys {
		return `(select provider, bucket, key, size from raw_duck.bucket_objects_current
		          where provider = $1 and bucket = $2 and key = any($4::text[]))`, nil
	}
	switch job {
	case JobSniff:
		return `(select provider, bucket, key, size from raw_duck.cf_sniff_candidates_20261002 where provider = $1 and bucket = $2)`, nil
	case JobZip:
		return `(select provider, bucket, key, size from raw_duck.cf_zip_candidates_20261002 where provider = $1 and bucket = $2)`, nil
	case JobHash:
		return `(select provider, bucket, key, size from raw_duck.cf_b2_hash_candidates_20261002 where provider = $1 and bucket = $2)`, nil
	}
	return "", fmt.Errorf("unknown job %q", job)
}

// notDone is the anti-join that leaves out what the job already recorded for the object's current size without an error.
func notDone(job Job) (string, error) {
	switch job {
	case JobSniff:
		return `not exists (select 1 from raw_duck.cf_format_sniff_20261002 d where d.provider = c.provider and d.bucket = c.bucket
		           and d.key = c.key and d.ruleset = $5 and d.size is not distinct from c.size and d.error is null)`, nil
	case JobZip:
		return `not exists (select 1 from raw_duck.cf_zip_archives_20261002 d where d.provider = c.provider and d.bucket = c.bucket
		           and d.key = c.key and d.size is not distinct from c.size and d.error is null and not d.truncated and $5::text is not null)`, nil
	case JobHash:
		return `not exists (select 1 from raw_duck.cf_b2_hashes_20261002 d where d.bucket = c.bucket
		           and d.key = c.key and d.size is not distinct from c.size and d.error is null and $5::text is not null)`, nil
	}
	return "", fmt.Errorf("unknown job %q", job)
}

// NextBatch returns the next page of a job's work list: objects after `After` that the job has not recorded cleanly.
func (c *PGCatalog) NextBatch(ctx context.Context, in NextBatchInput) (NextBatchResult, error) {
	scope := in.Scope.withDefaults()
	source, err := candidateSource(in.Job, len(scope.Keys) > 0)
	if err != nil {
		return NextBatchResult{}, err
	}
	anti, err := notDone(in.Job)
	if err != nil {
		return NextBatchResult{}, err
	}
	query := `select c.key, c.size from ` + source + ` c where c.key > $3 and $4::text[] is not null and ` + anti + ` order by c.key limit $6`
	keys := scope.Keys
	if keys == nil {
		keys = []string{}
	}
	rows, err := c.Pool.Query(ctx, query, scope.Provider, scope.Bucket, in.After, keys, rulesetOrMarker(in), in.Limit)
	if err != nil {
		return NextBatchResult{}, fmt.Errorf("next batch (%s): %w", in.Job, err)
	}
	defer rows.Close()
	out := NextBatchResult{Items: []WorkItem{}}
	for rows.Next() {
		var item WorkItem
		if err := rows.Scan(&item.Key, &item.Size); err != nil {
			return NextBatchResult{}, err
		}
		out.Items = append(out.Items, item)
	}
	if err := rows.Err(); err != nil {
		return NextBatchResult{}, err
	}
	if n := len(out.Items); n > 0 {
		out.Last = out.Items[n-1].Key
	}
	return out, nil
}

// rulesetOrMarker supplies $5: the ruleset for the sniff job, and any non-null marker for the others.
func rulesetOrMarker(in NextBatchInput) string {
	if in.Job == JobSniff {
		if in.Ruleset == "" {
			return DefaultRuleset
		}
		return in.Ruleset
	}
	return "-"
}

// Plan counts the objects and bytes of a job's remaining work list (the first MaxItems of it when MaxItems is set).
func (c *PGCatalog) Plan(ctx context.Context, in PlanInput) (int64, int64, error) {
	scope := in.Scope.withDefaults()
	source, err := candidateSource(in.Job, len(scope.Keys) > 0)
	if err != nil {
		return 0, 0, err
	}
	anti, err := notDone(in.Job)
	if err != nil {
		return 0, 0, err
	}
	keys := scope.Keys
	if keys == nil {
		keys = []string{}
	}
	limit := in.MaxItems
	if limit <= 0 {
		limit = 1 << 30
	}
	marker := "-"
	if in.Job == JobSniff {
		marker = in.Ruleset
		if marker == "" {
			marker = DefaultRuleset
		}
	}
	query := `select count(*), coalesce(sum(size), 0)::bigint from
	            (select c.size from ` + source + ` c where c.key > $3 and $4::text[] is not null and ` + anti + ` order by c.key limit $6) t`
	var objects, bytes int64
	if err := c.Pool.QueryRow(ctx, query, scope.Provider, scope.Bucket, "", keys, marker, limit).Scan(&objects, &bytes); err != nil {
		return 0, 0, fmt.Errorf("plan (%s): %w", in.Job, err)
	}
	return objects, bytes, nil
}

// UpsertSniff records the sniffer's rows for one scope and ruleset.
func (c *PGCatalog) UpsertSniff(ctx context.Context, runID string, scope Scope, ruleset string, rows []SniffRow) error {
	if len(rows) == 0 {
		return nil
	}
	scope = scope.withDefaults()
	keys := make([]string, len(rows))
	sizes := make([]int64, len(rows))
	read := make([]int32, len(rows))
	formats := make([]string, len(rows))
	kinds := make([]string, len(rows))
	conf := make([]float64, len(rows))
	sources := make([]string, len(rows))
	errs := make([]string, len(rows))
	for i, r := range rows {
		keys[i], read[i], formats[i], kinds[i], conf[i], sources[i], errs[i] = r.Key, int32(r.BytesRead), r.Format, r.SignatureKind, r.Confidence, r.RuleSource, r.Error
		sizes[i] = -1
		if r.Size != nil {
			sizes[i] = *r.Size
		}
	}
	_, err := c.Pool.Exec(ctx, `
insert into raw_duck.cf_format_sniff_20261002
  (provider, bucket, key, ruleset, size, bytes_read, format, signature_kind, confidence, rule_source, error, run_id, sniffed_at)
select $1, $2, t.key, $3, nullif(t.size, -1), t.bytes_read, nullif(t.format, ''), nullif(t.kind, ''), case when t.format = '' then null else t.conf end,
       nullif(t.source, ''), nullif(t.err, ''), $4, now()
from unnest($5::text[], $6::bigint[], $7::int[], $8::text[], $9::text[], $10::float8[], $11::text[], $12::text[])
     as t(key, size, bytes_read, format, kind, conf, source, err)
on conflict (provider, bucket, key, ruleset) do update set
  size = coalesce(excluded.size, raw_duck.cf_format_sniff_20261002.size), bytes_read = excluded.bytes_read, format = excluded.format,
  signature_kind = excluded.signature_kind, confidence = excluded.confidence, rule_source = excluded.rule_source,
  error = excluded.error, run_id = excluded.run_id, sniffed_at = excluded.sniffed_at`,
		scope.Provider, scope.Bucket, ruleset, runID, keys, sizes, read, formats, kinds, conf, sources, errs)
	if err != nil {
		return fmt.Errorf("upsert sniff rows: %w", err)
	}
	return nil
}

// UpsertZipMembers records a chunk of one archive's members, in statements of at most membersPerStatement rows.
func (c *PGCatalog) UpsertZipMembers(ctx context.Context, runID string, scope Scope, key string, rows []ZipMember) error {
	scope = scope.withDefaults()
	for start := 0; start < len(rows); start += membersPerStatement {
		end := min(start+membersPerStatement, len(rows))
		part := rows[start:end]
		idx := make([]int64, len(part))
		names := make([]string, len(part))
		comp := make([]int64, len(part))
		size := make([]int64, len(part))
		crc := make([]string, len(part))
		method := make([]int32, len(part))
		flags := make([]int32, len(part))
		enc := make([]bool, len(part))
		dir := make([]bool, len(part))
		lho := make([]int64, len(part))
		mtime := make([]string, len(part))
		for i, m := range part {
			idx[i], names[i], comp[i], size[i], crc[i] = m.Index, m.Name, m.CompSize, m.Size, m.CRC32
			method[i], flags[i], enc[i], dir[i], lho[i], mtime[i] = int32(m.Method), int32(m.Flags), m.Encrypted, m.IsDir, m.LocalHeaderOffset, m.MTime
		}
		_, err := c.Pool.Exec(ctx, `
insert into raw_duck.cf_zip_members_20261002
  (provider, bucket, key, member_index, name, comp_size, size, crc32, method, flags, encrypted, is_dir, local_header_offset, mtime, run_id)
select $1, $2, $3, t.idx, replace(t.name, chr(0), ''), t.comp, t.size, t.crc, t.method, t.flags, t.enc, t.dir, t.lho, nullif(t.mtime, ''), $4
from unnest($5::bigint[], $6::text[], $7::bigint[], $8::bigint[], $9::text[], $10::int[], $11::int[], $12::bool[], $13::bool[], $14::bigint[], $15::text[])
     as t(idx, name, comp, size, crc, method, flags, enc, dir, lho, mtime)
on conflict (provider, bucket, key, member_index) do update set
  name = excluded.name, comp_size = excluded.comp_size, size = excluded.size, crc32 = excluded.crc32, method = excluded.method,
  flags = excluded.flags, encrypted = excluded.encrypted, is_dir = excluded.is_dir, local_header_offset = excluded.local_header_offset,
  mtime = excluded.mtime, run_id = excluded.run_id`,
			scope.Provider, scope.Bucket, key, runID, idx, names, comp, size, crc, method, flags, enc, dir, lho, mtime)
		if err != nil {
			return fmt.Errorf("upsert zip members of %s: %w", key, err)
		}
	}
	return nil
}

// UpsertZipArchive records an archive's summary or error and removes member rows beyond what the archive now lists.
func (c *PGCatalog) UpsertZipArchive(ctx context.Context, runID string, scope Scope, a ZipArchive) error {
	scope = scope.withDefaults()
	tx, err := c.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if a.Error == "" {
		if _, err := tx.Exec(ctx, `delete from raw_duck.cf_zip_members_20261002 where provider = $1 and bucket = $2 and key = $3 and member_index >= $4`,
			scope.Provider, scope.Bucket, a.Key, a.EntriesListed); err != nil {
			return fmt.Errorf("drop stale members of %s: %w", a.Key, err)
		}
	}
	if _, err := tx.Exec(ctx, `
insert into raw_duck.cf_zip_archives_20261002
  (provider, bucket, key, size, entries_declared, entries_listed, cd_offset, cd_size, zip64, comment_len, prefix_bytes, truncated, requests, bytes_read, error, run_id, listed_at)
values ($1, $2, $3, nullif($4, 0), $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, nullif($15, ''), $16, now())
on conflict (provider, bucket, key) do update set
  size = coalesce(excluded.size, raw_duck.cf_zip_archives_20261002.size), entries_declared = excluded.entries_declared, entries_listed = excluded.entries_listed,
  cd_offset = excluded.cd_offset, cd_size = excluded.cd_size, zip64 = excluded.zip64, comment_len = excluded.comment_len,
  prefix_bytes = excluded.prefix_bytes, truncated = excluded.truncated, requests = excluded.requests, bytes_read = excluded.bytes_read,
  error = excluded.error, run_id = excluded.run_id, listed_at = excluded.listed_at`,
		scope.Provider, scope.Bucket, a.Key, a.Size, a.EntriesDecl, a.EntriesListed, a.CDOffset, a.CDSize, a.Zip64, a.CommentLen, a.PrefixBytes,
		a.Truncated, a.Requests, a.BytesRead, a.Error, runID); err != nil {
		return fmt.Errorf("upsert zip archive %s: %w", a.Key, err)
	}
	return tx.Commit(ctx)
}

// UpsertHash records one object's digests or error.
func (c *PGCatalog) UpsertHash(ctx context.Context, runID string, scope Scope, r HashResult) error {
	scope = scope.withDefaults()
	if scope.Provider != "b2" {
		return errors.New("the hash backfill records B2 objects only")
	}
	_, err := c.Pool.Exec(ctx, `
insert into raw_duck.cf_b2_hashes_20261002
  (bucket, key, size, sha1, sha256, bytes_hashed, b2_content_sha1, b2_file_id, b2_sha1_mismatch, ms, error, run_id, hashed_at)
values ($1, $2, nullif($3, 0), nullif($4, ''), nullif($5, ''), nullif($6, 0), nullif($7, ''), nullif($8, ''), $9, nullif($10, 0), nullif($11, ''), $12, now())
on conflict (bucket, key) do update set
  size = coalesce(excluded.size, raw_duck.cf_b2_hashes_20261002.size), sha1 = excluded.sha1, sha256 = excluded.sha256, bytes_hashed = excluded.bytes_hashed,
  b2_content_sha1 = excluded.b2_content_sha1, b2_file_id = excluded.b2_file_id, b2_sha1_mismatch = excluded.b2_sha1_mismatch,
  ms = excluded.ms, error = excluded.error, run_id = excluded.run_id, hashed_at = excluded.hashed_at`,
		scope.Bucket, r.Key, r.Size, r.SHA1, r.SHA256, r.BytesHashed, r.B2ContentSHA1, r.B2FileID, r.B2SHA1Mismatch, r.MS, strings.TrimSpace(r.Error), runID)
	if err != nil {
		return fmt.Errorf("upsert hash of %s: %w", r.Key, err)
	}
	return nil
}

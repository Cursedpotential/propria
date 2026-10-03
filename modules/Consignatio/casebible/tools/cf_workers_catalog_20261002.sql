-- Byline: Claude Code · Sonnet 5.5 · 2026-10-02
-- Catalog side of the three Cloudflare bulk-read Workers (casebible/tools/cf_workers/): the tables their Go Temporal Activities
-- (engine/cfjobs) write, and the candidate views that are their work lists. Owner approval 2026-10-02 20:50 EDT.
--
-- Idempotent: every statement is `if not exists` / `or replace`, and the Activities write by upsert on the primary keys, so this file
-- and the jobs can be re-run. Nothing here reads a bucket. The catalog stays the one source of truth: a Worker keeps nothing, and
-- the human-readable side of a result is a query over these tables.
--
--   cf_format_sniff_20261002   casebible-format-sniffer   one row per (object, ruleset): detected format, signature kind, confidence
--   cf_zip_archives_20261002   casebible-zip-lister       one row per archive: end-of-central-directory facts, or the error
--   cf_zip_members_20261002    casebible-zip-lister       one row per member: name, sizes, CRC32, method (central directory only)
--   cf_b2_hashes_20261002      casebible-b2-hasher        SHA-1 + SHA-256 of B2 objects that had no hash in the listing
--
-- Candidate views (what each job looks at when it is started without explicit keys):
--   cf_sniff_candidates_20261002   the AI-chat candidates of raw_duck.ai_chat_probe_20260918 with extension txt or html (30,048 + 9,308)
--   cf_zip_candidates_20261002     the ZIPs of that probe table (probe_class zip)
--   cf_b2_hash_candidates_20261002 B2 objects whose current listing row has no SHA-1
-- An object leaves a job's list once the job has recorded it for its current size without an error.
--
-- Applying it writes to the catalog, so it is run on ovh-files by the owner's go-ahead:
--   docker exec -i <casebible-pg container> psql -U postgres -d casebible -v ON_ERROR_STOP=1 < cf_workers_catalog_20261002.sql
\set ON_ERROR_STOP 1
begin;

create table if not exists raw_duck.cf_format_sniff_20261002 (
  provider       text        not null,
  bucket         text        not null,
  key            text        not null,
  ruleset        text        not null,            -- proffer-v1 | casebible-probe-v1
  size           bigint,                          -- listing size when the sniffed head was read
  bytes_read     integer,
  format         text,                            -- Proffer registry id (chatgpt_official_json, ai_markdown_transcript, generic_html_document, ...)
  signature_kind text,
  confidence     numeric(3, 2),                   -- per-rule score assigned by the Worker (the Go and Python sources return none)
  rule_source    text,                            -- which signature source decided: handler_ai_chat_signature.go, html_signature.go, ...
  error          text,
  run_id         text        not null,            -- Temporal run id of the job that wrote the row
  sniffed_at     timestamptz not null default now(),
  primary key (provider, bucket, key, ruleset)
);
create index if not exists cf_format_sniff_20261002_format_idx on raw_duck.cf_format_sniff_20261002 (format);

create table if not exists raw_duck.cf_zip_archives_20261002 (
  provider         text        not null,
  bucket           text        not null,
  key              text        not null,
  size             bigint,
  entries_declared bigint,                        -- from the end record (ZIP64 record when the EOCD is saturated)
  entries_listed   bigint,
  cd_offset        bigint,
  cd_size          bigint,
  zip64            boolean,
  comment_len      integer,
  prefix_bytes     bigint,                        -- bytes in front of the archive (self-extractor, wrapper)
  truncated        boolean     not null default false,   -- the Worker stopped at max_members or its central-directory byte cap
  requests         integer,
  bytes_read       bigint,                        -- bytes the Worker read from the bucket: the tail and the central directory only
  error            text,
  run_id           text        not null,
  listed_at        timestamptz not null default now(),
  primary key (provider, bucket, key)
);

create table if not exists raw_duck.cf_zip_members_20261002 (
  provider            text    not null,
  bucket              text    not null,
  key                 text    not null,           -- the archive
  member_index        bigint  not null,           -- order in the central directory
  name                text    not null,
  comp_size           bigint,
  size                bigint,
  crc32               text,                       -- 8 lowercase hex digits
  method              integer,                    -- 0 stored, 8 deflate, ...
  flags               integer,
  encrypted           boolean,
  is_dir              boolean,
  local_header_offset bigint,
  mtime               text,                       -- ISO; local DOS time, or UTC with a Z when the extended timestamp field is present
  run_id              text    not null,
  primary key (provider, bucket, key, member_index)
);
create index if not exists cf_zip_members_20261002_crc_idx on raw_duck.cf_zip_members_20261002 (crc32, size);

create table if not exists raw_duck.cf_b2_hashes_20261002 (
  bucket           text        not null,
  key              text        not null,
  size             bigint,
  sha1             text,
  sha256           text,
  bytes_hashed     bigint,
  b2_content_sha1  text,                          -- B2's own x-bz-content-sha1 header ("none" for objects B2 never hashed)
  b2_file_id       text,
  b2_sha1_mismatch boolean     not null default false,
  ms               bigint,
  error            text,
  run_id           text        not null,
  hashed_at        timestamptz not null default now(),
  primary key (bucket, key)
);
create index if not exists cf_b2_hashes_20261002_sha1_idx on raw_duck.cf_b2_hashes_20261002 (sha1);

create or replace view raw_duck.cf_sniff_candidates_20261002 as
select distinct o.provider, o.bucket, o.key, o.size
from raw_duck.ai_chat_probe_20260918 p
join raw_duck.bucket_objects_current o
  on o.provider = 'b2' and o.bucket = 'salem-data' and o.key = p.vault_key
where p.ext in ('txt', 'html') and o.size > 0;

create or replace view raw_duck.cf_zip_candidates_20261002 as
select distinct o.provider, o.bucket, o.key, o.size
from raw_duck.ai_chat_probe_20260918 p
join raw_duck.bucket_objects_current o
  on o.provider = 'b2' and o.bucket = 'salem-data' and o.key = p.vault_key
where p.probe_class = 'zip' and o.size >= 22;

create or replace view raw_duck.cf_b2_hash_candidates_20261002 as
select o.provider, o.bucket, o.key, o.size
from raw_duck.bucket_objects_current o
where o.provider = 'b2' and o.bucket = 'salem-data' and (o.sha1 is null or o.sha1 = '') and o.size > 0;

comment on table raw_duck.cf_format_sniff_20261002 is 'Format sniffer Worker results (2026-10-02): head-of-object content signatures, Proffer registry format ids. Written by engine/cfjobs SniffFormatsWorkflow.';
comment on table raw_duck.cf_zip_archives_20261002 is 'ZIP lister Worker results (2026-10-02): one row per archive. Written by engine/cfjobs ListZipMembersWorkflow.';
comment on table raw_duck.cf_zip_members_20261002 is 'ZIP lister Worker results (2026-10-02): central-directory members, no archive downloaded. Written by engine/cfjobs ListZipMembersWorkflow.';
comment on table raw_duck.cf_b2_hashes_20261002 is 'B2 hasher Worker results (2026-10-02): SHA-1 + SHA-256 of B2 objects that had no SHA-1 in the listing. Written by engine/cfjobs BackfillB2HashesWorkflow.';
commit;

select 'cf_sniff_candidates' as list, count(*) as objects, pg_size_pretty(sum(size)) as bytes from raw_duck.cf_sniff_candidates_20261002
union all select 'cf_zip_candidates', count(*), pg_size_pretty(sum(size)) from raw_duck.cf_zip_candidates_20261002
union all select 'cf_b2_hash_candidates', count(*), pg_size_pretty(sum(size)) from raw_duck.cf_b2_hash_candidates_20261002;

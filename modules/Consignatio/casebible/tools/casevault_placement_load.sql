-- Byline: Claude Code · Opus 5.5 · 2026-10-02
-- Record each old -> new casevault placement (owner 2026-10-02: casevault is the home root; old keys stay).
-- Input: /tmp/placement_result.tsv from casevault_placement.sh run mode:
--   src_key dst_key src_size dst_size src_sha1 dst_sha1 status   (keys relative to bucket salem-data)
-- psql -v plan_id='<dated plan name>' -v placed_at='<ISO UTC>'. Append-only; a re-run of the same plan
-- keeps the first row per (plan_id, dst_key).
\set PREFIX ''

begin;

create table if not exists raw_duck.casevault_placement (
  plan_id text not null,
  src_key text not null,
  dst_key text not null,
  src_size bigint,
  dst_size bigint,
  src_sha1 text,
  dst_sha1 text,
  status text not null check (status in ('ok', 'mismatch', 'missing')),
  method text not null default 'b2_server_side_copy',
  placed_at timestamptz not null,
  primary key (plan_id, dst_key)
);

create temp table placement_in (src_key text, dst_key text, src_size text, dst_size text, src_sha1 text, dst_sha1 text, status text);
\copy placement_in from '/tmp/placement_result.tsv' with (format csv, delimiter E'\t', quote E'\x01', escape E'\x01')
insert into raw_duck.casevault_placement (plan_id, src_key, dst_key, src_size, dst_size, src_sha1, dst_sha1, status, placed_at)
select :'plan_id', src_key, dst_key, nullif(src_size, '')::bigint, nullif(dst_size, '')::bigint,
       nullif(src_sha1, ''), nullif(dst_sha1, ''), status, :'placed_at'::timestamptz
from placement_in
on conflict do nothing;

commit;

select status, count(*), coalesce(sum(dst_size), 0) as bytes
from raw_duck.casevault_placement where plan_id = :'plan_id' group by status order by status;

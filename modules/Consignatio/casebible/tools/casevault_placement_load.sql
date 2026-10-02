-- Byline: Claude Code · Opus 5.5 · 2026-10-02
-- Record each old -> new casevault placement (owner 2026-10-02: casevault is the home root; old keys stay).
-- Input: /tmp/placement_result.tsv from casevault_placement.sh run|verify mode:
--   src_key dst_key src_size dst_size src_sha1 dst_sha1 status proof_level   (keys relative to bucket salem-data)
-- psql -v plan_id='<dated plan name>' -v placed_at='<ISO UTC>'.
-- placed_at is kept from the first load of a (plan_id, dst_key); a later verify run of the same plan updates
-- the measured sizes, hashes, status and proof_level and stamps verified_at (catalog context data, not an
-- audit trail: the newest verification is the truth).
-- proof_level: b2_stored_sha1 (both SHA-1s from B2 metadata) | vps_computed_sha1 (at least one side streamed
-- and hashed on ovh-files) | none. status ok requires equal sizes AND equal SHA-1s.

begin;

create table if not exists raw_duck.casevault_placement (
  plan_id text not null,
  src_key text not null,
  dst_key text not null,
  src_size bigint,
  dst_size bigint,
  src_sha1 text,
  dst_sha1 text,
  status text not null,
  method text not null default 'b2_server_side_copy',
  placed_at timestamptz not null,
  primary key (plan_id, dst_key)
);
alter table raw_duck.casevault_placement add column if not exists proof_level text;
alter table raw_duck.casevault_placement add column if not exists verified_at timestamptz;
alter table raw_duck.casevault_placement drop constraint if exists casevault_placement_status_check;
alter table raw_duck.casevault_placement add constraint casevault_placement_status_check
  check (status in ('ok', 'mismatch', 'missing', 'unverified'));
alter table raw_duck.casevault_placement drop constraint if exists casevault_placement_ok_needs_hash;
alter table raw_duck.casevault_placement add constraint casevault_placement_ok_needs_hash
  check (status <> 'ok' or (src_sha1 is not null and src_sha1 = dst_sha1 and src_size = dst_size
                            and proof_level in ('b2_stored_sha1', 'vps_computed_sha1'))) not valid;

create temp table placement_in (src_key text, dst_key text, src_size text, dst_size text, src_sha1 text, dst_sha1 text,
                                status text, proof_level text);
\copy placement_in from '/tmp/placement_result.tsv' with (format csv, delimiter E'\t', quote E'\x01', escape E'\x01')
insert into raw_duck.casevault_placement (plan_id, src_key, dst_key, src_size, dst_size, src_sha1, dst_sha1, status,
                                          proof_level, placed_at, verified_at)
select :'plan_id', src_key, dst_key, nullif(src_size, '')::bigint, nullif(dst_size, '')::bigint,
       nullif(src_sha1, ''), nullif(dst_sha1, ''), status, nullif(proof_level, ''), :'placed_at'::timestamptz, now()
from placement_in
on conflict (plan_id, dst_key) do update set
  src_size = excluded.src_size, dst_size = excluded.dst_size, src_sha1 = excluded.src_sha1,
  dst_sha1 = excluded.dst_sha1, status = excluded.status, proof_level = excluded.proof_level,
  verified_at = excluded.verified_at;

commit;

select status, proof_level, count(*), coalesce(sum(dst_size), 0) as bytes
from raw_duck.casevault_placement where plan_id = :'plan_id' group by 1, 2 order by 1, 2;

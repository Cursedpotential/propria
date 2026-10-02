-- Byline: Claude Code · Opus 5.5 · 2026-10-02
-- casevault (owner 2026-10-02: "everything's new home root" = b2:salem-data/consignatio/casevault/) in the catalog.
-- Input staged by casevault_listing_load.sh: /tmp/casevault_listing.tsv = rel_key\tsize\tsha1 from rclone lsf
-- of the casevault root; psql -v listed_at = the listing file's mtime (UTC). Append-only, one generation per load.
\set PREFIX 'consignatio/casevault/'

begin;

create table if not exists raw_duck.casevault_objects (
  key text not null,
  size bigint not null,
  sha1 text,
  listed_at timestamptz not null,
  primary key (key, listed_at)
);

create temp table casevault_in (rel text, size bigint, sha1 text);
\copy casevault_in from '/tmp/casevault_listing.tsv' with (format csv, delimiter E'\t', quote E'\x01', escape E'\x01')
insert into raw_duck.casevault_objects (key, size, sha1, listed_at)
select :'PREFIX' || rel, size, nullif(sha1, ''), :'listed_at'::timestamptz from casevault_in
on conflict do nothing;

create or replace view raw_duck.casevault_objects_current as
select distinct on (key) key, size, sha1, listed_at
from raw_duck.casevault_objects
where listed_at = (select max(listed_at) from raw_duck.casevault_objects)
order by key, listed_at desc;

commit;

select count(*) as objects, coalesce(sum(size), 0) as bytes, max(listed_at) as listed_at
from raw_duck.casevault_objects_current;

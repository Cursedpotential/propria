-- Byline: Claude Code · Opus 5.5 · 2026-10-01
-- One identity store: the Case Bible catalog reads person <-> identifier rows from Probata registry instead of
-- keeping its own editable copy. Owner order 2026-10-01 07:57: registry is the ONE identity store; the Workbench
-- Case page edits it; everything else reads it.
--
-- Before: raw_duck.msg_identity_20260924 was a table every Case Bible extraction read and several tools wrote.
-- After:  raw_duck.msg_identity_20260924 is a VIEW over the foreign table registry_fdw.vw_case_identifier
--         (postgres_fdw -> platform database, registry.vw_case_identifier, login registry_catalog_reader, SELECT only).
--         Same columns, same values, so msg_extract_worklist_20260924.sql, msg_cross_device_20260924.sql and every other
--         reader keep working and now read registry live. A write (insert/update/delete) fails: the remote login has no
--         write privilege and the view's write privileges are revoked here. Identities are edited on the Case page.
--         The old table is renamed raw_duck.msg_identity_20260924_retired_20261001 and kept (never deleted).
--
-- Run ON ovh-files, after scripts/2026-10-01-seed-registry-identity.sh committed (Probata repo), with the reader's
-- password passed as a psql variable read from the host secret file (never in git):
--   docker exec -i fgz1n7useplhk0t91uk7k1aw psql -U postgres -d casebible -v ON_ERROR_STOP=1 \
--     -v reader_password="$(cat /data/probata/secrets/registry_catalog_reader.password)" < identity_registry_fdw_20261001.sql
-- Idempotent: a re-run replaces the server options, user mapping, foreign tables and view and leaves the retired table.

begin;

create extension if not exists postgres_fdw;

do $$
begin
  if not exists (select 1 from pg_foreign_server where srvname = 'probata_registry') then
    execute $q$create server probata_registry foreign data wrapper postgres_fdw
             options (host '100.91.190.107', port '5432', dbname 'platform', fetch_size '1000')$q$;
  end if;
end $$;

-- Every catalog login reads the projection through one SELECT-only remote login.
drop user mapping if exists for public server probata_registry;
create user mapping for public server probata_registry
  options (user 'registry_catalog_reader', password :'reader_password');

create schema if not exists registry_fdw;
comment on schema registry_fdw is 'Foreign tables over Probata registry (read-only). Identities are edited on the Workbench Case page.';
drop foreign table if exists registry_fdw.vw_case_identifier;
drop foreign table if exists registry_fdw.vw_identifier_dismissed;
import foreign schema registry limit to (vw_case_identifier, vw_identifier_dismissed)
  from server probata_registry into registry_fdw;

do $$
begin
  if to_regclass('raw_duck.msg_identity_20260924_retired_20261001') is null
     and exists (select 1 from pg_class c join pg_namespace n on n.oid = c.relnamespace
                 where n.nspname = 'raw_duck' and c.relname = 'msg_identity_20260924' and c.relkind = 'r') then
    alter table raw_duck.msg_identity_20260924 rename to msg_identity_20260924_retired_20261001;
    comment on table raw_duck.msg_identity_20260924_retired_20261001 is
      'RETIRED 2026-10-01: seeded into Probata registry (scripts/2026-10-01-seed-registry-identity.sh). Kept as the seed source; never edited. raw_duck.msg_identity_20260924 is now a view over registry.';
    revoke insert, update, delete, truncate on raw_duck.msg_identity_20260924_retired_20261001 from public;
  end if;
end $$;

create or replace view raw_duck.msg_identity_20260924 as
select person, identifier, raw_value, kind, status, period, basis, added_at
from registry_fdw.vw_case_identifier;
comment on view raw_duck.msg_identity_20260924 is
  'Person <-> identifier, read live from Probata registry (registry.vw_case_identifier over postgres_fdw). Edit on the Workbench Case page; this view is read-only.';
revoke insert, update, delete, truncate on raw_duck.msg_identity_20260924 from public;

create or replace view raw_duck.msg_identity_dismissed_20261001 as
select normalized as identifier, raw_value, basis, recorded_by, recorded_at from registry_fdw.vw_identifier_dismissed;
comment on view raw_duck.msg_identity_dismissed_20261001 is 'Identifiers the owner set aside on the Case page (registry.identifier_triage), read live.';

grant usage on schema registry_fdw to metabase_ro;
grant select on registry_fdw.vw_case_identifier, registry_fdw.vw_identifier_dismissed to metabase_ro;
grant select on raw_duck.msg_identity_20260924, raw_duck.msg_identity_dismissed_20261001 to metabase_ro;

-- read-back
select person, status, count(*) from raw_duck.msg_identity_20260924 group by 1, 2 order by 1, 2;
select (select count(*) from raw_duck.msg_identity_20260924_retired_20261001) as retired_rows,
       (select count(*) from raw_duck.msg_identity_20260924) as registry_rows,
       (select count(*) from raw_duck.msg_identity_20260924_retired_20261001 r
          where not exists (select 1 from raw_duck.msg_identity_20260924 v
                            where v.person = r.person and v.identifier = r.identifier and v.raw_value = r.raw_value)) as missing_in_registry;

commit;

-- Byline: Claude Code · Opus 5.5 · 2026-10-02
-- Owner 2026-10-02 19:04 EDT: "the B2 objects [table] needs to match whatever it's listing"; 20:18 "go" on repointing
-- its live readers (Probata engine find_other_version, tools/contacts_manifest.py, catalog_reconcile/run.py) and then
-- renaming it. Run only after the proffer-worker that carries the repointed engine is deployed.
-- Moves the last three superseded tables that live code read into raw_duck_superseded under names that say what they
-- hold, and updates raw_duck.catalog_registry. Nothing is deleted. Log: modules/Consignatio/docs/LOG.md.
\set ON_ERROR_STOP 1
begin;
alter table raw_duck.b2_objects set schema raw_duck_superseded;
alter table raw_duck_superseded.b2_objects rename to b2_intake_objects_20260914;
alter table raw_duck.vault_objects set schema raw_duck_superseded;
alter table raw_duck_superseded.vault_objects rename to vault_objects_20260916_0810_prededupe;
alter table raw_duck.vault_content_v0 set schema raw_duck_superseded;

update raw_duck.catalog_registry set object_schema = 'raw_duck_superseded', object_name = 'b2_intake_objects_20260914',
  moved_from = 'raw_duck.b2_objects', evidence = evidence || ' | renamed 2026-10-02 after its readers moved to bucket_objects'
where object_schema = 'raw_duck' and object_name = 'b2_objects';
update raw_duck.catalog_registry set object_schema = 'raw_duck_superseded', object_name = 'vault_objects_20260916_0810_prededupe',
  moved_from = 'raw_duck.vault_objects', evidence = evidence || ' | renamed 2026-10-02'
where object_schema = 'raw_duck' and object_name = 'vault_objects';
update raw_duck.catalog_registry set object_schema = 'raw_duck_superseded', moved_from = 'raw_duck.vault_content_v0'
where object_schema = 'raw_duck' and object_name = 'vault_content_v0';
commit;

select object_schema || '.' || object_name, status, moved_from from raw_duck.catalog_registry
where moved_from in ('raw_duck.b2_objects', 'raw_duck.vault_objects', 'raw_duck.vault_content_v0');
select to_regclass('raw_duck.b2_objects') is null as old_name_gone,
       (select count(*) from raw_duck_superseded.b2_intake_objects_20260914) as intake_rows;

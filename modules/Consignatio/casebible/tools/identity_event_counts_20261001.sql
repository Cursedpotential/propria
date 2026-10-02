-- Byline: Claude Code · Opus 5.5 · 2026-10-01
-- What the Case Bible catalog holds per identifier, for the Workbench Case page (owner order 2026-10-01 07:56:
-- "view what's been extracted as far as what's in there"). Read-only to the Workbench (metabase_ro); rebuilt by
-- re-running this file. Identities themselves are NOT here: they live in Probata registry and the catalog reads them
-- through raw_duck.msg_identity_20260924 (a postgres_fdw view since 2026-10-01, identity_registry_fdw_20261001.sql).
--
-- raw_duck.identity_event_counts_20261001: one row per (identifier, match_on, event_kind) over
-- raw_duck.comm_events_20260918 (messages, calls, AI chats; 551,877 rows on 2026-10-01).
--   identifier  raw_duck.norm_phone(...) -- the same key registry.norm_identifier produces
--   match_on    counterparty_phone: the number is the other party on a phone backup (SMS / calls)
--               sender:             the value is the sender label of the event (any export: Facebook name, "owner", +1...)
--   event_kind  message | call | ai_chat_turn | document
-- Two expression indexes let the Case page open the events behind a count without scanning the table.

begin;

drop table if exists raw_duck.identity_event_counts_20261001;
create table raw_duck.identity_event_counts_20261001 as
with keyed as (
  select raw_duck.norm_phone(counterparty_phone) as identifier, 'counterparty_phone'::text as match_on,
         event_kind, conversation_id, event_ts_utc, source_format
  from raw_duck.comm_events_20260918
  where counterparty_phone is not null and btrim(counterparty_phone) <> ''
  union all
  select raw_duck.norm_phone(sender), 'sender', event_kind, conversation_id, event_ts_utc, source_format
  from raw_duck.comm_events_20260918
  where sender is not null and btrim(sender) <> ''
)
select identifier, match_on, event_kind,
       count(*)::bigint as events,
       count(distinct conversation_id)::bigint as conversations,
       min(event_ts_utc) as first_at,
       max(event_ts_utc) as last_at,
       array_agg(distinct source_format order by source_format) as sources
from keyed
where identifier is not null
group by identifier, match_on, event_kind;

create index identity_event_counts_20261001_identifier_idx on raw_duck.identity_event_counts_20261001 (identifier);
comment on table raw_duck.identity_event_counts_20261001 is
  'Per-identifier event counts over raw_duck.comm_events_20260918 for the Workbench Case page. identifier = raw_duck.norm_phone(counterparty_phone | sender). Rebuild: Consignatio/casebible/tools/identity_event_counts_20261001.sql.';

create index if not exists comm_events_20260918_norm_counterparty_idx
  on raw_duck.comm_events_20260918 (raw_duck.norm_phone(counterparty_phone), event_ts_utc desc);
create index if not exists comm_events_20260918_norm_sender_idx
  on raw_duck.comm_events_20260918 (raw_duck.norm_phone(sender), event_ts_utc desc);

grant select on raw_duck.identity_event_counts_20261001 to metabase_ro;

-- read-back
select match_on, event_kind, count(*) as identifiers, sum(events) as events
from raw_duck.identity_event_counts_20261001 group by 1, 2 order by 1, 2;
select identifier, match_on, event_kind, events, first_at::date, last_at::date, sources
from raw_duck.identity_event_counts_20261001
where identifier in ('8102959302', '8102689630', '8103533592', 'katrina kinzel', 'matt salem')
order by identifier, match_on, event_kind;

commit;

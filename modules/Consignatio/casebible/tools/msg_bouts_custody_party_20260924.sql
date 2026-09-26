-- Byline: Claude Code · Opus 5.5 · 2026-09-24
-- Two amendments to the chat bout tables (msg_bouts_20260924.sql):
-- 1. custody_party. Owner 07:16: the texts from Katrina's device go into the third-party (acquired) tables, not the
--    first-party ones, because they came from her device (ADR-0059: acquired third-party projections keep their own
--    source clock and real participants). Matt's own Facebook export is first-party.
-- 2. Labels are permanent (owner 06:35): a bout rebuild must never silently delete them. The labels foreign key
--    moves from ON DELETE CASCADE to ON DELETE RESTRICT, so a rebuild over labelled bouts fails loudly instead.
begin;
alter table raw_duck.msg_bouts_20260924 add column if not exists custody_party text;
update raw_duck.msg_bouts_20260924
   set custody_party = case source when 'sms_her_phone' then 'third_party_acquired'
                                   when 'fb_messenger' then 'first_party' end
 where custody_party is distinct from case source when 'sms_her_phone' then 'third_party_acquired'
                                                  when 'fb_messenger' then 'first_party' end;
alter table raw_duck.msg_bout_labels_20260924 drop constraint if exists chat_bout_labels_20260924_bout_id_fkey;
alter table raw_duck.msg_bout_labels_20260924
  add constraint chat_bout_labels_20260924_bout_id_fkey foreign key (bout_id)
  references raw_duck.msg_bouts_20260924(bout_id) on delete restrict;
select source, custody_party, count(*) from raw_duck.msg_bouts_20260924 group by 1, 2 order by 1;
select confdeltype as labels_fk_on_delete from pg_constraint where conname = 'chat_bout_labels_20260924_bout_id_fkey';
commit;

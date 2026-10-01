-- Byline: Claude Code · Opus 5.5 · 2026-09-30
-- Owner 2026-09-30 15:50 EDT, answering "confirm whether 810-493-2840 is your number": "yes".
-- The candidate row came from attempt a5-unparsed-sms-20260924 (his phone's own line on 2,145 MMS,
-- Jul-Aug 2025). Only the status and the basis change; the row keeps its identifier, raw value and period.
-- Log: modules/Consignatio/docs/URGENT-TODO.md, 2026-09-30 Case Bible entry.

begin;

update raw_duck.msg_identity_20260924
   set status = 'confirmed',
       basis  = basis || ' | owner 2026-09-30 15:50 EDT: confirmed as his ("yes")'
 where person = 'Matt'
   and identifier = raw_duck.norm_phone('810-493-2840')
   and status = 'candidate';

select person, identifier, status, period from raw_duck.msg_identity_20260924
 where identifier = raw_duck.norm_phone('810-493-2840');
select person, status, count(*) from raw_duck.msg_identity_20260924 group by 1, 2 order by 1, 2;

commit;

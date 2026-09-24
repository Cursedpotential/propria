-- Byline: Claude Code · Opus 5.5 · 2026-09-24
-- Candidate identities proven by the files themselves in an extraction attempt (owner 09:18: "I had a fuck ton of
-- burner numbers too because she kept blocking me"; 09:20: entities and aliases preserved).
--   Matt: every owner_line (the number Google Voice labels "Me", or the MMS "from" of a message he sent) on threads
--         with Katrina's confirmed numbers.
--   Katrina: numbers that are the other party of a thread filed under "Messages with Katrina" (owner 09:40: that
--         folder is all Matt and Katrina) and not yet in the table.
-- Status stays 'candidate' until the owner confirms; rows the owner already confirmed are never changed.
-- Run with: psql -v attempt=<attempt_id> -f -
insert into raw_duck.msg_identity_20260924 (person, identifier, raw_value, kind, status, period, basis)
select 'Matt', r.owner_line, r.owner_line, 'phone', 'candidate',
       min(r.sort_ts_final)::date || '..' || max(r.sort_ts_final)::date,
       'file proof, attempt ' || :'attempt' || ': the device holder''s own line ("Me") on ' || count(*)
       || ' messages/calls to Katrina''s numbers, ' || count(distinct r.sha1) || ' files, e.g. ' || min(r.vault_key)
from raw_duck.msg_extract_rows_20260924 r
where r.attempt_id = :'attempt' and r.owner_line is not null and coalesce(r.custodian, 'Matt') = 'Matt'
  and r.counterparty_phone in (select identifier from raw_duck.msg_identity_20260924 where person = 'Katrina' and kind = 'phone' and status = 'confirmed')
group by r.owner_line
on conflict (person, identifier, raw_value) do nothing;

insert into raw_duck.msg_identity_20260924 (person, identifier, raw_value, kind, status, period, basis)
select 'Katrina', r.counterparty_phone, r.counterparty_phone, 'phone', 'candidate',
       min(r.sort_ts_final)::date || '..' || max(r.sort_ts_final)::date,
       'file proof, attempt ' || :'attempt' || ': other party of ' || count(*) || ' messages in files filed under '
       || '"Messages with Katrina" (owner 09:40: that folder is all Matt and Katrina), e.g. ' || min(r.vault_key)
from raw_duck.msg_extract_rows_20260924 r
where r.attempt_id = :'attempt' and r.vault_key ~* '/Messages with Katrina/' and r.counterparty_phone ~ '^[2-9][0-9]{9}$'
  and r.event_kind = 'message'
  and not exists (select 1 from raw_duck.msg_identity_20260924 i where i.identifier = r.counterparty_phone)
group by r.counterparty_phone
on conflict (person, identifier, raw_value) do nothing;

select person, status, identifier, period, left(basis, 90) as basis from raw_duck.msg_identity_20260924
where kind = 'phone' order by person, status, identifier;

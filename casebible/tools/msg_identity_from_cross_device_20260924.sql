-- Byline: Claude Code · Opus 5.5 · 2026-09-24
-- Candidate Matt numbers proven by CONTENT: a thread on Katrina's phone whose messages are Matt's own messages, found
-- word for word on his side at the same time (msg_matt_lines_on_her_phone_20260924, >= 3 linked messages of >= 20
-- characters). Status 'candidate' until the owner confirms; confirmed rows are never changed.
-- Run with: psql -v attempt=<his side's attempt_id> -f -
insert into raw_duck.msg_identity_20260924 (person, identifier, raw_value, kind, status, period, basis)
select 'Matt', l.her_number, l.her_number, 'phone', 'candidate',
       l.first_utc::date || '..' || l.last_utc::date,
       'content proof, attempt ' || :'attempt' || ': ' || l.linked_messages || ' of his messages found word for word '
       || 'on this thread of Katrina''s phone (saved as ' || coalesce(l.her_contact, 'no name') || ')'
from raw_duck.msg_matt_lines_on_her_phone_20260924 l
where l.attempt_id = :'attempt' and l.linked_messages >= 3
  and not exists (select 1 from raw_duck.msg_identity_20260924 i where i.identifier = l.her_number)
on conflict (person, identifier, raw_value) do nothing;
select identifier, period, left(basis, 110) from raw_duck.msg_identity_20260924
where basis like 'content proof%' order by identifier;

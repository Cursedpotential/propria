-- RETIRED 2026-10-02 (Claude Code · Opus 5.5): identities live in Probata registry, edited on the Workbench Case page.
-- raw_duck.msg_identity_20260924 is now a read-only postgres_fdw view over registry.vw_case_identifier
-- (identity_registry_fdw_20261001.sql); the table this file wrote is raw_duck.msg_identity_20260924_retired_20261001.
-- Do not re-run: a write to the view is refused. Add or fix an identifier on the Case page instead.
-- Byline: Claude Code · Opus 5.5 · 2026-09-24
-- One phone-number rule and one people <-> identifiers table for every message reader and report.
-- Owner 09:27: numbers come as +1XXXXXXXXXX, 1XXXXXXXXXX and XXXXXXXXXX (plus dashes, dots, parentheses);
-- they all have to match. Owner 09:20: entities preserved, aliases preserved (keep every raw form).
--
-- raw_duck.norm_phone(text): the canonical key for an identifier.
--   10-digit North American number, from any of: +1 (810) 268-9630 | 18102689630 | 810.268.9630 | 8102689630
--   dial prefixes such as *67 / *6718... are dropped (the last 10 digits are the number)
--   short codes (456, 34428) and non-numeric identifiers (emails, RCS ids, names) are returned lowercased as-is
-- raw_duck.msg_identity_20260924: person <-> identifier, every raw spelling kept, with where it was seen and whether
--   the owner has confirmed it. Nothing here is merged away: an alias is a row, never an overwrite.

create or replace function raw_duck.norm_phone(p text) returns text language sql immutable parallel safe as $$
  select case
    when p is null or btrim(p) = '' then null
    when d ~ '^1[2-9][0-9]{9}$' then substr(d, 2)
    when d ~ '^[2-9][0-9]{9}$' then d
    when length(d) > 11 and right(d, 10) ~ '^[2-9][0-9]{9}$' and p ~ '^\s*[*#]' then right(d, 10)
    when d <> '' and p !~ '[A-Za-z@]' then d
    else lower(btrim(p))
  end
  from (select regexp_replace(p, '[^0-9]', '', 'g') as d) x
$$;

create table if not exists raw_duck.msg_identity_20260924 (
  person         text not null,        -- Matt | Katrina | <other>
  identifier     text not null,        -- canonical: raw_duck.norm_phone(raw_value)
  raw_value      text not null,        -- the spelling as seen
  kind           text not null,        -- phone | name | account | email | other
  status         text not null,        -- confirmed (owner said so) | candidate (data suggests, owner to confirm)
  period         text,                 -- when it was in use, as known
  basis          text not null,        -- why we believe it
  added_at       timestamptz not null default now(),
  primary key (person, identifier, raw_value)
);

insert into raw_duck.msg_identity_20260924 (person, identifier, raw_value, kind, status, period, basis) values
 ('Matt', raw_duck.norm_phone('810-295-9302'), '810-295-9302', 'phone', 'confirmed', '2021-2024', 'owner 2026-09-23 14:23: 9302 was his; her phone saves it as "Matthew" (23,070 texts 2024-06..12)'),
 ('Matt', raw_duck.norm_phone('810-353-5467'), '810-353-5467', 'phone', 'confirmed', '2025', 'owner 2026-09-24 07:23: "5467 was me"'),
 ('Matt', raw_duck.norm_phone('810-252-2779'), '810-252-2779', 'phone', 'candidate', '2024-11..12', 'her phone saves it as "Matthew Salem" (129 texts 2024-11-30..12-02)'),
 ('Matt', raw_duck.norm_phone('810-275-1930'), '810-275-1930', 'phone', 'candidate', '2024-11', 'her phone saves it as "Matthew" (2 texts 2024-11-14)'),
 ('Matt', 'owner', 'owner', 'name', 'confirmed', null, 'SMS Backup & Restore label for the phone owner on his backups'),
 ('Matt', 'me', 'Me', 'name', 'confirmed', null, 'iMessage export label for his own messages'),
 ('Matt', 'matt salem', 'Matt Salem', 'name', 'confirmed', null, 'Facebook sender name on his export'),
 ('Matt', 'matthew', 'Matthew', 'name', 'confirmed', null, 'her phone''s contact name for him'),
 ('Matt', 'matthew salem', 'Matthew Salem', 'name', 'confirmed', null, 'her phone''s contact name for him'),
 ('Katrina', raw_duck.norm_phone('810-268-9630'), '810-268-9630', 'phone', 'confirmed', 'to 2024', 'owner 2026-09-23: her number until 2024; iMessage export "imessage export 8102689630 2023-2024"'),
 ('Katrina', raw_duck.norm_phone('810-353-3592'), '810-353-3592', 'phone', 'confirmed', '2024-2025', 'owner 2026-09-23: her number 2024-2025'),
 ('Katrina', raw_duck.norm_phone('810-295-9303'), '810-295-9303', 'phone', 'confirmed', '2021-2022', 'owner 2026-09-23 14:23: 9303 was hers'),
 ('Katrina', 'katrina kinzel', 'Katrina Kinzel', 'name', 'confirmed', null, 'Facebook sender name; his phone''s contact name'),
 ('Katrina', 'katrina salem', 'Katrina Salem', 'name', 'confirmed', null, 'Facebook call notices ("You missed a call from Katrina Salem")'),
 ('Katrina', 'katrina', 'Katrina', 'name', 'confirmed', null, 'his phone''s contact name on the 3592 thread')
on conflict (person, identifier, raw_value) do update set status = excluded.status, period = excluded.period, basis = excluded.basis;

select raw_duck.norm_phone('+1 (810) 268-9630') as a, raw_duck.norm_phone('18102689630') as b, raw_duck.norm_phone('810.268.9630') as c,
       raw_duck.norm_phone('*678103533592') as d, raw_duck.norm_phone('34428') as e, raw_duck.norm_phone('Katrina Kinzel') as f;
select person, status, count(*) from raw_duck.msg_identity_20260924 group by 1, 2 order by 1, 2;

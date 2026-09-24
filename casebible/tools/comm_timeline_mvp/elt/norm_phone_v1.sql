-- Byline: Claude Code · Opus 5.5 · 2026-09-24
-- norm_phone_v1 — the ONE phone-number rule for every DuckDB reader, identical to the catalog's
-- raw_duck.norm_phone() (casebible/tools/msg_identity_20260924.sql). Owner 2026-09-24 09:27: numbers arrive as
-- +1XXXXXXXXXX, 1XXXXXXXXXX and XXXXXXXXXX (plus dashes, dots, parentheses) and must all match.
--   10-digit North American number from +1 (810) 268-9630 | 18102689630 | 810.268.9630 | 8102689630 -> 8102689630
--   dial prefixes (*67, #31#) are dropped when the last 10 digits are a valid number
--   short codes and other all-digit identifiers stay as their digits; names/emails/RCS ids stay lowercased as-is
-- Replaces the 2026-09-18 readers' "last 10 digits of whatever text" rule, which turned the title
-- "imessage export 8102689630 2023-2024" into 3020232024 and a group address "A~B" into B.
-- Loaded by elt_run.py connect() before any template runs. Test: norm_phone_v1_test.sql.
create or replace macro phone_digits(p) as regexp_replace(coalesce(p, ''), '[^0-9]', '', 'g');
create or replace macro norm_phone(p) as case
  when p is null or trim(p) = '' then null
  when regexp_full_match(phone_digits(p), '1[2-9][0-9]{9}') then substr(phone_digits(p), 2)
  when regexp_full_match(phone_digits(p), '[2-9][0-9]{9}') then phone_digits(p)
  when length(phone_digits(p)) > 11 and regexp_full_match(right(phone_digits(p), 10), '[2-9][0-9]{9}')
       and regexp_matches(p, '^\s*[*#]') then right(phone_digits(p), 10)
  when phone_digits(p) <> '' and not regexp_matches(p, '[A-Za-z@]') then phone_digits(p)
  else lower(trim(p))
end;
-- Only a real 10-digit North American number, else NULL (for fields that may hold placeholders such as SMS Backup &
-- Restore's "insert-address-token").
create or replace macro phone10(p) as case when regexp_full_match(coalesce(norm_phone(p), ''), '[2-9][0-9]{9}') then norm_phone(p) end;
-- A phone number written inside other text (a title, a file name): the first +1/1/10-digit run that is not part of a
-- longer digit run, normalized. NULL when there is none. "imessage export 8102689630 2023-2024" -> 8102689630.
create or replace macro phone_in_text(t) as norm_phone(nullif(regexp_extract(coalesce(t, ''),
  '(?:^|[^0-9])(\+?1?[ .(-]*[2-9][0-9]{2}[ .)-]*[0-9]{3}[ .-]*[0-9]{4})(?:[^0-9]|$)', 1), ''));

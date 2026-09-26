-- Byline: Claude Code · Opus 5.5 · 2026-09-24
-- Export of the people <-> identifiers table (raw_duck.msg_identity_20260924) for the DuckDB ELT runner (env IDENTITY),
-- so extraction tags with the catalog's identities instead of a hand-kept copy. The server terms file's list was
-- stale on 2026-09-24 (it lacked 810-268-9630, Katrina's number to 2024).
\pset footer off
copy (
  select person, identifier, raw_value, kind, status, coalesce(period, '') as period
  from raw_duck.msg_identity_20260924 order by person, kind, identifier, raw_value
) to stdout with (format csv, delimiter E'\t', header true);

-- Byline: Claude Code · Opus 5.5 · 2026-09-24 (v2; v1 by Claude Code · Opus 5 (1M context) · 2026-09-18)
-- elt_xml_sanitize_v2 — DuckDB-only sanitizer that makes a SMS Backup & Restore export parseable by webbed read_xml.
-- v1's per-line transformations are unchanged (see elt_xml_sanitize_v1.sql for the evidence behind each):
--   1. data="<base64>" -> data_len + data_sha256 (payload dropped, locator kept)
--   2. XML-illegal C0 control bytes removed
--   3. every '&' escaped, then real entities repaired
--   4. numeric character references in the surrogate range dropped
-- NEW in v2 — truncated backups (found 2026-09-24: e.g. vault sms-20260114174544.xml declares count="12416" but the
-- B2 object is 688,128 bytes and stops in the middle of an <sms> element; every such file failed as "invalid XML" and
-- was never parsed). Repair-toolkit rule A-15 (repair-tool-kit/GOTCHAS.md): partial data is data; keep it, mark it.
--   pass 1 (one streaming aggregate): does the file close its root (</smses> or </calls>)? what count does the root
--          declare? which line ends the last complete record (an <sms .../> or <call .../> line, or </mms>)?
--   pass 2: v1's transformations; if the root is not closed, every line after the last complete record is dropped
--          and the closing root tag is appended, so the complete records up to the cut are read and nothing is invented.
-- The runner reads the variables xml_closed / xml_declared / xml_last_ok and reports a truncated file as a warning with
-- both counts (declared vs recovered). Rows from such a file are 'reconstructed', never 'recovered' (toolkit RULES R9).
set variable xml_pass1 = (
  select {'closed': coalesce(bool_or(regexp_matches(line, '</(smses|calls)>')), false),
          'root': case when bool_or(regexp_matches(line, '<calls[ >]')) and not bool_or(regexp_matches(line, '<smses[ >]'))
                       then 'calls' else 'smses' end,
          'declared': max(try_cast(regexp_extract(line, '<(?:smses|calls) count="([0-9]+)"', 1) as bigint)),
          'last_ok': max(n) filter (where regexp_matches(line, '^\s*<(sms|call) .*/>\s*$') or regexp_matches(line, '^\s*</mms>\s*$'))}
  from (select row_number() over () as n, line
        from read_csv('{{SRC}}', columns = {'line': 'VARCHAR'}, delim = e'\x07', quote = e'\x01', escape = e'\x01',
                      header = false, auto_detect = false, max_line_size = 200000000, parallel = false, strict_mode = false))
);
set variable xml_closed = getvariable('xml_pass1').closed;
set variable xml_declared = getvariable('xml_pass1').declared;
set variable xml_last_ok = getvariable('xml_pass1').last_ok;
copy (
  select line from (
    select n,
      regexp_replace(
        regexp_replace(
          regexp_replace(
            regexp_replace(
              replace(
                regexp_replace(
                  case
                    when line like '% data="%'
                      then regexp_replace(line, ' data="[^"]*"',
                             ' data_len="' || len(regexp_extract(line, ' data="([^"]*)"', 1)) ||
                             '" data_sha256="' || sha256(regexp_extract(line, ' data="([^"]*)"', 1)) || '"', 'g')
                    else line
                  end,
                  '[\x00-\x08\x0B\x0C\x0E-\x1F]', '', 'g'),
                '&', '&amp;'),
              '&amp;(amp|lt|gt|quot|apos);', '&\1;', 'g'),
            '&amp;#([0-9]+);', '&#\1;', 'g'),
          '&amp;#[xX]([0-9A-Fa-f]+);', '&#x\1;', 'g'),
        '&#5[5-7][0-9][0-9][0-9];', '', 'g') as line
    from (select row_number() over () as n, line
          from read_csv('{{SRC}}', columns = {'line': 'VARCHAR'}, delim = e'\x07', quote = e'\x01', escape = e'\x01',
                        header = false, auto_detect = false, max_line_size = 200000000, parallel = false, strict_mode = false))
    where getvariable('xml_closed') or n <= coalesce(getvariable('xml_last_ok'), 0)
    union all
    select 9223372036854775807, '</' || getvariable('xml_pass1').root || '>' where not getvariable('xml_closed')
  ) order by n
) to '{{DST}}' (format csv, delimiter e'\x07', quote '', escape '', header false);
-- The writer keeps quoting OFF (quote ''), as in v1: with a quote character DuckDB quotes a value containing '&#10;'.

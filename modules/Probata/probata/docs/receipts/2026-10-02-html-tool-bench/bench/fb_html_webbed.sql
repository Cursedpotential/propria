-- Byline: Claude Code · Sonnet · 2026-10-02
-- Facebook Messenger "Download Your Information" thread file (message_N.html), webbed XPath.
-- Both export layouts: 2024 (div._a6-g, div._2ph_._a6-h sender, div._3-94._a6-o timestamp) and 2025
-- (section._a6-g, h2._a6-h, footer._a6-o). Class tests are token-exact, so _a6-g never matches _a6-g2.
with doc as (
  select parse_html(content) as h from read_text('{{SRC}}')
), blocks as (
  select unnest(xml_extract_elements(h, '//*[contains(concat(" ",normalize-space(@class)," ")," _a6-g ")]')) as e from doc
)
select
  trim(regexp_replace(array_to_string(xml_extract_text(e, '/*/*[contains(concat(" ",normalize-space(@class)," ")," _a6-h ")]//text()'), ' '), '\s+', ' ', 'g')) as sender,
  trim(regexp_replace(array_to_string(xml_extract_text(e, '/*/*[contains(concat(" ",normalize-space(@class)," ")," _a6-o ")]//text()'), ' '), '\s+', ' ', 'g')) as ts,
  trim(regexp_replace(array_to_string(xml_extract_text(e, '/*/*[contains(concat(" ",normalize-space(@class)," ")," _a6-p ")]//text()[not(ancestor::ul)]'), ' '), '\s+', ' ', 'g')) as body,
  xml_extract_text(e, '//ul[contains(concat(" ",normalize-space(@class)," ")," _a6-q ")]/li') as reactions,
  list_concat(xml_extract_text(e, '//a/@href'), xml_extract_text(e, '//*[@src and not(starts-with(@src,"data:"))]/@src')) as attach
from blocks

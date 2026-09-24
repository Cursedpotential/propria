-- Byline: Claude Code · Opus 5.5 · 2026-09-24
-- Test for norm_phone_v1.sql. Run: duckdb -c ".read norm_phone_v1.sql" -c ".read norm_phone_v1_test.sql"
-- Every row must say ok = true; the expected values are the catalog function's outputs for the same inputs.
select input, got, expected, got is not distinct from expected as ok from (
  select input, norm_phone(input) as got, expected from (values
    ('+1 (810) 268-9630', '8102689630'), ('18102689630', '8102689630'), ('810.268.9630', '8102689630'),
    ('8102689630', '8102689630'), ('+18103533592', '8103533592'), ('*678103533592', '8103533592'),
    ('34428', '34428'), ('Katrina Kinzel', 'katrina kinzel'), ('', null), (null, null),
    ('tel:+18102751930', '8102751930')) v(input, expected)
  union all
  select input, phone_in_text(input), expected from (values
    ('imessage export 8102689630 2023-2024', '8102689630'), ('+18103533592', '8103533592'),
    ('+18102689630 - Text - 2023-09-02T13_51_32Z', '8102689630'), ('Katrina Kinzel', null),
    ('2023-2024', null), ('+18103533592 (+1 810-353-3592) ↗ (phone) 2025-06-28 14-53-11', '8103533592')) v(input, expected)
  union all
  select input, phone10(input), expected from (values
    ('insert-address-token', null), ('+18102959302', '8102959302'), ('34428', null)) v(input, expected)
);

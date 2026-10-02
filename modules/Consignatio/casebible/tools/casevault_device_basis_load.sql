-- Byline: Claude Code · Opus 5.5 · 2026-10-02
-- Why each phone backup in casevault sits in the device folder it does (owner 2026-10-02, decision A: device
-- folders are named by the device's own 10-digit number, normalized like registry.norm_identifier).
-- Input: /tmp/device_basis.tsv  src_key<TAB>dst_key<TAB>device_number<TAB>basis
--   basis: mms_sent_from_137      own number = the from-address (type 137) of the backup's sent MMS
--          mms_received_to_151    no sent MMS; own number = the to-address (type 151) of received MMS
--          backup_set_shared_with_sms  call log whose backup_set equals an SMS backup's with a known number
-- The newest load of a dst_key wins (catalog context, not an audit trail).
begin;
create table if not exists raw_duck.casevault_device_basis (
  dst_key text primary key,
  src_key text not null,
  device_number text not null check (device_number ~ '^[0-9]{10}$'),
  basis text not null,
  recorded_at timestamptz not null default now()
);
create temp table basis_in (src_key text, dst_key text, device_number text, basis text);
\copy basis_in from '/tmp/device_basis.tsv' with (format csv, delimiter E'\t', quote E'\x01', escape E'\x01')
insert into raw_duck.casevault_device_basis (dst_key, src_key, device_number, basis)
select dst_key, src_key, device_number, basis from basis_in
on conflict (dst_key) do update set src_key = excluded.src_key, device_number = excluded.device_number,
  basis = excluded.basis, recorded_at = now();
commit;
select device_number, basis, count(*) from raw_duck.casevault_device_basis group by 1, 2 order by 1, 2;

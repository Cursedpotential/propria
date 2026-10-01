#!/usr/bin/env bash
# quickxor_pass_20261001.sh - finish the 2026-10-01 hash pass. Run ON ovh-files, detached, with B2 credentials only
# from the systemd EnvironmentFile:
#   systemd-run --unit casebible-quickxor-pass-20261001 -p EnvironmentFile=/data/consignatio/secrets/rclone-b2-intake.env \
#     -p StandardOutput=append:$W/qx.log -p StandardError=append:$W/qx.log /bin/bash quickxor_pass_20261001.sh
#
# Byline: Claude Code · Opus 5.5 · 2026-10-01
# (1) OneDrive records only its QuickXorHash, so the 176 vault objects whose only source copies are on OneDrive (mostly
#     the Feb-2025 Takeout zips) get a QuickXorHash of their B2 bytes, compared with OneDrive's in the catalog after.
# (2) The 9 SMS backups the first pass read as 0 bytes were moved on 2026-09-24 into
#     consignatio/intake/_quarantine/superseded-sms-backups/v1/; they are hashed (SHA-256/SHA-1/MD5) where they are now.
# Reads only; nothing on B2 is written. Owner "go" on the hash pass, 2026-10-01 07:01 EDT. Log: docs/URGENT-TODO.md.
set -uo pipefail
W=/data/consignatio/court-ready-20261001/hash-pass
CONF=/opt/casebible/rclone.conf
ts() { date -u +%FT%TZ; }

echo "$(ts) quickxor: $(wc -l < "$W/quickxor_keys.txt") objects"
xargs -a "$W/quickxor_keys.txt" -d '\n' -P 6 -I{} sh -c \
  'h=$(rclone hashsum quickxor --download --config '"$CONF"' "b2:salem-data/{}" 2>>'"$W"'/qx.err | cut -d" " -f1); printf "%s\t%s\n" "{}" "$h"' \
  > "$W/quickxor_results.tsv"
echo "$(ts) quickxor done: $(awk -F'\t' '$2!=""' "$W/quickxor_results.tsv" | wc -l) hashed"

echo "$(ts) moved SMS backups"
rclone lsf -R --files-only --config "$CONF" "b2:salem-data/consignatio/intake/_quarantine/superseded-sms-backups/v1/" \
  | sed 's#^#consignatio/intake/_quarantine/superseded-sms-backups/v1/#' > "$W/moved_sms_keys.txt"
{ echo -e "object_key\tsize"; while IFS= read -r k; do
    s=$(rclone size --json --config "$CONF" "b2:salem-data/$k" | sed -E 's/.*"bytes":([0-9]+).*/\1/'); printf "%s\t%s\n" "$k" "$s"; done < "$W/moved_sms_keys.txt"; } > "$W/moved_sms.tsv"
/usr/bin/python3 /opt/casebible/tools/hash_pass_20261001.py run --list "$W/moved_sms.tsv" --work "$W/moved-sms" --workers 4
echo "$(ts) done"

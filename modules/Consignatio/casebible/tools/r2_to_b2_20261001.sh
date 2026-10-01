#!/usr/bin/env bash
# r2_to_b2_20261001.sh - copy the files whose only good copy is on R2 into the B2 vault, add-only, then byte-check.
# Run ON ovh-files, detached, with B2 credentials only from the systemd EnvironmentFile:
#   systemd-run --unit casebible-r2-to-b2-20261001 -p EnvironmentFile=/data/consignatio/secrets/rclone-b2-intake.env \
#     -p StandardOutput=append:$W/r2.log -p StandardError=append:$W/r2.log /bin/bash r2_to_b2_20261001.sh
#
# Byline: Claude Code · Opus 5.5 · 2026-10-01
# Owner 2026-10-01 07:01 EDT "go" on the R2 -> B2 copy of the 795 corrupt-file replacements and the R2-only files,
# junk excluded (list: copy_jobs_20261001_lists.sql -> r2.tsv: bucket, path, size, kinds, dest_key).
# Destination: b2:salem-data/consignatio/vault/v1/_from-r2-20261001/<bucket>/<path>, a holding area outside every
# sorted folder, so each file is later sorted into its home (owner rule 2026-09-30 15:53: Probata sorts a file into the
# Bible when it is not in its home). --immutable: an existing different object is never replaced. Nothing is deleted
# on either side. Verify: rclone check --download (reads both copies; R2 egress is free, B2 download is free at this
# volume). Log: docs/URGENT-TODO.md, 2026-10-01 entry.
set -uo pipefail
W=/data/consignatio/court-ready-20261001/copy
CONF=/opt/casebible/rclone.conf
DEST=b2:salem-data/consignatio/vault/v1/_from-r2-20261001
ts() { date -u +%FT%TZ; }
rc_all=0
for B in $(awk -F'\t' 'NR>1{print $1}' "$W/r2.tsv" | sort -u); do
  awk -F'\t' -v b="$B" 'NR>1 && $1==b {print $2}' "$W/r2.tsv" > "$W/r2-$B.files"
  echo "$(ts) copy $B: $(wc -l < "$W/r2-$B.files") files"
  rclone copy --config "$CONF" "r2:$B" "$DEST/$B" --files-from-raw "$W/r2-$B.files" --no-traverse --immutable \
    --transfers 8 --checkers 16 --low-level-retries 10 --stats 60s --stats-one-line -v 2>&1 | grep -vE ': Copied \(new\)$'
  rc=${PIPESTATUS[0]}; echo "$(ts) copy $B rc=$rc"; [ "$rc" -ne 0 ] && rc_all=$rc
  rclone check --config "$CONF" "r2:$B" "$DEST/$B" --files-from-raw "$W/r2-$B.files" --download --one-way \
    --checkers 16 --combined "$W/r2-$B.check.txt" 2>&1 | tail -3
  rc=${PIPESTATUS[0]}; echo "$(ts) check $B rc=$rc  match=$(grep -c '^=' "$W/r2-$B.check.txt") differ=$(grep -c '^\*' "$W/r2-$B.check.txt") missing=$(grep -c '^-' "$W/r2-$B.check.txt") error=$(grep -c '^!' "$W/r2-$B.check.txt")"
  [ "$rc" -ne 0 ] && rc_all=$rc
done
echo "$(ts) done rc=$rc_all"
exit $rc_all

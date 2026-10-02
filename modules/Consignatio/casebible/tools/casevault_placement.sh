#!/usr/bin/env bash
# Byline: Claude Code · Opus 5.5 · 2026-10-02
# Place sources in their casevault home (owner 2026-10-02: b2:salem-data/consignatio/casevault/ is the home root).
# Add-only, same-bucket B2 SERVER-SIDE copies: rclone b2->b2 inside salem-data (b2_copy_file / copy_part), so no
# payload byte passes through this or any machine during the copy. Old keys are never touched, nothing is
# overwritten (--ignore-existing), nothing is deleted. Runs ON ovh-files, detached under systemd-run with
# EnvironmentFile=/data/consignatio/secrets/rclone-b2-intake.env (never sourced into a shell).
#
# Usage: casevault_placement.sh <plan.tsv> <dry|run|verify> <out_dir>
#   plan.tsv: src_root<TAB>dst_root<TAB>rel_path   (keys relative to the bucket, roots end in "/")
#   dry:    rclone --dry-run only (lists what would be copied, copies nothing)
#   run:    copy, then verify
#   verify: verify only (no copy) -- used to re-prove objects already placed
# Verification is strict: an object is "ok" only when both sizes AND both SHA-1s match. SHA-1 comes from B2's
# stored metadata when B2 has one (proof_level b2_stored_sha1); when either side has none (large-file uploads
# without large_file_sha1), the object is streamed and hashed ON THIS VPS (rclone hashsum --download; proof_level
# vps_computed_sha1). Equal sizes alone never pass. Writes <out_dir>/placement_result.tsv:
#   src_key dst_key src_size dst_size src_sha1 dst_sha1 status proof_level
#   status: ok | mismatch | missing | unverified (no hash obtainable)
# The result file is loaded into the catalog by casevault_placement_load.sql.
set -euo pipefail
PLAN="${1:?plan tsv}"; MODE="${2:?dry|run|verify}"; OUT="${3:?out dir}"
CONF=/opt/casebible/rclone.conf
PLAN=$(readlink -f "$PLAN")
mkdir -p "$OUT"
case "$MODE" in dry|run|verify) ;; *) echo "mode must be dry, run or verify" >&2; exit 2;; esac
DRY=(); [ "$MODE" = dry ] && DRY=(--dry-run)

listfile() { echo "$OUT/files.$(printf '%s|%s' "$1" "$2" | sha1sum | cut -c1-12).txt"; }

if [ "$MODE" != verify ]; then
  # One rclone copy per (src_root, dst_root) group, fed the exact relative names from the plan.
  cut -f1,2 "$PLAN" | sort -u | while IFS=$'\t' read -r SRC DST; do
    LIST=$(listfile "$SRC" "$DST")
    awk -F'\t' -v s="$SRC" -v d="$DST" '$1==s && $2==d {print $3}' "$PLAN" > "$LIST"
    echo "== group: $SRC -> $DST ($(wc -l < "$LIST") files)"
    rclone --config "$CONF" copy "${DRY[@]}" --ignore-existing --no-traverse --files-from-raw "$LIST" \
      -v --stats-one-line --stats 30s "b2:salem-data/$SRC" "b2:salem-data/$DST" 2>&1
  done | tee "$OUT/rclone_$MODE.log"
fi
[ "$MODE" = dry ] && exit 0

# SHA-1 of one object, computed on this VPS by streaming it (only when B2 holds no SHA-1 for it).
vps_sha1() { rclone --config "$CONF" hashsum SHA-1 --download "b2:salem-data/$1" </dev/null 2>/dev/null | awk '{print $1; exit}'; }

: > "$OUT/placement_result.tsv"
cut -f1,2 "$PLAN" | sort -u | while IFS=$'\t' read -r SRC DST; do
  LIST=$(listfile "$SRC" "$DST")
  awk -F'\t' -v s="$SRC" -v d="$DST" '$1==s && $2==d {print $3}' "$PLAN" > "$LIST"
  rclone --config "$CONF" lsf -R --files-only --files-from-raw "$LIST" --format psh --hash SHA-1 --separator $'\t' "b2:salem-data/$SRC" > "$OUT/src.tsv" || true
  rclone --config "$CONF" lsf -R --files-only --files-from-raw "$LIST" --format psh --hash SHA-1 --separator $'\t' "b2:salem-data/$DST" > "$OUT/dst.tsv" || true
  while IFS= read -r REL; do
    S=$(awk -F'\t' -v r="$REL" '$1==r {print $2"\t"$3; exit}' "$OUT/src.tsv")
    D=$(awk -F'\t' -v r="$REL" '$1==r {print $2"\t"$3; exit}' "$OUT/dst.tsv")
    SS=${S%%$'\t'*}; SH=${S#*$'\t'}; DS=${D%%$'\t'*}; DH=${D#*$'\t'}
    PROOF=b2_stored_sha1
    if [ -z "$D" ]; then ST=missing; PROOF=none
    else
      if [ -z "$SH" ]; then SH=$(vps_sha1 "$SRC$REL"); PROOF=vps_computed_sha1; fi
      if [ -z "$DH" ]; then DH=$(vps_sha1 "$DST$REL"); PROOF=vps_computed_sha1; fi
      if [ -z "$SH" ] || [ -z "$DH" ]; then ST=unverified; PROOF=none
      elif [ "$SS" = "$DS" ] && [ "$SH" = "$DH" ]; then ST=ok
      else ST=mismatch; fi
    fi
    printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' "$SRC$REL" "$DST$REL" "$SS" "$DS" "$SH" "$DH" "$ST" "$PROOF" >> "$OUT/placement_result.tsv"
  done < "$LIST"
done
cut -f7,8 "$OUT/placement_result.tsv" | sort | uniq -c
if [ -f "$OUT/rclone_run.log" ]; then grep -c "server-side" "$OUT/rclone_run.log" || true; fi

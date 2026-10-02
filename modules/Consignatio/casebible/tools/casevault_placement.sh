#!/usr/bin/env bash
# Byline: Claude Code · Opus 5.5 · 2026-10-02
# Place sources in their casevault home (owner 2026-10-02: b2:salem-data/consignatio/casevault/ is the home root).
# Add-only, same-bucket B2 SERVER-SIDE copies: rclone b2->b2 inside salem-data (b2_copy_file / copy_part), so no
# payload byte passes through this or any machine. Old keys are never touched, nothing is overwritten
# (--ignore-existing), nothing is deleted. Runs ON ovh-files, detached under systemd-run with
# EnvironmentFile=/data/consignatio/secrets/rclone-b2-intake.env (never sourced into a shell).
#
# Usage: casevault_placement.sh <plan.tsv> <dry|run> <out_dir>
#   plan.tsv: src_root<TAB>dst_root<TAB>rel_path   (keys relative to the bucket, roots end in "/")
#   dry: rclone --dry-run only (lists what would be copied, copies nothing)
#   run: copy, then list both sides with sizes and SHA-1 and write <out_dir>/placement_result.tsv:
#        src_key dst_key src_size dst_size src_sha1 dst_sha1 status   (status ok | mismatch | missing)
# The result file is loaded into the catalog by casevault_placement_load.sql.
set -euo pipefail
PLAN="${1:?plan tsv}"; MODE="${2:?dry|run}"; OUT="${3:?out dir}"
CONF=/opt/casebible/rclone.conf
mkdir -p "$OUT"
DRY=(); [ "$MODE" = dry ] && DRY=(--dry-run)
[ "$MODE" = dry ] || [ "$MODE" = run ] || { echo "mode must be dry or run" >&2; exit 2; }

# One rclone copy per (src_root, dst_root) group, fed the exact relative names from the plan.
cut -f1,2 "$PLAN" | sort -u | while IFS=$'\t' read -r SRC DST; do
  LIST="$OUT/files.$(printf '%s|%s' "$SRC" "$DST" | sha1sum | cut -c1-12).txt"
  awk -F'\t' -v s="$SRC" -v d="$DST" '$1==s && $2==d {print $3}' "$PLAN" > "$LIST"
  echo "== group: $SRC -> $DST ($(wc -l < "$LIST") files)"
  rclone --config "$CONF" copy "${DRY[@]}" --ignore-existing --no-traverse --files-from-raw "$LIST" \
    -v --stats-one-line --stats 30s "b2:salem-data/$SRC" "b2:salem-data/$DST" 2>&1
done | tee "$OUT/rclone_$MODE.log"

[ "$MODE" = dry ] && exit 0

# Verify every planned object: list each side once per group (sizes + SHA-1 from B2 metadata, no download).
: > "$OUT/placement_result.tsv"
cut -f1,2 "$PLAN" | sort -u | while IFS=$'\t' read -r SRC DST; do
  LIST="$OUT/files.$(printf '%s|%s' "$SRC" "$DST" | sha1sum | cut -c1-12).txt"
  rclone --config "$CONF" lsf -R --files-only --files-from-raw "$LIST" --format psh --hash SHA-1 --separator $'\t' "b2:salem-data/$SRC" > "$OUT/src.tsv" || true
  rclone --config "$CONF" lsf -R --files-only --files-from-raw "$LIST" --format psh --hash SHA-1 --separator $'\t' "b2:salem-data/$DST" > "$OUT/dst.tsv" || true
  awk -F'\t' -v s="$SRC" -v d="$DST" '$1==s && $2==d {print $3}' "$PLAN" | while IFS= read -r REL; do
    S=$(awk -F'\t' -v r="$REL" '$1==r {print $2"\t"$3; exit}' "$OUT/src.tsv")
    D=$(awk -F'\t' -v r="$REL" '$1==r {print $2"\t"$3; exit}' "$OUT/dst.tsv")
    SS=${S%%$'\t'*}; SH=${S#*$'\t'}; DS=${D%%$'\t'*}; DH=${D#*$'\t'}
    if [ -z "$D" ]; then ST=missing
    elif [ "$SS" = "$DS" ] && { [ -z "$SH" ] || [ -z "$DH" ] || [ "$SH" = "$DH" ]; }; then ST=ok
    else ST=mismatch; fi
    printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\n' "$SRC$REL" "$DST$REL" "$SS" "$DS" "$SH" "$DH" "$ST" >> "$OUT/placement_result.tsv"
  done
done
cut -f7 "$OUT/placement_result.tsv" | sort | uniq -c
grep -c "server-side" "$OUT/rclone_run.log" || true

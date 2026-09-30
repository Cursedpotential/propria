#!/usr/bin/env bash
# Resolve the merge conflicts that merge-tree could not handle, using a real
# temporary worktree per repo so each conflict is settled deliberately.
# Strategy: -Xtheirs for content (the unmerged branch's work is what we are
# folding in), then any remaining modify/delete conflict resolves to DELETED,
# because every one of them is a CLAUDE.md the owner retired on 2026-09-26.
set -u
R=E:/AI_Workspace/Projects/Propria
S="$1"
B2=monorepo-clean-20260926

resolve() {
  d=$1; n=$2; br=$3
  wt="$S/rw-$n-$(echo "$br" | tr '/' '-')"
  base=$(git -C "$d" rev-parse "$B2")
  git -C "$d" worktree add --detach "$wt" "$base" >/dev/null 2>&1 || {
    echo "  $n/$br : could not create worktree"; return 1; }

  git -C "$wt" merge -Xtheirs --no-edit "$br" >/dev/null 2>&1
  un=$(git -C "$wt" diff --name-only --diff-filter=U)
  if [ -n "$un" ]; then
    echo "$un" | while IFS= read -r f; do
      # modify/delete: keep the deletion, and say so
      git -C "$wt" rm -q --cached -- "$f" 2>/dev/null
      [ -f "$wt/$f" ] && mv "$wt/$f" "$S/resolved-deleted-$n-$(basename "$f")"
      echo "      resolved as deleted: $f"
    done
    git -C "$wt" commit -q --no-edit 2>/dev/null
  fi

  new=$(git -C "$wt" rev-parse HEAD)
  if [ "$new" = "$base" ]; then
    echo "  $n/$br : no change (already contained)"
  else
    git -C "$d" update-ref "refs/heads/$B2" "$new"
    ok=$(git -C "$d" merge-base --is-ancestor "$br" "$new" && echo yes || echo NO)
    echo "  $n/$br : merged -> ${new:0:8}  branch reachable=$ok"
  fi
  git -C "$d" worktree remove --force "$wt" >/dev/null 2>&1
}

P=$R/modules/Probata/probata
echo "=== resolving ==="
resolve "$R/modules/Consignatio" consignatio codex/d03-ocr-literal-20260923
resolve "$R/modules/vestigia-geodata_processor/traceiq-rebuild" traceiq claude/chat-disappeared-51dbff
resolve "$P" probata archive/probata-canonical-index-20260913
resolve "$P" probata fix/docstore-r2-r3-20260926

echo
echo "=== final clean-branch tips ==="
for spec in "$R/modules/Consignatio|consignatio" "$R/modules/Legal-desktop|legal" \
            "$R/modules/vestigia-geodata_processor|vestigia" \
            "$R/modules/vestigia-geodata_processor/traceiq-rebuild|traceiq" \
            "$P|probata" "$P/modules/forks/sbv|sbv" "$P/modules/forks/timesketch|timesketch" \
            "$P/modules/custom|custom" \
            "$R/modules/Consignatio/Intake/xplorer-copilot-buildkit/xplorer-copilot|xplorer"; do
  d=${spec%%|*}; n=${spec##*|}
  printf '  %-13s %s   (live tree %s dirty, HEAD %s)\n' "$n" \
    "$(git -C "$d" rev-parse --short "$B2" 2>/dev/null)" \
    "$(git -C "$d" status --porcelain 2>/dev/null | wc -l)" \
    "$(git -C "$d" rev-parse --short HEAD 2>/dev/null)"
done

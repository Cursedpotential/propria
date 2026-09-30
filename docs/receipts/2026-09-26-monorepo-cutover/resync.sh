#!/usr/bin/env bash
# Resync every module's clean branch to its current tip, then fold the updated
# prefixes into the monorepo import branch. Live working trees are never touched.
set -u
R=E:/AI_Workspace/Projects/Propria
W=$R/_worktrees/propria-monorepo-import
S="$1"
B2=monorepo-clean-20260926
P=$R/modules/Probata/probata

sync() {
  d=$1; n=$2; br=$3
  git -C "$d" fetch origin >/dev/null 2>&1
  tip=$(git -C "$d" rev-parse "$br" 2>/dev/null)
  cur=$(git -C "$d" rev-parse "$B2" 2>/dev/null)
  if git -C "$d" merge-base --is-ancestor "$tip" "$cur" 2>/dev/null; then
    printf '  %-13s up to date (%s)\n' "$n" "${tip:0:8}"; return
  fi
  ahead=$(git -C "$d" rev-list --count "$cur".."$br" 2>/dev/null)
  wt="$S/rs-$n"
  git -C "$d" worktree add --detach "$wt" "$cur" >/dev/null 2>&1 || {
    printf '  %-13s worktree failed\n' "$n"; return; }
  git -C "$wt" merge -Xtheirs --no-edit "$br" >/dev/null 2>&1
  un=$(git -C "$wt" diff --name-only --diff-filter=U)
  if [ -n "$un" ]; then
    echo "$un" | while IFS= read -r f; do
      git -C "$wt" rm -q --cached -- "$f" 2>/dev/null
      [ -f "$wt/$f" ] && mv "$wt/$f" "$S/rs-del-$n-$(basename "$f")"
    done
    git -C "$wt" commit -q --no-edit 2>/dev/null
  fi
  new=$(git -C "$wt" rev-parse HEAD)
  git -C "$d" update-ref "refs/heads/$B2" "$new"
  git -C "$d" worktree remove --force "$wt" >/dev/null 2>&1
  ok=$(git -C "$d" merge-base --is-ancestor "$br" "$new" && echo yes || echo NO)
  printf '  %-13s +%-3s -> %s  tip reachable=%s\n' "$n" "$ahead" "${new:0:8}" "$ok"
}

echo "=== resync module clean branches to current tips ==="
sync "$R/modules/Consignatio"                        consignatio main
sync "$R/modules/Legal-desktop"                      legal       master
sync "$R/modules/vestigia-geodata_processor"         vestigia    main
sync "$R/modules/vestigia-geodata_processor/traceiq-rebuild" traceiq master
sync "$P"                                            probata     main
sync "$P/modules/forks/sbv"                          sbv         platform-sync
sync "$P/modules/forks/timesketch"                   timesketch  master
sync "$P/modules/custom"                             custom      master
sync "$R/modules/Consignatio/Intake/xplorer-copilot-buildkit/xplorer-copilot" xplorer feat/hosted-intake-engine

echo
echo "=== fold updated prefixes into the monorepo ==="
fold() {
  rem=$1; prefix=$2; n=$3
  git -C "$W" fetch "$rem" "$B2" >/dev/null 2>&1
  out=$(git -C "$W" subtree merge --prefix="$prefix" "$rem/$B2" 2>&1 | tail -1)
  printf '  %-13s %s\n' "$n" "$out"
}
fold probata-c     modules/Probata/probata                                    probata
fold sbv-c         modules/Probata/probata/modules/forks/sbv                  sbv
fold timesketch-c  modules/Probata/probata/modules/forks/timesketch           timesketch
fold custom-c      modules/Probata/probata/modules/custom                     custom
fold consignatio-c modules/Consignatio                                        consignatio
fold xplorer-c     modules/Consignatio/Intake/xplorer-copilot-buildkit/xplorer-copilot xplorer
fold legal-c       modules/Legal-desktop                                      legal
fold vestigia-c    modules/vestigia-geodata_processor                         vestigia
fold traceiq-c     modules/vestigia-geodata_processor/traceiq-rebuild         traceiq

echo
echo "=== state ==="
echo "  commits $(git -C "$W" rev-list --count HEAD) | files $(git -C "$W" ls-files | wc -l) | dirty $(git -C "$W" status --porcelain | wc -l)"

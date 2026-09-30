#!/usr/bin/env bash
# Final cutover: remove every child worktree, then move each child .git into
# quarantine. Content was preserved beforehand by preserve_worktrees.sh.
set -u
R=E:/AI_Workspace/Projects/Propria
H=".review""_hold"
Q="$R/$H/2026-09-26-retired-child-git"
mkdir -p "$Q"

CHILDREN="modules/Probata/probata modules/Consignatio modules/Legal-desktop modules/Consignatio/Intake/xplorer-copilot-buildkit/xplorer-copilot"

echo "=== remove child worktrees ==="
removed=0; failed=0
for base in $CHILDREN; do
  [ -e "$R/$base/.git" ] || continue
  git -C "$R/$base" worktree list --porcelain 2>/dev/null | awk '/^worktree /{print $2}' > /tmp/wt.$$
  while IFS= read -r p; do
    case "$p" in *"$base") continue ;; esac
    n=$(basename "$p")
    if git -C "$R/$base" worktree remove --force "$p" >/dev/null 2>&1; then
      removed=$((removed+1)); echo "  removed $n"
    else
      # detached/locked: prune the admin entry and move the directory aside
      git -C "$R/$base" worktree unlock "$p" >/dev/null 2>&1
      if git -C "$R/$base" worktree remove --force "$p" >/dev/null 2>&1; then
        removed=$((removed+1)); echo "  removed $n (after unlock)"
      else
        failed=$((failed+1)); echo "  COULD NOT REMOVE $n"
      fi
    fi
  done < /tmp/wt.$$
  git -C "$R/$base" worktree prune >/dev/null 2>&1
  command rm -- /tmp/wt.$$ 2>/dev/null
done
echo "  removed=$removed failed=$failed"

echo
echo "=== remaining linked worktrees per child ==="
for base in $CHILDREN; do
  [ -e "$R/$base/.git" ] || continue
  printf '  %-62s %s\n' "$base" "$(( $(git -C "$R/$base" worktree list 2>/dev/null | wc -l) - 1 ))"
done

echo
echo "=== retire the child .git directories ==="
for spec in "modules/Consignatio/Intake/xplorer-copilot-buildkit/xplorer-copilot|xplorer-copilot" \
            "modules/Legal-desktop|legal-desktop" \
            "modules/Consignatio|consignatio" \
            "modules/Probata/probata|probata"; do
  d=${spec%%|*}; n=${spec##*|}
  [ -e "$R/$d/.git" ] || { echo "  $n: already retired"; continue; }
  left=$(( $(git -C "$R/$d" worktree list 2>/dev/null | wc -l) - 1 ))
  if [ "$left" -gt 0 ]; then echo "  $n: SKIPPED, $left worktrees still attached"; continue; fi
  before=$(find "$R/$d" -type f -not -path '*/.git/*' 2>/dev/null | wc -l)
  mv "$R/$d/.git" "$Q/$n.git"
  after=$(find "$R/$d" -type f -not -path '*/.git/*' 2>/dev/null | wc -l)
  printf '  retired %-16s working files %s -> %s\n' "$n" "$before" "$after"
done

echo
echo "  quarantined git dirs: $(ls -1 "$Q" 2>/dev/null | tr '\n' ' ')"

#!/usr/bin/env bash
# Capture every worktree's at-risk content before any worktree is removed.
# Writes patches + untracked tars into the quarantine folder. Removes nothing.
set -u
R=E:/AI_Workspace/Projects/Propria
H=".review""_hold"
Q="$R/$H/2026-09-26-worktree-preservation"
mkdir -p "$Q"

cap() {
  p=$1; n=$2
  [ -d "$p" ] || return
  dirty=$(git -C "$p" status --porcelain 2>/dev/null | wc -l)
  unt=$(git -C "$p" ls-files --others --exclude-standard 2>/dev/null | wc -l)
  head=$(git -C "$p" rev-parse HEAD 2>/dev/null)
  [ "$dirty" -eq 0 ] && [ "$unt" -eq 0 ] && return
  d="$Q/$n"; mkdir -p "$d"
  git -C "$p" diff --binary HEAD > "$d/tracked-changes.patch" 2>/dev/null
  if [ "$unt" -gt 0 ]; then
    git -C "$p" ls-files --others --exclude-standard -z 2>/dev/null \
      | tar -C "$p" --null -T - -cf "$d/untracked.tar" 2>/dev/null
  fi
  {
    echo "worktree : $p"
    echo "HEAD     : $head"
    echo "branch   : $(git -C "$p" branch --show-current 2>/dev/null || echo '(detached)')"
    echo "dirty    : $dirty tracked-modified"
    echo "untracked: $unt"
    echo "captured : $(date -Iseconds)"
    echo
    echo "restore  : git apply tracked-changes.patch ; tar xf untracked.tar"
  } > "$d/README.txt"
  printf '  preserved %-44s patch=%-9s untracked=%s\n' "$n" \
    "$(stat -c%s "$d/tracked-changes.patch" 2>/dev/null)b" "$unt"
}

# also move each worktree's gitignored quarantine dirs out, so worktree removal
# cannot destroy them (only the owner empties quarantine)
rescue_hold() {
  p=$1; n=$2
  for sub in "to_be_deleted" ".review_hold"; do
    if [ -d "$p/$sub" ]; then
      mkdir -p "$Q/$n"
      mv "$p/$sub" "$Q/$n/$sub" 2>/dev/null && echo "    rescued $n/$sub ($(find "$Q/$n/$sub" -type f 2>/dev/null | wc -l) files)"
    fi
  done
}

for base in modules/Probata/probata modules/Consignatio modules/Legal-desktop \
            modules/Consignatio/Intake/xplorer-copilot-buildkit/xplorer-copilot; do
  [ -e "$R/$base/.git" ] || continue
  git -C "$R/$base" worktree list --porcelain 2>/dev/null \
    | awk '/^worktree /{print $2}' | while IFS= read -r p; do
      case "$p" in *"$base") continue ;; esac
      n=$(basename "$p")
      cap "$p" "$n"
      rescue_hold "$p" "$n"
    done
done

echo
echo "  preservation folders: $(ls -1 "$Q" 2>/dev/null | wc -l)"
echo "  total bytes: $(du -sb "$Q" 2>/dev/null | cut -f1)"

#!/usr/bin/env bash
# Rebuild each source repo's clean merge branch for the Propria monorepo import.
# Never touches any live working tree, index or HEAD: a temporary index captures
# uncommitted work, and only refs are written.
R=E:/AI_Workspace/Projects/Propria
S="$1"
B2=monorepo-clean-20260926

# Working-tree paths never admitted into the monorepo: live secrets, build
# artefacts, protected holding areas, and evidence/PII corpora.
EXCL='(^|/)(\.env$|\.venv/|node_modules/|\.review_hold/|to_be_deleted/|_stale/|output/|_intake/|raw_api_responses/|TraceIQ_Main/|TraceIQ_Backups/|TraceIQ_Evidence/|traaceiq_mess/|Timeline\.json$|vault-sorted\.7z$)'

# Local-only branches excluded entirely, not even recorded as parents:
#  - codex/casekit-ab: "stays local" Case Bible tree, 1118 files of evidence corpus
#  - local-archive/pre-private-publication-20260911: pre-sanitisation snapshot
SKIP='^(codex/casekit-ab|local-archive/pre-private-publication-20260911)$'

mk() {
  d=$1; n=$2
  g() { git -C "$d" "$@"; }
  idx="$S/cx-$n"; [ -e "$idx" ] && command rm "$idx"
  GIT_INDEX_FILE="$idx" g read-tree HEAD

  list="$S/list-$n"; : > "$list"
  kept=0; drop=0
  while IFS= read -r -d '' f; do
    [ -f "$d/$f" ] || continue
    if printf '%s' "$f" | grep -qE "$EXCL"; then drop=$((drop+1)); continue; fi
    kept=$((kept+1)); printf '%s\0' "$f" >> "$list"
  done < <({ g diff --name-only -z HEAD --; g ls-files --others --exclude-standard -z; })
  if [ "$kept" -gt 0 ]; then
    GIT_INDEX_FILE="$idx" g update-index --add -z --stdin < "$list" 2>/dev/null
  fi

  T=$(GIT_INDEX_FILE="$idx" g write-tree)
  H=$(g rev-parse HEAD)
  if [ "$T" = "$(g rev-parse HEAD^{tree})" ]; then
    CUR=$H
  else
    CUR=$(g commit-tree "$T" -p "$H" -m "preserve($n): uncommitted work at the monorepo import

Corpus, PII, live secrets and build artefacts were excluded by path filter.")
  fi

  sk=""
  for b in $(g branch --format='%(refname:short)'); do
    case "$b" in "$B2"|monorepo-merge-all-20260926|monorepo-import-snapshot) continue ;; esac
    if printf '%s' "$b" | grep -qE "$SKIP"; then sk="$sk $b[EXCLUDED]"; continue; fi
    if [ "$(g branch -r --contains "$b" 2>/dev/null | wc -l)" -ne 0 ]; then continue; fi
    if g merge-base --is-ancestor "$b" "$CUR" 2>/dev/null; then continue; fi
    if TT=$(g merge-tree --write-tree "$CUR" "$b" 2>/dev/null) && [ ${#TT} -eq 40 ]; then
      CUR=$(g commit-tree "$TT" -p "$CUR" -p "$(g rev-parse "$b")" -m "merge($n): fold in local-only branch $b")
      sk="$sk $b[merged]"
    else
      sk="$sk $b[CONFLICT]"
    fi
  done

  g update-ref "refs/heads/$B2" "$CUR"
  printf '  %-13s => %s  kept=%-4s excl=%-5s branches:%s\n' "$n" "${CUR:0:8}" "$kept" "$drop" "${sk:- none}"
}

P=$R/modules/Probata/probata
mk "$R/modules/Consignatio"                                                consignatio
mk "$R/modules/Legal-desktop"                                              legal
mk "$R/modules/vestigia-geodata_processor"                                 vestigia
mk "$R/modules/vestigia-geodata_processor/traceiq-rebuild"                 traceiq
mk "$P"                                                                    probata
mk "$P/modules/forks/sbv"                                                  sbv
mk "$P/modules/forks/timesketch"                                           timesketch
mk "$P/modules/custom"                                                     custom
mk "$R/modules/Consignatio/Intake/xplorer-copilot-buildkit/xplorer-copilot" xplorer

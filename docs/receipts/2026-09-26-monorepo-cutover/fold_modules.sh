#!/usr/bin/env bash
# Fold every module's clean branch into the Propria monorepo import branch.
set -u
R=E:/AI_Workspace/Projects/Propria
W=$R/_worktrees/propria-monorepo-import
S="$1"
B2=monorepo-clean-20260926
P=$R/modules/Probata/probata
CO="Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"

g() { git -C "$W" "$@"; }

echo "=== 1. rescue the 23 root-tracked buildkit files (exist in no repo) ==="
mkdir -p "$S/bk2"
( cd "$W" && tar cf - modules/Consignatio ) | ( cd "$S/bk2" && tar xf - )
echo "    rescued $(find "$S/bk2" -type f | wc -l) files"
g rm -q -r --cached modules/Consignatio
g commit -q -m "chore(import): unstage the stray Consignatio buildkit ahead of the subtree import

Restored into the imported subtree two commits later.

$CO"

echo "=== 2. stop ignoring the product modules ==="
sed -i '/^\/modules\/Consignatio\/$/d; /^\/modules\/Probata\/$/d; /^\/modules\/Legal-desktop\/$/d; /^\/modules\/vestigia-geodata_processor\/$/d' "$W/.gitignore"
cat >> "$W/.gitignore" <<'IGN'

# Evidence corpora and location data: kept on disk, never committed to the monorepo.
/modules/vestigia-geodata_processor/raw_api_responses/
/modules/vestigia-geodata_processor/TraceIQ_Main/
/modules/vestigia-geodata_processor/TraceIQ_Backups/
/modules/vestigia-geodata_processor/TraceIQ_Evidence/
/modules/vestigia-geodata_processor/traaceiq_mess/
/modules/vestigia-geodata_processor/Timeline.json
/modules/Consignatio/_intake/
/modules/Consignatio/casebible/vault-sorted.7z
IGN
g add .gitignore
g commit -q -m "chore(monorepo): track the product modules, keep evidence corpora out

The imported modules can no longer be ignored by the repository that now owns
them. In their place, the evidence and location-data corpora are ignored
explicitly so they stay on disk and never enter this history.

$CO"

echo "=== 3. clear leftover directories so the prefixes are free ==="
for d in modules/Consignatio; do
  [ -e "$W/$d" ] && mv "$W/$d" "$S/leftover2-$(basename $d)" && echo "    moved $d aside"
done

echo "=== 4. subtree add each module ==="
add() {
  rem=$1; src=$2; prefix=$3
  g remote add "$rem" "$src" 2>/dev/null
  g fetch "$rem" "$B2" >/dev/null 2>&1
  out=$(g subtree add --prefix="$prefix" "$rem" "$B2" 2>&1 | tail -1)
  printf '    %-58s %6s files   %s\n' "$prefix" "$(g ls-files "$prefix" | wc -l)" "$out"
}
add probata-c    "$P"                                      modules/Probata/probata
add sbv-c        "$P/modules/forks/sbv"                    modules/Probata/probata/modules/forks/sbv
add timesketch-c "$P/modules/forks/timesketch"             modules/Probata/probata/modules/forks/timesketch
add custom-c     "$P/modules/custom"                       modules/Probata/probata/modules/custom
add consignatio-c "$R/modules/Consignatio"                 modules/Consignatio
add xplorer-c    "$R/modules/Consignatio/Intake/xplorer-copilot-buildkit/xplorer-copilot" modules/Consignatio/Intake/xplorer-copilot-buildkit/xplorer-copilot
add legal-c      "$R/modules/Legal-desktop"                modules/Legal-desktop
add vestigia-c   "$R/modules/vestigia-geodata_processor"   modules/vestigia-geodata_processor
add traceiq-c    "$R/modules/vestigia-geodata_processor/traceiq-rebuild" modules/vestigia-geodata_processor/traceiq-rebuild

echo "=== 5. restore the rescued buildkit into the Consignatio subtree ==="
( cd "$S/bk2" && tar cf - modules/Consignatio ) | ( cd "$W" && tar xf - )
g add -f modules/Consignatio/Intake/xplorer-copilot-buildkit 2>/dev/null
g commit -q -m "chore(import): preserve the Consignatio xplorer-copilot buildkit

23 files that were tracked in the Propria root while the module was gitignored.
They exist in no other repository, and Consignatio's own .gitignore excludes the
path, so they are force-added rather than lost.

$CO" 2>/dev/null
echo "    buildkit files: $(g ls-files modules/Consignatio/Intake/xplorer-copilot-buildkit | wc -l)"

echo "=== 6. probata_build_crew source (zero commits; live .env excluded) ==="
BC=$R/modules/Probata/probata_build_crew
T=$W/modules/Probata/probata_build_crew
mkdir -p "$T"
for item in README.md .gitignore .env.example crew.jsonc pyproject.toml agents knowledge llm_chain; do
  [ -e "$BC/$item" ] && cp -r "$BC/$item" "$T/"
done
g add modules/Probata/probata_build_crew
g commit -q -m "feat(import): add probata_build_crew source

A git repository with zero commits, so its content enters as new files rather
than history. Owner ruling 2026-09-26: forks and tools the application uses are
included. Excluded: .env (live credentials), .venv, output, _stale and the
protected holding directory. .env.example is kept.

$CO" 2>/dev/null
echo "    build_crew files: $(g ls-files modules/Probata/probata_build_crew | wc -l)"

echo
echo "=== result ==="
echo "  commits : $(g rev-list --count HEAD)"
echo "  files   : $(g ls-files | wc -l)"
echo "  dirty   : $(g status --porcelain | wc -l)"
echo "  tracked .env files : $(g ls-files | grep -cE '(^|/)\.env$')"

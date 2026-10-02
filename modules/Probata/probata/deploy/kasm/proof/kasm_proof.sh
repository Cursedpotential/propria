#!/usr/bin/env bash
# Byline: Claude Code · Opus 5.5 · 2026-10-02
# Runs kasm_proof.mjs inside the Probata devbox (Coolify app pd3xc78ahqkfswq12bpfqgy1) on ovh-files, as kasm-user.
# Run ON ovh-files as root, from the folder holding kasm_proof.mjs:
#   bash kasm_proof.sh <label>
# The owner's Kasm password is read from /data/probata/secrets/kasm/kasm.env and handed to the container
# environment of this one docker exec; it is never printed. Screenshots land in
# /data/probata/volumes/devbox/home/work/kasm-proof/<label>/ and Kasm sessions the proof opened are ended after.
set -euo pipefail
label="${1:?usage: kasm_proof.sh <label>}"
here="$(cd "$(dirname "$0")" && pwd)"
home=/data/probata/volumes/devbox/home
work="$home/work/kasm-proof"
C="$(docker ps --filter name=devbox-pd3xc78ahqkfswq12bpfqgy1 --format '{{.Names}}' | head -1)"
[ -n "$C" ] || { echo "no running devbox container"; exit 2; }
install -d -o 1000 -g 1000 "$work"
install -m 0644 -o 1000 -g 1000 "$here/kasm_proof.mjs" "$work/kasm_proof.mjs"
docker exec -u 1000 -w /home/kasm-user/work/kasm-proof "$C" bash -lc \
  '[ -d node_modules/playwright-core ] || npm install --no-audit --no-fund --silent playwright-core@1 >/dev/null'
val() { sed -nE "s/^[[:space:]]*$1[[:space:]]*=[[:space:]]*(.*)$/\1/p" /data/probata/secrets/kasm/kasm.env | tail -1; }
KASM_PROOF_USER="$(val KASM_OWNER_USER)" KASM_PROOF_PASSWORD="$(val KASM_OWNER_PASSWORD)" \
  docker exec -u 1000 -w /home/kasm-user/work/kasm-proof -e KASM_PROOF_USER -e KASM_PROOF_PASSWORD "$C" \
  node kasm_proof.mjs "/home/kasm-user/work/kasm-proof/$label"
ls -la "$work/$label"
# end the sessions the proof opened (Devbox, Devbox (RDP)); they must not stay running
python3 "$here/end_sessions.py"

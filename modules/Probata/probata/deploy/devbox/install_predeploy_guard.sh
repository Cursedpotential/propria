#!/usr/bin/env bash
# Host prep on ovh-files for the coolify-mcp pre-deploy guard (owner 2026-10-02 08:38, auto-guard A).
# Copies the tracked guard + dispatcher into /data/probata/config/predeploy-guard/ and authorizes ONE public key
# for root, locked to the dispatcher: no shell, no pty, no forwarding. Idempotent; re-run after changing either file.
#
# Run from the Propria checkout (desktop; nothing runs locally):
#   scp -i ~/.ssh/ovh deploy/devbox/{pre_redeploy_check.py,predeploy_guard_dispatch.sh,install_predeploy_guard.sh} \
#       ~/.secrets/coolify-guard/id_ed25519.pub root@100.91.190.107:/tmp/predeploy-guard-install/
#   ssh -i ~/.ssh/ovh root@100.91.190.107 bash /tmp/predeploy-guard-install/install_predeploy_guard.sh /tmp/predeploy-guard-install
# Byline: Claude Code · Opus 5.5 · 2026-10-02
set -euo pipefail
SRC="${1:?usage: install_predeploy_guard.sh <dir holding the three files and id_ed25519.pub>}"
DST=/data/probata/config/predeploy-guard
AK=/root/.ssh/authorized_keys
KEY_COMMENT=coolify-mcp-predeploy-guard

install -d -m 0755 "$DST"
install -m 0644 "$SRC/pre_redeploy_check.py" "$DST/pre_redeploy_check.py"
install -m 0755 "$SRC/predeploy_guard_dispatch.sh" "$DST/dispatch.sh"
sha256sum "$DST/pre_redeploy_check.py" "$DST/dispatch.sh"

pub="$(awk '{print $1" "$2}' "$SRC/id_ed25519.pub")"
[[ "$pub" == ssh-ed25519\ * ]] || { echo "STOP: not an ed25519 public key"; exit 2; }
line="restrict,no-pty,no-port-forwarding,no-agent-forwarding,no-X11-forwarding,command=\"$DST/dispatch.sh\" $pub $KEY_COMMENT"
touch "$AK"; chmod 600 "$AK"
if grep -qF "$pub" "$AK"; then
  # replace the existing entry for this key (keeps the options current)
  cp -p "$AK" "$AK.bak-$(date -u +%Y%m%dT%H%M%SZ)"
  awk -v k="$pub" -v l="$line" 'index($0,k){print l; next} {print}' "$AK" > "$AK.new" && mv "$AK.new" "$AK" && chmod 600 "$AK"
  echo "updated the guard key entry"
else
  cp -p "$AK" "$AK.bak-$(date -u +%Y%m%dT%H%M%SZ)"
  printf '%s\n' "$line" >> "$AK"
  echo "added the guard key entry"
fi
grep -c "$KEY_COMMENT" "$AK"

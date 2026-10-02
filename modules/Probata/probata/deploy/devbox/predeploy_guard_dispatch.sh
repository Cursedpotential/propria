#!/bin/sh
# Forced command for the coolify-mcp pre-deploy guard key on ovh-files (owner 2026-10-02 08:38, auto-guard A).
# The key's authorized_keys entry is `restrict,no-pty,no-port-forwarding,command="<this file>"`, so whatever the
# client asks for arrives here as SSH_ORIGINAL_COMMAND and only a known guard name runs. There is no shell.
# Installed to /data/probata/config/predeploy-guard/dispatch.sh by install_predeploy_guard.sh.
# Byline: Claude Code · Opus 5.5 · 2026-10-02
set -u
GUARD_DIR=/data/probata/config/predeploy-guard
case "${SSH_ORIGINAL_COMMAND:-}" in
  devbox)
    exec /usr/bin/python3 "${GUARD_DIR}/pre_redeploy_check.py"
    ;;
  *)
    echo "REFUSED: this key runs only a known pre-deploy guard; got '${SSH_ORIGINAL_COMMAND:-<none>}'"
    exit 64
    ;;
esac

#!/usr/bin/env bash
# What the ttyd web terminal runs (portal item P-3): Claude Code in the devbox's work folder, and a
# login shell when Claude exits or is not signed in yet. ttyd wraps this in `tmux new-session -A -s
# claude`, so a closed browser tab re-attaches to the same session instead of starting a new one.
# Sign-in: the owner signs in once with `claude` (Anthropic's own login, his own subscription); the
# credentials stay in /home/kasm-user/.claude on the devbox's host volume and are never copied out.
# Byline: Claude Code · Opus 5.5 · 2026-10-02
set -u
cd "$HOME/work" 2>/dev/null || cd "$HOME"
claude || true
exec bash -l

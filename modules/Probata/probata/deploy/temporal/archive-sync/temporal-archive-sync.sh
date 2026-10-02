#!/usr/bin/env bash
# temporal-archive-sync.sh - copy Temporal's history archive off ovh-files, add-only.
# Byline: Claude Code · Opus 5.5 · 2026-10-02
#
# Owner 2026-10-02 02:55 EDT: keep every run's history; B2 for now, then a sync location that pushes
# to B2 and his machine or Google Drive. Temporal's filestore archiver writes
# /data/probata/volumes/temporal-archive (bind-mounted in deploy/temporal/compose.temporal.yaml).
# This copies new files out and never deletes or overwrites anything at the destination (--immutable).
# More destinations later are one more line each.
# Runs from temporal-archive-sync.timer; B2 credentials come only from the unit's EnvironmentFile.
set -euo pipefail
SRC=/data/probata/volumes/temporal-archive
CONF=/opt/casebible/rclone.conf
rclone copy --config "$CONF" "$SRC" b2:salem-data/propria/temporal-archive --immutable --transfers 4 --stats-one-line -v

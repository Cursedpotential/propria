#!/usr/bin/env bash
# Byline: Claude Code · Opus 5.5 · 2026-09-26
# Move ovh-files' /data (~53 GB: probata volumes/config/secrets, consignatio, coolify, milvus, ...) off the system disk
# onto the 500 GB data disk (owner 2026-09-26 08:07 "for the last time yes"; first ordered 2026-09-23).
#   /data -> /mnt/data/data, bind-mounted back at /data in fstab - the same pattern as /var/lib/{docker,containerd} in
#   ovh-files-data-disk-20260924.sh - so no path inside a container, compose file or Coolify config changes.
# Downtime is kept short: a warm copy runs while everything is up; then every container is stopped (Milvus gently, then
# etcd, then the rest), a final delta copy runs and is verified, /data is swapped for the bind mount, and the containers
# start again. Before the swap any failure restarts everything on the untouched old /data; a failure during the swap puts
# the old /data back. Nothing is deleted: the old directory stays on the system disk as /data._superseded-20260926 for
# the owner to remove.
# PRECONDITION (as 2026-09-24): Coolify's Docker cleanup is off for this server, so stopped containers are not pruned.
# Usage (on ovh-files, as root):  bash ovh-files-data-dir-20260926.sh check   # read-only report
#                                  bash ovh-files-data-dir-20260926.sh apply   # does the move
set -euo pipefail
MODE="${1:-check}"
MNT=/mnt/data
SRC=/data
DST=$MNT/data
STAMP=20260926
OLD=/data._superseded-$STAMP
LIST=/root/ovh-files-containers-before-$STAMP.txt
FSTAB_LINE="$DST $SRC none bind,x-systemd.requires-mounts-for=$MNT 0 0"

say() { printf '%s %s\n' "$(date +%H:%M:%S)" "$*"; }
# host processes (not container processes, whose paths are namespaced) holding a file or cwd under /data
holders() { { find /proc/[0-9]*/fd /proc/[0-9]*/cwd -maxdepth 1 -lname "$SRC/*" 2>/dev/null || true; } | cut -d/ -f3 | sort -u; }

# --- checks (both modes) -------------------------------------------------------------------------------------------
mountpoint -q "$MNT" || { say "$MNT is not mounted - refusing"; exit 1; }
if mountpoint -q "$SRC"; then say "$SRC is already a mount point ($(findmnt -no SOURCE "$SRC")) - nothing to do"; exit 0; fi
[ -e "$OLD" ] && { say "$OLD already exists - refusing"; exit 1; }
need_kb=$(du -xsk "$SRC" | cut -f1)
free_kb=$(df -k --output=avail "$MNT" | tail -1)
say "$SRC: $((need_kb / 1024 / 1024)) GB; $MNT free: $((free_kb / 1024 / 1024)) GB; system disk: $(df -h / | awk 'NR==2{print $3" used, "$4" free, "$5}')"
[ "$free_kb" -gt $((need_kb + 20 * 1024 * 1024)) ] || { say "not enough room on $MNT - refusing"; exit 1; }
say "running containers: $(docker ps -q | wc -l); host processes holding files under $SRC: $(holders | wc -l)"
[ "$MODE" = apply ] || { say "check only; run with 'apply' to move"; exit 0; }

# --- apply ---------------------------------------------------------------------------------------------------------
docker ps --format '{{.Names}}' | sort > "$LIST"
say "saved running container list ($(wc -l < "$LIST"))"
mkdir -p "$DST"
say "warm copy while everything is up"
rsync -aHAX --numeric-ids --delete "$SRC/" "$DST/"
say "warm copy done ($(du -xsh "$DST" | cut -f1))"

start_saved() {  # etcd, then Milvus, then everything else: every container is stopped by hand, so none comes back alone
  for c in $(grep -E 'etcd' "$LIST"); do docker start "$c" >/dev/null || true; done
  sleep 20
  for c in $(grep -E 'milvus' "$LIST" | grep -v etcd); do docker start "$c" >/dev/null || true; done
  for c in $(grep -v -E 'etcd|milvus' "$LIST"); do docker start "$c" >/dev/null || true; done
  for _ in $(seq 1 30); do
    n=$(docker ps --format '{{.Names}}' | sort | comm -23 "$LIST" - | wc -l)
    [ "$n" = 0 ] && break
    sleep 10
  done
  missing=$(docker ps --format '{{.Names}}' | sort | comm -23 "$LIST" - | tr '\n' ' ')
  say "containers back: $(docker ps -q | wc -l) running; missing: ${missing:-none}"
}
restore_old() {  # before the swap: the old /data is untouched, so start everything again on it
  trap - ERR
  say "FAILED before the swap - starting the containers again on the old $SRC"
  start_saved
  exit 1
}
trap restore_old ERR

for c in $(docker ps --format '{{.Names}}' | grep -E 'milvus' | grep -v etcd); do say "stop $c"; docker stop -t 120 "$c" >/dev/null; done
for c in $(docker ps --format '{{.Names}}' | grep -E 'etcd'); do say "stop $c"; docker stop -t 60 "$c" >/dev/null; done
docker ps -q | xargs -r docker stop -t 60 >/dev/null
running=$(docker ps -q | wc -l)
say "all containers stopped ($running still running)"
[ "$running" = 0 ]
held=$(holders)
if [ -n "$held" ]; then say "host processes still hold files under $SRC:"; ps -o pid,comm -p "$(echo "$held" | paste -sd,)"; false; fi

say "final copy"
rsync -aHAX --numeric-ids --delete "$SRC/" "$DST/"
changes=$(rsync -aHAXn --numeric-ids --delete --itemize-changes "$SRC/" "$DST/" | wc -l)
say "verify: $changes differences after the final copy"
[ "$changes" = 0 ]

# --- swap ----------------------------------------------------------------------------------------------------------
undo_swap() {
  trap - ERR
  say "FAILED during the swap - putting the old $SRC back"
  umount "$SRC" 2>/dev/null || true
  rmdir "$SRC" 2>/dev/null || true
  [ -e "$SRC" ] || mv "$OLD" "$SRC"
  sed -i "\|^$DST $SRC none bind|d" /etc/fstab
  systemctl daemon-reload
  start_saved
  exit 1
}
trap undo_swap ERR
mv "$SRC" "$OLD"
mkdir "$SRC"
grep -qxF "$FSTAB_LINE" /etc/fstab || echo "$FSTAB_LINE" >> /etc/fstab
systemctl daemon-reload
mount "$SRC"
mountpoint -q "$SRC"
[ -d "$SRC/probata" ] && [ -d "$SRC/coolify" ] && [ -d "$SRC/consignatio" ]
trap - ERR
say "$SRC is now a bind mount of $DST ($(findmnt -no SOURCE "$SRC")); old copy kept at $OLD"

start_saved
say "system disk: $(df -h / | awk 'NR==2{print $3" used, "$4" free, "$5}'); $MNT: $(df -h "$MNT" | awk 'NR==2{print $3" used, "$4" free, "$5}')"
say "done"

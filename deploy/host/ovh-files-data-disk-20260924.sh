#!/usr/bin/env bash
# Byline: Claude Code · Opus 5.5 · 2026-09-24
# Move ovh-files' container stores onto its blank 500 GB disk (owner 2026-09-24 ~13:15: "Yes, now", start once the
# Weaviate publish finishes; same method as the ovh-app move of 06:06-06:40 the same day).
#   /dev/sdb (blank, 500 GB) -> ext4, label ovhfiles-data, mounted at /mnt/data (fstab by UUID, nofail)
#   /var/lib/containerd (57 GB) and /var/lib/docker (51 GB, mostly named volumes) -> /mnt/data/{containerd,docker},
#   bind-mounted back in fstab; systemd drop-ins make containerd and docker wait for the disk.
# Milvus is stopped gently first (Milvus, then etcd; an abrupt stop of both corrupted etcd before) and started again
# explicitly afterwards (containers stopped by hand do not restart with the daemon under "unless-stopped").
# Nothing is deleted: the old directories are renamed *._superseded-20260924 on the system disk for the owner to remove.
# Usage (on ovh-files, as root):  bash ovh-files-data-disk-20260924.sh check   # read-only report
#                                  bash ovh-files-data-disk-20260924.sh apply   # does the move (outage ~30-45 min)
set -euo pipefail
MODE="${1:-check}"
DISK=/dev/sdb
MNT=/mnt/data
STAMP=20260924
STORES=(containerd docker)

say() { printf '%s %s\n' "$(date +%H:%M:%S)" "$*"; }

# --- checks (both modes) -------------------------------------------------------------------------------------------
[ -b "$DISK" ] || { say "no $DISK"; exit 1; }
if [ -n "$(blkid -o value -s TYPE "$DISK" 2>/dev/null)" ] || [ -n "$(wipefs -n "$DISK" 2>/dev/null | tail -n +2)" ]; then
  if ! blkid -o value -s LABEL "$DISK" | grep -qx ovhfiles-data; then
    say "$DISK is not blank and not ours (label $(blkid -o value -s LABEL "$DISK")) - refusing"; exit 1
  fi
fi
say "disk: $(lsblk -dno SIZE "$DISK") $DISK, fs=$(blkid -o value -s TYPE "$DISK" 2>/dev/null || echo none)"
say "system disk: $(df -h / | awk 'NR==2{print $3" used, "$4" free, "$5}')"
for s in "${STORES[@]}"; do say "store /var/lib/$s: $(du -xsh /var/lib/$s 2>/dev/null | cut -f1)"; done
say "running containers: $(docker ps -q | wc -l)"
[ "$MODE" = apply ] || { say "check only; run with 'apply' to move"; exit 0; }

# --- apply ---------------------------------------------------------------------------------------------------------
docker ps --format '{{.Names}}' | sort > /root/ovh-files-containers-before-$STAMP.txt
say "saved running container list ($(wc -l < /root/ovh-files-containers-before-$STAMP.txt))"

if [ -z "$(blkid -o value -s TYPE "$DISK" 2>/dev/null)" ]; then
  mkfs.ext4 -q -L ovhfiles-data "$DISK"; say "formatted $DISK ext4 ovhfiles-data"
fi
UUID=$(blkid -o value -s UUID "$DISK")
mkdir -p "$MNT"
grep -q "$UUID" /etc/fstab || echo "UUID=$UUID $MNT ext4 defaults,discard,nofail 0 2" >> /etc/fstab
mountpoint -q "$MNT" || mount "$MNT"
say "mounted $MNT ($(df -h "$MNT" | awk 'NR==2{print $2}'))"

UNITS=(docker.socket docker.service containerd.service)
start_saved() {  # etcd, then Milvus, then everything else: every container was stopped by hand (see below), so none
  # comes back with the daemon by itself
  for c in $(grep -E 'etcd' /root/ovh-files-containers-before-$STAMP.txt); do docker start "$c" >/dev/null || true; done
  sleep 20
  for c in $(grep -E 'milvus' /root/ovh-files-containers-before-$STAMP.txt | grep -v etcd); do docker start "$c" >/dev/null || true; done
  for c in $(grep -v -E 'etcd|milvus' /root/ovh-files-containers-before-$STAMP.txt); do docker start "$c" >/dev/null || true; done
  for i in $(seq 1 30); do
    n=$(docker ps --format '{{.Names}}' | sort | comm -23 /root/ovh-files-containers-before-$STAMP.txt - | wc -l)
    [ "$n" = 0 ] && break; sleep 10
  done
  missing=$(docker ps --format '{{.Names}}' | sort | comm -23 /root/ovh-files-containers-before-$STAMP.txt -)
  say "containers back: $(docker ps -q | wc -l) running; missing: ${missing:-none}"
}
restore_old() {  # a failed copy or verify must not leave the host without Docker: start again on the untouched old store
  trap - ERR
  systemctl unmask --runtime "${UNITS[@]}" || true
  systemctl start containerd docker
  say "restored service on the old store"
  start_saved
}

# gentle Milvus stop, then every other container by hand: this host runs Docker with "live-restore": true, so stopping
# the daemons leaves containers running and writing (second apply, 02:50 UTC: verify found 2 changed entries).
# PRECONDITION (checked by the operator, not here): Coolify's Docker cleanup must be off for this server. It prunes
# stopped containers and, with delete_unused_volumes, their named volumes; at 02:50 UTC it pruned the stopped Milvus
# container in the seconds between unmask and restart (data safe: Milvus binds /data/probata/volumes/memsearch-milvus).
for c in $(docker ps --format '{{.Names}}' | grep -E 'milvus' | grep -v etcd); do say "stop $c"; docker stop -t 120 "$c" >/dev/null; done
for c in $(docker ps --format '{{.Names}}' | grep -E 'etcd'); do say "stop $c"; docker stop -t 60 "$c" >/dev/null; done
docker ps -q | xargs -r docker stop -t 60 >/dev/null
say "all containers stopped ($(docker ps -q | wc -l) still running)"
systemctl stop "${UNITS[@]}"
# 2026-09-24 02:41 UTC, first apply: Coolify found Docker down and ran apt-get install docker-ce containerd.io, which
# upgraded and restarted both daemons mid-copy (verify then found 16 changed entries). Masked for the copy window.
systemctl mask --runtime "${UNITS[@]}"
say "docker + containerd stopped and masked for the copy"
trap 'say "error during copy - restoring"; restore_old; exit 1' ERR

for s in "${STORES[@]}"; do
  say "copy /var/lib/$s -> $MNT/$s"
  mkdir -p "$MNT/$s"
  # --delete: the target is this script's own mirror; a rerun drops what the source no longer has
  rsync -aHAXx --delete --numeric-ids --info=stats1 "/var/lib/$s/" "$MNT/$s/"
  left=$(rsync -aHAXxn --delete --numeric-ids --out-format='%n' "/var/lib/$s/" "$MNT/$s/" | grep -vc '/$' || true)
  [ "$left" = 0 ] || { say "verify failed for $s: $left entries differ - old store untouched"; restore_old; exit 1; }
  say "verified $s: 0 differences"
done
trap - ERR

for s in "${STORES[@]}"; do
  mv "/var/lib/$s" "/var/lib/$s._superseded-$STAMP"
  mkdir -p "/var/lib/$s"
  grep -q " /var/lib/$s none bind" /etc/fstab || \
    echo "$MNT/$s /var/lib/$s none bind,x-systemd.requires-mounts-for=$MNT 0 0" >> /etc/fstab
done
for u in containerd docker; do
  mkdir -p "/etc/systemd/system/$u.service.d"
  printf '[Unit]\nRequiresMountsFor=/var/lib/%s\n' "$u" > "/etc/systemd/system/$u.service.d/10-data-volume.conf"
done
systemctl daemon-reload
mount -a
for s in "${STORES[@]}"; do mountpoint -q "/var/lib/$s" || { say "/var/lib/$s not mounted - stopping"; exit 1; }; done
systemctl unmask --runtime "${UNITS[@]}"
systemctl start containerd docker
say "daemons started on $MNT; waiting for containers"
start_saved
say "system disk now: $(df -h / | awk 'NR==2{print $3" used, "$4" free, "$5}')"
say "old stores kept at /var/lib/*._superseded-$STAMP for the owner to delete"

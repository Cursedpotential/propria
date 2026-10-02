#!/usr/bin/env python3
"""Pre-redeploy guard for the devbox: rescue everything that lives only in the container layer.

A Coolify redeploy recreates the devbox container. Whatever an agent or the owner wrote OUTSIDE the
host-volume mounts (the "writable layer") is destroyed with the old container. That is how the
2026-09-28 deploy lost .npm, .duckdb, .claude.json and two agent work folders (P-1 state report,
2026-10-02). Owner, 2026-10-02 01:27 EDT: "Can we address the loss of the agents' work so that
doesn't happen again?"

What it does, on ovh-files as root (stdlib only; reads the container, writes only into the home volume):
  1. Finds the running devbox container (Coolify app pd3xc78ahqkfswq12bpfqgy1) and its mounts.
  2. Lists its writable-layer changes with `docker diff`, drops paths under mounts, and sorts the rest
     into IGNORED (named runtime churn: X/ICE/dbus sockets, Kasm's self-extracting service binaries,
     Chrome singleton dirs, /run, Python bytecode caches) and RESCUE (everything else).
  3. Copies every RESCUE root into <home volume>/rescued/<UTC stamp>/files/ with tar, keeping paths.
  4. Verifies the copy: sha256 of every regular file inside the container == the rescued copy.
  5. Records what was installed at runtime (apt manual packages, npm globals, uv tools) so it can be
     moved into the Dockerfile, writes manifest.tsv / verify.tsv / report.txt, chowns to uid 1000.
  6. Prints "SAFE TO REDEPLOY: yes" and exits 0 ONLY when every rescued file verified; otherwise
     prints "NOT SAFE" with the reasons and exits 2. Exit 3 = could not run (no docker, etc.).

Run (from the desktop, nothing runs locally):
  ssh -i ~/.ssh/ovh root@100.91.190.107 python3 - < deploy/devbox/pre_redeploy_check.py
  ... python3 - --dry-run < ...   classify and print only, copy nothing

Byline: Claude Code · Opus 5.5 · 2026-10-02
"""
from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import os
import re
import subprocess
import sys

APP_UUID = "pd3xc78ahqkfswq12bpfqgy1"
HOME_VOLUME = "/data/probata/volumes/devbox/home"

# Runtime churn that carries no work: recreated by the next container start. Each entry names why.
IGNORE = [
    (r"^/tmp/\.X11-unix(/|$)", "X server sockets"),
    (r"^/tmp/\.ICE-unix(/|$)", "ICE sockets"),
    (r"^/tmp/\.X\d+-lock$", "X display lock"),
    (r"^/tmp/\.xfsm-ICE-[^/]+$", "XFCE session socket"),
    (r"^/tmp/dbus-[^/]+(/|$)", "dbus socket"),
    (r"^/tmp/ssh-[^/]+(/|$)", "ssh-agent socket"),
    (r"^/tmp/tmux-\d+(/|$)", "tmux sockets"),
    (r"^/tmp/printer$", "Kasm printer relay socket"),
    (r"^/tmp/_MEI[^/]+(/|$)", "self-extracted PyInstaller binary of a running Kasm service"),
    (r"^/tmp/staticx-[^/]+(/|$)", "self-extracted staticx binary of a running Kasm service"),
    (r"^/tmp/com\.google\.Chrome\.[^/]+(/|$)", "Chrome singleton socket dir"),
    (r"^/(var/)?run(/|$)", "runtime state (/run)"),
    (r"(^|/)__pycache__(/|$)", "Python bytecode cache"),
    (r"\.pyc$", "Python bytecode cache"),
]
IGNORE_RE = [(re.compile(p), why) for p, why in IGNORE]


def sh(cmd: list[str], *, inp: bytes | None = None, check: bool = True) -> bytes:
    r = subprocess.run(cmd, input=inp, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    if check and r.returncode != 0:
        raise RuntimeError(f"{' '.join(cmd[:4])}… exit {r.returncode}: {r.stderr.decode(errors='replace')[:400]}")
    return r.stdout


def under(path: str, roots: list[str]) -> bool:
    return any(path == r or path.startswith(r.rstrip("/") + "/") for r in roots)


def find_containers(prefix: str) -> list[str]:
    out = sh(["docker", "ps", "--filter", f"name=^{prefix}", "--format", "{{.Names}}"]).decode()
    return [n for n in out.split() if n]


def check_container(name: str, home_volume: str, stamp: str, dry_run: bool) -> tuple[bool, list[str]]:
    problems: list[str] = []
    mounts = sh(["docker", "inspect", name, "--format",
                 "{{range .Mounts}}{{.Destination}}\n{{end}}"]).decode().split()
    diff = sh(["docker", "diff", name]).decode().splitlines()
    rows = []  # (kind, path, cls, reason)
    for line in diff:
        kind, _, path = line.partition(" ")
        if under(path, mounts) or any(m.startswith(path.rstrip("/") + "/") for m in mounts) and kind == "C":
            rows.append((kind, path, "mount", "host volume or its parent dir"))
            continue
        if kind == "D":
            rows.append((kind, path, "deleted", "removed from the image at runtime (nothing to copy)"))
            continue
        why = next((w for rx, w in IGNORE_RE if rx.search(path)), None)
        rows.append((kind, path, "ignored" if why else "candidate", why or ""))

    cand = [(k, p) for k, p, c, _ in rows if c == "candidate"]
    # file types for candidates (one exec, NUL-separated)
    types: dict[str, str] = {}
    if cand:
        out = sh(["docker", "exec", "-i", name, "xargs", "-0", "-r", "stat", "--printf", "%F\t%n\\0"],
                 inp=b"\0".join(p.encode() for _, p in cand), check=False).decode(errors="replace")
        for rec in out.split("\0"):
            if "\t" in rec:
                t, p = rec.split("\t", 1)
                types[p] = t
    added = {p for k, p in cand if k == "A"}
    roots: list[str] = []
    for k, p in cand:
        if p not in types:  # vanished since docker diff (temp file) — nothing left to lose
            continue
        parent = os.path.dirname(p)
        if under(parent, sorted(added)) and parent != "/":
            continue  # covered by an added ancestor
        if k == "C" and types[p] == "directory":
            continue  # changed dir: its changed children are listed themselves
        roots.append(p)
    roots = sorted(set(roots))
    # collapse roots nested in other roots
    final: list[str] = []
    for p in roots:
        if not any(p != r and p.startswith(r.rstrip("/") + "/") for r in final):
            final.append(p)

    rescue_dir = os.path.join(home_volume, "rescued", stamp, name)
    report = [f"container {name}", f"docker diff entries: {len(diff)}",
              f"  mount-related {sum(1 for r in rows if r[2]=='mount')}, ignored {sum(1 for r in rows if r[2]=='ignored')}, "
              f"deleted {sum(1 for r in rows if r[2]=='deleted')}, candidates {len(cand)}",
              f"rescue roots: {len(final)}"]
    for p in final:
        report.append(f"  RESCUE {types.get(p,'?'):<14} {p}")
    ign: dict[str, int] = {}
    for _, _, c, why in rows:
        if c == "ignored":
            ign[why] = ign.get(why, 0) + 1
    for why, n in sorted(ign.items(), key=lambda x: -x[1]):
        report.append(f"  ignored {n:>5}  {why}")
    if dry_run:
        report.append("DRY RUN: nothing copied")
        return (not final), report

    os.makedirs(os.path.join(rescue_dir, "files"), exist_ok=True)
    with open(os.path.join(rescue_dir, "manifest.tsv"), "w") as f:
        f.write("kind\tpath\tclass\treason\n")
        for k, p, c, why in rows:
            f.write(f"{k}\t{p}\t{'rescue' if p in final else c}\t{why}\n")
    # runtime installs, so they can move into the Dockerfile
    inv = []
    for label, cmd in [("apt-mark showmanual", "apt-mark showmanual"),
                       ("npm -g", "npm ls -g --depth=0 2>/dev/null"),
                       ("uv tools (/opt/uv-tools)", "UV_TOOL_DIR=/opt/uv-tools uv tool list 2>/dev/null")]:
        inv.append(f"## {label}\n" + sh(["docker", "exec", name, "sh", "-c", cmd], check=False).decode(errors="replace"))
    with open(os.path.join(rescue_dir, "installed-tools.txt"), "w") as f:
        f.write("\n".join(inv))

    verified = 0
    if final:
        rel = b"\0".join(p.lstrip("/").encode() for p in final)
        tar = subprocess.Popen(["docker", "exec", "-i", name, "tar", "-C", "/", "--null", "-T", "-", "-cpf", "-"],
                               stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        untar = subprocess.Popen(["tar", "-C", os.path.join(rescue_dir, "files"), "-xpf", "-"],
                                 stdin=tar.stdout, stderr=subprocess.PIPE)
        tar.stdin.write(rel)
        tar.stdin.close()
        tar.stdout.close()
        uerr = untar.communicate()[1]
        terr = tar.stderr.read()
        tar.wait()
        if untar.returncode != 0:
            problems.append(f"untar failed: {uerr.decode(errors='replace')[:300]}")
        if tar.returncode not in (0, 1):  # 1 = "file changed as we read it" / sockets ignored
            problems.append(f"tar in container exit {tar.returncode}: {terr.decode(errors='replace')[:300]}")
        # verify: sha256 of every regular file, container vs rescued copy
        out = sh(["docker", "exec", "-i", name, "sh", "-c",
                  "cd / && xargs -0 -I{} find {} -type f -print0 | xargs -0 -r sha256sum"],
                 inp=rel, check=False).decode(errors="replace")
        src = {}
        for line in out.splitlines():
            h, _, p = line.partition("  ")
            if p:
                src[p] = h
        bad = []
        with open(os.path.join(rescue_dir, "verify.tsv"), "w") as f:
            f.write("path\tcontainer_sha256\trescued_sha256\tok\n")
            for p, h in sorted(src.items()):
                dst = os.path.join(rescue_dir, "files", p)
                try:
                    with open(dst, "rb") as fh:
                        d = hashlib.file_digest(fh, "sha256").hexdigest() if hasattr(hashlib, "file_digest") \
                            else hashlib.sha256(fh.read()).hexdigest()
                except OSError:
                    d = "MISSING"
                ok = d == h
                verified += ok
                if not ok:
                    bad.append(p)
                f.write(f"/{p}\t{h}\t{d}\t{'yes' if ok else 'NO'}\n")
        report.append(f"verified {verified}/{len(src)} regular files by sha256")
        if bad:
            problems.append(f"{len(bad)} files differ or are missing in the copy (first: /{bad[0]}); "
                            "a live program may still be writing them — stop it and re-run")
    sh(["chown", "-R", "1000:1000", os.path.join(home_volume, "rescued")], check=False)
    report.append(f"rescue folder: {rescue_dir}")
    with open(os.path.join(rescue_dir, "report.txt"), "w") as f:
        f.write("\n".join(report + [f"PROBLEM: {p}" for p in problems]) + "\n")
    return (not problems), report


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("--container-prefix", default=f"devbox-{APP_UUID}")
    ap.add_argument("--home-volume", default=HOME_VOLUME)
    ap.add_argument("--dry-run", action="store_true")
    a = ap.parse_args()
    try:
        names = find_containers(a.container_prefix)
    except Exception as e:  # noqa: BLE001
        print(f"CANNOT RUN: {e}")
        return 3
    if not names:
        print(f"no running container matches {a.container_prefix}: no writable layer to lose")
        print("SAFE TO REDEPLOY: yes")
        return 0
    if not os.path.isdir(a.home_volume):
        print(f"CANNOT RUN: home volume {a.home_volume} missing")
        return 3
    stamp = dt.datetime.now(dt.timezone.utc).strftime("%Y-%m-%dT%H%M%SZ")
    all_ok = True
    for n in names:
        ok, rep = check_container(n, a.home_volume, stamp, a.dry_run)
        print("\n".join(rep))
        all_ok &= ok
    if a.dry_run:
        print("DRY RUN — not a safety verdict")
        return 0
    print("SAFE TO REDEPLOY: yes" if all_ok else "NOT SAFE: see PROBLEM lines in report.txt; do not redeploy")
    return 0 if all_ok else 2


if __name__ == "__main__":
    sys.exit(main())

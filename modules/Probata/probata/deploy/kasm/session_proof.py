#!/usr/bin/env python3
"""Live proof for a Kasm workspace (P-1 step 6): launch a session through Kasm's client API, check it runs, write a
marker into its home, end the session, launch again and read the marker back. Runs ON ovh-files as root; stdlib only.

  ssh root@100.91.190.107 "python3 /data/probata/kasm-installer/session_proof.py Devbox"

The marker is a dated file under ~/work/kasm-persistence-proof/ in the devbox home volume; the script removes it at the
end (test data is purged), after printing its sha256 from both sessions. Credentials come from kasm.env, never printed.

Byline: Claude Code · Opus 5.5 · 2026-10-02
"""
from __future__ import annotations

import datetime as dt
import hashlib
import json
import re
import ssl
import subprocess
import sys
import time
import urllib.error
import urllib.request

ENV_FILE = "/data/probata/secrets/kasm/kasm.env"
API = "https://127.0.0.1:8443/api"
CTX = ssl._create_unverified_context()


def env() -> dict:
    out = {}
    for line in open(ENV_FILE, encoding="utf-8"):
        m = re.match(r"^\s*([A-Z_]+)\s*=\s*(.*?)\s*$", line)
        if m:
            out[m.group(1)] = m.group(2)
    return out


def call(path: str, body: dict) -> dict:
    req = urllib.request.Request(API + path, data=json.dumps(body).encode(), method="POST",
                                 headers={"Content-Type": "application/json"})
    try:
        out = json.load(urllib.request.urlopen(req, context=CTX, timeout=300))
    except urllib.error.HTTPError as e:
        out = {"error_message": f"HTTP {e.code}: {e.read()[:300].decode(errors='replace')}"}
    return out


def launch(tok: dict, image_id: str) -> tuple[str, str]:
    r = call("/request_kasm", {**tok, "image_id": image_id, "enable_sharing": False})
    if r.get("error_message"):
        raise SystemExit(f"request_kasm failed: {r['error_message']}")
    kasm_id = r["kasm_id"]
    for _ in range(120):
        s = call("/get_kasm_status", {**tok, "kasm_id": kasm_id, "skip_agent_check": False})
        k = s.get("kasm") or {}
        op = s.get("operational_status") or k.get("operational_status")
        if op == "running" and k.get("container_id"):
            return kasm_id, k["container_id"]
        if s.get("error_message"):
            print("status:", s["error_message"])
        time.sleep(5)
    raise SystemExit(f"session {kasm_id} did not reach running")


def sh(*args: str) -> str:
    return subprocess.run(args, capture_output=True, text=True, check=True).stdout


def main() -> int:
    name = sys.argv[1] if len(sys.argv) > 1 else "Devbox"
    e = env()
    user = e.get("KASM_OWNER_USER", e["KASM_ADMIN_USER"])
    pw = e.get("KASM_OWNER_PASSWORD", e["KASM_ADMIN_PASSWORD"])
    a = call("/authenticate", {"username": user, "password": pw, "logout_other_sessions": False})
    if a.get("error_message"):
        raise SystemExit(f"login failed: {a['error_message']}")
    tok = {"token": a["token"], "username": user, "user_id": a["user_id"]}
    imgs = call("/get_user_images", dict(tok)).get("images", [])
    img = next((i for i in imgs if i.get("friendly_name") == name), None)
    if not img:
        raise SystemExit(f"workspace {name!r} not offered to {user}; offered: {[i.get('friendly_name') for i in imgs]}")
    print(f"user {user} sees {[i.get('friendly_name') for i in imgs]}")

    kasm_id, cid = launch(tok, img["image_id"])
    print(f"session 1 running: kasm {kasm_id}, container {cid[:12]}")
    print(sh("docker", "inspect", cid, "--format",
             "{{range .Mounts}}{{.Source}} -> {{.Destination}}{{println}}{{end}}").strip())
    stamp = dt.datetime.now(dt.timezone.utc).strftime("%Y%m%dT%H%M%SZ")
    rel = f"work/kasm-persistence-proof/{stamp}.txt"
    sh("docker", "exec", "-u", "1000", cid, "sh", "-c",
       f"mkdir -p ~/work/kasm-persistence-proof && echo 'written in Kasm session {kasm_id} at {stamp}' > ~/{rel}")
    h1 = sh("docker", "exec", cid, "sha256sum", f"/home/kasm-user/{rel}").split()[0]
    for probe in ("command -v synaptic claude ttyd", "id -un"):
        print(f"$ {probe}:", sh("docker", "exec", "-u", "1000", cid, "sh", "-c", probe).strip().replace("\n", " "))
    print("destroy 1:", call("/destroy_kasm", {**tok, "kasm_id": kasm_id}).get("error_message") or "ok")
    time.sleep(10)

    kasm_id2, cid2 = launch(tok, img["image_id"])
    print(f"session 2 running: kasm {kasm_id2}, container {cid2[:12]} (new container: {cid2 != cid})")
    h2 = sh("docker", "exec", cid2, "sha256sum", f"/home/kasm-user/{rel}").split()[0]
    print(f"marker sha256 session1={h1[:16]} session2={h2[:16]} -> {'PERSISTED' if h1 == h2 else 'MISMATCH'}")
    sh("docker", "exec", "-u", "1000", cid2, "rm", f"/home/kasm-user/{rel}")  # purge the test marker
    print("destroy 2:", call("/destroy_kasm", {**tok, "kasm_id": kasm_id2}).get("error_message") or "ok")
    return 0 if h1 == h2 else 1


if __name__ == "__main__":
    sys.exit(main())

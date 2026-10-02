#!/usr/bin/env python3
"""End every Kasm session of the owner user (msalem): the proof runs open sessions and must not leave them running.
Runs ON ovh-files as root; credentials from /data/probata/secrets/kasm/kasm.env, never printed.
Byline: Claude Code · Opus 5.5 · 2026-10-02
"""
import json, re, ssl, urllib.request

env = {}
for line in open("/data/probata/secrets/kasm/kasm.env", encoding="utf-8"):
    m = re.match(r"^\s*([A-Z_]+)\s*=\s*(.*?)\s*$", line)
    if m:
        env[m.group(1)] = m.group(2)
ctx = ssl._create_unverified_context()


def call(path, body):
    r = urllib.request.Request("https://127.0.0.1:8443/api" + path, data=json.dumps(body).encode(), method="POST",
                               headers={"Content-Type": "application/json"})
    return json.load(urllib.request.urlopen(r, context=ctx, timeout=120))


a = call("/authenticate", {"username": env["KASM_OWNER_USER"], "password": env["KASM_OWNER_PASSWORD"],
                           "logout_other_sessions": False})
tok = {"token": a["token"], "username": env["KASM_OWNER_USER"], "user_id": a["user_id"]}
kasms = call("/get_user_kasms", dict(tok)).get("kasms", [])
for k in kasms:
    r = call("/destroy_kasm", {**tok, "kasm_id": k["kasm_id"]})
    print("ended", k["kasm_id"], (k.get("image") or {}).get("friendly_name"), r.get("error_message") or "ok")
print(f"{len(kasms)} session(s) ended")

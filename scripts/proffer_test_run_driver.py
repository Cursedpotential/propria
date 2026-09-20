# Byline: Claude Code · Fable 5.1 · 2026-09-20 — pick a handler for a waiting TEST run, then poll.
# usage: select_handler.py <preview_handle> <recommended|alternative>
import json
import sys
import time
import urllib.request

BASE = "https://workbench.tilapia-skilift.ts.net/api/proffer"
handle, which = sys.argv[1], sys.argv[2]


def get(path):
    return json.load(urllib.request.urlopen(f"{BASE}/{path}?mode=TEST", timeout=30))


p = get(f"previews/{handle}")
print("phase:", p["phase"])
if p["phase"] == "awaiting_handler_selection":
    h = p["recommended_handler"] if which == "recommended" else p["alternative_handlers"][0]
    body = {k: h[k] for k in ("handler_id", "handler_version", "execution_path", "compatibility_ref")}
    body["recommendation_ref"] = p["handler_recommendation_ref"]
    req = urllib.request.Request(
        f"{BASE}/previews/{handle}/handler-selection?mode=TEST",
        data=json.dumps(body).encode(),
        headers={"content-type": "application/json"},
        method="POST",
    )
    print("selected", body["handler_id"], "->", urllib.request.urlopen(req, timeout=60).status)
last = None
for _ in range(int(sys.argv[3]) if len(sys.argv) > 3 else 8):
    time.sleep(10)
    d = get(f"operations/{handle}")
    s = d["stages"][-1]
    line = f"{d['lifecycle']} | wait={d.get('wait')} | done={d['completed_stage_count']} | {s['stage']} {s['status']} {s['reason'][-280:]}"
    if line != last:
        print(line)
        last = line
    if d.get("terminal") or d.get("wait"):
        break
print("final phase:", get(f"previews/{handle}")["phase"])

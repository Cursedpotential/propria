"""Pick a small review set of bouts, one per kind, so the owner can check Opus's tone marks without reading all 645.

Byline: Claude Code · Opus 5.5 · 2026-09-24. Owner 04:24: "Grab 5."
Picks are deterministic. Each has 8–40 messages with both people talking, and the picks come from different
months where possible.

Usage: python pick_review_set.py <bouts.jsonl> <raw/bout_tone dir> <out.json>
"""

import json
import pathlib
import sys

bouts_path, tone_dir, out_path = pathlib.Path(sys.argv[1]), pathlib.Path(sys.argv[2]), pathlib.Path(sys.argv[3])
bouts = {b["bout_id"]: b for b in (json.loads(x) for x in bouts_path.read_text(encoding="utf-8").split("\n") if x.strip())}
tone = {}
for p in tone_dir.glob("c2024-*.json"):
    d = json.loads(p.read_text(encoding="utf-8"))
    if d.get("ok"):
        tone[d["bout_id"]] = d["output"]

WARM, HARSH = {"affectionate", "friendly", "neutral"}, {"tense", "hostile"}


def share(t, names):
    n = sum(s["to_i"] - s["from_i"] + 1 for s in t["stretches"])
    return sum(s["to_i"] - s["from_i"] + 1 for s in t["stretches"] if s["tone"] in names) / n


KINDS = [
    ("A fight that flares fast",
     lambda t: any(x["speed"] == "abrupt" and x["to_tone"] == "hostile" for x in t["shifts"]),
     lambda t: share(t, {"hostile"})),
    ("Warm and loving",
     lambda t: share(t, {"affectionate"}) >= 0.5,
     lambda t: share(t, {"affectionate"})),
    ("Making up after conflict",
     lambda t: any(x["from_tone"] in HARSH and x["to_tone"] in {"conciliatory", "affectionate"} for x in t["shifts"]),
     lambda t: share(t, {"conciliatory", "affectionate"})),
    ("Hurt or pleading",
     lambda t: share(t, {"distressed"}) >= 0.4,
     lambda t: share(t, {"distressed"})),
    ("Friendly that turns suddenly",
     lambda t: t["stretches"][0]["tone"] in WARM
     and any(x["speed"] == "abrupt" and x["from_tone"] in WARM and x["to_tone"] in HARSH for x in t["shifts"]),
     lambda t: share(t, WARM)),
]

picked, months = [], set()
for why, keep, score in KINDS:
    cands = [bid for bid, t in tone.items()
             if 8 <= bouts[bid]["n_messages"] <= 40 and bouts[bid]["n_matt"] and bouts[bid]["n_katrina"]
             and bid not in {p["bout_id"] for p in picked} and keep(t)]
    cands.sort(key=lambda bid: (bouts[bid]["day"][:7] in months, -score(tone[bid]), bid))
    if cands:
        bid = cands[0]
        months.add(bouts[bid]["day"][:7])
        picked.append({"bout_id": bid, "why": why, "day": bouts[bid]["day"], "n_messages": bouts[bid]["n_messages"],
                       "candidates": len(cands)})

out_path.write_text(json.dumps(picked, indent=1), encoding="utf-8")
for p in picked:
    print(f"{p['bout_id']}  {p['day']}  {p['n_messages']:>3} msgs  {p['why']}  (of {p['candidates']})")

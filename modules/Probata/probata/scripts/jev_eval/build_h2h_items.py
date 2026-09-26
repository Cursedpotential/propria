"""Build head-to-head items v2: the owner's reviewed stretches with his own notes, plus clearly hostile stretches.

Byline: Claude Code · Opus 5.5 · 2026-09-25. Owner 00:31: "It was easily the worst year of my entire life. You didn't put
a clearly hostile or bad message in there ... this isn't a mix." Owner 00:29-00:41: the page's numbers made no sense; his
own review is what a model's answer is read against, so each reviewed item carries his words (owner_note), not a summary.
- Reviewed items: items v1, with owner_note from the Tone Disagreements page (artifact db "conflicts") and the Bout Review
  page ("bouts"), saved by Artifact read_db as <notes dir>/<collection>/<bout id>.json.
- Hostile items: h2h_hostile_stretches.sql output (longest Opus "hostile" stretch per bout). Ranked by how much of the
  bout Opus marked hostile; at most one per day; a mix of who drove it (Katrina, Matt, both); never a bout already in the
  test or a worked example. The focus is the first 30 messages of that stretch (the bouts run 300-800 messages).
All inputs and the output hold case text or the owner's notes: they live on the devbox, never in git.
    .venv/bin/python code/build_h2h_items.py prompts/h2h_items_v1.json prompts/owner_notes hostile.jsonl \
        prompts/block_examples_v2.json prompts/h2h_items_v2.json [--n 10]
"""

import argparse
import json
import pathlib

CALL = {"jev": "Jev was right (tense)", "opus": "Opus was right (neutral)", "": "no call either way"}
BOUT_CALL = {"right": "Opus's reading was right", "missed": "Opus missed something"}
QUOTA = {"Katrina": 3, "Matt": 3, "both": 4}  # a mix of who drove the hostile stretch; filled in rank order


def note_for(bout_id: str, notes: pathlib.Path) -> str:
    parts = []
    f = notes / "conflicts" / f"{bout_id}.json"
    if f.exists():
        d = json.loads(f.read_text(encoding="utf-8"))
        d = d.get("data", d)
        parts.append(f"Tone Disagreements (Opus said {d.get('opus', '?')}, Jev said {d.get('jev', '?')}): "
                     f"your call: {CALL.get(d.get('verdict') or '', d.get('verdict'))}."
                     + (f" Your note: {d['note'].strip()}" if (d.get("note") or "").strip() else ""))
    f = notes / "bouts" / f"{bout_id}.json"
    if f.exists():
        d = json.loads(f.read_text(encoding="utf-8"))
        d = d.get("data", d)
        parts.append(f"Bout Review: {BOUT_CALL.get(d.get('verdict') or '', d.get('verdict') or 'no call')}."
                     + (f" Your note: {d['note'].strip()}" if (d.get("note") or "").strip() else ""))
    return " ".join(parts)


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("items_v1")
    ap.add_argument("notes_dir")
    ap.add_argument("hostile_jsonl")
    ap.add_argument("examples")
    ap.add_argument("out")
    ap.add_argument("--n", type=int, default=10)
    a = ap.parse_args()
    notes = pathlib.Path(a.notes_dir)
    items = json.loads(pathlib.Path(a.items_v1).read_text(encoding="utf-8"))
    for it in items:
        it["group"] = "reviewed"
        it["owner_note"] = note_for(it["bout_id"], notes)
    taken = {it["bout_id"] for it in items} | {e["bout_id"] for e in json.loads(pathlib.Path(a.examples).read_text(encoding="utf-8"))}
    picks, days, have = [], set(), {k: 0 for k in QUOTA}
    rows = [json.loads(x) for x in pathlib.Path(a.hostile_jsonl).read_text(encoding="utf-8").split("\n") if x.strip()]
    for r in rows:
        drv = r["driver"] if r["driver"] in QUOTA else "both"
        if r["bout_id"] in taken or r["day"] in days or have[drv] >= QUOTA[drv]:
            continue
        fi, ti = r["from_i"], min(r["to_i"], r["from_i"] + 29)
        picks.append({"id": f"{r['bout_id']}-h{fi}", "bout_id": r["bout_id"], "source": "c2024", "group": "clearly hostile",
                      "focus_i": [fi, ti], "owner_note": "",
                      "why": (f"Opus marked {r['bout_hostile_msgs']} of this conversation's {r['bout_msgs']} messages hostile; "
                              f"this is the start of its longest hostile stretch ({r['n']} messages), driven by {r['driver']}.")})
        days.add(r["day"])
        have[drv] += 1
        if len(picks) >= a.n:
            break
    out = items + picks
    pathlib.Path(a.out).write_text(json.dumps(out, ensure_ascii=False, indent=1), encoding="utf-8")
    print(f"{len(items)} reviewed ({sum(1 for it in items if it['owner_note'])} with your notes) + {len(picks)} clearly hostile "
          f"(Katrina {have['Katrina']}, Matt {have['Matt']}, both {have['both']}) -> {a.out}")
    for p in picks:
        print(f"  {p['id']:<24} {p['why']}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

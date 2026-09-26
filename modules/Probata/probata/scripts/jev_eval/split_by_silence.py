"""How often each model starts a new conversation, by the silence before a message (are the splits following the clock?).

Byline: Claude Code · Opus 5.5 · 2026-09-24. Owner 22:29: a 30-minute (or longer) wait does not end a conversation, it is
a prompting issue. On the v2 window run the models split at 62% of silences of an hour or more and at 1% of gaps under
5 minutes. This prints that table for any head-to-head dir, all models together and per model, so a prompt change can be
checked against it. Reads only the run's own files (_run.json texts and each model's episodes).
Runs in the ovh-files devbox from the jev-eval dir:
    .venv/bin/python code/split_by_silence.py raw/h2h_v3 [--per-model]
"""

import collections
import datetime
import json
import pathlib
import re
import sys

CLASSES = ["under 5 min", "5-30 min", "30-60 min", "1 h or more"]


def cls(minutes: float) -> str:
    return CLASSES[0] if minutes < 5 else CLASSES[1] if minutes < 30 else CLASSES[2] if minutes < 60 else CLASSES[3]


def times(text: str) -> list[datetime.datetime]:
    out = []
    for ln in text.split("\n"):
        m = re.match(r"^\[(\d+)\] (\S+) (\d\d:\d\d) ", ln)
        if m:
            out.append(datetime.datetime.fromisoformat(m.group(2) + "T" + m.group(3)))
    return out


def main() -> int:
    d = pathlib.Path(sys.argv[1])
    per_model = "--per-model" in sys.argv
    run = json.loads((d / "_run.json").read_text(encoding="utf-8"))
    ts_by = {b: times(v[0]) for b, v in run["texts"].items()}
    tables = collections.defaultdict(lambda: (collections.Counter(), collections.Counter()))
    for f in sorted(d.glob("*/*.json")):
        r = json.loads(f.read_text(encoding="utf-8"))
        if not r.get("ok"):
            continue
        ts = ts_by[r["bout_id"]]
        starts = {e.get("from_i") for e in r["output"].get("episodes", [])}
        for key in ("ALL", f.parent.name) if per_model else ("ALL",):
            gaps, splits = tables[key]
            for j in range(1, len(ts)):
                c = cls((ts[j] - ts[j - 1]).total_seconds() / 60)
                gaps[c] += 1
                splits[c] += j in starts
    for key, (gaps, splits) in sorted(tables.items(), key=lambda kv: kv[0] != "ALL"):
        print(key + ": new conversation started, by the silence before the message")
        for c in CLASSES:
            if gaps[c]:
                print(f"   {c:<12} {splits[c]:>4} of {gaps[c]:>5}  {round(100 * splits[c] / gaps[c]):>3}%")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

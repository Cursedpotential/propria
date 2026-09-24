"""Export Jev per-bout answers (bouts_jev.py, raw/jev_bouts/) as CSV rows for raw_duck.msg_bout_labels_20260924.

Byline: Claude Code · Opus 5.5 · 2026-09-24. Owner ~13:15: send the Jev run; results into the catalog.
Same columns as export_bout_labels.py, so Consignatio casebible/tools/msg_bout_labels_upsert_20260924.sql loads it:
bout_id, pass (= question set version, bout-q-v1), model, prompt_sha256 (= questions_sha256), labelled_at, ok,
output (JSON), raw_ref. One row per bout: a long bout sent as several pieces keeps every piece's answers in
output.windows, in order; output.main_tone is the choice of the piece covering the most messages.
    python export_jev_bout_labels.py raw/jev_bouts > jev_labels.csv
"""

import collections
import csv
import json
import pathlib
import sys

d = pathlib.Path(sys.argv[1])
by_bout = collections.defaultdict(list)
for p in sorted(d.glob("*-b*.json")):
    r = json.loads(p.read_text(encoding="utf-8"))
    by_bout[r["bout_id"]].append((r, p.name))

w = csv.writer(sys.stdout)
for bout_id, items in sorted(by_bout.items()):
    items.sort(key=lambda x: x[0]["from_i"])
    windows, ok, model = [], True, ""
    for r, _name in items:
        answers = None
        try:
            resp = json.loads(r.get("response_raw") or "")
            answers = resp.get("answers")
            model = resp.get("model") or model
        except (ValueError, TypeError):
            pass
        ok = ok and r.get("http_status") == 200 and answers is not None
        windows.append({"window_id": r["window_id"], "from_i": r["from_i"], "to_i": r["to_i"],
                        "n_messages": r["n_messages"], "answers": answers})
    biggest = max(windows, key=lambda x: x["n_messages"])
    main = ((biggest["answers"] or {}).get("main_tone") or {}).get("choice")
    first = items[0][0]
    w.writerow([bout_id, first["question_set_version"], model or first.get("model", ""), first["questions_sha256"],
                min(r["request_ts"] for r, _ in items), "true" if ok else "false",
                json.dumps({"main_tone": main, "windows": windows}, ensure_ascii=False),
                f"persist/jev-eval/{d.as_posix().rstrip('/')}/{bout_id}*.json"])

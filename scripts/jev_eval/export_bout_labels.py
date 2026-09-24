"""Export Opus bout-pass results as CSV rows for the catalog table raw_duck.msg_bout_labels_20260924.

Byline: Claude Code · Opus 5.5 · 2026-09-24. Owner 06:35: bouts and their labels live in permanent tables.
One job: read raw/<pass dir>/<bout_id>.json files and write CSV to stdout. Loading and upserting is done by
Consignatio casebible/tools/msg_bout_labels_upsert_20260924.sql.
    python export_bout_labels.py raw/bout_tone raw/bout_discover > labels.csv
Columns: bout_id, pass, model, prompt_sha256, labelled_at, ok, output (JSON), raw_ref
"""

import csv
import json
import pathlib
import sys

w = csv.writer(sys.stdout)
for d in sys.argv[1:]:
    for p in sorted(pathlib.Path(d).glob("*-b*.json")):
        r = json.loads(p.read_text(encoding="utf-8"))
        w.writerow([r["bout_id"], r.get("version", ""), r.get("model", ""), r.get("prompt_sha256", ""),
                    r.get("request_ts") or "", "true" if r.get("ok") else "false",
                    json.dumps(r.get("output"), ensure_ascii=False) if r.get("output") is not None else "",
                    f"persist/jev-eval/{d.rstrip('/')}/{p.name}"])

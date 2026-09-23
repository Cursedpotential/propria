"""Spread table + 10 random rows for the Jev sample checkpoint (handoff Phase 2).

Byline: Claude Code · Opus 5.5 · 2026-09-23. Reads the JSONL sample; writes nothing.
"""
import collections
import json
import random
import sys

rows = [json.loads(line) for line in open(sys.argv[1], encoding="utf-8") if line.strip()]
c = collections.Counter
print(f"rows={len(rows)} unique_ids={len({r['msg_id'] for r in rows})} unique_texts={len({r['text'] for r in rows})}")
print("\n== stratum x direction x busy_day")
for k, v in sorted(c((r["stratum"], r["direction"], "busy" if r["busy_day"] else "quiet") for r in rows).items()):
    print(f"  {k[0]:<17} {k[1]:<9} {k[2]:<6} {v}")
print("\n== stratum x year")
by = collections.defaultdict(c)
for r in rows:
    by[r["stratum"]][r["year"]] += 1
for s in sorted(by):
    print(f"  {s:<17} " + "  ".join(f"{y}:{n}" for y, n in sorted(by[s].items())))
lens = sorted(len(r["text"]) for r in rows)
ctx = c(len(r["prior_context"]) for r in rows)
print(f"\n== text length chars: min {lens[0]} median {lens[len(lens)//2]} p95 {lens[int(len(lens)*.95)]} max {lens[-1]}")
print(f"== prior_context sizes: {dict(sorted(ctx.items()))}")
print(f"== provenance: source_sha1 {sum(1 for r in rows if r['source_sha1'])}/{len(rows)}  source_file {sum(1 for r in rows if r['source_file'])}/{len(rows)}")
print(f"== tz_status: {dict(c(r['tz_status'] for r in rows))}")
print("\n== 10 random rows (seed 20260923)")
for r in random.Random(20260923).sample(rows, 10):
    t = r["text"].replace("\n", " ")
    print(f"- [{r['stratum']}] {r['ts_utc'][:16]} {r['sender_label']:>7}: {t[:180]}{'…' if len(t) > 180 else ''}  (context {len(r['prior_context'])})")

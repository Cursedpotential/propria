"""Jev per-bout run: one Decisions API call per bout (or per piece of a long bout), question set bout-q-v1.

Byline: Claude Code · Opus 5.5 · 2026-09-24. Owner 04:43: Jev picks from a list and gives a probability for every
option; 04:44 "sure" to setting up the per-bout run. Nothing is sent to Jev without --send.
Answer key = Opus bout-tone-v1 labels (owner spot-check 2026-09-24: 5 of 5 right overall). Scoring is a separate step.

Runs in the ovh-files devbox, key passed with `docker exec --env-file /data/probata/secrets/jev-eval/openrouter.env`:
    .venv/bin/python code/bouts_jev.py bouts/c2024_bouts_v2.jsonl --dry-run
    .venv/bin/python code/bouts_jev.py bouts/c2024_bouts_v2.jsonl --send [--only N] [--max-cost 1.00]
Standard library only (plus the tone wording shared with bouts_tone_opus.py).
"""

import concurrent.futures
import datetime
import hashlib
import json
import os
import pathlib
import sys
import threading
import time
import urllib.error
import urllib.request

from bouts_tone_opus import TONES  # same tone wording Opus labelled with

ENDPOINT = "https://openrouter.ai/api/v1/systemone"
MODEL = "typesafe/jev-1.13"
VERSION = "bout-q-v1"
WORK = pathlib.Path(os.environ.get("JEV_WORK", "/home/kasm-user/persist/jev-eval"))
MAX_STATE_CHARS = 60_000  # ~15k tokens: state plus the longest question stays well under the 32K OpenRouter limit
OVERLAP = 5  # messages shared between neighbouring pieces of a long bout
CONCURRENCY = 8
RETRIES = 3

ABOUT = ("Text messages between Matt and Katrina, co-parents of a young daughter, from one day in 2024. "
         "The messages are consecutive, with no gap longer than 30 minutes.")

WARM = "warmth: affection, love, care, support, missing each other, or friendly joking"
QUESTIONS = {
    "main_tone": {"type": "choice", "instructions": "Which tone best describes this exchange of text messages as a whole?",
                  "criteria": dict(TONES)},
    "any_warm": {"type": "noul", "instructions": f"Does any message in this exchange express {WARM}?",
                 "criteria": {"true": f"At least one message expresses {WARM}", "false": "No message expresses warmth"}},
    "any_conflict": {"type": "noul", "instructions": "Does any message in this exchange show tension or hostility?",
                     "criteria": {"true": "At least one message is irritated, strained, defensive, curt, insulting, threatening, contemptuous or demeaning",
                                  "false": "No message shows tension or hostility"}},
    "any_hostile": {"type": "noul", "instructions": "Does any message in this exchange contain hostility?",
                    "criteria": {"true": "At least one message contains insults, threats, contempt, demeaning language or yelling in text",
                                 "false": "No message is hostile; irritation alone does not count"}},
    "any_distress": {"type": "noul", "instructions": "Does anyone in this exchange express distress?",
                     "criteria": {"true": "Someone expresses hurt, crying, pleading, fear or being overwhelmed",
                                  "false": "No one expresses distress"}},
    "any_conciliatory": {"type": "noul", "instructions": "Does anyone in this exchange try to repair things after conflict?",
                         "criteria": {"true": "Someone apologises, makes up, calms things down or reassures after conflict",
                                      "false": "No repair after conflict"}},
    "tone_changes": {"type": "noul", "instructions": "Does the emotional tone change during this exchange?",
                     "criteria": {"true": "The tone at some point differs from the tone before it",
                                  "false": "One tone holds throughout"}},
    "abrupt_change": {"type": "noul", "instructions": "Does the emotional tone flip suddenly during this exchange?",
                      "criteria": {"true": "The tone flips within a message or two",
                                   "false": "The tone holds, or changes only gradually"}},
    "conflict_driver": {"type": "choice", "instructions": "Who drives the tension or hostility in this exchange?",
                        "criteria": {"Matt": "Matt's messages start or escalate it",
                                     "Katrina": "Katrina's messages start or escalate it",
                                     "both": "Both start or escalate it",
                                     "no conflict": "There is no tension or hostility"}},
}
QUESTIONS_SHA256 = hashlib.sha256((ABOUT + json.dumps(QUESTIONS, sort_keys=True)).encode()).hexdigest()


def windows(bout: dict) -> list[dict]:
    """The bout as one window, or as overlapping pieces when its messages exceed MAX_STATE_CHARS."""
    msgs = [{"time": m["ts_local"], "from": m["who"], "text": m["text"]} for m in bout["messages"]]
    sizes = [len(json.dumps(m, ensure_ascii=False)) + 1 for m in msgs]
    if sum(sizes) <= MAX_STATE_CHARS:
        return [{"window_id": bout["bout_id"], "from_i": 0, "to_i": len(msgs) - 1, "messages": msgs}]
    out, start = [], 0
    while start < len(msgs):
        end, total = start, 0
        while end < len(msgs) and total + sizes[end] <= MAX_STATE_CHARS:
            total += sizes[end]
            end += 1
        end = max(end, start + 1)
        out.append({"window_id": f"{bout['bout_id']}-p{len(out) + 1:02d}", "from_i": start, "to_i": end - 1,
                    "messages": msgs[start:end]})
        if end >= len(msgs):
            break
        start = max(end - OVERLAP, start + 1)
    return out


def body(win: dict) -> dict:
    return {"model": MODEL, "state": {"about": ABOUT, "messages": win["messages"]}, "questions": QUESTIONS}


spent_lock, spent = threading.Lock(), {"cost": 0.0, "calls": 0}


def send(win: dict, bout: dict, key: str, outdir: pathlib.Path, max_cost: float) -> str:
    fname = outdir / f"{win['window_id']}.json"
    if fname.exists() and json.loads(fname.read_text(encoding="utf-8")).get("http_status") == 200:
        return "skip"
    with spent_lock:
        if spent["cost"] >= max_cost:
            return "cap"
    req_body = body(win)
    for attempt in range(1, RETRIES + 1):
        req = urllib.request.Request(ENDPOINT, data=json.dumps(req_body, ensure_ascii=False).encode(), method="POST",
                                     headers={"Authorization": "Bearer " + key, "Content-Type": "application/json"})
        t0 = time.monotonic()
        try:
            with urllib.request.urlopen(req, timeout=120) as resp:
                status, raw = resp.status, resp.read().decode("utf-8", "replace")
        except urllib.error.HTTPError as e:
            status, raw = e.code, e.read().decode("utf-8", "replace")
        except (urllib.error.URLError, TimeoutError) as e:
            status, raw = 0, repr(e)
        record = {"window_id": win["window_id"], "bout_id": bout["bout_id"], "day": bout["day"],
                  "from_i": win["from_i"], "to_i": win["to_i"], "n_messages": len(win["messages"]),
                  "question_set_version": VERSION, "questions_sha256": QUESTIONS_SHA256, "provider": "openrouter",
                  "model": MODEL, "attempt": attempt, "http_status": status,
                  "request_ts": datetime.datetime.now(datetime.timezone.utc).isoformat(),
                  "latency_ms": round((time.monotonic() - t0) * 1000), "response_raw": raw}
        fname.write_text(json.dumps(record, ensure_ascii=False, indent=1), encoding="utf-8")
        if status == 200:
            cost = json.loads(raw).get("usage", {}).get("cost") or 0.0
            with spent_lock:
                spent["cost"] += cost
                spent["calls"] += 1
            return "ok"
        if status not in (0, 429) and status < 500:
            return f"http {status}"
        time.sleep(5 * attempt)
    return f"http {status}"


def main() -> None:
    bouts = [json.loads(x) for x in pathlib.Path(sys.argv[1]).read_text(encoding="utf-8").split("\n") if x.strip()]
    if "--only" in sys.argv:
        bouts = bouts[: int(sys.argv[sys.argv.index("--only") + 1])]
    max_cost = float(sys.argv[sys.argv.index("--max-cost") + 1]) if "--max-cost" in sys.argv else 1.00
    work = [(w, b) for b in bouts for w in windows(b)]
    chars = [len(json.dumps(body(w)["state"], ensure_ascii=False)) for w, _ in work]
    q_chars = len(json.dumps(QUESTIONS, ensure_ascii=False))
    est_tokens = sum(c + q_chars for c in chars) / 3.5
    print(f"{VERSION} questions {QUESTIONS_SHA256[:12]}: {len(bouts)} bouts -> {len(work)} windows "
          f"({sum(1 for w, _ in work if '-p' in w['window_id'])} are pieces of long bouts); "
          f"largest state {max(chars):,} chars; est. {est_tokens / 1e6:.2f}M input tokens, ~${est_tokens / 1e6 * 0.042:.2f}", flush=True)
    if "--send" not in sys.argv:
        print("dry run: nothing sent", flush=True)
        return
    key = os.environ.get("OPENROUTER_API_KEY", "").strip().strip("\"'")
    if not key:
        raise SystemExit("OPENROUTER_API_KEY not in environment (pass --env-file)")
    outdir = WORK / "raw" / "jev_bouts"
    outdir.mkdir(parents=True, exist_ok=True)
    (outdir / "_questions.json").write_text(json.dumps({"version": VERSION, "about": ABOUT, "questions": QUESTIONS,
                                                        "questions_sha256": QUESTIONS_SHA256, "model": MODEL}, indent=1), encoding="utf-8")
    print(f"start={datetime.datetime.now(datetime.timezone.utc).isoformat()}", flush=True)
    results = {}
    with concurrent.futures.ThreadPoolExecutor(CONCURRENCY) as pool:
        futs = {pool.submit(send, w, b, key, outdir, max_cost): w["window_id"] for w, b in work}
        for f in concurrent.futures.as_completed(futs):
            r = f.result()
            results[r] = results.get(r, 0) + 1
            if r not in ("ok", "skip"):
                print(f"{r:>8} {futs[f]}", flush=True)
    print(f"done: {results}; spent ${spent['cost']:.4f} over {spent['calls']} calls", flush=True)
    print(f"end={datetime.datetime.now(datetime.timezone.utc).isoformat()}", flush=True)


if __name__ == "__main__":
    main()

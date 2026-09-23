"""Jev eval Phase 0: one OpenRouter call with a synthetic state (no case data).

Byline: Claude Code · Opus 5.5 · 2026-09-23. Handoff: docs/handoffs/HANDOFF-2026-09-23-jev-tier1-eval.md
Raw request and response are written to <workdir>/raw/phase0/. The key is read from an env file and
never printed or logged. Standard library only.
"""

import datetime
import json
import os
import pathlib
import re
import sys
import time
import urllib.error
import urllib.request

ENDPOINT = "https://openrouter.ai/api/v1/systemone"
MODEL = "typesafe/jev-1.13"
WORKDIR = pathlib.Path(sys.argv[1] if len(sys.argv) > 1 else "/data/probata/jev-eval")
KEYFILE = pathlib.Path("/data/probata/secrets/jev-eval/openrouter.env")


def read_key() -> str:
    for line in KEYFILE.read_text(encoding="utf-8").splitlines():
        m = re.match(r"^\s*OPENROUTER_API_KEY\s*=\s*(.+?)\s*$", line)
        if m:
            return m.group(1).strip().strip("\"'")
    raise SystemExit("OPENROUTER_API_KEY missing from key file")


request_body = {
    "model": MODEL,
    "state": {
        "target_message": {
            "sender": "Parent A",
            "ts": "2030-01-01T12:00:00Z",
            "text": "You are not taking her this weekend unless you pay me first.",
        }
    },
    "questions": {
        "q_denial": {
            "type": "noul",
            "instructions": "Does the target message refuse, cancel, shorten, delay, or put conditions on a parent's time or contact with the child?",
            "criteria": {
                "true": "Parenting time or contact is blocked, reduced, delayed, or conditioned",
                "false": "Time or contact proceeds, or isn't discussed",
            },
        },
        "q_financial": {
            "type": "noul",
            "instructions": "Is the target message about child support, payments, expenses, or money?",
            "criteria": {"true": "Money is a subject", "false": "Money isn't mentioned"},
        },
        "register": {
            "type": "choice",
            "instructions": "What kind of statement is the target message primarily?",
            "criteria": {
                "factual": "verifiable facts or logistics",
                "opinion": "judgments or beliefs",
                "emotional": "feelings or venting",
                "mixed": "substantial blend",
            },
        },
    },
}

out = WORKDIR / "raw" / "phase0"
out.mkdir(parents=True, exist_ok=True)
stamp = datetime.datetime.now(datetime.timezone.utc).strftime("%Y%m%dT%H%M%SZ")
req = urllib.request.Request(
    ENDPOINT,
    data=json.dumps(request_body).encode(),
    method="POST",
    headers={"Authorization": "Bearer " + read_key(), "Content-Type": "application/json"},
)
t0 = time.monotonic()
try:
    with urllib.request.urlopen(req, timeout=60) as resp:
        status, raw = resp.status, resp.read().decode("utf-8", "replace")
except urllib.error.HTTPError as e:
    status, raw = e.code, e.read().decode("utf-8", "replace")
latency_ms = round((time.monotonic() - t0) * 1000)
record = {
    "request_ts": stamp,
    "provider": "openrouter",
    "endpoint": ENDPOINT,
    "model": MODEL,
    "question_set_version": "phase0-smoke",
    "http_status": status,
    "latency_ms": latency_ms,
    "request": request_body,
    "response_raw": raw,
}
path = out / f"smoke_{stamp}.json"
path.write_text(json.dumps(record, indent=1), encoding="utf-8")
print(f"status={status} latency_ms={latency_ms} saved={path}")
print(raw[:1500])

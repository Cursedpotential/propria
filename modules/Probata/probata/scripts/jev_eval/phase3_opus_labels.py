"""Jev eval Phase 3: Opus reference labels for the 300-message sample (question set tags-v0).

Byline: Claude Code · Opus 5.5 · 2026-09-23. Handoff: docs/handoffs/HANDOFF-2026-09-23-jev-tier1-eval.md

- Model claude-opus-5-5 through the Claude Agent SDK, authenticated by the owner's long-lived
  CLAUDE_CODE_OAUTH_TOKEN (owner 2026-09-23 14:15). The SDK has no temperature setting (see RUNLOG).
- Tag wording and criteria are exactly handoff section 2. Opus gets the same state as Jev cell B
  (target + up to 5 prior messages, marked) and labels the TARGET only.
- No tools, no project settings, a strict JSON schema; each raw result is saved per message.
- Resumable: a message with a saved successful result is skipped. `--repeat` re-labels a seeded 10%
  sample into a separate folder for the self-consistency check.

Usage (inside the ovh-files devbox, from the work dir, token passed with docker exec --env-file):
    .venv/bin/python code/phase3_opus_labels.py sample/sample_v2.jsonl            # main pass
    .venv/bin/python code/phase3_opus_labels.py sample/sample_v2.jsonl --repeat   # 10% repeat pass
"""

import asyncio
import dataclasses
import datetime
import hashlib
import json
import os
import pathlib
import random
import sys
import time

from claude_agent_sdk import ClaudeAgentOptions, ResultMessage, query

MODEL = "claude-opus-5-5"
QUESTION_SET = "tags-v0"
CONCURRENCY = 4
RETRIES = 3
# Runs in the ovh-files devbox (owner 14:33: reuse the existing box). The work dir is the devbox's
# persistent home folder; override with JEV_WORK.
WORK = pathlib.Path(os.environ.get("JEV_WORK", "/home/kasm-user/persist/jev-eval"))

# Handoff section 2, verbatim.
GATE = [
    ("case_relevant",
     "Does the target message relate to the child, parenting time, custody, co-parenting, money for the child, or court proceedings?",
     "Touches any of those subjects", "Unrelated small talk, spam, or other topics"),
]
CONTENT = [
    ("parenting_time_denial",
     "Does the target message refuse, cancel, shorten, delay, or put conditions on a parent's time or contact with the child?",
     "Parenting time or contact is blocked, reduced, delayed, or conditioned", "Time or contact proceeds, or isn't discussed"),
    ("child_referenced", "Does the target message mention or discuss the child (Kailah)?",
     "The child is named, nicknamed, or clearly the subject", "No reference to the child"),
    ("info_gatekeeping",
     "Does the target message withhold or refuse information about the child's school, health, location, or activities?",
     "Information is refused, deflected, or conditioned", "Information is shared, or no such request exists"),
    ("hostility_threat", "Does the target message contain insults, threats, intimidation, or demeaning language?",
     "Any insult, threat, intimidation, or contempt", "Neutral or civil, even if disagreeing"),
    ("parent_disparagement", "Does the target message criticize or run down a parent's character or fitness as a parent?",
     "Negative claims about a parent's character or parenting", "No character or fitness attack"),
    ("financial", "Is the target message about child support, payments, expenses, or money?",
     "Money is a subject", "Money isn't mentioned"),
    ("logistics", "Is the target message about scheduling, pickup, drop-off, or exchange arrangements?",
     "Times, places, or arrangements are discussed", "No scheduling content"),
    ("legal_reference",
     "Does the target message mention court, attorneys, orders, filings, police, or the Friend of the Court?",
     "Any legal-system reference", "None"),
    ("third_party",
     "Does the target message involve a third party in the child's life, such as a partner, relative, or Mike Joubran?",
     "A non-parent person is involved or discussed", "Only the two parents and the child"),
    ("child_wellbeing", "Does the target message address the child's health, safety, school, or emotional state?",
     "Wellbeing is a subject", "Not addressed"),
    ("cooperation_offer", "Does the target message offer flexibility, a compromise, or extra time?",
     "A cooperative offer is made", "No offer"),
    ("admission", "Does the sender of the target message admit, concede, or apologize for something?",
     "Admission, concession, or apology", "None"),
]
EXPERIMENTAL = [
    ("reframes_prior_event", "Does the target message deny or recast something that happened earlier in the conversation?",
     "Contradicts or reinterprets a prior event", "No such reframing"),
    ("blame_shift", "Does the target message shift responsibility for a problem onto the other person?",
     "Responsibility is redirected", "No redirection"),
]
REGISTER = ("register", "What kind of statement is the target message primarily?", {
    "factual": "verifiable facts or logistics",
    "opinion": "judgments or beliefs",
    "emotional": "feelings or venting",
    "mixed": "substantial blend",
})
BOOL_TAGS = [t[0] for t in GATE + CONTENT + EXPERIMENTAL]


def tag_block() -> str:
    lines = []
    for group, tags in (("Gate", GATE), ("Content tags", CONTENT), ("Experimental flags", EXPERIMENTAL)):
        lines.append(f"## {group}")
        for tid, ins, t, f in tags:
            lines.append(f"- `{tid}`: {ins}\n  - true: {t}\n  - false: {f}")
    lines.append(f"## Choice\n- `{REGISTER[0]}`: {REGISTER[1]}")
    lines += [f"  - `{k}`: {v}" for k, v in REGISTER[2].items()]
    return "\n".join(lines)


SYSTEM_PROMPT = f"""You produce reference labels for evaluating a message classifier. You read one message
exchange and answer a fixed set of questions about the TARGET message.

Rules:
- Label only `target_message`. `prior_messages` (when present) are earlier messages in the same
  conversation, given for context only; a tag is true only if the target message itself meets it.
- Apply each question's "true" and "false" criteria literally. If the target does not meet the "true"
  criterion, the answer is false. Do not reward or penalize either sender.
- Judge what the message says. Do not decide who is right, who is credible, or what is true.
- Sender, date and platform are given facts; do not question them.
- For every tag you mark true, give a one-line rationale that points to the words in the target
  message. Give no rationale for false tags.
- `register` is exactly one of the listed options.

Questions (question set {QUESTION_SET}):
{tag_block()}

Return only the structured output."""

SCHEMA = {
    "type": "object",
    "properties": {
        **{t: {"type": "boolean"} for t in BOOL_TAGS},
        "register": {"type": "string", "enum": list(REGISTER[2])},
        "rationales": {
            "type": "object",
            "properties": {t: {"type": "string"} for t in BOOL_TAGS},
            "additionalProperties": False,
        },
    },
    "required": [*BOOL_TAGS, "register", "rationales"],
    "additionalProperties": False,
}
PROMPT_SHA256 = hashlib.sha256((SYSTEM_PROMPT + json.dumps(SCHEMA, sort_keys=True)).encode()).hexdigest()


def state_for(row: dict) -> dict:
    return {
        "target_message": {"sender": row["sender_label"], "ts": row["ts_utc"], "text": row["text"]},
        "prior_messages": [{"sender": p["sender"], "ts": p["ts"], "text": p["text"]} for p in row["prior_context"]],
        "platform": row["platform"],
    }


def to_jsonable(obj):
    if dataclasses.is_dataclass(obj):
        return {"_type": type(obj).__name__, **{k: to_jsonable(v) for k, v in dataclasses.asdict(obj).items()}}
    if isinstance(obj, dict):
        return {k: to_jsonable(v) for k, v in obj.items()}
    if isinstance(obj, (list, tuple)):
        return [to_jsonable(v) for v in obj]
    if isinstance(obj, (str, int, float, bool)) or obj is None:
        return obj
    return repr(obj)


def check(labels: dict) -> list[str]:
    """Structural validation of one label set (rationale for every true tag, none for false ones)."""
    problems = []
    rats = labels.get("rationales", {})
    for t in BOOL_TAGS:
        if labels.get(t) is True and not rats.get(t, "").strip():
            problems.append(f"true without rationale: {t}")
        if labels.get(t) is False and rats.get(t, "").strip():
            problems.append(f"rationale on false tag: {t}")
    return problems


async def label_one(row: dict, outdir: pathlib.Path, sem: asyncio.Semaphore) -> None:
    fname = outdir / (hashlib.sha1(row["msg_id"].encode()).hexdigest() + ".json")
    if fname.exists() and json.loads(fname.read_text(encoding="utf-8")).get("ok"):
        return
    state = state_for(row)
    async with sem:
        for attempt in range(1, RETRIES + 1):
            started = datetime.datetime.now(datetime.timezone.utc).isoformat()
            t0 = time.monotonic()
            messages, result = [], None
            try:
                opts = ClaudeAgentOptions(
                    model=MODEL, system_prompt=SYSTEM_PROMPT, tools=[], allowed_tools=[],
                    setting_sources=[], max_turns=4, cwd=str(WORK / "cwd"),
                    output_format={"type": "json_schema", "schema": SCHEMA},
                )
                async for m in query(prompt=json.dumps(state, ensure_ascii=False), options=opts):
                    messages.append(to_jsonable(m))
                    if isinstance(m, ResultMessage):
                        result = m
                ok = bool(result and result.subtype == "success" and result.structured_output)
            except Exception as e:  # recorded, then retried
                messages.append({"_exception": repr(e)})
                ok = False
            record = {
                "ok": ok, "msg_id": row["msg_id"], "stratum": row["stratum"], "attempt": attempt,
                "request_ts": started, "latency_ms": round((time.monotonic() - t0) * 1000),
                "model": MODEL, "question_set_version": QUESTION_SET, "prompt_sha256": PROMPT_SHA256,
                "state": state,
                "labels": result.structured_output if ok else None,
                "label_problems": check(result.structured_output) if ok else None,
                "session_id": getattr(result, "session_id", None), "sdk_messages": messages,
            }
            fname.write_text(json.dumps(record, ensure_ascii=False, indent=1), encoding="utf-8")
            if ok:
                print(f"ok  {row['stratum']:<16} {record['latency_ms']:>6} ms  {fname.name}", flush=True)
                return
            print(f"ERR {row['stratum']:<16} attempt {attempt}  {fname.name}", flush=True)
            await asyncio.sleep(10 * attempt)


async def main() -> None:
    sample = pathlib.Path(sys.argv[1])
    repeat = "--repeat" in sys.argv
    rows = [json.loads(line) for line in sample.read_text(encoding="utf-8").splitlines() if line.strip()]
    if repeat:
        rows = random.Random(20260923).sample(rows, round(len(rows) * 0.10))
    outdir = WORK / "raw" / ("opus_repeat" if repeat else "opus")
    outdir.mkdir(parents=True, exist_ok=True)
    (WORK / "cwd").mkdir(exist_ok=True)
    (outdir / "_prompt.json").write_text(json.dumps(
        {"system_prompt": SYSTEM_PROMPT, "schema": SCHEMA, "prompt_sha256": PROMPT_SHA256, "model": MODEL}, indent=1),
        encoding="utf-8")
    print(f"{'repeat' if repeat else 'main'} pass: {len(rows)} messages, prompt {PROMPT_SHA256[:12]}", flush=True)
    sem = asyncio.Semaphore(CONCURRENCY)
    await asyncio.gather(*(label_one(r, outdir, sem) for r in rows))
    done = [json.loads(p.read_text(encoding="utf-8")) for p in outdir.glob("*.json") if not p.name.startswith("_")]
    ok = [d for d in done if d.get("ok")]
    out = WORK / ("labels_opus_repeat.jsonl" if repeat else "labels_opus.jsonl")
    with out.open("w", encoding="utf-8") as fh:
        for d in ok:
            fh.write(json.dumps({"msg_id": d["msg_id"], "stratum": d["stratum"], "session_id": d["session_id"],
                                 "prompt_sha256": d["prompt_sha256"], "label_problems": d["label_problems"],
                                 **d["labels"]}, ensure_ascii=False) + "\n")
    print(f"done: {len(ok)}/{len(rows)} ok, {sum(1 for d in ok if d['label_problems'])} with label problems -> {out}", flush=True)


if __name__ == "__main__":
    asyncio.run(main())

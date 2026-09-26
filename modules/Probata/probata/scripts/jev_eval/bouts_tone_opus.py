"""Bout pilot: Opus marks tone stretches and tone shifts inside each day-bout (2024 texts from Katrina's phone).

Byline: Claude Code · Opus 5.5 · 2026-09-24.
Owner rulings this follows:
- 2026-09-23 21:06: judge natural bouts within the day and catch rapid shifts.
- 2026-09-24 03:18: "try a and a": 30-minute bouts, tone marks the shifts.
- 2026-09-24 03:19: flag normal, healthy, loving behaviour as carefully as conflict; the whole relationship
  cycle has to be captured.
- 2026-09-23 09:30: Opus does the classification.
Records describe conduct only. Interpreting the cycle is a later step, per the owner's own recipe.

Runs in the ovh-files devbox (see Propria/docs/reference/DEVBOX-ON-OVH-FILES.md), token via --env-file.
    .venv/bin/python code/bouts_tone_opus.py bouts/c2024_bouts_v2.jsonl [--only N]
"""

import asyncio
import dataclasses
import datetime
import hashlib
import json
import os
import pathlib
import sys
import time

from claude_agent_sdk import ClaudeAgentOptions, ResultMessage, query

MODEL = "claude-opus-5-5"
VERSION = "bout-tone-v1"
CONCURRENCY = 4
RETRIES = 3
WORK = pathlib.Path(os.environ.get("JEV_WORK", "/home/kasm-user/persist/jev-eval"))

TONES = {
    "affectionate": "warm, loving, caring, flirtatious, supportive, missing each other",
    "friendly": "light, joking, casual, easy",
    "neutral": "plain logistics or information, no emotional colour",
    "tense": "irritated, strained, defensive, curt, guarded",
    "hostile": "insults, threats, contempt, demeaning language, yelling in text",
    "distressed": "hurt, crying, pleading, afraid, overwhelmed",
    "conciliatory": "apologising, making up, calming things down, reassuring after conflict",
}

SYSTEM_PROMPT = f"""You mark tone in one stretch of text messages between Matt and Katrina, who are co-parents.
Their daughter was born around January 2020; there was no child before then.

You receive one "bout": a run of messages from one day with no silence longer than 30 minutes. Each
message has an index `i`, a local time, the sender, and its text.

Your job:
1. Split the bout into consecutive tone stretches. Every message belongs to exactly one stretch; stretches
   are contiguous, in order, and cover the bout from the first index to the last.
2. Mark every point where the tone changes between stretches, with its speed ("abrupt" when it flips within
   a message or two, "gradual" otherwise) and the index of the message that turns it.
3. Say who is driving each stretch: Matt, Katrina, or both.

Tone labels (use only these):
{chr(10).join(f"- {k}: {v}" for k, v in TONES.items())}

Rules:
- Warm, loving, normal and healthy moments matter exactly as much as conflict. Mark them just as carefully.
- Describe conduct in plain words ("asks where he is", "says she loves him", "calls him a liar"). Do not
  diagnose or characterize people (no "manipulative", "narcissistic", "gaslighting", "abusive").
- Judge only what is in the messages. Don't guess at what happened outside them.
- A bout with one message is one stretch with no shifts.
- Notes are one short line each and quote or point to the actual words.

Return only the structured output."""

SCHEMA = {
    "type": "object",
    "properties": {
        "stretches": {"type": "array", "items": {"type": "object", "properties": {
            "from_i": {"type": "integer"}, "to_i": {"type": "integer"},
            "tone": {"type": "string", "enum": list(TONES)},
            "driver": {"type": "string", "enum": ["Matt", "Katrina", "both"]},
            "note": {"type": "string"}}, "required": ["from_i", "to_i", "tone", "driver", "note"],
            "additionalProperties": False}},
        "shifts": {"type": "array", "items": {"type": "object", "properties": {
            "at_i": {"type": "integer"}, "from_tone": {"type": "string", "enum": list(TONES)},
            "to_tone": {"type": "string", "enum": list(TONES)}, "speed": {"type": "string", "enum": ["abrupt", "gradual"]},
            "trigger_i": {"type": "integer"}, "note": {"type": "string"}},
            "required": ["at_i", "from_tone", "to_tone", "speed", "trigger_i", "note"], "additionalProperties": False}},
        "summary": {"type": "string"},
    },
    "required": ["stretches", "shifts", "summary"],
    "additionalProperties": False,
}
PROMPT_SHA256 = hashlib.sha256((SYSTEM_PROMPT + json.dumps(SCHEMA, sort_keys=True)).encode()).hexdigest()


def to_jsonable(obj):
    if dataclasses.is_dataclass(obj):
        return {"_type": type(obj).__name__, **{k: to_jsonable(v) for k, v in dataclasses.asdict(obj).items()}}
    if isinstance(obj, dict):
        return {k: to_jsonable(v) for k, v in obj.items()}
    if isinstance(obj, (list, tuple)):
        return [to_jsonable(v) for v in obj]
    return obj if isinstance(obj, (str, int, float, bool)) or obj is None else repr(obj)


def check(out: dict, n: int) -> list[str]:
    """Structural checks: stretches contiguous and complete, shifts inside the bout."""
    problems, expect = [], 0
    for s in out["stretches"]:
        if s["from_i"] != expect or s["to_i"] < s["from_i"]:
            problems.append(f"stretch gap/overlap at {s['from_i']}")
        expect = s["to_i"] + 1
    if expect != n:
        problems.append(f"stretches end at {expect - 1}, bout has {n} messages")
    for sh in out["shifts"]:
        if not (0 <= sh["at_i"] < n and 0 <= sh["trigger_i"] < n):
            problems.append(f"shift index out of range {sh['at_i']}/{sh['trigger_i']}")
    return problems


async def label_bout(bout: dict, outdir: pathlib.Path, sem: asyncio.Semaphore) -> None:
    fname = outdir / f"{bout['bout_id']}.json"
    if fname.exists() and json.loads(fname.read_text(encoding="utf-8")).get("ok"):
        return
    msgs = [{"i": i, "time": m["ts_local"], "who": m["who"], "text": m["text"]} for i, m in enumerate(bout["messages"])]
    state = {"day": bout["day"], "bout": bout["bout_id"], "messages": msgs}
    async with sem:
        for attempt in range(1, RETRIES + 1):
            t0, messages, result = time.monotonic(), [], None
            try:
                opts = ClaudeAgentOptions(model=MODEL, system_prompt=SYSTEM_PROMPT, tools=[], allowed_tools=[],
                                          setting_sources=[], max_turns=4, cwd=str(WORK / "cwd"),
                                          output_format={"type": "json_schema", "schema": SCHEMA})
                async for m in query(prompt=json.dumps(state, ensure_ascii=False), options=opts):
                    messages.append(to_jsonable(m))
                    if isinstance(m, ResultMessage):
                        result = m
                ok = bool(result and result.subtype == "success" and result.structured_output)
            except Exception as e:
                messages.append({"_exception": repr(e)})
                ok = False
            out = result.structured_output if ok else None
            record = {"ok": ok, "bout_id": bout["bout_id"], "day": bout["day"], "n_messages": len(msgs),
                      "attempt": attempt, "request_ts": datetime.datetime.now(datetime.timezone.utc).isoformat(),
                      "latency_ms": round((time.monotonic() - t0) * 1000), "model": MODEL, "version": VERSION,
                      "prompt_sha256": PROMPT_SHA256, "output": out,
                      "problems": check(out, len(msgs)) if ok else None,
                      "session_id": getattr(result, "session_id", None), "sdk_messages": messages}
            fname.write_text(json.dumps(record, ensure_ascii=False, indent=1), encoding="utf-8")
            if ok:
                print(f"ok  {bout['bout_id']} {len(msgs):>4} msgs {record['latency_ms']:>6} ms problems={len(record['problems'])}", flush=True)
                return
            print(f"ERR {bout['bout_id']} attempt {attempt}", flush=True)
            await asyncio.sleep(10 * attempt)


async def main() -> None:
    # split on "\n" only: str.splitlines() also breaks on U+2028/U+0085 etc. that occur inside message text
    bouts = [json.loads(line) for line in pathlib.Path(sys.argv[1]).read_text(encoding="utf-8").split("\n") if line.strip()]
    if "--only" in sys.argv:
        k = int(sys.argv[sys.argv.index("--only") + 1])
        pick = [b for b in bouts if b["n_katrina"] and b["n_matt"]]
        bouts = sorted(pick, key=lambda b: b["n_messages"])[len(pick) // 2 - k // 2: len(pick) // 2 - k // 2 + k]
    outdir = WORK / "raw" / "bout_tone"
    outdir.mkdir(parents=True, exist_ok=True)
    (WORK / "cwd").mkdir(exist_ok=True)
    (outdir / "_prompt.json").write_text(json.dumps({"system_prompt": SYSTEM_PROMPT, "schema": SCHEMA,
                                                     "prompt_sha256": PROMPT_SHA256, "model": MODEL}, indent=1), encoding="utf-8")
    print(f"bouts: {len(bouts)}, prompt {PROMPT_SHA256[:12]}", flush=True)
    sem = asyncio.Semaphore(CONCURRENCY)
    await asyncio.gather(*(label_bout(b, outdir, sem) for b in bouts))
    done = [json.loads(p.read_text(encoding="utf-8")) for p in outdir.glob("c2024-*.json")]
    ok = [d for d in done if d.get("ok")]
    print(f"done: {len(ok)} ok of {len(done)} files; {sum(1 for d in ok if d['problems'])} with structural problems", flush=True)


if __name__ == "__main__":
    asyncio.run(main())

"""Discovery pass: Opus marks tone stretches and shifts (same view as bout-tone-v1) and records the acts it sees in the
chunk, naming new categories freely. The category list is guidance, not a closed set.

Byline: Claude Code · Opus 5.5 · 2026-09-24. Owner rulings this follows:
- 06:28: same visualization, but don't lock the categories; let it discover them. Give it guidance on what to look
  for, so the categories it finds can make the other years easier.
- 06:31: beware reactive behaviour. Read what came before; don't look at a situation shallowly.
- 06:32: the prompt stays neutral. It looks for behaviours in both people and gets no background on either person.
- Owner recipe: records describe conduct and quote the messages; no diagnoses or character labels.
- 06:45 (v2): minimize psychoanalysis and judgment. Classify narrowly, only what is in the chunk: no looking backward
  or forward, no provocation or blame calls, no reading of either person as a whole. Interpretation is a later step.
  v1 (125 Facebook bouts, raw/bout_discover/) is kept as a record.
Runs in the ovh-files devbox (Propria/docs/reference/DEVBOX-ON-OVH-FILES.md); token via --env-file.
    .venv/bin/python code/bouts_discover_opus.py bouts/<file>.jsonl [--only N] [--ids id1,id2]
"""

import asyncio
import datetime
import hashlib
import json
import os
import pathlib
import sys
import time

from claude_agent_sdk import ClaudeAgentOptions, ResultMessage, query

from bouts_tone_opus import CONCURRENCY, MODEL, RETRIES, TONES, check, to_jsonable

VERSION = "bout-discover-v2"
WORK = pathlib.Path(os.environ.get("JEV_WORK", "/home/kasm-user/persist/jev-eval"))

GUIDE = """What to record. Record observable acts: what a message says or does, in plain words. This is a guide,
not a closed list: use these names when they fit, and name any other act you see as a new category, with a
one-line definition of the act.

Questions, accusations, answers
- asks_whereabouts: asks where someone is, who they are with, or what they are doing
- accusation: accuses the other person of something (say what, in the note)
- denial: denies an accusation, or denies saying or doing something
- contradiction_in_chunk: a statement conflicts with another statement inside this chunk
- unanswered_question: a question the other person does not answer inside this chunk
- says_other_caused_it: says the other person caused their feelings or actions ("you made me")

Pressure and conflict
- insult_or_swearing: insults, name-calling or swearing at the other person
- threat: states a consequence if something happens or doesn't (court, police, leaving, exposure, the child)
- demand: tells the other person to do something
- refusal: refuses a request
- repeated_messages: several messages in a row with no reply in between

The child
- child_logistics: pickups, schedules, school, meals, belongings
- child_wellbeing: the child's health, safety, feelings or needs
- child_contact_condition: sets a condition on seeing or talking to the child
- child_time_refused_or_changed: parenting time or contact refused, cancelled, shortened or moved
- child_in_argument: the child is brought into an argument between the adults

Money and property
- money: money, payments, bills or property mentioned, asked for or refused

Warmth and repair
- affection: love, affection, missing each other, flirting
- care_or_support: concern, help or support offered
- thanks_or_praise: thanks, credit or a compliment
- humour: joking, teasing in a friendly way
- apology: says sorry or admits a mistake
- reassurance: calms the other person or makes peace
- promise: states a commitment ("I will…")
- agreement_or_plan: agrees, or makes or confirms a plan

Health and events
- mental_health_mentioned: mood, anxiety, medication, diagnosis or crisis mentioned (quote it; never diagnose)
- call_or_contact_event: a call, missed call, unsent message or reaction notice"""

SYSTEM_PROMPT = f"""You read one chunk of messages between Matt and Katrina, who are co-parents of a daughter born around
January 2020. The chunk is one "bout": messages from one day with no silence longer than 30 minutes. Each message has
an index `i`, a local time, the sender, and its text. Platform notices (calls, missed calls, unsent messages,
reactions) are messages too.

Do three things.
1. Tone: split the chunk into consecutive tone stretches that cover every message, mark every point where the tone
   changes (abrupt or gradual, and the message where it changes), and give who sent most of the messages in each
   stretch.
2. Observations: record each act you see, citing the message indices and quoting the key words.
3. New categories: list every category you used that is not in the guide, with a one-line definition.

Tone labels (use only these for tone):
{chr(10).join(f"- {k}: {v}" for k, v in TONES.items())}

{GUIDE}

Stay narrow:
- Classify only what is in this chunk. Do not look backward or forward, and do not guess at history, motives,
  intentions or what happened outside these messages.
- Describe acts, not people. No psychological reading, no diagnosis, no judgment of either person's character, and
  no judgment of who provoked whom or who is to blame.
- Treat both people the same way. Any act can come from either of them.
- Warm, loving, normal and healthy acts matter exactly as much as conflict. Record them just as carefully.
- Say how sure you are that the act is there; "low" is fine.
- A chunk with one message still gets one stretch; it may have no observations.
- Keep notes to one short line each. The summary says what happens in one or two plain sentences, without judgment.

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
        "patterns": {"type": "array", "items": {"type": "object", "properties": {
            "category": {"type": "string"},
            "from_guide": {"type": "boolean"},
            "who": {"type": "string", "enum": ["Matt", "Katrina", "both"]},
            "message_is": {"type": "array", "items": {"type": "integer"}},
            "quote": {"type": "string"},
            "note": {"type": "string"},
            "confidence": {"type": "string", "enum": ["low", "medium", "high"]}},
            "required": ["category", "from_guide", "who", "message_is", "quote", "note", "confidence"],
            "additionalProperties": False}},
        "new_categories": {"type": "array", "items": {"type": "object", "properties": {
            "name": {"type": "string"}, "definition": {"type": "string"}},
            "required": ["name", "definition"], "additionalProperties": False}},
        "summary": {"type": "string"},
    },
    "required": ["stretches", "shifts", "patterns", "new_categories", "summary"],
    "additionalProperties": False,
}
PROMPT_SHA256 = hashlib.sha256((SYSTEM_PROMPT + json.dumps(SCHEMA, sort_keys=True)).encode()).hexdigest()


def check_patterns(out: dict, n: int) -> list[str]:
    """Tone checks from bout-tone-v1, plus: every cited index is inside the bout."""
    problems = check(out, n)
    for p in out["patterns"]:
        if not p["message_is"] or not all(0 <= i < n for i in p["message_is"]):
            problems.append(f"pattern {p['category']} cites messages outside the bout")
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
                      "problems": check_patterns(out, len(msgs)) if ok else None,
                      "session_id": getattr(result, "session_id", None), "sdk_messages": messages}
            fname.write_text(json.dumps(record, ensure_ascii=False, indent=1), encoding="utf-8")
            if ok:
                print(f"ok  {bout['bout_id']} {len(msgs):>4} msgs {record['latency_ms']:>6} ms "
                      f"patterns={len(out['patterns'])} new={len(out['new_categories'])} problems={len(record['problems'])}",
                      flush=True)
                return
            print(f"ERR {bout['bout_id']} attempt {attempt}", flush=True)
            await asyncio.sleep(10 * attempt)


async def main() -> None:
    # split on "\n" only: str.splitlines() also breaks on U+2028/U+0085 etc. that occur inside message text
    bouts = [json.loads(line) for line in pathlib.Path(sys.argv[1]).read_text(encoding="utf-8").split("\n") if line.strip()]
    if "--ids" in sys.argv:
        want = set(sys.argv[sys.argv.index("--ids") + 1].split(","))
        bouts = [b for b in bouts if b["bout_id"] in want]
    if "--only" in sys.argv:
        k = int(sys.argv[sys.argv.index("--only") + 1])
        pick = [b for b in bouts if b["n_katrina"] and b["n_matt"]]
        bouts = sorted(pick, key=lambda b: b["n_messages"])[len(pick) // 2 - k // 2: len(pick) // 2 - k // 2 + k]
    outdir = WORK / "raw" / "bout_discover_v2"
    outdir.mkdir(parents=True, exist_ok=True)
    (WORK / "cwd").mkdir(exist_ok=True)
    (outdir / "_prompt.json").write_text(json.dumps({"system_prompt": SYSTEM_PROMPT, "schema": SCHEMA, "version": VERSION,
                                                     "prompt_sha256": PROMPT_SHA256, "model": MODEL}, indent=1), encoding="utf-8")
    print(f"start={datetime.datetime.now(datetime.timezone.utc).isoformat()} bouts: {len(bouts)}, prompt {PROMPT_SHA256[:12]}", flush=True)
    sem = asyncio.Semaphore(CONCURRENCY)
    await asyncio.gather(*(label_bout(b, outdir, sem) for b in bouts))
    ids = {b["bout_id"] for b in bouts}
    done = [json.loads((outdir / f"{i}.json").read_text(encoding="utf-8")) for i in ids if (outdir / f"{i}.json").exists()]
    ok = [d for d in done if d.get("ok")]
    print(f"done: {len(ok)} ok of {len(ids)}; {sum(1 for d in ok if d['problems'])} with structural problems", flush=True)
    print(f"end={datetime.datetime.now(datetime.timezone.utc).isoformat()}", flush=True)


if __name__ == "__main__":
    asyncio.run(main())

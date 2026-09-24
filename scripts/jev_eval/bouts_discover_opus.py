"""Discovery pass: Opus marks tone stretches and shifts (same view as bout-tone-v1) and records the behaviour patterns it
finds, naming new categories freely. The category list is guidance, not a closed set.

Byline: Claude Code · Opus 5.5 · 2026-09-24. Owner rulings this follows:
- 06:28: same visualization, but don't lock the categories; let it discover them. Give it guidance on what to look
  for, so the categories it finds can make the other years easier.
- 06:31: beware reactive behaviour. Read what came before; don't look at a situation shallowly.
- 06:32: the prompt stays neutral. It looks for behaviours in both people and gets no background on either person.
- Owner recipe: records describe conduct and quote the messages; no diagnoses or character labels.
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

VERSION = "bout-discover-v1"
WORK = pathlib.Path(os.environ.get("JEV_WORK", "/home/kasm-user/persist/jev-eval"))

GUIDE = """Behaviours to look for. This is a guide, not a closed list: use these names when they fit, and name anything
else you notice as a new category, in your own words, with a one-line definition.

Control and pressure
- contact_conditions: conditions put on seeing or talking to the child ("only if…", "not until…")
- time_withheld: parenting time or contact refused, cancelled, cut short or delayed
- info_gatekeeping: information about the child held back or used as leverage
- threats: threats of court, police, exposure, leaving, taking the child, or harm
- money_pressure: money demanded, withheld, or used as leverage
- monitoring: checking up on, tracking, or demanding accounts of the other person's whereabouts or contacts

The child in the conflict
- child_as_leverage: the child used as a bargaining chip or a messenger
- child_drawn_in: the child pulled into adult conflict, or told about it
- disparaging_parent: the other parent run down to or around the child

Truthfulness
- contradiction: a statement contradicted by an earlier or later message
- denial_of_record: denying something said or done that the messages show ("I never said that")
- darvo_sequence: in answer to a complaint, deny it, attack the complainer, and claim to be the victim
- deflection: changing the subject, whataboutism, or answering a different question
- blame_shift: responsibility put on the other person
- lying_cheating_stealing: lies, infidelity, or theft alleged, admitted, or shown

The relationship cycle
- warmth: affection, love, care, support, humour, missing each other
- tension_building: growing irritation, strain, curtness
- blowup: an outburst
- repair: an apology, making up, calming things down, reassurance
- promise: a promise or commitment (note whether it is later kept or broken, if the messages show it)
- love_bombing: intense affection or promises right after a conflict
- withdrawal: silent treatment, ignoring, going quiet as pressure

Health and stress
- mental_health_mentioned: mood, medication, diagnosis, crisis or trauma mentioned (record what is said; never diagnose)
- mental_health_as_weapon: someone's mental health used to dismiss, insult or discredit them
- support_given: help or support offered when someone is struggling

Healthy co-parenting
- cooperation: plans made and kept, logistics handled smoothly, flexibility, credit given

Reactions (read every heated message against what came before it)
- provocation: a message that sets off a reaction (a threat, a taunt, an accusation, a lie, being ignored)
- reaction: a message that answers a provocation; set reaction_to_i to the message it answers
- reaction_used_against: a reaction later held up as proof against the person who reacted ("look how you talk to me")"""

SYSTEM_PROMPT = f"""You read one stretch of messages between Matt and Katrina, who are co-parents of a daughter born around
January 2020. You receive one "bout": messages from one day with no silence longer than 30 minutes. Each message has
an index `i`, a local time, the sender, and its text. Platform notices (calls, missed calls, unsent messages,
reactions) are messages too.

Do three things.
1. Tone, as before: split the bout into consecutive tone stretches that cover every message, mark every point where the
   tone changes (abrupt or gradual, and the message that turns it), and say who drives each stretch.
2. Patterns: record each behaviour you find, citing the message indices it rests on and quoting the key words.
3. New categories: list every category you used that is not in the guide, with a one-line definition.

Tone labels (use only these for tone):
{chr(10).join(f"- {k}: {v}" for k, v in TONES.items())}

{GUIDE}

How to read:
- Treat both people the same way. Any behaviour can come from either of them. Judge only from these messages, not
  from who you expect to behave which way.
- Read in order and look below the surface. A calm message can be the controlling one; an explosive one can be a
  reaction. Before recording anything heated, look at what came before it. Record a provocation and the reaction to it
  as separate patterns, and link the reaction to what it answers.
- Warm, loving, normal and healthy moments matter exactly as much as conflict. Record them just as carefully.
- Describe conduct in plain words and quote it. Do not diagnose or label people: no "narcissist", "bipolar",
  "manipulative", "abuser", "crazy". Pattern names such as darvo_sequence describe a sequence of messages, never a person.
- Say how sure you are. "low" is fine; a pattern with thin support should say so rather than be left out.
- A bout with one message still gets one stretch; it may have no patterns.
- Keep notes to one short line each.

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
            "reaction_to_i": {"type": "integer"},
            "quote": {"type": "string"},
            "note": {"type": "string"},
            "confidence": {"type": "string", "enum": ["low", "medium", "high"]}},
            "required": ["category", "from_guide", "who", "message_is", "reaction_to_i", "quote", "note", "confidence"],
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
    """Tone checks from bout-tone-v1, plus: every cited index is inside the bout (-1 means 'not a reaction')."""
    problems = check(out, n)
    for p in out["patterns"]:
        if not p["message_is"] or not all(0 <= i < n for i in p["message_is"]):
            problems.append(f"pattern {p['category']} cites messages outside the bout")
        if p["reaction_to_i"] >= n or p["reaction_to_i"] < -1:
            problems.append(f"pattern {p['category']} reaction_to_i out of range")
    return problems


async def label_bout(bout: dict, outdir: pathlib.Path, sem: asyncio.Semaphore) -> None:
    fname = outdir / f"{bout['bout_id']}.json"
    if fname.exists() and json.loads(fname.read_text(encoding="utf-8")).get("ok"):
        return
    msgs = [{"i": i, "time": m["ts_local"], "who": m["who"], "text": m["text"]} for i, m in enumerate(bout["messages"])]
    state = {"day": bout["day"], "bout": bout["bout_id"], "messages": msgs,
             "note": "reaction_to_i is -1 when a pattern is not a reaction to an earlier message"}
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
    outdir = WORK / "raw" / "bout_discover"
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

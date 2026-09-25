"""Block review: one block of consecutive texts (her phone, 2024) read whole by several models with the episode prompt.

Byline: Claude Code · Opus 5.5 · 2026-09-24. Owner 20:46: "review what I have so far ... tweak the prompt and make this
better, because this is a hot mess"; 20:47-20:48: run a block of texts through the newest Gemini Flash and Pro (100
messages each) and through Nemotron 3.5 Lightning on NIM.

Why the prompt changed (the owner's notes on 29 Opus-vs-Jev disagreements, Tone Disagreements page, 2026-09-24 20:21-20:46):
- Chunking destroyed context: 30-minute day bouts cut conversations apart and judged single messages alone ("there was
  more to this conversation", "why was one message evaluated"). -> The model reads the whole block and makes its own
  episodes by topic and flow, not by the clock.
- "Neutral" swallowed kindness, cooperation and planning together ("hasn't marked one good thing"). -> No catch-all
  neutral; positive labels are explicit; "logistics" only when there is no emotional content at all.
- Being upset about someone else read as hostility at her. -> Every label says who, and at whom.
- Her silences/blocks were not flagged. -> Unanswered runs and long gaps are recorded; gaps are shown in the text.
- Multiple labels are expected (owner 06:48); narrow, no psychoanalysis (06:45); neutral prompt with no background on
  either person (06:32).
Runs in the ovh-files devbox. Keys come from --env-file (GEMINI_API_KEY, NVIDIA_API_KEY); never printed.
    python block_review_llm.py run <bouts.jsonl> <start_day> <n_messages> <outdir> <provider:model> [...]
Standard library only.
"""

import datetime
import hashlib
import json
import os
import pathlib
import sys
import time
import urllib.error
import urllib.request

VERSION = "block-episodes-v1"
LABELS = {
    "warm": "affection, love, care, missing each other, compliments",
    "cooperative": "planning, coordinating or working together; getting along; pleasant and cordial",
    "playful": "joking or teasing in good humour",
    "conciliatory": "apology, reassurance, making up, an olive branch",
    "logistics": "purely factual exchange with no emotional content at all (use only when nothing else fits)",
    "edgy": "attitude, sarcasm, curt or cold replies, mild tension",
    "conflict": "argument, accusation or open disagreement between the two of them",
    "hostile": "insults, threats or contempt aimed at the other person",
    "upset": "hurt, anxious, distressed, crying, overwhelmed (about anyone or anything)",
    "pleading": "begging for a reply, for contact or for something; repeated attempts to be heard",
    "stonewalling": "ignoring, refusing to engage, one-word dismissals",
    "child": "about their daughter: her care, schedule, time with a parent, her wellbeing",
    "blocked": "mentions being blocked, cut off, or kept from contact or from the daughter",
}
AT = ["each other", "a third party", "a situation", "self"]

SYSTEM = f"""You read a block of consecutive text messages between Matt and Katrina, who are co-parents of a young
daughter (born around January 2020). Each message has an index [i], a date and local time, and the sender. Long
silences are shown as [— N h no messages —].

Read the whole block first. Then:
1. Split it into episodes: stretches that belong together by topic and conversational flow. Follow the conversation,
   not the clock: a conversation can pause for hours and continue, and one hour can hold two different topics. Every
   message belongs to exactly one episode; episodes are contiguous and cover the block.
2. For each episode give a short topic, a one-to-three sentence summary of what happens, and every label that applies
   (several usually do). For each label: who shows it (Matt, Katrina, both), at whom it is directed (each other, a third
   party, a situation, self), intensity 1-3 (1 slight, 2 clear, 3 strong), the message indexes that show it, and a note
   quoting or pointing to the words.
3. Record silences that matter: one person sending several messages with no reply, or going quiet mid-conversation.

Labels (use only these):
{chr(10).join(f"- {k}: {v}" for k, v in LABELS.items())}

Rules:
- Positive moments matter exactly as much as conflict. Cooperation, planning together, kindness and getting along are
  labels, not "logistics". Use "logistics" only when a stretch has no emotional content at all.
- Say at whom a feeling is directed. Being upset or angry about someone else is not hostility toward the other parent.
- Describe conduct in plain words ("asks where he is", "says she loves him", "calls him a liar"). Do not diagnose or
  characterize people. Judge only what is in these messages.
Return only JSON matching the schema."""

SCHEMA = {
    "type": "object",
    "properties": {
        "episodes": {"type": "array", "items": {"type": "object", "properties": {
            "from_i": {"type": "integer"}, "to_i": {"type": "integer"},
            "topic": {"type": "string"}, "summary": {"type": "string"},
            "labels": {"type": "array", "items": {"type": "object", "properties": {
                "label": {"type": "string", "enum": list(LABELS)},
                "who": {"type": "string", "enum": ["Matt", "Katrina", "both"]},
                "directed_at": {"type": "string", "enum": AT},
                "intensity": {"type": "integer"},
                "evidence_i": {"type": "array", "items": {"type": "integer"}},
                "note": {"type": "string"}},
                "required": ["label", "who", "directed_at", "intensity", "evidence_i", "note"]}},
            "unanswered": {"type": "array", "items": {"type": "object", "properties": {
                "from_i": {"type": "integer"}, "to_i": {"type": "integer"},
                "who": {"type": "string", "enum": ["Matt", "Katrina"]}, "note": {"type": "string"}},
                "required": ["from_i", "to_i", "who", "note"]}}},
            "required": ["from_i", "to_i", "topic", "summary", "labels", "unanswered"]}},
        "overall": {"type": "string"},
    },
    "required": ["episodes", "overall"],
}
PROMPT_SHA256 = hashlib.sha256((SYSTEM + json.dumps(SCHEMA, sort_keys=True)).encode()).hexdigest()

# ---- v2 (owner 20:52: "the model is going to first need some context on the relationship and the case, not so much as
# to give it away or bias anything, but enough so that it knows what it's looking for in terms of the more nuanced
# behaviors. We also are going to need several examples, maybe a bunch of them.") This supersedes the 06:32 "no
# background" rule for this prompt. Background states facts only; the behaviors list applies to BOTH people (owner 06:40:
# flag his own behaviors too). Worked examples hold real messages, so they live in a devbox file (--examples), never git.
LABELS_V2 = dict(LABELS, **{
    "leverage": "contact with the daughter, belongings, pets, money or a place to live used as leverage, or conditions put on them",
    "deflecting": "denying something said earlier, shifting blame, or turning an accusation back on the other person",
    "threat": "threats or ultimatums: leaving, court, police, keeping things, cutting off contact",
    "checking": "asking where the other person is, what they are doing, or who they are with",
})
BACKGROUND_V2 = """Background (facts only. Do not let it decide any label: label only what the messages themselves show):
- Matt and Katrina are the parents of a daughter born around January 2020. They have been together and apart more than
  once, and there is a custody dispute between them.
- At times one of them blocked the other's number; Matt then wrote from other numbers. There were periods when one
  parent could not reach the other or see their daughter.
- These messages are from Katrina's phone, 2024. Money, housing, work, moving and family members come up often.

Behaviors to watch for, from either person alike:
- Who starts an escalation and who reacts to it: when a harsh message answers something said earlier, record which
  message it responds to.
- Warmth, kindness, cooperation and plans made together, including warmth right after a conflict or alongside a request.
  Label it as warmth or cooperation; do not judge whether it was sincere.
- Contact with the daughter, belongings or money used as leverage, or conditions put on them.
- Denying something said earlier in the block, shifting blame, or turning an accusation back on the other person.
- Threats and ultimatums.
- One person writing repeatedly while the other does not answer; blocking or being blocked.
- Checking where the other person is or who they are with."""


def system_v2(examples: list[dict]) -> str:
    base = SYSTEM.replace("Labels (use only these):\n" + chr(10).join(f"- {k}: {v}" for k, v in LABELS.items()),
                          "Labels (use only these):\n" + chr(10).join(f"- {k}: {v}" for k, v in LABELS_V2.items()))
    base = base.replace("   quoting or pointing to the words.",
                        "   quoting or pointing to the words, and responds_to_i: the index of an earlier message this\n"
                        "   one reacts to, or -1 when it does not react to anything in the block.")
    ex = "\n\n".join(f"Example {n} ({e['why']}):\n{e['text']}\nGood labels:\n{e['labels']}" for n, e in enumerate(examples, 1))
    return (BACKGROUND_V2 + "\n\n" + base + ("\n\nWorked examples (from the owner's own review; the style to follow):\n\n" + ex
                                              if examples else ""))


SCHEMA_V2 = json.loads(json.dumps(SCHEMA))
_lab = SCHEMA_V2["properties"]["episodes"]["items"]["properties"]["labels"]["items"]
_lab["properties"]["label"]["enum"] = list(LABELS_V2)
_lab["properties"]["responds_to_i"] = {"type": "integer"}
_lab["required"] = _lab["required"] + ["responds_to_i"]


def build_block(bouts_path: pathlib.Path, start_day: str, n: int) -> tuple[list[dict], str]:
    bouts = [json.loads(x) for x in bouts_path.read_text(encoding="utf-8").split("\n") if x.strip()]
    bouts = sorted((b for b in bouts if b["day"] >= start_day), key=lambda b: b["start_local"])
    msgs, lines, prev = [], [], None
    for b in bouts:
        for m in b["messages"]:
            if len(msgs) >= n:
                break
            ts = datetime.datetime.fromisoformat(b["day"] + "T" + m["ts_local"])
            if prev is not None and (ts - prev).total_seconds() >= 3600:
                lines.append(f"[— {round((ts - prev).total_seconds() / 3600, 1)} h no messages —]")
            i = len(msgs)
            msgs.append({"i": i, "bout_id": b["bout_id"], "day": b["day"], "ts": m["ts_local"], "who": m["who"],
                         "text": m["text"], "msg_id": m.get("msg_id")})
            lines.append(f"[{i}] {b['day']} {m['ts_local']} {m['who']}: {m['text']}")
            prev = ts
    return msgs, "\n".join(lines)


def post(url: str, body: dict, headers: dict, timeout: int = 600) -> dict:
    req = urllib.request.Request(url, data=json.dumps(body).encode(), headers={"content-type": "application/json", **headers})
    for attempt in range(4):
        try:
            with urllib.request.urlopen(req, timeout=timeout) as r:
                return json.loads(r.read().decode())
        except urllib.error.HTTPError as e:
            detail = e.read().decode(errors="replace")[:500]
            if e.code in (429, 500, 502, 503, 504) and attempt < 3:
                time.sleep(20 * (attempt + 1))
                continue
            raise RuntimeError(f"HTTP {e.code}: {detail}")
    raise RuntimeError("retries exhausted")


def gemini_keys() -> list[str]:
    """Every distinct Gemini key in the environment (GEMINI_API_KEY, GOOGLE_API_KEY, GEMINI_API_KEY_2..): a busy model or
    an exhausted quota on one key moves on to the next (2026-09-24: 503 'high demand' and 429 quota on single keys)."""
    names = ["GEMINI_API_KEY", "GOOGLE_API_KEY"] + [f"GEMINI_API_KEY_{i}" for i in range(2, 10)]
    out = []
    for n in names:
        v = os.environ.get(n, "").strip()
        if v and v not in out:
            out.append(v)
    return out


def post_gemini(model: str, body: dict) -> dict:
    last = None
    for key in gemini_keys():
        try:
            return post(f"https://generativelanguage.googleapis.com/v1beta/models/{model}:generateContent", body,
                        {"x-goog-api-key": key})
        except RuntimeError as e:
            last = e
            if "HTTP 503" in str(e) or "HTTP 429" in str(e) or "HTTP 403" in str(e):
                continue
            raise
    raise last or RuntimeError("no Gemini key in the environment")


def call(provider: str, model: str, block: str, system: str, schema: dict) -> tuple[str, dict]:
    if provider == "gemini":
        body = {"systemInstruction": {"parts": [{"text": system}]},
                "contents": [{"role": "user", "parts": [{"text": block}]}],
                "generationConfig": {"responseMimeType": "application/json", "responseSchema": schema, "temperature": 0.2}}
        d = post_gemini(model, body)
        text = "".join(p.get("text", "") for p in d["candidates"][0]["content"]["parts"])
        return text, {"usage": d.get("usageMetadata"), "model_version": d.get("modelVersion")}
    if provider == "gemma":
        # Gemma on the Gemini API: no system instruction or schema mode assumed, so the instructions and the schema go
        # into the one user turn and the JSON is parsed from the reply text (owner 20:52: "for shits and giggles").
        prompt = (SYSTEM + "\n\nJSON schema to follow exactly:\n" + json.dumps(SCHEMA) + "\n\nThe block:\n" + block)
        body = {"contents": [{"role": "user", "parts": [{"text": prompt}]}], "generationConfig": {"temperature": 0.2}}
        d = post_gemini(model, body)
        text = "".join(p.get("text", "") for p in d["candidates"][0]["content"]["parts"])
        return text, {"usage": d.get("usageMetadata"), "model_version": d.get("modelVersion")}
    if provider == "nim":
        body = {"model": model, "temperature": 0.2, "max_tokens": 16000,
                "messages": [{"role": "system", "content": system}, {"role": "user", "content": block}],
                # 2026-09-24: this NIM endpoint rejects nvext.guided_json; the OpenAI-style json_schema response format is used
                "response_format": {"type": "json_schema",
                                    "json_schema": {"name": "episodes", "schema": schema, "strict": False}}}
        d = post("https://integrate.api.nvidia.com/v1/chat/completions", body,
                 {"Authorization": f"Bearer {os.environ['NVIDIA_API_KEY']}"})
        return d["choices"][0]["message"]["content"], {"usage": d.get("usage"), "model_version": d.get("model")}
    raise SystemExit(f"unknown provider {provider}")


def check(out: dict, n: int, labels: dict) -> list[str]:
    problems, expect = [], 0
    for e in out.get("episodes", []):
        if e["from_i"] != expect:
            problems.append(f"episode gap/overlap at {e['from_i']} (expected {expect})")
        expect = e["to_i"] + 1
        for lab in e["labels"]:
            if lab["label"] not in labels:
                problems.append(f"unknown label {lab['label']}")
    if expect != n:
        problems.append(f"episodes end at {expect - 1}, block has {n} messages")
    return problems


def main() -> int:
    if sys.argv[1] != "run":
        raise SystemExit(__doc__)
    args = sys.argv[2:]
    prompt, examples_path = "v1", None
    while args and args[0].startswith("--"):
        flag, val, args = args[0], args[1], args[2:]
        if flag == "--prompt":
            prompt = val
        elif flag == "--examples":
            examples_path = pathlib.Path(val)
    bouts_path, start_day, n, outdir = pathlib.Path(args[0]), args[1], int(args[2]), pathlib.Path(args[3])
    specs = args[4:]
    if prompt == "v2":
        examples = json.loads(examples_path.read_text(encoding="utf-8")) if examples_path else []
        by_id = {b["bout_id"]: b for b in (json.loads(x) for x in bouts_path.read_text(encoding="utf-8").split("\n") if x.strip())}
        for e in examples:
            if "text" not in e:
                b = by_id[e["bout_id"]]
                e["text"] = "\n".join(f"[{i}] {b['day']} {m['ts_local']} {m['who']}: {m['text']}" for i, m in enumerate(b["messages"]))
        system, schema, labels, version = system_v2(examples), SCHEMA_V2, LABELS_V2, "block-episodes-v2"
    else:
        system, schema, labels, version = SYSTEM, SCHEMA, LABELS, VERSION
    sha = hashlib.sha256((system + json.dumps(schema, sort_keys=True)).encode()).hexdigest()
    outdir.mkdir(parents=True, exist_ok=True)
    msgs, block = build_block(bouts_path, start_day, n)
    (outdir / f"_block_{version}.json").write_text(json.dumps({"version": version, "prompt_sha256": sha, "system": system,
                                                               "schema": schema, "messages": msgs, "block_text": block},
                                                              ensure_ascii=False, indent=1), encoding="utf-8")
    print(f"{version} prompt {sha[:12]}: {len(msgs)} messages {msgs[0]['day']} {msgs[0]['ts']} -> "
          f"{msgs[-1]['day']} {msgs[-1]['ts']}, {len(block)} chars", flush=True)
    for spec in specs:
        provider, model = spec.split(":", 1)
        t0 = time.time()
        rec = {"version": version, "prompt_sha256": sha, "provider": provider, "model": model,
               "request_ts": datetime.datetime.now(datetime.timezone.utc).isoformat()}
        try:
            text, meta = call(provider, model, block, system, schema)
            rec.update(meta, raw=text)
            s = text.strip()
            if "```" in s:  # fenced JSON (models without a schema mode), possibly after prose or thinking text
                s = s.split("```", 1)[1].split("\n", 1)[1].rsplit("```", 1)[0]
            elif not s.startswith("{"):
                s = s[s.find("{"): s.rfind("}") + 1]
            rec["output"] = json.loads(s)
            rec["problems"] = check(rec["output"], len(msgs), labels)
            rec["ok"] = True
        except Exception as e:  # recorded, never silent
            rec.update(ok=False, error=f"{type(e).__name__}: {e}"[:2000])
        rec["seconds"] = round(time.time() - t0, 1)
        suffix = "" if version == VERSION else "_" + version
        (outdir / f"{provider}_{model.replace('/', '_')}{suffix}.json").write_text(
            json.dumps(rec, ensure_ascii=False, indent=1), encoding="utf-8")
        eps = len(rec.get("output", {}).get("episodes", [])) if rec.get("ok") else 0
        print(f"{provider}:{model} ok={rec['ok']} episodes={eps} problems={len(rec.get('problems', []))} "
              f"{rec['seconds']}s {rec.get('error', '')[:200]}", flush=True)
    return 0


if __name__ == "__main__":
    sys.exit(main())

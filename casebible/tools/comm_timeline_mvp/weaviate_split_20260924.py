#!/usr/bin/env python3
"""Split Weaviate ChatEvents20260918 into MsgEvents20260918 and AiChatEvents20260918, copying stored vectors.

Byline: Claude Code · Opus 5.5 · 2026-09-24. Owner 2026-09-24: "Chats are with AI. Messages are with people."
Collection names picked by the owner at 09:30: MsgEvents20260918 = messages and calls with people,
AiChatEvents20260918 = conversations with AI. Nothing is re-embedded: each object keeps its UUID, properties and
`text_nim` vector, and gets `record_kind` (message | call | ai_chat). An unknown source_format stops the run.

Stdlib only; run on ovh-files (Weaviate at WEAVIATE_URL). Safe to re-run: the same UUIDs are overwritten.
  python3 weaviate_split_20260924.py copy        # create targets if missing, copy every object
  python3 weaviate_split_20260924.py verify      # per-format counts old vs new, sample vector equality
  python3 weaviate_split_20260924.py probe       # AI-chat vector equality; keyword + vector search on each new collection
  python3 weaviate_split_20260924.py delete-old  # only after verify passes (owner 09:30: delete once counts match)
"""
import json
import os
import random
import sys
import urllib.error
import urllib.request

WV = os.environ.get("WEAVIATE_URL", "http://100.91.190.107:8082").rstrip("/")
OLD = "ChatEvents20260918"
MSG = "MsgEvents20260918"
AI = "AiChatEvents20260918"
PAGE = int(os.environ.get("PAGE", "400"))
VECTOR = "text_nim"

AI_FORMATS = {"gemini_activity_json", "ai_markdown_transcript", "chatgpt_conversations_json",
              "claude_conversations_json", "ai_generic_json", "ai_conversations_json", "ai_chat_file"}
MSG_FORMATS = {"fb_messenger_json", "sms_backup_xml", "imessage_html", "imessage_txt", "google_chat_json",
               "whatsapp_txt", "fb_messenger_html", "google_voice_html", "mbox"}
CALL_FORMATS = {"calls_backup_xml"}
DESCRIPTIONS = {
    MSG: "Messages and calls with people (SMS, Messenger, iMessage, Google Chat, call log), split from "
         "ChatEvents20260918 on 2026-09-24 with vectors copied. Rebuildable from B2 via comm_timeline_mvp.",
    AI: "Conversations with AI (ChatGPT, Claude, Gemini, other AI transcripts), split from ChatEvents20260918 on "
        "2026-09-24 with vectors copied. Rebuildable from B2 via comm_timeline_mvp.",
}


def call(method, path, body=None):
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(WV + path, data=data, method=method, headers={"Content-Type": "application/json"})
    try:
        with urllib.request.urlopen(req, timeout=300) as r:
            raw = r.read()
            return json.loads(raw) if raw else None
    except urllib.error.HTTPError as e:
        if e.code == 404:
            return None
        raise SystemExit(f"{method} {path}: HTTP {e.code} {e.read()[:500]!r}")


def target(obj):
    fmt = (obj.get("properties") or {}).get("source_format")
    if fmt in AI_FORMATS:
        return AI, "ai_chat"
    if fmt in MSG_FORMATS:
        return MSG, "message"
    if fmt in CALL_FORMATS:
        return MSG, "call"
    raise SystemExit(f"unclassified source_format {fmt!r} on {obj.get('id')}; add it to a format set first")


def ensure_targets():
    old = call("GET", f"/v1/schema/{OLD}")
    if not old:
        raise SystemExit(f"{OLD} not found")
    for name in (MSG, AI):
        if call("GET", f"/v1/schema/{name}"):
            continue
        schema = {"class": name, "description": DESCRIPTIONS[name], "properties": old["properties"],
                  "vectorConfig": old["vectorConfig"]}
        for key in ("invertedIndexConfig", "replicationConfig", "shardingConfig", "multiTenancyConfig"):
            if key in old:
                schema[key] = old[key]
        schema.get("shardingConfig", {}).pop("actualCount", None)
        schema.get("shardingConfig", {}).pop("actualVirtualCount", None)
        call("POST", "/v1/schema", schema)
        print("created", name, flush=True)


def pages(cls, with_vector=True):
    after = None
    while True:
        q = f"/v1/objects?class={cls}&limit={PAGE}" + ("&include=vector" if with_vector else "")
        if after:
            q += f"&after={after}"
        got = (call("GET", q) or {}).get("objects") or []
        if not got:
            return
        yield got
        after = got[-1]["id"]


def copy():
    ensure_targets()
    done = 0
    for batch in pages(OLD):
        objs = []
        for o in batch:
            cls, kind = target(o)
            vec = (o.get("vectors") or {}).get(VECTOR)
            if not vec:
                raise SystemExit(f"object {o['id']} has no {VECTOR} vector")
            props = dict(o.get("properties") or {})
            props["record_kind"] = kind
            objs.append({"class": cls, "id": o["id"], "properties": props, "vectors": {VECTOR: vec}})
        res = call("POST", "/v1/batch/objects", {"objects": objs}) or []
        errors = [r for r in res if (r.get("result") or {}).get("errors")]
        if errors:
            raise SystemExit(f"batch errors after {done}: {json.dumps(errors[:3])[:1500]}")
        done += len(objs)
        if done % 10000 < PAGE:
            print("copied", done, flush=True)
    print("copied total", done, flush=True)


def by_format(cls, field="source_format"):
    q = {"query": f'{{Aggregate{{{cls}(groupBy:["{field}"]){{meta{{count}} groupedBy{{value}}}}}}}}'}
    got = (call("POST", "/v1/graphql", q) or {}).get("data", {}).get("Aggregate", {}).get(cls) or []
    return {g["groupedBy"]["value"]: g["meta"]["count"] for g in got}


def verify():
    old, new = by_format(OLD), {MSG: by_format(MSG), AI: by_format(AI)}
    ok = True
    for fmt, n in sorted(old.items(), key=lambda kv: -kv[1]):
        cls, _ = target({"properties": {"source_format": fmt}, "id": "-"})
        got = new[cls].get(fmt, 0)
        ok &= got == n
        print(f"{fmt:32s} old {n:7d}  {cls:22s} {got:7d}  {'ok' if got == n else 'MISMATCH'}")
    for cls in (MSG, AI):
        extra = set(new[cls]) - set(old)
        ok &= not extra
        print(f"{cls:22s} total {sum(new[cls].values()):7d}  record_kind {by_format(cls, 'record_kind')}"
              + (f"  UNEXPECTED FORMATS {sorted(extra)}" if extra else ""))
    ok &= sum(old.values()) == sum(new[MSG].values()) + sum(new[AI].values())
    first = next(pages(OLD, with_vector=False))
    for o in random.sample(first, min(5, len(first))):
        a = call("GET", f"/v1/objects/{OLD}/{o['id']}?include=vector")
        cls, _ = target(a)
        b = call("GET", f"/v1/objects/{cls}/{o['id']}?include=vector")
        same = b is not None and a["vectors"][VECTOR] == b["vectors"][VECTOR]
        ok &= same
        print("vector", o["id"], cls, "same" if same else "DIFFERENT")
    print("VERIFY", "PASS" if ok else "FAIL", flush=True)
    return ok


def probe():
    """AI-chat vectors equal the old ones, and each new collection answers keyword and vector searches."""
    ok = True
    for o in (call("GET", f"/v1/objects?class={AI}&limit=3") or {}).get("objects") or []:
        a = call("GET", f"/v1/objects/{OLD}/{o['id']}?include=vector")
        b = call("GET", f"/v1/objects/{AI}/{o['id']}?include=vector")
        same = a is not None and a["vectors"][VECTOR] == b["vectors"][VECTOR]
        ok &= same
        print("ai vector", o["id"], "same" if same else "DIFFERENT")
    for cls in (MSG, AI):
        kw = call("POST", "/v1/graphql", {"query": f'{{Get{{{cls}(limit:2, bm25:{{query:"custody"}}){{record_kind}}}}}}'})
        seed = call("GET", f"/v1/objects?class={cls}&limit=1&include=vector")["objects"][0]["vectors"][VECTOR]
        near = call("POST", "/v1/graphql", {"query": f'{{Get{{{cls}(limit:2, nearVector:{{vector:{json.dumps(seed)}, '
                                                     f'targetVectors:["{VECTOR}"]}}){{record_kind _additional{{distance}}}}}}}}'})
        hits_kw, hits_near = kw["data"]["Get"][cls], near["data"]["Get"][cls]
        ok &= bool(hits_near)
        print(cls, "keyword hits", len(hits_kw), "vector hits", hits_near)
    print("PROBE", "PASS" if ok else "FAIL", flush=True)
    return ok


def delete_old():
    if not (verify() and probe()):
        raise SystemExit("verify failed; old collection kept")
    call("DELETE", f"/v1/schema/{OLD}")
    print("deleted", OLD, "exists now:", call("GET", f"/v1/schema/{OLD}") is not None)


if __name__ == "__main__":
    {"copy": copy, "verify": verify, "probe": probe, "delete-old": delete_old}[sys.argv[1]]()

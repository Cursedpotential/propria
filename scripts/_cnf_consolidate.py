#!/usr/bin/env python3
"""Consolidate CNF realtime_memories (conservative, lossless).

Removes ONLY: exact-duplicate entries, and obvious NOISE (greetings/thanks/praise
without context, transient state markers). Merges true same-content duplicates
losslessly. Never drops semantically-distinct facts. Leaves manual_memories and
all other keys untouched.
"""

import json
import re
import hashlib

PATH = r"E:/AI_Workspace/Projects/the-platform-workspace/Agno-MCP-Platform/.claude/memories/project_memory.json"

with open(PATH, encoding="utf-8") as f:
    d = json.load(f)

rm = d.get("realtime_memories", [])
orig_count = len(rm)


def text_of(m):
    if isinstance(m, str):
        return m
    if not isinstance(m, dict):
        return json.dumps(m, sort_keys=True, ensure_ascii=False)
    for k in ("content", "message", "text", "summary", "value", "event", "body", "note"):
        v = m.get(k)
        if isinstance(v, str) and v.strip():
            return v
    # fall back to all string values joined
    return " ".join(str(v) for v in m.values() if isinstance(v, str))


NOISE_PATTERNS = [
    r"^\s*(thanks|thank you|thx|ty|cheers|great|awesome|perfect|got it|understood|ok|okay|nice|cool|appreciate[a-z]*)\s*[!.?]*\s*$",
    r"^\s*(hi|hello|hey|yo|gm|good morning|good evening)\s*[!.?]*\s*$",
]
NOISE_RE = [re.compile(p, re.IGNORECASE) for p in NOISE_PATTERNS]

# transient-state markers: keep these only if they also carry a durable fact.
# Conservative: do NOT auto-delete transient state (could lose context); only delete
# exact duplicates and pure greetings/thanks/praise.


def is_pure_noise(txt):
    t = txt.strip()
    if not t:
        return True
    for r in NOISE_RE:
        if r.match(t):
            return True
    return False


def sig(m):
    """Stable signature for exact-duplicate detection (ignores timestamp)."""
    if isinstance(m, dict):
        parts = {k: v for k, v in m.items() if k not in ("timestamp", "time", "date", "ts", "id", "_id")}
        return hashlib.sha256(json.dumps(parts, sort_keys=True, ensure_ascii=False).encode("utf-8")).hexdigest()
    return hashlib.sha256(json.dumps(m, sort_keys=True, ensure_ascii=False).encode("utf-8")).hexdigest()


kept = []
seen_sig = set()
removed_noise = 0
removed_dup = 0
removed_empty = 0

for m in rm:
    txt = text_of(m)
    if not txt.strip():
        removed_empty += 1
        continue
    if is_pure_noise(txt):
        removed_noise += 1
        continue
    s = sig(m)
    if s in seen_sig:
        removed_dup += 1
        continue
    seen_sig.add(s)
    kept.append(m)

d["realtime_memories"] = kept

with open(PATH, "w", encoding="utf-8") as f:
    json.dump(d, f, ensure_ascii=False, indent=2)

new_count = len(kept)
print(
    f"Consolidated: {orig_count} -> {new_count} realtime memories (removed {removed_noise} noise, {removed_dup} exact dup, {removed_empty} empty)."
)

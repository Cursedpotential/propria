"""Pick the critical flags that bear on one search query, and trim them.

Byline: Claude Code · Opus 5.5 · 2026-09-28

Every search used to attach EVERY active critical flag in the domain as `critical_context`, with
full summaries, rationale, change hashes and source paths: about 10 KB on each call, unrelated to
the question (owner, 2026-09-28: "doc store is still not being properly filtered"). Flags still
matter, because a critical decision can override what a search returns, so they are not dropped.
They are ranked against the query, the few that share its terms are kept as short excerpts, and
the response says how many exist and how to read them all (`docstore_flags`).

Pure function: flags and query in, bounded dict out. No I/O.
"""

from __future__ import annotations

import re

MAX_FLAGS = 3
EXCERPT_CHARS = 280

_STOPWORDS = frozenset(
    "the and for with from that this into over under what when where which while about after "
    "before does done have has had are was were will would should could not but any all one two "
    "its our your their them they then than also just only more most some such very each "
    "docs doc document documents decision decisions note notes".split()
)
_WORD = re.compile(r"[a-z0-9][a-z0-9_\-]{2,}")


def _terms(text: str) -> set[str]:
    return {w for w in _WORD.findall((text or "").lower()) if w not in _STOPWORDS}


def _excerpt(text: str) -> str:
    text = " ".join((text or "").split())
    return text if len(text) <= EXCERPT_CHARS else text[:EXCERPT_CHARS].rstrip() + "…"


def relevant_flags(flag_list: dict, query: str, limit: int = MAX_FLAGS) -> dict:
    """Return the flags relevant to `query`, trimmed, plus counts and how to see the rest.

    A flag is relevant when it shares at least one query term, and at least two once the query
    has four or more terms; ties go to the most recently updated. Status keys from the caller
    ("status", "warning") pass through unchanged.
    """
    flags = list(flag_list.get("flags") or [])
    wanted = _terms(query)
    need = 2 if len(wanted) >= 4 else 1

    scored = []
    for position, flag in enumerate(flags):
        haystack = _terms(" ".join(str(flag.get(k) or "") for k in ("title", "subject", "summary")))
        score = len(wanted & haystack)
        if score >= need:
            scored.append((score, -position, flag))  # input is newest-first
    scored.sort(key=lambda item: (item[0], item[1]), reverse=True)

    shown = [
        {
            "id": flag.get("id"),
            "title": flag.get("title"),
            "authority": flag.get("authority"),
            "updated_at": flag.get("updated_at"),
            "excerpt": _excerpt(flag.get("summary") or flag.get("subject") or ""),
        }
        for _, _, flag in scored[:limit]
    ]

    out = {
        "active_in_domain": len(flags) if not flag_list.get("truncated") else f"{len(flags)}+",
        "relevant_shown": len(shown),
        "flags": shown,
        "read_all": "docstore_flags(domain=...) lists every active critical flag in full",
    }
    for key in ("status", "warning"):
        if key in flag_list:
            out[key] = flag_list[key]
    return out

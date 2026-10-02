# Byline: Claude Code · Sonnet · 2026-10-02
"""Turn bench results into (a) per-family markdown tables and (b) the registry quality ranks.

Rule (stated in the receipt README): per family, a tool's fidelity is the weighted mean of
text recall (0.60), emoji recall (0.25, when the family has emoji) and link/media-reference recall
(0.15, when the family has references); a file the tool failed on (error, timeout, memory cap) counts 0.
A tool whose output carries more than 1.3x the visible text (CSS and script leakage) loses 10 percent.
primary = best fidelity (ties within 0.003 go to the lower median wall time);
fallback = within 0.05 of the primary; experimental = the rest.
"""
from __future__ import annotations

import json
import statistics
import sys
from collections import defaultdict

FAMILY_TO_FORMAT = {
    "fb_messenger_thread": "facebook_messenger_html",
    "fb_other_section": "facebook_export_section_html",
    "google_voice": "google_voice_html",
    "google_my_activity": "google_takeout_activity_html",
    "imessage_export": "imessage_export_html",
    "whatsapp_chat": "whatsapp_chat_html",
    "snapchat_export": "snapchat_export_html",
    "software_docs_and_misc": "generic_html_document",
    "number_named_export": "generic_html_document",
}
REGISTERED = {
    "docling": "docling", "unstructured": "unstructured", "markitdown": "markitdown", "html2text": "html2text",
    "beautifulsoup4": "beautifulsoup4", "lxml": "lxml", "selectolax_lexbor": "selectolax",
}
ALL_GENERIC = list(REGISTERED) + ["webbed_generic_template", "webbed_html_extract_text", "webbed_read_html_blocks", "trafilatura", "readability_lxml"]


def load(paths):
    seen = {}
    for p in paths:
        for line in open(p, encoding="utf-8"):
            r = json.loads(line)
            if "myactivity" in r["file"].lower():
                r["family"] = "google_my_activity"  # the 62 MB Takeout page was sampled under a different name
            seen.setdefault((r["file"], r["tool"], r["mode"]), r)
    return list(seen.values())


def fidelity(rows):
    scores, noise, walls = [], [], []
    for r in rows:
        walls.append(r["wall_s"])
        if r["status"] != "ok":
            scores.append(0.0)
            continue
        parts = [(0.60, r.get("text_recall"))]
        if r.get("emoji_recall") is not None:
            parts.append((0.25, r["emoji_recall"]))
        if r.get("refs_recall") is not None:
            parts.append((0.15, r["refs_recall"]))
        parts = [(w, v) for w, v in parts if v is not None and v == v]
        total = sum(w for w, _ in parts)
        scores.append(sum(w * v for w, v in parts) / total if total else 0.0)
        noise.append(r.get("noise_ratio") or 0)
    score = statistics.mean(scores) if scores else 0.0
    if noise and statistics.mean(noise) > 1.3:
        score *= 0.9
    return score, (statistics.median(walls) if walls else 0)


def main(paths):
    rows = load(paths)
    by = defaultdict(lambda: defaultdict(list))
    for r in rows:
        if r["mode"] == "generic":
            by[r["family"]][r["tool"]].append(r)
    fmt_scores = defaultdict(lambda: defaultdict(list))  # format -> tool -> [(fid, wall, files)]
    md = []
    for family in sorted(by):
        fmt = FAMILY_TO_FORMAT.get(family, family)
        md.append(f"\n#### {family} -> `{fmt}`  ({max(len(v) for v in by[family].values())} files)\n")
        md.append("| tool | files ok/total | text recall | emoji recall | link recall | text volume vs truth | median wall s | peak RSS MB | fidelity |")
        md.append("|---|---|---|---|---|---|---|---|---|")
        table = []
        for tool in ALL_GENERIC:
            rs = by[family].get(tool)
            if not rs:
                continue
            ok = [r for r in rs if r["status"] == "ok"]
            f, wall = fidelity(rs)
            def avg(key):
                vals = [r[key] for r in ok if r.get(key) is not None and r[key] == r[key]]
                return f"{statistics.mean(vals):.3f}" if vals else "n/a"
            table.append((f, wall, tool, f"| {tool} | {len(ok)}/{len(rs)} | {avg('text_recall')} | {avg('emoji_recall')} | {avg('refs_recall')} | {avg('noise_ratio')}x | {wall:.2f} | {max(r['peak_rss_mb'] for r in rs)} | {f:.3f} |"))
            if tool in REGISTERED:
                fmt_scores[fmt][REGISTERED[tool]].append((f, wall, len(rs)))
        for _, _, _, line in sorted(table, key=lambda t: (-t[0], t[1])):
            md.append(line)
    # registry ranks per format (pool the families that map to one format, weighting by file count)
    quality = {}
    notes = {}
    for fmt, tools in sorted(fmt_scores.items()):
        pooled = {}
        for tool, entries in tools.items():
            n = sum(e[2] for e in entries)
            pooled[tool] = (sum(e[0] * e[2] for e in entries) / n, statistics.median([e[1] for e in entries]))
        best = max(v[0] for v in pooled.values())
        if best < 0.5:
            # No registered tool reads this family (the iMessage export keeps its messages inside a script
            # string); every tool is experimental and the family belongs to a dedicated template.
            for tool in pooled:
                quality.setdefault(tool, {})[fmt] = "experimental"
            notes[fmt] = (None, round(best, 3))
            continue
        primary_pool = [t for t, v in pooled.items() if v[0] >= best - 0.003]
        primary = min(primary_pool, key=lambda t: pooled[t][1])
        for tool, (f, wall) in pooled.items():
            rank = "primary" if tool == primary else ("fallback" if f >= pooled[primary][0] - 0.15 else "experimental")
            quality.setdefault(tool, {})[fmt] = rank
        notes[fmt] = (primary, round(pooled[primary][0], 3))
    out = {"markdown": "\n".join(md), "quality": quality, "primary": notes}
    json.dump(out, open("ranks.json", "w"), indent=1)
    print("\n".join(md))
    print("\nPRIMARY PER FORMAT:", json.dumps(notes, indent=1))


if __name__ == "__main__":
    main(sys.argv[1:])

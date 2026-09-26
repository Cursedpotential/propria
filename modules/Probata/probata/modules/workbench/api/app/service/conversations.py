"""Bout review data for any conversation and any label pass. Byline: Claude Code · Opus 5.5 · 2026-09-24.

Label passes store different shapes; `normalize_label` turns each known shape into one: message segments
`[from_i, to_i, tone, driver, note]`, shifts `[at_i, from_tone, to_tone, speed, trigger_i, note]` and a summary.
Known shapes: `stretches`/`shifts`/`summary` (bout-tone-v1) and `windows[].answers.main_tone` (bout-q-v1, windows may
overlap: a message takes the tone of the latest window that covers it). A pass in neither shape comes back with no
segments and `shape: "unknown"`, so the page flags it instead of guessing.
"""
from __future__ import annotations

import re
from collections import Counter
from datetime import date
from itertools import pairwise
from typing import Any

from app.repo import conversations as repo
from app.repo.intake_discovery import DiscoveryError

DAY = re.compile(r"^\d{4}-\d{2}-\d{2}$")


def normalize_label(output: dict[str, Any] | None) -> dict[str, Any]:
    if not output:
        return {"shape": None, "segments": [], "shifts": [], "summary": ""}
    if isinstance(output.get("stretches"), list):
        segments = [[s.get("from_i"), s.get("to_i"), s.get("tone"), s.get("driver") or "", s.get("note") or ""]
                    for s in output["stretches"] if isinstance(s, dict)]
        shifts = [[s.get("at_i"), s.get("from_tone"), s.get("to_tone"), s.get("speed") or "", s.get("trigger_i"),
                   s.get("note") or ""] for s in output.get("shifts") or [] if isinstance(s, dict)]
        return {"shape": "stretches", "segments": segments, "shifts": shifts, "summary": output.get("summary") or ""}
    if isinstance(output.get("windows"), list):
        tone_at: dict[int, tuple[str, str, str]] = {}
        for w in sorted((w for w in output["windows"] if isinstance(w, dict)), key=lambda w: w.get("from_i", 0)):
            answers = w.get("answers") or {}
            main = answers.get("main_tone") or {}
            if not main.get("choice"):
                continue
            driver = (answers.get("conflict_driver") or {}).get("choice") or ""
            note = f"confidence {main['confidence']:.2f}" if isinstance(main.get("confidence"), (int, float)) else ""
            for i in range(int(w.get("from_i", 0)), int(w.get("to_i", -1)) + 1):
                tone_at[i] = (main["choice"], driver, note)
        segments: list[list[Any]] = []
        for i in sorted(tone_at):
            tone, driver, note = tone_at[i]
            if segments and segments[-1][1] == i - 1 and segments[-1][2] == tone:
                segments[-1][1] = i
            else:
                segments.append([i, i, tone, driver, note])
        shifts = [[b[0], a[2], b[2], "", None, ""] for a, b in pairwise(segments) if a[2] != b[2]]
        return {"shape": "windows", "segments": segments, "shifts": shifts, "summary": output.get("summary") or ""}
    return {"shape": "unknown", "segments": [], "shifts": [], "summary": ""}


def _pick(options: list[dict[str, Any]], key: str, wanted: str | None) -> str | None:
    names = [o[key] for o in options]
    if wanted is None:
        return names[0] if names else None
    if wanted not in names:
        raise DiscoveryError(f"Unknown {key} for this conversation", 404)
    return wanted


def bout_review(conv_key: str, rules: str | None = None, label_pass: str | None = None) -> dict[str, Any]:
    conv = repo.conversation(conv_key)
    rules = _pick(conv["bout_sets"], "rules", rules)
    if rules is None:
        raise DiscoveryError("This conversation has no bouts yet", 404)
    passes = [p for p in conv["label_passes"] if p["rules"] == rules]
    label_pass = _pick(passes, "pass", label_pass)
    model = next((p["model"] for p in passes if p["pass"] == label_pass), None)

    bouts, tones, senders, shapes = [], Counter(), Counter(), Counter()
    shifts = abrupt = labelled = messages = 0
    for row in repo.bouts(conv_key, rules, label_pass):
        label = normalize_label(row["label"])
        who = row["senders"] or {}
        senders.update(who)
        messages += row["n_messages"]
        if label["shape"]:
            labelled += 1
            shapes[label["shape"]] += 1
        for seg in label["segments"]:
            tones[seg[2]] += seg[1] - seg[0] + 1
        shifts += len(label["shifts"])
        abrupt += sum(1 for s in label["shifts"] if s[3] == "abrupt")
        day = row["day"].isoformat() if isinstance(row["day"], date) else str(row["day"])
        bouts.append({"id": row["bout_id"], "day": day, "start": row["start"], "end": row["end"],
                      "messages": row["n_messages"], "senders": who, "labelled": bool(label["shape"]),
                      "segments": label["segments"], "shifts": label["shifts"], "summary": label["summary"]})
    return {
        "conversation": {k: conv[k] for k in ("conv_key", "description", "custody_party", "source_format")},
        "bout_sets": conv["bout_sets"], "label_passes": passes,
        "rules": rules, "pass": label_pass, "model": model, "timezone": repo.LOCAL_TZ,
        "facts": {"days": len({b["day"] for b in bouts}), "bouts": len(bouts), "messages": messages,
                  "labelled": labelled, "shifts": shifts, "abrupt": abrupt,
                  "first_day": bouts[0]["day"] if bouts else None, "last_day": bouts[-1]["day"] if bouts else None},
        "tone_messages": dict(tones), "senders": dict(senders.most_common()), "label_shapes": dict(shapes),
        "bouts": bouts,
    }


def day(conv_key: str, rules: str, day_value: str) -> dict[str, Any]:
    if not DAY.match(day_value):
        raise DiscoveryError("Day must be YYYY-MM-DD", 422)
    conv = repo.conversation(conv_key)
    _pick(conv["bout_sets"], "rules", rules)
    rows = repo.day_messages(conv_key, rules, day_value)
    return {"day": day_value, "timezone": repo.LOCAL_TZ,
            "messages": [{"bout_id": r["bout_id"], "i": r["ordinal"], "time": r["time"], "who": r["who"],
                          "body": r["body"] or "", "files": list(r["files"] or [])} for r in rows]}

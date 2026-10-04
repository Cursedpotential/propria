"""Match a screenshot's OCR text to message records that already exist. Pure functions, no I/O, no
storage.

> _Byline: Claude Code · Sonnet 5.5 · 2026-10-03_

Owner ruling 2026-10-03 (Intake MASTER-TODO CBX-P8-003): a screenshot of a conversation helps
VALIDATE the text copy of
that conversation. It corroborates; it does not stand on its own and it never becomes a message. OCR
text is a
derivative of the picture and is never equated with a native export's text. So this module only ASKS
"which existing
message records does this screenshot show?" and returns, for each OCR passage, either the records it
matches (with a
score and the exact span of the OCR text) or nothing, which is a possible gap or discrepancy to
flag. It creates no
message, writes nothing and knows nothing about where records or links are stored: the caller
supplies candidate
records (already narrowed by the visible names or numbers, the visible date and the screenshot's
capture time, see
``narrow_candidates``) and decides what to do with the result.

    method id: ``difflib-lines-v1``. A passage is one to three consecutive OCR lines (a wrapped
    bubble is several lines).
    Score: the ``difflib`` similarity between the normalized passage and the normalized message
    body, or, when one
    contains the other, the length of the shorter over the longer (so a window that swallows a
    neighbouring bubble scores
    lower than the window that is exactly the bubble, and a bubble showing only part of a long
    message scores by its share).
"""

from __future__ import annotations

import difflib
import re
import unicodedata
from dataclasses import dataclass, field
from datetime import datetime, timedelta

METHOD = "difflib-lines-v1"
MIN_CHARS = 12  # normalized characters a passage needs before it can match or be flagged
MIN_SCORE = 0.82
MAX_WINDOW_LINES = 3

# Phone chrome that is not message text: clock times, delivery marks, day separators.
_CHROME = re.compile(
    r"^(?:\d{1,2}:\d{2}\s*(?:am|pm)?|delivered|read|sent|seen|today|yesterday|"
    r"(?:mon|tue|wed|thu|fri|sat|sun)[a-z]*\.?(?:,?\s+\d{1,2}:\d{2}\s*(?:am|pm)?)?|"
    r"(?:jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)[a-z]*\.?\s+\d{1,2}(?:,\s*\d{4})?)$"
)


@dataclass(frozen=True)
class MessageRef:
    """One existing message record, as much as matching needs (``record_id`` is the normalized
    record's id)."""

    record_id: str
    body: str
    occurred_at: datetime | None = None
    sender: str = ""
    recipients: tuple[str, ...] = ()


@dataclass(frozen=True)
class Passage:
    start: int  # character offsets into the OCR text, end exclusive
    end: int
    text: str


@dataclass(frozen=True)
class Match:
    record_id: str
    score: float
    span: tuple[int, int]  # the OCR span matched
    passage: str


@dataclass
class MatchResult:
    method: str = METHOD
    matches: list[Match] = field(default_factory=list)
    # screenshot text with no matching record: flag, never create
    unmatched: list[Passage] = field(default_factory=list)


def normalize(text: str) -> str:
    """NFC, casefold, punctuation to space, collapsed whitespace: what two renderings of one message
    share."""
    folded = unicodedata.normalize("NFC", text).casefold()
    return re.sub(r"\s+", " ", re.sub(r"[^\w$:%@#+\-]+", " ", folded)).strip()


def ocr_lines(ocr_text: str) -> list[Passage]:
    """The OCR text as lines with offsets, minus empty lines and phone chrome (clock times,
    'Delivered', day headers)."""
    lines = []
    offset = 0
    for raw in ocr_text.splitlines(keepends=True):
        stripped = raw.strip()
        start = offset + (len(raw) - len(raw.lstrip()))
        offset += len(raw)
        if stripped and not _CHROME.match(normalize(stripped)):
            lines.append(Passage(start, start + len(stripped), stripped))
    return lines


def passages(ocr_text: str) -> list[Passage]:
    """Windows of 1 to 3 consecutive OCR lines. The longest window that matches a record wins in
    ``match_screenshot``."""
    lines = ocr_lines(ocr_text)
    out: list[Passage] = []
    for i in range(len(lines)):
        for size in range(1, MAX_WINDOW_LINES + 1):
            if i + size > len(lines):
                break
            chunk = lines[i : i + size]
            out.append(Passage(chunk[0].start, chunk[-1].end, " ".join(p.text for p in chunk)))
    return out


def similarity(passage: str, body: str) -> float:
    a, b = normalize(passage), normalize(body)
    if not a or not b:
        return 0.0
    if a in b or b in a:
        return min(len(a), len(b)) / max(len(a), len(b))
    if difflib.SequenceMatcher(None, a, b).quick_ratio() < 0.6:
        return 0.0
    return difflib.SequenceMatcher(None, a, b, autojunk=False).ratio()


def narrow_candidates(
    candidates: list[MessageRef],
    *,
    capture_time: datetime | None = None,
    window: timedelta = timedelta(days=3),
    visible_names: tuple[str, ...] = (),
) -> list[MessageRef]:
    """Keep records that could be in the picture: sent or received within ``window`` of the
    screenshot's capture time
    (a record with no time is kept: no time is not evidence of a different time), and, when names or
    numbers are
    visible, involving at least one of them (matched on digits for numbers, case-folded for
    names)."""
    names = [normalize(n) for n in visible_names if normalize(n)]
    digits = [re.sub(r"\D", "", n) for n in visible_names]
    digits = [d for d in digits if len(d) >= 7]
    kept = []
    for message in candidates:
        if (
            capture_time is not None
            and message.occurred_at is not None
            and abs(message.occurred_at - capture_time) > window
        ):
            continue
        if names or digits:
            parties = [normalize(message.sender), *[normalize(r) for r in message.recipients]]
            party_digits = [re.sub(r"\D", "", p) for p in (message.sender, *message.recipients)]
            if not any(n in p or p in n for n in names for p in parties if p) and not any(
                d[-7:] == pd[-7:] for d in digits for pd in party_digits if len(pd) >= 7
            ):
                continue
        kept.append(message)
    return kept


def match_screenshot(
    ocr_text: str,
    candidates: list[MessageRef],
    *,
    min_score: float = MIN_SCORE,
    min_chars: int = MIN_CHARS,
) -> MatchResult:
    """Match OCR passages to candidate message records; return the matches and the passages nothing
    matched.

    Inputs: the screenshot's OCR text and the candidate records (narrowed by the caller). Output:
    ``MatchResult``.
    Greedy: the best-scoring (passage, record) pairs are taken first, longer passages before shorter
    on a tie, and a
    record or a stretch of OCR text is used once. Passages shorter than ``min_chars`` normalized
    characters are neither
    matched nor flagged (a bare 'ok' proves nothing either way). An empty candidate list flags every
    substantial passage:
    the caller decides whether that means 'no records in range' or 'a possible gap'."""
    result = MatchResult()
    options: list[tuple[float, int, Passage, MessageRef]] = []
    pool = [p for p in passages(ocr_text) if len(normalize(p.text)) >= min_chars]
    for passage in pool:
        for message in candidates:
            score = similarity(passage.text, message.body)
            if score >= min_score:
                options.append((score, len(normalize(passage.text)), passage, message))
    options.sort(key=lambda o: (-o[0], -o[1]))
    used_records: set[str] = set()
    used_spans: list[tuple[int, int]] = []
    for score, _, passage, message in options:
        if message.record_id in used_records or any(
            passage.start < e and s < passage.end for s, e in used_spans
        ):
            continue
        used_records.add(message.record_id)
        used_spans.append((passage.start, passage.end))
        result.matches.append(
            Match(message.record_id, round(score, 3), (passage.start, passage.end), passage.text)
        )
    result.matches.sort(key=lambda m: m.span)
    for line in ocr_lines(ocr_text):
        if len(normalize(line.text)) >= min_chars and not any(
            line.start < e and s < line.end for s, e in used_spans
        ):
            result.unmatched.append(line)
    return result

"""Chonkie chunking of a message thread into message spans, with at least ``overlap`` messages shared.

Byline: Claude Code · Sonnet 5.5 · 2026-10-02

The default is Chonkie's Neural chunker on ``mirth/chonky_distilbert_base_uncased_1`` with
``min_characters_per_chunk=1``: exactly the model and settings of scripts/jev_eval/chunk_all.py (the 25-stretch
head-to-head, ``neural_distilbert``). The platform is a toolbox, so the other Chonkie chunkers that fit a text-message
stream stay selectable by name (CHUNKERS); register more with ``register``. Slumber (an LLM), Semantic and Late (they
need an embedder of their own) are not in the default registry.

Method: one line per message (render.py). An engine turns the lines into the message index each chunk starts at
(``firsts``); the spans between those starts partition the thread; a span longer than MAX_CHUNK_CHARS is cut into even
parts so the whole chunk fits the embedder's input; ``widen`` then extends every span forward by ``overlap`` messages,
so chunk i and chunk i+1 share at least ``overlap`` messages.

Why the Neural chunker is windowed (measured 2026-10-02, transformers 5.17 and 5.18, the same stack as the head-to-head):
it classifies only the first ~512 tokens of its input. The head-to-head fed it stretches of at most 89 messages, where
that did not show. A 1000-message thread fed whole came back as 3 chunks with every split inside the first window.
WindowedNeural therefore feeds it overlapping windows of at most NEURAL_WINDOW_TOKENS tokens and keeps each window's
splits from the middle of each overlap, so every part of the thread is classified once.
"""

from __future__ import annotations

import functools
import importlib.metadata
import math
import threading
from bisect import bisect_right
from collections.abc import Callable
from typing import Any

from server.context_chunks.config import DEFAULT_CHUNKER, DEFAULT_OVERLAP, MAX_CHUNK_CHARS, NEURAL_WINDOW_TOKENS

NEURAL_DISTILBERT_MODEL = "mirth/chonky_distilbert_base_uncased_1"
NEURAL_MODERNBERT_MODEL = "mirth/chonky_modernbert_large_1"
MAX_AI_NEURAL_WINDOWS = 1024
TEXT_FORMAT = "line-v1"  # the chunk-text construction (render.render_line); part of the chunker version


def _chonkie():
    import chonkie

    return chonkie


def line_starts(lines: list[str]) -> list[int]:
    """Character offset of each line in ``"\\n".join(lines)``."""
    starts, position = [], 0
    for line in lines:
        starts.append(position)
        position += len(line) + 1
    return starts


def firsts_from_offsets(chunk_offsets: list[int], starts: list[int]) -> list[int]:
    """Message index each chunk starts in; chunks that start in the same message merge. Always begins at 0."""
    firsts: list[int] = []
    for offset in chunk_offsets:
        index = max(0, bisect_right(starts, offset) - 1)
        if not firsts or index > firsts[-1]:
            firsts.append(index)
    if not firsts or firsts[0] != 0:
        firsts.insert(0, 0)
    return firsts


class TextEngine:
    """Any Chonkie chunker run over the whole joined text (the model-free chunkers handle any length)."""

    def __init__(self, chunker: Any):
        self.chunker = chunker

    def firsts(self, lines: list[str], beat: Callable[[str], None] | None = None) -> list[int]:
        chunks = self.chunker.chunk("\n".join(lines))
        return firsts_from_offsets([c.start_index for c in chunks], line_starts(lines))


class WindowedNeural:
    """The Neural chunker fed overlapping windows of at most ``budget`` model tokens, central-half splits kept."""

    def __init__(self, neural: Any, budget: int = NEURAL_WINDOW_TOKENS):
        self.neural, self.budget = neural, budget
        # One model instance serves every Activity thread of the worker; inference runs one thread at a time (it is
        # CPU-bound and already uses every core).
        self._lock = threading.Lock()

    def windows(self, counts: list[int]) -> list[tuple[int, int]]:
        """Half-overlapping [start, end) message windows, each at most ``budget`` tokens (one message at least)."""
        n, out, s = len(counts), [], 0
        while True:
            e, used = s, 0
            while e < n and (e == s or used + counts[e] <= self.budget):
                used += counts[e]
                e += 1
            out.append((s, e))
            if e >= n:
                return out
            half, k = 0, s
            while k < e and half < used // 2:
                half += counts[k]
                k += 1
            s = max(k, s + 1)

    def firsts(self, lines: list[str], beat: Callable[[str], None] | None = None) -> list[int]:
        counts = [c + 1 for c in self.neural.tokenizer.count_tokens_batch(lines)]  # +1: the joining newline
        windows = self.windows(counts)
        # Window k decides the splits in [cut_{k-1}, cut_k), cut_k = the middle of its overlap with window k+1, so the
        # decided regions tile the thread exactly and every split is judged with context on both sides.
        cuts = [(windows[k + 1][0] + windows[k][1]) // 2 for k in range(len(windows) - 1)] + [len(lines)]
        firsts = {0}
        for k, (s, e) in enumerate(windows):
            if beat and k % 20 == 0:
                beat(f"window {k}/{len(windows)}")
            segment = lines[s:e]
            with self._lock:
                chunks = self.neural.chunk("\n".join(segment))
            local = firsts_from_offsets([c.start_index for c in chunks], line_starts(segment))
            low = 1 if k == 0 else cuts[k - 1]
            firsts.update(s + f for f in local[1:] if low <= s + f < cuts[k])
        return sorted(firsts)


CHUNKERS: dict[str, Callable[[], Any]] = {
    # chunk_all.py's neural_distilbert / neural_modernbert settings, windowed (see the module docstring).
    "neural_distilbert": lambda: WindowedNeural(
        _chonkie().NeuralChunker(model=NEURAL_DISTILBERT_MODEL, min_characters_per_chunk=1)
    ),
    "neural_modernbert": lambda: WindowedNeural(
        _chonkie().NeuralChunker(model=NEURAL_MODERNBERT_MODEL, min_characters_per_chunk=1)
    ),
    # The model-free chunkers of the same head-to-head, character-sized.
    "token_1000": lambda: TextEngine(_chonkie().TokenChunker(tokenizer="character", chunk_size=1000, chunk_overlap=0)),
    "fast_1000": lambda: TextEngine(_chonkie().FastChunker(chunk_size=1000, delimiters="\n")),
    "sentence_1000": lambda: TextEngine(
        _chonkie().SentenceChunker(tokenizer="character", chunk_size=1000, delim=["\n"], min_characters_per_sentence=1)
    ),
    "recursive_1000": lambda: TextEngine(
        _chonkie().RecursiveChunker(tokenizer="character", chunk_size=1000, min_characters_per_chunk=1)
    ),
}


def register(name: str, factory: Callable[[], Any]) -> None:
    """Add a chunker to the toolbox: a callable returning an engine with ``firsts(lines, beat=None) -> list[int]`` (see
    TextEngine; ``beat`` is the Activity heartbeat), or any object with ``chunk(text)`` returning chunks that carry
    ``start_index``."""
    CHUNKERS[name] = factory


def chunker_version(name: str, overlap: int = DEFAULT_OVERLAP) -> str:
    """The exact construction, part of every chunk's identity: a new model, setting or overlap is a new version."""
    if name not in CHUNKERS:
        raise ValueError(f"unknown chunker {name!r}; available: {sorted(CHUNKERS)}")
    try:
        library = f"chonkie-{importlib.metadata.version('chonkie')}"
    except importlib.metadata.PackageNotFoundError:  # reported when the chunker is built
        library = "chonkie-unknown"
    model = {"neural_distilbert": NEURAL_DISTILBERT_MODEL, "neural_modernbert": NEURAL_MODERNBERT_MODEL}.get(name)
    window = f"|window={NEURAL_WINDOW_TOKENS}" if model else ""
    return f"{name}|{model or '-'}|{library}{window}|maxchars={MAX_CHUNK_CHARS}|overlap={overlap}|text={TEXT_FORMAT}"


@functools.lru_cache(maxsize=8)
def _built(name: str):
    return CHUNKERS[name]()


def spans_from_firsts(firsts: list[int], n: int) -> list[tuple[int, int]]:
    return [(first, (firsts[i + 1] - 1) if i + 1 < len(firsts) else n - 1) for i, first in enumerate(firsts)]


def split_oversize(spans: list[tuple[int, int]], lines: list[str], max_chars: int) -> list[tuple[int, int]]:
    """Cut a span whose text exceeds ``max_chars`` into even parts by characters, at message boundaries. A single
    message longer than the limit stays one part (a message is never cut)."""
    if max_chars <= 0:
        return spans
    out: list[tuple[int, int]] = []
    for first, last in spans:
        size = sum(len(lines[i]) + 1 for i in range(first, last + 1))
        if size <= max_chars or first == last:
            out.append((first, last))
            continue
        parts = math.ceil(size / max_chars)
        target, start, used = size / parts, first, 0.0
        for i in range(first, last + 1):
            used += len(lines[i]) + 1
            if i < last and used >= target:
                out.append((start, i))
                start, used = i + 1, 0.0
        out.append((start, last))
    return out


def widen(spans: list[tuple[int, int]], n: int, overlap: int = DEFAULT_OVERLAP) -> list[tuple[int, int]]:
    """Extend every span forward by ``overlap`` messages (clamped to the thread's end).

    Adjacent spans [f_i, l_i] and [l_i + 1, l_{i+1}] become [f_i, l_i + o] and [l_i + 1, l_{i+1} + o]; their
    intersection is [l_i + 1, min(l_i + o, l_{i+1} + o)], at least ``o`` messages whenever the thread continues.
    A hit's first message stays the chunk's own first message, which is the message the Workbench opens.
    """
    if overlap < 0:
        raise ValueError("overlap must not be negative")
    return [(first, min(last + overlap, n - 1)) for first, last in spans]


def chunk_spans(
    lines: list[str],
    name: str = DEFAULT_CHUNKER,
    overlap: int = DEFAULT_OVERLAP,
    *,
    chunker: Any = None,
    max_chars: int = MAX_CHUNK_CHARS,
    beat: Callable[[str], None] | None = None,
) -> list[tuple[int, int]]:
    """Cut one thread's rendered lines into overlapping message spans. ``chunker`` overrides the registry (tests)."""
    n = len(lines)
    if n == 0:
        return []
    if n == 1:
        return [(0, 0)]
    engine = chunker if chunker is not None else _built(name)
    if not hasattr(engine, "firsts"):
        engine = TextEngine(engine)
    spans = split_oversize(spans_from_firsts(engine.firsts(lines, beat), n), lines, max_chars)
    return widen(spans, n, overlap)


def neural_text_spans(
    text: str,
    *,
    max_chars: int = MAX_CHUNK_CHARS,
    neural: Any = None,
) -> list[tuple[int, int]]:
    """Partition one native text, including a long single turn, at NeuralChunker topic boundaries.

    Inputs: source text, maximum returned characters, optional injected Chonkie
    NeuralChunker. Outputs: adjacent code-point spans covering the exact text.
    Effects: model inference only when no chunker is injected; choose for AI
    export prose rather than message-boundary thread chunking. The bounded
    windows avoid NeuralChunker's first-token-window truncation.
    """
    if not text:
        return []
    if max_chars <= 0:
        raise ValueError("max_chars must be positive")
    engine = neural if neural is not None else _built("neural_distilbert").neural
    # Classify short windows with context on both sides; a long message must not
    # bypass inference merely because there is only one message in its thread.
    # Chonkie's model sees a bounded token prefix, so token-count each proposed
    # window before asking it to split the text.
    windows: list[tuple[int, int]] = []
    start = 0
    while start < len(text):
        end = min(len(text), start + min(max_chars, 1500))
        tokenizer = getattr(engine, "tokenizer", None)
        if tokenizer is not None:
            while end > start + 1 and tokenizer.count_tokens_batch([text[start:end]])[0] > NEURAL_WINDOW_TOKENS:
                end = start + max(1, (end - start) // 2)
        windows.append((start, end))
        if len(windows) > MAX_AI_NEURAL_WINDOWS:
            raise ValueError("AI conversation exceeds bounded NeuralChunker windows")
        if end == len(text):
            break
        start += max(1, (end - start) * 3 // 4)
    cuts_between = [(windows[i][1] + windows[i + 1][0]) // 2 for i in range(len(windows) - 1)]
    cuts = {0, len(text)}
    for index, (start, end) in enumerate(windows):
        chunks = engine.chunk(text[start:end])
        keep_low = 0 if index == 0 else cuts_between[index - 1]
        keep_high = len(text) if index == len(windows) - 1 else cuts_between[index]
        for chunk in chunks:
            cut = start + int(chunk.start_index)
            if keep_low < cut < keep_high:
                cuts.add(cut)
    ordered = sorted(cuts)
    # Neural classification can place adjacent boundaries a few characters
    # apart. Those fragments are poor search units and convey no useful topic.
    minimum = min(80, max(1, max_chars // 4))
    compact = [ordered[0]]
    for cut in ordered[1:-1]:
        if cut - compact[-1] >= minimum and len(text) - cut >= minimum:
            compact.append(cut)
    compact.append(ordered[-1])
    spans = []
    for left, right in zip(compact, compact[1:]):
        while right - left > max_chars:
            spans.append((left, left + max_chars))
            left += max_chars
        if left < right:
            spans.append((left, right))
    return spans

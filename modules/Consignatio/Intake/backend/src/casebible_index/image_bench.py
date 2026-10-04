"""Synthetic recall bench for the image index's selectable embedders and OCR engines.

> _Byline: Claude Code · Sonnet 5.5 · 2026-10-03_

Owner direction 2026-10-03: every option stays selectable and the defaults per image type are chosen
by a small
bench, on SYNTHETIC images only (no case image is ever sent anywhere). This module is that bench:

* ``make_screenshots`` / ``make_photos``   deterministic synthetic sets with one natural-language
question per image
* ``recall_table``                         recall@1, recall@5 and MRR of text -> image, for single
vectors (cosine)
                                           and multi-vector bags (MaxSim, as Weaviate scores a
                                           ``multivector`` index)
* ``Backend`` implementations              ``nim``, ``google``, ``jina_maxsim`` (the production
                                           ``image_embedders.ImageEmbedders`` code paths) and
                                           ``clip`` (sentence-
                                           transformers CLIP, the model Weaviate's
                                           ``multi2vec-clip`` runs)
* ``ocr_table``                            Tesseract character and word accuracy on the synthetic
screenshots

Nothing here writes to an index. A backend that needs a key or a model is run only when named on the
command line;
``python -m casebible_index.image_bench --help`` lists them. Photos are procedural scenes (sky,
ground, object),
not camera photographs, so photo recall is indicative, not a measurement of real photo recall.
"""

from __future__ import annotations

import asyncio
import difflib
import io
import json
import re
import shutil
import subprocess
import sys
import tempfile
import time
from collections.abc import Sequence
from dataclasses import dataclass, field
from pathlib import Path
from typing import Protocol

import numpy as np
from PIL import Image, ImageDraw, ImageFilter, ImageFont

# -------------------------------------------------- synthetic data

# (screenshot text, question that paraphrases it without sharing its distinctive words)
SCREENSHOT_SCRIPTS: list[tuple[list[tuple[str, str]], str]] = [
    (
        [
            ("Dana", "Can you pick up Mia from school on Friday at 4:30?"),
            ("Sam", "Yes, I will be at the front gate."),
        ],
        "who is collecting the child at the end of the school week",
    ),
    (
        [
            ("Dana", "The dentist moved her appointment to Tuesday at 9am."),
            ("Sam", "Ok, I will take her."),
        ],
        "which day was the tooth doctor visit rescheduled to",
    ),
    (
        [
            ("Sam", "Soccer fees are $85 and due by the 14th."),
            ("Dana", "I will send the money tonight."),
        ],
        "how much is the sports registration payment",
    ),
    (
        [
            ("Dana", "I will be late, the car will not start."),
            ("Sam", "Do you want me to call the tow company?"),
        ],
        "someone has a broken vehicle and may be delayed",
    ),
    (
        [
            ("Sam", "The hearing is set for the 22nd at 1:15 in room 4."),
            ("Dana", "Thanks, I will bring the folder."),
        ],
        "when and where is the court date",
    ),
    (
        [
            ("Dana", "She had a fever of 101 last night."),
            ("Sam", "Keep her home and I will call the nurse."),
        ],
        "the kid was sick with a high temperature",
    ),
    (
        [
            ("Sam", "Your new apartment lease starts on the first of March."),
            ("Dana", "Great, I get the keys Monday."),
        ],
        "when does the housing rental agreement begin",
    ),
    (
        [
            ("Dana", "Please sign the permission slip for the field trip."),
            ("Sam", "Signed, it is in her backpack."),
        ],
        "a form for the class outing needs a signature",
    ),
    (
        [
            ("Sam", "Grandma's flight lands at 6:40 on Saturday."),
            ("Dana", "I will meet her at arrivals."),
        ],
        "what time does the relative arrive by airplane",
    ),
    (
        [
            ("Dana", "The bank rejected the transfer again."),
            ("Sam", "Call them and ask for the reference number."),
        ],
        "a money movement failed at the financial institution",
    ),
    (
        [
            ("Sam", "Parent teacher conference is Thursday after lunch."),
            ("Dana", "I put it on the calendar."),
        ],
        "meeting with the educators is scheduled midweek",
    ),
    (
        [
            ("Dana", "Mia lost her glasses at the park."),
            ("Sam", "We can order a spare pair tomorrow."),
        ],
        "the daughter misplaced her eyewear outdoors",
    ),
    (
        [
            ("Sam", "I drove 42 miles to the exchange point and nobody came."),
            ("Dana", "I was stuck in traffic."),
        ],
        "a custody handoff location was missed after a long drive",
    ),
    (
        [
            ("Dana", "Birthday party is at the trampoline place, noon on the 9th."),
            ("Sam", "She will love that."),
        ],
        "a celebration at a jumping gym for the girl",
    ),
    (
        [
            ("Sam", "The insurance card expired, I mailed the new one."),
            ("Dana", "Got it, thank you."),
        ],
        "health coverage document renewal sent in the post",
    ),
    (
        [
            ("Dana", "Do not discuss the case in front of her."),
            ("Sam", "Agreed, I will keep it neutral."),
        ],
        "an agreement not to talk about the lawsuit near the child",
    ),
    (
        [
            ("Sam", "Pharmacy says the prescription is ready."),
            ("Dana", "I can grab it after work."),
        ],
        "medication is waiting to be collected",
    ),
    (
        [
            ("Dana", "Swim lessons start next Wednesday at five."),
            ("Sam", "I bought her a new swimsuit."),
        ],
        "water sport classes begin soon and clothing was purchased",
    ),
    (
        [("Sam", "Rent receipt attached, paid on the 3rd."), ("Dana", "Saved it to the folder.")],
        "proof of housing payment was sent along",
    ),
    (
        [
            ("Dana", "The babysitter cancelled for tonight."),
            ("Sam", "I can stay with her until nine."),
        ],
        "childcare fell through for the evening",
    ),
]

SKIES = {  # name -> (top rgb, bottom rgb)
    "bright blue daytime": ((70, 140, 235), (190, 225, 250)),
    "orange sunset": ((235, 110, 50), (250, 205, 120)),
    "dark night": ((10, 15, 45), (40, 50, 100)),
    "grey overcast": ((120, 125, 130), (185, 190, 195)),
}
GROUNDS = {
    "green hills": (50, 140, 60),
    "blue sea": (30, 90, 170),
    "white snow": (240, 245, 250),
    "yellow sand": (225, 195, 120),
}
OBJECTS = ["red house", "yellow sun", "white moon", "small boat", "tall tree"]


@dataclass(frozen=True)
class Item:
    """One bench image, its ground-truth question and (screenshots) the literal text drawn in it."""

    name: str
    image: bytes
    question: str
    literal: str = ""


def _font(size: int) -> ImageFont.FreeTypeFont | ImageFont.ImageFont:
    for candidate in (
        "arial.ttf",
        "DejaVuSans.ttf",
        "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
        "C:/Windows/Fonts/arial.ttf",
    ):
        try:
            return ImageFont.truetype(candidate, size)
        except OSError:
            continue
    return ImageFont.load_default(size)


def _wrap(draw: ImageDraw.ImageDraw, text: str, font, width: int) -> list[str]:
    lines: list[str] = []
    current = ""
    for word in text.split():
        trial = f"{current} {word}".strip()
        if draw.textlength(trial, font=font) <= width:
            current = trial
        else:
            lines.append(current)
            current = word
    return [*lines, current] if current else lines


def render_chat(script: list[tuple[str, str]], index: int) -> tuple[bytes, str]:
    """A phone-style chat screenshot (1080 x 1500) and the literal message text drawn in it."""
    image = Image.new("RGB", (1080, 1500), (245, 245, 247))
    draw = ImageDraw.Draw(image)
    header_font, body_font, small = _font(40), _font(44), _font(30)
    draw.rectangle([0, 0, 1080, 130], fill=(230, 230, 235))
    draw.text(
        (40, 40), script[0][0] if index % 2 else script[1][0], font=header_font, fill=(20, 20, 20)
    )
    y = 190
    drawn: list[str] = []
    for sender, text in script:
        left = sender == script[0][0]
        lines = _wrap(draw, text, body_font, 700)
        height = 60 * len(lines) + 40
        x0 = 40 if left else 1080 - 40 - 760
        draw.rounded_rectangle(
            [x0, y, x0 + 760, y + height], 36, fill=(255, 255, 255) if left else (90, 160, 255)
        )
        for n, line in enumerate(lines):
            draw.text(
                (x0 + 30, y + 20 + 60 * n),
                line,
                font=body_font,
                fill=(10, 10, 10) if left else (255, 255, 255),
            )
        draw.text(
            (x0 + 30, y + height + 8),
            f"{9 + index % 3}:{10 + index * 2 % 50:02d} AM",
            font=small,
            fill=(130, 130, 130),
        )
        drawn.append(text)
        y += height + 90
    out = io.BytesIO()
    image.save(out, format="PNG")
    return out.getvalue(), " ".join(drawn)


def make_screenshots(count: int = 20) -> list[Item]:
    """Deterministic chat screenshots with a paraphrased question each (distinct vocabulary on
    purpose)."""
    items = []
    for i, (script, question) in enumerate(SCREENSHOT_SCRIPTS[:count]):
        png, literal = render_chat(script, i)
        items.append(Item(f"shot{i:02d}", png, question, literal))
    return items


def _gradient(
    size: tuple[int, int], top: tuple[int, int, int], bottom: tuple[int, int, int]
) -> Image.Image:
    width, height = size
    rows = np.linspace(0, 1, height)[:, None, None]
    pixels = np.array(top)[None, None, :] * (1 - rows) + np.array(bottom)[None, None, :] * rows
    return Image.fromarray(np.broadcast_to(pixels, (height, width, 3)).astype("uint8"))


def render_scene(sky: str, ground: str, obj: str, seed: int) -> bytes:
    """A procedural 'photo': gradient sky, textured ground, one object, blur and sensor-style
    noise."""
    rng = np.random.default_rng(seed)
    width, height = 640, 480
    image = _gradient((width, height), *SKIES[sky])
    draw = ImageDraw.Draw(image)
    horizon = int(height * 0.62)
    draw.rectangle([0, horizon, width, height], fill=GROUNDS[ground])
    if ground == "green hills":
        draw.ellipse([-120, horizon - 70, 360, horizon + 140], fill=(40, 120, 50))
        draw.ellipse([260, horizon - 50, 780, horizon + 150], fill=(60, 150, 70))
    cx = 120 + int(rng.integers(0, 320))
    if obj == "yellow sun":
        draw.ellipse([cx, 50, cx + 110, 160], fill=(255, 220, 40))
    elif obj == "white moon":
        draw.ellipse([cx, 50, cx + 100, 150], fill=(245, 245, 235))
    elif obj == "red house":
        draw.rectangle([cx, horizon - 90, cx + 140, horizon + 30], fill=(190, 40, 35))
        draw.polygon(
            [(cx - 15, horizon - 90), (cx + 70, horizon - 160), (cx + 155, horizon - 90)],
            fill=(90, 40, 30),
        )
    elif obj == "small boat":
        draw.polygon(
            [
                (cx, horizon + 40),
                (cx + 160, horizon + 40),
                (cx + 125, horizon + 85),
                (cx + 35, horizon + 85),
            ],
            fill=(240, 240, 240),
        )
        draw.polygon(
            [(cx + 80, horizon - 80), (cx + 80, horizon + 35), (cx + 140, horizon + 35)],
            fill=(250, 250, 250),
        )
    elif obj == "tall tree":
        draw.rectangle([cx + 40, horizon - 60, cx + 70, horizon + 40], fill=(100, 65, 35))
        draw.ellipse([cx - 20, horizon - 190, cx + 130, horizon - 40], fill=(25, 105, 40))
    image = image.filter(ImageFilter.GaussianBlur(1.4))
    noise = rng.normal(0, 7, (height, width, 1))
    arr = np.clip(np.asarray(image).astype("float32") + noise, 0, 255).astype("uint8")
    out = io.BytesIO()
    Image.fromarray(arr).save(out, format="JPEG", quality=88)
    return out.getvalue()


def make_photos(count: int = 20) -> list[Item]:
    """Distinct (sky, ground, object) scenes with a question that names all three."""
    combos = [(s, g, o) for s in SKIES for g in GROUNDS for o in OBJECTS]
    rng = np.random.default_rng(7)
    chosen = [combos[i] for i in rng.choice(len(combos), size=count, replace=False)]
    return [
        Item(f"photo{i:02d}", render_scene(s, g, o, i), f"a {o} with {g} under a {s} sky")
        for i, (s, g, o) in enumerate(chosen)
    ]


# -------------------------------------------------- scoring


def _unit(vectors: np.ndarray) -> np.ndarray:
    return vectors / np.maximum(np.linalg.norm(vectors, axis=-1, keepdims=True), 1e-12)


def cosine_matrix(
    queries: Sequence[Sequence[float]], images: Sequence[Sequence[float]]
) -> np.ndarray:
    """Rows = queries, columns = images, cosine similarity (higher is better)."""
    return (
        _unit(np.asarray(queries, dtype="float32")) @ _unit(np.asarray(images, dtype="float32")).T
    )


def maxsim_matrix(
    query_bags: Sequence[Sequence[Sequence[float]]], image_bags: Sequence[Sequence[Sequence[float]]]
) -> np.ndarray:
    """MaxSim: for each query vector the best image vector, summed over the query vectors."""
    scores = np.zeros((len(query_bags), len(image_bags)), dtype="float32")
    docs = [_unit(np.asarray(bag, dtype="float32")) for bag in image_bags]
    for i, bag in enumerate(query_bags):
        q = _unit(np.asarray(bag, dtype="float32"))
        for j, d in enumerate(docs):
            scores[i, j] = float((q @ d.T).max(axis=1).sum())
    return scores


def recall_table(scores: np.ndarray) -> dict[str, float]:
    """Query i's right answer is image i. Returns recall@1, recall@5 and mean reciprocal rank."""
    ranks = []
    for i in range(scores.shape[0]):
        order = np.argsort(-scores[i])
        ranks.append(int(np.where(order == i)[0][0]) + 1)
    n = len(ranks)
    return {
        "n": n,
        "recall@1": round(sum(r <= 1 for r in ranks) / n, 3),
        "recall@5": round(sum(r <= 5 for r in ranks) / n, 3),
        "mrr": round(sum(1 / r for r in ranks) / n, 3),
    }


# -------------------------------------------------- backends


class Backend(Protocol):
    """One embedder under test: images in, a score matrix against questions out, with timing."""

    name: str
    kind: str  # "single" or "maxsim"

    async def embed_images(self, images: Sequence[bytes], extension: str) -> list: ...

    async def embed_queries(self, questions: Sequence[str]) -> list: ...


@dataclass
class Timing:
    image_seconds: list[float] = field(default_factory=list)
    query_seconds: list[float] = field(default_factory=list)


@dataclass
class ProductionBackend:
    """The production ``ImageEmbedders`` (nim, google, jina) so the bench measures the code the
    index runs."""

    name: str
    kind: str
    embedders: object
    timing: Timing = field(default_factory=Timing)

    async def embed_images(self, images: Sequence[bytes], extension: str) -> list:
        out = []
        for data in images:
            started = time.perf_counter()
            out.append(
                await (
                    self.embedders.embed_multi(data)
                    if self.kind == "maxsim"
                    else self.embedders.embed_single(data, extension)
                )
            )
            self.timing.image_seconds.append(time.perf_counter() - started)
        return out

    async def embed_queries(self, questions: Sequence[str]) -> list:
        out = []
        for text in questions:
            started = time.perf_counter()
            single, multi = await self.embedders.embed_query(text)
            out.append(multi if self.kind == "maxsim" else single)
            self.timing.query_seconds.append(time.perf_counter() - started)
        return out


@dataclass
class ClipBackend:
    """sentence-transformers ``clip-ViT-B-32``: the model family Weaviate ``multi2vec-clip`` serves
    (CPU is fine).

    Needs ``sentence-transformers`` and the model weights; neither is a dependency of the index, so
    this backend
    is run on the box that will host the CLIP container (or a scratch environment), never in the
    worker image.
    """

    name: str = "clip_vit_b32"
    kind: str = "single"
    model_name: str = "clip-ViT-B-32"
    timing: Timing = field(default_factory=Timing)
    _model: object = None

    def _load(self):
        if self._model is None:
            from sentence_transformers import SentenceTransformer

            self._model = SentenceTransformer(self.model_name, device="cpu")
        return self._model

    async def embed_images(self, images: Sequence[bytes], extension: str) -> list:
        model = self._load()
        out = []
        for data in images:
            started = time.perf_counter()
            out.append(model.encode(Image.open(io.BytesIO(data)).convert("RGB")).tolist())
            self.timing.image_seconds.append(time.perf_counter() - started)
        return out

    async def embed_queries(self, questions: Sequence[str]) -> list:
        model = self._load()
        out = []
        for text in questions:
            started = time.perf_counter()
            out.append(model.encode(text).tolist())
            self.timing.query_seconds.append(time.perf_counter() - started)
        return out


async def run_backend(backend: Backend, items: Sequence[Item], extension: str) -> dict:
    """Embed the set and the questions, score, and return the metrics plus mean latencies."""
    images = await backend.embed_images([i.image for i in items], extension)
    queries = await backend.embed_queries([i.question for i in items])
    scores = (
        maxsim_matrix(queries, images)
        if backend.kind == "maxsim"
        else cosine_matrix(queries, images)
    )
    timing: Timing = backend.timing  # type: ignore[attr-defined]
    result = recall_table(scores)
    result["image_latency_s"] = round(
        sum(timing.image_seconds) / max(len(timing.image_seconds), 1), 3
    )
    result["query_latency_s"] = round(
        sum(timing.query_seconds) / max(len(timing.query_seconds), 1), 3
    )
    return result


# -------------------------------------------------- OCR


def _words(text: str) -> list[str]:
    return re.findall(r"[a-z0-9$:]+", text.lower())


def ocr_scores(truth: str, produced: str) -> dict[str, float]:
    """Character similarity (difflib ratio over normalized text) and the fraction of true words
    recovered."""
    a, b = " ".join(_words(truth)), " ".join(_words(produced))
    ratio = difflib.SequenceMatcher(None, a, b, autojunk=False).ratio()
    have = set(_words(produced))
    want = _words(truth)
    return {
        "char_similarity": round(ratio, 3),
        "word_recall": round(sum(w in have for w in want) / max(len(want), 1), 3),
    }


def tesseract_table(items: Sequence[Item]) -> dict:
    """Tesseract (the index's fallback OCR) on the synthetic screenshots, via the production
    ``image_facts`` reader."""
    from .image_facts import _ocr

    if shutil.which("tesseract") is None:
        return {"error": "tesseract is not installed"}
    sims, recalls, seconds = [], [], []
    with tempfile.TemporaryDirectory() as folder:
        for item in items:
            path = Path(folder) / f"{item.name}.png"
            path.write_bytes(item.image)
            started = time.perf_counter()
            read = _ocr(path, 20000)
            seconds.append(time.perf_counter() - started)
            scores = ocr_scores(item.literal, read[0] if read else "")
            sims.append(scores["char_similarity"])
            recalls.append(scores["word_recall"])
    n = len(items)
    version = subprocess.run(["tesseract", "--version"], capture_output=True, check=False)
    return {
        "engine": (version.stdout or version.stderr).decode("utf-8", "replace").splitlines()[0],
        "n": n,
        "char_similarity": round(sum(sims) / n, 3),
        "word_recall": round(sum(recalls) / n, 3),
        "seconds_per_image": round(sum(seconds) / n, 2),
    }


def keyword_overlap_recall(items: Sequence[Item], ocr_texts: Sequence[str]) -> dict[str, float]:
    """Keyword search over OCR text: how often the paraphrased question retrieves its screenshot,
    and how often the
    EXACT phrase (first six words of the drawn text) does. The second is the literal search the
    index promises."""
    from .image_search import _literal_span

    def bm25ish(question: str) -> np.ndarray:
        q = set(_words(question))
        return np.array([len(q & set(_words(t))) for t in ocr_texts], dtype="float32")

    para = recall_table(
        np.stack(
            [
                bm25ish(item.question) + np.random.default_rng(n).random(len(items)) * 1e-3
                for n, item in enumerate(items)
            ]
        )
    )
    hits = 0
    for index, item in enumerate(items):
        phrase = " ".join(item.literal.split()[:6])
        found = [j for j, text in enumerate(ocr_texts) if _literal_span(text, phrase) is not None]
        hits += index in found
    return {
        "paraphrase_recall@1": para["recall@1"],
        "literal_phrase_hit_rate": round(hits / len(items), 3),
    }


# -------------------------------------------------- CLI


def _production_backend(name: str) -> ProductionBackend:
    import httpx

    from .image_embedders import ImageEmbedders
    from .secrets import get_secret

    if name == "nim":
        return ProductionBackend(
            "nim_vl_single",
            "single",
            ImageEmbedders(
                httpx.AsyncClient(timeout=180), "nim", get_secret("NVIDIA_API_KEY") or "", None
            ),
        )
    if name == "google":
        return ProductionBackend(
            "google_gemini_embedding_2",
            "single",
            ImageEmbedders(
                httpx.AsyncClient(timeout=180), "google", get_secret("GOOGLE_API_KEY") or "", None
            ),
        )
    if name == "jina_maxsim":
        # The single provider is unused for the bag; query embedding still calls it, so give it the
        # NIM key.
        return ProductionBackend(
            "jina_v4_maxsim",
            "maxsim",
            ImageEmbedders(
                httpx.AsyncClient(timeout=180),
                "nim",
                get_secret("NVIDIA_API_KEY") or "",
                get_secret("JINA_API_KEY") or "",
            ),
        )
    raise SystemExit(f"unknown backend {name!r}; known: nim, google, jina_maxsim, clip")


async def _main(backends: list[str], count: int, out: Path | None) -> dict:
    shots, photos = make_screenshots(count), make_photos(count)
    report: dict = {
        "synthetic": True,
        "screenshots": len(shots),
        "photos": len(photos),
        "embedders": {},
    }
    for name in backends:
        backend = ClipBackend() if name == "clip" else _production_backend(name)
        row = {}
        for label, items, ext in (("screenshot", shots, ".png"), ("photo", photos, ".jpg")):
            row[label] = await run_backend(backend, items, ext)
            backend.timing = Timing()  # type: ignore[attr-defined]
        report["embedders"][backend.name] = row
    report["tesseract"] = tesseract_table(shots)
    if "error" not in report["tesseract"]:
        from .image_facts import _ocr

        with tempfile.TemporaryDirectory() as folder:
            texts = []
            for item in shots:
                path = Path(folder) / f"{item.name}.png"
                path.write_bytes(item.image)
                read = _ocr(path, 20000)
                texts.append(read[0] if read else "")
        report["ocr_keyword_search"] = keyword_overlap_recall(shots, texts)
    if out:
        out.write_text(json.dumps(report, indent=2), encoding="utf-8")
    return report


def main(argv: list[str] | None = None) -> None:
    """``python -m casebible_index.image_bench [--backends nim,google,jina_maxsim,clip] [--count 20]
    [--out file]``.

    With no ``--backends`` only the local, keyless OCR part runs. Named backends send SYNTHETIC
    images (generated
    here, never read from disk) to the named provider; keys come from the environment or the Windows
    keychain."""
    import argparse

    parser = argparse.ArgumentParser(description=main.__doc__)
    parser.add_argument("--backends", default="")
    parser.add_argument("--count", type=int, default=20)
    parser.add_argument("--out", type=Path)
    args = parser.parse_args(argv)
    names = [n for n in args.backends.split(",") if n]
    print(json.dumps(asyncio.run(_main(names, args.count, args.out)), indent=2))


if __name__ == "__main__":
    main(sys.argv[1:])

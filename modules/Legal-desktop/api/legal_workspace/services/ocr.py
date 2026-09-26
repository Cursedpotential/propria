"""OCR for screenshots and photographed documents, with Tesseract.

> _Byline: Claude Code · Fable 5.1 · 2026-09-21_
CAT5 section B / DOC-06. Tesseract is installed in the legal-api image (CPU only).
Output is machine-read text with per-line confidence and pixel boxes so a reader
can be pointed at the place on the image. OCR text is a derivative: it is never
the same thing as a native export's text, and it is not court-safe.
"""

from __future__ import annotations

import csv
import hashlib
import io
import shutil
import subprocess
from pathlib import Path

from pydantic import BaseModel

IMAGE_SUFFIXES = frozenset({".png", ".jpg", ".jpeg", ".webp", ".tif", ".tiff", ".bmp", ".gif"})

# Tesseract page-segmentation modes, named for what the image looks like.
_LAYOUTS = {"auto": "3", "block": "6", "sparse": "11"}


class OcrUnavailable(RuntimeError):
    """The tesseract binary is not installed where legal-api runs."""


class OcrLine(BaseModel):
    text: str
    confidence: float
    left: int
    top: int
    width: int
    height: int


class OcrResult(BaseModel):
    ok: bool
    engine: str
    source_name: str
    content_hash: str
    language: str
    layout: str
    text: str
    lines: list[OcrLine]
    word_count: int
    mean_confidence: float
    derivative: bool = True
    court_safe: bool = False


def _version(binary: str) -> str:
    done = subprocess.run([binary, "--version"], capture_output=True, timeout=20, check=False)
    first = (done.stdout or done.stderr).decode("utf-8", "replace").splitlines()
    return first[0].strip() if first else "tesseract"


def ocr_image(src: Path, *, language: str = "eng", layout: str = "auto") -> OcrResult:
    """Read the text in an image. `layout`: auto, block (one column of text), sparse (scattered UI text)."""
    if src.suffix.lower() not in IMAGE_SUFFIXES:
        raise ValueError(f"unsupported image format: {src.suffix or '(none)'}")
    if layout not in _LAYOUTS:
        raise ValueError(f"layout must be one of {sorted(_LAYOUTS)}")
    if not language.replace("+", "").replace("_", "").isalnum():
        raise ValueError("language must look like 'eng' or 'eng+spa'")
    binary = shutil.which("tesseract")
    if binary is None:
        raise OcrUnavailable("tesseract is not installed on this host")
    try:
        done = subprocess.run(  # fixed argv, no shell
            [binary, str(src), "stdout", "-l", language, "--psm", _LAYOUTS[layout], "tsv"],
            capture_output=True,
            timeout=180,
            check=False,
        )
    except subprocess.TimeoutExpired as exc:
        raise ValueError(f"tesseract timed out on {src.name}") from exc
    if done.returncode != 0:
        detail = done.stderr.decode("utf-8", "replace").strip()[:200]
        raise ValueError(f"tesseract could not read {src.name}: {detail}")

    rows = csv.DictReader(
        io.StringIO(done.stdout.decode("utf-8", "replace")), delimiter="\t", quoting=csv.QUOTE_NONE
    )
    grouped: dict[tuple[str, str, str, str], list[dict[str, str]]] = {}
    for row in rows:
        word = (row.get("text") or "").strip()
        if row.get("level") != "5" or not word:
            continue
        key = (row["page_num"], row["block_num"], row["par_num"], row["line_num"])
        grouped.setdefault(key, []).append(row)

    lines: list[OcrLine] = []
    confidences: list[float] = []
    for words in grouped.values():
        left = min(int(w["left"]) for w in words)
        top = min(int(w["top"]) for w in words)
        right = max(int(w["left"]) + int(w["width"]) for w in words)
        bottom = max(int(w["top"]) + int(w["height"]) for w in words)
        confs = [float(w["conf"]) for w in words]
        confidences.extend(confs)
        lines.append(
            OcrLine(
                text=" ".join(w["text"].strip() for w in words),
                confidence=round(sum(confs) / len(confs), 1),
                left=left,
                top=top,
                width=right - left,
                height=bottom - top,
            )
        )
    return OcrResult(
        ok=True,
        engine=_version(binary),
        source_name=src.name,
        content_hash=hashlib.sha256(src.read_bytes()).hexdigest(),
        language=language,
        layout=layout,
        text="\n".join(line.text for line in lines),
        lines=lines,
        word_count=len(confidences),
        mean_confidence=round(sum(confidences) / len(confidences), 1) if confidences else 0.0,
    )

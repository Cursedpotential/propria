"""Facts read from an image without any model: metadata, original time, device, fallback text.

> _Byline: Claude Code · Fable 5.1 · 2026-09-21_
exiftool and tesseract are optional binaries: when one is missing the fact is left out and a
note says so; nothing is guessed. Tesseract is the text FALLBACK (owner 2026-09-21): the
vision embedding finds the image, OCR text makes exact words, names and numbers searchable.
OCR text is a derivative and is never equated with a native export's text.
"""

from __future__ import annotations

import json
import shutil
import subprocess
from dataclasses import dataclass, field
from pathlib import Path

from .image_decode import HEIF_EXTENSIONS, jpeg_derivative
from .original_time import OriginalTime, resolve_original_time

_SCREENSHOT_HINTS = (
    "screenshot",
    "screen shot",
    "screen_shot",
    "screen-shot",
    "screencap",
    "screen_cap",
)


@dataclass
class ImageFacts:
    original_time: OriginalTime
    device: str = ""
    software: str = ""
    gps: str = ""
    width: int = 0
    height: int = 0
    is_screenshot: bool = False
    ocr_text: str = ""
    ocr_confidence: float = 0.0
    notes: list[str] = field(default_factory=list)


def _run(argv: list[str], timeout: int, data: bytes | None = None) -> subprocess.CompletedProcess[bytes] | None:
    """Run a metadata or OCR tool with fixed arguments and a bounded timeout.

    Inputs: argv, timeout and optional stdin bytes. Output: process result or None on launch or
    timeout failure. Side effects: launches one local subprocess. Pick this for exiftool/Tesseract
    so paths are never interpolated into a shell command.
    """
    try:
        return subprocess.run(
            argv, input=data, capture_output=True, timeout=timeout, check=False
        )  # fixed argv, no shell
    except (subprocess.TimeoutExpired, OSError):
        return None


def _exif(path: Path) -> dict[str, object] | None:
    binary = shutil.which("exiftool")
    if binary is None:
        return None
    done = _run([binary, "-json", "-G1", "-a", "-struct", "-c", "%+.6f", str(path)], 120)
    if done is None:
        return None
    try:
        fields = dict(json.loads(done.stdout.decode("utf-8", "replace"))[0])
    except (ValueError, IndexError):
        return None
    fields.pop("SourceFile", None)
    return fields


def _ocr(path: Path | bytes, max_chars: int) -> tuple[str, float] | None:
    """Read Tesseract words from a path or JPEG bytes sent on stdin.

    Inputs: image source and character cap. Output: text/confidence or None on unavailable OCR.
    Side effects: runs Tesseract. Pick bytes for HEIC/HEIF derivatives that Tesseract cannot open.
    """
    binary = shutil.which("tesseract")
    if binary is None:
        return None
    stdin = isinstance(path, bytes)
    done = _run([binary, "-" if stdin else str(path), "stdout", "-l", "eng", "--psm", "3", "tsv"], 180, path if stdin else None)
    if done is None or done.returncode != 0:
        return None
    words: list[str] = []
    confidences: list[float] = []
    last_line: tuple[str, ...] | None = None
    for row in done.stdout.decode("utf-8", "replace").splitlines()[1:]:
        cells = row.split("\t")
        if len(cells) < 12 or cells[0] != "5" or not cells[11].strip():
            continue
        line = tuple(cells[1:5])
        if last_line is not None and line != last_line:
            words.append("\n")
        last_line = line
        words.append(cells[11].strip())
        confidences.append(float(cells[10]))
    text = " ".join(words).replace(" \n ", "\n")[:max_chars]
    return text, round(sum(confidences) / len(confidences), 1) if confidences else 0.0


def read_image_facts(
    path: Path,
    original_name: str,
    *,
    takeout_sidecar_json: str | None,
    ocr: bool,
    max_ocr_chars: int,
) -> ImageFacts:
    fields = _exif(path)
    facts = ImageFacts(
        original_time=resolve_original_time(fields or {}, original_name, takeout_sidecar_json)
    )
    if fields is None:
        facts.notes.append("exiftool unavailable or unreadable: embedded metadata not read")
    else:
        by_tag = {key.split(":", 1)[-1]: value for key, value in fields.items()}
        for key, value in fields.items():
            if key.startswith("Composite:GPS"):
                by_tag[key.split(":", 1)[-1]] = value
        facts.device = " ".join(str(by_tag[t]) for t in ("Make", "Model") if t in by_tag)
        facts.software = str(by_tag.get("Software") or by_tag.get("CreatorTool") or "")
        if "GPSLatitude" in by_tag and "GPSLongitude" in by_tag:
            facts.gps = f"{by_tag['GPSLatitude']},{by_tag['GPSLongitude']}"
        facts.width = int(by_tag.get("ImageWidth") or 0)
        facts.height = int(by_tag.get("ImageHeight") or 0)
        comment = str(by_tag.get("UserComment") or "").casefold()
        facts.is_screenshot = "screenshot" in comment
    lowered = original_name.casefold()
    facts.is_screenshot = facts.is_screenshot or any(hint in lowered for hint in _SCREENSHOT_HINTS)
    if ocr:
        source = jpeg_derivative(path, edge=2048) if path.suffix.casefold() in HEIF_EXTENSIONS else path
        read = _ocr(source, max_ocr_chars)
        if read is None:
            facts.notes.append("tesseract unavailable or failed: no fallback text")
        else:
            facts.ocr_text, facts.ocr_confidence = read
    return facts

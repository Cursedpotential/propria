"""OCR engines for the image stage: one function per engine, selected by name. One job: read the
text in a picture.

> _Byline: Claude Code · Sonnet 5.5 · 2026-10-03_

Toolbox rule (owner 2026-10-03): every option stays selectable and the default is a setting,
``INTAKE_IMAGES_OCR_ENGINE``.

    tesseract   the default. The Tesseract binary in the worker image (``image_facts._ocr``):
    English, page
                segmentation 3, word confidences averaged. Measured on synthetic chat screenshots:
                word recall 0.946,
                0.32 s per image on CPU (``image_bench.tesseract_table``).
    doctr       docTR (``python-doctr``), a small detection + recognition model pair. OPTIONAL:
    neither the package nor
                its weights are in the worker image, and nothing here downloads them. Selecting it
                without them raises
                ``OcrUnavailable`` (a configuration error, never a silent empty result). Not yet run
                against the
                bench: it is a candidate to measure on the box that hosts it.

OCR text is a derivative of the picture. It is stored beside the image and never equated with a
native export's text.
"""

from __future__ import annotations

from collections.abc import Callable
from pathlib import Path

from .image_facts import _ocr

DEFAULT_ENGINE = "tesseract"


class OcrUnavailable(ValueError):
    """The selected engine cannot run in this environment. A ValueError so the Activity treats it as
    non-retryable."""


def _tesseract(path: Path, max_chars: int) -> tuple[str, float]:
    result = _ocr(path, max_chars)
    if result is None:
        raise OcrUnavailable("tesseract is not installed or failed to start in this environment")
    return result


_DOCTR_MODEL: object | None = None


def _doctr(path: Path, max_chars: int) -> tuple[str, float]:
    global _DOCTR_MODEL
    try:
        from doctr.io import DocumentFile
        from doctr.models import ocr_predictor
    except ImportError as exc:
        raise OcrUnavailable(
            "python-doctr is not installed; install it and its weights on the box that runs OCR"
        ) from exc
    if _DOCTR_MODEL is None:
        _DOCTR_MODEL = ocr_predictor(pretrained=True)
    result = _DOCTR_MODEL(DocumentFile.from_images(str(path)))  # type: ignore[operator]
    confidences: list[float] = []
    for page in result.pages:
        for block in page.blocks:
            for line in block.lines:
                confidences.extend(float(word.confidence) * 100.0 for word in line.words)
    text = result.render()[:max_chars]
    return text, round(sum(confidences) / len(confidences), 1) if confidences else 0.0


OCR_ENGINES: dict[str, Callable[[Path, int], tuple[str, float]]] = {
    "tesseract": _tesseract,
    "doctr": _doctr,
}


def read_text(engine: str, path: Path, max_chars: int) -> tuple[str, float]:
    """Text and mean word confidence (0-100) of one image file with the named engine.

    Inputs: an engine name from ``OCR_ENGINES``, an image file, a character cap. Output: ``(text,
    confidence)``; an
    image with no text returns ``("", 0.0)``. Raises ``OcrUnavailable`` when the engine cannot run
    and ``ValueError``
    for an unknown engine. Side effects: runs a subprocess (tesseract) or a model (docTR). Pick this
    over a gateway
    OCR tool when the file is already on the spool; the gateway tools (``ocr_image.*``) take a
    bucket locator."""
    try:
        run = OCR_ENGINES[engine]
    except KeyError:
        raise OcrUnavailable(
            f"unknown OCR engine {engine!r}; known: {sorted(OCR_ENGINES)}"
        ) from None
    return run(path, max_chars)

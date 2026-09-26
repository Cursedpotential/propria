"""Owner-work-product PDF redaction via pikepdf content-stream edit.

> _Byline: Grok · grok-4.6 · 2026-08-18_
Not a black-box overlay. Not Agno evidence. Not court-safe.
Replaces token bytes inside the page content stream so they are gone
from the stream text, not merely covered.
"""

from __future__ import annotations

from pathlib import Path

from pikepdf import Array, Pdf
from pydantic import BaseModel, Field


class RedactionResult(BaseModel):
    ok: bool
    tokens: list[str]
    remaining: list[str] = Field(default_factory=list)
    output_path: str
    court_safe: bool = False
    exportable: bool = False
    disclosure: str = "private_strategy"


def write_text_pdf(path: Path, text: str) -> Path:
    """Tiny owner-produced PDF for tests and local redaction trials."""
    from pikepdf import Dictionary, Name

    path.parent.mkdir(parents=True, exist_ok=True)
    pdf = Pdf.new()
    font = pdf.make_indirect(
        Dictionary(Type=Name.Font, Subtype=Name.Type1, BaseFont=Name.Helvetica)
    )
    escaped = text.replace("\\", "\\\\").replace("(", "\\(").replace(")", "\\)")
    stream = pdf.make_stream(f"BT /F1 12 Tf 72 720 Td ({escaped}) Tj ET".encode("latin-1"))
    page = pdf.add_blank_page(page_size=(612, 792))
    page.Contents = stream
    page.Resources = Dictionary(Font=Dictionary(F1=font))
    pdf.save(path)
    return path


def extract_content_stream_text(path: Path) -> str:
    """Raw decompressed content-stream bytes as text. Used to prove removal."""
    pdf = Pdf.open(path)
    chunks: list[str] = []
    for page in pdf.pages:
        contents = page.get("/Contents")
        if contents is None:
            continue
        streams = list(contents) if isinstance(contents, Array) else [contents]
        for item in streams:
            chunks.append(item.read_bytes().decode("latin-1", errors="replace"))
    return "".join(chunks)


def redact_content_stream(src: Path, dest: Path, tokens: list[str]) -> RedactionResult:
    """Replace each token in page content streams. Fail closed if any remain."""
    if not tokens:
        raise ValueError("at least one redaction token is required")
    dest.parent.mkdir(parents=True, exist_ok=True)
    pdf = Pdf.open(src)
    for page in pdf.pages:
        if page.get("/Contents") is None:
            continue
        page.contents_coalesce()
        data = bytearray(page.Contents.read_bytes())
        for token in tokens:
            needle = token.encode("latin-1")
            replacement = b" " * len(needle)
            start = 0
            while True:
                idx = data.find(needle, start)
                if idx < 0:
                    break
                data[idx : idx + len(needle)] = replacement
                start = idx + len(needle)
        page.Contents = pdf.make_stream(bytes(data))
    pdf.save(dest)
    leftover = extract_content_stream_text(dest)
    remaining = [token for token in tokens if token in leftover]
    return RedactionResult(
        ok=not remaining,
        tokens=list(tokens),
        remaining=remaining,
        output_path=str(dest),
        court_safe=False,
        exportable=False,
    )

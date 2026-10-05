"""Bind existing page/HTML extractors to the exact verified local snapshot supplied by the Go adapter.

Inputs: one stdin descriptor with path, SHA-256, provider version and media type.
Outputs: bounded JSON pages, extractor version and actual input identity; errors carry no source content.
Side effects: reads only the staged file; never deletes, fetches sources, changes a proposal or invokes a model.
Choose this same-host bridge when generic gateway output does not attest the bytes it extracted.
Byline: Codex · GPT-6.1 · 2026-10-04.
"""
from __future__ import annotations

import hashlib
import json
import sys
from importlib import metadata
from pathlib import Path

MAX_INPUT_BYTES = 8 << 20
MAX_TEXT_BYTES = 1 << 20
MAX_PAGES = 512
RUNTIME_CONTRACT = "library-validation-runtime/v1"


def runtime_info() -> dict:
    """Check the exact existing parser imports without loading the whole tool registry or fetching anything.

    Inputs: interpreter environment. Outputs: runtime contract and installed parser versions.
    Side effects: imports only; choose at worker admission and image build before registering validation Activities.
    """
    import lxml.html
    import pypdf
    from server.tools.extractors.extract_text import parse
    from server.tools.extractors.html_text.lxml_html import extract_html_lxml
    if not callable(parse) or not callable(extract_html_lxml):
        raise ValueError("existing extractor contract unavailable")
    return {"contract": RUNTIME_CONTRACT, "python_version": sys.version.split()[0],
            "html_extractor": "html.lxml", "html_version": metadata.version("lxml"),
            "pdf_extractor": "documents.extract-text/pypdf", "pdf_version": metadata.version("pypdf")}


def checked_bytes(path: Path, expected: str) -> bytes:
    """Read one bounded regular snapshot and compare its full digest before and after extraction.

    Inputs: staged path and expected raw hex digest. Outputs: exact bytes. Side effects: file read only.
    Choose over trusting the caller's hash or a parser-returned URL.
    """
    if path.is_symlink() or not path.is_file() or path.stat().st_size > MAX_INPUT_BYTES:
        raise ValueError("invalid staged snapshot")
    raw = path.read_bytes()
    if not raw or hashlib.sha256(raw).hexdigest() != expected:
        raise ValueError("staged snapshot digest differs")
    return raw


def extract(descriptor: dict) -> dict:
    """Invoke only the existing PDF native-page or lxml HTML extractor on the verified snapshot.

    Inputs: exact staged descriptor. Outputs: pages and independently checked input bindings.
    Side effects: existing extractor imports and file reads; no OCR/model fallback or remote requests.
    Choose for source quote checks, retaining PDF page boundaries rather than decoding PDF bytes as text.
    """
    path = Path(descriptor["path"])
    expected = descriptor["input_sha256"]
    version = descriptor["version_id"]
    if not path.is_absolute() or len(expected) != 64 or not version or version == "null":
        raise ValueError("invalid extraction identity")
    raw = checked_bytes(path, expected)
    if descriptor["media_type"] == "application/pdf":
        if not raw.startswith(b"%PDF-"):
            raise ValueError("invalid PDF signature")
        from server.tools.extractors.extract_text import parse
        result = parse({"path": str(path), "native_only": True})
        method = result["stats"]["method"]
        if method not in ("pypdf", "pdfplumber") or result["stats"].get("ocr_used"):
            raise ValueError("unsupported PDF extraction method")
        extractor = "documents.extract-text/" + method
        extractor_version = metadata.version(method)
    elif descriptor["media_type"] in ("text/html", "application/xhtml+xml"):
        raw.decode("utf-8", errors="strict")
        from server.tools.extractors.html_text.lxml_html import extract_html_lxml
        result = extract_html_lxml({"path": str(path)})
        extractor = "html.lxml"
        extractor_version = result["stats"]["library_version"]
    else:
        raise ValueError("unsupported snapshot media type")
    checked_bytes(path, expected)
    pages = result["pages"]
    if not pages or len(pages) > MAX_PAGES or sum(len(p.encode("utf-8")) for p in pages) > MAX_TEXT_BYTES:
        raise ValueError("extraction budget exceeded")
    return {"pages": pages, "input_sha256": hashlib.sha256(raw).hexdigest(), "version_id": version,
            "extractor": extractor, "extractor_version": extractor_version,
            "low_confidence": bool(result["stats"].get("low_confidence"))}


def main() -> None:
    """Read one bounded descriptor and emit the authenticated-input extraction envelope.

    Inputs: stdin JSON. Outputs: stdout JSON or nonzero exit. Side effects: extractor file read.
    Choose as the ProcessExtractor child entry point; exception content is never printed.
    """
    if sys.argv[1:] == ["--check-runtime"]:
        json.dump(runtime_info(), sys.stdout)
        return
    if sys.argv[1:]:
        raise ValueError("unsupported bridge operation")
    descriptor = json.loads(sys.stdin.buffer.read(16385))
    json.dump(extract(descriptor), sys.stdout, ensure_ascii=False)


if __name__ == "__main__":
    try:
        main()
    except Exception:
        sys.stderr.write("pinned snapshot extraction failed\n")
        sys.exit(1)

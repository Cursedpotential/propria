"""Bates overlay stamps the number onto an owner-produced PDF.

> _Byline: Grok · grok-4.6 · 2026-08-18_
Drives stamp_bates_pdf and POST /v1/bates:stamp. Not Agno evidence.
"""

from pathlib import Path

from fastapi.testclient import TestClient

from legal_workspace.api import main as main_mod
from legal_workspace.api.main import app
from legal_workspace.services import workspace as workspace_mod
import re

from legal_workspace.services.bates import stamp_bates_pdf
from legal_workspace.services.redaction import extract_content_stream_text, write_text_pdf

STAMP = "GENESEE-000001"


def _stream_text(path: Path) -> str:
    raw = extract_content_stream_text(path)
    return re.sub(r"\\([0-7]{3})", lambda match: chr(int(match.group(1), 8)), raw)


def test_stamp_bates_pdf_puts_number_on_page(tmp_path: Path) -> None:
    src = write_text_pdf(tmp_path / "in.pdf", "Owner draft body.")
    dest = tmp_path / "out.pdf"
    result = stamp_bates_pdf(src, dest, STAMP)
    assert result.ok is True
    assert result.court_safe is False
    assert result.bates_number == STAMP
    leftover = _stream_text(dest)
    assert STAMP in leftover
    assert dest.is_file()


def test_http_bates_stamp(tmp_path: Path) -> None:
    store = workspace_mod.get_workspace(tmp_path)
    main_mod.WORKSPACE = store
    src = write_text_pdf(tmp_path / "upload.pdf", "Exhibit draft.")
    client = TestClient(app)
    response = client.post(
        "/v1/bates:stamp",
        data={"bates_number": STAMP},
        files={"file": ("upload.pdf", src.read_bytes(), "application/pdf")},
    )
    assert response.status_code == 200, response.text
    body = response.json()
    assert body["ok"] is True
    assert body["bates_number"] == STAMP
    assert body["court_safe"] is False
    leftover = _stream_text(Path(body["output_path"]))
    assert STAMP in leftover

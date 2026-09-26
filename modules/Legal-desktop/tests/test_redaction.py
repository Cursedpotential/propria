"""pikepdf content-stream redaction removes the token from the stream.

> _Byline: Grok · grok-4.6 · 2026-08-18_
Drives redact_content_stream on a tiny owner-produced PDF. Not court-safe.
"""

from pathlib import Path

from fastapi.testclient import TestClient

from legal_workspace.api import main as main_mod
from legal_workspace.api.main import app
from legal_workspace.services import workspace as workspace_mod
from legal_workspace.services.redaction import (
    extract_content_stream_text,
    redact_content_stream,
    write_text_pdf,
)

TOKEN = "SECRET-TOKEN-XYZ"


def test_redaction_removes_token_from_content_stream(tmp_path: Path) -> None:
    src = write_text_pdf(tmp_path / "in.pdf", f"Visible line with {TOKEN} still in the stream.")
    assert TOKEN in extract_content_stream_text(src)
    dest = tmp_path / "out.pdf"
    result = redact_content_stream(src, dest, [TOKEN])
    leftover = extract_content_stream_text(dest)
    assert result.ok is True
    assert result.court_safe is False
    assert result.exportable is False
    assert TOKEN not in leftover
    assert result.remaining == []
    assert dest.is_file()


def test_redaction_fail_closed_when_token_missing_from_request(tmp_path: Path) -> None:
    src = write_text_pdf(tmp_path / "in.pdf", "no secret here")
    try:
        redact_content_stream(src, tmp_path / "out.pdf", [])
        raise AssertionError("empty token list must fail")
    except ValueError as exc:
        assert "token" in str(exc)


def test_http_redaction_removes_token(tmp_path: Path) -> None:
    store = workspace_mod.get_workspace(tmp_path)
    main_mod.WORKSPACE = store
    src = write_text_pdf(tmp_path / "upload.pdf", f"Owner draft mentions {TOKEN}.")
    client = TestClient(app)
    response = client.post(
        "/v1/redactions",
        data={"tokens": TOKEN},
        files={"file": ("upload.pdf", src.read_bytes(), "application/pdf")},
    )
    assert response.status_code == 200, response.text
    body = response.json()
    assert body["ok"] is True
    assert body["court_safe"] is False
    leftover = extract_content_stream_text(Path(body["output_path"]))
    assert TOKEN not in leftover

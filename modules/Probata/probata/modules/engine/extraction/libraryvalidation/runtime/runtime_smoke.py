"""Exercise the baked existing extractors against synthetic HTML and two PDF pages without network or credentials.

Inputs: bounded worker image environment. Outputs: input-binding and parser fixture proof JSON.
Side effects: retains two private fixture files; no source, database, deployment or deletion operations.
Choose as an isolated image smoke test; success does not prove a deployed Temporal workflow or legal currency.
Byline: Codex · GPT-6.1 · 2026-10-04.
"""
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import sys


def fixture_pdf() -> bytes:
    """Generate two synthetic PDF pages for page-boundary assertions; inputs: none; outputs: test bytes; effects: none."""
    objects = ["<< /Type /Catalog /Pages 2 0 R >>", "<< /Type /Pages /Kids [3 0 R 4 0 R] /Count 2 >>",
               "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 5 0 R >> >> /Contents 6 0 R >>",
               "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 5 0 R >> >> /Contents 7 0 R >>",
               "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>"]
    for text in ["Synthetic first page.", "Synthetic exact second page."]:
        stream = f"BT /F1 11 Tf 50 700 Td ({text}) Tj ET"
        objects.append(f"<< /Length {len(stream)} >>\nstream\n{stream}\nendstream")
    raw = b"%PDF-1.4\n"
    offsets = [0]
    for number, obj in enumerate(objects, 1):
        offsets.append(len(raw))
        raw += f"{number} 0 obj\n{obj}\nendobj\n".encode()
    start = len(raw)
    raw += f"xref\n0 {len(offsets)}\n0000000000 65535 f \n".encode()
    for offset in offsets[1:]:
        raw += f"{offset:010d} 00000 n \n".encode()
    raw += f"trailer\n<< /Size {len(offsets)} /Root 1 0 R >>\nstartxref\n{start}\n%%EOF\n".encode()
    return raw


def main() -> None:
    """Verify the shipped regular interpreter and actual pinned extraction fixtures; inputs: image paths; outputs: proof JSON."""
    assert Path(sys.executable).is_file() and not Path(sys.executable).is_symlink()
    sys.path.insert(0, os.environ["TOOLKIT_VALIDATION_PROJECT_ROOT"])
    spec = importlib.util.spec_from_file_location("validation_bridge", os.environ["TOOLKIT_VALIDATION_BRIDGE_FILE"])
    bridge = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(bridge)
    root = Path(os.environ["TOOLKIT_VALIDATION_SCRATCH_ROOT"]) / "runtime-smoke"
    root.mkdir(mode=0o700, exist_ok=True)
    fixtures = [("source.html", "text/html", b"<html><body><script>False text.</script><p>Synthetic exact HTML.</p></body></html>", 1),
                ("source.pdf", "application/pdf", fixture_pdf(), 2)]
    for name, media, raw, count in fixtures:
        path = root / name
        path.write_bytes(raw)
        sha = hashlib.sha256(raw).hexdigest()
        output = bridge.extract({"path": str(path), "input_sha256": sha, "version_id": "synthetic-pinned-v1", "media_type": media})
        assert output["input_sha256"] == sha and output["version_id"] == "synthetic-pinned-v1"
        assert len(output["pages"]) == count
        assert "False text" not in "\n".join(output["pages"])
        assert "Synthetic exact" in output["pages"][-1]
        if count == 2:
            assert "second page" not in output["pages"][0]
    print(json.dumps({"runtime": bridge.runtime_info(), "regular_interpreter": True,
                      "html_and_pdf_input_bindings": True, "pdf_page_boundaries": True, "network": "disabled", "production_writes": 0}))


if __name__ == "__main__":
    main()

"""The installed API wheel must carry the SQLite schema resource."""

from __future__ import annotations

import os
import subprocess
import sys
import zipfile
from pathlib import Path


def test_sqlite_schema_is_available_from_built_wheel(tmp_path: Path) -> None:
    repo = Path(__file__).resolve().parents[1]
    wheelhouse = tmp_path / "wheelhouse"
    wheelhouse.mkdir()
    subprocess.run(
        [
            sys.executable,
            "-m",
            "pip",
            "wheel",
            "--no-deps",
            "--no-build-isolation",
            "--wheel-dir",
            str(wheelhouse),
            str(repo),
        ],
        check=True,
        capture_output=True,
        text=True,
    )
    wheel = next(wheelhouse.glob("legal_workspace-*.whl"))
    with zipfile.ZipFile(wheel) as archive:
        resource_name = "legal_workspace/sql/0001_legal_os_sqlite.sql"
        assert resource_name in archive.namelist()
        assert archive.read(resource_name) == (repo / "sql/0001_legal_os_sqlite.sql").read_bytes()
        installed = tmp_path / "installed"
        archive.extractall(installed)

    probe = """
from pathlib import Path
from legal_workspace.services.sqlite_store import load_settings

settings = load_settings(Path.cwd() / 'probe-store')
assert settings.theme == "dark"
assert settings.case_phase == "Discovery"
"""
    environment = os.environ.copy()
    environment["PYTHONPATH"] = str(installed)
    subprocess.run(
        [sys.executable, "-c", probe],
        check=True,
        cwd=tmp_path,
        env=environment,
        capture_output=True,
        text=True,
    )

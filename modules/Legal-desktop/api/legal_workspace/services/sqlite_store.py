"""SQLite WAL store for Category 1 app settings.

> _Byline: Grok · grok-4.6 · 2026-08-18_
Local file legal.sqlite. Does not apply sql/0001_legal_os_bootstrap.sql
to the live Postgres cluster (Type 1 HOLD).
"""

from __future__ import annotations

import sqlite3
from dataclasses import dataclass
from datetime import UTC, datetime
from importlib.resources import files
from pathlib import Path

_SCHEMA_RESOURCE = files("legal_workspace").joinpath("sql/0001_legal_os_sqlite.sql")
_ALLOWED_PHASES = frozenset({"Discovery", "Motions", "Hearing", "Trial"})


@dataclass(frozen=True)
class AppSettingsRow:
    confidential_mode: bool
    theme: str
    display_timezone: str
    case_phase: str


def db_path(store_dir: Path) -> Path:
    return Path(store_dir) / "legal.sqlite"


def connect(store_dir: Path) -> sqlite3.Connection:
    path = db_path(store_dir)
    path.parent.mkdir(parents=True, exist_ok=True)
    conn = sqlite3.connect(path)
    conn.row_factory = sqlite3.Row
    conn.execute("PRAGMA journal_mode=WAL")
    conn.execute("PRAGMA foreign_keys=ON")
    conn.executescript(_SCHEMA_RESOURCE.read_text(encoding="utf-8"))
    conn.commit()
    return conn


def load_settings(store_dir: Path) -> AppSettingsRow:
    conn = connect(store_dir)
    try:
        row = conn.execute(
            "SELECT theme, display_timezone, confidential_mode, case_phase "
            "FROM legal_core_app_settings WHERE id = 1"
        ).fetchone()
    finally:
        conn.close()
    if row is None:
        return AppSettingsRow(
            confidential_mode=False,
            theme="dark",
            display_timezone="America/New_York",
            case_phase="Discovery",
        )
    return AppSettingsRow(
        confidential_mode=bool(row["confidential_mode"]),
        theme=row["theme"],
        display_timezone=row["display_timezone"],
        case_phase=row["case_phase"],
    )



def save_settings(
    store_dir: Path,
    *,
    confidential_mode: bool | None = None,
    theme: str | None = None,
    display_timezone: str | None = None,
    case_phase: str | None = None,
) -> AppSettingsRow:
    current = load_settings(store_dir)
    next_conf = current.confidential_mode if confidential_mode is None else bool(confidential_mode)
    next_theme = current.theme if theme is None else theme
    next_tz = current.display_timezone if display_timezone is None else display_timezone.strip()
    next_phase = current.case_phase if case_phase is None else case_phase
    if next_theme != "dark":
        raise ValueError("theme is dark-only")
    if not next_tz:
        next_tz = "America/New_York"
    if next_phase not in _ALLOWED_PHASES:
        raise ValueError(f"case_phase must be one of {sorted(_ALLOWED_PHASES)}")
    stamp = datetime.now(UTC).isoformat()
    with connect(store_dir) as conn:
        conn.execute(
            """
            UPDATE legal_core_app_settings
            SET theme = ?, display_timezone = ?, confidential_mode = ?,
                case_phase = ?, updated_at = ?
            WHERE id = 1
            """,
            (next_theme, next_tz, 1 if next_conf else 0, next_phase, stamp),
        )
        conn.commit()
    return load_settings(store_dir)

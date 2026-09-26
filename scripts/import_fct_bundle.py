# /// script
# requires-python = ">=3.11"
# dependencies = ["psycopg[binary]"]
# ///
"""scripts/import_fct_bundle.py — import a family-court-toolkit (FCT) platform export
bundle into the platform's context-layer PostgreSQL tables.

Byline: Claude Code · Sonnet 5 · 2026-09-07 (owner order 2026-09-07 13:16: "create an
adapter or import/export mechanism to export to the platform").

WHAT THIS DOES
    Reads a bundle directory written by the family-court-toolkit plugin
    (``~/.config/family-court-toolkit/exports/platform-<ISO ts>/``): a
    ``manifest.json`` plus one ``<table>.ndjson`` per bundle table and one
    ``edges.ndjson``. Maps each bundle record onto the platform's existing
    context-layer tables (``working.candidate_entity`` / ``candidate_event`` /
    ``candidate_fact``, ``timeline.event_candidate``, ``context.source``) —
    see ``docs/pending-review/2026-09-07-fct-bundle-import-mapping.md`` for the
    full table-by-table rationale. Bundle tables with no clean, schema-change-free
    home are counted and reported as unmapped; THIS SCRIPT NEVER PROPOSES OR
    APPLIES A SCHEMA CHANGE.

    Every one of the platform tables this script writes to is a "candidate"
    landing zone the schema already designed for exactly this shape of
    externally-produced, pending-review data (D-082: "an AI-chat-derived row is
    a lead, never evidence"). No identity registry row (``registry.*``) and no
    evidence-tier row (``evidence.*``) is ever created or promoted by this
    script — promotion out of the candidate tables is a separate, deliberate,
    human-gated step elsewhere in the platform.

MODES
    ``--dry-run`` (default, no ``--apply``): validates and maps every record,
    prints per-table counts, unmapped tables, and validation errors. Attempts a
    *read-only-in-effect* database round trip (a transaction that is ALWAYS
    rolled back) so the insert-vs-already-present counts are accurate; if the
    database is unreachable it degrades to validation-only counts and says so —
    dry-run must work with no database connectivity at all.
    ``--apply``: runs every planned insert inside ONE transaction. Unless
    ``--commit`` is ALSO given, the transaction is rolled back after printing
    counts (this repository's migration-verification convention — see
    scripts/rehearse_platform_migrations.py). ``--apply`` without a reachable
    database is a hard failure (exit 2).
    ``--commit``: only meaningful together with ``--apply`` — actually persists.

IDEMPOTENCY
    Every mapped target table already carries a unique index this script keys
    off with ``ON CONFLICT ... DO NOTHING``:
      - working.candidate_entity / candidate_event / candidate_fact:
        UNIQUE(source_raw_table, source_raw_id, content_sha256)
      - timeline.event_candidate: UNIQUE(source_system, source_record_id,
        source_record_version) — source_record_version is set to the record's
        own content hash, so a corrected record (different content) lands as a
        NEW row rather than silently overwriting the old one (D-082: a
        correction is a new row, never an edit).
      - context.source: UNIQUE(source_key)
    No pre-check branch is needed — a real unique key exists for every mapped
    table.

CONNECTION
    DB_HOST defaults to the tailnet IP (100.91.190.107, per AGENTS.md) —
    override with the DB_HOST env var. DB_PORT/DB_DATABASE/DB_USER/DB_PASS come
    from the environment first, else are tolerant-regex-parsed (NEVER
    `source`d, NEVER printed) from ~/.secrets/probata.env, falling back to the
    legacy ~/.secrets/Agno-MCP-Platform.env name.

USAGE
    uv run python scripts/import_fct_bundle.py --bundle <dir>                 # dry-run
    uv run python scripts/import_fct_bundle.py --bundle <dir> --apply         # apply, rollback
    uv run python scripts/import_fct_bundle.py --bundle <dir> --apply --commit  # apply, persist
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import sys
from dataclasses import dataclass, field
from datetime import date, datetime
from pathlib import Path
from typing import Any

try:
    import psycopg
except ImportError:  # pragma: no cover - exercised only outside the project venv
    psycopg = None  # type: ignore[assignment]

TAILNET_DB_HOST = "100.91.190.107"
SECRET_ENV_CANDIDATES = (
    Path.home() / ".secrets" / "probata.env",
    Path.home() / ".secrets" / "Agno-MCP-Platform.env",
)
EXPECTED_MANIFEST_SCHEMA = "fct-platform-bundle/v1"
SCRIPT_VERSION = "2026-09-07.1"

# Every table name the plugin side has committed to writing (per the owner's
# task spec). Anything found in the bundle NOT in this set is reported as
# "unexpected" rather than silently ignored.
KNOWN_BUNDLE_TABLES = (
    "person",
    "child",
    "order",
    "hearing",
    "deadline",
    "event",
    "message",
    "exhibit",
    "factor",
    "source",
    "note",
    "court",
    "court_event",
    "filing",
    "draft",
    "memo",
    "reference",
    "evidence_log",
    "eval",
    "case_status",
)

# Bundle tables this importer maps onto an existing context-layer table.
# Everything else in KNOWN_BUNDLE_TABLES is unmapped by design (see the
# mapping doc for the reasoning per table) — this importer never invents a
# schema change to make room for them.
MAPPED_BUNDLE_TABLES = {
    "person",
    "child",
    "court",
    "event",
    "note",
    "order",
    "hearing",
    "deadline",
    "court_event",
    "filing",
    "source",
}

TIMELINE_EVENT_TYPE = {
    "order": "court_order",
    "hearing": "hearing",
    "deadline": "deadline",
    "filing": "filing",
}


class ValidationError(ValueError):
    """A single record failed validation and cannot be mapped."""


@dataclass
class PlannedInsert:
    """One row this script would insert, fully bound and ready to execute."""

    bundle_table: str
    record_id: str
    target_table: str
    sql: str
    params: tuple[Any, ...]


@dataclass
class TableReport:
    bundle_table: str
    target_table: str | None
    total_rows: int = 0
    planned: int = 0
    validation_errors: list[tuple[str, str]] = field(default_factory=list)

    @property
    def unmapped(self) -> bool:
        return self.target_table is None


# ---------------------------------------------------------------------------
# credential resolution — never `source`, never print a value
# ---------------------------------------------------------------------------


def parse_env_file(path: Path) -> dict[str, str]:
    """Tolerant ``KEY=value`` parser. Never executes the file's contents."""
    pattern = re.compile(r"^\s*([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.+?)\s*$")
    out: dict[str, str] = {}
    if not path.is_file():
        return out
    for line in path.read_text(encoding="utf-8", errors="replace").splitlines():
        match = pattern.match(line)
        if not match:
            continue
        key, value = match.group(1), match.group(2)
        if value.startswith(("'", '"')) and value.endswith(value[0]) and len(value) >= 2:
            value = value[1:-1]
        out[key] = value
    return out


def resolve_db_settings() -> dict[str, str]:
    """Resolve connection settings: process env wins, else a secrets file."""
    file_env: dict[str, str] = {}
    for candidate in SECRET_ENV_CANDIDATES:
        file_env = parse_env_file(candidate)
        if file_env:
            break
    host = os.environ.get("DB_HOST") or TAILNET_DB_HOST
    port = os.environ.get("DB_PORT") or file_env.get("DB_PORT") or "5432"
    # NOTE: the shared secrets file's own DB_DATABASE points at the legacy
    # `ai` database (verified live 2026-09-07: connecting with the file's
    # value 500s "database ai does not exist" against a host that HAS
    # dropped it). This importer's tables (working.*/context.*/timeline.*)
    # live in `platform`, the platform's actual current database (D-142) — so
    # the file's value is deliberately NOT consulted for dbname; only an
    # explicit DB_DATABASE env override beats the "platform" default.
    dbname = os.environ.get("DB_DATABASE") or "platform"
    user = os.environ.get("DB_USER") or file_env.get("DB_USER") or file_env.get("POSTGRES_USER") or ""
    password = os.environ.get("DB_PASS") or file_env.get("DB_PASS") or file_env.get("POSTGRES_PASSWORD") or ""
    return {"host": host, "port": str(port), "dbname": dbname, "user": user, "password": password}


def connect(settings: dict[str, str]):  # -> psycopg.Connection
    if psycopg is None:
        raise RuntimeError("psycopg is not importable in this environment")
    dsn = f"host={settings['host']} port={settings['port']} dbname={settings['dbname']} user={settings['user']}"
    return psycopg.connect(dsn, password=settings["password"], connect_timeout=10)


# ---------------------------------------------------------------------------
# small value helpers
# ---------------------------------------------------------------------------


def require(record: dict[str, Any], key: str) -> Any:
    value = record.get(key)
    if value is None or (isinstance(value, str) and not value.strip()):
        raise ValidationError(f"missing required field '{key}'")
    return value


def content_sha256(record: dict[str, Any]) -> bytes:
    """32-byte sha256 of the record's canonical JSON — the dedup key."""
    canonical = json.dumps(record, sort_keys=True, separators=(",", ":"), default=str)
    return hashlib.sha256(canonical.encode("utf-8")).digest()


def confidence_value(raw: Any) -> float | None:
    """Map a bundle confidence (numeric 0..1 or high/medium/low) to a float."""
    if raw is None:
        return None
    if isinstance(raw, (int, float)):
        value = float(raw)
        return value if 0.0 <= value <= 1.0 else None
    if isinstance(raw, str):
        return {"high": 0.9, "medium": 0.6, "low": 0.3}.get(raw.strip().lower())
    return None


def parse_when(raw: Any) -> datetime | None:
    """Parse an ISO-8601 date/datetime; ``None``/``"unknown"``/garbage -> None."""
    if raw is None:
        return None
    if isinstance(raw, (datetime, date)):
        return raw if isinstance(raw, datetime) else datetime(raw.year, raw.month, raw.day)
    if not isinstance(raw, str):
        return None
    text = raw.strip()
    if not text or text.lower() == "unknown":
        return None
    try:
        return datetime.fromisoformat(text.replace("Z", "+00:00"))
    except ValueError:
        try:
            return datetime.fromisoformat(text + "T00:00:00")
        except ValueError:
            return None


def clean_attrs(**fields: Any) -> dict[str, Any]:
    """Drop None/empty values so attrs jsonb stays legible."""
    return {k: v for k, v in fields.items() if v not in (None, "", [], {})}


CHILD_STRIP_KEYS = {"dob", "ssn", "address", "school", "birthdate", "date_of_birth"}


# ---------------------------------------------------------------------------
# per-bundle-table -> target-table mapping
# ---------------------------------------------------------------------------


def _map_person_like(bundle_table: str, record: dict[str, Any], run_id: str) -> PlannedInsert:
    """person / child / court -> working.candidate_entity."""
    record_id = str(require(record, "id"))
    provenance = record.get("source")

    if bundle_table == "court":
        name = record.get("court") or record.get("name") or record.get("title")
        if not name:
            raise ValidationError("court record has no 'court'/'name'/'title' field")
        entity_type = "organization"
        attrs = clean_attrs(
            fct_table=bundle_table,
            entity_subtype="court",
            jurisdiction=record.get("jurisdiction"),
            judge_or_referee=record.get("judge_or_referee") or record.get("judge"),
            provenance=provenance,
        )
    else:
        name = record.get("name")
        if not name:
            raise ValidationError(f"{bundle_table} record has no 'name' field")
        entity_type = "person"
        safe_record = dict(record)
        if bundle_table == "child":
            # Defense in depth: never persist a child's DOB/address/school even
            # if the plugin side sent one by mistake. The bundle's own
            # redaction contract (manifest.redaction.children) is
            # "initials+age" — this importer refuses to widen it.
            for key in CHILD_STRIP_KEYS:
                safe_record.pop(key, None)
        attrs = clean_attrs(
            fct_table=bundle_table,
            role=safe_record.get("role"),
            aliases=safe_record.get("aliases"),
            relationship=safe_record.get("relationship"),
            contact=safe_record.get("contact") if bundle_table != "child" else None,
            age=safe_record.get("age"),
            notes=safe_record.get("notes"),
            provenance=provenance,
        )

    sql = """
        INSERT INTO working.candidate_entity
            (extraction_run_id, source_raw_table, source_raw_id, entity_type,
             name, normalized_name, confidence, attrs, content_sha256)
        VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)
        ON CONFLICT (source_raw_table, source_raw_id, content_sha256) DO NOTHING
        RETURNING id
    """
    params = (
        run_id,
        f"fct_bundle:{bundle_table}",
        record_id,
        entity_type,
        str(name),
        str(name).strip().lower(),
        confidence_value(record.get("confidence")),
        json.dumps(attrs),
        content_sha256(record),
    )
    return PlannedInsert(bundle_table, record_id, "working.candidate_entity", sql, params)


def _map_event(record: dict[str, Any], run_id: str) -> PlannedInsert:
    """event (master-timeline narrative) -> working.candidate_event."""
    record_id = str(require(record, "id"))
    summary = require(record, "description")
    occurred_at = parse_when(record.get("occurred_at"))
    if occurred_at is None:
        raise ValidationError("event.occurred_at is missing/unparseable ('unknown' with no usable date)")
    attrs = clean_attrs(
        known_at=record.get("known_at"),
        location=record.get("location"),
        participants=record.get("participants"),
        evidence_refs=record.get("evidence_refs"),
        factors=record.get("factors"),
        tags=record.get("tags"),
        quote=record.get("quote"),
        source_locator=record.get("source_locator"),
        occurred_text=record.get("occurred_text"),
        provenance=record.get("source"),
    )
    sql = """
        INSERT INTO working.candidate_event
            (extraction_run_id, source_raw_table, source_raw_id, event_type,
             summary, occurred_at, confidence, attrs, content_sha256)
        VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)
        ON CONFLICT (source_raw_table, source_raw_id, content_sha256) DO NOTHING
        RETURNING id
    """
    params = (
        run_id,
        "fct_bundle:event",
        record_id,
        "narrative_event",
        str(summary),
        occurred_at,
        confidence_value(record.get("confidence")),
        json.dumps(attrs),
        content_sha256(record),
    )
    return PlannedInsert("event", record_id, "working.candidate_event", sql, params)


def _map_note(record: dict[str, Any], run_id: str) -> PlannedInsert:
    """note -> working.candidate_fact."""
    record_id = str(require(record, "id"))
    statement = require(record, "text")
    predicate = record.get("kind") or "note"
    attrs = clean_attrs(about=record.get("about"), author=record.get("author"), provenance=record.get("source"))
    sql = """
        INSERT INTO working.candidate_fact
            (extraction_run_id, source_raw_table, source_raw_id, predicate,
             statement, confidence, attrs, content_sha256)
        VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
        ON CONFLICT (source_raw_table, source_raw_id, content_sha256) DO NOTHING
        RETURNING id
    """
    params = (
        run_id,
        "fct_bundle:note",
        record_id,
        str(predicate),
        str(statement),
        confidence_value(record.get("confidence")),
        json.dumps(attrs),
        content_sha256(record),
    )
    return PlannedInsert("note", record_id, "working.candidate_fact", sql, params)


def _map_timeline_event(bundle_table: str, record: dict[str, Any], run_id: str) -> PlannedInsert:
    """order / hearing / deadline / court_event / filing -> timeline.event_candidate."""
    record_id = str(require(record, "id"))
    display_summary = record.get("title") or record.get("description")
    if not display_summary:
        raise ValidationError(f"{bundle_table} record has no 'title'/'description' field")

    # NOTE: 'filed_or_planned' on a filing record is a STATUS enum
    # (filed|planned|draft-only), never a date — only 'date'/'occurred_at' are
    # ever treated as the timeline date, for every bundle_table here.
    when_raw = record.get("date") or record.get("occurred_at")
    occurred_at = parse_when(when_raw) if isinstance(when_raw, str) else None
    temporal_precision = "point" if occurred_at is not None else "uncertain"

    if bundle_table == "court_event":
        event_type = record.get("kind") or "court_event"
    else:
        event_type = TIMELINE_EVENT_TYPE[bundle_table]

    consumed = {"id", "title", "description", "date", "occurred_at", "kind", "source"}
    extra = {k: v for k, v in record.items() if k not in consumed}
    source_locator = clean_attrs(provenance=record.get("source"), extra=extra, raw_when=when_raw)
    version = content_sha256(record).hex()

    sql = """
        INSERT INTO timeline.event_candidate
            (source_system, source_record_id, source_record_version, source_locator,
             extraction_run_id, temporal_precision, occurred_at, display_summary, event_type)
        VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)
        ON CONFLICT (source_system, source_record_id, source_record_version) DO NOTHING
        RETURNING id
    """
    params = (
        "fct_bundle",
        record_id,
        version,
        json.dumps(source_locator),
        run_id,
        temporal_precision,
        occurred_at,
        str(display_summary),
        str(event_type),
    )
    return PlannedInsert(bundle_table, record_id, "timeline.event_candidate", sql, params)


PROVENANCE_CLASS_BY_AUTHOR = {
    "owner": "first_party_authored",
    "court": "acquired_third_party",
    "third-party": "acquired_third_party",
}


def _map_source(record: dict[str, Any], run_id: str) -> PlannedInsert:  # noqa: ARG001 - run_id unused here on purpose
    """source (intake-file registration) -> context.source. No extraction_run FK."""
    record_id = str(require(record, "id"))
    source_key = record.get("path") or record.get("source_key") or record_id
    authored_by = str(record.get("authored_by", "")).strip().lower()
    if authored_by.startswith("ai:"):
        provenance_class = "system_generated"
    else:
        provenance_class = PROVENANCE_CLASS_BY_AUTHOR.get(authored_by, "unknown")
    sql = """
        INSERT INTO context.source (source_key, provenance_class)
        VALUES (%s, %s)
        ON CONFLICT (source_key) DO NOTHING
        RETURNING id
    """
    params = (str(source_key), provenance_class)
    return PlannedInsert("source", record_id, "context.source", sql, params)


def map_record(bundle_table: str, record: dict[str, Any], run_id: str) -> PlannedInsert:
    """Dispatch one bundle record to its target-table builder.

    Raises ``ValidationError`` for a record this importer cannot map even
    though its table is in ``MAPPED_BUNDLE_TABLES`` (missing required field,
    unparseable date, etc). Callers must only call this for
    ``bundle_table in MAPPED_BUNDLE_TABLES``.
    """
    if bundle_table in ("person", "child", "court"):
        return _map_person_like(bundle_table, record, run_id)
    if bundle_table == "event":
        return _map_event(record, run_id)
    if bundle_table == "note":
        return _map_note(record, run_id)
    if bundle_table in TIMELINE_EVENT_TYPE or bundle_table == "court_event":
        return _map_timeline_event(bundle_table, record, run_id)
    if bundle_table == "source":
        return _map_source(record, run_id)
    raise AssertionError(f"map_record called for unmapped table {bundle_table!r}")  # pragma: no cover


# ---------------------------------------------------------------------------
# bundle reading
# ---------------------------------------------------------------------------


def read_ndjson(path: Path) -> list[dict[str, Any]]:
    records: list[dict[str, Any]] = []
    with path.open("r", encoding="utf-8") as handle:
        for line_no, line in enumerate(handle, start=1):
            line = line.strip()
            if not line:
                continue
            try:
                obj = json.loads(line)
            except json.JSONDecodeError as exc:
                raise ValidationError(f"{path.name}:{line_no}: invalid JSON ({exc})") from exc
            if not isinstance(obj, dict):
                raise ValidationError(f"{path.name}:{line_no}: expected a JSON object")
            records.append(obj)
    return records


def read_manifest(bundle_dir: Path) -> dict[str, Any] | None:
    manifest_path = bundle_dir / "manifest.json"
    if not manifest_path.is_file():
        return None
    return json.loads(manifest_path.read_text(encoding="utf-8"))


# ---------------------------------------------------------------------------
# plan building
# ---------------------------------------------------------------------------


def build_plans(bundle_dir: Path, run_id: str) -> tuple[list[PlannedInsert], dict[str, TableReport], int]:
    """Read every ndjson file and build the full set of planned inserts.

    Returns ``(plans, per_table_reports, edge_row_count)``.
    """
    plans: list[PlannedInsert] = []
    reports: dict[str, TableReport] = {}

    present_files = {p.stem: p for p in bundle_dir.glob("*.ndjson")}

    for table in KNOWN_BUNDLE_TABLES:
        target = "mapped" if table in MAPPED_BUNDLE_TABLES else None
        report = TableReport(bundle_table=table, target_table="pending" if target else None)
        path = present_files.pop(table, None)
        if path is None:
            reports[table] = report
            continue
        try:
            records = read_ndjson(path)
        except ValidationError as exc:
            report.validation_errors.append(("<file>", str(exc)))
            reports[table] = report
            continue
        report.total_rows = len(records)
        for record in records:
            record_id = str(record.get("id", "<no-id>"))
            if table not in MAPPED_BUNDLE_TABLES:
                continue
            try:
                planned = map_record(table, record, run_id)
            except ValidationError as exc:
                report.validation_errors.append((record_id, str(exc)))
                continue
            report.target_table = planned.target_table
            report.planned += 1
            plans.append(planned)
        reports[table] = report

    # Unexpected ndjson files (not in KNOWN_BUNDLE_TABLES).
    for stem, path in present_files.items():
        if stem == "edges":
            continue
        report = TableReport(bundle_table=stem, target_table=None)
        try:
            report.total_rows = len(read_ndjson(path))
        except ValidationError as exc:
            report.validation_errors.append(("<file>", str(exc)))
        report.validation_errors.append(("<table>", "unexpected table name — not in the known bundle-table set"))
        reports[stem] = report

    edge_path = bundle_dir / "edges.ndjson"
    edge_count = len(read_ndjson(edge_path)) if edge_path.is_file() else 0

    return plans, reports, edge_count


# ---------------------------------------------------------------------------
# execution
# ---------------------------------------------------------------------------


def create_extraction_run(cursor, bundle_dir: Path, plan_count: int) -> str:
    cursor.execute(
        """
        INSERT INTO working.extraction_run
            (extractor, extractor_version, source_summary, status, finished_at, stats)
        VALUES (%s, %s, %s, 'completed', now(), %s)
        RETURNING id
        """,
        (
            "fct_bundle_importer",
            SCRIPT_VERSION,
            f"FCT bundle import from {bundle_dir}",
            json.dumps({"planned_inserts": plan_count}),
        ),
    )
    row = cursor.fetchone()
    return str(row[0])


def execute_plans(cursor, plans: list[PlannedInsert]) -> dict[str, dict[str, int]]:
    """Execute every planned insert; return per-target inserted/conflicted counts."""
    counts: dict[str, dict[str, int]] = {}
    for plan in plans:
        bucket = counts.setdefault(plan.target_table, {"inserted": 0, "conflicted": 0})
        cursor.execute(plan.sql, plan.params)
        if cursor.fetchone() is not None:
            bucket["inserted"] += 1
        else:
            bucket["conflicted"] += 1
    return counts


# ---------------------------------------------------------------------------
# reporting
# ---------------------------------------------------------------------------


def print_report(
    bundle_dir: Path,
    manifest: dict[str, Any] | None,
    reports: dict[str, TableReport],
    edge_count: int,
    db_counts: dict[str, dict[str, int]] | None,
    db_note: str,
    mode_label: str,
) -> None:
    print("=" * 78)
    print(f"FCT bundle import — {mode_label}")
    print(f"bundle: {bundle_dir}")
    if manifest is None:
        print("manifest.json: MISSING")
    else:
        schema = manifest.get("schema")
        note = "" if schema == EXPECTED_MANIFEST_SCHEMA else f"  (expected {EXPECTED_MANIFEST_SCHEMA!r})"
        print(f"manifest.json: schema={schema!r}{note}, exported_at={manifest.get('exported_at')!r}")
    print(f"database: {db_note}")
    print("-" * 78)
    print(f"{'table':<14} {'target':<28} {'rows':>6} {'planned':>8} {'errors':>7}")
    for table in KNOWN_BUNDLE_TABLES:
        report = reports.get(table)
        if report is None:
            print(f"{table:<14} {'(no file)':<28} {0:>6} {0:>8} {0:>7}")
            continue
        target = report.target_table if not report.unmapped else "UNMAPPED"
        print(f"{table:<14} {target:<28} {report.total_rows:>6} {report.planned:>8} {len(report.validation_errors):>7}")

    extras = [t for t in reports if t not in KNOWN_BUNDLE_TABLES]
    for table in extras:
        report = reports[table]
        print(f"{table:<14} {'UNEXPECTED TABLE':<28} {report.total_rows:>6} {0:>8} {len(report.validation_errors):>7}")

    print(f"{'edges':<14} {'UNMAPPED (out of scope)':<28} {edge_count:>6} {0:>8} {0:>7}")
    print("-" * 78)

    if db_counts is not None:
        print("insert-vs-already-present (from the database round trip):")
        for target, bucket in db_counts.items():
            print(f"  {target:<28} inserted={bucket['inserted']:>5}  already-present={bucket['conflicted']:>5}")
        print("-" * 78)

    any_errors = False
    for table, report in reports.items():
        if report.validation_errors:
            any_errors = True
            print(f"validation errors — {table}:")
            for record_id, message in report.validation_errors[:20]:
                print(f"  [{record_id}] {message}")
            if len(report.validation_errors) > 20:
                print(f"  ... and {len(report.validation_errors) - 20} more")
    if not any_errors:
        print("validation errors: none")
    print("=" * 78)


# ---------------------------------------------------------------------------
# CLI
# ---------------------------------------------------------------------------


def build_arg_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--bundle", required=True, type=Path, help="path to the FCT export bundle directory")
    mode = parser.add_mutually_exclusive_group()
    mode.add_argument("--dry-run", action="store_true", help="explicit dry-run (this is the default with no --apply)")
    mode.add_argument("--apply", action="store_true", help="execute inserts inside a transaction")
    parser.add_argument(
        "--commit",
        action="store_true",
        help="only meaningful with --apply: commit instead of rolling back after printing counts",
    )
    return parser


def main(argv: list[str] | None = None) -> int:
    args = build_arg_parser().parse_args(argv)
    bundle_dir: Path = args.bundle
    if not bundle_dir.is_dir():
        print(f"ERROR: bundle directory not found: {bundle_dir}", file=sys.stderr)
        return 2

    manifest = read_manifest(bundle_dir)
    run_id_placeholder = "00000000-0000-0000-0000-000000000000"
    plans, reports, edge_count = build_plans(bundle_dir, run_id_placeholder)

    apply_mode = bool(args.apply)
    if not apply_mode:
        # Dry-run: best-effort DB round trip, ALWAYS rolled back, never fatal.
        try:
            settings = resolve_db_settings()
            with connect(settings) as conn:
                conn.autocommit = False
                with conn.cursor() as cursor:
                    run_id = create_extraction_run(cursor, bundle_dir, len(plans))
                    rebound = [
                        PlannedInsert(p.bundle_table, p.record_id, p.target_table, p.sql, _rebind(p, run_id))
                        for p in plans
                    ]
                    db_counts = execute_plans(cursor, rebound)
                conn.rollback()
            print_report(
                bundle_dir, manifest, reports, edge_count, db_counts, f"reachable ({settings['host']})", "DRY RUN"
            )
        except Exception as exc:  # noqa: BLE001 - dry-run must degrade, never crash
            print_report(bundle_dir, manifest, reports, edge_count, None, f"unreachable ({exc})", "DRY RUN")
        return 0

    # --apply
    try:
        settings = resolve_db_settings()
        conn = connect(settings)
    except Exception as exc:  # noqa: BLE001
        print(f"ERROR: --apply requires a reachable database: {exc}", file=sys.stderr)
        return 2

    try:
        conn.autocommit = False
        with conn.cursor() as cursor:
            run_id = create_extraction_run(cursor, bundle_dir, len(plans))
            rebound = [
                PlannedInsert(p.bundle_table, p.record_id, p.target_table, p.sql, _rebind(p, run_id)) for p in plans
            ]
            db_counts = execute_plans(cursor, rebound)
        if args.commit:
            conn.commit()
            mode_label = "APPLY - COMMITTED"
        else:
            conn.rollback()
            mode_label = "APPLY - ROLLBACK MODE (pass --commit to persist)"
        print_report(
            bundle_dir, manifest, reports, edge_count, db_counts, f"reachable ({settings['host']})", mode_label
        )
    finally:
        conn.close()
    return 0


def _rebind(plan: PlannedInsert, run_id: str) -> tuple[Any, ...]:
    """Swap the placeholder extraction_run_id in a planned insert's params.

    ``timeline.event_candidate``/``context.source`` params don't carry the
    working.extraction_run FK placeholder at the same position as the
    candidate_* tables, but they DO carry the placeholder string itself
    (event_candidate.extraction_run_id is a free-text column; context.source
    has no such column at all) — a plain value substitution is safe because
    the placeholder UUID string never legitimately appears in bundle data.
    """
    placeholder = "00000000-0000-0000-0000-000000000000"
    return tuple(run_id if value == placeholder else value for value in plan.params)


if __name__ == "__main__":
    raise SystemExit(main())

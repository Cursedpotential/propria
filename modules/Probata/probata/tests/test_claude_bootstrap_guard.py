"""Exercise the snapshot's Claude guard expressions with synthetic, read-only SQL.

Input: snapshot SQL and PROPRIA_TEST_PSQL_JSON (a JSON argv list for remote psql).
Output: predicate assertions. Side effects: read-only queries, no schema/data writes.
Use for bootstrap guard changes; this does not exercise the complete trigger.
"""

import json
import os
import re
import subprocess
from pathlib import Path

import pytest


def _literal(value):
    """Encode a synthetic scalar as SQL without executing it or changing state."""
    return "NULL" if value is None else "'" + str(value).replace("'", "''") + "'"


def _cte(name, values, casts=None, empty=False):
    """Build one synthetic CTE row with optional SQL types and no database writes."""
    casts = casts or {}
    fields = [f"{_literal(value)}{casts.get(key, '')} AS {key}" for key, value in values.items()]
    return f"{name} AS (SELECT {', '.join(fields)}{' WHERE false' if empty else ''})"


def test_claude_bootstrap_guard_expressions():
    """Verify exact snapshot predicates accept valid proofs and reject broken bindings.

    Inputs: synthetic selection/row fixtures and configured psql argv. Output: assertions.
    Side effects: one read-only transaction, rolled back. Prefer over textual SQL checks.
    """
    command = os.environ.get("PROPRIA_TEST_PSQL_JSON")
    if not command:
        pytest.skip("set PROPRIA_TEST_PSQL_JSON to a remote psql argv list")
    snapshot = (Path(__file__).resolve().parents[1] / "sql/bootstrap/schema_snapshot_20260907.sql").read_text()
    guard = snapshot.split("CREATE FUNCTION context.guard_raw_generation_transition()", 1)[1]
    guard = guard.split("CREATE FUNCTION context.register_raw_format_subtype", 1)[0]
    case = guard.split("verification.observed->>'duckdb_template'=", 1)[1].split("    LIMIT 1;", 1)[0].strip()
    case = case.replace("NEW.", "candidate.").replace("context.activity_receipt", "fixture_receipt")
    case = case.replace("context.activity_execution", "fixture_execution")
    row_match = re.search(
        r"EXECUTE format\('SELECT (count\(\*\)>0 AND bool_and\(.*?)\s+FROM context\.raw_record_identity",
        guard,
        re.DOTALL,
    )
    assert row_match, "snapshot row proof expression missing"
    row_proof = row_match[1].replace("''", "'").replace("$2", "params.template").replace("$3", "params.declared")
    assert "USING NEW.id, context_template, NEW.format_id;" in guard
    template = "claude_ai_export_json_v1"
    queries = ["BEGIN READ ONLY;", "SET LOCAL statement_timeout = '10s';"]
    names = []

    pin_cases = [
        ("valid pin", {}, True),
        ("missing pin", {"missing": True}, False),
        ("failed receipt", {"status": "failed"}, False),
        ("late receipt", {"completed_at": "2026-10-16"}, False),
        ("wrong source", {"source_version_id": "other"}, False),
        ("wrong workflow", {"workflow_id": "other"}, False),
        ("wrong activity", {"activity_name": "other"}, False),
        ("wrong parser", {"parser_id": "other"}, False),
        ("wrong version", {"parser_version": "other"}, False),
        ("wrong declaration", {"declared_format": "other"}, False),
        ("wrong template", {"duckdb_template": "other"}, False),
        ("disallowed format", {"format_id": "smsbackuprestore_xml", "declared_format": "smsbackuprestore_xml"}, False),
    ]
    for name, changes, allowed in pin_cases:
        candidate = {
            "source_version_id": "source",
            "parser_id": "duckdb_structured_elt",
            "parser_version": "1.0.0",
            "format_id": changes.get("format_id", "claude_ai_export_json"),
            "created_at": "2026-10-15",
        }
        pin = {
            "parser_id": "duckdb_structured_elt",
            "parser_version": "1.0.0",
            "declared_format": "claude_ai_export_json",
            "duckdb_template": template,
        }
        pin.update({key: value for key, value in changes.items() if key in pin})
        execution = {
            "id": "execution",
            "source_version_id": "source",
            "workflow_id": "workflow",
            "activity_name": "select_parser_activity",
        }
        execution.update({key: value for key, value in changes.items() if key in execution})
        receipt = {
            "activity_execution_id": "execution",
            "status": changes.get("status", "success"),
            "completed_at": changes.get("completed_at", "2026-10-14"),
            "result_ref": json.dumps(pin),
        }
        ctes = [
            _cte("candidate", candidate, {"created_at": "::timestamptz"}),
            _cte("source", {"workflow_id": "workflow"}),
            _cte("detected", {"format_id": "claude_ai_export_json"}),
            _cte("fixture_execution", execution),
            _cte(
                "fixture_receipt",
                receipt,
                {"completed_at": "::timestamptz", "result_ref": "::jsonb"},
                changes.get("missing", False),
            ),
        ]
        expected = _literal(template if allowed else None)
        queries.append(
            f"WITH {', '.join(ctes)} SELECT {_literal(name)}, ({case}) IS NOT DISTINCT FROM {expected} FROM candidate CROSS JOIN source CROSS JOIN detected;"
        )
        names.append(name)

    row_cases = [("parsed", {}, True)]
    row_cases += [
        (status, {"record_status": status}, True)
        for status in ("envelope", "unknown", "malformed", "rejected", "unparsed")
    ]
    row_cases += [
        ("blank reason", {"record_status": "envelope", "status_reason": " "}, False),
        ("null reason", {"record_status": "envelope", "status_reason": None}, False),
        ("bad status", {"record_status": "other"}, False),
        ("wrong row template", {"row_template": "other"}, False),
        ("wrong proof template", {"record_status": "envelope", "template": "other"}, False),
        ("human declaration", {"record_status": "envelope", "declared": "smsbackuprestore_xml"}, False),
        ("absent bytes", {"stored_bytes": None}, False),
        ("source locator", {"locator_object_id": "locator"}, False),
        ("empty rows", {"empty": True}, False),
        ("mixed rows", {"mixed": True}, False),
    ]
    for name, changes, allowed in row_cases:
        raw = {
            "stored_bytes": "bytes",
            "locator_object_id": None,
            "record_status": "parsed",
            "status_reason": "retained envelope",
        }
        raw.update({key: value for key, value in changes.items() if key in raw})
        raw_cte = _cte("raw", raw, empty=changes.get("empty", False))
        if changes.get("mixed"):
            raw_cte = raw_cte[:-1] + " UNION ALL SELECT NULL, NULL, 'parsed', 'missing bytes')"
        ctes = [
            raw_cte,
            _cte(
                "params",
                {
                    "template": changes.get("template", template),
                    "declared": changes.get("declared", "claude_ai_export_json"),
                },
            ),
            _cte(
                "subtype",
                {"native_metadata": json.dumps({"duckdb_template": changes.get("row_template", template)})},
                {"native_metadata": "::jsonb"},
            ),
        ]
        queries.append(
            f"WITH {', '.join(ctes)} SELECT {_literal(name)}, ({row_proof}) IS NOT DISTINCT FROM {'true' if allowed else 'false'} FROM raw CROSS JOIN subtype CROSS JOIN params;"
        )
        names.append(name)
    queries.append("ROLLBACK;")
    result = subprocess.run(
        json.loads(command) + ["-X", "-qAt", "-F", "=", "-v", "ON_ERROR_STOP=1"],
        input="\n".join(queries),
        text=True,
        capture_output=True,
        timeout=90,
        check=False,
    )
    assert result.returncode == 0, result.stderr
    assert result.stdout.splitlines() == [name + "=t" for name in names], result.stdout

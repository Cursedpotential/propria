"""Exercise new read-only attribution SQL on an explicitly disposable database.

Inputs: PROPRIA_OPERATOR_REPAIR_TEST_DSN for the exact dedicated database prefix.
Outputs: real PostgreSQL count/readback assertions. Effects: synthetic transaction
only, rolled back; never accepts a production database or a default connection.
"""
import os
from urllib.parse import urlparse
import pytest


def test_real_postgres_attribution_impact_and_overlap(monkeypatch):
    """Count selected identifier scope without mutating unrelated synthetic rows."""
    dsn = os.environ.get("PROPRIA_OPERATOR_REPAIR_TEST_DSN", "")
    if not dsn:
        pytest.skip("Dedicated disposable operator-results PostgreSQL DSN is required")
    assert urlparse(dsn).path.lstrip("/").startswith("backend_package_review_operator_results_20261015")
    import psycopg
    from psycopg.rows import dict_row
    from app.repo import imported_pg, imported_sources_pg, case_identity_catalog
    with psycopg.connect(dsn, row_factory=dict_row) as connection:
        with connection.transaction(force_rollback=True):
            connection.execute("""CREATE SCHEMA context; CREATE SCHEMA working; CREATE SCHEMA raw_duck;
              CREATE TABLE context.source(id uuid,source_key text);
              CREATE TABLE context.source_version(id uuid,source_id uuid,matter_id uuid,acquired_at timestamptz,version_ordinal int);
              CREATE TABLE context.activity_execution(id uuid,source_version_id uuid,activity_name text);
              CREATE TABLE context.activity_receipt(activity_execution_id uuid,status text);
              CREATE TABLE context.normalized_record_identity(id uuid,source_version_id uuid,normalized_payload jsonb);
              CREATE TABLE raw_duck.comm_events_20260918(dedup_key text,sender text,counterparty_phone text);
              CREATE FUNCTION raw_duck.norm_phone(text) RETURNS text LANGUAGE sql IMMUTABLE AS 'SELECT right(regexp_replace($1,''[^0-9]'','''',''g''),10)';
              INSERT INTO context.source VALUES('00000000-0000-0000-0000-000000000001','fixture.xml');
              INSERT INTO context.source_version VALUES('00000000-0000-0000-0000-000000000002','00000000-0000-0000-0000-000000000001','01a0f751-e07b-75cc-9ad5-63ad9449a8ba',now(),1);
              INSERT INTO context.normalized_record_identity VALUES
                ('00000000-0000-0000-0000-000000000003','00000000-0000-0000-0000-000000000002','{"participants":[{"role":"sender","identifier":"+15555550101"}]}'),
                ('00000000-0000-0000-0000-000000000004','00000000-0000-0000-0000-000000000002','{"participants":[{"role":"sender","identifier":"+15555550102"}]}');
              INSERT INTO raw_duck.comm_events_20260918 VALUES('a','+15555550101','+15555550101'),('b','+15555550102','+15555550101'),('c','+15555550102','+15555550102');""")
            def query(sql, params):
                """Execute the owned read SQL on the synthetic transaction only."""
                return connection.execute(sql, params).fetchall()
            monkeypatch.setattr(imported_pg, "_query", query)
            monkeypatch.setattr(case_identity_catalog.intake_discovery, "_query", query)
            before = connection.execute("SELECT id, normalized_payload::text FROM context.normalized_record_identity ORDER BY id").fetchall()
            impact = imported_sources_pg.attribution_impact("01a0f751-e07b-75cc-9ad5-63ad9449a8ba", "5555550101", "phone")
            assert impact == {"available": True, "records": 1, "source_versions": 1, "conversations": 1}
            overlap = case_identity_catalog.overlap("5555550101")
            assert overlap["unique_records"] == 2 and overlap["overlap"] == 1
            assert overlap["sender"] == 1 and overlap["counterparty"] == 2
            assert before == connection.execute("SELECT id, normalized_payload::text FROM context.normalized_record_identity ORDER BY id").fetchall()


def test_real_postgres_preflight_text_size_sql():
    """Exercise the production preflight SQL against long Unicode fixture messages."""
    dsn = os.environ.get("PROPRIA_OPERATOR_REPAIR_TEST_DSN", "")
    if not dsn:
        pytest.skip("Dedicated disposable operator-results PostgreSQL DSN is required")
    assert urlparse(dsn).path.lstrip("/").startswith("backend_package_review_operator_results_20261015")
    import re
    import json
    from pathlib import Path
    import psycopg
    # Execute the actual owned Go query, rather than a retyped SQL assertion.
    source = (Path(__file__).resolve().parents[3] / "engine/postgres/conversation_preflight.go").read_text()
    sql = re.search(r"QueryRow\(ctx,\s*`(SELECT count\(\*\).*?)`", source, re.S).group(1)
    sql = sql.replace("$1::uuid", "%s::uuid")
    generation = "00000000-0000-0000-0000-000000000005"
    bodies = ["short", "é🙂", "Long pasted document " * 1000, "", "final"]
    with psycopg.connect(dsn) as connection:
        with connection.transaction(force_rollback=True):
            connection.execute("CREATE SCHEMA context; CREATE TABLE context.normalized_record_identity(normalized_generation_id uuid,record_type text,normalized_payload jsonb)")
            for body in bodies:
                connection.execute("INSERT INTO context.normalized_record_identity VALUES(%s,'message',%s::jsonb)", (generation, json.dumps({"content": {"body": body}})))
            result = connection.execute(sql, (generation,)).fetchone()
            assert result == (5, sum(map(len, bodies)), sum(len(b.encode("utf-8")) for b in bodies), max(map(len, bodies)), max(len(b.encode("utf-8")) for b in bodies))

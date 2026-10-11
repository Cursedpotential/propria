"""Validate restricted occurrence functions against retained synthetic VPS fixtures.

Inputs: reviewed migration path, live PostgreSQL container and retained proof directory.
Outputs: isolated schema/NOLOGIN role identities, exact substitution diff and proof JSON.
Effects: creates only named synthetic schemas, role and insert-only fixtures; never removes them.
Choose before parent-owned admission; uses no owner notes and never applies production grants.
Byline: Codex · GPT-6.1 · 2026-10-05.
"""
import argparse
import concurrent.futures
import copy
import datetime
import difflib
import hashlib
import json
import pathlib
import subprocess
import uuid


def main():
    """Execute immutable admission, collision, concurrency and restricted-read proofs.

    Inputs: explicit server paths/container. Outputs: independently queried proof JSON.
    Effects: retained synthetic database fixtures only; choose instead of production test rows.
    """
    p = argparse.ArgumentParser()
    p.add_argument("--migration", required=True)
    p.add_argument("--database-container", required=True)
    p.add_argument("--proof-directory", required=True)
    args = p.parse_args()
    proof = pathlib.Path(args.proof_directory)
    proof.mkdir(parents=True, exist_ok=True)
    suffix = datetime.datetime.now(datetime.timezone.utc).strftime("%Y%m%d_%H%M%S") + "_" + uuid.uuid4().hex[:8]
    raw_schema, api_schema, role = [s + suffix for s in ("ai_fixture_", "ai_api_", "ai_role_")]
    table = raw_schema + ".source_occurrences"
    source = pathlib.Path(args.migration).read_text(encoding="utf-8")
    variant = source.replace("raw_duck.source_occurrences", table).replace("source_occurrence_api", api_schema).replace("casebible_toolkit_recovery_service", role)
    (proof / "fixture-migration.sql").write_text(variant, encoding="utf-8")
    diff = "".join(difflib.unified_diff(source.splitlines(True), variant.splitlines(True), fromfile="production-reviewed.sql", tofile="fixture-schema-role-only.sql"))
    (proof / "fixture-substitutions.diff").write_text(diff, encoding="utf-8")
    checks = []

    def sql(query, expected_error=False):
        """Run one explicit synthetic SQL statement through the server-local DBA socket.

        Inputs: fixture-only SQL and expected failure flag. Outputs: bounded text.
        Effects: declared fixture operations only; choose without exporting database credentials.
        """
        result = subprocess.run(["docker", "exec", "-i", "-u", "postgres", args.database_container, "psql", "-X", "-A", "-t", "-v", "ON_ERROR_STOP=1", "-U", "postgres", "-d", "casebible"], input=query, text=True, capture_output=True)
        if expected_error:
            if result.returncode == 0:
                raise AssertionError("expected restricted fixture operation to fail")
            return "rejected"
        if result.returncode:
            raise RuntimeError(result.stderr[:4000])
        return result.stdout.strip()

    sql(f"CREATE SCHEMA {raw_schema}; CREATE ROLE {role} NOLOGIN; CREATE TABLE {table} (LIKE raw_duck.source_occurrences INCLUDING ALL);")
    sql(variant)

    def batch(op, names):
        """Construct complete synthetic occurrence metadata without owner content or fake provider identities.

        Inputs: test operation and names. Outputs: exact fixed-scope payload. Effects: none.
        """
        pin = "a" * 64
        b = {"schema": "ai-workproduct-source-occurrences/v1", "request": {"operation": op, "readback_ref": "file:///synthetic/readback.json", "readback_sha256": pin, "metadata_ref": "file:///synthetic/metadata.json"}, "rows": []}
        for name in names:
            key = "consignatio/casevault/KnowledgeBase/ai-chats/_derived/misc/synthetic-fixture/" + name
            import urllib.parse
            m = dict(schema=b["schema"], operation=op, original_path="F:/Users/matts/Downloads/" + name,
                     observed_mtime_ns=1759681000123456789, transport_observed_at="2026-10-05T12:00:00Z",
                     original_unchanged_size_mtime=True, event_date_status="unknown", provider="unknown", source_unit="synthetic-fixture",
                     source_ref="file:///synthetic/incoming/source-01", transport_server_path="/synthetic/incoming/source-01",
                     provenance_ref="file:///synthetic/transport.json", provenance_sha256="b" * 64,
                     placement_manifest_ref="file:///synthetic/manifest.json", placement_manifest_sha256="c" * 64,
                     readback_ref=b["request"]["readback_ref"], readback_sha256=pin,
                     object_ref="b2://salem-data/" + urllib.parse.quote(key, safe="/") + "?" + urllib.parse.urlencode({"versionId": "fixture+ v1/é="}), version_id="fixture+ v1/é=", sha256="d" * 64)
            b["rows"].append(dict(source="local/F-Downloads", scope="F:/Users/matts/Downloads", path=name, source_id="", size=100, disposition="copied", b2_key=key, metadata=m))
        return b

    def admission(b, expected_error=False, override_pin=None):
        """Call the admitted fixture function with an exact complete payload pin.

        Inputs: synthetic batch. Outputs: count or rejection. Effects: insert-only fixture transaction.
        """
        payload = json.dumps(b, ensure_ascii=False, separators=(",", ":"))
        pin = hashlib.sha256(payload.encode()).hexdigest()
        query = f"SET ROLE {role}; SELECT {api_schema}.register_ai_workproduct($payload${payload}$payload$,'{override_pin or pin}');"
        return sql(query, expected_error), pin

    b = batch("synthetic-first", ["Synthetic résumé + — one.md", "Synthetic two.md"])
    _, pin = admission(b)
    before = sql(f"SELECT jsonb_agg(to_jsonb(s) ORDER BY path) FROM {table} s;")
    admission(b)
    assert before == sql(f"SELECT jsonb_agg(to_jsonb(s) ORDER BY path) FROM {table} s;")
    checks.append("identical replay preserves complete rows including recorded_at")
    rows = json.loads(sql(f"SET ROLE {role}; SELECT {api_schema}.read_ai_workproduct('synthetic-first','{pin}');").splitlines()[-1])
    expected = copy.deepcopy(b["rows"])
    for r in expected:
        r["metadata"]["catalog_batch_sha256"] = pin
    assert sorted(rows, key=lambda r: r["path"]) == sorted(expected, key=lambda r: r["path"])
    checks.append("independent readback exact Unicode source identities, empty IDs and nanoseconds")
    for name, mutate in [
        ("different existing hash", lambda x: x["rows"][0]["metadata"].update(sha256="e" * 64)),
        ("different existing version", lambda x: x["rows"][0]["metadata"].update(version_id="fixture-v2", object_ref=x["rows"][0]["metadata"]["object_ref"].split("?",1)[0]+"?versionId=fixture-v2")),
        ("different existing root provenance", lambda x: [r["metadata"].update(provenance_sha256="e" * 64) for r in x["rows"]]),
        ("nonempty invented source ID", lambda x: x["rows"][0].update(source_id="fabricated")),
        ("wrong source scope", lambda x: x["rows"][0].update(scope="elsewhere")),
        ("path escape", lambda x: x["rows"][0].update(path="../escape.md")),
        ("wrong destination prefix", lambda x: x["rows"][0].update(b2_key="legal/forbidden.md")),
        ("object/version mismatch", lambda x: x["rows"][0]["metadata"].update(version_id="different")),
        ("inferred original date", lambda x: x["rows"][0]["metadata"].update(event_date_status="inferred")),
        ("provider-native hash smuggling", lambda x: x["rows"][0].update(native_hash="d" * 64)),
        ("root pin absent", lambda x: x["rows"][0]["metadata"].pop("provenance_sha256")),
        ("oversized source", lambda x: x["rows"][0].update(size=1048577)),
        ("wrong batch schema", lambda x: x.update(schema="wrong-schema")),
        ("NULL batch schema", lambda x: x.update(schema=None)),
        ("extra batch fields", lambda x: x.update(unexpected=True)),
        ("extra request fields", lambda x: x["request"].update(unexpected=True)),
        ("extra metadata fields", lambda x: x["rows"][0]["metadata"].update(unexpected=True)),
        ("NULL provider", lambda x: x["rows"][0]["metadata"].update(provider=None)),
        ("NULL content hash", lambda x: x["rows"][0]["metadata"].update(sha256=None)),
        ("NULL provenance root", lambda x: x["rows"][0]["metadata"].update(provenance_ref=None)),
        ("NULL manifest root hash", lambda x: x["rows"][0]["metadata"].update(placement_manifest_sha256=None)),
        ("same operation changed count", lambda x: x["rows"].pop()),
    ]:
        changed = copy.deepcopy(b)
        mutate(changed)
        admission(changed, expected_error=True)
        assert before == sql(f"SELECT jsonb_agg(to_jsonb(s) ORDER BY path) FROM {table} s;")
        checks.append(name + " rejects and preserves rows")
    admission(b, expected_error=True, override_pin="0" * 64)
    admission(batch("oversized-batch", [f"oversized-{i}.md" for i in range(17)]), expected_error=True)
    checks.append("wrong canonical SHA and seventeen-row batch reject")
    late = batch("synthetic-late-conflict", ["Candidate never committed.md", b["rows"][0]["path"]])
    admission(late, expected_error=True)
    assert sql(f"SELECT count(*) FROM {table} WHERE path='Candidate never committed.md';") == "0"
    checks.append("late collision atomically rolls back earlier candidate insert")
    for query in [f"SELECT * FROM {table};", f"INSERT INTO {table}(source,scope,path,source_id,size,disposition) VALUES('x','','x','',1,'x');", f"SELECT {api_schema}.decode_component('x',false);"]:
        sql(f"SET ROLE {role}; {query}", expected_error=True)
    checks.append("NOLOGIN fixture role cannot access raw table or private URI helper")
    assert sql(f"SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.pronamespace='{api_schema}'::regnamespace AND a.grantee=0 AND a.privilege_type='EXECUTE';") == "0"
    checks.append("PUBLIC has no EXECUTE on any API/private helper function")
    aux = batch("synthetic-auxiliary", ["Preexisting observed date.md"])
    aux_row = aux["rows"][0]
    aux_json = json.dumps(aux_row["metadata"], ensure_ascii=False, separators=(",", ":"))
    sql(f"INSERT INTO {table}(source,scope,path,source_id,size,disposition,b2_key,metadata,modtime,native_hash_kind,native_hash) VALUES('local/F-Downloads','F:/Users/matts/Downloads','Preexisting observed date.md','',100,'copied','{aux_row['b2_key']}',$metadata${aux_json}$metadata$::jsonb,'2026-01-01T00:00:00Z','provider-native','retained-existing');")
    aux_before = sql(f"SELECT to_jsonb(s) FROM {table} s WHERE path='Preexisting observed date.md';")
    admission(aux, expected_error=True)
    assert aux_before == sql(f"SELECT to_jsonb(s) FROM {table} s WHERE path='Preexisting observed date.md';")
    checks.append("preexisting modtime/native hash values block admission and remain unchanged")
    parallel = batch("synthetic-concurrent", ["Concurrent fixture.md"])
    with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:
        result = list(pool.map(lambda _: admission(parallel), range(2)))
    assert result[0][1] == result[1][1]
    assert sql(f"SELECT count(*) FROM {table} WHERE path='Concurrent fixture.md';") == "1"
    checks.append("concurrent identical function calls retain one exact occurrence")
    privilege = json.loads(sql(f"SELECT json_build_object('login',rolcanlogin,'super',rolsuper,'bypass',rolbypassrls,'raw_usage',has_schema_privilege('{role}','{raw_schema}','USAGE'),'api_usage',has_schema_privilege('{role}','{api_schema}','USAGE'),'api_owner',(SELECT proowner::regrole::text FROM pg_proc WHERE oid='{api_schema}.register_ai_workproduct(text,text)'::regprocedure),'search_path',(SELECT proconfig FROM pg_proc WHERE oid='{api_schema}.register_ai_workproduct(text,text)'::regprocedure)) FROM pg_roles WHERE rolname='{role}';"))
    assert not privilege["login"] and not privilege["super"] and not privilege["bypass"] and not privilege["raw_usage"] and privilege["api_usage"]
    result = dict(stage="synthetic-only", complete=True, observed_at=datetime.datetime.now(datetime.timezone.utc).isoformat(), database="casebible", schema=raw_schema, api_schema=api_schema, retained_nologin_role=role,
                  source_sql_sha256=hashlib.sha256(source.encode()).hexdigest(), fixture_sql_sha256=hashlib.sha256(variant.encode()).hexdigest(), substitutions="qualified table/schema/service-role names only", checks=checks, privilege=privilege, owner_notes_used=False, production_admission=False, fixtures_retained=True)
    (proof / "sql-validation.json").write_text(json.dumps(result, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(result))


if __name__ == "__main__":
    main()

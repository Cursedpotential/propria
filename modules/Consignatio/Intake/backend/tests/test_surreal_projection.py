from __future__ import annotations

import hashlib
from dataclasses import dataclass
from datetime import UTC, datetime

import pytest
from surrealdb.cbor import CBORSimpleValue

from casebible_index.projections.surreal import (
    REQUIRED_TABLES,
    GraphOccurrenceProjectionError,
    GraphRecordRef,
    OperationRunWrite,
    ProjectionConflictError,
    ProjectionSnapshotWrite,
    SurrealGraphClient,
    SurrealGraphConfig,
    SurrealGraphError,
    _bound_nulls,
)


def server_bound_values(value):
    """Model object NONE field erasure versus explicit CBOR NULL storage."""
    if isinstance(value, CBORSimpleValue) and value.value == 22:
        return None
    if isinstance(value, dict):
        return {key: server_bound_values(item) for key, item in value.items() if item is not None}
    if isinstance(value, (list, tuple)):
        return [server_bound_values(item) for item in value]
    return value


@dataclass(frozen=True)
class FakeRecordID:
    table: str
    key: str

    def __str__(self):
        return f"{self.table}:{self.key}"


class FakeConnection:
    def __init__(self, endpoint: str = "wss://graph.example"):
        self.endpoint = endpoint
        self.connected = False
        self.closed = False
        self.credentials = None
        self.scope = None
        self.queries = []
        self.records = {}
        self.schema = {name: "DEFINE TABLE" for name in REQUIRED_TABLES}
        self.neighborhood = None
        self.occurrence_rows = []
        self.completion_rows = []
        self.occurrence_result_override = None

    async def __aenter__(self):
        self.connected = True
        return self

    async def __aexit__(self, *args):
        self.closed = True

    async def signin(self, credentials):
        self.credentials = credentials

    async def use(self, namespace, database):
        self.scope = (namespace, database)

    async def version(self):
        return "surrealdb-3.2.4"

    async def query(self, statement, variables=None):
        variables = server_bound_values(variables or {})
        self.queries.append((statement, variables))
        if statement == "INFO FOR DB;":
            return {"tables": self.schema}
        if statement.startswith("SELECT * FROM ONLY"):
            return self.records.get(variables["record_id"])
        if statement.startswith("CREATE ONLY"):
            record_id = variables["record_id"]
            if record_id in self.records:
                raise RuntimeError("duplicate plus provider secret")
            result = {"id": record_id, **variables["document"]}
            self.records[record_id] = result
            return result
        if statement.startswith("UPDATE"):
            record_id = variables["record_id"]
            if self.records.get(record_id) != variables["expected"]:
                return None
            result = {"id": record_id, **variables["document"]}
            self.records[record_id] = result
            return result
        if "RETURN { root:" in statement:
            return self.neighborhood
        if "snapshot: (SELECT * FROM ONLY $snapshot)" in statement:
            return self.neighborhood
        if "FROM stored_at" in statement:
            if self.occurrence_result_override is not None:
                return self.occurrence_result_override
            filtered = [
                {key: value for key, value in row.items() if not key.startswith("fixture_")}
                for row in self.occurrence_rows
                if row["fixture_source_id"] == variables["source_id"]
                and row["document_id"] == variables["document_id"]
                and ("version_id" not in variables or row["version_id"] == variables["version_id"])
            ]
            return filtered[: variables["limit"]]
        if "FROM produced_by" in statement:
            snapshot_ids = {str(value) for value in variables["snapshot_ids"]}
            return [row for row in self.completion_rows if str(row["snapshot_id"]) in snapshot_ids][
                : variables["limit"]
            ]
        raise AssertionError(f"Unexpected statement: {statement}")


CONFIG = SurrealGraphConfig(
    endpoint="wss://graph.example",
    namespace="consignatio",
    database="intake",
    username="intake_runtime",
    password="test-only-secret",
)
NOW = datetime(2026, 9, 12, 14, 0, tzinfo=UTC)


def make_client(connection=None):
    return SurrealGraphClient(CONFIG, connection or FakeConnection(), FakeRecordID)


@pytest.mark.parametrize(
    "change",
    [
        {"endpoint": "mem://"},
        {"endpoint": "file://E:/intake.db"},
        {"endpoint": "wss://user:secret@graph.example"},
        {"namespace": "evidence"},
        {"database": "docs"},
        {"password": ""},
    ],
)
def test_config_fails_closed_for_embedded_credentials_or_wrong_database(change):
    values = {
        "endpoint": CONFIG.endpoint,
        "namespace": CONFIG.namespace,
        "database": CONFIG.database,
        "username": CONFIG.username,
        "password": CONFIG.password,
        **change,
    }
    with pytest.raises(ValueError):
        SurrealGraphConfig(**values).validate()


@pytest.mark.asyncio
async def test_connect_uses_database_scoped_credentials_and_verifies_schema():
    connection = FakeConnection()
    client = await SurrealGraphClient.connect(
        CONFIG,
        connection_factory=lambda endpoint: connection,
        record_factory=FakeRecordID,
    )
    assert connection.connected
    assert connection.credentials == {
        "username": "intake_runtime",
        "password": "test-only-secret",
        "namespace": "consignatio",
        "database": "intake",
    }
    assert connection.scope == ("consignatio", "intake")
    assert await client.verify_health_and_schema() == "surrealdb-3.2.4"
    await client.close()
    assert connection.closed


@pytest.mark.asyncio
async def test_readiness_closes_connection_and_sanitizes_provider_failure():
    connection = FakeConnection()
    connection.schema.pop("occurrence")
    with pytest.raises(SurrealGraphError, match="schema is incomplete") as exc:
        await SurrealGraphClient.connect(
            CONFIG,
            connection_factory=lambda endpoint: connection,
            record_factory=FakeRecordID,
        )
    assert connection.closed
    assert "test-only-secret" not in str(exc.value)


@pytest.mark.asyncio
async def test_operation_run_replay_is_idempotent_and_terminal_status_is_immutable():
    connection = FakeConnection()
    client = make_client(connection)
    running = OperationRunWrite("scan-001", "filesystem-scanner", NOW, "running")
    first = await client.write_operation_run(running)
    query_count = len(connection.queries)
    assert await client.write_operation_run(running) == first
    assert len(connection.queries) == query_count + 1  # replay reads but performs no write

    completed = OperationRunWrite(
        "scan-001", "filesystem-scanner", NOW, "completed", completed_at=NOW
    )
    terminal = await client.write_operation_run(completed)
    assert terminal["status"] == "completed"
    with pytest.raises(ProjectionConflictError, match="cannot change status"):
        await client.write_operation_run(
            OperationRunWrite("scan-001", "filesystem-scanner", NOW, "failed", completed_at=NOW)
        )


@pytest.mark.asyncio
async def test_snapshot_exact_replay_is_noop_and_changed_manifest_fails_closed():
    connection = FakeConnection()
    client = make_client(connection)
    snapshot = ProjectionSnapshotWrite(
        "snapshot-001", "b2://lake/manifests/001.json", "a" * 64, NOW
    )
    first = await client.write_projection_snapshot(snapshot)
    query_count = len(connection.queries)
    assert await client.write_projection_snapshot(snapshot) == first
    assert len(connection.queries) == query_count + 1
    with pytest.raises(ProjectionConflictError, match="conflicts"):
        await client.write_projection_snapshot(
            ProjectionSnapshotWrite(
                "snapshot-001", "b2://lake/manifests/changed.json", "b" * 64, NOW
            )
        )


@pytest.mark.asyncio
async def test_queries_are_constant_and_values_are_parameter_bound():
    connection = FakeConnection()
    client = make_client(connection)
    hostile = "run'; DELETE occurrence; --"
    await client.write_operation_run(OperationRunWrite(hostile, "scanner", NOW, "running"))
    statements = "\n".join(statement for statement, _ in connection.queries)
    assert hostile not in statements
    assert all("password" not in variables for _, variables in connection.queries)
    assert any(
        variables.get("document", {}).get("run_key") == hostile
        for _, variables in connection.queries
    )


@pytest.mark.asyncio
async def test_neighborhood_is_allowlisted_parameterized_and_bounded():
    connection = FakeConnection()
    connection.neighborhood = {
        "root": {"id": FakeRecordID("atomic_unit", "unit-1")},
        "edges": [{"relation": "contains"}],
    }
    client = make_client(connection)
    result = await client.graph_neighborhood(GraphRecordRef("atomic_unit", "unit-1"), edge_limit=5)
    assert len(result.edges) == 1
    statement, variables = connection.queries[-1]
    assert "unit-1" not in statement
    assert variables == {"root": FakeRecordID("atomic_unit", "unit-1"), "edge_limit": 5}
    with pytest.raises(ValueError):
        await client.graph_neighborhood(GraphRecordRef("contains", "edge-1"))
    with pytest.raises(ValueError):
        await client.graph_neighborhood(GraphRecordRef("atomic_unit", "unit-1"), edge_limit=101)


@pytest.mark.asyncio
async def test_projection_summary_is_parameterized_and_validates_shape():
    connection = FakeConnection()
    connection.neighborhood = {
        "snapshot": {"snapshot_key": "safe:key"},
        "completion": [],
        "occurrences": 2,
        "stores": 1,
        "stored_at": 2,
        "content_links": 1,
    }
    client = make_client(connection)
    result = await client.projection_summary("safe:key")
    assert result["occurrences"] == 2
    statement, variables = connection.queries[-1]
    assert "safe:key" not in statement
    assert variables == {
        "snapshot": FakeRecordID("projection_snapshot", hashlib.sha256(b"safe:key").hexdigest())
    }
    with pytest.raises(ValueError):
        await client.projection_summary("bad/key")


def occurrence_fixture(
    source_id,
    snapshot_key,
    document_id="doc-a",
    version_id="v1",
    tool_name="intake-index-run-projection",
):
    """Build one synthetic occurrence row and matching completed operation marker."""
    snapshot_digest = hashlib.sha256(snapshot_key.encode()).hexdigest()
    occurrence_stable_key = f"{snapshot_key}:occurrence:{source_id}:{document_id}"
    return (
        {
            "fixture_source_id": source_id,
            "occurrence_id": FakeRecordID(
                "occurrence", hashlib.sha256(occurrence_stable_key.encode()).hexdigest()
            ),
            "snapshot_id": FakeRecordID("projection_snapshot", snapshot_digest),
            "snapshot_key": snapshot_key,
            "manifest_sha256": snapshot_digest,
            "document_id": document_id,
            "version_id": version_id,
        },
        {
            "snapshot_id": FakeRecordID("projection_snapshot", snapshot_digest),
            "completion_id": FakeRecordID("produced_by", "edge-" + snapshot_digest),
            "run_id": FakeRecordID(
                "operation_run",
                hashlib.sha256(f"{snapshot_key}:complete".encode()).hexdigest(),
            ),
            "run_key": f"{snapshot_key}:complete",
            "tool_name": tool_name,
            "tool_version": "fixture-v1",
            "status": "completed",
            "completed_at": NOW,
        },
    )


@pytest.mark.asyncio
async def test_occurrence_resolver_binds_source_and_document_and_returns_typed_ids():
    connection = FakeConnection()
    wanted, completion = occurrence_fixture("source-a", "snapshot-a")
    other_source, _ = occurrence_fixture("source-b", "snapshot-b")
    connection.occurrence_rows = [wanted, other_source]
    connection.completion_rows = [completion]
    client = make_client(connection)

    result = await client.resolve_occurrences("source-a", "doc-a")

    assert len(result.matches) == 1
    match = result.matches[0]
    assert match.occurrence_id == str(wanted["occurrence_id"])
    assert match.occurrence_key == wanted["occurrence_id"].key
    assert match.snapshot_id == str(wanted["snapshot_id"])
    assert match.snapshot_key == "snapshot-a"
    assert match.manifest_sha256 == wanted["manifest_sha256"]
    assert match.version_id == "v1"
    assert match.completion_run_id == str(completion["run_id"])
    assert (result.ambiguous, result.overflow) == (False, False)

    occurrence_sql, occurrence_vars = next(
        item for item in connection.queries if "FROM stored_at" in item[0]
    )
    assert "source-a" not in occurrence_sql and "doc-a" not in occurrence_sql
    assert occurrence_vars == {
        "source_id": "source-a",
        "document_id": "doc-a",
        "limit": 101,
    }


@pytest.mark.asyncio
async def test_occurrence_resolver_returns_all_snapshots_and_optional_version_filters():
    connection = FakeConnection()
    first, first_completion = occurrence_fixture("source-a", "snapshot-a", version_id="v1")
    second, second_completion = occurrence_fixture("source-a", "snapshot-b", version_id="v2")
    connection.occurrence_rows = [first, second]
    connection.completion_rows = [first_completion, second_completion]
    client = make_client(connection)

    all_versions = await client.resolve_occurrences("source-a", "doc-a")
    selected_version = await client.resolve_occurrences("source-a", "doc-a", version_id="v2")

    assert all_versions.ambiguous is True
    assert [match.snapshot_key for match in all_versions.matches] == ["snapshot-a", "snapshot-b"]
    assert selected_version.ambiguous is False
    assert [match.version_id for match in selected_version.matches] == ["v2"]
    versioned_sql, versioned_vars = [
        item for item in connection.queries if "FROM stored_at" in item[0]
    ][-1]
    assert "AND in.metadata.version_id = $version_id" in versioned_sql
    assert versioned_vars["version_id"] == "v2"


@pytest.mark.asyncio
@pytest.mark.parametrize(
    "tool_name",
    [
        "intake-index-run-projection",
        "inventory-manifest-projection",
        "r2-b2-occurrence-map-projection",
        "legacy-catalog-projection",
    ],
)
async def test_occurrence_resolver_accepts_existing_projection_completion_writers(tool_name):
    connection = FakeConnection()
    occurrence, marker = occurrence_fixture("source-a", "snapshot-a", tool_name=tool_name)
    connection.occurrence_rows = [occurrence]
    connection.completion_rows = [marker]

    result = await make_client(connection).resolve_occurrences("source-a", "doc-a")

    assert len(result.matches) == 1
    assert result.matches[0].completion_tool_version == "fixture-v1"


@pytest.mark.asyncio
async def test_occurrence_resolver_reports_overflow_without_returning_partial_ids():
    connection = FakeConnection()
    for index in range(101):
        row, completion = occurrence_fixture("source-a", f"snapshot-{index:03}")
        connection.occurrence_rows.append(row)
        connection.completion_rows.append(completion)
    result = await make_client(connection).resolve_occurrences("source-a", "doc-a")
    assert result.matches == ()
    assert result.ambiguous is True
    assert result.overflow is True
    assert not any("FROM produced_by" in statement for statement, _ in connection.queries)


@pytest.mark.asyncio
@pytest.mark.parametrize("failure", ["missing", "failed", "wrong_tool", "wrong_run", "duplicate"])
async def test_occurrence_resolver_rejects_invalid_completion_markers(failure):
    connection = FakeConnection()
    occurrence, marker = occurrence_fixture("source-a", "snapshot-a")
    connection.occurrence_rows = [occurrence]
    if failure != "missing":
        if failure == "failed":
            marker["status"] = "failed"
        elif failure == "wrong_tool":
            marker["tool_name"] = "other-projection"
        elif failure == "wrong_run":
            marker["run_key"] = "wrong-run"
        connection.completion_rows = [marker]
    if failure == "duplicate":
        duplicate = dict(marker)
        duplicate["completion_id"] = FakeRecordID("produced_by", "edge-duplicate")
        duplicate["run_id"] = FakeRecordID("operation_run", "f" * 64)
        connection.completion_rows = [marker, duplicate]

    with pytest.raises(GraphOccurrenceProjectionError):
        await make_client(connection).resolve_occurrences("source-a", "doc-a")


@pytest.mark.asyncio
async def test_occurrence_resolver_rejects_sdk_shape_mismatch_without_body_leak():
    connection = FakeConnection()
    connection.occurrence_result_override = {"unexpected": "private body"}
    with pytest.raises(SurrealGraphError, match="identity response is invalid") as error:
        await make_client(connection).resolve_occurrences("source-a", "doc-a")
    assert "private body" not in str(error.value)


@pytest.mark.asyncio
async def test_snapshot_replay_cannot_omit_previously_stored_checkpoint():
    client = make_client()
    await client.write_projection_snapshot(
        ProjectionSnapshotWrite(
            "snapshot-optional", "fixture://manifest", "a" * 64, NOW, source_checkpoint="A"
        )
    )
    with pytest.raises(ProjectionConflictError):
        await client.write_projection_snapshot(
            ProjectionSnapshotWrite("snapshot-optional", "fixture://manifest", "a" * 64, NOW)
        )


@pytest.mark.asyncio
async def test_concurrent_terminal_run_cannot_be_overwritten():
    class RacingConnection(FakeConnection):
        async def query(self, statement, variables=None):
            if statement.startswith("UPDATE"):
                record = self.records[variables["record_id"]]
                record.update(status="failed", completed_at=NOW)
            return await super().query(statement, variables)

    connection = RacingConnection()
    client = make_client(connection)
    await client.write_operation_run(OperationRunWrite("race", "scanner", NOW, "running"))
    with pytest.raises(ProjectionConflictError, match="concurrently"):
        await client.write_operation_run(
            OperationRunWrite("race", "scanner", NOW, "completed", completed_at=NOW)
        )
    assert next(iter(connection.records.values()))["status"] == "failed"


def test_sdk_codec_preserves_null_keys_and_does_not_encode_them_as_none():
    from surrealdb.cbor import loads
    from surrealdb.data.cbor import decode, encode

    original = {"metadata": {"unknown": None, "nested": [None, {"missing_hash": None}]}}
    # The SDK decoder collapses both NONE and NULL to Python None; inspect wire bytes.
    assert encode({"x": None}).hex() == "a16178c6f6"  # CBOR tag 6 + null
    assert encode(_bound_nulls({"x": None})).hex() == "a16178f6"  # plain CBOR null
    bound = _bound_nulls(original)
    encoded = encode(bound)
    assert loads(encoded)["metadata"]["unknown"] is None
    assert decode(encoded) == original
    assert original["metadata"]["unknown"] is None  # caller document is unchanged
    assert server_bound_values(original) != original  # mock detects the old data-loss behavior


@pytest.mark.asyncio
async def test_explicit_null_metadata_survives_create_cas_and_exact_replay():
    from dataclasses import replace

    connection = FakeConnection()
    client = make_client(connection)
    run = OperationRunWrite("null-run", "scanner", NOW, "running", metadata={"unknown": None})
    stored = await client.write_operation_run(run)
    assert stored["metadata"] == {"unknown": None}
    completed = replace(run, status="completed", completed_at=NOW)
    assert (await client.write_operation_run(completed))["metadata"] == {"unknown": None}
    assert (await client.write_operation_run(completed))["status"] == "completed"
    snapshot = ProjectionSnapshotWrite(
        "null-snapshot", "fixture://manifest", "c" * 64, NOW, metadata={"unknown": None}
    )
    await client.write_projection_snapshot(snapshot)
    assert (await client.write_projection_snapshot(snapshot))["metadata"] == {"unknown": None}
    with pytest.raises(ProjectionConflictError):
        await client.write_projection_snapshot(replace(snapshot, metadata={}))

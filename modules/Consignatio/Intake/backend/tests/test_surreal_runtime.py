from __future__ import annotations

from dataclasses import replace

import pytest
from fastapi.testclient import TestClient

from casebible_index.api import create_api
from casebible_index.config import Settings
from casebible_index.projections import runtime
from casebible_index.projections.surreal import (
    GraphOccurrenceMatch,
    GraphOccurrenceProjectionError,
    GraphOccurrenceResolution,
)


def test_factory_uses_secret_bridge_and_fixed_database_identity(monkeypatch):
    monkeypatch.setenv("INTAKE_SURREAL_URL", "http://graph.example:8473")
    monkeypatch.setenv("INTAKE_SURREAL_PASSWORD", "untrusted-env-password")
    monkeypatch.setenv("INTAKE_SURREAL_DATABASE", "evidence")
    seen = []

    def secret(name):
        seen.append(name)
        return "keychain-test-password"

    monkeypatch.setattr(runtime, "get_secret", secret)
    config = runtime.graph_config_from_env()
    assert config.password == "keychain-test-password"
    assert (config.namespace, config.database, config.username) == (
        "consignatio",
        "intake",
        "intake_runtime",
    )
    assert seen == ["INTAKE_SURREAL_PASSWORD"]
    assert "keychain-test-password" not in repr(config)
    with pytest.raises(ValueError, match="intake_runtime"):
        replace(config, username="intake_root").validate()


def test_factory_requires_explicit_endpoint_and_credential(monkeypatch):
    monkeypatch.delenv("INTAKE_SURREAL_URL", raising=False)
    monkeypatch.setattr(runtime, "get_secret", lambda name: None)
    with pytest.raises(ValueError):
        runtime.graph_config_from_env()


def test_graph_api_validates_before_connecting_and_sanitizes_failure(monkeypatch, tmp_path):
    import casebible_index.api as api_module

    calls = []

    async def fail():
        calls.append(True)
        raise RuntimeError("private-provider-secret")

    monkeypatch.setattr(api_module, "connect_graph", fail)
    settings = replace(
        Settings.from_env(), source_dir=tmp_path / "source", output_dir=tmp_path / "output"
    )
    client = TestClient(create_api(settings))
    assert client.get("/filesystem/graph/neighbors/contains/key").status_code == 422
    oversized = client.get("/filesystem/graph/neighbors/occurrence/key?edge_limit=101")
    assert oversized.status_code == 422
    assert not calls
    response = client.get("/filesystem/graph/status")
    assert response.status_code == 503
    assert "private-provider-secret" not in response.text


def test_projection_summary_api_reports_completion_and_rejects_bad_key(monkeypatch, tmp_path):
    import casebible_index.api as api_module

    class Graph:
        async def __aenter__(self):
            return self

        async def __aexit__(self, *_):
            return None

        async def projection_summary(self, key):
            return {
                "snapshot": {"snapshot_key": key},
                "completion": [{"status": "completed"}],
                "occurrences": 3,
                "stores": 1,
                "stored_at": 3,
                "content_links": 2,
            }

    async def connect():
        return Graph()

    monkeypatch.setattr(api_module, "connect_graph", connect)
    settings = replace(
        Settings.from_env(), source_dir=tmp_path / "source", output_dir=tmp_path / "output"
    )
    client = TestClient(create_api(settings))
    response = client.get("/filesystem/graph/projections/safe:key")
    assert response.status_code == 200
    assert response.json()["completion"] == [{"status": "completed"}]
    assert client.get("/filesystem/graph/projections/bad%2Fkey").status_code in {404, 422}


def test_graph_resolver_returns_only_typed_identity_and_provenance(monkeypatch, tmp_path):
    import casebible_index.api as api_module

    class Graph:
        async def __aenter__(self):
            return self

        async def __aexit__(self, *_):
            return None

        async def resolve_occurrences(self, source_id, document_id, *, version_id=None):
            assert (source_id, document_id, version_id) == ("source-a", "doc-a", "v1")
            return GraphOccurrenceResolution(
                matches=(
                    GraphOccurrenceMatch(
                        occurrence_id="occurrence:" + "a" * 64,
                        occurrence_key="a" * 64,
                        snapshot_id="projection_snapshot:" + "b" * 64,
                        snapshot_key="index-run:source-a:0123456789abcdef",
                        manifest_sha256="c" * 64,
                        version_id="v1",
                        completion_run_id="operation_run:" + "d" * 64,
                        completion_tool_version="fixture-v1",
                    ),
                ),
                ambiguous=False,
                overflow=False,
            )

    async def connect():
        return Graph()

    monkeypatch.setattr(api_module, "connect_graph", connect)
    settings = replace(
        Settings.from_env(), source_dir=tmp_path / "source", output_dir=tmp_path / "output"
    )
    client = TestClient(create_api(settings))
    response = client.post(
        "/filesystem/graph/resolve",
        json={"source_id": "source-a", "document_id": "doc-a", "version_id": "v1"},
    )
    assert response.status_code == 200
    body = response.json()
    assert body["ambiguous"] is False
    assert body["overflow"] is False
    assert body["matches"][0]["record_id"] == "occurrence:" + "a" * 64
    assert body["matches"][0]["key"] == "a" * 64
    assert body["matches"][0]["snapshot_key"] == "index-run:source-a:0123456789abcdef"
    assert not {"text", "path_raw", "source_path", "manifest_uri"} & set(body["matches"][0])


def test_graph_resolver_rejects_invalid_input_and_projection_integrity_failures(
    monkeypatch, tmp_path
):
    import casebible_index.api as api_module

    class Graph:
        async def __aenter__(self):
            return self

        async def __aexit__(self, *_):
            return None

        async def resolve_occurrences(self, *_args, **_kwargs):
            raise GraphOccurrenceProjectionError("private graph detail")

    async def connect():
        return Graph()

    monkeypatch.setattr(api_module, "connect_graph", connect)
    settings = replace(
        Settings.from_env(), source_dir=tmp_path / "source", output_dir=tmp_path / "output"
    )
    client = TestClient(create_api(settings))
    invalid = client.post(
        "/filesystem/graph/resolve", json={"source_id": " ", "document_id": "doc-a"}
    )
    assert invalid.status_code == 422
    failed = client.post(
        "/filesystem/graph/resolve", json={"source_id": "source-a", "document_id": "doc-a"}
    )
    assert failed.status_code == 503
    assert "private graph detail" not in failed.text

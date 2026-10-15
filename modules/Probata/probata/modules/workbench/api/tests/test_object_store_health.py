"""Active-source readiness tests; all provider calls are isolated read-only fakes.

Byline: Codex · GPT-6.1-Sol · Workbench storage-health lane · 2026-10-06.
Attribution verified and integrated by Codex orchestrator, 2026-10-06.
"""

from __future__ import annotations

import asyncio
import json
import logging
from unittest.mock import Mock

import pytest
from app.repo import object_store_client as store
from app.types.source_roots import parse_source_roots
from botocore.exceptions import ClientError


@pytest.fixture
def active_store(monkeypatch):
    """Configure an explicit source prefix and forbid all non-list SDK effects."""
    roots = parse_source_roots('[{"id":"active","label":"Active","url":"b2://fixture-vault/owner/case/"}]')
    monkeypatch.setattr(store, "SOURCE_ROOTS", roots)
    monkeypatch.setenv("OBJECT_STORES_JSON", '{"b2":"/fixture/b2.json"}')
    monkeypatch.setattr(store, "get_r2_client", Mock(side_effect=AssertionError("retired provider queried")))

    class ReadOnlyClient:
        def __init__(self):
            self.calls: list[dict] = []
            self.response = {"ResponseMetadata": {"HTTPStatusCode": 200}, "Contents": []}
            self.error: Exception | None = None

        def list_objects_v2(self, **kwargs):
            self.calls.append(kwargs)
            if self.error is not None:
                raise self.error
            return self.response

        def __getattr__(self, name):
            raise AssertionError(f"forbidden SDK effect: {name}")

    client = ReadOnlyClient()
    client_factory = Mock(return_value=client)
    monkeypatch.setattr(store, "get_store_client", client_factory)
    return client, client_factory


def test_b2_readiness_uses_only_one_key_inside_the_active_prefix(active_store) -> None:
    client, factory = active_store
    assert store.check_connectivity() is True
    assert client.calls == [{"Bucket": "fixture-vault", "Prefix": "owner/case/", "MaxKeys": 1}]
    factory.assert_called_once_with("b2")
    store.get_r2_client.assert_not_called()


def test_empty_prefix_listing_is_valid_readiness(active_store) -> None:
    client, _ = active_store
    assert client.response["Contents"] == []
    assert store.check_connectivity() is True


def test_all_active_roots_are_correlated_to_their_provider_and_prefix(monkeypatch, active_store) -> None:
    client, factory = active_store
    monkeypatch.setattr(
        store,
        "SOURCE_ROOTS",
        parse_source_roots(
            json.dumps(
                [
                    {"id": "a", "label": "A", "url": "b2://first-vault/first/case/"},
                    {"id": "b", "label": "B", "url": "archive://second-vault/second/case/"},
                ]
            )
        ),
    )
    monkeypatch.setenv("OBJECT_STORES_JSON", '{"b2":"/fixture/b2.json","archive":"/fixture/archive.json"}')
    assert store.check_connectivity() is True
    assert [call.args for call in factory.call_args_list] == [("b2",), ("archive",)]
    assert client.calls == [
        {"Bucket": "first-vault", "Prefix": "first/case/", "MaxKeys": 1},
        {"Bucket": "second-vault", "Prefix": "second/case/", "MaxKeys": 1},
    ]


@pytest.mark.parametrize(
    "raw",
    [None, "", "not-json", "[]", "{}", '{"archive":"/fixture/archive.json"}', '{"b2":"relative.json"}'],
)
def test_missing_or_wrong_store_configuration_fails_before_provider_io(monkeypatch, active_store, raw) -> None:
    client, factory = active_store
    if raw is None:
        monkeypatch.delenv("OBJECT_STORES_JSON", raising=False)
    else:
        monkeypatch.setenv("OBJECT_STORES_JSON", raw)
    assert store.check_connectivity() is False
    assert client.calls == []
    factory.assert_not_called()
    store.get_r2_client.assert_not_called()


def test_retired_roots_never_trigger_a_storage_probe(monkeypatch, active_store) -> None:
    """Reject the retired provider before making any storage request."""
    client, factory = active_store
    monkeypatch.setenv("OBJECT_STORES_JSON", '{"b2":"/fixture/b2.json","r2":"/fixture/retired.json"}')
    monkeypatch.setattr(
        store, "SOURCE_ROOTS", parse_source_roots('[{"id":"bad","label":"Bad","url":"r2://retired-vault/historical/"}]')
    )
    assert store.check_connectivity() is False
    assert client.calls == []
    factory.assert_not_called()
    store.get_r2_client.assert_not_called()


def test_explicit_whole_bucket_root_uses_a_bounded_read_only_probe(monkeypatch, active_store) -> None:
    """Allow browsing an explicitly configured bucket using one metadata key only."""
    client, factory = active_store
    monkeypatch.setattr(
        store, "SOURCE_ROOTS", parse_source_roots('[{"id":"bucket","label":"Bucket","url":"b2://fixture-vault/"}]')
    )
    assert store.check_connectivity() is True
    assert client.calls == [{"Bucket": "fixture-vault", "Prefix": "", "MaxKeys": 1}]
    factory.assert_called_once_with("b2")
    store.get_r2_client.assert_not_called()


def test_no_source_roots_fails_closed(monkeypatch, active_store) -> None:
    _, factory = active_store
    monkeypatch.setattr(store, "SOURCE_ROOTS", {})
    assert store.check_connectivity() is False
    factory.assert_not_called()


def test_unused_historical_credentials_are_never_queried(monkeypatch, active_store) -> None:
    _, factory = active_store
    monkeypatch.setenv("OBJECT_STORES_JSON", '{"b2":"/fixture/b2.json","r2":"/fixture/retired.json"}')
    assert store.check_connectivity() is True
    factory.assert_called_once_with("b2")
    store.get_r2_client.assert_not_called()


def test_one_misconfigured_root_prevents_even_partial_provider_io(monkeypatch, active_store) -> None:
    _, factory = active_store
    monkeypatch.setattr(
        store,
        "SOURCE_ROOTS",
        parse_source_roots(
            '[{"id":"a","label":"A","url":"b2://fixture-vault/owner/case/"},'
            '{"id":"b","label":"B","url":"archive://another-vault/other/case/"}]'
        ),
    )
    assert store.check_connectivity() is False
    factory.assert_not_called()


def test_a_denied_second_root_cannot_be_hidden_by_a_healthy_first_root(monkeypatch, active_store) -> None:
    client, factory = active_store
    monkeypatch.setattr(
        store,
        "SOURCE_ROOTS",
        parse_source_roots(
            '[{"id":"a","label":"A","url":"b2://fixture-vault/owner/case/"},'
            '{"id":"b","label":"B","url":"b2://fixture-vault/other/case/"}]'
        ),
    )
    factory.side_effect = [client, RuntimeError("second root unavailable")]
    assert store.check_connectivity() is False
    assert factory.call_count == 2
    assert client.calls == [{"Bucket": "fixture-vault", "Prefix": "owner/case/", "MaxKeys": 1}]


@pytest.mark.parametrize("credential_state", ["absent", "malformed", "wrong-schema"])
def test_actual_missing_or_invalid_credential_file_fails_before_sdk(
    monkeypatch, active_store, tmp_path, credential_state
) -> None:
    credential_path = tmp_path / "fixture-only.json"
    if credential_state != "absent":
        credential_path.write_text("not-json" if credential_state == "malformed" else "{}", encoding="utf-8")
    monkeypatch.setattr(store, "configured_object_stores", lambda **_: {"b2": str(credential_path)})
    monkeypatch.setattr(store, "get_store_client", store._other_store_client)
    sdk_factory = Mock(side_effect=AssertionError("SDK must not run"))
    monkeypatch.setattr(store.boto3, "client", sdk_factory)
    store._other_store_client.cache_clear()
    try:
        assert store.check_connectivity() is False
        sdk_factory.assert_not_called()
    finally:
        store._other_store_client.cache_clear()


@pytest.mark.parametrize("error_type", [ClientError, RuntimeError, TimeoutError, OSError])
def test_upstream_denial_or_client_failure_is_false_with_safe_logs(active_store, caplog, error_type) -> None:
    client, factory = active_store
    sensitive = "credential=/private/secret.json; object=private-object; token=never-log-this"
    error = (
        ClientError({"Error": {"Code": "AccessDenied", "Message": sensitive}}, "ListObjectsV2")
        if error_type is ClientError
        else error_type(sensitive)
    )
    if error_type is OSError:
        factory.side_effect = error
    else:
        client.error = error
    with caplog.at_level(logging.WARNING, logger=store.logger.name):
        assert store.check_connectivity() is False
    assert "Object store connectivity check failed" in caplog.text
    assert error_type.__name__ in caplog.text
    assert sensitive not in caplog.text
    assert all(record.exc_info is None for record in caplog.records)


@pytest.mark.parametrize("response", [{}, None, {"ResponseMetadata": {"HTTPStatusCode": 403}}])
def test_unconfirmed_provider_response_fails_closed(active_store, response) -> None:
    client, _ = active_store
    client.response = response
    assert store.check_connectivity() is False


@pytest.mark.parametrize("denied", [False, True])
def test_public_health_contract_tracks_active_storage_without_relaxation(monkeypatch, active_store, denied) -> None:
    from app.runtime import health

    client, _ = active_store
    if denied:
        client.error = RuntimeError("provider unavailable")
    monkeypatch.setattr(health, "check_lancedb_connectivity", lambda: True)
    monkeypatch.setattr(health, "check_connectivity", store.check_connectivity)
    body = asyncio.run(health.health())
    assert body == {"status": "degraded" if denied else "ok", "lancedb": True, "object_store": not denied}

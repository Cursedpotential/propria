# Byline: Claude Code · Fable 5.1 · 2026-09-20
"""Object stores and source roots are configuration, not code."""

import pytest

from app.repo import object_store_client as client
from app.types.source_roots import parse_object_stores, parse_source_roots


def test_roots_parse_any_provider_and_keep_legacy_default() -> None:
    roots = parse_source_roots(
        '[{"id": "b2-vault", "label": "B2 / Vault", "url": "b2://salem-data/consignatio/vault/v1/"},'
        ' {"id": "w", "label": "Wasabi", "url": "wasabi://bucket/", "temporary": true}]'
    )
    assert list(roots) == ["b2-vault", "w"]
    vault = roots["b2-vault"]
    assert (vault.scheme, vault.bucket, vault.key_prefix, vault.temporary) == (
        "b2",
        "salem-data",
        "consignatio/vault/v1/",
        False,
    )
    assert vault.source_ref("Takeout/a.xml") == "b2://salem-data/consignatio/vault/v1/Takeout/a.xml"
    assert next(iter(parse_source_roots(""))) == "r2-sorted"


@pytest.mark.parametrize(
    "raw",
    [
        '[{"id": "a", "label": "x", "url": "b2://b/no-slash"}]',
        '[{"id": "a", "label": "x", "url": "b2://b/../p/"}]',
        '[{"id": "a", "label": "x", "url": "b2://user@b/p/"}]',
        '[{"id": "a", "label": "x", "url": "b2://b/"}, {"id": "a", "label": "y", "url": "r2://c/"}]',
        "[]",
    ],
)
def test_roots_reject_malformed(raw: str) -> None:
    with pytest.raises(ValueError):
        parse_source_roots(raw)


def test_stores_parse_and_fall_back_to_legacy_r2() -> None:
    assert parse_object_stores('{"b2": "/run/secrets/b2.json"}') == {"b2": "/run/secrets/b2.json"}
    assert parse_object_stores("", legacy_r2_path="/run/secrets/r2.json") == {"r2": "/run/secrets/r2.json"}
    for bad in ('{"upload": "/x"}', '{"B2": "/x"}', '{"b2": "relative"}', "{}"):
        with pytest.raises(ValueError):
            parse_object_stores(bad)


def test_listing_prepends_the_root_prefix_and_returns_relative_keys(monkeypatch) -> None:
    root = parse_source_roots('[{"id": "v", "label": "Vault", "url": "b2://salem-data/consignatio/vault/v1/"}]')["v"]
    monkeypatch.setattr(client, "SOURCE_ROOTS", {"v": root})
    calls: dict[str, object] = {}

    class Fake:
        def list_objects_v2(self, **request):
            calls["list"] = request
            return {
                "Contents": [
                    {"Key": "consignatio/vault/v1/Takeout/a.xml", "Size": 3},
                    {"Key": "consignatio/vault/v1/"},
                ],
                "CommonPrefixes": [{"Prefix": "consignatio/vault/v1/Takeout/sub/"}],
            }

        def head_object(self, **request):
            calls["head"] = request
            return {}

    monkeypatch.setattr(client, "get_store_client", lambda scheme: calls.setdefault("scheme", scheme) and Fake())
    page = client.list_source_objects(root_id="v", prefix="Takeout/", start_after="Takeout/0.xml")
    assert calls["scheme"] == "b2"
    assert calls["list"]["Bucket"] == "salem-data"
    assert calls["list"]["Prefix"] == "consignatio/vault/v1/Takeout/"
    assert calls["list"]["StartAfter"] == "consignatio/vault/v1/Takeout/0.xml"
    assert [row["Key"] for row in page["Contents"]] == ["Takeout/a.xml"]
    assert [row["Prefix"] for row in page["CommonPrefixes"]] == ["Takeout/sub/"]
    client.head_source_object("v", "Takeout/a.xml")
    assert calls["head"] == {"Bucket": "salem-data", "Key": "consignatio/vault/v1/Takeout/a.xml"}

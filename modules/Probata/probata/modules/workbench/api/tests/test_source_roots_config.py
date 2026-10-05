# Byline: Claude Code · Fable 5.1 · 2026-09-20; Codex · GPT-6-Luna · 2026-10-04
"""Object stores and source roots are configuration, not code."""

import pytest

from app.repo import object_store_client as client
from app.types.source_roots import (
    configured_object_stores,
    configured_source_roots,
    parse_object_stores,
    parse_source_roots,
    validate_authorized_source_ref,
)


def test_roots_parse_any_provider_without_an_implicit_default() -> None:
    roots = parse_source_roots(
        '[{"id": "b2-vault", "label": "B2 / Casevault", "url": "b2://salem-data/consignatio/casevault/"},'
        ' {"id": "w", "label": "Wasabi", "url": "wasabi://bucket/", "temporary": true}]'
    )
    assert list(roots) == ["b2-vault", "w"]
    vault = roots["b2-vault"]
    assert (vault.scheme, vault.bucket, vault.key_prefix, vault.temporary) == (
        "b2",
        "salem-data",
        "consignatio/casevault/",
        False,
    )
    key = "SourceCorpus/messaging/test-source.xml"
    assert vault.source_ref(key) == "b2://salem-data/consignatio/casevault/" + key
    assert parse_source_roots("") == {"b2-vault": vault}


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


def test_stores_require_explicit_config_and_reject_legacy_r2_fallback() -> None:
    assert parse_object_stores('{"b2": "/run/secrets/b2.json"}') == {"b2": "/run/secrets/b2.json"}
    assert parse_object_stores("") == {}
    with pytest.raises(ValueError, match="legacy R2 credential fallback is retired"):
        parse_object_stores("", legacy_r2_path="/run/secrets/r2.json")
    for bad in ('{"upload": "/x"}', '{"B2": "/x"}', '{"b2": "relative"}', "{}"):
        with pytest.raises(ValueError):
            parse_object_stores(bad)


def test_unset_environment_uses_only_b2_root_and_no_credential_fallback(monkeypatch) -> None:
    monkeypatch.delenv("SOURCE_ROOTS_JSON", raising=False)
    monkeypatch.delenv("OBJECT_STORES_JSON", raising=False)
    assert list(configured_source_roots()) == ["b2-vault"]
    assert configured_source_roots()["b2-vault"].root_ref == "b2://salem-data/consignatio/casevault/"
    assert configured_object_stores() == {}


@pytest.mark.parametrize(
    "source_ref",
    [
        "r2://casebible-sorted/source.zip",
        "r2://nexus/workbench/staging/" + "a" * 64 + "/source.zip",
    ],
)
def test_current_acquisition_explicitly_rejects_retired_r2(source_ref: str, monkeypatch) -> None:
    monkeypatch.setenv("SOURCE_ROOTS_JSON", '[{"id":"r2","label":"R2","url":"r2://bucket/"}]')
    with pytest.raises(ValueError, match="r2:// is retired for new acquisitions"):
        validate_authorized_source_ref(source_ref)


def test_current_acquisition_keeps_b2_locator_identity(monkeypatch) -> None:
    monkeypatch.setenv(
        "SOURCE_ROOTS_JSON",
        '[{"id":"casevault","label":"B2 Casevault","url":"b2://salem-data/consignatio/casevault/"}]',
    )
    source_ref = "b2://salem-data/consignatio/casevault/SourceCorpus/messages/file.xml"
    assert validate_authorized_source_ref(source_ref) == source_ref


def test_listing_prepends_the_root_prefix_and_returns_relative_keys(monkeypatch) -> None:
    root = parse_source_roots('[{"id": "v", "label": "Casevault", "url": "b2://salem-data/consignatio/casevault/"}]')["v"]
    monkeypatch.setattr(client, "SOURCE_ROOTS", {"v": root})
    calls: dict[str, object] = {}

    class Fake:
        def list_objects_v2(self, **request):
            calls["list"] = request
            return {
                "Contents": [
                    {"Key": "consignatio/casevault/Takeout/a.xml", "Size": 3},
                    {"Key": "consignatio/casevault/"},
                ],
                "CommonPrefixes": [{"Prefix": "consignatio/casevault/Takeout/sub/"}],
            }

        def head_object(self, **request):
            calls["head"] = request
            return {}

    monkeypatch.setattr(client, "get_store_client", lambda scheme: calls.setdefault("scheme", scheme) and Fake())
    page = client.list_source_objects(root_id="v", prefix="Takeout/", start_after="Takeout/0.xml")
    assert calls["scheme"] == "b2"
    assert calls["list"]["Bucket"] == "salem-data"
    assert calls["list"]["Prefix"] == "consignatio/casevault/Takeout/"
    assert calls["list"]["StartAfter"] == "consignatio/casevault/Takeout/0.xml"
    assert [row["Key"] for row in page["Contents"]] == ["Takeout/a.xml"]
    assert [row["Prefix"] for row in page["CommonPrefixes"]] == ["Takeout/sub/"]
    client.head_source_object("v", "Takeout/a.xml")
    assert calls["head"] == {"Bucket": "salem-data", "Key": "consignatio/casevault/Takeout/a.xml"}


# Byline: Codex · GPT-6 · 2026-10-05.
def test_acquisition_preserves_an_exact_b2_version(monkeypatch) -> None:
    """Keep pinned provider identity through admission without changing the locator.

    Inputs: isolated configured B2 root. Outputs: assertions. Effects: environment fixture only.
    Choose for acquisition pin compatibility rather than generic configured-provider parsing.
    """
    monkeypatch.delenv("SOURCE_ROOTS_JSON", raising=False)
    locator = "b2://salem-data/consignatio/casevault/KnowledgeBase/legal/reference-data/guide.md?versionId=retained%2Fversion%2B1"
    assert validate_authorized_source_ref(locator) == locator


@pytest.mark.parametrize("query", ["versionId=", "versionId=null", "versionId=a&versionId=b", "other=a", "versionId=a&other=b", "versionId=%0A", "versionId=%00", "versionId=%xx", "versionId=" + "a" * 2049])
def test_acquisition_rejects_ambiguous_version_pins(monkeypatch, query) -> None:
    """Reject malformed pins so an exact source request cannot silently resolve latest.

    Inputs: invalid retained-version query. Outputs: assertions. Effects: isolated environment only.
    Choose for pin validation; legacy unversioned admission is covered separately.
    """
    monkeypatch.delenv("SOURCE_ROOTS_JSON", raising=False)
    with pytest.raises(ValueError):
        validate_authorized_source_ref("b2://salem-data/consignatio/casevault/example.md?" + query)

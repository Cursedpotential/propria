"""Multi-bucket catalog objects. Byline: Claude Code · Sonnet 5.5 · 2026-10-02"""

from types import MappingProxyType

import pytest

from casebible_index.catalog_source import CatalogObject
from casebible_index.pipeline import catalog_stable_key
from casebible_index.vault_source import VaultFile


def obj(provider: str, bucket: str, primary: bool) -> CatalogObject:
    return CatalogObject(
        key="k/file.txt",
        byte_size=3,
        sha1="aa",
        fields=MappingProxyType({"provider": provider, "bucket": bucket, "primary": primary}),
    )


def test_stable_keys_keep_existing_memo_for_the_primary_bucket_and_qualify_the_rest():
    assert catalog_stable_key("b2", "salem-data", "k/file.txt", True) == "k/file.txt"
    assert (
        catalog_stable_key("r2", "casebible-raw", "k/file.txt", False)
        == "r2:casebible-raw:k/file.txt"
    )


def test_locator_is_the_plain_key_for_the_primary_bucket_and_a_uri_elsewhere():
    assert VaultFile(obj("b2", "salem-data", True)).locator == "k/file.txt"
    assert VaultFile(obj("r2", "casebible-raw", False)).locator == "r2://casebible-raw/k/file.txt"


def test_the_object_store_is_chosen_per_provider_and_bucket(monkeypatch):
    import casebible_index.vault_source as vs

    stores = {("b2", "salem-data"): "B2", ("r2", "casebible-raw"): "R2"}
    monkeypatch.setattr(vs.coco, "use_context", lambda key: stores)
    assert VaultFile(obj("r2", "casebible-raw", False)).store == "R2"
    assert VaultFile(obj("b2", "salem-data", True)).store == "B2"
    with pytest.raises(ValueError, match="No object store configured for r2:other"):
        VaultFile(obj("r2", "other", False)).store  # noqa: B018

"""The batched decode-state check behind the Sources state mark.

Byline: Claude Code · Opus 5 · 2026-09-22.
"""

import pytest

from app.service import proffer_decoded_exists as service
from app.service.proffer_errors import ProfferError


@pytest.fixture
def located(monkeypatch):
    def locate(source_ref):
        if "outside" in source_ref:
            raise ProfferError("this source is outside a configured source root", 403)
        return "b2", "salem-data", source_ref.rsplit("/", 1)[0] + "/derived/"

    monkeypatch.setattr(service, "_locate", locate)


def test_each_source_gets_its_own_answer(monkeypatch, located):
    monkeypatch.setattr(service, "object_exists", lambda scheme, bucket, key: "yes" in key)
    result = service.decoded_exists(["b2://salem-data/yes/a.xml", "b2://salem-data/no/b.xml"])
    assert [(item.source_ref.endswith("a.xml"), item.decoded) for item in result.items] == [(True, True), (False, False)]
    assert all(item.reason == "" for item in result.items)


def test_one_unreachable_source_does_not_blank_the_page(monkeypatch, located):
    monkeypatch.setattr(service, "object_exists", lambda scheme, bucket, key: True)
    result = service.decoded_exists(["b2://salem-data/outside/a.xml", "b2://salem-data/in/b.xml"])
    assert result.items[0].decoded is False and "configured source root" in result.items[0].reason
    assert result.items[1].decoded is True


def test_an_object_store_failure_is_reported_per_row(monkeypatch, located):
    def boom(scheme, bucket, key):
        raise RuntimeError("media head failed")

    monkeypatch.setattr(service, "object_exists", boom)
    item = service.decoded_exists(["b2://salem-data/in/a.xml"]).items[0]
    assert item.decoded is False and item.reason == "object store is unreachable"


def test_the_batch_is_bounded(located):
    with pytest.raises(ProfferError):
        service.decoded_exists([f"b2://salem-data/in/{index}.xml" for index in range(service.MAX_SOURCES + 1)])

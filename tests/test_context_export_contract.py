"""Synthetic D11 manifest gates; no case corpus or filesystem export."""

from dataclasses import replace
from datetime import UTC, datetime, timedelta

import pytest

from server.contracts.context_export import (
    ApprovedRoute,
    ContextExportManifest,
    NarrativeEntry,
    PackageFile,
    SourceRef,
    validate_manifest,
)

NOW = datetime(2026, 9, 23, tzinfo=UTC)
HASH = "a" * 64


def fixture():
    route = ApprovedRoute(
        "family_story", 3, "family_story", "matter1", "KnowledgeBase", "matter1/stories", "approval7", NOW
    )
    manifest = ContextExportManifest(
        1,
        "export1",
        1,
        None,
        "matter1",
        "owner1",
        "family_context",
        "generator1",
        "template1",
        "private1",
        "private",
        "family_story",
        NOW,
        "as_lived",
        NOW,
        "family_story",
        3,
        "KnowledgeBase/matter1/stories/export1/v1",
        ("entry1",),
        (SourceRef("source1", 2, "r2://bucket/key", HASH, "available", NOW),),
        (
            NarrativeEntry(
                "entry1",
                "record1",
                4,
                ("source1",),
                "context",
                "recollection",
                "ordinary",
                "circa 2020",
                "approximate",
                "event",
                "unreviewed",
                "not_promoted",
            ),
        ),
        (),
        (PackageFile("timeline.json", HASH),),
        ("timeline.json",),
        (),
    )
    return manifest, route


def test_context_before_evidence_and_deterministic_digest():
    manifest, route = fixture()
    assert validate_manifest(manifest, route) == validate_manifest(manifest, route)
    assert len(validate_manifest(manifest, route)) == 64
    assert manifest.release_state == "prepared_unreleased"
    hindsight = replace(manifest, time_mode="hindsight", as_lived_cutoff=None)
    assert validate_manifest(hindsight, route) != validate_manifest(manifest, route)


def test_exact_versions_time_mode_relations_and_digest_change():
    manifest, route = fixture()
    changed = replace(manifest, entries=(replace(manifest.entries[0], relation="contradicts"),))
    assert validate_manifest(changed, route) != validate_manifest(manifest, route)
    assert validate_manifest(replace(manifest, redaction_profile="private2"), route) != validate_manifest(
        manifest, route
    )
    for bad in (True, 0, "1"):
        with pytest.raises(ValueError):
            validate_manifest(replace(manifest, version=bad), route)
    with pytest.raises(ValueError):
        validate_manifest(replace(manifest, time_mode="hindsight"), route)
    with pytest.raises(ValueError):
        validate_manifest(replace(manifest, entries=(replace(manifest.entries[0], source_ids=("other",)),)), route)
    with pytest.raises(ValueError):
        validate_manifest(replace(manifest, requested_entry_ids=("entry1", "missing")), route)
    with pytest.raises(ValueError):
        validate_manifest(
            replace(manifest, sources=(replace(manifest.sources[0], source_available_from=NOW + timedelta(days=1)),)),
            route,
        )
    with pytest.raises(ValueError):
        validate_manifest(replace(manifest, sources=list(manifest.sources)), route)


def test_route_binding_and_no_overwrite_or_unapproved_root():
    manifest, route = fixture()
    for destination in (
        "KnowledgeBase/matter1/stories/export1/v2",
        "CaseManagement/matter1/stories/export1/v1",
        "KnowledgeBase/../Vault/export1/v1",
        "KnowledgeBase/matter1/stories/export1/v1/extra",
    ):
        with pytest.raises(ValueError):
            validate_manifest(replace(manifest, destination=destination), route)
    with pytest.raises(ValueError):
        validate_manifest(manifest, replace(route, route_map_version=4))
    with pytest.raises(ValueError):
        validate_manifest(manifest, replace(route, scope_id="other"))


def test_source_copy_is_explicit_and_offline_link_resolves():
    manifest, route = fixture()
    with pytest.raises(ValueError):
        validate_manifest(
            replace(manifest, files=manifest.files + (PackageFile("source.bin", HASH, "source1"),)), route
        )
    included = replace(
        manifest,
        requested_source_copies=("source1",),
        files=manifest.files + (PackageFile("source.bin", HASH, "source1"),),
        offline_links=("timeline.json", "source.bin"),
    )
    assert validate_manifest(included, route)
    with pytest.raises(ValueError):
        validate_manifest(replace(included, offline_links=("missing.bin",)), route)
    with pytest.raises(ValueError):
        validate_manifest(replace(included, offline_links=("../source.bin",)), route)
    with pytest.raises(ValueError):
        PackageFile("%2e%2e/source.bin", HASH)


def test_private_strategy_and_release_fail_closed():
    manifest, route = fixture()
    private = replace(manifest.entries[0], privacy="private_strategy")
    assert validate_manifest(replace(manifest, entries=(private,)), route)
    with pytest.raises(ValueError):
        validate_manifest(replace(manifest, audience="external", entries=(private,)), route)
    with pytest.raises(ValueError):
        validate_manifest(replace(manifest, release_state="released"), route)


def test_unavailable_source_visibly_excluded_not_copied():
    manifest, route = fixture()
    missing = replace(manifest.sources[0], status="revoked")
    with pytest.raises(ValueError):
        validate_manifest(replace(manifest, sources=(missing,)), route)
    excluded = replace(manifest, sources=(missing,), excluded_source_ids=("source1",))
    assert validate_manifest(excluded, route)
    with pytest.raises(ValueError):
        validate_manifest(replace(excluded, requested_source_copies=("source1",)), route)

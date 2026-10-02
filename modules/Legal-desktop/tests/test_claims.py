"""Behavioral proof for durable claim records, exact references and atomic actions."""

import sqlite3
from concurrent.futures import ThreadPoolExecutor
from datetime import UTC, datetime
from uuid import uuid4

import pytest
from fastapi import FastAPI
from fastapi.testclient import TestClient
from legal_workspace.contracts.claims import (
    ClaimCreate,
    ClaimPatch,
    FollowupCreate,
    FollowupPatch,
    FromProbataCreate,
    GapCreate,
    GapPatch,
    LinkCreate,
    LinkPatch,
    OriginReference,
)
from legal_workspace.contracts.source_package import (
    LegalSourcePackage,
    LegalSourcePackageItem,
    ReviewState,
)
from legal_workspace.services.claims import ClaimNotFound, ClaimRevisionConflict, ClaimService
from pydantic import ValidationError


@pytest.fixture
def context(tmp_path):
    matter_id = uuid4()
    package = LegalSourcePackage(
        package_id=uuid4(),
        matter_id=matter_id,
        manifest_hash="sha256:" + "a" * 64,
        created_at=datetime.now(UTC),
        items=[
            LegalSourcePackageItem(
                item_id=uuid4(),
                assertion_id=uuid4(),
                assertion_version=3,
                span_locator="message:17:characters:0-84",
                custody_locator="source:accepted:42",
                content_hash="sha256:" + "b" * 64,
                review_state=ReviewState.APPROVED,
            )
        ],
    )
    current = {"package": package}
    service = ClaimService(tmp_path, matter_id, lambda: current["package"])
    return service, current


def create(service, **kwargs):
    return service.create(
        ClaimCreate(text="The parenting exchange did not occur.", **kwargs), actor="owner"
    )


def link(service, claim, relationship="supports", **kwargs):
    return service.add_link(
        claim.claim_id,
        LinkCreate(
            expected_revision=claim.revision,
            relationship=relationship,
            source=service.sources().items[0],
            **kwargs,
        ),
        actor="owner",
    )


def test_durable_claim_response_actions_history_and_matter_isolation(context):
    service, current = context
    claim = create(service, kind="allegation", claimant="Other party", response="Disputed.")
    claim = service.add_gap(
        claim.claim_id,
        GapCreate(expected_revision=1, description="Find exchange messages."),
        actor="owner",
    )
    gap = claim.gaps[0]
    claim = service.add_followup(
        claim.claim_id,
        FollowupCreate(
            expected_revision=2,
            kind="investigate",
            description="Locate messages around exchange.",
            gap_id=gap.gap_id,
        ),
        actor="agent",
    )
    followup = claim.followups[0]
    claim = service.patch_followup(
        claim.claim_id,
        followup.followup_id,
        FollowupPatch(expected_revision=3, status="done"),
        actor="owner",
    )
    claim = service.patch_gap(
        claim.claim_id, gap.gap_id, GapPatch(expected_revision=4, status="resolved"), actor="owner"
    )
    recreated = ClaimService(service.db_path.parent, service.matter_id, lambda: current["package"])
    assert recreated.get(claim.claim_id) == claim
    assert recreated.list()[0].response == "Disputed."
    assert recreated.get(claim.claim_id).followups[0].status == "done"
    history = recreated.history(claim.claim_id)
    assert [row.revision for row in history] == [5, 4, 3, 2, 1]
    assert history[2].actor == "agent"
    assert history[-1].record.gaps == []
    other = ClaimService(service.db_path.parent, uuid4(), lambda: current["package"])
    assert other.list() == []
    assert not other.sources().available
    with pytest.raises(ClaimNotFound):
        other.get(claim.claim_id)


@pytest.mark.parametrize(
    "field,value",
    [
        ("package_id", uuid4()),
        ("item_id", uuid4()),
        ("assertion_id", uuid4()),
        ("assertion_version", 4),
        ("manifest_hash", "sha256:" + "c" * 64),
        ("content_hash", "sha256:" + "d" * 64),
        ("span_locator", "message:18:characters:0-84"),
        ("custody_locator", "source:42"),
    ],
)
def test_exact_link_fields_reject_invalid_reference_without_revision(context, field, value):
    service, _ = context
    claim = create(service)
    ref = service.sources().items[0].model_copy(update={field: value})
    with pytest.raises(ValueError, match="exact span"):
        service.add_link(
            claim.claim_id,
            LinkCreate(expected_revision=1, relationship="supports", source=ref),
            actor="owner",
        )
    assert service.get(claim.claim_id).revision == 1
    assert len(service.history(claim.claim_id)) == 1


def test_partial_contradiction_context_revocation_and_soft_deactivation(context):
    service, current = context
    claim = create(service)
    claim = service.add_link(
        claim.claim_id,
        LinkCreate(
            expected_revision=1,
            relationship="context",
            context_reference="catalog:42",
            note="Unreviewed lead.",
        ),
        actor="owner",
    )
    assert claim.evidence_status == "context_only"
    claim = link(service, claim, "partial")
    assert claim.evidence_status == "partially_supported"
    claim = link(service, claim, "contradicts")
    assert claim.evidence_status == "conflicting_evidence"
    claim = service.patch_link(
        claim.claim_id,
        claim.links[-1].link_id,
        LinkPatch(expected_revision=4, active=False),
        actor="owner",
    )
    assert claim.evidence_status == "partially_supported"
    assert len(claim.links) == 3
    current["package"].items[0].review_state = ReviewState.REVOKED
    assert service.get(claim.claim_id).evidence_status == "evidence_needed"
    assert service.get(claim.claim_id).links[1].validation_status == "unavailable"
    assert service.history(claim.claim_id)[1].record.evidence_status == "conflicting_evidence"
    with pytest.raises(ValueError):
        service.patch_link(
            claim.claim_id,
            claim.links[-1].link_id,
            LinkPatch(expected_revision=5, active=True),
            actor="owner",
        )


@pytest.mark.parametrize("invalid", ["missing", "wrong_matter", "not_permitted", "candidate"])
def test_package_eligibility(context, invalid):
    service, current = context
    ref = service.sources().items[0]
    claim = create(service)
    if invalid == "missing":
        current["package"] = None
    elif invalid == "wrong_matter":
        current["package"].matter_id = uuid4()
    elif invalid == "not_permitted":
        current["package"].items[0].permitted_use = "research_only"
    else:
        current["package"].items[0].review_state = ReviewState.CANDIDATE
    assert not service.sources().available
    with pytest.raises(ValueError):
        service.add_link(
            claim.claim_id,
            LinkCreate(expected_revision=1, relationship="supports", source=ref),
            actor="owner",
        )


def test_edited_claim_preserves_original_and_deactivates_old_relationships(context):
    service, _ = context
    claim = link(service, create(service))
    assert claim.evidence_status == "evidence_linked"
    claim = service.patch(
        claim.claim_id,
        ClaimPatch(expected_revision=2, text="The parenting exchange was rescheduled."),
        actor="owner",
    )
    assert claim.evidence_status == "evidence_needed"
    assert not claim.links[0].active
    assert service.history(claim.claim_id)[1].record.links[0].active
    assert service.history(claim.claim_id)[1].record.text == "The parenting exchange did not occur."


def test_one_claim_link_does_not_support_another_claim(context):
    service, _ = context
    linked = link(service, create(service))
    unlinked = service.create(ClaimCreate(text="A different event occurred."), actor="owner")
    assert service.get(linked.claim_id).evidence_status == "evidence_linked"
    assert service.get(unlinked.claim_id).evidence_status == "evidence_needed"
    assert service.get(unlinked.claim_id).links == []
    assert [row.claim_id for row in service.gap_report()] == [unlinked.claim_id]


def test_malformed_legacy_source_is_unavailable_and_gap_can_reopen(context):
    service, current = context
    current["package"].items[0].span_locator = ""
    assert not service.sources().available
    claim = create(service)
    claim = service.add_gap(
        claim.claim_id, GapCreate(expected_revision=1, description="Find source."), actor="owner"
    )
    claim = service.patch_gap(
        claim.claim_id,
        claim.gaps[0].gap_id,
        GapPatch(expected_revision=2, status="resolved"),
        actor="owner",
    )
    claim = service.patch_gap(
        claim.claim_id,
        claim.gaps[0].gap_id,
        GapPatch(expected_revision=3, status="open"),
        actor="owner",
    )
    assert service.gap_report()[0].gap_id == claim.gaps[0].gap_id


def test_concurrent_base_revision_has_one_winner(context):
    service, _ = context
    claim = create(service)

    def save(description):
        try:
            return service.add_gap(
                claim.claim_id,
                GapCreate(expected_revision=1, description=description),
                actor="owner",
            )
        except ClaimRevisionConflict as exc:
            return exc

    with ThreadPoolExecutor(max_workers=2) as pool:
        results = list(pool.map(save, ["Locate text", "Locate order"]))
    assert sum(isinstance(row, ClaimRevisionConflict) for row in results) == 1
    assert service.get(claim.claim_id).revision == 2
    assert len(service.get(claim.claim_id).gaps) == 1


def test_failed_revision_commit_rolls_back_claim_and_action(context):
    service, _ = context
    claim = create(service)
    with sqlite3.connect(service.db_path) as conn:
        conn.execute("""CREATE TRIGGER fail_claim_revision BEFORE INSERT ON legal_claim_revision
                        BEGIN SELECT RAISE(ABORT, 'simulated write failure'); END""")
    with pytest.raises(sqlite3.IntegrityError):
        service.add_followup(
            claim.claim_id,
            FollowupCreate(
                expected_revision=1, kind="research", description="Find relevant authority."
            ),
            actor="owner",
        )
    assert service.get(claim.claim_id).revision == 1
    assert service.get(claim.claim_id).followups == []
    assert len(service.history(claim.claim_id)) == 1


def test_followup_wrong_gap_missing_ids_and_gap_report(context):
    service, _ = context
    first = create(service)
    second = service.create(ClaimCreate(text="What happened?", kind="question"), actor="owner")
    second = service.add_gap(
        second.claim_id,
        GapCreate(expected_revision=1, description="Ask for context."),
        actor="owner",
    )
    with pytest.raises(ValueError, match="does not belong"):
        service.add_followup(
            first.claim_id,
            FollowupCreate(
                expected_revision=1,
                kind="discovery",
                description="Prepare a request.",
                gap_id=second.gaps[0].gap_id,
            ),
            actor="owner",
        )
    with pytest.raises(ClaimNotFound):
        service.patch_gap(
            first.claim_id, uuid4(), GapPatch(expected_revision=1, status="resolved"), actor="owner"
        )
    report = service.gap_report()
    assert len(report) == 2
    assert {row.description for row in report} == {"Evidence needed.", "Ask for context."}
    assert service.get(first.claim_id).revision == 1


def test_context_cannot_be_promoted_without_evidence_and_blank_fields_rejected():
    with pytest.raises(ValidationError):
        LinkCreate(expected_revision=1, relationship="supports", context_reference="catalog:42")
    with pytest.raises(ValidationError):
        ClaimCreate(text="   ")
    with pytest.raises(ValidationError):
        ClaimPatch(expected_revision=1, text=None)


def test_api_contract_auth_and_conflicts(context):
    from legal_workspace.api import claim_routes

    service, _ = context
    app = FastAPI()
    app.include_router(claim_routes.router)
    app.dependency_overrides[claim_routes.get_claim_service] = lambda: service
    with TestClient(app) as client:
        assert client.get("/v1/claims").status_code == 401
        app.dependency_overrides[claim_routes.actor] = lambda: "test:owner"
        response = client.post("/v1/claims", json={"text": "An allegation.", "kind": "allegation"})
        assert response.status_code == 201
        claim = response.json()
        result = client.post(
            f"/v1/claims/{claim['claim_id']}/gaps",
            json={"expected_revision": 1, "description": "Find evidence."},
        )
        assert result.status_code == 200
        stale = client.patch(
            f"/v1/claims/{claim['claim_id']}",
            json={"expected_revision": 1, "response": "Disputed."},
        )
        assert stale.status_code == 409
        assert stale.json()["detail"]["current_revision"] == 2
        assert len(client.get(f"/v1/claims/{claim['claim_id']}/history").json()) == 2
        assert client.get("/v1/claim-sources").json()["available"]
        assert client.get("/v1/claim-gaps").json()[0]["description"] == "Find evidence."


@pytest.fixture
def origin_context(tmp_path):
    origin = OriginReference(kind="event", record_id="event:42", record_version="v3")
    descriptor = {
        "origin": origin.model_dump(),
        "title": "Synthetic source event title",
        "record": {"when": "2026-01-01", "source_field": "Synthetic original payload"},
    }
    current = {"descriptor": descriptor}
    service = ClaimService(
        tmp_path, uuid4(), lambda: None, record_loader=lambda kind, record_id: current["descriptor"]
    )
    return service, origin, current


def test_probata_overlay_idempotence_version_change_and_no_source_copy(origin_context):
    service, origin, current = origin_context
    claim = service.from_probata(
        FromProbataCreate(origin=origin, response="Review exchange details."), actor="owner"
    )
    assert claim.origin == origin
    assert claim.origin_state == "unchanged"
    assert claim.origin_record.payload["source_field"] == "Synthetic original payload"
    assert claim.text == "Legal review of this event"
    assert claim.evidence_status == "evidence_needed"
    retry = service.from_probata(
        FromProbataCreate(origin=origin, response="Do not replace response"), actor="owner"
    )
    assert retry.claim_id == claim.claim_id
    assert retry.response == "Review exchange details."
    assert retry.revision == 1
    current["descriptor"]["origin"]["record_version"] = "v4"
    newer = OriginReference(**current["descriptor"]["origin"])
    opened = service.from_probata(FromProbataCreate(origin=newer), actor="owner")
    assert opened.claim_id == claim.claim_id
    assert opened.origin.record_version == "v3"
    assert opened.origin_state == "changed"
    assert service.get(claim.claim_id).origin_state == "changed"
    updated = service.patch(
        claim.claim_id,
        ClaimPatch(expected_revision=1, response="Legal overlay revised."),
        actor="owner",
    )
    assert updated.origin_record.title == "Synthetic source event title"
    with sqlite3.connect(service.db_path) as conn:
        persisted = conn.execute("SELECT record_json FROM legal_claim").fetchone()[0]
        history = conn.execute("SELECT record_json FROM legal_claim_revision").fetchall()
    for raw in [persisted, *(row[0] for row in history)]:
        assert "Synthetic source event title" not in raw
        assert "Synthetic original payload" not in raw
        assert "origin_record" not in raw
        assert "origin_state" not in raw
    assert service.history(claim.claim_id)[0].record.origin_record is None
    current["descriptor"] = None
    assert service.get(claim.claim_id).origin_state == "unavailable"
    assert service.get(claim.claim_id).origin_record is None


def test_remote_source_refresh_does_not_hold_writer_lock(origin_context):
    service, origin, current = origin_context
    claim = service.from_probata(FromProbataCreate(origin=origin), actor="owner")

    def read_source(kind, record_id):
        with sqlite3.connect(service.db_path, timeout=0.01) as connection:
            connection.execute("BEGIN IMMEDIATE")
        return current["descriptor"]

    service.record_loader = read_source
    updated = service.patch(
        claim.claim_id, ClaimPatch(expected_revision=1, response="Preserved response"), actor="owner"
    )
    assert updated.origin_state == "unchanged"
    assert updated.response == "Preserved response"


def test_origin_lookup_is_scoped_read_only_and_survives_source_outage(origin_context):
    service, origin, current = origin_context
    assert service.by_origin(origin.kind, origin.record_id) is None
    assert service.list() == []
    saved = service.from_probata(FromProbataCreate(origin=origin, response="Saved legal response"), actor="owner")
    current["descriptor"] = None
    found = service.by_origin(origin.kind, origin.record_id)
    assert found.claim_id == saved.claim_id
    assert found.response == "Saved legal response" and found.origin_state == "unavailable"
    assert len(service.history(saved.claim_id)) == 1
    assert service.by_origin("entity", origin.record_id) is None
    other = ClaimService(service.db_path.parent, uuid4(), lambda: None)
    assert other.by_origin(origin.kind, origin.record_id) is None


def test_origin_lookup_http_auth_identity_and_no_mutation(origin_context):
    from legal_workspace.api import claim_routes

    service, origin, current = origin_context
    identity = str(uuid4())
    native = OriginReference(kind="event", record_id=identity, record_version="synthetic-v1")
    current["descriptor"]["origin"] = native.model_dump()
    app = FastAPI()
    app.include_router(claim_routes.router)
    app.dependency_overrides[claim_routes.get_claim_service] = lambda: service
    path = f"/v1/claims/by-origin?kind=event&record_id={identity}"
    with TestClient(app) as client:
        assert client.get(path).status_code == 401
        app.dependency_overrides[claim_routes.actor] = lambda: "test:owner"
        assert client.get(path).json() is None
        assert service.list() == []
        saved = service.from_probata(FromProbataCreate(origin=native), actor="owner")
        assert client.get(path).json()["claim_id"] == str(saved.claim_id)
        assert client.get(path.replace("kind=event", "kind=other")).status_code == 422
        assert client.get(path.replace(identity, "../../different")).status_code == 422
        assert len(service.history(saved.claim_id)) == 1


@pytest.mark.parametrize("problem", ["type", "id", "version", "missing"])
def test_probata_overlay_requires_real_matching_record_before_creation(origin_context, problem):
    service, origin, current = origin_context
    if problem == "type":
        current["descriptor"]["origin"]["kind"] = "entity"
    elif problem == "id":
        current["descriptor"]["origin"]["record_id"] = "event:wrong"
    elif problem == "version":
        current["descriptor"]["origin"]["record_version"] = "v4"
    else:
        current["descriptor"] = None
    with pytest.raises(ValueError):
        service.from_probata(FromProbataCreate(origin=origin), actor="owner")
    assert service.list() == []


def test_concurrent_probata_open_creates_single_overlay(origin_context):
    service, origin, _ = origin_context
    with ThreadPoolExecutor(max_workers=2) as pool:
        records = list(
            pool.map(
                lambda _: service.from_probata(FromProbataCreate(origin=origin), actor="owner"),
                [1, 2],
            )
        )
    assert records[0].claim_id == records[1].claim_id
    assert len(service.list()) == 1
    assert len(service.history(records[0].claim_id)) == 1


def test_probata_api_and_pointer_bounds(origin_context):
    from legal_workspace.api import claim_routes

    service, origin, _ = origin_context
    app = FastAPI()
    app.include_router(claim_routes.router)
    app.dependency_overrides[claim_routes.get_claim_service] = lambda: service
    app.dependency_overrides[claim_routes.actor] = lambda: "test:owner"
    with TestClient(app) as client:
        response = client.post("/v1/claims/from-probata", json={"origin": origin.model_dump()})
        assert response.status_code == 200
        assert response.json()["origin_state"] == "unchanged"
        assert (
            response.json()["origin_record"]["payload"]["source_field"]
            == "Synthetic original payload"
        )
        blocked = client.post(
            "/v1/claims", json={"origin": origin.model_dump(), "text": "Do not copy source"}
        )
        assert blocked.status_code == 400
    with pytest.raises(ValidationError):
        OriginReference(kind="event", record_id="a" * 251, record_version="v1")
    with pytest.raises(ValidationError):
        OriginReference(kind="event", record_id="line\nbreak", record_version="v1")

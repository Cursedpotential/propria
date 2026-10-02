"""Synthetic HTTP and aggregate proof for explicit investigation dispatch."""

import json
from datetime import UTC, datetime
from types import SimpleNamespace
from uuid import uuid4

import httpx
import pytest
from fastapi import FastAPI
from fastapi.testclient import TestClient
from legal_workspace.api import claim_routes
from legal_workspace.api.auth import AuthenticatedPrincipal
from legal_workspace.contracts.claims import (
    ClaimCreate,
    ClaimPatch,
    FollowupCreate,
    FollowupPatch,
    InvestigationResponse,
)
from legal_workspace.services import probata_investigations as transport
from legal_workspace.services.claims import ClaimService
from legal_workspace.services.probata_records import Listing


@pytest.fixture
def setup(tmp_path, monkeypatch):
    secret = tmp_path / "service-token"
    secret.write_text("synthetic-investigation-service-token-1234")
    settings = SimpleNamespace(
        probata_records_base_url="https://probata.invalid",
        probata_records_token_file=str(secret),
        authentik_review_groups="advocatio-users",
    )
    monkeypatch.setattr(transport, "get_settings", lambda: settings)
    from legal_workspace.services import probata_records

    monkeypatch.setattr(probata_records, "get_settings", lambda: settings)
    from legal_workspace.api import auth

    monkeypatch.setattr(auth, "get_settings", lambda: settings)
    scope = Listing(available=True, matter_id=str(uuid4()), court_case_id=str(uuid4()))
    service = ClaimService(
        tmp_path, uuid4(), lambda: None, investigation_scope_loader=lambda _: scope
    )
    claim = service.create(ClaimCreate(text="Synthetic disputed statement"), actor="owner")
    claim = service.add_followup(
        claim.claim_id,
        FollowupCreate(
            expected_revision=claim.revision, kind="investigate", description="Find corroboration"
        ),
        actor="owner",
    )
    return service, claim, scope


def response(request, request_id):
    now = datetime.now(UTC).isoformat()
    return {
        **request,
        "request_id": str(request_id),
        "status": "received",
        "created_at": now,
        "updated_at": now,
        "results": [],
    }


def send(service, claim, revision=None):
    return service.dispatch_followup(
        claim.claim_id,
        claim.followups[0].followup_id,
        revision or claim.revision,
        actor="authentik:verified-owner",
        actor_uid="verified-owner",
        actor_username="owner",
    )


def test_timeout_frozen_retry_after_revision_move_and_no_duplicate_ack(setup):
    service, claim, scope = setup
    calls = []
    request_id = uuid4()
    acknowledged = None

    def mock(request):
        nonlocal acknowledged
        calls.append(
            (
                request.content,
                request.headers["idempotency-key"],
                request.headers["x-authentik-uid"],
            )
        )
        if len(calls) == 1:
            raise httpx.ReadTimeout("synthetic timeout", request=request)
        if acknowledged is None:
            acknowledged = response(json.loads(request.content), request_id)
        return httpx.Response(200, json=acknowledged)

    with httpx.Client(transport=httpx.MockTransport(mock)) as client:
        service.investigation_exchange = lambda saved, write: transport.exchange(
            saved, write=write, client=client
        )
        pending = send(service, claim)
        assert pending.followups[0].investigation.state == "prepared"
        assert pending.followups[0].investigation.last_error
        assert pending.followups[0].investigation.request.matter_id != service.matter_id
        assert str(pending.followups[0].investigation.request.matter_id) == scope.matter_id
        with pytest.raises(ValueError, match="pending dispatch"):
            service.patch_followup(
                claim.claim_id,
                claim.followups[0].followup_id,
                FollowupPatch(expected_revision=pending.revision, status="done"),
                actor="owner",
            )
        edited = service.patch(
            claim.claim_id,
            ClaimPatch(expected_revision=pending.revision, text="Changed local wording"),
            actor="owner",
        )
        confirmed = send(service, claim, edited.revision)
        saved = confirmed.followups[0].investigation
        assert saved.request_id == request_id and saved.remote_status == "received"
        assert confirmed.followups[0].status == "open"
        assert calls[0] == calls[1]
        count = len(service.history(claim.claim_id))
        again = send(service, confirmed)
        assert (
            again.revision == confirmed.revision and len(service.history(claim.claim_id)) == count
        )
        assert len(calls) == 2
    reloaded = ClaimService(service.db_path.parent, service.matter_id, lambda: None).get(
        claim.claim_id
    )
    assert reloaded.followups[0].investigation.request_id == request_id


@pytest.mark.parametrize("changed", ["matter_id", "court_case_id", "claim_id", "status", "results"])
def test_wrong_remote_identity_never_acknowledged(setup, changed):
    service, claim, _scope = setup

    def mock(request):
        data = response(json.loads(request.content), uuid4())
        data[changed] = (
            [{"evidence_bytes": "unexpected bytes"}]
            if changed == "results"
            else "mystery"
            if changed == "status"
            else str(uuid4())
        )
        return httpx.Response(201, json=data)

    with httpx.Client(transport=httpx.MockTransport(mock)) as client:
        service.investigation_exchange = lambda saved, write: transport.exchange(
            saved, write=write, client=client
        )
        current = send(service, claim)
    saved = current.followups[0].investigation
    assert saved.state == "prepared" and saved.request_id is None and saved.last_error


def test_status_outage_keeps_ack_and_refresh_never_posts(setup):
    service, claim, scope = setup
    request_id = uuid4()
    methods = []

    def mock(request):
        methods.append(request.method)
        if request.method == "POST":
            return httpx.Response(201, json=response(json.loads(request.content), request_id))
        assert request.url.params["matter_id"] == scope.matter_id
        assert request.headers["x-authentik-uid"] == "verified-owner"
        return httpx.Response(503)

    with httpx.Client(transport=httpx.MockTransport(mock)) as client:
        service.investigation_exchange = lambda saved, write: transport.exchange(
            saved, write=write, client=client
        )
        current = send(service, claim)
        current = service.refresh_followup(
            claim.claim_id, claim.followups[0].followup_id, current.revision, actor="other-owner"
        )
    assert methods == ["POST", "GET"]
    saved = current.followups[0].investigation
    assert saved.request_id == request_id and saved.remote_status == "received" and saved.last_error


def test_unavailable_scope_and_noninvestigation_never_send(setup):
    service, claim, _ = setup
    service.investigation_scope_loader = lambda _: Listing(available=False)
    with pytest.raises(ValueError, match="scope"):
        send(service, claim)
    assert service.get(claim.claim_id).followups[0].investigation is None
    current = service.add_followup(
        claim.claim_id,
        FollowupCreate(expected_revision=claim.revision, kind="research", description="Research"),
        actor="owner",
    )
    with pytest.raises(ValueError, match="open investigation"):
        service.dispatch_followup(
            claim.claim_id,
            current.followups[-1].followup_id,
            current.revision,
            actor="owner",
            actor_uid="owner",
            actor_username="owner",
        )


@pytest.mark.parametrize("planning_source", ["authentik", "signed-bff", "tailnet"])
def test_route_requires_planning_principal_and_blocks_body_identity(setup, planning_source):
    service, claim, _ = setup
    principal = [
        AuthenticatedPrincipal(
            subject="service", username=None, email=None, groups=(), source="mcp-gateway"
        )
    ]
    app = FastAPI()

    @app.middleware("http")
    async def identity(request, call_next):
        request.state.auth = principal[0]
        return await call_next(request)

    app.include_router(claim_routes.router)
    app.dependency_overrides[claim_routes.get_claim_service] = lambda: service
    service.investigation_exchange = lambda saved, write: InvestigationResponse.model_validate(
        response(saved.request.model_dump(mode="json"), uuid4())
    )
    path = f"/v1/claims/{claim.claim_id}/followups/{claim.followups[0].followup_id}/dispatch"
    with TestClient(app) as client:
        assert client.post(path, json={"expected_revision": claim.revision}).status_code == 403
        principal[0] = AuthenticatedPrincipal(
            subject="verified-human",
            username=None,
            email=None,
            groups=(),
            source=planning_source,
        )
        assert (
            client.post(
                path, json={"expected_revision": claim.revision, "actor_uid": "spoof"}
            ).status_code
            == 422
        )
        result = client.post(path, json={"expected_revision": claim.revision})
        assert result.status_code == 200
        saved = result.json()["followups"][0]["investigation"]
        assert saved["actor_uid"] == saved["actor_username"] == f"{planning_source}:verified-human"


def test_ack_merges_concurrent_owner_edit_and_native_source(setup):
    from legal_workspace.contracts.claims import FromProbataCreate, OriginReference
    from legal_workspace.services.probata_records import Record

    service, _, scope = setup
    native_id = str(uuid4())
    source = OriginReference(kind="entity", record_id=native_id, record_version="version:1")
    descriptor = {"origin": source.model_dump(), "title": "Synthetic native record", "record": {}}
    service.record_loader = lambda kind, record_id: descriptor
    scope.records = [Record.model_validate(descriptor)]
    claim = service.from_probata(FromProbataCreate(origin=source), actor="owner")
    claim = service.add_followup(
        claim.claim_id,
        FollowupCreate(
            expected_revision=claim.revision, kind="investigate", description="Inspect source"
        ),
        actor="owner",
    )

    def mock(request):
        payload = json.loads(request.content)
        assert payload["sources"] == [
            {"kind": "entity", "record_id": native_id, "record_version": "version:1"}
        ]
        current = service.get(claim.claim_id)
        service.patch(
            claim.claim_id,
            ClaimPatch(expected_revision=current.revision, response="Owner edit during HTTP"),
            actor="owner",
        )
        return httpx.Response(201, json=response(payload, uuid4()))

    with httpx.Client(transport=httpx.MockTransport(mock)) as client:
        service.investigation_exchange = lambda saved, write: transport.exchange(
            saved, write=write, client=client
        )
        acknowledged = send(service, claim)
    assert acknowledged.response == "Owner edit during HTTP"
    assert acknowledged.followups[0].investigation.remote_status == "received"


def test_changed_native_source_blocks_preparation(setup):
    from legal_workspace.contracts.claims import FromProbataCreate, OriginReference
    from legal_workspace.services.probata_records import Record

    service, _, scope = setup
    source = OriginReference(kind="event", record_id=str(uuid4()), record_version="version:1")
    descriptor = {"origin": source.model_dump(), "title": "Synthetic event", "record": {}}
    service.record_loader = lambda kind, record_id: descriptor
    claim = service.from_probata(FromProbataCreate(origin=source), actor="owner")
    claim = service.add_followup(
        claim.claim_id,
        FollowupCreate(
            expected_revision=claim.revision, kind="investigate", description="Find corroboration"
        ),
        actor="owner",
    )
    with pytest.raises(ValueError, match="source is unavailable"):
        send(service, claim)
    descriptor["origin"]["record_version"] = "version:2"
    scope.records = [Record.model_validate(descriptor)]
    with pytest.raises(ValueError, match="source is unavailable"):
        send(service, claim)
    assert service.get(claim.claim_id).followups[0].investigation is None


@pytest.mark.parametrize("source", ["mcp-gateway", "office-session", "explicit-test-bypass"])
def test_dispatch_actor_rejects_transport_and_service_principals(source):
    from legal_workspace.api.auth import PrincipalAuthorizationDenied, require_investigation_actor

    principal = AuthenticatedPrincipal(
        subject="device-or-service",
        username="spoofed-name",
        email=None,
        groups=("advocatio-users",),
        source=source,
    )
    with pytest.raises(PrincipalAuthorizationDenied):
        require_investigation_actor(principal)


def test_dispatch_eligibility_does_not_change_substantive_review_gate():
    from legal_workspace.api.auth import (
        PrincipalAuthorizationDenied,
        require_human_review_actor,
        require_investigation_actor,
    )

    principal = AuthenticatedPrincipal(
        subject="verified-person", username=None, email=None, groups=(), source="authentik"
    )
    assert require_investigation_actor(principal) == "authentik:verified-person"
    with pytest.raises(PrincipalAuthorizationDenied):
        require_human_review_actor(
            principal, settings=SimpleNamespace(authentik_review_groups="reviewers")
        )


@pytest.mark.parametrize("value", ["a" * 201, "actor\nheader", ""])
def test_dispatch_identity_bounds(value):
    from legal_workspace.api.auth import PrincipalAuthorizationDenied, require_investigation_actor

    with pytest.raises(PrincipalAuthorizationDenied):
        require_investigation_actor(
            AuthenticatedPrincipal(
                subject=value, username=None, email=None, groups=(), source="authentik"
            )
        )


def test_native_source_exact_lookup_when_listing_truncated(setup):
    from legal_workspace.contracts.claims import FromProbataCreate, OriginReference

    service, _, scope = setup
    native_id = str(uuid4())
    source = OriginReference(kind="entity", record_id=native_id, record_version="version:1")
    calls = []

    def loader(kind, record_id):
        calls.append((kind, record_id))
        return {"origin": source.model_dump(), "title": "Synthetic lookup", "record": {}}

    service.record_loader = loader
    claim = service.from_probata(FromProbataCreate(origin=source), actor="owner")
    claim = service.add_followup(
        claim.claim_id,
        FollowupCreate(
            expected_revision=claim.revision, kind="investigate", description="Inspect exact source"
        ),
        actor="owner",
    )
    scope.truncated = True
    service.investigation_exchange = lambda saved, write: InvestigationResponse.model_validate(
        response(saved.request.model_dump(mode="json"), uuid4())
    )
    saved = send(service, claim).followups[0].investigation
    assert (
        saved.request.sources[0].record_id == source.record_id
        or str(saved.request.sources[0].record_id) == source.record_id
    )
    assert ("entity", native_id) in calls


@pytest.mark.parametrize(
    "source,subject", [("signed-bff", "legal-web-bff"), ("tailnet", "tailnet-device")]
)
def test_planning_transport_principal_remains_truthful_and_cannot_review(source, subject):
    from legal_workspace.api.auth import (
        PrincipalAuthorizationDenied,
        require_human_review_actor,
        require_investigation_actor,
    )

    principal = AuthenticatedPrincipal(
        subject=subject, username=None, email=None, groups=(), source=source
    )
    assert require_investigation_actor(principal) == f"{source}:{subject}"
    with pytest.raises(PrincipalAuthorizationDenied):
        require_human_review_actor(
            principal, settings=SimpleNamespace(authentik_review_groups="reviewers")
        )

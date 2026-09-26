"""FastAPI entry. Domain mutations stay on versioned HTTP, not Next actions.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from __future__ import annotations

from contextlib import asynccontextmanager
from datetime import UTC, datetime
from typing import Any
from uuid import UUID

from fastapi import FastAPI, File, Form, HTTPException, Request, UploadFile
from pydantic import BaseModel

from fastmcp.utilities.lifespan import combine_lifespans

from legal_workspace import __version__
from legal_workspace.api.auth import (
    AuthenticatedPrincipal,
    LegalWorkspaceAuthMiddleware,
    PrincipalAuthorizationDenied,
)
from legal_workspace.api.mcp_face import mcp_app
from legal_workspace.config import get_settings
from legal_workspace.contracts.citations import AuthorityCitation, EvidenceCitation
from legal_workspace.contracts.events import EventEnvelope
from legal_workspace.contracts.identity import CourtCaseRef, MatterRef
from legal_workspace.contracts.source_package import LegalSourcePackage
from legal_workspace.domain.agents import AgentRun, AgentRunCreate
from legal_workspace.domain.authority_library import CuratedAuthority
from legal_workspace.domain.calendar import DocketEvent, DocketEventCreate
from legal_workspace.domain.discovery import (
    DiscoveryCreate,
    DiscoveryRequest,
    DiscoveryStatusUpdate,
)
from legal_workspace.domain.exhibits import ExhibitAnnotationCreate, ExhibitCandidate
from legal_workspace.domain.factors import (
    FactorCitationLink,
    FactorEntry,
    FactorLetter,
)
from legal_workspace.domain.filing import FilingOverride, FilingOverrideCreate, FilingReadiness
from legal_workspace.domain.investigation import InvestigationCreate, InvestigationRequest
from legal_workspace.domain.issue import (
    LegalIssue,
)
from legal_workspace.domain.privilege import PrivilegeScan, PrivilegeScanRequest, scan_text
from legal_workspace.domain.redteam import RedTeamCreate, RedTeamRun
from legal_workspace.domain.release import ReleaseBuild, ReleaseCreate, ReleaseManifest
from legal_workspace.domain.research import (
    CurrencyFlag,
    ResearchCreate,
    ResearchQuestion,
    ResearchStatus,
)
from legal_workspace.domain.review import ReviewCreate, ReviewDecision
from legal_workspace.domain.strategy import StrategyCreate, StrategyNote, StrategyPatch
from legal_workspace.domain.support_map import SupportMap, build_support_map
from legal_workspace.domain.templates import (
    DraftingTemplate,
    TemplateInstantiate,
    catalog as template_catalog,
)
from legal_workspace.domain.todos import CaseTodo, TodoCreate, TodoPatch, TodoStatus
from legal_workspace.domain.work_product import WorkProductVersion
from legal_workspace.services.agno_client import read_status
from legal_workspace.services.bates import BatesStampResult, stamp_bates_pdf
from legal_workspace.services.citation_gate import (
    validate_authority_citations,
    validate_factual_citations,
)
from legal_workspace.services.redaction import RedactionResult, redact_content_stream
from legal_workspace.services.revocation import apply_evidence_event
from legal_workspace.services.source_package import import_legal_source_package
from legal_workspace.services.workspace import WORKSPACE, DraftSection


@asynccontextmanager
async def _lifespan(_app: FastAPI):
    from legal_workspace.services.automation.scheduler import shutdown_scheduler, start_scheduler

    try:
        start_scheduler()
    except Exception:
        pass
    yield
    try:
        shutdown_scheduler()
    except Exception:
        pass


settings = get_settings()

app = FastAPI(
    title="Legal Workspace API",
    version=__version__,
    description="Legal practice sibling of the Evidence Platform. Not a second evidence store.",
    lifespan=combine_lifespans(_lifespan, mcp_app.lifespan),
)
app.add_middleware(LegalWorkspaceAuthMiddleware)


class HealthResponse(BaseModel):
    status: str
    service: str
    version: str
    evidence_platform: str
    model_gateway: str
    timezone: str
    docker: str = "not-required-locally"


class AuthIdentityResponse(BaseModel):
    subject: str
    username: str | None
    email: str | None
    groups: list[str]
    source: str


@app.get("/v1/auth/whoami", response_model=AuthIdentityResponse)
def auth_whoami(request: Request) -> AuthIdentityResponse:
    """Return only the validated, minimal server-side identity projection."""

    principal: AuthenticatedPrincipal = request.state.auth
    return AuthIdentityResponse(
        subject=principal.subject,
        username=principal.username,
        email=principal.email,
        groups=list(principal.groups),
        source=principal.source,
    )


class AgnoStatusResponse(BaseModel):
    reachable: bool
    evidence_platform: str
    matters_visible: bool


class ImportResponse(BaseModel):
    blocked: bool
    accepted_item_count: int
    omitted_item_ids: list[str]
    reason: str | None = None


class GateRequest(BaseModel):
    package: LegalSourcePackage
    facts: list[EvidenceCitation] = []
    authorities: list[AuthorityCitation] = []


class GateResponse(BaseModel):
    ok: bool
    blockers: list[str]


class RevocationRequest(BaseModel):
    event: EventEnvelope
    products: list[WorkProductVersion]


class RevocationResponse(BaseModel):
    products: list[WorkProductVersion]


class SurfaceLink(BaseModel):
    path: str
    label: str
    help: str = ""


class HomeResponse(BaseModel):
    matter: MatterRef
    court_case: CourtCaseRef
    package_imported: bool
    accepted_item_count: int
    omitted_item_ids: list[str]
    issue: LegalIssue
    factor_count: int
    draft_count: int
    authority_count: int
    strategy_count: int
    redteam_count: int
    open_todo_count: int
    review_count: int
    release_count: int
    open_research_count: int
    discovery_count: int
    investigation_count: int
    upcoming_event_count: int
    upcoming_events: list[DocketEvent]
    agent_run_count: int
    exhibit_count: int
    audit_count: int
    judge_confirmed: bool
    foc_confirmed: bool
    next_surfaces: list[SurfaceLink]


def _surface_links() -> list[SurfaceLink]:
    from legal_workspace.services.routing import public_routing

    return [
        SurfaceLink(
            path=str(item["path"]),
            label=str(item["label"]),
            help=str(item.get("help") or ""),
        )
        for item in public_routing()["surfaces"]
    ]


class DraftRequest(BaseModel):
    letter: FactorLetter
    heading: str
    body: str


class DraftEditRequest(BaseModel):
    heading: str
    body: str


class DraftView(BaseModel):
    section_id: UUID
    heading: str
    body: str
    factor_letter: FactorLetter | None
    citation_count: int
    unsupported: bool = False
    support: SupportMap | None = None


def _draft_view(section: DraftSection) -> DraftView:
    unsupported = False
    citations_ok = False
    state = WORKSPACE.load()
    if state.package is not None:
        gate = validate_factual_citations(section.citations, state.package)
        unsupported = not gate.ok
        citations_ok = gate.ok
    elif section.citations:
        unsupported = True
    support = build_support_map(
        section_id=str(section.section_id),
        heading=section.heading,
        body=section.body,
        citation_count=len(section.citations),
        citations_ok=citations_ok,
    )
    return DraftView(
        section_id=section.section_id,
        heading=section.heading,
        body=section.body,
        factor_letter=section.factor_letter,
        citation_count=len(section.citations),
        unsupported=unsupported,
        support=support,
    )


@app.get("/health", response_model=HealthResponse)
def health() -> HealthResponse:
    return HealthResponse(
        status="ok",
        service=settings.legal_api_service,
        version=__version__,
        evidence_platform=settings.evidence_platform_service,
        model_gateway=settings.model_gateway_service,
        timezone=settings.display_timezone,
    )


@app.get("/v1/agno/status", response_model=AgnoStatusResponse)
def agno_status() -> AgnoStatusResponse:
    """Browser talks to legal-api. legal-api probes Agno by service name."""
    snap = read_status()
    return AgnoStatusResponse(
        reachable=snap.reachable,
        evidence_platform=snap.evidence_platform,
        matters_visible=snap.matters_visible,
    )


@app.get("/v1/matter", response_model=HomeResponse)
def matter_home() -> HomeResponse:
    state = WORKSPACE.home()
    return HomeResponse(
        matter=state.matter,
        court_case=state.court_case,
        package_imported=state.package is not None,
        accepted_item_count=len(state.package.items) if state.package else 0,
        omitted_item_ids=list(state.omitted_item_ids),
        issue=state.issue,
        factor_count=len(state.factors),
        draft_count=len(state.drafts),
        authority_count=len(state.authorities),
        strategy_count=len(state.strategy_notes),
        redteam_count=len(state.redteam_runs),
        open_todo_count=sum(1 for item in state.todos if item.status == TodoStatus.OPEN),
        review_count=len(state.reviews),
        release_count=len(state.releases),
        open_research_count=sum(
            1
            for item in state.research_questions
            if item.status in {ResearchStatus.OPEN, ResearchStatus.UNRESOLVED}
        ),
        discovery_count=len(state.discovery),
        investigation_count=len(state.investigations),
        upcoming_event_count=WORKSPACE.upcoming_event_count(),
        audit_count=WORKSPACE.audit_count(),
        upcoming_events=WORKSPACE.upcoming_events(),
        agent_run_count=len(state.agent_runs),
        exhibit_count=len(WORKSPACE.list_exhibits()),
        judge_confirmed=bool(state.court_case.judge),
        foc_confirmed=bool(state.court_case.foc),
        next_surfaces=_surface_links(),
    )


@app.get("/v1/authorities", response_model=list[CuratedAuthority])
def list_authorities() -> list[CuratedAuthority]:
    return WORKSPACE.load().authorities


@app.get("/v1/research", response_model=list[ResearchQuestion])
def list_research() -> list[ResearchQuestion]:
    return WORKSPACE.load().research_questions


@app.post("/v1/research", response_model=ResearchQuestion)
def create_research(body: ResearchCreate) -> ResearchQuestion:
    return WORKSPACE.add_research_question(body)


@app.post("/v1/research/{question_id}:status", response_model=ResearchQuestion)
def update_research(question_id: UUID, status: ResearchStatus) -> ResearchQuestion:
    try:
        return WORKSPACE.set_research_status(question_id, status)
    except StopIteration as exc:
        raise HTTPException(status_code=404, detail="unknown research question") from exc


class CurrencyRequest(BaseModel):
    identifier: str
    note: str


@app.get("/v1/currency-flags", response_model=list[CurrencyFlag])
def list_currency_flags() -> list[CurrencyFlag]:
    return WORKSPACE.load().currency_flags


@app.post("/v1/currency-flags", response_model=CurrencyFlag)
def create_currency_flag(body: CurrencyRequest) -> CurrencyFlag:
    try:
        return WORKSPACE.flag_authority_currency(body.identifier, body.note)
    except ValueError as exc:
        raise HTTPException(status_code=409, detail=str(exc)) from exc


@app.get("/v1/strategy", response_model=list[StrategyNote])
def list_strategy() -> list[StrategyNote]:
    return WORKSPACE.list_strategy_notes()


@app.post("/v1/strategy", response_model=StrategyNote)
def create_strategy(body: StrategyCreate) -> StrategyNote:
    return WORKSPACE.add_strategy_note(body)


@app.put("/v1/strategy/{note_id}", response_model=StrategyNote)
def patch_strategy(note_id: UUID, body: StrategyPatch) -> StrategyNote:
    try:
        return WORKSPACE.patch_strategy_note(note_id, body)
    except StopIteration as exc:
        raise HTTPException(status_code=404, detail="unknown strategy note") from exc


@app.get("/v1/redteam", response_model=list[RedTeamRun])
def list_redteam() -> list[RedTeamRun]:
    return WORKSPACE.load().redteam_runs


@app.post("/v1/redteam", response_model=RedTeamRun)
def create_redteam(body: RedTeamCreate) -> RedTeamRun:
    return WORKSPACE.add_redteam_run(body)


@app.get("/v1/todos", response_model=list[CaseTodo])
def list_todos() -> list[CaseTodo]:
    return WORKSPACE.load().todos


@app.post("/v1/todos", response_model=CaseTodo)
def create_todo(body: TodoCreate) -> CaseTodo:
    return WORKSPACE.add_todo(body)


@app.post("/v1/todos/{todo_id}:status", response_model=CaseTodo)
def update_todo(todo_id: UUID, status: TodoStatus) -> CaseTodo:
    try:
        return WORKSPACE.set_todo_status(todo_id, status)
    except StopIteration as exc:
        raise HTTPException(status_code=404, detail="unknown todo") from exc


@app.put("/v1/todos/{todo_id}", response_model=CaseTodo)
def patch_todo(todo_id: UUID, body: TodoPatch) -> CaseTodo:
    try:
        return WORKSPACE.patch_todo(todo_id, body)
    except StopIteration as exc:
        raise HTTPException(status_code=404, detail="unknown todo") from exc


@app.get("/v1/reviews", response_model=list[ReviewDecision])
def list_reviews() -> list[ReviewDecision]:
    return WORKSPACE.load().reviews


@app.post("/v1/reviews", response_model=ReviewDecision)
def create_review(request: Request, body: ReviewCreate) -> ReviewDecision:
    try:
        return WORKSPACE.add_review(body, principal=request.state.auth)
    except PrincipalAuthorizationDenied as exc:
        raise HTTPException(status_code=403, detail=str(exc)) from exc
    except StopIteration as exc:
        raise HTTPException(status_code=404, detail="unknown draft section") from exc
    except ValueError as exc:
        raise HTTPException(status_code=409, detail=str(exc)) from exc


@app.get("/v1/releases", response_model=list[ReleaseManifest])
def list_releases() -> list[ReleaseManifest]:
    return WORKSPACE.load().releases


@app.post("/v1/releases", response_model=ReleaseBuild)
def create_release(body: ReleaseCreate) -> ReleaseBuild:
    return WORKSPACE.build_release_candidate(body)


@app.post("/v1/legal-source-packages:import", response_model=ImportResponse)
def import_package(package: LegalSourcePackage) -> ImportResponse:
    try:
        result = WORKSPACE.import_package(package)
    except ValueError as exc:
        raise HTTPException(status_code=409, detail=str(exc)) from exc
    return ImportResponse(
        blocked=result.blocked,
        accepted_item_count=len(result.accepted.items),
        omitted_item_ids=list(result.omitted_item_ids),
        reason=result.reason,
    )


@app.get("/v1/factors", response_model=list[FactorEntry])
def list_factors() -> list[FactorEntry]:
    return WORKSPACE.load().factors


@app.post("/v1/factors/{letter}/citations", response_model=FactorEntry)
def link_factor_citation(letter: FactorLetter, body: FactorCitationLink) -> FactorEntry:
    if body.letter != letter:
        raise HTTPException(status_code=400, detail="path letter does not match body")
    try:
        return WORKSPACE.attach_factor_citation(body)
    except ValueError as exc:
        raise HTTPException(status_code=409, detail=str(exc)) from exc


@app.get("/v1/agent-runs", response_model=list[AgentRun])
def list_agent_runs() -> list[AgentRun]:
    return WORKSPACE.load().agent_runs


@app.post("/v1/agent-runs", response_model=AgentRun)
def create_agent_run(body: AgentRunCreate) -> AgentRun:
    return WORKSPACE.add_agent_run(body)


@app.get("/v1/filing-readiness", response_model=FilingReadiness)
def filing_readiness() -> FilingReadiness:
    return WORKSPACE.filing_readiness()


@app.post("/v1/filing-overrides", response_model=FilingOverride)
def create_filing_override(body: FilingOverrideCreate) -> FilingOverride:
    try:
        return WORKSPACE.add_filing_override(body)
    except ValueError as exc:
        raise HTTPException(status_code=409, detail=str(exc)) from exc


@app.get("/v1/docket-events", response_model=list[DocketEvent])
def list_docket_events() -> list[DocketEvent]:
    return WORKSPACE.list_docket_events()


@app.post("/v1/docket-events", response_model=DocketEvent)
def create_docket_event(body: DocketEventCreate) -> DocketEvent:
    return WORKSPACE.add_docket_event(body)


@app.delete("/v1/docket-events/{event_id}")
def delete_docket_event(event_id: UUID) -> dict[str, str]:
    try:
        WORKSPACE.delete_docket_event(event_id)
    except StopIteration as exc:
        raise HTTPException(status_code=404, detail="unknown docket event") from exc
    return {"status": "deleted"}


@app.get("/v1/exhibits", response_model=list[ExhibitCandidate])
def list_exhibits() -> list[ExhibitCandidate]:
    return WORKSPACE.list_exhibits()


@app.post("/v1/exhibits", response_model=ExhibitCandidate)
def annotate_exhibit(body: ExhibitAnnotationCreate) -> ExhibitCandidate:
    try:
        return WORKSPACE.annotate_exhibit(body)
    except ValueError as exc:
        raise HTTPException(status_code=409, detail=str(exc)) from exc


@app.post("/v1/exhibits/{item_id}:bates", response_model=ExhibitCandidate)
def assign_exhibit_bates(item_id: UUID) -> ExhibitCandidate:
    try:
        return WORKSPACE.assign_bates(item_id)
    except ValueError as exc:
        raise HTTPException(status_code=409, detail=str(exc)) from exc


@app.get("/v1/investigations", response_model=list[InvestigationRequest])
def list_investigations() -> list[InvestigationRequest]:
    return WORKSPACE.list_investigations()


@app.post("/v1/investigations", response_model=InvestigationRequest)
def create_investigation(body: InvestigationCreate) -> InvestigationRequest:
    return WORKSPACE.add_investigation(body)


@app.get("/v1/discovery", response_model=list[DiscoveryRequest])
def list_discovery() -> list[DiscoveryRequest]:
    return WORKSPACE.load().discovery


@app.post("/v1/discovery", response_model=DiscoveryRequest)
def create_discovery(body: DiscoveryCreate) -> DiscoveryRequest:
    return WORKSPACE.add_discovery(body)


@app.post("/v1/discovery/{request_id}:status", response_model=DiscoveryRequest)
def update_discovery(request_id: UUID, body: DiscoveryStatusUpdate) -> DiscoveryRequest:
    try:
        return WORKSPACE.set_discovery_status(request_id, body)
    except StopIteration as exc:
        raise HTTPException(status_code=404, detail="unknown discovery request") from exc
    except ValueError as exc:
        raise HTTPException(status_code=409, detail=str(exc)) from exc


@app.get("/v1/templates", response_model=list[DraftingTemplate])
def list_templates() -> list[DraftingTemplate]:
    return template_catalog()


@app.post("/v1/templates:instantiate", response_model=DraftView)
def instantiate_template(body: TemplateInstantiate) -> DraftView:
    try:
        section = WORKSPACE.instantiate_template(body)
    except StopIteration as exc:
        raise HTTPException(status_code=404, detail="unknown template") from exc
    return _draft_view(section)


@app.post("/v1/drafts", response_model=DraftView)
def create_draft(body: DraftRequest) -> DraftView:
    try:
        section = WORKSPACE.draft_factor_section(body.letter, body.heading, body.body)
    except StopIteration as exc:
        raise HTTPException(status_code=404, detail="unknown factor") from exc
    return _draft_view(section)


@app.get("/v1/drafts", response_model=list[DraftView])
def list_drafts() -> list[DraftView]:
    return [_draft_view(row) for row in WORKSPACE.load().drafts]


@app.post("/v1/drafts/{section_id}:gate", response_model=GateResponse)
def gate_draft(section_id: UUID) -> GateResponse:
    """Static :gate suffix MUST register before PUT /v1/drafts/{section_id}."""
    try:
        result = WORKSPACE.gate_draft(section_id)
    except (ValueError, StopIteration) as exc:
        raise HTTPException(status_code=409, detail=str(exc)) from exc
    return GateResponse(ok=result.ok, blockers=list(result.blockers))


@app.get("/v1/drafts/{section_id}/support-map", response_model=SupportMap)
def draft_support_map(section_id: UUID) -> SupportMap:
    try:
        section = next(row for row in WORKSPACE.load().drafts if row.section_id == section_id)
    except StopIteration as exc:
        raise HTTPException(status_code=404, detail="unknown draft section") from exc
    view = _draft_view(section)
    assert view.support is not None
    return view.support


@app.put("/v1/drafts/{section_id}", response_model=DraftView)
def edit_draft(section_id: UUID, body: DraftEditRequest) -> DraftView:
    try:
        section = WORKSPACE.update_draft(section_id, body.heading, body.body)
    except StopIteration as exc:
        raise HTTPException(status_code=404, detail="unknown draft section") from exc
    return _draft_view(section)


@app.post("/v1/citation-resolutions:batch", response_model=GateResponse)
def citation_gate(body: GateRequest) -> GateResponse:
    imported = import_legal_source_package(body.package)
    if imported.blocked:
        raise HTTPException(status_code=409, detail=imported.reason)
    facts = validate_factual_citations(body.facts, imported.accepted)
    authorities = validate_authority_citations(body.authorities)
    blockers = list(facts.blockers) + list(authorities.blockers)
    return GateResponse(ok=not blockers, blockers=blockers)


@app.post("/v1/privilege:scan", response_model=PrivilegeScan)
def privilege_scan(body: PrivilegeScanRequest) -> PrivilegeScan:
    """Keyword first-pass only. Never a privilege legal conclusion."""
    text = (body.text or "").strip()
    section_id = body.section_id
    source = "text"
    if not text:
        if section_id is None:
            raise HTTPException(status_code=400, detail="text or section_id is required")
        try:
            section = next(row for row in WORKSPACE.load().drafts if row.section_id == section_id)
        except StopIteration as exc:
            raise HTTPException(status_code=404, detail="unknown draft section") from exc
        text = f"{section.heading}\n\n{section.body}"
        source = "section"
    scan = scan_text(text)
    return scan.model_copy(update={"source": source, "section_id": section_id})


class ConfidentialBody(BaseModel):
    on: bool


class ConfidentialView(BaseModel):
    on: bool
    court_safe: bool = False
    disclosure: str = "Confidential Mode is routing chrome, not a privilege holding."


class AppSettingsBody(BaseModel):
    confidential_mode: bool | None = None
    theme: str | None = None
    display_timezone: str | None = None
    case_phase: str | None = None


class AppSettingsView(BaseModel):
    confidential_mode: bool
    theme: str
    display_timezone: str
    case_phase: str
    court_safe: bool = False


def _settings_view(state) -> AppSettingsView:
    return AppSettingsView(
        confidential_mode=state.confidential_mode,
        theme=state.theme,
        display_timezone=state.display_timezone,
        case_phase=state.case_phase,
    )


class SurfaceContextView(BaseModel):
    path: str
    matter: str
    court_case: str
    case_phase: str
    confidential_mode: bool
    saved: object
    background: str = ""


@app.get("/v1/surface-context", response_model=SurfaceContextView)
def get_surface_context(path: str = "/") -> SurfaceContextView:
    pack = WORKSPACE.surface_context(path)
    return SurfaceContextView.model_validate(pack)


@app.get("/v1/settings", response_model=AppSettingsView)
def get_settings_row() -> AppSettingsView:
    return _settings_view(WORKSPACE.load())


@app.put("/v1/settings", response_model=AppSettingsView)
def put_settings_row(body: AppSettingsBody) -> AppSettingsView:
    try:
        state = WORKSPACE.update_settings(
            confidential_mode=body.confidential_mode,
            theme=body.theme,
            display_timezone=body.display_timezone,
            case_phase=body.case_phase,
        )
    except ValueError as exc:
        raise HTTPException(status_code=400, detail=str(exc)) from exc
    return _settings_view(state)


@app.get("/v1/confidential", response_model=ConfidentialView)
def get_confidential() -> ConfidentialView:
    return ConfidentialView(on=WORKSPACE.load().confidential_mode)


@app.put("/v1/confidential", response_model=ConfidentialView)
def put_confidential(body: ConfidentialBody) -> ConfidentialView:
    return ConfidentialView(on=WORKSPACE.set_confidential(body.on))


@app.post("/v1/redactions", response_model=RedactionResult)
async def redact_owner_pdf(
    tokens: str = Form(...),
    file: UploadFile = File(...),
) -> RedactionResult:
    """Content-stream redaction of an owner-produced PDF. Not Agno evidence."""
    names = [item.strip() for item in tokens.split(",") if item.strip()]
    if not names:
        raise HTTPException(status_code=400, detail="at least one redaction token is required")
    folder = WORKSPACE.store_dir / "redactions"
    folder.mkdir(parents=True, exist_ok=True)
    src = folder / "source.pdf"
    dest = folder / "redacted.pdf"
    src.write_bytes(await file.read())
    try:
        return redact_content_stream(src, dest, names)
    except ValueError as exc:
        raise HTTPException(status_code=400, detail=str(exc)) from exc


@app.post("/v1/bates:stamp", response_model=BatesStampResult)
async def stamp_owner_pdf(
    bates_number: str = Form(...),
    file: UploadFile = File(...),
) -> BatesStampResult:
    """pypdf overlay of an owner-produced PDF. Not Agno evidence."""
    if not bates_number.strip():
        raise HTTPException(status_code=400, detail="bates_number is required")
    folder = WORKSPACE.store_dir / "bates"
    folder.mkdir(parents=True, exist_ok=True)
    src = folder / "source.pdf"
    dest = folder / "stamped.pdf"
    src.write_bytes(await file.read())
    try:
        return stamp_bates_pdf(src, dest, bates_number.strip())
    except ValueError as exc:
        raise HTTPException(status_code=400, detail=str(exc)) from exc


@app.post("/v1/events:apply", response_model=RevocationResponse)
def apply_event(body: RevocationRequest) -> RevocationResponse:
    try:
        products = apply_evidence_event(body.event, body.products)
    except ValueError as exc:
        raise HTTPException(status_code=400, detail=str(exc)) from exc
    return RevocationResponse(products=products)


class JsonExportResponse(BaseModel):
    """Response model for JSON export endpoint."""
    exported_at: str
    version: str
    workspace_state: dict[str, Any]


@app.get("/v1/export/json", response_model=JsonExportResponse)
def export_json() -> JsonExportResponse:
    """Export complete workspace state as JSON for migration/backup.

    This endpoint must be explicitly enabled via LEGAL_WORKSPACE_JSON_EXPORT_ENABLED=true
    for security reasons. Returns a complete snapshot of the current workspace state
    including all entities, relationships, and metadata.
    """
    settings = get_settings()
    if not settings.json_export_enabled:
        raise HTTPException(
            status_code=403,
            detail="JSON export is disabled. Set LEGAL_WORKSPACE_JSON_EXPORT_ENABLED=true to enable."
        )

    # Get raw dict state for JSON export (bypassing WorkspaceState validation)
    from legal_workspace.db.store import WorkspaceStore
    state_dict = WorkspaceStore.load(
        matter_id=WORKSPACE._matter_id,
        store_dir=WORKSPACE._store_dir_str
    )
    if state_dict is None:
        # Fallback to loading through workspace and converting to dict
        state = WORKSPACE.load()
        state_dict = state.model_dump(mode='json')

    return JsonExportResponse(
        exported_at=datetime.now(UTC).isoformat(),
        version=__version__,
        workspace_state=state_dict
    )


from legal_workspace.api.automation_routes import router as automation_router
from legal_workspace.api.calendar_routes import router as calendar_router
from legal_workspace.api.citation_routes import router as citation_router
from legal_workspace.api.document_routes import router as document_router
from legal_workspace.api.work_document_routes import router as work_document_router
from legal_workspace.api.office_template_routes import router as office_template_router
from legal_workspace.api.office_routes import router as office_router
from legal_workspace.api.evidence_catalog_routes import router as evidence_catalog_router
from legal_workspace.api.factor_routes import router as factor_router
from legal_workspace.api.ops_routes import router as ops_router
from legal_workspace.api.privilege_routes import router as privilege_router
from legal_workspace.api.routing_routes import router as routing_router
from legal_workspace.api.source_routes import source_router

app.include_router(source_router)
app.include_router(automation_router)
app.include_router(calendar_router)
app.include_router(citation_router)
app.include_router(document_router)
app.include_router(work_document_router)
app.include_router(office_template_router)
app.include_router(office_router)
app.include_router(ops_router)
app.include_router(privilege_router)
app.include_router(factor_router)
app.include_router(routing_router)
app.include_router(evidence_catalog_router)

# MCP face: one `advocatio` gateway for ContextForge (see api/mcp_face.py).
app.mount("/mcp", mcp_app)

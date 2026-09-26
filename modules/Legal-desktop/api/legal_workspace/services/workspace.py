"""Disk-backed Matter workspace. Every mutation is written before return.

> _Byline: Claude Code · Kimi K2.7 · 2026-08-18_
"""

from __future__ import annotations

from datetime import UTC, datetime
from pathlib import Path
from uuid import UUID, uuid4

from pydantic import BaseModel, Field

from legal_workspace.api.auth import AuthenticatedPrincipal, require_human_review_actor
from legal_workspace.contracts.citations import AuthorityCitation, AuthorityLevel, EvidenceCitation
from legal_workspace.contracts.identity import CourtCaseRef, MatterRef
from legal_workspace.contracts.source_package import LegalSourcePackage, ReviewState
from legal_workspace.domain.agents import (
    AgentRun,
    AgentRunCreate,
    AgentRunStatus,
    forbidden_intent,
    route_role,
)
from legal_workspace.domain.authority_library import CuratedAuthority, seedable_authorities
from legal_workspace.domain.calendar import (
    DocketEvent,
    DocketEventCreate,
    default_status_for,
)
from legal_workspace.domain.filing import (
    CheckState,
    FilingOverride,
    FilingOverrideCreate,
    FilingReadiness,
    evaluate_filing_readiness,
)
from legal_workspace.domain.discovery import (
    DiscoveryCreate,
    DiscoveryRequest,
    DiscoveryStatus,
    DiscoveryStatusUpdate,
    structural_discovery,
)
from legal_workspace.domain.exhibits import (
    ExhibitAnnotation,
    ExhibitAnnotationCreate,
    ExhibitCandidate,
    ExhibitReadiness,
    bates_prefix,
    next_bates_number,
)
from legal_workspace.domain.investigation import (
    InvestigationCreate,
    InvestigationRequest,
    to_created_event,
)
from legal_workspace.domain.redteam import RedTeamCreate, RedTeamRun
from legal_workspace.domain.release import ReleaseBuild, ReleaseCreate, ReleaseManifest, canonical_content_hash
from legal_workspace.domain.research import (
    CurrencyFlag,
    ResearchCreate,
    ResearchQuestion,
    ResearchStatus,
    structural_research_questions,
)
from legal_workspace.domain.review import ReviewCreate, ReviewDecision, ReviewVerdict
from legal_workspace.domain.strategy import StrategyCreate, StrategyNote
from legal_workspace.domain.todos import CaseTodo, TodoCreate, TodoStatus, structural_todos
from legal_workspace.domain.factors import (
    FactorCitationLink,
    FactorEntry,
    FactorLetter,
    empty_factor_matrix,
)
from legal_workspace.domain.issue import (
    IssueChildCreate,
    IssueElementCreate,
    IssuePatch,
    LegalIssue,
    default_custody_modification_issue,
)
from legal_workspace.domain.templates import TemplateInstantiate, get_template, render_body
from legal_workspace.domain.work_product import WorkProductState, WorkProductVersion
from legal_workspace.services.citation_gate import CitationGateResult, validate_factual_citations
from legal_workspace.services.persist import (
    append_jsonl,
    atomic_write_json,
    default_store_dir,
    read_json,
)
from legal_workspace.services import sqlite_store
from legal_workspace.config import get_settings
from legal_workspace.db.store import WorkspaceStore, create_tables, to_uuid
from legal_workspace.services.agno_client import list_matters, verify_package_hashes
from legal_workspace.domain.provider_grid import confidential_blocked_reason
from legal_workspace.services.gateway import invoke_chat
from legal_workspace.services.routing import load_routing, model_for_role, surface_for_role
from legal_workspace.services.source_package import (
    ImportResult,
    import_legal_source_package,
    validate_consumer_package,
)


class DraftSection(BaseModel):
    section_id: UUID
    heading: str
    body: str
    citations: list[EvidenceCitation] = Field(default_factory=list)
    factor_letter: FactorLetter | None = None


class WorkspaceState(BaseModel):
    matter: MatterRef
    court_case: CourtCaseRef
    package: LegalSourcePackage | None = None
    omitted_item_ids: list[str] = Field(default_factory=list)
    issue: LegalIssue
    factors: list[FactorEntry]
    drafts: list[DraftSection] = Field(default_factory=list)
    work_product: WorkProductVersion | None = None
    authorities: list[CuratedAuthority] = Field(default_factory=seedable_authorities)
    strategy_notes: list[StrategyNote] = Field(default_factory=list)
    redteam_runs: list[RedTeamRun] = Field(default_factory=list)
    todos: list[CaseTodo] = Field(default_factory=structural_todos)
    reviews: list[ReviewDecision] = Field(default_factory=list)
    releases: list[ReleaseManifest] = Field(default_factory=list)
    research_questions: list[ResearchQuestion] = Field(default_factory=structural_research_questions)
    currency_flags: list[CurrencyFlag] = Field(default_factory=list)
    discovery: list[DiscoveryRequest] = Field(default_factory=structural_discovery)
    investigations: list[InvestigationRequest] = Field(default_factory=list)
    docket_events: list[DocketEvent] = Field(default_factory=list)
    filing_overrides: list[FilingOverride] = Field(default_factory=list)
    agent_runs: list[AgentRun] = Field(default_factory=list)
    exhibit_annotations: list[ExhibitAnnotation] = Field(default_factory=list)
    last_agno_verify: list[dict[str, object]] = Field(default_factory=list)
    confidential_mode: bool = False
    theme: str = "dark"
    display_timezone: str = "America/New_York"
    case_phase: str = "Discovery"


def _blank_state() -> WorkspaceState:
    matter_id = uuid4()
    case_id = uuid4()
    return WorkspaceState(
        matter=MatterRef(
            matter_id=matter_id,
            display_name="Genesee County custody matter",
            is_friendly_primary=True,
        ),
        court_case=CourtCaseRef(
            court_case_id=case_id,
            matter_id=matter_id,
            display_name="7th Circuit Family Division — primary proceeding",
            court="7th Judicial Circuit, Genesee County, Michigan",
            is_primary=True,
        ),
        issue=default_custody_modification_issue(),
        factors=empty_factor_matrix(),
        authorities=seedable_authorities(),
        todos=structural_todos(),
        research_questions=structural_research_questions(),
        discovery=structural_discovery(),
    )


def _agno_verify_check(
    hashes: list[str],
    results: tuple,
) -> tuple[CheckState, str]:
    if not hashes:
        return CheckState.FAIL, "No package hashes to send to Agno."
    if any(row.reason == "not-a-sha256-hex" for row in results):
        return (
            CheckState.FAIL,
            "One or more package hashes are not 64-hex; Agno verify was not called.",
        )
    if any(not row.reachable for row in results):
        return CheckState.FAIL, "Agno verify unreachable (fail-closed)."
    if all(row.ok for row in results):
        return CheckState.PASS, f"{len(results)} hash(es) verified by Agno."
    return CheckState.FAIL, "Agno verify rejected one or more package hashes."


def _is_artificial_package(package: LegalSourcePackage | None) -> bool:
    if package is None:
        return False
    if package.manifest_hash == "sha256:pkg":
        return True
    locators = {item.span_locator for item in package.items}
    return locators <= {"s:1", "s:2"}


class Workspace:
    """Single-Matter session persisted under data/workspace/."""

    def __init__(self, store_dir: Path | None = None) -> None:
        self.store_dir = Path(store_dir) if store_dir is not None else default_store_dir()
        self.state_path = self.store_dir / "state.json"
        self.events_path = self.store_dir / "events.jsonl"
        self._store_dir_str = str(self.store_dir)
        create_tables(store_dir=self._store_dir_str)

        # Determine the matter identity for this workspace.
        # SQLite is canonical: if a matter already exists, adopt it. Only mint a
        # new blank identity when the DB file is truly empty.
        existing = WorkspaceStore.stored_matter_id(store_dir=self._store_dir_str)
        if existing is not None:
            self._matter_id = existing
        else:
            legacy = read_json(self.state_path)
            if legacy is not None and isinstance(legacy, dict):
                self._matter_id = to_uuid(legacy.get("matter", {}).get("matter_id"))
            else:
                blank = _blank_state()
                self._matter_id = blank.matter.matter_id
                self._write(blank, action="init")

        # Backwards compatibility: migrate existing JSON state to SQLite once.
        legacy = read_json(self.state_path)
        if legacy is not None and self.store_dir.resolve() == default_store_dir().resolve():
            self._reconcile_real_data()

    def _write(self, state: WorkspaceState, action: str) -> None:
        # SQLite is the source of truth.
        payload = state.model_dump(mode="json")
        WorkspaceStore.save(payload, action, store_dir=self._store_dir_str)

        if get_settings().debug_json:
            atomic_write_json(self.state_path, payload)
            append_jsonl(
                self.events_path,
                {
                    "action": action,
                    "matter_id": str(state.matter.matter_id),
                    "package_id": str(state.package.package_id) if state.package else None,
                    "draft_count": len(state.drafts),
                    "strategy_count": len(state.strategy_notes),
                    "redteam_count": len(state.redteam_runs),
                    "open_todos": sum(1 for item in state.todos if item.status.value == "open"),
                    "review_count": len(state.reviews),
                    "release_count": len(state.releases),
                    "open_research": sum(
                        1
                        for item in state.research_questions
                        if item.status.value in {"open", "unresolved"}
                    ),
                    "docket_count": len(state.docket_events),
                    "investigation_count": len(state.investigations),
                },
            )

    def _reconcile_real_data(self) -> None:
        """Drop incomplete/test rows. Keep complete applicable authorities."""
        state = WorkspaceState.model_validate(read_json(self.state_path))
        changed = False
        if _is_artificial_package(state.package):
            state.package = None
            state.omitted_item_ids = []
            state.exhibit_annotations = []
            changed = True
        known = {row.identifier for row in state.authorities}
        for row in seedable_authorities():
            if row.identifier not in known:
                state.authorities.append(row)
                changed = True
        if not state.todos:
            state.todos = structural_todos()
            changed = True
        if not state.research_questions:
            state.research_questions = structural_research_questions()
            changed = True
        if not state.discovery:
            state.discovery = structural_discovery()
            changed = True
        if not state.issue.children:
            state.issue = default_custody_modification_issue()
            changed = True
        if not state.matter.county:
            state.matter.county = "Genesee"
            changed = True
        if not state.court_case.division:
            state.court_case.division = "Family Division"
            state.court_case.courthouse = "900 S. Saginaw St., Flint, Michigan"
            state.court_case.court_url = "https://7thcircuitcourt.com/"
            changed = True
        if changed:
            self._write(state, action="reconcile-real-data")

    def load(self) -> WorkspaceState:
        # Prefer SQLite; fall back to legacy JSON for one migration pass.
        raw = WorkspaceStore.load(self._matter_id, store_dir=self._store_dir_str)
        if raw is None:
            raw = read_json(self.state_path)
        if raw is None:
            state = _blank_state()
            self._matter_id = state.matter.matter_id
            self._write(state, action="init-missing")
            return self._overlay_sql_settings(state)
        return self._overlay_sql_settings(WorkspaceState.model_validate(raw))

    def surface_context(self, path: str) -> dict[str, object]:
        """Saved workspace snapshot for the open page. Not live unsaved form text."""
        state = self.load()
        key = (path or "/").split("?")[0].rstrip("/") or "/"
        pack: dict[str, object] = {
            "path": key,
            "matter": state.matter.display_name,
            "court_case": state.court_case.display_name,
            "case_phase": state.case_phase,
            "confidential_mode": state.confidential_mode,
        }
        if key in {"/", ""}:
            pack["saved"] = {
                "issue": state.issue.title,
                "drafts": len(state.drafts),
                "reviews": len(state.reviews),
                "releases": len(state.releases),
                "package_items": len(state.package.items) if state.package else 0,
            }
        elif key == "/custody-factors":
            pack["background"] = (
                "Both-parent rule — assistant only, not a screen dump. "
                "Address each parent. Conduct, not diagnoses. "
                "Do not invent facts. Not a finding. Weighting is yours."
            )
            pack["saved"] = [
                {
                    "letter": item.letter.value,
                    "title": item.title,
                    "petitioner_notes": item.petitioner.for_parent[:6],
                    "respondent_notes": item.respondent.for_parent[:6],
                    "contradictions": item.contradictions[:6],
                    "missing_proof": item.missing_proof[:6],
                }
                for item in state.factors
            ]
        elif key == "/questions":
            pack["saved"] = {
                "title": state.issue.title,
                "authority": state.issue.governing_authority,
                "elements": [row.label for row in state.issue.elements],
            }
        elif key in {"/drafts", "/templates", "/review", "/final-copy"}:
            pack["saved"] = [
                {"heading": item.heading, "body": item.body[:800]} for item in state.drafts[:8]
            ]
        elif key == "/tasks":
            pack["saved"] = [
                {"title": item.title, "status": item.status.value} for item in state.todos[:20]
            ]
        elif key in {"/calendar", "/timeline"}:
            pack["saved"] = [
                {"title": item.title, "occurs_at": item.occurs_at.isoformat()}
                for item in state.docket_events[:20]
            ]
        elif key == "/laws":
            pack["saved"] = [
                {"identifier": item.identifier, "proposition": item.proposition[:200]}
                for item in state.authorities[:20]
            ]
        elif key == "/case-search":
            pack["saved"] = {"note": "Case search is CourtListener identity hits only. It does not check whether a case is still good law."}
        elif key == "/citation-check":
            pack["saved"] = {"note": "Citation check parses structure only. It does not verify whether a case is still valid."}
        elif key == "/open-questions":
            pack["saved"] = [
                {"question": item.question, "status": item.status.value}
                for item in state.research_questions[:20]
            ]
        elif key == "/confidentiality-check":
            pack["saved"] = {"note": "Privilege scan is a first-pass. Not a legal conclusion."}
        else:
            pack["saved"] = {"note": "No extra saved snapshot for this page."}
        return pack

    def _overlay_sql_settings(self, state: WorkspaceState) -> WorkspaceState:
        row = sqlite_store.load_settings(self.store_dir)
        state.confidential_mode = row.confidential_mode
        state.theme = row.theme
        state.display_timezone = row.display_timezone
        state.case_phase = row.case_phase
        return state

    def apply_agno_projection(self, *, client=None) -> WorkspaceState:
        """Replace local Matter identity with Agno GET /v1/matters. Fail closed."""
        state = self.load()
        listed = list_matters(client=client)
        if not listed.ok or not listed.matters:
            return state
        chosen = None
        for item in listed.matters:
            try:
                if UUID(str(item.matter_id)) == state.matter.matter_id:
                    chosen = item
                    break
            except ValueError:
                continue
        if chosen is None:
            chosen = listed.matters[0]
        try:
            mid = UUID(str(chosen.matter_id))
        except ValueError:
            return state
        if (
            state.matter.matter_id == mid
            and state.matter.display_name == chosen.display_name
            and state.court_case.matter_id == mid
        ):
            return state
        state.matter.matter_id = mid
        state.matter.display_name = chosen.display_name
        state.court_case.matter_id = mid
        self._write(state, action="agno-project-identity")
        return state

    def home(self, *, agno_http=None) -> WorkspaceState:
        return self.apply_agno_projection(client=agno_http)

    def import_package(self, package: LegalSourcePackage) -> ImportResult:
        state = self.load()
        validate_consumer_package(package, state.matter.matter_id)
        result = import_legal_source_package(package)
        if result.blocked:
            return result
        state.package = result.accepted
        state.omitted_item_ids = list(result.omitted_item_ids)
        accepted_ids = {item.item_id for item in result.accepted.items}
        state.exhibit_annotations = [
            row for row in state.exhibit_annotations if row.item_id in accepted_ids
        ]
        self._write(state, action="import")
        return result

    def attach_factor_citation(self, link: FactorCitationLink) -> FactorEntry:
        state = self.load()
        if state.package is None:
            raise ValueError("import an approved LegalSourcePackage first")
        if str(link.package_id) != str(state.package.package_id):
            raise ValueError("citation is not from the imported package")
        gate = validate_factual_citations([link.citation], state.package)
        if not gate.ok:
            raise ValueError("; ".join(gate.blockers))
        if link.side not in {"petitioner", "respondent"}:
            raise ValueError("side must be petitioner or respondent")
        entry = next(row for row in state.factors if row.letter is link.letter)
        side = entry.petitioner if link.side == "petitioner" else entry.respondent
        side.citations.append(link.citation)
        self._write(state, action=f"factor-{link.letter.value}-cite")
        return entry

    def instantiate_template(self, created: TemplateInstantiate) -> DraftSection:
        template = get_template(created.template_id)
        section = DraftSection(
            section_id=uuid4(),
            heading=created.heading or template.title,
            body=render_body(template),
            citations=[],
            factor_letter=None,
        )
        state = self.load()
        state.drafts.append(section)
        self._write(state, action=f"template-{template.template_id}")
        return section

    def draft_factor_section(self, letter: FactorLetter, heading: str, body: str) -> DraftSection:
        state = self.load()
        entry = next(row for row in state.factors if row.letter is letter)
        citations = list(entry.petitioner.citations) + list(entry.respondent.citations)
        section = DraftSection(
            section_id=uuid4(),
            heading=heading,
            body=body,
            citations=citations,
            factor_letter=letter,
        )
        state.drafts.append(section)
        state.work_product = WorkProductVersion(
            work_product_id=section.section_id,
            state=WorkProductState.PRIVATE_DRAFT,
            source_package_id=state.package.package_id if state.package else None,
            content_hash=f"draft:{section.section_id}",
            cited_assertion_ids=[c.assertion_id for c in citations],
            cited_authority_ids=["MCL 722.23"],
        )
        self._write(state, action="draft")
        return section

    def update_draft(self, section_id: UUID, heading: str, body: str) -> DraftSection:
        state = self.load()
        section = next(row for row in state.drafts if row.section_id == section_id)
        released_ids = {sid for item in state.releases for sid in item.section_ids}
        if section.section_id in released_ids:
            forked = DraftSection(
                section_id=uuid4(),
                heading=heading,
                body=body,
                citations=list(section.citations),
                factor_letter=section.factor_letter,
            )
            state.drafts.append(forked)
            self._write(state, action="draft-fork")
            return forked
        section.heading = heading
        section.body = body
        self._write(state, action="draft-edit")
        return section

    def gate_draft(self, section_id: UUID) -> CitationGateResult:
        state = self.load()
        if state.package is None:
            raise ValueError("import an approved LegalSourcePackage first")
        section = next(row for row in state.drafts if row.section_id == section_id)
        result = validate_factual_citations(section.citations, state.package)
        if get_settings().debug_json:
            append_jsonl(
                self.events_path,
                {
                    "action": "gate",
                    "section_id": str(section_id),
                    "ok": result.ok,
                    "blockers": list(result.blockers),
                },
            )
        return result

    def add_strategy_note(self, created: StrategyCreate) -> StrategyNote:
        state = self.load()
        note = StrategyNote(
            kind=created.kind,
            title=created.title,
            body=created.body,
            source_chat=created.source_chat,
        )
        state.strategy_notes.append(note)
        self._write(state, action=f"strategy-{created.kind.value}")
        return note

    def list_strategy_notes(self) -> list[StrategyNote]:
        return list(self.load().strategy_notes)

    def add_redteam_run(self, created: RedTeamCreate) -> RedTeamRun:
        state = self.load()
        run = RedTeamRun(
            target_type=created.target_type,
            target_id=created.target_id,
            lens=created.lens,
            prompt_or_notes=created.prompt_or_notes,
            findings=created.findings,
            model=created.model,
        )
        state.redteam_runs.append(run)
        self._write(state, action=f"redteam-{created.lens.value}")
        return run

    def add_research_question(self, created: ResearchCreate) -> ResearchQuestion:
        state = self.load()
        question = ResearchQuestion(
            question=created.question,
            plan=created.plan,
            adverse_notes=created.adverse_notes,
            uncertainty=created.uncertainty,
            authority_ids=list(created.authority_ids),
            jurisdiction=created.jurisdiction,
        )
        state.research_questions.append(question)
        self._write(state, action="research-add")
        return question

    def set_research_status(self, question_id: UUID, status: ResearchStatus) -> ResearchQuestion:
        state = self.load()
        question = next(row for row in state.research_questions if row.question_id == question_id)
        question.status = status
        self._write(state, action=f"research-{status.value}")
        return question

    def flag_authority_currency(self, identifier: str, note: str) -> CurrencyFlag:
        state = self.load()
        known = {row.identifier for row in state.authorities}
        if identifier not in known:
            raise ValueError(f"unknown authority: {identifier}")
        flag = CurrencyFlag(identifier=identifier, note=note)
        state.currency_flags.append(flag)
        if state.work_product and identifier in state.work_product.cited_authority_ids:
            state.work_product.state = WorkProductState.REVALIDATION_REQUIRED
        self._write(state, action="authority-currency")
        return flag

    def add_agent_run(self, created: AgentRunCreate, invoker=None) -> AgentRun:
        table = load_routing()
        role = route_role(created.intent, created.role)
        requested = model_for_role(role.value, created.requested_model, table)
        blocked = forbidden_intent(created.intent)
        if blocked:
            run = AgentRun(
                role=role,
                intent=created.intent,
                prompt=created.prompt,
                requested_model=requested,
                effective_model="not-invoked",
                target_type=created.target_type,
                target_id=created.target_id,
                output=f"Blocked: agents cannot {blocked}. Owner must use Your review / Filing readiness checklist.",
                status=AgentRunStatus.BLOCKED,
            )
        else:
            surface = surface_for_role(role.value, table)
            confidential = self.load().confidential_mode
            if confidential and confidential_blocked_reason(requested) is not None:
                run = AgentRun(
                    role=role,
                    intent=created.intent,
                    prompt=created.prompt,
                    requested_model=requested,
                    effective_model="not-invoked",
                    target_type=created.target_type,
                    target_id=created.target_id,
                    output="confidential_blocked. No silent fallback to Claude/ChatGPT consumer.",
                    status=AgentRunStatus.BLOCKED,
                )
            else:
                should_invoke = invoker is not None or get_settings().invoke_models
                if should_invoke:
                    if invoker is not None:
                        result = invoker(created.prompt, requested)
                    else:
                        result = invoke_chat(
                            created.prompt, requested, confidential=confidential
                        )
                    run = AgentRun(
                        role=role,
                        intent=created.intent,
                        prompt=created.prompt,
                        requested_model=requested,
                        effective_model=result.model,
                        target_type=created.target_type,
                        target_id=created.target_id,
                        output=result.text + "\n\n[hypothesis only · not court-safe]",
                        status=AgentRunStatus.NEEDS_OWNER_REVIEW,
                    )
                else:
                    run = AgentRun(
                        role=role,
                        intent=created.intent,
                        prompt=created.prompt,
                        requested_model=requested,
                        effective_model="not-invoked",
                        target_type=created.target_type,
                        target_id=created.target_id,
                        output=(
                            f"Routed to {role.value}. No model was called. "
                            f"Continue on {surface}. "
                            "Output is a hypothesis, not a finding."
                        ),
                        status=AgentRunStatus.NEEDS_OWNER_REVIEW,
                    )
        state = self.load()
        state.agent_runs.append(run)
        self._write(state, action=f"agent-{run.role.value}-{run.status.value}")
        return run

    def add_filing_override(self, created: FilingOverrideCreate) -> FilingOverride:
        if created.reviewer != "owner":
            raise ValueError("only the owner may verify a filing check")
        state = self.load()
        override = FilingOverride(
            check_id=created.check_id,
            state=created.state,
            note=created.note,
            reviewer=created.reviewer,
        )
        state.filing_overrides.append(override)
        self._write(state, action=f"filing-{created.check_id}")
        return override

    def filing_readiness(self, *, agno_http=None) -> FilingReadiness:
        state = self.load()
        approved = any(item.verdict is ReviewVerdict.APPROVE for item in state.reviews)
        last_filed = bool(state.releases and state.releases[-1].filed)
        hashes = [item.content_hash for item in state.package.items] if state.package else []
        results = verify_package_hashes(hashes, client=agno_http) if hashes else ()
        verify_state, verify_reason = _agno_verify_check(hashes, results)
        snapshot = [
            {
                "digest": row.digest,
                "ok": row.ok,
                "reachable": row.reachable,
                "verdict": row.verdict,
                "reason": row.reason,
            }
            for row in results
        ]
        if snapshot != state.last_agno_verify:
            state.last_agno_verify = snapshot
            self._write(state, action="agno-verify")
        return evaluate_filing_readiness(
            package_imported=state.package is not None,
            docket_number=state.court_case.docket_number,
            judge=state.court_case.judge,
            review_approved=approved,
            release_count=len(state.releases),
            last_release_filed=last_filed,
            overrides=state.filing_overrides,
            agno_verify_state=verify_state,
            agno_verify_reason=verify_reason,
        )

    def add_docket_event(self, created: DocketEventCreate) -> DocketEvent:
        state = self.load()
        event = DocketEvent(
            occurs_at=created.occurs_at,
            title=created.title,
            kind=created.kind,
            detail=created.detail,
            location=created.location,
            source=created.source,
            confirmed=created.confirmed,
            status=created.status or default_status_for(created.occurs_at),
        )
        state.docket_events.append(event)
        state.docket_events.sort(key=lambda row: row.occurs_at)
        self._write(state, action=f"docket-{event.kind.value}")
        return event

    def list_docket_events(self) -> list[DocketEvent]:
        return sorted(self.load().docket_events, key=lambda row: row.occurs_at)

    def upcoming_events(self, now: datetime | None = None, limit: int = 5) -> list[DocketEvent]:
        moment = now or datetime.now(UTC)
        rows = [item for item in self.list_docket_events() if item.occurs_at > moment]
        return rows[:limit]

    def upcoming_event_count(self, now: datetime | None = None) -> int:
        return len(self.upcoming_events(now, limit=10_000))

    def audit_count(self) -> int:
        from legal_workspace.services.persist import read_jsonl

        return len(read_jsonl(self.events_path, limit=10_000))

    def delete_docket_event(self, event_id: UUID) -> None:
        state = self.load()
        before = len(state.docket_events)
        state.docket_events = [item for item in state.docket_events if item.event_id != event_id]
        if len(state.docket_events) == before:
            raise StopIteration
        self._write(state, action="docket-delete")

    def add_investigation(self, created: InvestigationCreate) -> InvestigationRequest:
        state = self.load()
        request = InvestigationRequest(
            needed=created.needed,
            why=created.why,
            kind=created.kind,
            linked_issue=created.linked_issue,
            factor_letter=created.factor_letter,
            contradiction=created.contradiction,
            existing_assertion_ids=list(created.existing_assertion_ids),
        )
        state.investigations.append(request)
        self._write(state, action=f"investigation-{created.kind.value}")
        envelope = to_created_event(
            request,
            matter_id=state.matter.matter_id,
            court_case_id=state.court_case.court_case_id,
            aggregate_version=len(state.investigations),
        )
        if get_settings().debug_json:
            append_jsonl(self.events_path, envelope.model_dump(mode="json"))
        return request

    def list_investigations(self) -> list[InvestigationRequest]:
        return list(self.load().investigations)

    def list_exhibits(self) -> list[ExhibitCandidate]:
        state = self.load()
        if state.package is None:
            return []
        notes = {row.item_id: row for row in state.exhibit_annotations}
        candidates: list[ExhibitCandidate] = []
        for item in state.package.items:
            if item.review_state is not ReviewState.APPROVED:
                continue
            note = notes.get(item.item_id)
            candidates.append(
                ExhibitCandidate(
                    item_id=item.item_id,
                    package_id=state.package.package_id,
                    assertion_id=item.assertion_id,
                    assertion_version=item.assertion_version,
                    span_locator=item.span_locator,
                    custody_locator=item.custody_locator,
                    content_hash=item.content_hash,
                    review_state=item.review_state,
                    exhibit_label=note.exhibit_label if note else "",
                    bates_number=note.bates_number if note else "",
                    foundation=note.foundation if note else "",
                    custody_note=note.custody_note if note else "",
                    redaction_note=note.redaction_note if note else "",
                    relevance=note.relevance if note else "",
                    issues=list(note.issues) if note else [],
                    objection_notes=note.objection_notes if note else "",
                    readiness=note.readiness if note else ExhibitReadiness.CANDIDATE,
                    bytes_present=False,
                )
            )
        return candidates

    def annotate_exhibit(self, created: ExhibitAnnotationCreate) -> ExhibitCandidate:
        state = self.load()
        if state.package is None:
            raise ValueError("import an approved LegalSourcePackage first")
        try:
            item = next(row for row in state.package.items if row.item_id == created.item_id)
        except StopIteration as exc:
            raise ValueError("exhibit candidates come only from approved package items") from exc
        if item.review_state is not ReviewState.APPROVED:
            raise ValueError("exhibit candidates come only from approved package items")
        annotation = ExhibitAnnotation(
            item_id=created.item_id,
            exhibit_label=created.exhibit_label,
            bates_number=created.bates_number,
            foundation=created.foundation,
            custody_note=created.custody_note,
            redaction_note=created.redaction_note,
            relevance=created.relevance,
            issues=list(created.issues),
            objection_notes=created.objection_notes,
            readiness=created.readiness,
        )
        state.exhibit_annotations = [
            row for row in state.exhibit_annotations if row.item_id != created.item_id
        ]
        state.exhibit_annotations.append(annotation)
        self._write(state, action="exhibit-annotate")
        return next(row for row in self.list_exhibits() if row.item_id == created.item_id)

    def assign_bates(self, item_id: UUID) -> ExhibitCandidate:
        """Assign the next {prefix}-{n:06d} stamp. Does not invent evidence bytes."""
        current = next((row for row in self.list_exhibits() if row.item_id == item_id), None)
        if current is None:
            raise ValueError("exhibit candidates come only from approved package items")
        if current.bates_number:
            return current
        state = self.load()
        prefix = bates_prefix(state.matter.display_name)
        existing = [row.bates_number for row in self.list_exhibits() if row.bates_number]
        assigned = next_bates_number(prefix, existing)
        notes = {row.item_id: row for row in state.exhibit_annotations}
        prior = notes.get(item_id)
        created = ExhibitAnnotationCreate(
            item_id=item_id,
            exhibit_label=prior.exhibit_label if prior else "",
            bates_number=assigned,
            foundation=prior.foundation if prior else "",
            custody_note=prior.custody_note if prior else "",
            redaction_note=prior.redaction_note if prior else "",
            relevance=prior.relevance if prior else "",
            issues=list(prior.issues) if prior else [],
            objection_notes=prior.objection_notes if prior else "",
            readiness=prior.readiness if prior else ExhibitReadiness.CANDIDATE,
        )
        return self.annotate_exhibit(created)

    def add_discovery(self, created: DiscoveryCreate) -> DiscoveryRequest:
        state = self.load()
        request = DiscoveryRequest(
            kind=created.kind,
            text=created.text,
            purpose=created.purpose,
            linked_issue=created.linked_issue,
            missing_proof=created.missing_proof,
            target=created.target,
            authority=created.authority,
        )
        state.discovery.append(request)
        self._write(state, action=f"discovery-{created.kind.value}")
        return request

    def set_discovery_status(self, request_id: UUID, update: DiscoveryStatusUpdate) -> DiscoveryRequest:
        state = self.load()
        request = next(row for row in state.discovery if row.request_id == request_id)
        if update.status is DiscoveryStatus.SERVED and update.served_on is None:
            raise ValueError("served_on is required to mark a request served")
        request.status = update.status
        request.served_on = update.served_on
        self._write(state, action=f"discovery-{update.status.value}")
        return request

    def add_todo(self, created: TodoCreate) -> CaseTodo:
        state = self.load()
        todo = CaseTodo(title=created.title, detail=created.detail, source=created.source)
        state.todos.append(todo)
        self._write(state, action="todo-add")
        return todo

    def set_todo_status(self, todo_id: UUID, status: TodoStatus) -> CaseTodo:
        state = self.load()
        todo = next(row for row in state.todos if row.todo_id == todo_id)
        todo.status = status
        self._write(state, action=f"todo-{status.value}")
        return todo

    def patch_todo(self, todo_id: UUID, patch) -> CaseTodo:
        from legal_workspace.domain.todos import TodoPatch

        body = patch if isinstance(patch, TodoPatch) else TodoPatch.model_validate(patch)
        state = self.load()
        todo = next(row for row in state.todos if row.todo_id == todo_id)
        if body.title is not None:
            todo.title = body.title
        if body.detail is not None:
            todo.detail = body.detail
        if body.status is not None:
            todo.status = body.status
        self._write(state, action="todo-edit")
        return todo

    def patch_strategy_note(self, note_id: UUID, patch) -> StrategyNote:
        from legal_workspace.domain.strategy import StrategyPatch

        body = patch if isinstance(patch, StrategyPatch) else StrategyPatch.model_validate(patch)
        state = self.load()
        note = next(row for row in state.strategy_notes if row.note_id == note_id)
        if body.kind is not None:
            note.kind = body.kind
        if body.title is not None:
            note.title = body.title
        if body.body is not None:
            note.body = body.body
        self._write(state, action="strategy-edit")
        return note

    def _walk_issue(self, node: LegalIssue, issue_id: UUID) -> LegalIssue | None:
        if node.issue_id == issue_id:
            return node
        for child in node.children:
            hit = self._walk_issue(child, issue_id)
            if hit is not None:
                return hit
        return None

    def add_issue_child(self, created: IssueChildCreate) -> LegalIssue:
        state = self.load()
        child = LegalIssue(title=created.title.strip(), governing_authority=created.governing_authority.strip())
        if created.parent_id is None:
            state.issue.children.append(child)
        else:
            parent = self._walk_issue(state.issue, created.parent_id)
            if parent is None:
                raise StopIteration
            parent.children.append(child)
        self._write(state, action="issue-add")
        return child

    def patch_issue(self, issue_id: UUID, patch: IssuePatch) -> LegalIssue:
        state = self.load()
        node = self._walk_issue(state.issue, issue_id)
        if node is None:
            raise StopIteration
        if patch.title is not None:
            node.title = patch.title
        if patch.governing_authority is not None:
            node.governing_authority = patch.governing_authority
        self._write(state, action="issue-edit")
        return node

    def add_issue_element(self, issue_id: UUID, created: IssueElementCreate):
        from legal_workspace.domain.issue import IssueElement

        state = self.load()
        node = self._walk_issue(state.issue, issue_id)
        if node is None:
            raise StopIteration
        element = IssueElement(label=created.label.strip(), missing_proof=list(created.missing_proof))
        node.elements.append(element)
        self._write(state, action="issue-element-add")
        return element

    def add_factor_note(self, letter: FactorLetter, side: str, text: str) -> FactorEntry:
        state = self.load()
        entry = next(row for row in state.factors if row.letter is letter)
        note = text.strip()
        if not note:
            raise ValueError("note text is required")
        bucket = side.strip().lower()
        if bucket == "petitioner":
            entry.petitioner.for_parent.append(note)
        elif bucket == "respondent":
            entry.respondent.for_parent.append(note)
        elif bucket == "contradiction":
            entry.contradictions.append(note)
        elif bucket == "missing":
            entry.missing_proof.append(note)
        else:
            raise ValueError("side must be petitioner, respondent, contradiction, or missing")
        self._write(state, action=f"factor-{letter.value}-note")
        return entry

    def _section_hash(self, section: DraftSection, state: WorkspaceState) -> str:
        package = state.package
        return canonical_content_hash(
            {
                "matter_id": str(state.matter.matter_id),
                "package_id": str(package.package_id) if package else None,
                "package_manifest_hash": package.manifest_hash if package else None,
                "package_schema_version": package.schema_version if package else None,
                "section_id": str(section.section_id),
                "heading": section.heading,
                "body": section.body,
                "factor_letter": section.factor_letter.value if section.factor_letter else None,
                "citations": sorted(
                    (
                        str(item.package_id),
                        str(item.assertion_id),
                        item.assertion_version,
                        item.span_locator,
                    )
                    for item in section.citations
                ),
            }
        )

    def add_review(
        self,
        created: ReviewCreate,
        *,
        principal: AuthenticatedPrincipal,
    ) -> ReviewDecision:
        reviewer = require_human_review_actor(principal)
        state = self.load()
        section = next(row for row in state.drafts if row.section_id == created.section_id)
        if created.verdict is ReviewVerdict.APPROVE:
            if state.package is None:
                raise ValueError("import an approved LegalSourcePackage first")
            if created.matter_id is not None and created.matter_id != state.matter.matter_id:
                raise ValueError("review matter_id does not match this workspace")
            if created.package_id is not None and created.package_id != state.package.package_id:
                raise ValueError("review package_id does not match the imported package")
            if (
                created.manifest_hash is not None
                and created.manifest_hash != state.package.manifest_hash
            ):
                raise ValueError("review manifest_hash does not match the imported package")
            if (
                created.package_schema_version is not None
                and created.package_schema_version != state.package.schema_version
            ):
                raise ValueError(
                    "review package_schema_version does not match the imported package"
                )
            gate = validate_factual_citations(section.citations, state.package)
            if not gate.ok:
                raise ValueError("; ".join(gate.blockers))
            if not section.citations:
                raise ValueError("cannot approve a section with no factual citations")
        content_hash = self._section_hash(section, state)
        decision = ReviewDecision(
            section_id=section.section_id,
            work_product_id=state.work_product.work_product_id if state.work_product else None,
            verdict=created.verdict,
            rationale=created.rationale,
            reviewer=reviewer,
            content_hash=content_hash,
        )
        state.reviews.append(decision)
        if state.work_product and created.verdict is ReviewVerdict.APPROVE:
            state.work_product.state = WorkProductState.OWNER_APPROVED
            state.work_product.content_hash = content_hash
        elif state.work_product and created.verdict is ReviewVerdict.REQUEST_CHANGES:
            state.work_product.state = WorkProductState.REVIEW_REQUIRED
        self._write(state, action=f"review-{created.verdict.value}")
        return decision

    def latest_valid_approval(self, section: DraftSection) -> ReviewDecision | None:
        state = self.load()
        current = self._section_hash(section, state)
        for decision in reversed(state.reviews):
            if decision.section_id != section.section_id:
                continue
            if decision.verdict is not ReviewVerdict.APPROVE:
                return None
            if decision.content_hash != current:
                return None
            return decision
        return None

    def build_release_candidate(self, created: ReleaseCreate) -> ReleaseBuild:
        state = self.load()
        blockers: list[str] = []
        if state.package is None:
            blockers.append("import an approved LegalSourcePackage first")
        if not created.section_ids:
            blockers.append("select at least one draft section")
        sections: list[DraftSection] = []
        for section_id in created.section_ids:
            try:
                sections.append(next(row for row in state.drafts if row.section_id == section_id))
            except StopIteration:
                blockers.append(f"unknown draft section: {section_id}")
        if state.package is not None:
            for section in sections:
                gate = validate_factual_citations(section.citations, state.package)
                if not gate.ok:
                    blockers.extend(gate.blockers)
                if self.latest_valid_approval(section) is None:
                    blockers.append(f"owner approval required: {section.heading}")
        omitted_private = [
            "strategy_notes",
            "redteam_runs",
            "todos",
            "review_rationale",
        ]
        if blockers:
            if get_settings().debug_json:
                append_jsonl(
                    self.events_path,
                    {"action": "release-blocked", "blockers": blockers},
                )
            return ReleaseBuild(blocked=True, blockers=blockers)
        assert state.package is not None
        payload = {
            "package_id": str(state.package.package_id),
            "sections": [
                {
                    "section_id": str(section.section_id),
                    "heading": section.heading,
                    "body": section.body,
                    "factor_letter": section.factor_letter.value if section.factor_letter else None,
                    "citations": [item.model_dump(mode="json") for item in section.citations],
                }
                for section in sorted(sections, key=lambda row: str(row.section_id))
            ],
        }
        content_hash = canonical_content_hash(payload)
        cited_assertions = [
            item.assertion_id for section in sections for item in section.citations
        ]
        work_product_id = (
            state.work_product.work_product_id if state.work_product else sections[0].section_id
        )
        manifest = ReleaseManifest(
            work_product_id=work_product_id,
            section_ids=[section.section_id for section in sections],
            package_id=state.package.package_id,
            content_hash=content_hash,
            cited_assertion_ids=cited_assertions,
            cited_authority_ids=["MCL 722.23"],
            omitted_private=omitted_private,
        )
        state.releases.append(manifest)
        if state.work_product:
            state.work_product.state = WorkProductState.RELEASE_CANDIDATE
            state.work_product.content_hash = content_hash
        self._write(state, action="release-candidate")
        self._write_release_export(manifest, sections)
        return ReleaseBuild(blocked=False, blockers=[], manifest=manifest)

    def _write_release_export(self, manifest: ReleaseManifest, sections: list[DraftSection]) -> None:
        from legal_workspace.services.docx_export import write_release_docx

        export_dir = self.store_dir / "releases"
        atomic_write_json(export_dir / f"{manifest.release_id}.json", manifest.model_dump(mode="json"))
        body = [
            f"# Final review copy {manifest.release_id}",
            "",
            f"content_hash: {manifest.content_hash}",
            f"package_id: {manifest.package_id}",
            f"state: {manifest.state.value}",
            "filed: false",
            "",
            "This is not a filing and not court-safe.",
            "Omitted: " + ", ".join(manifest.omitted_private),
            "",
        ]
        for section in sections:
            body.extend([f"## {section.heading}", "", section.body, ""])
        (export_dir / f"{manifest.release_id}.md").write_text("\n".join(body), encoding="utf-8")
        write_release_docx(export_dir / f"{manifest.release_id}.docx", manifest, sections)

    def default_authority(self) -> AuthorityCitation:
        return AuthorityCitation(
            proposition="The court shall consider, evaluate, and determine the best-interest factors.",
            identifier="MCL 722.23",
            authority_level=AuthorityLevel.STATUTE,
            snapshot_hash="custody-packet:M2:mcl-722.23",
            jurisdiction="US-MI",
        )

    def set_confidential(self, on: bool) -> bool:
        """Persist Confidential Mode. Chrome is not a privilege conclusion."""
        return self.update_settings(confidential_mode=bool(on)).confidential_mode

    def update_settings(
        self,
        *,
        confidential_mode: bool | None = None,
        theme: str | None = None,
        display_timezone: str | None = None,
        case_phase: str | None = None,
    ) -> WorkspaceState:
        """SQLite legal_core_app_settings. Live PG 0001 stays HOLD."""
        row = sqlite_store.save_settings(
            self.store_dir,
            confidential_mode=confidential_mode,
            theme=theme,
            display_timezone=display_timezone,
            case_phase=case_phase,
        )
        state = self.load()
        state.confidential_mode = row.confidential_mode
        state.theme = row.theme
        state.display_timezone = row.display_timezone
        state.case_phase = row.case_phase
        self._write(state, action="settings")
        return state


_STORE: Workspace | None = None


def get_workspace(store_dir: Path | None = None) -> Workspace:
    global _STORE
    if store_dir is not None:
        _STORE = Workspace(store_dir)
        return _STORE
    if _STORE is None:
        _STORE = Workspace()
    return _STORE


# Compatibility name used by the API module.
WORKSPACE = get_workspace()

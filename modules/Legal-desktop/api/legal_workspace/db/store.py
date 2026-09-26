"""Workspace state store: maps the Pydantic WorkspaceState to SQLite/Postgres tables.

This is the intermediate persistence layer. It keeps the Workspace business logic
working on Pydantic aggregates while storing each aggregate in normalized tables
that mirror the PostgreSQL schema. When the app moves to PostgreSQL, only the
engine URL changes.

> _Byline: Claude Code · Kimi K2.7 · 2026-08-18_
"""

from __future__ import annotations

import json
import uuid
from datetime import UTC, datetime
from typing import Any

from sqlalchemy import select

from legal_workspace.contracts.identity import CourtCaseRef, MatterRef
from legal_workspace.db import models
from legal_workspace.db.engine import get_engine, is_sqlite
from legal_workspace.db.session import db_session
from legal_workspace.domain.agents import AgentRun, AgentRunStatus
from legal_workspace.domain.authority_library import CuratedAuthority
from legal_workspace.domain.calendar import DocketEvent
from legal_workspace.domain.discovery import DiscoveryRequest, DiscoveryStatus
from legal_workspace.domain.exhibits import ExhibitAnnotation, ExhibitReadiness
from legal_workspace.domain.factors import FactorCitationLink, FactorEntry, FactorLetter
from legal_workspace.domain.investigation import InvestigationRequest
from legal_workspace.domain.issue import LegalIssue
from legal_workspace.domain.redteam import RedTeamCreate, RedTeamRun
from legal_workspace.domain.release import ReleaseManifest
from legal_workspace.domain.research import CurrencyFlag, ResearchQuestion, ResearchStatus
from legal_workspace.domain.review import ReviewDecision
from legal_workspace.domain.strategy import StrategyNote
from legal_workspace.domain.todos import CaseTodo, TodoStatus
from legal_workspace.domain.work_product import WorkProductState, WorkProductVersion
from legal_workspace.domain.release import canonical_content_hash as _canonical_content_hash


def to_uuid(value: Any) -> uuid.UUID | None:
    if value is None:
        return None
    if isinstance(value, uuid.UUID):
        return value
    return uuid.UUID(str(value))


def _now() -> datetime:
    return datetime.now(UTC)


def _iso_utc(value: datetime | str | None) -> str | None:
    """Return an ISO-8601 string with explicit UTC offset, or None."""
    if value is None:
        return None
    if isinstance(value, str):
        dt = datetime.fromisoformat(value)
    else:
        dt = value
    if dt.tzinfo is None:
        dt = dt.replace(tzinfo=UTC)
    return dt.isoformat()


def create_tables(store_dir: str | None = None) -> None:
    """Create all tables on the current engine. Safe to call repeatedly."""
    models.Base.metadata.create_all(get_engine(store_dir=store_dir))


def _audit_event(
    *,
    action: str,
    matter_id: uuid.UUID,
    court_case_id: uuid.UUID | None,
    aggregate_id: uuid.UUID,
    aggregate_version: int,
    payload: dict[str, Any],
) -> models.LegalAuditEventOutbox:
    """Build an outbox row for every mutating action."""
    payload_text = json.dumps(payload, sort_keys=True, default=str)
    return models.LegalAuditEventOutbox(
        event_id=uuid.uuid4(),
        event_type=f"legal.{action}",
        schema_version="1",
        occurred_at=_now(),
        matter_id=matter_id,
        court_case_id=court_case_id,
        aggregate_id=aggregate_id,
        aggregate_version=aggregate_version,
        trace_id=str(uuid.uuid4())[:8],
        payload_hash=_canonical_content_hash(payload_text),
        payload=payload,
    )


def _factor_letter_from_value(value: str | None) -> FactorLetter | None:
    if value is None:
        return None
    return FactorLetter(value)


class WorkspaceStore:
    """Load and save WorkspaceState from/to the database."""

    @staticmethod
    def stored_matter_id(*, store_dir: str | None = None) -> uuid.UUID | None:
        """Return the active matter_id in this DB file, or None if empty.

        Single-case workspace: if more than one row exists (e.g. import/Agno
        residue), pick the most recently synced and let the next save clean up
        the rest.
        """
        create_tables(store_dir=store_dir)
        with db_session(store_dir=store_dir) as session:
            row = session.execute(
                select(models.LegalCoreMatterRef.matter_id)
                .order_by(models.LegalCoreMatterRef.synced_at.desc())
                .limit(1)
            ).scalar_one_or_none()
        return row

    @staticmethod
    def load(
        matter_id: uuid.UUID, *, store_dir: str | None = None
    ) -> dict[str, Any] | None:
        """Return a raw dict that WorkspaceState.model_validate can consume, or None."""
        create_tables(store_dir=store_dir)
        with db_session(store_dir=store_dir) as session:
            matter = session.get(models.LegalCoreMatterRef, matter_id)
            if matter is None:
                return None

            case = session.execute(
                select(models.LegalCoreCourtCaseRef).where(
                    models.LegalCoreCourtCaseRef.matter_id == matter_id
                )
            ).scalar_one_or_none()

            pkg = session.execute(
                select(models.LegalCoreSourcePackage).where(
                    models.LegalCoreSourcePackage.matter_id == matter_id
                )
            ).scalar_one_or_none()

            settings = session.get(models.LegalCoreAppSettings, 1)

            drafts = session.execute(
                select(models.LegalWorkProductDraftSection).where(
                    models.LegalWorkProductDraftSection.matter_id == matter_id
                )
            ).scalars().all()

            wp = session.execute(
                select(models.LegalWorkProductWorkProduct).where(
                    models.LegalWorkProductWorkProduct.matter_id == matter_id
                )
            ).scalar_one_or_none()

            wp_version = None
            if wp is not None:
                wp_version = session.execute(
                    select(models.LegalWorkProductWorkProductVersion)
                    .where(
                        models.LegalWorkProductWorkProductVersion.work_product_id
                        == wp.work_product_id
                    )
                    .order_by(models.LegalWorkProductWorkProductVersion.version.desc())
                ).scalar_one_or_none()

            authorities = session.execute(
                select(models.LegalResearchAuthority).where(
                    models.LegalResearchAuthority.matter_id == matter_id
                )
            ).scalars().all()

            strategy = session.execute(
                select(models.LegalWorkProductStrategyNote).where(
                    models.LegalWorkProductStrategyNote.matter_id == matter_id
                )
            ).scalars().all()

            redteam = session.execute(
                select(models.LegalWorkProductRedTeamRun).where(
                    models.LegalWorkProductRedTeamRun.matter_id == matter_id
                )
            ).scalars().all()

            todos = session.execute(
                select(models.LegalCoreTodo).where(models.LegalCoreTodo.matter_id == matter_id)
            ).scalars().all()

            reviews = session.execute(
                select(models.LegalWorkProductReviewDecision).where(
                    models.LegalWorkProductReviewDecision.matter_id == matter_id
                )
            ).scalars().all()

            releases = session.execute(
                select(models.LegalWorkProductReleaseManifest).where(
                    models.LegalWorkProductReleaseManifest.matter_id == matter_id
                )
            ).scalars().all()

            questions = session.execute(
                select(models.LegalResearchQuestion).where(
                    models.LegalResearchQuestion.matter_id == matter_id
                )
            ).scalars().all()

            flags = session.execute(
                select(models.LegalResearchCurrencyFlag).where(
                    models.LegalResearchCurrencyFlag.matter_id == matter_id
                )
            ).scalars().all()

            discovery = session.execute(
                select(models.LegalCoreDiscoveryRequest).where(
                    models.LegalCoreDiscoveryRequest.matter_id == matter_id
                )
            ).scalars().all()

            investigations = session.execute(
                select(models.LegalCoreInvestigationRequest).where(
                    models.LegalCoreInvestigationRequest.matter_id == matter_id
                )
            ).scalars().all()

            events = session.execute(
                select(models.LegalCoreDocketEvent).where(
                    models.LegalCoreDocketEvent.matter_id == matter_id
                )
            ).scalars().all()

            overrides = session.execute(
                select(models.LegalCoreFilingOverride).where(
                    models.LegalCoreFilingOverride.matter_id == matter_id
                )
            ).scalars().all()

            agent_runs = session.execute(
                select(models.LegalCoreAgentRun).where(
                    models.LegalCoreAgentRun.matter_id == matter_id
                )
            ).scalars().all()

            annotations = session.execute(
                select(models.LegalCoreExhibitAnnotation).where(
                    models.LegalCoreExhibitAnnotation.matter_id == matter_id
                )
            ).scalars().all()

            # Extract matter/case primitives while session is open.
            matter_id_value = matter.matter_id
            matter_display_name = matter.display_name
            matter_is_friendly_primary = matter.is_friendly_primary
            matter_issue_tree = matter.issue_tree
            matter_factor_matrix = matter.factor_matrix
            matter_last_agno_verify = matter.last_agno_verify

            case_court_case_id = case.court_case_id if case else None
            case_display_name = case.display_name if case else None
            case_docket_number = case.docket_number if case else None
            case_court = case.court if case else None
            case_is_primary = case.is_primary if case else None

            pkg_payload = pkg.payload if pkg else None
            settings_confidential = settings.confidential_mode if settings else False
            settings_theme = settings.theme if settings else "dark"
            settings_tz = settings.display_timezone if settings else "America/New_York"
            settings_phase = settings.case_phase if settings else "Discovery"

            # Package omissions
            omissions = []
            if pkg:
                omissions = [
                    str(row.item_id)
                    for row in session.execute(
                        select(models.LegalCoreSourcePackageOmission).where(
                            models.LegalCoreSourcePackageOmission.package_id == pkg.package_id
                        )
                    ).scalars().all()
                ]

            # Authorities
            authority_rows = []
            for row in authorities:
                authority_rows.append(
                    {
                        "authority_id": str(row.authority_id),
                        "identifier": row.identifier,
                        "proposition": row.proposition,
                        "authority_level": row.authority_level,
                        "court": row.court,
                        "pinpoint": row.pinpoint,
                        "source_path": row.source_path,
                        "complete": row.complete,
                        "applicable": row.applicable,
                        "skip_reason": row.skip_reason,
                        "is_citator_verified": row.is_citator_verified,
                        "snapshot_hash": row.snapshot_hash,
                        "jurisdiction": row.jurisdiction,
                        "source_url": row.source_url,
                        "retrieved_at": _iso_utc(row.retrieved_at),
                        "subsequent_history": row.subsequent_history,
                        "last_currentness_check": _iso_utc(row.last_currentness_check),
                    }
                )

            # Drafts
            draft_rows = []
            for row in drafts:
                draft_rows.append(
                    {
                        "section_id": str(row.section_id),
                        "heading": row.heading,
                        "body": row.body,
                        "factor_letter": row.factor_letter,
                        "citations": row.citations,
                    }
                )

            # Work product version
            work_product = None
            if wp and wp_version:
                work_product = {
                    "work_product_id": str(wp.work_product_id),
                    "state": wp_version.state,
                    "source_package_id": (
                        str(wp_version.source_package_id) if wp_version.source_package_id else None
                    ),
                    "content_hash": wp_version.content_hash,
                    "body_md": wp_version.body_md,
                    "cited_assertion_ids": [str(uid) for uid in wp_version.cited_assertion_ids],
                    "cited_authority_ids": wp_version.cited_authority_ids,
                }

            # Strategy notes
            strategy_rows = []
            for row in strategy:
                strategy_rows.append(
                    {
                        "note_id": str(row.note_id),
                        "kind": row.kind,
                        "title": row.title,
                        "body": row.body,
                        "source_chat": row.source_chat,
                    }
                )

            # Red team runs
            redteam_rows = []
            for row in redteam:
                redteam_rows.append(
                    {
                        "run_id": str(row.run_id),
                        "target_type": row.target_type,
                        "target_id": row.target_id,
                        "lens": row.lens,
                        "prompt_or_notes": row.prompt_or_notes,
                        "findings": row.findings or [],
                        "model": row.model,
                    }
                )

            # Todos
            todo_rows = []
            for row in todos:
                todo_rows.append(
                    {
                        "todo_id": str(row.todo_id),
                        "title": row.title,
                        "detail": row.detail,
                        "source": row.source,
                        "status": row.status,
                    }
                )

            # Reviews
            review_rows = []
            for row in reviews:
                review_rows.append(
                    {
                        "review_id": str(row.review_id),
                        "section_id": str(row.section_id),
                        "work_product_id": str(row.work_product_id) if row.work_product_id else None,
                        "verdict": row.verdict,
                        "rationale": row.rationale,
                        "reviewer": row.reviewer,
                        "content_hash": row.content_hash,
                        "created_at": _iso_utc(row.created_at),
                        "disclosure": row.disclosure,
                        "court_safe": row.court_safe,
                        "exportable": row.exportable,
                    }
                )

            # Releases
            release_rows = []
            for row in releases:
                release_rows.append(
                    {
                        "release_id": str(row.release_id),
                        "work_product_id": str(row.work_product_id),
                        "section_ids": [str(uid) for uid in row.section_ids],
                        "package_id": str(row.package_id),
                        "content_hash": row.content_hash,
                        "cited_assertion_ids": [str(uid) for uid in row.cited_assertion_ids],
                        "cited_authority_ids": row.cited_authority_ids,
                        "omitted_private": row.omitted_private,
                        "state": row.state,
                        "filed": row.filed,
                    }
                )

            # Research questions
            question_rows = []
            for row in questions:
                question_rows.append(
                    {
                        "question_id": str(row.question_id),
                        "question": row.question,
                        "plan": row.plan,
                        "adverse_notes": row.adverse_notes,
                        "uncertainty": row.uncertainty,
                        "authority_ids": row.authority_ids,
                        "jurisdiction": row.jurisdiction,
                        "status": row.status,
                    }
                )

            # Currency flags
            flag_rows = []
            for row in flags:
                flag_rows.append(
                    {
                        "flag_id": str(row.flag_id),
                        "identifier": row.identifier,
                        "note": row.note,
                    }
                )

            # Discovery
            discovery_rows = []
            for row in discovery:
                discovery_rows.append(
                    {
                        "request_id": str(row.request_id),
                        "kind": row.kind,
                        "text": row.text,
                        "purpose": row.purpose,
                        "linked_issue": row.linked_issue,
                        "missing_proof": row.missing_proof,
                        "target": row.target,
                        "authority": row.authority,
                        "status": row.status,
                        "served_on": _iso_utc(row.served_on),
                    }
                )

            # Investigations
            investigation_rows = []
            for row in investigations:
                investigation_rows.append(
                    {
                        "request_id": str(row.request_id),
                        "needed": row.needed,
                        "why": row.why,
                        "kind": row.kind,
                        "linked_issue": row.linked_issue,
                        "factor_letter": row.factor_letter,
                        "contradiction": row.contradiction,
                        "existing_assertion_ids": row.existing_assertion_ids,
                        "event_envelope": row.envelope,
                    }
                )

            # Docket events
            event_rows = []
            for row in events:
                event_rows.append(
                    {
                        "event_id": str(row.event_id),
                        "occurs_at": _iso_utc(row.occurs_at),
                        "title": row.title,
                        "kind": row.kind,
                        "detail": row.detail,
                        "location": row.location,
                        "source": row.source,
                        "confirmed": row.confirmed,
                        "status": row.status,
                    }
                )

            # Filing overrides
            override_rows = []
            for row in overrides:
                override_rows.append(
                    {
                        "override_id": str(row.override_id),
                        "check_id": row.check_id,
                        "state": row.state,
                        "note": row.note,
                        "reviewer": row.reviewer,
                    }
                )

            # Agent runs
            agent_run_rows = []
            for row in agent_runs:
                agent_run_rows.append(
                    {
                        "run_id": str(row.run_id),
                        "role": row.role,
                        "intent": row.intent,
                        "prompt": row.prompt,
                        "requested_model": row.requested_model,
                        "effective_model": row.effective_model,
                        "target_type": row.target_type,
                        "target_id": row.target_id,
                        "output": row.output,
                        "status": row.status,
                    }
                )

            # Exhibit annotations
            annotation_rows = []
            for row in annotations:
                annotation_rows.append(
                    {
                        "annotation_id": str(row.annotation_id),
                        "item_id": str(row.item_id),
                        "exhibit_label": row.exhibit_label,
                        "bates_number": row.bates_number,
                        "foundation": row.foundation,
                        "custody_note": row.custody_note,
                        "redaction_note": row.redaction_note,
                        "relevance": row.relevance,
                        "issues": row.issues,
                        "objection_notes": row.objection_notes,
                        "readiness": row.readiness,
                    }
                )

        # Reconstruct WorkspaceState dict from extracted primitives.
        state: dict[str, Any] = {
            "matter": {
                "matter_id": str(matter_id_value),
                "display_name": matter_display_name,
                "is_friendly_primary": matter_is_friendly_primary,
            },
            "court_case": {
                "court_case_id": str(case_court_case_id) if case_court_case_id else str(uuid.uuid4()),
                "matter_id": str(matter_id_value),
                "display_name": case_display_name or "",
                "docket_number": case_docket_number,
                "court": case_court,
                "is_primary": case_is_primary if case_is_primary is not None else True,
            },
            "package": pkg_payload,
            "omitted_item_ids": omissions,
            "issue": matter_issue_tree or {"title": "", "governing_authority": "", "children": [], "elements": []},
            "factors": matter_factor_matrix or [],
            "drafts": draft_rows,
            "work_product": work_product,
            "authorities": authority_rows,
            "strategy_notes": strategy_rows,
            "redteam_runs": redteam_rows,
            "todos": todo_rows,
            "reviews": review_rows,
            "releases": release_rows,
            "research_questions": question_rows,
            "currency_flags": flag_rows,
            "discovery": discovery_rows,
            "investigations": investigation_rows,
            "docket_events": event_rows,
            "filing_overrides": override_rows,
            "agent_runs": agent_run_rows,
            "exhibit_annotations": annotation_rows,
            "last_agno_verify": matter_last_agno_verify or [],
            "confidential_mode": settings_confidential,
            "theme": settings_theme,
            "display_timezone": settings_tz,
            "case_phase": settings_phase,
        }

        return state

    @staticmethod
    def save(
        state_dict: dict[str, Any], action: str, *, store_dir: str | None = None
    ) -> None:
        """Persist a WorkspaceState dict to the database and append an audit event."""
        create_tables(store_dir=store_dir)
        matter_id = to_uuid(state_dict["matter"]["matter_id"])
        court_case_id = to_uuid(state_dict["court_case"].get("court_case_id"))
        assert matter_id is not None

        with db_session(store_dir=store_dir) as session:
            # Snapshot the stored matter before we replace anything. Single-case
            # workspace: pick the most recently synced row if more than one exists.
            stored_matter_id = session.execute(
                select(models.LegalCoreMatterRef.matter_id)
                .order_by(models.LegalCoreMatterRef.synced_at.desc())
                .limit(1)
            ).scalar_one_or_none()

            # Replace the entire workspace contents. Delete dependent rows before
            # their parents so foreign-key constraints stay satisfied.
            for model in (
                models.LegalWorkProductDraftSection,
                models.LegalWorkProductStrategyNote,
                models.LegalWorkProductRedTeamRun,
                models.LegalWorkProductReviewDecision,
                models.LegalWorkProductReleaseManifest,
                models.LegalCoreTodo,
                models.LegalCoreDocketEvent,
                models.LegalCoreDiscoveryRequest,
                models.LegalCoreInvestigationRequest,
                models.LegalCoreExhibitAnnotation,
                models.LegalCoreAgentRun,
                models.LegalCoreFilingOverride,
                models.LegalResearchAuthority,
                models.LegalResearchCurrencyFlag,
                models.LegalResearchQuestion,
            ):
                session.execute(model.__table__.delete())

            session.execute(models.LegalWorkProductWorkProductVersion.__table__.delete())
            session.execute(models.LegalWorkProductWorkProduct.__table__.delete())
            session.execute(models.LegalCoreSourcePackageOmission.__table__.delete())
            session.execute(models.LegalCoreSourcePackage.__table__.delete())
            session.execute(models.LegalCoreCourtCaseRef.__table__.delete())

            # If the matter identity changed, remove the old matter row after its
            # dependents are gone. The new matter row is upserted below.
            if stored_matter_id is not None and stored_matter_id != matter_id:
                session.execute(
                    models.LegalCoreMatterRef.__table__.delete().where(
                        models.LegalCoreMatterRef.matter_id == stored_matter_id
                    )
                )

            session.commit()

            # Upsert matter
            matter = session.get(models.LegalCoreMatterRef, matter_id)
            issue_tree = state_dict.get("issue")
            factor_matrix = state_dict.get("factors")
            if matter is None:
                matter = models.LegalCoreMatterRef(
                    matter_id=matter_id,
                    display_name=state_dict["matter"]["display_name"],
                    is_friendly_primary=state_dict["matter"].get("is_friendly_primary", True),
                    issue_tree=issue_tree,
                    factor_matrix=factor_matrix,
                    last_agno_verify=state_dict.get("last_agno_verify"),
                )
                session.add(matter)
            else:
                matter.display_name = state_dict["matter"]["display_name"]
                matter.is_friendly_primary = state_dict["matter"].get(
                    "is_friendly_primary", True
                )
                matter.issue_tree = issue_tree
                matter.factor_matrix = factor_matrix
                matter.last_agno_verify = state_dict.get("last_agno_verify")
                matter.synced_at = _now()

            # Upsert court case by its own id (matter_id may change during Agno projection).
            case_data = state_dict["court_case"]
            case_id = to_uuid(case_data.get("court_case_id")) or uuid.uuid4()
            existing_case = session.get(models.LegalCoreCourtCaseRef, case_id)
            if existing_case is None:
                session.add(
                    models.LegalCoreCourtCaseRef(
                        court_case_id=case_id,
                        matter_id=matter_id,
                        display_name=case_data.get("display_name", ""),
                        docket_number=case_data.get("docket_number"),
                        court=case_data.get("court"),
                        is_primary=case_data.get("is_primary", True),
                    )
                )
            else:
                existing_case.matter_id = matter_id
                existing_case.display_name = case_data.get("display_name", "")
                existing_case.docket_number = case_data.get("docket_number")
                existing_case.court = case_data.get("court")
                existing_case.synced_at = _now()

            # Source package
            pkg_data = state_dict.get("package")
            if pkg_data:
                pkg_id = to_uuid(pkg_data["package_id"])
                existing_pkg = session.get(models.LegalCoreSourcePackage, pkg_id)
                if existing_pkg is None:
                    session.add(
                        models.LegalCoreSourcePackage(
                            package_id=pkg_id,
                            schema_version=pkg_data.get("schema_version", "1"),
                            manifest_hash=pkg_data.get("manifest_hash", ""),
                            matter_id=matter_id,
                            court_case_id=court_case_id,
                            payload=pkg_data,
                        )
                    )
                else:
                    existing_pkg.payload = pkg_data

                # Omissions
                session.execute(
                    models.LegalCoreSourcePackageOmission.__table__.delete().where(
                        models.LegalCoreSourcePackageOmission.package_id == pkg_id
                    )
                )
                for item_id in state_dict.get("omitted_item_ids", []):
                    session.add(
                        models.LegalCoreSourcePackageOmission(
                            package_id=pkg_id,
                            item_id=to_uuid(item_id),
                            reason="omitted",
                        )
                    )

            # Settings
            settings = session.get(models.LegalCoreAppSettings, 1)
            if settings is None:
                session.add(
                    models.LegalCoreAppSettings(
                        id=1,
                        theme=state_dict.get("theme", "dark"),
                        display_timezone=state_dict.get("display_timezone", "America/New_York"),
                        confidential_mode=state_dict.get("confidential_mode", False),
                        case_phase=state_dict.get("case_phase", "Discovery"),
                    )
                )
            else:
                settings.theme = state_dict.get("theme", "dark")
                settings.display_timezone = state_dict.get(
                    "display_timezone", "America/New_York"
                )
                settings.confidential_mode = state_dict.get("confidential_mode", False)
                settings.case_phase = state_dict.get("case_phase", "Discovery")
                settings.updated_at = _now()


            # Drafts
            for draft in state_dict.get("drafts", []):
                session.add(
                    models.LegalWorkProductDraftSection(
                        section_id=to_uuid(draft["section_id"]),
                        matter_id=matter_id,
                        heading=draft["heading"],
                        body=draft["body"],
                        factor_letter=draft.get("factor_letter"),
                        citations=draft.get("citations", []),
                    )
                )

            # Work product
            wp_data = state_dict.get("work_product")
            if wp_data:
                wp_id = to_uuid(wp_data["work_product_id"]) or uuid.uuid4()
                session.add(
                    models.LegalWorkProductWorkProduct(
                        work_product_id=wp_id,
                        matter_id=matter_id,
                        court_case_id=court_case_id,
                        kind="motion",
                        title="Motion",
                    )
                )
                session.add(
                    models.LegalWorkProductWorkProductVersion(
                        work_product_id=wp_id,
                        version=1,
                        state=wp_data.get("state", "private_draft"),
                        source_package_id=to_uuid(wp_data.get("source_package_id")),
                        content_hash=wp_data.get("content_hash", ""),
                        body_md=wp_data.get("body_md"),
                        cited_assertion_ids=[
                            to_uuid(uid) for uid in wp_data.get("cited_assertion_ids", [])
                        ],
                        cited_authority_ids=wp_data.get("cited_authority_ids", []),
                    )
                )

            # Authorities
            for auth in state_dict.get("authorities", []):
                session.add(
                    models.LegalResearchAuthority(
                        authority_id=to_uuid(auth.get("authority_id")) or uuid.uuid4(),
                        matter_id=matter_id,
                        identifier=auth["identifier"],
                        proposition=auth["proposition"],
                        authority_level=auth.get("authority_level", "statute"),
                        court=auth.get("court"),
                        pinpoint=auth.get("pinpoint"),
                        source_path=auth.get("source_path", ""),
                        complete=auth.get("complete", True),
                        applicable=auth.get("applicable", True),
                        skip_reason=auth.get("skip_reason"),
                        is_citator_verified=auth.get("is_citator_verified", False),
                        snapshot_hash=auth.get("snapshot_hash", ""),
                        jurisdiction=auth.get("jurisdiction"),
                        source_url=auth.get("source_url"),
                        retrieved_at=(
                            datetime.fromisoformat(auth["retrieved_at"])
                            if auth.get("retrieved_at")
                            else None
                        ),
                        subsequent_history=auth.get("subsequent_history"),
                        last_currentness_check=(
                            datetime.fromisoformat(auth["last_currentness_check"])
                            if auth.get("last_currentness_check")
                            else None
                        ),
                    )
                )

            # Strategy notes
            for note in state_dict.get("strategy_notes", []):
                session.add(
                    models.LegalWorkProductStrategyNote(
                        note_id=to_uuid(note.get("note_id")) or uuid.uuid4(),
                        matter_id=matter_id,
                        kind=note.get("kind", "note"),
                        title=note["title"],
                        body=note["body"],
                        source_chat=note.get("source_chat"),
                    )
                )

            # Red team runs
            for run in state_dict.get("redteam_runs", []):
                session.add(
                    models.LegalWorkProductRedTeamRun(
                        run_id=to_uuid(run.get("run_id")) or uuid.uuid4(),
                        matter_id=matter_id,
                        target_type=run["target_type"],
                        target_id=run["target_id"],
                        lens=run["lens"],
                        prompt_or_notes=run.get("prompt_or_notes"),
                        findings=run.get("findings", []),
                        model=run.get("model"),
                    )
                )

            # Todos
            for todo in state_dict.get("todos", []):
                session.add(
                    models.LegalCoreTodo(
                        todo_id=to_uuid(todo.get("todo_id")) or uuid.uuid4(),
                        matter_id=matter_id,
                        title=todo["title"],
                        detail=todo.get("detail"),
                        source=todo.get("source"),
                        status=todo.get("status", "open"),
                    )
                )

            # Reviews
            for review in state_dict.get("reviews", []):
                session.add(
                    models.LegalWorkProductReviewDecision(
                        review_id=to_uuid(review.get("review_id")) or uuid.uuid4(),
                        matter_id=matter_id,
                        section_id=to_uuid(review["section_id"]),
                        work_product_id=to_uuid(review.get("work_product_id")),
                        verdict=review["verdict"],
                        rationale=review.get("rationale"),
                        reviewer=review["reviewer"],
                        content_hash=review["content_hash"],
                        created_at=(
                            datetime.fromisoformat(review["created_at"])
                            if review.get("created_at")
                            else None
                        ),
                        disclosure=review.get("disclosure", "work_product_claimed"),
                        court_safe=review.get("court_safe", False),
                        exportable=review.get("exportable", False),
                    )
                )

            # Releases
            for release in state_dict.get("releases", []):
                session.add(
                    models.LegalWorkProductReleaseManifest(
                        release_id=to_uuid(release["release_id"]),
                        matter_id=matter_id,
                        work_product_id=to_uuid(release["work_product_id"]),
                        section_ids=[to_uuid(uid) for uid in release["section_ids"]],
                        package_id=to_uuid(release["package_id"]),
                        content_hash=release["content_hash"],
                        cited_assertion_ids=[
                            to_uuid(uid) for uid in release.get("cited_assertion_ids", [])
                        ],
                        cited_authority_ids=release.get("cited_authority_ids", []),
                        omitted_private=release.get("omitted_private", []),
                        state=release.get("state", "release_candidate"),
                        filed=release.get("filed", False),
                    )
                )

            # Research questions
            for q in state_dict.get("research_questions", []):
                session.add(
                    models.LegalResearchQuestion(
                        question_id=to_uuid(q.get("question_id")) or uuid.uuid4(),
                        matter_id=matter_id,
                        question=q["question"],
                        plan=q.get("plan"),
                        adverse_notes=q.get("adverse_notes"),
                        uncertainty=q.get("uncertainty"),
                        authority_ids=q.get("authority_ids", []),
                        jurisdiction=q.get("jurisdiction"),
                        status=q.get("status", "open"),
                    )
                )

            # Currency flags
            for flag in state_dict.get("currency_flags", []):
                session.add(
                    models.LegalResearchCurrencyFlag(
                        flag_id=to_uuid(flag.get("flag_id")) or uuid.uuid4(),
                        matter_id=matter_id,
                        identifier=flag["identifier"],
                        note=flag["note"],
                    )
                )

            # Discovery
            for d in state_dict.get("discovery", []):
                session.add(
                    models.LegalCoreDiscoveryRequest(
                        request_id=to_uuid(d.get("request_id")) or uuid.uuid4(),
                        matter_id=matter_id,
                        kind=d["kind"],
                        text=d["text"],
                        purpose=d.get("purpose"),
                        linked_issue=d.get("linked_issue"),
                        missing_proof=d.get("missing_proof"),
                        target=d.get("target"),
                        authority=d.get("authority"),
                        status=d.get("status", "draft"),
                        served_on=(
                            datetime.fromisoformat(d["served_on"]) if d.get("served_on") else None
                        ),
                    )
                )

            # Investigations
            for inv in state_dict.get("investigations", []):
                session.add(
                    models.LegalCoreInvestigationRequest(
                        request_id=to_uuid(inv.get("request_id")) or uuid.uuid4(),
                        matter_id=matter_id,
                        needed=inv["needed"],
                        why=inv.get("why"),
                        kind=inv.get("kind", "missing_proof"),
                        linked_issue=inv.get("linked_issue"),
                        factor_letter=inv.get("factor_letter"),
                        contradiction=inv.get("contradiction"),
                        existing_assertion_ids=inv.get("existing_assertion_ids", []),
                        envelope=inv.get("event_envelope"),
                    )
                )

            # Docket events
            for e in state_dict.get("docket_events", []):
                session.add(
                    models.LegalCoreDocketEvent(
                        event_id=to_uuid(e.get("event_id")) or uuid.uuid4(),
                        matter_id=matter_id,
                        court_case_id=court_case_id,
                        occurs_at=datetime.fromisoformat(e["occurs_at"]),
                        title=e["title"],
                        kind=e["kind"],
                        detail=e.get("detail"),
                        location=e.get("location"),
                        source=e.get("source"),
                        confirmed=e.get("confirmed", False),
                        status=e.get("status"),
                    )
                )

            # Filing overrides
            for fo in state_dict.get("filing_overrides", []):
                session.add(
                    models.LegalCoreFilingOverride(
                        override_id=to_uuid(fo.get("override_id")) or uuid.uuid4(),
                        matter_id=matter_id,
                        check_id=fo["check_id"],
                        state=fo["state"],
                        note=fo.get("note"),
                        reviewer=fo["reviewer"],
                    )
                )

            # Agent runs
            for run in state_dict.get("agent_runs", []):
                session.add(
                    models.LegalCoreAgentRun(
                        run_id=to_uuid(run.get("run_id")) or uuid.uuid4(),
                        matter_id=matter_id,
                        role=run["role"],
                        intent=run["intent"],
                        prompt=run["prompt"],
                        requested_model=run["requested_model"],
                        effective_model=run["effective_model"],
                        target_type=run.get("target_type"),
                        target_id=run.get("target_id"),
                        output=run["output"],
                        status=run["status"],
                    )
                )

            # Exhibit annotations
            for ann in state_dict.get("exhibit_annotations", []):
                session.add(
                    models.LegalCoreExhibitAnnotation(
                        annotation_id=to_uuid(ann.get("annotation_id")) or uuid.uuid4(),
                        matter_id=matter_id,
                        item_id=to_uuid(ann["item_id"]),
                        exhibit_label=ann.get("exhibit_label", ""),
                        bates_number=ann.get("bates_number", ""),
                        foundation=ann.get("foundation"),
                        custody_note=ann.get("custody_note"),
                        redaction_note=ann.get("redaction_note"),
                        relevance=ann.get("relevance"),
                        issues=ann.get("issues", []),
                        objection_notes=ann.get("objection_notes"),
                        readiness=ann.get("readiness", "candidate"),
                    )
                )

            # Audit event
            audit = _audit_event(
                action=action,
                matter_id=matter_id,
                court_case_id=court_case_id,
                aggregate_id=matter_id,
                aggregate_version=1,
                payload={
                    "action": action,
                    "matter_id": str(matter_id),
                    "draft_count": len(state_dict.get("drafts", [])),
                    "strategy_count": len(state_dict.get("strategy_notes", [])),
                    "redteam_count": len(state_dict.get("redteam_runs", [])),
                    "review_count": len(state_dict.get("reviews", [])),
                    "release_count": len(state_dict.get("releases", [])),
                    "docket_count": len(state_dict.get("docket_events", [])),
                    "investigation_count": len(state_dict.get("investigations", [])),
                    "open_todos": sum(
                        1 for t in state_dict.get("todos", []) if t.get("status") == "open"
                    ),
                    "open_research": sum(
                        1
                        for q in state_dict.get("research_questions", [])
                        if q.get("status") in {"open", "unresolved"}
                    ),
                },
            )
            session.add(audit)

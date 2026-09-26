"""SQLAlchemy models mirroring the PostgreSQL schema shape.

Table names use schema prefixes (`legal_core_*`, `legal_research_*`, etc.) because
SQLite does not support `CREATE SCHEMA`. When the app moves to PostgreSQL, the
prefixes can be remapped to real schemas with no model changes.

> _Byline: Claude Code · Kimi K2.7 · 2026-08-18_
"""

from __future__ import annotations

import json
import uuid
from datetime import datetime
from typing import Any

from sqlalchemy import (
    Boolean,
    DateTime,
    ForeignKey,
    Integer,
    String,
    Text,
    TypeDecorator,
    func,
)
from sqlalchemy.dialects.postgresql import JSONB, UUID as PG_UUID
from sqlalchemy.orm import Mapped, mapped_column, relationship

from legal_workspace.db.base import Base


class UUIDType(TypeDecorator[Any]):
    """Platform-neutral UUID column: CHAR(36) on SQLite, native UUID on Postgres."""

    impl = String(36)
    cache_ok = True

    def load_dialect_impl(self, dialect: Any) -> Any:
        if dialect.name == "postgresql":
            return dialect.type_descriptor(PG_UUID(as_uuid=True))
        return dialect.type_descriptor(String(36))

    def process_bind_param(self, value: Any | None, dialect: Any) -> Any:
        if value is None:
            return None
        if dialect.name == "postgresql":
            return value
        return str(value)

    def process_result_value(self, value: Any | None, dialect: Any) -> Any:
        if value is None:
            return None
        if isinstance(value, uuid.UUID):
            return value
        return uuid.UUID(str(value))


class JSONDocument(TypeDecorator[Any]):
    """JSON document: JSONB on Postgres, TEXT JSON on SQLite."""

    impl = Text
    cache_ok = True

    def load_dialect_impl(self, dialect: Any) -> Any:
        if dialect.name == "postgresql":
            return dialect.type_descriptor(JSONB)
        return dialect.type_descriptor(Text)

    def process_bind_param(self, value: Any | None, dialect: Any) -> Any | None:
        if value is None:
            return None
        return json.dumps(value, default=str)

    def process_result_value(self, value: Any | None, dialect: Any) -> Any | None:
        if value is None:
            return None
        if isinstance(value, (dict, list)):
            return value
        if isinstance(value, bytes):
            value = value.decode("utf-8")
        return json.loads(value)

    def copy(self, **kw: Any) -> "JSONDocument":
        return JSONDocument(**kw)


# ---------------------------------------------------------------------------
# legal_core
# ---------------------------------------------------------------------------


class LegalCoreAppSettings(Base):
    __tablename__ = "legal_core_app_settings"

    id: Mapped[bool] = mapped_column(Integer, primary_key=True, default=1)
    theme: Mapped[str] = mapped_column(String, default="dark")
    display_timezone: Mapped[str] = mapped_column(String, default="America/New_York")
    confidential_mode: Mapped[bool] = mapped_column(Boolean, default=False)
    case_phase: Mapped[str] = mapped_column(String, default="Discovery")
    updated_at: Mapped[datetime] = mapped_column(DateTime, server_default=func.now())


class LegalCoreMatterRef(Base):
    __tablename__ = "legal_core_matter_ref"

    matter_id: Mapped[uuid.UUID] = mapped_column(UUIDType, primary_key=True)
    display_name: Mapped[str] = mapped_column(String, nullable=False)
    is_friendly_primary: Mapped[bool] = mapped_column(Boolean, default=True)
    source_revision: Mapped[str | None] = mapped_column(String, nullable=True)
    synced_at: Mapped[datetime] = mapped_column(DateTime, server_default=func.now())
    # Nested domain structures stored as JSONB until fully normalized.
    issue_tree: Mapped[dict[str, Any] | None] = mapped_column(JSONDocument, nullable=True)
    factor_matrix: Mapped[list[dict[str, Any]] | None] = mapped_column(JSONDocument, nullable=True)
    last_agno_verify: Mapped[list[dict[str, Any]] | None] = mapped_column(JSONDocument, nullable=True)

    court_cases: Mapped[list["LegalCoreCourtCaseRef"]] = relationship(
        back_populates="matter", cascade="all, delete-orphan"
    )


class LegalCoreCourtCaseRef(Base):
    __tablename__ = "legal_core_court_case_ref"

    court_case_id: Mapped[uuid.UUID] = mapped_column(UUIDType, primary_key=True)
    matter_id: Mapped[uuid.UUID] = mapped_column(
        UUIDType, ForeignKey("legal_core_matter_ref.matter_id"), nullable=False
    )
    display_name: Mapped[str] = mapped_column(String, nullable=False)
    docket_number: Mapped[str | None] = mapped_column(String, nullable=True)
    court: Mapped[str | None] = mapped_column(String, nullable=True)
    is_primary: Mapped[bool] = mapped_column(Boolean, default=True)
    synced_at: Mapped[datetime] = mapped_column(DateTime, server_default=func.now())

    matter: Mapped["LegalCoreMatterRef"] = relationship(back_populates="court_cases")


class LegalCoreSourcePackage(Base):
    __tablename__ = "legal_core_source_package"

    package_id: Mapped[uuid.UUID] = mapped_column(UUIDType, primary_key=True)
    schema_version: Mapped[str] = mapped_column(String, nullable=False)
    manifest_hash: Mapped[str] = mapped_column(String, nullable=False)
    matter_id: Mapped[uuid.UUID] = mapped_column(UUIDType, nullable=False)
    court_case_id: Mapped[uuid.UUID | None] = mapped_column(UUIDType, nullable=True)
    payload: Mapped[dict[str, Any]] = mapped_column(JSONDocument, nullable=False)
    imported_at: Mapped[datetime] = mapped_column(DateTime, server_default=func.now())


class LegalCoreSourcePackageOmission(Base):
    __tablename__ = "legal_core_source_package_omission"

    package_id: Mapped[uuid.UUID] = mapped_column(
        UUIDType, ForeignKey("legal_core_source_package.package_id"), primary_key=True
    )
    item_id: Mapped[uuid.UUID] = mapped_column(UUIDType, primary_key=True)
    reason: Mapped[str] = mapped_column(String, nullable=False)


class LegalCoreDocketEvent(Base):
    __tablename__ = "legal_core_docket_event"

    event_id: Mapped[uuid.UUID] = mapped_column(UUIDType, primary_key=True)
    matter_id: Mapped[uuid.UUID] = mapped_column(UUIDType, nullable=False, index=True)
    court_case_id: Mapped[uuid.UUID | None] = mapped_column(UUIDType, nullable=True)
    occurs_at: Mapped[datetime] = mapped_column(DateTime, nullable=False)
    title: Mapped[str] = mapped_column(String, nullable=False)
    kind: Mapped[str] = mapped_column(String, nullable=False)
    detail: Mapped[str | None] = mapped_column(Text, nullable=True)
    location: Mapped[str | None] = mapped_column(String, nullable=True)
    source: Mapped[str | None] = mapped_column(String, nullable=True)
    confirmed: Mapped[bool] = mapped_column(Boolean, default=False)
    status: Mapped[str | None] = mapped_column(String, nullable=True)


class LegalCoreTodo(Base):
    __tablename__ = "legal_core_todo"

    todo_id: Mapped[uuid.UUID] = mapped_column(UUIDType, primary_key=True)
    matter_id: Mapped[uuid.UUID] = mapped_column(UUIDType, nullable=False, index=True)
    title: Mapped[str] = mapped_column(String, nullable=False)
    detail: Mapped[str | None] = mapped_column(Text, nullable=True)
    source: Mapped[str | None] = mapped_column(String, nullable=True)
    status: Mapped[str] = mapped_column(String, default="open")


class LegalCoreDiscoveryRequest(Base):
    __tablename__ = "legal_core_discovery_request"

    request_id: Mapped[uuid.UUID] = mapped_column(UUIDType, primary_key=True)
    matter_id: Mapped[uuid.UUID] = mapped_column(UUIDType, nullable=False, index=True)
    kind: Mapped[str] = mapped_column(String, nullable=False)
    text: Mapped[str] = mapped_column(Text, nullable=False)
    purpose: Mapped[str | None] = mapped_column(Text, nullable=True)
    linked_issue: Mapped[str | None] = mapped_column(String, nullable=True)
    missing_proof: Mapped[str | None] = mapped_column(Text, nullable=True)
    target: Mapped[str | None] = mapped_column(String, nullable=True)
    authority: Mapped[str | None] = mapped_column(String, nullable=True)
    status: Mapped[str] = mapped_column(String, default="draft")
    served_on: Mapped[datetime | None] = mapped_column(DateTime, nullable=True)


class LegalCoreInvestigationRequest(Base):
    __tablename__ = "legal_core_investigation_request"

    request_id: Mapped[uuid.UUID] = mapped_column(UUIDType, primary_key=True)
    matter_id: Mapped[uuid.UUID] = mapped_column(UUIDType, nullable=False, index=True)
    needed: Mapped[str] = mapped_column(Text, nullable=False)
    why: Mapped[str | None] = mapped_column(Text, nullable=True)
    kind: Mapped[str] = mapped_column(String, nullable=False)
    linked_issue: Mapped[str | None] = mapped_column(String, nullable=True)
    factor_letter: Mapped[str | None] = mapped_column(String, nullable=True)
    contradiction: Mapped[str | None] = mapped_column(Text, nullable=True)
    existing_assertion_ids: Mapped[list[str]] = mapped_column(JSONDocument, default=list)
    envelope: Mapped[dict[str, Any] | None] = mapped_column(JSONDocument, nullable=True)


class LegalCoreExhibitAnnotation(Base):
    __tablename__ = "legal_core_exhibit_annotation"

    annotation_id: Mapped[uuid.UUID] = mapped_column(UUIDType, primary_key=True)
    matter_id: Mapped[uuid.UUID] = mapped_column(UUIDType, nullable=False, index=True)
    item_id: Mapped[uuid.UUID] = mapped_column(UUIDType, nullable=False)
    exhibit_label: Mapped[str] = mapped_column(String, default="")
    bates_number: Mapped[str] = mapped_column(String, default="")
    foundation: Mapped[str | None] = mapped_column(Text, nullable=True)
    custody_note: Mapped[str | None] = mapped_column(Text, nullable=True)
    redaction_note: Mapped[str | None] = mapped_column(Text, nullable=True)
    relevance: Mapped[str | None] = mapped_column(Text, nullable=True)
    issues: Mapped[list[str]] = mapped_column(JSONDocument, default=list)
    objection_notes: Mapped[str | None] = mapped_column(Text, nullable=True)
    readiness: Mapped[str] = mapped_column(String, default="candidate")


class LegalCoreAgentRun(Base):
    __tablename__ = "legal_core_agent_run"

    run_id: Mapped[uuid.UUID] = mapped_column(UUIDType, primary_key=True)
    matter_id: Mapped[uuid.UUID] = mapped_column(UUIDType, nullable=False, index=True)
    role: Mapped[str] = mapped_column(String, nullable=False)
    intent: Mapped[str] = mapped_column(String, nullable=False)
    prompt: Mapped[str] = mapped_column(Text, nullable=False)
    requested_model: Mapped[str] = mapped_column(String, nullable=False)
    effective_model: Mapped[str] = mapped_column(String, nullable=False)
    target_type: Mapped[str | None] = mapped_column(String, nullable=True)
    target_id: Mapped[str | None] = mapped_column(String, nullable=True)
    output: Mapped[str] = mapped_column(Text, nullable=False)
    status: Mapped[str] = mapped_column(String, nullable=False)


class LegalCoreFilingOverride(Base):
    __tablename__ = "legal_core_filing_override"

    override_id: Mapped[uuid.UUID] = mapped_column(UUIDType, primary_key=True)
    matter_id: Mapped[uuid.UUID] = mapped_column(UUIDType, nullable=False, index=True)
    check_id: Mapped[str] = mapped_column(String, nullable=False)
    state: Mapped[str] = mapped_column(String, nullable=False)
    note: Mapped[str | None] = mapped_column(Text, nullable=True)
    reviewer: Mapped[str] = mapped_column(String, nullable=False)


# ---------------------------------------------------------------------------
# legal_research
# ---------------------------------------------------------------------------


class LegalResearchAuthority(Base):
    __tablename__ = "legal_research_authority"

    authority_id: Mapped[uuid.UUID] = mapped_column(UUIDType, primary_key=True)
    matter_id: Mapped[uuid.UUID] = mapped_column(UUIDType, nullable=False, index=True)
    identifier: Mapped[str] = mapped_column(String, nullable=False)
    proposition: Mapped[str] = mapped_column(Text, nullable=False)
    authority_level: Mapped[str] = mapped_column(String, nullable=False)
    court: Mapped[str | None] = mapped_column(String, nullable=True)
    pinpoint: Mapped[str | None] = mapped_column(String, nullable=True)
    source_path: Mapped[str] = mapped_column(String, nullable=False)
    complete: Mapped[bool] = mapped_column(Boolean, nullable=False)
    applicable: Mapped[bool] = mapped_column(Boolean, nullable=False)
    skip_reason: Mapped[str | None] = mapped_column(Text, nullable=True)
    is_citator_verified: Mapped[bool] = mapped_column(Boolean, default=False)
    snapshot_hash: Mapped[str] = mapped_column(String, nullable=False)
    # Generic authority snapshot fields (used when an authority is imported from a search).
    jurisdiction: Mapped[str | None] = mapped_column(String, nullable=True)
    source_url: Mapped[str | None] = mapped_column(String, nullable=True)
    retrieved_at: Mapped[datetime | None] = mapped_column(DateTime, nullable=True)
    subsequent_history: Mapped[str | None] = mapped_column(Text, nullable=True)
    last_currentness_check: Mapped[datetime | None] = mapped_column(DateTime, nullable=True)


class LegalResearchCurrencyFlag(Base):
    __tablename__ = "legal_research_currency_flag"

    flag_id: Mapped[uuid.UUID] = mapped_column(UUIDType, primary_key=True)
    matter_id: Mapped[uuid.UUID] = mapped_column(UUIDType, nullable=False, index=True)
    identifier: Mapped[str] = mapped_column(String, nullable=False)
    note: Mapped[str] = mapped_column(Text, nullable=False)


class LegalResearchQuestion(Base):
    __tablename__ = "legal_research_question"

    question_id: Mapped[uuid.UUID] = mapped_column(UUIDType, primary_key=True)
    matter_id: Mapped[uuid.UUID] = mapped_column(UUIDType, nullable=False, index=True)
    question: Mapped[str] = mapped_column(Text, nullable=False)
    plan: Mapped[str | None] = mapped_column(Text, nullable=True)
    adverse_notes: Mapped[str | None] = mapped_column(Text, nullable=True)
    uncertainty: Mapped[str | None] = mapped_column(Text, nullable=True)
    authority_ids: Mapped[list[str]] = mapped_column(JSONDocument, default=list)
    jurisdiction: Mapped[str | None] = mapped_column(String, nullable=True)
    status: Mapped[str] = mapped_column(String, default="open")


# ---------------------------------------------------------------------------
# legal_work_product
# ---------------------------------------------------------------------------


class LegalWorkProductDraftSection(Base):
    __tablename__ = "legal_work_product_draft_section"

    section_id: Mapped[uuid.UUID] = mapped_column(UUIDType, primary_key=True)
    matter_id: Mapped[uuid.UUID] = mapped_column(UUIDType, nullable=False, index=True)
    heading: Mapped[str] = mapped_column(String, nullable=False)
    body: Mapped[str] = mapped_column(Text, nullable=False)
    factor_letter: Mapped[str | None] = mapped_column(String, nullable=True)
    citations: Mapped[list[dict[str, Any]]] = mapped_column(JSONDocument, default=list)


class LegalWorkProductWorkProduct(Base):
    __tablename__ = "legal_work_product_work_product"

    work_product_id: Mapped[uuid.UUID] = mapped_column(UUIDType, primary_key=True)
    matter_id: Mapped[uuid.UUID] = mapped_column(UUIDType, nullable=False, index=True)
    court_case_id: Mapped[uuid.UUID | None] = mapped_column(UUIDType, nullable=True)
    kind: Mapped[str] = mapped_column(String, nullable=False)
    title: Mapped[str] = mapped_column(String, nullable=False)


class LegalWorkProductWorkProductVersion(Base):
    __tablename__ = "legal_work_product_work_product_version"

    work_product_id: Mapped[uuid.UUID] = mapped_column(
        UUIDType, ForeignKey("legal_work_product_work_product.work_product_id"), primary_key=True
    )
    version: Mapped[int] = mapped_column(Integer, primary_key=True)
    state: Mapped[str] = mapped_column(String, nullable=False)
    source_package_id: Mapped[uuid.UUID | None] = mapped_column(UUIDType, nullable=True)
    content_hash: Mapped[str] = mapped_column(String, nullable=False)
    body_md: Mapped[str | None] = mapped_column(Text, nullable=True)
    cited_assertion_ids: Mapped[list[uuid.UUID]] = mapped_column(JSONDocument, default=list)
    cited_authority_ids: Mapped[list[str]] = mapped_column(JSONDocument, default=list)


class LegalWorkProductStrategyNote(Base):
    __tablename__ = "legal_work_product_strategy_note"

    note_id: Mapped[uuid.UUID] = mapped_column(UUIDType, primary_key=True)
    matter_id: Mapped[uuid.UUID] = mapped_column(UUIDType, nullable=False, index=True)
    kind: Mapped[str] = mapped_column(String, nullable=False)
    title: Mapped[str] = mapped_column(String, nullable=False)
    body: Mapped[str] = mapped_column(Text, nullable=False)
    source_chat: Mapped[str | None] = mapped_column(String, nullable=True)


class LegalWorkProductRedTeamRun(Base):
    __tablename__ = "legal_work_product_red_team_run"

    run_id: Mapped[uuid.UUID] = mapped_column(UUIDType, primary_key=True)
    matter_id: Mapped[uuid.UUID] = mapped_column(UUIDType, nullable=False, index=True)
    target_type: Mapped[str] = mapped_column(String, nullable=False)
    target_id: Mapped[str] = mapped_column(String, nullable=False)
    lens: Mapped[str] = mapped_column(String, nullable=False)
    prompt_or_notes: Mapped[str | None] = mapped_column(Text, nullable=True)
    findings: Mapped[list[dict[str, Any]]] = mapped_column(JSONDocument, nullable=False, default=list)
    model: Mapped[str | None] = mapped_column(String, nullable=True)


class LegalWorkProductReviewDecision(Base):
    __tablename__ = "legal_work_product_review_decision"

    review_id: Mapped[uuid.UUID] = mapped_column(UUIDType, primary_key=True)
    matter_id: Mapped[uuid.UUID] = mapped_column(UUIDType, nullable=False, index=True)
    section_id: Mapped[uuid.UUID] = mapped_column(UUIDType, nullable=False)
    work_product_id: Mapped[uuid.UUID | None] = mapped_column(UUIDType, nullable=True)
    verdict: Mapped[str] = mapped_column(String, nullable=False)
    rationale: Mapped[str | None] = mapped_column(Text, nullable=True)
    reviewer: Mapped[str] = mapped_column(String, nullable=False)
    content_hash: Mapped[str] = mapped_column(String, nullable=False)
    created_at: Mapped[datetime] = mapped_column(DateTime, server_default=func.now())
    disclosure: Mapped[str] = mapped_column(String, default="work_product_claimed")
    court_safe: Mapped[bool] = mapped_column(Boolean, default=False)
    exportable: Mapped[bool] = mapped_column(Boolean, default=False)


class LegalWorkProductReleaseManifest(Base):
    __tablename__ = "legal_work_product_release_manifest"

    release_id: Mapped[uuid.UUID] = mapped_column(UUIDType, primary_key=True)
    matter_id: Mapped[uuid.UUID] = mapped_column(UUIDType, nullable=False, index=True)
    work_product_id: Mapped[uuid.UUID] = mapped_column(UUIDType, nullable=False)
    section_ids: Mapped[list[uuid.UUID]] = mapped_column(JSONDocument, nullable=False)
    package_id: Mapped[uuid.UUID] = mapped_column(UUIDType, nullable=False)
    content_hash: Mapped[str] = mapped_column(String, nullable=False)
    cited_assertion_ids: Mapped[list[uuid.UUID]] = mapped_column(JSONDocument, default=list)
    cited_authority_ids: Mapped[list[str]] = mapped_column(JSONDocument, default=list)
    omitted_private: Mapped[list[str]] = mapped_column(JSONDocument, default=list)
    state: Mapped[str] = mapped_column(String, default="release_candidate")
    filed: Mapped[bool] = mapped_column(Boolean, default=False)


# ---------------------------------------------------------------------------
# legal_audit
# ---------------------------------------------------------------------------


class LegalAuditEventOutbox(Base):
    __tablename__ = "legal_audit_event_outbox"

    event_id: Mapped[uuid.UUID] = mapped_column(UUIDType, primary_key=True)
    event_type: Mapped[str] = mapped_column(String, nullable=False)
    schema_version: Mapped[str] = mapped_column(String, nullable=False)
    occurred_at: Mapped[datetime] = mapped_column(DateTime, nullable=False)
    matter_id: Mapped[uuid.UUID] = mapped_column(UUIDType, nullable=False, index=True)
    court_case_id: Mapped[uuid.UUID | None] = mapped_column(UUIDType, nullable=True)
    aggregate_id: Mapped[uuid.UUID] = mapped_column(UUIDType, nullable=False)
    aggregate_version: Mapped[int] = mapped_column(Integer, nullable=False)
    trace_id: Mapped[str] = mapped_column(String, nullable=False)
    payload_hash: Mapped[str] = mapped_column(String, nullable=False)
    payload: Mapped[dict[str, Any]] = mapped_column(JSONDocument, nullable=False)
    published_at: Mapped[datetime | None] = mapped_column(DateTime, nullable=True)


class LegalAuditConsumedEvent(Base):
    __tablename__ = "legal_audit_consumed_event"

    event_id: Mapped[uuid.UUID] = mapped_column(UUIDType, primary_key=True)
    consumed_at: Mapped[datetime] = mapped_column(DateTime, server_default=func.now())

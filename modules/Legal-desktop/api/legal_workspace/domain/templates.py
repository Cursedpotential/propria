"""Drafting templates for THIS Genesee postjudgment matter.

> _Byline: Grok · grok-4.6 · 2026-08-18_
Structural outlines only. Not official SCAO forms. Not filing-ready.
Caption, docket, judge, and dates stay blank until clerk-confirmed.
"""

from __future__ import annotations

from enum import Enum

from pydantic import BaseModel, Field


class TemplateKind(str, Enum):
    MOTION = "motion"
    AFFIDAVIT = "affidavit"
    BRIEF = "brief"
    PROPOSED_ORDER = "proposed_order"
    NOTICE = "notice"
    PROOF_OF_SERVICE = "proof_of_service"
    OBJECTION = "objection"


class DraftingTemplate(BaseModel):
    template_id: str
    title: str
    kind: TemplateKind
    proceeding: str = "postjudgment_domestic_relations"
    governing_authority: str
    outline: list[str]
    official_form_url: str | None = None
    notes: str
    court_safe: bool = False


class TemplateInstantiate(BaseModel):
    template_id: str
    heading: str | None = None


def catalog() -> list[DraftingTemplate]:
    """Only templates applicable to this Matter's posture."""
    scao = "https://www.courts.michigan.gov/SCAO-forms/"
    return [
        DraftingTemplate(
            template_id="motion-parenting-time-specific",
            title="Motion for parenting time in specific terms",
            kind=TemplateKind.MOTION,
            governing_authority="MCL 722.27a(8); MCR 2.119; MCR 3.213",
            official_form_url=scao,
            notes="Packet P1 structure. Interaction with Vodvarka/MCL 722.27(1)(c) is unresolved — do not plead a holding that is not in the packet.",
            outline=[
                "Caption / case number: [clerk-confirm — do not invent]",
                "Relief in one sentence: parenting time in specific terms (dates, times, places).",
                "Authority: MCL 722.27a(8) (shall / in specific terms / if requested / by either party / at any time).",
                "Facts: only accepted LegalSourcePackage spans. No diagnostic labels.",
                "MCL 722.23 factors only where a fact is actually cited.",
                "Verification / concurrence / proposed order / notice: [verify current MCR 2.119 and local practice].",
            ],
        ),
        DraftingTemplate(
            template_id="motion-custody-modification",
            title="Motion to modify custody / parenting time (threshold + best interests)",
            kind=TemplateKind.MOTION,
            governing_authority="MCL 722.27(1)(c); MCL 722.23; Vodvarka; Shade; Pierron",
            official_form_url=scao,
            notes="Requires last-order date before Vodvarka lookback. Leave the date blank until imported.",
            outline=[
                "Caption / case number: [clerk-confirm]",
                "Proper cause or change of circumstances since the last custody order (date: [unknown]).",
                "Established custodial environment and resulting burden.",
                "What the judge must consider (a)–(l), each parent, conduct-first.",
                "Requested relief: narrow, specific, enforceable.",
                "Verification / notice / proposed order.",
            ],
        ),
        DraftingTemplate(
            template_id="affidavit",
            title="Affidavit / verification",
            kind=TemplateKind.AFFIDAVIT,
            governing_authority="MCR 2.119(B); personal knowledge",
            official_form_url=scao,
            notes="Personal knowledge only. Not a vehicle for theories or strategy notes.",
            outline=[
                "Identity of affiant.",
                "Basis of personal knowledge.",
                "Numbered factual paragraphs with exhibit/span pins.",
                "Notary / verification block: [current form].",
            ],
        ),
        DraftingTemplate(
            template_id="brief-in-support",
            title="Brief in support",
            kind=TemplateKind.BRIEF,
            governing_authority="MCR 2.119; cited packet authorities only",
            notes="Every proposition needs a pinned snapshot. No unpublished opinions without owner weight review.",
            outline=[
                "Issue presented.",
                "Governing authority and standard.",
                "Argument mapped to accepted facts.",
                "Adverse authority / uncertainty acknowledged.",
                "Conclusion and requested relief.",
            ],
        ),
        DraftingTemplate(
            template_id="proposed-order",
            title="Proposed order",
            kind=TemplateKind.PROPOSED_ORDER,
            governing_authority="MCR 2.602; local entry practice",
            official_form_url=scao,
            notes="Specific, enforceable terms. No argument. No strategy language.",
            outline=[
                "Caption.",
                "Findings reserved or stated only if the court has made them.",
                "Ordered terms: dates, times, places, exchange logistics.",
                "Signature / entry block: [clerk-confirm].",
            ],
        ),
        DraftingTemplate(
            template_id="notice-of-hearing",
            title="Notice of hearing",
            kind=TemplateKind.NOTICE,
            governing_authority="MCR 2.119; Genesee clerk practice",
            official_form_url=scao,
            notes="Hearing date is blank until clerk-set. Do not invent a date.",
            outline=[
                "Caption.",
                "What will be heard: [motion title].",
                "When / where: [clerk-set — blank].",
                "Service certificate.",
            ],
        ),
        DraftingTemplate(
            template_id="proof-of-service",
            title="Proof of service",
            kind=TemplateKind.PROOF_OF_SERVICE,
            governing_authority="MCR 2.107; MCR 3.203",
            official_form_url=scao,
            notes="Record what was actually served. Do not backfill.",
            outline=[
                "Documents served.",
                "On whom.",
                "Method and date: [actual only].",
                "Server identity.",
            ],
        ),
        DraftingTemplate(
            template_id="objection-to-referee",
            title="Objection to referee recommendation (de novo)",
            kind=TemplateKind.OBJECTION,
            governing_authority="MCR 3.215; Genesee FOC / referee practice",
            official_form_url=scao,
            notes="Deadline is computed from the actual recommendation date. Do not invent it.",
            outline=[
                "Caption.",
                "Recommendation identified by date: [unknown until received].",
                "Specific objections — not a general disagreement.",
                "Request for de novo hearing.",
                "Service.",
            ],
        ),
    ]


def get_template(template_id: str) -> DraftingTemplate:
    return next(row for row in catalog() if row.template_id == template_id)


def render_body(template: DraftingTemplate) -> str:
    lines = [
        template.title,
        "",
        f"Authority: {template.governing_authority}",
        f"Notes: {template.notes}",
        "",
        "This is a working outline. Not an official form. Not filing-ready.",
        "",
    ]
    for index, item in enumerate(template.outline, start=1):
        lines.append(f"{index}. {item}")
    return "\n".join(lines)

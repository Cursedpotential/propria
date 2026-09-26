"""Issue / element map for the first slice.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from __future__ import annotations

from uuid import UUID, uuid4

from pydantic import BaseModel, Field

from legal_workspace.contracts.citations import EvidenceCitation
from legal_workspace.domain.factors import FACTOR_TITLES


class IssueElement(BaseModel):
    element_id: UUID = Field(default_factory=uuid4)
    label: str
    supporting: list[EvidenceCitation] = Field(default_factory=list)
    contradicting: list[EvidenceCitation] = Field(default_factory=list)
    missing_proof: list[str] = Field(default_factory=list)


class LegalIssue(BaseModel):
    issue_id: UUID = Field(default_factory=uuid4)
    title: str
    governing_authority: str
    elements: list[IssueElement] = Field(default_factory=list)
    children: list[LegalIssue] = Field(default_factory=list)


class IssueChildCreate(BaseModel):
    title: str
    governing_authority: str = ""
    parent_id: UUID | None = None


class IssuePatch(BaseModel):
    title: str | None = None
    governing_authority: str | None = None


class IssueElementCreate(BaseModel):
    label: str
    missing_proof: list[str] = Field(default_factory=list)


def default_custody_modification_issue() -> LegalIssue:
    """Structural issue tree only — no invented case facts or citations."""
    return LegalIssue(
        title="Modification of custody / parenting time (threshold + best interests)",
        governing_authority="MCL 722.27(1)(c); MCL 722.23; Vodvarka; Shade; Pierron",
        elements=[
            IssueElement(label="Proper cause or change of circumstances (if a final order exists)"),
            IssueElement(label="Established custodial environment and resulting burden"),
            IssueElement(label="What the judge must consider (a)–(l) considered individually"),
        ],
        children=[
            LegalIssue(
                title="Proper cause / change of circumstances",
                governing_authority="Vodvarka (if a final order exists)",
                elements=[
                    IssueElement(
                        label="Proper cause or change of circumstances (if a final order exists)"
                    ),
                ],
            ),
            LegalIssue(
                title="Established custodial environment / burden",
                governing_authority="Pierron / Shade",
                elements=[
                    IssueElement(label="Established custodial environment and resulting burden"),
                ],
            ),
            LegalIssue(
                title="Best interests MCL 722.23 (a)–(l)",
                governing_authority="MCL 722.23",
                elements=[
                    IssueElement(label=f"({letter.value}) {title}")
                    for letter, title in FACTOR_TITLES.items()
                ],
            ),
        ],
    )


def issue_tree(issue: LegalIssue | None = None) -> LegalIssue:
    """Return a legal issue tree.

    Parent should mount ``GET /v1/issues`` from ``WORKSPACE.load().issue``.
    Passing ``None`` returns the structural custody-modification seed.
    Seed citations stay empty — do not invent package facts here.
    """
    return issue if issue is not None else default_custody_modification_issue()

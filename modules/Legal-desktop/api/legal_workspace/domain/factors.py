"""MCL 722.23 (a)–(l) workspace types.

> _Byline: Grok · grok-4.6 · 2026-08-18_
> Labels taken from the custody-packet Module 2 draft (verbatim statute
> headings). Diagnostic labels are not allowed as court-facing facts.
"""

from __future__ import annotations

from enum import Enum
from uuid import UUID

from pydantic import BaseModel, Field

from legal_workspace.contracts.citations import EvidenceCitation


class FactorLetter(str, Enum):
    A = "a"
    B = "b"
    C = "c"
    D = "d"
    E = "e"
    F = "f"
    G = "g"
    H = "h"
    I = "i"
    J = "j"
    K = "k"
    L = "l"


# Statutory short titles from MCL 722.23 as quoted in packet M2.
FACTOR_TITLES: dict[FactorLetter, str] = {
    FactorLetter.A: "Love, affection, and other emotional ties",
    FactorLetter.B: "Capacity to give love, affection, guidance, and to continue education/religion",
    FactorLetter.C: "Capacity to provide food, clothing, medical care, and material needs",
    FactorLetter.D: "Length of time in a stable, satisfactory environment; continuity",
    FactorLetter.E: "Permanence, as a family unit, of the existing or proposed home",
    FactorLetter.F: "Moral fitness of the parties (as it affects parenting)",
    FactorLetter.G: "Mental and physical health of the parties",
    FactorLetter.H: "Home, school, and community record of the child",
    FactorLetter.I: "Reasonable preference of the child, if of sufficient age",
    FactorLetter.J: "Willingness to facilitate a relationship with the other parent",
    FactorLetter.K: "Domestic violence, whether or not directed at or witnessed by the child",
    FactorLetter.L: "Any other factor the court considers relevant",
}

AUTHORITY_PIN = "MCL 722.23"


class FactorSideNotes(BaseModel):
    for_parent: list[str] = Field(default_factory=list)
    citations: list[EvidenceCitation] = Field(default_factory=list)


class FactorEntry(BaseModel):
    letter: FactorLetter
    title: str
    authority: str = AUTHORITY_PIN
    petitioner: FactorSideNotes = Field(default_factory=FactorSideNotes)
    respondent: FactorSideNotes = Field(default_factory=FactorSideNotes)
    contradictions: list[str] = Field(default_factory=list)
    missing_proof: list[str] = Field(default_factory=list)
    proposed_finding: str | None = None
    owner_weight_note: str | None = None
    both_parent_analysis: str | None = None


class FactorNoteCreate(BaseModel):
    side: str
    text: str


class FactorAnalysis(BaseModel):
    letter: FactorLetter
    petitioner: FactorSideNotes
    respondent: FactorSideNotes
    contradictions: list[str]
    missing_proof: list[str]
    both_parent_analysis: str


def structural_both_parent_prompt(entry: FactorEntry) -> str:
    """Court-safe-false scratch. Requires both sides. Not a finding."""
    petitioner_notes = len(entry.petitioner.for_parent)
    respondent_notes = len(entry.respondent.for_parent)
    petitioner_cites = len(entry.petitioner.citations)
    respondent_cites = len(entry.respondent.citations)
    contradictions = (
        "; ".join(entry.contradictions) if entry.contradictions else "none recorded"
    )
    missing = "; ".join(entry.missing_proof) if entry.missing_proof else "none recorded"
    return (
        "STRUCTURAL BOTH-PARENT PROMPT — not a finding. "
        "court_safe=false. Scratch only. Do not diagnose. Do not invent facts. "
        f"Factor ({entry.letter.value}) {entry.title} under {entry.authority}. "
        "Address BOTH sides — petitioner and respondent — including "
        f"petitioner notes ({petitioner_notes}) and citations ({petitioner_cites}), "
        f"respondent notes ({respondent_notes}) and citations ({respondent_cites}). "
        f"Name contradictions ({contradictions}) and missing proof ({missing}) explicitly. "
        "Weighting stays an owner decision."
    )


def analyze_factor(entry: FactorEntry) -> FactorAnalysis:
    """Live factor view. Prompt text is not a screen dump — owner notes only."""
    return FactorAnalysis(
        letter=entry.letter,
        petitioner=entry.petitioner,
        respondent=entry.respondent,
        contradictions=list(entry.contradictions),
        missing_proof=list(entry.missing_proof),
        both_parent_analysis=entry.both_parent_analysis or "",
    )


def lookup_factor(factors: list[FactorEntry], letter: FactorLetter) -> FactorEntry | None:
    for entry in factors:
        if entry.letter == letter:
            return entry
    return None


def empty_factor_matrix() -> list[FactorEntry]:
    return [FactorEntry(letter=letter, title=title) for letter, title in FACTOR_TITLES.items()]


class FactorCitationLink(BaseModel):
    letter: FactorLetter
    side: str
    citation: EvidenceCitation
    package_id: UUID

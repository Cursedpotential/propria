"""Keyword first-pass privilege / sensitivity markers.

> _Byline: Grok · grok-4.6 · 2026-08-18_
Hits are hypothesized markers only. Never a privilege determination,
work-product claim, or other legal conclusion. No LLM. Does not
route Confidential Mode.
"""

from __future__ import annotations

import re
from enum import Enum
from uuid import UUID

from pydantic import BaseModel, Field, model_validator

DISCLAIMER = (
    "Keyword first-pass only. Hits are hypothesized markers, not a privilege "
    "determination, work-product claim, confidentiality holding, or other "
    "legal conclusion. A court decides privilege. Do not treat this scan as "
    "advice, and do not treat AI chat as attorney-client privileged."
)


class MarkerKind(str, Enum):
    ATTORNEY_CLIENT = "attorney_client"
    WORK_PRODUCT = "work_product"
    STRATEGY = "strategy"
    MEDICAL = "medical"
    CHILD_IDENTIFYING = "child_identifying"


KIND_LABELS: dict[MarkerKind, str] = {
    MarkerKind.ATTORNEY_CLIENT: "attorney-client",
    MarkerKind.WORK_PRODUCT: "work product",
    MarkerKind.STRATEGY: "strategy",
    MarkerKind.MEDICAL: "medical",
    MarkerKind.CHILD_IDENTIFYING: "child-identifying",
}


def _rx(*alts: str) -> re.Pattern[str]:
    return re.compile("|".join(f"(?:{alt})" for alt in alts), re.IGNORECASE)


# Compound phrases only. Lone "attorney" / "privilege" / "child" fire too often.
MARKER_PATTERNS: tuple[tuple[MarkerKind, re.Pattern[str]], ...] = (
    (
        MarkerKind.ATTORNEY_CLIENT,
        _rx(
            r"attorney[-\s/]client",
            r"privileged and confidential",
            r"privileged communication",
            r"advice of counsel",
            r"communication with (?:my )?(?:attorney|counsel|lawyer)",
            r"legal advice from (?:my )?(?:attorney|counsel|lawyer)",
            r"in confidence with (?:my )?(?:attorney|counsel|lawyer)",
        ),
    ),
    (
        MarkerKind.WORK_PRODUCT,
        _rx(
            r"work[-\s]product",
            r"prepared in anticipation of litigation",
            r"trial[-\s]preparation materials",
            r"mental impressions of counsel",
        ),
    ),
    (
        MarkerKind.STRATEGY,
        _rx(
            r"litigation strategy",
            r"trial strategy",
            r"settlement (?:posture|range|position|authority)",
            r"theory of the case",
            r"bargaining position",
            r"negotiating position",
            r"walk-away(?: number)?",
            r"(?:our|case|do not disclose(?: this)?)\s+strategy",
        ),
    ),
    (
        MarkerKind.MEDICAL,
        _rx(
            r"physician[-\s]patient",
            r"therapist[-\s]patient",
            r"psychologist[-\s]patient",
            r"therapy records",
            r"therapist(?:'s)? (?:notes|records|file)",
            r"mental[-\s]health records",
            r"medical records",
            r"counseling records",
            r"treatment records",
            r"\bHIPAA\b",
            r"42\s*C\.?F\.?R\.?\s*Part\s*2",
            r"psychiatrist",
            r"prescription(?:s)?",
            r"\bDSM-?[IVX0-9]+\b",
            r"\bdiagnos(?:is|ed|es)\b",
        ),
    ),
    (
        MarkerKind.CHILD_IDENTIFYING,
        _rx(
            r"child(?:'s)? (?:full )?name",
            r"minor(?:'s)? (?:full )?name",
            r"child(?:'s)? date of birth",
            r"child(?:'s)?\s*DOB",
            r"child(?:'s)? social security",
            r"child(?:'s)?\s*SSN",
            r"social security number",
            r"\bSSN\b",
            r"\b\d{3}-\d{2}-\d{4}\b",
            r"date of birth",
            r"child(?:'s)? (?:home )?address",
            r"child(?:'s)? school",
            r"school records",
            r"child(?:'s)? therapist",
            r"identifying information of the (?:child|minor)",
        ),
    ),
)


class HypothesizedMarker(BaseModel):
    kind: MarkerKind
    label: str
    matched: str
    excerpt: str
    start: int
    end: int


class PrivilegeScan(BaseModel):
    method: str = "keyword_first_pass"
    source: str = "text"
    section_id: UUID | None = None
    hypothesized_markers: list[HypothesizedMarker] = Field(default_factory=list)
    kinds_hit: list[MarkerKind] = Field(default_factory=list)
    court_safe: bool = False
    legal_conclusion: bool = False
    exportable: bool = False
    disclosure: str = "hypothesized_marker"
    epistemic_class: str = "hypothesized_marker"
    disclaimer: str = DISCLAIMER

    @model_validator(mode="after")
    def never_a_legal_conclusion(self) -> PrivilegeScan:
        self.court_safe = False
        self.legal_conclusion = False
        self.exportable = False
        return self


class PrivilegeScanRequest(BaseModel):
    text: str | None = None
    section_id: UUID | None = None


def _excerpt(text: str, start: int, end: int, radius: int = 40) -> str:
    lo = max(0, start - radius)
    hi = min(len(text), end + radius)
    prefix = "…" if lo else ""
    suffix = "…" if hi < len(text) else ""
    snippet = text[lo:hi].replace("\n", " ").replace("\r", " ")
    return f"{prefix}{snippet}{suffix}"


def scan_text(text: str) -> PrivilegeScan:
    """Return hypothesized keyword markers. Never a legal conclusion."""
    markers: list[HypothesizedMarker] = []
    seen: set[tuple[str, int, int]] = set()
    kinds: list[MarkerKind] = []
    for kind, pattern in MARKER_PATTERNS:
        for match in pattern.finditer(text or ""):
            key = (kind.value, match.start(), match.end())
            if key in seen:
                continue
            seen.add(key)
            markers.append(
                HypothesizedMarker(
                    kind=kind,
                    label=KIND_LABELS[kind],
                    matched=match.group(0),
                    excerpt=_excerpt(text, match.start(), match.end()),
                    start=match.start(),
                    end=match.end(),
                )
            )
            if kind not in kinds:
                kinds.append(kind)
    return PrivilegeScan(
        hypothesized_markers=markers,
        kinds_hit=kinds,
        court_safe=False,
        legal_conclusion=False,
    )

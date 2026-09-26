"""Paragraph-level support map. Not sentence-level OCR.

> _Byline: Grok · grok-4.6 · 2026-08-18_
A paragraph is supported only if the section carries approved
citations. Outline/instruction lines are not treated as facts.
"""

from __future__ import annotations

from enum import Enum

from pydantic import BaseModel


class SupportState(str, Enum):
    SUPPORTED = "supported"
    UNSUPPORTED = "unsupported"
    NOT_FACTUAL = "not_factual"


class SupportParagraph(BaseModel):
    index: int
    text: str
    state: SupportState


class SupportMap(BaseModel):
    section_id: str
    heading: str
    citation_count: int
    paragraphs: list[SupportParagraph]
    unsupported_count: int


_INSTRUCTION = (
    "not filing-ready",
    "not an official",
    "working outline",
    "[clerk",
    "[verify",
    "[unknown",
    "[actual",
    "do not invent",
    "authority:",
    "notes:",
)


def is_instruction(text: str) -> bool:
    lowered = text.lower()
    if lowered[:1].isdigit() and ". " in lowered[:4]:
        return True
    return any(marker in lowered for marker in _INSTRUCTION)


def build_support_map(
    *,
    section_id: str,
    heading: str,
    body: str,
    citation_count: int,
    citations_ok: bool,
) -> SupportMap:
    chunks = [part.strip() for part in body.replace("\r\n", "\n").split("\n") if part.strip()]
    paragraphs: list[SupportParagraph] = []
    for index, text in enumerate(chunks):
        if is_instruction(text):
            state = SupportState.NOT_FACTUAL
        elif citation_count and citations_ok:
            state = SupportState.SUPPORTED
        else:
            state = SupportState.UNSUPPORTED
        paragraphs.append(SupportParagraph(index=index, text=text, state=state))
    return SupportMap(
        section_id=section_id,
        heading=heading,
        citation_count=citation_count,
        paragraphs=paragraphs,
        unsupported_count=sum(1 for item in paragraphs if item.state is SupportState.UNSUPPORTED),
    )

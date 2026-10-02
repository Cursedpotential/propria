"""Paragraph-level evidence needs, independent of section citation validity.

A section citation check establishes that cited records resolve; it does not
establish that those records support each paragraph. Until explicit passage
support is provided, factual paragraphs remain unsupported. The DTO retains
the supported state for compatibility with a future passage support check.
"""

from __future__ import annotations

import re
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
    # Numbered facts are common in affidavits. Strip the list prefix only to
    # recognize an explicit instruction, never to classify numbering as one.
    lowered = re.sub(r"^\d+[.)]\s+", "", text.strip().lower())
    return any(lowered.startswith(marker) for marker in _INSTRUCTION)


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
        else:
            # citations_ok belongs to the independent source-resolution check.
            # This function receives no paragraph-to-evidence relationship, so
            # neither a valid citation nor its count can establish support.
            state = SupportState.UNSUPPORTED
        paragraphs.append(SupportParagraph(index=index, text=text, state=state))
    return SupportMap(
        section_id=section_id,
        heading=heading,
        citation_count=citation_count,
        paragraphs=paragraphs,
        unsupported_count=sum(1 for item in paragraphs if item.state is SupportState.UNSUPPORTED),
    )

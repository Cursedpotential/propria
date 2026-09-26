"""Release citation gate. Structure is not subsequent-history.

> _Byline: Grok · grok-4.6 · 2026-08-18_

legal-mcp's CitationParser is regex structure-only and says so in its
docstring. CourtListener is not a citator (custody-guide GUARDRAILS).
This gate blocks release when a factual sentence has no approved
assertion, or a legal proposition has no pinned snapshot.
"""

from __future__ import annotations

from dataclasses import dataclass

from legal_workspace.contracts.citations import AuthorityCitation, EvidenceCitation
from legal_workspace.contracts.source_package import LegalSourcePackage, ReviewState


@dataclass(frozen=True)
class CitationGateResult:
    ok: bool
    blockers: tuple[str, ...]


def validate_factual_citations(
    statements: list[EvidenceCitation],
    package: LegalSourcePackage,
) -> CitationGateResult:
    approved = {
        (str(item.assertion_id), item.assertion_version)
        for item in package.items
        if item.review_state is ReviewState.APPROVED
    }
    blockers: list[str] = []
    for citation in statements:
        key = (str(citation.assertion_id), citation.assertion_version)
        if str(citation.package_id) != str(package.package_id):
            blockers.append(f"statement not in imported package: {citation.statement[:80]}")
        elif key not in approved:
            blockers.append(f"UNSUPPORTED: {citation.statement[:80]}")
        elif not citation.span_locator:
            blockers.append(f"missing span locator: {citation.statement[:80]}")
    return CitationGateResult(ok=not blockers, blockers=tuple(blockers))


def validate_authority_citations(citations: list[AuthorityCitation]) -> CitationGateResult:
    blockers: list[str] = []
    for citation in citations:
        if not citation.identifier.strip():
            blockers.append("empty authority identifier")
            continue
        if citation.snapshot_hash is None:
            blockers.append(f"no pinned snapshot: {citation.identifier}")
        if citation.authority_level.value == "unpublished_opinion":
            blockers.append(
                f"unpublished opinion needs human weight review: {citation.identifier}"
            )
        if citation.is_citator_verified:
            blockers.append(
                f"citator-verified flag is not allowed without a reviewed subsequent-history check: {citation.identifier}"
            )
    return CitationGateResult(ok=not blockers, blockers=tuple(blockers))

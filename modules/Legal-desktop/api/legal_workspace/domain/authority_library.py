"""Curated Michigan authorities for THIS matter.

> _Byline: Grok · grok-4.6 · 2026-08-18_
Only rows that are both complete and applicable are seeded.
Source: custody-packet Module 2 + verification ledger. Not citator-verified.
"""

from __future__ import annotations

from pydantic import BaseModel, Field

from legal_workspace.contracts.citations import AuthorityLevel


class CuratedAuthority(BaseModel):
    identifier: str
    proposition: str
    authority_level: AuthorityLevel
    court: str | None = None
    pinpoint: str | None = None
    source_path: str
    complete: bool
    applicable: bool
    skip_reason: str | None = None
    is_citator_verified: bool = False
    snapshot_hash: str


def packet_authorities() -> list[CuratedAuthority]:
    """All candidates. Callers must filter to complete and applicable."""
    return [
        CuratedAuthority(
            identifier="MCL 722.23",
            proposition="Best interests of the child means the sum total of factors (a) through (l).",
            authority_level=AuthorityLevel.STATUTE,
            pinpoint="(a)–(l)",
            source_path="custody-guide/draft/M2-standards-that-decide-your-case.md",
            complete=True,
            applicable=True,
            snapshot_hash="packet:M2:mcl-722.23",
        ),
        CuratedAuthority(
            identifier="MCL 722.27(1)(c)",
            proposition="Modification that changes an established custodial environment requires clear and convincing evidence it is in the child's best interests; ECE is defined in the same subsection.",
            authority_level=AuthorityLevel.STATUTE,
            pinpoint="(1)(c)",
            source_path="custody-guide/draft/M2-standards-that-decide-your-case.md",
            complete=True,
            applicable=True,
            snapshot_hash="packet:M2:mcl-722.27-1-c",
        ),
        CuratedAuthority(
            identifier="MCL 722.27a(3)",
            proposition="A child has a right to parenting time with a parent unless clear and convincing evidence shows it would endanger the child's physical, mental, or emotional health.",
            authority_level=AuthorityLevel.STATUTE,
            pinpoint="(3)",
            source_path="custody-guide/draft/M2-standards-that-decide-your-case.md",
            complete=True,
            applicable=True,
            snapshot_hash="packet:M2:mcl-722.27a-3",
        ),
        CuratedAuthority(
            identifier="Pierron v Pierron, 486 Mich 81 (2010)",
            proposition="Where a proposed change would alter the ECE, clear and convincing evidence is required; otherwise preponderance governs and the burden sits on the parent proposing the change.",
            authority_level=AuthorityLevel.PUBLISHED_OPINION,
            court="Michigan Supreme Court",
            source_path="custody-guide/draft/M2-standards-that-decide-your-case.md",
            complete=True,
            applicable=True,
            snapshot_hash="packet:M2:pierron",
        ),
        CuratedAuthority(
            identifier="Vodvarka v Grasmeyer, 259 Mich App 499 (2003)",
            proposition="To modify custody, proper cause or change of circumstances looks at events since the last custody order; ordinary life changes are generally not enough.",
            authority_level=AuthorityLevel.PUBLISHED_OPINION,
            court="Michigan Court of Appeals",
            source_path="custody-guide/draft/M2-standards-that-decide-your-case.md",
            complete=True,
            applicable=True,
            snapshot_hash="packet:M2:vodvarka",
        ),
        CuratedAuthority(
            identifier="Shade v Wright, 291 Mich App 17 (2010)",
            proposition="A parenting-time-only change that would not alter the ECE is not held to the full Vodvarka custody threshold.",
            authority_level=AuthorityLevel.PUBLISHED_OPINION,
            court="Michigan Court of Appeals",
            source_path="custody-guide/draft/M2-standards-that-decide-your-case.md",
            complete=True,
            applicable=True,
            snapshot_hash="packet:M2:shade",
        ),
        CuratedAuthority(
            identifier="Fletcher v Fletcher, 447 Mich 871 (1994)",
            proposition="The court must consider and make findings on every best-interest factor but need not give them equal weight.",
            authority_level=AuthorityLevel.PUBLISHED_OPINION,
            court="Michigan Supreme Court",
            source_path="custody-guide/draft/M2-standards-that-decide-your-case.md",
            complete=True,
            applicable=True,
            snapshot_hash="packet:M2:fletcher",
        ),
        CuratedAuthority(
            identifier="Baker v Baker, 411 Mich 567 (1981)",
            proposition="Custody is decided on the sum total of the factors, not by counting which side won more of them.",
            authority_level=AuthorityLevel.PUBLISHED_OPINION,
            court="Michigan Supreme Court",
            source_path="custody-guide/draft/M2-standards-that-decide-your-case.md",
            complete=True,
            applicable=True,
            snapshot_hash="packet:M2:baker",
        ),
        CuratedAuthority(
            identifier="Hayes v Hayes, 209 Mich App 385 (1995)",
            proposition="An established custodial environment is a question of fact and may exist regardless of how it was created.",
            authority_level=AuthorityLevel.PUBLISHED_OPINION,
            court="Michigan Court of Appeals",
            source_path="custody-guide/draft/M2-standards-that-decide-your-case.md",
            complete=True,
            applicable=True,
            snapshot_hash="packet:M2:hayes",
        ),
        CuratedAuthority(
            identifier="Bofysil v Bofysil, 332 Mich App 232 (2020)",
            proposition="Working outside the home does not, by itself, cost a parent the established custodial environment.",
            authority_level=AuthorityLevel.PUBLISHED_OPINION,
            court="Michigan Court of Appeals",
            source_path="custody-guide/draft/M2-standards-that-decide-your-case.md",
            complete=True,
            applicable=True,
            snapshot_hash="packet:M2:bofysil",
        ),
        CuratedAuthority(
            identifier="MCR 3.210(D)(1)",
            proposition="Findings of fact and conclusions of law are required on contested postjudgment motions to modify a final judgment or order.",
            authority_level=AuthorityLevel.COURT_RULE,
            pinpoint="(D)(1)",
            source_path="custody-guide/verification_ledger.md",
            complete=True,
            applicable=True,
            snapshot_hash="packet:ledger:mcr-3.210-d1",
        ),
        CuratedAuthority(
            identifier="MCR 7.215(C)(1)",
            proposition="Unpublished opinions are not precedentially binding and should not be cited for propositions that have published authority.",
            authority_level=AuthorityLevel.COURT_RULE,
            pinpoint="(C)(1)",
            source_path="custody-guide/verification_ledger.md",
            complete=True,
            applicable=True,
            snapshot_hash="packet:ledger:mcr-7.215-c1",
        ),
        # Incomplete — kept in the catalog so we can prove they were excluded.
        CuratedAuthority(
            identifier="Ludema v Ludema (Mich Ct App 2003)",
            proposition="Quotes MCL 722.27(1)(c) ECE definition.",
            authority_level=AuthorityLevel.PUBLISHED_OPINION,
            court="Michigan Court of Appeals",
            source_path="custody-guide/draft/M2-standards-that-decide-your-case.md",
            complete=False,
            applicable=True,
            skip_reason="No volume/reporter/page in the packet pin.",
            snapshot_hash="packet:M2:ludema-incomplete",
        ),
        CuratedAuthority(
            identifier="Grew v Knox",
            proposition="Unpublished; docket not confirmed in the packet.",
            authority_level=AuthorityLevel.UNPUBLISHED_OPINION,
            source_path="custody-guide/verification_ledger.md",
            complete=False,
            applicable=False,
            skip_reason="Unpublished and docket/date incomplete; MCR 7.215(C)(1).",
            snapshot_hash="packet:ledger:grew-incomplete",
        ),
        CuratedAuthority(
            identifier="MCL 722.27a(7) lettered list",
            proposition="Parenting-time factors; subsection lettering was renumbered.",
            authority_level=AuthorityLevel.STATUTE,
            pinpoint="(7)",
            source_path="custody-guide/draft/M2-standards-that-decide-your-case.md",
            complete=False,
            applicable=True,
            skip_reason="Packet requires verbatim lettering confirmation before quoting.",
            snapshot_hash="packet:M2:722.27a-7-unconfirmed",
        ),
    ]


def seedable_authorities() -> list[CuratedAuthority]:
    return [row for row in packet_authorities() if row.complete and row.applicable]

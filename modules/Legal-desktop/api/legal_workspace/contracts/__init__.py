"""Public contracts for Legal Workspace.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from legal_workspace.contracts.citations import AuthorityCitation, EvidenceCitation
from legal_workspace.contracts.events import EventEnvelope
from legal_workspace.contracts.identity import CourtCaseRef, MatterRef
from legal_workspace.contracts.source_package import (
    LegalSourcePackage,
    LegalSourcePackageItem,
)

__all__ = [
    "AuthorityCitation",
    "CourtCaseRef",
    "EventEnvelope",
    "EvidenceCitation",
    "LegalSourcePackage",
    "LegalSourcePackageItem",
    "MatterRef",
]

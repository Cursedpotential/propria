"""Only complete, applicable packet authorities are seeded.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from legal_workspace.domain.authority_library import packet_authorities, seedable_authorities
from legal_workspace.services.workspace import Workspace


def test_seed_skips_incomplete_and_inapplicable() -> None:
    seeded = {row.identifier for row in seedable_authorities()}
    assert "Pierron v Pierron, 486 Mich 81 (2010)" in seeded
    assert "MCL 722.23" in seeded
    assert "Grew v Knox" not in seeded
    assert "Ludema v Ludema (Mich Ct App 2003)" not in seeded
    assert "MCL 722.27a(7) lettered list" not in seeded
    skipped = [row for row in packet_authorities() if not (row.complete and row.applicable)]
    assert skipped
    assert all(row.skip_reason for row in skipped)


def test_workspace_loads_seeded_authorities(tmp_path) -> None:
    state = Workspace(tmp_path).load()
    assert len(state.authorities) == len(seedable_authorities())
    assert all(row.complete and row.applicable for row in state.authorities)
    assert all(row.is_citator_verified is False for row in state.authorities)

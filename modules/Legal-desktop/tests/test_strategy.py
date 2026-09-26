"""My private notes store persists and is never court-safe.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from legal_workspace.domain.strategy import StrategyCreate, StrategyKind
from legal_workspace.services.workspace import Workspace


def test_strategy_note_persists_and_cannot_export(tmp_path) -> None:
    first = Workspace(tmp_path)
    note = first.add_strategy_note(
        StrategyCreate(
            kind=StrategyKind.THEORY,
            title="Factor (j) as facilitation, not alienation",
            body="Map dated denials to MCL 722.23(j). Do not plead PAS.",
            source_chat="owner direction",
        )
    )
    assert note.court_safe is False
    assert note.exportable is False
    assert note.disclosure == "private_strategy"

    reloaded = Workspace(tmp_path)
    stored = reloaded.list_strategy_notes()
    assert len(stored) == 1
    assert stored[0].note_id == note.note_id
    assert stored[0].court_safe is False
    assert all(not item.exportable for item in stored)

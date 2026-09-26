"""Red-team runs and todos persist and stay non-exportable.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from legal_workspace.domain.redteam import RedTeamCreate, RedTeamFinding, RedTeamLens
from legal_workspace.domain.todos import TodoCreate, TodoStatus
from legal_workspace.services.workspace import Workspace


def test_redteam_and_todos_roundtrip(tmp_path) -> None:
    first = Workspace(tmp_path)
    assert first.load().todos
    assert all(item.court_safe is False for item in first.load().todos)

    run = first.add_redteam_run(
        RedTeamCreate(
            target_type="strategy",
            target_id="theory-j",
            lens=RedTeamLens.OPPOSING_COUNSEL,
            prompt_or_notes="Attack facilitation theory as overreach.",
            findings=[
                RedTeamFinding(
                    severity="high",
                    claim="Opposing counsel will call this alienation without a statute.",
                    why="Packet forbids pleading PAS; they may still try the word.",
                )
            ],
            model="unevaluated-manual",
        )
    )
    assert run.court_safe is False
    assert run.exportable is False

    first.add_todo(TodoCreate(title="Pull the last order PDF", detail="Owner task", source="owner"))
    reloaded = Workspace(tmp_path)
    assert any(item.run_id == run.run_id for item in reloaded.load().redteam_runs)
    opened = [item for item in reloaded.load().todos if item.status is TodoStatus.OPEN]
    assert opened
    target = next(item for item in reloaded.load().todos if item.title == "Pull the last order PDF")
    reloaded.set_todo_status(target.todo_id, TodoStatus.DONE)
    assert Workspace(tmp_path).load().todos
    assert next(
        item for item in Workspace(tmp_path).load().todos if item.todo_id == target.todo_id
    ).status is TodoStatus.DONE

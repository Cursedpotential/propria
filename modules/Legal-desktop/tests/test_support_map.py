"""Support maps mark outline lines as not factual.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from legal_workspace.domain.support_map import SupportState, build_support_map
from legal_workspace.domain.templates import TemplateInstantiate
from legal_workspace.services.workspace import Workspace


def test_template_body_is_mostly_not_factual() -> None:
    mapped = build_support_map(
        section_id="x",
        heading="Motion",
        body="Motion for parenting time\n\nNot an official form. Not filing-ready.\n\n1. Caption / case number: [clerk-confirm]\n\nA factual claim without a pin.",
        citation_count=0,
        citations_ok=False,
    )
    states = [item.state for item in mapped.paragraphs]
    assert SupportState.NOT_FACTUAL in states
    assert SupportState.UNSUPPORTED in states
    assert mapped.unsupported_count >= 1


def test_instantiated_template_has_a_support_map(tmp_path) -> None:
    workspace = Workspace(tmp_path)
    section = workspace.instantiate_template(
        TemplateInstantiate(template_id="notice-of-hearing")
    )
    from legal_workspace.api.main import _draft_view
    from legal_workspace.api import main as main_mod
    from legal_workspace.services import workspace as workspace_mod

    main_mod.WORKSPACE = workspace_mod.get_workspace(tmp_path)
    view = _draft_view(section)
    assert view.support is not None
    assert view.support.paragraphs
    assert any(item.state.value == "not_factual" for item in view.support.paragraphs)

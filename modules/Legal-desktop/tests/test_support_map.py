"""Support maps mark outline lines as not factual.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

import pytest

from legal_workspace.domain.support_map import SupportState, build_support_map
from legal_workspace.domain.templates import TemplateInstantiate
from legal_workspace.services.workspace import Workspace


@pytest.mark.parametrize("citations_ok", [True, False])
def test_section_citations_do_not_support_its_paragraphs(citations_ok) -> None:
    mapped = build_support_map(
        section_id="citation-section",
        heading="Parenting contact",
        body=(
            "The exchange occurred at the ordered location.\n\n"
            "The other parent refused all telephone contact for six weeks."
        ),
        citation_count=1,
        citations_ok=citations_ok,
    )
    assert mapped.citation_count == 1
    assert [item.state for item in mapped.paragraphs] == [
        SupportState.UNSUPPORTED,
        SupportState.UNSUPPORTED,
    ]
    assert mapped.unsupported_count == 2


@pytest.mark.parametrize(
    "text",
    [
        "1. I arrived at the exchange location at 6 p.m.",
        "12. The other parent did not appear.",
        "3) I called twice and received no response.",
        "She said the document was not an official order.",
        "I wrote [unknown] in my contemporaneous notes.",
    ],
)
def test_numbering_or_embedded_instruction_words_do_not_hide_facts(text) -> None:
    mapped = build_support_map(
        section_id="facts",
        heading="Affidavit",
        body=text,
        citation_count=2,
        citations_ok=True,
    )
    assert mapped.paragraphs[0].state is SupportState.UNSUPPORTED
    assert mapped.unsupported_count == 1


def test_explicit_instructions_do_not_count_as_missing_factual_support() -> None:
    mapped = build_support_map(
        section_id="instructions",
        heading="Outline",
        body=(
            "Working outline\n"
            "1. Authority: verify the current rule.\n"
            "2) [clerk-confirm the case number]\n"
            "Do not invent dates.\n"
            "4. The exchange was cancelled on March 12."
        ),
        citation_count=3,
        citations_ok=True,
    )
    assert [item.state for item in mapped.paragraphs] == [
        SupportState.NOT_FACTUAL,
        SupportState.NOT_FACTUAL,
        SupportState.NOT_FACTUAL,
        SupportState.NOT_FACTUAL,
        SupportState.UNSUPPORTED,
    ]
    assert mapped.unsupported_count == 1


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

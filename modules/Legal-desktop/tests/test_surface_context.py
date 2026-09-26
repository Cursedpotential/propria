"""Saved surface snapshot for F1 context injection.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from legal_workspace.services.workspace import Workspace


def test_surface_context_home_and_factors(tmp_path) -> None:
    store = Workspace(tmp_path)
    home = store.surface_context("/")
    assert home["path"] == "/"
    assert "matter" in home
    assert isinstance(home["saved"], dict)
    factors = store.surface_context("/custody-factors")
    assert isinstance(factors["saved"], list)
    assert factors["saved"]
    assert "letter" in factors["saved"][0]
    assert "assistant only" in str(factors["background"]).lower()
    assert str(factors["background"]).count("Both-parent") == 1

"""Verify independent feature admission preserves the shared worker.

Byline: Codex · GPT-6 · 2026-10-11.
Inputs: explicit deployment settings. Outputs: Activity admission assertions.
Effects: temporary directories only; choose to verify failure isolation without infrastructure.
"""

from server.temporal import discovery_registration


def test_disabled_discovery_does_not_import_model(monkeypatch):
    """An independent disabled flag admits no model operation despite unusable settings."""
    monkeypatch.setenv("ADAPTIVE_DISCOVERY_ENABLED", "false")
    monkeypatch.setenv("ADAPTIVE_DISCOVERY_MODEL_CONFIG_JSON", "broken")
    assert discovery_registration.optional_discovery_activities() == []


def test_invalid_discovery_configuration_does_not_stop_worker(monkeypatch):
    """Missing derivative storage disables only the optional Activity."""
    monkeypatch.setenv("ADAPTIVE_DISCOVERY_ENABLED", "true")
    monkeypatch.setenv("ADAPTIVE_DISCOVERY_ROOT", "relative")
    assert discovery_registration.optional_discovery_activities() == []


def test_admitted_discovery_registers_without_loading_encoder(monkeypatch, tmp_path):
    """Pinned local configuration admits the Activity with no encoder allocation."""
    import json

    from server.analysis.discovery import scoring
    from server.temporal.discovery_activities import DISCOVERY_ACTIVITIES

    monkeypatch.setenv("ADAPTIVE_DISCOVERY_ENABLED", "true")
    monkeypatch.setenv("ADAPTIVE_DISCOVERY_ROOT", str(tmp_path))
    monkeypatch.setenv(
        "ADAPTIVE_DISCOVERY_MODEL_CONFIG_JSON",
        json.dumps({"model_path": str(tmp_path), "model_revision": "retained-snapshot-v1"}),
    )
    monkeypatch.setattr(discovery_registration.importlib.util, "find_spec", lambda name: object())

    def unexpected_load(config):
        raise AssertionError("startup must not allocate a CPU model")

    monkeypatch.setattr(scoring, "local_encoder", unexpected_load)
    assert discovery_registration.optional_discovery_activities() == DISCOVERY_ACTIVITIES

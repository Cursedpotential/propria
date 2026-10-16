"""Compose optional local discovery Activities into the existing worker.

Byline: Codex · GPT-6 · 2026-10-11.
Inputs: explicit feature flag, local model pins and shared derivative path.
Outputs: optional Activity list. Effects: bounded configuration/metadata reads;
choose independently of intake, with failure isolated to discovery.
"""

from __future__ import annotations

import importlib.util
import logging
import os
from pathlib import Path
from typing import Any

log = logging.getLogger("temporal.discovery")


def optional_discovery_activities() -> list[Any]:
    """Admit the CPU Activity only when its explicit deployment settings are usable.

    Inputs: ADAPTIVE_DISCOVERY_ENABLED, model configuration and shared root.
    Outputs: zero or one Activity. Effects: metadata/imports only, no model load;
    choose when composing the shared worker so setup failures leave intake running.
    """
    flag = os.environ.get("ADAPTIVE_DISCOVERY_ENABLED", "false").strip().lower()
    if flag in {"", "false", "0", "f"}:
        return []
    if flag not in {"true", "1", "t"}:
        log.error("adaptive discovery disabled: feature flag is invalid")
        return []
    try:
        from server.analysis.discovery.scoring import model_config
        from server.analysis.discovery.storage import DiscoveryInvalid, root
        from server.temporal.discovery_activities import DISCOVERY_ACTIVITIES

        root()
        config = model_config({})
        model = Path(config.get("model_path", ""))
        if not model.is_absolute() or not model.is_dir():
            raise DiscoveryInvalid("a retained absolute CPU model directory is required")
        if importlib.util.find_spec("sentence_transformers") is None:
            raise DiscoveryInvalid("the CPU discovery encoder dependency is unavailable")
    except (ValueError, ImportError, OSError):
        log.error("adaptive discovery disabled: local model or derivative configuration unavailable")
        return []
    log.info("adaptive discovery CPU Activity registered")
    return list(DISCOVERY_ACTIVITIES)

from __future__ import annotations

import json
from pathlib import Path
from typing import Any


CONFIG_ROOT = Path("deploy/docker/gateway/portkey/configs")
ACTIVE_CONFIGS = (
    "context-vl-embed.json",
    "context-vl-rerank.json",
    "context-omni-extract.json",
)
FAILOVER_CODES = [429, 500, 502, 503, 504, 524, 529]


def _read(name: str) -> dict[str, Any]:
    return json.loads((CONFIG_ROOT / name).read_text(encoding="utf-8"))


def _providers(branch: dict[str, Any]) -> list[str]:
    return [str(target["custom_host"]) for target in branch["targets"]]


def test_active_configs_rotate_opposite_provider_orders_with_failover() -> None:
    for name in ACTIVE_CONFIGS:
        config = _read(name)
        assert config["strategy"] == {"mode": "loadbalance"}
        assert len(config["targets"]) == 2
        first, second = config["targets"]
        assert first["weight"] == second["weight"] == 1
        assert first["strategy"] == {"mode": "fallback", "on_status_codes": FAILOVER_CODES}
        assert second["strategy"] == {"mode": "fallback", "on_status_codes": FAILOVER_CODES}
        assert _providers(first) == list(reversed(_providers(second)))
        assert all(target["api_key"].startswith("$") for branch in (first, second) for target in branch["targets"])


def test_embedding_fallbacks_cannot_mix_models_or_dimensions() -> None:
    config = _read("context-vl-embed.json")
    targets = [target for branch in config["targets"] for target in branch["targets"]]
    models = {target["override_params"]["model"].removesuffix(":free") for target in targets}
    assert models == {"nvidia/llama-nemotron-embed-vl-1b-v2"}
    assert "2048" in config["_comment"]


def test_granite_fallback_is_config_only_and_zero_weight() -> None:
    config = _read("document-extract-granite.json")
    assert "DISABLED" in config["_comment"]
    assert config["targets"][0]["weight"] == 0
    assert config["targets"][0]["custom_host"] == "$HF_GRANITE_DOCLING_ENDPOINT"
    assert config["targets"][0]["api_key"] == "$HF_TOKEN"

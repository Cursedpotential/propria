"""Deployment contract for LibreChat: baked config, ContextForge MCP servers, Portkey chat lane.

Byline: Codex · GPT-5.6-Sol · 2026-08-29
Byline: Claude Code · Fable 5.1 · 2026-09-26 (MCP straight to ContextForge virtual servers; the
Portkey-published "platform-tools" endpoint of 2026-08-29 was never built)
"""

import re
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[1]
DEPLOY = ROOT / "deploy" / "librechat.yaml"
DOCKERFILE = ROOT / "deploy" / "docker" / "librechat" / "Dockerfile"
CONFIG = ROOT / "deploy" / "docker" / "librechat" / "librechat.yaml"
CONTEXTFORGE_HOST = "contextforge.tilapia-skilift.ts.net"


def test_librechat_bakes_the_tracked_mcp_config() -> None:
    manifest = yaml.safe_load(DEPLOY.read_text(encoding="utf-8"))
    service = manifest["services"]["librechat"]

    # Coolify builds with --project-directory = base_directory, so paths are module-root relative.
    assert service["build"] == {"context": ".", "dockerfile": "deploy/docker/librechat/Dockerfile"}
    assert all("librechat.yaml:/app/librechat.yaml" not in volume for volume in service["volumes"])
    dockerfile = DOCKERFILE.read_text(encoding="utf-8")
    assert "COPY deploy/docker/librechat/librechat.yaml /app/librechat.yaml" in dockerfile
    assert "@sha256:" in dockerfile


def test_librechat_mcp_servers_are_contextforge_virtual_servers() -> None:
    manifest = yaml.safe_load(DEPLOY.read_text(encoding="utf-8"))
    environment = manifest["services"]["librechat"]["environment"]
    raw = CONFIG.read_text(encoding="utf-8")
    config = yaml.safe_load(raw)

    # Fail closed: the deploy stops when the token is missing instead of starting without tools.
    assert environment["CONTEXTFORGE_MCP_TOKEN"].startswith("${CONTEXTFORGE_MCP_TOKEN:?")
    servers = config["mcpServers"]
    assert servers
    url_shape = re.compile(rf"https://{re.escape(CONTEXTFORGE_HOST)}/servers/[0-9a-f]{{32}}/mcp")
    for name, server in servers.items():
        assert server["type"] == "streamable-http", name
        assert url_shape.fullmatch(server["url"]), name
        assert server["headers"] == {"Authorization": "Bearer ${CONTEXTFORGE_MCP_TOKEN}"}, name
        # A bearer server: skip v0.8.7's header-less OAuth probe, which would list 0 tools.
        assert server["requiresOAuth"] is False, name
    # LibreChat treats the tailnet (100.64.0.0/10) as private: the host must be exempted exactly.
    assert f"{CONTEXTFORGE_HOST}:443" in config["mcpSettings"]["allowedAddresses"]
    assert "allowedDomains" not in config["mcpSettings"]
    assert "PORTKEY_MCP_API_KEY" not in raw
    assert "PORTKEY_PLATFORM_TOOLS_MCP_URL" not in raw
    assert [key for key in environment if key.startswith("PORTKEY_")] == ["PORTKEY_CHAT_CONFIG"]
    assert "agentos" not in raw.lower()


def test_librechat_chat_lane_stays_on_the_portkey_gateway() -> None:
    config = yaml.safe_load(CONFIG.read_text(encoding="utf-8"))
    (endpoint,) = config["endpoints"]["custom"]

    assert endpoint["baseURL"] == "http://100.72.169.40:8787/v1"
    assert endpoint["headers"] == {"x-portkey-config": "${PORTKEY_CHAT_CONFIG}"}
    assert endpoint["models"] == {"default": ["platform-chat"], "fetch": False}

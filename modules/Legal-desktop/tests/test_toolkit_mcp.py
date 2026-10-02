"""Toolkit MCP client: fail-closed behavior with no console configured.

Byline: Claude Code · Opus 5.5 · 2026-10-02
No server stand-in: discovery and invocation are verified against the deployed
console (live receipt in the commit and URGENT-TODO). These tests cover what must
hold with no console at all, and the desk-side write policy.
"""

import pytest
from legal_workspace.services import family_court_toolkit, toolkit_mcp


@pytest.fixture(autouse=True)
def no_console(monkeypatch):
    monkeypatch.delenv("FAMILY_COURT_CONSOLE_MCP_URL", raising=False)
    monkeypatch.delenv("FAMILY_COURT_CONSOLE_MCP_TOKEN", raising=False)


def test_connection_reports_not_configured(client_with_auth):
    body = client_with_auth.get("/v1/mcp/connections").json()
    assert body == [
        {
            "id": "family-court-console",
            "name": "Family Law Toolkit console",
            "transport": "streamable-http",
            "configured": False,
            "reachable": False,
            "tool_count": 0,
            "detail": "FAMILY_COURT_CONSOLE_MCP_URL is not set",
        }
    ]


def test_tools_and_invocations_fail_closed(client_with_auth):
    assert client_with_auth.get("/v1/mcp/tools").status_code == 503
    response = client_with_auth.post("/v1/mcp/invocations", json={"tool": "search_guide", "arguments": {"query": "x y"}})
    assert response.status_code == 503


def test_unregistered_connection_is_refused(client_with_auth):
    response = client_with_auth.post("/v1/mcp/invocations", json={"connection_id": "other", "tool": "search_guide"})
    assert response.status_code == 404


def test_console_url_must_be_a_name(monkeypatch):
    monkeypatch.setenv("FAMILY_COURT_CONSOLE_MCP_URL", "http://100.91.190.107:9077/mcp")
    with pytest.raises(ValueError):
        toolkit_mcp.get_settings()


class _Tool:
    def __init__(self, name, read_only):
        self.name = name
        self.title = None
        self.description = ""
        self.input_schema = {"type": "object", "properties": {}}
        self.annotations = None if read_only is None else _Annotations(read_only)


class _Annotations:
    def __init__(self, read_only):
        self.read_only = read_only

    def model_dump(self, **_):
        return {"readOnlyHint": self.read_only}


@pytest.mark.parametrize(
    ("name", "read_only", "writes"),
    [
        ("search_guide", True, False),
        ("case_put", False, True),
        ("case_status", None, True),  # no annotation: assume it writes
        ("case_query", True, True),  # runs CREATE/UPDATE without write: true
        ("case_export", True, True),
        ("case_import", False, True),
    ],
)
def test_write_policy(name, read_only, writes):
    assert toolkit_mcp._definition(_Tool(name, read_only)).writes is writes


def test_factors_are_shared_with_the_desk():
    assert "factor" in family_court_toolkit.TOOLKIT_TABLES

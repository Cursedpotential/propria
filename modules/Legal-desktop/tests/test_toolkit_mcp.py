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
        ("case_query", True, False),  # SELECT-only shared inspection
        ("case_export", True, True),
        ("family-court-case-export", True, True),
        ("family-court-case-query", True, False),
        ("case_import", False, True),
    ],
)
def test_write_policy(name, read_only, writes):
    assert toolkit_mcp._definition(_Tool(name, read_only)).writes is writes


def test_toolkit_read_allowlist_covers_personal_case_tables_without_workflow_internals():
    """Verify DATA_TABLES coverage and preserve synthetic full-name record refs.

    Inputs: toolkit service allowlist and synthetic identifiers. Outputs: assertions.
    Effects: no store or database access. Sibling: test_toolkit_mcp write-policy coverage.
    Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04.
    """
    # Synthetic full-name IDs verify personal references remain ordinary values.
    expected_tables = {
        "person", "child", "order", "hearing", "deadline", "event", "message",
        "exhibit", "factor", "source", "note", "court", "court_event", "filing",
        "draft", "memo", "reference", "evidence_log", "eval", "case_status",
    }
    assert set(family_court_toolkit.TOOLKIT_TABLES) == expected_tables
    assert {"library_validation", "library_proposal", "library_revision"}.isdisjoint(
        family_court_toolkit.TOOLKIT_TABLES
    )
    assert family_court_toolkit._split_ref("person:Jordan Example") == ("person", "Jordan Example")
    assert family_court_toolkit._split_ref("child:Casey Example") == ("child", "Casey Example")
    for table in ("library_validation", "library_proposal", "library_revision"):
        with pytest.raises(ValueError, match="not shared"):
            family_court_toolkit._split_ref(f"{table}:synthetic-id")

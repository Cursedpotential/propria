"""Search attaches only the critical flags that bear on the query.

Byline: Claude Code · Opus 5.5 · 2026-09-28. Titles and summaries mirror live flags returned on
2026-09-28, when every search carried all 20 in full.
"""

from flag_relevance import EXCERPT_CHARS, relevant_flags

LIVE_SHAPED = {
    "status": "available",
    "truncated": True,
    "flags": [
        {"id": "note:ws00", "title": "WS00 remaining-source matrix: no new history is import-eligible",
         "subject": "Persist the exact no-import boundary", "authority": "verified_finding",
         "summary": "Verified WS00 matrix. Consignatio, Legal/Advocatio, TraceIQ, Vestigia remain HOLD. " * 6,
         "rationale": "long rationale " * 40, "change_hash": "a" * 64, "source_ref": "E:/x/y.md",
         "updated_at": "2026-09-24T12:20:33Z"},
        {"id": "note:d07", "title": "D07 Indagatio contracts mapped; cross-store horizon and authored state remain held",
         "subject": "analysis roles", "authority": "verified_finding",
         "summary": "D-143/144 settle LlamaIndex retrieval plus SAT extraction and LangGraph Activity-shaped "
                    "stitch/verify/retrieval roles. No bounded first-party LlamaIndex/LangGraph imports were found.",
         "updated_at": "2026-09-24T11:47:59Z"},
        {"id": "note:r2", "title": "Workbench R2 blocked by account entitlement; key-only retry cannot repair 10042",
         "subject": "R2 entitlement", "authority": "verified_finding",
         "summary": "Both credential pairs return R2 error 10042 NotEntitled for casebible-sorted and nexus.",
         "updated_at": "2026-09-24T11:41:57Z"},
        {"id": "note:ports", "title": "Port class prefix, stable product suffix, and Tailscale DNS only",
         "subject": "service addressing", "authority": "owner_decision",
         "summary": "Applications and agents use only Tailscale service DNS; raw 100.x ports are host routing.",
         "updated_at": "2026-09-12T17:32:26Z"},
    ],
}


def test_unrelated_query_attaches_no_flags():
    out = relevant_flags(LIVE_SHAPED, "MCP Inspector deploy for testing servers")
    assert out["relevant_shown"] == 0 and out["flags"] == []
    assert out["active_in_domain"] == "4+"  # truncated upstream: the count says so
    assert "docstore_flags" in out["read_all"]


def test_related_query_keeps_only_the_matching_flag():
    out = relevant_flags(LIVE_SHAPED, "replace Agno retire AgentOS adapter LangGraph LlamaIndex")
    assert [f["id"] for f in out["flags"]] == ["note:d07"]


def test_short_query_needs_one_shared_term():
    out = relevant_flags(LIVE_SHAPED, "Tailscale DNS")
    assert [f["id"] for f in out["flags"]] == ["note:ports"]


def test_flags_are_trimmed_to_excerpts_without_bulk_fields():
    out = relevant_flags(LIVE_SHAPED, "Consignatio Vestigia matrix")
    flag = out["flags"][0]
    assert flag["id"] == "note:ws00"
    assert len(flag["excerpt"]) <= EXCERPT_CHARS + 1
    assert set(flag) == {"id", "title", "authority", "updated_at", "excerpt"}


def test_at_most_three_and_status_passes_through():
    many = {"status": "available", "flags": [
        {"id": f"note:{i}", "title": f"entitlement item {i}", "summary": "R2 entitlement"} for i in range(8)]}
    out = relevant_flags(many, "R2 entitlement")
    assert out["relevant_shown"] == 3 and out["status"] == "available"


def test_unavailable_flags_keep_their_warning():
    out = relevant_flags({"status": "unavailable", "flags": [], "warning": "Critical decisions could not be verified"}, "x y z")
    assert out["warning"] == "Critical decisions could not be verified" and out["flags"] == []

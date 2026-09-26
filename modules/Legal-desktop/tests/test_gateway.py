"""Gateway invoke is optional and fail-closed.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from legal_workspace.domain.agents import AgentRunCreate, AgentRunStatus
from legal_workspace.services.gateway import GatewayResult, invoke_chat
from legal_workspace.services.workspace import Workspace


def test_forbidden_intent_does_not_call_invoker(tmp_path) -> None:
    called = []

    def invoker(prompt: str, model: str) -> GatewayResult:
        called.append(prompt)
        return GatewayResult(ok=True, text="should not run", model=model)

    run = Workspace(tmp_path).add_agent_run(
        AgentRunCreate(intent="file this motion", prompt="no"),
        invoker=invoker,
    )
    assert run.status is AgentRunStatus.BLOCKED
    assert called == []


def test_invoker_output_is_hypothesis(tmp_path) -> None:
    run = Workspace(tmp_path).add_agent_run(
        AgentRunCreate(intent="research Vodvarka lookback", prompt="outline only"),
        invoker=lambda prompt, model: GatewayResult(ok=True, text="Need the last order date.", model="test-model"),
    )
    assert run.effective_model == "test-model"
    assert "hypothesis only" in run.output
    assert run.court_safe is False
    assert Workspace(tmp_path).load().agent_runs


def test_invoke_chat_fail_closed_without_gateway() -> None:
    import httpx

    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(503, text="down")

    transport = httpx.MockTransport(handler)
    with httpx.Client(transport=transport) as client:
        result = invoke_chat("ping", "unevaluated-manual", client=client)
    assert result.ok is False
    assert result.model == "gateway-unreachable"


def test_confidential_blocks_consumer_claude_without_http() -> None:
    import httpx

    called: list[str] = []

    def handler(request: httpx.Request) -> httpx.Response:
        called.append(str(request.url))
        return httpx.Response(200, json={"choices": [{"message": {"content": "should not run"}}]})

    transport = httpx.MockTransport(handler)
    with httpx.Client(transport=transport, timeout=2.0) as client:
        result = invoke_chat("secret", "claude-consumer", client=client, confidential=True)
    assert result.ok is False
    assert result.text == "confidential_blocked"
    assert result.model == "claude-consumer"
    assert called == []


def test_confidential_blocks_chatgpt_no_silent_fallback() -> None:
    import httpx

    called: list[str] = []

    def handler(request: httpx.Request) -> httpx.Response:
        called.append(str(request.url))
        return httpx.Response(200, json={"choices": [{"message": {"content": "fallback"}}]})

    transport = httpx.MockTransport(handler)
    with httpx.Client(transport=transport, timeout=2.0) as client:
        result = invoke_chat("secret", "chatgpt-consumer", client=client, confidential=True)
    assert result.ok is False
    assert result.text == "confidential_blocked"
    assert result.model == "chatgpt-consumer"
    assert called == []


def test_confidential_allows_ollama_cloud() -> None:
    import json

    import httpx

    def handler(request: httpx.Request) -> httpx.Response:
        payload = json.loads(request.content)
        assert payload["model"] == "ollama-cloud"
        return httpx.Response(200, json={"choices": [{"message": {"content": "hypothesis"}}]})

    transport = httpx.MockTransport(handler)
    with httpx.Client(transport=transport, timeout=2.0) as client:
        result = invoke_chat("ping", "ollama-cloud", client=client, confidential=True)
    assert result.ok is True
    assert result.model == "ollama-cloud"
    assert result.text == "hypothesis"


def test_ineligible_model_gate_does_not_call_transport() -> None:
    """Shipped invoke_chat must not POST when the model is not eligible."""
    import httpx

    called: list[str] = []

    def handler(request: httpx.Request) -> httpx.Response:
        called.append(str(request.url))
        return httpx.Response(200, json={"choices": [{"message": {"content": "LEAK"}}]})

    transport = httpx.MockTransport(handler)
    with httpx.Client(transport=transport, timeout=2.0) as client:
        result = invoke_chat(
            "privileged strategy",
            "claude-consumer",
            client=client,
            confidential=True,
        )
    assert result.ok is False
    assert result.text == "confidential_blocked"
    assert result.model == "claude-consumer"
    assert "LEAK" not in result.text
    assert called == []


def test_confidential_does_not_rewrite_blocked_model() -> None:
    import httpx

    called: list[str] = []

    def handler(request: httpx.Request) -> httpx.Response:
        called.append(json_model(request))
        return httpx.Response(200, json={"choices": [{"message": {"content": "nope"}}]})

    def json_model(request: httpx.Request) -> str:
        import json

        return json.loads(request.content)["model"]

    transport = httpx.MockTransport(handler)
    with httpx.Client(transport=transport, timeout=2.0) as client:
        claude = invoke_chat("x", "claude-consumer", client=client, confidential=True)
        nim = invoke_chat("x", "nim-hosted", client=client, confidential=True)
        unknown = invoke_chat("x", "unevaluated-manual", client=client, confidential=True)
    assert claude.model == "claude-consumer"
    assert nim.model == "nim-hosted"
    assert unknown.model == "unevaluated-manual"
    assert {claude.text, nim.text, unknown.text} == {"confidential_blocked"}
    assert called == []

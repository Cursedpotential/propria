"""Which Claude model ids answer through the Claude Agent SDK on the owner's Max login (owner 21:40: "What models are
available through the Claude SDK? Is any of the version 4 models available?").

Byline: Claude Code · Opus 5.5 · 2026-09-24. One tiny prompt per model id; prints the model that answered or the error.
Runs in the ovh-files devbox with the jev-eval venv and `--env-file /data/probata/secrets/jev-eval/claude.env`.
    .venv/bin/python code/claude_models_probe.py
"""

import asyncio

from claude_agent_sdk import AssistantMessage, ClaudeAgentOptions, ResultMessage, TextBlock, query

CANDIDATES = [
    "claude-opus-5-5", "claude-sonnet-5", "claude-fable-5-1", "claude-haiku-4-5-20251001", "claude-haiku-4-5",
    "claude-opus-4-5", "claude-opus-4-1-20250805", "claude-opus-4-1", "claude-opus-4-20250514", "claude-opus-4-0",
    "claude-sonnet-4-5-20250929", "claude-sonnet-4-5", "claude-sonnet-4-20250514", "claude-sonnet-4-0",
    "claude-3-7-sonnet-20250219", "claude-3-5-haiku-20241022", "opus", "sonnet", "haiku",
]


async def probe(model: str) -> str:
    opts = ClaudeAgentOptions(model=model, system_prompt="Reply with the single word ok.", tools=[], allowed_tools=[],
                              max_turns=1)
    text, used, err = "", "", ""
    try:
        async for m in query(prompt="ok?", options=opts):
            if isinstance(m, AssistantMessage):
                used = getattr(m, "model", "") or used
                text += "".join(b.text for b in m.content if isinstance(b, TextBlock))
            if isinstance(m, ResultMessage) and m.is_error:
                err = str(getattr(m, "result", ""))[:160]
    except Exception as e:  # recorded, never silent
        err = f"{type(e).__name__}: {e}"[:160]
    ok = bool(text.strip()) and not err
    return f"{'OK  ' if ok else 'FAIL'} {model:<30} answered_by={used or '-'} {repr(text.strip()[:20]) if ok else err}"


async def main() -> None:
    for m in CANDIDATES:
        print(await probe(m), flush=True)


asyncio.run(main())

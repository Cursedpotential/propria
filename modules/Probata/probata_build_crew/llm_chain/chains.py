"""The crew's LLM chain: Ollama Cloud nemotron primary, OpenRouter FREE models as fallback.

Referenced from agents/*.jsonc as {"python": "llm_chain.chains.default_chain"} (instance) or
{"python": "llm_chain.chains.DefaultChainLLM"} (class), whichever the JSON loader prefers.

Fallback members are FREE OpenRouter ids only (owner ruling 2026-09-06 22:00: "query for free models, use
them only, add fallback"), live-probed 2026-09-06 22:04 with openrouter-free-probe: all four answered chat and
returned a real tool call. Ordered by family closeness to the primary, then speed.

Byline: Claude Code · Fable 5.1 · 2026-09-06
"""

import os

from llm_chain.fallback import FallbackLLM

PRIMARY = os.environ.get("CREW_PRIMARY_LLM", "ollama/nemotron-3-super")

FREE_FALLBACKS = [
    "openrouter/nvidia/nemotron-3-super-120b-a12b:free",           # same model as primary, tools 0.8 s
    "openrouter/nvidia/nemotron-3-nano-omni-30b-a3b-reasoning:free",  # tools 2.4 s
    "openrouter/google/gemma-4-31b-it:free",                       # tools 1.1 s, 262k ctx
    "openrouter/minimax/minimax-m3:free",                          # tools 3.1 s, 1M ctx
]

CHAIN = [PRIMARY, *FREE_FALLBACKS]


class DefaultChainLLM(FallbackLLM):
    """Zero-argument class form of the default chain (for loaders that instantiate a class)."""

    def __init__(self, **kwargs: object) -> None:
        # Long markdown deliverables: keep the output budget high on every member (Ollama maps this to num_predict).
        # Reasoning models spend part of this budget thinking (nemotron exposes `reasoning`), so 16k truncated a
        # ~20 KB answer on run-08d; 40k leaves room for both.
        kwargs.setdefault("max_tokens", int(os.environ.get("CREW_MAX_TOKENS", "40000")))
        # Every member has >= 256k context (nemotron-3-super 262k, nano-omni 256k, gemma-4-31b 262k, minimax-m3 1M);
        # CrewAI would otherwise assume ~8k for these unknown ids and summarize the research away.
        kwargs.setdefault("context_window", int(os.environ.get("CREW_CONTEXT_WINDOW", "200000")))
        super().__init__(chain=list(CHAIN), **kwargs)


default_chain = DefaultChainLLM()

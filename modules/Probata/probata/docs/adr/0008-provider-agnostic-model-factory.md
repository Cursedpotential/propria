# ADR-0008: Provider-agnostic model factory, no hard default, pinned IDs

> **SUPERSEDED** by ADR-0042 (2026-07-29) — the D7 provider-selection question this ADR left open was closed by ADR-0011, re-closed by ADR-0015, and ADR-0015 is itself now superseded by ADR-0042 (Portkey is the model gateway); this file's "D7 still open" status line is stale. Body below is preserved as history.
>
> _Byline amendment: Claude Code · Sonnet 5 · 2026-09-09 — supersession banner added; see ADR-0042._

- Status: ~~Accepted (provider selection (D7) still open)~~ **Superseded** — D7 closed via ADR-0011 → ADR-0015 → ADR-0042; see banner above.
- Date: 2026-06-01

## Context
The v1 repo hard-defaulted to `gpt-4o`; the skeleton defaults to NVIDIA NIM / OpenRouter. The handoff
requires a provider-agnostic factory that selects by available credentials, with **no hard default** and
**pinned versioned model IDs**. Structured-output-strong models matter for LearningMachine extraction.

## Decision
`app/settings.py` exposes a **provider-agnostic factory** that selects by available credentials in a
fixed preference order, with **no hard default**, and **pins versioned IDs** (Anthropic `claude-opus-4-8`
/ `claude-sonnet-4-6` per the handoff). Embeddings use OpenAI `text-embedding-3-small` (1536-d) and
require an OpenAI key regardless of the chat provider. Point the **extraction** step at a
structured-output-strong model if chat agents run on a smaller/local model.

## Open (D7)
Which provider agents actually run on (Anthropic pinned vs skeleton's NVIDIA/OpenRouter vs OpenAI) and
which keys are available — to be confirmed before Phase 1 implementation. This ADR locks the *factory
shape*, not the runtime provider.

## Consequences
- Switching the embedder requires re-embedding the whole corpus (dimension contract).
- No silent fallback to a stale default model.

## Alternatives considered
- Hard default model (v1 `gpt-4o`; skeleton NVIDIA) — rejected: stale/locked-in.

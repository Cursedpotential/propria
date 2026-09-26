# ADR-0039: Graphiti extraction LLM — hosted structured-output provider (no local GPU)

> **SUPERSEDED** by D-070 (2026-08-25) — "Graphiti is retired for now"; this ADR is one of the five (ADR-0014/0031/0037/0038/0039) D-070 names explicitly as suspended, not deleted. Body below is preserved as history.
>
> _Byline amendment: Claude Code · Sonnet 5 · 2026-09-09 — supersession banner added; see D-070._

> _Byline: Claude (Opus 4.8, chat) + owner · 2026-07-13 · DRAFT for review_

**Status:** ~~**Accepted**~~ **Suspended (superseded)** — see banner above. Owner 2026-07-29 (Proposed 2026-07-13). Already implemented in practice: extraction moved to NIM `nemotron` guided-JSON on 2026-07-04 (glm-5.1 could not conform — the exact silent-failure class this ADR guards against), and the lane now routes through the Portkey gateway (ADR-0042). The embedder Open item is unblocked: ADR-0040 (Weaviate) keeps the nv-embed-v1 4096-d contract, no re-embed.
**Supersedes/relates:** [ADR-0011](0011-nvidia-nim-provider-and-embedder-dimension-contract.md) (NIM provider + embedder dimension contract), [ADR-0015](0015-litellm-gateway-ollama-primary.md) (LiteLLM gateway, Ollama primary), [ADR-0037](0037-graphiti-mcp-contextforge-write-enabled.md).

## Context

Every Graphiti write runs LLM extraction (entity/edge extraction, dedup, contradiction resolution) that depends on **reliable structured (JSON) output**. Zep's own guidance warns that small/local models frequently emit schema-invalid JSON, surfacing as silent ingestion failures. For forensic memory, silent data loss is unacceptable. Hardware constraint: **no local GPU, and no cloud GPU desired.** Available: NVIDIA NIM cloud APIs, OpenRouter (paid history, higher limits), Ollama Cloud Pro, Colab Pro.

## Decision

Use a **hosted, structured-output-reliable provider** for Graphiti extraction:

- **Primary: NVIDIA NIM cloud** (already provisioned; strong structured output).
- **Alternate: OpenRouter** with a JSON-reliable model (paid tier).
- **Ollama Cloud** reserved for embeddings or lighter passes — **never** extraction on forensic data.
- Never a small/local model for extraction.

Operational: tune `SEMAPHORE_LIMIT` to the chosen provider's rate tier; **disable graphiti-core anonymous telemetry** (privacy).

## Consequences

- Reliable extraction; per-write API spend is the trade. Throughput is bounded by provider rate tier, not hardware.
- For deterministic facts that don't need extraction, prefer `add_triplet` (no LLM call).

## Open

- Specific NIM / OpenRouter model id for extraction.
- Embedder choice + dimension — **gated by the pending vector-DB decision** (nv-embed 4096-d vs pgvector index cap). Do not finalize the embedder here until that ADR lands. Relates to ADR-0010/0011/0026/0027.

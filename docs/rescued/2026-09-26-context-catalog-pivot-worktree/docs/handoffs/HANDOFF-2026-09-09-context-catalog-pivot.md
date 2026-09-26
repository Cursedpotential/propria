# Context Catalog Pivot — Active Handoff

> _Byline: Codex · GPT-5 · 2026-09-09._
>
> Status: active implementation on branch `codex/context-catalog-pivot` in the
> isolated worktree `probata-worktrees/context-catalog-pivot`. The original dirty
> checkout is intentionally untouched.

## Owner direction recovered

- Keep the refactor bounded to roughly two days.
- Strip PostgreSQL out of non-messaging/non-evidence ingestion wherever possible.
- AI chat data can never become evidence.
- Store ordinary context, including screenshots, in B2 under a durable SHA-256
  catalog and project it into Weaviate for search.
- Search screenshots using multimodal vectors without requiring bulk OCR at ingest.
  Extract text only when a user or downstream workflow actually needs it.
- Keep PostgreSQL for evidence-qualified material and operational/control state.
- Use CocoIndex v1 for incremental change detection and reconciliation, not as a
  hidden parse/embed/project orchestrator.
- Rotate the free OpenRouter and direct NVIDIA routes through Portkey because both
  are rate-limited.
- Configure Granite Docling as the inexpensive document-extraction fallback. The
  owner will add Hugging Face credits on 2026-09-10, so deployment and live proof of
  that paid endpoint are deferred without blocking the main refactor.

## Locked first slice

```text
source locator
  -> context SHA-256 fingerprint (not custody H1)
  -> content-addressed B2 object
  -> B2 catalog manifest
  -> Portkey multimodal embedding
  -> Weaviate ContextObjectV1 projection
  -> retrieve and rerank
  -> optional on-demand Omni extraction
```

The context lane is a sibling of Proffer. It does not modify the evidence ingest
workflow and does not write context content or context metadata to PostgreSQL.

## Authority boundaries

| Material | Canonical home | Search | PostgreSQL |
|---|---|---|---|
| AI chats | B2 object + SHA catalog | Weaviate `ContextObjectV1` | forbidden |
| Ordinary screenshots | B2 object + SHA catalog | Weaviate `ContextObjectV1` | none before an independently eligible evidence ingest |
| Derived text/captions | B2 derivative + catalog receipt | Weaviate projection | none |
| Evidence-qualified material | existing evidence/Proffer path | existing evidence projection | existing evidence/control authority |
| Workflow/control state | current control plane | not applicable | allowed |

`promotion_policy=forbidden` is permanent for AI chats. No owner shortcut,
classifier result, or workflow default may override it.

## Provider lanes

- Embed: `nvidia/llama-nemotron-embed-vl-1b-v2:free` on OpenRouter plus
  `nvidia/llama-nemotron-embed-vl-1b-v2` on NVIDIA direct. Both returned exactly
  2,048 dimensions for the same synthetic text and cosine similarity `1.0` in the
  2026-09-09 live compatibility probe. Image compatibility still requires proof.
- Rerank: `nvidia/llama-nemotron-rerank-vl-1b-v2:free` plus NVIDIA direct. A small
  adapter must normalize their different request and response schemas.
- On-demand extraction: `nvidia/nemotron-3-nano-omni-30b-a3b-reasoning:free` plus
  NVIDIA direct.
- Document fallback: `ibm-granite/granite-docling-258M`, disabled until the Hugging
  Face endpoint is funded, deployed with scale-to-zero, and proven against a real
  screenshot and rendered PDF page. `google/paligemma2-3b-mix-448` is tertiary only.

Portkey uses a 50/50 load balancer containing opposite-order fallback chains:
OpenRouter -> NVIDIA and NVIDIA -> OpenRouter. Fallback status codes are
`429, 500, 502, 503, 504, 524, 529`. Request bodies and base64 media must not be
logged. Free-provider privacy/logging constraints remain accepted owner policy.

## Work in progress

1. Cross-language context-index contract and a reference-only Go workflow package.
2. B2 content-addressed object/catalog package with no PostgreSQL dependency.
3. Fail-closed multimodal gateway and capability-locked Portkey configurations.
4. Contract, catalog, provider-normalization, and promotion-fence tests.

## Explicitly deferred

- Hugging Face endpoint creation and live Granite extraction proof, pending credits
  and CLI/MCP authentication visibility.
- Choosing and mutating the canonical Weaviate instance; two live instances were
  previously reported and must be reconciled before deployment.
- Bulk OCR, lakehouse/Unity Catalog, changes to Proffer, and evidence-lane work.

## Completion boundary

Local tests prove only code and contract behavior. The capability is not production
complete until the isolated worker is deployed through Coolify and live B2,
Weaviate, Portkey failover, provider media, and zero-PostgreSQL-write receipts are
captured.

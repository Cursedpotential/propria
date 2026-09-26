# Category 6 — Privilege, privacy & LLM routing

> _Byline: Grok · grok-4.6 · 2026-08-18_
> Source: `Legal-desktop/artifacts (3)/HANDOFF — Category 6  Privilege, Privacy & LLM Routing.md`

## Confidential Mode (agreed)

Not “route to local Ollama.” Owner has no local inference.

1. Presidio / hybrid privilege pre-flight on **every** egress.
2. Portkey prefers a verified-tier provider.
3. If none available or risk exceeds policy → **hard-block**.
4. No silent fallback.

legal-mcp `PROVIDER_POSTURES["ollama"] = fully local` is **false**
for this stack (Ollama Cloud via Portkey). Rebuild the grid.

## Provider verdicts (cited)

| Provider | What the terms actually say | Confidential-eligible? |
|---|---|---|
| **Ollama Cloud** | Privacy policy: cloud-hosted prompts processed transiently; “never train on it”; “not stored beyond the time required to fulfill the request.” [ollama.com/privacy](https://ollama.com/privacy). Pricing page repeats no-log / no-train and NCP partner ZDR. [ollama.com/pricing](https://ollama.com/pricing) | **Yes, if** we stay on Cloud API (not a consumer chat UI) and re-check terms on each upgrade |
| **NVIDIA NIM hosted** | Marketing: “data isn’t used for model training” ([NIM product](https://www.nvidia.com/en-us/ai-data-science/products/nim-microservices/)). Trial API ToS: unless disclosed for a specific API, User/Generated Content used solely to provide the service ([NVIDIA API Trial ToS PDF](https://assets.ngc.nvidia.com/products/api-catalog/legal/NVIDIA%20API%20Trial%20Terms%20of%20Service.pdf) §§2–3, cited by NVIDIA staff). General NVIDIA privacy policy still discusses training in other product lines — **do not** treat that as NIM-hosted inference | **Conditional** — eligible for embed/rerank already in Agno; for long legal prompts, treat as backup not primary until the *hosted NIM* ToS (not the website privacy policy) is filed as an AuthoritySnapshot |
| **Venice.ai** | Privacy: does not store prompt/output content (except generated video until download); ZDR contract with model providers. [venice.ai/legal/privacy-policy](https://venice.ai/legal/privacy-policy). ToS 10.2: no training on User Content; 10.11 reserves a monitoring right. Third-party models (e.g. Claude) leave Venice’s no-log perimeter ([no-log guide](https://venice.ai/blog/no-log-ai-privacy-buyers-guide)) | **Yes only on Venice-hosted / Private / TEE modes.** Not eligible if the request is relayed to Anthropic/OpenAI |
| **OpenRouter ZDR** | `provider.zdr: true` + `data_collection: "deny"` restricts routing to ZDR endpoints. ZDR does **not** apply to plugins/web search. If no ZDR endpoint can serve, the request should fail rather than fall through — confirm in Portkey that we do not set a non-ZDR fallback. [OpenRouter ZDR](https://openrouter.ai/docs/guides/features/zdr) | **Yes, only with both flags and no plugin tools** |
| Claude / ChatGPT **consumer** | Not ZDR-equivalent. *Heppner* / ABA 1.6 framing from legal-mcp stays: consumer ToS ≠ reasonable expectation of confidentiality | **No** |

## Trust grid (owner-accessible only)

| Route | Train? | Retain? | Use in Confidential Mode |
|---|---|---|---|
| Portkey → Ollama Cloud | No (cited policy) | Transient | Primary |
| Portkey → OpenRouter ZDR+deny | No (if honoured) | No (if honoured) | Secondary |
| Portkey → NIM hosted | No training claim; retention less clear | Unclear | Embed/rerank; not long privileged prose until ToS snapshotted |
| Venice Private/TEE | No | No (contract) | Optional signup; never third-party relay |
| Claude.ai / ChatGPT apps | Assume yes / retain | Yes | Block in Confidential Mode |

## Portkey (question 5)

Reuse Agno’s existing gateway. Express policy in **Legal OS code**
as well as Portkey config: try verified list in order; on empty
eligible set return 409 `confidential_blocked`. Do not rely on
Portkey alone — a mis-click in the dashboard must not silently
open Claude consumer.

## Hybrid classifier (question 6)

1. Regex/keyword first pass: attorney-client, work product, strategy,
   settlement posture, child identifiers, addresses, SSNs (reuse
   Presidio recognizers).
   **Implemented 2026-08-18:** `domain/privilege.py` `scan_text` →
   `hypothesized_markers` (attorney-client, work product, strategy,
   medical, child-identifying). `POST /v1/privilege:scan`. Never a
   legal conclusion. `court_safe=false`. No LLM on this pass.
   The scan still does not route. Confidential Mode routing is a
   separate gate in `gateway.invoke_chat(confidential=)` (2026-08-18).
2. Ambiguous only → cheap classifier via Portkey, e.g. a small
   Ollama Cloud instruct model already on the gateway (not a
   frontier model). **Not built.**
3. Hit or high score → require Confidential Mode or hard-block.
   **Not wired.** This first-pass does not change routing.

## Appendix — Colab Pro break-glass runbook (manual)

Not automated.

1. Open a **new** Colab Pro notebook. Runtime → GPU (T4 if that is
   what you get; A100 if offered).
2. Do not connect Google Drive if the doc is maximum-sensitivity.
   Upload the single file for that session only.
3. Install a quantized open-weight instruct model that fits the
   allocated GPU (e.g. an 8B–14B Q4 GGUF via `llama-cpp-python`, or
   a 7B–8B HF model in 4-bit). Exact ID is chosen the day you run
   it — GPU SKU varies.
4. System prompt: Michigan family-law analyst; no filings; quote
   only from the uploaded file; do not store chat.
5. Run the analysis. Copy the output to an encrypted local note.
   Do not paste into Claude/ChatGPT consumer.
6. Runtime → Disconnect and delete runtime. File → locate the
   notebook in Drive (if any) and delete. Empty trash.
7. Record in `legal_audit` by hand: date, model name, file hash,
   that Colab was used.

## Implemented (2026-08-18)

Policy lives in Legal OS code, not Portkey-dashboard-only:

- `domain/provider_grid.py` — static CAT6 rows +
  `eligible_confidential_models()` + `confidential_blocked_reason()`
- `gateway.invoke_chat(..., confidential=False)` — Confidential Mode
  allows only the verified list; otherwise `GatewayResult(ok=False,
  text="confidential_blocked")`. No silent fallback to consumer
  Claude/ChatGPT. Existing `/v1/chat/completions` call only. 2s
  timeout fail-closed.
- `GET /v1/providers` — grid. PACER-free.
  `local_ollama_as_trust_posture=false`.
- `POST /v1/gateway:invoke` — `{prompt, model, confidential}`. 409
  `invoke-disabled` unless `LEGAL_WORKSPACE_INVOKE_MODELS=true`. 409
  `confidential_blocked` for ineligible models.
- PRIV renders the grid (Confidential-eligible column) and labels
  it “not a privilege legal conclusion.”
- Shell CONF toggle still writes `lw-confidential`; PRIV/CHAT read
  it. Chrome routing remains PARTIAL: the toggle itself does not POST.

## Non-goals

No local/self-hosted inference. No automated Colab. No silent
fallback to unverified providers.

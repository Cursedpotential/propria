# HANDOFF — Legal OS Category 6: Privilege, Privacy & LLM Routing — Implementation Research

Status: rough feature set agreed with owner (2026-08-17); this handoff requests in-depth research to scaffold the actual implementation. Research only — no code, no routing rules activated, no provider signups performed on the owner's behalf.

## Context

Legal OS is a single-user, single-case, self-represented-litigant practice-management app for a Michigan family-law (custody) matter, sibling package to "Agno MCP Platform" in one monorepo. Owner has **no local/self-hosted inference capability** — this rules out the "self-hosted Ollama = safest option" pattern legal-mcp's own docs recommend, and requires a different privacy model built entirely on cloud providers with verified favorable data-handling terms, reusing Agno's existing Portkey gateway (Ollama Cloud primary, NVIDIA embed/rerank/backup — already running in Agno's stack).

## Feature set agreed (do not re-litigate — build research around this)

1. **No local-inference routing.** Do not design, research, or scaffold a "route to local Ollama" path — not viable given the owner's infrastructure. This is a deliberate scope change from a typical "Confidential Mode" implementation.
2. **Confidential Mode redefined as mandatory redaction + verified-provider routing, not local fallback:**
   - When active, content is run through a PII/privilege pre-flight check (reusing Category 5's Presidio-based redaction pipeline) **before** being sent to any provider, regardless of which provider.
   - Routing preference goes to whichever **verified low-data-retention provider** is configured, via the same Portkey gateway Agno already runs (reuse, don't stand up a second gateway — this was already decided in Category 3).
   - If no verified-acceptable provider is configured or the content's risk level exceeds what's available, **hard-block** — stop and tell the owner, don't silently fall back to a risky default.
3. **Candidate verified-tier providers to confirm (not assume) — this is the primary research task of this handoff:**
   - **Ollama Cloud** — already Agno's primary gateway target; owner believes some of its models don't collect/train on data but wants this confirmed, not assumed
   - **NVIDIA NIM** (hosted models, already Agno's backup) — data policy currently unclear even to the owner; needs research
   - **Venice.ai** — privacy-focused platform, not yet part of the stack, owner is open to signing up if its actual terms hold up to scrutiny
   - **OpenRouter with ZDR routing** — `"provider": {"zdr": true}` + `data_collection: "deny"` per OpenRouter's own documented feature (already referenced in legal-mcp's privacy docs); confirm current mechanics and which specific models support it
4. **Manual "break glass" procedure for maximum-sensitivity documents:** Google Colab Pro as an ephemeral, on-demand GPU session to run an open-weight model directly for a single highly-sensitive document, then tear the session down. **This is a documented runbook, not an automated integration** — Colab is an interactive notebook environment, not something Legal OS should call as a stable API. Research should produce the runbook content (what model, what setup steps), not code.
5. **Privilege/sensitive-content detection: hybrid, not pure keyword-matching.** Cheap rule-based first pass (privilege markers: "attorney-client," "work product," litigation-strategy language) escalating to a lightweight LLM classification call only when the first pass is ambiguous — avoid legal-mcp's toy keyword-only approach, but also avoid an LLM call on every single check given cost-consciousness.
6. **Provider trust grid must reflect real, current terms**, not the six-provider list legal-terminal ships with with assumed properties. Rebuild it around what the owner can actually access: Ollama Cloud, NVIDIA NIM, Venice.ai (if adopted), OpenRouter-ZDR, plus whatever consumer-tier subscriptions the owner already has (Claude, GPT) — clearly marked as **not** ZDR-equivalent unless proven otherwise, since consumer apps typically don't carry the same terms as API/enterprise tiers.
7. **Legal grounding stays:** *Heppner* / ABA Model Rule 1.6 framing carries over from the original catalog.

## Research questions to answer (with citations/links)

1. **Ollama Cloud data-retention/training policy** — find and cite Ollama's actual published terms for its cloud-hosted models (not the local/open-source project's general reputation). Distinguish between "the model weights are open" and "the hosted service doesn't log/train on your inputs" — these are different claims.
2. **NVIDIA NIM hosted-model data policy** — same rigor: find NVIDIA's actual terms for NIM-hosted inference, cite the source.
3. **Venice.ai's actual privacy terms** — read their terms of service/privacy policy directly rather than relying on marketing claims; confirm what "private" specifically means there (no logging? no training? both? for how long?).
4. **OpenRouter ZDR routing mechanics** — confirm current implementation (`zdr: true` provider flag, `data_collection: "deny"`), which underlying model providers actually honor it, and any gaps (e.g., does ZDR routing silently fall back to a non-ZDR provider if the preferred one is unavailable — this would be a dangerous silent failure mode worth flagging if true).
5. **Portkey gateway routing-rule capability.** Confirm Portkey can express "prefer verified-tier provider A, else B, else hard-fail" logic natively, or whether that policy needs to live in Legal OS's own code calling Portkey. Reference Agno's existing Portkey configuration as the starting point, don't design from zero.
6. **Hybrid privilege-classifier implementation.** Research concrete approaches for the rule-based-first/LLM-escalation-second pattern — e.g., a keyword/regex first pass plus a small, cheap model (not a frontier model) for the ambiguous-case escalation. Recommend a specific cheap model suitable for this narrow classification task.
7. **Colab Pro runbook content.** Research what open-weight model realistically runs well on Colab Pro's typical GPU allocation (commonly T4 or A100, availability varies) for legal-document analysis quality — e.g. a quantized Llama 3.x or Mistral variant — and produce the actual step-by-step runbook (setup, model load, teardown) as part of the deliverable, not just a model recommendation.

## Deliverable

A single markdown report, saved to the workspace, with:
- A verified (cited, not assumed) data-policy summary for each of the four candidate providers (questions 1-4), with a clear verdict on which qualify for "Confidential Mode eligible" status
- A rebuilt provider trust grid table reflecting only what's actually true for the owner's real access
- A Portkey routing-rule recommendation (question 5)
- A concrete hybrid privilege-classifier recommendation with a specific cheap-model suggestion (question 6)
- A complete, actionable Colab Pro runbook (question 7) as an appendix — this is meant to be followed by hand, not automated
- Explicit non-goals restated: no local/self-hosted inference, no automated Colab integration, no silent fallback to unverified providers

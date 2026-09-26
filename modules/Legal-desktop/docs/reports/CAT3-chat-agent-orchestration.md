# Category 3 — Chat / AI agent orchestration

> _Byline: Grok · grok-4.6 · 2026-08-18_

## Recommendations

1. **Surface:** F1 / `CHAT` pins a split pane with the **current**
   Matter, open draft, factor, or exhibit injected. Not a landing
   page. Shape borrowed from legal-terminal-master ChatPanel +
   legal-terminal-main `@` file mention (idea only).

2. **Runtime:** Agno adapter behind a neutral `AgentRun` contract
   (build guide). Public HTTP stays FastAPI. Do not make the browser
   talk to AgentOS objects.

3. **Router + specialists** (narrow, from the guide + custody-guide
   plugin agents):
   - intake / issue spotter
   - Michigan research
   - evidence-to-element mapper
   - drafting
   - citation verifier
   - adversarial reviewer
   - discovery
   - filing-readiness

   ~~Addressable via mnemonic (`DRFT`, `CITE`)~~ **corrected 2026-08-18:** addressable by English page name. ~~`Write a paper`~~ **corrected again 2026-08-18:** catalog label is `Brief Builder`. Same words in API and UI.

4. **Grounding:** `LegalSourcePackage` + custody-guide primaries.
   Optional later Surreal read. Never raw Agno candidates.

5. **Actions:** HITL confirm for anything that queues analysis,
   mutates a draft, or touches release state. Agents cannot approve,
   file, or send (AC-AUTOMATION-001).

6. **Deep research:** sequential tool loop over Cat 4 search
   functions. Record every hop on `AgentRun` / `ModelInvocation`
   (ADR-0054 style).

7. **Personas:** seed from
   `custodyguide/.../toolkit-package/plugin/agents/`
   (case-law-researcher, evidence-organizer,
   family-court-document-drafter, michigan-source-verifier).
   Rewrite so they cite this packet’s guardrails.

## Action-confirmation tiers (question 7)

| Action | Tier | Why |
|---|---|---|
| Search PREC/STAT, explain a statute | Immediate | Read-only |
| Queue work-product analysis (JOBS) | Light confirm | Reversible, costs tokens |
| Insert a suggested paragraph into a **private_draft** | Light confirm | Fork-safe |
| Save / overwrite a draft version | Confirm | New version row; still not released |
| Mark a packet item reviewed / change epistemic class | Confirm | Crosses into fact-adjacent state; Agno owns establishment |
| Run a WKFL playbook that only calls Legal OS tools | Confirm | Multi-step side effects |
| Trigger n8n notify | Confirm | Leaves the box |
| Approve release, file, serve, email a party, sign | **Denied** | AC-AUTOMATION-001 |

Pattern: Agno `@approval` / `requires_confirmation=True`
(`apply_db_modification`). Same idea, Legal OS HTTP.

## Deep-research loop (question 8)

Bounded ReAct: decompose → call Cat 4 search functions → evaluate →
refine. Stop at 6 hops or a token budget. Every hop recorded on
`AgentRun`. Not a Cat 4 feature.

## Personas (question 6)

Seed from custody-guide plugin agents + `genesee-family-court-toolkit`
substance. File shape like Agno `server/agents/instructions.py`.
Greenfield — Agno `knowledge/legal/` has only forensic rubrics.

## Module tree

```
api/legal_workspace/agents/
  router.py
  specialists/
  instructions/
  context.py          # Zustand shell snapshot → structured turn context
  approval.py
```

## Non-goals

No new model gateway (reuse Portkey). No Agno-framework migration.
No unconfirmed high-stakes actions. No silent auto-execution.
No claiming attorney-client privilege for AI chat.

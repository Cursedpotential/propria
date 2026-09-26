# Agno MCP Platform — Interface & Integration Analysis for Legal OS

> _Naming (D-138, 2026-09-05; applied 2026-09-06): this product is **advocatio** (formerly Legal-Workspace / Legal Workspace); the evidence platform it consumes is **Indicia Probata** / `probata` (formerly Agno-MCP-Platform). Directory: `probata/modules/advocatio/` (old name kept as a junction). GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._


Analyzed directly from source: [github.com/Cursedpotential/mcp-platform-agno-mvp](https://github.com/Cursedpotential/mcp-platform-agno-mvp), cloned at commit head as of 2026-08-17 (repo updated same day). ~2,400 files. This is a live, actively-developed platform, not a stub — treat everything below as current state, not aspiration, except where marked "not yet built."

**Key decision from this conversation, recorded here:** the "AI Legal Team" (Part 3 of the platform's own roadmap) moves to Legal OS instead of being built inside the Agno Platform. This changes the ownership boundary from what the Agno Platform's own docs currently assume — see the "Ownership Boundary" section and "Open Items" below for what needs to be reconciled.

---

## What the Agno Platform actually is

Per its own [README](https://github.com/Cursedpotential/mcp-platform-agno-mvp/blob/main/README.md) and `docs/PROJECT_CANON.md` (the project's own "durable source of truth"): a **personal, pro se family-law evidence-processing + analysis + legal-strategy platform**, built around a three-part arc:

1. **Evidence** — custody (SHA-256 + manifest) → parse → normalize → store → court-ready export, over a polyglot tool mesh.
2. **Analysis** — multi-pass psychological/abuse-pattern analysis over a **bitemporal** graph; the delta between what was known contemporaneously vs. what's known with full disclosure ("hindsight") is the core product mechanism.
3. **AI Legal Team** *(originally scoped here, now moving to Legal OS per this conversation)* — agents using evidence + knowledge base to produce strategy, motions, filings, discovery. Was to be "ported from the owner's Gemini Gems personas" — **confirmed not yet built**: no persona files, no motion/filing/strategy code exists anywhere in the repo today. Only two knowledge files exist under `knowledge/legal/` and both are forensic coercive-control classification rubrics, not legal-strategy content.

**Current infrastructure (real, running):** PostgreSQL 18 (pg_duckdb + pgvector + PostGIS, dual evidence/analysis schema) · Neo4j + Graphiti (bitemporal graph) · Portkey model gateway (Ollama Cloud primary, NVIDIA embed/rerank/backup) · Weaviate vectors · OpenCode · Cloudflare R2 blob storage · isolated agent sandbox · Kasm desktop · n8n. Root `compose.yaml` is explicitly "mirrored to the VPS and is production-facing, not a disposable local-only stack."

**Runtime/UI direction (their own stated target, from README):** moving away from the Agno/AgentOS UI clone toward a **custom Next.js/FastAPI "Workbench"** as the primary product surface — same direction as Legal OS conceptually, just for a different job (evidence ops vs. legal practice).

---

## Ownership Boundary (updated for the AI Legal Team decision)

| Owns | Agno Platform | Legal OS |
|---|---|---|
| Evidence custody (SHA-256, manifest, chain of custody) | ✅ exclusive | — |
| Parsing/ingestion (chat exports, documents → normalized records) | ✅ exclusive | — |
| Bitemporal/knowledge-horizon analysis (contemporaneous vs. hindsight) | ✅ exclusive | — |
| Coercive-control / abuse-pattern forensic analysis | ✅ exclusive | — |
| Matter / CourtCase / EvidenceItem data model + review workflow | ✅ exclusive (real API + DB today) | Legal OS **consumes**, does not duplicate |
| Postgres/Weaviate/Neo4j knowledge base | ✅ exclusive (data plane) | Legal OS **reads via API**, does not stand up its own copy |
| Ops agents (ingestion_orchestrator, analysis_orchestrator, review_gatekeeper, transcript_miner, forensic_data_agent, dev_copilot, project_pal) | ✅ exclusive | — |
| Audit ledger (`ops.audit_ledger`) | ✅ exclusive | Legal OS's Audit Log panel should **read from this**, not maintain a second ledger |
| Ops Copilot chat (`/copilot` — "ask about a run, a staged file") | ✅ exclusive, platform-ops scope | — (different purpose than Legal OS's Paralegal chat, see below) |
| **AI Legal Team** — strategy, motions, filings, discovery, Michigan legal skills | ~~Agno~~ → **Legal OS, per this session's decision** | ✅ exclusive as of now |
| Precedent/case-law/statute research tools | not present | ✅ Legal OS |
| Citation validation/normalization | not present | ✅ Legal OS |
| Contract/agreement clause analysis, negotiation guidance | not present | ✅ Legal OS |
| Document risk analysis (DOCA-style) | partially adjacent (their "classification-test" page does document classification, but for forensic/evidence classification, not contract/clause risk) | ✅ Legal OS, distinct purpose |
| Privilege-risk / AI-provider trust checking (consumer-facing) | not present | ✅ Legal OS |
| Bloomberg-style command/mnemonic UI shell | not present (their Workbench is a conventional Next.js dashboard, not a terminal-style UI) | ✅ Legal OS |
| Brief/motion drafting UI | not present | ✅ Legal OS (this is now literally Part 3) |

---

## The Chat Surface Question (you asked about this specifically)

Two chat surfaces exist or will exist, and they are **not the same thing** — no overlap to resolve, but a component worth sharing:

- **Agno's Ops Copilot** (`workbench/web/src/app/copilot/page.tsx`) — a thin chat pane over the platform's headless OpenCode server, scoped explicitly to "a run, a staged file" — i.e., pipeline/ops questions ("why did this ingestion fail," "what's in this staged file"). Not reachable from the nav yet (URL-only). Not a legal assistant.
- **Legal OS's Paralegal** (the CHAT panel from the earlier feature catalog) — legal research/strategy assistant: case law, citations, contracts, privilege, briefs. This is genuinely new and belongs entirely in Legal OS.

**Where they should actually connect:** once Legal OS's Paralegal needs to reference *your specific evidence* (e.g., "what does the record show about the March incident"), it should call into Agno's knowledge/evidence API (`/v1/knowledge/items`, `/v1/matters/{id}/evidence-items`) as a tool, the same way legal-mcp tools are called today — not reimplement evidence retrieval. The two chat UIs can share a common `<ChatPanel>` component in a shared monorepo package (composer, message list, attachment-card rendering, thinking indicator) since the interaction shape is identical even though the backends and content differ.

---

## Concrete Integration Points (API surface Legal OS should call, not rebuild)

All confirmed live and mounted in `server/api/main.py` today:

| Endpoint | Purpose | Legal OS use |
|---|---|---|
| `GET /v1/matters`, `POST /v1/matters`, `GET /v1/matters/{id}` | Matter (case) CRUD | Legal OS's "your case" context should be *this* Matter record, not a new one |
| `POST /v1/matters/{id}/court-cases` | Attach a court case number to a matter | Same case-number concept Legal OS's Docket Watch/Case Monitor would key off of |
| `POST /v1/matters/{id}/knowledge/resolve` | Resolve a knowledge source into the matter | Feed point for anything Legal OS analyzes that should join the shared knowledge base |
| `GET/POST /v1/matters/{id}/evidence-items`, `GET /v1/matters/{id}/evidence-items/{item_id}` | Evidence item list/detail | Legal OS's DOCA/document tools should pull real evidence from here instead of a separate upload store, when the document is case evidence rather than a one-off draft |
| `GET /v1/matters/{id}/evidence-items/{item_id}/court-readiness` | Court-readiness check on an evidence item | Directly useful for a "ready to file" indicator in Legal OS's brief/motion flow |
| `POST/GET /v1/matters/{id}/evidence-items/{item_id}/reviews` | Human review workflow on evidence | Legal OS's own review/approval steps (e.g., before using a document in a motion) could reuse this same review primitive instead of inventing a parallel one |
| `GET /v1/knowledge/items`, `GET /v1/knowledge/items/{artifact_id}` | Browse the shared knowledge base | This is the real backing for the "connection to your knowledge bases" status pill from the earlier matrix revision |
| `GET /v1/records`, `GET /v1/inspect/schemas`, `GET /v1/inspect/tables/{schema}/{table}`, `GET /v1/inspect/weaviate/{collection}` | Low-level data inspection | Dev/diagnostics only — not end-user-facing in Legal OS |
| `GET /v1/runs`, `GET /v1/runs/{id}`, `GET /v1/runs/{id}/report` | Ingestion/analysis pipeline run status | Legal OS's Home status pill / diagnostics view can surface "is the evidence pipeline healthy" from here |
| `POST /v1/verify/{sha256}` | Custody/integrity verification | If Legal OS ever needs to confirm a document's evidentiary integrity before using it in a filing, call this rather than reimplementing hashing/verification |

**What this means structurally:** Legal OS should treat the Agno Platform the way legal-terminal treats legal-mcp — as a backend it calls over an API — except the "tools" here are case/evidence/knowledge operations instead of research tools, and the API is already REST/FastAPI rather than MCP tool calls. Confirm during scaffolding whether Legal OS should also expose its own operations (precedent search, citation validation, brief drafting) as MCP tools so Agno's future agents could someday call *into* Legal OS too — that's a real two-way door worth designing for now rather than retrofitting later.

---

## SurrealDB Analytical Surface — In-Flight, Not Yet Production (added 2026-08-17)

While discussing Legal OS's own persistence layer, you proposed SurrealDB as "the backend" because it aggregates all the specialized stores and is "where it gets promoted from claims to actual evidence." That description is accurate to real, active work in this repo — worth being precise about scope and maturity before Legal OS designs against it.

**What's actually true, per `docs/SURREAL-INVESTIGATION-BLUEPRINT-2026-08-15.md` and the live test run in `docs/HANDOFF-2026-08-17-R14-phase1-surreal-live-core-pass.md` (dated today):**

- **PostgreSQL remains authoritative — this is a locked governing boundary, not up for debate.** SurrealDB is explicitly scoped as "a governed analytical projection and experimental walk-memory runtime," not a replacement backend. A prior attempt to use SurrealDB as the operational store/session/memory layer (ADR-0024, June 2026) was fully retired (ADR-0043, Aug 2 2026) — "SurrealDB exits the critical path," zero callers in production code. What's being investigated now (ADR-0056) is a *different, narrower* role.
- **The pipeline your framing described is real:** raw sources → custody/normalization → Postgres (authority) → outbox/CDC → Weaviate (vectors) + Neo4j (operational graph) → Semantica (candidate-claim extraction) → human-governed review/promotion → **SurrealDB** (curated, promoted analysis + "as-lived" walk-memory). A "claim candidate" only becomes an "established fact" after governed review, and only established facts + their evidence subgraphs get promoted into SurrealDB. This is genuinely the "claims → evidence" promotion mechanism you described.
- **Current maturity: Phase 1, core-gate pass only, as of literally today.** The live test (`SurrealDB 3.2.3`, disposable/isolated instance) passed its core horizon-safety gates (no future-fact leakage, forbidden writes correctly denied, quarantine correctly excludes results) but explicitly has **not** passed the full Phase-1 gate set (sealed-snapshot, linked-rewalk, export/import parity still pending) and the blueprint states plainly: "no target, schema, activation, corpus copy, deployment, or agent binding follows" yet. Nothing is running against real case data. Multiple "R9 holds" remain active blocking production use.

**What this means for Legal OS:**

1. **Don't build a competing version of this.** The claim-review-promotion pipeline and the SurrealDB analytical surface are squarely Agno's Analysis pillar (Part 2 of its own three-part arc) — already actively engineered, tested today. This reinforces the existing ownership boundary rather than changing it.
2. **Owner decision (2026-08-17): SurrealDB is the primary data source the AI Legal Team consumes from, not just "a" source.** The owner's read is that this surface is closer to beta than experimental (my read from the docs alone was more cautious — core-gate pass only, holds still active as of today's handoff — both things can be true: architecturally settled/beta-quality code, not yet production-activated). Net effect either way: build the AI Legal Team's evidence-retrieval interface as a first-class SurrealDB client from the start, not a bolted-on option. Point it at Agno's REST API in the interim if SurrealDB isn't reachable yet, but design the primary path around SurrealDB's promoted-facts/evidence-subgraph query shape, not Postgres's raw tables.
3. **State management confirmed:** Legal OS's own plain app state (settings, workflow definitions, drafts, UI preferences) stays on the shared Postgres `legal_os` schema, per owner agreement — SurrealDB's governed/immutable-facts ceremony is the right fit for evidentiary data, not for "what theme did I pick."
4. **Open question, deliberately deferred (owner call — do not spend further research credits on this without instruction):** should Legal OS's own **document parking** (uploaded case documents, generated drafts/briefs) live in SurrealDB (native document data type, same surface as the facts they'll be cited from) or in Postgres (pg_duckdb already provisioned there, and Postgres already holds custody-bound original bytes per the blueprint's "object storage retains original bytes and custody bindings" boundary)? Leaning consideration for whichever agent picks this up: the blueprint already assigns custody-bound originals to Postgres/object storage as a locked boundary (§1 of the Surreal blueprint) — SurrealDB's role is curated/promoted *analysis*, not custody of originals — so parking raw documents in Postgres/R2 and only promoted extracts/citations in SurrealDB may already be dictated by that existing boundary rather than being a free choice. Flag this for the next agent to confirm against the blueprint before deciding, rather than re-deriving it from scratch.
5. **Watch, don't wait.** This is moving fast (three handoffs in the last three days). Worth checking `docs/HANDOFFS.md` / the latest `HANDOFF-*surreal*` doc before locking the evidence-retrieval interface, rather than assuming today's snapshot is final.

---

## Monorepo Structure Implication

Given the ownership boundary above, a workspace layout like this keeps the seam clean:

```
the-platform-workspace/          (or whatever the monorepo root becomes)
├── apps/
│   ├── agno-platform/           ← this repo, evidence/analysis/knowledge backend
│   │   ├── server/               (unchanged: evidence, analysis, case_management, ingest, api)
│   │   └── workbench/            (unchanged: evidence-ops Next.js UI)
│   └── legal-os/                 ← new
│       ├── terminal/             (Bloomberg-shell web + TUI, from legal-terminal shape)
│       ├── tools/                 (legal-mcp-shape: research/citation/contract/brief/privilege tools)
│       └── ai-legal-team/         ← Part 3 lands HERE now, not in agno-platform
│           (strategy, motion drafting, discovery — reads Agno's evidence+knowledge via its API)
├── packages/
│   ├── shared-ui/                 (shared <ChatPanel>, design tokens, PanelChrome-equivalent)
│   ├── shared-contracts/          (typed client for Agno's /v1/* API, shared by both apps)
│   └── shared-audit/              (if Legal OS reads Agno's audit ledger directly)
└── docs/
    └── PROJECT_CANON.md           (needs the Part-3-moved-to-legal-os update — see Open Items)
```

---

## Open Items — need your decision, not assumed

1. **Agno's own `docs/PROJECT_CANON.md` currently states Part 3 lives in this repo** ("AI Legal Team... to build — Part 3," §ownership language throughout). Since you're moving it to Legal OS, that canon doc needs a correction pass once the monorepo is real — otherwise the next person (or agent) reading Agno's canon will assume Part 3 is still owed there and duplicate it. I did not edit that file — say the word and I will, or leave it for whichever agent does the actual monorepo migration.
2. **API vs. shared-package integration:** should Legal OS call Agno's `/v1/*` endpoints over HTTP (loose coupling, works even if they're deployed separately) or import Agno's Python modules directly in-process (tighter coupling, only works if they truly live in one deployable unit)? The compose.yaml evidence suggests Agno is deployed as its own service today — I'd default to HTTP unless you tell me otherwise.
3. **Should Legal OS's research/citation/contract tools also be exposed as MCP tools that Agno's *own* agents (ingestion_orchestrator, etc.) could call?** Nothing in Agno's current agent roster needs this today, but if a future Agno agent ever needs "check privilege risk before this gets embedded" or "validate this citation," that's a real cross-call, not just Legal-OS-consumes-Agno one-way.
4. **Michigan legal skills** — canon mentions "the Michigan legal skills" as an input to Part 3 but no such skill files exist yet anywhere in either repo. This needs to be authored as part of building the AI Legal Team in Legal OS — worth checking whether your `genesee-family-court-toolkit` skill (already in your Perplexity setup) is meant to be the seed for this, since it already covers Genesee County / 7th Circuit Family Division procedure.

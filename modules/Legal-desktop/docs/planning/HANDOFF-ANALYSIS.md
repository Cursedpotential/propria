# Handoff analysis

> _Naming (D-138, 2026-09-05; applied 2026-09-06): this product is **advocatio** (formerly Legal-Workspace / Legal Workspace); the evidence platform it consumes is **Indicia Probata** / `probata` (formerly Agno-MCP-Platform). Directory: `probata/modules/advocatio/` (old name kept as a junction). GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._


> _Byline: Grok · grok-4.6 · 2026-08-18_

## Slice context

Canonical copies reviewed in full on 2026-08-18 from
`Legal-desktop/artifacts (3)/` (all 11 files). Duplicated under
`docs/planning/artifacts/` for the Legal-Workspace tree.

The zip handoffs are **research briefs**, not an implementation spec.
The approved product contract is
`docs/LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md`.
Where they disagree, the build guide wins.

The custody packet (`Legal-desktop/donors/custodyguide_v1complete_20260812`)
is **this case**. It is not a later “content pack” and it is not a
multi-client library.

## What the seven handoffs asked

| Cat | Title | Agreed feature set | Open research | Locked here |
|---|---|---|---|---|
| 1 | Persistence & Settings | Same PG18 cluster, `legal_*` schemas, numbered SQL, pydantic-settings | Surreal vs R2 for document bytes; Surreal SDK | Bytes in R2; metadata in Postgres; Surreal is read-only projection when Agno activates it |
| 2 | Command bar & shell | Mnemonics, split view, dark-only, Ctrl+K, Confidential Mode chrome | Vite vs Next | **Next.js 16.3.x** (build guide). Vite donor is a shape source only |
| 3 | Chat / agents | F1 context-aware summon, router + addressable specialists, HITL on writes | Action-tiering, deep-research loop | No generic chat home. First AI is inside a scoped task |
| 4 | Research tools | Pluggable sources, CourtListener default, eyecite | Midpage, hybrid search | eyecite for structure; CourtListener is **not** a citator |
| 5 | Documents | Collabora vs OnlyOffice vs TipTap; real redaction; Bates; MI forms | Renderer choice | Markdown-first drafts; LibreOffice sidecar for DOCX/PDF; pikepdf for redaction |
| 6 | Privilege / routing | No local Ollama; redaction + verified Portkey route; hard-block | Provider terms | Rebuild trust grid from live terms; legal-mcp’s `PROVIDER_POSTURES` is stale (it treats `ollama` as local-only) |
| 7 | Workflows | Own sequential runner + real server scheduler; n8n for external | APScheduler vs BullMQ | APScheduler in `legal-api`; n8n for email/push only |

## Feature-matrix vs later handoffs

The Decision Matrix (74 Keep / 28 Rescope / 10 Exclude) is older than
the category handoffs. Later owner calls win:

| Matrix | Later handoff | Winner |
|---|---|---|
| Keep light/dark toggle | Cat 2: dark-only, no toggle | Dark-only |
| Rescope “repeat last” | Cat 2: explicitly cut | Cut |
| Rescope hardware Help keys | Cat 2: do not build | Cut |
| PRIV 6-provider list incl. local Ollama | Cat 6: no local inference; rebuild grid | Cat 6 |
| AUTM “real scheduler” | Cat 7: server-side APScheduler, not tab timer | Cat 7 |

Deep analysis licensing: **rewrite** legal-terminal (proprietary);
**fork logic** from legal-mcp (AGPL, single-user).

Agno interface analysis: AI Legal Team lives here; consume `/v1/matters`,
`/v1/verify/{sha256}`, knowledge items; do not clone evidence APIs.
HTTP coupling, not in-process imports.

## Conflicts resolved

1. **Name:** Legal-Workspace (guide) vs Legal OS (zip). Product name:
   Legal Workspace. Internal schema prefix may stay `legal_*`.
2. **First screen:** Guide says Matter Home. Zip keeps CHAT as a module.
   Matter Home is first. `CHAT`/`F1` is a pinned split, not the landing page.
3. **Surreal as “primary data source” (Cat 1 §5):** Agno’s own blueprint
   keeps original bytes and custody in Postgres/R2. Legal Workspace
   grounds facts in `LegalSourcePackage`, with an optional Surreal
   read adapter later. Do not write legal drafts into Surreal.
4. **legal-mcp citation tools:** They validate **structure** and say so.
   Do not port `check_demo_database` as authority.
5. **PACER:** Fee-risk is real. Default off. CourtListener/RECAP only
   unless the owner opts in.

## Custody packet as the case core

Port into Legal Workspace as first-class substance, not inspiration:

- `GUARDRAILS.md` — dispositions VERIFIED / PROVISIONAL / CONFLICTED /
  UNVERIFIED; no diagnostic labels as pleaded facts; no filing-ready
  templates; Genesee local facts are clerk-confirmed or marked
  PROVISIONAL.
- `draft/M2-standards-that-decide-your-case.md` + MCL 722.23 workspace.
- `toolkit-reference/scripts/deadline_calculator.py` — candidate
  arithmetic only, clerk confirm required (matches AC-DEADLINE-001).
- Templates under `toolkit-reference/templates/` for motion, affidavit,
  proposed order, discovery, proof of service — always with
  “do not use if”.
- Archived primaries in `sources/primary/` as the first authority
  snapshot corpus (MCL, MCR, MRE, Genesee LCR, support formula).

## Sequential-react-ship (process donor)

Used for this build cycle:

- Type 2: local scaffold, docs, tests.
- Type 1 hold: Coolify writes, applying `0001` to the live Agno PG
  cluster, any PACER enablement.
- Maps (handoffs) are not territory (running API + pytest).

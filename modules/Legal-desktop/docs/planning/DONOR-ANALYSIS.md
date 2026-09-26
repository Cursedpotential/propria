# Donor analysis

> _Naming (D-138, 2026-09-05; applied 2026-09-06): this product is **advocatio** (formerly Legal-Workspace / Legal Workspace); the evidence platform it consumes is **Indicia Probata** / `probata` (formerly Agno-MCP-Platform). Directory: `probata/modules/advocatio/` (old name kept as a junction). GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._


> _Byline: Grok · grok-4.6 · 2026-08-18_

Primary donors (owner-named):

1. `sequential-react-ship` — process
2. `custodyguide_v1complete_20260812` — **this case**
3. `legal-mcp-main` — tool shapes
4. `legal-terminal-master` — Bloomberg shell

Other archives were scanned for shape only. Foreign-jurisdiction
substance is never copied into court-facing templates.

---

## 1. sequential-react-ship

Path: `Legal-desktop/donors/sequential-react-ship/`

**Keep:** Type 1 vs Type 2 doors; pre-mortem before first Act; map ≠
territory; `BUILD_STATUS` only after named checks run.

**Ignore:** Horizon-specific failure seeds (SBV, Chonkie, Surreal pane)
except as reminders that Agno’s parked systems stay parked.

---

## 2. custodyguide — the case packet

Path: `Legal-desktop/donors/custodyguide_v1complete_20260812/Projects/custody-guide/`

This is not a generic Michigan treatise to later “instantiate.”
The Legal Workspace **is** the operating system for this packet:

- 7th Circuit, Family Division, Genesee County
- Owner as the only user
- Modules M1–M16, P1 packets, factor analysis, FOC/referee, discovery

**Port as product substance**

| Piece | Use |
|---|---|
| `GUARDRAILS.md` | Binding epistemic rules for every draft |
| `draft/FM-*`, `M1`, `M2`, `M4`–`M16` | Issue trees + factor workspace copy |
| `toolkit-reference/templates/` | Drafting Studio starters |
| `toolkit-reference/scripts/deadline_calculator.py` | Deadline engine (candidate + confirm) |
| `toolkit-reference/scripts/exhibit_indexer.py`, `redaction_helper.py` | Exhibit + redaction helpers |
| `sources/primary/` | First `AuthoritySnapshot` corpus |
| `toolkit-package/plugin/agents/` | Persona seeds for the AI legal team |
| `toolkit-reference/checklists/` | Filing-readiness gates |

**Do not port**

- Publication-to-strangers attorney-review gate as a blocker on the
  owner’s own drafts (guardrails already split this).
- Diagnostic labels as pleaded facts.
- MiCOURT Case Search (assessed, not activated, needs secrets).

---

## 3. legal-mcp-main — toolset

Path: `Legal-desktop/donors/legal-mcp-main/legal-mcp-main/`

FastMCP server, 27 tools, Python 3.10+, demo data **off** by default.

**Port as adapters (FastAPI services, not a second MCP runtime)**

- Tool *names and job shapes*: `search_precedents`, `extract_statute`,
  `validate_citation` (structure only), `analyze_document`,
  `queue_document_analysis`, `check_privilege_risk`,
  `generate_brief_outline`.
- `utils.audit` append-only invocation log → `legal_audit`.
- CourtListener/RECAP integration as an opt-in source adapter.
- Feature flags that disable PACER unless explicitly enabled.
- Privacy diagram: inference provider is the high-risk boundary.

**Drop or rewrite**

- Bundled `data/cases.json` as if it were Michigan authority.
- `CitationParser` regex instead of eyecite.
- `PROVIDER_POSTURES["ollama"]` = “fully local” — **wrong for this
  stack**. Owner uses Ollama Cloud via Portkey. Rebuild the grid.
- `US v. Heppner` framing can stay as a research note, not a
  Michigan holding.
- Demo mode in production.

**Grounding fix:** every research hit is a *lead*. Holdings come from
an archived opinion or `sources/primary/`. CourtListener is not a
citator (custody-guide §3.1).

---

## 4. legal-terminal-master — UI shell

Path: `Legal-desktop/donors/legal-terminal-master/legal-terminal-master/`

React + Vite + Zustand webterm + Textual TUI. Showcase license —
**rewrite, do not vendor**.

**Borrow the shape**

- Mnemonic bar (`CommandLine.tsx`): `CMD + rest` parse, Tab complete,
  prefix or description match. Mnemonics to keep, remapped:

  | Donor | Legal Workspace |
  |---|---|
  | HOME | HOME (Matter command center) |
  | CHAT | F1 split, not landing |
  | PREC / STAT / CITE | PREC / STAT / CITE |
  | CTRX | later (agreements / FOC / parenting-plan clauses) |
  | DOCA | work-product analyzer, not raw evidence |
  | PRIV / CONF | PRIV / CONF |
  | BRF | DRFT / BRF |
  | JOBS | JOBS |
  | WKFL / AUTM | Phase 2 |
  | AUDT | AUDT (reads Agno ledger + legal_audit) |
  | WTCH | rescope to **this** docket only |
  | — | FCTR (MCL 722.23), DEAD, ISSUE |

- Ctrl+K via `cmdk` (Recent + Modules).
- Dark graphite/slate, Playfair / Inter / IBM Plex Mono.
- Confidential Mode as a **visible posture**, routing in Cat 6.
- Split/pin second panel.

**Do not borrow**

- Client-side `automationScheduler.ts` (only runs while the tab is open).
- Mock/live toggle as product UX.
- TUI (explicitly deferred).
- “High-volume legal teams” copy.
- Fixture JSON as legal knowledge.

---

## Other applications (shape only)

### legal-terminal-main (JuriSupport Electron, Korea)

Different product that shares a name. Integrated Drafting Environment:
case folder, split record+draft, markdown live preview, `@` file
mentions into an agent, hearing notes, PDF viewer, CodeMirror.

**Borrow:** split “record on the left, draft on the right”; agent
summoned with the open document injected; hearing-day notes panel;
folder-is-the-case.

**Ignore:** HWP/HWPX, JuriSupport, SSH-to-court, Korean locale,
Electron shell (we are Next standalone).

### LexRAG

Python RAG API. Pattern: retrieval engine separate from chat. Useful
as a sketch for hybrid search over **this** packet + imported
package, not as a second vector store. Do not stand up another
Weaviate.

### LIGHT-2 (India civic kiosk)

LangGraph swarm, voice, BNS/IPC RAG, APScheduler for source refresh,
QR handoff then wipe.

**Borrow:** specialist micro-agents with a router; scheduled
authority-currency refresh; session scrub of model traces from
exports.

**Ignore:** Indian statutes, kiosk/voice, ChromaDB, touts/Aadhaar.

### themis (AT/DE/EU/US Claude plugin)

Skill-pack shape: one skill per instrument (brief, contract,
complaint).

**Borrow:** “one skill / one work-product kind” matching our
Drafting Studio templates.

**Ignore:** Austrian/German statutes and any US general-law templates
that are not Michigan family.

### suna (partial extract)

Large agent OS. Too heavy to copy. Note only: durable run reports +
tool-call traces — already required by Agno ADR-0054 and AC-ROUTING-001.

### OpenLegalDataSkills / claude-power-skills

Citation and caselaw skill wrappers. Use as prompt/skill text after
jurisdiction filter (US-MI only).

### Casememory M0–M3

Graphiti bitemporal memory. That is **Agno’s** lane. Do not build a
second case graph here.

---

## Possibly missing vs the build guide

Things donors do **not** cover that P0 still needs:

- `LegalSourcePackage` import + omission log (we implemented the gate).
- Independent-source-family count vs raw hit count.
- Factor (a)–(l) **both parents**, contradictions, missing proof.
- Release-candidate export manifest with hashes.
- Four-eye optional review.
- Model-egress policy per document class.
- No autonomous file/serve/email.

Those stay first-party.

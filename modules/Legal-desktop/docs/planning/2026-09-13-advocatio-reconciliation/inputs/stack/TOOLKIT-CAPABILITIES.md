# Family Court Toolkit capability inventory

Byline: Codex delegated recovery, 2026-09-13. Full staged-port planning input. All claims below are static source verification unless explicitly described as a historical receipt or current HTTP probe. No toolkit skills were executed; their files were inventoried as product capabilities.

## Outcome

The full baseline is the local Claude plugin **family-court-toolkit 3.2.0**. It is materially larger than the Codex cache **family-court-toolkit-codex 2.0.0**. Full convergence should preserve all useful agents, commands, skills, tools, reference packs and outside-resource integrations. A shared callable bridge is the transitional delivery seam; it does not replace the complete port objective.

The machine-readable companion `toolkit-capabilities.json` has a row for every discovered capability, its origin/version, declaration/implementation/wiring state, proposed incorporation disposition, verification gate and source hash where applicable. It contains source-registry metadata, not case-record contents.

## Baselines and discrepancies

| Item | Full local plugin | Codex cached adaptation |
|---|---|---|
| Plugin manifest version | 3.2.0 | 2.0.0 |
| MCP package version | 2.0.0 (version drift) | 2.0.0 |
| Server-reported source constant | 3.0.0 (version drift) | inspect during protocol parity |
| Skills | 36 files: entry plus 35 members | 3 |
| Agents | 9 prompt definitions | 0 |
| Root slash commands | 7 | 0 |
| On-demand procedures | 11 | 0 |
| MCP tools | 27 actual registrations: 11 core + 16 store | 8 |
| Built server artifact | present | present |
| Built store module | present | absent |
| External MCP | CourtListener HTTP declared | none |

`members.json` lists only 34 members; `referee-hearing-survival/SKILL.md` exists but is missing from that map. README still describes an older 31-member/18-command organization. Use observed file/register counts rather than those stale descriptions. `FL-MCP` is a junction to `projects/family-court-workbench`, not an additional product or MCP implementation.

Only the eight smaller Codex console tools are exposed in this task's native tool catalog. The parent additionally verified `mcp__family_court_console__get_checklist({kind: 'source-review'})` successfully in this session. That proves a current smaller-console call works; it does not prove the full 27-tool remote server or database migration. No toolkit operation was called against case data.

## All instruction capabilities

### Agents

| Name | Origin version | Current meaning |
|---|---|---|
| case-law-researcher | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| custody-support | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| evidence-organizer | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| evidence-tech | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| family-court-document-drafter | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| forensic | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| litigation | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| michigan-law | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| michigan-source-verifier | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |

### Root commands

| Name | Origin version | Current meaning |
|---|---|---|
| discovery | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| evidence | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| family-court | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| hearing | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| motion | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| packet | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| verify-sources | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |

### On-demand procedures

| Name | Origin version | Current meaning |
|---|---|---|
| analyze-behavior | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| case-law | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| case-lookup | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| custody-intake | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| factor-analysis | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| gal-prep | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| motion-strategy | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| research-issue | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| source-audit | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| verify | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| vulnerability-check | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |

### Skills

| Name | Origin version | Current meaning |
|---|---|---|
| behavioral-pattern-analyzer | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| best-interest-factors | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| case-intake | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| case-research | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| case-store | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| child-support-worksheet | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| court-language | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| court-resources | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| cps-resources | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| custody-evaluation-summary | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| custody-packet-builder | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| custody-packet-gen1 | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| decision-record-verification | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| documentation-methods | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| dv-resources | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| dvro-petition | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| evidence-templates | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| family-court-toolkit | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| foc-resources | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| irac-formatter | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| manipulation-patterns | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| mcl-factor-mapper | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| michigan-family-court-guide | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| motion-for-temporary-relief | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| mre-authentication | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| order-modification | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| parenting-plan | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| referee-hearing-survival | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| secondary-source-auditor | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| survival-guide | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| tax-return-analysis | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| tax-return-summary | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| toolkit-gen1 | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| track-deposits | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| trusted-sources | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| verify-michigan-legal-sources | 3.2.0 | Full baseline; instruction artifact, not standalone executable tool |
| custody-packet-builder | 2.0.0 | Codex adaptation; instruction artifact, not standalone executable tool |
| michigan-family-court-guide | 2.0.0 | Codex adaptation; instruction artifact, not standalone executable tool |
| verify-michigan-legal-sources | 2.0.0 | Codex adaptation; instruction artifact, not standalone executable tool |

All of these should have an explicit destination: agent role, workflow action, human-readable guidance, contextual help, or archived provenance when superseded. Importing only the three advertised Codex skills would omit most of the requested toolkit. Model labels and host tool names in agent frontmatter are source configuration, not authorization to recreate those model/runtime assumptions.

## Actual MCP handler inventory

| Tool | Full version | Behavior | Implementation source |
|---|---|---|
| open_dashboard | 3.2.0 | readOnly | server.ts:60 |
| route_issue | 3.2.0 | readOnly | server.ts:66 |
| calculate_planning_date | 3.2.0 | readOnly | server.ts:72 |
| get_packet_plan | 3.2.0 | readOnly | server.ts:85 |
| get_checklist | 3.2.0 | readOnly | server.ts:91 |
| audit_sources | 3.2.0 | readOnly | server.ts:97 |
| search_guide | 3.2.0 | readOnly | server.ts:103 |
| build_chronology | 3.2.0 | readOnly | server.ts:109 |
| case_facts | 3.2.0 | readOnly | server.ts:116 |
| survival_guide | 3.2.0 | readOnly | server.ts:124 |
| court_language_review | 3.2.0 | readOnly | server.ts:139 |
| case_put | 3.2.0 | write or mixed; inspect action-level contract | store-tools.ts:88 |
| case_query | 3.2.0 | readOnly | store-tools.ts:117 |
| case_search | 3.2.0 | readOnly | store-tools.ts:141 |
| case_graph | 3.2.0 | readOnly | store-tools.ts:167 |
| case_factor_map | 3.2.0 | readOnly | store-tools.ts:191 |
| case_timeline | 3.2.0 | readOnly | store-tools.ts:210 |
| case_export | 3.2.0 | write or mixed; inspect action-level contract | store-tools.ts:240 |
| case_import | 3.2.0 | write or mixed; inspect action-level contract | store-tools.ts:268 |
| case_summary | 3.2.0 | readOnly | store-tools.ts:292 |
| case_status | 3.2.0 | write or mixed; inspect action-level contract | store-tools.ts:316 |
| case_docket | 3.2.0 | readOnly | store-tools.ts:340 |
| case_memo | 3.2.0 | write or mixed; inspect action-level contract | store-tools.ts:364 |
| case_evidence_log | 3.2.0 | write or mixed; inspect action-level contract | store-tools.ts:405 |
| case_eval | 3.2.0 | write or mixed; inspect action-level contract | store-tools.ts:441 |
| case_reference | 3.2.0 | write or mixed; inspect action-level contract | store-tools.ts:478 |
| case_source | 3.2.0 | readOnly | store-tools.ts:517 |

Handlers in `server.ts` call core, court-language and survival-guide implementations. `registerStoreTools(server)` registers all sixteen store handlers; their imported store functions exist. This is implementation proof, not operational/API parity. `case_export` can write files; status/memo/evidence-log/eval/reference tools mix read and write actions. `case_query` is separately guarded in source. Do not infer safety solely from tool names or MCP annotations.

The smaller Codex adapter implements only: open_dashboard, route_issue, calculate_planning_date, get_packet_plan, get_checklist, audit_sources, search_guide, build_chronology.

Both variants expose release-status and verified-sources resources, guarded intake/claim-verification prompts, and eight UI widget roles (dashboard, issue route, planning date, packet, checklist, sources, search, chronology). Keep tool, prompt, resource, and widget parity distinct.

## Script, hook and sidecar capabilities

| Kind | Name | Incorporation |
|---|---|---|
| utility_script | chronology_builder | Port behind bounded job/API adapter or keep server-side callable utility |
| utility_script | citation_url_checker | Port behind bounded job/API adapter or keep server-side callable utility |
| utility_script | deadline_calculator | Port behind bounded job/API adapter or keep server-side callable utility |
| utility_script | exhibit_indexer | Port behind bounded job/API adapter or keep server-side callable utility |
| utility_script | redaction_helper | Port behind bounded job/API adapter or keep server-side callable utility |
| utility_script | traceability_check | Port behind bounded job/API adapter or keep server-side callable utility |
| utility_script | update_checker | Port behind bounded job/API adapter or keep server-side callable utility |
| utility_script | court_language | Port behind bounded job/API adapter or keep server-side callable utility |
| utility_script | route | Port behind bounded job/API adapter or keep server-side callable utility |
| utility_script | survival_guide | Port behind bounded job/API adapter or keep server-side callable utility |
| utility_script | case-cli | Port behind bounded job/API adapter or keep server-side callable utility |
| utility_script | import-vincent-schema | Port behind bounded job/API adapter or keep server-side callable utility |
| utility_script | load-content-to-store | Port behind bounded job/API adapter or keep server-side callable utility |
| maintenance_script | install_case_db_service | Retain operational provenance; do not install during port |
| maintenance_script | install_console_home | Retain operational provenance; do not install during port |
| hook | case_law_prompt_hook | Adapt hook semantics into workflow policy |
| hook | case_law_tool_hook | Adapt hook semantics into workflow policy |
| sidecar_route | GET /api/health | Transitional bridge source; replace machine-specific loading with governed transport during full port |
| sidecar_route | GET /api/auth/status | Transitional bridge source; replace machine-specific loading with governed transport during full port |
| sidecar_route | GET /api/store/summary | Transitional bridge source; replace machine-specific loading with governed transport during full port |
| sidecar_route | GET /api/store/search | Transitional bridge source; replace machine-specific loading with governed transport during full port |
| sidecar_route | GET /api/store/graph | Transitional bridge source; replace machine-specific loading with governed transport during full port |
| sidecar_route | GET /api/store/timeline | Transitional bridge source; replace machine-specific loading with governed transport during full port |
| sidecar_route | GET /api/store/factor-map | Transitional bridge source; replace machine-specific loading with governed transport during full port |
| sidecar_route | GET /api/store/docket | Transitional bridge source; replace machine-specific loading with governed transport during full port |
| sidecar_route | GET /api/store/memos | Transitional bridge source; replace machine-specific loading with governed transport during full port |
| sidecar_route | GET /api/store/status | Transitional bridge source; replace machine-specific loading with governed transport during full port |
| sidecar_route | GET /api/store/source | Transitional bridge source; replace machine-specific loading with governed transport during full port |
| sidecar_route | GET /api/store/reference | Transitional bridge source; replace machine-specific loading with governed transport during full port |
| sidecar_route | GET /api/store/evidence | Transitional bridge source; replace machine-specific loading with governed transport during full port |
| sidecar_route | GET /api/store/evals | Transitional bridge source; replace machine-specific loading with governed transport during full port |
| sidecar_route | POST /api/store/export | Transitional bridge source; replace machine-specific loading with governed transport during full port |
| sidecar_route | POST /api/chat | Transitional bridge source; replace machine-specific loading with governed transport during full port |
| runtime_adapter | built-store-loader | Reuse contract through callable bridge; port host-specific wiring in later stage |
| runtime_adapter | Claude-Agent-SDK-chat | Reuse contract through callable bridge; port host-specific wiring in later stage |

Hooks bind UserPromptSubmit plus CourtListener PostToolUse/PostToolUseFailure. Their behavior should become explicit workflow/source-verification policy; importing a hook file does not activate it in Advocatio. Installer scripts are retained inventory, not a port execution step.

Workbench `sidecar/lib/store-client.mjs` imports the **built full local plugin** store via an absolute machine path and defaults to local shared SurrealDB on port 8471. `sidecar/lib/chat.mjs` uses Claude Agent SDK with explicit stdio full-console launch, user settings discovery, and wildcard console-tool allowlists plus Read. That allows potentially writable store tools; it is not a demonstrated read-only bridge. The UI server has 16 explicit routes and loopback-only binding. No evidence showed a shared Advocatio bridge already consuming it.

## Outside resources and complete content scope

CourtListener is explicitly configured at `https://mcp.courtlistener.com/` in the full plugin. Local integration notes describe interactive authentication; this inventory did not log in or fetch its authenticated tool list. Historical federation inventory distinguishes pre-auth tools from post-auth research tools. Do not claim current counts or coverage from those historical numbers. MiCOURT Case Search is assessment-only: no handler or MCP entry exists. Its documented access assumptions need fresh official verification when integration begins.

The local ledger contains **193 source records**. Each is listed by ID and public source URL in JSON, with recorded authority/currentness metadata; this does not newly verify any legal claim. Domain breakdown:

- 7thcircuitcourt.com: 1
- circuit7.org: 1
- cms2.revize.com: 16
- cms7files.revize.com: 5
- community.icle.org: 3
- icle.org: 2
- mdhhs-pres-prod.michigan.gov: 7
- michiganlegalhelp.org: 1
- micourt.courts.michigan.gov: 1
- mifile.courts.michigan.gov: 2
- uscode.house.gov: 2
- www.courts.michigan.gov: 104
- www.ecfr.gov: 3
- www.geneseecountymi.gov: 3
- www.govinfo.gov: 1
- www.legislature.mi.gov: 23
- www.michigan.gov: 3

| Resource pack | Observed files | Incorporation |
|---|---:|---|
| content/toolkit/cheatsheet | 2 | Reconcile existing store copy, then staged full port with provenance |
| content/toolkit/checklists | 13 | Reconcile existing store copy, then staged full port with provenance |
| content/toolkit/decision-trees | 10 | Reconcile existing store copy, then staged full port with provenance |
| content/toolkit/templates | 25 | Reconcile existing store copy, then staged full port with provenance |
| content/toolkit/references | 61 | Reconcile existing store copy, then staged full port with provenance |
| content/custody-guide | 194 | Reconcile existing store copy, then staged full port with provenance |
| content/reference | 2 | Reconcile existing store copy, then staged full port with provenance |
| content/tools | 20 | Reconcile existing store copy, then staged full port with provenance |
| skills/court-language | 3 | Reconcile existing store copy, then staged full port with provenance |

The 17 survival-guide event packs are individually indexed in JSON. These are structured guidance contexts, not automatic document drafting or filing. Existing content includes checklists, decision trees, templates, authority references, source ledgers, court-language lexicon/examples/templates, and guide drafts. Preserve source status and draft/release distinction when moving them.

## SurrealDB and remote MCP reconciliation

**The user's recollection has a concrete source basis.** `mcp-app/src/content-store.ts` implements store-first reads from `fct/case` tables `reference` and `source`, with five-minute cache and local-file fallback. `README-store.md` lines 305–364 and `scripts/load-content-to-store.mjs` map the corpus into those rows. The cloud compose points the console at `ws://surreal-case:8000`; the same source supports stdio and Streamable HTTP.

| Content | Row mapping | Local payload count |
|---|---|---:|
| Verification ledger | source:<ledger ID> | 193 |
| Master directory | reference:master-source-directory | 1 |
| Court-language lexicon | reference:court-language-lexicon | 1 |
| Court-language and survival-guide templates | Four stable reference keys | 4 |
| Survival guide event packs | reference:<event ID> | 17 |

These are **local payload counts / implemented loader targets**, not a verified current database count. A source or reference row can be absent while tools still work through file fallback. The README also says R2 pointers are only populated when the primary-file path resolves; ledger write-up paths and downloaded-primary paths were not fully reconciled. Do not infer all original bytes are already in R2.

Historical evidence:

- `Probata/probata/docs/planning/2026-09-08-contextforge-federation-inventory.md:83` reports the console HTTP app already deployed at tailnet port 8765, talking to surreal-case. The same inventory reports an empty ContextForge catalog and proposes the case-work virtual server.
- `Probata/probata/docs/handoffs/HANDOFF-2026-09-10-cloud-services-tailscale-devbox.md:34` retains the federation gap; lines 64–67 leave federation/cloud ingress awaiting phase approval.
- Claude session `42902756-7c05-4c14-af49-33e520a19692`, line 10058, timestamp `2026-09-08T15:57:38.848Z`, compacted history lists content-load, gateway registration and plugin repoint as pending. This is prior summary evidence, not an execution receipt.
- Current full plugin `.mcp.json` still uses local stdio. No remote repoint is present in that file. The historical compose build-source path `deploy/docker/family-court-console/src` was not present at the inspected current path; do not recreate it without source/runtime reconciliation.

Current bounded probes of **historical addresses only** (2026-09-13): unauthenticated HTTP GET to `100.91.190.107:8765/healthz`, `100.91.190.107:8765/mcp`, and local `127.0.0.1:8471/health` all returned connection refused (`URLError`, `WinError 10061`). This verifies failure from this workstation at those historical routes, not absence of data or a deployment under a newer route. No credentials or private records were read.

**No completed current content-migration count receipt or live remote tool-list parity was recovered in this bounded pass.** Reconcile existing database identity, row IDs/hashes and any newer deployment receipt before running another import. Do not use the existing loader's `--dry-run` as a read-only database probe: it calls `getStore()` before the dry-run write branch, and `getStore()` initializes schema.

## Current addressing constraint

The parent verified the September 12 service-port/tailnet addressing standard and `deploy/service-port-registry.json`. Eventual bringup must use registered product NN: PostgreSQL 54NN, Surreal 84NN, API/MCP 80NN, portal 90NN; clients use Tailscale Service DNS over HTTPS/WSS 443, not raw backend IP/ports. Current allocations reported by the parent: Probata 71, Docstore 72, Intake 73; toolkit/Advocatio remains unallocated. The old 8471/8765 addresses above are recovery evidence only. Do not carry them into new bridge configuration or invent an allocation.

## Staged complete-port disposition

1. Recover existing remote inventory through a verified read-only DB connection and current deployment routing. Compare stable source/reference IDs, SHA-256, loaded_at and source_path. Record existing versus missing content; do not blindly duplicate the 193/23 payload.
2. Build the shared callable bridge against the full 27-tool contract, prompts/resources and external research seam. Allowlist by operation and action, preserve product authority, return source/receipt identity and explicit unavailable status. This provides useful early access while full port proceeds.
3. Port all 36 skills, 9 agents, 7 commands, 11 procedures and useful utility logic into governed workflow/help/job surfaces. Resolve broken member routing and host-specific paths; preserve provenance and product safety/review behavior.
4. Port the UI into settled React/TanStack/Storybook/Glide foundation. Keep MCP widgets as transitional views and adapters only where they serve a distinct host need.
5. Reconcile/port remaining corpus and reference artifacts, then retire duplicate runtime/config paths only after counts, hashes, behavior, auth, read/write boundaries and synthetic end-to-end fixtures pass. Preserve originals in the owning quarantine when retirement is authorized.

No installation, model call, service change, file deletion, data mutation or connector activation occurred. No source/code migration is claimed. This file and JSON are new planning inputs only; governed Docstore registration remains the parent's lane.

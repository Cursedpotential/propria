# What's inside this archive

Everything built for the Genesee County pro se custody project, as of **August 12, 2026**.
All paths below are relative to this archive's root.

> **Milestone: Volume 1 (the first-60-days volume) is COMPLETE in this archive.**
> Beyond-Volume-1 modules are still being drafted and will appear in a later archive.

---

## 1. The installable plugin — **Family Court Toolkit** (v1.3.0)

- `.claude-plugin/marketplace.json` — the marketplace catalog (install entry point)
- `Projects/custody-guide/toolkit-package/plugin/` — the plugin itself:
  - `skills/custody-packet/` — the reason-and-act (ReAct) packet-builder skill + all bundled legal content
  - `skills/toolkit/` — the general Genesee procedure/evidence/verification skill (references, templates, checklists, decision-trees, scripts, tests)
  - `commands/`, `agents/`, `hooks/`, `.mcp.json` — slash commands, subagents, the CourtListener hook, the case-law MCP wiring
  - `tests/` — 19-test suite (all green): manifest/schema, runtime-path resolution, MCP-name consistency, docs-only guards

**To install:** point Claude Code at `marketplace.json` and add the `family-court-toolkit` plugin.

## 2. The treatise — Volume 1 complete

`Projects/custody-guide/draft/`

**Front matter**
- `FM-0-triage-router.md` — the "What Happened Today?" deadline router
- `FM-1-6-front-matter.md` — disclaimer-as-content, safety override, edition/flag legend, Genesee quick-reference card, pre-filing safety, how-to-use

**Modules (Volume 1 — the first 60 days)**
- `M1-case-stage-jurisdiction-triage.md` — the ten procedural roads + jurisdiction screen
- `M2-standards-that-decide-your-case.md` — ECE, the two burdens, the Vodvarka/Shade gate, the 12 best-interest factors (verbatim), the parenting-time factors
- `M4-genesee-court-map-filing.md` — the Genesee court map, filing locations, motion-practice clocks
- `M5-where-evidence-rules-apply.md` — where the rules of evidence apply (and where they don't)
- `M6-relevance-hearsay.md` — relevance, the hearsay rule, the exceptions that matter
- `M7-authentication.md` — authenticating your exhibits (the foundation skeletons)
- `M8-custody-journal-documentation.md` — the journal as scaffolding, litigation hold, native files
- `M9-recording-law-do-not-do.md` — Michigan recording law: the hard stop
- `M10-discovery-records-subpoenas.md` — discovery, records, and non-party subpoenas
- `M12-pleadings-motions-service.md` — pleadings, motions, and the two service clocks
- `M13-temporary-orders.md` — temporary orders
- `M14-emergency-ex-parte-orders.md` — emergency ex parte orders and the 14-day objection
- `M15-foc-investigation-report-paradox.md` — the Friend of the Court report paradox
- `M16-referee-hearing-objection-de-novo.md` — the referee hearing, the 21-day objection, and what "de novo" really means
- `M26-parenting-time-enforcement-makeup.md` — parenting-time enforcement and make-up time

**Appendices (Volume 1)**
- `A-master-deadline-table.md` — every clock in the book, one 8-column table, swept against every module
- `C-evidence-foundation-objection-cards.md` — 16 podium-ready foundation/objection cards
- (Appendix N, the 231-term master glossary, is drafted inside `custody_guide_outline_v2.md`)

**Worked packet (the specific-parenting-time motion)**
- `P1-*.md` — motion packet, service pack, hearing prep, documentation protocol, translation card, worked examples

**Status:** Volume 1 (first-60-days) **COMPLETE**. Remaining (a later archive): Modules 3, 11, 17–25, 27–29; Appendices B, D, E, F, G, and the rest.

## 3. Project foundation (the spec and the verified sources)

`Projects/custody-guide/`
- `custody_guide_outline_v2.md` — the full drafting specification (Part 0 + 29 modules + 15 appendices, incl. the 231-term glossary)
- `custody_guide_plan.md`, `custody_guide_research.md` — scope/style and the legal research base
- `GUARDRAILS.md`, `CHEAT-SHEET.md`, `DRAFTING-HANDOFF.md`, `PRE-DRAFTING-CHECKLIST.md`
- `verification_ledger.md` — every citation's disposition (VERIFIED / PROVISIONAL / CONFLICTED / UNVERIFIED); now includes the closed Hayes holding and Duperon official-reporter cross-check
- `master_source_directory.md` — 213 sources; `council/` — the independent model reviews
- `sources/primary/` — archived primary Michigan law as structured markdown (court rules, evidence rules, local rules, support formula, MJI benchbooks) + `SHA256SUMS` + `fetch-sources.sh`

## 4. Evaluation results (how the skill was tested)

`Projects/custody-guide/custody-packet-workspace/`
- `iteration-1/` and `iteration-2/` — the full skill evaluation: 12 subagent runs (3 standard + 3 adversarial cases × the new ReAct skill vs. the old linear version), gradings, benchmarks, and **`review.html`** in each (open in a browser to click through every run). Both skill versions passed every check; the rework's value is a stricter intake gate and cleaner stop-behavior.

---

*Nothing here is legal advice or filing-ready. Statewide legal facts are verified against primary Michigan law as of the edition date; Genesee County local facts are flagged PROVISIONAL and must be confirmed with the court. Several items await licensed-attorney review before any reader use.*

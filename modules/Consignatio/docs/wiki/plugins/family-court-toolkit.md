---
title: "family-court-toolkit"
type: tool-reference
status: SOURCE_VERIFIED
date: 2026-10-04
generated_by: "Codex / GPT-6"
revision: 1
tags: [propria, tools, wiki]
---

# family-court-toolkit

> _Byline: Codex · GPT-6 · 2026-10-04 — generated from the cited sources._

Michigan Family Court Console for the owner's Genesee County custody matter — ONE entry skill (safety/jurisdiction gate first) routing to 35 on-demand members (intake, MCL 722.23 factors, evidence/MRE 901, documentation, financial discovery, Michigan drafting, FOC/CPS/DV/court resources, source verification, decision-record/appeal prep), 9 agents, 7 slash commands + 11 on-demand procedures, case-law hooks, a bundled MCP console (27 tools: route_issue with gazetteer + deadline proximity, audit_sources over the 193-record ledger, MCR-preset deadlines, case_facts, survival_guide, court_language_review, and an embedded SurrealDB case store — case_put/query/search/graph/factor_map/timeline/export/import/summary plus case_status/case_docket/case_memo/case_evidence_log/case_eval/case_reference/case_source for filed/draft works, a court-event-vs-master-timeline split, analysis/strategy/weakness memos, an evidence log, evals, and behavior-pattern/ontology reference data) and the official CourtListener MCP. 3.1.0 (2026-09-07) adds the owner's 13:09-13:16 case-store orders — filed/draft-works register, court vs master timeline lanes, memos, reference data, evidence log, evals, case status, per-record source provenance, a case-extract/v1 importer, and a platform (probata) export bundle — on top of the 3.0.0 restoration. Legal information only — not legal advice; guide publication remains attorney-review blocked.

Source: `E:/AI_Workspace/plugins/plugins/family-court-toolkit`. Version: `3.3.0`.
Registered: `True`. Installed manifests: .claude.

## How to invoke it

Slash commands run inside an agent app. Terminal commands run in PowerShell. MCP tools require an attached server and are called by the agent or an MCP client. A skill is an instruction package, not a standalone executable.

Descriptions below are extracted from source metadata/docstrings. This is discovery evidence, not a claim every service invocation passed.

## Commands

### `discovery`

(family-court-toolkit) Prepare Michigan RFA/RFP/deficiency/compel information

```text
/family-court-toolkit:discovery
```

Arguments: `[discovery issue]`

Source: [discovery.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/commands/discovery.md:1>) · SHA-256 `1995d6adb6c7c1b3e5431442b1c1cc60b6bed1030e3496a959b17d9d1549083f`

### `evidence`

(family-court-toolkit) Organize or assess evidence with lawful-access gating

```text
/family-court-toolkit:evidence
```

Arguments: `[evidence type]`

Source: [evidence.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/commands/evidence.md:1>) · SHA-256 `a9467ecbe945ae6ec939b6a46b33f3592951db9856d3ddc53c3f94c60a83f7a8`

### `family-court`

(family-court-toolkit) Route a Genesee family-court task to the toolkit

```text
/family-court-toolkit:family-court
```

Arguments: `[issue or goal]`

Source: [family-court.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/commands/family-court.md:1>) · SHA-256 `65e2b7476804b51c5657c1b32b43b428e119eddf24217ca5b04c2cf0cfe4d55e`

### `hearing`

(family-court-toolkit) Prepare a hearing-day plan and exhibit organization

```text
/family-court-toolkit:hearing
```

Arguments: `[hearing date or issue]`

Source: [hearing.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/commands/hearing.md:1>) · SHA-256 `9eb92fe8ea2141d94ae4ed195b3a4fb14ffa51f29b5b7b35fca3daa4436c29c8`

### `motion`

(family-court-toolkit) Prepare a motion-practice information/drafting plan

```text
/family-court-toolkit:motion
```

Arguments: `[requested relief]`

Source: [motion.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/commands/motion.md:1>) · SHA-256 `1935a6cb870e880b8406f55910a291b88c1c786e6eed3d2068b89175ee6b06f3`

### `packet`

(family-court-toolkit) Build a tailored parenting-time filing set from assembled case facts, or list what is still missing

```text
/family-court-toolkit:packet
```

Arguments: `[filing goal, or 'intake']`

Source: [packet.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/commands/packet.md:1>) · SHA-256 `04ad2c3a6d9749b3774eef0489e30f3ed97df46d5bc6fbfc496611025af4ed0a`

### `verify-sources`

(family-court-toolkit) Verify currentness and traceability of cited sources

```text
/family-court-toolkit:verify-sources
```

Arguments: `[source IDs or topic]`

Source: [verify-sources.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/commands/verify-sources.md:1>) · SHA-256 `e4414b62d29d3188a674169d0f1af6ade5a0cd5b7145d59dd788204f407520b7`

## Skills

### `fct-behavioral-pattern-analyzer`

(family-court-toolkit) Analyzes behavioral cues, manipulation patterns, and psychological abuse indicators for custody dispute defense strategy. Identifies reactive abuse dynamics, DARVO patterns, coercive control, alienation, gaslighting, and triangulation. Maps findings to MCL 722.23 factors and MRE 702 evidentiary standards. Use when analyzing communication records, incident reports, or behavioral patterns.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/skills/behavioral-pattern-analyzer/SKILL.md:1>) · SHA-256 `723b412ec8850c6b85223cafc02036b8ffe95ffb6c2b83cccc12b4ea17d4ba2b`

### `fct-best-interest-factors`

(family-court-toolkit) Michigan MCL 722.23 Best Interest Factors (a)-(l) with court analysis guidance, documentation prompts for strengths/deficits, and factor tracking worksheets for custody disputes.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/skills/best-interest-factors/SKILL.md:1>) · SHA-256 `e1a54a902fc5840cca361df315a74a405deee681018678d8b2c996b2dc537316`

### `fct-case-intake`

(family-court-toolkit) Structured case intake questionnaire for Michigan family law custody disputes. Use at the start of any new case analysis to systematically gather facts before performing IRAC analysis. Ensures no critical factors are missed.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/skills/case-intake/SKILL.md:1>) · SHA-256 `29b5270ba116f9451feffd7b203b39a5b096ebaa5e2b080fa74922b7528c05ba`

### `fct-case-research`

(family-court-toolkit) Structured Michigan case law research workflow with verification. Use when searching for case authority, verifying citations, or building legal arguments requiring specific Michigan precedent.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/skills/case-research/SKILL.md:1>) · SHA-256 `f62409ffcf6df95210df075e8beb3c7a2cff8f8e365ccb8bc51d9f6c03b90641`

### `fct-case-store`

Use the shared Family Court SurrealDB case and reference library through hosted ContextForge tools: full private case context, exact revisions, source provenance, citation validation and separate timeline lanes.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/skills/case-store/SKILL.md:1>) · SHA-256 `a5fc6e47bfcb668c2a827154e5061beaa7243c9def44e8ee0d39ddd9a3e2a5f9`

### `fct-child-support-worksheet`

(family-court-toolkit) Drafts a child support guidelines worksheet by extracting financial data, applying jurisdiction-specific guideline models, and calculating obligations. Triggers when preparing child support calculations, modification filings, or guideline worksheets for court submission.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/skills/child-support-worksheet/SKILL.md:1>) · SHA-256 `327cf73319bbf2e660115f71c477376967944d2e0a4bd6f4ddf8dc87451340bc`

### `fct-court-language`

(family-court-toolkit) Court-safe language reviewer/rewriter for Michigan family-court documents and testimony prep: run court_language_review to flag banned clinical labels, absolutes, mind-reading, characterizations, threats, child-as-witness, speculation, legal conclusions, recording admissions, and minor PII — then rewrite following the tool''s rewrite_plan, TEMPLATES.md, and EXAMPLES.md, and re-review until the score clears 90 with zero stop flags. Use for affidavits, motion briefs, testimony prep, messages to the other parent, incident logs, and referee-recommendation objections.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/skills/court-language/SKILL.md:1>) · SHA-256 `ee5fe90e7601556d61d963559e45bfcfaab7e0a44e19ead75cd21732113feb2d`

### `fct-court-resources`

(family-court-toolkit) Michigan One Court of Justice resource directory. Courts.michigan.gov navigation, SCAO forms, published opinions, court rules, benchbooks, free case law sites, Genesee County 7th Circuit resources, and self-help tools.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/skills/court-resources/SKILL.md:1>) · SHA-256 `903cfa19ead796e872a095b3062e7d0d5af53d9f23b69f4564bda0ccc85fc434`

### `fct-cps-resources`

(family-court-toolkit) Michigan MDHHS Children's Protective Services process, investigation stages, dispositions, Central Registry, mandated reporters, forms, record access, and CPS-custody intersection.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/skills/cps-resources/SKILL.md:1>) · SHA-256 `802e30de18b24616655960052f7fcdcba3d438cdc63606bb95d33a602b57cffe`

### `fct-custody-evaluation-summary`

(family-court-toolkit) Summarizes custody evaluation reports into a structured memorandum covering evaluator credentials, methodology, parental findings, recommendations, and best-interests factor mapping. Use when reviewing custody evaluations, preparing for custody hearings or settlement conferences, or onboarding to contested parenting matters.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/skills/custody-evaluation-summary/SKILL.md:1>) · SHA-256 `e9aa24de79e2c45641ebc41ff89a507b2ab559af4b210b7b62108bca13f44962`

### `fct-custody-packet-builder`

(family-court-toolkit) Organize a Michigan custody or parenting-time packet without claiming court acceptance or legal sufficiency. Use for hearing binders, exhibit indexes, fact chronologies, order collections, evidence packets, redaction checks, and guarded drafting plans.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/skills/custody-packet-builder/SKILL.md:1>) · SHA-256 `60082f675afc81d7fb4e6a50c89e0c3bd88ecd766fafc4dabbd13151f54db671`

### `fct-custody-packet-gen1`

(family-court-toolkit) Use to build a real, tailored Genesee County parenting-time/custody filing set from assembled case facts — motion for specific parenting time, proposed order, concurrence certificate, notice of hearing, service pack, and hearing prep — driven by an explicit reason-and-act loop under the project's verified-source and translation discipline.

Arguments: `[filing goal, or 'intake' to see what is still needed]`

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/skills/custody-packet-gen1/SKILL.md:1>) · SHA-256 `73bd9cc783cac73c6b372c8308d0f7e0d091d3acdbd2f2efc4c4202506c517b0`

### `fct-decision-record-verification`

(family-court-toolkit) Cross-references a judge's decision, court opinion, or final order against one or more hearing/trial/deposition transcripts to verify whether the record supports each finding. Produces strict dual-citation blocks with document names, page/line references, and verbatim quotes from both sources. Use when preparing appellate review, post-trial motions, record discrepancy audits, or transcript-based fact checks of judicial findings.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/skills/decision-record-verification/SKILL.md:1>) · SHA-256 `d531b16f94d980810b22ccdce4c3cd10df9206a47b5c801d3ba79994ff04b7e6`

### `fct-documentation-methods`

(family-court-toolkit) FACT method incident documentation, BIFF communication for high-conflict situations, pattern logging, evidence preservation workflows, and reframing emotions for court presentation.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/skills/documentation-methods/SKILL.md:1>) · SHA-256 `aa8bc13e54a784df3310e2e9b732040fcf87be693d09f901d0c06c2aa2bbd33b`

### `fct-dv-resources`

(family-court-toolkit) Michigan domestic violence statutes, PPO process, psychological abuse and coercive control, DV-custody intersection, key case law topics, Genesee County DV resources, and DV benchbook references.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/skills/dv-resources/SKILL.md:1>) · SHA-256 `5f49358bb0463c1c5252eab4774e8670ea3a850506f6ef89cdfd60b622a52e75`

### `fct-dvro-petition`

(family-court-toolkit) Drafts court-ready Domestic Violence Restraining Order petitions compiling chronological abuse incidents into element-driven pleadings supporting ex parte TRO and permanent protective order relief. Covers personal conduct orders, stay-away orders, custody/visitation, move-out orders, property control, and firearms relinquishment. Use when drafting DVRO petitions, protective order requests, ex parte TRO applications, or domestic violence pleadings.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/skills/dvro-petition/SKILL.md:1>) · SHA-256 `92168387094c15fc289286429d19c487f69ddab2f2f7379e984177180e930c79`

### `fct-evidence-templates`

(family-court-toolkit) Evidence intake, cataloging, chain of custody, hash verification, exhibit preparation, and quality control templates for digital evidence processing in legal proceedings.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/skills/evidence-templates/SKILL.md:1>) · SHA-256 `1130f63055ee53d97bec5b4aae257f3ef772125aa2cc446ff731538d152465be`

### `family-court-toolkit`

(family-court-toolkit) Entry point for the owner's Michigan (Genesee County) custody / parenting-time matter: safety and jurisdiction gate first, then routes to 35 on-demand members — intake, MCL 722.23 factors, evidence and MRE 901 authentication, documentation (FACT/BIFF), financial discovery, Michigan drafting, FOC/CPS/DV/court resources, source verification, decision-record and appeal prep — plus a bundled MCP console (27 tools: safety router, ledger-backed source audit, MCR-preset deadlines, case_facts, survival_guide, court_language_review, and a SurrealDB case store with search + graph, filed/draft-works docket, memos, evidence log, evals, and behavior-pattern reference data) and the official CourtListener MCP. Triggers: custody, parenting time, FOC, referee, objection, PPO, CPS, UCCJEA, hearing prep, exhibit, motion, Genesee, MCL 722, best interest factors, GAL, transcript, discovery, child support.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/skills/family-court-toolkit/SKILL.md:1>) · SHA-256 `5632332b5c1403313a3db5fbf508d66ec761bc03b6d83d1182685602fdfb7718`

### `fct-foc-resources`

(family-court-toolkit) Michigan Friend of the Court procedures, handbook, child support formula, parenting time guidelines, FOCB policy memoranda, objection process, and Genesee County FOC specifics.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/skills/foc-resources/SKILL.md:1>) · SHA-256 `45821e9fb1b1d431d5b3a7b120b8a29f51fe4bc98641f7d20acddc42cce5edd1`

### `fct-irac-formatter`

(family-court-toolkit) Enforces strict IRAC format for Michigan family law analysis. Use to structure any legal analysis output into proper Issue-Rule-Analysis-Conclusion format with required disclaimer and citations.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/skills/irac-formatter/SKILL.md:1>) · SHA-256 `db6799a5415e052db23b82bbc7cecdee0fc8009878d8c8d06be11f1cf36b7d65`

### `fct-manipulation-patterns`

(family-court-toolkit) Recognition guide for psychological manipulation in custody disputes: narcissistic abuse cycle, DARVO, gaslighting, triangulation, flying monkeys, coercive control, NPD/BPD patterns, alcoholism, and parental alienation. Includes reverse-DARVO strategy and court-specific implications.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/skills/manipulation-patterns/SKILL.md:1>) · SHA-256 `ae0c37310aebf348c2113961a679e7fcb3daf9e6c385fc99274bb3d40a661223`

### `fct-mcl-factor-mapper`

(family-court-toolkit) Maps case facts to all 12 MCL 722.23 best interest factors with evidence strength ratings. Use when analyzing custody disputes, preparing for hearings, or building factor-by-factor arguments.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/skills/mcl-factor-mapper/SKILL.md:1>) · SHA-256 `1b833b3669f7c29db7c32d0eb96312ce3bf9eb83b77ea6435bef75c5e584dad0`

### `fct-michigan-family-court-guide`

(family-court-toolkit) Route and explain Michigan family-court and custody procedure using current primary authority, explicit uncertainty, and safety gates. Use for Michigan custody, parenting time, FOC, PPO, CPS, UCCJEA, evidence, hearing, order, or appeal questions.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/skills/michigan-family-court-guide/SKILL.md:1>) · SHA-256 `5231fe763c8af6605ceae2dfbf24716712ba231905f7c198367bd34bd40f2d07`

### `fct-motion-for-temporary-relief`

(family-court-toolkit) Drafts a Motion for Temporary Relief (Pendente Lite) in U.S. family law cases seeking interim orders for custody, support, property use, and fees. Use when filing for temporary orders at the outset of a dissolution, separation, or custody matter to stabilize rights and obligations during pendency.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/skills/motion-for-temporary-relief/SKILL.md:1>) · SHA-256 `419844ed351fad7af3e17214111693be9b24032e524582c0995f15699013d7a7`

### `fct-mre-authentication`

(family-court-toolkit) Michigan Rules of Evidence 901 authentication framework for digital evidence. Distinctive characteristics analysis, reply doctrine, source evaluation, and admissibility assessment templates.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/skills/mre-authentication/SKILL.md:1>) · SHA-256 `40ff1f82552bb08c1a8f37dd987b6bac2d5035c6b9da7218ddaf63b8ddc2f3cf`

### `fct-order-modification`

(family-court-toolkit) Drafts post-judgment motions to modify existing family law court orders based on material changes in circumstances. Structures changed-circumstances arguments with jurisdictional compliance, factual chronologies, and precise relief specifications. Use when drafting modification motions, changed-circumstances motions, post-judgment family law motions, or petitions to modify custody, support, or visitation orders.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/skills/order-modification/SKILL.md:1>) · SHA-256 `60653113557b98934608e4838b8b90f8de027092e507b974a24e69b16df6e8d4`

### `fct-parenting-plan`

(family-court-toolkit) Drafts jurisdiction-compliant parenting plans covering custody frameworks, time-sharing schedules, decision-making allocation, financial provisions, relocation, and dispute resolution. Adapts terminology and structure to state-specific statutory requirements. Use when drafting custody agreements, parenting time schedules, time-sharing plans, or co-parenting arrangements in family law proceedings.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/skills/parenting-plan/SKILL.md:1>) · SHA-256 `e5467eddffa28d8f4f5e983d3f39f5d184c5d40fa54336b763734e7f7d17c2ee`

### `fct-referee-hearing-survival`

(family-court-toolkit) In-the-room guide for a Michigan FOC referee hearing as a pro se litigant: opening-statement options, question types with minimal/clarifying/contextual/boundary answers, trap questions, emotional-state management, 'do not do this' list. Use the day before and the morning of a referee hearing.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/skills/referee-hearing-survival/SKILL.md:1>) · SHA-256 `6d751263161f9ba72cb887ec9e1b05e0c4e5b3e78ecb8bf7691f392237812a19`

### `fct-secondary-source-auditor`

(family-court-toolkit) Comprehensive audit of all Michigan family law secondary admissible sources. Checks MJI Benchbooks, FOCB policies, MDHHS/CPS protocols, SCAO forms/orders, and ICLE publications for relevant authority on a given issue. Use when building a complete evidentiary foundation beyond primary case law.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/skills/secondary-source-auditor/SKILL.md:1>) · SHA-256 `b72f14da9b5606e7ef6d3b88f568dc97d2af08aadd2cd34deb2257143ce1e612`

### `fct-survival-guide`

(family-court-toolkit) In-the-room survival guide generator for any Michigan family-court hearing, interview, or filing (referee hearing, de novo hearing, motion hearing, evidentiary hearing, show-cause/contempt, PPO hearing, FOC interview, GAL/LGAL interview, custody evaluation, mediation, and 7 document types). Call survival_guide to get a context pack + writing template; the calling model writes the guide, enriched with case_facts and the matching draft module. Use the day before or the morning of a hearing, or before drafting a motion/objection/affidavit.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/skills/survival-guide/SKILL.md:1>) · SHA-256 `5ad1ca18e15adacaf42e0d8e541510060f3b71b2ee25bc4d3b2b6c7b2c3d9e52`

### `fct-tax-return-analysis`

(family-court-toolkit) Produces litigation-ready financial memoranda from multi-year tax returns, covering income trends, filing status changes, deduction patterns, red flags, and trustee considerations. Use when reviewing Form 1040s for bankruptcy means testing, family law support calculations, personal injury damages, or financial discovery analysis.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/skills/tax-return-analysis/SKILL.md:1>) · SHA-256 `50a7662663e61d5257796a8bd643cbd8ce95416230ce20a81c7b337db06c9eb3`

### `fct-tax-return-summary`

(family-court-toolkit) Produces structured, citation-backed summaries of tax returns (Form 1040, schedules, W-2s, 1099s) for divorce, personal injury, and bankruptcy litigation. Extracts income sources, deductions, credits, and flags anomalies. Use when summarizing tax returns, analyzing financial discovery, assessing earning capacity, reviewing 1040s, or preparing financial profiles for litigation.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/skills/tax-return-summary/SKILL.md:1>) · SHA-256 `f7bd6e43654df4f4635dccd70afb845436f90d90e859fd3ac033450efa352817`

### `fct-toolkit-gen1`

(family-court-toolkit) Use for read-only legal-information, evidence organization, source verification, motion/discovery drafting structure, Friend of the Court/referee workflow, hearing preparation, or safety gating in a Genesee County, Michigan family-court matter.

Arguments: `[issue, document, or hearing goal]`

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/skills/toolkit-gen1/SKILL.md:1>) · SHA-256 `d19a3f7d98bb2b0f2c7f6a45660bf1a4ded605b81092f2d3d41167ad0d35ff94`

### `fct-track-deposits`

(family-court-toolkit) Traces deposits from receipt through disbursement across bank statements and financial records, producing transaction matrices, fund-flow timelines, and evidentiary chains of custody. Flags structuring, commingling, trust account violations, and unexplained gaps. Use when tracking deposits, tracing funds, auditing trust accounts, or analyzing bank statements during discovery.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/skills/track-deposits/SKILL.md:1>) · SHA-256 `405e31937494ab2b8d0161d0d802ae1e3d008ed30e77e1933ca5c4225c91e4df`

### `fct-trusted-sources`

(family-court-toolkit) Acquire authoritative reference documents — statutes, regulations, official policies, terms of service, agency guidance — into the case folder with provenance and plaintext extraction. Triggers when the user names a specific law/regulation/policy/ToS the case will cite, asks 'do we have a copy of X' or 'where do I find the official text of Y', supplies a copy and asks 'can we use this', or when packet-builder needs a reference appendix that isn't on disk.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/skills/trusted-sources/SKILL.md:1>) · SHA-256 `3dd4a9160ce75ce0d36b597edbe9126a71d6dac023a39f73f596501cd2417c45`

### `fct-verify-michigan-legal-sources`

(family-court-toolkit) Audit Michigan family-law claims against current official primary sources with authority level, pinpoint, currency, claim-to-source fit, and conflict status. Use for statutes, court rules, SCAO forms, cases, MDHHS policy, local procedure, deadlines, or citation validation.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/skills/verify-michigan-legal-sources/SKILL.md:1>) · SHA-256 `310dd6a837381366565d4bab814965c8cce9e8e2635cef86817e0f3d5bbba54a`

## Agents

### `case-law-researcher`

Research case law conservatively using CourtListener discovery and primary-source verification.

Source: [case-law-researcher.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/agents/case-law-researcher.md:1>) · SHA-256 `1227eda3ca6881eff3ab1d0d239c58252d741e2e5db3e5c011611f30136260f7`

### `Custody Support Specialist`

Trauma-informed support for high-conflict custody cases involving NPD/BPD/alcoholism, coercive control, and parental alienation. DARVO recognition, BIFF communication, documentation coaching, and MCL 722.23 best interest analysis.

Source: [custody-support.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/agents/custody-support.md:1>) · SHA-256 `81e02d05ddd00097f6531671bfa0ec62999eaa9f4caf127c0e5f3fcb4c282a18`

### `evidence-organizer`

Organize lawful evidence, exhibit indexes, chronologies, and foundation issues.

Source: [evidence-organizer.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/agents/evidence-organizer.md:1>) · SHA-256 `b0f44a9e3cb30cc2ec4730866b8778552a0d9cec82482e876d902ee9dadba85f`

### `Digital Evidence Technician`

Digital evidence processing specialist. Text messages, emails, social media, photos, metadata extraction, hash verification, chain of custody, and court-ready exhibit preparation.

Source: [evidence-tech.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/agents/evidence-tech.md:1>) · SHA-256 `18fb68400d8e072ede281d2bc8d7dc9e4151f127781aabd6417721c215fe40c5`

### `family-court-document-drafter`

Create source-marked, informational working drafts for Genesee family-court documents.

Source: [family-court-document-drafter.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/agents/family-court-document-drafter.md:1>) · SHA-256 `3b19223a0c7a7c79e3733368c173e9562b9d356b34735b399d920f5ea4aa775f`

### `Forensic Evidence & Legal Research`

Forensic evidence analyst for court evidence authentication, digital forensics, timeline reconstruction, Bayesian evidence weighing, and Michigan case law research.

Source: [forensic.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/agents/forensic.md:1>) · SHA-256 `db1ceca1ce9949a15a99124f8fa6db7420e2820503d661806117a96686261ff7`

### `Litigation & Discovery`

Strategic discovery specialist for high-conflict Michigan family law. Drafts RFAs, interrogatories, RFPs, subpoenas, and affidavits. Expert in hostile witness impeachment, DARVO counter-strategies, and pro se litigation.

Source: [litigation.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/agents/litigation.md:1>) · SHA-256 `45355740ddea3837b2544fbeece0a02a14da7c3ad6244fee7fd8d99e50577920`

### `Michigan Family Law Attorney`

Michigan family law attorney specializing in custody disputes, MCL 722.23 Best Interest factors, FOC procedures, high-conflict cases involving NPD/BPD/alcoholism, IRAC motion strategy, and pro se litigation support.

Source: [michigan-law.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/agents/michigan-law.md:1>) · SHA-256 `a77f4cca6bf7f9a5a9ccb7c3d79cec11c2f7592542357138a6ba4a6c7ffed17a`

### `michigan-source-verifier`

Verify source identity/currentness and record limits for Michigan/Genesee material.

Source: [michigan-source-verifier.md:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/agents/michigan-source-verifier.md:1>) · SHA-256 `a3dabeb7d8429d75e8842d885f8d91058833236fbc542e7c7e250457520a626e`

## Cli Entries

No entries found in the inspected declarations.

## Scripts

### `content/toolkit/scripts/chronology_builder.py`

Normalize events into a chronology; drafting aid only.

```text
python "E:/AI_Workspace/plugins/plugins/family-court-toolkit/content/toolkit/scripts/chronology_builder.py" --help
```

Declared arguments: `--dry-run`, `--output`, `input_json`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| input_json | — | positional | — | — |
| --output | — | False | — | — |
| --dry-run | store_true | False | — | — |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [chronology_builder.py:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/content/toolkit/scripts/chronology_builder.py:1>) · SHA-256 `ac4f3dc9e9c2b4693ac4d0e6d8d9b1ace555c83b9f5bc45166921cf69154a5d2`

### `content/toolkit/scripts/citation_url_checker.py`

Read-only URL status checker; respects ordinary HTTP failures and does not evade blocks.

```text
python "E:/AI_Workspace/plugins/plugins/family-court-toolkit/content/toolkit/scripts/citation_url_checker.py" --help
```

Declared arguments: `--dry-run`, `--output`, `--timeout`, `ledger`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| ledger | — | positional | — | — |
| --output | — | False | — | — |
| --timeout | int | False | — | — |
| --dry-run | store_true | False | — | — |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [citation_url_checker.py:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/content/toolkit/scripts/citation_url_checker.py:1>) · SHA-256 `c195e0e4bd18421e8988ca9326b7a435c1673e4f1a47999c9bd67cc2195efd3f`

### `content/toolkit/scripts/deadline_calculator.py`

Candidate deadline arithmetic only; no appellate deadlines.

```text
python "E:/AI_Workspace/plugins/plugins/family-court-toolkit/content/toolkit/scripts/deadline_calculator.py" --help
```

Declared arguments: `--dry-run`, `--holidays-json`, `--kind`, `--output`, `--service-method`, `--trigger-date`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --trigger-date | — | True | — | — |
| --kind | — | True | ['referee-objection', 'motion-before-hearing', 'response-before-hearing', 'local-objection-service', 'local-objection-certificate', 'service-addon'] | — |
| --service-method | — | False | — | — |
| --holidays-json | — | False | — | — |
| --output | — | False | — | — |
| --dry-run | store_true | False | — | — |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [deadline_calculator.py:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/content/toolkit/scripts/deadline_calculator.py:1>) · SHA-256 `abdf5dada2f459e97dc77d9d47ef3a007e0abc093ba9f950c54fd8140d07c140`

### `content/toolkit/scripts/exhibit_indexer.py`

Create a read-only exhibit index and hashes after possession representation.

```text
python "E:/AI_Workspace/plugins/plugins/family-court-toolkit/content/toolkit/scripts/exhibit_indexer.py" --help
```

Declared arguments: `--authorized-possessor`, `--dry-run`, `--output`, `paths`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| paths | — | positional | — | — |
| --authorized-possessor | — | True | — | — |
| --output | — | False | — | — |
| --dry-run | store_true | False | — | — |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [exhibit_indexer.py:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/content/toolkit/scripts/exhibit_indexer.py:1>) · SHA-256 `6fcf8cb69f0d503e98841b431a0cdf4d476283916c6f08994926ae4226daee74`

### `content/toolkit/scripts/redaction_helper.py`

Flag likely PII in text; never creates a redacted file.

```text
python "E:/AI_Workspace/plugins/plugins/family-court-toolkit/content/toolkit/scripts/redaction_helper.py" --help
```

Declared arguments: `--dry-run`, `--output`, `input_text`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| input_text | — | positional | — | — |
| --output | — | False | — | — |
| --dry-run | store_true | False | — | — |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [redaction_helper.py:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/content/toolkit/scripts/redaction_helper.py:1>) · SHA-256 `89591a84a3fa23327f699a3ec241019120d19b21088e981d30ade7bd336681e9`

### `content/toolkit/scripts/traceability_check.py`

Check Markdown source markers against the ledger without changing files.

```text
python "E:/AI_Workspace/plugins/plugins/family-court-toolkit/content/toolkit/scripts/traceability_check.py" --help
```

Declared arguments: `--dry-run`, `--output`, `root`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| root | — | positional | — | — |
| --output | — | False | — | — |
| --dry-run | store_true | False | — | — |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [traceability_check.py:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/content/toolkit/scripts/traceability_check.py:1>) · SHA-256 `112826af7bc1b998e05339e3a7f4f668463413aff435256d129fbd92627713c2`

### `content/toolkit/scripts/update_checker.py`

Run URL checks and flag manifest staleness without modifying package sources.

```text
python "E:/AI_Workspace/plugins/plugins/family-court-toolkit/content/toolkit/scripts/update_checker.py" --help
```

Declared arguments: `--dry-run`, `--output`, `root`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| root | — | positional | — | — |
| --output | — | False | — | — |
| --dry-run | store_true | False | — | — |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [update_checker.py:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/content/toolkit/scripts/update_checker.py:1>) · SHA-256 `6e6437dafbd5cb81fd2ddc23fca35e63db5527f469cfc5c64a39817bdd72a40c`

### `hooks/case_law_prompt_hook.py`

Read-only Claude hook: add a narrow case-law research reminder or fail open.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [case_law_prompt_hook.py:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/hooks/case_law_prompt_hook.py:1>) · SHA-256 `e1a5973313854ec8910ba190e585e3662f7b223999089aaba2a13434f12e75b6`

### `hooks/case_law_tool_hook.py`

Read-only Claude hook: preserve CourtListener output and add verification/fallback context.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [case_law_tool_hook.py:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/hooks/case_law_tool_hook.py:1>) · SHA-256 `1b9634e7fb8aa35ca3fa6dd34b028d05594a06e36b85d8a14b0977e5d54e276a`

### `scripts/install_case_db_service.py`

Run SurrealDB as ONE local service in ~ so every harness (Claude Code, Codex, OpenCode, Gemini, the
Tauri app) shares the same case store. Owner ruling 2026-09-07 15:27-15:34: "install its binary in ~ ...
all the data there so other harnesses can use it" — embedded RocksDB is single-process (proven: a second
process hangs 60 s on the LOCK file), so the store must be a server, not an in-process engine.

What this does (idempotent, never deletes):
  ~/bin/surreal.exe                      server binary (downloaded from the GitHub release; already present)
  ~/.secrets/family-court-toolkit.env    CUSTODY_CASE_DB_USER / CUSTODY_CASE_DB_PASS (generated once; never printed)
  ~/bin/family-court-db.cmd              launcher: surreal start rocksdb:<~/.config/family-court-toolkit/case.db> --bind 127.0.0.1:8471
  Task Scheduler "family-court-db"       runs the launcher hidden at logon; also started now
  prints the CUSTODY_CASE_DB URL every client must use: ws://127.0.0.1:8471

Byline: Claude Code · Fable 5.1 · 2026-09-07

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [install_case_db_service.py:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/scripts/install_case_db_service.py:1>) · SHA-256 `286430720dca11dc72a7ff3661f9fdf3ccb257db337e59f3b5780248735ee697`

### `scripts/install_console_home.py`

Install the family-court-console MCP server + its data under the HOME dir so every harness
(Claude Code, Codex, OpenCode, Gemini CLI, ...) launches the SAME binary against the SAME case store.

Owner order 2026-09-07 15:27-15:30: "install its binary in ~ ... all the data there so other harnesses can use it".

Layout (all under ~/.config/family-court-toolkit/, which already holds case.db + intake/):
  console/mcp-app/dist/*.js        built server (surrealdb + @surrealdb/node stay external -> node_modules)
  console/mcp-app/node_modules/    production deps only (npm ci --omit=dev)
  console/content/, console/skills/ runtime content the server resolves relative to dist/../..
  HARNESSES.md                     copy-paste MCP config for each harness + what was registered
  ~/bin/family-court-console.cmd   launcher: node <home>/console/mcp-app/dist/server.js

Re-run after every plugin rebuild (`node build.mjs`) to refresh the home copy. Copies only; never deletes.
Byline: Claude Code · Fable 5.1 · 2026-09-07

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [install_console_home.py:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/scripts/install_console_home.py:1>) · SHA-256 `b6f83eebc5794dcfd4cb72e8f8d5cf837bc40566da7fe0babcf69bf3dc2cdf8e`

### `skills/family-court-toolkit/scripts/court_language.py`

Byline: Claude Code · Fable 5.1 · 2026-09-07
CLI twin of the court_language_review MCP tool. Same lexicon
(content/tools/court-language/lexicon.json), same deterministic review logic,
same safe_phrasebank loaded from EXAMPLES.md. Does not rewrite anything --
see `references/court-language/SKILL.md` for the rewrite protocol.

Usage:
  court_language.py review <file|-> --doc affidavit [--json]
  court_language.py phrasebank <doc_type>

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [court_language.py:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/skills/family-court-toolkit/scripts/court_language.py:1>) · SHA-256 `2de43081786f73a5c23bce42733ebe9ba2bd4673eca0ce01259ef95e5d2efacb`

### `skills/family-court-toolkit/scripts/survival_guide.py`

Byline: Claude Code · Fable 5.1 · 2026-09-07
CLI twin of the survival_guide MCP tool. Reads the same JSON knowledge base
(content/tools/survival-guide/events/*.json) and mechanically fills the
template with the fields the context pack actually carries — no LLM step, so
the opening-statement and question-type sections (which need live model
authorship from case facts) are left as pointers back to the MCP tool + the
`survival-guide` skill member, not fabricated.

Usage:
  survival_guide.py list
  survival_guide.py <event> [--card] [--out file.md]

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [survival_guide.py:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/skills/family-court-toolkit/scripts/survival_guide.py:1>) · SHA-256 `cf4cc289467396107af3822a04aa0c047838e51ce9f5ea83fb261bace1763de1`

## Mcp Tools

No entries found in the inspected declarations.

## Mcp Servers

### `family-court-console`

http

Validation: configured; health not inferred.

Source: [.mcp.json:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/.mcp.json:1>) · SHA-256 `2d6d190d8f8000041ca28924c0507ab15c5badce45e0b6c8a3ac6c332edbd461`

### `courtlistener`

http

Validation: configured; health not inferred.

Source: [.mcp.json:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/.mcp.json:1>) · SHA-256 `2d6d190d8f8000041ca28924c0507ab15c5badce45e0b6c8a3ac6c332edbd461`

### `courtlistener`

http

Validation: configured; health not inferred.

Source: [mcp.json:1](<E:/AI_Workspace/plugins/plugins/family-court-toolkit/.codex-plugin/mcp.json:1>) · SHA-256 `1aa258a290c6b72e64dbc2467fb00d65b9f22eb7005faabd024c572516f8f2fe`


Back to [[Code/wiki/plugin-inventory|Plugin inventory]].

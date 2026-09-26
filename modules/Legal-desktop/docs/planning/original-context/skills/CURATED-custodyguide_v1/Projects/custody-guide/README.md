# Pro Se Custody Guide — Genesee County, Michigan

A self-help treatise on custody, parenting time, and child support for a self-represented
parent in the 7th Judicial Circuit Court, Family Division (Genesee County, Michigan).

**Status: verification pass 1 complete; treatise not yet drafted.**

The repo now holds **two generations** of this project:

- **`/` (root)** — the *treatise* project: plan, research, drafting specification, council reviews.
- **`toolkit-package/`** — a later, more advanced generation: a master prompt, a 74-source verified ledger and audit, four council reviews, a **built Claude Code plugin** (194 files, v1.1.1, with a credential-free CourtListener MCP integration), the toolkit source (54 files), and `html-app/` — a built browser app for the toolkit.
- **`GUARDRAILS.md`** — project governance: what this may and may not become, who decides what, the four evidence dispositions, the publication gate, and the maintenance clocks.
- **`critical_review_outline_v2.md`** — adversarial review of the v2 drafting spec itself.
- **`verification_ledger.md`** — results of the August 9, 2026 verification pass against primary sources.
- **`research_interference_coercive_control.md`** — research addendum mapping interference / coercive control / weaponized process onto recognized Michigan statutory hooks.

## Files

| File | What it is |
|---|---|
| `custody_guide_plan.md` | Project plan — purpose, audience, scope, pedagogy, disclaimer plan, style guide |
| `custody_guide_research.md` | Research compendium — statutes, case law, court rules, Genesee local practice, SCAO forms |
| `custody_guide_outline_v2.md` | **The drafting specification.** Part 0 governing spec + front matter + 29 modules + 15 appendices |
| `master_source_directory.md` | 213 unique source URLs across 14 categories, extracted Aug 9 2026 |
| `council/model-council-claude_opus_5_0.md` | Council review — evidentiary/procedural-mechanics angle |
| `council/model-council-gpt_5_6_sol.md` | Council review — systems/pathway angle |
| `council/model-council-gemini_3_1_pro.md` | Council review — topical-completeness angle |
| `council/model-council-synthesis.md` | Merged verdict and rebuild recommendation |

Filenames match the cross-references used inside the documents themselves
(`custody_guide_plan.md`, `custody_guide_research.md`, `model-council-*.md`), so internal
references resolve.

## How this got here

1. Plan and research compendium drafted.
2. A v1 outline was drafted (not in this package — superseded).
3. Three models reviewed v1 independently. All three concluded v1 was **not safe to hand to a
   drafting model**: it taught strategy before proof, mirrored an attorney CLE's lesson
   numbering instead of the litigant's decision sequence, and omitted evidence/admissibility,
   subpoena practice, parenting-time enforcement, change of domicile, emergency ex parte
   practice, and appellate triage.
4. `custody_guide_outline_v2.md` is the rebuild: Sol's pathway spine, Opus's content map,
   Gemini's additions, with a 12-point drafting contract and 12 acceptance tests as the gate
   every module must pass. Nothing from any council report was dropped; conflicts are preserved
   side by side under `⊗ COUNCIL CONFLICT` notes.

## What v2 already contains

- **Part 0** — drafting contract, acceptance tests, two-tier flag system plus three hard labels,
  stop-and-seek-counsel gates, banned vocabulary, conflict-of-authority protocol.
- **Front matter** — the 48-hour triage router ("what happened today?" → module + clock + flag),
  safety override, pre-filing safety planning, Genesee quick-reference card.
- **Modules 1–29** across five parts: Choose the Path → Build Proof → Ask/Respond/Resolve →
  Try the Case → Orders/Enforce/Change/Review.
- **Appendices A–O.** Appendix N (Master Glossary) is **already fully drafted** — 231 terms,
  each with plain meaning, precise legal meaning, controlling source + URL, "do not confuse
  with," and owning module.

## Gates before drafting

Carried forward from outline §0.8:

- **Unverified ledger.** Exact *Vodvarka* / *Shade* citations and holdings; MCSF Parental Time
  Offset brackets (§3.03); MCR 2.300-series discovery limits; MCR 3.219; current text of
  MCR 3.207 and 3.210 post-ADM File 2021-27; MRE restyling effective date; full text of
  MCL 722.26a, 552.505a, 552.605b–c; holdings of *Shade*, *Berger*, *Dailey*; *Grew v Knox*
  (unpublished — persuasive only, MCR 7.215(C)(1)); whether any published Michigan opinion
  recognizes parental alienation as a freestanding doctrine (current research: none);
  FOC 89 / FOC 57 titles; **and every Genesee County local fact** (fees, judicial and referee
  rosters, FOC procedures, CDRP, MiFILE status, local administrative orders, form revision dates).
- **Verified in the v2 pass:** the 18 deviation factors in §1.04(E) of the 2025 MCSF Manual,
  fetched from the primary PDF (confirms the reduction from 20; resolves research §12 item 5).
- **Three items no model may settle — a Michigan family-law attorney must clear them before
  publication:** (1) the vicarious-consent recording question; (2) every Genesee County local
  fact; (3) any template a reader might mistake for filing-ready.

## Constraints that bind any drafting pass

- No template may be labeled "filing-ready." Every sample states the procedural posture and
  facts it assumes, plus a "do not use if" list.
- Never publish a competing unofficial version of an official SCAO or Genesee form — link the
  official form and supply drafting aids for the narrative attachments it does not generate.
- Clinical vocabulary ("narcissist," "alienator," "PAS") is banned from reader-facing templates.
  Conduct is pleaded under MCL 722.23(j) and MCL 722.27a(7), not as a syndrome.
- No exercise may ask the reader to interview, recruit, coach, diagnose, or emotionally rely on
  the child (acceptance test 8).
- Legal information, not legal advice — informational framing throughout, never imperative.

# Additional legal capability discovery — 2026-09-13

Bounded names/manifests inventory and selected static source review across Claude, Codex and .agents locations. No installs, enabling, configuration changes, model calls, source processing or skill execution occurred. This is a reusable-capability inventory, not current legal-source verification. Presence on disk does not establish enabled/callable status.

## Coverage and exclusions

Searched:
- C:/Users/matts/.claude/local-plugins
- C:/Users/matts/.claude/skills
- C:/Users/matts/.codex/plugins/cache
- C:/Users/matts/.codex/skills
- C:/Users/matts/.agents/skills

First pass enumerated relevant names/manifests and SKILL paths. Expanded unique legal/family/custody/evidence/drafting/citation/IRAC/library candidates. Found standalone .agents/skills/behavioral-pattern-analyzer; no specifically legal standalone skill emerged in the bounded .codex/skills name/content search. Generic technical skills matching incidental “legal” language were not expanded. Stale/retired paths were identified but not counted as active capabilities. No private corpora or secret/runtime files were read.

The current family-court-toolkit and family-court-toolkit-codex baseline is covered by the parallel stack lane. Pointers only:
- C:/Users/matts/.claude/local-plugins/plugins/family-court-toolkit
- C:/Users/matts/.codex/plugins/cache/custody-guide-codex-local/family-court-toolkit-codex/2.0.0

## Highest-value additions

The separate **law 1.1.0** plugin declares 19 first-class skills plus its index, explicitly for generic legal work outside the owner's custody case (manifest at .claude/local-plugins/plugins/law/.claude-plugin/plugin.json:2-4). Its most transferable material is a **fourteen-item legal-quality checklist**, a **claim/citation hallucination taxonomy**, **draft skeleton/field validation**, **tables of authorities**, and **plain-English contract explanation**.

The quality tool's limits matter: cite_check.py:278-318 mechanically evaluates only two citation-adjacency items; remaining checks return manual. Its overall status must not be promoted into “all legal checks passed” while manual checks remain. The skill preflight cites scripts/validate_legal_opinion.py, but no such file was found within this law plugin. The checklist also retains a “13-item” heading while enumerating fourteen items, and includes Korean-language instructions for operative-language verification. Preserve methods, normalize only in a future reviewed adaptation.

The package also contains additional graph-related code: law/scripts/entities.py produces entity/co-occurrence outputs using spaCy when available and regex otherwise; map-entities is the document-relationship visualization skill. This is a new outside-project lead relevant to the user's graph recollection, but co-occurrence output is not an established relationship and no local NLP/model execution was performed.

## Capability details

### law-quality: law 1.1.0

- Source: `C:/Users/matts/.claude/local-plugins/plugins/law/skills/legal-quality-checker/SKILL.md`
- Jurisdiction: general legal; jurisdiction must be selected.
- Intended use: Independent reviewer.
- Reusable method: Fourteen research checks: primary support, hierarchy, currency, counterarguments, summary consistency, source laundering and quoted operative language.
- Evidence: SKILL.md:14 preflight; :25 checklist; scripts/cite_check.py:278-318 mechanical/manual split.
- Flags: citation heuristic only; other checks manual; cited validate_legal_opinion.py not found within law plugin.

### law-hallucinations: law 1.1.0

- Source: `C:/Users/matts/.claude/local-plugins/plugins/law/skills/hallucination-taxonomy/SKILL.md`
- Jurisdiction: general legal.
- Intended use: Citation reviewer.
- Reusable method: Classify mismatch, paragraph hallucination, fabricated quotation, distorted paraphrase, nonexistent authority and malformed citation per claim-citation pair; unavailable public source stays unverifiable rather than nonexistent.
- Evidence: SKILL.md:7-87.
- Flags: Source retrieval and verification required; taxonomy is not factual truth.

### law-draft: law 1.1.0

- Source: `C:/Users/matts/.claude/local-plugins/plugins/law/skills/law/scripts/draft.py`
- Jurisdiction: generic US family/estate; forum-specific review needed.
- Intended use: Draft assistant plus owner reviewer.
- Reusable method: Template discovery, required-field extraction, facts validation and working-draft Markdown skeleton.
- Evidence: draft.py:147,156,183,212,256; law/SKILL.md domain tools.
- Flags: Skeleton only; no filed or court-ready certification.

### law-appellate: law 1.1.0

- Source: `C:/Users/matts/.claude/local-plugins/plugins/law/skills/appellate-formatting/SKILL.md`
- Jurisdiction: federal appellate plus state variations.
- Intended use: Formatting clerk/reviewer.
- Reusable method: Preserve exact headings, derive TOC/TOA, expose word-count exclusions and unknown pagination.
- Evidence: SKILL.md:14-20,44-48; law/scripts/toa.py.
- Flags: FRAP-based patterns cannot silently become Michigan trial-court requirements.

### law-irac: law 1.1.0

- Source: `C:/Users/matts/.claude/local-plugins/plugins/law/skills/irac-practice/SKILL.md`
- Jurisdiction: education; jurisdiction-specific rule review.
- Intended use: Independent educational evaluator.
- Reusable method: Grade issue spotting, rule accuracy, application depth, organization; top three fixes; does not rewrite original answer.
- Evidence: SKILL.md:3,10-15.
- Flags: Teaching workflow with student-tracker mutation instruction; adaptation needed for app review records.

### law-plain: law 1.1.0

- Source: `C:/Users/matts/.claude/local-plugins/plugins/law/skills/plain-english/SKILL.md`
- Jurisdiction: contracts, general.
- Intended use: Human reading support/translator.
- Reusable method: Clause-by-clause original-to-plain-English explanation, confusing-language flags, obligations and restrictions summary.
- Evidence: SKILL.md:18,48-85.
- Flags: Reverse direction from requested original-voice-to-court-language; preserve as complementary method.

### law-entities: law 1.1.0

- Source: `C:/Users/matts/.claude/local-plugins/plugins/law/skills/map-entities/SKILL.md`
- Jurisdiction: general document analysis.
- Intended use: Researcher/entity graph assistant.
- Reusable method: Entity extraction and co-occurrence mapping; source-preserving output and optional redaction.
- Evidence: law/scripts/entities.py:3,14,134,185,194; map-entities/scripts/map_entities.py.
- Flags: Local spaCy model/fallback code present; not executed; co-occurrence is not proven relationship; keep evidence authority upstream.

### law-kb: law 1.1.0

- Source: `C:/Users/matts/.claude/local-plugins/plugins/law/skills/lexwiki-legal-kb/SKILL.md`
- Jurisdiction: generic legal knowledge library.
- Intended use: Librarian.
- Reusable method: Index/search structured reference material; available fallback implementation uses SQLite FTS5 and optional LexWiki passthrough.
- Evidence: SKILL.md:46-60; law/scripts/kb.py:77,124,173.
- Flags: Named LexWiki MCP surface described as unavailable in source; current registration not probed.

### law-cited: law 1.1.0

- Source: `C:/Users/matts/.claude/local-plugins/plugins/law/skills/cited-research/SKILL.md`
- Jurisdiction: generic research.
- Intended use: Research workflow.
- Reusable method: Acquire sources, materialize manifest, NotebookLM synthesis, import Markdown and validate outputs.
- Evidence: SKILL.md:38-56.
- Flags: Requires separately reviewed connectors; no notebook/source upload performed.

### law-specialized: law 1.1.0

- Source: `C:/Users/matts/.claude/local-plugins/plugins/law/skills/family-law-summons/SKILL.md`
- Jurisdiction: generic US family/estate; QDRO federal retirement law.
- Intended use: Specialized draft assistant.
- Reusable method: Summons, dissolution petition/response, paternity petition, guardianship nomination/petition, elder summary, QDRO, appellate mandate and precedent reference exist.
- Evidence: law/.claude-plugin/plugin.json:2-4; individual SKILL frontmatter and prerequisites.
- Flags: Not Michigan custody defaults; generic deadline examples and court-ready wording require verification.

### legal-legacy: legal 0.1.0-test

- Source: `C:/Users/matts/.codex/plugins/cache/legal/legal/skills/lc-case-research/SKILL.md`
- Jurisdiction: Michigan.
- Intended use: Researcher; resource librarian.
- Reusable method: Frame issue, primary source search, citation existence/currentness, opposing authority, structured results; six lc skills cover factors/court/CPS/DV/FOC.
- Evidence: plugin.json:2-4; SKILL.md:12-63.
- Flags: Legacy proof-of-pattern; cache presence is not enabled/callable proof.

### agents-behavior: .agents standalone unversioned

- Source: `C:/Users/matts/.agents/skills/behavioral-pattern-analyzer/SKILL.md`
- Jurisdiction: Michigan-framed behavioral/custody analysis.
- Intended use: Private risk/red-team reviewer.
- Reusable method: Both-side analysis, observed conduct categories, trigger/escalation/reaction sequence and anticipated accusation/rebuttal prompts.
- Evidence: SKILL.md:8-75.
- Flags: Variant differs from toolkit copy; psychological/causal claims unverified; unsuitable as automatic findings or diagnoses.

### casebible: case-bible 0.6.1

- Source: `C:/Users/matts/.codex/plugins/cache/casebible-local/case-bible/0.6.1/skills/case-bible-custody/SKILL.md`
- Jurisdiction: technical provenance; evidence-platform boundary.
- Intended use: Evidence technician upstream.
- Reusable method: Versioned H1/H2/H3 recipes, verification, no in-place recipe change; adjacent forensics hash/fuzzy/string tools.
- Evidence: custody/SKILL.md:13-35; case-bible-forensics/SKILL.md:13-46.
- Flags: Do not duplicate evidence custody in Advocatio.

### memo-template: openai-templates 0.1.1

- Source: `C:/Users/matts/.codex/plugins/cache/openai-curated-remote/openai-templates/0.1.1/skills/artifact-template-legal-memorandum/SKILL.md`
- Jurisdiction: general memo format.
- Intended use: Draft/export clerk.
- Reusable method: Issue, brief answer, facts, analysis and conclusion using retained DOCX reference; preserve styles, render/verify output.
- Evidence: SKILL.md:3,11-23; artifact-template.json:4.
- Flags: Formatting resource, no substantive authority validation.

### research-start: .claude standalone unversioned

- Source: `C:/Users/matts/.claude/skills/research-start/SKILL.md`
- Jurisdiction: Chinese legal clinic/PRC databases.
- Intended use: Educational research planner.
- Reusable method: Seed-file-first scoping and gap roadmap; explicitly distinguish research leads from authority.
- Evidence: SKILL.md:3-6,32-45.
- Flags: Different jurisdiction; adapt method only, not databases/rules.

### finance-land: finance 1.1.0

- Source: `C:/Users/matts/.claude/local-plugins/plugins/finance/skills/michigan-land-contract/SKILL.md`
- Jurisdiction: Michigan real estate.
- Intended use: Specialized document reviewer.
- Reusable method: Structured prerequisites, estate seller authority, finance terms and verification markers.
- Evidence: SKILL.md:3,14,27,39; finance plugin.json:2-3.
- Flags: Adjacent specialty; not custody workflow.

## Duplicates and variants actually checked

SHA-256 checks apply to individual files, not entire plugin payloads:

| Files | Result |
|---|---|
| .codex/plugins/cache/legal/legal/skills/lc-case-research/SKILL.md and .codex/plugins/cache/casebible-local/legal/0.1.0-test/skills/lc-case-research/SKILL.md | Exact match: A0DC62FB145CE529EF74BBF81A58E96EA67A7AED3ACB9FAC43F184D66187A40E |
| .claude/local-plugins/plugins/case-bible/skills/case-bible-custody/SKILL.md and .codex/plugins/cache/casebible-local/case-bible/0.6.1/skills/case-bible-custody/SKILL.md | Exact match: 15C6546205A1A3F7381BE346C19BC8775E32B555988221B9863A4C57FB06B814 |
| .agents/skills/behavioral-pattern-analyzer/SKILL.md and toolkit's skills/behavioral-pattern-analyzer/SKILL.md | Different: 32D9F36258013E8013F00C673A652388D255B6431BADA2D60E768E3EC5293E1D vs 723B412EC8850C6B85223CAFC02036B8FFE95FFB6C2B83CCCC12B4EA17D4BA2B |

The legal 0.1.0-test manifest calls itself a proof-of-pattern Michigan plugin. Its six lc-prefixed skills are court resources, case research, best-interest factors, CPS, DV and FOC resources. Retired variants also appear in .claude/local-plugins/plugins/_stale. Cache copies and retired provenance should be reconciled before any future consolidation; this audit did not move or merge anything.

## Proposed use in the Advocatio plan

1. **Versioned resource library:** register each selected skill with source path, package version, hash, jurisdiction, intended role and review status. Full skills remain external source material until explicitly adapted.
2. **Independent review records:** turn quality checklist/taxonomy items into separate results tied to exact draft and source versions. Use concise inline flags: missing source, unverified quotation, manual review, contrary authority.
3. **Draft creation:** use field/prerequisite extraction and skeleton methods as input contracts, not a claim of legal correctness. Keep original user wording and proposed generated wording distinct.
4. **Two translation directions:** law:plain-english explains legalese to the user. The toolkit's court-language capability, owned by the parallel lane, addresses the other direction. They are complementary.
5. **Research/librarian roles:** preserve source-manifest and primary-source/counterauthority methods; avoid silently importing NotebookLM, LexWiki or a separate SQLite knowledge store into the current application.
6. **Risk/behavior analysis:** retain both-side observed-conduct and chronology prompts as private hypotheses. Do not turn psychological labels, assumed provocation or causation into accepted findings.
7. **Evidence technical skills:** custody/forensics remain upstream services whose verified outputs Advocatio references.
8. **Specialty boundaries:** federal appellate, QDRO, generic summons, Chinese legal-clinic and Michigan real-estate skills need their own applicability gates; they are not Michigan custody filing defaults.

All 16 records, paths, evidence and duplicate hashes are saved in legal-capabilities.json. Existing source content and external configuration remain unchanged.

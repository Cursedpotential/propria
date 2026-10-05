---
title: "law"
type: tool-reference
status: SOURCE_VERIFIED
date: 2026-10-04
generated_by: "Codex / GPT-6"
revision: 1
tags: [propria, tools, wiki]
---

# law

> _Byline: Codex · GPT-6 · 2026-10-04 — generated from the cited sources._

Generic legal work outside the owner's custody case: family/estate drafting templates, appellate tables, citation QA and research quality gates, legal document analysis (entities, plain-English), and a legal knowledge-base wrapper — with one exposed tool per domain. One progressive-disclosure entry skill (`law:law`) routing to 19 member skills loaded on demand.

Source: `E:/AI_Workspace/plugins/plugins/law`. Version: `1.1.0`.
Registered: `True`. Installed manifests: not found in inspected manifests.

## How to invoke it

Slash commands run inside an agent app. Terminal commands run in PowerShell. MCP tools require an attached server and are called by the agent or an MCP client. A skill is an instruction package, not a standalone executable.

Descriptions below are extracted from source metadata/docstrings. This is discovery evidence, not a claim every service invocation passed.

## Commands

No entries found in the inspected declarations.

## Skills

### `law-appellate-formatting`

(law) Generates appellate-filing-ready Tables of Contents, Tables of Authorities, and Certificates of Compliance under FRAP 32(g) and related rules. Produces defensible word-count calculations with transparent exclusions and anti-hallucination guardrails for pagination and citations. Covers federal circuits, state appellate courts, and U.S. Supreme Court variations. Use when building TOC, TOA, compliance certificates, formatting briefs for filing, or when the user mentions word count calculation, FRAP 32 compliance, appellate brief assembly, or page numbering.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/law/skills/appellate-formatting/SKILL.md:1>) · SHA-256 `89a39963e7c666a7e29b4f725ddeb0b9e1eb5fbac38123765988d8bea484df60`

### `law-appellate-mandate`

(law) Drafts formal appellate mandates that conclude the appeal process and direct trial courts to implement appellate decisions. Extracts disposition language, remand directives, and procedural history from appellate records to construct jurisdiction-specific mandate orders. Use when preparing mandate orders, returning jurisdiction to lower courts, or formalizing appellate dispositions after ruling.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/law/skills/appellate-mandate/SKILL.md:1>) · SHA-256 `b3e3c22d97393b489996aed7ec15e74cd744f4223cee32faa82a3e7fb18fe69b`

### `law-cited-research`

(law) Build a grounded research loop from source acquisition through NotebookLM synthesis into local markdown/CMS projections. Invoke with /workflow cited-research.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/law/skills/cited-research/SKILL.md:1>) · SHA-256 `4daac1a70aea6da7a2f82207f3eb9f7a5bc5fe90e9005234952004c81e4c8fa9`

### `law-dissolution-petition`

(law) Drafts a Petition for Dissolution of Marriage for filing in US state family courts. Covers jurisdictional standing, grounds, child custody and support, property and debt division, spousal support, and prayer for relief. Use when preparing initial divorce petitions, dissolution pleadings, or family law filing documents.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/law/skills/dissolution-petition/SKILL.md:1>) · SHA-256 `051c7b11c2aad2d5aa1ec54a6381b59c0f67beb8bd5a587276880978dec166d5`

### `law-elder-law-summary`

(law) Generates structured elder law summaries covering estate planning, elder abuse, healthcare rights, Medicaid eligibility, and guardianship with prioritized action plans. Triggers when the user requests an elderly client matter summary, elder care legal overview, long-term care planning review, or guardianship assessment.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/law/skills/elder-law-summary/SKILL.md:1>) · SHA-256 `1eb0c1234e91cd1795a91ee567df0e0136eda7cfaf7429c55f51c6cc5f752726`

### `law-family-law-summons`

(law) Drafts procedurally compliant family law summons for dissolution, custody modification, support enforcement, and other domestic proceedings. Covers jurisdiction-specific formatting, mandatory statutory warnings (ATROs), response deadlines adjusted by service method, service of process instructions, and proof of service sections. Use when initiating family law litigation, preparing summons for service, or drafting notice documents for domestic relations cases.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/law/skills/family-law-summons/SKILL.md:1>) · SHA-256 `eda14f9cb70d7f2517c3c2796d272402a016ca1cf001918eb889a93fcfa75b57`

### `law-guardianship-nomination`

(law) Drafts jurisdiction-specific Nomination of Guardian for Minor Children documents for estate planning. Gathers parent and child details, applies state guardianship statutes and execution formalities, and produces a professionally formatted nomination designating primary and alternate guardians. Use when drafting guardian nominations, minor guardianship documents, parental nomination of guardians, or estate planning for children.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/law/skills/guardianship-nomination/SKILL.md:1>) · SHA-256 `9a3b98cc228486a0cb26052da07eeb97bb67528ab501d87d641460d436161128`

### `law-guardianship-petition`

(law) Drafts court-ready Petitions for Guardianship for US state courts, covering incapacity allegations, least-restrictive-alternative analysis, scope-of-authority requests, and statutory compliance. Use when drafting guardianship petitions, conservatorship filings, or petitions for appointment of guardian over incapacitated adults or minors.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/law/skills/guardianship-petition/SKILL.md:1>) · SHA-256 `3ef3469112af749954eca7ae415ffe0ed0aeb6678bcc90a2a59a5fbac4fb08cf`

### `law-hallucination-taxonomy`

(law) Define hallucination categories and evidence thresholds for legal citation errors

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/law/skills/hallucination-taxonomy/SKILL.md:1>) · SHA-256 `c6b092c68e085a9f086653b4057ec7989d928d6a16d45edb8098ffb15afb1738`

### `law-irac-practice`

(law) Grade an IRAC essay for structure, issue-spotting, rule accuracy, analysis depth, and organization. Does NOT rewrite the essay or show a model answer; tracks patterns across sessions. Use when the user says 'grade my IRAC', 'check my essay', or 'I wrote this, give me feedback'.

Arguments: `[paste essay OR path to draft OR --generate-hypo]`

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/law/skills/irac-practice/SKILL.md:1>) · SHA-256 `ff2bff179dd2693880916e13e63c6721b5785281de9d8426f3cb70bb4d63729a`

### `law`

(law) Generic legal work outside the owner's custody case: family/estate drafting templates, appellate tables, citation QA and research quality gates, legal document analysis (entities, plain-English), and a legal knowledge-base wrapper — with one exposed tool per domain. Entry point / router — read this first, then load one member from references/. Triggers: petition, summons, qdro, guardianship, elder law, appellate brief, table of authorities, citation check, precedent, stare decisis, legal research quality, plain english contract, entity extraction legal documents, lexwiki. Members: dissolution-petition, response-dissolution, family-law-summons, paternity-petition, qdro-draft, guardianship-petition, guardianship-nomination, elder-law-summary, appellate-formatting, appellate-mandate, precedent, cited-research, legal-quality-checker, hallucination-taxonomy, irac-practice,…

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/law/skills/law/SKILL.md:1>) · SHA-256 `c28de5a1bf46cd1e046b49fecce3021dce5488c8c20e4e6f8fa8b72f895c41ca`

### `law-legal-quality-checker`

(law) Run the 14-item legal research quality gate and decide pass/fail with remediation steps.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/law/skills/legal-quality-checker/SKILL.md:1>) · SHA-256 `cf2136097266e622cc8b6653e2b7edd6b0c3435552fd02f7ae0a3b6bd78fb6d7`

### `law-lexwiki-legal-kb`

(law) Organize, query, and analyze legal documents using the LexWiki knowledge base tools

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/law/skills/lexwiki-legal-kb/SKILL.md:1>) · SHA-256 `04927ab3ac10c88fdc18f48bf5568dc92ed41ad444b8b6ebb5f9d1fa47769220`

### `law-map-entities`

(law) Extract named entities from legal documents and map relationships between them using NLP. Processes PDF, DOCX, TXT, and MD files or directories of documents. Uses spaCy for named entity recognition to identify people, organizations, dates, monetary amounts, jurisdictions, and legal references, then builds interactive relationship graphs showing how entities connect across documents. Use when: (1) a user provides legal documents and asks to identify entities or map relationships, (2) a user says 'find all entities', 'map relationships', 'who is mentioned in these documents', 'extract names and dates', or 'analyze entity connections', (3) any legal analysis task requiring entity extraction, relationship mapping, or cross-document entity tracking, (4) a user needs to understand which people, organizations, and dates appear across a set of legal documents.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/law/skills/map-entities/SKILL.md:1>) · SHA-256 `4b7881e7eecdf8d7db8770f215ac124336a2693a8baa3678b3e831ac06c39033`

### `law-paternity-petition`

(law) Drafts a Petition to Establish Paternity for family court filings. Covers court captions, standing allegations, factual bases, and relief requests with state-specific statutory requirements. Use when initiating paternity actions, child support petitions tied to parentage, custody filings requiring parentage determination, or birth certificate amendments.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/law/skills/paternity-petition/SKILL.md:1>) · SHA-256 `6d3139244601e029983089918e93b6ee091b5ae3434aff79fd403cf543caf9ee`

### `law-plain-english`

(law) Translates every clause of a contract into plain language at an 8th-grade reading level and flags deliberately confusing language patterns. Use when a user says 'explain this contract', 'what does this mean', or needs a non-lawyer to understand an agreement. Trigger with '/plain-english' or 'translate this contract to plain English'. '

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/law/skills/plain-english/SKILL.md:1>) · SHA-256 `4405491df531e491a06b53b5196b8ff24a3f3ebf214b258adf340588fcbf41e1`

### `law-precedent`

(law) Legal precedent reference — stare decisis, case law hierarchy, distinguishing, overruling, persuasive authority. Use when researching binding case law, analyzing judicial reasoning, or building legal arguments.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/law/skills/precedent/SKILL.md:1>) · SHA-256 `7760083bafe9e60bbe852bfa7e9d2a88c40608ff62cca02fdca062585e777bce`

### `law-qdro-draft`

(law) Drafts Qualified Domestic Relations Orders (QDROs) compliant with ERISA §206(d)(3) and IRC §414(p) to divide retirement benefits in divorce. Covers defined benefit pensions, 401(k)s, and defined contribution plans with plan-specific division formulas and alternate payee protections. Use when drafting QDROs, dividing retirement assets post-judgment, or preparing domestic relations orders for plan administrator review.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/law/skills/qdro-draft/SKILL.md:1>) · SHA-256 `8b8a73775fb86a68e492dfec3a7e44b51a5661da1b2972c9fca617a5b3e58f1d`

### `law-response-dissolution`

(law) Drafts a Response to Petition for Dissolution of Marriage addressing each allegation with admit/deny/lack-of-information responses and stating positions on custody, support, property, and fees. Triggers when user needs to respond to a divorce petition, file an answer to dissolution, or avoid default judgment in family law proceedings.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/law/skills/response-dissolution/SKILL.md:1>) · SHA-256 `24bedc950f87ae8b1c34523be42355432d98db74b46406d98802654b54d81d8a`

## Agents

No entries found in the inspected declarations.

## Cli Entries

No entries found in the inspected declarations.

## Scripts

### `skills/law/scripts/cite_check.py`

cite_check.py — citation existence QA against CourtListener (law plugin, cited-research domain).

    cite_check.py <file>          Extract case citations and check each against CourtListener v4
                                   search (existence only, never "good law"). Prints compact JSON.
    cite_check.py gate <file>     Run the 14-item quality gate from
                                   references/legal-quality-checker/SKILL.md as a checklist,
                                   pass/fail where mechanically checkable, "manual" otherwise.

Owner rule (2026-09-07): resolve COURTLISTENER_API_TOKEN from the environment first, else a
tolerant regex parse of ~/.secrets/*.env (never `source`, never print the value). The token is
optional — CourtListener's search endpoint works unauthenticated, just more rate-limited.

Respects rate limits: sleeps 1.1s between calls, checks at most 40 citations per run unless --all.

Python 3.12+ stdlib only. Prints compact JSON on stdout; errors go to stderr (exit 1).

```text
python "E:/AI_Workspace/plugins/plugins/law/skills/law/scripts/cite_check.py" --help
```

Declared arguments: `--all`, `--max`, `file`

Declared subcommands:

| Command | Source help |
|---|---|
| `check` | (default) check citations in a file against CourtListener |
| `gate` | run the 14-item legal research quality gate |

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| file | — | positional | — | — |
| --all | store_true | False | — | check every citation found (ignores --max) |
| --max | int | False | — | computed in source |
| file | — | positional | — | — |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [cite_check.py:1](<E:/AI_Workspace/plugins/plugins/law/skills/law/scripts/cite_check.py:1>) · SHA-256 `33e9ab638e5301bfb69faf573bb0f775ace08343916d6ccfc378bce822101ee5`

### `skills/law/scripts/draft.py`

draft.py — drafting-domain skeleton generator (law plugin).

Wraps the eight drafting member SKILL.md files as reusable templates. Every member has a
`## Prerequisites` section (a bulleted/numbered list of `**Term** — description` items) — that
is the one structural signal common to all eight, so it is the primary field source. Bracketed
placeholders (`[PARENT NAME(S)]`, `{{field}}`) found in the template body are collected too, as
a secondary, lower-confidence field source used only during render.

    draft.py templates
    draft.py fields <template>
    draft.py validate <template> facts.json
    draft.py render <template> facts.json [--out file.md]

`render` never claims filing-readiness: every output carries a fixed
"WORKING DRAFT — informational, not legal advice, not filing-ready" header, and any field
missing from facts.json is left in the output as `[VERIFY: field]`.

Python 3.12+ stdlib only. Prints compact JSON on stdout (except `render` without --out, which
prints the generated markdown itself); errors go to stderr (exit 1).

```text
python "E:/AI_Workspace/plugins/plugins/law/skills/law/scripts/draft.py" --help
```

Declared arguments: `--out`, `facts`, `template`

Declared subcommands:

| Command | Source help |
|---|---|
| `fields` | extract required facts from a template |
| `validate` | report missing/extra fields in a facts.json |
| `render` | render a markdown skeleton from facts.json |
| `templates` | list drafting templates |

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| template | — | positional | computed in source | — |
| template | — | positional | computed in source | — |
| facts | — | positional | — | — |
| template | — | positional | computed in source | — |
| facts | — | positional | — | — |
| --out | — | False | — | write markdown to this file instead of stdout |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [draft.py:1](<E:/AI_Workspace/plugins/plugins/law/skills/law/scripts/draft.py:1>) · SHA-256 `ad096729412ca92bc3f10459fac013bb35c4d520933c923f2b544dd0f205e93a`

### `skills/law/scripts/entities.py`

entities.py — legal document entity extraction and co-occurrence mapping (law plugin,
analysis domain / map-entities member).

Wraps references/map-entities/scripts/map_entities.py when spaCy + en_core_web_sm are
importable (checked at runtime, never assumed); otherwise falls back to a regex NER pass
covering dates, money, MCL/MCR/case cites (reusing toa.py's regexes), capitalized name
sequences, emails, and phone numbers.

    entities.py <file|dir> [--out json|md] [--graph out.json] [--redact]

Redaction (--redact) replaces PERSON entity display text with initials wherever it appears
in the output (entity list and co-occurrence edges) — never mutates the source file.

Supported formats: .txt, .md always; .docx via stdlib zipfile/xml (no python-docx needed);
.pdf via the `pdftotext` binary if present on PATH, else skipped with a note.

Python 3.12+ stdlib only. Prints compact JSON on stdout (or markdown with --out md);
errors go to stderr (exit 1).

```text
python "E:/AI_Workspace/plugins/plugins/law/skills/law/scripts/entities.py" --help
```

Declared arguments: `--graph`, `--out`, `--redact`, `path`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| path | — | positional | — | file or directory |
| --out | — | False | ['json', 'md'] | — |
| --graph | — | False | — | write co-occurrence graph (nodes+edges) to this JSON file |
| --redact | store_true | False | — | replace PERSON entities with initials |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [entities.py:1](<E:/AI_Workspace/plugins/plugins/law/skills/law/scripts/entities.py:1>) · SHA-256 `fc6dba203242c2d33ccb7bc30ef7c04d8c634959eca2d8faa24cb58da524a663`

### `skills/law/scripts/kb.py`

kb.py — legal knowledge-base wrapper (law plugin, lexwiki-legal-kb domain).

references/lexwiki-legal-kb/Skill.md documents an MCP-tool surface (lexwiki_search,
lexwiki_ingest, ...), not a local CLI — and no such CLI/MCP is installed on this machine
(checked via `shutil.which("lexwiki")` and env `LEXWIKI_CLI` at runtime, never assumed).
So kb.py stands on its own: a SQLite FTS5 full-text index over a directory of legal docs.

    kb.py index <dir> [--db path]      Build/rebuild a SQLite FTS5 index of .md/.txt/.pdf files
    kb.py search "<query>" [--limit N] Ranked snippets with file paths
    kb.py stats                        Row/doc counts, indexed root, db path and size
    kb.py lexwiki <args...>            Passthrough to a local LexWiki CLI, if one is ever
                                        installed; otherwise reports it is unavailable.

DB path: env LAW_KB_DB, else ~/.config/law/kb.sqlite (created on first index).
.pdf files are indexed via the `pdftotext` binary when present on PATH; skipped otherwise
(reported in the index summary, never silently dropped).

Python 3.12+ stdlib only. Prints compact JSON on stdout; errors go to stderr (exit 1).

```text
python "E:/AI_Workspace/plugins/plugins/law/skills/law/scripts/kb.py" --help
```

Declared arguments: `--db`, `--limit`, `dir`, `lexwiki_args`, `query`

Declared subcommands:

| Command | Source help |
|---|---|
| `index` | build/rebuild the FTS5 index over a directory |
| `search` | ranked full-text search |
| `stats` | index statistics |
| `lexwiki` | passthrough to a local LexWiki CLI, if installed |

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| dir | — | positional | — | — |
| --db | — | False | — | override the DB path (else env LAW_KB_DB, else ~/.config/law/kb.sqlite) |
| query | — | positional | — | — |
| --limit | int | False | — | — |
| --db | — | False | — | — |
| --db | — | False | — | — |
| lexwiki_args | — | positional | — | — |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [kb.py:1](<E:/AI_Workspace/plugins/plugins/law/skills/law/scripts/kb.py:1>) · SHA-256 `06dcae5ad40e38ca0ec3d385debd80006ca466f8dc1acc4f70fb47a1e8a49336`

### `skills/law/scripts/toa.py`

toa.py — appellate Table of Authorities extractor (law plugin, appellate-formatting domain).

Extracts case, statute, rule, and constitutional citations from a brief and groups them
into a Table of Authorities. Also supports a FRAP 32(g)-style word count that excludes
caption/TOA/TOC/certificate blocks marked with `<!-- exclude --> ... <!-- /exclude -->`.

Usage:
    toa.py <brief.md|.txt> [--format md|json]
    toa.py <brief.md|.txt> --wordcount

Python 3.12+ stdlib only. Prints compact JSON on stdout; errors go to stderr (exit 1).
Regexes here are reused by cite_check.py via `from toa import extract_citations`.

```text
python "E:/AI_Workspace/plugins/plugins/law/skills/law/scripts/toa.py" --help
```

Declared arguments: `--format`, `--wordcount`, `brief`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| brief | — | positional | — | Path to a .md or .txt brief |
| --format | — | False | ['md', 'json'] | Output shape (default json) |
| --wordcount | store_true | False | — | Print word count (excluding marked blocks) as JSON |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [toa.py:1](<E:/AI_Workspace/plugins/plugins/law/skills/law/scripts/toa.py:1>) · SHA-256 `628d0cdb1541bde69c1ea78fd37ac1d23a2ed86d544e804edbad4d39ce623a74`

### `skills/map-entities/scripts/check_dependencies.py`

Check and auto-install dependencies for the legal-entity-mapper skill.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [check_dependencies.py:1](<E:/AI_Workspace/plugins/plugins/law/skills/map-entities/scripts/check_dependencies.py:1>) · SHA-256 `ea7a75175ebef77517eff2e7046bbb37208e5320cdec997e14c5bf6c88b99cc1`

### `skills/map-entities/scripts/map_entities.py`

Entity & Relationship Mapper

Extracts named entities from legal documents using spaCy NLP and maps
relationships between them via co-occurrence analysis and network graphs.

Outputs JSON to stdout for Claude to parse. Progress/errors go to stderr.

```text
python "E:/AI_Workspace/plugins/plugins/law/skills/map-entities/scripts/map_entities.py" --help
```

Declared arguments: `--input`, `--min-mentions`, `--model`, `--output-dir`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --input | — | True | — | Document file or directory path |
| --output-dir | — | True | — | Output directory |
| --model | — | False | — | spaCy model name (default: en_core_web_sm) |
| --min-mentions | int | False | — | Minimum mentions to include an entity (default: 2) |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [map_entities.py:1](<E:/AI_Workspace/plugins/plugins/law/skills/map-entities/scripts/map_entities.py:1>) · SHA-256 `ddef4d128237bd0ee94353a03939c5f83a65ad730a0bebf90b8ebff1e2c28c7c`

## Mcp Tools

No entries found in the inspected declarations.

## Mcp Servers

No entries found in the inspected declarations.


Back to [[Code/wiki/plugin-inventory|Plugin inventory]].

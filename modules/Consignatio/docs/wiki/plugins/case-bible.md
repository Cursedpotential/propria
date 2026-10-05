---
title: "case-bible"
type: tool-reference
status: SOURCE_VERIFIED
date: 2026-10-04
generated_by: "Codex / GPT-6"
revision: 1
tags: [propria, tools, wiki]
---

# case-bible

> _Byline: Codex · GPT-6 · 2026-10-04 — generated from the cited sources._

Case Bible governed PostgreSQL catalog and verified B2 Parquet publication; current whole-bucket readers, source-preserving server execution, Temporal-tracked publication and custody tools.

Source: `E:/AI_Workspace/plugins/plugins/case-bible`. Version: `0.6.4`.
Registered: `True`. Installed manifests: .claude.

## How to invoke it

Slash commands run inside an agent app. Terminal commands run in PowerShell. MCP tools require an attached server and are called by the agent or an MCP client. A skill is an instruction package, not a standalone executable.

Descriptions below are extracted from source metadata/docstrings. This is discovery evidence, not a claim every service invocation passed.

## Commands

### `cb-chat-dedup`

AI-chat format detector + native-first dedup. `scan`: report-only detection of the 5 chat-export formats and their duplicate sets. `apply`: moves native re-exports into _re-exports/ and tags AI-chat md per tagging_spec.md (dry-run unless --go). `find`: read-only machine-wide sweep for chat-export files. `crosscheck`: read-only byte-identical dupe detection across multiple scan reports, with quarantine proposal and --same-drive scoping. --provider scopes scan/apply/find.

```text
/case-bible:cb-chat-dedup
```

Arguments: `scan|apply|find|crosscheck <folder|--roots R1 R2...|--from-scans DIR> [--provider chatgpt,gemini,claude,perplexity,opencode|all] [--same-drive] [--drives C,D] [--go] [--json out.json] [--md report.md]`

Source: [cb-chat-dedup.md:1](<E:/AI_Workspace/plugins/plugins/case-bible/commands/cb-chat-dedup.md:1>) · SHA-256 `2459b51d2d73c55c0624388aa32e832d92a1c14a255af976d54bc5b1e0f4f7e4`

### `cb-custody`

Verify or build the H1/H2/H3 custody hash chain on Case Bible evidence (files, message exports, new ingests).

```text
/case-bible:cb-custody
```

Source: [cb-custody.md:1](<E:/AI_Workspace/plugins/plugins/case-bible/commands/cb-custody.md:1>) · SHA-256 `658cb37aa607f509dc982df7aa78bd1e7b8648d310a8614b9cf5e112d1617def`

### `cb-init`

Build the new vault scaffold (8 v4 domains + governance files) in the local mirror, ready to sync up to casebible-sorted. Non-destructive (folders + INDEX/AGENTS/Dashboard only).

```text
/case-bible:cb-init
```

Arguments: `[--vault-root <path>] [--dry-run]`

Source: [cb-init.md:1](<E:/AI_Workspace/plugins/plugins/case-bible/commands/cb-init.md:1>) · SHA-256 `acf0a767124dccc22f9fdeb8ef5e0335c4177052d8ae28386a6861e9cf502932`

### `cb-lake`

Read the B2 lake publication and query the working Case Bible catalog.

```text
/case-bible:cb-lake
```

Source: [cb-lake.md:1](<E:/AI_Workspace/plugins/plugins/case-bible/commands/cb-lake.md:1>) · SHA-256 `e3ea02c50d284b7578054b42e92ce27417ea2c6d95926ed32837c01602df63d3`

### `cb-orchestrate`

Run the full Case Bible raw->sorted pipeline step by step, fanning out parallel sub-agent teams on the classify + verify passes. HITL gates before any copy/quarantine; never deletes.

```text
/case-bible:cb-orchestrate
```

Arguments: `[--from <pass>] [--scope chats_messaging|all]`

Source: [cb-orchestrate.md:1](<E:/AI_Workspace/plugins/plugins/case-bible/commands/cb-orchestrate.md:1>) · SHA-256 `911558cc0eda6a2701e9b4f7714d519924e7c7c398f3db33f53f07bd637705eb`

### `cb-quarantine`

Mimic delete SAFELY — move superseded duplicates / stale files into quarantine (casebible-quarantine/.review_hold), original folder structure preserved, fully reversible. Never hard-deletes.

```text
/case-bible:cb-quarantine
```

Arguments: `[--where "<sql>"] [--execute]`

Source: [cb-quarantine.md:1](<E:/AI_Workspace/plugins/plugins/case-bible/commands/cb-quarantine.md:1>) · SHA-256 `5dbcfc4a4fce3195fac839502dc8f7504c87ded537d6b48c41ab0e644ee8e674`

### `cb-r2-sort`

Build the raw->sorted provenance ledger (domain mapping + best-copy dedup) and optionally emit the rclone copy batch. Dry-run by default; never deletes.

```text
/case-bible:cb-r2-sort
```

Arguments: `[--all] [--execute]`

Source: [cb-r2-sort.md:1](<E:/AI_Workspace/plugins/plugins/case-bible/commands/cb-r2-sort.md:1>) · SHA-256 `18ee63af70fcf53107351a84c77128902bacb4cfc23cdb25a14e270ef47069ab`

### `cb-sort-assist`

Agent-assisted sort for ambiguous files — uses a cheap Claude (Haiku) programmatically to classify the needs_agent rows into the v4 taxonomy and write decisions back to the ledger. Dry-run by default.

```text
/case-bible:cb-sort-assist
```

Arguments: `[--apply] [--batch 15] [--limit N]`

Source: [cb-sort-assist.md:1](<E:/AI_Workspace/plugins/plugins/case-bible/commands/cb-sort-assist.md:1>) · SHA-256 `137a22d4198dbc6af24600d41eec74f032d1f801a463e7951b9d0491f56a7941`

### `cb-status`

Inspect current catalog and B2 publication generations without corpus scans.

```text
/case-bible:cb-status
```

Source: [cb-status.md:1](<E:/AI_Workspace/plugins/plugins/case-bible/commands/cb-status.md:1>) · SHA-256 `c7a497ed93a6f199ad3309e038a58e929199dfca59a53255f1491f436587bc99`

### `cb-sync`

Plan source-preserving transfer through the governed Case Bible workflow.

```text
/case-bible:cb-sync
```

Source: [cb-sync.md:1](<E:/AI_Workspace/plugins/plugins/case-bible/commands/cb-sync.md:1>) · SHA-256 `a72c43763b7232aa09fc7707120ae773c90ea53921321655547c9358696aa445`

### `cb-vsearch`

Search Case Bible files, messages and AI chats by meaning or keywords; filter and present source-linked results with in-memory DuckDB.

```text
/case-bible:cb-vsearch
```

Arguments: `"<query>" [--corpus all|documents|messages|chats] [--mode hybrid|keyword] [--k 8] [--source text] [--contains text] [--from YYYY-MM-DD] [--to YYYY-MM-DD] [--presentation human|compact|json]`

Source: [cb-vsearch.md:1](<E:/AI_Workspace/plugins/plugins/case-bible/commands/cb-vsearch.md:1>) · SHA-256 `9840110c2890060d44a64746198c7a289a06b78d181b74f050b6fdd6af2d9156`

## Skills

### `case-bible-architect`

Use this skill when designing, documenting, restructuring, or operating within the Case Bible vault. Covers domain routing, evidence handling, sidecar management, entity work, legal lane, platform symlinks, legacy salvage, archive mirroring, and all vault governance rules.


Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/case-bible-architect/SKILL.md:1>) · SHA-256 `8d661e32dde97404648555c0086f570dbb6569f2da3d9475c2f81725d9a4ab84`

### `case-bible-custody`

Multi-level custody hashing (H1 file / H2 per-message / H3 tamper-evident chain) for Case Bible evidence. Compute and verify hash chains on message exports, verify source files against custody rows, and build chains for new ingests. Use whenever evidence integrity, chain-of-custody, hash verification, or "was this tampered with" comes up.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/case-bible-custody/SKILL.md:1>) · SHA-256 `94f8f30650b3a5ce2333a293575a9e0db813161f76db896ee559dfd4ca2dd3b6`

### `case-bible-forensics`

Forensic operations for the Case Bible, providing cryptographic hashing, fuzzy hashing, string extraction, and fuzzy matching using Python wrappers.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/case-bible-forensics/SKILL.md:1>) · SHA-256 `c1f4b6a9d840cbb9ca4210c1a580e2f64c6b0ba5ec27a61e054dd26f16b63e93`

### `case-bible-lakehouse`

Query and refresh Case Bible lakehouse projections with verified source generation, coverage and freshness. Includes historical R2 Data Catalog / Iceberg coordinates that must be verified live. Use for corpus analytics, projection refresh or dashboard provenance checks.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/case-bible-lakehouse/SKILL.md:1>) · SHA-256 `91ebb315ae1f62235cee4577ea445913070c64cf567f5318452f794b03a06986`

### `case-bible-organizer`

Inventory, recovery review and organization of Case Bible source material into the coherent casevault hierarchy. Applies the shared consolidation policy, preserves all local copies unchanged, and routes structural placement through case-bible-architect. Use for intake, sort, audit, status or vault bootstrap review.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/case-bible-organizer/SKILL.md:1>) · SHA-256 `654a7e4b8decba6875d434a4ff5576064304498f0ecfb4778bb66f25e0562a0f`

### `case-bible-r2-sorter`

Review Case Bible corpus consolidation and physical routing into b2:salem-data/consignatio/casevault/, including historical R2 raw-to-sorted work. Requires streaming integrity, complete export units and independent date/metadata survivor criteria; preserves every local source unchanged. Use for corpus sort, dedup review or copy provenance ledgers.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/case-bible-r2-sorter/SKILL.md:1>) · SHA-256 `672e5f6785cbedecb5d09ee99f919b75b2f68a3b410998a6707574c3a84ba703`

### `cb-catalog`

(case-bible) "Is this already in the Case Bible?" — compares a local folder against THE catalog (PostgreSQL lakehouse casebible.raw_duck on ovh-files, the same one Intake reads) by SHA-1/MD5 + size or by name + size, read-only, no downloads. Also read-only SQL and corpus stats against that catalog. Use before adding files to the corpus, when deduping, or when asking what the corpus already holds.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/SKILL.md:1>) · SHA-256 `11595dd9ec0afb6b1ceb2f2a9e48dbe35ea0528770270bc1a95e4296f1f35eec`

### `cb-vsearch`

Search Case Bible content through the maintained server search tool.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-vsearch/SKILL.md:1>) · SHA-256 `3c5cf71c5ee7e75cf63bab36785f8a110d0aefd2624a80b0e33ec48dc9d26bc3`

### `mp-architecture-approaches`

Use when the planning-architect agent needs to evaluate architecture and implementation approaches for a codebase migration. Provides detailed descriptions, fit criteria, key artifacts, and migration-specific guidance for ADR, Specification-driven, DDD, Hexagonal, CQRS, Clean Architecture, Strangler Fig, and Microservices Decomposition.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/mp-architecture-approaches/SKILL.md:1>) · SHA-256 `1315a94d33f2b475f9cc8de5fa687544aa230fdbb52489f0ff0d70f88d228444`

### `mp-branch-thinking`

Use when a pass needs to explore multiple alternative solutions or approaches in parallel. Best for the planning architect, transcript analyzer, and any pass where multiple valid paths exist. Triggers on phrases like "explore alternatives", "consider options", "compare approaches", "what if".

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/mp-branch-thinking/SKILL.md:1>) · SHA-256 `c1466d27382140b896b5476ad4d374c9dc9dda130b7548e0dccb25e23ae23a75`

### `mp-codebase-scanning`

Use when an agent needs to walk, catalog, hash, or analyze a codebase directory tree. Provides patterns for manifesting files, identifying hot spots, and computing change-detecting hashes.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/mp-codebase-scanning/SKILL.md:1>) · SHA-256 `618745ca783364f389d75dda0ecdab412bff36f1adc1d0bbdf6a39611416c822`

### `mp-decision-making`

Use when the planning architect or any pass needs to choose among multiple approaches,
evaluate trade-offs, or make architecture decisions. Provides a route rubric matching
decision shapes to techniques (tribunal, adversarial, red-team, pre-mortem, council,
ladder-of-abstraction, calibration), structured verdict format, and chaining idioms.
Adapted from agent-foundry's decision-making plugin for migration planning contexts.


Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/mp-decision-making/SKILL.md:1>) · SHA-256 `15791c3f8c7635c3808ce402e443a60cd6934b8edadd4f3382bcef9411f8adc6`

### `mp-doc-patching`

Use when an agent needs to repair or update documentation. Provides rules for detecting stale references, missing coverage, intent mismatches, example drift, and parameter drift, plus risk assessment for doc changes.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/mp-doc-patching/SKILL.md:1>) · SHA-256 `60b691fe1e644eedc46a88c9b8d5c9bc262d885fe7ef139fd5b0850b3ee88385`

### `mp-handoff-protocol`

Use when working with migration-passes pipeline agents. Provides the standard handoff block, planned change block, and checkpoint block formats that all passes must use for structured communication between agents.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/mp-handoff-protocol/SKILL.md:1>) · SHA-256 `4036624ae093f18cb2cb3930144f957de419cebda63e0d3abadd8885cc72eda3`

### `mp-obsidian-export`

Use when exporting migration pipeline output to Obsidian-compatible markdown. Provides patterns for YAML frontmatter, wikilinks, map-of-content notes, and multi-format export.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/mp-obsidian-export/SKILL.md:1>) · SHA-256 `7d48b72191fc9f3d619add7417ebcfbd3be800c6a503400d5314b2522da63254`

### `mp-principles-extractor`

Use when the planning architect needs to extract project principles from codebase
patterns, conventions, and existing architecture for approach fit scoring. Adapts
the constitution plugin's checklist-generator pattern for migration contexts.
Extracts principles from code, configs, tests, and docs, then distills them into
actionable verification items with semantic deduplication.


Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/mp-principles-extractor/SKILL.md:1>) · SHA-256 `9a341d467ec846b95445e553dec62f183284e0666a96727e217aceb9c58362d4`

### `mp-react-pattern`

Use when a pass needs to iteratively reason about a problem, take action based on that reasoning, observe the result, and repeat. Best for the migration agent, doc patcher, and any pass where the agent must act and then evaluate the outcome before proceeding. Triggers on phrases like "iterate", "try and verify", "act and observe", "reason-act-observe".

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/mp-react-pattern/SKILL.md:1>) · SHA-256 `bf0b01e32d0ac7fd8ae24fbd64149cc4241a523da2a20b4b68d2935bac897080`

### `mp-report-format`

Use when generating or consuming JSON reports in .migration-passes/. Defines the
consistent schema for all pass reports including status fields, timestamps, error
reporting, cross-pass linking, and byline requirements. Every pass must follow this
format for interoperability between agents. Includes report templates for each pass
that agents should copy and fill in.


Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/mp-report-format/SKILL.md:1>) · SHA-256 `1e6ae09961b1bba0bc43dc6f348372e13da978a71348fa3d937ffbe31aa8c316`

### `mp-safe-operations`

Use when performing file modifications during migration, documentation patching, or any
pass that alters files. Provides backup-before-modify, atomic write, and rollback patterns
adapted from agent-foundry's atomic.sh for Windows-compatible Python. Invoke before any
Write or Edit operation that modifies codebase files to ensure safe rollback on failure.


Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/mp-safe-operations/SKILL.md:1>) · SHA-256 `91eb8b121c3434d8544cb2b42444fcd39235784ae2a1cf24372dc65ce17100c5`

### `mp-self-reflection`

Use when a pass needs to review and critique its own output before finalizing. Best for doc patching, gap auditing, and migration where quality control matters. Triggers on phrases like "review your work", "verify accuracy", "self-critique", "double-check".

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/mp-self-reflection/SKILL.md:1>) · SHA-256 `48c9420aa9e177aa61b615b1bedd56a4aa6bf957f049b3aaee03fbd0d2a3a6da`

### `mp-sequential-thinking`

Use when a pass needs to reason through complex problems step-by-step in a linear, ordered fashion. Best for inventory mapping, gap auditing, and other passes where order matters and each step depends on the previous one. Triggers on phrases like "analyze sequentially", "step by step", "chain of thought", "linear reasoning".

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/mp-sequential-thinking/SKILL.md:1>) · SHA-256 `649e19cb2bbf77449b203adda39400280a6fd3a83038deab18d1b2192e6fe665`

## Agents

### `cb-sort-orchestrator`

Orchestrates the Case Bible raw->sorted pipeline step by step, delegating each pass to tools/sub-agents and FANNING OUT PARALLEL TEAMS of sub-agents on the heavy passes (classify, verify) for speed. Validates handoffs, pauses for HITL approval before any write/copy, never deletes. Use for /cb-orchestrate or "orchestrate the sort", "run the sort pipeline", "fan out agents to sort the corpus". Modeled on the migration-passes pipeline-runner pattern.


Source: [cb-sort-orchestrator.md:1](<E:/AI_Workspace/plugins/plugins/case-bible/agents/cb-sort-orchestrator.md:1>) · SHA-256 `d0607be1dc3cbd83be340669548cc4ac72fd7dc43776c59cbeebae39ba3a277b`

## Cli Entries

No entries found in the inspected declarations.

## Scripts

### `hooks/block_delete.py`

CRITICAL SAFETY HOOK: Block all delete operations.

This hook intercepts any command that attempts to delete files or directories.
It blocks: rm, del, Remove-Item, shutil.rmtree, os.remove, os.unlink,
pathlib.Path.unlink, pathlib.Path.rmdir, send2trash, rd, rmdir, and
any operation targeting .review_hold (or legacy .to_be_deleted) contents for purging.

Exit codes:
  0 = allowed (not a delete operation)
  1 = BLOCKED (delete operation detected)

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [block_delete.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/hooks/block_delete.py:1>) · SHA-256 `bcc050d80b79382b11446ba348aa8691b132efbba0f6cf4cdfffe5fcb80e2f53`

### `hooks/block_overwrite.py`

CRITICAL SAFETY HOOK: Block content-destroying overwrite operations.

This hook intercepts commands that would destroy file contents without
technically "deleting" the file. Complements block_delete.py which
handles deletion commands.

Exit codes:
  0 = allowed (not an overwrite operation)
  5 = BLOCKED (overwrite operation detected)

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [block_overwrite.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/hooks/block_overwrite.py:1>) · SHA-256 `14e906dbc595a64ac88b5b38ad80a7dc00cd065c96ef0b8186ec206fcfb27b4b`

### `hooks/check_symlink.py`

SAFETY HOOK: Detect symlinks and NTFS junction points before modification.

Checks whether a target path or any of its parent directories is a symlink
or Windows NTFS junction point. This prevents agents from accidentally
modifying canonical source files through vault-visible aliases.

On Windows, detects both:
  - Directory symlinks (mklink /D) — caught by Path.is_symlink()
  - Junction points (mklink /J) — require FILE_ATTRIBUTE_REPARSE_POINT check

Exit codes:
  0 = safe (no symlinks/junctions detected, or --allow-read flag)
  4 = BLOCKED (symlink or junction detected on write target)

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [check_symlink.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/hooks/check_symlink.py:1>) · SHA-256 `3e6162baadf5b87baa9c856e0c36f63dfe9b236427273bc922f6b704e3d774ca`

### `hooks/log_action.py`

SAFETY HOOK: Log all operations to the audit trail.

Appends a timestamped, schema-compliant JSON entry for every operation.
Writes to both JSONL flat file and SQLite registry (if available).

Log location: discovered by walking up to find .case_bible/ directory.
Fallback: .agent_logs/ in current working directory.

This hook NEVER blocks — exit code is always 0.
Logging failures are swallowed silently to prevent blocking operations.

Exit codes:
  0 = always (logging never blocks)

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [log_action.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/hooks/log_action.py:1>) · SHA-256 `908e4458e5aa141bea6ddec2aa3fd374b5a9fcc50df69fcdff3360e1742763fc`

### `hooks/protect_review_hold.py`

SAFETY HOOK: Protect .review_hold (and legacy .to_be_deleted) quarantine areas.

This hook prevents agents from modifying, emptying, or purging quarantine
areas. Read-only operations are allowed; destructive operations are blocked.

Logic order (IMPORTANT — destructive check BEFORE read whitelist):
  1. If .review_hold/.to_be_deleted not referenced → allow (exit 0)
  2. If .review_hold/.to_be_deleted referenced AND destructive pattern found → BLOCK (exit 3)
  3. If .review_hold/.to_be_deleted referenced AND only read patterns → allow (exit 0)
  4. If .review_hold/.to_be_deleted referenced AND unknown operation → BLOCK (exit 3)

Exit codes:
  0 = allowed
  3 = BLOCKED (quarantine area modification attempted)

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [protect_review_hold.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/hooks/protect_review_hold.py:1>) · SHA-256 `0aa0721c71eb149767c0ad0010784e3cabefa80ff321855dbaf65094101a0d29`

### `hooks/require_explicit_go.py`

SAFETY HOOK: Require explicit human approval for non-trivial operations.

This hook implements a challenge-response approval mechanism. When triggered:
1. Generates a random 6-character nonce
2. Writes pending approval details to .case_bible/pending_approval.json
3. Prompts the human to enter the nonce
4. Validates the response
5. Logs the approval to the registry (if available)

The nonce expires after 5 minutes or one use.

Override:
  Setting CASE_BIBLE_GO=1 bypasses the nonce check but is LOGGED as
  'auto-approved (env override)' for audit purposes. This is intentional —
  it allows scripted batch operations while maintaining accountability.

Exit codes:
  0 = approved
  2 = BLOCKED (not approved)

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [require_explicit_go.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/hooks/require_explicit_go.py:1>) · SHA-256 `61c675b6d550e2928afd2a7c6b0bb7159681265d5137919956865763f18a8f01`

### `hooks/require_tracked_code.py`

SAFETY HOOK: no production writes from untracked (scratchpad/temp) code.

Root cause this prevents (2026-07-02): the pilot ingest's H2 hashing recipe ran
from a session scratchpad against live PG and the code died with the session —
the recipe is now unrecoverable. Rule: THE MOMENT CODE PRODUCES A DURABLE
ARTIFACT, THE CODE ITSELF IS PROVENANCE. Persist it (board specs/ or the repo)
BEFORE it runs, and run it from the persisted path.

Blocks a Bash command only when BOTH hold:
  1. a Temp/scratchpad path is being EXECUTED or fed as INPUT
     (python/bash/node <temp>, `-f <temp>`, `< <temp>`) — output redirects
     INTO temp (`> temp/x.csv`) stay allowed;
  2. the command shows prod-reach signals (fleet ssh, psql, docker exec,
     vector-store/graph ports or clients).

Override: CASE_BIBLE_TRACKED_OK=1 (for emergencies; use consciously).
Exit codes: 0 = allow, 2 = BLOCK (stderr shown to Claude).

Byline: Claude Code . Fable 5 . 2026-07-02

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [require_tracked_code.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/hooks/require_tracked_code.py:1>) · SHA-256 `0b8f7a0d742d7ca112363e9c5e5ffb585ec0c2dfba01c2b56740a21ac4e370cf`

### `preserved-disabled/case-bible-guard-0.6.0/cc_guard.py`

Claude Code-native PreToolUse safety guard for the Case Bible plugin.
Reads CC's JSON on stdin; exit 2 = BLOCK (stderr shown to Claude), exit 0 = allow.

Blocks ONLY filesystem hard-deletes and purges of protected quarantine areas.
Protected quarantine convention renamed .to_be_deleted -> .review_hold 2026-07-11
(owner: an agent once read "to be deleted" as an instruction). BOTH names stay
protected -- legacy .to_be_deleted dirs still exist (e.g. in casebible-quarantine).
Deliberately does NOT match SQL DELETE so normal data work is unaffected.
(OpenCode variants of these hooks live alongside in hooks/*.py.)
Byline: Claude Code - Opus 4.8 - 2026-06-23; renamed convention Fable 5 - 2026-07-11

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [cc_guard.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/preserved-disabled/case-bible-guard-0.6.0/cc_guard.py:1>) · SHA-256 `e6944bbeddbcb9f1d512975dfada5ab399dd1d424b8fcdecfe1bae4179aa09f1`

### `scripts/bump-revision.py`

bump-revision.py - Bump byline revision numbers in migration-passes files.

Adapted from agent-foundry's bump-version.sh pattern for semver-like revision
management in byline comments.

Usage:
    python bump-revision.py --patch <file>         # Rev: 2 -> Rev: 3
    python bump-revision.py --minor <file>         # Rev: 2 -> Rev: 3 (same as patch for bylines)
    python bump-revision.py --major <file>         # Rev: 2 -> Rev: 3 (same as patch for bylines)
    python bump-revision.py --set 5 <file>         # Rev: 2 -> Rev: 5
    python bump-revision.py --plugin                 # Bump plugin.json version
    python bump-revision.py --plugin --patch        # Bump plugin.json patch version
    python bump-revision.py --plugin --minor        # Bump plugin.json minor version
    python bump-revision.py --plugin --major        # Bump plugin.json major version

Byline format:
    <!-- Generated by: Claude (migration-passes/<command>) | Date: 2026-05-23 | Rev: 2 | Platform: Claude Code / win32 -->

```text
python "E:/AI_Workspace/plugins/plugins/case-bible/scripts/bump-revision.py" --help
```

Declared arguments: `--current`, `--major`, `--minor`, `--patch`, `--plugin`, `--set`, `files`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| files | — | positional | — | Files to bump revision in |
| --plugin | store_true | False | — | Bump plugin.json version instead |
| --patch | store_true | False | — | Bump patch version (default) |
| --minor | store_true | False | — | Bump minor version |
| --major | store_true | False | — | Bump major version |
| --set | int | False | — | Set exact revision number |
| --current | store_true | False | — | Show current revision without modifying |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [bump-revision.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/scripts/bump-revision.py:1>) · SHA-256 `375d53d677854bb9899b26659201dc02b952f8d51c7be1695c8e0db06a7302ec`

### `scripts/checkpoint-notify.py`

Notify about checkpoint progress in migration-passes pipeline.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [checkpoint-notify.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/scripts/checkpoint-notify.py:1>) · SHA-256 `13223417b42f16228087b22c5a7b6c7240e6f8d490f0d565f33028113220c2d1`

### `scripts/lib/formatting.py`

formatting.py - Terminal color and formatting utilities for migration-passes scripts.

Adapted from agent-foundry's colors.sh for Python/Windows compatibility.
Provides print_error, print_success, print_warning, print_info, print_section,
and JSON output mode.

Usage:
    from lib.formatting import print_error, print_success, print_warning, print_info, print_section

    # Or run directly for JSON output mode:
    python -m lib.formatting --json-start   # Enable JSON mode
    python -m lib.formatting --json-output   # Print collected JSON results

```text
python "E:/AI_Workspace/plugins/plugins/case-bible/scripts/lib/formatting.py" --help
```

Declared arguments: `--json-output`, `--json-start`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --json-start | store_true | False | — | Enable JSON mode |
| --json-output | store_true | False | — | Print collected JSON results |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [formatting.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/scripts/lib/formatting.py:1>) · SHA-256 `bfdcd56ea996014395337dcc0c13bb5bbfb3b0064a3dfd0cc7b20698cf27a48f`

### `scripts/manifest-diff.py`

Compare two inventory manifests to identify differences.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [manifest-diff.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/scripts/manifest-diff.py:1>) · SHA-256 `c7b6a219243c5fd7d325c21baeb9c521fa86fe0e7e7c4acd2634171641e76f6b`

### `scripts/pipeline-status.py`

pipeline-status.py - Display migration-passes pipeline status.

Shows which passes are complete, partial, blocked, or missing,
with timestamps and next-step guidance.

Usage:
    python pipeline-status.py [--json] [--verbose] [--dir <project_dir>]

```text
python "E:/AI_Workspace/plugins/plugins/case-bible/scripts/pipeline-status.py" --help
```

Declared arguments: `--dir`, `--json`, `--verbose`, `-v`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --json | store_true | False | — | Output as JSON |
| --verbose, -v | store_true | False | — | Show detailed information |
| --dir | — | False | — | Path to project directory containing .migration-passes/ |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [pipeline-status.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/scripts/pipeline-status.py:1>) · SHA-256 `f5db4710a6082b1a735aeef8d4958550b9d04dfd75ec5aaee09c2f687390ea88`

### `scripts/port-to-opencode.py`

port-to-opencode.py - Convert Claude Code plugin agents/commands to OpenCode format.

Usage:
    python port-to-opencode.py

This script reads agents/*.md and commands/*.md from the Claude plugin format
and writes them to .agents/agents/ and .agents/commands/ in OpenCode format.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [port-to-opencode.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/scripts/port-to-opencode.py:1>) · SHA-256 `1784d1cfd4f18db0b82cc2d02abbd7be80a43675b56fc58c685b1f700d8bf9cc`

### `scripts/restructure-claude-skills.py`

restructure-claude-skills.py - Group Claude plugin skills by label with prefixed names.

Claude Code spec:
- Skills go in skills/<name>/SKILL.md (subdirectories)
- name field is optional; defaults to directory name
- Must be lowercase, hyphens, digits; max 64 chars
- Plugin skills are auto-namespaced: /plugin-name:skill-name
- plugin.json "skills" field ADDS to default skills/ scan

This script:
1. Moves skills from skills/<name>/ to skills/migration-passes/mp-<name>/
2. Updates skill frontmatter names to mp-<name>
3. Updates skill cross-references in agents/ and commands/
4. Updates plugin.json with explicit skills path

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [restructure-claude-skills.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/scripts/restructure-claude-skills.py:1>) · SHA-256 `ccd3f51b01c75f593bfecac78f6d6befe2409ac09d0a0e7a62a117bbc4ed6af3`

### `scripts/restructure-skills.py`

restructure-skills.py - Group migration-passes skills by label with prefixed names.

Per OpenCode spec:
- Skills go in subfolders for organization
- name field must match the leaf directory name
- name must be lowercase, digits, hyphens only (max 64 chars)

This script:
1. Removes old flat skills from ~/.agents/skills/ and local .agents/skills/
2. Creates new structure: .agents/skills/migration-passes/mp-<name>/SKILL.md
3. Updates skill frontmatter names to mp-<name>
4. Updates skill cross-references in agents/commands
5. Also updates kimi-port/skills/

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [restructure-skills.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/scripts/restructure-skills.py:1>) · SHA-256 `28defe4c196db9764bad94e07676bb6e4cbf4783cc81e1be85a32853b2ccdd06`

### `scripts/safe-operations.py`

safe-operations.py - Safe file modification with backup, restore, and atomic writes.

Adapted from agent-foundry's atomic.sh pattern, ported to Python for Windows compatibility.

Usage:
    python safe-operations.py backup <file>
    python safe-operations.py restore <backup_file>
    python safe-operations.py remove-backup <backup_file>
    python safe-operations.py atomic-write <dest> <content>
    python safe-operations.py verify-backup <file>

```text
python "E:/AI_Workspace/plugins/plugins/case-bible/scripts/safe-operations.py" --help
```

Declared arguments: `backup_file`, `content`, `dest`, `file`

Declared subcommands:

| Command | Source help |
|---|---|
| `backup` | Create timestamped backup of a file |
| `restore` | Restore file from backup |
| `remove-backup` | Remove a backup file |
| `atomic-write` | Atomic write with backup |
| `verify-backup` | Verify backup exists and matches |

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| file | — | positional | — | File to back up |
| backup_file | — | positional | — | Backup file to restore from |
| backup_file | — | positional | — | Backup file to remove |
| dest | — | positional | — | Destination file path |
| content | — | positional | — | Content to write |
| file | — | positional | — | File to verify backup for |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [safe-operations.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/scripts/safe-operations.py:1>) · SHA-256 `95571c333234822e1074e572570c6a529b751313f065c49624ffb4550664816b`

### `scripts/validate-handoff.py`

Validate handoff block format and byline presence in migration-passes output files.

```text
python "E:/AI_Workspace/plugins/plugins/case-bible/scripts/validate-handoff.py" --help
```

Declared arguments: `--check-bylines`, `--file`, `--store`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --file | str | False | — | File to validate |
| --check-bylines | store_true | False | — | Check bylines in runtime store |
| --store | str | False | — | Runtime store directory |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [validate-handoff.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/scripts/validate-handoff.py:1>) · SHA-256 `392a6a9657fdeb71c84cb865586e11f4ab0da3f56390896d5c0539ff7e9c15f3`

### `scripts/validate-pipeline-state.py`

validate-pipeline-state.py - Validate migration-passes pipeline state before session end.

Checks .migration-passes/ for in-progress or incomplete passes. Reports pipeline
state and whether the session can safely end.

Used by the Stop hook to ensure passes complete in order.

Usage:
    python validate-pipeline-state.py [--dir <project_dir>]
    python validate-pipeline-state.py --check-complete <pass_number>

Exit codes:
    0 - Pipeline state is valid (all completed passes are consistent)
    1 - Pipeline state has issues (incomplete pass, out-of-order execution)

```text
python "E:/AI_Workspace/plugins/plugins/case-bible/scripts/validate-pipeline-state.py" --help
```

Declared arguments: `--check-complete`, `--dir`, `--json`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --dir | — | False | — | Path to project directory containing .migration-passes/ |
| --check-complete | int | False | — | Check if a specific pass has completed (0-7) |
| --json | store_true | False | — | Output as JSON |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [validate-pipeline-state.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/scripts/validate-pipeline-state.py:1>) · SHA-256 `6f1ed2c1c4475bce46c7ad3edafd0d436cb6e2409bd3b746187bedadb7c9b9e0`

### `scripts/validate-prerequisites.py`

validate-prerequisites.py - Validate that previous passes have completed before starting a new pass.

Checks .migration-passes/ for completed previous passes, validates handoff
integrity, and reports whether the current pass can proceed.

Usage:
    python validate-prerequisites.py <pass_number>
    python validate-prerequisites.py 5 --check   # Check only, don't block
    python validate-prerequisites.py 3 --verbose   # Show detailed status

Pass sequence:
    0: inventory.json
    1: context-summary.json
    2: transcript-intelligence.json
    3: doc-patch-report.json
    4: gap-audit-report.json
    5: migration-plan.md (Markdown, not JSON)
    6: migration-report.json
    7: export-report.json

```text
python "E:/AI_Workspace/plugins/plugins/case-bible/scripts/validate-prerequisites.py" --help
```

Declared arguments: `--check`, `--dir`, `--verbose`, `-v`, `pass_number`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| pass_number | int | positional | — | Pass number to validate prerequisites for (0-7) |
| --check | store_true | False | — | Check only, don't exit with error code |
| --verbose, -v | store_true | False | — | Show detailed status |
| --dir | — | False | — | Path to project directory containing .migration-passes/ |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [validate-prerequisites.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/scripts/validate-prerequisites.py:1>) · SHA-256 `ac73dc0c616b590163b608dedff3028a5765322fe36376f40ea991f696accaee`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/certifi/__main__.py`

Missing module docstring.

```text
python "E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/certifi/__main__.py" --help
```

Declared arguments: `--contents`, `-c`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| -c, --contents | store_true | False | — | — |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [__main__.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/certifi/__main__.py:1>) · SHA-256 `c410688fdd394d45812d118034e71fee88ba7beddd30fe1c1281bd3b232cd758`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/charset_normalizer/__main__.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [__main__.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/charset_normalizer/__main__.py:1>) · SHA-256 `dac8ff052e87d2c536e42d5b32acfd0d5c1aea438af657214846d253efe2bbb3`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/charset_normalizer/cli/__main__.py`

Missing module docstring.

```text
python "E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/charset_normalizer/cli/__main__.py" --help
```

Declared arguments: `--force`, `--minimal`, `--no-preemptive`, `--normalize`, `--replace`, `--threshold`, `--verbose`, `--version`, `--with-alternative`, `-a`, `-f`, `-i`, `-m`, `-n`, `-r`, `-t`, `-v`, `files`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| files | FileType('rb') | positional | — | File(s) to be analysed |
| -v, --verbose | store_true | False | — | Display complementary information about file if any. Stdout will contain logs about the detection process. |
| -a, --with-alternative | store_true | False | — | Output complementary possibilities if any. Top-level JSON WILL be a list. |
| -n, --normalize | store_true | False | — | Permit to normalize input file. If not set, program does not write anything. |
| -m, --minimal | store_true | False | — | Only output the charset detected to STDOUT. Disabling JSON output. |
| -r, --replace | store_true | False | — | Replace file when trying to normalize it instead of creating a new one. |
| -f, --force | store_true | False | — | Replace file without asking if you are sure, use this flag with caution. |
| -i, --no-preemptive | store_true | False | — | Disable looking at a charset declaration to hint the detector. |
| -t, --threshold | float | False | — | Define a custom maximum amount of noise allowed in decoded content. 0. <= noise <= 1. |
| --version | version | False | — | Show version information and exit. |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [__main__.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/charset_normalizer/cli/__main__.py:1>) · SHA-256 `f137ce27ba44dff512325aaecb38354e712f49676c863dd59b272c9ac7741090`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/duckdb/query_graph/__main__.py`

Missing module docstring.

```text
python "E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/duckdb/query_graph/__main__.py" --help
```

Declared arguments: `--open`, `--out`, `--profile_input`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --profile_input | — | False | — | profile input in json |
| --out | — | False | — | — |
| --open | store_true | False | — | — |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [__main__.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/duckdb/query_graph/__main__.py:1>) · SHA-256 `1341e76396b1800d13eef21d3c89c86c80b720c3c78eaa6547d33b1615fa9391`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/fsspec/fuse.py`

Missing module docstring.

```text
python "E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/fsspec/fuse.py" --help
```

Declared arguments: `--foreground`, `--log-file`, `--option`, `--ready-file`, `--threads`, `--version`, `-f`, `-l`, `-o`, `-r`, `-t`, `mount_point`, `source_path`, `url`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --version | version | False | — | — |
| url | str | positional | — | fs url |
| source_path | str | positional | — | source directory in fs |
| mount_point | str | positional | — | local directory |
| -o, --option | append | False | — | Any options of protocol included in the chained URL |
| -l, --log-file | str | False | — | Logging FUSE debug info (Default: '') |
| -f, --foreground | store_false | False | — | Running in foreground or not (Default: False) |
| -t, --threads | store_false | False | — | Running with threads support (Default: False) |
| -r, --ready-file | store_false | False | — | The '.fuse_ready' file will exist after FUSE is ready. (Debugging purpose, Default: False) |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [fuse.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/fsspec/fuse.py:1>) · SHA-256 `43edcd38ec8ba817d86b80dbe44d7dcff658dfacf3607b48b3598e51ab08b414`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/idna/__main__.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [__main__.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/idna/__main__.py:1>) · SHA-256 `e0930aeba5a3e2e2d94ca6c5fac4f72c0c4eb2be9bba283becf98e909091471c`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/idna/cli.py`

Command-line interface for the :mod:`idna` package.

Invoked via ``python -m idna``. See :func:`main` for the entry point.

```text
python "E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/idna/cli.py" --help
```

Declared arguments: `--decode`, `--encode`, `--strict`, `--version`, `-d`, `-e`, `domain`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| -e, --encode | store_const | False | — | Encode the input to its ASCII A-label form. |
| -d, --decode | store_const | False | — | Decode the input from its ASCII A-label form. |
| --strict | store_true | False | — | Disable the default UTS #46 mapping and apply IDNA 2008 rules verbatim. |
| --version | version | False | — | — |
| domain | — | positional | — | One or more domain names to convert. Omit to read from stdin. |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [cli.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/idna/cli.py:1>) · SHA-256 `b30a892cc35cf14ceceb42b388da5b5a71eeaa51b75914302526e8e27f310c0a`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/markdown_it/cli/parse.py`

CLI interface to markdown-it-py

Parse one or more markdown files, convert each to HTML, and print to stdout.

```text
python "E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/markdown_it/cli/parse.py" --help
```

Declared arguments: `--stdin`, `--version`, `-v`, `filenames`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| -v, --version | version | False | — | — |
| --stdin | store_true | False | — | read Markdown from standard input |
| filenames | — | positional | — | specify an optional list of files to convert |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [parse.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/markdown_it/cli/parse.py:1>) · SHA-256 `d941b1f5074f74c1a9a7a0ae0dd39523c04ecd0f66fe9812c73ace2a8e28425b`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/pygments/cmdline.py`

pygments.cmdline
~~~~~~~~~~~~~~~~

Command line interface.

:copyright: Copyright 2006-present by the Pygments team, see AUTHORS.
:license: BSD, see LICENSE for details.

```text
python "E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/pygments/cmdline.py" --help
```

Declared arguments: `--help`, `--json`, `-C`, `-F`, `-H`, `-L`, `-N`, `-O`, `-P`, `-S`, `-V`, `-a`, `-f`, `-g`, `-h`, `-l`, `-o`, `-s`, `-v`, `-x`, `INPUTFILE`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| -l | — | False | — | Specify the lexer to use. (Query names with -L.) If not given and -g is not present, the lexer is guessed from the filename. |
| -g | store_true | False | — | Guess the lexer from the file contents, or pass through as plain text if nothing can be guessed. |
| -F | append | False | — | Add a filter to the token stream. (Query names with -L.) Filter options are given after a colon if necessary. |
| -f | — | False | — | Specify the formatter to use. (Query names with -L.) If not given, the formatter is guessed from the output filename, and defaults to the terminal formatter if the output is to the terminal or an unknown file extension. |
| -O | append | False | — | Give options to the lexer and formatter as a comma-separated list of key-value pairs. Example: '-O bg=light,python=cool'. |
| -P | append | False | — | Give a single option to the lexer and formatter - with this you can pass options whose value contains commas and equal signs. Example: '-P "heading=Pygments, the Python highlighter"'. |
| -o | — | False | — | Where to write the output. Defaults to standard output. |
| INPUTFILE | — | positional | — | Where to read the input. Defaults to standard input. |
| -v | store_true | False | — | Print a detailed traceback on unhandled exceptions, which is useful for debugging and bug reports. |
| -s | store_true | False | — | Process lines one at a time until EOF, rather than waiting to process the entire file. This only works for stdin, only for lexers with no line-spanning constructs, and is intended for streaming input such as you get from 'tail -f'. Example usage: 'tail -f sql.log \| pygmentize -s -l sql'. |
| -x | store_true | False | — | Allow custom lexers and formatters to be loaded from a .py file relative to the current working directory. For example, '-l ./customlexer.py -x'. By default, this option expects a file with a class named CustomLexer or CustomFormatter; you can also specify your own class name with a colon ('-l ./lexer.py:MyLexer'). Users should be very careful not to use this option with untrusted files, because it will import and run them. |
| --json | store_true | False | — | Output as JSON. This can be only used in conjunction with -L. |
| -S | — | False | — | Print style definitions for STYLE for a formatter given with -f. The argument given by -a is formatter dependent. |
| -L | — | False | — | List lexers, formatters, styles or filters -- give additional arguments for the thing(s) you want to list (e.g. "styles"), or omit them to list everything. |
| -N | — | False | — | Guess and print out a lexer name based solely on the given filename. Does not take input or highlight anything. If no specific lexer can be determined, "text" is printed. |
| -C | store_true | False | — | Like -N, but print out a lexer name based solely on a given content from standard input. |
| -H | store | False | — | Print detailed help for the object <name> of type <type>, where <type> is one of "lexer", "formatter" or "filter". |
| -V | store_true | False | — | Print the package version. |
| -h, --help | store_true | False | — | Print this help. |
| -a | — | False | — | Formatter-specific additional argument for the -S (print style sheet) mode. |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [cmdline.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/pygments/cmdline.py:1>) · SHA-256 `fdd3a7aed6bfd8621ef0983ed58d296f9bd68bc2fa413889cf2a13b87ad8adb3`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/pygments/lexers/_cocoa_builtins.py`

pygments.lexers._cocoa_builtins
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

This file defines a set of types used across Cocoa frameworks from Apple.
There is a list of @interfaces, @protocols and some other (structs, unions)

File may be also used as standalone generator for above.

:copyright: Copyright 2006-present by the Pygments team, see AUTHORS.
:license: BSD, see LICENSE for details.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [_cocoa_builtins.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/pygments/lexers/_cocoa_builtins.py:1>) · SHA-256 `69beacabe8b2e4b6a913570d63297c3c95cb83a108357c7e94c45bc0445d298b`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/pygments/lexers/_lua_builtins.py`

pygments.lexers._lua_builtins
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

This file contains the names and modules of lua functions
It is able to re-generate itself, but for adding new functions you
probably have to add some callbacks (see function module_callbacks).

Do not edit the MODULES dict by hand.

Run with `python -I` to regenerate.

:copyright: Copyright 2006-present by the Pygments team, see AUTHORS.
:license: BSD, see LICENSE for details.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [_lua_builtins.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/pygments/lexers/_lua_builtins.py:1>) · SHA-256 `32c0d5f6c11b26780aa17499bb4c576b01af77a5e315fd76e67b9704da1f7dd5`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/pygments/lexers/_mysql_builtins.py`

pygments.lexers._mysql_builtins
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Self-updating data files for the MySQL lexer.

Run with `python -I` to update.

:copyright: Copyright 2006-present by the Pygments team, see AUTHORS.
:license: BSD, see LICENSE for details.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [_mysql_builtins.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/pygments/lexers/_mysql_builtins.py:1>) · SHA-256 `4aa895615562aed9cde254a0b5f7913f1b55a5cba68d4bef52af67f0939bb90e`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/pygments/lexers/_php_builtins.py`

pygments.lexers._php_builtins
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

This file loads the function names and their modules from the
php webpage and generates itself.

Run with `python -I` to regenerate.

:copyright: Copyright 2006-present by the Pygments team, see AUTHORS.
:license: BSD, see LICENSE for details.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [_php_builtins.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/pygments/lexers/_php_builtins.py:1>) · SHA-256 `f60f962d3dea6d21a8950241ff83d2e855bfafb1d55c15af239396c4ab5bc289`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/pygments/lexers/_postgres_builtins.py`

pygments.lexers._postgres_builtins
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Self-updating data files for PostgreSQL lexer.

Run with `python -I` to update itself.

:copyright: Copyright 2006-present by the Pygments team, see AUTHORS.
:license: BSD, see LICENSE for details.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [_postgres_builtins.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/pygments/lexers/_postgres_builtins.py:1>) · SHA-256 `4068369814d3860bf82dd5e3166e70aca68a9c253d0ee6e6262ea55c1ef2d8c8`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/pygments/lexers/_scilab_builtins.py`

pygments.lexers._scilab_builtins
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Builtin list for the ScilabLexer.

:copyright: Copyright 2006-present by the Pygments team, see AUTHORS.
:license: BSD, see LICENSE for details.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [_scilab_builtins.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/pygments/lexers/_scilab_builtins.py:1>) · SHA-256 `137dd1d2692735cfb9799f5482eeeeec0bffbd4c40690a0c4ed771dcc71ecb15`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/pygments/lexers/_sourcemod_builtins.py`

pygments.lexers._sourcemod_builtins
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

This file contains the names of SourceMod functions.

Do not edit the FUNCTIONS list by hand.

Run with `python -I` to regenerate.

:copyright: Copyright 2006-present by the Pygments team, see AUTHORS.
:license: BSD, see LICENSE for details.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [_sourcemod_builtins.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/pygments/lexers/_sourcemod_builtins.py:1>) · SHA-256 `f2ab292f108bb1f37d27b37449665136cd00faa24e20644100968d31e59a3814`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/pygments/unistring.py`

pygments.unistring
~~~~~~~~~~~~~~~~~~

Strings of all Unicode characters of a certain category.
Used for matching in Unicode-aware languages. Run to regenerate.

Inspired by chartypes_create.py from the MoinMoin project.

:copyright: Copyright 2006-present by the Pygments team, see AUTHORS.
:license: BSD, see LICENSE for details.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [unistring.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/pygments/unistring.py:1>) · SHA-256 `678c381df395521b9e09445191f840a90d9bfbaf54ce4858e31b877af604cb98`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/pyparsing/tools/cvt_pyparsing_pep8_names.py`

Missing module docstring.

```text
python "E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/pyparsing/tools/cvt_pyparsing_pep8_names.py" --help
```

Declared arguments: `--encoding`, `--exit-zero-even-if-changed`, `--update`, `--verbose`, `-exit0`, `-u`, `-v`, `-vv`, `source_filename`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --verbose, -v | store_true | False | — | Show unified diff for each source file |
| -vv | store_true | False | — | Show unified diff for each source file, plus names of scanned files with no changes |
| --update, -u | store_true | False | — | Update source files in-place |
| --encoding | str | False | — | Encoding of source files (default: utf-8) |
| --exit-zero-even-if-changed, -exit0 | store_true | False | — | Exit with status code 0 even if changes were made |
| source_filename | — | positional | — | Source filenames or filename patterns of Python files to be converted |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [cvt_pyparsing_pep8_names.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/pyparsing/tools/cvt_pyparsing_pep8_names.py:1>) · SHA-256 `1fc0281dd62ceeb53c6b4451178ce3bab8c6ff167ca4e2d5061022d3e16054e6`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/requests/certs.py`

requests.certs
~~~~~~~~~~~~~~

This module returns the preferred default CA certificate bundle. There is
only one — the one from the certifi package.

If you are packaging Requests, e.g., for a Linux distribution or a managed
environment, you can change the definition of where() to return a separately
packaged CA bundle.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [certs.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/requests/certs.py:1>) · SHA-256 `fd9c6b83359cef90ff6c4eeeab8dcc2388da382ebca7d00a499b3c0b434a87e4`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/requests/help.py`

Module containing bug report helper(s).

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [help.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/requests/help.py:1>) · SHA-256 `723519bb1884da18d84f6b2fb78f7ebf7fb57f732070b6025ae071dac6a2d179`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/__init__.py`

Rich text and beautiful formatting in the terminal.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [__init__.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/__init__.py:1>) · SHA-256 `8fb000f7fcff3c03506e6453c1391665cfc355daad0411f106a1ab32ccf6d422`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/__main__.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [__main__.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/__main__.py:1>) · SHA-256 `91ccbc06d04452460b0a8811930842749b3454dc7a285d67f24154da0fbba658`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/_log_render.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [_log_render.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/_log_render.py:1>) · SHA-256 `c41282c6a88ee05664f1e1b9e9ff1cac576b989c45ac9b1013757717e7c57a47`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/_ratio.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [_ratio.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/_ratio.py:1>) · SHA-256 `20eb65efcb1009866c987cb185ee3992b91bebcbbdc55cfbcc5b175490fe6f66`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/_win32_console.py`

Light wrapper around the Win32 Console API - this module should only be imported on Windows

The API that this module wraps is documented at https://docs.microsoft.com/en-us/windows/console/console-functions

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [_win32_console.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/_win32_console.py:1>) · SHA-256 `a3640dfc8471d746e218fdc0a759de6aa5fc1411a550a4a658ce8ac3839283e5`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/_windows.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [_windows.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/_windows.py:1>) · SHA-256 `8acdd6a5b1cc8fc59a4c7601d7585cea53f6b78865bede135624e51d29a3b22d`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/_wrap.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [_wrap.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/_wrap.py:1>) · SHA-256 `1654aca26e445f42d5900dca5b2df8c879c27cbb6a5fe6487a95ca87eef4ae97`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/abc.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [abc.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/abc.py:1>) · SHA-256 `7402cc3867ca54d7806efaaaeba2294d0c5051eaf10fb004e052b0a9dd1e40a9`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/align.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [align.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/align.py:1>) · SHA-256 `6bc31b3fe8898e8385a27c6a8283f76247ad28c594dd61596b18ea02fd907391`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/ansi.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [ansi.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/ansi.py:1>) · SHA-256 `02fb352c76d275cc8ebc339da442d952850b7018987b063be9e341a7ab85061b`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/box.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [box.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/box.py:1>) · SHA-256 `492a2583cfe9cc7cd8f50bc9428faaa74b5b3ec9e3f0eff65b88668b597e668d`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/color.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [color.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/color.py:1>) · SHA-256 `dc74942d50e3eea4245d47455afefc24e8926737f2e72d6791c6219dadbde95d`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/columns.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [columns.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/columns.py:1>) · SHA-256 `1d45f429c326f5db0a362d757d36e233f876883b65f3248269573195a944ceaf`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/console.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [console.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/console.py:1>) · SHA-256 `b2032877362a9e947e1e3f86901b111356e84a3fdfbec03f083ca4c24e8bd914`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/control.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [control.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/control.py:1>) · SHA-256 `1e7b2b6854f305a5100f3289597b1f3eff8f3e6806ca94a04afee800d69c92ab`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/default_styles.py`

Missing module docstring.

```text
python "E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/default_styles.py" --help
```

Declared arguments: `--html`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --html | store_true | False | — | Export as HTML table |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [default_styles.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/default_styles.py:1>) · SHA-256 `4e4d6967d6ce678d04557edc7c16d80c4a3b86ba7006f646b225025dd8743f43`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/diagnose.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [diagnose.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/diagnose.py:1>) · SHA-256 `d515a7428a693d78c2ff8f40078bed574e3c0cade4b104aaebbd42f3763a7fe8`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/emoji.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [emoji.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/emoji.py:1>) · SHA-256 `cfb001bb4ffad5624f58e33f7718541cd680d01e9142e657deb7a58ecd62584f`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/highlighter.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [highlighter.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/highlighter.py:1>) · SHA-256 `3086a95b08d1f1a85472c0e2720378c4b45b6a76e1e71fd0ce1e647dd0eb28a9`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/json.py`

Missing module docstring.

```text
python "E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/json.py" --help
```

Declared arguments: `--indent`, `-i`, `path`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| path | — | positional | — | path to file, or - for stdin |
| -i, --indent | int | False | — | Number of spaces in an indent |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [json.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/json.py:1>) · SHA-256 `a260b65874e0511c44a2c9dad5fb684890a77b6117ecc0ee42d09db304b7aac9`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/layout.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [layout.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/layout.py:1>) · SHA-256 `591f0f092ae8627b5e213df36b0c50de4dda775b103b9c061b54993a8781b813`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/live.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [live.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/live.py:1>) · SHA-256 `50abea2d2b92ccd1c8073e69c51ad4ed23ec9c6e3cbc9271d9330789a1e59a72`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/logging.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [logging.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/logging.py:1>) · SHA-256 `eef864eebfcb8564dcc2d7b28cb2cce4d7dcf37f5fc01b888884af059706c33a`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/markdown.py`

Missing module docstring.

```text
python "E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/markdown.py" --help
```

Declared arguments: `--code-theme`, `--force-color`, `--hyperlinks`, `--inline-code-lexer`, `--justify`, `--page`, `--width`, `-c`, `-i`, `-j`, `-p`, `-t`, `-w`, `-y`, `path`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| path | — | positional | — | path to markdown file, or - for stdin |
| -c, --force-color | store_true | False | — | force color for non-terminals |
| -t, --code-theme | — | False | — | pygments code theme |
| -i, --inline-code-lexer | — | False | — | inline_code_lexer |
| -y, --hyperlinks | store_true | False | — | enable hyperlinks |
| -w, --width | int | False | — | width of output (default will auto-detect) |
| -j, --justify | store_true | False | — | enable full text justify |
| -p, --page | store_true | False | — | use pager to scroll output |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [markdown.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/markdown.py:1>) · SHA-256 `9830c99946d64589bbfef29e9d5abbe8ae9a1b8dca4c9278b6260ccf15ceee7b`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/markup.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [markup.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/markup.py:1>) · SHA-256 `6eda6bdbbd412e18824758cd825467bf606923355c35e7d8c1231e5bdb5e0db7`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/padding.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [padding.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/padding.py:1>) · SHA-256 `87c5e7222bcbacdb65c48def40f2875e1e210308ce26a671d2c94cd33de0d7f3`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/pager.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [pager.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/pager.py:1>) · SHA-256 `48efc44c114a6e0de7fc080ecd79b8d52bf7e98c57032237fd1f8a398dbfb927`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/palette.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [palette.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/palette.py:1>) · SHA-256 `02be9952b607885b7af91af693e93d17c57b87180960735daa3936bd55ec2e07`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/panel.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [panel.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/panel.py:1>) · SHA-256 `f6c425d3484f22a1f91b68002d0a3835ea45c293f493dc13facfe03a6b39a487`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/pretty.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [pretty.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/pretty.py:1>) · SHA-256 `9715bf6e105f0d329efa0aedccfd4a42142ca8f6717d9375e2d8782515682f34`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/progress.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [progress.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/progress.py:1>) · SHA-256 `67a633ea960edba2c55f1db87172af9366c45fa1648cad99e5ed9fcb406defcb`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/progress_bar.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [progress_bar.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/progress_bar.py:1>) · SHA-256 `9994cfa4953071f71d8100934f3de4c98f9f73bf5d74bc2dc7a1a18717e8d3ae`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/prompt.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [prompt.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/prompt.py:1>) · SHA-256 `d5222e0447a5a2fbf0086db0a76d8f670b24037224980b0ec5fd2bb8590e84e9`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/repr.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [repr.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/repr.py:1>) · SHA-256 `9857b119356579d7ac662303de3041041a080a85a1f973c5014ab71eeee25f85`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/rule.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [rule.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/rule.py:1>) · SHA-256 `ba63b6d568f0d0571801e4c1dd4ba634b0ac0d685e8f3c678e57f65708978832`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/scope.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [scope.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/scope.py:1>) · SHA-256 `c7e8e428813e8fb67103d234ee5c9ae62540b8303250a339f075ad359414cd17`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/segment.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [segment.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/segment.py:1>) · SHA-256 `d7b240c386e6b15fcbac9f0f9e123d5c7d8368df9f3843c5fbc9a0c7b8e82c33`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/spinner.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [spinner.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/spinner.py:1>) · SHA-256 `a27221a4a9658d11e9a5365ab313bc91782d632087524a5280a825449de8e758`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/status.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [status.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/status.py:1>) · SHA-256 `9243e987761e019068f97fb8c0fa7c813a99c94e3ae8d2f06410383d94d37b0a`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/styled.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [styled.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/styled.py:1>) · SHA-256 `c258d5b154d76c004c319be4ce43b8dd9124ffe8abcc4b6f52243eb0d9e2910f`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/syntax.py`

Missing module docstring.

```text
python "E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/syntax.py" --help
```

Declared arguments: `--background-color`, `--force-color`, `--highlight-line`, `--indent-guides`, `--lexer`, `--line-numbers`, `--padding`, `--soft-wrap`, `--theme`, `--width`, `--wrap`, `-b`, `-c`, `-i`, `-l`, `-p`, `-r`, `-s`, `-t`, `-w`, `-x`, `path`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| path | — | positional | — | path to file, or - for stdin |
| -c, --force-color | store_true | False | — | force color for non-terminals |
| -i, --indent-guides | store_true | False | — | display indent guides |
| -l, --line-numbers | store_true | False | — | render line numbers |
| -w, --width | int | False | — | width of output (default will auto-detect) |
| -r, --wrap | store_true | False | — | word wrap long lines |
| -s, --soft-wrap | store_true | False | — | enable soft wrapping mode |
| -t, --theme | — | False | — | pygments theme |
| -b, --background-color | — | False | — | Override background color |
| -x, --lexer | — | False | — | Lexer name |
| -p, --padding | int | False | — | Padding |
| --highlight-line | int | False | — | The line number (not index!) to highlight |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [syntax.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/syntax.py:1>) · SHA-256 `cad81e194b7dd5f76bf0b68728e76468d31ec2b3d1c60a1c8b998c77730f1382`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/table.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [table.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/table.py:1>) · SHA-256 `eb2bfbc0c2d76603ac1a0cc40e9297a5e71740ec1c2c11cd3a7f8c61c6e8d599`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/text.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [text.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/text.py:1>) · SHA-256 `b7627c88caaec11d015148209be715369df45b1c3933c7c817fa748e2a18455d`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/theme.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [theme.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/theme.py:1>) · SHA-256 `6e261959dfd6bb5296f15b53ed4b13403ac353cf51c402b9fd5cf5cb7ced1982`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/traceback.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [traceback.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/traceback.py:1>) · SHA-256 `4d3d3778b2848f95daa57169f6481fd1b07036ea4816b1eab8ac59b70061cfff`

### `skills/cb-catalog/.venv-ice/Lib/site-packages/rich/tree.py`

Missing module docstring.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [tree.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/.venv-ice/Lib/site-packages/rich/tree.py:1>) · SHA-256 `4283b0838db816477019f47c2b4b59e90eeab7358d0143ff9b8b05698b86ea7c`

### `skills/cb-catalog/cbcat_compare_sql.py`

Build the read-only comparison SQL for `cbcat compare`.

Reads an `rclone lsjson --hash` listing of a local directory and prints SQL that checks each
file against the PostgreSQL catalog (`casebible.raw_duck`). Candidates travel inline as a VALUES
list because a read-only session may not create a temp table.

Usage: cbcat_compare_sql.py <lsjson file> <corpus table> <hash|name|both>

> Byline: Codex, 2026-10-04. Current inventory reader repair.
> Byline: Claude Code · Fable 5.1 · 2026-09-20

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [cbcat_compare_sql.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/skills/cb-catalog/cbcat_compare_sql.py:1>) · SHA-256 `adfbe104c1c6c9d250aee2e9a7d25cd7f461d62964a653200d3a58dea5332f77`

### `tools/case_bible_bootstrap.py`

Case Bible Bootstrap & Safe Operations Script

Commands:
  init               Create the vault structure from templates
  legacy-intake      Move existing files into Legacy/.to_be_sorted/
  manifest           Generate a hash manifest for a path
  copy-verify        Copy a file/tree and verify integrity
  quarantine-original Move verified original to .to_be_deleted/

This script is intentionally cautious:
  - No hard deletes ever
  - All operations logged
  - Copy-verify-quarantine is the standard flow
  - Legacy intake preserves everything, sorts nothing

```text
python "E:/AI_Workspace/plugins/plugins/case-bible/tools/case_bible_bootstrap.py" --help
```

Declared arguments: `--dry-run`, `--dst`, `--force`, `--log-root`, `--out`, `--package-root`, `--reason`, `--source`, `--src`, `--target`, `--vault-root`

Declared subcommands:

| Command | Source help |
|---|---|
| `init` | Create vault structure from templates |
| `legacy-intake` | Move existing files into Legacy/.to_be_sorted/ |
| `manifest` | Generate hash manifest for a path |
| `copy-verify` | Copy and verify integrity |
| `quarantine-original` | Move original to .to_be_deleted/ after verified copy |

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --vault-root | — | True | — | Path to create the vault |
| --package-root | — | True | — | Path to this package |
| --source | — | True | — | Path to existing files to intake |
| --vault-root | — | True | — | Path to the vault root |
| --reason | — | False | — | Reason for intake |
| --dry-run | store_true | False | — | Show what would happen without moving |
| --target | — | True | — | Path to manifest |
| --out | — | True | — | Output file (.json or .csv) |
| --src | — | True | — | Source path |
| --dst | — | True | — | Destination path |
| --log-root | — | True | — | Log directory |
| --force | store_true | False | — | Force copy even if source is symlink |
| --src | — | True | — | Original file to quarantine |
| --log-root | — | True | — | Log directory |
| --reason | — | False | — | Reason |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [case_bible_bootstrap.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/tools/case_bible_bootstrap.py:1>) · SHA-256 `370a5c0e7915dbf4b069cdb9a523cd25ab7f2ab6ded7a24564507fdd63b0f41a`

### `tools/cb_chat_dedup.py`

cb_chat_dedup.py — AI-chat format detector + native-first dedup SCANNER + APPLY + FIND.

  python cb_chat_dedup.py scan  <folder> [--json out.json] [--md report.md]
  python cb_chat_dedup.py apply <folder> [--go] [--json out.json] [--md report.md]
  python cb_chat_dedup.py find  [--roots R1 R2 ...] [--json out.json] [--md report.md]
                                [--max-content-sniff BYTES]
  python cb_chat_dedup.py crosscheck --from-scans <dir-of-scan-jsons> [--json out] [--md out]
                                [--same-drive] [--drives C,D]

`crosscheck` (0.5.0) is a READ-ONLY CROSS-LOCATION duplicate detector that runs AFTER a batch of
`scan` reports already exist (e.g. one `scan` per location found by `find`). It sha256-hashes
every file listed across ALL the given scan reports' `files` lists (full content, byte-identical
only — no near-dupe/fuzzy matching, see build_crosscheck_groups()), groups identical hashes that
span MORE THAN ONE scanned location, and classifies each copy RECOVERY-CLASS (lives under a
backup/salvage/quarantine-adjacent path — see RECOVERY_MARKERS) or PRIMARY-CLASS (everywhere
else). A group with both classes present proposes quarantining the recovery copies (6th
disposition: `stage -> .review_hold/ (recovery dupe)`, naming the surviving PRIMARY path) — a
group that is ALL-recovery or ALL-primary gets NO auto-proposal (`owner_review`). Never moves,
tags, or deletes anything itself — same report-only contract as `scan`/`find`; see
run_crosscheck()/build_crosscheck_groups() for the report shape.

`--same-drive` (0.6.0, crosscheck-only) scopes cross-location grouping to copies on the SAME
drive letter only, per the owner's sequencing rule: same-drive dedup happens FIRST, cross-drive
is a deliberately deferred later pass. With the flag, a hash present on both C: and D: yields
SEPARATE per-drive groups (never a single group spanning drives); a copy whose hash has no other
member on its own drive (i.e. its only same-hash sibling lives on a different drive) cannot form
a same-drive group and is reported under a "cross-drive only (deferred)" count instead of being
proposed — see split_groups_by_drive() (the pure per-drive splitter) and drive_of() (the pure
path -> drive-letter extractor). `--drives C,D` (optional, only meaningful with `--same-drive`)
further restricts processing to the named drive letters; a candidate on any other drive (or with
no drive letter, e.g. a UNC path) is excluded from both the formed groups and the deferred count
for that run. DEFAULT (flag absent) is BYTE-IDENTICAL to 0.5.0's cross-everything behavior — the
same-drive code path is never entered unless `--same-drive` is explicitly passed.

`find` is a READ-ONLY, machine-wide DISCOVERY sweep — it answers "where are all the AI-chat
export files on this box", it does NOT dedup/match/tag/move anything (that is `scan`'s and
`apply`'s job, run per-location afterward). Two-tier detection, both read-only:
  Tier 1 — filename regexes (FIND_REGEXES below): fast, no file contents touched.
  Tier 2 — content sniff: reads only the FIRST --max-content-sniff bytes (default 8KB) of any
    candidate .md/.json/.txt file — regardless of total file size — and runs the SAME delimiter
    signatures scan/apply use (detect_md_format(), the native-json "mapping"+"author" substring
    check) to confirm or refine the tier-1 guess, or to catch an unnamed file tier 1 missed.
  OneDrive Files-On-Demand cloud-only placeholders are NEVER force-hydrated: file attributes are
  checked via os.stat() (itself hydration-safe) before any open()/read(); a cloud-only file is
  classified by filename regex only, with content_sniffed=false in the report.
  See run_find()/render_find_md_report() for the report shape (grouped by directory, JSON + MD).

`scan` is REPORT-ONLY: it only reads files and writes a report (JSON + Markdown) describing
what it found. Never moves/modifies/renames/deletes anything.

`apply` acts on a scan's duplicate sets, per owner directives 2026-07-11 (tagging + re-export
retention are now AUTHORIZED — the "parked" note below on suggested-tags-only is superseded for
the `apply` path; `scan` remains suggestion-only). DEFAULT is DRY-RUN (prints the plan, touches
nothing); pass `--go` to execute (plugin's require_explicit_go convention). What `apply --go`
does, scoped to md files that are re-exports of a NATIVE conversation (deduplication_spec.md
decision_state duplicate_candidate/near_duplicate_candidate, matched_on
title_exact/title_fuzzy/first_message_fuzzy against a native chatgpt_native_json/zip
conversation — NOT md-vs-md near-dupes, NOT the native json-vs-zip container dupe, NOT
work_product/unknown files):
  1. Creates `<folder>/_re-exports/` sibling to the native export.
  2. Tags the AI-chat md file (tagging_spec.md controlled vocabulary — see suggest_tags())
     as Obsidian YAML frontmatter `tags:` (bare words, no `#` — Obsidian's own convention;
     merged with any pre-existing frontmatter tags, never clobbered) written IN PLACE at the
     file's current path, so the tag travels with the content.
  3. Safe-moves the (now-tagged) file into `_re-exports/`: copy to dest -> sha256(source) ==
     sha256(dest) verify -> remove source only after verify passes (mirrors cb_quarantine.py's
     copy-verify-then-remove pattern). Every move is logged to `_re-exports/move-log.json`
     conforming to schemas/move-log-schema.json (schema permits additional properties; a few
     extra fields are carried for INDEX.md regeneration).
  4. Writes/refreshes `_re-exports/INDEX.md`: file | format | matched native conversation
     (title + index) | match method | size | sha256(12) | suggested-use notes, built from the
     move-log so it stays accurate across repeated apply runs.
  5. Also tags (frontmatter, in place, NOT moved) the "kept canonical" AI-chat md files that
     were not re-exports of anything (e.g. a chatgpt_md/customgpt_md/gemini_md file with no
     native match) — same controlled vocabulary, same merge behavior.
  Native `.json`/`.zip` files are NEVER edited — their tags are suggestions only (`scan`
  output / this tool's report), never written into the file. Work-products, `unknown` files,
  and anything not part of a native-re-export duplicate set are UNTOUCHED by `apply`.
  `_re-exports/` itself is excluded from all future scans/applies (added to ATOMIC_DIRS) so
  reruns are idempotent — a second `apply --go` finds nothing left to move and just re-verifies
  tags (no-op if already applied).

`apply` never full-evidence-sidecars AI chats (owner: sidecar.json = the evidence-grade lane,
per sidecar_provenance_spec.md / sidecar-schema.json; AI chats get tags only, not sidecars).

Neither mode calls cb_quarantine.py — this stays decoupled from the raw->sorted R2 quarantine
flow; `_re-exports/` is a same-folder retention subfolder, not the quarantine bucket.

What it does (stage-1, by-SOURCE organizing only — NO domain/facet classification, see
chat-ingest-format-learnings memory note "STAGE MODEL"):
  1. Walks <folder> (skips atomic/system dirs — .obsidian .claude .git .infio_json_db
     .smart-env mi-legal-resources __pycache__ .venv node_modules).
  2. DETECTS which of 5 known AI-chat export formats (+ work_product / unknown) each file is:
       chatgpt_native_json  — conversations.json: list of {mapping:{...}} conversation trees
       chatgpt_native_zip   — a .zip containing conversations.json (peeked via zipfile, never
                               extracted to disk)
       chatgpt_md           — "## Prompt:" / "## Response:" markdown
       customgpt_md         — "You asked:" / "ChatGPT Replied:" markdown (AI Lawyer / Scholar GPT)
       gemini_md            — filename prefixed "Gemini - ..." (invisible-unicode-prefix
                               normalized) and/or "**You:**" / "**Gemini:**" markdown
       dated_export_md      — filename "*-YYYY-MM-DD-HH-MM-SS.md", OR bare "---" turn
                               separators with an "Exported on:" fingerprint and no other
                               marker style. Flagged multi_topic_risk=true (the "---" separator
                               is unreliable — files often concatenate unrelated conversations,
                               see chat-ingest-format-learnings memory note).
       work_product         — .docx, or .md/folder with NO chat-turn delimiters at all
                               (summaries, plans) → route: not-a-transcript.
       unknown               — everything else.
  3. EXTRACTS dedup keys per conversation: (title, first_user_message_prefix[120 chars,
     normalized], date). For native JSON/ZIP this walks EVERY conversation in the array.
  4. MATCHES an .md file against a native conversation (title match OR first-message-prefix
     match >=0.9 via difflib) => a RE-EXPORT duplicate; best copy = the native source
     (native-first rule, chat-ingest-format-learnings memory note). Also detects md<->md
     near-dupes (best copy = larger/more-complete file) and a native-json-vs-native-zip
     CONTAINER duplicate (same conversations, packaged differently).
  5. REPORTS (JSON + Markdown): per-file records, DUPLICATE SETS with a recommended best copy
     and action, suggested (NOT applied) tags per tagging_spec.md, and a summary block.

Conventions followed from skills/case-bible-architect/specs/:
  - deduplication_spec.md: decision_state vocabulary (canonical / duplicate_candidate /
    near_duplicate_candidate / unresolved), the "propose, don't discard" workflow (this tool
    implements steps 1-4 of that spec's Required Workflow: Group / Score / Compare / Propose —
    Review/Approve/Mark/Log are human + cb_quarantine.py's job), and "never assume newest/
    first-found = canonical" (matches are by content key, not by date or path order).
  - tagging_spec.md: suggested tags are drawn ONLY from that spec's controlled vocabulary
    (#chat-export, #context-corpus, #canonical, #duplicate-candidate, #needs-review, #artifact)
    — emitted as suggestions in the report, never written to any file (tagging scheme is
    parked per owner 2026-07-11).

Windows-safe: stdlib only (Python 3.14, no deps), UTF-8 console (filenames in this corpus can
carry an invisible LRM/ZWSP prefix, e.g. Gemini web-clipper exports).

`--provider chatgpt|gemini|claude|perplexity|opencode|all` (comma-separated for multiple; default
`all`), added on scan/apply/find in 0.4.0: narrows which PROVIDER's files are treated as in-scope
for THIS run. Detection itself is UNCHANGED — the tool's own detect_md_format()/
is_native_conversation_list()/FIND_REGEXES functions still run exactly as before (this is why
default/`all` is byte-identical to 0.3.0's output). What changes is a post-classification filter:
any file whose detected format belongs to a provider NOT in the requested set is relabeled
`unmatched/other` for this run (its true detected format is preserved as `detected_format` in the
JSON output) and excluded from dedup matching/tagging/moving — so e.g. `--provider gemini` reports
ChatGPT files as `unmatched/other` instead of `chatgpt_native_json` etc. Files with no specific
provider (`work_product`, `unknown`, and `find`'s generic `chat_html_export` guess) always pass
through untouched regardless of filter — they aren't any provider's claimed territory. The
format->provider mapping is loaded from `../schemas/chat-formats/*.json` at runtime (one
descriptor per provider — see that dir's README.md for the shape and current verified/unverified
status per provider), falling back to a small hardcoded constant if that directory is missing
(e.g. an older plugin cache predating it) so filtering degrades gracefully instead of crashing.

Byline: Claude Code - Sonnet - 2026-07-11

```text
python "E:/AI_Workspace/plugins/plugins/case-bible/tools/cb_chat_dedup.py" --help
```

Declared arguments: `--drives`, `--from-scans`, `--go`, `--json`, `--max-content-sniff`, `--md`, `--provider`, `--roots`, `--same-drive`, `folder`

Declared subcommands:

| Command | Source help |
|---|---|
| `scan` | Scan a folder for AI-chat exports (read-only). |
| `apply` | Move native re-exports into _re-exports/ + tag AI-chat md files. DEFAULT = dry-run; pass --go to execute. |
| `find` | Machine-wide READ-ONLY discovery sweep for AI-chat export files/locations (filename regexes + light content sniff). |
| `crosscheck` | READ-ONLY cross-location byte-identical duplicate detection across a set of prior 'scan' reports, with a recovery-vs-primary quarantine proposal. |

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| folder | — | positional | — | Folder to scan. |
| --json | — | False | — | Write machine-readable JSON report here. |
| --md | — | False | — | Write human-readable Markdown report here. |
| --provider | — | False | — | computed in source |
| folder | — | positional | — | Folder to apply to (same folder previously scanned). |
| --go | store_true | False | — | Execute. Without this flag: dry-run only. |
| --json | — | False | — | Write machine-readable JSON plan/result here. |
| --md | — | False | — | Write human-readable Markdown plan/result here. |
| --provider | — | False | — | computed in source |
| --roots | — | False | — | Root folders to sweep (default: DEFAULT_FIND_ROOTS — OneDrive, Desktop, Downloads, Documents, D:\casebible, D:\Backup, E:\AI_Workspace). |
| --json | — | False | — | Write machine-readable JSON report here. |
| --md | — | False | — | Write human-readable Markdown report here. |
| --max-content-sniff | int | False | — | Bytes read from the HEAD of each candidate .md/.json/.txt file for tier-2 content sniffing (default 8192 = 8KB). This is a head-read cap, not a file-size gate — large files are still sniffed, just only their first N bytes, so huge conversations.json exports are never fully read. |
| --provider | — | False | — | computed in source |
| --from-scans | — | True | — | Directory containing the *.json scan reports to cross-check (one JSON per previously-scanned location). |
| --json | — | False | — | Write machine-readable JSON report here. |
| --md | — | False | — | Write human-readable Markdown report here. |
| --same-drive | store_true | False | — | Scope cross-location duplicate grouping to copies on the SAME drive letter only (a hash present on C: and D: yields separate per-drive groups; a copy whose only same-hash sibling is on a different drive is reported under a 'cross-drive only (deferred)' count, not proposed). Owner sequencing: same-drive dedup runs FIRST, cross-drive is a later pass. Default (flag absent): cross-everything behavior, unchanged. |
| --drives | — | False | — | Comma-separated drive letters to process under --same-drive (e.g. 'C,D'). Only valid together with --same-drive. Default: every drive found among the scan reports. |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [cb_chat_dedup.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/tools/cb_chat_dedup.py:1>) · SHA-256 `cc4bded356c2dd7a84e59b226cf16bd6abefb3e61124edbf5b586a62c85eb273`

### `tools/cb_clean_names.py`

cb_clean_names.py — normalize filenames in the sort ledger: strip duplicate-marker noise
from the BASENAME only — "Copy of", "Copy of Copy of", " - Copy", " - Copy (2)", "(2)",
" (copy)", " (copy 3)", trailing " copy". Never touches folders, dates, phone numbers, or
IDs (those live elsewhere in the path or aren't trailing dup markers).

Operates on work.sort_map (SQLite). Detects collisions (different files cleaning to the same
name) and disambiguates only those, by appending a short file_id. Dry-run prints; --apply writes
`clean_new_path` back to work.sort_map.

  python cb_clean_names.py            # preview what would change
  python cb_clean_names.py --apply    # write clean_new_path into work.sort_map

Reusable: cb_r2_sort imports clean_path() so first-pass ledgers come out clean.
Byline: Claude Code - Opus 4.8 - 2026-06-23

```text
python "E:/AI_Workspace/plugins/plugins/case-bible/tools/cb_clean_names.py" --help
```

Declared arguments: `--apply`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --apply | store_true | False | — | — |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [cb_clean_names.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/tools/cb_clean_names.py:1>) · SHA-256 `6249de67b5bbe55f5c4bd63e5ca7400aab951f6c51d29dcb25b95ec2680755de`

### `tools/cb_connect.py`

cb_connect.py — single connection helper for all Case Bible plugin tools.
Resolves historical SQL/analytics connection helpers; maintained reads use cbcat and cb_vsearch.
Legacy connection definitions below are retained only for their explicit callers. Resolves endpoints
+ credentials from the standard secret locations. No secrets are hard-coded here.

  python cb_connect.py            # self-check: what's reachable / what creds resolve

Byline: Claude Code - Opus 4.8 - 2026-06-23

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [cb_connect.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/tools/cb_connect.py:1>) · SHA-256 `c7547c5da9a0e091587a1d07e49fd17acfd4cfe6999a0b92bf82165661fd52ea`

### `tools/cb_consolidate_typefirst.py`

cb_consolidate_typefirst.py — collapse the 3-run hybrid in casebible-sorted into ONE clean
type-first structure via server-side folder MOVES (within the bucket; raw archive untouched).
Merges old domain/parallel folders into the canonical type folders. Logs a provenance ledger.

  python cb_consolidate_typefirst.py            # DRY-RUN (counts only)
  python cb_consolidate_typefirst.py --execute  # do the server-side moves

Byline: Claude Code - Opus 4.8 - 2026-06-25

```text
python "E:/AI_Workspace/plugins/plugins/case-bible/tools/cb_consolidate_typefirst.py" --help
```

Declared arguments: `--execute`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --execute | store_true | False | — | — |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [cb_consolidate_typefirst.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/tools/cb_consolidate_typefirst.py:1>) · SHA-256 `012c8521a66bb15ee71cf3fba9a19c7d3e3ce1d0c5828923eeb56c4b8f954962`

### `tools/cb_custody_chain.py`

Case Bible Multi-Level Custody Hash Tool (H1/H2/H3)

Byline: Claude Code . Fable 5 . 2026-07-02

Implements and verifies the Case Bible 3-level tamper-evident custody scheme:

  H1  file hash    sha256 over the raw source-file bytes (streaming).
                   Canon: h1-rawbytes-v1 (matches evidence/custody.py).
  H2  message hash sha256 per message, file-bound (includes the H1 hash so a
                   message can never be detached from its source file).
  H3  chain hash   entry_hash = sha256(hex(previous_hash) + hex(H2)) as ASCII.
                   Genesis previous_hash = the H1 file hash. The final entry
                   hash is the CHAIN HEAD, sealed into the append-only
                   evidence.evidence_hash H1 row (meta.chain_head).
                   Canon: h3-chain-v1 (PROVEN against the 2026-06-27 pilot).

CANON REGISTRY (never silently change a recipe -- add a new canon version):
  h1-rawbytes-v1     sha256(file_bytes)                                 PROVEN
  h3-chain-v1        sha256(utf8(prev_hex + h2_hex))                    PROVEN
  h2-filebound-v1    recipe LOST (pilot agent scratchpad, bestoffort-v2-
                     2026-06-26). Stored H2 values remain tamper-evident
                     through the H3 chain + sealed head, but CANNOT be
                     independently recomputed from message content.
                     verify-chain still fully verifies H3 over them.
  h2-canonical-v2    sha256(utf8(f"{file_hash_hex}|{sequence_number}|{role}|"
                     f"{occurred_at_utc_iso}|{content}"))               CURRENT
                     occurred_at_utc_iso: "YYYY-MM-DD HH:MM:SS+00:00".
                     Use for ALL new ingests.

Usage:
  python cb_custody_chain.py hash-file <path>
  python cb_custody_chain.py verify-file <path> --sha256 <hex>
  python cb_custody_chain.py verify-chain <export.csv|.jsonl> [--file <src>] [--head <hex>] [--recompute-h2]
  python cb_custody_chain.py build-chain <records.jsonl> --file-hash <hex> [--out <path.jsonl>]

Input formats:
  .csv    the psql --csv export of analysis.normalized_record rows. Must carry
          columns: content, role, attrs (json text). occurred_at optional.
  .jsonl  one record per line: {"content","role","occurred_at",...} with either
          top-level or "attrs"-nested sequence_number / message_hash /
          chain_entry_hash / chain_previous_hash / file_hash.

Exit codes: 0 = verified/ok, 1 = verification FAILED, 2 = usage/input error.

```text
python "E:/AI_Workspace/plugins/plugins/case-bible/tools/cb_custody_chain.py" --help
```

Declared arguments: `--file`, `--file-hash`, `--head`, `--out`, `--recompute-h2`, `--sha256`, `path`, `records`

Declared subcommands:

| Command | Source help |
|---|---|
| `hash-file` | H1: sha256 a source file |
| `verify-file` | H1: verify a file against a known sha256 |
| `verify-chain` | H3 (+optional H1/H2) verification of an export |
| `build-chain` | compute H2 v2 + H3 chain for parsed records |

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| path | — | positional | — | — |
| path | — | positional | — | — |
| --sha256 | — | True | — | — |
| records | — | positional | — | .csv (psql export) or .jsonl |
| --file | — | False | — | also re-hash the original source file (H1) |
| --head | — | False | — | expected chain head (evidence_hash meta.chain_head) |
| --recompute-h2 | store_true | False | — | recompute H2 (canon h2-canonical-v2 ONLY; v1 pilot data will mismatch) |
| records | — | positional | — | .jsonl of parsed records (content/role/occurred_at/sequence_number) |
| --file-hash | — | True | — | H1 hex of the source file |
| --out | — | False | — | output path (default: <records>.hashed.jsonl) |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [cb_custody_chain.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/tools/cb_custody_chain.py:1>) · SHA-256 `66e8c220295c7e37aa4ea92cd9944123277d6118d764c4ec8428653ace638558`

### `tools/cb_execute_copy.py`

cb_execute_copy.py — execute the reconciled raw->sorted copy.

Reads work.sort_map; for every RESOLVED, non-superseded row, server-side-copies
  r2:casebible-raw/<r2_key>  ->  r2:casebible-sorted/<new_path>
in parallel (rclone copyto, no shell quoting issues). COPY ONLY — never deletes; raw untouched.
Every copy logged to work.copy_log (provenance). Unresolved rows are skipped + reported.

  python cb_execute_copy.py            # DRY-RUN: counts + sample, no copy
  python cb_execute_copy.py --go       # actually copy (server-side, parallel)
Byline: Claude Code - Opus 4.8 - 2026-06-23

```text
python "E:/AI_Workspace/plugins/plugins/case-bible/tools/cb_execute_copy.py" --help
```

Declared arguments: `--go`, `--workers`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --go | store_true | False | — | — |
| --workers | int | False | — | — |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [cb_execute_copy.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/tools/cb_execute_copy.py:1>) · SHA-256 `fa746c7afe01e7f608c758fc6c4739b9fc5ff7696854d944e064859f0607587f`

### `tools/cb_execute_typefirst.py`

cb_execute_typefirst.py — execute the type-first root-pile ledger as server-side R2 COPIES.

COPY-ONLY (never delete): content -> r2:casebible-sorted/<new_path>; obvious junk ->
r2:casebible-quarantine/<new_path>. Raw is never modified. Writes an executed ledger
(old_path -> new_path + status) for provenance.

  python cb_execute_typefirst.py --limit 3     # test N rows first
  python cb_execute_typefirst.py               # full run

Byline: Claude Code - Opus 4.8 - 2026-06-25

```text
python "E:/AI_Workspace/plugins/plugins/case-bible/tools/cb_execute_typefirst.py" --help
```

Declared arguments: `--limit`, `--workers`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --limit | int | False | — | — |
| --workers | int | False | — | — |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [cb_execute_typefirst.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/tools/cb_execute_typefirst.py:1>) · SHA-256 `ca8989c5221c7de7d291ed8704c5323340b6cc2cfd14b4163523066f8381e480`

### `tools/cb_lakehouse.py`

cb_lakehouse.py — publish Case Bible tables to R2 Data Catalog (Apache Iceberg)
so they're queryable cloud-native via DuckDB / R2 SQL with no object scans.

Pulls from the LIVE casebible Postgres (the data here changes — other processes
add/remove/consolidate), NOT a frozen snapshot. Requires the ovh2 SSH tunnel
on localhost:15432 (run the Desktop resume script if it's down).

  python cb_lakehouse.py            # enrichment only
  python cb_lakehouse.py --all      # enrichment + faces/faces_scanned/photos/screenshots

Env: R2_CATALOG_TOKEN  (Bearer for the catalog REST endpoint)

> Byline: Claude Code · Opus 4.8 · 2026-06-23

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [cb_lakehouse.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/tools/cb_lakehouse.py:1>) · SHA-256 `7b16ca10295f6f6c068c4b819d4950561ea258f17b773d9eddbb1d928354ed79`

### `tools/cb_postrestore_verify.py`

Verify the maintained Case Bible readers without changing source or index data.

Inputs: installed tool environment; outputs: catalog/search diagnostics and exit status.
Effects: read-only metadata/query calls; choose for a current reader check.
Byline: Codex, GPT-6, 2026-10-04.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [cb_postrestore_verify.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/tools/cb_postrestore_verify.py:1>) · SHA-256 `3e37c9d712a4f2675eeb18c999f99cf3bc24eca07f21d8a48093703232bc02b7`

### `tools/cb_quarantine.py`

cb_quarantine.py — "mimic delete" SAFELY: MOVE superseded/stale files into quarantine.

After a sort (raw->sorted) the non-canonical duplicates and any replaced/stale files should
leave the working view — but the Case Bible NEVER hard-deletes. This moves them to
`casebible-quarantine/.to_be_deleted/<stamp>/` (reversible) and logs every move. It only emits
a reviewable rclone batch (no auto-fire). `rclone move` removes the source only after the copy
lands in quarantine — the file is preserved, just relocated.

  python cb_quarantine.py                                  # DRY-RUN: superseded dupes from sort_map
  python cb_quarantine.py --where "decision_state='duplicate_superseded'"
  python cb_quarantine.py --execute                        # write the quarantine move batch

Byline: Claude Code - Opus 4.8 - 2026-06-23

```text
python "E:/AI_Workspace/plugins/plugins/case-bible/tools/cb_quarantine.py" --help
```

Declared arguments: `--execute`, `--where`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --where | — | False | — | SQL filter over work.sort_map |
| --execute | store_true | False | — | — |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [cb_quarantine.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/tools/cb_quarantine.py:1>) · SHA-256 `f80e42803fbef9d0567f2dd0d9315d508b06e743b3121af447c7e4110ebea9a8`

### `tools/cb_r2_sort.py`

cb_r2_sort.py — build the raw->sorted provenance ledger and (gated) execute the copy.

Reads the live enrichment index (PG via tunnel), maps every file to its sorted-domain
path per the v4 taxonomy (NO numbers), keeps packages intact (convo + attachment tails),
de-dups by md5 best-version (from D:/casebible/casebible.duckdb r2_files), and writes the
old_path->new_path ledger. NEVER deletes; raw is the source, sorted is a copy.

  python cb_r2_sort.py                      # DRY-RUN: ledger + summary for AI chats + messaging
  python cb_r2_sort.py --all                # whole corpus
  python cb_r2_sort.py --execute            # also write the rclone batch script (still no auto-fire)

Output ledger: D:/casebible/exports/sort_map_<scope>_<stamp>.parquet
Byline: Claude Code - Opus 4.8 - 2026-06-23

```text
python "E:/AI_Workspace/plugins/plugins/case-bible/tools/cb_r2_sort.py" --help
```

Declared arguments: `--all`, `--execute`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --all | store_true | False | — | whole corpus (default: AI chats + messaging) |
| --execute | store_true | False | — | also emit the rclone batch script (no auto-fire) |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [cb_r2_sort.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/tools/cb_r2_sort.py:1>) · SHA-256 `55a89a034d73e59ab349c057a11c70af771a963b9dfead64463e6f70ebd2141c`

### `tools/cb_reclassify_pilot.py`

cb_reclassify_pilot.py — PILOT: re-classify the mis-tagged `ai_chat` batch by FUNCTION,
using content-summary + original PATH (provenance signal) + extension. Provenance becomes a
tag; source structure (platform/account) is preserved. Low-confidence -> review (HITL).
Read-only: writes a before->after proposal CSV, moves nothing.

  python cb_reclassify_pilot.py

Byline: Claude Code - Opus 4.8 - 2026-06-25

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [cb_reclassify_pilot.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/tools/cb_reclassify_pilot.py:1>) · SHA-256 `5d1e7ae5a8fcb04ef282a4755886b0dda14d37b339ee98c0609e31c5655dc65c`

### `tools/cb_review_board.py`

cb_review_board.py — generates the self-contained local HTML "review board" for
cb_chat_dedup.py's `scan` (per-location duplicate sets) and `crosscheck` (cross-location
byte-identical recovery-dupe) reports.

  python cb_review_board.py --scans <dir-with-scan-jsons> [--crosscheck <crosscheck.json>]
                             [--extra <extra-scan.json>] [--extra <extra-scan.json> ...]
                             [--out <board.html>]

Promoted from the session scratchpad's `build_review_board.py` v4 (2026-07-12) into a durable
plugin tool — same HTML/CSS/JS output, same 6-action-per-file disposition model, refactored into
a proper argparse CLI with no session-specific hardcoded paths (mirrors cb_chat_dedup.py's
conventions: report-only, never moves/tags/deletes anything — this tool only ever READS the JSON
reports named on the command line and WRITES one HTML file).

What it does:
  1. Loads every `*.json` file in `--scans <dir>` that looks like a `cb_chat_dedup.py scan`
     report (has a top-level `duplicate_sets` key, or `tool` == "cb_chat_dedup.py") — anything
     else in that directory (a stray `.md` report, an unrelated JSON) is skipped with a printed
     note, never a hard error.
  2. `--extra <path>` (repeatable) adds individual scan-report JSON files living OUTSIDE
     `--scans <dir>` (e.g. a one-off vault sweep report) to the same pool, same validity check.
  3. `--crosscheck <path>` (optional) loads a `cb_chat_dedup.py crosscheck` report and renders
     the TOP "Cross-location recovery duplicates" section (byte-identical dupes where a
     RECOVERY-CLASS copy — backup/salvage-marked path — duplicates a PRIMARY-CLASS copy living
     elsewhere). Omitted entirely (not just empty) when `--crosscheck` isn't passed.
  4. Renders one self-contained HTML file (`--out`, default `./chat-dedup-review-board.html`):
     dashboards, per-location collapsible duplicate-set lists, a 6-action `<select>` per
     non-canonical file (keep in place / keep — provenance copy / move → _re-exports/ /
     move → provider dir / move → _extracted/ / stage → .review_hold/ — recovery dupe), a
     search box, dark-mode support, and two clipboard exporters (approval JSON, a "GO"
     instruction paragraph for Claude to act on). Nothing in the generated page or this script
     ever moves/tags/deletes a real file — the page is a proposal-review UI only.

Byline: Claude Code - Sonnet 5 - 2026-07-12

```text
python "E:/AI_Workspace/plugins/plugins/case-bible/tools/cb_review_board.py" --help
```

Declared arguments: `--crosscheck`, `--extra`, `--out`, `--scans`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --scans | — | True | — | Directory holding *.json scan reports (one per previously-scanned location, e.g. the output of running 'scan' on every location 'find' turned up). Every *.json in the dir is tried; anything that doesn't look like a cb_chat_dedup.py scan report is skipped with a printed note. |
| --crosscheck | — | False | — | Optional cb_chat_dedup.py 'crosscheck' JSON report. Renders the top 'Cross-location recovery duplicates' section when given; that section is omitted entirely (not just empty) when this flag is absent. |
| --extra | append | False | — | Additional individual scan-report JSON file(s) outside --scans dir (repeatable: --extra a.json --extra b.json). |
| --out | — | False | — | Output HTML path (default: ./chat-dedup-review-board.html). |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [cb_review_board.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/tools/cb_review_board.py:1>) · SHA-256 `bdfd26deb62e7c222b011066f8c263eceb6169a85e177bf516fac2dac83db7ca`

### `tools/cb_safemove_fs.py`

cb_safemove_fs.py — SAFE local-filesystem move: COPY -> VERIFY -> QUARANTINE (never a bare move).

Owner mandate 2026-07-12: a "move" is three gated steps —
  1. COPY   src -> dst
  2. VERIFY dst hash == src hash (md5 default; xxh3 if xxhash lib present and --algo xxh3)
  3. QUARANTINE the source into <quarantine-root>/<job>/<original-relpath>  (NEVER delete)
Any failure leaves the source exactly where it was and marks the row FAILED.

ONEDRIVE MODE (--onedrive, owner 2026-07-12): files-on-demand stubs must NOT hydrate.
Stubs (FILE_ATTRIBUTE_RECALL_ON_DATA_ACCESS 0x400000) are handled with RENAME semantics
(server-side move, zero download) + size-metadata verify instead of copy+hash — weaker
verification, accepted because OneDrive's 30-day online recycle backstops it. Locally
available files inside OneDrive still get the full copy->hash->quarantine treatment.
Quarantine root should be INSIDE the same OneDrive tree (e.g. <root>/.review_hold) so
quarantining is itself a server-side rename.

Input = an APPROVED move-map CSV with headers: src,dst   (extra columns ignored, preserved in ledger)
Every row is ledgered: src,dst,quarantine_path,size,hash,status,timestamp

Usage:
  python cb_safemove_fs.py --map approved-map.csv --quarantine "E:\CaseBible\.review_hold"       --job onedrive-drain-01 --ledger ledger.csv [--dry-run] [--algo md5|xxh3]

Byline: Claude Code - Fable 5 - 2026-07-12

```text
python "E:/AI_Workspace/plugins/plugins/case-bible/tools/cb_safemove_fs.py" --help
```

Declared arguments: `--algo`, `--dry-run`, `--job`, `--ledger`, `--map`, `--onedrive`, `--quarantine`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --map | — | True | — | approved move-map CSV (src,dst) |
| --quarantine | — | True | — | quarantine ROOT (sources land under <root>/<job>/) |
| --job | — | True | — | job name (quarantine subdir + ledger tag) |
| --ledger | — | True | — | ledger CSV to append (created if missing) |
| --dry-run | store_true | False | — | report what would happen; touch nothing |
| --algo | — | False | ['md5', 'xxh3'] | — |
| --onedrive | store_true | False | — | files-on-demand aware: stubs use rename semantics + size verify (NO hydration) |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [cb_safemove_fs.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/tools/cb_safemove_fs.py:1>) · SHA-256 `71c3226f8ee9e3386aa048c93abfae521ec8669b2687b1f83310cb6868abd935`

### `tools/cb_safemove_rclone.py`

cb_safemove_rclone.py — SAFE rclone/R2 move: COPY -> VERIFY -> QUARANTINE (never a bare move).

Owner mandate 2026-07-12. Cloud twin of cb_safemove_fs.py:
  1. COPY   rclone copyto src -> dst           (server-side within same remote: no egress)
  2. VERIFY MD5 of dst == MD5 of src           (R2 exposes MD5 as plain ETag via rclone lsf --hash)
  3. QUARANTINE the source object: rclone moveto src -> <quarantine-prefix>/<job>/<src-path>
     (object relocation, NEVER a delete; quarantine prefix defaults to .review_hold)
Any failure leaves the source object untouched and marks the row FAILED.

Input = APPROVED move-map CSV with headers: src,dst — full rclone paths, e.g.
  r2:casebible-sorted/Triage/x.pdf , r2:casebible-sorted/EvidenceVault/_intake/x.pdf
Ledgered identically to the fs tool.

Usage:
  python cb_safemove_rclone.py --map approved-map.csv --job sorted-pass-01       --quarantine "r2:casebible-quarantine/.review_hold" --ledger ledger.csv [--dry-run]

Byline: Claude Code - Fable 5 - 2026-07-12

```text
python "E:/AI_Workspace/plugins/plugins/case-bible/tools/cb_safemove_rclone.py" --help
```

Declared arguments: `--dry-run`, `--job`, `--ledger`, `--map`, `--quarantine`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --map | — | True | — | — |
| --job | — | True | — | — |
| --quarantine | — | True | — | quarantine prefix, e.g. r2:casebible-quarantine/.review_hold |
| --ledger | — | True | — | — |
| --dry-run | store_true | False | — | — |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [cb_safemove_rclone.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/tools/cb_safemove_rclone.py:1>) · SHA-256 `a4eaab96d59df68ff6efd2f1bf9e78b5cccaebec863707e4c99c4e5db549e571`

### `tools/cb_search_server.py`

Search existing corpus indexes and shape bounded results with in-memory DuckDB.

Inputs: one JSON request on stdin; outputs: compact JSON or readable search hits.
Effects: read-only index queries and one hosted query embedding; no indexing or source writes.
Choose for file-content discovery; use cbcat for inventory and Docstore for project documentation.
Byline: Codex, GPT-6, 2026-10-04.
Derived retrieval contract: Consignatio Intake filesystem_search.py and comm_timeline_mvp/search.py.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [cb_search_server.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/tools/cb_search_server.py:1>) · SHA-256 `cae6c1387cddbcfd996f696c09b60ee6aa39e726fce925addf3e873d1d1218af`

### `tools/cb_sort_assist.py`

cb_sort_assist.py — agent-assisted sort for the AMBIGUOUS files.

Heuristics in cb_r2_sort handle the clear cases and flag the rest `needs_agent=true`. This
tool pulls those rows + their enrichment context (doc_type, title, summary) and asks a CHEAP
Claude model (Haiku) — programmatically, in batches — to pick the right destination from the
fixed v4 taxonomy. Decisions are written back to work.sort_map (domain_path + new_path).

  python cb_sort_assist.py              # DRY-RUN: print agent proposals
  python cb_sort_assist.py --apply      # write decisions back to work.sort_map
  python cb_sort_assist.py --batch 15 --limit 200

Needs ANTHROPIC_API_KEY (env or ~/.secrets/anthropic.env). Model via CB_ASSIST_MODEL.
Byline: Claude Code - Opus 4.8 - 2026-06-23

```text
python "E:/AI_Workspace/plugins/plugins/case-bible/tools/cb_sort_assist.py" --help
```

Declared arguments: `--apply`, `--batch`, `--limit`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --apply | store_true | False | — | — |
| --batch | int | False | — | — |
| --limit | int | False | — | — |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [cb_sort_assist.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/tools/cb_sort_assist.py:1>) · SHA-256 `6556d658552102396c81c472923a08df5e731a599d2fd31a7d8ccf75cfb80cec`

### `tools/cb_typefirst_ledger.py`

cb_typefirst_ledger.py — DRAFT proposal ledger for the raw-ROOT pile, TYPE-FIRST model.

Folders = TYPE (Knowledge/Evidence/Legal/Entities/Case Management/Tools & Platform/
Documents/Exports & Bundles/Inbox; + Quarantine). Inside Evidence -> by SOURCE/platform.
Domain is NEVER a folder: domain/type/function tags ride along as multi-tag metadata so the
DB can pivot by domain later with zero re-sorting. Restricted to the enriched-AND-at-R2-root
intersection (the real 396). DRY-RUN ONLY: writes a local ledger, no R2 copy/move.

Output: D:/casebible/exports/sort_proposal_typefirst_root_<STAMP>.{parquet,csv}
Byline: Claude Code - Opus 4.8 - 2026-06-25

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [cb_typefirst_ledger.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/tools/cb_typefirst_ledger.py:1>) · SHA-256 `5277bd2aa82f7338304152d6c8edd8bad01193e6ec6f8751b1f636b1506a6a5b`

### `tools/cb_vsearch.py`

Search Case Bible content through the maintained server search tool.

Inputs: natural-language query, corpus, filters and presentation options.
Outputs: source-linked excerpts in human text or compact/full JSON.
Effects: read-only SSH query; embedding and DuckDB shaping run on ovh-files.
Choose for content; cbcat reads file inventories and lake publications.
Byline: Codex, GPT-6, 2026-10-04.

```text
python "E:/AI_Workspace/plugins/plugins/case-bible/tools/cb_vsearch.py" --help
```

Declared arguments: `--contains`, `--corpus`, `--excerpt-chars`, `--fetch`, `--from`, `--id`, `--k`, `--mode`, `--presentation`, `--source`, `--to`, `query`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| query | — | positional | — | — |
| --corpus | — | False | ['all', 'documents', 'messages', 'chats'] | — |
| --mode | — | False | ['hybrid', 'keyword'] | — |
| --k | int | False | — | — |
| --fetch | int | False | — | Candidate window per corpus, maximum 500 |
| --source | — | False | — | Literal source-path substring |
| --contains | — | False | — | Literal text substring within retrieved candidates |
| --from | — | False | — | YYYY-MM-DD |
| --to | — | False | — | YYYY-MM-DD, inclusive |
| --id | — | False | — | Open a result ID; also specify --corpus |
| --excerpt-chars | int | False | — | — |
| --presentation | — | False | ['human', 'compact', 'json'] | — |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [cb_vsearch.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/tools/cb_vsearch.py:1>) · SHA-256 `1f3ac7f903dbdfa8e73a898af74d8e8934a90222839b59239688f6da072b14fb`

### `tools/cbcat_compare_sql.py`

Build the read-only comparison SQL for `cbcat compare`.

Reads an `rclone lsjson --hash` listing of a local directory and prints SQL that checks each
file against the PostgreSQL catalog (`casebible.raw_duck`). Candidates travel inline as a VALUES
list because a read-only session may not create a temp table.

Usage: cbcat_compare_sql.py <lsjson file> <corpus table> <hash|name|both>

> Byline: Codex, 2026-10-04. Current inventory reader repair.
> Byline: Claude Code · Fable 5.1 · 2026-09-20

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [cbcat_compare_sql.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/tools/cbcat_compare_sql.py:1>) · SHA-256 `adfbe104c1c6c9d250aee2e9a7d25cd7f461d62964a653200d3a58dea5332f77`

### `tools/forensics.py`

Case Bible Forensics & Deduplication Tools

Provides CLI access to forensic extraction, fast hashing, similarity analysis,
batch operations, and sidecar generation.

Usage:
  python forensics.py hash <file_path> [--type blake3|sha256] [--output json]
  python forensics.py fuzzy-hash <file_path> [--output json]
  python forensics.py fuzzy-compare <file1> <file2> [--output json]
  python forensics.py fuzzy-compare --hash <tlsh1> <tlsh2> [--output json]
  python forensics.py extract-strings <file_path> [--min-len 4] [--strict]
  python forensics.py fuzzy-match <string1> <string2> [--output json]
  python forensics.py batch-hash <directory> [--type blake3|sha256] [--output json]
  python forensics.py create-sidecar <file_path> --source-type <type> --domain <domain>

```text
python "E:/AI_Workspace/plugins/plugins/case-bible/tools/forensics.py" --help
```

Declared arguments: `--domain`, `--hash`, `--hash-type`, `--min-len`, `--output`, `--source-platform`, `--source-type`, `--strict`, `--type`, `--write`, `directory`, `file`, `string1`, `string2`, `target1`, `target2`

Declared subcommands:

| Command | Source help |
|---|---|
| `hash` | Generate cryptographic hash for a file |
| `fuzzy-hash` | Generate TLSH similarity hash for a file |
| `fuzzy-compare` | Compare two files or TLSH hashes for similarity |
| `extract-strings` | Extract readable strings from binary files |
| `fuzzy-match` | Compare two strings for similarity (0-100) |
| `batch-hash` | Hash all files in a directory tree |
| `create-sidecar` | Generate a schema-compliant sidecar for a file |

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| file | — | positional | — | Path to the file |
| --type | — | False | ['blake3', 'sha256'] | Hashing algorithm |
| --output | — | False | ['text', 'json'] | — |
| file | — | positional | — | Path to the file |
| --output | — | False | ['text', 'json'] | — |
| target1 | — | positional | — | First file path or TLSH hash |
| target2 | — | positional | — | Second file path or TLSH hash |
| --hash | store_true | False | — | Treat targets as TLSH hash strings instead of file paths |
| --output | — | False | ['text', 'json'] | — |
| file | — | positional | — | Path to the file |
| --min-len | int | False | — | Minimum string length |
| --strict | store_true | False | — | Only output ML-verified interesting strings |
| --output | — | False | ['text', 'json'] | — |
| string1 | — | positional | — | First string |
| string2 | — | positional | — | Second string |
| --output | — | False | ['text', 'json'] | — |
| directory | — | positional | — | Directory to scan |
| --type | — | False | ['blake3', 'sha256'] | Hashing algorithm |
| --output | — | False | ['text', 'json'] | — |
| file | — | positional | — | Path to the file |
| --source-type | — | True | ['screenshot', 'document', 'audio', 'video', 'image', 'chat_export', 'email', 'text_message', 'financial_record', 'public_record', 'web_capture', 'social_capture', 'court_document', 'filing', 'declaration', 'discovery', 'other'] | Type of source material |
| --domain | — | False | — | Domain assignment |
| --source-platform | — | False | — | Platform of origin |
| --hash-type | — | False | ['blake3', 'sha256'] | — |
| --write | store_true | False | — | Write sidecar file to disk |
| --output | — | False | ['text', 'json'] | — |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [forensics.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/tools/forensics.py:1>) · SHA-256 `b1a403f93a9d1cc1c8c37c3dfb0454220e85158e743aecbb7dfa276ce61f5bef`

### `tools/generate_search_skill.py`

Generate the Case Bible search skill from executable docstrings and CLI help.

Inputs: sibling search modules; outputs: skills/cb-vsearch/SKILL.md.
Effects: replaces only the generated skill; choose after changing search contracts.
Byline: Codex, GPT-6, 2026-10-04.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [generate_search_skill.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/tools/generate_search_skill.py:1>) · SHA-256 `df1a1aaa37b1e4bbb072a5b1ddad3f42ae0f920128b5b0eef419f16bf47aa924`

### `tools/lake_read.py`

Read a manifest-verified B2 Parquet table through the existing server catalog.

Inputs: table name and optional SELECT over lake_table on stdin.
Outputs: generation, SHA-256 verification, and PostgreSQL query results.
Effects: retains a derived server cache; never changes source objects or catalog rows.
Choose for published snapshots; cbcat query reads the working catalog instead.
Byline: Codex, 2026-10-04.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [lake_read.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/tools/lake_read.py:1>) · SHA-256 `66d719394393908e9150e24483e8bebad9a37c8e21fc7c01c675f71da8e9ce5e`

### `tools/registry.py`

Case Bible SQLite Registry

The file-level operational index for the Case Bible system. Serves as the
canonical machine-readable tracking layer, audit/provenance ledger, and source
for generating refreshed manifests, indexes, and dashboards.

Scope: FILE-LEVEL ONLY. Per-message and per-component hashing belongs
downstream in the Postgres layer.

Usage:
  python registry.py init --vault-root /path/to/vault
  python registry.py register <file_path> --domain Evidence --source-type screenshot
  python registry.py query --hash <hash_value>
  python registry.py query --path <file_path>
  python registry.py log-action --action move --description "Moved file to Evidence"
  python registry.py status
  python registry.py export --format json

```text
python "E:/AI_Workspace/plugins/plugins/case-bible/tools/registry.py" --help
```

Declared arguments: `--artifact-id`, `--domain`, `--hash`, `--hash-type`, `--output`, `--path`, `--source-platform`, `--source-type`, `--vault-root`, `file`

Declared subcommands:

| Command | Source help |
|---|---|
| `init` | Initialize the registry database |
| `register` | Register a file as an artifact |
| `query` | Query the registry |
| `status` | Show registry statistics |
| `export` | Export all artifacts as JSON |

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --vault-root | — | False | — | Vault root directory (auto-discovered if omitted) |
| --output | — | False | ['text', 'json'] | Output format |
| --vault-root | — | True | — | Vault root directory |
| file | — | positional | — | Path to the file |
| --domain | — | False | — | Domain assignment (Evidence, Inbox, etc.) |
| --source-type | — | False | — | Source type (screenshot, document, etc.) |
| --source-platform | — | False | — | Source platform (Facebook, Gmail, etc.) |
| --hash-type | — | False | ['blake3', 'sha256'] | — |
| --output | — | False | ['text', 'json'] | — |
| --hash | — | False | — | Search by hash value |
| --path | — | False | — | Search by file path |
| --artifact-id | — | False | — | Search by artifact ID |
| --output | — | False | ['text', 'json'] | — |
| --output | — | False | ['text', 'json'] | — |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [registry.py:1](<E:/AI_Workspace/plugins/plugins/case-bible/tools/registry.py:1>) · SHA-256 `95e4f6d18d37b08f74279033e246026399b0ffdde3053cc3ec54adca49ba3fe9`

## Mcp Tools

No entries found in the inspected declarations.

## Mcp Servers

No entries found in the inspected declarations.

## Incomplete checks

- commands/cb-chat-dedup.md: Frontmatter fallback: ScannerError
- skills/mp-codebase-scanning/scripts/hash-files.py: SyntaxError

Back to [[Code/wiki/plugin-inventory|Plugin inventory]].

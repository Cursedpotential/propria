---
title: "migration-passes"
type: tool-reference
status: SOURCE_VERIFIED
date: 2026-10-04
generated_by: "Codex / GPT-6"
revision: 1
tags: [propria, tools, wiki]
---

# migration-passes

> _Byline: Codex · GPT-6 · 2026-10-04 — generated from the cited sources._

Multi-agent pipeline for codebase migration, documentation repair, gap auditing, and architecture planning with structured handoffs and Plannotator integration

Source: `E:/AI_Workspace/plugins/plugins/migration-passes`. Version: `0.5.0`.
Registered: `True`. Installed manifests: not found in inspected manifests.

## How to invoke it

Slash commands run inside an agent app. Terminal commands run in PowerShell. MCP tools require an attached server and are called by the agent or an MCP client. A skill is an instruction package, not a standalone executable.

Descriptions below are extracted from source metadata/docstrings. This is discovery evidence, not a claim every service invocation passed.

## Commands

### `analyze-transcripts`

Extract structured intelligence from transcript chunks — architectural decisions, known issues, intended behavior, doc gaps, deferred work, feature mentions, and conflicts.

```text
/migration-passes:analyze-transcripts
```

Arguments: `[--focus <categories>]`

Source: [analyze-transcripts.md:1](<E:/AI_Workspace/plugins/plugins/migration-passes/commands/analyze-transcripts.md:1>) · SHA-256 `b71402cdf5608f94defcd2b7e2f04117670d526602b43c65c50aff597a623189`

### `audit-gaps`

Read-only structural audit after documentation repair. Identifies undocumented exports, dead code, import integrity issues, type inconsistencies, test coverage gaps, and config/env drift.

```text
/migration-passes:audit-gaps
```

Arguments: `[--categories all|exports,dead-code,imports,types,tests,config]`

Source: [audit-gaps.md:1](<E:/AI_Workspace/plugins/plugins/migration-passes/commands/audit-gaps.md:1>) · SHA-256 `0f80fb731009ae449b4dbe6aa38296503338d6497d945bd12e98474821bec4c3`

### `export-obsidian`

Export the completed migration run into Obsidian-compatible markdown with YAML frontmatter, wikilinks, and a map-of-content note. Also supports JSON, CSV, and Parquet exports.

```text
/migration-passes:export-obsidian
```

Arguments: `[--formats obsidian,json,csv] [--output <directory>]`

Source: [export-obsidian.md:1](<E:/AI_Workspace/plugins/plugins/migration-passes/commands/export-obsidian.md:1>) · SHA-256 `63fa8df70f4d010632d249635861351c3cd59be928c25fabf4b3205822b8db92`

### `inventory`

Map both NEW and OLD codebases before any other pass runs. Walks directories, builds manifests, identifies hot spots, records hashes for change detection.

```text
/migration-passes:inventory
```

Arguments: `<new-codebase-root> <old-codebase-root> [--exclude node_modules,.git,dist] [--max-size 5]`

Source: [inventory.md:1](<E:/AI_Workspace/plugins/plugins/migration-passes/commands/inventory.md:1>) · SHA-256 `f35551331167df8aa64381a3fa90ff48110e26178bcd7fe57ac6d1bcf49a5fc8`

### `migrate`

Compare OLD and NEW codebases, identify missing or partial legacy features, and port them safely. Three-phase process: inventory diff, porting plan, execution.

```text
/migration-passes:migrate
```

Arguments: `[--dry-run] [--skip-present]`

Source: [migrate.md:1](<E:/AI_Workspace/plugins/plugins/migration-passes/commands/migrate.md:1>) · SHA-256 `3e8ce424fbdab6157ae5d373fca16f49b007f7c717eea0e71c680d980cd3350a`

### `patch-docs`

Walk the codebase and patch documentation to match current implementation and known intended behavior. Emits planned change blocks before modifying.

```text
/migration-passes:patch-docs
```

Arguments: `[--style standard|minimal|detailed] [--risk-threshold low|medium|high]`

Source: [patch-docs.md:1](<E:/AI_Workspace/plugins/plugins/migration-passes/commands/patch-docs.md:1>) · SHA-256 `47f0f017eec0ad935a3c3ce914a26b1b7c927d5fa4678dc1356ffed4fa3c46aa`

### `plan-architecture`

Heavily interactive interview-based planning pass. Evaluates architecture approaches, interviews the user, and produces a migration plan reviewed via Plannotator's interactive annotation UI.

```text
/migration-passes:plan-architecture
```

Arguments: `[--approach <name>] [--skip-interview]`

Source: [plan-architecture.md:1](<E:/AI_Workspace/plugins/plugins/migration-passes/commands/plan-architecture.md:1>) · SHA-256 `3465c3eae960a6f6c10df5f973e489b028684bac6ec757d805d50567c60ff78d`

### `run`

Run the migration pipeline sequentially from the beginning or from a specific pass. Executes Passes 0-4 automatically, then Pass 5 (interactive planning) by default, then Passes 6-7. Use --skip-interactive to auto-generate a plan without the interview. Use --from to resume from a specific pass.

```text
/migration-passes:run
```

Arguments: `[--from <pass_number>] [--skip-interactive] [--pass <number>]`

Source: [run.md:1](<E:/AI_Workspace/plugins/plugins/migration-passes/commands/run.md:1>) · SHA-256 `fd4c9c45b1ddf3311f2c17a93b24d3542c5df3ad2c4ee26cfa98f73c466d9af2`

### `status`

Show current migration pipeline state — which passes are complete, partial, blocked, or missing, with timestamps and next-step guidance.

```text
/migration-passes:status
```

Arguments: `[--json] [--verbose]`

Source: [status.md:1](<E:/AI_Workspace/plugins/plugins/migration-passes/commands/status.md:1>) · SHA-256 `6a2ee14223a92684db6de419b609978bc14108df411bb43681a58caa88a5b992`

### `summarize-context`

Pre-process and compress transcript material into stable chunks. Prompts user for additional context sources before starting.

```text
/migration-passes:summarize-context
```

Arguments: `[--transcripts <path>] [--additional <path1,path2>]`

Source: [summarize-context.md:1](<E:/AI_Workspace/plugins/plugins/migration-passes/commands/summarize-context.md:1>) · SHA-256 `e5139a59089ee89651c571e623782216f26c979c086c480892cb6b0ab67b4dea`

## Skills

### `mp-architecture-approaches`

Use when the planning-architect agent needs to evaluate architecture and implementation approaches for a codebase migration. Provides detailed descriptions, fit criteria, key artifacts, and migration-specific guidance for ADR, Specification-driven, DDD, Hexagonal, CQRS, Clean Architecture, Strangler Fig, and Microservices Decomposition.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/migration-passes/skills/migration-passes/mp-architecture-approaches/SKILL.md:1>) · SHA-256 `2d475dbd26e2f315fe5d0f09fe375ecb2c49a3d8c030946943430e79b051a2d8`

### `mp-branch-thinking`

Use when a pass needs to explore multiple alternative solutions or approaches in parallel. Best for the planning architect, transcript analyzer, and any pass where multiple valid paths exist. Triggers on phrases like "explore alternatives", "consider options", "compare approaches", "what if".

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/migration-passes/skills/migration-passes/mp-branch-thinking/SKILL.md:1>) · SHA-256 `df7b6d7efd4d6b0b07e03adce9dc0c83bfcc343f7655f024e3a62d98772a5715`

### `mp-codebase-scanning`

Use when an agent needs to walk, catalog, hash, or analyze a codebase directory tree. Provides patterns for manifesting files, identifying hot spots, and computing change-detecting hashes.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/migration-passes/skills/migration-passes/mp-codebase-scanning/SKILL.md:1>) · SHA-256 `ada09b8b98142a049c73c2eacea314c5e284824e734cfdd152b026cf451fae6c`

### `mp-decision-making`

Use when the planning architect or any pass needs to choose among multiple approaches,
evaluate trade-offs, or make architecture decisions. Provides a route rubric matching
decision shapes to techniques (tribunal, adversarial, red-team, pre-mortem, council,
ladder-of-abstraction, calibration), structured verdict format, and chaining idioms.
Adapted from agent-foundry's decision-making plugin for migration planning contexts.


Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/migration-passes/skills/migration-passes/mp-decision-making/SKILL.md:1>) · SHA-256 `41c76bb0460396dc9cd3d140e319a46229e96ad8b649dedc58b47c4eba305f78`

### `mp-doc-patching`

Use when an agent needs to repair or update documentation. Provides rules for detecting stale references, missing coverage, intent mismatches, example drift, and parameter drift, plus risk assessment for doc changes.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/migration-passes/skills/migration-passes/mp-doc-patching/SKILL.md:1>) · SHA-256 `0ea392d2a287be8cdd0a67c411280b8af81a7d3a60ee399435162b87520bba9f`

### `mp-handoff-protocol`

Use when working with migration-passes pipeline agents. Provides the standard handoff block, planned change block, and checkpoint block formats that all passes must use for structured communication between agents.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/migration-passes/skills/migration-passes/mp-handoff-protocol/SKILL.md:1>) · SHA-256 `489ccf21fe4faf1e2de783bdf373483199c25d53ed09a246c61cad26cd5c7ee2`

### `mp-obsidian-export`

Use when exporting migration pipeline output to Obsidian-compatible markdown. Provides patterns for YAML frontmatter, wikilinks, map-of-content notes, and multi-format export.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/migration-passes/skills/migration-passes/mp-obsidian-export/SKILL.md:1>) · SHA-256 `a974f3fd82ae6287f075199e7c86aab1dc9d47f8a1a3260ecd589429ff8be13b`

### `mp-principles-extractor`

Use when the planning architect needs to extract project principles from codebase
patterns, conventions, and existing architecture for approach fit scoring. Adapts
the constitution plugin's checklist-generator pattern for migration contexts.
Extracts principles from code, configs, tests, and docs, then distills them into
actionable verification items with semantic deduplication.


Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/migration-passes/skills/migration-passes/mp-principles-extractor/SKILL.md:1>) · SHA-256 `bf4d380d2a16b412dfd71fd80022620ed19836b462ee1941d8f99ff1aaf1940e`

### `mp-react-pattern`

Use when a pass needs to iteratively reason about a problem, take action based on that reasoning, observe the result, and repeat. Best for the migration agent, doc patcher, and any pass where the agent must act and then evaluate the outcome before proceeding. Triggers on phrases like "iterate", "try and verify", "act and observe", "reason-act-observe".

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/migration-passes/skills/migration-passes/mp-react-pattern/SKILL.md:1>) · SHA-256 `15ec415228f23499da822638f9a976c067198402bb531a7737dc766a2949ffbf`

### `mp-report-format`

Use when generating or consuming JSON reports in .migration-passes/. Defines the
consistent schema for all pass reports including status fields, timestamps, error
reporting, cross-pass linking, and byline requirements. Every pass must follow this
format for interoperability between agents. Includes report templates for each pass
that agents should copy and fill in.


Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/migration-passes/skills/migration-passes/mp-report-format/SKILL.md:1>) · SHA-256 `f48fcdf8c9df5164cc61dfef69ed1bcfbc45de06f41572be37ac3b1b117a2755`

### `mp-safe-operations`

Use when performing file modifications during migration, documentation patching, or any
pass that alters files. Provides backup-before-modify, atomic write, and rollback patterns
adapted from agent-foundry's atomic.sh for Windows-compatible Python. Invoke before any
Write or Edit operation that modifies codebase files to ensure safe rollback on failure.


Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/migration-passes/skills/migration-passes/mp-safe-operations/SKILL.md:1>) · SHA-256 `a4154d538ca0ca0bcc1293505277967cb2360dc84d016067bb6caf29c8e99e89`

### `mp-self-reflection`

Use when a pass needs to review and critique its own output before finalizing. Best for doc patching, gap auditing, and migration where quality control matters. Triggers on phrases like "review your work", "verify accuracy", "self-critique", "double-check".

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/migration-passes/skills/migration-passes/mp-self-reflection/SKILL.md:1>) · SHA-256 `29341d30a4385ac8c99fc2777a5bcc8a9aa7248e5b42ea80dfe9ba385f8e732c`

### `mp-sequential-thinking`

Use when a pass needs to reason through complex problems step-by-step in a linear, ordered fashion. Best for inventory mapping, gap auditing, and other passes where order matters and each step depends on the previous one. Triggers on phrases like "analyze sequentially", "step by step", "chain of thought", "linear reasoning".

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/migration-passes/skills/migration-passes/mp-sequential-thinking/SKILL.md:1>) · SHA-256 `b81c5e20020b2ecb524579674939b9ebbdf91ff813f1c1fa4e210f4e0ad5ec56`

## Agents

### `context-summarizer`

Pre-processes and compresses transcript material into stable chunks before deeper extraction. Prompts user for additional context sources. Use when processing meeting notes, conversation logs, or other unstructured context for the migration pipeline.

Source: [context-summarizer.md:1](<E:/AI_Workspace/plugins/plugins/migration-passes/agents/context-summarizer.md:1>) · SHA-256 `3b83fc6b1912abd6221ece224a150ce3e9429d99d6691a914dfe45d5213b5856`

### `doc-patcher`

Walks the codebase and patches documentation to match current implementation and known intended behavior. Emits planned change blocks before modifying. Use when repairing documentation drift during the migration pipeline.

Source: [doc-patcher.md:1](<E:/AI_Workspace/plugins/plugins/migration-passes/agents/doc-patcher.md:1>) · SHA-256 `bdce7ca4b9b4b6e76f4543db22e38590f1e325e2ce920b5dc91597c977d88d15`

### `gap-auditor`

Performs a read-only structural audit after documentation has been repaired. Identifies undocumented exports, dead code, import integrity issues, type inconsistencies, test coverage gaps, and config/env drift. Use when auditing structural gaps in the migration pipeline.

Source: [gap-auditor.md:1](<E:/AI_Workspace/plugins/plugins/migration-passes/agents/gap-auditor.md:1>) · SHA-256 `cfcdb4178201aa94b0ce8d1d67dfd1390ee51cb96afa4a80882ed60b38abab89`

### `inventory-mapper`

Maps both NEW and OLD codebases before any other pass runs. Walks directories, builds file manifests, identifies hot spots, and records hashes for change detection. Use when starting the migration pipeline to catalog both codebases.

Source: [inventory-mapper.md:1](<E:/AI_Workspace/plugins/plugins/migration-passes/agents/inventory-mapper.md:1>) · SHA-256 `bb2faeda5cd3c414abe0675ed01fd321872212711d30f6213183da9122e4813f`

### `migration-agent`

Compares OLD and NEW codebases, identifies missing or partial legacy features, and ports them safely into the NEW codebase. Three-phase process: inventory diff, porting plan, execution. Uses small-change stacking with build integrity verification.

Source: [migration-agent.md:1](<E:/AI_Workspace/plugins/plugins/migration-passes/agents/migration-agent.md:1>) · SHA-256 `1d56e6f3be3188129fb6022fb3e8b6ab79fe82559d4a5659414931673ce2e547`

### `obsidian-exporter`

Exports completed migration run into Obsidian-compatible markdown with YAML frontmatter, wikilinks, and a map-of-content note. Also supports JSON, CSV, and Parquet exports. Use when exporting migration pipeline output to an Obsidian vault.

Source: [obsidian-exporter.md:1](<E:/AI_Workspace/plugins/plugins/migration-passes/agents/obsidian-exporter.md:1>) · SHA-256 `5358a52a11df40aaec69ffc7935445db1b8a6a15b789767e26bae8312e100b66`

### `pipeline-runner`

Orchestrates the full migration pipeline by running all 8 passes sequentially. Validates handoffs between passes, handles interactive and automated modes, and produces a final pipeline summary. Use when running the complete migration pipeline with /migration-passes:run.

Source: [pipeline-runner.md:1](<E:/AI_Workspace/plugins/plugins/migration-passes/agents/pipeline-runner.md:1>) · SHA-256 `a359bef0abf6d656baad9994a902f547876d762bac5f5acbb420cf4f2008af6d`

### `planning-architect`

Heavily interactive interview-based pass that evaluates architecture approaches (ADR, DDD, Hexagonal, CQRS, Clean Architecture, Strangler Fig, Spec-driven, etc.) and produces a migration plan. Uses the Plannotator plugin for interactive review and annotation. Use when planning a migration strategy or evaluating architecture approaches.

Source: [planning-architect.md:1](<E:/AI_Workspace/plugins/plugins/migration-passes/agents/planning-architect.md:1>) · SHA-256 `9e51b93ed4828cf9106ed14a12dc7635c9cfce6246345ec4228d43b04e5d2b14`

### `transcript-analyzer`

Extracts structured intelligence from transcript chunks — architectural decisions, known issues, intended behavior, documentation gaps, deferred work, feature mentions, and conflicts. Use when analyzing processed transcripts for the migration pipeline.

Source: [transcript-analyzer.md:1](<E:/AI_Workspace/plugins/plugins/migration-passes/agents/transcript-analyzer.md:1>) · SHA-256 `2984ba0b26301280fe78078580cfdd91bcf5855251f956e3ac3e3faa3036d079`

## Cli Entries

No entries found in the inspected declarations.

## Scripts

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
python "E:/AI_Workspace/plugins/plugins/migration-passes/scripts/bump-revision.py" --help
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

Source: [bump-revision.py:1](<E:/AI_Workspace/plugins/plugins/migration-passes/scripts/bump-revision.py:1>) · SHA-256 `375d53d677854bb9899b26659201dc02b952f8d51c7be1695c8e0db06a7302ec`

### `scripts/checkpoint-notify.py`

Notify about checkpoint progress in migration-passes pipeline.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [checkpoint-notify.py:1](<E:/AI_Workspace/plugins/plugins/migration-passes/scripts/checkpoint-notify.py:1>) · SHA-256 `13223417b42f16228087b22c5a7b6c7240e6f8d490f0d565f33028113220c2d1`

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
python "E:/AI_Workspace/plugins/plugins/migration-passes/scripts/lib/formatting.py" --help
```

Declared arguments: `--json-output`, `--json-start`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --json-start | store_true | False | — | Enable JSON mode |
| --json-output | store_true | False | — | Print collected JSON results |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [formatting.py:1](<E:/AI_Workspace/plugins/plugins/migration-passes/scripts/lib/formatting.py:1>) · SHA-256 `bfdcd56ea996014395337dcc0c13bb5bbfb3b0064a3dfd0cc7b20698cf27a48f`

### `scripts/manifest-diff.py`

Compare two inventory manifests to identify differences.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [manifest-diff.py:1](<E:/AI_Workspace/plugins/plugins/migration-passes/scripts/manifest-diff.py:1>) · SHA-256 `c7b6a219243c5fd7d325c21baeb9c521fa86fe0e7e7c4acd2634171641e76f6b`

### `scripts/pipeline-status.py`

pipeline-status.py - Display migration-passes pipeline status.

Shows which passes are complete, partial, blocked, or missing,
with timestamps and next-step guidance.

Usage:
    python pipeline-status.py [--json] [--verbose] [--dir <project_dir>]

```text
python "E:/AI_Workspace/plugins/plugins/migration-passes/scripts/pipeline-status.py" --help
```

Declared arguments: `--dir`, `--json`, `--verbose`, `-v`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --json | store_true | False | — | Output as JSON |
| --verbose, -v | store_true | False | — | Show detailed information |
| --dir | — | False | — | Path to project directory containing .migration-passes/ |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [pipeline-status.py:1](<E:/AI_Workspace/plugins/plugins/migration-passes/scripts/pipeline-status.py:1>) · SHA-256 `f5db4710a6082b1a735aeef8d4958550b9d04dfd75ec5aaee09c2f687390ea88`

### `scripts/port-to-opencode.py`

port-to-opencode.py - Convert Claude Code plugin agents/commands to OpenCode format.

Usage:
    python port-to-opencode.py

This script reads agents/*.md and commands/*.md from the Claude plugin format
and writes them to .agents/agents/ and .agents/commands/ in OpenCode format.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [port-to-opencode.py:1](<E:/AI_Workspace/plugins/plugins/migration-passes/scripts/port-to-opencode.py:1>) · SHA-256 `1784d1cfd4f18db0b82cc2d02abbd7be80a43675b56fc58c685b1f700d8bf9cc`

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

Source: [restructure-claude-skills.py:1](<E:/AI_Workspace/plugins/plugins/migration-passes/scripts/restructure-claude-skills.py:1>) · SHA-256 `ccd3f51b01c75f593bfecac78f6d6befe2409ac09d0a0e7a62a117bbc4ed6af3`

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

Source: [restructure-skills.py:1](<E:/AI_Workspace/plugins/plugins/migration-passes/scripts/restructure-skills.py:1>) · SHA-256 `28defe4c196db9764bad94e07676bb6e4cbf4783cc81e1be85a32853b2ccdd06`

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
python "E:/AI_Workspace/plugins/plugins/migration-passes/scripts/safe-operations.py" --help
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

Source: [safe-operations.py:1](<E:/AI_Workspace/plugins/plugins/migration-passes/scripts/safe-operations.py:1>) · SHA-256 `95571c333234822e1074e572570c6a529b751313f065c49624ffb4550664816b`

### `scripts/validate-handoff.py`

Validate handoff block format and byline presence in migration-passes output files.

```text
python "E:/AI_Workspace/plugins/plugins/migration-passes/scripts/validate-handoff.py" --help
```

Declared arguments: `--check-bylines`, `--file`, `--store`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --file | str | False | — | File to validate |
| --check-bylines | store_true | False | — | Check bylines in runtime store |
| --store | str | False | — | Runtime store directory |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [validate-handoff.py:1](<E:/AI_Workspace/plugins/plugins/migration-passes/scripts/validate-handoff.py:1>) · SHA-256 `392a6a9657fdeb71c84cb865586e11f4ab0da3f56390896d5c0539ff7e9c15f3`

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
python "E:/AI_Workspace/plugins/plugins/migration-passes/scripts/validate-pipeline-state.py" --help
```

Declared arguments: `--check-complete`, `--dir`, `--json`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --dir | — | False | — | Path to project directory containing .migration-passes/ |
| --check-complete | int | False | — | Check if a specific pass has completed (0-7) |
| --json | store_true | False | — | Output as JSON |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [validate-pipeline-state.py:1](<E:/AI_Workspace/plugins/plugins/migration-passes/scripts/validate-pipeline-state.py:1>) · SHA-256 `6f1ed2c1c4475bce46c7ad3edafd0d436cb6e2409bd3b746187bedadb7c9b9e0`

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
python "E:/AI_Workspace/plugins/plugins/migration-passes/scripts/validate-prerequisites.py" --help
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

Source: [validate-prerequisites.py:1](<E:/AI_Workspace/plugins/plugins/migration-passes/scripts/validate-prerequisites.py:1>) · SHA-256 `ac73dc0c616b590163b608dedff3028a5765322fe36376f40ea991f696accaee`

## Mcp Tools

No entries found in the inspected declarations.

## Mcp Servers

No entries found in the inspected declarations.

## Incomplete checks

- skills/migration-passes/mp-codebase-scanning/scripts/hash-files.py: SyntaxError

Back to [[Code/wiki/plugin-inventory|Plugin inventory]].

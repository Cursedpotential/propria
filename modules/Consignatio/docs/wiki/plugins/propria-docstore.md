---
title: "propria-docstore"
type: tool-reference
status: SOURCE_VERIFIED
date: 2026-10-04
generated_by: "Codex / GPT-6"
revision: 1
tags: [propria, tools, wiki]
---

# propria-docstore

> _Byline: Codex · GPT-6 · 2026-10-04 — generated from the cited sources._

Universal Propria documentation retrieval, notes, decisions, revisions, CDC verification, project registry, graph resources, and memory access (moved from the retired probata marketplace 2026-09-15). 0.6.4 (2026-09-16): fixed the memory half, which was broken end to end — deployed the never-before-applied memory schema/functions to the surreal-case VPS instance (ns probata_memory, db memory), added the missing surreal-ns/surreal-db headers to the memory MCP entry in .mcp.json (root cause of every 'Specify a namespace to use' error), made preflight's memory=up check an authenticated fn::memory_stats call instead of a bare /health ping, and corrected the memory skill/agent docs (no longer provisional; documented the $ql sentinel needed for fn::recall's $vec and fn::reflect's $since). 0.6.3 (2026-09-16): fixed stale project settings.json marketplace refs, absolute script paths, MCP record-id $ql examples, fn::handoff_write's dangerous same-domain-set default supersede (plus the control tool's own empty-array/single-row bugs), docs_register misuse guidance, and wired Codex/OpenCode/Gemini to ContextForge.

Source: `E:/AI_Workspace/plugins/plugins/propria-docstore`. Version: `0.9.1`.
Registered: `True`. Installed manifests: .claude.

## How to invoke it

Slash commands run inside an agent app. Terminal commands run in PowerShell. MCP tools require an attached server and are called by the agent or an MCP client. A skill is an instruction package, not a standalone executable.

Descriptions below are extracted from source metadata/docstrings. This is discovery evidence, not a claim every service invocation passed.

## Commands

No entries found in the inspected declarations.

## Skills

### `adr`

Read or update authoritative ADR records and migrate legacy decisions (alias of the decisions skill).

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/propria-docstore/skills/adr/SKILL.md:1>) · SHA-256 `39cf132bb5da02fc65e8a64c63b923f483b20b04af82c8a3c4fd7396e3bf3346`

### `decisions`

Read or update authoritative ADR records and migrate legacy decisions.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/propria-docstore/skills/decisions/SKILL.md:1>) · SHA-256 `afe331cc341e27f2d7837a3a35796f7ee4a460ef03da16153840e05108ae3d2f`

### `diagnostics`

Check endpoint identity, schema, run status and optional integrations.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/propria-docstore/skills/diagnostics/SKILL.md:1>) · SHA-256 `2f4953ebe44de05a994e79e3e90546e0c73678162f6f3e01c701daf40db6ccc9`

### `docs`

Search and retrieve remote project documentation.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/propria-docstore/skills/docs/SKILL.md:1>) · SHA-256 `dba308ebcd8890f86594f36b0e392b2722cab12d61b41a920ddb2afc23e466fe`

### `docs-write`

Write documentation or a governed file-less note.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/propria-docstore/skills/docs-write/SKILL.md:1>) · SHA-256 `c9be33d519ee94daf27425c7d78351ae6e6c0aae210aae8c47fdcc29108aacb9`

### `docstore`

General entry point for the Propria Docstore plugin. Use when the user asks how to use Docstore, what it can do, to search project documentation, recall decisions or memory, prepare a handoff, inspect document relationships, check status, or manage the documentation index. Routes to focused skills and discovers remote operations on demand.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/propria-docstore/skills/docstore/SKILL.md:1>) · SHA-256 `149baafaa56149fb24f93895b2b9f82a7173ce526870ff0f102fae1f54018993`

### `docstore-status`

Read current remote Docstore health and source attribution (alias of the status skill).

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/propria-docstore/skills/docstore-status/SKILL.md:1>) · SHA-256 `09ab3ac7a85dc563b078b3f5bc81c0e7b8ad54f1f4c45d66cfc60a54dc55e8b4`

### `docstore-sync`

Synchronize the exact five docs roots and run incremental indexing (alias of the index skill).

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/propria-docstore/skills/docstore-sync/SKILL.md:1>) · SHA-256 `67b84a4e25400aa1ee995738521babe039e56c432cc2cb1562b8862df8b12d5c`

### `docstore-upgrade`

Plan, apply and verify versioned Docstore upgrades (alias of the upgrade skill).

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/propria-docstore/skills/docstore-upgrade/SKILL.md:1>) · SHA-256 `8386d75df407d3216aaf6ac20ecfe11a82f9be6ac168d915f1bd901779d25a40`

### `get`

Fetch a full document by stable record ID.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/propria-docstore/skills/get/SKILL.md:1>) · SHA-256 `a05be144bd84f7e13ebd5f9d0065e8b85c984ab1b378349079dc56ca3ce47c95`

### `graphs`

Create and traverse the cross-document SurrealDB knowledge graph.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/propria-docstore/skills/graphs/SKILL.md:1>) · SHA-256 `9035f91488d140ed2cd5601316fb4f043f5fbff4ffb48b142855e45bbca2ee16`

### `handoff`

Persist a governed session handoff and verify it.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/propria-docstore/skills/handoff/SKILL.md:1>) · SHA-256 `03d3fe1af37f5cf990ccad6ccc1263496ffe0d99e36187c18506e28fe584bc47`

### `index`

Keep the docs index current through the required local live updater; sync and run one-shot indexing only for a forced reprocess.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/propria-docstore/skills/index/SKILL.md:1>) · SHA-256 `4e59dfb808dd58fca26838edbf53f673c7e3f9310adfd411680dc5c2d186a5e7`

### `memory`

Recall across local histories and independent remote memory, and write durable claims to the shared remote memory.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/propria-docstore/skills/memory/SKILL.md:1>) · SHA-256 `8a923a370cb444c327dc0f356c5ce5cf045a40356b598ac312457edb06e2056f`

### `propria-search`

Find documentation or code with the correct independent system.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/propria-docstore/skills/propria-search/SKILL.md:1>) · SHA-256 `345d8eab13f608dda5a5d2b485dd9199d5fd0471641586333e5d510cb0aa49b5`

### `query`

Inspect structured Docstore records and native graphs.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/propria-docstore/skills/query/SKILL.md:1>) · SHA-256 `8fcf2c937d7cd38a6720e535f0e19ebff8005d11b285682639cc98ce36167d6d`

### `recall`

Federate Claude, Codex, read-memories, memsearch, CNF, .remember, remote memory and code where available.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/propria-docstore/skills/recall/SKILL.md:1>) · SHA-256 `5267a6e05dd060486ba831d62b4c263d9d128cbc8548735a41e8c52dfc467fd4`

### `recall-adr`

Recall authoritative ADR records by number, title or topic (alias of the decisions skill).

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/propria-docstore/skills/recall-adr/SKILL.md:1>) · SHA-256 `de00b8182c7e9b58971e72321a71b8368be79764831af8b90f132b70ac712398`

### `recall-doc`

Search and retrieve remote project documentation (alias of the docs skill).

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/propria-docstore/skills/recall-doc/SKILL.md:1>) · SHA-256 `f1fd0dedb8af9f15c400592e83f8c4cbf3c1e5e0494781f14b6cdfe2c7948089`

### `reconcile`

Compare documentation, implementation and memory without merging stores.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/propria-docstore/skills/reconcile/SKILL.md:1>) · SHA-256 `0f836c97932d8abeb4eebc380873148da380682739c586c7c99ead09eedd29a2`

### `search`

Search remote documentation with server-side processing.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/propria-docstore/skills/search/SKILL.md:1>) · SHA-256 `d6e9dda995e4541d35c33fb26220fe1c1efb1312a480896dd26cc9e9e20138f0`

### `status`

Read current remote Docstore health and source attribution.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/propria-docstore/skills/status/SKILL.md:1>) · SHA-256 `821933bd8a28dd793bb752b1eb0c14295edcd88c2b7dda568cd2a03678c9ca3a`

### `todo`

Inspect and record scoped work without losing provenance.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/propria-docstore/skills/todo/SKILL.md:1>) · SHA-256 `ef35263cd8756b718c6901026c6fb34a8970bfe8b4d6e17092dfcf961da152ab`

### `update-adr`

Update an authoritative ADR record through the governed ADR workflow (alias of the decisions skill).

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/propria-docstore/skills/update-adr/SKILL.md:1>) · SHA-256 `aa5329d9dea74a93c73db8078d0b7317e5abf3dbcb23deaa824a9e44eeefb69c`

### `upgrade`

Plan, apply and verify versioned Docstore upgrades.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/propria-docstore/skills/upgrade/SKILL.md:1>) · SHA-256 `2222ed1f84e4021f064cbabf97ce32d7e066b99191786511c7d76c1e13adb918`

## Agents

### `docstore-librarian`

Retrieve and maintain project documentation through hosted Docstore ctl, preserving IDs, history and source attribution.

Source: [docstore-librarian.md:1](<E:/AI_Workspace/plugins/plugins/propria-docstore/agents/docstore-librarian.md:1>) · SHA-256 `aa828336633fafc88e1f4490e1ad4b417dd7e92a55df05b63fd6d788723bbea3`

### `docstore-reconciler`

Compare historical documentation and current implementation with explicit provenance, uncertainty and user authority.

Source: [docstore-reconciler.md:1](<E:/AI_Workspace/plugins/plugins/propria-docstore/agents/docstore-reconciler.md:1>) · SHA-256 `ca73ecacea10fe6d87bc188ed51712e14f366f09b1b8a6021335f29cfd30be60`

### `memory-curator`

Recall and record explicitly authorized shared-memory claims through hosted ctl while preserving scope, provenance and conflicts.

Source: [memory-curator.md:1](<E:/AI_Workspace/plugins/plugins/propria-docstore/agents/memory-curator.md:1>) · SHA-256 `c2af903c50aee9a1154041c821505b0c5924dcc0c83504402a5c3ea856cf6b86`

## Cli Entries

No entries found in the inspected declarations.

## Scripts

### `client.py`

Portable thin client. All Docstore operations use the mandatory hosted ctl MCP.

```text
python "E:/AI_Workspace/plugins/plugins/propria-docstore/client.py" --help
```

Declared arguments: `--apply`, `--hold`, `--hold-file`, `--json`, `--retract`, `--root`, `tool`

Declared subcommands:

| Command | Source help |
|---|---|
| `catalog` | — |
| `call` | — |
| `sync` | — |

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| tool | — | positional | — | — |
| --json | — | False | — | — |
| --root | — | True | — | — |
| --apply | store_true | False | — | — |
| --retract | append | False | — | retract this exact mirror document (repeatable); nothing is retracted otherwise |
| --hold | append | False | — | keep the mirror version of matching project/path keys (glob, repeatable) |
| --hold-file | — | False | — | file of --hold patterns, one per line |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [client.py:1](<E:/AI_Workspace/plugins/plugins/propria-docstore/client.py:1>) · SHA-256 `20368a98e6c006b9289639e80a68a3b5befeaeaa1916a7d8fd59635aa1b88a11`

### `federation.py`

Bounded local memory federation. No indexing, provider installation or hidden fallback.

```text
python "E:/AI_Workspace/plugins/plugins/propria-docstore/federation.py" --help
```

Declared arguments: `--adapters`, `--code`, `--root`, `--since`, `--until`, `query`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| query | — | positional | — | — |
| --root | — | False | — | — |
| --code | store_true | False | — | — |
| --since | — | False | — | — |
| --until | — | False | — | — |
| --adapters | — | False | — | — |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [federation.py:1](<E:/AI_Workspace/plugins/plugins/propria-docstore/federation.py:1>) · SHA-256 `84751aa496dc7a8ab35a021f41c011ea70836fabb39c0a3313d270d1c49515d6`

## Mcp Tools

No entries found in the inspected declarations.

## Mcp Servers

### `ctl`

http

Validation: configured; health not inferred.

Source: [.mcp.json:1](<E:/AI_Workspace/plugins/plugins/propria-docstore/.mcp.json:1>) · SHA-256 `22a313d21b7fb9a09f7df1df5468b4246d97fd78b341c1995ccff5af9add4a7f`


## Existing hosted MCP exposure

The live ContextForge server associated with this plugin exposes the following names. Complete argument schemas and descriptions are in [[Code/wiki/contextforge-tools]]. This is registry discovery, not invocation proof.

- `docstore-capabilities`
- `docstore-get`
- `docstore-health`
- `docstore-query`
- `docstore-search`

Back to [[Code/wiki/plugin-inventory|Plugin inventory]].

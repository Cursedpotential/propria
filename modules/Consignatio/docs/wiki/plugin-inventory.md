---
title: "Custom plugins and tools"
type: tool-reference
status: SOURCE_VERIFIED
date: 2026-10-04
generated_by: "Codex / GPT-6"
revision: 1
tags: [propria, tools, wiki]
---

# Custom plugins and tools

> _Byline: Codex · GPT-6 · 2026-10-04 — generated from the cited sources._

30 plugin sources inspected: 30 marketplace registrations and 0 additional direct manifests.

## Choose an entry point

- [[Code/wiki/case-bible-search|Search Case Bible content]]
- [[Code/wiki/search-and-recall|Terminal commands for code search and memory recall]]
- [[Code/wiki/atomic-tools|Atomic tools from the existing gateway]]
- [[Code/wiki/contextforge-tools|Tools exposed through ContextForge]]

- [[Code/wiki/mcp-servers|Virtual servers and tool associations]]
- [[Code/wiki/docstore-operations|Docstore operation schemas]]

Declared surfaces: 350 skills, 124 slash commands, 25 agents, 63 local MCP declarations, 201 Python helpers/CLI candidates and 2 launcher/entry-point declarations.

Live registries returned 55 gateway tools, 10 virtual servers and 212 ContextForge tools. These counts overlap local declarations and must not be added into a unique-tool total.

| Plugin | Version | Skills | Slash commands | MCP declarations | CLI entries |
|---|---|---:|---:|---:|---:|
| [[Code/wiki/plugins/app-planning-handoff\|app-planning-handoff]] | 1.1.0 | 6 | 0 | 0 | 0 |
| [[Code/wiki/plugins/case-bible\|case-bible]] | 0.6.4 | 21 | 11 | 0 | 0 |
| [[Code/wiki/plugins/cf\|cf]] | 1.1.0 | 12 | 2 | 0 | 0 |
| [[Code/wiki/plugins/claude-never-forgets\|claude-never-forgets]] | 1.0.2 | 1 | 4 | 0 | 0 |
| [[Code/wiki/plugins/claude-reflect\|claude-reflect]] | 1.5.0-propria.2 | 0 | 3 | 0 | 0 |
| [[Code/wiki/plugins/codebase\|codebase]] | 1.1.0 | 7 | 0 | 0 | 0 |
| [[Code/wiki/plugins/coolify-write\|coolify-write]] | 1.4.1 | 1 | 0 | 43 | 0 |
| [[Code/wiki/plugins/crewai\|crewai]] | 1.1.0 | 5 | 0 | 0 | 0 |
| [[Code/wiki/plugins/family-court-toolkit\|family-court-toolkit]] | 3.3.0 | 36 | 7 | 0 | 0 |
| [[Code/wiki/plugins/finance\|finance]] | 1.1.0 | 14 | 0 | 0 | 0 |
| [[Code/wiki/plugins/graphrag\|graphrag]] | 0.2.0 | 7 | 3 | 0 | 0 |
| [[Code/wiki/plugins/law\|law]] | 1.1.0 | 19 | 0 | 0 | 0 |
| [[Code/wiki/plugins/llm-probes\|llm-probes]] | 1.2.1 | 6 | 0 | 0 | 0 |
| [[Code/wiki/plugins/memsearch\|memsearch]] | 0.4.18-local.16 | 3 | 0 | 4 | 0 |
| [[Code/wiki/plugins/mental-health\|mental-health]] | 1.1.1 | 4 | 0 | 0 | 0 |
| [[Code/wiki/plugins/michigan-construction-project\|michigan-construction-project]] | 0.2.0 | 1 | 0 | 0 | 0 |
| [[Code/wiki/plugins/migration-passes\|migration-passes]] | 0.5.0 | 13 | 10 | 0 | 0 |
| [[Code/wiki/plugins/n8n\|n8n]] | 1.1.0 | 4 | 0 | 0 | 0 |
| [[Code/wiki/plugins/ops\|ops]] | 1.1.1 | 20 | 0 | 0 | 0 |
| [[Code/wiki/plugins/platform-engineering-skills\|platform-engineering-skills]] | 0.1.0 | 8 | 0 | 0 | 0 |
| [[Code/wiki/plugins/portkey\|portkey]] | 1.1.0 | 4 | 0 | 0 | 0 |
| [[Code/wiki/plugins/propria-docstore\|propria-docstore]] | 0.9.1 | 25 | 0 | 0 | 0 |
| [[Code/wiki/plugins/propria-toolbox\|propria-toolbox]] | 0.1.0 | 1 | 0 | 0 | 0 |
| [[Code/wiki/plugins/scout\|scout]] | 2.0.1 | 1 | 0 | 0 | 1 |
| [[Code/wiki/plugins/search\|search]] | 1.2.2 | 2 | 0 | 16 | 1 |
| [[Code/wiki/plugins/semantica-skill-router\|semantica-skill-router]] | 1.0.1 | 1 | 0 | 0 | 0 |
| [[Code/wiki/plugins/sequential-react-ship\|sequential-react-ship]] | 1.0.0 | 8 | 0 | 0 | 0 |
| [[Code/wiki/plugins/surrealdb\|surrealdb]] | 1.0.0 | 13 | 0 | 0 | 0 |
| [[Code/wiki/plugins/tanstack\|tanstack]] | 1.1.0 | 12 | 0 | 0 | 0 |
| [[Code/wiki/plugins/think\|think]] | 2.0.1 | 95 | 84 | 0 | 0 |

## Complete machine-readable inventory

[Tool inventory JSON](assets/tool-inventory.json) contains full descriptions, parameters and source identities. [Human commands CSV](assets/human-commands.csv) is filterable by plugin and command type.

## Refresh and validation

Generated descriptions come from source docstrings/metadata or existing registries. Update those sources first, then regenerate. Missing docstrings and absent input contracts remain explicit findings.

Source generator: [generate_wiki_inventory.py:1](<E:/AI_Workspace/Projects/Propria/modules/Consignatio/casebible/tools/generate_wiki_inventory.py:1>) · SHA-256 `3387006202bd60a86b7cd543d0c7cea1dd61f8d0dd4b7d72c44d80ecb4f7beb7`

```powershell
python "E:/AI_Workspace/Projects/Propria/modules/Consignatio/casebible/tools/generate_wiki_inventory.py" --output "<wiki-output-folder>" --live
```

This is a bounded documentation inventory, not a new operational catalog or index. Source inspection does not validate credentials, live service health, semantic relevance or every mutation command. The search guide records the commands tested read-only.

Marketplace evidence: [marketplace.json:1](<E:/AI_Workspace/plugins/.claude-plugin/marketplace.json:1>) · SHA-256 `a5f146241065639c6d1f7a30a23520fabcfbfd830f03ff7ed5a4125dfa0b2c9c`

---
title: Case Bible tools wiki
type: moc
status: source-and-help-verified
date: 2026-10-04
byline: Codex / GPT-6, with Documentation agent GPT-6.1
revision: 1
tags: [case-bible, tools, wiki]
---

# Case Bible tools wiki

> _Byline: Codex · GPT-6 · 2026-10-04, with Documentation agent GPT-6.1._

Start here to find a tool, search your material, or run a command yourself. This wiki lives in the existing **`Code/wiki/`** section of `b2:salem-data/consignatio/casevault/`. It uses Markdown, YAML frontmatter and Obsidian wikilinks. The source pages remain in Propria under `modules/Consignatio/docs/wiki/`; the B2 pages are a published documentation copy.

## What do you want to do?

| Your task | Open this page |
|---|---|
| Search documents, messages or AI chat exports | [[Code/wiki/case-bible-search\|Case Bible content search]] |
| Search code or recall earlier decisions from a terminal | [[Code/wiki/search-and-recall\|Search and recall commands]] |
| Understand the catalog, inventory and Parquet lake | [[Code/wiki/catalog-guide\|Catalog and lake guide]] |
| Find a custom plugin and everything it exposes | [[Code/wiki/plugin-inventory\|Custom plugin inventory]] |
| Find a parser, extractor, inspection or repair tool | [[Code/wiki/atomic-tools\|Atomic tools]] |
| Find an MCP tool and its input fields | [[Code/wiki/contextforge-tools\|ContextForge tool reference]] |
| See which MCP server exposes a tool | [[Code/wiki/mcp-servers\|MCP server map]] |
| Use a Docstore operation behind its discovery interface | [[Code/wiki/docstore-operations\|Docstore operations]] |

## Start searching

Inside the agent app:

```text
/cb-vsearch "problems arranging school pickup"
/cb-vsearch "school pickup" --corpus messages
/cb-vsearch "parenting time" --corpus documents --mode keyword
/cb-vsearch "school pickup" --corpus messages --presentation compact
```

From PowerShell, the same Case Bible reader has an actual terminal entry point:

```powershell
python 'E:/AI_Workspace/plugins/plugins/case-bible/tools/cb_vsearch.py' 'school pickup' --corpus messages
```

The other Search plugin also has a terminal launcher:

```powershell
$search = 'E:/AI_Workspace/plugins/plugins/search/search.cmd'
& $search --help
& $search semantic --help
& $search recall --help
```

See [[Code/wiki/case-bible-search]] for filters, readable/compact/JSON output and source references. See [[Code/wiki/search-and-recall]] for the full terminal command reference, configuration names, write effects and exact help checks.

## What the inventory includes

The generated [[Code/wiki/plugin-inventory]] covers the Propria marketplace source manifests, declared skills, slash commands, agent definitions, MCP tools, Python CLI candidates and launcher entry points. It also reads the **existing** atomic-tool gateway and ContextForge registries, and documents Docstore's discoverable operation schemas.

- [Full structured tool inventory](assets/tool-inventory.json)
- [Human-callable commands, filterable CSV](assets/human-commands.csv)
- [Discovered Docstore input contracts](assets/docstore-operation-schemas.json)
- [Publication file hashes](assets/publication-manifest.csv)

The machine inventory preserves complete extracted descriptions and source path/line/SHA-256 identities. The concise landing pages link to individual plugin pages so you do not have to read a single enormous README.

## How to read the checks

**Source verified** means the documentation was extracted or checked against the implementation or registration. **Help verified** means the actual CLI help ran successfully. **Live reader verified** applies to the Case Bible search checks cited in its page. Finding a registration is not proof every live tool invocation works.

The inventory names malformed metadata, missing descriptions and missing argument contracts where encountered. Query results cover indexed content and bounded candidate windows; they do not prove that every stored object has been indexed.

## Existing structure governs placement

Use the vault's existing domains and their existing capitalization. **`Recovered/` is the recovery-material domain.** Do not create a top-level `recovery/` beside it. This documentation task adds pages only beneath `Code/wiki/`; it does not relocate source material or reorganize the old OneDrive vault.

## Source and validation chain

Each reference page identifies the implementation, command help or registry behind its claims. The generated inventory includes source hashes; `assets/publication-manifest.csv` identifies the bytes published to B2. The publication receipt records independent remote readback. The authoritative catalog guide is `modules/Consignatio/docs/CASE-BIBLE-CATALOG-GUIDE.md`, surfaced here as [[Code/wiki/catalog-guide]].

The source generator is `modules/Consignatio/casebible/tools/generate_wiki_inventory.py`. Descriptions are refreshed from manifests/docstrings and existing registries, rather than edited into a competing tool registry.

---
title: Case Bible content search
date: 2026-10-04
status: source-verified
byline: Documentation agent GPT-6.1
tags:
  - case-bible
  - search
---

# Case Bible content search

Find passages in indexed documents, messages and AI chats with **cb-vsearch**. Start with a question, narrow the results, then follow the citations back to the source. For inventory questions, use the [[Code/wiki/catalog-guide#Search for content|Case Bible catalog guide]]. [C1, C2]

Related notes: [[Code/wiki/README]], [[Code/wiki/plugin-inventory]], [[Code/wiki/search-and-recall]], [[Code/wiki/case-bible-search]].

Publication home: the existing B2 CaseVault `salem-data/consignatio/casevault/Code/wiki/` hierarchy. The parent handles publication and extends its existing `INDEX` scaffold. The old OneDrive vault is a legacy reference. [Owner clarification, 2026-10-04]

Contents: [Start searching](#start-searching) · [Responsibilities](#responsibilities) · [Options](#options) · [Read and cite results](#read-and-cite-results) · [Troubleshooting](#troubleshooting) · [Sources and validation](#sources-and-validation).

## Start searching

Prerequisites: the Case Bible plugin for the agent slash command, or local Python and SSH for terminal use. The terminal wrapper contacts the existing server reader on ovh-files. Hybrid retrieval additionally needs the server's configured embedding credentials. Configuration names are `CBCAT_PG_SSH`, `CBCAT_PG_SSH_KEY`, `CB_WEAVIATE_URL`, `NVIDIA_API_KEY` and `NVIDIA_API_KEY_FILE`; keep their values in the governed environment. [C2: `main`; C3: `search`]

1. Ask a concrete question in the agent interface:

   ```text
   /cb-vsearch "arranging school pickup"
   ```

2. Narrow the category and candidate text if needed:

   ```text
   /cb-vsearch "school pickup" --corpus messages --contains pickup
   /cb-vsearch "parenting schedule" --corpus documents --source Takeout
   /cb-vsearch "budget planning" --corpus chats --presentation compact
   ```

3. Read the excerpt, recorded source, date and result ID. Preserve the available hash and locator when using a passage. [C1; C3: `render`]

The slash command belongs to an agent plugin. It is not a PowerShell executable. The terminal equivalent uses the script directly:

```powershell
python E:/AI_Workspace/plugins/plugins/case-bible/tools/cb_vsearch.py --help
python E:/AI_Workspace/plugins/plugins/case-bible/tools/cb_vsearch.py "school pickup" --corpus messages --mode keyword
```

The default searches all three categories in hybrid mode and returns at most eight hits. `keyword` uses keyword retrieval without generating a query embedding. These search recipes are checked against the parser; only the help command was executed for this page. The supplied reader receipt records earlier live searches. [C2: `main`; C3: `search`; C4]

## Responsibilities

Each component has a different job. [C1: “How the tool works”; C3: `search`, `shape`]

| Component | Job |
|---|---|
| CocoIndex | Framework for the owning Intake discovery pipeline to process source changes and maintain derived index data. |
| Existing content indexes in Weaviate | Store and retrieve indexed content candidates. Document, message and chat scopes have separate collection contracts. |
| DuckDB inside the search reader | Filter a bounded candidate window and deduplicate identities in memory. It does not maintain a persistent result catalog. |
| cb-vsearch | Send a structured search request over SSH and present source-linked excerpts. |
| cbcat | Query file inventories and lake publications; an inventory entry does not prove content indexing. |

The query path is: question → query embedding in hybrid mode → existing indexes → bounded candidates → DuckDB filters → cited results. Embedding, retrieval and shaping run on ovh-files. Searching does not refresh the index or write source data. [C2: module docstring, `main`; C3: module docstring, `shape`, `search`]

CocoIndex's role in Intake does not establish that every historical message/chat producer uses it. Automatic synchronization and complete source coverage remain separate checks. For project guides and decisions use Docstore; for repository code use CCC, as explained in [[Code/wiki/search-and-recall]]. [C1: “How the tool works”, “Coverage and filter behavior”]

## Options

The terminal and slash command accept the same arguments. Values and boundaries below come from `main` and server `validate`; local `--help` passed on 2026-10-04. [C2, C3]

| Argument | Default | Use and limits |
|---|---|---|
| `query` | Empty | Natural-language or keyword question; required unless using `--id`. Server requires 1–500 characters. |
| `--corpus` | `all` | `all`, `documents`, `messages`, `chats`. |
| `--mode` | `hybrid` | `hybrid` combines meaning and keywords; `keyword` avoids query embedding. |
| `--k` | `8` | Maximum returned hits, 1–100. |
| `--fetch` | `80` | Candidates per requested category; must be at least `k` and at most 500. |
| `--source` | Empty | Case-insensitive literal substring of the selected recorded source path. |
| `--contains` | Empty | Case-insensitive literal substring of candidate body text. |
| `--from`, `--to` | Empty | Inclusive `YYYY-MM-DD` boundaries over indexed date text. The range must not be reversed. |
| `--id` | Empty | Indexed object UUID; requires one explicit corpus. Retrieves that record without query embedding. |
| `--excerpt-chars` | `600` | Displayed source-text length, 100–4000. |
| `--presentation` | `human` | `human`, `compact`, `json`. |
| `-h`, `--help` | — | Print usage and exit without contacting the reader. |

Try a date range or widen the candidate window:

```text
/cb-vsearch "school" --corpus messages --from 2024-01-01 --to 2024-12-31
/cb-vsearch "school" --fetch 200 --k 12
/cb-vsearch "school" --presentation json --excerpt-chars 1200
```

To revisit a hit, substitute its UUID and category:

```text
/cb-vsearch --corpus messages --id <result-id>
```

`--id` reads the indexed record; it does not open or download the source file. The excerpt limit still applies. [C1; C3: `search`, `validate`]

## Read and cite results

Choose the presentation for the reader. [C3: `render`]

| Format | What it returns |
|---|---|
| `human` | Numbered excerpts with category, source, available date, ID, available source hash and locators. Shortened text ends with `[excerpt]`. |
| `compact` | JSON marked `compact-columns-v1`. `hits.columns` names fields once; each array in `hits.rows` supplies values in that order. |
| `json` | Expanded JSON with named fields in every hit. Useful when inspecting citation metadata. |

Machine output retains citation objects and `excerpt_truncated`. Duplicate identities preserve distinct citations. Citation metadata can include collection/object ID, source path or vault key, source version, hashes, document/chunk/conversation IDs, archive member, parser and indexing metadata when those fields exist. Missing source hashes are marked `not_indexed`; the reader does not manufacture them. [C3: `SOURCE_FIELDS`, `search`, `shape`, `render`]

When supporting a claim, carry the excerpt's citation object with the claim. Retain source identity, version/hash, and the relevant record or chunk locator. Verify the substantive statement against the original record; AI-chat passages supply context and need that same verification. A recorded path alone does not prove that the current physical object is still there. [C1: “Read a result”; repository source-traceability rule]

## Troubleshooting

| Symptom | What to check |
|---|---|
| No filtered hits | Filters apply only after retrieval. Increase `--fetch`, adjust the question, or inspect the file inventory. |
| Date filter hides a hit | An unindexed date does not satisfy a date boundary. |
| One category unavailable | Inspect `partial`, `failures`, and `coverage` in machine output. Partial results do not establish success for every scope. |
| All categories fail | The server reports a failure rather than an empty successful result. Check the existing SSH/index route. |
| Hybrid query fails | Check server embedding configuration and named-vector/model compatibility. Keyword mode avoids the embedding request. |
| ID lookup rejected | Supply a valid UUID and one explicit corpus. |
| Hash or locator missing | Preserve the missing-field status and obtain validation from the root source before relying on the claim. |

These checks follow server `search` and `validate`, plus the guide's coverage boundaries. Candidate ranking is reciprocal rank within a category, not a confidence percentage. Counts describe indexed objects, which may be chunks or events; they do not prove unique-file counts or complete corpus coverage. [C1; C3]

## Sources and validation

Checked on **2026-10-04**. Source locators use the owning repository or external plugin root. SHA256 identifies the exact reviewed bytes.

| ID | Source and locator | SHA256 |
|---|---|---|
| C1 | `modules/Consignatio/docs/CASE-BIBLE-CATALOG-GUIDE.md:42`, “Search for content” | `e232bdda47981d9ba65e1f4aa3b2498c09a25c2655f0ba0330753ed8168dff6e` |
| C2 | `E:/AI_Workspace/plugins/plugins/case-bible/tools/cb_vsearch.py:17`, `main` and module docstring | `1f3ac7f903dbdfa8e73a898af74d8e8934a90222839b59239688f6da072b14fb` |
| C3 | `E:/AI_Workspace/plugins/plugins/case-bible/tools/cb_search_server.py:42`, `shape`; `:82` `search`; `:151` `validate`; `:174` `render` | `cae6c1387cddbcfd996f696c09b60ee6aa39e726fce925addf3e873d1d1218af` |
| C4 | [search-reader-20261004.md](assets/search-reader-20261004.md), “Verified live” and implementation hash | `d1869d683784c8f50223488b1bbb8de5c5923d0f86a7afe681827e21b0168f69` |

**Validation boundary:** this documentation pass read the sources and executed `cb_vsearch.py --help`. C4 records live hybrid/keyword retrieval, all categories, filters, ID lookup, presentation checks and five synthetic tests. Its server hash matches C3. Those are attributed receipt results, not new live tests by this documentation agent. Query examples were checked against parser definitions, not executed against the corpus.

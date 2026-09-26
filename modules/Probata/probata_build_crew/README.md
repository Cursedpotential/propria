# probata_build_crew

> _Byline: Claude Code · Fable 5.1 · 2026-09-06_

Read-only build-helper crew for the probata repository (option A in
`probata/docs/pending-review/2026-09-06-crewai-builder-crew-proposal.md`). v2 pipeline (2026-09-06 22:05):
`canon_reader` (constraints brief) -> `code_reviewer` (code review, then gap analysis) -> `build_planner` (work package)
-> `gatekeeper` (verifies citations, rules, traceability) -> `output/work-package.md` for the `{target}` in `crew.jsonc`.
It never writes into probata.

**LLM:** Ollama Cloud `nemotron-3-super` primary with an OpenRouter FREE-model fallback chain (`llm_chain/chains.py`,
`llm_chain/fallback.py`; cross-provider fallback is not native to CrewAI 1.15). Sequential process = one request in flight.

**Tools (all read-only, `tools/`):** FileReadTool · `rg_search` (ripgrep) · `list_dir` (one level) · `ccc_search` (CocoIndex
semantic) · `ccc_grep` (structural) · `smart_explore` (tree-sitter search/outline/unfold/refs/imports) · `duckdb_query`
(read-only SQL, read_text sweeps, read-only attach) · `duckdb_read_file` (profile CSV/JSON/Parquet) · `duckdb_docs` (BM25 over
the cached DuckDB docs index).

## Running

```bash
CREWAI_DMN=true crewai run
```

`CREWAI_DMN=true` = plain output (no Textual TUI). `.env` sets `CREWAI_TOOLS_ALLOW_UNSAFE_PATHS=true` so the read-only file
tools may read the probata tree outside this directory.

## Project Structure

- `agents/` - Agent definitions (JSONC)
- `crew.jsonc` - Crew definition with tasks and configuration
- `tools/` - Custom tools (Python)
- `knowledge/` - Knowledge files for agents

> **Note:** `custom:<name>` tool references execute `tools/<name>.py` as local
> Python code when the crew loads. Only run crew projects from sources you
> trust.

## Scripts

- `scripts/run_gate.py <brief> <review> <gaps> <draft>` - re-run only the gatekeeper against saved stage outputs (`crewai replay`
  does not support JSON crews); exports live in `%LOCALAPPDATA%/CrewAI/probata_build_crew/latest_kickoff_task_outputs.db`.
- `scripts/assemble_package.py` - deterministic stitching of draft + gap table + findings table + gatekeeper section into `output/work-package.md`.
- `scripts/check_citations.py <doc> --append` - mechanical file/line/keyword check of every citation; the LLM gatekeeper is NOT the
  verifier of record (it fabricated line numbers on 2026-09-06).

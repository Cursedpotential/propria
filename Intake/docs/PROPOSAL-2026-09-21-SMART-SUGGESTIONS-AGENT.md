<!-- tags: proposal, intake, xplorer, smart-suggestions, agent, owner-directive, ui-components -->
# Proposal — Smart Suggestions driven by the in-app agent

> _Byline: Claude Code · Fable 5.1 · 2026-09-21. STATUS: PROPOSAL — nothing built. Owner approved the approach 2026-09-21 01:50 EDT ("ok": build on the in-app agent, keep the model swappable)._

## Why

- Owner 2026-09-20 23:33 EDT, on the Intake panel text "Smart Suggestions (0) — Organization suggestions disabled for project directories": "but why? why not use claude or codex sdk and our rules and prompts and such to use this feature??"
- Today the feature is the donor's `apps/src-tauri/src/file_organizer.rs`:
  - `analyze_directory` returns no suggestions whenever `detect_project_directory`, `parent_is_project` or `is_source_code_subdirectory` finds a marker (`package.json`, `.git`, `Cargo.toml`, `src/`, `lib/` …). The vault is full of those, so it is off almost everywhere it is wanted.
  - When it does run it is three fixed rules (≥3 files of one type, ≥5 files of one month, ≥3 files sharing a name prefix). No AI, no catalog, no owner rules.

## What changes

1. **Same panel, same buttons.** `analyze_directory` → `preview_organization` → `execute_organization` keep their shapes, so the UI does not change.
2. **Suggestions come from the agent already in the engine** (`src/agent/`, `ai_portkey.rs`; Gemini 3.8 Flash free, Kimi K3 fallback). The model stays a setting, so Claude or Codex can be selected per run when better judgment is worth the usage.
3. **What the agent is given for a folder:**
   - the listing (names, sizes, dates, types), not file contents by default;
   - what the catalog knows for those files (copies, hashes, original locations) from `raw_duck`;
   - the owner's rules as its instructions, loaded from one tracked rules file, e.g.: Takeouts are atomic; folders are units of organization (never rename one folder into another's name); chats vs message transcripts; dev-junk policy; never suggest deleting, only moving.
4. **The project-directory check becomes an input** ("this looks like a code project: suggest nothing inside it unless asked"), not a hard switch-off. An explicit "suggest anyway" control overrides it.
5. **Output is the existing `FolderSuggestion` list**, each with a plain-language reason and the rule it relied on. The owner clicks to accept; the move runs through the existing direct file-operation path (no approval gate beyond that click, per CBX-DONE-009). The three fixed rules stay as the instant, no-model fallback when the model is unavailable.
6. **Where it runs:** inside the existing `intake-engine` Coolify app. No new containers, services or images.

## Build steps (each verified live in the hosted page before the next)

1. Rules file + prompt; `analyze_directory` calls the agent with listing + catalog facts; fixed rules as fallback.
2. Project-directory check turned into an input + "suggest anyway".
3. Model selector wired to the existing provider setting.
4. Live check on three real folders (a Takeout, a loose-files folder, a code folder); record results in `docs/URGENT-TODO.md`.

## Open points for the owner

- Which folders it may read file *contents* for (default: none; listing + catalog only).
- Whether accepted suggestions should be remembered as new rules (default: no, not silently).

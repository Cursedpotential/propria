# Harness load audit: duplicates and stale entries (2026-09-28)

> _Byline: Claude Code · Opus 5.5 · 2026-09-28 (agent `harness-dedupe`, for the owner's 05:00 EDT
> order "comprehensively scan for duplicated skills/commands/plugins each harness loads")._
> Load rules and their citations: [2026-09-28-harness-load-rules-research.md](2026-09-28-harness-load-rules-research.md).
> Scripts are tracked in [2026-09-28-harness-load-audit/](2026-09-28-harness-load-audit/) (`inventory.py`,
> `analyze.py`, `crossharness.py`, `harness_dedupe_settings.py`). CLI dumps (`plugin_list.json`,
> `oc_skills*.err`, `codex_prompt*.json`) stay in the job folder `C:/Users/matts/.claude/jobs/b6731d54/tmp/`.

## Answer first

- **Fixed now (quarantined, nothing deleted):** 4 Codex skill copies, 5 OpenCode skill copies, the
  `~/.opencode/skills` symlink, 10 dead Probata skill junctions, 15 orphan marketplace clones.
  OpenCode duplicate warnings fell from 80 to 44; Codex duplicate skill names fell from 1 to 0.
- **One owner command** applies the settings fixes (backup first, idempotent):

  ```
  ! python3 C:/Users/matts/.claude/jobs/b6731d54/tmp/harness_dedupe_settings.py
  ```

  The tracked copy is `docs/receipts/2026-09-28-harness-load-audit/harness_dedupe_settings.py`.
  Add `--with-defaults` to also take the three owner-decision defaults D1-D3 below. Run it after
  `marketplace-move` finishes, then restart Claude Code and Codex.
- **The `smart-explore@synced: false` flag works.** `claude plugin list --json` (CLI 2.1.283) reports
  `smart-explore@synced` as `enabled: false`, and this session lists only `search:smart-explore`.
  The triple listing the owner saw came from a session that started before the flag was added
  (between 2026-09-26 19:00 and 2026-09-28 04:43) or from the desktop app, which this audit could
  not observe.
- **Biggest remaining duplication is cross-harness:** 26 skills live as identical copies in both
  `~/.claude/skills` (Claude Code) and `~/.agents/skills` (Codex, OpenCode, Gemini). Claude Code
  does not read `~/.agents/skills`, so neither copy can simply go. That needs the owner's call (O1).

## How each harness loads (verified)

| Harness | Version | Verified by | What it loads |
|---|---|---|---|
| Claude Code CLI + desktop app local sessions | 2.1.283 | docs + `claude plugin list --json` + this session's skill list | `~/.claude/skills/*/SKILL.md` (by folder name), `~/.claude/commands`, `~/.claude/agents`, project `.claude/skills` from cwd up, enabled plugins (`name@marketplace`, `name@skills-dir` for a plugin folder inside `~/.claude/skills`, `name@synced` from claude.ai), synced claude.ai skills as `anthropic-skills:<name>`. **Not** `~/.agents/skills`: its unique skills (`weaviate`, `tailscale`, `behavioral-pattern-analyzer`) never appear in the session list. |
| Codex CLI | 0.149.1 | docs + `codex debug prompt-input` (the model-visible skill table) | `~/.codex/skills` (documented as deprecated, still scanned as root `r0`), `~/.agents/skills` (`r1`), `~/.codex/skills/.system`, repo `.agents/skills` from the git root to cwd, enabled plugins from `~/.codex/plugins/cache`. Duplicate names are not merged; both appear. |
| OpenCode | 1.18.5 | docs + `opencode debug skill --print-logs` | `~/.config/opencode/skills`, **`~/.claude/skills` recursively** (including `synced/`, `.system/` and nested plugin folders), `~/.agents/skills`, and `~/.opencode/skills`. It logs every duplicate name; per its source (`packages/opencode/src/skill/index.ts`) the later copy overwrites the earlier one. |
| Gemini CLI | not installed | `which gemini` | `~/.gemini/skills` and `~/.gemini/extensions` exist but nothing loads them. Left untouched. |

Two findings contradict the docs and are recorded as observed behaviour:

- The Claude Code skills page says a skill's frontmatter `name` replaces its folder name. CLI 2.1.283
  lists folder names instead: `ops:tailscale` (frontmatter `ops-tailscale`), `tanstack:tanstack-ai`
  (`ts-tanstack-ai`), `confidence-check` (`Confidence Check`).
- The Codex docs no longer list `$CODEX_HOME/skills`, but Codex 0.149.1 still scans it.

## Claude Code

| Item | Origins | Duplicate | Rule / source | Action |
|---|---|---|---|---|
| `smart-explore` | synced plugin, `search@casebible-local` | was | synced plugins are disabled with `"<name>@synced": false` (plugins/install.md) | Already disabled; CLI confirms `enabled: false`. No action. |
| `ffmpeg-master@claude-plugin-marketplace` 3.6.0 | plugin, same 60 skills/commands as `ffmpeg-core`, `-effects`, `-platforms`, `-python`, `-social-video` (all enabled) | yes, 60 items | upstream 4.0.0 turned ffmpeg-master into an empty meta-bundle | **M2** in the script: set it to `false`. |
| `context-mode@context-mode` | `enabledPlugins: true` only | stale | enabled but not installed, marketplace not registered | **M1** in the script: remove the entry. |
| 12 records in `installed_plugins.json` (osgrep, sentiment-analysis-tool, markdown-tools, computer-vision-processor, deep-research, skills-search, deep-learning-optimizer, pi-pathfinder, windags-skills, cc10x, 2x exa local) | install folder or project folder gone, all disabled | stale | registry entries with no files | **M3** in the script: drop them. |
| 15 marketplace clones in `~/.claude/plugins/marketplaces` (ai-summary-request, api-fuzzer, pi-pathfinder, presumption-guard, tasict-opencode-plugin-cc, …) | not in `known_marketplaces.json` | stale | unreferenced | **Quarantined** to `~/.claude/plugins/to_be_deleted/2026-09-28-harness-dedupe/marketplaces/`. |
| 10 skill junctions in `Propria/modules/Probata/probata/.claude/skills` (surrealdb-*, weaviate*) | project scope | stale | every target is under the retired `the-platform-workspace` path | **Quarantined** to `probata/.claude/to_be_deleted/2026-09-28-harness-dedupe/skills/` (gitignored). |
| `code-review` command | `code-review@claude-code-plugins` and `code-review@claude-plugins-official`, plus the built-in `/code-review` | yes, same plugin name so the namespaces collide | two copies of one Anthropic plugin | Owner call **D1** (default: disable the `claude-code-plugins` copy). |
| `mineru` | user skill `mineru` (root `SKILL.md`) and plugin `mineru@skills-dir` (`skills/mineru/SKILL.md`) from one folder | yes, the two SKILL.md files differ | both come from one upstream repo | Owner call **D2** (default: disable `mineru@skills-dir`; the root skill is the fuller one). |
| `adr-code-traceability`, `developing-llamaindex-systems`, `nemo-retriever` | user skill and `anthropic-skills:` (claude.ai upload) | yes, identical | docs: the synced copy then runs only under `/anthropic-skills:<name>` | Owner call **O2**. |
| `context-engineering`, `duck-time-travel`, `transcript-digest` | user skill and `anthropic-skills:` | yes, the copies differ | same rule | Owner call **O2** (pick which copy wins before removing the other). |
| `cocoindex` | user skill and synced plugin `cocoindex:cocoindex` | yes, the copies differ | synced | Owner call **O3**. |
| `app-planning-handoff` | synced plugin (enabled) and `app-planning-handoff@skills-dir` (disabled) | no longer, the skills-dir copy is off | | No action; O3 covers the synced one. |
| `read-memories` | user skill and `duckdb-skills:read-memories` | yes, differ (the user copy adds OpenCode) | | Owner call **O4** (default: keep both; the user copy is the owner's fork). |
| `recall` | user skill, `recall-skill:recall`, `propria-docstore:recall`, plus `recall@FlineDev` | 4 recall surfaces | | Owner call **O4**. |
| `handoff` | user skill and `propria-docstore:handoff` | differ | | Owner call **O4**. |
| `env-inventory` | user command and `ops:env-inventory` | differ | skill wins a name clash only within one scope | Owner call **O4**. |
| `propria-docstore` skills and commands | the plugin ships a skill and a command with the same name for diagnostics, docstore, get, graphs, index, memory, query, recall, search, status, upgrade, handoff | yes, inside one plugin | docs: a skill beats a command of the same name | Sent to `marketplace-move` (plugin source is in the marketplace). |
| `propria-docstore` versions | user 0.8.4, project installs 0.8.2 (`modules/Probata/probata`) and 0.8.1 (`modules/Probata`) | stale versions | local > project > user | Sent to `marketplace-move`: reinstall after the move. |
| Desktop Commander | synced `desktop-commander` MCP and the claude.ai "Remote Desktop Commander" connector | yes, two tool sets | MCP precedence: plugin before claude.ai connectors | Owner call **O3**. |
| Figma, Google Drive | claude.ai connectors and synced `design` / `google-drive` plugins | overlapping | | Owner call **O3**. |
| Memory hooks | `remember` and `memsearch` both hook SessionStart, UserPromptSubmit and SessionEnd; the Propria project runs FlineDev recall's compaction scripts beside the user-level compaction hooks | overlapping, not identical | docs: every matching hook runs | Owner call **O5**. |
| `think@casebible-local` | disabled | | `E:/AI_Workspace/AGENTS.md` tells every session to load `/think:*` skills | Owner call **O6**. |

## Codex

| Item | Origins | Duplicate | Action |
|---|---|---|---|
| `project-planning` | `~/.codex/skills` and `~/.agents/skills` | identical folders, both listed | **Quarantined** the `~/.codex` copy. |
| `duckdb-convert-file`, `duckdb-docs`, `duckdb-query` | `~/.codex/skills` and the enabled `duckdb-skills@claude-plugins-official` | identical folders | **Quarantined** the `~/.codex` copies to `~/.codex/to_be_deleted/2026-09-28-harness-dedupe/skills/`. Re-check: `codex debug prompt-input` lists no duplicate names. |
| Compaction hooks | `~/.codex/hooks/{precompact_handoff,postcompact_summary,sessionstart_compact_handoff}.py` | byte-identical copies of `~/.claude/hooks/` | Owner call **D3** (default: point Codex at the `~/.claude/hooks` copies and quarantine its own). |
| `[skills.config]` in `config.toml` | 256 entries whose `path` no longer exists (old `~/.agents/skills` legal and finance skills, `/mnt/c/...` WSL paths) | stale | Harmless; owner call **O7** whether to prune. |
| `propria-docstore-local`, `scout-local` marketplaces | copies at 0.8.2 and 2.0.0; primary is 0.8.4 and 2.0.1 | stale copies | Sent to `marketplace-move`. |
| `casebible-local` marketplace | junctions into `~/.claude/local-plugins/plugins/*`; the `capability-discovery` junction already dangles | breaks on the move | Sent to `marketplace-move`. |
| `nim-chat-probe`, `nim-embed-probe`, `get-api-docs`, `google-takeout-parser` in `~/.codex/skills` | single copy each inside Codex | no | Cross-harness copies of `llm-probes` and a user skill; covered by O1. |

## OpenCode

| Item | Origins | Duplicate | Action |
|---|---|---|---|
| `~/.opencode/skills` | symlink to `~/.agents/skills`, which OpenCode already reads | every `~/.agents` skill twice | **Quarantined** the symlink to `~/.opencode/to_be_deleted/2026-09-28-harness-dedupe/skills` (it dangles there, harmless). `~/.opencode/commands`, `agents` and `prompts` links were left alone. |
| `chatgpt-apps`, `fastapi-project-structure`, `senior-fullstack`, `uv-tdd`, `variant-analysis` | `~/.config/opencode/skills` and `~/.claude/skills` | identical folders | **Quarantined** the `~/.config/opencode` copies. OpenCode still loads them from `~/.claude/skills` (verified). |
| 26 skills in both `~/.claude/skills` and `~/.agents/skills` (ccc, clean-*, surrealdb-*, surrealql*, surrealkit, lancedb, zilliz, …) | two roots | identical | Owner call **O1**. |
| Synced claude.ai skills | OpenCode walks into `~/.claude/skills/synced/` and loads them as bare names | duplicates of the local copies | Owner call **O1** / **O2**. |
| Nested copies inside skill folders (`surrealdb/skills/surrealkit`, `surrealdb/.github/skills/surrealdb`, `reasoning-trace-optimizer/generated_skills/…`, `skill-porter/examples/…`, `mineru/.review_hold/…`, `smart-explore/smart-explore`) | recursive scan | yes | Owner call **O1** (only OpenCode sees them). |
| `duckdb` | `duckdb` npm plugin, `duckdb-*` skills and `duckdb-*` commands in `~/.config/opencode` | overlapping | Owner call **O7**. |
| `plannotator-*` commands | `~/.config/opencode/commands` and the `@plannotator/opencode` plugin | overlapping | Owner call **O7**. |
| Default model | `ollama-cloud/glm-5.2`; the provider block has no `ollama-cloud`, and `~/.claude/AGENTS.md` records nemotron-3-ultra as the OpenCode default | drift | Owner call **O8**. |

## Owner decisions (default in bold)

- **O1 One source for skills that two harnesses need.** (A) **Keep the real folder in
  `~/.agents/skills` and make each `~/.claude/skills/<name>` a junction to it.** Claude Code follows
  it, Codex and Gemini read `~/.agents`, and OpenCode still sees both paths and logs a duplicate
  warning, but both paths hold the same content. (B) Set `OPENCODE_DISABLE_CLAUDE_CODE_SKILLS=1` and keep only `~/.agents/skills`
  copies for OpenCode; Claude-only skills then stay out of OpenCode. (C) Leave the copies.
- **O2 Local skills that you also uploaded to claude.ai.** (A) **Keep the local copy and turn the
  claude.ai copy off on claude.ai** (only you can; claude.ai chat loses them, Claude Code does not).
  (B) Keep both.
- **O3 Synced claude.ai plugins that overlap local ones** (`cocoindex`, `desktop-commander`,
  `design`, `google-drive`, `exa`, `engineering`). (A) **Add `"<name>@synced": false` for
  `cocoindex` and `desktop-commander` only**, and keep the rest. (B) Turn each one off on claude.ai,
  which removes it everywhere.
- **O4 Personal overrides of plugin skills** (`read-memories`, `recall`, `handoff`,
  `env-inventory`). (A) **Keep them; they are your customised versions.** (B) Retire the user copy
  and keep the plugin one.
- **O5 Two memory hook stacks** (`remember` and `memsearch`, plus two compaction handoffs).
  (A) **Keep both; they write different stores.** (B) Disable `remember@claude-plugins-official`.
- **O6 `think` plugin.** (A) **Enable `think@casebible-local`**, because the workspace router tells
  every session to load `/think:*`. (B) Drop that directive from `E:/AI_Workspace/AGENTS.md`.
- **O7 Codex dead `[skills.config]` paths and OpenCode duckdb/plannotator overlaps.**
  (A) **Prune the 256 dead Codex entries in a follow-up script; leave OpenCode as is.** (B) Leave all.
- **O8 OpenCode default model.** (A) **Set it to `nvidia/nemotron-3-ultra-550b-a55b`** as recorded.
  (B) Keep `ollama-cloud/glm-5.2` and update the record.
- **D1-D3** are in the script behind `--with-defaults`: D1 disable `code-review@claude-code-plugins`,
  D2 disable `mineru@skills-dir`, D3 point Codex's compaction hooks at `~/.claude/hooks`.

## Not verified

- The desktop app's own plugin panel. It reads the same settings files as the CLI (plugins/install.md),
  but this audit only inspected CLI output.
- `enabledPlugins` changes and the script's effects need a restart; they were dry-run only
  (`--dry-run --with-defaults` printed 4 settings changes, 12 registry drops and the D3 hook repoint;
  nothing was written).

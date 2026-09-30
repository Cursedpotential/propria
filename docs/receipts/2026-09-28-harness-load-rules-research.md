# Harness load rules: Claude Code, Codex, OpenCode, Gemini (researched 2026-09-28)

> _Byline: Claude Code · Opus 5.5 · 2026-09-28. Condensed from two research subagents' reports, which
> used Context7 on official source (`main`) and live docs fetched 2026-09-28. Saved because the CLI
> restart stops the `harness-dedupe` audit agent that was using them._

## Claude Code (code.claude.com/docs/en/*)
- **Skills:** personal `~/.claude/skills/<name>/SKILL.md`; project `.claude/skills/` from cwd up to the repo root (nested ones load lazily); managed; plugin `skills/`; `--add-dir`. **`~/.agents/skills` and `.agents/skills` are NOT scanned** (skills.md; memory.md excludes `.agents/`). Same name: enterprise > personal > project; a plugin skill and a same-named skill elsewhere both load (the plugin one namespaced `/plugin:skill`).
- **Commands:** `.claude/commands/*.md` still works (merged into skills); on a clash, the skill wins.
- **Subagents:** managed > `--agents` > project `.claude/agents` > `~/.claude/agents` > plugin `agents/`.
- **Plugins:** id `<name>@<origin>` (a marketplace, `inline`, `skills-dir`, `synced`). The `enabledPlugins` merge is add-dir < user < project < local < `--settings` < managed; the highest source that names the id wins. Registries: `~/.claude/plugins/known_marketplaces.json`, `installed_plugins.json`, `cache/`, `synced/`. A directory-source marketplace loads plugins **in place**.
- **claude.ai-synced plugins:** `<name>@synced`; `"<name>@synced": false` in `enabledPlugins` disables one (or `claude plugin disable <name>@synced`); `syncClaudeAiPlugins: false` turns all off. Precedence: managed-pinned > `--plugin-dir` > marketplace > skills-dir > `@synced`.
- **Hooks:** every source's matching hooks run and results merge; for PreToolUse the most restrictive wins (deny > defer > ask > allow).
- **MCP:** local > project `.mcp.json` > user `~/.claude.json` > plugin > claude.ai connectors; whole entry, no field merge.
- **AGENTS.md:** read natively (v2.1.277+). Default `claude-md-or-agents-md`: when any CLAUDE.md exists up the tree, AGENTS.md is skipped; `~/.claude/CLAUDE.md` doesn't count. Never reads `AGENTS.override.md`, `AGENTS.local.md`, or `.agents/`.
- **Listing tools:** `claude plugin list [--json]`, `claude plugin marketplace list`, `/plugin list`, `/hooks`, `/memory`, `/context`, `/doctor prompt-audit`.

## Codex (openai/codex source + learn.chatgpt.com docs)
- **Home:** `CODEX_HOME` or `~/.codex`; config is a layered stack: defaults, system, MDM, user, project `.codex/`, flags.
- **Skills:** project `.codex/skills`; `$CODEX_HOME/skills` (**deprecated**); `~/.agents/skills`; `$CODEX_HOME/skills/.system`; admin; every `.agents/skills` from the repo root to cwd. Duplicate names are **not merged** (both appear).
- **Custom prompts** `~/.codex/prompts` are **deprecated**; use skills (flaky: issues #15972, #14459).
- **Marketplaces:** discovered at `.agents/plugins/marketplace.json`, `.agents/plugins/api_marketplace.json`, **`.claude-plugin/marketplace.json`** and `.cursor-plugin/marketplace.json`. Sources: local, url, git-subdir, npm. Enabled as `[plugins."<p>@<m>"] enabled = true`. Per-plugin manifest `.codex-plugin/plugin.json`.
- **AGENTS.md:** `$CODEX_HOME/AGENTS.override.md`, then `AGENTS.md`; project root to cwd, concatenated, 32 KiB cap.
- **Hooks:** `hooks.json` per layer or `[hooks]`. `[features].codex_hooks` is **deprecated**; use `[features].hooks`. Hooks need trust.
- **Deprecated keys:** `windows.sandbox_private_desktop`, `network_proxy`, `allowed_permissions`, `include_view_image_tool`. MCP has no `type` key (`command` means stdio, `url` means HTTP).

## OpenCode
- **Config:** remote, then `~/.config/opencode/opencode.json[c]`, `OPENCODE_CONFIG`, project `opencode.json`, `.opencode/`, `OPENCODE_CONFIG_DIR`, `OPENCODE_CONFIG_CONTENT`, managed. Plural directory names are canonical.
- **Skills:** `.opencode/skills`, **`.claude/skills`**, **`.agents/skills`** (project and global). Duplicate names: the last loaded wins, with a warning. `OPENCODE_DISABLE_CLAUDE_CODE_SKILLS=1` turns off `.claude/skills`.
- **AGENTS.md** first; CLAUDE.md is only a fallback.

## Gemini CLI
- **Settings:** system defaults < `~/.gemini/settings.json` < `.gemini/settings.json` < system overrides.
- **Skills:** built-in < extension < user `~/.gemini/skills` (alias `~/.agents/skills`) < workspace `.gemini/skills` (alias `.agents/skills`). `context.fileName` can include `AGENTS.md`.

## Implications for the owner's setup
- `~/.claude/AGENTS.md` says Claude Code scans `~/.agents/skills`. **Per the docs it does not.** Skills there reach Codex, OpenCode and Gemini, not Claude.
- A skill kept in both `~/.claude/skills` and `~/.agents/skills` appears twice in OpenCode.
- Codex reads `.claude-plugin/marketplace.json`, so one marketplace file serves both harnesses; each plugin keeps `.claude-plugin/` and `.codex-plugin/` manifests side by side.

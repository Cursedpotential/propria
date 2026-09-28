# Propria Docstore — Claude plugin source has moved

The Claude Code client plugin (skills, commands, agents, hooks, `client.py`, `.mcp.json`) is no
longer tracked here. Its canonical, tracked source is the private plugin marketplace:

```
E:\AI_Workspace\plugins\plugins\propria-docstore\
```

marketplace entry `propria-docstore` in `propria-plugins`
(`github.com/Cursedpotential/propria-plugins`), one folder that Claude Code and Codex both install.

Owner rule (`~/.claude/AGENTS.md`): every Claude Code plugin lives in that marketplace,
especially one that serves several projects, like Docstore does. The copy that previously lived
at this path was quarantined 2026-09-28 to
`modules/Probata/probata/to_be_deleted/2026-09-28-docstore-claude-plugin-source/claude/` — only
the owner deletes from quarantine. Confirmed before moving: nothing in `~/.codex/config.toml`
registers `modules/Probata/probata/plugins/.agents/plugins/marketplace.json` (marketplace name
`propria`), so this move does not affect Codex. Codex's own Docstore copies (the Propria-root `plugins/docstore` and
`~/.codex/local-marketplaces/propria-docstore`) were retired 2026-09-28; Codex now installs the same
marketplace folder.

This does not affect the deployed Docstore server. That stays exactly where it is:
`modules/Probata/probata/plugins/docstore/control/` (Coolify watch path
`modules/Probata/probata/plugins/docstore/control/**`, app `probata-docstore-control`).

> _Byline: Claude Sonnet 5 · 2026-09-28_

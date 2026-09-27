# Changelog

## 0.7.0 ctl hotfix — 2026-09-19

> _Byline: Claude Code · Opus 5 · 2026-09-19_

- 0.7.0 pointed `ctl` at ContextForge virtual server `be14a066…` (`propria-docs`), which serves only the raw SurrealDB tools, so none of the control tools existed. `ctl` now points at virtual server `30b97521e7f14cd7b7e2f09ece2b8b67` (`propria-docstore-control`). That server fronts the real control MCP server (`Probata/probata/plugins/docstore/control`), hosted on ovh-files as Coolify app `ywo2qvc5catoa79zgdur5o2j` (`http://100.91.190.107:8172/mcp`, branch `docstore-control-host-20260919`).
- ContextForge prefixes and hyphenates tool names: `docstore_index_full` becomes `propria-docstore-control-docstore-index-full`. Skills and agents that name tools by their bare control names need that mapping in 0.8.

## 0.7.0 — 2026-09-16

Claude-specific reliability repair for the remotely hosted deployment.

- Replaced the machine-specific local stdio control launcher with a remote HTTP MCP client.
- Renamed the plugin MCP server key from `control` to `ctl` to keep fully scoped tool names compact.
- Corrected plugin MCP tool names to use the exact `propria-docstore` manifest namespace.
- Removed fail-closed network preflight/read-gate/tool-tracker hooks and stale `PreCompact` behavior.
- Replaced shell hooks with two narrow PowerShell guidance hooks that make no network calls.
- Removed local `sq.py`, `recall.py`, `memory.py`, Python/uv, venv, and absolute-workspace dependencies from Claude workflows.
- Converted agent `skills` frontmatter to YAML lists and verified all preloaded skill names exist.
- Removed command/skill collisions (`docstore`, `memory`).
- Corrected the handoff MCP input envelope and explicit supersession guidance.
- Added a secret-free environment example and remote endpoint overrides.
- Bumped plugin version from 0.6.4 to 0.7.0 so Claude does not reuse the old cache entry.
- Added validation output, audit report, and source patches.

## 0.8.2 packaging fix (2026-09-26, Claude Code · Opus 5.5)

- Added `connection_settings.py` to the Claude bundle: `client.py` imported it, but only the Codex bundle shipped it, so the fallback client failed with ModuleNotFoundError.
- Added `bin/docstore-client.sh`, which runs `client.py` with the pinned requirements through uv (the global Python has an incompatible fastmcp/mcp pair).
- Local fallback (2026-09-26, Claude Code · Opus 5.5): `python3 client.py ...` now works under a desktop Python with an incompatible fastmcp/mcp pair. On that import failure the client re-runs itself under the pinned requirements through uv (guarded by `DOCSTORE_CLIENT_REEXEC`).

## 0.8.2 sync repair and retraction guard (2026-09-26, Claude Code · Opus 5.5)

- `sync` read the pre-2026-09-19 folders (`Probata/probata/docs` etc., now under `modules/`) and ignored the
  registry's exclusions, so every upload failed and nothing reached the source mirror after 2026-09-20. Folders
  and exclusions now come from `Propria/docs/docstore-source-registry.json` (0.8.2-sync-r2).
- Retraction guard (0.8.2-sync-r3, owner order): a sync never retracts on its own. Missing documents are restored
  from the mirror (hash-verified) by default; `--retract PROJECT/PATH` names one to retract; `--hold` / `--hold-file`
  keep the mirror's version. Docs that git does not track are flagged. Needs the server's 0.8.1-r3
  `docstore_source_read`. Record: Probata `docs/pending-review/2026-09-26-docstore-0.8.1-r2/`.

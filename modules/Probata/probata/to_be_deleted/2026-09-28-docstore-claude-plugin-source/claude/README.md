# Propria Docstore — Claude upload package 0.8.2

By Codex for Matthew Salem, 2026-09-20.

Upload `propria-docstore-0.8.2-claude-upload.zip` using Claude's custom **plugin** upload, not the individual-skill upload or MCP desktop-extension installer. This archive is the complete client plugin; its root contains `.claude-plugin/plugin.json`, `.mcp.json`, `skills/`, `commands/`, and `agents/`. The separate 0.8.1 consolidated release ZIP includes server/source material and is not the direct plugin-upload artifact.

## Start here

Use `/propria-docstore:docstore` for the general guide. Its description allows Claude to discover it from requests about Docstore, documentation search, memory, decisions, handoffs, graphs, status, or indexing. It routes to focused skills as needed; direct commands remain available.

Examples:

- `/propria-docstore:docstore how do I use this?`
- `/propria-docstore:query TEST`
- `/propria-docstore:search ADR authority`

## Remote connection

The `ctl` connection defaults to the hosted gateway at `https://mcp.mitechconsult.com/servers/aca1b85df0ef49acaf152617f043bc96/mcp`. Set `DOCSTORE_CONTROL_MCP_URL` only to override it. Authentication uses the host's `CF_MCP_CLIENT_TOKEN` environment variable. No credential is embedded in the ZIP. The portable Python client reads the same `.mcp.json` connection contract and uses the same `CF_MCP_CLIENT_TOKEN`; a separate token alias is not used.

For Claude Code, make the token available to the process that launches Claude. Other Claude upload surfaces must support and configure this authenticated remote connection; importing a ZIP does not automatically transfer Windows environment variables into a cloud or Cowork runtime. Successful package validation does not prove that authentication is attached in every host.

Five public tools provide health, capabilities, query, search, and get. Other operations are discovered with capabilities and invoked through query, preserving read/write and plan/verification controls. There is no raw-Surreal fallback and no local service launcher. Hooks are empty. CCC remains a separate local code-search tool.

## Optional helpers and version boundary

Install `requirements.txt` only to use the optional Python diagnostic/federation helpers. Normal attached-MCP operation does not require launching these helpers. Memory providers report their own availability.

Version 0.8.2 is a client packaging/documentation update for the hosted 0.8.1 service. It adds a clearer general entry skill and corrects setup instructions. It does not redeploy or modify the server. Earlier bundled audit documents describe the original implementation, not a completed upload test on every Claude surface.

At the last verified server check, API and storage were up but the latest sync was degraded with six enrichment failures. Recheck health for current runtime status.

## Configuration and local state

Native connector configuration and the portable fallback share `.mcp.json` as their connection contract. The host expands its environment references; the fallback uses `connection_settings.py` to expand that same file. Endpoint and credential values come from the dedicated host environment, not from plugin-cache ancestry. Neither read-only path requires a local state directory.

This upload excludes the legacy 0.5.4 `control/cli.py` that derived state under the plugin cache and rejected it on C:. Do not run that obsolete fallback to diagnose this hosted plugin. Its old configuration defect is not proof that the remote service is unavailable.

The client does not invent a new local database or working-state location. Source synchronization uses an explicitly supplied project root; server working state remains remote. If a future operation requires durable local state, require an explicitly configured stable E: path for this Windows deployment, validate it at that operation boundary, and never derive it from the installed plugin path.

The portable command distinguishes configuration, authentication, network, and remote HTTP failures without printing credentials. Registration remains pending until a real remote record and verified readback establish success; packaging or local receipt creation is not registration proof.

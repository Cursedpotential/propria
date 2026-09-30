# 0.8.1 deployment entry point

Byline: Codex / GPT-6 · 2026-09-20.

Use the consolidated release's [INSTALL.md](../../INSTALL.md) and UPGRADE.md for the current contract. The production entry point is `scripts/docstore/service.py`: authenticated hosted `ctl` plus the loopback worker API. The Claude plugin uses an explicit remote URL and token. Earlier operational notes below are retained as historical source material; they do not override the 0.8 five-root registry, version ledger, retention policy or remote-client configuration.

---

# Propria universal Docstore plugin

Byline: Codex / GPT-6, 2026-09-12 — canonical Codex registration repair.

This directory is the source package for Propria's universal documentation
plane. It is separate from project-local CCC code indexes and from Intake's
filesystem/evidence index.

## Canonical surfaces

- `control/`: the governed FastMCP control server and its tests. It exposes 22
  tools, seven resources, four resource templates and one prompt. API resources
  `docstore://api/openapi` and `docstore://api/surreal` retrieve live schemas.
- The client plugin (skills, commands, agents, `client.py`) is not here. Its only source is
  `E:/AI_Workspace/plugins/plugins/propria-docstore`, plugin `propria-docstore@propria-plugins`,
  one folder that Claude Code and Codex both install (owner 2026-09-28: one primary source).
  `claude/` holds only a pointer README. The former root `.claude-plugin/plugin.json` wrapper and
  the `propria` Codex marketplace in `../.agents/` were retired to
  `../../to_be_deleted/2026-09-28-plugin-wrappers-and-search-source/`; the older
  `docstore@probata`, `probata-docstore@probata` and `propria-docstore@propria` identities are
  gone from both hosts.

_Updated 2026-09-28 (Claude Code · Opus 5.5)._

## Transport and federation

Local agent hosts use stdio by default. For ContextForge, set:

```text
DOCSTORE_MCP_TRANSPORT=http
DOCSTORE_MCP_HOST=0.0.0.0
DOCSTORE_MCP_PORT=8084
```

The Streamable HTTP endpoint is `/mcp`. The server is stateless and accepts only
`127.0.0.1` or `0.0.0.0` as bind values. ContextForge must register it with
transport `STREAMABLEHTTP`; its default SSE selection is not equivalent.

Do not expose the backend port publicly. Keep it on the shared private network
behind ContextForge, and provide Docstore API/Surreal credentials only through
deployment environment variables.

## Deployment truth

The local stdio and HTTP protocols are verified. Production federation and
multi-root ingestion are not complete until the mandatory gate in the Propria
root `docs/MONOREPO-MIGRATION-PLAN-2026-09-12.md` passes. In particular, the
current Probata-only Docker build context cannot prove access to every Propria
documentation root.

Byline amendment: Codex · GPT-5 · 2026-09-12 (Codex marketplace installation repair)

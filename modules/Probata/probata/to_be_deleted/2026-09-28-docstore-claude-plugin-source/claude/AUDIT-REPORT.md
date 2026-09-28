# Propria Docstore Claude Integration Audit

**Audit date:** 2026-09-16  
**Repaired Claude client:** `propria-docstore` 0.7.0  
**Input:** uploaded package + environment export (credential values were not copied into this report or plugin)

## Executive finding

The control application itself is not the main reason Claude behaves worse than Codex. The Claude client package had accumulated several mutually incompatible assumptions:

1. the Docstore control service was documented/deployed as remotely hosted, but Claude still launched a machine-specific local stdio process from an `E:/...` checkout;
2. Claude hooks attempted to police the whole session and could tell Claude to stop ordinary filesystem/code work when a remote health check failed;
3. some MCP tool references used the wrong plugin namespace, which is especially damaging in subagent `tools:` declarations;
4. the installed 0.6.4 client had drifted from the source 0.6.4 client;
5. stale command/skill duplication and agent frontmatter made discovery and precedence less deterministic;
6. the package carried local helper-script assumptions even though the actual data/services live remotely.

The repair makes the Claude package one thing only: **a thin remote client**. It no longer starts the Docstore server, reaches into a local Propria checkout, or treats remote availability as permission to do unrelated coding work.

## What was in the uploaded package

The archive contains three related but different surfaces:

- `source/plugins/docstore/` — the broader source package whose own README says its root bundle is used for Codex;
- `source/plugins/docstore/claude/` — the intended Claude client;
- `source/plugins/docstore/control/` — the FastMCP control application and a separate compatibility plugin identity;
- `installed-plugin-0.6.4/` — an installed Claude snapshot that is not byte-for-byte the same as the Claude source tree.

These are valid concepts when kept deliberately separate. They become dangerous when treated as interchangeable plugin roots. The repaired distribution has exactly one `.claude-plugin/plugin.json` and exactly one Claude package identity.

## Root causes

### Critical — Claude was still launching a local server

The old Claude `.mcp.json` launched either `uv.exe` or a local `.venv/Scripts/python.exe` from hard-coded Windows paths under the Propria checkout. The installed 0.6.4 snapshot and source 0.6.4 even disagreed on which local launcher to use.

That directly conflicts with the supplied environment export, which describes remotely hosted docs, memory, and ContextForge MCP endpoints.

**Repair:** `ctl`, `docs`, and `memory` are all HTTP MCP definitions. No Claude MCP definition contains `command`, a venv path, `uv`, or a repository path.

### Critical — scoped MCP tool names were inconsistent with the plugin identity

Claude scopes plugin MCP tools using the plugin name and server name. The manifest name is `propria-docstore`; older instructions used variants such as `propria_docstore` or `probata-docstore` in tool references.

This is not cosmetic. A subagent with a `tools:` list whose entries do not resolve may fail to launch or lose the intended tool set.

**Repair:** explicit references use:

- `mcp__plugin_propria-docstore_ctl__...`
- `mcp__plugin_propria-docstore_docs__...`
- `mcp__plugin_propria-docstore_memory__...`

Every referenced `ctl` tool was checked against the actual decorated FastMCP functions in the bundled control source.

### Critical — hooks could disable unrelated Claude work

The previous SessionStart preflight checked remote services and could inject instructions equivalent to “do not read the filesystem” / stop when they were unavailable. A private-network blip therefore changed Claude's behavior outside Docstore operations.

The package also ran a UserPromptSubmit gate on every prompt and a PostToolUse tracker on every tool call.

**Repair:** hooks are guidance-only and narrow:

- `SessionStart` only with matcher `compact`;
- `PostToolUse` only for `Write|Edit`, and the script emits context only for Markdown under a `docs/` path.

Neither hook performs a network call or blocks the session.

### High — stale `PreCompact` design

The old package tried to use `PreCompact` output to inject a warning into the conversation. Current Claude Code hook semantics do not preserve `systemMessage`/`continue` from `PreCompact` in the way that design expected.

**Repair:** the plugin does not pretend compaction automatically persisted a handoff. A narrow post-compaction SessionStart reminder tells Claude to use the handoff skill when durable state is actually needed.

### High — packaged shell hooks lost executable state

The uploaded ZIP stored the `.sh` hook files without executable mode. Directly executing those scripts is therefore unreliable even before considering Windows shell differences.

**Repair:** the Claude client has no `.sh` hooks. On Windows it invokes `.ps1` files through `powershell.exe` explicitly.

### High — installed/source drift

The installed 0.6.4 snapshot and `source/plugins/docstore/claude` are different. Most notably, their local control launcher differs, and the installed copy contains extra `docstore` command/skill material.

**Repair:** 0.7.0 is a self-contained canonical Claude client. A source patch is included under `patches/` to bring `source/plugins/docstore/claude` to this layout.

### High — agent frontmatter and tool resolution

The original agents represented `skills` as a scalar comma-like value instead of a YAML list. Claude's current subagent format defines `skills` as a list of skill names to preload. Tool names in a restrictive `tools:` list must also resolve.

**Repair:** all three agents use YAML lists, every preloaded skill exists, and every explicit control MCP tool resolves to a function in the control source.

### Medium — command/skill collisions

Claude now treats legacy `commands/` entries and skills as the same user-invocable skill surface. A command and a skill with the same name create precedence rather than two independent workflows.

The installed package had a `docstore` command plus a `docstore` skill; the first repaired draft also still had a redundant `memory` command plus `memory` skill.

**Repair:** `docstore` and `memory` each have one canonical skill. `recall-doc`, `recall-adr`, and `update-adr` remain distinct legacy command workflows.

### Medium — handoff instructions disagreed with the actual tool schema

`docstore_handoff_write` is defined as `docstore_handoff_write(handoff: HandoffWrite)`. Claude therefore needs the `handoff: {...}` input envelope. The actual Python wrapper also defaults `supersedes` to a real empty array, so an omitted list becomes `[]` and supersedes nothing. One control-server docstring still describes the older raw-function fallback behavior.

**Repair:** both Claude skills now document the envelope and explicit record-ID supersession. `patches/REMOTE-CONTROL-TOOL-DESCRIPTION.patch` updates the server's exposed tool description so a future remote deployment stops teaching Claude the stale rule. That patch changes documentation only, not server behavior.

### Compatibility hardening — scoped tool-name length

The server key was shortened from `control` to `ctl`. In addition to reducing noise, this keeps the longest explicit scoped MCP reference in this package at **61 characters**. This avoids a class of historical tool-reference length failures seen with long plugin/server/tool combinations.

## Repaired architecture

```text
Claude Code
  |
  |-- plugin: propria-docstore 0.7.0
  |     |-- skills / agents / three legacy recall-update commands
  |     `-- two narrow non-network hooks
  |
  |-- MCP ctl ------HTTP------> ContextForge-hosted Propria control endpoint
  |-- MCP docs -----HTTP------> native docs Surreal MCP (probata/docs)
  `-- MCP memory ---HTTP------> native memory Surreal MCP (probata_memory/memory)
```

There is no local Docstore runtime in this Claude package.

## Remote endpoint assumption

The supplied environment export contains one ContextForge endpoint identified for `propria-docs`, and that is the endpoint used as the default `ctl` URL in this repaired client.

I could not live-query private/Tailscale endpoints from the audit sandbox. Therefore one runtime fact still has to be verified on your machine: **the `ctl` endpoint must expose the control tools such as `docstore_health`, `coco_docstore_search`, and `docstore_get`.**

If `/mcp` shows that `ctl` is connected but those tools are absent, do not rewrite the plugin. Set `DOCSTORE_CONTROL_MCP_URL` to the actual remotely hosted control-server `/mcp` URL and reload the plugin.

## Installation / smoke test on Windows

The three secret variables must be present in the environment inherited by the process that launches Claude Code:

- `CF_MCP_CLIENT_TOKEN`
- `DOCSTORE_BASIC_AUTH`
- `MEMORY_BASIC_AUTH`

If those User environment variables were created or changed after your terminal/IDE was opened, restart that host process first.

From the directory containing the ZIP:

```powershell
claude --plugin-dir .\propria-docstore-claude-0.7.0.zip --debug
```

Direct ZIP loading requires Claude Code v2.1.128 or newer. An extracted directory works as well:

```powershell
claude --plugin-dir .\propria-docstore-claude-0.7.0 --debug
```

Inside Claude Code:

1. Run `/plugin` and inspect the Errors view.
2. Run `/mcp`; `ctl`, `docs`, and `memory` should be present. Remote servers may connect lazily.
3. Invoke `/propria-docstore:docstore` and `/propria-docstore:memory` to confirm skill discovery.
4. Ask Claude to call `docstore_health`.
5. Run a small `coco_docstore_search` and then `docstore_get` on one returned record.
6. Check `/agents` for the three plugin agents.
7. After replacing an already loaded version, run `/reload-plugins` or start a fresh session.

For an explicit schema/frontmatter check from the plugin/marketplace directory, current Claude documentation also provides:

```powershell
claude plugin validate .
```

## What *not* to do

- Do not add back a local `uv`, Python, or venv launcher to Claude just because the control source exists in the repo.
- Do not load the Codex root package, the Claude subpackage, and the control compatibility package simultaneously as if they were one Claude plugin.
- Do not make remote service health a global filesystem permission gate.
- Do not copy these plugin skills into a bare `.claude/skills` tree as a second installation.
- Do not put actual credentials into `.mcp.json`, `.env.example`, skills, hooks, or the ZIP.
- Do not “fix” missing `ctl` tools by renaming skill tool references until `/mcp` proves which remote endpoint is actually connected.

## Validation performed here

The included `VALIDATION.txt` records **67 passing static checks and 0 failures**. The validator checked, among other things:

- JSON syntax for manifest, MCP config, and hooks;
- YAML frontmatter for every skill, command, and agent;
- one plugin manifest only;
- no command/skill name collision;
- remote HTTP-only MCP configuration;
- exact plugin MCP namespace;
- every referenced `ctl` tool against actual control-source registration;
- agent preloaded skill existence;
- hook target existence;
- no `.sh` hook remnants;
- no machine-specific Windows workspace paths or local launchers;
- no old MCP namespace variants;
- no supplied high-entropy credential value copied into the package;
- longest explicit MCP tool reference length.

The control Python source also compiled successfully during the audit. The audit environment does not contain the Claude CLI or PowerShell, so I could not run Claude's own `plugin validate` command or execute the Windows hook scripts here. The sandbox also cannot reach your private remote endpoints, so the final live MCP handshake must be performed on your host.

## Included patches

- `patches/CLAUDE-CLIENT-SOURCE.patch` — source-level diff from the uploaded `source/plugins/docstore/claude` tree to the repaired 0.7.0 client.
- `patches/REMOTE-CONTROL-TOOL-DESCRIPTION.patch` — optional but recommended remote control-source docstring correction for `docstore_handoff_write`; no runtime logic change.

## Authoritative Claude references used

- Plugin creation and local ZIP testing: https://code.claude.com/docs/en/plugins
- Plugin technical reference/layout/MCP namespacing/cache/debugging: https://code.claude.com/docs/en/plugins-reference
- Custom subagents and `skills`/`tools` frontmatter: https://code.claude.com/docs/en/sub-agents
- MCP configuration: https://code.claude.com/docs/en/mcp
- Hooks reference: https://code.claude.com/docs/en/hooks
- Skills: https://code.claude.com/docs/en/skills

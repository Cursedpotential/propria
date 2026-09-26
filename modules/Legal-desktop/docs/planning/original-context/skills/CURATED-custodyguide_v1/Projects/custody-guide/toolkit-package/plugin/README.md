# Genesee Family Court Toolkit — Claude Code Plugin

A self-contained, informational-only Claude Code plugin for Genesee County, Michigan family-court workflow, evidence organization, source verification, and drafting structure. It is not legal advice and has no external services, connectors, credentials, autonomous filing, device access, MCP server, or hooks.

## Local install/test

```bash
claude --plugin-dir .family-court-claude-plugin
claude plugin validate .family-court-claude-plugin --strict
pytest -q .family-court-claude-plugin/tests .family-court-claude-plugin/skills/toolkit/content/toolkit/tests
```

## Usage

- `/toolkit:toolkit custody hearing exhibits`
- `motion motion to compel outline`
- `discovery RFA authenticity and RFP wage statements`
- `evidence shared-device screenshot`
- `hearing parenting-time hearing`
- `verify-sources MCR 3.215 and FOC 68`
- `@toolkit:michigan-source-verifier`

Plugin skills and agents use the `toolkit:` namespace. Commands are plugin components and should be invoked by their displayed plugin command names.

## Content and migration audit

The exact pre-consolidation baseline is copied under `skills/toolkit/content/toolkit/`. See `MIGRATION-AUDIT.md` and `migration-audit.json` for verified source fingerprint, all old-path-to-new-path mappings, and exact-hash results.

## Compatibility
For Claude Code 2.1.140 compatibility, the manifest intentionally omits newer optional `displayName` and `defaultEnabled` fields. Claude displays the plugin `name` as the label and applies its default enablement behavior. The plugin remains valid under the current schema.

## CourtListener legal research MCP

The plugin includes the official Free Law Project CourtListener remote HTTP MCP server. No token or OAuth credential is embedded. To add it separately in Claude Code:

```bash
claude mcp add --transport http courtlistener https://mcp.courtlistener.com/
```

First CourtListener tool use opens interactive browser OAuth; never rely on an automatic or silent authorization. Use `/mcp` to inspect connection state. A CourtListener account is required; standard API access is automatic, while elevated access may require membership/donation. See `LEGAL-RESEARCH-INTEGRATIONS.md` for privacy, limits, Michigan cross-check, and fallback guidance. Official CourtListener MCP guidance: <https://wiki.free.law/c/courtlistener/help/api/mcp/model-context-protocol-mcp-server-for-agentic-access>.

## MiCOURT Case Search — documentation only

MiCOURT Case Search is **not connected**: no MCP endpoint, API client, credentials, or API call is included. It is documented only as a guarded case/docket lookup assessment. CourtListener remains the active opinion/case-law MCP. Read `MICOURT-CASE-SEARCH-INTEGRATION.md`; use `case-lookup` only for manual/public lookup guidance or separately approved external MiCOURT access. Do not provide credentials in chat.

## Validator compatibility and discovery

Use the broadly compatible validator command first:

```bash
claude plugin validate .family-court-claude-plugin
```

On Claude Code versions that support it, also run:

```bash
claude plugin validate .family-court-claude-plugin --strict
```

Claude Code **2.1.140** does not support `--strict`; use the first command plus the bundled local schema/migration tests on that version. The plugin intentionally omits newer optional `displayName` and `defaultEnabled` fields so 2.1.140 accepts the manifest.

Discovery test recorded on 2026-08-09:

```bash
claude --plugin-dir /home/user/workspacefamily-court-claude-plugin plugin list
```

reported `toolkit@inline`, version `1.0.0`, status `loaded`.

Claude Code auto-discovers the standard root `hooks/hooks.json`; it is intentionally not duplicated in `plugin.json`, because Claude Code 2.1.140 treats that as a duplicate hook file and loads the plugin with an error.

# Security

This plugin has no external connectors, credentials, MCP server, autonomous court filing, device access, account access, or legal-service integration. Hooks are intentionally absent: safety gating is implemented in the skill, commands, agents, tests, and bundled read-only scripts, avoiding autonomous side effects. Do not include protected case records in a security report.

## CourtListener MCP

The root `.mcp.json` contains only the official remote endpoint `https://mcp.courtlistener.com/`; it embeds no secrets, headers, API keys, user IDs, or OAuth material. Authentication is interactive OAuth managed by Claude Code and CourtListener. Use `/mcp` to inspect/disconnect connections. The deterministic hooks do not make network calls, read transcripts, write files, or log data.

## MiCOURT assessment

MiCOURT is not configured in `.mcp.json`, has no connector, and accepts no credentials through this plugin. Do not paste a `Jis-Api-Subscription-Key`, bearer token, client ID, or client secret into a chat, command, source file, or plugin configuration. Access to restricted court data must remain within separately approved JIS/MiCOURT authorization.

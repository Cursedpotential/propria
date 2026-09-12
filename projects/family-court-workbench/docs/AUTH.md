# Auth — Claude Code OAuth token

> _Byline: Claude Code · Sonnet 5 · 2026-09-07_

## How to get a token

```
claude setup-token
```

This prints a long-term OAuth token tied to your Claude subscription/account, meant for
non-interactive use of the official Claude Code CLI / Agent SDK (the docs describe it as intended
for exactly this: driving Claude Code programmatically without an interactive login each time).

## How this app resolves it

`sidecar/lib/auth.mjs`, in order:

1. `process.env.CLAUDE_CODE_OAUTH_TOKEN`, if the sidecar process was started with it set.
2. The first `CLAUDE_CODE_OAUTH_TOKEN=...` line found in any `~/.secrets/*.env` file, parsed with
   a tolerant regex (`^\s*([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.+?)\s*$`) — **never `source`d**, per the
   owner's standing hard rule that `KEY = value` spacing in some `.secrets/*.env` files makes a
   naive `source` execute the value as a shell command.
3. Neither present: `sidecar/lib/chat.mjs`'s `checkAuthAvailable()` does a best-effort local check
   for `~/.claude/.credentials.json` or `~/.claude.json` and, if found, optimistically assumes the
   CLI's own interactive login (`claude login`) will work when the Agent SDK spawns its subprocess.
   **This heuristic is unverified** — no doc consulted while building this app named the exact
   file/format Claude Code CLI uses to persist an interactive login on this platform. If neither
   an explicit token nor that heuristic file is found, `/api/chat` returns a clear `401` with a
   fix message instead of spawning the CLI subprocess and waiting for it to fail.

The token itself is never logged, never returned over HTTP (`/api/auth/status` returns
`tokenLength` only — see `sidecar/tests/server.test.mjs`), and never reaches the webview.

## ⚠️ TODO for the root agent — terms-of-use question, not resolved here

This app uses `@anthropic-ai/claude-agent-sdk`'s official `query()` function exclusively to drive
Claude — it never makes a raw HTTP call to `api.anthropic.com` with the token, and never bypasses
the CLI harness the SDK wraps. That is the intended use pattern for a `claude setup-token` token
as far as the SDK's own design suggests.

However: **whether using a Claude subscription's long-term OAuth token to power a custom desktop
application's chat pane (as opposed to interactive `claude` CLI sessions or CI automation) is
within Anthropic's current consumer terms of service is NOT verified by anything read or checked
while building this scaffold.** This is flagged deliberately rather than asserted either way. The
root agent (or the owner) should check current Anthropic ToS/Usage Policy before this ships beyond
personal local/desktop use on the owner's own machine, with the owner's own subscription, for the
owner's own case data.

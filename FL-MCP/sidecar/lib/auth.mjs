// Byline: Claude Code · Sonnet 5 · 2026-09-07
//
// Resolves the Claude Code long-term OAuth token (`claude setup-token`) for
// the Agent SDK's spawned CLI subprocess. Never prints the value; never
// `source`s a secrets file (an owner hard rule — `KEY = value` spacing in
// some `.secrets/*.env` files makes a shell execute the value as a command
// and echo token fragments into a transcript). Parses with a tolerant
// regex instead.
//
// Resolution order:
//   1. `CLAUDE_CODE_OAUTH_TOKEN` already in the sidecar's own environment.
//   2. The first `CLAUDE_CODE_OAUTH_TOKEN=...` line found in any
//      `~/.secrets/*.env` file.
//   3. Neither present: the Agent SDK's spawned `claude` CLI subprocess
//      falls back to its OWN login state (e.g. `claude login` run
//      interactively elsewhere on this machine) — we do not fail here,
//      we just report `source: "cli-login"` as an unverified guess and let
//      the first real /api/chat call surface the actual auth error if the
//      CLI also has no session.

import { existsSync, readFileSync, readdirSync } from "node:fs";
import { homedir } from "node:os";
import { join } from "node:path";

const TOKEN_VAR = "CLAUDE_CODE_OAUTH_TOKEN";
const SECRETS_DIR = join(homedir(), ".secrets");

/** Tolerant `KEY = value` / `KEY=value` line parser. Never executed as shell. */
function parseEnvLine(line) {
  const match = /^\s*([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.+?)\s*$/.exec(line);
  if (!match) return null;
  const [, key, rawValue] = match;
  const value = rawValue.replace(/^["']|["']$/g, "");
  return { key, value };
}

function findTokenInSecretsDir() {
  if (!existsSync(SECRETS_DIR)) return null;
  let files;
  try {
    files = readdirSync(SECRETS_DIR).filter((f) => f.endsWith(".env"));
  } catch {
    return null;
  }
  for (const file of files) {
    let contents;
    try {
      contents = readFileSync(join(SECRETS_DIR, file), "utf8");
    } catch {
      continue;
    }
    for (const line of contents.split(/\r?\n/)) {
      if (line.trim().startsWith("#") || !line.includes("=")) continue;
      const parsed = parseEnvLine(line);
      if (parsed?.key === TOKEN_VAR && parsed.value) {
        return { token: parsed.value, source: `secrets-file:${file}` };
      }
    }
  }
  return null;
}

/**
 * Returns `{ token: string | null, source: "env" | "secrets-file:<name>" | "cli-login" }`.
 * NEVER logs or returns the token in a way that would be printed by a naive
 * caller — callers must reference it by presence/length only in logs.
 */
export function resolveAuthToken() {
  const envToken = process.env[TOKEN_VAR];
  if (envToken) return { token: envToken, source: "env" };

  const fromSecrets = findTokenInSecretsDir();
  if (fromSecrets) return { token: fromSecrets.token, source: fromSecrets.source };

  return { token: null, source: "cli-login" };
}

/** Safe-to-log summary — length only, never the value. */
export function describeAuth() {
  const { token, source } = resolveAuthToken();
  return {
    configured: Boolean(token) || source === "cli-login",
    source,
    tokenLength: token ? token.length : null,
  };
}

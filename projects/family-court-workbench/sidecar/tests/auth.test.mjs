// Byline: Claude Code · Sonnet 5 · 2026-09-07
import assert from "node:assert/strict";
import { test } from "node:test";
import { resolveAuthToken } from "../lib/auth.mjs";

test("resolveAuthToken prefers CLAUDE_CODE_OAUTH_TOKEN from the environment", () => {
  const prior = process.env.CLAUDE_CODE_OAUTH_TOKEN;
  process.env.CLAUDE_CODE_OAUTH_TOKEN = "test-token-value";
  try {
    const { token, source } = resolveAuthToken();
    assert.equal(token, "test-token-value");
    assert.equal(source, "env");
  } finally {
    if (prior === undefined) delete process.env.CLAUDE_CODE_OAUTH_TOKEN;
    else process.env.CLAUDE_CODE_OAUTH_TOKEN = prior;
  }
});

test("resolveAuthToken never throws when no token is configured anywhere reachable", () => {
  const prior = process.env.CLAUDE_CODE_OAUTH_TOKEN;
  delete process.env.CLAUDE_CODE_OAUTH_TOKEN;
  try {
    const result = resolveAuthToken();
    assert.ok("token" in result && "source" in result);
  } finally {
    if (prior !== undefined) process.env.CLAUDE_CODE_OAUTH_TOKEN = prior;
  }
});

// Byline: OpenAI Codex / GPT-5.6, 2026-08-13 — rewritten for the Claude Code plugin layout by Claude Code · Fable 5.1 · 2026-09-07
import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import { resolve } from "node:path";
import test from "node:test";

const pluginRoot = resolve("..");

test("plugin uses the Claude Code manifest and a plugin-root-relative MCP launch", () => {
  const manifest = JSON.parse(readFileSync(resolve(pluginRoot, ".claude-plugin/plugin.json"), "utf8"));
  assert.equal(manifest.name, "family-court-toolkit");
  assert.match(manifest.version, /^\d+\.\d+\.\d+$/);
  assert.equal(existsSync(resolve(pluginRoot, ".codex-plugin")), false, "Codex manifest must not ship in the Claude plugin");
  const mcp = JSON.parse(readFileSync(resolve(pluginRoot, ".mcp.json"), "utf8"));
  assert.deepEqual(mcp.mcpServers["family-court-console"], {
    type: "stdio",
    command: "node",
    args: ["${CLAUDE_PLUGIN_ROOT}/mcp-app/dist/server.js"],
  });
  assert.deepEqual(mcp.mcpServers.courtlistener, { type: "http", url: "https://mcp.courtlistener.com/" });
});

test("hooks are declared, python-launched explicitly, and the MCP launch has no shell surface", () => {
  const hooks = JSON.parse(readFileSync(resolve(pluginRoot, "hooks/hooks.json"), "utf8"));
  for (const event of ["UserPromptSubmit", "PostToolUse", "PostToolUseFailure"]) {
    assert.ok(Array.isArray(hooks.hooks[event]) && hooks.hooks[event].length > 0, `hook event missing: ${event}`);
    for (const matcher of hooks.hooks[event]) {
      for (const h of matcher.hooks) {
        assert.equal(h.type, "command");
        assert.match(h.command, /python3(\.exe)? "\$\{CLAUDE_PLUGIN_ROOT\}\/hooks\/case_law_(prompt|tool)_hook\.py"/,
          `hook must launch through an explicit interpreter: ${h.command}`);
      }
    }
  }
  assert.ok(existsSync(resolve(pluginRoot, "hooks/case_law_prompt_hook.py")));
  assert.ok(existsSync(resolve(pluginRoot, "hooks/case_law_tool_hook.py")));
  const mcpText = readFileSync(resolve(pluginRoot, ".mcp.json"), "utf8").toLowerCase();
  for (const forbidden of ["cmd", "powershell", "pwsh", "bash", "wsl", "uvx", "npx"]) {
    assert.equal(mcpText.includes(forbidden), false, `unexpected launcher: ${forbidden}`);
  }
});

test("entry skill plus first-class member skills (owner ruling 2026-09-07 15:23)", () => {
  const entry = resolve(pluginRoot, "skills/family-court-toolkit/SKILL.md");
  assert.ok(existsSync(entry));
  assert.ok(existsSync(resolve(pluginRoot, "skills/verify-michigan-legal-sources/SKILL.md")));
  const skillsDir = resolve(pluginRoot, "skills");
  const { readdirSync } = require_fs();
  const advertised = readdirSync(skillsDir, { withFileTypes: true }).filter((d) => d.isDirectory());
  assert.ok(advertised.length >= 30, `expected the entry skill plus first-class members, got ${advertised.length}`);
  assert.ok(advertised.every((d) => existsSync(resolve(skillsDir, d.name, "SKILL.md"))), "every advertised skill dir carries a SKILL.md");
});

function require_fs() {
  // tiny indirection so the import list above stays minimal
  return { readdirSync: (p, o) => import_fs.readdirSync(p, o) };
}
import * as import_fs from "node:fs";

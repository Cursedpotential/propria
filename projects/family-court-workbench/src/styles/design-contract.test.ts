// Byline: OpenAI Codex · GPT-5 · 2026-09-12
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

const projectRoot = process.cwd();
const vendorRoot = resolve(projectRoot, "vendor/propria-design-contract/1.0.0");

describe("SDA-04 design contract adoption", () => {
  it("pins the accepted contract and keeps tokens.json authoritative", () => {
    const tokens = JSON.parse(readFileSync(resolve(vendorRoot, "tokens.json"), "utf8")) as {
      contract: string;
      version: string;
    };
    const pin = readFileSync(resolve(projectRoot, "vendor/propria-design-contract/PIN.md"), "utf8");

    expect(tokens.contract).toBe("propria-surface-design");
    expect(tokens.version).toBe("1.0.0");
    expect(pin).toContain("c6da141");
    expect(pin).toContain("9dcb135efedfb05b423e8e7e4b64529b09b556a9");
  });

  it("imports only the vendored runtime and maps local names to shared semantics", () => {
    const globals = readFileSync(resolve(projectRoot, "src/styles/globals.css"), "utf8");
    const adapter = readFileSync(resolve(projectRoot, "src/styles/tokens.css"), "utf8");

    expect(globals).toContain("../../vendor/propria-design-contract/1.0.0/tokens.css");
    expect(globals).toContain("../../vendor/propria-design-contract/1.0.0/theme.css");
    expect(globals).not.toContain("../../../../resources/design");
    expect(adapter).toContain("--accent: var(--pr-action)");
    expect(adapter).toContain("--good: var(--pr-positive)");
    expect(adapter).toContain("--warn: var(--pr-caution)");
    expect(adapter).toContain("--critical: var(--pr-destructive)");
    expect(adapter).toContain("--font-body: var(--pr-font-ui)");
    expect(adapter).toContain("--font-data: var(--pr-font-data)");
    expect(globals).toContain("--font-mono: var(--font-data)");
    expect(globals).not.toContain("--font-mono: var(--font-mono)");
  });

  it("boots dark-first while keeping experience tier independent", () => {
    const html = readFileSync(resolve(projectRoot, "index.html"), "utf8");
    expect(html).toContain('data-pr-theme="dark"');
    expect(html).toContain('data-pr-experience="general"');
  });
});

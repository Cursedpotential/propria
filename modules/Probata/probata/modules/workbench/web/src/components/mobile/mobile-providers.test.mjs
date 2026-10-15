// Byline: Claude Code · Opus 5.5 · 2026-10-15
// Regression (owner UX test 2026-10-15): /m/search crashed with "useFixedCase must
// be used within FixedCaseProvider". MobileShell has no FixedCaseProvider (the
// mobile view is always the live case), so no mobile screen may call useFixedCase.
import assert from "node:assert/strict";
import { readdirSync, readFileSync } from "node:fs";
import test from "node:test";

test("no mobile screen depends on the desktop FixedCaseProvider", () => {
  const dir = new URL("./", import.meta.url);
  const shell = readFileSync(new URL("mobile-shell.tsx", dir), "utf8");
  assert.ok(!shell.includes("FixedCaseProvider"), "update this test if MobileShell starts providing the fixed case");
  const offenders = readdirSync(dir)
    .filter((name) => name.endsWith(".tsx"))
    .filter((name) => /\buseFixedCase\s*\(/.test(readFileSync(new URL(name, dir), "utf8")));
  assert.deepEqual(offenders, []);
});

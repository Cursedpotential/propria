// Byline: Claude Code · Opus 5.5 · 2026-09-26
// Preflight for `npm run smoke:matter-flow`. The browser journeys skip themselves when they cannot
// run, which keeps the default `npm run smoke` gate browser-free, but an explicit browser run that
// skipped everything would read as a pass. This check fails that run instead. The journeys never
// run on Windows: the owner's desktop is the only Windows host, and no Chrome, Edge, headless
// browser or Playwright may launch there (owner rule 2026-09-24). Run them on a VPS.
import { existsSync } from "node:fs";

const browser = process.env.SMOKE_BROWSER;
const refusal =
  process.platform === "win32" ? "browser journeys never run on Windows (the owner's desktop)"
  : !browser ? "SMOKE_BROWSER is not set; point it at a Chromium-family binary"
  : !existsSync(browser) ? `SMOKE_BROWSER does not exist: ${browser}`
  : null;

if (refusal) {
  console.error(
    `smoke:matter-flow refused: ${refusal}. Run the browser journeys on a VPS; ` +
      `see README.md, "Browser journeys run on a VPS only".`,
  );
  process.exit(1);
}

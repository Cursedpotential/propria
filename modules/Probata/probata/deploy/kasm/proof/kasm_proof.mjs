// Byline: Claude Code · Opus 5.5 · 2026-10-02
// P-1 step 6: headless-Chrome proof of Kasm Workspaces, run INSIDE the Probata devbox on ovh-files (never on the
// owner's desktop). Launched by kasm_proof.sh, which installs playwright-core into ~/work/kasm-proof and drives the
// devbox's own Google Chrome (no browser download).
//
//   node kasm_proof.mjs <out-dir>
//   env: KASM_URL (default https://kasm.tilapia-skilift.ts.net), KASM_PROOF_USER, KASM_PROOF_PASSWORD (from the host's
//        kasm.env, passed by docker exec -e; never printed), KASM_PUBLIC_URL (default https://kasm.int.mitechconsult.com)
//
// Shots: 01 login page (tailnet name) · 02 signed-in workspace list · 03 a launched Devbox session · 04 a launched
// "Devbox (RDP)" Guacamole session · 05 the public route signed out (the Authentik login is the expected result).
// Each page's title, URL and console errors go to <out>/proof.json. Sessions started here are ended by the caller.
import { mkdirSync, writeFileSync } from "node:fs";
import { createRequire } from "node:module";

const require = createRequire(process.cwd() + "/");
const { chromium } = require("playwright-core");

const out = process.argv[2] || "./out";
mkdirSync(out, { recursive: true });
const BASE = process.env.KASM_URL || "https://kasm.tilapia-skilift.ts.net";
const PUBLIC = process.env.KASM_PUBLIC_URL || "https://kasm.int.mitechconsult.com";
const USER = process.env.KASM_PROOF_USER;
const PASS = process.env.KASM_PROOF_PASSWORD;
const record = { base: BASE, public: PUBLIC, user: USER, steps: [] };

const browser = await chromium.launch({
  executablePath: "/opt/google/chrome/chrome",
  args: ["--no-sandbox", "--disable-dev-shm-usage", "--ignore-certificate-errors"],
});

async function snap(page, name, extra = {}) {
  await page.screenshot({ path: `${out}/${name}.png` });
  const step = { name, url: page.url(), title: await page.title().catch(() => ""), ...extra };
  record.steps.push(step);
  console.log(JSON.stringify(step));
}

function watch(page, bucket) {
  page.on("console", (m) => { if (m.type() === "error") bucket.push(m.text().slice(0, 200)); });
  page.on("pageerror", (e) => bucket.push(String(e).slice(0, 200)));
}

async function launch(page, name, shotName) {
  await page.goto(`${BASE}/#/userdashboard`, { waitUntil: "domcontentloaded" });
  await page.waitForTimeout(6000);
  const card = page.getByText(name, { exact: true }).first();
  await card.click({ timeout: 30000 });
  await page.waitForTimeout(2000);
  const launchBtn = page.getByRole("button", { name: /launch/i }).first();
  if (await launchBtn.count()) await launchBtn.click({ timeout: 15000 });
  // the session opens in the same tab at #/kasm/<id> (or a connect page); give the desktop time to stream
  await page.waitForTimeout(45000);
  await snap(page, shotName, { workspace: name });
}

try {
  // 01 login page
  const ctx = await browser.newContext({ viewport: { width: 1600, height: 1000 }, ignoreHTTPSErrors: true });
  const page = await ctx.newPage();
  const errs = [];
  watch(page, errs);
  await page.goto(`${BASE}/#/login`, { waitUntil: "domcontentloaded" });
  await page.waitForTimeout(6000);
  await snap(page, "01-login");

  // 02 sign in, workspace list
  if (!USER || !PASS) throw new Error("KASM_PROOF_USER / KASM_PROOF_PASSWORD not set");
  await page.locator('input[name="username"], input#username, input[type="email"], input[type="text"]').first().fill(USER);
  await page.locator('input[type="password"]').first().fill(PASS);
  await page.keyboard.press("Enter");
  await page.waitForTimeout(10000);
  const bodyText = (await page.locator("body").innerText().catch(() => "")).slice(0, 4000);
  await snap(page, "02-workspaces", {
    sees: ["Devbox", "Sandbox", "Devbox (RDP)"].filter((n) => bodyText.includes(n)),
  });

  // 03 Devbox session, 04 RDP (Guacamole) session
  await launch(page, "Devbox", "03-devbox-session");
  await launch(page, "Devbox (RDP)", "04-rdp-session");
  record.console_errors_signed_in = errs.slice(0, 30);

  // 05 public route, signed out
  const pub = await browser.newContext({ viewport: { width: 1400, height: 900 } });
  const p2 = await pub.newPage();
  await p2.goto(PUBLIC, { waitUntil: "domcontentloaded" });
  await p2.waitForTimeout(8000);
  const t2 = (await p2.locator("body").innerText().catch(() => "")).slice(0, 2000);
  await snap(p2, "05-public-signed-out", { authentik: /authentik|Welcome|Username|Email/i.test(t2) || p2.url().includes("auth.int") });
} catch (e) {
  record.error = String(e).slice(0, 500);
  console.log("ERROR", record.error);
} finally {
  writeFileSync(`${out}/proof.json`, JSON.stringify(record, null, 2));
  await browser.close();
}

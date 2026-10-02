// Byline: Claude Code · Opus 5.5 · 2026-10-01
// Live proof of the Workbench Case page (case identity over registry), driven by headless Chrome
// INSIDE the Probata devbox on ovh-files (never on the owner's desktop; owner rule 2026-09-24).
// Launched by audit.sh with AUDIT_SCRIPT=case-page.mjs. The browser plumbing (Chrome over the
// DevTools protocol, the devbox's Authentik machine identity) is the same as audit.mjs.
//
//   node case-page.mjs <out-dir>
//
// Read: opens /case?mode=REAL and records the case header and every person with their identifiers
// and counts. Edit (only when CASE_EDIT_RAW is set): opens that identifier's "Edit" dialog ON THE
// PAGE, sets CASE_EDIT_KIND / CASE_EDIT_STATUS / CASE_EDIT_BASIS / CASE_EDIT_REASON, saves, and
// records the version the page shows afterwards. The new registry row is read back in PostgreSQL by
// the caller. Writes <out>/case-*.png and <out>/case-page.json.
import { spawn } from "node:child_process";
import { existsSync, mkdirSync, mkdtempSync, readFileSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { setTimeout as sleep } from "node:timers/promises";

const outDir = process.argv[2];
if (!outDir) {
  console.error("usage: node case-page.mjs <out-dir>");
  process.exit(2);
}
mkdirSync(outDir, { recursive: true });
const BASE = (process.env.WORKBENCH_URL || "https://workbench.tilapia-skilift.ts.net").replace(/\/$/, "");
const chromeBin = process.env.CHROME_BIN || "/opt/google/chrome/chrome";
const CREDENTIALS_FILE = process.env.AUTHENTIK_MACHINE_CREDENTIALS_FILE || "/run/secrets/devbox-authentik.env";

async function launchChrome() {
  const profile = mkdtempSync(join(tmpdir(), "case-page-"));
  const args = [
    "--headless=new",
    "--remote-debugging-port=0",
    `--user-data-dir=${profile}`,
    "--no-first-run",
    "--no-default-browser-check",
    "--disable-gpu",
    "--mute-audio",
  ];
  if (process.env.AUDIT_RESOLVER_RULE) args.push(`--host-resolver-rules=${process.env.AUDIT_RESOLVER_RULE}`);
  const proc = spawn(chromeBin, [...args, "about:blank"], { stdio: ["ignore", "ignore", "ignore"] });
  const portFile = join(profile, "DevToolsActivePort");
  for (let i = 0; i < 200; i += 1) {
    if (existsSync(portFile)) {
      const [port] = readFileSync(portFile, "utf8").split("\n");
      if (port) return { proc, port };
    }
    await sleep(100);
  }
  proc.kill();
  throw new Error("Chrome did not report a DevTools port");
}

function cdpSession(wsUrl) {
  const ws = new WebSocket(wsUrl);
  let nextId = 0;
  const pending = new Map();
  const listeners = new Map();
  ws.onmessage = (event) => {
    const msg = JSON.parse(event.data);
    if (msg.id && pending.has(msg.id)) {
      const { resolve, reject } = pending.get(msg.id);
      pending.delete(msg.id);
      if (msg.error) reject(new Error(`${msg.error.message} (${msg.error.code})`));
      else resolve(msg.result);
    } else if (msg.method) {
      for (const fn of listeners.get(msg.method) || []) fn(msg.params);
    }
  };
  const opened = new Promise((resolve, reject) => {
    ws.onopen = resolve;
    ws.onerror = reject;
  });
  return {
    opened,
    send(method, params = {}) {
      nextId += 1;
      const id = nextId;
      return new Promise((resolve, reject) => {
        pending.set(id, { resolve, reject });
        ws.send(JSON.stringify({ id, method, params }));
      });
    },
    on(method, fn) {
      listeners.set(method, [...(listeners.get(method) || []), fn]);
    },
    close() {
      ws.close();
    },
  };
}

// --- machine identity (DF-30) ---------------------------------------------------------------
function readCredentials() {
  if (!existsSync(CREDENTIALS_FILE)) return null;
  const values = {};
  for (const line of readFileSync(CREDENTIALS_FILE, "utf8").split(/\r?\n/)) {
    const m = line.match(/^\s*([A-Z_]+)\s*=\s*(.*?)\s*$/);
    if (m) values[m[1]] = m[2];
  }
  const need = ["AUTHENTIK_TOKEN_URL", "AUTHENTIK_CLIENT_ID", "AUTHENTIK_USERNAME", "AUTHENTIK_APP_PASSWORD"];
  const missing = need.filter((k) => !values[k]);
  if (missing.length) throw new Error(`${CREDENTIALS_FILE} lacks ${missing.join(", ")}`);
  return values;
}

const machine = { credentials: readCredentials(), token: null, expiresAt: 0 };

async function machineToken() {
  if (!machine.credentials) return null;
  if (machine.token && Date.now() < machine.expiresAt - 60000) return machine.token;
  const c = machine.credentials;
  const response = await fetch(c.AUTHENTIK_TOKEN_URL, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams({
      grant_type: "client_credentials",
      client_id: c.AUTHENTIK_CLIENT_ID,
      username: c.AUTHENTIK_USERNAME,
      password: c.AUTHENTIK_APP_PASSWORD,
      scope: "openid profile",
    }),
  });
  const body = await response.json().catch(() => ({}));
  if (!response.ok || !body.access_token) throw new Error(`token endpoint answered ${response.status} ${body.error || ""}`.trim());
  machine.token = body.access_token;
  machine.expiresAt = Date.now() + Number(body.expires_in || 300) * 1000;
  return machine.token;
}

class Page {
  constructor(port) {
    this.port = port;
    this.consoleErrors = [];
  }

  async open(width = 1920, height = 1080) {
    const created = await fetch(`http://127.0.0.1:${this.port}/json/new?about:blank`, { method: "PUT" });
    this.target = await created.json();
    this.cdp = cdpSession(this.target.webSocketDebuggerUrl);
    await this.cdp.opened;
    this.cdp.on("Runtime.exceptionThrown", (p) =>
      this.consoleErrors.push(p.exceptionDetails?.exception?.description || p.exceptionDetails?.text),
    );
    this.cdp.on("Log.entryAdded", (p) => {
      if (p.entry.level === "error") this.consoleErrors.push(`${p.entry.source}: ${p.entry.text} ${p.entry.url || ""}`.trim());
    });
    this.cdp.on("Runtime.consoleAPICalled", (p) => {
      if (p.type === "error") this.consoleErrors.push(`console.error: ${p.args.map((a) => a.value ?? a.description ?? "").join(" ")}`);
    });
    await this.cdp.send("Runtime.enable");
    await this.cdp.send("Log.enable");
    await this.cdp.send("Page.enable");
    if (machine.credentials) {
      // Scope the Bearer to the Workbench origin: Fetch interception adds it to matching
      // requests only, where Network.setExtraHTTPHeaders would send it everywhere.
      this.cdp.on("Fetch.requestPaused", async (p) => {
        const headers = Object.entries(p.request.headers)
          .filter(([name]) => name.toLowerCase() !== "authorization")
          .map(([name, value]) => ({ name, value }));
        try {
          headers.push({ name: "Authorization", value: `Bearer ${await machineToken()}` });
        } catch (error) {
          this.consoleErrors.push(`machine token: ${error.message}`);
        }
        await this.cdp.send("Fetch.continueRequest", { requestId: p.requestId, headers }).catch(() => {});
      });
      await this.cdp.send("Fetch.enable", { patterns: [{ urlPattern: `${BASE}/*`, requestStage: "Request" }] });
    }
    await this.cdp.send("Emulation.setDeviceMetricsOverride", { width, height, deviceScaleFactor: 1, mobile: false });
  }

  async goto(path) {
    this.consoleErrors = [];
    const loaded = new Promise((resolve) => this.cdp.on("Page.loadEventFired", resolve));
    await this.cdp.send("Page.navigate", { url: BASE + path });
    await Promise.race([loaded, sleep(30000)]);
  }

  async eval(expression) {
    const result = await this.cdp.send("Runtime.evaluate", { expression, returnByValue: true, awaitPromise: true });
    if (result.exceptionDetails) throw new Error(result.exceptionDetails.exception?.description || result.exceptionDetails.text);
    return result.result?.value;
  }

  async waitFor(expression, timeoutMs = 60000, label = expression) {
    const deadline = Date.now() + timeoutMs;
    while (Date.now() < deadline) {
      try {
        // A DOM node does not survive returnByValue, so it is reported as true.
        const value = await this.eval(`(() => { const v = (${expression}); return v instanceof Node ? true : v; })()`);
        if (value) return value;
      } catch {
        // page still loading
      }
      await sleep(500);
    }
    throw new Error(`timed out waiting for ${label}`);
  }

  async click(x, y) {
    for (const type of ["mousePressed", "mouseReleased"]) {
      await this.cdp.send("Input.dispatchMouseEvent", { type, x, y, button: "left", clickCount: 1 });
    }
  }

  async shot(name) {
    const png = await this.cdp.send("Page.captureScreenshot", { format: "png" });
    writeFileSync(join(outDir, `${name}.png`), Buffer.from(png.data, "base64"));
  }

  async api(path, init) {
    // Same-origin fetch from the page: the request carries exactly the identity the UI has.
    return this.eval(`fetch(${JSON.stringify(path)}, ${JSON.stringify(init || {})}).then(async (r) => ({ status: r.status, body: await r.json().catch(() => null) }))`);
  }

  async close() {
    this.cdp.close();
    await fetch(`http://127.0.0.1:${this.port}/json/close/${this.target.id}`).catch(() => {});
  }
}


const MODE = process.env.CASE_MODE || "REAL";
const EDIT = process.env.CASE_EDIT_RAW
  ? {
      raw: process.env.CASE_EDIT_RAW,
      person: process.env.CASE_EDIT_PERSON || "",
      kind: process.env.CASE_EDIT_KIND || "",
      status: process.env.CASE_EDIT_STATUS || "",
      basis: process.env.CASE_EDIT_BASIS || "",
      reason: process.env.CASE_EDIT_REASON || "",
    }
  : null;

// React keeps its own value tracker: set through the native setter, then fire the event React listens to.
const setField = (selector, value) => `(() => {
  const el = document.querySelector(${JSON.stringify(selector)});
  if (!el) return false;
  const proto = el.tagName === "SELECT" ? HTMLSelectElement.prototype : el.tagName === "TEXTAREA" ? HTMLTextAreaElement.prototype : HTMLInputElement.prototype;
  Object.getOwnPropertyDescriptor(proto, "value").set.call(el, ${JSON.stringify(value)});
  el.dispatchEvent(new Event(el.tagName === "SELECT" ? "change" : "input", { bubbles: true }));
  return true;
})()`;

const readPage = `(() => {
  const header = document.querySelector('[data-testid="case-header"]');
  return {
    header: header ? header.innerText : null,
    people: [...document.querySelectorAll('[data-testid="case-person"]')].map((card) => ({
      id: card.dataset.personId,
      name: card.querySelector("h2")?.innerText,
      summary: card.querySelector("header")?.innerText,
      identifiers: [...card.querySelectorAll('[data-testid="case-identifier"]')].map((row) => ({
        raw: row.dataset.identifier,
        status: row.dataset.status,
        text: row.innerText.split(String.fromCharCode(10)).join(" | ").slice(0, 400),
      })),
    })),
    unknowns: document.querySelector('[data-testid="case-unknowns"]')?.innerText.slice(0, 1500) ?? null,
  };
})()`;

const report = { base: BASE, mode: MODE, started_at: new Date().toISOString(), edit: EDIT };
report.identity = machine.credentials ? `authentik-sa:${machine.credentials.AUTHENTIK_USERNAME}` : "none";
if (machine.credentials) await machineToken();
const { proc, port } = await launchChrome();
let exitCode = 0;
const page = new Page(port);
try {
  await page.open(1920, 1200);
  await page.goto(`/case?mode=${MODE}`);
  await page.waitFor(`document.querySelector('[data-testid="case-header"]')`, 60000, "case header");
  await page.waitFor(`document.querySelectorAll('[data-testid="case-person"]').length > 0 || document.body.innerText.includes("No people")`, 30000, "people");
  await sleep(2500);
  report.api = await page.api(`/api/case-identity?mode=${MODE}`).then((r) => ({
    status: r.status,
    matter: r.body?.matter?.title,
    caption: r.body?.court_case?.caption,
    people: (r.body?.people || []).map((p) => ({ id: p.id, name: p.display_name, identifiers: p.identifiers.length })),
    catalog_available: r.body?.catalog?.available,
    catalog_error: r.body?.catalog?.error ?? null,
    catalog_counts: r.body?.catalog?.counts?.length,
    probata_counts: r.body?.probata_counts?.length,
    detail: r.status === 200 ? null : r.body,
  }));
  report.before = await page.eval(readPage);
  report.console_errors_read = [...page.consoleErrors];
  await page.shot("case-page");

  if (EDIT) {
    const opened = await page.eval(`(() => {
      const cards = [...document.querySelectorAll('[data-testid="case-person"]')];
      const card = cards.find((c) => !${JSON.stringify(EDIT.person)} || c.querySelector("h2")?.innerText === ${JSON.stringify(EDIT.person)});
      const row = card && [...card.querySelectorAll('[data-testid="case-identifier"]')].find((r) => r.dataset.identifier === ${JSON.stringify(EDIT.raw)});
      const button = row && row.querySelector('button[aria-label="Edit"]');
      if (!button) return false;
      button.click();
      return true;
    })()`);
    if (!opened) throw new Error(`no Edit button for ${EDIT.raw}`);
    await page.waitFor(`document.querySelector('[role="dialog"] textarea[name="basis"]')`, 15000, "edit dialog");
    for (const [name, value] of [["kind", EDIT.kind], ["status", EDIT.status], ["basis", EDIT.basis], ["change_reason", EDIT.reason]]) {
      if (!value) continue;
      const tag = name === "basis" ? "textarea" : name === "kind" || name === "status" ? "select" : "input";
      if (!(await page.eval(setField(`[role="dialog"] ${tag}[name="${name}"]`, value)))) throw new Error(`no ${name} field`);
    }
    await sleep(500);
    await page.shot("case-edit-dialog");
    const submitted = await page.eval(`(() => {
      const button = [...document.querySelectorAll('[role="dialog"] button[type="submit"]')].find((b) => !b.disabled);
      if (!button) return false;
      button.click();
      return true;
    })()`);
    if (!submitted) throw new Error("the save button is disabled");
    await page.waitFor(`!document.querySelector('[role="dialog"] textarea[name="basis"]') && document.body.innerText.includes("Saved as a new version")`, 30000, "saved toast");
    await sleep(2500);
    report.after = await page.eval(readPage);
    await page.shot("case-page-after-edit");
    const versions = await page.eval(`(() => {
      const row = [...document.querySelectorAll('[data-testid="case-identifier"]')].find((r) => r.dataset.identifier === ${JSON.stringify(EDIT.raw)});
      row?.querySelector('button[aria-label="Versions"]')?.click();
      return Boolean(row);
    })()`);
    if (versions) {
      await page.waitFor(`document.body.innerText.includes("Versions of")`, 15000, "versions drawer");
      await sleep(1000);
      report.versions_drawer = await page.eval(`document.querySelector('[role="dialog"]')?.innerText.slice(0, 2000)`);
      await page.shot("case-versions");
    }
    report.console_errors_edit = [...page.consoleErrors];
  }
  await page.close();
} catch (error) {
  console.error(`case page proof aborted: ${error.message}`);
  report.aborted = error.message;
  report.aborted_console_errors = [...page.consoleErrors];
  await page.shot("case-aborted").catch(() => {});
  exitCode = 1;
} finally {
  proc.kill();
}
report.finished_at = new Date().toISOString();
writeFileSync(join(outDir, "case-page.json"), JSON.stringify(report, null, 1));
console.log(JSON.stringify({ api: report.api, aborted: report.aborted ?? null }));
process.exit(exitCode);

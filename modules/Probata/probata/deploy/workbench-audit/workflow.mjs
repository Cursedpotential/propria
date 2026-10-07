// Byline: Codex · GPT-6 · 2026-10-06.
/** Prove the Sources → Activity → Read workflow through read-only remote browser interactions.
 * Inputs: private output directory; existing audit.sh Chrome/AuthentiK environment.
 * Outputs: redacted workflow.json/counts and private workflow-*.png screenshots.
 * Effects: remote Chrome/CDP, machine-token exchange, GETs and two allowlisted lookup POSTs.
 * Pick with AUDIT_SCRIPT=workflow.mjs after integration/deploy; never run a desktop browser.
 * Plumbing follows case-page.mjs; that script cannot be imported without executing its audit.
 * Run `node workflow.mjs --self-test` anywhere for pure, browser-free guard/report checks.
 */
import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { chmodSync, existsSync, mkdirSync, mkdtempSync, readFileSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { setTimeout as sleep } from "node:timers/promises";

const BASE = (process.env.WORKBENCH_URL || "https://workbench.tilapia-skilift.ts.net").replace(/\/$/, "");
const CHROME_BIN = process.env.CHROME_BIN || "/opt/google/chrome/chrome";
const CREDENTIALS_FILE = process.env.AUTHENTIK_MACHINE_CREDENTIALS_FILE || "/run/secrets/devbox-authentik.env";
const WAIT_MS = 30_000;
const CDP_MS = 20_000;
const DESKTOP_ROW_HEIGHT = 34; // ImportedGrid's desktop header and row height.
// These existing handlers only read manifests/catalog membership; all other POSTs fail closed.
const READ_POSTS = new Set(["/api/proffer/decoded/exists", "/api/intake/discovery/unit-lookup"]);

/** Classify browser requests without permitting corpus/job/gate mutations.
 * Inputs: request URL/method and Workbench origin. Output: allow/deny boolean.
 * Effects: none. Pick before Fetch.continueRequest; POST exceptions are exact known read paths.
 */
export function allowedRequest(url, method, base = BASE) {
  if (["GET", "HEAD", "OPTIONS"].includes(method)) return true;
  const target = new URL(url);
  return target.origin === new URL(base).origin && method === "POST" && READ_POSTS.has(target.pathname);
}

/** Count proof outcomes without carrying private URLs, names, IDs, search words or error text.
 * Input: fixed-name check records. Output: numeric totals; effects: none.
 * Pick for stdout and the report header; screenshots are deliberately private.
 */
export function outcomeCounts(checks) {
  return {
    total: checks.length,
    pass: checks.filter((check) => check.status === "pass").length,
    fail: checks.filter((check) => check.status === "fail").length,
    blocked: checks.filter((check) => check.status === "blocked").length,
  };
}

/** Describe a safe fixed failure code without retaining underlying exception text.
 * Inputs: fixed code and whether prerequisite data is absent. Output: an Error.
 * Effects: none. Pick for assertions; raw CDP/DOM errors never enter the report.
 */
function proofError(code, blocked = false) {
  return Object.assign(new Error(code), { proofCode: code, blocked });
}

/** Assert a read-only proof condition using a static, redacted failure code.
 * Inputs: condition and code. Output: none; effects: throws on failure.
 * Pick instead of persisting DOM/API response bodies in assertion errors.
 */
function ensure(condition, code) {
  if (!condition) throw proofError(code);
}

/** Parse the devbox machine credentials without executing or logging the secret file.
 * Input: configured credential file. Output: in-memory values; effects: one file read.
 * Pick for the same client-credentials flow as case-page.mjs; no tunnel fallback here.
 */
function readCredentials() {
  ensure(existsSync(CREDENTIALS_FILE), "machine_credentials_unavailable");
  const values = {};
  for (const line of readFileSync(CREDENTIALS_FILE, "utf8").split(/\r?\n/)) {
    const match = line.match(/^\s*([A-Z_]+)\s*=\s*(.*?)\s*$/);
    if (match) values[match[1]] = match[2];
  }
  ensure(["AUTHENTIK_TOKEN_URL", "AUTHENTIK_CLIENT_ID", "AUTHENTIK_USERNAME", "AUTHENTIK_APP_PASSWORD"].every((key) => values[key]), "machine_credentials_incomplete");
  return values;
}

/** Create the existing devbox Authentik token cache, scoped to this audit process.
 * Input: parsed credentials. Output: async token getter; effects: OAuth POST on cache expiry.
 * Pick for request interception; never expose the token through DOM, reports or console.
 */
function tokenGetter(credentials) {
  let token;
  let expiresAt = 0;
  let pending;
  return async () => {
    if (token && Date.now() < expiresAt - 60_000) return token;
    if (pending) return pending;
    pending = (async () => {
      const response = await fetch(credentials.AUTHENTIK_TOKEN_URL, {
        method: "POST", signal: AbortSignal.timeout(CDP_MS),
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
        body: new URLSearchParams({ grant_type: "client_credentials", client_id: credentials.AUTHENTIK_CLIENT_ID, username: credentials.AUTHENTIK_USERNAME, password: credentials.AUTHENTIK_APP_PASSWORD, scope: "openid profile" }),
      });
      const body = await response.json().catch(() => ({}));
      ensure(response.ok && body.access_token, "machine_auth_failed");
      token = body.access_token;
      expiresAt = Date.now() + Number(body.expires_in || 300) * 1000;
      return token;
    })();
    try { return await pending; } finally { pending = null; }
  };
}

/** Launch only a devbox Linux Chrome child with an ephemeral loopback CDP port.
 * Input: configured binary/resolver rule. Output: child and port; effects: remote browser/profile.
 * Pick after the platform and machine-credential gates; never deletes retained profile files.
 */
async function launchChrome() {
  ensure(process.platform === "linux", "remote_linux_devbox_required");
  const profile = mkdtempSync(join(tmpdir(), "workflow-proof-"));
  const args = ["--headless=new", "--remote-debugging-port=0", "--remote-debugging-address=127.0.0.1", `--user-data-dir=${profile}`, "--no-first-run", "--no-default-browser-check", "--disable-gpu", "--mute-audio"];
  if (process.env.AUDIT_RESOLVER_RULE) args.push(`--host-resolver-rules=${process.env.AUDIT_RESOLVER_RULE}`);
  const proc = spawn(CHROME_BIN, [...args, "about:blank"], { stdio: "ignore" });
  let failed = false;
  proc.on("error", () => { failed = true; });
  for (let count = 0; count < 200; count += 1) {
    const portFile = join(profile, "DevToolsActivePort");
    if (existsSync(portFile)) {
      const port = readFileSync(portFile, "utf8").split("\n")[0];
      if (/^\d+$/.test(port)) return { proc, port };
    }
    if (failed || proc.exitCode !== null) break;
    await sleep(100);
  }
  proc.kill();
  throw proofError("chrome_launch_failed");
}

/** Connect to Chrome with bounded CDP commands and redacted transport failures.
 * Inputs: target WebSocket URL and redacted event-failure counter callback.
 * Output: send/on/close session; effects: loopback WebSocket.
 * Pick for the existing case-page.mjs protocol rather than adding a browser dependency.
 */
function cdpSession(url, onEventFailure) {
  const ws = new WebSocket(url);
  let nextId = 0;
  const pending = new Map();
  const listeners = new Map();
  const opened = new Promise((resolve, reject) => {
    const timer = setTimeout(() => reject(proofError("cdp_open_timeout")), CDP_MS);
    ws.onopen = () => { clearTimeout(timer); resolve(); };
    ws.onerror = () => { clearTimeout(timer); reject(proofError("cdp_transport_failed")); };
  });
  ws.onmessage = (event) => {
    let message;
    try { message = JSON.parse(event.data); } catch { onEventFailure(); return; }
    if (message.id && pending.has(message.id)) {
      const command = pending.get(message.id);
      pending.delete(message.id);
      clearTimeout(command.timer);
      if (message.error) command.reject(proofError("cdp_command_failed"));
      else command.resolve(message.result);
    } else {
      for (const listener of listeners.get(message.method) || []) {
        Promise.resolve().then(() => listener(message.params)).catch(onEventFailure);
      }
    }
  };
  return {
    opened,
    send(method, params = {}) {
      return new Promise((resolve, reject) => {
        const id = ++nextId;
        const timer = setTimeout(() => { pending.delete(id); reject(proofError("cdp_command_timeout")); }, CDP_MS);
        pending.set(id, { resolve, reject, timer });
        try { ws.send(JSON.stringify({ id, method, params })); }
        catch { clearTimeout(timer); pending.delete(id); reject(proofError("cdp_transport_failed")); }
      });
    },
    on(method, listener) { listeners.set(method, [...(listeners.get(method) || []), listener]); },
    close() {
      for (const command of pending.values()) { clearTimeout(command.timer); command.reject(proofError("cdp_closed")); }
      pending.clear();
      ws.close();
    },
  };
}

/** Drive one remote page while blocking mutation requests and retaining only error counts.
 * Inputs: Chrome port, token getter and private outDir. Output: page helper instance.
 * Effects: CDP navigation/clicks/screenshots; only allowlisted reads reach the application.
 * Pick for this workflow proof; no job, gate, edit, cancel or promotion controls are clicked.
 */
class Page {
  /** Initialize audit state from port, token getter and outDir; no I/O until open(). */
  constructor(port, getToken, outDir) {
    this.port = port;
    this.getToken = getToken;
    this.outDir = outDir;
    this.loadedDocuments = new Set();
    this.counts = { blocked_requests: 0, auth_failures: 0, interception_failures: 0, cdp_event_failures: 0, runtime_errors: 0 };
  }
  /** Open an authenticated remote target with interception enabled.
   * Inputs: constructor state. Output: initialized session; effects: remote CDP target.
   * Pick before other page methods; credentials stay in the interception process.
   */
  async open() {
    const response = await fetch(`http://127.0.0.1:${this.port}/json/new?about:blank`, { method: "PUT", signal: AbortSignal.timeout(CDP_MS) });
    this.target = await response.json();
    this.cdp = cdpSession(this.target.webSocketDebuggerUrl, () => { this.counts.cdp_event_failures += 1; });
    await this.cdp.opened;
    this.cdp.on("Runtime.exceptionThrown", () => { this.counts.runtime_errors += 1; });
    this.cdp.on("Page.lifecycleEvent", (event) => {
      if (event.name === "DOMContentLoaded") this.loadedDocuments.add(event.loaderId);
    });
    this.cdp.on("Fetch.requestPaused", async (event) => {
      try {
        if (!allowedRequest(event.request.url, event.request.method)) {
          this.counts.blocked_requests += 1;
          await this.cdp.send("Fetch.failRequest", { requestId: event.requestId, errorReason: "BlockedByClient" });
          return;
        }
        const headers = Object.entries(event.request.headers).filter(([name]) => name.toLowerCase() !== "authorization").map(([name, value]) => ({ name, value }));
        if (new URL(event.request.url).origin === new URL(BASE).origin) {
          try { headers.push({ name: "Authorization", value: `Bearer ${await this.getToken()}` }); }
          catch { this.counts.auth_failures += 1; throw proofError("machine_auth_failed"); }
        }
        await this.cdp.send("Fetch.continueRequest", { requestId: event.requestId, headers });
      } catch {
        this.counts.interception_failures += 1;
        await this.cdp.send("Fetch.failRequest", { requestId: event.requestId, errorReason: "BlockedByClient" }).catch(() => { this.counts.cdp_event_failures += 1; });
      }
    });
    await this.cdp.send("Runtime.enable");
    await this.cdp.send("Page.enable");
    await this.cdp.send("Page.setLifecycleEventsEnabled", { enabled: true });
    await this.cdp.send("Fetch.enable", { patterns: [{ urlPattern: "*", requestStage: "Request" }] });
    await this.cdp.send("Emulation.setDeviceMetricsOverride", { width: 1920, height: 1200, deviceScaleFactor: 1, mobile: false });
  }
  /** Evaluate a bounded DOM/read operation.
   * Input: private expression. Output: explicit serializable value; effects: specified page read.
   * Pick for assertions; exception details are replaced with a fixed failure code.
   */
  async eval(expression) {
    const result = await this.cdp.send("Runtime.evaluate", { expression, returnByValue: true, awaitPromise: true });
    if (result.exceptionDetails) throw proofError("page_evaluation_failed");
    return result.result?.value;
  }
  /** Poll a boolean DOM condition without recording its expression.
   * Inputs: expression, fixed failure code, bounded timeout. Output: none; effects: page reads.
   * Pick for asynchronous rendering; a missing condition throws the supplied static code.
   */
  async waitFor(expression, code, timeout = WAIT_MS) {
    const deadline = Date.now() + timeout;
    while (Date.now() < deadline) {
      if (await this.eval(`Boolean(${expression})`).catch(() => false)) return;
      await sleep(250);
    }
    throw proofError(code);
  }
  /** Navigate to one same-origin path and wait for that document to become interactive.
   * Input: relative path/query. Output: none; effects: navigation and authenticated reads.
   * Pick for an independent journey; loader identity prevents assertions against stale DOM.
   */
  async goto(path) {
    ensure(path.startsWith("/") && !path.startsWith("//"), "same_origin_navigation_required");
    this.loadedDocuments.clear();
    const result = await this.cdp.send("Page.navigate", { url: BASE + path });
    ensure(!result.errorText, "page_navigation_failed");
    if (result.loaderId) {
      const deadline = Date.now() + WAIT_MS;
      while (!this.loadedDocuments.has(result.loaderId) && Date.now() < deadline) await sleep(100);
      ensure(this.loadedDocuments.has(result.loaderId), "page_document_load_timeout");
    }
  }
  /** Dispatch a physical mouse click at a measured element location.
   * Inputs: viewport x/y. Output: none; effects: remote mouse events.
   * Pick for a measured canvas cell or a specifically identified read control.
   */
  async click(x, y) {
    for (const type of ["mousePressed", "mouseReleased"]) await this.cdp.send("Input.dispatchMouseEvent", { type, x, y, button: "left", clickCount: 1 });
  }
  /** Find and physically click a specific read/navigation control.
   * Input: expression selecting a known control. Output: none; effects: scroll and click.
   * Pick for semantic selectors; unavailable/disabled controls fail explicitly.
   */
  async clickElement(expression) {
    const point = await this.eval(`(() => { const el = (${expression}); if (!el || el.disabled) return null; el.scrollIntoView({ block: 'center' }); const r = el.getBoundingClientRect(); return r.width && r.height ? { x: r.x + r.width / 2, y: r.y + r.height / 2 } : null; })()`);
    ensure(point, "read_control_unavailable");
    await this.click(point.x, point.y);
  }
  /** Set a React-controlled filter through its native setter and expected event.
   * Inputs: filter selector and value. Output: none; effects: local filter/navigation events.
   * Pick for known input/select filters; never used for edit forms or job arguments.
   */
  async field(selector, value) {
    ensure(await this.eval(`(() => { const el = document.querySelector(${JSON.stringify(selector)}); if (!el) return false; const select = el.tagName === 'SELECT'; Object.getOwnPropertyDescriptor(select ? HTMLSelectElement.prototype : HTMLInputElement.prototype, 'value').set.call(el, ${JSON.stringify(value)}); el.dispatchEvent(new Event(select ? 'change' : 'input', { bubbles: true })); return true; })()`), "filter_field_unavailable");
  }
  /** Read an existing same-origin GET API in the page's machine-auth context.
   * Input: /api path/query. Output: private in-memory JSON; effects: one GET.
   * Pick for bounded fixture discovery; non-200 responses fail without body disclosure.
   */
  async api(path) {
    ensure(path.startsWith("/api/") && !path.startsWith("//"), "read_api_path_required");
    const response = await this.eval(`fetch(${JSON.stringify(path)}).then(async r => ({ status: r.status, body: await r.json().catch(() => null) }))`);
    ensure(response?.status === 200, "read_api_unavailable");
    return response.body;
  }
  /** Write a private screenshot with a static filename.
   * Input: fixed screenshot name. Output: private PNG; effects: CDP capture and mode-0600 write.
   * Pick for visual evidence; filenames never include record identity.
   */
  async shot(name) {
    const png = await this.cdp.send("Page.captureScreenshot", { format: "png" });
    writeFileSync(join(this.outDir, `${name}.png`), Buffer.from(png.data, "base64"), { mode: 0o600 });
  }
  /** Close this audit's remote target and CDP socket.
   * Input: open session. Output: none; effects: closes audit-owned browser resources.
   * Pick during finalization; no host/service lifecycle action occurs.
   */
  async close() {
    this.cdp?.close();
    if (this.target) await fetch(`http://127.0.0.1:${this.port}/json/close/${this.target.id}`, { signal: AbortSignal.timeout(CDP_MS) }).catch(() => {});
  }
}

/** Find a bounded, existing source/thread/message fixture without creating or modifying records.
 * Input: authenticated page. Output: private in-memory fixture; effects: bounded GETs.
 * Pick for real UI selection; absence blocks the proof rather than seeding canonical data.
 */
async function readingFixture(page) {
  const sources = await page.api("/api/imported/sources?limit=20&offset=0");
  const candidates = sources.items.filter((source) => source.messages > 0 && source.file_name && sources.items.filter((other) => other.file_name.toLowerCase().includes(source.file_name.toLowerCase())).length === 1).slice(0, 5);
  for (const source of candidates) {
    const threads = await page.api(`/api/imported/sources/${encodeURIComponent(source.id)}/threads?limit=25&offset=0`);
    const thread = threads.items.find((item) => item.messages > 0);
    if (!thread) continue;
    const messages = await page.api(`/api/imported/threads/${encodeURIComponent(thread.id)}/messages?limit=40&direction=before`);
    const message = messages.items.find((item) => item.body?.match(/[a-zA-Z]{4,}/));
    if (message) {
      const words = [...new Set(message.body.match(/[a-zA-Z]{4,}/g))].sort((left, right) => right.length - left.length).slice(0, 3);
      return { source, thread, message, words };
    }
  }
  throw proofError("existing_readable_fixture_unavailable", true);
}

/** Execute independent user journeys and persist only static verdicts and private screenshots.
 * Input: private outDir. Output: redacted report and process outcome; effects: remote reads only.
 * Pick through audit.sh after the parent's router, Read and Activity commits are deployed.
 */
async function run(outDir) {
  ensure(process.platform === "linux", "remote_linux_devbox_required");
  ensure(outDir, "private_output_directory_required");
  ensure(new URL(BASE).protocol === "https:", "https_workbench_required");
  mkdirSync(outDir, { recursive: true, mode: 0o700 });
  chmodSync(outDir, 0o700);
  const report = { schema: 1, started_at: new Date().toISOString(), read_only: true, machine_auth: false, checks: [] };
  let chrome;
  let page;
  let fixture;
  /** Record one fixed-name check, preserving failure evidence without private exception text.
   * Inputs: static name and read-only action. Output: verdict; effects: private screenshot/report row.
   * Pick for every journey so later independent checks still run after a failure.
   */
  async function check(name, action) {
    let status = "pass";
    let code = "ok";
    try { await action(); } catch (error) { status = error.blocked ? "blocked" : "fail"; code = error.proofCode || "unexpected_error"; }
    const screenshot = `workflow-${String(report.checks.length + 1).padStart(2, "0")}-${name}`;
    try { await page.shot(screenshot); } catch { status = "fail"; code = "screenshot_failed"; }
    report.checks.push({ name, status, code, screenshot: `${screenshot}.png` });
  }
  try {
    const getToken = tokenGetter(readCredentials());
    await getToken();
    report.machine_auth = true;
    chrome = await launchChrome();
    page = new Page(chrome.port, getToken, outDir);
    await page.open();
    await page.goto("/sources");
    const routes = [
      ["sources", "/sources", "document.querySelector('[data-testid=\"sources-screen\"]')"],
      ["activity", "/activity", "document.querySelector('#activity-ledger-heading')"],
      ["read", "/read", "document.querySelector('#read-content-search')"],
      ["case", "/case", "document.querySelector('[data-testid=\"case-header\"]')"],
    ];
    for (const [name, path, ready] of routes) {
      await check(`nav-${name}`, async () => {
        await page.waitFor(`document.querySelector('a[href="${path}"]')`, "navigation_link_unavailable");
        await page.clickElement(`document.querySelector('a[href="${path}"]')`);
        await page.waitFor(`location.pathname === ${JSON.stringify(path)} && (${ready})`, "navigation_destination_unavailable");
      });
    }
    await check("sources-folder-navigation", async () => {
      await page.goto("/sources");
      await page.waitFor("document.querySelector('[aria-label=\"Source location\"] option[value]:not([value=\"\"])') && document.querySelector('button[aria-label^=\"Open folder \"]')", "source_folders_unavailable");
      await page.clickElement("document.querySelector('button[aria-label^=\"Open folder \"]')");
      await page.waitFor("new URLSearchParams(location.search).get('prefix') && document.querySelector('[aria-label=\"Folder path\"] button[title]')", "folder_location_not_preserved");
      const prefix = await page.eval("new URLSearchParams(location.search).get('prefix')");
      await page.goto(await page.eval("location.pathname + location.search"));
      await page.waitFor(`new URLSearchParams(location.search).get('prefix') === ${JSON.stringify(prefix)} && document.querySelector('[aria-label="Folder path"] button[title]')`, "folder_reload_lost_location");
      await page.clickElement("document.querySelector('[aria-label=\"Folder path\"] button')");
      await page.waitFor("!new URLSearchParams(location.search).get('prefix')", "folder_root_return_failed");
    });
    await check("sources-indexed-search", async () => {
      await page.goto("/sources");
      const mode = "[...document.querySelectorAll('[aria-label=\"Search mode\"] button')].find(b => b.textContent.trim() === 'Contents')";
      await page.waitFor(mode, "source_contents_capability_unavailable");
      await page.clickElement(mode);
      await page.field('[aria-label="Search the corpus"]', "document");
      await page.waitFor("!document.querySelector('[aria-label=\"Search sources\"] button[type=\"submit\"]').disabled", "source_search_not_ready");
      await page.clickElement("document.querySelector('[aria-label=\"Search sources\"] button[type=\"submit\"]')");
      const result = "[...document.querySelectorAll('[data-testid=\"sources-screen\"] li details')].find(d => d.querySelector('dl dt')?.textContent === 'Recorded source')";
      await page.waitFor(result, "source_index_results_unavailable");
      await page.clickElement(`(${result}).querySelector('summary')`);
      ensure(await page.eval(`(${result}).open && Boolean((${result}).querySelector('dd')?.textContent)`), "source_excerpt_or_locator_unavailable");
    });
    await check("read-source-thread", async () => {
      await page.goto("/read");
      await page.waitFor("document.querySelector('#read-content-search')", "read_page_unavailable");
      fixture = await readingFixture(page);
      const retainedQuery = fixture.words[0];
      await page.goto(`/read?${new URLSearchParams({ q: retainedQuery })}`);
      await page.waitFor("document.querySelector('[aria-label=\"Filter loaded source files\"]')", "source_picker_unavailable");
      await page.field('[aria-label="Filter loaded source files"]', fixture.source.file_name);
      await page.waitFor("document.querySelector('[aria-label=\"Source files\"] p[aria-live=\"polite\"]')?.textContent.trim().startsWith('1 row ·') && document.querySelector('[aria-label=\"Source files\"] [data-testid=\"imported-grid\"] canvas')", "source_grid_filter_not_unique");
      // Glide renders canvas cells; use its documented sibling row/header dimensions, then verify exact selected ID.
      const point = await page.eval(`(() => { const el = document.querySelector('[aria-label="Source files"] [data-testid="imported-grid"] canvas'); el.scrollIntoView({ block: 'center' }); const r = el.getBoundingClientRect(); return { x: r.x + 60, y: r.y + ${DESKTOP_ROW_HEIGHT * 1.5} }; })()`);
      await page.click(point.x, point.y);
      await page.waitFor(`new URLSearchParams(location.search).get('source') === ${JSON.stringify(fixture.source.id)} && new URLSearchParams(location.search).get('q') === ${JSON.stringify(retainedQuery)}`, "source_selection_identity_or_query_mismatch");
      const link = `[...document.querySelectorAll('#read-conversations a')].find(a => new URL(a.href).searchParams.get('thread') === ${JSON.stringify(fixture.thread.id)})`;
      await page.waitFor(link, "thread_link_unavailable");
      await page.clickElement(link);
      await page.waitFor(`new URLSearchParams(location.search).get('thread') === ${JSON.stringify(fixture.thread.id)} && new URLSearchParams(location.search).get('source') === ${JSON.stringify(fixture.source.id)} && new URLSearchParams(location.search).get('q') === ${JSON.stringify(retainedQuery)} && document.querySelector('[aria-label="Messages"] [data-testid="message-bubble"]')`, "thread_messages_or_context_unavailable");
    });
    await check("read-message-citation", async () => {
      if (!fixture) throw proofError("existing_readable_fixture_unavailable", true);
      const record = fixture.message.id;
      await page.goto(`/read?${new URLSearchParams({ source: fixture.source.id, thread: fixture.thread.id, around: record })}`);
      const article = `document.getElementById(${JSON.stringify(`read-message-${record}`)})`;
      await page.waitFor(article, "exact_message_not_loaded");
      await page.clickElement(`(${article}).querySelector('summary')`);
      const link = `(${article}).querySelector('a')`;
      ensure(await page.eval(`(() => { const url = new URL((${link}).href); return url.pathname === '/read' && url.searchParams.get('source') === ${JSON.stringify(fixture.source.id)} && url.searchParams.get('thread') === ${JSON.stringify(fixture.thread.id)} && url.searchParams.get('around') === ${JSON.stringify(record)}; })()`), "citation_identity_mismatch");
      await page.clickElement(link);
      await page.waitFor(`(${article})?.querySelector('.ring-2')`, "message_focus_not_highlighted");
    });
    await check("read-extracted-context", async () => {
      if (!fixture) throw proofError("existing_readable_fixture_unavailable", true);
      const data = await page.api(`/api/imported/threads/${encodeURIComponent(fixture.thread.id)}/extractions`);
      ensure(Array.isArray(data.extractors), "extraction_response_invalid");
      await page.goto(`/read?${new URLSearchParams({ source: fixture.source.id, thread: fixture.thread.id })}`);
      await page.waitFor("document.querySelector('[aria-label=\"Extracted context and citations\"]')", "extracted_context_panel_unavailable");
      await page.waitFor("(() => { const panel = document.querySelector('[aria-label=\"Extracted context and citations\"]'); return panel && !panel.querySelector('[role=\"alert\"]') && !panel.querySelector('[data-slot=\"skeleton\"]'); })()", "extracted_context_not_loaded");
    });
    await check("read-content-search", async () => {
      if (!fixture) throw proofError("existing_readable_fixture_unavailable", true);
      let word;
      for (const candidate of [...fixture.words, "the", "you"]) {
        const hits = await page.api(`/api/imported/search?${new URLSearchParams({ q: candidate, limit: "20", offset: "0" })}`);
        if (hits.items.some((hit) => hit.thread_id)) { word = candidate; break; }
      }
      if (!word) throw proofError("existing_searchable_fixture_unavailable", true);
      await page.goto("/read");
      await page.waitFor("document.querySelector('#read-content-search')", "search_input_unavailable");
      await page.field("#read-content-search", word);
      await page.clickElement("document.querySelector('[aria-label=\"Content search\"] button[type=\"submit\"]')");
      await page.waitFor(`new URLSearchParams(location.search).get('q') === ${JSON.stringify(word)}`, "search_query_not_preserved");
      const resultLink = "[...document.querySelectorAll('[aria-label=\"Content search\"] a')].find(a => a.textContent.trim() === 'Read in conversation')";
      await page.waitFor(resultLink, "search_index_result_unavailable");
      const href = await page.eval(`(${resultLink}).getAttribute('href')`);
      const expected = new URL(href, BASE).searchParams;
      ensure(expected.get("thread") && expected.get("around"), "search_result_locator_missing");
      await page.clickElement(resultLink);
      await page.waitFor(`new URLSearchParams(location.search).get('thread') === ${JSON.stringify(expected.get("thread"))} && new URLSearchParams(location.search).get('around') === ${JSON.stringify(expected.get("around"))} && new URLSearchParams(location.search).get('q') === ${JSON.stringify(word)} && document.getElementById(${JSON.stringify(`read-message-${expected.get("around")}`)})`, "search_result_exact_message_unavailable");
    });
    await check("read-processing-previews-return", async () => {
      if (!fixture) throw proofError("existing_readable_fixture_unavailable", true);
      const context = new URLSearchParams({ source: fixture.source.id, thread: fixture.thread.id, around: fixture.message.id, q: fixture.words[0] });
      await page.goto(`/read?${context}`);
      const previews = "[...document.querySelectorAll('a')].find(a => a.textContent.trim() === 'Processing previews')";
      await page.waitFor(previews, "processing_previews_link_unavailable");
      await page.clickElement(previews);
      await page.waitFor("new URLSearchParams(location.search).get('view') === 'review' && document.querySelector('#review-resources-heading')", "processing_previews_not_opened");
      const back = "[...document.querySelectorAll('a')].find(a => a.textContent.trim() === 'Back to imported reading')";
      await page.clickElement(back);
      await page.waitFor(`document.querySelector('#read-content-search') && (() => { const expected = new URLSearchParams(${JSON.stringify(context.toString())}); const actual = new URLSearchParams(location.search); return [...expected].every(([key, value]) => actual.get(key) === value) && !actual.has('view') && !actual.has('resource') && !actual.has('preview_handle') && !actual.has('attempt'); })()`, "imported_reading_context_not_restored");
    });
    await check("activity-filters", async () => {
      await page.goto("/activity");
      await page.waitFor("document.querySelector('#activity-ledger-heading')", "activity_page_unavailable");
      await page.field('input[placeholder="Filename or folder"]', "workflow-proof-no-match-20261006");
      await page.waitFor("new URLSearchParams(location.search).get('q') === 'workflow-proof-no-match-20261006'", "activity_source_filter_not_preserved");
      await page.field('section[aria-labelledby="activity-ledger-heading"] select', "completed");
      await page.waitFor("new URLSearchParams(location.search).get('status') === 'completed' && new URLSearchParams(location.search).get('q') === 'workflow-proof-no-match-20261006'", "activity_status_filter_not_preserved");
      await page.waitFor("document.body.innerText.includes('No imports match these filters') || document.body.innerText.includes('No imports yet')", "activity_filtered_empty_unavailable");
    });
    await check("activity-details", async () => {
      await page.goto("/activity");
      const button = "[...document.querySelectorAll('ul[aria-label=\"Proffer imports\"] button')].find(b => b.textContent.trim() === 'Details')";
      await page.waitFor(`(${button}) || document.body.innerText.includes('No imports yet') || document.querySelector('section[aria-labelledby="activity-ledger-heading"] [role="alert"]')`, "activity_list_not_settled");
      ensure(await page.eval("document.querySelector('section[aria-labelledby=\"activity-ledger-heading\"] [role=\"alert\"]') === null"), "activity_list_failed");
      if (!await page.eval(`Boolean(${button})`)) throw proofError("existing_activity_detail_unavailable", true);
      const handle = await page.eval(`new URL((${button}).closest('article').querySelector('a').href).searchParams.get('resource')`);
      ensure(handle, "activity_row_locator_missing");
      await page.clickElement(button);
      await page.waitFor(`document.querySelector('[aria-label="Import details"] details summary') && new URLSearchParams(location.search).get('preview_handle') === ${JSON.stringify(handle)}`, "activity_details_not_loaded");
      await page.clickElement("document.querySelector('[aria-label=\"Import details\"] details summary')");
      ensure(await page.eval("document.querySelector('[aria-label=\"Import details\"] details').open"), "activity_technical_details_not_open");
      ensure(await page.eval(`document.querySelector('[aria-label="Import details"] details').textContent.includes(${JSON.stringify(handle)})`), "activity_detail_locator_mismatch");
      await page.shot("workflow-activity-details-open");
      await page.clickElement("[...document.querySelectorAll('[aria-label=\"Import details\"] button')].find(b => b.textContent.trim() === 'Close')");
      await page.waitFor("!new URLSearchParams(location.search).has('operation') && !new URLSearchParams(location.search).has('preview_handle') && !document.querySelector('[aria-label=\"Import details\"]')", "activity_details_not_closed");
    });
    await check("legacy-review-opaque-query", async () => {
      const query = new URLSearchParams({ source: "source/+==", thread: "thread/+==", around: "message/+==", q: "opaque + & ? = /", attempt: "a".repeat(32), mode: "LIVE" });
      await page.goto(`/review?${query}#workflow-proof`);
      await page.waitFor("location.pathname === '/read' && document.querySelector('#review-resources-heading')", "legacy_review_not_redirected_to_preview");
      ensure(await page.eval(`(() => { const expected = new URLSearchParams(${JSON.stringify(query.toString())}); const actual = new URLSearchParams(location.search); return [...expected].every(([key, value]) => actual.get(key) === value) && location.hash === '#workflow-proof'; })()`), "legacy_review_query_changed");
    });
    await check("legacy-review-default-preview", async () => {
      const query = new URLSearchParams({ q: "opaque + & ? = /", opaque_context: "context/+==", mode: "LIVE" });
      await page.goto(`/review?${query}`);
      await page.waitFor("location.pathname === '/read' && new URLSearchParams(location.search).get('view') === 'review' && document.querySelector('#review-resources-heading')", "legacy_review_default_preview_unavailable");
      ensure(await page.eval(`(() => { const expected = new URLSearchParams(${JSON.stringify(query.toString())}); const actual = new URLSearchParams(location.search); return [...expected].every(([key, value]) => actual.get(key) === value); })()`), "legacy_review_default_query_changed");
    });
    await check("read-only-network", async () => {
      ensure(page.counts.blocked_requests === 0, "mutating_browser_request_blocked");
      ensure(page.counts.auth_failures === 0 && page.counts.interception_failures === 0 && page.counts.cdp_event_failures === 0, "request_guard_or_auth_failed");
      ensure(page.counts.runtime_errors === 0, "browser_runtime_errors_detected");
    });
  } catch (error) {
    report.checks.push({ name: "audit-setup", status: "fail", code: error.proofCode || "unexpected_error" });
  } finally {
    if (page) { report.browser_counts = { ...page.counts }; await page.close().catch(() => {}); }
    chrome?.proc.kill();
  }
  report.finished_at = new Date().toISOString();
  report.counts = outcomeCounts(report.checks);
  writeFileSync(join(outDir, "workflow.json"), JSON.stringify(report, null, 2), { mode: 0o600 });
  console.log(JSON.stringify({ counts: report.counts, browser_counts: report.browser_counts ?? null }));
  process.exitCode = report.counts.fail || report.counts.blocked ? 1 : 0;
}

if (process.argv[2] === "--self-test") {
  for (const method of ["GET", "HEAD", "OPTIONS"]) assert.equal(allowedRequest(`${BASE}/api/imported/sources`, method), true);
  for (const method of ["POST", "PUT", "PATCH", "DELETE"]) assert.equal(allowedRequest(`${BASE}/api/proffer/previews/opaque/decision`, method), false);
  assert.equal(allowedRequest(`${BASE}/api/proffer/decoded/exists`, "POST"), true);
  assert.equal(allowedRequest(`${BASE}/api/intake/discovery/unit-lookup`, "POST"), true);
  assert.equal(allowedRequest("https://other.invalid/api/proffer/decoded/exists", "POST"), false);
  assert.equal(allowedRequest(`${BASE}/api/proffer/decoded/exists/other`, "POST"), false);
  assert.equal(allowedRequest(`${BASE}/api/proffer/decoded/exists`, "DELETE"), false);
  for (const path of ["/api/proffer/start", "/api/proffer/start-batch", "/api/proffer/repair/run", "/api/sources/unit-marks", "/api/proffer/previews/opaque/cancel", "/api/proffer/previews/opaque/repair-decision"]) assert.equal(allowedRequest(`${BASE}${path}`, "POST"), false);
  assert.deepEqual(outcomeCounts([{ status: "pass" }, { status: "fail" }, { status: "blocked" }]), { total: 3, pass: 1, fail: 1, blocked: 1 });
  console.log(JSON.stringify({ self_test: "pass", browser_launched: false, credential_reads: 0 }));
} else {
  await run(process.argv[2]).catch((error) => {
    console.error(JSON.stringify({ status: "fail", code: error.proofCode || "unexpected_error" }));
    process.exitCode = 1;
  });
}

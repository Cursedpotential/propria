// Byline: Codex · GPT-6 · 2026-10-06.
/** Prove the Sources → Activity → Read workflow through read-only remote browser interactions.
 * Inputs: private output directory; existing audit.sh Chrome/AuthentiK environment.
 * Outputs: redacted workflow.json/counts and private workflow-*.png screenshots.
 * Effects: remote Chrome/CDP, machine-token exchange, GETs and four exact allowlisted read POSTs.
 * Pick with AUDIT_SCRIPT=workflow.mjs after integration/deploy; never run a desktop browser.
 * Plumbing follows case-page.mjs; that script cannot be imported without executing its audit.
 * Run `node workflow.mjs --self-test` anywhere for pure, browser-free guard/contract/fetch checks.
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
const READ_POST_MS = 35_000;
const MAX_POST_BYTES = 32 << 10;
const MAX_READ_BYTES = 2 << 20;
const SEARCH_PANEL = 'section[aria-label="Search content across sources"]';
const SEARCH_INPUT = "#combined-search-query";
const SEARCH_MODES = ["keyword", "hybrid", "vector"];
const SEARCH_LEGS = ["intake", "proffer"];
const SEARCH_LIMIT = 30; // Match searchContext's UI limit so fixture ranking uses the same candidate budget.
const DESKTOP_ROW_HEIGHT = 34; // ImportedGrid's desktop header and row height.
// These existing handlers read membership/context/graph identity; every other POST fails closed.
const RETRIEVAL_POSTS = new Set(["/api/retrieval/search", "/api/retrieval/relationships"]);
const READ_POSTS = new Set(["/api/proffer/decoded/exists", "/api/intake/discovery/unit-lookup", ...RETRIEVAL_POSTS]);

/** Classify browser requests without permitting corpus/job/gate mutations.
 * Inputs: request URL/method and Workbench origin. Output: allow/deny boolean.
 * Effects: none. Pick before Fetch.continueRequest; POST exceptions are exact known read paths.
 */
export function allowedRequest(url, method, base = BASE) {
  try {
    if (["GET", "HEAD", "OPTIONS"].includes(method)) return true;
    const target = new URL(url);
    return target.origin === new URL(base).origin && !target.username && !target.password
      && method === "POST" && READ_POSTS.has(target.pathname);
  } catch { return false; }
}

/** Build one bounded same-origin context/relationship fixture POST without sending it.
 * Inputs: exact public read route and JSON object. Output: fetch specification; effects: none.
 * Pick for fixture discovery only; URLs, query aliases and arbitrary lookup/mutation routes are rejected.
 */
export function readPostRequest(path, payload) {
  ensure(RETRIEVAL_POSTS.has(path), "read_post_path_not_allowlisted");
  ensure(payload && typeof payload === "object" && !Array.isArray(payload), "read_post_object_required");
  const body = JSON.stringify(payload);
  ensure(Buffer.byteLength(body, "utf8") <= MAX_POST_BYTES, "read_post_request_too_large");
  return { path, body };
}

/** Generate a bounded page fetch expression for an already-validated fixture POST.
 * Input: readPostRequest output. Output: expression yielding private status/JSON; effects occur only when evaluated.
 * Pick through Page.readPost; no credentials enter JavaScript and redirects cannot leave the same origin.
 */
export function readPostExpression(spec) {
  return `(async () => {
    const target = new URL(${JSON.stringify(spec.path)}, location.origin);
    if (target.origin !== location.origin) throw new Error('read_post_origin');
    const response = await fetch(target.href, { method: 'POST', redirect: 'error',
      headers: { 'Content-Type': 'application/json' }, body: ${JSON.stringify(spec.body)},
      signal: AbortSignal.timeout(${READ_POST_MS}) });
    if (response.status !== 200) return { status: response.status, body: null };
    const reader = response.body?.getReader();
    if (!reader) throw new Error('read_post_body');
    const decoder = new TextDecoder(); let text = ''; let bytes = 0;
    try { while (true) {
      const part = await reader.read(); if (part.done) break;
      bytes += part.value.byteLength;
      if (bytes > ${MAX_READ_BYTES}) throw new Error('read_post_response_bound');
      text += decoder.decode(part.value, { stream: true });
    } text += decoder.decode(); } finally { await reader.cancel().catch(() => {}); }
    return { status: response.status, body: JSON.parse(text) };
  })()`;
}

/** Validate actual retrieval outcomes while allowing successful source families to return zero hits.
 * Inputs: private response, requested mode/families. Output: unchanged response; effects: safe assertion failures only.
 * Pick for API fixtures; failed legs are explicit proof failures, never disguised as missing corpus matches.
 */
export function validateRetrieval(data, mode, legs) {
  ensure(data?.mode === mode && Array.isArray(data.items) && data.items.length <= 50
    && Array.isArray(data.legs) && data.legs.length === legs.length, "retrieval_response_invalid");
  ensure(new Set(data.legs.map((leg) => leg.name)).size === legs.length
    && data.legs.every((leg) => legs.includes(leg.name) && ["success", "failed"].includes(leg.status)
      && Number.isInteger(leg.count) && leg.count >= 0), "retrieval_legs_invalid");
  const failures = data.legs.filter((leg) => leg.status === "failed").length;
  ensure(data.status === (failures === legs.length ? "failed" : failures ? "partial" : "success"), "retrieval_status_inconsistent");
  ensure(data.items.every((hit) => Number.isFinite(hit.score) && typeof hit.text === "string"
    && hit.citation?.collection && hit.citation.object_id && hit.citation.locator
    && Object.keys(hit.citation.locator).length && Array.isArray(hit.citation.source_version_ids)
    && Array.isArray(hit.legs) && hit.legs.length && hit.legs.every((leg) => legs.includes(leg))), "retrieval_citation_invalid");
  ensure(failures === 0, "retrieval_leg_failed");
  return data;
}

/** Parse the parent graphReference contract without deriving a record key from source identity.
 * Input: returned typed reference. Output: table/key/canonical identity or null; effects: none.
 * Pick before neighborhood GETs and connected-edge assertions; original SDK IDs remain available separately.
 */
export function graphReference(value) {
  if (typeof value !== "string") return null;
  const match = /^([A-Za-z_][A-Za-z0-9_]{0,63}):(?:([A-Za-z0-9_-]{1,255})|⟨([0-9]{64})⟩)$/.exec(value);
  if (!match) return null;
  const key = match[2] ?? match[3];
  return { table: match[1], key, identity: `${match[1]}:${key}` };
}

/** Validate graph resolver provenance without picking a newest snapshot or synthesizing a graph ID.
 * Inputs: response and exact cited source/document pair. Output: unchanged matches; effects: safe assertions only.
 * Pick for completed graph fixtures; no matches/overflow are handled as blocked by the caller.
 */
export function validateResolution(data, citation) {
  ensure(data?.source_id === citation.source_id && data.document_id === citation.document_id
    && typeof data.ambiguous === "boolean" && typeof data.overflow === "boolean"
    && Array.isArray(data.matches) && data.matches.length <= 100, "graph_resolution_identity_invalid");
  ensure(data.matches.every((match) => {
    const reference = graphReference(match.record_id);
    return reference?.table === "occurrence" && match.table === reference.table && match.key === reference.key
      && typeof match.snapshot_key === "string" && match.snapshot_key.length > 0
      && /^[a-fA-F0-9]{64}$/.test(match.manifest_sha256)
      && (match.version_id === null || typeof match.version_id === "string");
  }), "graph_resolution_provenance_invalid");
  return data;
}

/** Choose only an edge actually connected to a returned graph node, retaining its exact target reference.
 * Inputs: bounded neighborhood and typed root. Output: edge/target or null; effects: none.
 * Pick for traversal fixtures; unrelated or malformed records cannot manufacture a successful graph path.
 */
export function connectedEdge(neighborhood, root) {
  ensure(graphReference(neighborhood?.root?.id)?.identity === root.identity
    && Array.isArray(neighborhood.edges) && neighborhood.edges.length <= 50, "graph_neighborhood_invalid");
  for (const edge of neighborhood.edges) {
    const incoming = graphReference(edge.in); const outgoing = graphReference(edge.out);
    const next = incoming?.identity === root.identity ? outgoing : outgoing?.identity === root.identity ? incoming : null;
    if (next && edge.id && typeof edge.relation === "string" && edge.relation) return { edge, next };
  }
  return null;
}

/** Locate one shared search row by the exact collection/object citation rendered in Source details.
 * Input: private hit. Output: DOM expression; effects: none until evaluated.
 * Pick instead of result position or excerpt matching, which could conflate equal text from distinct origins.
 */
function searchRow(hit) {
  const identity = `${hit.citation.collection} / ${hit.citation.object_id}`;
  return `[...document.querySelectorAll(${JSON.stringify(`${SEARCH_PANEL} > div[aria-live="polite"] > ul > li`)})].find(row => [...row.querySelectorAll('details dl dd')].some(dd => dd.previousElementSibling?.textContent === 'Search object' && dd.textContent === ${JSON.stringify(identity)}))`;
}

/** Open source details and compare the rendered coordinates against the exact returned citation.
 * Inputs: page and private hit. Output: none; effects: one disclosure click and DOM read.
 * Pick before following source links to verify provenance rather than only a reachable destination.
 */
async function showCitation(page, hit) {
  const row = searchRow(hit);
  await page.waitFor(row, "cited_search_result_unavailable");
  const details = `[...(${row}).querySelectorAll('details')].find(d => d.querySelector('summary')?.textContent === 'Source details')`;
  await page.clickElement(`(${details}).querySelector('summary')`);
  const values = [...Object.values(hit.citation.locator), ...hit.citation.source_version_ids];
  ensure(await page.eval(`(${details}).open && ${JSON.stringify(values)}.every(value => [...(${details}).querySelectorAll('dd')].some(dd => dd.textContent === value))`), "rendered_citation_coordinates_changed");
  await page.shot("workflow-search-source-details");
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
    send(method, params = {}, timeout = CDP_MS) {
      return new Promise((resolve, reject) => {
        const id = ++nextId;
        const timer = setTimeout(() => { pending.delete(id); reject(proofError("cdp_command_timeout")); }, timeout);
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
    this.readRequests = [];
    this.readSerial = 0;
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
        const path = new URL(event.request.url).pathname;
        if (event.request.method === "POST" && RETRIEVAL_POSTS.has(path)) {
          let payload;
          try {
            const raw = event.request.postData ?? "";
            if (Buffer.byteLength(raw, "utf8") <= MAX_POST_BYTES) payload = JSON.parse(raw);
          } catch { /* An unparseable request cannot satisfy a payload proof. */ }
          this.readRequests.push({ serial: ++this.readSerial, path, payload });
          this.readRequests = this.readRequests.slice(-64); // Private memory only; never copied into reports.
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
  async eval(expression, timeout = CDP_MS) {
    const result = await this.cdp.send("Runtime.evaluate", { expression, returnByValue: true, awaitPromise: true }, timeout);
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
  /** Await an observed read POST without disclosing its private payload.
   * Inputs: exact route, prior serial and predicate. Output: matched request; effects: bounded local polling.
   * Pick after a UI click so a previous rendered result cannot satisfy a new search proof.
   */
  async readRequest(path, before, predicate) {
    const deadline = Date.now() + READ_POST_MS;
    while (Date.now() < deadline) {
      const request = this.readRequests.find((item) => item.serial > before && item.path === path && predicate(item.payload));
      if (request) return request;
      await sleep(100);
    }
    throw proofError("shared_read_payload_mismatch");
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
  /** Read one exact same-origin POST fixture with bounded body, response and deadline.
   * Inputs: allowlisted path and private JSON. Output: private response; effects: one authenticated read POST.
   * Pick for retrieval/relationship discovery only; errors and bodies never enter redacted reports.
   */
  async readPost(path, payload) {
    const response = await this.eval(readPostExpression(readPostRequest(path, payload)), READ_POST_MS + 1000);
    ensure(response?.status === 200, "read_post_api_unavailable");
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

/** Discover indexed fixtures using at most four keyword reads without assuming raw messages have published chunks.
 * Input: authenticated page. Output: private query/results; effects: bounded existing context POSTs only.
 * Pick for navigation/graph evidence; a successful empty index remains valid and fixture-dependent checks block.
 */
async function indexedFixtures(page) {
  const fixtures = [];
  for (const query of ["document", "the", "call", "message"]) {
    const data = validateRetrieval(await page.readPost("/api/retrieval/search", {
      request_id: `workflow-${crypto.randomUUID()}`, query, mode: "keyword", legs: SEARCH_LEGS, limit: SEARCH_LIMIT,
    }), "keyword", SEARCH_LEGS);
    fixtures.push({ query, data });
    if (fixtures.some((item) => item.data.items.some((hit) => hit.location?.href?.startsWith("/sources?")))
      && fixtures.some((item) => item.data.items.some((hit) => hit.navigation?.sources?.some((source) => source.href.startsWith("/read?"))))) break;
  }
  return fixtures;
}

/** Open the current shared search workspace on Sources or Read without invoking processing controls.
 * Inputs: page and /sources or /read. Output: ready shared form; effects: same-origin navigation/read-control click.
 * Pick before mode/family assertions; file-name filtering remains a separate Sources control.
 */
async function openSearch(page, route) {
  ensure(["/sources", "/read"].includes(route), "search_route_invalid");
  await page.goto(route);
  if (route === "/sources") {
    const button = "[...document.querySelectorAll('[data-testid=\"sources-screen\"] button')].find(b => b.textContent.trim() === 'Search content and relationships')";
    await page.waitFor(button, "shared_search_toggle_unavailable");
    await page.clickElement(button);
  }
  await page.waitFor(`document.querySelector(${JSON.stringify(SEARCH_INPUT)})`, "shared_search_form_unavailable");
}

/** Submit shared controls and verify their actual read-only payload and settled result state.
 * Inputs: ready page, text, mode and selected families. Output: none; effects: bounded UI search POST.
 * Pick for every mode/family proof. Empty results pass; backend failures and hidden leg errors fail explicitly.
 */
async function submitSearch(page, text, mode, legs = SEARCH_LEGS) {
  await page.field(SEARCH_INPUT, text);
  await page.field(`${SEARCH_PANEL} form select`, mode);
  for (const [index, leg] of SEARCH_LEGS.entries()) {
    const checkbox = `document.querySelectorAll(${JSON.stringify(`${SEARCH_PANEL} form input[type="checkbox"]`)})[${index}]`;
    if (await page.eval(`Boolean((${checkbox})?.checked)`) !== legs.includes(leg)) await page.clickElement(checkbox);
  }
  const before = page.readSerial;
  await page.clickElement(`document.querySelector(${JSON.stringify(`${SEARCH_PANEL} button[type="submit"]`)})`);
  await page.readRequest("/api/retrieval/search", before, (payload) => payload?.query === text && payload.mode === mode
    && Array.isArray(payload.legs) && payload.legs.length === legs.length
    && [...payload.legs].sort().join(",") === [...legs].sort().join(","));
  await page.waitFor(`(() => { const panel = document.querySelector(${JSON.stringify(SEARCH_PANEL)}); return panel && !panel.querySelector('button[type="submit"]').disabled && (panel.querySelector('[role="alert"]') || panel.querySelector('div[aria-live="polite"]')); })()`, "shared_search_not_settled", READ_POST_MS);
  ensure(await page.eval(`(() => { const panel = document.querySelector(${JSON.stringify(SEARCH_PANEL)}); return !panel.querySelector('[role="alert"]') && !panel.querySelector('[role="status"]') && !panel.querySelector('button[type="submit"]').disabled; })()`), "shared_search_backend_failed");
  ensure(await page.eval(`document.querySelector(${JSON.stringify(`${SEARCH_PANEL} div[aria-live="polite"] > p`)})?.textContent.includes(${JSON.stringify(`ranked results for “${text}”`)})`), "shared_search_result_query_mismatch");
}

/** Find a completed resolver fixture and optionally an actual connected edge within fixed read limits.
 * Inputs: page and indexed fixtures. Output: private source/snapshot/edge fixture or blocked error.
 * Effects: at most five resolver POSTs and six neighborhood GETs; no graph creation or guessed identifiers.
 */
async function completedGraphFixture(page, fixtures) {
  const candidates = fixtures.flatMap((fixture) => fixture.data.items.map((hit) => ({ query: fixture.query, hit })))
    .filter(({ hit }) => hit.citation.source_id && hit.citation.document_id);
  const seen = new Set(); let attempts = 0; let reads = 0; let completed;
  for (const candidate of candidates) {
    const citation = candidate.hit.citation;
    const identity = JSON.stringify([citation.source_id, citation.document_id]);
    if (seen.has(identity)) continue;
    if (++attempts > 5) break;
    seen.add(identity);
    const resolution = validateResolution(await page.readPost("/api/retrieval/relationships", {
      source_id: citation.source_id, document_id: citation.document_id,
    }), citation);
    if (resolution.overflow) continue;
    for (const match of resolution.matches.slice(0, 2)) {
      const fixture = { ...candidate, resolution, match, traversal: null };
      completed ??= fixture;
      if (reads >= 6) continue;
      reads += 1;
      const root = graphReference(match.record_id);
      const neighbors = await page.api(`/api/intake/discovery/neighbors/${encodeURIComponent(root.table)}/${encodeURIComponent(root.key)}?limit=50`);
      const traversal = connectedEdge(neighbors, root);
      if (traversal) return { ...fixture, traversal };
    }
  }
  if (completed) return completed;
  throw proofError("existing_completed_graph_fixture_unavailable", true);
}

/** Open a genuine result's completed graph snapshot through the shared UI.
 * Inputs: page and discovered fixture. Output: settled selected neighborhood; effects: search/resolver/graph reads.
 * Pick for both graph checks; multiple snapshots require an explicit returned record choice.
 */
async function openGraph(page, fixture) {
  await openSearch(page, "/sources");
  await submitSearch(page, fixture.query, "keyword");
  const row = searchRow(fixture.hit);
  await page.waitFor(row, "graph_search_result_unavailable");
  const before = page.readSerial;
  await page.clickElement(`[...(${row}).querySelectorAll('button')].find(b => b.textContent.trim() === 'Explore file relationships')`);
  await page.readRequest("/api/retrieval/relationships", before, (payload) => payload?.source_id === fixture.hit.citation.source_id
    && payload.document_id === fixture.hit.citation.document_id);
  await page.waitFor(`(${row})?.querySelector('select') || [...((${row})?.querySelectorAll('details summary') ?? [])].some(s => s.textContent === 'Record reference') || (${row})?.querySelector('[role="alert"]')`, "graph_resolution_not_settled");
  ensure(await page.eval(`!(${row}).querySelector('[role="alert"]')`), "graph_resolution_failed");
  if (fixture.resolution.matches.length > 1) {
    ensure(await page.eval(`(() => { const select = (${row}).querySelector('select'); if (!select || ![...select.options].some(o => o.value === ${JSON.stringify(fixture.match.record_id)})) return false; Object.getOwnPropertyDescriptor(HTMLSelectElement.prototype, 'value').set.call(select, ${JSON.stringify(fixture.match.record_id)}); select.dispatchEvent(new Event('change', { bubbles: true })); return true; })()`), "graph_snapshot_choice_unavailable");
  }
  await waitGraph(page, row, fixture.match.record_id);
  const details = `[...(${row}).querySelectorAll('details')].find(d => d.querySelector('summary')?.textContent === 'Record reference')`;
  await page.clickElement(`(${details}).querySelector('summary')`);
}

/** Await the exact selected graph record and a successful settled neighborhood.
 * Inputs: page, result expression and returned ID. Output: none; effects: bounded DOM reads.
 * Pick after snapshot/edge/back navigation; an immediate local trail change alone is insufficient proof.
 */
async function waitGraph(page, row, recordId) {
  await page.waitFor(`(() => { const row = (${row}); const details = [...(row?.querySelectorAll('details') ?? [])].find(d => d.querySelector('summary')?.textContent === 'Record reference'); return details?.querySelector('p')?.textContent === ${JSON.stringify(recordId)} && !row.querySelector('[role="status"]') && !row.querySelector('[role="alert"]'); })()`, "graph_neighborhood_not_settled");
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
  let indexedPromise;
  let graphPromise;
  /** Cache bounded discovery reads across independent checks, including failures.
   * Input: active page. Output: private fixture promise; effects: discovery reads once per run.
   * Pick for dependent navigation checks to keep their combined discovery budget fixed.
   */
  const getIndexed = () => indexedPromise ??= indexedFixtures(page);
  /** Cache the genuine completed graph fixture without replacing absent data with a synthetic node.
   * Input: indexed fixture promise. Output: graph fixture promise; effects: bounded graph reads once per run.
   * Pick for resolution and edge checks so missing data remains blocked consistently.
   */
  const getGraph = () => graphPromise ??= getIndexed().then((fixtures) => completedGraphFixture(page, fixtures));
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
      ["read", "/read", `document.querySelector(${JSON.stringify(SEARCH_INPUT)})`],
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
      const indexed = await getIndexed();
      await openSearch(page, "/sources");
      await submitSearch(page, indexed[0].query, "keyword");
      await page.clickElement("[...document.querySelectorAll('[data-testid=\"sources-screen\"] button')].find(b => b.textContent.trim() === 'Back to files')");
      await page.waitFor("document.querySelector('form[aria-label=\"Filter file names and paths\"] #source-name-filter') && !document.querySelector('#combined-search-query')", "source_file_filter_not_restored");
    });
    await check("read-source-thread", async () => {
      await page.goto("/read");
      await page.waitFor(`document.querySelector(${JSON.stringify(SEARCH_INPUT)})`, "read_page_unavailable");
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
    for (const mode of SEARCH_MODES) {
      await check(`read-content-search-${mode}`, async () => {
        const query = "document";
        validateRetrieval(await page.readPost("/api/retrieval/search", {
          request_id: `workflow-${crypto.randomUUID()}`, query, mode, legs: SEARCH_LEGS, limit: SEARCH_LIMIT,
        }), mode, SEARCH_LEGS);
        await openSearch(page, "/read");
        await submitSearch(page, query, mode);
        ensure(await page.eval(`new URLSearchParams(location.search).get('q') === ${JSON.stringify(query)}`), "search_query_not_preserved");
      });
    }
    for (const leg of SEARCH_LEGS) {
      await check(`read-search-${leg}-only`, async () => {
        validateRetrieval(await page.readPost("/api/retrieval/search", {
          request_id: `workflow-${crypto.randomUUID()}`, query: "document", mode: "keyword", legs: [leg], limit: SEARCH_LIMIT,
        }), "keyword", [leg]);
        await openSearch(page, "/read");
        await submitSearch(page, "document", "keyword", [leg]);
      });
    }
    await check("sources-search-original-link", async () => {
      const candidate = (await getIndexed()).flatMap(({ query, data }) => data.items.map((hit) => ({ query, hit })))
        .find(({ hit }) => hit.legs.includes("intake") && hit.location?.href?.startsWith("/sources?"));
      if (!candidate) throw proofError("existing_locatable_index_fixture_unavailable", true);
      await openSearch(page, "/sources");
      await submitSearch(page, candidate.query, "keyword");
      const row = searchRow(candidate.hit);
      await showCitation(page, candidate.hit);
      const link = `[...(${row}).querySelectorAll('a')].find(a => a.textContent.trim() === 'Locate original file')`;
      const href = await page.eval(`(${link})?.getAttribute('href')`);
      ensure(href === candidate.hit.location.href, "original_file_locator_changed");
      const expected = new URL(href, BASE).searchParams;
      ensure(expected.get("root") && expected.get("file"), "original_file_locator_incomplete");
      await page.clickElement(link);
      await page.waitFor(`new URLSearchParams(location.search).get('root') === ${JSON.stringify(expected.get("root"))} && new URLSearchParams(location.search).get('file') === ${JSON.stringify(expected.get("file"))} && document.querySelector('#source-name-filter')?.value === ${JSON.stringify(expected.get("file"))}`, "original_file_not_located");
    });
    // Verify compact excerpts expand to the unchanged indexed passage and close again.
    // Input: existing cited hit; output: browser proof; effects: read-only search and disclosure clicks.
    await check("read-search-passage-expansion", async () => {
      const candidate = (await getIndexed()).flatMap(({ query, data }) => data.items.map((hit) => ({ query, hit })))
        .find(({ hit }) => hit.text.length > 400 || hit.text.split("\n").length > 5);
      if (!candidate) throw proofError("existing_long_passage_fixture_unavailable", true);
      await openSearch(page, "/read");
      await submitSearch(page, candidate.query, "keyword");
      const row = searchRow(candidate.hit);
      await page.waitFor(row, "cited_search_result_unavailable");
      ensure(await page.eval(`(() => { const excerpt = (${row}).querySelector('p.line-clamp-5'); return excerpt && excerpt.clientHeight <= parseFloat(getComputedStyle(excerpt).lineHeight) * 5 + 2; })()`), "search_excerpt_not_bounded");
      const details = `[...(${row}).querySelectorAll('details')].find(d => d.querySelector('summary')?.textContent === 'Read indexed passage')`;
      await page.clickElement(`(${details}).querySelector('summary')`);
      ensure(await page.eval(`(${details}).open && (${details}).querySelector('p')?.textContent === ${JSON.stringify(candidate.hit.text)}`), "expanded_passage_changed_or_unavailable");
      await page.clickElement(`(${details}).querySelector('summary')`);
      ensure(await page.eval(`!(${details}).open`), "indexed_passage_did_not_close");
    });
    await check("read-search-canonical-source-link", async () => {
      const candidate = (await getIndexed()).flatMap(({ query, data }) => data.items.map((hit) => ({ query, hit })))
        .find(({ hit }) => hit.legs.includes("proffer") && hit.navigation?.sources?.some((source) => source.href.startsWith("/read?") && new URL(source.href, BASE).searchParams.has("thread")));
      if (!candidate) throw proofError("existing_canonical_thread_fixture_unavailable", true);
      const source = candidate.hit.navigation.sources.find((item) => item.href.startsWith("/read?") && new URL(item.href, BASE).searchParams.has("thread"));
      const expected = new URL(source.href, BASE).searchParams;
      ensure(!expected.has("around"), "canonical_thread_has_unexpected_message_locator");
      await openSearch(page, "/read");
      await submitSearch(page, candidate.query, "keyword");
      const row = searchRow(candidate.hit);
      await showCitation(page, candidate.hit);
      const link = `[...(${row}).querySelectorAll('a')].find(a => new URL(a.href).pathname === '/read' && new URL(a.href).searchParams.get('thread') === ${JSON.stringify(expected.get("thread"))})`;
      ensure(await page.eval(`(() => { const a = (${link}); if (!a) return false; const url = new URL(a.href); return !url.searchParams.has('around') && url.searchParams.get('q') === ${JSON.stringify(candidate.query)} && [...new URLSearchParams(${JSON.stringify(expected.toString())})].every(([key, value]) => key === 'q' || url.searchParams.get(key) === value); })()`), "canonical_thread_link_context_changed");
      await page.clickElement(link);
      await page.waitFor(`new URLSearchParams(location.search).get('thread') === ${JSON.stringify(expected.get("thread"))} && new URLSearchParams(location.search).get('q') === ${JSON.stringify(candidate.query)} && !new URLSearchParams(location.search).has('around') && document.querySelector('[aria-label="Messages"] [data-testid="message-bubble"]')`, "canonical_thread_not_opened");
    });
    await check("search-graph-resolution", async () => {
      await openGraph(page, await getGraph());
    });
    await check("search-graph-edge-and-back", async () => {
      const graph = await getGraph();
      if (!graph.traversal) throw proofError("existing_completed_graph_edges_unavailable", true);
      await openGraph(page, graph);
      const { edge, next } = graph.traversal;
      const target = await page.api(`/api/intake/discovery/neighbors/${encodeURIComponent(next.table)}/${encodeURIComponent(next.key)}?limit=50`);
      connectedEdge(target, next); // Validate target root even when it has no further connected edges.
      const row = searchRow(graph.hit);
      const label = `${edge.relation.replaceAll("_", " ")} → ${next.table.replaceAll("_", " ")}`;
      const button = `[...(${row}).querySelectorAll('button')].find(b => b.textContent.trim() === ${JSON.stringify(label)})`;
      await page.clickElement(button);
      await waitGraph(page, row, next.identity);
      await page.shot("workflow-graph-related-record");
      await page.clickElement(`[...(${row}).querySelectorAll('button')].find(b => b.textContent.trim() === 'Back')`);
      await waitGraph(page, row, graph.match.record_id);
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
      await page.waitFor(`document.querySelector(${JSON.stringify(SEARCH_INPUT)}) && (() => { const expected = new URLSearchParams(${JSON.stringify(context.toString())}); const actual = new URLSearchParams(location.search); return [...expected].every(([key, value]) => actual.get(key) === value) && !actual.has('view') && !actual.has('resource') && !actual.has('preview_handle') && !actual.has('attempt'); })()`, "imported_reading_context_not_restored");
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
  for (const path of RETRIEVAL_POSTS) {
    assert.equal(allowedRequest(`${BASE}${path}`, "POST"), true);
    for (const method of ["PUT", "PATCH", "DELETE"]) assert.equal(allowedRequest(`${BASE}${path}`, method), false);
    for (const suffix of ["/", "/start", "/decision"]) assert.equal(allowedRequest(`${BASE}${path}${suffix}`, "POST"), false);
    assert.equal(allowedRequest(`https://other.invalid${path}`, "POST"), false);
    assert.equal(allowedRequest(`https://user:password@${new URL(BASE).host}${path}`, "POST"), false);
    assert.deepEqual(readPostRequest(path, { query: "synthetic" }), { path, body: '{"query":"synthetic"}' });
  }
  assert.equal(allowedRequest("invalid", "POST"), false);
  for (const path of ["https://other.invalid/api/retrieval/search", "//other.invalid/api/retrieval/search", "/api/retrieval/search?write=true", "/api/proffer/decoded/exists", "/api/retrieval/relationships/start"]) {
    assert.throws(() => readPostRequest(path, {}), /read_post_path_not_allowlisted/);
  }
  for (const payload of [null, [], "text"]) assert.throws(() => readPostRequest("/api/retrieval/search", payload), /read_post_object_required/);
  assert.throws(() => readPostRequest("/api/retrieval/search", { query: "x".repeat(MAX_POST_BYTES) }), /read_post_request_too_large/);
  const empty = (mode, legs = SEARCH_LEGS) => ({ mode, status: "success", items: [], legs: legs.map((name) => ({ name, status: "success", count: 0 })) });
  for (const mode of SEARCH_MODES) {
    for (const legs of [SEARCH_LEGS, ["intake"], ["proffer"]]) assert.deepEqual(validateRetrieval(empty(mode, legs), mode, legs), empty(mode, legs));
  }
  const hit = {
    text: "Identical synthetic passage", score: 0.5, legs: ["intake"],
    citation: { collection: "Synthetic", object_id: "object-a", source_version_ids: ["version-a"],
      source_id: "source-a", document_id: "document-a", locator: { source_path: "synthetic.txt" } },
  };
  const cited = { ...empty("vector"), items: [hit, { ...hit, citation: { ...hit.citation, object_id: "object-b", source_version_ids: ["version-b"] } }] };
  assert.equal(validateRetrieval(cited, "vector", SEARCH_LEGS).items.length, 2);
  const partial = { ...empty("keyword"), status: "partial", legs: [{ name: "intake", status: "success", count: 0 }, { name: "proffer", status: "failed", count: 0, error: "timeout" }] };
  assert.throws(() => validateRetrieval(partial, "keyword", SEARCH_LEGS), /retrieval_leg_failed/);
  assert.throws(() => validateRetrieval({ ...cited, items: [{ ...hit, score: null }] }, "vector", SEARCH_LEGS), /retrieval_citation_invalid/);
  assert.throws(() => validateRetrieval({ ...cited, items: [{ ...hit, citation: { ...hit.citation, locator: {} } }] }, "vector", SEARCH_LEGS), /retrieval_citation_invalid/);
  assert.throws(() => validateRetrieval(empty("hybrid"), "keyword", SEARCH_LEGS), /retrieval_response_invalid/);
  assert.throws(() => validateRetrieval({ ...empty("keyword"), legs: [{ name: "intake", status: "success", count: 0 }, { name: "intake", status: "success", count: 0 }] }, "keyword", SEARCH_LEGS), /retrieval_legs_invalid/);
  const spec = readPostRequest("/api/retrieval/search", { query: "synthetic" });
  const execute = new Function("fetch", "location", `return ${readPostExpression(spec)}`);
  let calls = 0;
  assert.deepEqual(await execute(async (url, options) => {
    calls += 1;
    assert.equal(url, "https://fixture.invalid/api/retrieval/search");
    assert.equal(options.method, "POST");
    assert.equal(options.redirect, "error");
    assert.equal(options.body, spec.body);
    assert.equal(options.headers["Content-Type"], "application/json");
    assert.ok(options.signal instanceof AbortSignal);
    return Response.json({ synthetic: true });
  }, { origin: "https://fixture.invalid" }), { status: 200, body: { synthetic: true } });
  assert.equal(calls, 1);
  assert.deepEqual(await execute(async () => ({ status: 503, get body() { assert.fail("Error bodies must remain unread"); } }), { origin: "https://fixture.invalid" }), { status: 503, body: null });
  await assert.rejects(execute(async () => new Response("x".repeat(MAX_READ_BYTES + 1)), { origin: "https://fixture.invalid" }), /read_post_response_bound/);
  await assert.rejects(execute(async () => new Response("not-json"), { origin: "https://fixture.invalid" }), SyntaxError);
  await assert.rejects(execute(async () => new Response(null), { origin: "https://fixture.invalid" }), /read_post_body/);
  await assert.rejects(execute(async () => { throw new Error("synthetic_timeout"); }, { origin: "https://fixture.invalid" }), /synthetic_timeout/);
  const foreign = new Function("fetch", "location", `return ${readPostExpression({ path: "https://other.invalid/api/retrieval/search", body: "{}" })}`);
  await assert.rejects(foreign(async () => assert.fail("Cross-origin fetch must not run"), { origin: "https://fixture.invalid" }), /read_post_origin/);
  const digest = "1".repeat(64);
  assert.deepEqual(graphReference(`occurrence:⟨${digest}⟩`), { table: "occurrence", key: digest, identity: `occurrence:${digest}` });
  for (const id of [null, "untyped", "occurrence:../../other", "occurrence:⟨short⟩", "https://fixture.invalid"]) assert.equal(graphReference(id), null);
  const root = graphReference("occurrence:synthetic-a");
  const match = { table: root.table, key: root.key, record_id: root.identity, snapshot_key: "synthetic-completed", manifest_sha256: digest, version_id: null };
  const resolution = { source_id: hit.citation.source_id, document_id: hit.citation.document_id, ambiguous: false, overflow: false, matches: [match] };
  assert.equal(validateResolution(resolution, hit.citation), resolution);
  assert.equal(validateResolution({ ...resolution, matches: [] }, hit.citation).matches.length, 0);
  assert.throws(() => validateResolution({ ...resolution, source_id: "another-source" }, hit.citation), /graph_resolution_identity_invalid/);
  assert.throws(() => validateResolution({ ...resolution, matches: [{ ...match, key: "invented" }] }, hit.citation), /graph_resolution_provenance_invalid/);
  assert.throws(() => validateResolution({ ...resolution, matches: [{ ...match, manifest_sha256: "missing" }] }, hit.citation), /graph_resolution_provenance_invalid/);
  assert.equal(connectedEdge({ root: { id: root.identity }, edges: [] }, root), null);
  const edge = { id: "related:synthetic-edge", in: root.identity, out: "document:synthetic-b", relation: "has_document" };
  assert.deepEqual(connectedEdge({ root: { id: root.identity }, edges: [edge] }, root), { edge, next: graphReference(edge.out) });
  assert.equal(connectedEdge({ root: { id: root.identity }, edges: [{ ...edge, in: "occurrence:unrelated" }] }, root), null);
  assert.throws(() => connectedEdge({ root: { id: "occurrence:unrelated" }, edges: [] }, root), /graph_neighborhood_invalid/);
  let discoveryReads = 0;
  const noHits = await indexedFixtures({ readPost: async (path, payload) => {
    discoveryReads += 1;
    assert.equal(path, "/api/retrieval/search");
    assert.equal(payload.limit, SEARCH_LIMIT);
    return empty(payload.mode, payload.legs);
  } });
  assert.equal(discoveryReads, 4);
  assert.equal(noHits.length, 4);
  await assert.rejects(completedGraphFixture({ readPost: async () => assert.fail("No coordinates means no resolver call") }, noHits), (error) => error.blocked && error.proofCode === "existing_completed_graph_fixture_unavailable");
  const fixturePage = {
    readPost: async (path, payload) => {
      assert.equal(path, "/api/retrieval/relationships");
      assert.deepEqual(payload, { source_id: hit.citation.source_id, document_id: hit.citation.document_id });
      return resolution;
    },
    api: async (path) => {
      assert.equal(path, "/api/intake/discovery/neighbors/occurrence/synthetic-a?limit=50");
      return { root: { id: root.identity }, edges: [] };
    },
  };
  const fixtureResults = [{ query: "synthetic", data: { items: [hit] } }];
  assert.equal((await completedGraphFixture(fixturePage, fixtureResults)).traversal, null);
  assert.deepEqual((await completedGraphFixture({ ...fixturePage, api: async () => ({ root: { id: root.identity }, edges: [edge] }) }, fixtureResults)).traversal.next, graphReference(edge.out));
  await assert.rejects(completedGraphFixture({ ...fixturePage, readPost: async () => ({ ...resolution, matches: [] }) }, fixtureResults), (error) => error.blocked && error.proofCode === "existing_completed_graph_fixture_unavailable");
  let resolves = 0; let neighborhoods = 0;
  const boundedResults = [{ query: "synthetic", data: { items: Array.from({ length: 8 }, (_, index) => ({ ...hit,
    citation: { ...hit.citation, document_id: `synthetic-document-${index}` } })) } }];
  const boundedGraph = await completedGraphFixture({
    readPost: async (_path, payload) => {
      resolves += 1;
      return { ...resolution, document_id: payload.document_id, ambiguous: true,
        matches: [match, { ...match, snapshot_key: "synthetic-other", record_id: "occurrence:synthetic-other", key: "synthetic-other" }] };
    },
    api: async (path) => {
      neighborhoods += 1;
      const key = path.split("?")[0].split("/").at(-1);
      return { root: { id: `occurrence:${key}` }, edges: [] };
    },
  }, boundedResults);
  assert.equal(resolves, 5);
  assert.equal(neighborhoods, 6);
  assert.equal(boundedGraph.traversal, null);
  assert.deepEqual(outcomeCounts([{ status: "pass" }, { status: "fail" }, { status: "blocked" }]), { total: 3, pass: 1, fail: 1, blocked: 1 });
  console.log(JSON.stringify({ self_test: "pass", browser_launched: false, credential_reads: 0 }));
} else {
  await run(process.argv[2]).catch((error) => {
    console.error(JSON.stringify({ status: "fail", code: error.proofCode || "unexpected_error" }));
    process.exitCode = 1;
  });
}

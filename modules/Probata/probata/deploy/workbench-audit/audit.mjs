// Byline: Claude Code · Opus 5.5 · 2026-09-27
// The Workbench six-step live audit (PR-24 / PR-27), driven by headless Chrome INSIDE the Probata
// devbox on ovh-files (never on the owner's desktop; owner rule 2026-09-24). Launched by audit.sh.
// No dependencies: Node 22's fetch + WebSocket drive Chrome over the DevTools protocol, the same
// way deploy/portal/shoot.mjs does.
//
//   node audit.mjs <out-dir>
//
// Environment:
//   WORKBENCH_URL        default https://workbench.tilapia-skilift.ts.net
//   AUDIT_TARGET_NAME    a file in the default Sources root that sits past the first 200 rows
//   AUDIT_RESOLVER_RULE  optional Chrome --host-resolver-rules value (see audit.sh)
//   AUDIT_RUN_WAIT_MS    how long to wait for the TEST run to reach a reviewable state
//
// Steps (docs/reviews/2026-09-23-probata-p0-function-checkpoint.md:48):
//   1 browse more than 200 files   2 pick a later-page file   3 inspect and hash it
//   4 run one TEST run             5 check the preview and gap controls
//   6 read the receipts back
// Plus page shots for the deploy check: Desk, Knowledge (no Graphiti pane) and Review.
// Every page records its console errors. Writes <out>/<name>.png and <out>/audit.json.
import { spawn } from "node:child_process";
import { existsSync, mkdirSync, mkdtempSync, readFileSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { setTimeout as sleep } from "node:timers/promises";

const outDir = process.argv[2];
if (!outDir) {
  console.error("usage: node audit.mjs <out-dir>");
  process.exit(2);
}
mkdirSync(outDir, { recursive: true });
const BASE = (process.env.WORKBENCH_URL || "https://workbench.tilapia-skilift.ts.net").replace(/\/$/, "");
const TARGET = process.env.AUDIT_TARGET_NAME || "calls-20250703043408.xml";
const RUN_WAIT_MS = Number(process.env.AUDIT_RUN_WAIT_MS || 600000);
const chromeBin = process.env.CHROME_BIN || "/opt/google/chrome/chrome";
const ROW_HEIGHT = 30; // source-rows-grid.tsx rowHeight
const HEADER_HEIGHT = 32; // source-rows-grid.tsx headerHeight
const PAGE_SIZE = 200; // sources-screen.tsx pageSize

async function launchChrome() {
  const profile = mkdtempSync(join(tmpdir(), "workbench-audit-"));
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

const buttonByText = (text) =>
  `[...document.querySelectorAll("button")].find((b) => b.textContent.trim() === ${JSON.stringify(text)} && !b.disabled)`;
const statusText = `(document.querySelector('[data-testid="sources-screen"] [role="status"]')?.textContent || "")`;

async function listingIndexOf(page, name) {
  // The UI lists the default root with page_size 200 and filter_scope root; read the same pages.
  let token = null;
  let index = 0;
  const seen = new Set();
  let duplicates = 0;
  for (let n = 0; n < 50; n += 1) {
    const query = new URLSearchParams({ mode: "TEST", filter_scope: "root", page_size: String(PAGE_SIZE) });
    if (token) query.set("continuation_token", token);
    const { status, body } = await page.api(`/api/proffer/sources?${query}`);
    if (status !== 200) throw new Error(`listing answered ${status}`);
    for (const object of body.objects) {
      if (seen.has(object.source_ref)) duplicates += 1;
      seen.add(object.source_ref);
      if (object.name === name) return { index, object, duplicates, activeRootId: body.active_root_id };
      index += 1;
    }
    if (!body.is_truncated) break;
    token = body.continuation_token;
  }
  throw new Error(`${name} is not in the default root listing`);
}

const report = { base: BASE, target: TARGET, started_at: new Date().toISOString(), steps: [], pages: [] };
const record = (step, pass, detail) => {
  report.steps.push({ step, pass, ...detail });
  console.log(`step ${step}: ${pass ? "PASS" : "FAIL"} ${JSON.stringify(detail)}`);
};

const { proc, port } = await launchChrome();
let exitCode = 0;
const page = new Page(port);
try {
  await page.open();

  // --- deploy check: the pages load, with their console errors ------------------------------
  for (const [name, path, ready] of [
    ["desk", "/?mode=TEST", `document.body.innerText.includes("Context Intake Desk")`],
    ["knowledge", "/knowledge?mode=TEST", `document.querySelector("#knowledge-search-tab")`],
  ]) {
    await page.goto(path);
    let loaded = true;
    try {
      await page.waitFor(ready, 30000, name);
      await sleep(3000);
    } catch {
      loaded = false;
    }
    const facts = await page.eval(`({
      graphMemoryTab: Boolean(document.querySelector("#knowledge-memory-tab")),
      graphitiText: /graphiti/i.test(document.body.innerText),
      title: document.title,
    })`);
    await page.shot(`page-${name}`);
    report.pages.push({ name, path, loaded, ...facts, consoleErrors: [...page.consoleErrors] });
  }

  // --- 1. browse more than 200 files -------------------------------------------------------
  await page.goto("/sources?mode=TEST");
  await page.waitFor(`document.querySelector('[data-testid="source-rows-grid"] canvas')`, 60000, "the Sources grid");
  const located = await listingIndexOf(page, TARGET);
  let status = await page.waitFor(`${statusText}.includes("files")  && ${statusText}`, 60000, "the Sources status line");
  for (let n = 0; n < 20; n += 1) {
    const loadedRows = Number((status.match(/(\d+) files/) || [])[1] || 0);
    if (loadedRows > located.index) break;
    await page.eval(`${buttonByText("Load more")}?.click()`);
    await page.waitFor(`!${statusText}.includes("Loading more") && ${statusText}`, 60000, "the next page");
    await sleep(500);
    status = await page.eval(statusText);
  }
  const loadedRows = Number((status.match(/(\d+) files/) || [])[1] || 0);
  await page.shot("step1-sources-loaded");
  record(1, loadedRows > 200, { loaded_rows: loadedRows, status, listing_duplicates: located.duplicates, console_errors: [...page.consoleErrors] });

  // --- 2. pick a later-page file ------------------------------------------------------------
  const scrolled = await page.eval(`(() => {
    const grid = document.querySelector('[data-testid="source-rows-grid"]');
    const scroller = grid.querySelector(".dvn-scroller");
    scroller.scrollTop = ${located.index * ROW_HEIGHT};
    return scroller.scrollTop;
  })()`);
  await sleep(1500);
  const box = await page.eval(`(() => {
    const b = document.querySelector('[data-testid="source-rows-grid"]').getBoundingClientRect();
    return { x: b.x, y: b.y, w: b.width, h: b.height };
  })()`);
  const rowTop = HEADER_HEIGHT + located.index * ROW_HEIGHT - scrolled;
  await page.click(box.x + 160, box.y + rowTop + ROW_HEIGHT / 2);
  const selectedName = await page
    .waitFor(`document.body.innerText.includes(${JSON.stringify(`Ready: ${TARGET}`)})`, 20000, "the selection")
    .then(() => TARGET)
    .catch(() => null);
  await page.shot("step2-later-file-selected");
  record(2, selectedName === TARGET && located.index >= PAGE_SIZE, {
    target: TARGET,
    listing_index: located.index,
    page: Math.floor(located.index / PAGE_SIZE) + 1,
    selected: selectedName,
  });

  // --- 3. inspect and hash it ---------------------------------------------------------------
  const hash = await page
    .waitFor(`(document.body.innerText.match(/Hash \\(sha256\\)\\s*([0-9a-f]{64})/) || [])[1]`, 120000, "the sha256")
    .catch(() => null);
  const inspection = await page.api(`/api/proffer/source-inspection?mode=TEST&root_id=${encodeURIComponent(located.activeRootId)}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      key: located.object.key,
      source_ref: located.object.source_ref,
      root_id: located.activeRootId,
      expected_byte_length: located.object.byte_length,
      expected_etag: located.object.etag ?? null,
    }),
  });
  await page.shot("step3-inspected-hashed");
  record(3, Boolean(hash) && inspection.body?.sha256 === hash, {
    panel_sha256: hash,
    api_sha256: inspection.body?.sha256 ?? null,
    api_status: inspection.status,
    byte_length: located.object.byte_length,
  });

  // --- 4. run one TEST run ------------------------------------------------------------------
  // Two minutes of slack for clock skew between the devbox and the engine.
  const startedAt = new Date(Date.now() - 120000).toISOString();
  await page.eval(`${buttonByText("Process")}?.click()`);
  const started = await page
    .waitFor(`/Started 1 run/.test(document.body.innerText) || (document.querySelector('[role="alert"]')?.textContent || "").trim()`, 60000, "the Process result")
    .catch((error) => error.message);
  await page.shot("step4-process-started");
  let run = null;
  const runDeadline = Date.now() + RUN_WAIT_MS;
  while (Date.now() < runDeadline) {
    const { body } = await page.api("/api/proffer/proposal-resources?mode=TEST&limit=25");
    run = (body?.items || []).find((item) => item.source_ref === located.object.source_ref && item.created_at >= startedAt.slice(0, 19)) || run;
    if (run?.preview_handle && (run.terminal || run.wait || run.record_preview_available)) break;
    await sleep(10000);
  }
  record(4, started === true && Boolean(run?.preview_handle), {
    process_result: started,
    preview_handle: run?.preview_handle ?? null,
    lifecycle: run?.lifecycle ?? null,
    wait: run?.wait ?? null,
    completed_stage_count: run?.completed_stage_count ?? null,
    reason: run?.reason ?? null,
  });

  // --- 5. check the preview and gap controls ------------------------------------------------
  if (run?.preview_handle) {
    await page.goto(`/review?mode=TEST&preview_handle=${encodeURIComponent(run.preview_handle)}`);
    await page.waitFor(`document.querySelector('[data-testid="review-mode-flag"]')`, 60000, "the Review page").catch(() => null);
    await sleep(8000);
    const review = await page.eval(`(() => {
      const has = (id) => Boolean(document.querySelector('[data-testid="' + id + '"]'));
      return {
        file_metadata: has("review-file-metadata"),
        message_browser: has("message-browser"),
        message_grid: has("message-browser-grid"),
        calls_table: has("calls-table") || has("calls-table-empty"),
        missing_payload_flag: has("message-browser-missing-payloads"),
        actions_panel: has("review-actions-panel"),
        repair_control: has("review-actions-repair") || has("repair-builder"),
        context_review: has("context-review-panel") || has("review-actions-context"),
        mark_event_buttons: [...document.querySelectorAll("button")].filter((b) => /Mark as event/.test(b.textContent)).length,
        text: document.body.innerText.slice(0, 1500),
      };
    })()`);
    await page.shot("step5-review");
    const previewReadable = review.message_browser || review.calls_table;
    const gapControls = review.actions_panel && (review.repair_control || review.context_review);
    record(5, Boolean(previewReadable && gapControls), { ...review, console_errors: [...page.consoleErrors] });

    // --- 6. read the receipts back ----------------------------------------------------------
    const handle = encodeURIComponent(run.preview_handle);
    const preview = await page.api(`/api/proffer/previews/${handle}?mode=TEST`);
    const operation = await page.api(`/api/proffer/operations/${handle}?mode=TEST`);
    const receipts = preview.body?.receipts || [];
    record(6, preview.status === 200 && operation.status === 200 && receipts.length > 0, {
      preview_status: preview.status,
      operation_status: operation.status,
      receipt_types: receipts.map((receipt) => receipt.receipt_type),
      preview_handle_echo: preview.body?.preview_handle === run.preview_handle,
      operation_lifecycle: operation.body?.lifecycle ?? null,
      preview_detail: preview.status === 200 ? null : preview.body,
    });
  } else {
    record(5, false, { skipped: "no preview handle from step 4" });
    record(6, false, { skipped: "no preview handle from step 4" });
  }
  await page.close();
} catch (error) {
  console.error(`audit aborted: ${error.message}`);
  report.aborted = error.message;
  report.aborted_console_errors = [...page.consoleErrors];
  await page.shot("aborted").catch(() => {});
  exitCode = 1;
} finally {
  proc.kill();
}
report.finished_at = new Date().toISOString();
writeFileSync(join(outDir, "audit.json"), JSON.stringify(report, null, 1));
if (report.steps.some((step) => !step.pass)) exitCode = 1;
process.exit(exitCode);

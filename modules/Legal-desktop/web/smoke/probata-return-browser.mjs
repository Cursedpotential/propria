// Run only inside the remote devbox. Uses its existing Chrome and Node; no installs.
// Synthetic API responses are intercepted before they can reach production.
import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { existsSync, mkdirSync, mkdtempSync, readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { setTimeout as sleep } from "node:timers/promises";

assert.equal(process.env.REMOTE_BROWSER_SMOKE, "yes", "Run in the remote devbox only");
const base = "https://legal.tilapia-skilift.ts.net";
const out = process.argv[2];
assert(out, "Supply a retained output directory");
mkdirSync(join(out, "to_be_deleted"), { recursive: true });
const profile = mkdtempSync(join(out, "to_be_deleted", "chrome-"));
const chrome = spawn("/opt/google/chrome/chrome", ["--headless=new", "--remote-debugging-port=0",
  `--user-data-dir=${profile}`, "--no-first-run", "--no-default-browser-check", "--disable-gpu", "about:blank"],
  { stdio: ["ignore", "ignore", "ignore"] });
let ws;
try {
  const portFile = join(profile, "DevToolsActivePort");
  for (let i = 0; i < 200 && !existsSync(portFile); i++) await sleep(100);
  assert(existsSync(portFile), "Remote Chrome did not start");
  const port = readFileSync(portFile, "utf8").split("\n")[0];
  const target = await (await fetch(`http://127.0.0.1:${port}/json/new?about:blank`, { method: "PUT" })).json();
  ws = new WebSocket(target.webSocketDebuggerUrl);
  let nextId = 0;
  const pending = new Map(), handlers = new Map(), errors = [], writes = [];
  const send = (method, params = {}) => new Promise((resolve, reject) => {
    const id = ++nextId;
    pending.set(id, { resolve, reject });
    ws.send(JSON.stringify({ id, method, params }));
  });
  ws.onmessage = ({ data }) => {
    const message = JSON.parse(data);
    if (message.id) {
      const waiter = pending.get(message.id);
      if (!waiter) return;
      pending.delete(message.id);
      if (message.error) waiter.reject(Error(message.error.message)); else waiter.resolve(message.result);
    } else handlers.get(message.method)?.(message.params);
  };
  await new Promise((resolve, reject) => { ws.onopen = resolve; ws.onerror = reject; });
  const identity = "a5aaf531-9fb9-4d35-a56e-a41b2a8a2d60";
  const origin = { system: "probata", kind: "entity", record_id: identity, record_version: "fixture-v1" };
  const source = { origin, title: "Synthetic person", record: { display_name: "Synthetic person" } };
  const saved = { claim_id: "987a1105-9bae-4313-9eee-56027ddcb3b6", revision: 1, text: "Synthetic person",
    kind: "assertion", claimant: "", response: "Preserved legal response", links: [], gaps: [], followups: [],
    evidence_status: "evidence_needed", updated_at: new Date().toISOString(), origin,
    origin_state: "unchanged", origin_record: { origin, title: source.title, payload: source.record } };
  let exists = false, available = true;
  handlers.set("Runtime.exceptionThrown", value => errors.push(value.exceptionDetails?.text));
  handlers.set("Fetch.requestPaused", async value => {
    try {
      const { request } = value, url = new URL(request.url);
      const path = url.pathname.replace("/api/legal", "");
      let body = null, status = 200;
      if (request.method !== "GET") {
        writes.push(path);
        if (path === "/v1/claims/from-probata" && request.method === "POST") {
          assert.deepEqual(JSON.parse(request.postData).origin, origin);
          exists = true; body = saved;
        } else { status = 400; body = { detail: "Unexpected synthetic mutation" }; }
      } else if (path === "/v1/claims/by-origin") {
        assert.equal(url.searchParams.get("record_id"), identity);
        assert.equal(url.searchParams.get("kind"), "entity");
        body = exists ? { ...saved, origin_state: available ? "unchanged" : "unavailable", origin_record: available ? saved.origin_record : null } : null;
      } else if (path === "/v1/claims") body = exists ? [saved] : [];
      else if (path === "/v1/claim-gaps") body = [];
      else if (path === "/v1/claim-sources") body = { available: false, items: [], reason: "Synthetic fixture" };
      else if (path === "/v1/probata/records") body = { available, records: available ? [source] : [] };
      else if (path === `/v1/claims/${saved.claim_id}`) body = saved;
      else { status = 404; body = { detail: "No synthetic route" }; }
      await send("Fetch.fulfillRequest", { requestId: value.requestId, responseCode: status,
        responseHeaders: [{ name: "Content-Type", value: "application/json" }],
        body: Buffer.from(JSON.stringify(body)).toString("base64") });
    } catch (error) {
      errors.push(error.message);
      await send("Fetch.failRequest", { requestId: value.requestId, errorReason: "Aborted" }).catch(() => {});
    }
  });
  await send("Page.enable"); await send("Runtime.enable");
  await send("Fetch.enable", { patterns: [{ urlPattern: `${base}/api/legal/*`, requestStage: "Request" }] });
  const evaluate = async expression => {
    const result = await send("Runtime.evaluate", { expression, returnByValue: true, awaitPromise: true });
    assert(!result.exceptionDetails, result.exceptionDetails?.text);
    return result.result.value;
  };
  const wait = async expression => {
    for (let i = 0; i < 80; i++) { if (await evaluate(expression).catch(() => false)) return; await sleep(250); }
    throw Error(`Browser did not reach expected state: ${expression}`);
  };
  const navigate = async () => {
    const loaded = new Promise(resolve => handlers.set("Page.loadEventFired", resolve));
    await send("Page.navigate", { url: `${base}/claims?probata_kind=entity&probata_id=${identity}` });
    await Promise.race([loaded, sleep(20000).then(() => { throw Error("Page load timed out"); })]);
    await wait("document.querySelector('h1')?.textContent === 'Claims and evidence'");
  };
  await send("Emulation.setDeviceMetricsOverride", { width: 1440, height: 1000, deviceScaleFactor: 1, mobile: false });
  await navigate();
  await wait("[...document.querySelectorAll('button')].some(b => b.textContent === 'Add legal response' && !b.disabled)");
  assert.equal(writes.length, 0, "Following a Probata link created a record");
  await evaluate("[...document.querySelectorAll('button')].find(b => b.textContent === 'Add legal response').click()");
  await wait("document.querySelector('textarea') && [...document.querySelectorAll('textarea')].some(t => t.value === 'Preserved legal response')");
  assert.deepEqual(writes, ["/v1/claims/from-probata"]);
  await navigate();
  await wait("document.body.innerText.includes('Opened the saved legal response')");
  assert.equal(writes.length, 1, "Reopening created a second response");
  available = false;
  await navigate();
  await wait("document.body.innerText.includes('Source unavailable')");
  assert(await evaluate("[...document.querySelectorAll('textarea')].some(t => t.value === 'Preserved legal response')"));
  assert.equal(writes.length, 1);
  await send("Emulation.setDeviceMetricsOverride", { width: 390, height: 844, deviceScaleFactor: 1, mobile: true });
  await sleep(500);
  assert.equal(await evaluate("document.documentElement.scrollWidth > innerWidth"), false, "Mobile page overflows");
  assert.deepEqual(errors, []);
  const screenshot = await send("Page.captureScreenshot", { format: "png" });
  writeFileSync(join(out, "legal-return-mobile.png"), Buffer.from(screenshot.data, "base64"));
  const proof = { passed: true, no_write_on_open: true, explicit_add_only: true, existing_response_reopened: true,
    response_preserved_on_outage: true, mobile_no_overflow: true, errors, production_records_written: 0,
    scope: "Deployed Legal page with intercepted synthetic API responses in remote devbox" };
  writeFileSync(join(out, "proof.json"), JSON.stringify(proof, null, 2));
  console.log(JSON.stringify(proof));
} finally {
  ws?.close(); chrome.kill();
}

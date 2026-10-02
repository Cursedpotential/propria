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
  const claimId = "987a1105-9bae-4313-9eee-56027ddcb3b6";
  const followupId = "1285e01b-87bc-408f-ac71-74b4dd253969";
  const requestId = "ad001ba9-4534-4ba0-bf3c-7f50ba90f24a";
  const request = { mode: "TEST", matter_id: "deadbeef-dead-beef-dead-beefdeadbeef",
    court_case_id: "cafebabe-cafe-babe-cafe-babecafebabe", legal_matter_id: "11111111-1111-4111-8111-111111111111",
    claim_id: claimId, followup_id: followupId, question: "Find the missing source for this statement.",
    sources: [{ kind: origin.kind, record_id: identity, record_version: origin.record_version }] };
  const source = { origin, title: "Synthetic person", record: { display_name: "Synthetic person" } };
  const followup = { followup_id: followupId, kind: "investigate", description: request.question,
    status: "open", gap_id: null, investigation: null };
  const saved = { claim_id: claimId, revision: 1, text: "Synthetic statement", kind: "assertion", claimant: "",
    response: "Preserved legal response", links: [], gaps: [], followups: [followup], evidence_status: "evidence_needed",
    updated_at: new Date().toISOString(), origin, origin_state: "unchanged",
    origin_record: { origin, title: source.title, payload: source.record } };
  let dispatches = 0, refreshes = 0;
  handlers.set("Runtime.exceptionThrown", value => errors.push(value.exceptionDetails?.text));
  handlers.set("Fetch.requestPaused", async value => {
    try {
      const { request: http } = value, url = new URL(http.url), path = url.pathname.replace("/api/legal", "");
      let body = null, status = 200;
      if (http.method !== "GET") {
        writes.push(path);
        assert.equal(JSON.parse(http.postData).expected_revision, saved.revision);
        const route = `/v1/claims/${claimId}/followups/${followupId}`;
        if (path === `${route}/dispatch` && http.method === "POST") {
          dispatches++;
          if (dispatches === 1) followup.investigation = { state: "prepared", request, request_id: null,
            idempotency_key: "12345678-1234-4234-8234-123456789abc", actor_uid: "signed-bff:legal-web-bff",
            actor_username: "signed-bff:legal-web-bff", remote_status: null, remote_updated_at: null,
            last_error: "Probata did not acknowledge the request. Retry sending.", results: [] };
          else {
            assert.equal(dispatches, 2);
            Object.assign(followup.investigation, { state: "acknowledged", request_id: requestId,
              remote_status: "received", remote_updated_at: new Date().toISOString(), last_error: null });
          }
        } else if (path === `${route}/refresh-status` && http.method === "POST") {
          refreshes++;
          if (refreshes === 1) followup.investigation.last_error = "Probata is unavailable. Refresh again.";
          else Object.assign(followup.investigation, { remote_status: "completed", last_error: null,
            results: [{ summary: "Synthetic finding with a native source reference.", sources: request.sources,
              tool: "Synthetic fixture only", run_id: "fixture-run" }] });
        } else throw Error(`Unexpected synthetic mutation ${path}`);
        saved.revision++; body = saved;
      } else if (path === "/v1/claims/by-origin" || path === `/v1/claims/${claimId}`) body = saved;
      else if (path === "/v1/claims") body = [saved];
      else if (path === "/v1/claim-gaps") body = [];
      else if (path === "/v1/claim-sources") body = { available: false, items: [], reason: "Synthetic fixture" };
      else if (path === "/v1/probata/records") body = { available: true, records: [source], mode: "TEST",
        matter_id: request.matter_id, court_case_id: request.court_case_id };
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
    const r = await send("Runtime.evaluate", { expression, returnByValue: true, awaitPromise: true });
    assert(!r.exceptionDetails, r.exceptionDetails?.exception?.description || r.exceptionDetails?.text); return r.result.value;
  };
  const wait = async expression => {
    for (let i = 0; i < 80; i++) { if (await evaluate(expression).catch(() => false)) return; await sleep(250); }
    throw Error(`Browser did not reach expected state: ${expression}`);
  };
  const click = async label => {
    await wait(`[...document.querySelectorAll('button')].some(b => b.textContent === ${JSON.stringify(label)} && !b.disabled)`);
    return evaluate(`[...document.querySelectorAll('button')].find(b => b.textContent === ${JSON.stringify(label)} && !b.disabled).click()`);
  };
  const navigate = async () => {
    await send("Page.navigate", { url: `${base}/claims?probata_kind=entity&probata_id=${identity}` });
    await wait("[...document.querySelectorAll('textarea')].some(t => t.value === 'Preserved legal response')");
    await sleep(300);
  };
  await send("Emulation.setDeviceMetricsOverride", { width: 1440, height: 1000, deviceScaleFactor: 1, mobile: false });
  await navigate(); assert.equal(writes.length, 0);
  await click("Send to Probata");
  await wait("[...document.querySelectorAll('button')].some(b => b.textContent === 'Retry send' && !b.disabled)");
  assert(await evaluate("[...document.querySelectorAll('button')].filter(b => ['Mark done','Cancel plan'].includes(b.textContent)).every(b => b.disabled)"));
  await navigate(); assert.equal(dispatches, 1, "Reopening dispatched automatically");
  await click("Retry send"); await wait("document.body.innerText.includes('Waiting for execution.')");
  assert.equal(followup.status, "open"); assert.equal(followup.investigation.request_id, requestId);
  await navigate(); assert.equal(dispatches, 2, "Acknowledged request sent twice on reopen");
  assert(!(await evaluate("[...document.querySelectorAll('button')].some(b => ['Send to Probata','Retry send'].includes(b.textContent))")));
  await click("Refresh request status"); await wait("document.body.innerText.includes('Probata is unavailable. Refresh again.')");
  assert.equal(followup.investigation.request_id, requestId); assert.equal(followup.investigation.remote_status, "received");
  await click("Refresh request status"); await wait("document.body.innerText.includes('Synthetic finding with a native source reference.')");
  await send("Emulation.setDeviceMetricsOverride", { width: 390, height: 844, deviceScaleFactor: 1, mobile: true });
  await sleep(500);
  await evaluate("document.querySelector('[aria-label=\"Follow-up plans\"]').scrollIntoView()");
  await click("Refresh request status");
  await wait("document.body.innerText.includes('Investigation status refreshed.')");
  assert.equal(refreshes, 3, "Phone refresh did not run");
  assert.equal(await evaluate("document.documentElement.scrollWidth > innerWidth"), false, "Mobile page overflows");
  assert.deepEqual(errors, []);
  const screenshot = await send("Page.captureScreenshot", { format: "png" });
  writeFileSync(join(out, "investigation-mobile.png"), Buffer.from(screenshot.data, "base64"));
  const proof = { passed: true, no_dispatch_on_open: true, timeout_prepared_reopened: true,
    retry_then_acknowledgement: true, request_id_preserved_on_refresh_outage: true,
    acknowledged_request_not_resent: true, typed_result_rendered: true, plan_remains_open: true,
    mobile_no_overflow: true, mobile_status_refresh: true, dispatches, refreshes, errors, production_records_written: 0,
    scope: "Deployed Legal UI, intercepted synthetic API responses in remote devbox; backend persistence proved separately" };
  writeFileSync(join(out, "proof.json"), JSON.stringify(proof, null, 2)); console.log(JSON.stringify(proof));
} finally { ws?.close(); chrome.kill(); }

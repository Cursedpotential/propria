// Byline: Claude Code · Opus 5.5 · 2026-09-26
// Headless-Chrome screenshots of the Propria Homepage portal. Runs INSIDE the Probata devbox on
// ovh-files (never on the owner's desktop; owner rule 2026-09-24). Launched by shoot.sh.
// No dependencies: Node 22's fetch + WebSocket drive Chrome over the DevTools protocol.
//
//   node shoot.mjs <plan.json> <out-dir>
//
// plan.json is a list of shots:
//   { "name": "after-tailnet", "url": "https://...", "width": 1600, "height": 1000,
//     "fullPage": true, "settleMs": 12000, "scrollY": 0, "typeText": "",
//     "rewrite": [{ "prefix": "http://127.0.0.1:3990/progress", "to": "http://100.72.169.40:3020" }] }
// scrollY scrolls the page before the viewport capture; typeText types on the page (quick launch).
// "rewrite" re-points same-origin /progress requests at the progress board, which is what
// tailscale serve (tailnet) and Traefik (public) do in production; it is only used for previews
// that are not served under those hostnames.
// Each shot writes <name>.png (the viewport), <name>-full.png (fullPage) and <name>.json (layout
// numbers: column boxes, group boxes, tile heights, horizontal overflow, console errors).
import { spawn } from "node:child_process";
import { existsSync, mkdirSync, mkdtempSync, readFileSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { setTimeout as sleep } from "node:timers/promises";

const [planPath, outDir] = process.argv.slice(2);
if (!planPath || !outDir) {
  console.error("usage: node shoot.mjs <plan.json> <out-dir>");
  process.exit(2);
}
const plan = JSON.parse(readFileSync(planPath, "utf8"));
const chromeBin = process.env.CHROME_BIN || "/opt/google/chrome/chrome";
mkdirSync(outDir, { recursive: true });

async function launchChrome() {
  const profile = mkdtempSync(join(tmpdir(), "portal-shoot-"));
  const proc = spawn(
    chromeBin,
    [
      "--headless=new",
      "--remote-debugging-port=0",
      `--user-data-dir=${profile}`,
      "--no-first-run",
      "--no-default-browser-check",
      "--hide-scrollbars",
      "--disable-gpu",
      "--mute-audio",
      "about:blank",
    ],
    { stdio: ["ignore", "ignore", "ignore"] },
  );
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

// Layout numbers read from the rendered page, so "no big empty gaps" is measured, not eyeballed.
const METRICS = `(() => {
  const box = (el) => {
    if (!el) return null;
    const b = el.getBoundingClientRect();
    return { x: Math.round(b.x), y: Math.round(b.y + window.scrollY), w: Math.round(b.width), h: Math.round(b.height) };
  };
  const groups = [...document.querySelectorAll(".services-group, .bookmark-group")].map((g) => ({
    name: (g.querySelector(".service-group-name, h2")?.textContent || "").trim(),
    ...box(g),
    tiles: g.querySelectorAll("li.service, li.bookmark").length,
  }));
  const cards = [...document.querySelectorAll("li.service > .service-card")].map((c) => ({
    name: c.parentElement.dataset.name,
    ...box(c),
  }));
  const widgetErrors = [...document.querySelectorAll("li.service")]
    .filter((li) => /API Error|Failed to load|unavailable/i.test(li.innerText))
    .map((li) => li.dataset.name);
  // Homepage scrolls inside #inner_wrapper, not the document, so page size is read from it.
  const scroller = document.getElementById("inner_wrapper") || document.documentElement;
  return {
    title: document.title,
    viewport: { w: window.innerWidth, h: window.innerHeight },
    document: { w: scroller.scrollWidth, h: scroller.scrollHeight },
    horizontalOverflow: scroller.scrollWidth > scroller.clientWidth || document.documentElement.scrollWidth > window.innerWidth,
    information: box(document.querySelector("#information-widgets")),
    layoutGroups: box(document.querySelector("#layout-groups")),
    services: box(document.querySelector("#services")),
    bookmarks: box(document.querySelector("#bookmarks")),
    footer: box(document.querySelector("#footer")),
    groups,
    cards,
    widgetErrors,
    text: document.body.innerText.slice(0, 600),
  };
})()`;

async function shoot(port, shot) {
  const width = shot.width || 1600;
  const height = shot.height || 1000;
  const mobile = shot.mobile ?? width < 768;
  const created = await fetch(`http://127.0.0.1:${port}/json/new?about:blank`, { method: "PUT" });
  const target = await created.json();
  const cdp = cdpSession(target.webSocketDebuggerUrl);
  await cdp.opened;
  const consoleErrors = [];
  cdp.on("Runtime.exceptionThrown", (p) => consoleErrors.push(p.exceptionDetails?.exception?.description || p.exceptionDetails?.text));
  cdp.on("Log.entryAdded", (p) => {
    if (p.entry.level === "error") consoleErrors.push(`${p.entry.source}: ${p.entry.text} ${p.entry.url || ""}`.trim());
  });
  await cdp.send("Runtime.enable");
  await cdp.send("Log.enable");
  await cdp.send("Page.enable");
  await cdp.send("Emulation.setDeviceMetricsOverride", { width, height, deviceScaleFactor: 1, mobile });
  if (shot.rewrite?.length) {
    cdp.on("Fetch.requestPaused", (p) => {
      const rule = shot.rewrite.find((r) => p.request.url.startsWith(r.prefix));
      const params = { requestId: p.requestId };
      if (rule) params.url = rule.to + p.request.url.slice(rule.prefix.length);
      cdp.send("Fetch.continueRequest", params).catch(() => {});
    });
    await cdp.send("Fetch.enable", { patterns: shot.rewrite.map((r) => ({ urlPattern: `${r.prefix}*` })) });
  }
  const loaded = new Promise((resolve) => cdp.on("Page.loadEventFired", resolve));
  await cdp.send("Page.navigate", { url: shot.url });
  await Promise.race([loaded, sleep(30000)]);
  await sleep(shot.settleMs ?? 12000);
  if (shot.scrollY) {
    // Homepage scrolls #inner_wrapper; this shows what stays in view part-way down the page.
    await cdp.send("Runtime.evaluate", {
      expression: `(document.getElementById("inner_wrapper") || document.scrollingElement).scrollTo(0, ${Number(shot.scrollY)})`,
    });
    await sleep(800);
  }
  if (shot.typeText) {
    // Typing on the page body opens Homepage's quick-launch search.
    for (const ch of shot.typeText) {
      await cdp.send("Input.dispatchKeyEvent", { type: "keyDown", key: ch, text: ch });
      await cdp.send("Input.dispatchKeyEvent", { type: "keyUp", key: ch });
    }
    await sleep(1500);
  }

  const evaluated = await cdp.send("Runtime.evaluate", { expression: METRICS, returnByValue: true });
  const metrics = { shot, ...(evaluated.result?.value || {}), consoleErrors };
  const viewport = await cdp.send("Page.captureScreenshot", { format: "png" });
  writeFileSync(join(outDir, `${shot.name}.png`), Buffer.from(viewport.data, "base64"));
  if (shot.fullPage) {
    const layout = await cdp.send("Page.getLayoutMetrics");
    const fullHeight = Math.max(Math.ceil(layout.cssContentSize.height), metrics.document?.h || 0);
    await cdp.send("Emulation.setDeviceMetricsOverride", { width, height: fullHeight, deviceScaleFactor: 1, mobile });
    await sleep(1500);
    const full = await cdp.send("Page.captureScreenshot", { format: "png", captureBeyondViewport: true });
    writeFileSync(join(outDir, `${shot.name}-full.png`), Buffer.from(full.data, "base64"));
    metrics.fullHeight = fullHeight;
  }
  writeFileSync(join(outDir, `${shot.name}.json`), JSON.stringify(metrics, null, 1));
  cdp.close();
  await fetch(`http://127.0.0.1:${port}/json/close/${target.id}`).catch(() => {});
  return metrics;
}

const { proc, port } = await launchChrome();
let failed = 0;
try {
  for (const shot of plan) {
    try {
      const m = await shoot(port, shot);
      console.log(
        `${shot.name}: ${m.title} | doc ${m.document?.w}x${m.document?.h} | overflow-x ${m.horizontalOverflow}` +
          ` | groups ${m.groups?.length} | tiles ${m.cards?.length} | console errors ${m.consoleErrors.length}`,
      );
    } catch (error) {
      failed += 1;
      console.error(`${shot.name}: FAILED ${error.message}`);
    }
  }
} finally {
  proc.kill();
}
process.exit(failed ? 1 : 0);

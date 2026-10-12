// Byline: Codex · GPT-6 · 2026-10-12. Remote-only checks of actual package stories.
import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { mkdir, mkdtemp, readFile } from "node:fs/promises";
import { createServer } from "node:http";
import { extname, resolve, sep } from "node:path";
import test from "node:test";

/** Send bounded Chrome protocol commands through a pipe, following the existing matter smoke donor.
 * Input: explicitly launched remote browser. Output: command responses. Effects: test browser only.
 * Pick for built Storybook checks; this harness never opens a desktop browser or live API.
 */
class StoryBrowser {
  constructor(browser) {
    this.browser = browser; this.id = 0; this.pending = new Map(); this.buffer = "";
    browser.stdio[4].on("data", (chunk) => {
      this.buffer += chunk.toString();
      let boundary;
      while ((boundary = this.buffer.indexOf("\0")) !== -1) {
        const raw = this.buffer.slice(0, boundary); this.buffer = this.buffer.slice(boundary + 1);
        if (!raw) continue;
        const message = JSON.parse(raw); const pending = this.pending.get(message.id);
        if (!pending) continue;
        this.pending.delete(message.id); clearTimeout(pending.timer);
        if (message.error) pending.reject(new Error(message.error.message)); else pending.accept(message.result);
      }
    });
  }
  /** Execute one protocol command with a timeout and optional page session.
   * Inputs: method, arguments, session. Output: response. Effects: test browser operation.
   */
  command(method, params = {}, sessionId) {
    const id = ++this.id;
    return new Promise((accept, reject) => {
      const timer = setTimeout(() => { this.pending.delete(id); reject(new Error(`Timeout: ${method}`)); }, 10000);
      this.pending.set(id, { accept, reject, timer });
      this.browser.stdio[3].write(`${JSON.stringify({ id, method, params, ...(sessionId ? { sessionId } : {}) })}\0`);
    });
  }
  /** Read a page expression without touching an external service.
   * Inputs: session and expression. Output: JSON value. Effects: test page evaluation only.
   */
  async evaluate(sessionId, expression) {
    const result = await this.command("Runtime.evaluate", { expression, returnByValue: true, awaitPromise: true }, sessionId);
    if (result.exceptionDetails) throw new Error(JSON.stringify(result.exceptionDetails));
    return result.result?.value;
  }
}

const browserPath = process.env.CONTEXT_STORY_BROWSER;
const skip = process.platform === "win32" ? "Never launch a browser on the owner's Windows desktop" : !browserPath ? "Opt in on the existing remote devbox with CONTEXT_STORY_BROWSER" : false;

test("actual package stories render category headers and complete save/failure/dirty/reopen interactions", { skip, timeout: 90000 }, async () => {
  const root = resolve("storybook-static");
  const index = JSON.parse(await readFile(resolve(root, "index.json"), "utf8"));
  const stories = Object.values(index.entries).filter((entry) => entry.title === "Context review/Extracted package" && entry.type === "story");
  assert.equal(stories.length, 6, "the package stories must actually be registered in Storybook");
  const server = createServer(async (request, response) => {
    try {
      const relative = decodeURIComponent(new URL(request.url, "http://localhost").pathname);
      const file = resolve(root, `.${relative === "/" ? "/index.html" : relative}`);
      if (!file.startsWith(`${root}${sep}`)) throw new Error("Outside static build");
      const types = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".json": "application/json", ".svg": "image/svg+xml" };
      response.writeHead(200, { "Content-Type": types[extname(file)] ?? "application/octet-stream" }); response.end(await readFile(file));
    } catch { response.writeHead(404); response.end("Not found"); }
  });
  await new Promise((accept) => server.listen(0, "127.0.0.1", accept));
  const quarantine = resolve("../../..", "to_be_deleted"); await mkdir(quarantine, { recursive: true });
  const profile = await mkdtemp(resolve(quarantine, "package-stories-browser-"));
  const browser = spawn(browserPath, ["--headless=new", "--disable-gpu", "--no-first-run", "--no-default-browser-check", "--remote-debugging-pipe", `--user-data-dir=${profile}`, "about:blank"], { stdio: ["ignore", "ignore", "pipe", "pipe", "pipe"] });
  const cdp = new StoryBrowser(browser);
  const errors = []; browser.stderr.on("data", (chunk) => errors.push(chunk.toString()));
  try {
    for (const story of stories) {
      const target = await cdp.command("Target.createTarget", { url: "about:blank" });
      const { sessionId } = await cdp.command("Target.attachToTarget", { targetId: target.targetId, flatten: true });
      await cdp.command("Page.enable", {}, sessionId);
      await cdp.command("Page.addScriptToEvaluateOnNewDocument", { source: `globalThis.__packageStoryEvents = []; const observer = setInterval(() => { const channel = globalThis.__STORYBOOK_ADDONS_CHANNEL__; if (!channel) return; clearInterval(observer); for (const name of ['storyFinished','storyErrored','storyThrewException','playFunctionThrewException']) channel.on(name, data => globalThis.__packageStoryEvents.push({name,data})); }, 5);` }, sessionId);
      await cdp.command("Page.navigate", { url: `http://127.0.0.1:${server.address().port}/iframe.html?id=${story.id}&viewMode=story` }, sessionId);
      const deadline = Date.now() + 12000;
      let state;
      while (Date.now() < deadline) {
        state = await cdp.evaluate(sessionId, `({events: globalThis.__packageStoryEvents ?? [], phase: globalThis.__STORYBOOK_PREVIEW__?.currentRender?.phase, text: document.body?.innerText ?? ''})`);
        if (state.events.some((event) => event.name !== "storyFinished")) throw new Error(`${story.id}: ${JSON.stringify(state)}`);
        if (state.events.some((event) => event.name === "storyFinished") || state.phase === "completed") break;
        await new Promise((accept) => setTimeout(accept, 100));
      }
      assert.ok(state.events.some((event) => event.name === "storyFinished") || state.phase === "completed", `${story.id} did not finish: ${JSON.stringify(state)}`);
      const failed = state.events.find((event) => event.name === "storyFinished" && event.data?.status === "error");
      assert.equal(failed, undefined, JSON.stringify(failed));
      assert.ok(state.text.includes(story.name === "Category Headers" ? "Fixture category" : "Synthetic extracted statement"), `${story.id} did not render its actual component`);
      console.log(`Verified ${story.id}`);
      await cdp.command("Target.closeTarget", { targetId: target.targetId });
    }
  } catch (failure) { failure.message += `\nBrowser stderr: ${errors.join("").slice(-1000)}`; throw failure; }
  finally { browser.kill(); await new Promise((accept) => server.close(accept)); }
});

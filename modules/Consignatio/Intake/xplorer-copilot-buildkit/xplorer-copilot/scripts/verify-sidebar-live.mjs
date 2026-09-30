// Byline: Claude Code - Opus 5 - 2026-09-27
// Live check of the Intake sidebar regression, run with headless Chrome inside the devbox
// on ovh-files (never on the owner's desktop -- browsers there hang the machine).
//
// The regression, owner 2026-09-26: "There's no file tree, like there's regression in the
// other pages." Clicking the magnifying glass used to swap the whole sidebar body out.
// This asserts that after the search tab is clicked, the file tree is STILL in the DOM.
//
// No dependencies: node 22 has global fetch and WebSocket.

const CDP = 'http://127.0.0.1:9222';
const PAGE = process.env.PAGE_URL || 'http://127.0.0.1:8899/';

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

async function target() {
  for (let i = 0; i < 40; i++) {
    try {
      const list = await fetch(CDP + '/json').then((r) => r.json());
      const page = list.find((t) => t.type === 'page' && t.webSocketDebuggerUrl);
      if (page) return page;
    } catch {
      /* chrome not up yet */
    }
    await sleep(500);
  }
  throw new Error('no CDP page target after 20s');
}

function client(url) {
  const ws = new WebSocket(url);
  let id = 0;
  const pending = new Map();
  const ready = new Promise((res, rej) => {
    ws.addEventListener('open', res);
    ws.addEventListener('error', rej);
  });
  ws.addEventListener('message', (ev) => {
    const msg = JSON.parse(ev.data);
    if (msg.id && pending.has(msg.id)) {
      const { resolve, reject } = pending.get(msg.id);
      pending.delete(msg.id);
      if (msg.error) reject(new Error(JSON.stringify(msg.error)));
      else resolve(msg.result);
    }
  });
  const send = (method, params) => {
    const mid = ++id;
    return new Promise((resolve, reject) => {
      pending.set(mid, { resolve, reject });
      ws.send(JSON.stringify({ id: mid, method, params: params || {} }));
    });
  };
  return { ready, send, close: () => ws.close() };
}

// Runs in the page. Returns the sidebar sections actually rendered, by their
// data-sidebar-section attribute, plus a little context for diagnosis.
const PROBE = `(() => {
  const secs = Array.from(document.querySelectorAll('[data-sidebar-section]'))
    .map((el) => el.getAttribute('data-sidebar-section'));
  const tabs = Array.from(document.querySelectorAll('button[role="tab"]'))
    .map((b) => b.getAttribute('aria-label') + (b.getAttribute('aria-selected') === 'true' ? ' *ACTIVE*' : ''));
  const nav = document.querySelector('nav[aria-label="File explorer sidebar"]');
  return JSON.stringify({
    sections: secs,
    tabs: tabs,
    sidebarPresent: Boolean(nav),
    bodyChars: document.body.innerText.length
  });
})()`;

const CLICK_SEARCH = `(() => {
  const btn = Array.from(document.querySelectorAll('button[role="tab"]'))
    .find((b) => (b.getAttribute('aria-label') || '').startsWith('Search files'));
  if (!btn) return 'NO_SEARCH_TAB';
  btn.click();
  return 'CLICKED';
})()`;

const main = async () => {
  const t = await target();
  const c = client(t.webSocketDebuggerUrl);
  await c.ready;
  await c.send('Page.enable');
  await c.send('Runtime.enable');
  await c.send('Page.navigate', { url: PAGE });
  await sleep(6000); // let the SPA mount

  const evalJS = async (expr) => {
    const r = await c.send('Runtime.evaluate', { expression: expr, returnByValue: true });
    if (r.exceptionDetails) return 'EXCEPTION: ' + JSON.stringify(r.exceptionDetails.text);
    return r.result.value;
  };

  const before = JSON.parse(await evalJS(PROBE));
  console.log('BEFORE click  :', JSON.stringify(before));

  const clicked = await evalJS(CLICK_SEARCH);
  console.log('click search  :', clicked);
  await sleep(2500);

  const after = JSON.parse(await evalJS(PROBE));
  console.log('AFTER  click  :', JSON.stringify(after));

  console.log('');
  if (!before.sidebarPresent) {
    console.log('INCONCLUSIVE: the sidebar never rendered, so the regression cannot be judged.');
    console.log('              body text length =', before.bodyChars);
    c.close();
    process.exit(2);
  }

  const treeAfter = after.sections.includes('fileTree');
  const searchAfter = after.sections.includes('search');

  console.log('file tree present AFTER opening search :', treeAfter);
  console.log('search section present                 :', searchAfter);
  console.log('');
  if (treeAfter && searchAfter) {
    console.log('PASS - search and the file tree are on screen together.');
    c.close();
    process.exit(0);
  }
  console.log('FAIL - the regression is still present.');
  c.close();
  process.exit(1);
};

main().catch((e) => {
  console.error('harness error:', e.message);
  process.exit(3);
});

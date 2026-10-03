// Byline: Claude Code · Sonnet · 2026-10-02
// Owner 2026-10-02: "a mobile-ready, slimmed-down version of Probata" at /m. These pins read the
// source; they are not a browser or live proof. The rules: the Imported client only reads, the one
// write inside components/mobile is the existing Review decision, the live case is the only case,
// and the shell never says TEST or REAL to the owner.
// Amended 2026-10-02 (Claude Code · Sonnet 5.5): Extract and Send to Surreal also start from /m, but they live in
// components/conversations and post through lib/conversation-actions-client.ts (pinned in
// conversation-actions.contract.test.mjs), so these pins about components/mobile hold unchanged.
import assert from "node:assert/strict";
import { readdirSync, readFileSync } from "node:fs";
import test from "node:test";

const read = (path) => readFileSync(new URL(path, import.meta.url), "utf8");
const mobileDir = new URL("../src/components/mobile/", import.meta.url);
const mobileFiles = readdirSync(mobileDir).map((name) => [name, readFileSync(new URL(name, mobileDir), "utf8")]);

test("the Imported client makes GET requests only", () => {
  const client = read("../src/lib/imported-client.ts");
  assert.doesNotMatch(client, /method:\s*["'](POST|PUT|PATCH|DELETE)/i);
  assert.match(client, /fetch\(`\$\{API_BASE\}\$\{path\}\$\{suffix\}`, \{ signal \}\)/);
});

test("the only write on the mobile shell is the existing Review decision, with the desktop gate", () => {
  const writers = mobileFiles.filter(([, text]) => /decideProffer\(|method:\s*["']POST/.test(text)).map(([name]) => name);
  assert.deepEqual(writers, ["review-views.tsx"]);
  const review = read("../src/components/mobile/review-views.tsx");
  assert.match(review, /decideProffer\(handle, LIVE_MODE, \{ approved, reason: approved \? "" : reason\.trim\(\) \}\)/);
  assert.match(review, /if \(!eligible \|\| pending\) return;/);
  assert.match(review, /if \(!approved && !reason\.trim\(\)\) return;/);
  assert.match(review, /every record shows|Every record shows where it came from/);
  assert.match(review, /PROFFER_CONTEXT_CHECKPOINTS/);
  // Approve and Reject each need a second tap on a confirm button.
  assert.match(review, /Confirm approve/);
  assert.match(review, /Confirm reject/);
});

test("the mobile shell is live-case only and never labels a mode TEST or REAL on screen", () => {
  for (const [name, text] of mobileFiles) {
    assert.doesNotMatch(text.replace(/LIVE_MODE = "REAL"/g, ""), />[^<]*\b(TEST|REAL)\b[^<]*</, `${name} shows a TEST/REAL label`);
    assert.doesNotMatch(text, /setMode\(|"TEST"/, `${name} can leave the live case`);
  }
});

test("routes: /m is a sibling of the desktop shell with Imported, Calls, Search and Review", () => {
  const router = read("../src/router.tsx");
  assert.match(router, /createRoute\(\{ getParentRoute: \(\) => rootRoute, path: "m", component: MobileShell \}\)/);
  for (const path of ['"/"', '"source/$sourceId"', '"thread/$threadId"', '"calls"', '"search"', '"review"', '"review/$handle"']) {
    assert.ok(router.includes(`mobileChild(${path},`), `missing mobile route ${path}`);
  }
  assert.match(router, /rootRoute\.addChildren\(\[desktopTree, mobileTree\]\)/);
});

test("phone ergonomics: 16px inputs, 48px+ targets, safe-area insets, dynamic viewport height", () => {
  const shell = read("../src/components/mobile/mobile-shell.tsx");
  assert.match(shell, /min-h-\[100dvh\]/);
  assert.match(shell, /env\(safe-area-inset-bottom\)/);
  assert.match(shell, /min-h-16/);
  const search = read("../src/components/mobile/search-view.tsx");
  assert.match(search, /h-12 min-w-0 flex-1[^"]*text-base/);
  assert.match(read("../index.html"), /viewport-fit=cover/);
});

test("Who is this? writes only through the governed case-identity API, never a registry table or a second path", () => {
  const sheet = read("../src/components/identity/who-is-this.tsx");
  assert.match(sheet, /import \{ addPlaceholders, editPerson, mergePerson, newIdempotencyKey \} from "@\/lib\/case-identity-client"/);
  assert.doesNotMatch(sheet, /fetch\(|method:\s*["']POST/);
  assert.match(sheet, /verification_state: "confirmed"/);
  const status = read("../src/components/identity/number-status.tsx");
  assert.match(status, /importedApi\.numberStatus/);
  const list = read("../src/components/identity/unknown-numbers-list.tsx");
  assert.match(list, /most frequent first/);
});

test("the imported views are built on existing components, not new UI", () => {
  const views = read("../src/components/mobile/imported-views.tsx");
  assert.match(views, /import \{ MessageBubble \} from "@\/components\/sbv\/message-bubble"/);
  assert.match(views, /import \{ CallsTable \} from "@\/components\/sbv\/calls-table"/);
  assert.match(views, /import \{ ConversationList, type ConversationListItem \} from "@\/components\/sbv\/conversation-list"/);
  assert.match(views, /import \{ ImportedGrid, type ImportedColumn \} from "@\/components\/imported\/imported-grid"/);
  const grid = read("../src/components/imported/imported-grid.tsx");
  assert.match(grid, /from "@glideapps\/glide-data-grid"/);
  assert.match(grid, /useGridTheme/);
  assert.match(read("../src/components/sbv/conversation-list.tsx"), /Ported from modules\/forks\/sbv\/frontend\/src\/components\/ConversationList\.jsx/);
  const sheet = read("../src/components/identity/who-is-this.tsx");
  assert.match(sheet, /from "@\/components\/ui\/sheet"/);
});

test("a split parent backup never reads failed or not finished", () => {
  const views = read("../src/components/mobile/imported-views.tsx");
  assert.match(views, /Split into \$\{total\}/);
  assert.match(views, /if \(source\.split\)/);
});

test("Who is this? shows the source, offers one-tap own-number, a searchable picker and confirm, all through the governed calls", () => {
  const sheet = read("../src/components/identity/who-is-this.tsx");
  assert.match(sheet, /See where this number appears/);
  assert.match(sheet, /This is my number/);
  assert.match(sheet, /person\.role === "user"/);
  assert.match(sheet, /Yes, this is \{currentName\}/);
  assert.match(sheet, /placeholder="Type to filter"/);
  assert.doesNotMatch(sheet, /fetch\(|method:\s*["']POST/);
  const source = read("../src/components/identity/number-source-sheet.tsx");
  assert.match(source, /import \{ CallsTable \} from "@\/components\/sbv\/calls-table"/);
  assert.match(source, /import \{ MessageBubble \} from "@\/components\/sbv\/message-bubble"/);
  assert.match(source, /Open this conversation at this message/);
  assert.match(source, /casevault_key/);
  assert.match(read("../src/lib/imported-client.ts"), /\/api\/imported\/number-records/);
});

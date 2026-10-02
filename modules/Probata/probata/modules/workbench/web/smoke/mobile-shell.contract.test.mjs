// Byline: Claude Code · Sonnet · 2026-10-02
// Owner 2026-10-02: "a mobile-ready, slimmed-down version of Probata" at /m. These pins read the
// source; they are not a browser or live proof. The rules: the Imported client only reads, the one
// write on the whole mobile shell is the existing Review decision, the live case is the only case,
// and the shell never says TEST or REAL to the owner.
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

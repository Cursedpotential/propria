// Byline: Claude Code · Sonnet 5.5 · 2026-10-02
// Owner 2026-10-02: a button on mobile and desktop to check conversations and move them to Surreal; a separate Extract button that
// asks which extractors to use; view the extractions, read-only on mobile. These pins read the source; they are not a browser or
// live proof. The rules: the picker lists whatever the engine's registry lists (none hard-coded), the default is pre-checked, every
// start carries an Idempotency-Key, the Extractions view only reads, the pieces are the app's own (shadcn Sheet, Checkbox, Button,
// Badge, WorkflowSteps, ConversationList, MessageBubble, ImportedGrid), and the actions exist on both /m and the desktop.
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const read = (path) => readFileSync(new URL(path, import.meta.url), "utf8");
const client = read("../src/lib/conversation-actions-client.ts");
const actions = read("../src/components/conversations/conversation-actions.tsx");
const view = read("../src/components/conversations/extractions-view.tsx");
const hooks = read("../src/hooks/use-conversation-actions.ts");
const list = read("../src/components/sbv/conversation-list.tsx");

test("both starts post with an Idempotency-Key; every other call is a GET", () => {
  assert.match(client, /"Idempotency-Key": key/);
  assert.match(client, /\/api\/imported\/threads\/extract/);
  assert.match(client, /\/api\/imported\/threads\/send-to-surreal/);
  const posts = client.match(/method: "POST"/g) ?? [];
  assert.equal(posts.length, 1, "one shared POST helper");
  assert.match(client, /extractions: \(threadId: string, signal\?: AbortSignal\) =>\s+request<ThreadExtractions>/);
  assert.match(hooks, /newIdempotencyKey\("extract"\)/);
  assert.match(hooks, /newIdempotencyKey\("send"\)/);
});

test("the picker lists the engine's registry, multi-select, with the default pre-checked; no extractor id is hard-coded in the UI", () => {
  assert.match(actions, /conversationApi|useExtractors/);
  assert.match(actions, /filter\(\(extractor\) => extractor\.default\)/);
  assert.match(actions, /extractors\.data\.extractors\.map\(\(extractor\) =>/);
  for (const id of ["go-kimi-k3", "semantica", "langextract"]) {
    for (const [name, text] of [["actions", actions], ["view", view], ["hooks", hooks], ["client", client]]) {
      assert.ok(!text.includes(`"${id}"`), `${name} hard-codes the extractor ${id}`);
    }
  }
  assert.match(actions, /Extract with \$\{checked\.size\} extractor/);
  assert.match(actions, /checked\.size === 0/, "extracting with nothing checked is not allowed");
});

test("the Extractions view only reads and groups by extractor", () => {
  assert.doesNotMatch(view, /\bfetch\(|method:\s*["']POST|useMutation|<Input|<Textarea|onChange=/);
  assert.match(view, /extraction-group-/);
  assert.match(view, /compare only/);
  assert.match(client, /\/extractions/);
  assert.match(actions, /Read-only/);
});

test("the actions are built from the app's own components, not new UI", () => {
  for (const component of ["sheet", "checkbox", "button", "badge", "skeleton"]) {
    assert.match(actions, new RegExp(`from "@/components/ui/${component}"`), `actions do not use ui/${component}`);
  }
  assert.match(actions, /import \{ WorkflowSteps \} from "@\/components\/entities\/workflow-steps"/);
  assert.match(view, /from "@\/components\/ui\/card"/);
  assert.match(read("../src/components/conversations/selectable-conversations.tsx"), /import \{ ConversationList, type ConversationListItem \} from "@\/components\/sbv\/conversation-list"/);
  const page = read("../src/app/conversations/page.tsx");
  assert.match(page, /ImportedGrid/);
  assert.match(page, /MessageBubble/);
  assert.match(page, /SelectableConversations/);
});

test("a conversation row's checkbox sits beside the link, so ticking it never opens the conversation", () => {
  assert.match(list, /selection\?: ConversationSelection/);
  const checkbox = list.indexOf('data-testid="conversation-select"');
  const link = list.indexOf("<AppLink href={item.href}");
  assert.ok(checkbox > 0 && link > checkbox, "the checkbox is rendered before and outside the AppLink");
  assert.match(list, /Select \$\{displayName\(item\)\}/);
});

test("mobile: threads are checkable, and a conversation has Extractions, Extract and Send to Surreal", () => {
  const views = read("../src/components/mobile/imported-views.tsx");
  assert.match(views, /selection=\{\{ selected, onToggle: toggle \}\}/);
  assert.match(views, /<ConversationSelectionBar[\s\S]*?placement="mobile"/);
  assert.match(views, /<ConversationToolbar threadId=\{threadId\} placement="mobile" \/>/);
  assert.match(actions, /fixed inset-x-0 bottom-\[calc\(4rem\+env\(safe-area-inset-bottom\)\)\]/, "the bar clears the tab bar");
  assert.match(actions, /className="h-11/, "44px+ tap targets");
});

test("desktop: Read retains conversation tools and legacy bookmarks", () => {
  assert.match(read("../src/router.tsx"), /legacyRoute\("conversations"\)/);
  assert.match(read("../src/surfaces/primary/navigation.ts"), /href: "\/read"/);
  const page = read("../src/components/read/read-conversation.tsx");
  assert.match(page, /<ConversationToolbar threadId=\{threadId\} placement="desktop" \/>/);
  assert.match(read("../src/components/read/read-context.tsx"), /<ExtractionsView threadId=\{threadId\} \/>/);
  assert.match(read("../src/components/read/read-workspace.tsx"), /<SelectableConversations items=\{items\} placement="desktop" \/>/);
});

test("only message conversations are offered: the ids are the Imported view's threads, never an AI chat", () => {
  for (const [name, text] of [["actions", actions], ["view", view], ["client", client], ["hooks", hooks]]) {
    assert.doesNotMatch(text, /ai[_ -]?chat/i, `${name} mentions AI chats`);
  }
  assert.match(client, /thread_ids: threadIds/);
});

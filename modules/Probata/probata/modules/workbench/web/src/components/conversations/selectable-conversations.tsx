// Byline: Claude Code · Sonnet 5.5 · 2026-10-02
// The Imported view's conversation list with a checkbox on every row and the Extract / Send to Surreal bar over the checked
// ones. It is the existing ConversationList (ported from the SBV fork) with its optional row selection switched on.
"use client";

import { ConversationSelectionBar } from "@/components/conversations/conversation-actions";
import { ConversationList, type ConversationListItem } from "@/components/sbv/conversation-list";
import { Button } from "@/components/ui/button";
import { useSelection } from "@/hooks/use-conversation-actions";

export type SelectableItem = ConversationListItem & { id: string };

/** Conversations the owner can check: whole conversations, to extract from or to send to Surreal. Only message conversations are listed here, never AI chats. */
export function SelectableConversations({ items, placement = "desktop" }: { items: SelectableItem[]; placement?: "mobile" | "desktop" }) {
  const { selected, toggle, clear, setAll } = useSelection();
  const ids = items.map((item) => item.id);
  const all = ids.length > 0 && ids.every((id) => selected.has(id));
  return (
    <>
      {items.length > 1 ? (
        <div className="flex items-center justify-between px-4 py-2">
          <p className="text-xs text-muted-foreground">Check conversations to extract from them or send them to Surreal.</p>
          <Button type="button" variant="ghost" className="h-10 shrink-0" onClick={() => (all ? clear() : setAll(ids))}>
            {all ? "Clear" : `Select all ${items.length}`}
          </Button>
        </div>
      ) : null}
      <ConversationList items={items} selection={{ selected, onToggle: toggle }} />
      <ConversationSelectionBar threadIds={ids.filter((id) => selected.has(id))} onClear={clear} placement={placement} />
    </>
  );
}

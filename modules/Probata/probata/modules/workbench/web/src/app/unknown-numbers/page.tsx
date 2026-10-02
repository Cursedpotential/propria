// Byline: Claude Code · Sonnet · 2026-10-02
// /unknown-numbers — placeholders still unnamed, most frequent first (desktop). Step 6: fill in missing context.
"use client";

import { UnknownNumbersList } from "@/components/identity/unknown-numbers-list";

export default function UnknownNumbersPage() {
  return (
    <div className="mx-auto max-w-2xl py-4">
      <h1 className="px-4 text-xl font-semibold">Unnamed numbers</h1>
      <p className="px-4 pt-1 text-sm text-muted-foreground">
        Every number in the imported calls and messages that nobody has named yet is a placeholder person. Name one, or say it is someone you already know.
      </p>
      <UnknownNumbersList />
    </div>
  );
}

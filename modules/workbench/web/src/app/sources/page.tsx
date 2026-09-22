// Byline: Claude Code · Opus 5 · 2026-09-22
// /sources — the front door. One viewport, no page scroll.
"use client";

import { SourcesScreen } from "@/components/sources/sources-screen";

export default function SourcesPage() {
  return (
    <div className="absolute inset-0 flex min-h-0 flex-col overflow-hidden">
      <SourcesScreen />
    </div>
  );
}

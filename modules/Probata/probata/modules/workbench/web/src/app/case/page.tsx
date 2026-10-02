// Byline: Claude Code · Opus 5.5 · 2026-10-01
// /case — the case identity: header, people, identifiers and what the data holds for each. One viewport.
"use client";

import { CaseIdentityScreen } from "@/components/case/case-identity-screen";

export default function CasePage() {
  return (
    <div className="absolute inset-0 flex min-h-0 flex-col overflow-hidden">
      <CaseIdentityScreen />
    </div>
  );
}

// Byline: Claude Code · Sonnet · 2026-10-02
import { UnknownNumbersList } from "@/components/identity/unknown-numbers-list";
import { PageBar } from "@/components/mobile/mobile-ui";

export default function MobileUnknownPage() {
  return (
    <div>
      <PageBar title="Unnamed numbers" subtitle="Every number starts as a placeholder person" back="/m/calls" />
      <UnknownNumbersList />
    </div>
  );
}

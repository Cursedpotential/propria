// Byline: Claude Code · Sonnet 5 · 2026-09-07
// Updated by: OpenAI Codex · GPT-5 · 2026-09-12 — readable navigation hierarchy.
import { Link } from "@tanstack/react-router";
import { CalendarClock, FileStack, Gavel, LayoutDashboard, ListChecks, MessageSquare, ScrollText, ShieldCheck, Wrench } from "lucide-react";
import { cn } from "@/lib/utils";

const NAV_ITEMS = [
  { to: "/", label: "Case Status", icon: LayoutDashboard },
  { to: "/docket", label: "Docket", icon: Gavel },
  { to: "/timeline", label: "Timeline", icon: CalendarClock },
  { to: "/memos", label: "Memos", icon: ScrollText },
  { to: "/evidence", label: "Evidence", icon: FileStack },
  { to: "/evals", label: "Evals", icon: ListChecks },
  { to: "/reference", label: "Reference", icon: ShieldCheck },
  { to: "/tools", label: "Tools", icon: Wrench },
  { to: "/chat", label: "Chat", icon: MessageSquare },
] as const;

export function Nav() {
  return (
    <nav className="flex w-48 shrink-0 flex-col gap-0.5 border-r border-border bg-surface px-2 py-3" aria-label="Primary">
      <div className="mb-2 px-2 text-[13px] font-semibold leading-5 text-text-secondary">Family Court Console</div>
      {NAV_ITEMS.map(({ to, label, icon: Icon }) => (
        <Link
          key={to}
          to={to}
          activeOptions={{ exact: to === "/" }}
          className="flex min-h-8 items-center gap-2 rounded-[var(--radius-sm)] px-2 py-1.5 text-sm leading-5 text-text-secondary hover:bg-surface-hover hover:text-text-primary"
          activeProps={{ className: cn("bg-surface-selected text-text-primary font-medium") }}
        >
          <Icon className="size-4 shrink-0" aria-hidden />
          {label}
        </Link>
      ))}
    </nav>
  );
}

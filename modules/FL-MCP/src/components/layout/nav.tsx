// Byline: Claude Code · Sonnet 5 · 2026-09-07
// Updated by: OpenAI Codex · GPT-5 · 2026-09-12 — Carbon shell and 200% reflow.
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
    <nav
      className="pr-shell flex w-48 shrink-0 flex-col gap-0.5 border-r border-border px-2 py-3 max-[720px]:w-full max-[720px]:flex-row max-[720px]:items-center max-[720px]:overflow-x-auto max-[720px]:border-r-0 max-[720px]:border-b max-[720px]:py-2"
      aria-label="Primary"
    >
      <div className="mb-2 px-2 text-[13px] font-semibold leading-5 text-[var(--pr-shell-text)] max-[720px]:sr-only">Family Court Console</div>
      {NAV_ITEMS.map(({ to, label, icon: Icon }) => (
        <Link
          key={to}
          to={to}
          activeOptions={{ exact: to === "/" }}
          className="flex min-h-9 shrink-0 items-center gap-2 rounded-[var(--radius-sm)] px-2 py-1.5 text-sm leading-5 text-[var(--pr-shell-text)]/75 hover:bg-[var(--pr-shell-surface)] hover:text-[var(--pr-shell-text)]"
          activeProps={{
            className: cn("bg-[var(--pr-action-soft)] font-semibold border-l-2 border-accent"),
            // Inline semantic value wins over the intentionally translucent shell
            // label utility; both Tailwind arbitrary-color classes otherwise have
            // equal specificity and stylesheet order can select the wrong one.
            style: { color: "var(--pr-ink)" },
          }}
        >
          <Icon className="size-4 shrink-0" aria-hidden />
          {label}
        </Link>
      ))}
    </nav>
  );
}

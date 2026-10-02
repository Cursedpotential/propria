// Byline: Claude Code · Sonnet · 2026-10-02
// Slim mobile Probata: the live case header, four tabs (Imported, Calls, Search, Review) and the page.
// Step 5 of the six steps (Preview). Read-only except the existing Review decision.
import { QueryClientProvider, useQuery } from "@tanstack/react-query";
import { Outlet } from "@tanstack/react-router";
import { ClipboardCheck, Inbox, Phone, Search } from "lucide-react";

import { CaseHeader } from "@/components/mobile/case-header";
import { ThemeProvider } from "@/components/layout/theme-provider";
import { importedApi } from "@/lib/imported-client";
import { queryClient } from "@/lib/query-client";
import { AppLink, useCurrentPath } from "@/lib/router-compat";
import { cn } from "@/lib/utils";

const TABS = [
  { href: "/m", match: (path: string) => path === "/m" || path.startsWith("/m/source") || path.startsWith("/m/thread"), label: "Imported", Icon: Inbox },
  { href: "/m/calls", match: (path: string) => path.startsWith("/m/calls"), label: "Calls", Icon: Phone },
  { href: "/m/search", match: (path: string) => path.startsWith("/m/search"), label: "Search", Icon: Search },
  { href: "/m/review", match: (path: string) => path.startsWith("/m/review"), label: "Review", Icon: ClipboardCheck },
] as const;

function TabBar() {
  const path = useCurrentPath().replace(/\/+$/, "") || "/";
  const queue = useQuery({ queryKey: ["m-review-queue"], queryFn: ({ signal }) => importedApi.reviewQueue(signal), staleTime: 30_000 });
  const waiting = queue.data?.total ?? 0;
  return (
    <nav
      aria-label="Sections"
      className="fixed inset-x-0 bottom-0 z-30 grid grid-cols-4 border-t border-border bg-card pb-[env(safe-area-inset-bottom)]"
    >
      {TABS.map(({ href, match, label, Icon }) => {
        const active = match(path);
        return (
          <AppLink
            key={href}
            href={href}
            aria-current={active ? "page" : undefined}
            className={cn(
              "relative flex min-h-16 flex-col items-center justify-center gap-1 text-xs font-semibold active:bg-muted",
              active ? "text-primary" : "text-muted-foreground",
            )}
          >
            <Icon className="size-6" />
            {label}
            {label === "Review" && waiting > 0 ? (
              <span className="absolute right-[calc(50%-1.6rem)] top-1.5 min-w-5 rounded-full bg-destructive px-1.5 text-center text-[11px] font-bold leading-5 text-white">
                {waiting}
              </span>
            ) : null}
          </AppLink>
        );
      })}
    </nav>
  );
}

export function MobileShell() {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <div
          className="min-h-[100dvh] bg-background text-foreground"
          style={{ "--m-header-h": "calc(3.5rem + env(safe-area-inset-top))" } as React.CSSProperties}
        >
          <CaseHeader />
          <main className="mx-auto w-full max-w-3xl pb-[calc(5rem+env(safe-area-inset-bottom))]">
            <Outlet />
          </main>
          <TabBar />
        </div>
      </ThemeProvider>
    </QueryClientProvider>
  );
}

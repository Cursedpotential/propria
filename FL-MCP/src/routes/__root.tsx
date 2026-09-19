// Byline: Claude Code · Sonnet 5 · 2026-09-07
import type { QueryClient } from "@tanstack/react-query";
import { createRootRouteWithContext, Link } from "@tanstack/react-router";
import { AppShell } from "@/components/layout/app-shell";

export interface RouterContext {
  queryClient: QueryClient;
}

export const Route = createRootRouteWithContext<RouterContext>()({
  component: RootComponent,
  notFoundComponent: NotFound,
});

function RootComponent() {
  return <AppShell />;
}

function NotFound() {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-2 text-text-secondary">
      <p className="text-sm">Route not found.</p>
      <Link to="/" className="text-sm text-accent-text underline underline-offset-2">
        Back to Case Status
      </Link>
    </div>
  );
}

// Byline: Claude Code · Sonnet 5 · 2026-09-07
import { QueryClient } from "@tanstack/react-query";
import { createRouter } from "@tanstack/react-router";
import { routeTree } from "./routeTree.gen";

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 10_000,
      retry: 1,
    },
  },
});

// router-default-options: sensible global defaults (TanStack Router best practice).
export const router = createRouter({
  routeTree,
  context: { queryClient },
  defaultPreload: "intent",
  defaultPreloadStaleTime: 0,
  scrollRestoration: true,
  defaultErrorComponent: ({ error }) => (
    <div className="p-4 text-sm text-critical-text">
      <p className="font-medium">Something went wrong.</p>
      <pre className="mt-2 whitespace-pre-wrap font-mono text-xs">{error instanceof Error ? error.message : String(error)}</pre>
    </div>
  ),
});

// ts-register-router: register the router type for global type inference
// across every `Link`/`useNavigate`/`useSearch` call in the app.
declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}

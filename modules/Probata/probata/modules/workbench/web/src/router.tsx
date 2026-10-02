// Byline: Codex · GPT-5.6-Sol · 2026-08-30
// Byline: Claude Code · Sonnet · 2026-10-02 (desktop and /m mobile shells as sibling layouts)
import {
  createRootRoute,
  createRoute,
  createRouter,
  lazyRouteComponent,
  Outlet,
} from "@tanstack/react-router";

import { AppShell } from "@/app-shell";
import HomePage from "@/app/page";
import { MobileShell } from "@/components/mobile/mobile-shell";

// The root only routes: the desktop workbench (AppShell) and the slim mobile shell (/m) are siblings.
function NotFound() {
  return (
    <section className="mx-auto grid min-h-full max-w-3xl place-content-center px-6 py-16 text-center">
      <p className="platform-kicker">Unknown destination</p>
      <h1 className="mt-3 text-3xl font-semibold tracking-tight">This Workbench route does not exist.</h1>
      <a className="mt-6 text-sm font-semibold text-primary underline underline-offset-4" href="/">
        Return to the Context Intake Desk
      </a>
    </section>
  );
}

const rootRoute = createRootRoute({
  component: Outlet,
  notFoundComponent: NotFound,
});

const desktopRoute = createRoute({ getParentRoute: () => rootRoute, id: "desktop", component: AppShell, notFoundComponent: NotFound });

function applicationRoute(path: string, importer: () => Promise<{ default: React.ComponentType }>) {
  return createRoute({
    getParentRoute: () => desktopRoute,
    path,
    component: lazyRouteComponent(importer),
  });
}

// Slim mobile Probata (Claude Code · Sonnet · 2026-10-02): Imported, Calls, Search and Review under the
// live case header. Step 5 of the six steps (Preview); read-only except the existing Review decision.
const mobileRoute = createRoute({ getParentRoute: () => rootRoute, path: "m", component: MobileShell });

function mobileChild(path: string, importer: () => Promise<{ default: React.ComponentType }>) {
  return createRoute({ getParentRoute: () => mobileRoute, path, component: lazyRouteComponent(importer) });
}

const mobileTree = mobileRoute.addChildren([
  mobileChild("/", () => import("@/app/m/sources-page")),
  mobileChild("source/$sourceId", () => import("@/app/m/source-page")),
  mobileChild("thread/$threadId", () => import("@/app/m/thread-page")),
  mobileChild("calls", () => import("@/app/m/calls-page")),
  mobileChild("unknown", () => import("@/app/m/unknown-page")),
  mobileChild("search", () => import("@/app/m/search-page")),
  mobileChild("review", () => import("@/app/m/review-page")),
  mobileChild("review/$handle", () => import("@/app/m/review-detail-page")),
]);

const desktopTree = desktopRoute.addChildren([
  createRoute({ getParentRoute: () => desktopRoute, path: "/", component: HomePage }),
  // The Case page over registry, the one identity store (Claude Code · Opus 5.5 · 2026-10-01).
  applicationRoute("case", () => import("@/app/case/page")),
  applicationRoute("classification-test", () => import("@/app/classification-test/page")),
  applicationRoute("copilot", () => import("@/app/copilot/page")),
  applicationRoute("evidence-queue", () => import("@/app/evidence-queue/page")),
  applicationRoute("review", () => import("@/app/evidence/preview/page")),
  // Preserve old deep links while Review becomes the canonical destination.
  applicationRoute("evidence/preview", () => import("@/app/evidence/preview/page")),
  applicationRoute("intake", () => import("@/app/intake/page")),
  applicationRoute("knowledge", () => import("@/app/knowledge/page")),
  applicationRoute("matter", () => import("@/app/matter/page")),
  applicationRoute("records", () => import("@/app/records/page")),
  applicationRoute("repairs", () => import("@/app/repairs/page")),
  applicationRoute("runs", () => import("@/app/runs/page")),
  applicationRoute("schemas", () => import("@/app/schemas/page")),
  // Sources replaces Intake as the front door (ratified 2026-09-22). /intake
  // stays reachable as a deep link while Activity and Read land.
  applicationRoute("sources", () => import("@/app/sources/page")),
  applicationRoute("surreal", () => import("@/app/surreal/page")),
  applicationRoute("tools", () => import("@/app/tools/page")),
  // Unnamed numbers (placeholder people), most frequent first (Claude Code · Sonnet · 2026-10-02).
  applicationRoute("unknown-numbers", () => import("@/app/unknown-numbers/page")),
]);

const routeTree = rootRoute.addChildren([desktopTree, mobileTree]);

export const router = createRouter({
  routeTree,
  defaultPreload: "intent",
  scrollRestoration: true,
});

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}

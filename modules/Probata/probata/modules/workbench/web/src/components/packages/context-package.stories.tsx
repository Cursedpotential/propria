// Byline: Codex · GPT-6 · 2026-10-12. Fixtures are synthetic and confined to Storybook.
import type { Meta, StoryObj } from "@storybook/react-vite";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createMemoryHistory, createRootRoute, createRoute, createRouter, RouterProvider } from "@tanstack/react-router";
import { useState } from "react";
import { expect, userEvent, within } from "storybook/test";
import { CategoryLanding } from "@/components/packages/category-landing";
import { ContextPackageWorkspace } from "@/components/packages/context-package-workspace";
import type { ContextPackageItem, ContextPackageSnapshot } from "@/lib/context-package-client";
import { useBrowserSearchParams } from "@/lib/router-compat";

const HANDLE = "storybook-package";
const SOURCE = { source_ref: "fixture://complete-document", source_version_ref: "fixture-version-1", locator: { page: 2, start: 10, end: 25 } };
const item: ContextPackageItem = {
  id: "statement-1", kind: "statement", label: "Synthetic extracted statement", revision: "r1", review_state: "pending",
  fields: { text: "The complete fixture document text remains readable, with exact source coordinates.", provenance_party: "unknown", basis: "Not established by this fixture", source_account: "account-1", source_device: "device-1", person_ref: null },
  source_refs: [SOURCE], edges: [{ edge_id: "edge-1", kind: "located_at", from_node_id: "statement-1", to_node_id: "locator-1", source_pins: [SOURCE] }],
  capabilities: ["edit", "accept", "reject"].map((operation) => ({ operation, available: true, path: `/api/context/packages/${HANDLE}/review`, method: "POST" })),
  placement: { domain: "Fixture domain", category: "Fixture category", path: ["Fixture domain", "Fixture category"], basis: "Persisted synthetic source placement" },
};
let serverItem: ContextPackageItem;
let saveFailure = false;

/** Install bounded synthetic API responses for the actual package components.
 * Input: Storybook scenario args. Output: fetch cleanup. Effects: story-only HTTP fixture.
 * Pick for visual/interaction checks; these fixtures are never included in product routes.
 */
function installApi({ args }: { args: { failSave?: boolean; reopened?: boolean } }) {
  serverItem = structuredClone(item); saveFailure = Boolean(args.failSave);
  if (args.reopened) {
    serverItem.revision = "r2";
    serverItem.fields.provenance_party = "first_party";
    serverItem.review_history = [{ revision: "r2", operation: "edit", fields: { provenance_party: "first_party" }, reason: "Synthetic saved correction", original_fields: item.fields }];
  }
  const previous = globalThis.fetch;
  globalThis.fetch = async (input, init) => {
    const raw = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
    const url = new URL(raw, "https://storybook.local");
    let body: unknown;
    let status = 200;
    if (url.pathname === "/api/context/categories") body = { items: [{ id: "fixture-category", label: "Fixture category", category: "Fixture category", domain: "Fixture domain", path: ["Fixture domain", "Fixture category"], classification_basis: "Persisted synthetic source placement", count: 1 }], next_cursor: null };
    else if (url.pathname === "/api/context/packages") body = { items: [{ handle: HANDLE, package_ref: "fixture-package-1", source_ref: SOURCE.source_ref, source_version_ref: SOURCE.source_version_ref, source_type: "document", label: "Complete fixture document", revision: serverItem.revision, placement: serverItem.placement }], next_cursor: null };
    else if (url.pathname === `/api/context/packages/${HANDLE}`) body = {
      package_ref: "fixture-package-1", source_ref: SOURCE.source_ref, source_version_ref: SOURCE.source_version_ref, source_type: "document", revision: serverItem.revision,
      metadata: { title: "Synthetic complete document", original_name: "fixture.txt" }, placement: serverItem.placement,
      sections: [{ id: "statements", label: "Statements", count: 1, available: true }, { id: "locations", label: "Locations", count: 0, available: false, reason: "No location artifact recorded" }], items: [serverItem], next_cursor: null,
    } satisfies ContextPackageSnapshot;
    else if (url.pathname === `/api/context/packages/${HANDLE}/review`) {
      if (saveFailure) { status = 503; body = { detail: "Synthetic save failure: the correction was not saved." }; }
      else {
        const request = JSON.parse(String(init?.body)) as { fields?: Record<string, unknown>; operation: string; reason: string; revision: string };
        if (request.revision !== serverItem.revision) { status = 409; body = { detail: "Synthetic stale revision" }; }
        else {
          serverItem = { ...serverItem, revision: `r${Number(String(serverItem.revision).slice(1)) + 1}`, fields: { ...serverItem.fields, ...request.fields }, review_history: [...(serverItem.review_history as unknown[] ?? []), request], review_state: request.operation === "accept" ? "accepted" : request.operation === "reject" ? "rejected" : serverItem.review_state };
          body = { saved: true, revision: serverItem.revision, operation: request.operation, source_version_ref: SOURCE.source_version_ref };
        }
      }
    } else { status = 404; body = { detail: `Unmocked Storybook request: ${url.pathname}` }; }
    return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
  };
  return () => { globalThis.fetch = previous; };
}

/** Render the real landing/workspace through a memory router and fresh query cache.
 * Input: scenario entry URL. Output: actual components. Effects: fixture reads/mutations.
 * Pick for dirty navigation, save-failure and reopen stories; no substitute product controls.
 */
function Scenario({ entry }: { entry: string }) {
  const [queryClient] = useState(() => new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } }));
  const [router] = useState(() => {
    const root = createRootRoute();
    const route = createRoute({ getParentRoute: () => root, path: "/read", component: StoryRoute });
    return createRouter({ routeTree: root.addChildren([route]), history: createMemoryHistory({ initialEntries: [entry] }) });
  });
  return <QueryClientProvider client={queryClient}><RouterProvider router={router} /></QueryClientProvider>;
}

/** Select only real package components from the story route.
 * Input: memory route. Output: landing or package. Effects: child fixture requests.
 * Pick inside this story harness without the unrelated app-shell case service.
 */
function StoryRoute() {
  const params = useBrowserSearchParams();
  return params.get("package") ? <ContextPackageWorkspace handle={params.get("package")!} /> : <CategoryLanding />;
}

const meta = {
  title: "Context review/Extracted package", component: Scenario, beforeEach: installApi,
  args: { entry: `/read?package=${HANDLE}`, failSave: false, reopened: false },
  parameters: { docs: { description: { component: "Actual package components with synthetic bounded API responses. Saves, failures, dirty navigation and reopened corrections exercise the production editor and query hooks." } } },
} satisfies Meta<React.ComponentProps<typeof Scenario> & { failSave?: boolean; reopened?: boolean }>;
export default meta;
type Story = StoryObj<typeof meta>;

export const CategoryHeaders: Story = { args: { entry: "/read" } };
export const CompletePackage: Story = {};
export const ReopenedCorrection: Story = { args: { reopened: true } };
export const SaveFailureRetainsDraft: Story = {
  args: { failSave: true },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await canvas.findByRole("heading", { name: "Synthetic extracted statement" });
    await userEvent.click(canvas.getByText("Supplement or edit fields"));
    await userEvent.click(canvas.getByLabelText("Supplemental fields JSON"));
    await userEvent.paste('{"provenance_party":"first_party"}');
    await userEvent.type(canvas.getByLabelText("Review reason"), "Synthetic correction");
    await userEvent.click(canvas.getByRole("button", { name: "Save supplemental fields" }));
    await canvas.findByText("Synthetic save failure: the correction was not saved.");
    await expect(canvas.getByLabelText("Supplemental fields JSON")).toHaveValue('{"provenance_party":"first_party"}');
  },
};
export const DirtyNavigation: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await canvas.findByRole("heading", { name: "Synthetic extracted statement" });
    await userEvent.type(canvas.getByLabelText("Review reason"), "Unfinished synthetic review");
    await userEvent.click(canvas.getByRole("link", { name: "Context review categories" }));
    const screen = within(canvasElement.ownerDocument.body);
    await screen.findByRole("dialog");
    await userEvent.click(screen.getByRole("button", { name: "Stay here" }));
    await expect(canvas.getByLabelText("Review reason")).toHaveValue("Unfinished synthetic review");
  },
};

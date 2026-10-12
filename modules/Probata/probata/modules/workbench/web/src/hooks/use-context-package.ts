// Byline: Codex · GPT-6 · 2026-10-12.
import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { getContextCategories, getContextPackage, getContextPackages, reviewContextPackage, type PackageReviewInput } from "@/lib/context-package-client";

/** Page the real Case Vault category inventory without preloading package contents.
 * Inputs: none. Output: category query. Effects: bounded abortable GETs.
 * Pick for the default Context review landing, before file selection.
 */
export function useContextCategories() {
  return useInfiniteQuery({ queryKey: ["context-categories"], initialPageParam: "", queryFn: ({ pageParam, signal }) => getContextCategories(pageParam, signal), getNextPageParam: (page) => page.next_cursor || undefined, retry: false });
}

/** Page intact packages under one persisted category header.
 * Inputs: category/domain identity. Output: package query. Effects: abortable GETs.
 * Pick per visible category; cursors are never shared across categories.
 */
export function useContextPackages(category: string, domain: string) {
  return useInfiniteQuery({ queryKey: ["context-packages", domain, category], initialPageParam: "", queryFn: ({ pageParam, signal }) => getContextPackages(category, domain, pageParam, signal), getNextPageParam: (page) => page.next_cursor || undefined, retry: false });
}

/** Page one dynamic package section with its own source-bound cursor history.
 * Inputs: native handle and section. Output: exact package query and first-page reset.
 * Effects: abortable GETs; retains at most four explicitly requested section pages.
 * Pick for grid/timeline/relations views over the same items; no load-all loop.
 */
export function useContextPackage(handle: string, section: string) {
  const client = useQueryClient();
  const queryKey = ["context-package", handle, section];
  const query = useInfiniteQuery({ queryKey, initialPageParam: "", queryFn: ({ pageParam, signal }) => getContextPackage(handle, section, pageParam, signal), getNextPageParam: (page) => page.next_cursor || undefined, maxPages: 4, gcTime: 60000, enabled: Boolean(handle), retry: false, refetchOnWindowFocus: false });
  return { ...query, restart: () => client.resetQueries({ queryKey, exact: true }) };
}

/** Save one explicit review and invalidate only this package plus placement listings.
 * Inputs: native handle. Output: mutation accepting stable action key and revision.
 * Effects: POST and scoped readback invalidation. Pick for corrections and decisions;
 * failures retain the editor draft and do not optimistically claim a saved result.
 */
export function useContextPackageReview(handle: string) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ input, key }: { input: PackageReviewInput; key: string }) => reviewContextPackage(handle, input, key),
    retry: false,
    onSuccess: async (_receipt, { input }) => {
      await client.invalidateQueries({ queryKey: ["context-package", handle] });
      if (["metadata", "placement"].includes(input.operation) || Object.keys(input.fields ?? {}).some((field) => /domain|category|placement|path/.test(field))) {
        await Promise.all([client.invalidateQueries({ queryKey: ["context-categories"] }), client.invalidateQueries({ queryKey: ["context-packages"] })]);
      }
    },
  });
}

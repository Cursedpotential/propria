import type { Surface } from "@/lib/surfaces";

/** Stable workdesk groups. Keep this order independent of case phase or route rank. */
export const TASK_NAVIGATION_GROUPS = [
  "Case overview",
  "Reference library",
  "Documents and drafting",
  "Evidence",
  "Case strategy",
  "Legal research and analysis",
  "Court and calendar",
  "Skills and processes",
  "Digital firm and tools",
] as const;

export type TaskNavigationGroup = (typeof TASK_NAVIGATION_GROUPS)[number];

// These paths reflect the current common-surface plan. Unknown catalog entries
// remain reachable through the digital-firm bucket until they receive an intentional home.
const GROUP_PATHS: Record<TaskNavigationGroup, readonly string[]> = {
  "Case overview": ["/"],
  "Reference library": ["/case-search", "/laws", "/external-sources"],
  "Documents and drafting": [
    "/documents",
    "/file-tools",
    "/confidentiality-check",
    "/drafts",
    "/agreements",
    "/review",
    "/final-copy",
    "/filing-checklist",
  ],
  Evidence: [
    "/evidence-catalog",
    "/evidence",
    "/missing-evidence",
    "/evidence-requests",
  ],
  "Case strategy": ["/private-notes", "/challenge-draft"],
  "Legal research and analysis": [
    "/citation-check",
    "/open-questions",
    "/questions",
    "/custody-factors",
    "/analysis-queue",
  ],
  "Court and calendar": ["/calendar", "/timeline", "/tasks"],
  "Skills and processes": ["/templates", "/playbooks"],
  "Digital firm and tools": [
    "/assistant",
    "/scheduled-jobs",
    "/notices",
    "/activity-log",
    "/assistant-log",
  ],
};

const PATH_TO_GROUP = new Map<string, TaskNavigationGroup>(
  Object.entries(GROUP_PATHS).flatMap(([group, paths]) =>
    paths.map((path) => [path, group as TaskNavigationGroup] as const),
  ),
);

export function taskNavigationGroupForPath(path: string): TaskNavigationGroup {
  return PATH_TO_GROUP.get(path) ?? "Digital firm and tools";
}

export function taskNavigationGroupForSurface(item: Surface): TaskNavigationGroup {
  return taskNavigationGroupForPath(item.path);
}

export function groupSurfacesByTaskNavigation(
  surfaces: Surface[],
): Array<[TaskNavigationGroup, Surface[]]> {
  const groups = new Map<TaskNavigationGroup, Surface[]>();
  for (const group of TASK_NAVIGATION_GROUPS) groups.set(group, []);
  for (const item of surfaces) groups.get(taskNavigationGroupForSurface(item))?.push(item);
  for (const label of ["Documents and drafting", "Evidence"] as const) {
    groups.get(label)?.sort((a, b) => GROUP_PATHS[label].indexOf(a.path) - GROUP_PATHS[label].indexOf(b.path));
  }
  return TASK_NAVIGATION_GROUPS
    .map((group) => [group, groups.get(group) ?? []] as [TaskNavigationGroup, Surface[]])
    .filter(([, items]) => items.length > 0);
}

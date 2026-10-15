import type { SourceUnitKind, SourceUnitMark } from "@/lib/shared/types";

/** Describe each selected folder under its actual source root without combining units.
 * Inputs: allowlisted root locator and relative folder prefixes. Output: exact root
 * plans and an overlap issue. Effects: none; choose before package processing or display.
 */
export function folderSelectionPlan(rootRef: string, folders: Iterable<string>) {
  const prefixes = [...new Set(folders)];
  const units = prefixes.map((prefix) => ({ prefix, name: prefix.replace(/\/$/, "").split("/").at(-1) ?? prefix,
    sourceRef: rootRef ? `${rootRef.replace(/\/$/, "")}/${prefix.replace(/^\//, "")}` : null }));
  const overlap = prefixes.some((left, index) => prefixes.some((right, other) => index !== other && right.startsWith(left.endsWith("/") ? left : `${left}/`)));
  const parents = prefixes.map((prefix) => prefix.replace(/\/$/, "").split("/").slice(0, -1));
  const common = (parents[0] ?? []).filter((part, index) => parents.every((parts) => parts[index] === part && parts.slice(0, index).every((value, previous) => value === parents[0][previous])));
  const folderRef = rootRef ? `${rootRef.replace(/\/$/, "")}/${common.length ? `${common.join("/")}/` : ""}` : null;
  return { units, folderRef, issue: prefixes.length > 32 ? "Choose up to 32 folder units in one batch." : overlap ? "Selected folders overlap. Choose the parent folder or its child folders so the same files are not processed twice." : prefixes.length && !rootRef ? "Loading the selected source location." : null };
}

/** Record one existing unit mark per selected folder and retain partial successes.
 * Inputs: separate folder roots, supported kind and existing mark API callback.
 * Output: successful marks and per-folder failures. Effects: only callback writes;
 * choose for an explicit bulk mark, never to create one combined parent unit.
 */
export async function markFolderUnits(folders: readonly string[], kind: SourceUnitKind, confirm: boolean,
  record: (request: { unit_root: string; unit_type: SourceUnitKind; confirm: boolean; label: string }) => Promise<SourceUnitMark>) {
  const marks: SourceUnitMark[] = [];
  const failures: string[] = [];
  for (const folder of [...new Set(folders)]) {
    const label = folder.replace(/\/$/, "").split("/").at(-1) ?? folder;
    try { marks.push(await record({ unit_root: folder, unit_type: kind, confirm, label })); }
    catch (error) { failures.push(`${label}: ${error instanceof Error ? error.message : "Could not save the unit mark"}`); }
  }
  return { marks, failures };
}

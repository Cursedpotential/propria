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
export async function markFolderUnits(rootId: string, folders: readonly string[], kind: SourceUnitKind, confirm: boolean,
  record: (request: { root_id: string; unit_root: string; unit_type: SourceUnitKind; confirm: boolean; label: string }) => Promise<SourceUnitMark>) {
  const marks: SourceUnitMark[] = [];
  const failures: string[] = [];
  for (const folder of [...new Set(folders)]) {
    const label = folder.replace(/\/$/, "").split("/").at(-1) ?? folder;
    try { marks.push(await record({ root_id: rootId, unit_root: folder, unit_type: kind, confirm, label })); }
    catch (error) { failures.push(`${label}: ${error instanceof Error ? error.message : "Could not save the unit mark"}`); }
  }
  return { marks, failures };
}

/** Identify saved marks by their recorded storage location and relative folder.
 * Input: saved mark. Output: exact identity. Effects: none; choose when merging
 * mark responses so another root or a legacy mark can never be replaced locally.
 */
export function unitMarkIdentity(mark: SourceUnitMark): string {
  return JSON.stringify([mark.identity_scope, mark.root_id, mark.source_root_ref, mark.unit_root.replace(/\/$/, "")]);
}

/** Partition marks by exact current root without assigning legacy records a location.
 * Inputs: marks and configured root identity. Output: applicable and legacy marks.
 * Effects: none; choose for folder badges and explicitly unscoped historical details.
 */
export function sourceRootMarks(marks: readonly SourceUnitMark[], rootId: string, rootRef: string) {
  return {
    scoped: marks.filter((mark) => mark.identity_scope === "root_scoped" && mark.root_id === rootId && mark.source_root_ref === rootRef),
    legacy: marks.filter((mark) => mark.identity_scope === "legacy_unscoped"),
  };
}

/** Toggle inclusion while inspecting a remaining included folder when possible.
 * Inputs: selected roots, toggled folder and checkbox state. Output: selection
 * and inspected folder. Effects: none; choose for checkbox changes, including zero selection.
 */
export function toggleFolderSelection(current: ReadonlySet<string>, folder: string, checked: boolean) {
  const folders = new Set(current);
  if (checked) folders.add(folder); else folders.delete(folder);
  return { folders, inspected: checked ? folder : [...folders].at(-1) ?? folder };
}

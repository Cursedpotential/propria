import { humanPreviewHandle, isHumanPreviewHandle } from "@/lib/human-context";

/** Resolve native chunk citations to source readers without interpreting chunk IDs as message or thread IDs.
 * Inputs: indexed origin, optional explicit handle and actual source-version coordinates.
 * Outputs: native flag and unique validated human handles. Effects: none.
 * Choose for imported and combined search; native origin alone never enables a legacy thread fallback.
 */
export function nativeSearchTargets(input: {
  origin_system?: string | null;
  preview_handle?: string | null;
  source_version_ref?: string | null;
  source_version_id?: string | null;
  source_version_ids?: string[];
}): { native: boolean; handles: string[] } {
  const explicit = input.preview_handle ?? "";
  const native = input.origin_system === "probata-native-context" || explicit.startsWith("human-");
  if (!native) return { native: false, handles: [] };
  const versions = [...new Set([input.source_version_ref, input.source_version_id, ...(input.source_version_ids ?? [])]
    .filter((version): version is string => Boolean(version)))];
  if (explicit) {
    return { native: true, handles: isHumanPreviewHandle(explicit) && (!versions.length || versions.some((version) => humanPreviewHandle(version) === explicit)) ? [explicit] : [] };
  }
  return { native: true, handles: versions.flatMap((version) => {
    const handle = humanPreviewHandle(version);
    return handle ? [handle] : [];
  }) };
}

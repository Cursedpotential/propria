import { humanPreviewHandle, isHumanPreviewHandle } from "@/lib/human-context";

/** Resolve native chunk citations to source readers without interpreting chunk IDs as message or thread IDs.
 * Inputs: indexed origin, optional explicit handle and actual source-version coordinates.
 * Outputs: native flag and unique validated human handles. Effects: none.
 * Choose for imported and combined search; native origin alone never enables a legacy thread fallback.
 * Human readers require an explicit validated handle or the native_human_context corpus; known AI never maps to them.
 */
export function nativeSearchTargets(input: {
  origin_system?: string | null;
  preview_handle?: string | null;
  source_version_ref?: string | null;
  source_version_id?: string | null;
  source_version_ids?: string[];
  corpus?: string | null;
  source_kind?: string | null;
  source_format?: string | null;
  record_kind?: string | null;
}): { native: boolean; handles: string[] } {
  const explicit = input.preview_handle ?? "";
  const aiFormats = new Set(["chatgpt", "claude", "gemini_markdown", "chatgpt_markdown", "claude_markdown", "chatgpt_official_json", "chatgpt_json_array", "chatgpt_conversations_json", "claude_conversations_json", "claude_ai_export_json", "gemini_activity_json", "ai_markdown_transcript", "ai_generic_json", "ai_conversations_json", "ai_chat_file"]);
  const ai = input.source_kind === "ai_chat" || input.record_kind === "ai_chat" || aiFormats.has((input.source_format ?? "").toLowerCase().trim());
  const humanCorpus = input.corpus === "native_human_context";
  const native = input.origin_system === "probata-native-context" || explicit.startsWith("human-") || humanCorpus || ai;
  if (!native) return { native: false, handles: [] };
  if (ai) return { native: true, handles: [] };
  const versions = [...new Set([input.source_version_ref, input.source_version_id, ...(input.source_version_ids ?? [])]
    .filter((version): version is string => Boolean(version)))];
  if (explicit) {
    return { native: true, handles: isHumanPreviewHandle(explicit) && (!versions.length || versions.some((version) => humanPreviewHandle(version) === explicit)) ? [explicit] : [] };
  }
  if (!humanCorpus) return { native: true, handles: [] };
  return { native: true, handles: versions.flatMap((version) => {
    const handle = humanPreviewHandle(version);
    return handle ? [handle] : [];
  }) };
}

// Byline: Codex · GPT-6 · 2026-10-12; extracted from the existing EntityCard donor.
import type { EntityAlias } from "@/lib/entity-extraction-client";
import { cn } from "@/lib/utils";

/** Display a native entity alias with its source-relative identity scope intact.
 * Input: recorded alias text/kind/address scope. Output: existing entity-review label.
 * Effects: none. Pick in both legacy entity review and full package review so a source
 * account/device alias is never silently displayed as a resolved person's identity.
 */
export function EntityAliasLabel({ alias }: { alias: Pick<EntityAlias, "text" | "kind" | "address_kind" | "scope"> }) {
  return <><span className={cn(alias.address_kind && "font-mono")}>{alias.address_kind === "self" ? "self (this source)" : alias.text}</span><span className="text-[10px] text-muted-foreground">{alias.kind}</span></>;
}

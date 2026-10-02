// Byline: Claude Code · Sonnet · 2026-10-02
// Desktop thread and calls views read previews, not the registry, so they ask the Workbench whether a
// number is a named person, a placeholder or nobody yet, and offer "Who is this?" unless it is named.
import { useQuery } from "@tanstack/react-query";

import { WhoIsThis } from "@/components/identity/who-is-this";
import { importedApi } from "@/lib/imported-client";

const PHONE = /^\+?[\d\s().-]{10,16}$/;

export function looksLikePhone(value: string | null | undefined): value is string {
  if (!value || !PHONE.test(value)) return false;
  const digits = value.replace(/\D/g, "");
  return digits.length === 10 || (digits.length === 11 && digits.startsWith("1"));
}

/** The name the registry holds for this number, and the control to give it one when it has none. */
export function NumberIdentity({ value, context }: { value: string | null | undefined; context: string }) {
  const phone = looksLikePhone(value) ? value : null;
  const status = useQuery({
    queryKey: ["number-status", phone],
    queryFn: ({ signal }) => importedApi.numberStatus([phone as string], signal),
    enabled: phone !== null,
    staleTime: 30_000,
  });
  const item = phone ? status.data?.items[phone] : undefined;
  if (!phone || !item) return null;
  if (item.state === "known") return <span className="text-xs text-muted-foreground">{item.label}</span>;
  return <WhoIsThis number={item.number} entityId={item.entity_id} context={context} />;
}

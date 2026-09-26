import { EventList, type AuditEvent } from "@/components/EventList";
import { legalApiBase } from "@/lib/api/client";

// Byline: Grok · grok-4.6 · 2026-08-18

export default async function AuditLogPage() {
  let events: AuditEvent[] = [];
  let error: string | null = null;
  try {
    const response = await fetch(`${legalApiBase()}/v1/audit`, { cache: "no-store" });
    if (!response.ok) throw new Error(`legal-api audit ${response.status}`);
    events = await response.json();
  } catch (exc) {
    error = exc instanceof Error ? exc.message : "legal-api unreachable";
  }

  return (
    <>
      <h1 className="legal">Activity log</h1>
      <p>Review recorded workspace events and their current status.</p>
      {error ? <p className="unsupported">{error}</p> : null}
      <EventList events={events} />
    </>
  );
}

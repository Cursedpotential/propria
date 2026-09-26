import { EventList, type AuditEvent } from "@/components/EventList";
import { legalApiBase } from "@/lib/api/client";

// Byline: Grok · grok-4.6 · 2026-08-18

export default async function InboundNoticesPage() {
  let events: AuditEvent[] = [];
  let error: string | null = null;
  try {
    const response = await fetch(`${legalApiBase()}/v1/triggers`, { cache: "no-store" });
    if (!response.ok) throw new Error(`legal-api triggers ${response.status}`);
    events = await response.json();
  } catch (exc) {
    error = exc instanceof Error ? exc.message : "legal-api unreachable";
  }

  return (
    <>
      <h1 className="legal">Inbound notices</h1>
      <p>
        Review inbound notices recorded by this workspace. Process opens the related page; IMAP is not connected.
      </p>
      {error ? <p className="unsupported">{error}</p> : null}
      <EventList events={events} />
    </>
  );
}

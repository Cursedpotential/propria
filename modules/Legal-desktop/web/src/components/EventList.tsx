// Byline: Grok · grok-4.6 · 2026-08-18

export type AuditEvent = {
  at?: string;
  action?: string;
  [key: string]: unknown;
};

export function EventList({ events }: { events: AuditEvent[] }) {
  if (!events.length) return <p className="muted">Nothing recorded yet.</p>;
  return (
    <table className="work-sheet">
      <thead>
        <tr>
          <th>When</th>
          <th>What happened</th>
        </tr>
      </thead>
      <tbody>
        {events.map((item, index) => (
          <tr key={`${item.at ?? "row"}-${index}`}>
            <td>{item.at ? new Date(item.at).toLocaleString("en-US", { timeZone: "America/New_York" }) : "—"}</td>
            <td>{String(item.action ?? "event")}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

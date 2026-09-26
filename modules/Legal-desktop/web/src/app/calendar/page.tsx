// Byline amendment: Codex · GPT-5 · 2026-09-12 (Propria semantic token adoption)
import { DocketForm } from "@/components/DocketForm";
import { EventDelete } from "@/components/EventDelete";
import { legalApiBase } from "@/lib/api/client";

type EventRow = {
  event_id: string;
  occurs_at: string;
  title: string;
  kind: string;
  location: string;
  confirmed: boolean;
};

function monthGrid(year: number, month: number) {
  const first = new Date(year, month, 1);
  const start = first.getDay();
  const days = new Date(year, month + 1, 0).getDate();
  const cells: Array<number | null> = [];
  for (let i = 0; i < start; i += 1) cells.push(null);
  for (let day = 1; day <= days; day += 1) cells.push(day);
  return cells;
}

function shiftMonth(year: number, month: number, delta: number) {
  const next = new Date(year, month + delta, 1);
  return { year: next.getFullYear(), month: next.getMonth() };
}

export default async function CalendarPage({
  searchParams,
}: {
  searchParams: Promise<{ year?: string; month?: string }>;
}) {
  let events: EventRow[] = [];
  let error: string | null = null;
  try {
    const response = await fetch(`${legalApiBase()}/v1/docket-events`, { cache: "no-store" });
    if (!response.ok) throw new Error(`legal-api docket ${response.status}`);
    events = await response.json();
  } catch (exc) {
    error = exc instanceof Error ? exc.message : "legal-api unreachable";
  }

  const now = new Date();
  const params = await searchParams;
  const year = Number(params.year) || now.getFullYear();
  const month = params.month !== undefined ? Number(params.month) : now.getMonth();
  const prev = shiftMonth(year, month, -1);
  const next = shiftMonth(year, month, 1);
  const cells = monthGrid(year, month);
  const label = new Date(year, month, 1).toLocaleString("en-US", {
    month: "long",
    year: "numeric",
    timeZone: "America/New_York",
  });

  return (
    <>
      <p className="section-eyebrow">Operations</p>
      <p className="muted">Court dates</p>
      <h1 style={{ fontFamily: "Georgia, serif", fontWeight: 500 }}>{label}</h1>
      <p>
        Enter and review the court dates recorded for this matter.
      </p>
      <p>
        <a href={`/calendar?year=${prev.year}&month=${prev.month}`}>Previous</a>
        {" · "}
        <a href={`/calendar?year=${now.getFullYear()}&month=${now.getMonth()}`}>This month</a>
        {" · "}
        <a href={`/calendar?year=${next.year}&month=${next.month}`}>Next</a>
        {" · "}
        <a href="/timeline">Timeline</a>
      </p>
      {error ? <p>{error}</p> : null}
      <DocketForm />
      <div
        style={{
          display: "grid",
          gridTemplateColumns: "repeat(7, 1fr)",
          gap: 8,
        }}
      >
        {["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"].map((day) => (
          <div key={day} style={{ color: "var(--text-muted)", fontSize: 12 }}>
            {day}
          </div>
        ))}
        {cells.map((day, index) => {
          const dayEvents =
            day === null
              ? []
              : events.filter((item) => {
                  const when = new Date(item.occurs_at);
                  return (
                    when.getFullYear() === year &&
                    when.getMonth() === month &&
                    when.getDate() === day
                  );
                });
          const isToday =
            day === now.getDate() && year === now.getFullYear() && month === now.getMonth();
          return (
            <div
              key={`${day ?? "e"}-${index}`}
              style={{
                minHeight: 88,
                border: "1px solid var(--border)",
                padding: 8,
                background: isToday ? "var(--surface-muted)" : "transparent",
              }}
            >
              <div style={{ color: "var(--text-muted)" }}>{day ?? ""}</div>
              {dayEvents.map((item) => (
                <div key={item.event_id} style={{ fontSize: 12, marginTop: 4 }}>
                  {item.kind}: {item.title}
                </div>
              ))}
            </div>
          );
        })}
      </div>
      <h2>All recorded dates</h2>
      {events.length === 0 ? <p>None yet.</p> : null}
      <ul>
        {events.map((item) => (
          <li key={item.event_id}>
            {new Date(item.occurs_at).toLocaleString("en-US", { timeZone: "America/New_York" })}{" "}
            — [{item.kind}] {item.title}
            {item.location ? ` · ${item.location}` : ""}
            {item.confirmed ? "" : " · unconfirmed"}
            <EventDelete eventId={item.event_id} />
          </li>
        ))}
      </ul>
    </>
  );
}

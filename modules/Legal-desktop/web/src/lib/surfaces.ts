// Byline: Claude Code · Kimi K2.7 · 2026-08-18
// Catalog matches Cat 2 / legal-terminal groups: Assistant, Research, Contracts, Drafting, Operations.
// All paths and labels are plain English — no legal abbreviations or jargon.

export type Surface = {
  path: string;
  label: string;
  group: string;
  help?: string;
  advanced?: boolean;
  icon?: string;
};

export const CASE_PHASE_KEY = "lw-case-phase";

export const CASE_PHASES = ["Discovery", "Motions", "Hearing", "Trial"] as const;

export type CasePhase = (typeof CASE_PHASES)[number];

export const PHASE_COPY: Record<CasePhase, { label: string; help: string }> = {
  Discovery: {
    label: "Information gathering",
    help: "Evidence requests, missing evidence, evidence list, open questions.",
  },
  Motions: {
    label: "Written requests to the court",
    help: "Motion writer, your review, final review copy, filing readiness checklist.",
  },
  Hearing: {
    label: "Hearing prep",
    help: "Court dates, timeline, filing readiness checklist.",
  },
  Trial: {
    label: "Trial prep",
    help: "Evidence list, custody factors, motion writer.",
  },
};

export const SURFACES: Surface[] = [
  { path: "/", label: "Case dashboard", group: "Assistant", help: "See the current matter, document counts, and recent activity.", icon: "home" },
  { path: "/assistant", label: "Ask the assistant", group: "Assistant", help: "Discuss the current page or ask for help with your case work.", icon: "message" },
  { path: "/case-search", label: "Case search", group: "Research", help: "Search CourtListener for published opinions and open the matching source records.", icon: "search" },
  { path: "/laws", label: "Laws", group: "Research", help: "Browse saved Michigan legal authorities and their source links.", icon: "book" },
  { path: "/citation-check", label: "Citation check", group: "Research", help: "Parse a citation into its parts and review its normalized format.", icon: "quote" },
  { path: "/open-questions", label: "Open questions", group: "Research", help: "Record research questions and track their status.", icon: "search" },
  { path: "/questions", label: "Questions the judge decides", group: "Research", help: "Break a case issue into questions and link supporting material.", icon: "scale" },
  { path: "/custody-factors", label: "What the judge must consider", group: "Research", help: "Organize case information around Michigan custody factors.", icon: "list" },
  { path: "/agreements", label: "Document review", group: "Drafting", help: "Read saved drafts side by side and make review notes.", icon: "file" },
  { path: "/documents", label: "Document viewer", group: "Contracts", help: "Open a PDF in the document viewer.", icon: "file" },
  { path: "/file-tools", label: "File tools", group: "Contracts", help: "Inspect file metadata, extract screenshot text, and prepare document conversions.", icon: "search" },
  { path: "/confidentiality-check", label: "Confidentiality check", group: "Contracts", help: "Scan a draft or pasted text for sensitive keywords and review matching excerpts.", icon: "shield" },
  { path: "/drafts", label: "Documents and writing", group: "Drafting", help: "Write, import, and edit office documents with saved versions.", icon: "pen" },
  { path: "/templates", label: "Starting templates", group: "Drafting", help: "Choose a starting outline and create a draft.", icon: "file" },
  { path: "/review", label: "Your review", group: "Drafting", help: "Record approval, rejection, or requested changes for a draft section.", icon: "eye" },
  { path: "/final-copy", label: "Final review copy", group: "Drafting", help: "Create and inspect a fixed review copy and its manifest.", icon: "package" },
  { path: "/filing-checklist", label: "Filing readiness checklist", group: "Drafting", help: "Track the remaining preparation checks for a document.", icon: "clipboard" },
  { path: "/analysis-queue", label: "Analysis queue", group: "Operations", help: "Inspect queued analysis work and its status.", icon: "clock" },
  { path: "/playbooks", label: "Playbooks", group: "Operations", help: "Browse the steps in reusable case-work procedures.", icon: "activity" },
  { path: "/scheduled-jobs", label: "Scheduled jobs", group: "Operations", help: "View scheduled jobs, their last run, and available controls.", icon: "activity" },
  { path: "/notices", label: "Inbound notices", group: "Operations", help: "Read notices received by the workspace.", icon: "mail" },
  { path: "/activity-log", label: "Activity log", group: "Operations", help: "Review recorded actions and changes in the workspace.", icon: "list" },
  { path: "/external-sources", label: "External sources", group: "Operations", help: "View external research services and provider privacy settings.", icon: "activity" },
  { path: "/calendar", label: "Court dates", group: "Operations", help: "Record and review court dates and deadlines.", icon: "calendar" },
  { path: "/evidence-requests", label: "Discovery requests", group: "Operations", help: "Draft requests for production, admissions, interrogatories, and subpoenas.", icon: "folder" },
  { path: "/evidence", label: "Evidence list", group: "Operations", help: "Review case evidence candidates and add annotations.", icon: "paperclip" },
  { path: "/evidence-catalog", label: "Evidence catalog", group: "Operations", help: "Browse catalog records and inspect their evidence-processing status.", icon: "folder" },
  { path: "/missing-evidence", label: "Missing evidence", group: "Operations", help: "Describe missing support and create an investigation request.", icon: "help" },
  { path: "/tasks", label: "Your tasks", group: "Operations", help: "Add personal tasks and track their status.", icon: "check" },
  { path: "/timeline", label: "Timeline", group: "Operations", help: "View recorded case events in date order.", icon: "clock" },
  { path: "/assistant-log", label: "Assistant activity log", group: "Operations", help: "Review assistant run requests and their results.", icon: "activity" },
  { path: "/private-notes", label: "My private notes", group: "Drafting", help: "Read saved private case notes, strategy, and ideas.", advanced: true, icon: "lock" },
  { path: "/challenge-draft", label: "Devil's advocate review", group: "Drafting", help: "Review recorded adversarial findings and weaknesses in a draft.", advanced: true, icon: "swords" },
];

export function navLabel(item: Surface): string {
  return item.label.trim();
}

export function navHelp(item: Surface): string {
  return item.help?.trim() || item.label;
}

export function enrichSurface(row: Surface): Surface {
  const local = SURFACES.find((item) => item.path === row.path);
  return {
    ...local,
    ...row,
    label: ((row.path === "/agreements" && row.label === "Agreement review") || (row.path === "/evidence-requests" && row.label === "Evidence requests")) ? local!.label : row.label || local?.label || row.path,
    help: local?.help || row.help || row.label,
    group: row.group || local?.group || "Assistant",
    advanced: row.advanced ?? local?.advanced ?? false,
    icon: row.icon || local?.icon,
  };
}

export function enrichCatalog(rows: Surface[]): Surface[] {
  return rows.map(enrichSurface);
}

let catalog: Surface[] = SURFACES;

export function setSurfaceCatalog(rows: Surface[]): void {
  if (rows.length) catalog = enrichCatalog(rows);
}

export function surfaceCatalog(): Surface[] {
  return catalog;
}

export function pathForQuery(raw: string, rows: Surface[] = catalog): string | null {
  const text = raw.trim().toLowerCase();
  if (!text) return null;
  const byPath = rows.find((item) => item.path.toLowerCase() === text);
  if (byPath) return byPath.path;
  const byLabel = rows.find((item) => {
    const label = item.label.toLowerCase();
    return label === text || label.startsWith(text) || label.includes(text);
  });
  return byLabel?.path ?? null;
}

export function surfaceForPath(pathname: string, rows: Surface[] = catalog): Surface {
  const hit = rows.find((item) => item.path === pathname);
  return hit ?? rows[0] ?? SURFACES[0];
}

export const PHASE_PRIORITY: Record<CasePhase, readonly string[]> = {
  Discovery: ["/evidence-requests", "/missing-evidence", "/evidence", "/open-questions"],
  Motions: ["/drafts", "/review", "/final-copy", "/filing-checklist", "/custody-factors"],
  Hearing: ["/calendar", "/timeline", "/filing-checklist", "/assistant-log"],
  Trial: ["/evidence", "/custody-factors", "/drafts"],
};

export function isCasePhase(value: string | null | undefined): value is CasePhase {
  return CASE_PHASES.some((phase) => phase === value);
}

export function compareSurfacesByPhase(a: Surface, b: Surface, phase: CasePhase): number {
  const order = PHASE_PRIORITY[phase];
  const ai = order.indexOf(a.path);
  const bi = order.indexOf(b.path);
  if (ai === -1 && bi === -1) return 0;
  if (ai === -1) return 1;
  if (bi === -1) return -1;
  return ai - bi;
}

export function surfacesForPhase(phase: CasePhase, rows: Surface[] = catalog): Surface[] {
  return [...rows].sort((a, b) => compareSurfacesByPhase(a, b, phase));
}

export function isPhasePriority(path: string, phase: CasePhase): boolean {
  return PHASE_PRIORITY[phase].includes(path);
}

// Byline: OpenAI Codex / GPT-5.6, 2026-08-13
// Byline: Claude Code · Sonnet 5 · 2026-09-07 — router gap/regression patch (UCCJEA gazetteer,
// custodial-interference criminal exposure, DEADLINE_PROXIMITY, UNCLASSIFIED_REVIEW_RECOMMENDED),
// ledger + master-source-directory wiring for audit_sources/search_guide, deadline rule presets
// (MCR 1.108-style counting), and the new case_facts tool.
// Byline: Claude Code · Sonnet 5 · 2026-09-08 — "content lives in the SurrealDB store": ledger
// and master-source-directory reads go store-first (content-store.ts) with an EXACT file-read
// fallback, so auditSources()/searchRecords() are now async. Every caller updated.

import { readFileSync } from "node:fs";
import { homedir } from "node:os";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { getReference, getSources } from "./content-store.js";

export type VerificationStatus =
  | "VERIFIED_PRIMARY"
  | "VERIFIED_OFFICIAL_METADATA_ONLY"
  | "PROVISIONAL_CURRENCY_NOT_CLEARED"
  | "CONFLICTED"
  | "ATTORNEY_REVIEW";

export interface SourceRecord {
  id: string;
  title: string;
  authority: string;
  url: string;
  status: VerificationStatus;
  supports: string;
  checkedOn: string;
}

export const RELEASE_STATUS = {
  label: "PUBLICATION BLOCKED",
  reason: "The source archive is an attorney-unreviewed draft with missing safety-critical modules and unresolved source currency.",
  missingModules: ["Module 3 — UCCJEA", "Module 17 — DV/CPS/PPO safety", "Module 28 — appeals"],
  attorneyReviewRequired: true,
} as const;

export const SOURCES: SourceRecord[] = [
  {
    id: "MCR-3",
    title: "Michigan Court Rules — Chapter 3",
    authority: "Michigan Supreme Court",
    url: "https://www.courts.michigan.gov/siteassets/rules-instructions-administrative-orders/michigan-court-rules/court-rules-book-ch-3-responsive-html5.zip/Court_Rules_Book_Ch_3/Court_Rules_Chapter_3/Court_Rules_Chapter_3.htm",
    status: "VERIFIED_PRIMARY",
    supports: "Domestic relations procedure, referee objections, UCCJEA affidavit procedure, PPO procedure",
    checkedOn: "2026-08-13",
  },
  {
    id: "MCR-7",
    title: "Michigan Court Rules — Chapter 7",
    authority: "Michigan Supreme Court",
    url: "https://www.courts.michigan.gov/siteassets/rules-instructions-administrative-orders/michigan-court-rules/court-rules-book-ch-7-responsive-html5.zip/Court_Rules_Book_Ch_7/Court_Rules_Chapter_7/Court_Rules_Chapter_7.htm",
    status: "VERIFIED_PRIMARY",
    supports: "Appeal timing and qualifying postjudgment motions",
    checkedOn: "2026-08-13",
  },
  {
    id: "MC-416",
    title: "Uniform Child Custody Jurisdiction Enforcement Act Affidavit",
    authority: "Michigan State Court Administrative Office",
    url: "https://www.courts.michigan.gov/siteassets/forms/scao-approved/mc416.pdf",
    status: "VERIFIED_PRIMARY",
    supports: "Current MC 416 form and cited authorities",
    checkedOn: "2026-08-13",
  },
  {
    id: "CC-379",
    title: "Motion to Modify, Extend, or Terminate Personal Protection Order",
    authority: "Michigan State Court Administrative Office",
    url: "https://www.courts.michigan.gov/siteassets/forms/scao-approved/cc379.pdf",
    status: "VERIFIED_PRIMARY",
    supports: "Current PPO motion form identity",
    checkedOn: "2026-08-13",
  },
  {
    id: "FOC-68",
    title: "Objection to Referee Recommended Order",
    authority: "Michigan State Court Administrative Office",
    url: "https://www.courts.michigan.gov/siteassets/forms/scao-approved/instfoc68.pdf",
    status: "VERIFIED_PRIMARY",
    supports: "Current FOC 68 instructions and notice-of-hearing section",
    checkedOn: "2026-08-13",
  },
  {
    id: "MCL-722",
    title: "Michigan Compiled Laws — Chapter 722",
    authority: "Michigan Legislature",
    url: "https://www.legislature.mi.gov/documents/mcl/pdf/mcl-chap722.pdf",
    status: "PROVISIONAL_CURRENCY_NOT_CLEARED",
    supports: "Child custody and UCCJEA statutes; each proposition still needs section-level currency review",
    checkedOn: "2026-08-13",
  },
  {
    id: "MDHHS-CPS",
    title: "Children's Protective Services Investigation Process",
    authority: "Michigan Department of Health and Human Services",
    url: "https://www.michigan.gov/mdhhs/adult-child-serv/abuse-neglect/childrens/report-process/investigation-process-and-results/childrens-protective-services-investigation-process",
    status: "VERIFIED_PRIMARY",
    supports: "Current high-level CPS investigation process",
    checkedOn: "2026-08-13",
  },
];

const DAY_MS = 86_400_000;

function parseIsoDate(value: string): Date {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) throw new Error("Date must use YYYY-MM-DD.");
  const date = new Date(`${value}T12:00:00Z`);
  if (Number.isNaN(date.getTime()) || date.toISOString().slice(0, 10) !== value) {
    throw new Error("Date is not a valid calendar date.");
  }
  return date;
}

function iso(date: Date): string {
  return date.toISOString().slice(0, 10);
}

function isUnavailable(date: Date, holidays: Set<string>): boolean {
  const day = date.getUTCDay();
  return day === 0 || day === 6 || holidays.has(iso(date));
}

// ---------------------------------------------------------------------------
// Deadline rule presets (MCR 1.108-style counting): exclude the anchor day,
// count every calendar day, roll forward over a Saturday/Sunday/holiday.
// ---------------------------------------------------------------------------

export const DEADLINE_RULE_PRESETS = {
  referee_objection: { days: 21, cite: "MCR 3.215(E)(4)", status: "PROVISIONAL_VERIFY" },
  appeal_of_right: { days: 21, cite: "MCR 7.204(A)(1)(a)", status: "PROVISIONAL_VERIFY" },
  motion_response: { days: 7, cite: "MCR 2.119(C)(1)", status: "PROVISIONAL_VERIFY" },
  mail_service_addon: { days: 3, cite: "MCR 2.107(C)(3)/1.108", status: "PROVISIONAL_VERIFY" },
} as const satisfies Record<string, { days: number; cite: string; status: "PROVISIONAL_VERIFY" }>;

export type DeadlineRulePreset = keyof typeof DEADLINE_RULE_PRESETS;

const COUNTING_RULE_NOTE =
  "Excludes the anchor day; counts every calendar day; rolls forward to the next court day on a Saturday, Sunday, or listed holiday (MCR 1.108(1)).";

export function calculateDirectionalDeadline(input: {
  anchorDate: string;
  days?: number;
  direction?: "after" | "before";
  includeAnchor?: boolean;
  holidays?: string[];
  rule?: DeadlineRulePreset;
}) {
  const preset = input.rule ? DEADLINE_RULE_PRESETS[input.rule] : undefined;
  if (input.rule && !preset) throw new Error(`Unknown deadline rule preset: ${input.rule}.`);

  const days = preset ? preset.days : input.days;
  const direction = preset ? "after" : input.direction;
  const includeAnchor = preset ? false : (input.includeAnchor ?? false);

  if (typeof days !== "number") throw new Error("Provide either days, or a rule preset.");
  if (!direction) throw new Error("Provide either direction, or a rule preset.");
  if (!Number.isInteger(days) || days < 0 || days > 3650) {
    throw new Error("Days must be a whole number from 0 through 3650.");
  }

  const anchor = parseIsoDate(input.anchorDate);
  const sign = direction === "after" ? 1 : -1;
  const offset = includeAnchor && days > 0 ? days - 1 : days;
  const raw = new Date(anchor.getTime() + sign * offset * DAY_MS);
  const adjusted = new Date(raw);
  const holidays = new Set((input.holidays ?? []).map((value) => iso(parseIsoDate(value))));
  while (isUnavailable(adjusted, holidays)) adjusted.setUTCDate(adjusted.getUTCDate() + sign);

  const base = {
    anchorDate: input.anchorDate,
    direction,
    days,
    rawDate: iso(raw),
    adjustedDate: iso(adjusted),
    adjustment: iso(raw) === iso(adjusted) ? "none" : direction === "after" ? "moved forward" : "moved backward",
    counting_rule: COUNTING_RULE_NOTE,
    warning: "Planning aid only. Court-rule counting, service method, order type, holidays, and qualifying motions can change a deadline. Confirm with the current rule and court clerk or a Michigan lawyer.",
  };

  if (preset) {
    return {
      ...base,
      rule: input.rule,
      cite: preset.cite,
      status: preset.status,
      warning: `${base.warning} This deadline preset (${preset.cite}) is a planning aid produced from a fixed day-count table, not a legal conclusion — verify against the current rule text before reliance.`,
    };
  }
  return base;
}

// ---------------------------------------------------------------------------
// routeIssue() — safety/jurisdiction router.
//
// 2026-09-07 patch: the previous 6-literal-regex router missed compound,
// buried, or paraphrased issues (documented failure: "moved the child to
// Ohio without notice, hearing in 10 days" -> zero flags). This adds:
//   - a 50-state + DC/PR gazetteer for UCCJEA_INTERSTATE detection
//   - custodial-interference phrasing under CRIMINAL_EXPOSURE
//   - a new DEADLINE_PROXIMITY flag
//   - a new UNCLASSIFIED_REVIEW_RECOMMENDED status for long, keyword-free intake
//   - a `matched` field showing which phrase fired each flag
// ---------------------------------------------------------------------------

const US_STATES: Array<[string, string]> = [
  ["Alabama", "AL"], ["Alaska", "AK"], ["Arizona", "AZ"], ["Arkansas", "AR"], ["California", "CA"],
  ["Colorado", "CO"], ["Connecticut", "CT"], ["Delaware", "DE"], ["Florida", "FL"], ["Georgia", "GA"],
  ["Hawaii", "HI"], ["Idaho", "ID"], ["Illinois", "IL"], ["Indiana", "IN"], ["Iowa", "IA"],
  ["Kansas", "KS"], ["Kentucky", "KY"], ["Louisiana", "LA"], ["Maine", "ME"], ["Maryland", "MD"],
  ["Massachusetts", "MA"], ["Michigan", "MI"], ["Minnesota", "MN"], ["Mississippi", "MS"], ["Missouri", "MO"],
  ["Montana", "MT"], ["Nebraska", "NE"], ["Nevada", "NV"], ["New Hampshire", "NH"], ["New Jersey", "NJ"],
  ["New Mexico", "NM"], ["New York", "NY"], ["North Carolina", "NC"], ["North Dakota", "ND"], ["Ohio", "OH"],
  ["Oklahoma", "OK"], ["Oregon", "OR"], ["Pennsylvania", "PA"], ["Rhode Island", "RI"], ["South Carolina", "SC"],
  ["South Dakota", "SD"], ["Tennessee", "TN"], ["Texas", "TX"], ["Utah", "UT"], ["Vermont", "VT"],
  ["Virginia", "VA"], ["Washington", "WA"], ["West Virginia", "WV"], ["Wisconsin", "WI"], ["Wyoming", "WY"],
  ["District of Columbia", "DC"], ["Puerto Rico", "PR"],
];

const NON_MICHIGAN_STATES = US_STATES.filter(([name]) => name !== "Michigan");
const STATE_ABBR_SET = new Set(NON_MICHIGAN_STATES.map(([, abbr]) => abbr));
const stateNamesForRegex = NON_MICHIGAN_STATES
  .map(([name]) => name.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"))
  .sort((a, b) => b.length - a.length)
  .join("|");
const STATE_NAME_REGEX = new RegExp(`\\b(${stateNamesForRegex})\\b`, "i");

const RELOCATION_VERB_REGEX = /\b(moved|relocated|took|lives|living|staying|went to|resid(?:es|ing))\b/i;
const INTERSTATE_KEYWORD_REGEX =
  /\b(another state|out[- ]of[- ]state|moved states|different state|across state lines|uccjea|home state|interstate|country|abroad|passport)\b/i;

function findStateAbbrMatch(rawDescription: string): string | null {
  const candidates = rawDescription.match(/\b[A-Z]{2}\b/g) ?? [];
  for (const code of candidates) {
    if (code !== "MI" && STATE_ABBR_SET.has(code)) return code;
  }
  return null;
}

const CUSTODIAL_INTERFERENCE_REGEX =
  /without (?:my )?(?:notice|consent|permission)|won'?t (?:return|bring) (?:him|her|the child)|kept (?:him|her|the child)|parental kidnapping|took (?:him|her|the child) and/i;

const NUMBER_WORDS: Record<string, number> = {
  "thirty-one": 31, "thirty one": 31, "thirty": 30,
  "twenty-nine": 29, "twenty nine": 29, "twenty-eight": 28, "twenty eight": 28,
  "twenty-seven": 27, "twenty seven": 27, "twenty-six": 26, "twenty six": 26,
  "twenty-five": 25, "twenty five": 25, "twenty-four": 24, "twenty four": 24,
  "twenty-three": 23, "twenty three": 23, "twenty-two": 22, "twenty two": 22,
  "twenty-one": 21, "twenty one": 21, "twenty": 20,
  nineteen: 19, eighteen: 18, seventeen: 17, sixteen: 16, fifteen: 15,
  fourteen: 14, thirteen: 13, twelve: 12, eleven: 11, ten: 10,
  nine: 9, eight: 8, seven: 7, six: 6, five: 5, four: 4, three: 3, two: 2, one: 1,
};
const NUMBER_WORD_KEYS = Object.keys(NUMBER_WORDS).sort((a, b) => b.length - a.length);
const DEADLINE_COUNT_REGEX = new RegExp(
  `\\b(\\d{1,3}|${NUMBER_WORD_KEYS.join("|")})\\s+(business days|days|day|hours)\\b`,
  "i",
);
const DEADLINE_PHRASE_REGEX = /\b(tomorrow|next week|this week|by friday|due|deadline|expires)\b/i;

function parseDeadlineDays(quantity: string, unit: string): number | null {
  if (/hour/i.test(unit)) return 0; // an hours-scale deadline is always urgent
  const numeric = Number.parseInt(quantity, 10);
  if (Number.isFinite(numeric) && String(numeric) === quantity.trim()) return numeric;
  return NUMBER_WORDS[quantity.toLowerCase()] ?? null;
}

const STOP_FLAGS = ["IMMEDIATE_DANGER", "DV_PPO_CPS", "UCCJEA_INTERSTATE", "APPEAL_DEADLINE", "RECORDING_RISK", "CRIMINAL_EXPOSURE"] as const;
const FLAG_ORDER = [...STOP_FLAGS, "DEADLINE_PROXIMITY"] as const;

export function routeIssue(description: string) {
  const text = description.toLowerCase();
  const matchedPhrases: Record<string, Set<string>> = {};
  const addMatch = (flag: string, phrase: string) => {
    (matchedPhrases[flag] ??= new Set()).add(phrase);
  };

  const simpleFlags: Array<[string, RegExp]> = [
    ["IMMEDIATE_DANGER", /immediate danger|weapon|threaten(?:ed|ing)? to kill|active violence|abduct(?:ion|ed)?/i],
    ["DV_PPO_CPS", /domestic violence|ppo|protective order|cps|child protective|police|sexual assault/i],
    ["APPEAL_DEADLINE", /appeal|reconsideration|postjudgment|final order/i],
    ["RECORDING_RISK", /record(?:ing|ed)?\s+(?:a |my |the |our )?(?:call|conversation|child|exchange|meeting)|secretly record(?:ing|ed)?|on tape|wiretap|eavesdrop/i],
  ];
  for (const [flag, pattern] of simpleFlags) {
    const match = text.match(pattern);
    if (match) addMatch(flag, match[0]);
  }

  // CRIMINAL_EXPOSURE: original literal terms + custodial-interference phrasing.
  const criminalMatch = text.match(/criminal|arrest|charged|probation|warrant/i);
  if (criminalMatch) addMatch("CRIMINAL_EXPOSURE", criminalMatch[0]);
  const custodialMatch = text.match(CUSTODIAL_INTERFERENCE_REGEX);
  if (custodialMatch) addMatch("CRIMINAL_EXPOSURE", custodialMatch[0]);

  // UCCJEA_INTERSTATE: keyword list, OR a non-Michigan state name/abbreviation
  // mentioned together with a relocation/residence verb or "in <State>".
  const keywordMatch = text.match(INTERSTATE_KEYWORD_REGEX);
  if (keywordMatch) addMatch("UCCJEA_INTERSTATE", keywordMatch[0]);
  const stateNameMatch = text.match(STATE_NAME_REGEX);
  const verbMatch = text.match(RELOCATION_VERB_REGEX);
  if (stateNameMatch) {
    const inStatePattern = new RegExp(`\\bin\\s+${stateNameMatch[1].replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}\\b`, "i");
    if (verbMatch || inStatePattern.test(text)) {
      addMatch("UCCJEA_INTERSTATE", stateNameMatch[0]);
      if (verbMatch) addMatch("UCCJEA_INTERSTATE", verbMatch[0]);
    }
  }
  const abbrMatch = findStateAbbrMatch(description);
  if (abbrMatch && (verbMatch || /\bin\b/i.test(text))) {
    addMatch("UCCJEA_INTERSTATE", abbrMatch);
  }

  // DEADLINE_PROXIMITY: explicit count+unit, or a generic near-term phrase.
  const countMatch = text.match(DEADLINE_COUNT_REGEX);
  const phraseMatch = text.match(DEADLINE_PHRASE_REGEX);
  let deadlineDays: number | null = null;
  if (countMatch) {
    addMatch("DEADLINE_PROXIMITY", countMatch[0]);
    deadlineDays = parseDeadlineDays(countMatch[1], countMatch[2]);
  }
  if (phraseMatch) {
    addMatch("DEADLINE_PROXIMITY", phraseMatch[0]);
    if (deadlineDays === null && phraseMatch[1].toLowerCase() === "tomorrow") deadlineDays = 1;
  }

  const matched = FLAG_ORDER.filter((flag) => flag in matchedPhrases);
  const matchedRecord: Record<string, string[]> = {};
  for (const flag of matched) matchedRecord[flag] = Array.from(matchedPhrases[flag]);

  const stop =
    matched.some((flag) => (STOP_FLAGS as readonly string[]).includes(flag)) ||
    (matched.includes("DEADLINE_PROXIMITY") && deadlineDays !== null && deadlineDays <= 14);

  const warning = "This routing result is legal information, not legal advice or an emergency service.";

  if (!stop && matched.length === 0 && description.length >= 240) {
    return {
      status: "UNCLASSIFIED_REVIEW_RECOMMENDED",
      flags: [] as string[],
      matched: matchedRecord,
      nextActions: [
        "No safety or jurisdiction keyword matched, but this intake is long enough that a real issue could be buried in it.",
        "Before proceeding, extract these facts yourself: the child's state of residence for the last 6 months, whether any court order already exists, any date within 21 days, and any DV/CPS/police contact.",
        "If extracting those facts surfaces a stop condition, route out immediately instead of continuing with guided intake.",
      ],
      warning,
    };
  }

  return {
    status: stop ? "STOP_AND_VERIFY" : "GUIDED_INTAKE",
    flags: matched,
    matched: matchedRecord,
    nextActions: matched.includes("IMMEDIATE_DANGER")
      ? ["If anyone faces immediate danger, call 911 or local emergency services.", "Do not use a child to gather evidence.", "Preserve existing orders and seek qualified legal help."]
      : stop
        ? ["Do not rely on a draft module for this issue.", "Preserve documents and exact dates without altering originals.", "Confirm current law and local procedure with primary sources and qualified counsel."]
        : ["Identify the exact order, filing, or event.", "Record entry, service, and hearing dates separately.", "Organize source documents before drafting."],
    warning,
  };
}

export function buildPacketPlan(stage: string, goal: string) {
  return {
    stage,
    goal,
    releaseStatus: RELEASE_STATUS.label,
    sections: [
      { id: "orders", label: "Controlling orders", status: "required", note: "Use the signed, file-stamped version." },
      { id: "dates", label: "Date ledger", status: "required", note: "Keep entry, service, notice, and hearing dates separate." },
      { id: "facts", label: "Fact chronology", status: "required", note: "Separate observed fact, source, and inference." },
      { id: "exhibits", label: "Exhibit index", status: "required", note: "Preserve originals; work from copies." },
      { id: "authority", label: "Authority table", status: "blocked", note: "Current primary-source and attorney review required." },
      { id: "redaction", label: "Privacy and redaction check", status: "required", note: "Minimize child identifiers and protected information." },
    ],
    stopConditions: ["Immediate danger or PPO conflict", "Interstate/UCCJEA issue", "Appeal deadline", "Recording-law question", "CPS/police/criminal overlap"],
  };
}

export function getChecklist(kind: "evidence" | "hearing" | "source-review") {
  const items = {
    evidence: ["Preserve the original", "Create a working copy", "Record source and acquisition date", "Hash when appropriate", "Separate fact from interpretation", "Redact only on a copy"],
    hearing: ["Confirm current order", "Confirm hearing notice", "Verify deadline from primary rule", "Prepare a short issue list", "Index exhibits", "Plan privacy-safe copies"],
    "source-review": ["Open the official primary source", "Verify issuing body", "Match claim to pinpoint", "Check amendment/effective date", "Check form revision", "Record verification date and status"],
  }[kind];
  return { kind, items: items.map((label, index) => ({ id: `${kind}-${index + 1}`, label, done: false })), warning: "A completed checklist is not attorney review or court acceptance." };
}

// ---------------------------------------------------------------------------
// Ledger + master-source-directory wiring (2026-09-07).
//
// audit_sources/search_guide used to read only the 7-record curated SOURCES
// array. This lazily loads (with try/catch, so a missing file degrades to the
// curated-only behavior rather than crashing) the 191-record ledger and the
// 213-URL master source directory, resolved relative to THIS module's own
// import.meta.url — which, once bundled by build.mjs to dist/core.js, sits
// beside dist/server.js, two directories below the plugin root that holds
// content/.
// ---------------------------------------------------------------------------

export interface NormalizedSourceRecord {
  id: string;
  title: string;
  authority: string;
  status: string;
  url: string;
  supports: string;
  checkedOn: string;
  superseded: string | null;
  origin: "curated" | "ledger" | "directory";
}

interface ExtraSources {
  ledger: NormalizedSourceRecord[];
  directory: NormalizedSourceRecord[];
  ledgerError: string | null;
  directoryError: string | null;
}

/** Test-only: drops the module-level loadExtraSources() cache so a test can
 * observe a store-vs-file transition (e.g. populate the store, then re-read)
 * within one process. */
export function resetCoreContentCacheForTests(): void {
  cachedExtraSources = null;
}

let cachedExtraSources: ExtraSources | null = null;

function pluginRootPath(...segments: string[]): string {
  // This module is bundled to dist/core.js by build.mjs, a sibling of
  // dist/server.js; both live at <plugin root>/mcp-app/dist/.
  const here = dirname(fileURLToPath(import.meta.url));
  return join(here, "..", "..", ...segments);
}

function contentPath(...segments: string[]): string {
  return pluginRootPath("content", ...segments);
}

function mcpAppPath(...segments: string[]): string {
  return join(pluginRootPath("mcp-app"), ...segments);
}

/** Maps one raw ledger.json entry OR one `source:<id>` store row onto the
 * common NormalizedSourceRecord shape. Identical either way: the loader
 * writes every ledger field through unchanged (see
 * scripts/load-content-to-store.mjs), so a store row and its origin file
 * entry carry the same fields — the only difference is the store row's `id`
 * comes back "source:<id>" (normalize()'s ref-string form) instead of bare,
 * so a leading "source:" is stripped first. This is what guarantees the
 * store-first path and the file-fallback path produce byte-identical output. */
function mapLedgerRow(record: Record<string, unknown>, index: number): NormalizedSourceRecord {
  const rawId = String(record.id ?? `ledger-${index}`);
  const id = rawId.startsWith("source:") ? rawId.slice("source:".length) : rawId;
  return {
    id,
    title: String(record.title ?? record.short_title ?? id ?? "Untitled ledger record"),
    authority: String(record.issuing_body ?? "Unspecified issuing body"),
    status: String(record.binding_status ?? record.authority_class ?? "UNSPECIFIED").toUpperCase(),
    url: String(record.official_url ?? ""),
    supports: [record.short_title, record.scope_note, record.citation].filter(Boolean).join(" — "),
    checkedOn: String(record.last_verified ?? record.date_accessed ?? "unknown"),
    superseded: record.superseded === undefined || record.superseded === null ? null : String(record.superseded),
    origin: "ledger",
  };
}

function loadLedgerRecordsFromFile(): NormalizedSourceRecord[] {
  const raw = JSON.parse(readFileSync(contentPath("toolkit", "ledger.json"), "utf8"));
  if (!Array.isArray(raw)) throw new Error("ledger.json did not contain an array.");
  return raw.map((record: Record<string, unknown>, index: number) => mapLedgerRow(record, index));
}

/** Store-first, exact-fallback: content lives in the SurrealDB store
 * (`source:<id>` rows the loader writes from ledger.json — see
 * scripts/load-content-to-store.mjs). When the store is unavailable or has no
 * rows yet (an empty mem:// store, or a fresh install before the loader has
 * ever run), this returns EXACTLY what the direct file read returns. */
async function loadLedgerRecords(): Promise<NormalizedSourceRecord[]> {
  try {
    const stored = await getSources();
    if (stored && stored.length > 0) {
      return stored.map((record, index) => mapLedgerRow(record, index));
    }
  } catch {
    // fall through to the file read
  }
  return loadLedgerRecordsFromFile();
}

function parseDirectoryMarkdown(markdown: string): NormalizedSourceRecord[] {
  const linkPattern = /^-\s*\[([^\]]+)\]\(([^)]+)\)(?:\s*—\s*(.*))?$/gm;
  const records: NormalizedSourceRecord[] = [];
  let match: RegExpExecArray | null;
  let index = 0;
  while ((match = linkPattern.exec(markdown))) {
    index += 1;
    const [, title, url, note] = match;
    records.push({
      id: `DIR-${index}`,
      title: title.trim(),
      authority: "Master Source Directory (extracted citation, not independently re-verified)",
      status: "UNVERIFIED_DIRECTORY_ENTRY",
      url: url.trim(),
      supports: (note ?? "").trim(),
      checkedOn: "2026-08-09",
      superseded: "unknown",
      origin: "directory",
    });
  }
  return records;
}

function loadDirectoryRecordsFromFile(): NormalizedSourceRecord[] {
  const markdown = readFileSync(contentPath("custody-guide", "master_source_directory.md"), "utf8");
  return parseDirectoryMarkdown(markdown);
}

/** Store-first, exact-fallback: the whole markdown file body lives in the
 * store as reference:master-source-directory (kind "directory"). Parsing is
 * identical either way — only the markdown text's origin differs. */
async function loadDirectoryRecords(): Promise<NormalizedSourceRecord[]> {
  try {
    const ref = await getReference("master-source-directory");
    if (ref && typeof ref.body === "string" && ref.body.length > 0) {
      return parseDirectoryMarkdown(ref.body);
    }
  } catch {
    // fall through to the file read
  }
  return loadDirectoryRecordsFromFile();
}

async function loadExtraSources(): Promise<ExtraSources> {
  if (cachedExtraSources) return cachedExtraSources;
  let ledger: NormalizedSourceRecord[] = [];
  let ledgerError: string | null = null;
  try {
    ledger = await loadLedgerRecords();
  } catch (error) {
    ledgerError = error instanceof Error ? error.message : String(error);
  }
  let directory: NormalizedSourceRecord[] = [];
  let directoryError: string | null = null;
  try {
    directory = await loadDirectoryRecords();
  } catch (error) {
    directoryError = error instanceof Error ? error.message : String(error);
  }
  cachedExtraSources = { ledger, directory, ledgerError, directoryError };
  return cachedExtraSources;
}

function curatedAsNormalized(): NormalizedSourceRecord[] {
  return SOURCES.map((source) => ({ ...source, superseded: null, origin: "curated" as const }));
}

export async function auditSources(ids?: string[]) {
  const extra = await loadExtraSources();
  const curated = curatedAsNormalized();
  const all = [...curated, ...extra.ledger, ...extra.directory];
  const selected = ids?.length ? all.filter((source) => ids.includes(source.id)) : all;

  const byStatus: Record<string, number> = {};
  const byOrigin: Record<string, number> = {};
  for (const source of selected) {
    byStatus[source.status] = (byStatus[source.status] ?? 0) + 1;
    byOrigin[source.origin] = (byOrigin[source.origin] ?? 0) + 1;
  }

  const notes: string[] = [];
  if (extra.ledgerError) notes.push(`Ledger unavailable, falling back to curated sources only: ${extra.ledgerError}`);
  if (extra.directoryError) notes.push(`Master source directory unavailable: ${extra.directoryError}`);

  return {
    releaseStatus: RELEASE_STATUS.label,
    sources: selected,
    counts: byStatus, // preserved for backward compatibility with existing callers/tests
    source_stats: {
      total: all.length,
      selected: selected.length,
      byStatus,
      byOrigin,
      ledgerLoaded: extra.ledgerError === null,
      directoryLoaded: extra.directoryError === null,
    },
    notes,
  };
}

function scoreRecord(record: NormalizedSourceRecord, terms: string[]): number {
  const haystack = `${record.title} ${record.supports} ${record.id}`.toLowerCase();
  return terms.reduce((score, term) => score + (haystack.split(term).length - 1), 0);
}

export async function searchRecords(query: string) {
  const terms = query.toLowerCase().split(/\s+/).filter((term) => term.length > 2);
  const extra = await loadExtraSources();
  const curated = curatedAsNormalized();
  // Search tiers by trust level (curated > ledger > directory) so a
  // hand-verified match always outranks a bulk-imported one, while still
  // searching across the whole corpus as required.
  const tiers: NormalizedSourceRecord[][] = [curated, extra.ledger, extra.directory];
  const results: Array<NormalizedSourceRecord & { score: number }> = [];
  for (const tier of tiers) {
    const scored = tier
      .map((record) => ({ ...record, score: scoreRecord(record, terms) }))
      .filter((record) => record.score > 0)
      .sort((a, b) => b.score - a.score);
    results.push(...scored);
  }
  return results.slice(0, 12);
}

export function buildChronology(events: Array<{ date: string; title: string; source?: string; knowledgeDate?: string }>) {
  const normalized = events.map((event, index) => ({
    id: index + 1,
    date: iso(parseIsoDate(event.date)),
    title: event.title.trim(),
    source: event.source?.trim() || "Unspecified",
    knowledgeDate: event.knowledgeDate ? iso(parseIsoDate(event.knowledgeDate)) : null,
  })).sort((a, b) => a.date.localeCompare(b.date) || a.id - b.id);
  return { events: normalized, warning: "Chronology order does not prove causation, truth, admissibility, or legal significance." };
}

// ---------------------------------------------------------------------------
// case_facts (new, 2026-09-07) — reads a local case-state JSON file. Never
// echoes a child's name: children[].name is replaced with initials, and the
// public shape only ever carries { count, entries: [{ initials, age }] }.
// ---------------------------------------------------------------------------

export interface CaseFactsConfigured {
  configured: true;
  county: string | null;
  court: string | null;
  judge: string | null;
  referee: string | null;
  controlling_orders: Array<{ title: string; entered: string; served: string | null }>;
  next_hearing: string | null;
  deadlines: Array<{ label: string; due: string; rule: string | null }>;
  parties: string[];
  children: { count: number; entries: Array<{ initials: string | null; age: number | string | null }> };
  flags: string[];
  source: string;
}

export interface CaseFactsUnconfigured {
  configured: false;
  example_path: string;
  hint: string;
}

function toInitials(value: unknown): string | null {
  if (typeof value !== "string") return null;
  const parts = value.trim().split(/\s+/).filter(Boolean);
  if (!parts.length) return null;
  return `${parts.map((part) => part[0]?.toUpperCase() ?? "").join(".")}.`;
}

function defaultCaseFilePath(): string {
  return join(homedir(), ".config", "family-court-toolkit", "case.json");
}

function resolveCaseFilePath(): string {
  const configured = process.env.CUSTODY_CASE_FILE?.trim();
  return configured && configured.length > 0 ? configured : defaultCaseFilePath();
}

export function getCaseFacts(): CaseFactsConfigured | CaseFactsUnconfigured {
  const path = resolveCaseFilePath();
  let raw: Record<string, unknown>;
  try {
    raw = JSON.parse(readFileSync(path, "utf8"));
  } catch {
    return {
      configured: false,
      example_path: mcpAppPath("case.example.json"),
      hint: `No case file found at ${path}. Set CUSTODY_CASE_FILE, or copy case.example.json to that path and fill in your own case details.`,
    };
  }

  const partiesInput = Array.isArray(raw.parties) ? raw.parties : [];
  const parties = partiesInput.map((party) => {
    if (typeof party === "string") return toInitials(party) ?? party;
    if (party && typeof party === "object") {
      const p = party as Record<string, unknown>;
      if (typeof p.initials === "string") return p.initials;
      if (typeof p.name === "string") return toInitials(p.name) ?? "?.";
    }
    return "?.";
  });

  const childrenInput = Array.isArray(raw.children) ? raw.children : [];
  const childEntries = childrenInput.map((child) => {
    if (child && typeof child === "object") {
      const c = child as Record<string, unknown>;
      const initials = typeof c.initials === "string" ? c.initials : toInitials(c.name); // never pass through c.name itself
      const age = typeof c.age === "number" || typeof c.age === "string" ? c.age : null;
      return { initials, age };
    }
    return { initials: null, age: null };
  });

  const controllingOrdersInput = Array.isArray(raw.controlling_orders) ? raw.controlling_orders : [];
  const controlling_orders = controllingOrdersInput.map((order) => {
    const o = (order ?? {}) as Record<string, unknown>;
    return {
      title: String(o.title ?? "Untitled order"),
      entered: String(o.entered ?? "unknown"),
      served: o.served ? String(o.served) : null,
    };
  });

  const deadlinesInput = Array.isArray(raw.deadlines) ? raw.deadlines : [];
  const deadlines = deadlinesInput.map((deadline) => {
    const d = (deadline ?? {}) as Record<string, unknown>;
    return {
      label: String(d.label ?? "Unlabeled deadline"),
      due: String(d.due ?? "unknown"),
      rule: d.rule ? String(d.rule) : null,
    };
  });

  return {
    configured: true,
    county: raw.county ? String(raw.county) : null,
    court: raw.court ? String(raw.court) : null,
    judge: raw.judge ? String(raw.judge) : null,
    referee: raw.referee ? String(raw.referee) : null,
    controlling_orders,
    next_hearing: raw.next_hearing ? String(raw.next_hearing) : null,
    deadlines,
    parties,
    children: { count: childrenInput.length, entries: childEntries },
    flags: Array.isArray(raw.flags) ? raw.flags.map(String) : [],
    source: path,
  };
}

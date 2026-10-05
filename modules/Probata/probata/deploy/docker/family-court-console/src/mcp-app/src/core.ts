// Byline: Codex / GPT-6 / 2026-10-04 - fresh shared reads; explicit mem:// fixture fallback only.
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
import { caseRecord, getStore, caseSummary } from "./store.js";
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

/** Pending marker used until a versioned shared release/audit record is available.
 * Inputs: none. Outputs: an explicit non-validation status for legacy synchronous surfaces.
 * Effects: none. Choose resolveReleaseStatus() for a fresh shared lookup; this marker never certifies compiled claims.
 * Byline: Codex · GPT-6-Luna · 2026-10-04.
 */
export const RELEASE_STATUS = {
  label: "PENDING_SHARED_RELEASE_AUDIT_RECORD",
  status: "pending",
  reason: "No versioned shared release/audit record was found by the current shared-source reader.",
} as const;

/** Compatibility export for callers being migrated to auditSources(); the compiled seven-row catalog is retired.
 * Inputs: none. Outputs: an always-empty array; current sources are read from shared source records.
 * Effects: none. Choose auditSources()/searchRecords() over this compatibility symbol.
 * Byline: Codex · GPT-6-Luna · 2026-10-04.
 */
export const SOURCES: SourceRecord[] = [];

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
// Legacy pure-calculation preset values remain for helper compatibility only.
// The MCP server resolves named presets from the versioned shared reference.
// ---------------------------------------------------------------------------

export const DEADLINE_RULE_PRESETS = {
  referee_objection: { days: 21, cite: "MCR 3.215(E)(4)", status: "PROVISIONAL_VERIFY" },
  appeal_of_right: { days: 21, cite: "MCR 7.204(A)(1)(a)", status: "PROVISIONAL_VERIFY" },
  motion_response: { days: 7, cite: "MCR 2.119(C)(1)", status: "PROVISIONAL_VERIFY" },
  mail_service_addon: { days: 3, cite: "MCR 2.107(C)(3)/1.108", status: "PROVISIONAL_VERIFY" },
} as const satisfies Record<string, { days: number; cite: string; status: "PROVISIONAL_VERIFY" }>;

export type DeadlineRulePreset = keyof typeof DEADLINE_RULE_PRESETS;

export interface RuleCorrectionProposal {
  id: string;
  rule_identity: string;
  rule_heading?: string;
  subdivision_structure?: "subdivisions_are_lists";
  reported_pinpoint?: string;
  source_http_status?: number;
  pinpoint_status?: "unresolved";
  authority_status: "PROVISIONAL_CURRENCY_NOT_CLEARED" | "NOT_STATED_IN_SUPPLIED_AUDIT" | "BLOCKED";
  proposal_status: "PROPOSED_NOT_APPLIED";
  recorded_on: "2026-10-04";
  audit_date: "2026-10-04";
  audit_source: "Dewey official-source audit";
  source_locations: readonly string[];
  gap: string;
  proposed_correction: string;
}

/** Record the supplied Dewey findings as a dated overlay, leaving source documents unchanged.
 * Inputs: none. Outputs: rule-keyed correction proposals with explicit currency/block status.
 * Side effects: none; this data does not edit or validate the original corpus.
 * Choose this proposal overlay over rewriting a draft or presenting the supplied audit as blanket validation.
 * Byline: Codex · GPT-6-Luna · 2026-10-04.
 */
export const RULE_CORRECTION_PROPOSALS: readonly RuleCorrectionProposal[] = [
  {
    id: "dewey-m16-finality-20261004", rule_identity: "MCR 3.215",
    rule_heading: "Rule 3.215 Domestic Relations Referees", subdivision_structure: "subdivisions_are_lists",
    authority_status: "PROVISIONAL_CURRENCY_NOT_CLEARED", proposal_status: "PROPOSED_NOT_APPLIED",
    recorded_on: "2026-10-04", audit_date: "2026-10-04",
    audit_source: "Dewey official-source audit",
    source_locations: ["M16 draft line 129"],
    gap: "The finality statement omits the court-approval and no-timely-objection conditions.",
    proposed_correction: "Propose language stating both conditions; retain the current-currency limitation on the MCR edition.",
  },
  {
    id: "dewey-m16-signature-waiver-20261004", rule_identity: "MCR 3.215",
    rule_heading: "Rule 3.215 Domestic Relations Referees", subdivision_structure: "subdivisions_are_lists",
    authority_status: "PROVISIONAL_CURRENCY_NOT_CLEARED", proposal_status: "PROPOSED_NOT_APPLIED",
    recorded_on: "2026-10-04", audit_date: "2026-10-04",
    audit_source: "Dewey official-source audit",
    source_locations: ["M16 draft line 159", "P1 draft line 254", "E8 immediate-entry requirement"],
    gap: "The signature wording can imply that signature alone waives review or objection.",
    proposed_correction: "Propose clarifying that signature alone does not waive; immediate entry under E8 requires written consent.",
  },
  {
    id: "dewey-m16-live-evidence-f2-20261004", rule_identity: "MCR 3.215",
    rule_heading: "Rule 3.215 Domestic Relations Referees", subdivision_structure: "subdivisions_are_lists",
    authority_status: "PROVISIONAL_CURRENCY_NOT_CLEARED", proposal_status: "PROPOSED_NOT_APPLIED",
    recorded_on: "2026-10-04", audit_date: "2026-10-04",
    audit_source: "Dewey official-source audit",
    source_locations: ["M16 draft line 232", "F2"],
    gap: "The draft must retain F2's live-evidence opportunity and discretionary limits.",
    proposed_correction: "Propose restoring both the live-evidence opportunity and the stated discretionary limits without expanding them.",
  },
  {
    id: "dewey-p1-mail-service-20261004", rule_identity: "MCR 2.107(C)(3)",
    authority_status: "NOT_STATED_IN_SUPPLIED_AUDIT", proposal_status: "PROPOSED_NOT_APPLIED",
    recorded_on: "2026-10-04", audit_date: "2026-10-04",
    audit_source: "Dewey official-source audit",
    source_locations: ["P1 draft line 214"],
    gap: "The service description treats arrival as completion.",
    proposed_correction: "Propose stating that service by mail is complete upon mailing under MCR 2.107(C)(3); do not mark this rule's currency validated from the supplied note.",
  },
  {
    id: "dewey-p1-d4d-exclusions-20261004", rule_identity: "MCR 3.215",
    rule_heading: "Rule 3.215 Domestic Relations Referees", subdivision_structure: "subdivisions_are_lists",
    authority_status: "PROVISIONAL_CURRENCY_NOT_CLEARED", proposal_status: "PROPOSED_NOT_APPLIED",
    recorded_on: "2026-10-04", audit_date: "2026-10-04",
    audit_source: "Dewey official-source audit",
    source_locations: ["P1 draft line 313", "D(4)(d)"],
    gap: "Both stated D(4)(d) exclusions must remain visible.",
    proposed_correction: "Propose retaining the party-requested limitation and the transcript ordered to resolve what happened.",
  },
  {
    id: "dewey-p1-parenting-time-limit-20261004", rule_identity: "MCR 3.215",
    rule_heading: "Rule 3.215 Domestic Relations Referees", subdivision_structure: "subdivisions_are_lists",
    authority_status: "PROVISIONAL_CURRENCY_NOT_CLEARED", proposal_status: "PROPOSED_NOT_APPLIED",
    recorded_on: "2026-10-04", audit_date: "2026-10-04",
    audit_source: "Dewey official-source audit",
    source_locations: ["P1 draft line 266"],
    gap: "Absence of the parenting-time exclusion does not create unconditional permission; custody and mootness limits remain.",
    proposed_correction: "Propose preserving the custody and mootness limits when describing the exclusion's absence.",
  },
  {
    id: "dewey-mcl-552-507-source-block-20261004", rule_identity: "MCL 552.507",
    reported_pinpoint: "MCL 552.507", source_http_status: 403, pinpoint_status: "unresolved",
    authority_status: "BLOCKED", proposal_status: "PROPOSED_NOT_APPLIED",
    recorded_on: "2026-10-04", audit_date: "2026-10-04",
    audit_source: "Dewey official-source audit",
    source_locations: ["MCL 552.507 — official source returned HTTP 403; exact statutory pinpoint remains unresolved"],
    gap: "The supplied audit marks this MCL source blocked and does not resolve the pinpoint.",
    proposed_correction: "Keep claims dependent on this source blocked pending access and exact pinpoint resolution; do not label them verified.",
  },
] as const;

/** Select dated proposals by exact rule identity, including explicit subdivisions.
 * Inputs: rule citations from a returned claim/context; outputs: matching proposals only.
 * Side effects: none. Choose over title or keyword similarity so MCR and MCL records cannot cross-match.
 * Byline: Codex · GPT-6-Luna · 2026-10-04.
 */
export function ruleCorrectionProposalsForCitations(citations: readonly string[]): RuleCorrectionProposal[] {
  const normalized = citations.filter((value) => typeof value === "string").map((value) => value.replace(/\s+/g, "").toUpperCase());
  return RULE_CORRECTION_PROPOSALS.filter((proposal) => {
    const rule = proposal.rule_identity.replace(/\s+/g, "").toUpperCase();
    return normalized.some((citation) => citation === rule || citation.startsWith(`${rule}(`));
  }).map((proposal) => ({ ...proposal }));
}

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

/** Calculate a named deadline only from the exact current shared preset reference.
 * Inputs: anchor date, bounded shared preset ID, and optional holiday dates.
 * Outputs: a calculated date with configuration record ID/version/hash, configured status, and exact shared-source audit; missing or unknown presets return a named pending result without a date.
 * Side effects: bounded read-only shared record and source-catalog queries. Choose this for server rule requests; the legacy pure helper remains for callers that explicitly need its historical calculation API.
 * Configuration contract: record `reference:deadline-rule-presets`, kind `deadline_rule_presets`, data schema `propria.deadline-rule-presets.v1`, with at most 100 unique presets containing id, days, direction, optional include_anchor, citation, and status.
 * Byline: Codex · GPT-6-Luna · 2026-10-04.
 */
export async function calculateSharedDeadlinePreset(input: {
  anchorDate: string;
  rule: string;
  holidays?: string[];
}): Promise<Record<string, unknown>> {
  if (typeof input.rule !== "string" || !/^[a-z][a-z0-9_-]{0,79}$/.test(input.rule)) {
    throw new Error("Invalid shared deadline preset ID");
  }
  const expectedRef = "reference:deadline-rule-presets";
  const store = await getStore();
  if (!store.available) return {
    configured: false,
    status: "PENDING_SHARED_DEADLINE_RULE_PRESET_CONFIGURATION",
    rule: input.rule,
    configuration_record_ref: expectedRef,
    configuration_record_version: null,
    configuration_record_hash: null,
    reason: store.reason,
  };
  const current = await caseRecord(store, { table: "reference", id: "deadline-rule-presets" });
  if (!current) return {
    configured: false,
    status: "PENDING_SHARED_DEADLINE_RULE_PRESET_CONFIGURATION",
    rule: input.rule,
    configuration_record_ref: expectedRef,
    configuration_record_version: null,
    configuration_record_hash: null,
    reason: "The exact shared deadline-rule-preset reference is missing.",
  };
  if (current.id !== expectedRef || !/^sha256:[a-f0-9]{64}$/.test(current.version)) {
    throw new Error("Malformed shared deadline-rule-preset record identity or version");
  }
  const row = current.record;
  const data = row.data;
  if (row.key !== "deadline-rule-presets" || row.kind !== "deadline_rule_presets"
    || !data || typeof data !== "object" || Array.isArray(data)) {
    throw new Error("Malformed shared deadline-rule-preset reference envelope");
  }
  const config = data as Record<string, unknown>;
  if (config.schema !== "propria.deadline-rule-presets.v1" || !Array.isArray(config.presets) || config.presets.length > 100) {
    throw new Error("Malformed shared deadline-rule-preset schema or bounded preset list");
  }
  const seen = new Set<string>();
  const presets: Array<{ id: string; days: number; direction: "after" | "before"; include_anchor: boolean; citation: string; status: string }> = [];
  for (const candidate of config.presets) {
    if (!candidate || typeof candidate !== "object" || Array.isArray(candidate)) throw new Error("Malformed shared deadline-rule-preset entry");
    const item = candidate as Record<string, unknown>;
    if (typeof item.id !== "string" || !/^[a-z][a-z0-9_-]{0,79}$/.test(item.id) || seen.has(item.id)
      || !Number.isSafeInteger(item.days) || (item.days as number) < 0 || (item.days as number) > 3650
      || (item.direction !== "after" && item.direction !== "before")
      || (item.include_anchor !== undefined && typeof item.include_anchor !== "boolean")
      || typeof item.citation !== "string" || !item.citation.trim() || item.citation.length > 256
      || typeof item.status !== "string" || !item.status.trim() || item.status.length > 80) {
      throw new Error("Malformed or duplicate shared deadline-rule-preset entry");
    }
    seen.add(item.id);
    presets.push({
      id: item.id, days: item.days as number, direction: item.direction,
      include_anchor: item.include_anchor === true, citation: item.citation, status: item.status,
    });
  }
  const configuration = {
    record_ref: current.id,
    record_version: current.version,
    record_hash: current.version.slice("sha256:".length),
    hash_basis: "caseRecord full shared row with embedding omitted",
    schema: config.schema,
  };
  const preset = presets.find((item) => item.id === input.rule);
  if (!preset) return {
    configured: false,
    status: "UNKNOWN_SHARED_DEADLINE_RULE_PRESET",
    rule: input.rule,
    configuration_record: configuration,
    available_rule_ids: presets.map((item) => item.id),
  };
  const calculation = calculateDirectionalDeadline({
    anchorDate: input.anchorDate,
    days: preset.days,
    direction: preset.direction,
    includeAnchor: preset.include_anchor,
    holidays: input.holidays,
  });
  const catalog = await getSharedRuleCatalogContext([preset.citation]);
  return {
    configured: true,
    status: "SHARED_DEADLINE_RULE_PRESET_CONFIGURED",
    ...calculation,
    rule: preset.id,
    cite: preset.citation,
    configured_status: preset.status,
    preset_provenance: {
      configuration_record: configuration,
      shared_rule_traceability: catalog.rule_source_traceability[0],
    },
    release_status: catalog.release_status,
  };
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
// array. Current content loads fresh; only explicit empty mem fixtures use the
// curated-only behavior rather than crashing) the 191-record ledger and the
// 213-URL master source directory, resolved relative to THIS module's own
// import.meta.url — which, once bundled by build.mjs to dist/core.js, sits
// beside dist/server.js, two directories below the plugin root that holds
// content/.
// ---------------------------------------------------------------------------

export interface NormalizedSourceRecord {
  id: string;
  record_ref: string;
  title: string;
  authority: string;
  status: string;
  url: string;
  supports: string;
  checkedOn: string;
  citation: string | null;
  citations: string[];
  pinpoint: string | null;
  source_sha256: string | null;
  record_version: string | null;
  record_version_status: "verified_current_record" | "not_loaded" | "version_lookup_gap";
  record_kind: string | null;
  release_state: string | null;
  superseded: string | null;
  origin: "shared-source";
}

/** Preserve the test reset API; current content reads no longer retain a cache.
 * Inputs/outputs: none. Effects: none. Use for compatibility with existing fixture setup. */
export function resetCoreContentCacheForTests(): void {
  // Compatibility hook: content is read fresh on every request.
}


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

/** Map one current shared source row while retaining its exact record identity and provenance.
 * Inputs: a row returned by the paginated shared source reader; outputs: catalog metadata without inferred verification.
 * Effects: none. Choose over compiled summaries because this preserves the live shared record's identity/hash/status.
 * Byline: Codex · GPT-6-Luna · 2026-10-04.
 */
function mapSharedSourceRow(record: Record<string, unknown>): NormalizedSourceRecord {
  const rawId = typeof record.id === "string" ? record.id : "";
  if (!rawId.startsWith("source:") || rawId.length <= "source:".length) throw new Error("Malformed shared source identity");
  const id = rawId.slice("source:".length);
  const sourceSha = typeof record.sha256 === "string" && /^[a-f0-9]{64}$/i.test(record.sha256) ? record.sha256.toLowerCase() : null;
  const citations = Array.isArray(record.citations) ? record.citations.filter((value): value is string => typeof value === "string")
    : typeof record.citation === "string" ? [record.citation] : [];
  return {
    id, record_ref: rawId,
    title: String(record.title ?? record.short_title ?? id),
    authority: String(record.issuing_body ?? "Unspecified issuing body"),
    status: String(record.binding_status ?? record.authority_class ?? "STATUS_NOT_RECORDED").toUpperCase(),
    url: String(record.official_url ?? record.url ?? ""),
    supports: [record.short_title, record.scope_note].filter((part): part is string => typeof part === "string" && part.length > 0).join(" — "),
    checkedOn: String(record.last_verified ?? record.date_accessed ?? "not recorded"),
    citation: typeof record.citation === "string" ? record.citation : null,
    citations,
    pinpoint: typeof record.pinpoint === "string" ? record.pinpoint : null,
    source_sha256: sourceSha,
    record_version: null,
    record_version_status: "not_loaded",
    record_kind: typeof record.record_kind === "string" ? record.record_kind : typeof record.record_type === "string" ? record.record_type : null,
    release_state: typeof record.release_state === "string" ? record.release_state.slice(0, 128)
      : typeof record.release_status === "string" ? record.release_status.slice(0, 128) : null,
    superseded: record.superseded === undefined || record.superseded === null ? null : String(record.superseded),
    origin: "shared-source",
  };
}

/** Load the complete bounded shared source catalog using the canonical paginated reader.
 * Inputs: none; getSources enforces ordered pages and the 10,000-row budget. Outputs: current rows only.
 * Effects: fresh read-only shared queries; missing/unavailable catalogs throw and never fall back to packaged files.
 * Pick this over SOURCES, source JSON, or the link directory for current source identity/status.
 * Byline: Codex · GPT-6-Luna · 2026-10-04.
 */
async function loadSharedSourceCatalog(): Promise<NormalizedSourceRecord[]> {
  const stored = await getSources();
  return (stored ?? []).map(mapSharedSourceRow);
}

/** Verify selected catalog rows against the canonical caseRecord version query.
 * Inputs: up to 32 shared-source rows; outputs: exact sha256-prefixed record versions or named lookup gaps.
 * Effects: bounded read-only version lookups; source SHA is kept separate from record version. Pick over guessing a row version.
 * Byline: Codex · GPT-6-Luna · 2026-10-04.
 */
async function loadCurrentRecordVersions(rows: readonly Pick<NormalizedSourceRecord, "id" | "record_ref">[]): Promise<Array<Pick<NormalizedSourceRecord, "id" | "record_ref"> & Pick<NormalizedSourceRecord, "record_version" | "record_version_status">>> {
  const targets = rows.slice(0, 32);
  if (!targets.length) return [];
  const notLoaded = rows.slice(32).map((row) => ({ ...row, record_version: null, record_version_status: "not_loaded" as const }));
  const store = await getStore();
  if (!store.available) return [
    ...targets.map((row) => ({ ...row, record_version: null, record_version_status: "version_lookup_gap" as const })),
    ...notLoaded,
  ];
  const versioned = await Promise.all(targets.map(async (row) => {
    try {
      const current = await caseRecord(store, { table: "source", id: row.id });
      if (!current || current.id !== row.record_ref || !/^sha256:[a-f0-9]{64}$/.test(current.version)) {
        return { ...row, record_version: null, record_version_status: "version_lookup_gap" as const };
      }
      return { ...row, record_version: current.version, record_version_status: "verified_current_record" as const };
    } catch {
      return { ...row, record_version: null, record_version_status: "version_lookup_gap" as const };
    }
  }));
  return [...versioned, ...notLoaded];
}

/** Resolve a compiled citation only through a shared source row's exact citation or pinpoint field.
 * Inputs: one claimed citation and the current bounded shared catalog; outputs: tied record metadata or named gaps.
 * Effects: none. Pick exact equality over title/body keyword overlap, which cannot establish claim-to-source fit.
 * Byline: Codex · GPT-6-Luna · 2026-10-04.
 */
export function traceCompiledRuleSource(citation: string, catalog: readonly NormalizedSourceRecord[]) {
  const normalizeCitation = (value: string) => value.trim().replace(/\s+/g, " ").toUpperCase();
  const target = normalizeCitation(citation);
  const matches = catalog.filter((source) => [...source.citations, ...(source.pinpoint ? [source.pinpoint] : [])]
    .some((value) => normalizeCitation(value) === target));
  if (!matches.length) return {
    claimed_citation: citation,
    traceability_status: "GAP_NO_EXACT_SHARED_SOURCE_RECORD" as const,
    shared_sources: [],
  };
  return {
    claimed_citation: citation,
    traceability_status: matches.length === 1 ? "EXACT_SHARED_SOURCE_RECORD" as const : "AMBIGUOUS_EXACT_SHARED_SOURCE_RECORDS" as const,
    shared_sources: matches.map((source) => ({
      record_ref: source.record_ref,
      title: source.title,
      official_url: source.url || null,
      authority_status: source.status,
      checked_on: source.checkedOn,
      source_sha256: source.source_sha256,
      record_version: source.record_version,
      pinpoint: source.pinpoint,
      traceability_gaps: [
        ...(source.source_sha256 ? [] : ["shared_source_sha256_missing"]),
        ...(source.record_version ? [] : ["shared_record_version_not_exposed_by_paginated_catalog"]),
      ],
    })),
  };
}

/** Trace several compiled citations against one fresh shared catalog traversal.
 * Inputs: citation strings from unchanged compiled rule/deadline claims; outputs: one trace or explicit gap per citation.
 * Effects: bounded shared source read only. Pick this over repeated per-citation catalog loads or keyword matching.
 * Byline: Codex · GPT-6-Luna · 2026-10-04.
 */
export async function traceCompiledRuleSources(citations: readonly string[]) {
  const catalog = await loadSharedSourceCatalog();
  const traces = citations.map((citation) => traceCompiledRuleSource(citation, catalog));
  const linkedRefs = new Set(traces.flatMap((trace) => trace.shared_sources.map((source) => source.record_ref)));
  const versioned = await loadCurrentRecordVersions(catalog.filter((source) => linkedRefs.has(source.record_ref)));
  const byRef = new Map(versioned.map((source) => [source.record_ref, source]));
  return traces.map((trace) => ({
    ...trace,
    shared_sources: trace.shared_sources.map((item) => {
      const current = byRef.get(String(item.record_ref));
      return current ? {
        ...item,
        record_version: current.record_version,
        record_version_status: current.record_version_status,
        traceability_gaps: [
          ...item.traceability_gaps.filter((gap) => gap !== "shared_record_version_not_exposed_by_paginated_catalog"),
          ...(current.record_version_status === "verified_current_record" ? [] : ["current_caseRecord_version_lookup_failed"]),
        ],
      } : item;
    }),
  }));
}

/** Assemble rule provenance and release status from one fresh shared catalog traversal.
 * Inputs: unchanged compiled citations; outputs: exact source links/gaps and a versioned-release-record status.
 * Effects: bounded shared reads only. Pick over separate catalog calls when preparing one guide response.
 * Byline: Codex · GPT-6-Luna · 2026-10-04.
 */
export async function getSharedRuleCatalogContext(citations: readonly string[]) {
  const catalog = await loadSharedSourceCatalog();
  const traces = citations.map((citation) => traceCompiledRuleSource(citation, catalog));
  const linkedRefs = new Set(traces.flatMap((trace) => trace.shared_sources.map((source) => source.record_ref)));
  const versioned = await loadCurrentRecordVersions(catalog.filter((source) => linkedRefs.has(source.record_ref)));
  const byRef = new Map(versioned.map((source) => [source.record_ref, source]));
  return {
    release_status: await resolveSharedReleaseStatus(catalog),
    rule_source_traceability: traces.map((trace) => ({
      ...trace,
      shared_sources: trace.shared_sources.map((item) => {
        const current = byRef.get(String(item.record_ref));
        return current ? {
          ...item,
          record_version: current.record_version,
          record_version_status: current.record_version_status,
          traceability_gaps: [
            ...item.traceability_gaps.filter((gap) => gap !== "shared_record_version_not_exposed_by_paginated_catalog"),
            ...(current.record_version_status === "verified_current_record" ? [] : ["current_caseRecord_version_lookup_failed"]),
          ],
        } : item;
      }),
    })),
  };
}

/** Resolve the release surface without manufacturing a release decision from compiled content.
 * Inputs: current shared source catalog; outputs: named pending status until a versioned release/audit record exists.
 * Effects: none. Pick this over static RELEASE_STATUS text; current reader exposes source rows only, no release record type.
 * Byline: Codex · GPT-6-Luna · 2026-10-04.
 */
async function resolveSharedReleaseStatus(catalog: readonly NormalizedSourceRecord[]) {
  const candidates = catalog.filter((source) => ["release_status", "release_audit"].includes(source.record_kind ?? "")
    || source.id === "family-court-release-status" || source.id === "family-court-release-audit");
  if (!candidates.length) return { ...RELEASE_STATUS, record_ref: null, record_version: null };
  const versioned = await loadCurrentRecordVersions(candidates);
  const byRef = new Map(versioned.map((source) => [source.record_ref, source]));
  const current = candidates.find((source) => byRef.get(source.record_ref)?.record_version_status === "verified_current_record" && source.release_state);
  if (current) return {
    label: current.release_state!,
    status: current.release_state!,
    reason: "Presented verbatim from the current versioned shared release/audit source record; this is not an independent validation claim.",
    record_ref: current.record_ref,
    source_sha256: current.source_sha256,
    record_version: byRef.get(current.record_ref)?.record_version ?? null,
  };
  return {
    label: "PENDING_VERSIONED_SHARED_RELEASE_AUDIT_RECORD",
    status: "pending",
    reason: "A release/audit candidate exists, but no state with a verified current caseRecord version was available.",
    candidate_refs: candidates.map((source) => source.record_ref),
    version_gaps: candidates.map((source) => ({ record_ref: source.record_ref, status: byRef.get(source.record_ref)?.record_version_status ?? "not_loaded" })),
  };
}

/** Audit the selected current source rows alongside dated rule-correction proposals.
 * Inputs: optional exact source IDs; outputs: source audit, status counts and non-applied proposals.
 * Side effects: fresh shared reads only. Choose over reading a compiled catalog without its current audit overlay.
 * Byline: Codex · GPT-6-Luna · 2026-10-04.
 */
export async function auditSources(ids?: string[]) {
  const all = await loadSharedSourceCatalog();
  const requested = ids?.length ? new Set(ids) : null;
  const selectedRaw = requested ? all.filter((source) => requested.has(source.id) || requested.has(source.record_ref)) : all;
  const versioned = requested ? await loadCurrentRecordVersions(selectedRaw) : [];
  const versionById = new Map(versioned.map((source) => [source.id, source]));
  const selected: NormalizedSourceRecord[] = requested
    ? selectedRaw.map((source) => {
      const current = versionById.get(source.id);
      return current ? { ...source, ...current } : { ...source, record_version: null, record_version_status: "not_loaded" as const };
    })
    : selectedRaw;

  const byStatus: Record<string, number> = {};
  const byOrigin: Record<string, number> = {};
  for (const source of selected) {
    byStatus[source.status] = (byStatus[source.status] ?? 0) + 1;
    byOrigin[source.origin] = (byOrigin[source.origin] ?? 0) + 1;
  }

  const notes: string[] = ["Compiled source summaries are not included; source identity and status come from current shared source records."];
  if (selected.some((source) => source.record_version_status !== "verified_current_record")) notes.push("Some selected rows lack a verified current caseRecord version; record_version is reported as a gap, not inferred from source_sha256.");
  if (!requested) notes.push("The unfiltered catalog omits per-row caseRecord lookups to keep the broad listing bounded; request exact source IDs for versioned record proofs.");
  if (requested && selectedRaw.length > 32) notes.push("Only the first 32 exact IDs receive caseRecord version lookups per request; remaining rows retain explicit missing-version status.");
  const releaseStatus = await resolveSharedReleaseStatus(all);

  return {
    releaseStatus: releaseStatus.label,
    release_status: releaseStatus,
    sources: selected,
    rule_correction_proposals: [...RULE_CORRECTION_PROPOSALS],
    counts: byStatus, // preserved for backward compatibility with existing callers/tests
    source_stats: {
      total: all.length,
      selected: selected.length,
      byStatus,
      byOrigin,
      ledgerLoaded: true,
      directoryLoaded: false,
      source_catalog: "shared-source-records",
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
  const catalog = await loadSharedSourceCatalog();
  const matches = catalog
    .map((record) => ({ ...record, score: scoreRecord(record, terms) }))
    .filter((record) => record.score > 0)
    .sort((a, b) => b.score - a.score || a.record_ref.localeCompare(b.record_ref))
    .slice(0, 12);
  const versioned = await loadCurrentRecordVersions(matches);
  const byRef = new Map(versioned.map((source) => [source.record_ref, source]));
  return matches.map((record) => ({ ...record, ...(byRef.get(record.record_ref) ?? {}) }));
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
// case_facts (2026-09-07) — the legacy local case-state reader preserves full
// party/child records, including private names and fields not known to this app.
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
  parties: unknown[];
  children: { count: number; entries: unknown[] };
  flags: string[];
  source: string;
}

export interface CaseFactsUnconfigured {
  configured: false;
  example_path: string;
  hint: string;
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

  const parties = Array.isArray(raw.parties) ? raw.parties : [];
  const children = Array.isArray(raw.children) ? raw.children : [];

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
    children: { count: children.length, entries: children },
    flags: Array.isArray(raw.flags) ? raw.flags.map(String) : [],
    source: path,
  };
}

/** Read full private case facts from the canonical shared case store while retaining compatible summary fields.
 * Inputs: none; opens the existing shared store and delegates shape construction to caseSummary().
 * Outputs: compatible summary fields plus full case_context, party rows, child rows, or a named absence/unavailable result.
 * Effects: fresh read-only queries that preserve full names and custom private fields. Pick over the legacy local-file reader for current shared case context.
 * Byline: Codex · GPT-6-Luna · 2026-10-04.
 */
export async function getSharedCaseFacts(): Promise<Record<string, unknown>> {
  const store = await getStore();
  if (!store.available) return {
    configured: false,
    status: "SHARED_CASE_STORE_UNAVAILABLE",
    source: "surrealdb-case-store",
    reason: store.reason,
  };
  const summary = await caseSummary(store);
  const counts = summary.counts;
  const hasCaseData = Boolean(
    summary.county || summary.court || summary.judge || summary.referee || summary.controlling_orders.length
    || summary.next_hearing || summary.deadlines.length || summary.parties.length || summary.children.count
    || ["court", "order", "hearing", "deadline", "person", "child"].some((table) => (counts[table] ?? 0) > 0),
  );
  if (!hasCaseData) return {
    configured: false,
    status: "SHARED_CASE_SUMMARY_EMPTY",
    source: summary.source,
    shape: "county, court, judge, referee, orders, hearing, deadlines, full party and child records, and case_context",
    traceability_gap: "The shared case summary contains no current case records; no local case-file fallback is used here.",
  };
  return {
    configured: true,
    county: summary.county,
    court: summary.court,
    judge: summary.judge,
    referee: summary.referee,
    controlling_orders: summary.controlling_orders,
    next_hearing: summary.next_hearing,
    deadlines: summary.deadlines,
    parties: summary.parties,
    children: summary.children,
    case_context: summary.case_context,
    flags: summary.flags,
    source: summary.source,
    source_record_refs: ["court:main", "order:*", "hearing:*", "deadline:*", "person:*", "child:*"],
    traceability_gap: "caseSummary aggregates current rows but does not expose a version/hash per contributing record.",
  };
}

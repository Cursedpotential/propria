// Byline: Codex / GPT-6 / 2026-10-04 - fresh shared reads; explicit mem:// fixture fallback only.
// Byline: Claude Code · Fable 5.1 · 2026-09-07
//
// court_language_review engine. Deterministic lexicon-driven flagging (owner
// framing 2026-09-07 09:52: "the flags could be mechanical"). The REWRITE
// itself is done by the calling model following
// skills/family-court-toolkit/references/court-language/SKILL.md + TEMPLATES.md
// + EXAMPLES.md — this module only guarantees the flagging is thorough and
// repeatable, and loads the safe_phrasebank from EXAMPLES.md at runtime so the
// tool output and the reference member never drift apart.
// Byline: Claude Code · Sonnet 5 · 2026-09-08 — "content lives in the SurrealDB store":
// lexicon and EXAMPLES.md go store-first (content-store.ts) with an EXACT file-read
// fallback. loadLexicon()/loadPhrasebanks()/findLexiconMatches()/reviewCourtLanguage()
// are now async; every caller updated.

import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { getReference } from "./content-store.js";

function pluginRootPath(...segments: string[]): string {
  const here = dirname(fileURLToPath(import.meta.url));
  return join(here, "..", "..", ...segments);
}

const LEXICON_PATH = pluginRootPath("content", "tools", "court-language", "lexicon.json");
const EXAMPLES_PATH = pluginRootPath("skills", "court-language", "EXAMPLES.md") // members are first-class skills (owner ruling 2026-09-07 15:23);

export type Severity = "stop" | "fix" | "soften";

export interface LexiconEntry {
  category: string;
  pattern: string;
  severity: Severity;
  why: string;
  rewrite_pattern: string;
  example_before: string;
  example_after: string;
}

export interface DocProfile {
  register: string;
  allowed: string[];
  forbidden: string[];
  rewrite_recipe: string;
}

export interface Lexicon {
  entries: LexiconEntry[];
  doc_profiles: Record<string, DocProfile>;
}

export type DocType =
  | "affidavit"
  | "motion_brief"
  | "testimony_answer"
  | "message_to_other_parent"
  | "incident_log"
  | "objection_to_recommendation";

export const DOC_TYPES: DocType[] = [
  "affidavit", "motion_brief", "testimony_answer", "message_to_other_parent", "incident_log", "objection_to_recommendation",
];


function loadLexiconFromFile(): Lexicon {
  return JSON.parse(readFileSync(LEXICON_PATH, "utf8")) as Lexicon;
}

/** Read the current shared court-language lexicon.
 * Inputs: none. Outputs: validated lexicon structure.
 * Effects: one shared read, or absent mem fixture file read. Pick this over loadLexiconFromFile for explicit fixtures.
 */
export async function loadLexicon(): Promise<Lexicon> {
  const ref = await getReference("court-language-lexicon");
  if (ref) {
    const data = ref.data as Lexicon | undefined;
    if (!data || !Array.isArray(data.entries) || !data.doc_profiles || typeof data.doc_profiles !== "object" || Array.isArray(data.doc_profiles)
      || data.entries.some(entry => !entry || typeof entry.pattern !== "string" || typeof entry.category !== "string"
        || !["stop", "fix", "soften"].includes(entry.severity))) {
      throw new Error("Malformed shared court-language lexicon");
    }
    for (const entry of data.entries) new RegExp(entry.pattern, "i");
    return data;
  }
  return loadLexiconFromFile();
}


// Parses EXAMPLES.md: a "## <doc_type>" heading starts a section; a
// "### Safe phrasebank" heading within that section is followed by a markdown
// bullet list, which becomes that doc_type's safe_phrasebank. Mirrors the
// existing loadDirectoryRecords() markdown-parsing pattern in core.ts.
function parsePhrasebanks(markdown: string): Record<string, string[]> {
  const sections = markdown.split(/^## /m).slice(1); // drop the front-matter/preamble chunk
  const banks: Record<string, string[]> = {};
  for (const section of sections) {
    const docType = section.split("\n", 1)[0]?.trim();
    if (!docType) continue;
    const phrasebankMatch = section.match(/### Safe phrasebank\s*\n((?:- .*\n?)+)/);
    if (!phrasebankMatch) continue;
    const bullets = phrasebankMatch[1]
      .split("\n")
      .map((line) => line.trim())
      .filter((line) => line.startsWith("- "))
      .map((line) => line.slice(2).trim().replace(/^"(.*)"$/, "$1"));
    banks[docType] = bullets;
  }
  return banks;
}

function loadPhrasebanksFromFile(): Record<string, string[]> {
  return parsePhrasebanks(readFileSync(EXAMPLES_PATH, "utf8"));
}

/** Read and parse the current shared court-language example text.
 * Inputs: none. Outputs: phrasebank mapping.
 * Effects: one shared read, or absent mem fixture file read. Pick this over loadPhrasebanksFromFile for explicit fixtures.
 */
export async function loadPhrasebanks(): Promise<Record<string, string[]>> {
  const ref = await getReference("court-language-examples");
  if (ref) {
    if (typeof ref.body !== "string" || !ref.body.length) throw new Error("Malformed shared court-language examples");
    return parsePhrasebanks(ref.body);
  }
  return loadPhrasebanksFromFile();
}

/** Preserve the lexicon test reset API after removal of retained content.
 * Inputs/outputs: none. Effects: none. Use for compatibility with existing fixture setup. */
export function resetCourtLanguageCacheForTests(): void {
  // Compatibility hook: content is read fresh on every request.
}

export interface Finding {
  category: string;
  severity: Severity;
  span: [number, number];
  excerpt: string;
  why: string;
  suggested_rewrite: string;
}

function excerptAround(text: string, start: number, end: number, maxLen = 60): string {
  const padStart = Math.max(0, start - 15);
  const padEnd = Math.min(text.length, end + 15);
  let excerpt = text.slice(padStart, padEnd).replace(/\s+/g, " ").trim();
  if (excerpt.length > maxLen) excerpt = `${excerpt.slice(0, maxLen - 1)}…`;
  return excerpt;
}

export async function findLexiconMatches(text: string): Promise<Finding[]> {
  const lexicon = await loadLexicon();
  const findings: Finding[] = [];
  for (const entry of lexicon.entries) {
    let regex: RegExp;
    try {
      regex = new RegExp(entry.pattern, "gi");
    } catch {
      continue; // a malformed pattern never crashes the review
    }
    let match: RegExpExecArray | null;
    while ((match = regex.exec(text))) {
      const start = match.index;
      const end = start + match[0].length;
      findings.push({
        category: entry.category,
        severity: entry.severity,
        span: [start, end],
        excerpt: excerptAround(text, start, end),
        why: entry.why,
        suggested_rewrite: entry.rewrite_pattern,
      });
      if (match[0].length === 0) regex.lastIndex += 1; // guard against zero-width loops
    }
  }
  findings.sort((a, b) => a.span[0] - b.span[0]);
  return findings;
}

function splitParagraphs(text: string): string[] {
  return text
    .split(/\n\s*\n|(?<=\.)\s*\n(?=\s*\d+[.)]\s)/)
    .map((p) => p.trim())
    .filter(Boolean);
}

function countSentences(text: string): number {
  const matches = text.match(/[^.!?]+[.!?]+/g);
  return matches ? matches.length : text.trim().length > 0 ? 1 : 0;
}

const ARGUMENT_WORDS = /\b(therefore|thus|accordingly|clearly (?:demonstrates|shows|proves)|proves that|the court should find|it is clear that)\b/i;
const HISTORY_RECAP = /\b(remember when|you always|in the past|this is exactly why|ever since|for years|every time we|the whole time we were)\b/i;
const DATE_TOKEN = /\b\d{1,2}[/-]\d{1,2}[/-]\d{2,4}\b|\b(?:January|February|March|April|May|June|July|August|September|October|November|December)\s+\d{1,2}\b/i;
const CITATION_TOKEN = /\b(exhibit|ex\.|record|transcript|page \d+|MCR|MCL)\b/i;
const FACT_STRUCTURE = /\bF\s*:|\bA\s*:|\bC\s*:|\bT\s*:/;
const FINDING_TOKEN = /\bfinding\b|\bMCR 3\.215/i;

export function checkProfileViolations(text: string, docType: DocType): string[] {
  const violations: string[] = [];
  switch (docType) {
    case "affidavit": {
      for (const paragraph of splitParagraphs(text)) {
        if (countSentences(paragraph) > 3) {
          violations.push(`Paragraph exceeds ~3 sentences for an affidavit: "${paragraph.slice(0, 50)}…" — split into one fact per numbered paragraph.`);
        }
      }
      if (ARGUMENT_WORDS.test(text)) {
        violations.push("Affidavit contains argument language (e.g. \"therefore\", \"clearly demonstrates\") — affidavits state facts only; argument belongs in the brief.");
      }
      break;
    }
    case "motion_brief": {
      if (DATE_TOKEN.test(text) && !CITATION_TOKEN.test(text)) {
        violations.push("A dated factual claim appears with no exhibit/record/rule citation nearby — every factual predicate in a brief must be cited to the record.");
      }
      break;
    }
    case "testimony_answer": {
      if (countSentences(text) > 2) {
        violations.push("Answer runs longer than 2 sentences — a testimony answer should be short and responsive, not a narrative.");
      }
      break;
    }
    case "message_to_other_parent": {
      if (HISTORY_RECAP.test(text)) {
        violations.push("Message recaps relationship history — BIFF messages carry no history, only the current logistical fact and position.");
      }
      if (countSentences(text) > 5) {
        violations.push("Message runs longer than ~5 sentences — BIFF messages should be brief (2-5 sentences).");
      }
      break;
    }
    case "incident_log": {
      if (!FACT_STRUCTURE.test(text)) {
        violations.push("Entry does not use the FACT structure (Facts / Action / Context / Time) — see references/documentation-methods/SKILL.md.");
      }
      break;
    }
    case "objection_to_recommendation": {
      if (!FINDING_TOKEN.test(text)) {
        violations.push("No specific finding or MCR 3.215 cite located — MCR 3.215(E)(4) requires naming the specific finding or application of law objected to, not a blanket objection.");
      }
      break;
    }
    default:
      break;
  }
  return violations;
}

const SEVERITY_WEIGHT: Record<Severity, number> = { stop: 30, fix: 10, soften: 3 };
const PROFILE_VIOLATION_WEIGHT = 8;

export interface RewriteInstruction {
  priority: number;
  instruction: string;
}

function buildRewritePlan(findings: Finding[], profileViolations: string[], docType: DocType, docProfile: DocProfile | undefined): string[] {
  const plan: string[] = [];
  plan.push(`Use the ${docType} template in TEMPLATES.md and the worked examples under "## ${docType}" in EXAMPLES.md.`);
  if (docProfile) plan.push(`Follow the doc-type recipe: ${docProfile.rewrite_recipe}`);

  const stopFindings = findings.filter((f) => f.severity === "stop");
  const fixFindings = findings.filter((f) => f.severity === "fix");
  const softenFindings = findings.filter((f) => f.severity === "soften");

  for (const f of stopFindings) {
    plan.push(`STOP — remove or fully rewrite "${f.excerpt}" (${f.category}): ${f.suggested_rewrite}`);
  }
  for (const f of fixFindings) {
    plan.push(`Fix "${f.excerpt}" (${f.category}): ${f.suggested_rewrite}`);
  }
  for (const f of softenFindings) {
    plan.push(`Soften "${f.excerpt}" (${f.category}): ${f.suggested_rewrite}`);
  }
  for (const v of profileViolations) {
    plan.push(`Structural fix: ${v}`);
  }
  plan.push("Never add a fact the source text did not contain — change how something is said, never what happened.");
  plan.push("Re-run court_language_review on the rewritten text (mode: \"review\") until score >= 90 and stop_flags is empty.");
  return plan;
}

function scoreFrom(findings: Finding[], profileViolations: string[]): number {
  let score = 100;
  for (const f of findings) score -= SEVERITY_WEIGHT[f.severity];
  score -= profileViolations.length * PROFILE_VIOLATION_WEIGHT;
  return Math.max(0, Math.min(100, score));
}

export interface CourtLanguageReviewResult {
  doc_type: DocType;
  mode: "review" | "rewrite_plan";
  score: number;
  stop_flags: string[];
  findings: Finding[];
  profile_violations: string[];
  rewrite_plan: string[];
  safe_phrasebank: string[];
  doc_profile: DocProfile | undefined;
}

export async function reviewCourtLanguage(input: { text: string; doc_type: DocType; mode?: "review" | "rewrite_plan" }): Promise<CourtLanguageReviewResult> {
  const lexicon = await loadLexicon();
  const mode = input.mode ?? "review";
  const findings = await findLexiconMatches(input.text);
  const profileViolations = checkProfileViolations(input.text, input.doc_type);
  const stopFlags = Array.from(new Set(findings.filter((f) => f.severity === "stop").map((f) => f.category)));
  const docProfile = lexicon.doc_profiles[input.doc_type];
  const phrasebanks = await loadPhrasebanks();

  return {
    doc_type: input.doc_type,
    mode,
    score: scoreFrom(findings, profileViolations),
    stop_flags: stopFlags,
    findings,
    profile_violations: profileViolations,
    rewrite_plan: buildRewritePlan(findings, profileViolations, input.doc_type, docProfile),
    safe_phrasebank: (phrasebanks[input.doc_type] ?? []).slice(0, 20),
    doc_profile: docProfile,
  };
}

export function renderMarkdownReport(result: CourtLanguageReviewResult): string {
  const lines: string[] = [];
  lines.push(`# Court-language review — ${result.doc_type} (${result.mode})`);
  lines.push("");
  lines.push(`**Score:** ${result.score}/100${result.stop_flags.length ? ` — STOP flags: ${result.stop_flags.join(", ")}` : " — no stop flags"}`);
  lines.push("");
  if (result.findings.length) {
    lines.push("## Findings");
    for (const f of result.findings) {
      lines.push(`- [${f.severity.toUpperCase()}] ${f.category}: "${f.excerpt}" — ${f.why}`);
    }
    lines.push("");
  } else {
    lines.push("## Findings\nNone.");
    lines.push("");
  }
  if (result.profile_violations.length) {
    lines.push("## Profile violations");
    for (const v of result.profile_violations) lines.push(`- ${v}`);
    lines.push("");
  }
  lines.push("## Rewrite plan");
  for (const step of result.rewrite_plan) lines.push(`1. ${step}`);
  lines.push("");
  lines.push("## Safe phrasebank");
  for (const phrase of result.safe_phrasebank) lines.push(`- ${phrase}`);
  return lines.join("\n");
}

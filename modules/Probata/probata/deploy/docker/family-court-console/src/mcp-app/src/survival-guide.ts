// Byline: Codex / GPT-6 / 2026-10-04 - fresh shared reads; explicit mem:// fixture fallback only.
// Byline: Claude Code · Fable 5.1 · 2026-09-07
//
// survival_guide loader. Owner correction (2026-09-07 09:51): this tool does NOT
// author complete static guides. It resolves a lightweight context pack (JSON,
// one file per event/document type under content/tools/survival-guide/events/),
// the master writing template (or the one-page card template), an optional
// merge of case_facts, and the first ~40 lines of each source file the context
// pack cites — the calling MODEL writes the actual guide from those inputs.
//
// Path resolution mirrors core.ts's pluginRootPath(): this module is bundled to
// dist/survival-guide.js, a sibling of dist/core.js and dist/server.js, both two
// directories below the plugin root that holds content/.
//
// Byline: Claude Code · Sonnet 5 · 2026-09-08 — "content lives in the SurrealDB store":
// the context pack (reference:<event>, kind "event_pack") and the two writing
// templates (reference:survival-guide-template / -card-template, kind "template") go
// store-first with an EXACT file-read fallback. listSurvivalGuideEvents() stays a
// plain filesystem listing on purpose — it fixes the zod enum of valid event ids at
// tool-registration time (buildServer() calls it synchronously), which must not
// depend on an async store round trip.

import { readFileSync, readdirSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { getCaseFacts } from "./core.js";
import { getReference } from "./content-store.js";

function pluginRootPath(...segments: string[]): string {
  const here = dirname(fileURLToPath(import.meta.url));
  return join(here, "..", "..", ...segments);
}

const EVENTS_DIR = pluginRootPath("content", "tools", "survival-guide", "events");
const TEMPLATE_PATH = pluginRootPath("content", "tools", "survival-guide", "TEMPLATE.md");
const CARD_TEMPLATE_PATH = pluginRootPath("content", "tools", "survival-guide", "CARD_TEMPLATE.md");

export interface SurvivalGuideContextPack {
  id: string;
  title: string;
  what_it_is: string;
  who_is_in_the_room?: string[];
  who_reads_it?: string[];
  sequence: string[];
  prepare: string[];
  applicable_rules: Array<{ cite: string; summary: string; source: string; section: string; disposition?: string }>;
  deadlines: Array<{ label: string; rule_preset: string | null; cite: string; status: string }>;
  traps: Array<{ trap: string; safe_move: string }>;
  do_not: string[];
  phrases: { openers: string[]; closers: string[] };
  safety_gates: string[];
  exit_checklist: string[];
  sources: Array<{ file: string; section: string }>;
}

let cachedIds: string[] | null = null;

export function listSurvivalGuideEvents(): string[] {
  if (cachedIds) return cachedIds;
  cachedIds = readdirSync(EVENTS_DIR)
    .filter((name) => name.endsWith(".json"))
    .map((name) => name.slice(0, -5))
    .sort();
  return cachedIds;
}

function loadContextPackFromFile(event: string): SurvivalGuideContextPack {
  const raw = readFileSync(join(EVENTS_DIR, `${event}.json`), "utf8");
  return JSON.parse(raw) as SurvivalGuideContextPack;
}

/** Read the current shared context pack for a known survival-guide event.
 * Inputs: event ID. Outputs: validated context pack.
 * Effects: shared read; missing/malformed production rows throw. Pick this over loadContextPackFromFile for explicit fixtures.
 */
export async function loadContextPack(event: string): Promise<SurvivalGuideContextPack> {
  const ids = listSurvivalGuideEvents();
  if (!ids.includes(event)) {
    throw new Error(`Unknown survival_guide event "${event}". Known events: ${ids.join(", ")}`);
  }
  const ref = await getReference(event);
  if (ref) {
    const data = ref.data as SurvivalGuideContextPack | undefined;
    if (!data || data.id !== event || typeof data.title !== "string" || !Array.isArray(data.sequence)
      || !Array.isArray(data.prepare) || !Array.isArray(data.applicable_rules) || !Array.isArray(data.deadlines)
      || !Array.isArray(data.traps) || !Array.isArray(data.do_not) || !data.phrases
      || !Array.isArray(data.safety_gates) || !Array.isArray(data.exit_checklist) || !Array.isArray(data.sources)) {
      throw new Error(`Malformed shared survival-guide context: ${event}`);
    }
    return data;
  }
  return loadContextPackFromFile(event);
}

function firstLines(path: string, n = 40): string {
  try {
    const text = readFileSync(pluginRootPath(...path.split("/")), "utf8");
    return text.split(/\r?\n/).slice(0, n).join("\n");
  } catch (error) {
    return `[UNAVAILABLE: ${path} — ${error instanceof Error ? error.message : String(error)}]`;
  }
}

export interface SurvivalGuideResult {
  event: string;
  format: "full" | "card" | "json";
  release_warning: string;
  template?: string;
  context_pack: SurvivalGuideContextPack;
  case_facts?: unknown;
  source_excerpts?: Record<string, string>;
}

/** Read the current shared full or card survival-guide template.
 * Inputs: full/card format. Outputs: nonempty template text.
 * Effects: shared read, or absent mem fixture file read. Pick this over direct template file reads only for fixtures.
 */
async function loadTemplate(format: "full" | "card"): Promise<string> {
  const key = format === "card" ? "survival-guide-card-template" : "survival-guide-template";
  const ref = await getReference(key);
  if (ref) {
    if (typeof ref.body !== "string" || !ref.body.length) throw new Error(`Malformed shared survival-guide template: ${key}`);
    return ref.body;
  }
  return readFileSync(format === "card" ? CARD_TEMPLATE_PATH : TEMPLATE_PATH, "utf8");
}

export async function buildSurvivalGuide(input: {
  event: string;
  format?: "full" | "card" | "json";
  include_case_facts?: boolean;
}): Promise<SurvivalGuideResult> {
  const format = input.format ?? "full";
  const contextPack = await loadContextPack(input.event);

  const releaseWarning =
    "Legal information, not legal advice. No attorney-client relationship is created. This is a " +
    "context pack and writing template, not a finished guide — the calling model must write the " +
    "guide by filling every fixed template section from this data, and every deadline/rule line " +
    "stays PROVISIONAL until checked against the current Michigan Court Rules / MCL. If you or a " +
    "child are in immediate danger, call 911. National Domestic Violence Hotline: 1-800-799-7233.";

  const result: SurvivalGuideResult = {
    event: input.event,
    format,
    release_warning: releaseWarning,
    context_pack: contextPack,
  };

  if (format !== "json") {
    result.template = await loadTemplate(format);
  }

  if (input.include_case_facts) {
    result.case_facts = getCaseFacts();
  }

  const excerpts: Record<string, string> = {};
  for (const source of contextPack.sources) {
    excerpts[source.file] = firstLines(source.file, 40);
  }
  result.source_excerpts = excerpts;

  return result;
}

export function renderSurvivalGuideMarkdown(result: SurvivalGuideResult): string {
  const lines: string[] = [];
  lines.push(`# survival_guide — ${result.context_pack.title} (${result.format})`);
  lines.push("");
  lines.push(result.release_warning);
  lines.push("");
  if (result.template) {
    lines.push("## Writing template");
    lines.push(result.template);
    lines.push("");
  }
  lines.push("## Context pack");
  lines.push("```json");
  lines.push(JSON.stringify(result.context_pack, null, 2));
  lines.push("```");
  if (result.case_facts !== undefined) {
    lines.push("");
    lines.push("## Case facts");
    lines.push("```json");
    lines.push(JSON.stringify(result.case_facts, null, 2));
    lines.push("```");
  }
  if (result.source_excerpts) {
    lines.push("");
    lines.push("## Source excerpts (first ~40 lines each)");
    for (const [file, excerpt] of Object.entries(result.source_excerpts)) {
      lines.push(`### ${file}`);
      lines.push("```");
      lines.push(excerpt);
      lines.push("```");
    }
  }
  return lines.join("\n");
}

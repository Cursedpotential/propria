// Byline: Claude Code · Opus 5.5 · 2026-09-26
// Pure helpers for the Review Actions repair builder (option A, ratified 2026-09-25:
// docs/pending-review/2026-09-25-repair-workflow-builder.md). The engine owns the tool
// registry, the proposals and every validation rule; this file only shapes the owner's
// draft into the contract's plan and reads the repaired-run link from a run's history.
// No I/O and no runtime imports, so the smoke tests import it directly.
import type {
  MatterMode,
  ProfferOperationStage,
  ProfferRepairParamValue,
  ProfferRepairParamsSchema,
  ProfferRepairPlan,
  ProfferRepairProposal,
  ProfferRepairStep,
} from "@/lib/shared/types";

/** The engine's bound: a plan has 1-12 steps (validator rule `bounded`). */
export const MAX_REPAIR_STEPS = 12;

/** One step of the owner's draft. `key` is a stable React identity; step ids follow the order. */
export interface RepairDraftStep {
  key: string;
  activity: string;
  params: Record<string, ProfferRepairParamValue>;
}

let draftKeys = 0;

function nextKey() {
  draftKeys += 1;
  return `repair-step-${draftKeys}`;
}

export function draftStep(activity: string, params: Record<string, ProfferRepairParamValue> = {}): RepairDraftStep {
  return { key: nextKey(), activity, params: { ...params } };
}

export function draftFromProposal(proposal: ProfferRepairProposal): RepairDraftStep[] {
  return proposal.steps.slice(0, MAX_REPAIR_STEPS).map((step) => draftStep(step.activity, step.params));
}

export function addStep(steps: readonly RepairDraftStep[], activity: string): RepairDraftStep[] {
  return steps.length >= MAX_REPAIR_STEPS ? [...steps] : [...steps, draftStep(activity)];
}

export function removeStep(steps: readonly RepairDraftStep[], index: number): RepairDraftStep[] {
  return steps.filter((_, position) => position !== index);
}

/** Moves one step up (-1) or down (+1); a move past either end changes nothing. */
export function moveStep(steps: readonly RepairDraftStep[], index: number, delta: -1 | 1): RepairDraftStep[] {
  const target = index + delta;
  if (index < 0 || index >= steps.length || target < 0 || target >= steps.length) return [...steps];
  const next = [...steps];
  [next[index], next[target]] = [next[target], next[index]];
  return next;
}

export function updateStepParams(
  steps: readonly RepairDraftStep[],
  index: number,
  params: Record<string, ProfferRepairParamValue>,
): RepairDraftStep[] {
  return steps.map((step, position) => (position === index ? { ...step, params } : step));
}

/** Step ids are the order: s1, s2, ... (the validator's reasons name steps by these ids). */
export function stepId(index: number) {
  return `s${index + 1}`;
}

/** A plan id the engine accepts (8-96 URL-safe characters), new for every draft. */
export function newPlanId(previewHandle: string, now = Date.now(), random = Math.random()) {
  const anchor = previewHandle.replace(/[^A-Za-z0-9_-]/g, "").slice(0, 12) || "run";
  const salt = Math.floor(random * 36 ** 4).toString(36).padStart(4, "0");
  return `rp-${anchor}-${now.toString(36)}${salt}`;
}

export function planFromDraft(input: {
  planId: string;
  sourceRef: string;
  previewHandle: string;
  mode: MatterMode;
  steps: readonly RepairDraftStep[];
}): ProfferRepairPlan {
  const steps: ProfferRepairStep[] = input.steps.map((step, index) => ({
    step_id: stepId(index),
    activity: step.activity,
    params: { ...step.params },
  }));
  return {
    plan_id: input.planId,
    source_ref: input.sourceRef,
    preview_handle: input.previewHandle,
    matter_mode: input.mode,
    steps,
  };
}

/** Identity of exactly what a plan asks for: a validation result belongs to one key. */
export function planKey(plan: ProfferRepairPlan) {
  return JSON.stringify([
    plan.plan_id,
    plan.source_ref,
    plan.preview_handle,
    plan.matter_mode,
    plan.steps.map((step) => [step.step_id, step.activity, Object.entries(step.params).sort(([a], [b]) => a.localeCompare(b))]),
  ]);
}

export type RepairParamKind = "integer" | "number" | "boolean" | "string" | "choice";

export interface RepairParamField {
  name: string;
  kind: RepairParamKind;
  description: string;
  required: boolean;
  minimum?: number;
  maximum?: number;
  choices?: ProfferRepairParamValue[];
  defaultValue?: ProfferRepairParamValue;
}

/** One editable field per params_schema property, in the schema's order. */
export function paramFields(schema: ProfferRepairParamsSchema | null | undefined): RepairParamField[] {
  const required = new Set(schema?.required ?? []);
  return Object.entries(schema?.properties ?? {}).map(([name, property]) => {
    const choices = property.enum && property.enum.length > 0 ? [...property.enum] : undefined;
    const kind: RepairParamKind = choices ? "choice" : property.type;
    return {
      name,
      kind,
      description: property.description ?? "",
      required: required.has(name),
      minimum: property.minimum,
      maximum: property.maximum,
      choices,
      defaultValue: property.default,
    };
  });
}

/** Sets one parameter; `undefined` removes it so the tool's own default applies. */
export function setParam(
  params: Record<string, ProfferRepairParamValue>,
  name: string,
  value: ProfferRepairParamValue | undefined,
): Record<string, ProfferRepairParamValue> {
  const next = { ...params };
  if (value === undefined) delete next[name];
  else next[name] = value;
  return next;
}

/** A number field's text as a parameter value: empty is "not set", never zero. */
export function numberParam(text: string): number | undefined {
  if (!text.trim()) return undefined;
  const value = Number(text);
  return Number.isFinite(value) ? value : undefined;
}

/** True when nothing proposed has a step to run (only the wait option, or nothing). */
export function noStepsProposed(proposals: readonly ProfferRepairProposal[]) {
  return proposals.every((proposal) => proposal.steps.length === 0);
}

/** The one line shown when no proposal has steps. A clean file is not called unrepairable. */
export function noRepairLine(signature: string) {
  return signature.endsWith(":clean")
    ? "Nothing to repair: the repair check found this file clean."
    : "No known repair for this file type";
}

/** The receipt the engine writes to the Review run it repaired (repairplan.ReentryReceiptActivity). */
export const REPAIR_REENTRY_ACTIVITY = "repair.reentry";

// How the engine names what a repair re-entered (modules/engine/repairplan/workflow.go `reenter`):
// a single run's workflow id (its request id) is `<repair workflow id>-reentry-<first 8 of the run
// id>`; a batch's workflow id is its batch id, `repair-<40 hex>`. The operation history carries
// that id as the receipt's `ref`.
const SINGLE_RUN_REENTRY = /^(repair-plan-[A-Za-z0-9_-]+)-reentry-[A-Za-z0-9]{1,8}$/;
const REENTRY_BATCH = /^repair-[0-9a-f]{40}$/;

export type RepairReentryLink =
  | { kind: "run"; repairWorkflowId: string; receiptRef: string | null }
  | { kind: "batch"; batchId: string; receiptRef: string | null };

/** The newest successful re-entry recorded in a run's operation history, if any. */
export function repairReentryFromStages(stages: readonly ProfferOperationStage[]): RepairReentryLink | null {
  const latest = [...stages]
    .reverse()
    .find((stage) => stage.stage === REPAIR_REENTRY_ACTIVITY && stage.status === "success" && stage.ref);
  const ref = latest?.ref?.trim() ?? "";
  const receiptRef = latest?.receipt_ref ?? null;
  const run = SINGLE_RUN_REENTRY.exec(ref);
  if (run && run[1].length <= 160) return { kind: "run", repairWorkflowId: run[1], receiptRef };
  if (REENTRY_BATCH.test(ref)) return { kind: "batch", batchId: ref, receiptRef };
  return null;
}

const TOOL_LABEL: Record<string, string> = {
  "repair.find_other_version": "Find another copy",
  "repair.salvage_truncated_xml": "Salvage to the last complete record",
  "repair.lenient_decode": "Decode leniently",
};

/** Plain words for a registered repair tool; an unknown id is shown as itself. */
export function toolLabel(activity: string) {
  return TOOL_LABEL[activity] ?? activity;
}

const RULE_LABEL: Record<string, string> = {
  anchor_resolves: "Belongs to this run",
  registered_activity: "Every step is a registered tool",
  params_valid: "Settings are valid",
  type_chain: "Each step can read what it is given",
  original_never_written: "The original is never written",
  reentry_ingestible: "The result can be imported again",
  destination_resolves: "The output location exists",
  test_live_mode: "Test / Live matches",
  bounded: "Bounded: 1 to 12 steps",
};

/** Plain words for a validator rule; an unknown rule is shown by its name. */
export function ruleLabel(rule: string) {
  return RULE_LABEL[rule] ?? rule.replaceAll("_", " ");
}

/** The last path segment of a locator, for a person to recognise. */
export function refName(ref: string | null | undefined) {
  if (!ref) return "";
  const segments = ref.split("/").filter(Boolean);
  return segments.at(-1) ?? ref;
}

/** A step's bounded summary as one short line of "name value" pairs. */
export function summaryLine(summary: unknown, limit = 6): string {
  if (summary === null || summary === undefined) return "";
  if (typeof summary !== "object" || Array.isArray(summary)) return String(JSON.stringify(summary)).slice(0, 160);
  const parts = Object.entries(summary as Record<string, unknown>)
    .filter(([, value]) => value === null || ["string", "number", "boolean"].includes(typeof value))
    .slice(0, limit)
    .map(([name, value]) => `${name.replaceAll("_", " ")} ${String(value)}`);
  return parts.join(" · ");
}

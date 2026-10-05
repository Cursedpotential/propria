// Byline: Codex · GPT-6-Luna · 2026-10-04.
// Bounded convergence tests compile the owned TypeScript with the app's canonical esbuild API, write:false.
import assert from "node:assert/strict";
import { mkdirSync, writeFileSync } from "node:fs";
import { createRequire } from "node:module";
import { dirname, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import test from "node:test";

const app = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const require = createRequire(resolve(process.env.FCT_TEST_ESBUILD_ROOT ?? app, "package.json"));
const { build } = require("esbuild");
const artifacts = resolve(app, "to_be_deleted", "shared-catalog-tests", `${Date.now()}-${process.pid}`);
mkdirSync(artifacts, { recursive: true });
const state = {
  sources: [],
  references: new Map(),
  events: [],
  queries: [],
  fileReads: 0,
  directoryReads: 0,
  summary: null,
  caseFile: null,
  deadlineConfig: null,
  deadlineConfigVersion: "c".repeat(64),
  missingVersionRefs: new Set(),
};
globalThis.__sharedCatalogTest = state;

/** Compile current owned modules against isolated read-only adapters.
 * Inputs: none. Outputs: an imported test artifact compiled by esbuild with write:false.
 * Effects: writes only the retained ignored test bundle beneath to_be_deleted; no dist/build output or real store access.
 * Pick over the full app build for this focused shared-source contract test.
 */
async function loadOwnedModules() {
  const result = await build({
    stdin: {
      contents: 'export * from "./src/core.ts"; export * from "./src/survival-guide.ts";',
      resolveDir: app,
      sourcefile: "shared-catalog-entry.ts",
    },
    bundle: true,
    platform: "node",
    format: "esm",
    target: "node20",
    write: false,
    plugins: [{ name: "shared-catalog-read-fixtures", setup(builder) {
      builder.onResolve({ filter: /^\.\/store\.js$/ }, () => ({ path: "store", namespace: "fixture" }));
      builder.onResolve({ filter: /^\.\/content-store\.js$/ }, () => ({ path: "content-store", namespace: "fixture" }));
      builder.onResolve({ filter: /^node:fs$/ }, () => ({ path: "fs", namespace: "fixture" }));
      builder.onLoad({ filter: /.*/, namespace: "fixture" }, ({ path }) => {
        if (path === "store") return { contents: `
          export const normalize = (value) => value;
          export async function getStore() {
            const state = globalThis.__sharedCatalogTest;
            return { available: true, db: { query: async (sql, params = {}) => {
              state.queries.push({ sql, params });
              const rows = state.events.filter((row) => params.after === undefined || row.content_cursor > params.after).slice(0, 100);
              return [rows.map((row) => ({ ...row }))];
            } } };
          }
          export async function caseSummary() { return globalThis.__sharedCatalogTest.summary; }
          export async function caseRecord(_store, ref) {
            const state = globalThis.__sharedCatalogTest;
            const id = ref.table + ":" + ref.id;
            if (id === "reference:deadline-rule-presets") {
              if (state.deadlineConfig === null) return null;
              return { id, version: "sha256:" + state.deadlineConfigVersion, record: state.deadlineConfig };
            }
            if (state.missingVersionRefs.has(id)) return null;
            return { id, version: "sha256:" + "b".repeat(64), record: {} };
          }
        ` };
        if (path === "content-store") return { contents: `
          export async function getSources() { return globalThis.__sharedCatalogTest.sources.map((row) => ({ ...row })); }
          export async function getReference(key) { return globalThis.__sharedCatalogTest.references.get(key) ?? null; }
          export async function getReferenceExcerpt(source_path, requested_pinpoint) {
            return { source_path, reference_id: null, source_sha256: null, record_version: null,
              requested_pinpoint, resolution_status: "source_gap", gap_reason: "fixture_source_not_seeded",
              resolved_heading: null, excerpt: "", excerpt_truncated: false };
          }
        ` };
        return { contents: `
          export function readFileSync() {
            const state = globalThis.__sharedCatalogTest;
            if (state.caseFile !== null) return JSON.stringify(state.caseFile);
            state.fileReads++;
            throw new Error("packaged fallback forbidden in this test");
          }
          export function readdirSync() { globalThis.__sharedCatalogTest.directoryReads++; return ["packaged-only.json"]; }
        ` };
      });
    } }],
  });
  const output = resolve(artifacts, "shared-catalog-entry.mjs");
  writeFileSync(output, result.outputFiles[0].contents);
  return import(pathToFileURL(output).href);
}

/** Create a valid synthetic shared event pack for discovery tests.
 * Inputs: stable event id and optional title. Outputs: the minimal complete pack accepted by the production validator.
 * Effects: none. Pick over packaged JSON so fixture identity is controlled by the shared catalog adapter.
 */
function eventPack(id, title = `Shared ${id}`) {
  return {
    id, title, what_it_is: "Synthetic shared fixture", sequence: [], prepare: [], applicable_rules: [],
    deadlines: [], traps: [], do_not: [], phrases: { openers: [], closers: [] }, safety_gates: [],
    exit_checklist: [], sources: [],
  };
}

const api = await loadOwnedModules();

test.after(() => delete globalThis.__sharedCatalogTest);

test("Dewey proposals retain exact identities, dated gaps, and non-validation status", () => {
  const proposals = api.ruleCorrectionProposalsForCitations(["Rule 3.215 Domestic Relations Referees", "MCR 3.215(F)(2)"]);
  assert.equal(proposals.length, 5);
  assert.ok(proposals.every((proposal) => proposal.rule_identity === "MCR 3.215"));
  assert.ok(proposals.every((proposal) => proposal.rule_heading === "Rule 3.215 Domestic Relations Referees"));
  assert.ok(proposals.every((proposal) => proposal.subdivision_structure === "subdivisions_are_lists"));
  assert.ok(proposals.every((proposal) => proposal.authority_status === "PROVISIONAL_CURRENCY_NOT_CLEARED"));
  assert.ok(proposals.every((proposal) => proposal.proposal_status === "PROPOSED_NOT_APPLIED"));
  assert.ok(proposals.some((proposal) => proposal.source_locations.includes("M16 draft line 232") && /live-evidence opportunity/.test(proposal.proposed_correction)));
  assert.ok(proposals.some((proposal) => proposal.source_locations.includes("P1 draft line 313") && /party-requested limitation/.test(proposal.proposed_correction) && /transcript ordered/.test(proposal.proposed_correction)));
  assert.ok(proposals.every((proposal) => proposal.recorded_on === "2026-10-04" && proposal.audit_date === "2026-10-04"));
  assert.ok(proposals.every((proposal) => proposal.audit_source === "Dewey official-source audit"));
  const mail = api.ruleCorrectionProposalsForCitations(["MCR 2.107(C)(3)"])[0];
  assert.equal(mail.authority_status, "NOT_STATED_IN_SUPPLIED_AUDIT");
  assert.match(mail.proposed_correction, /complete upon mailing/);
  const blocked = api.ruleCorrectionProposalsForCitations(["MCL 552.507(4)"])[0];
  assert.equal(blocked.authority_status, "BLOCKED");
  assert.equal(blocked.reported_pinpoint, "MCL 552.507");
  assert.equal(blocked.source_http_status, 403);
  assert.equal(blocked.pinpoint_status, "unresolved");
});

test("shared source audit/search use exact live rows and expose source/version gaps", async () => {
  state.sources = [{
    id: "source:mcr-3215", title: "Rule 3.215 Domestic Relations Referees", citation: "MCR 3.215(F)(2)",
    issuing_body: "Michigan Supreme Court", binding_status: "PROVISIONAL_CURRENCY_NOT_CLEARED", official_url: "https://example.invalid/rule",
    sha256: "a".repeat(64), last_verified: "2026-10-04", scope_note: "Referee proceedings",
  }];
  const first = await api.auditSources();
  assert.equal(first.sources.length, 1);
  assert.equal(first.sources[0].record_ref, "source:mcr-3215");
  assert.equal(first.sources[0].source_sha256, "a".repeat(64));
  assert.equal(first.sources[0].record_version, null);
  assert.equal(first.sources[0].record_version_status, "not_loaded");
  assert.ok(first.notes.some((note) => note.includes("unfiltered catalog omits per-row caseRecord lookups")));
  assert.equal(first.release_status.label, "PENDING_SHARED_RELEASE_AUDIT_RECORD");
  assert.equal((await api.auditSources(["mcr-3215"])).sources.length, 1);
  assert.equal((await api.auditSources(["MCR-3215"])).sources.length, 0, "identity filtering is exact, not fuzzy");
  const linked = await api.getSharedRuleCatalogContext(["MCR 3.215(F)(2)", "MCR 3.215(E)(4)"]);
  assert.equal(linked.rule_source_traceability[0].traceability_status, "EXACT_SHARED_SOURCE_RECORD");
  assert.equal(linked.rule_source_traceability[0].shared_sources[0].record_ref, "source:mcr-3215");
  assert.equal(linked.rule_source_traceability[1].traceability_status, "GAP_NO_EXACT_SHARED_SOURCE_RECORD");
  assert.equal(linked.rule_source_traceability[0].shared_sources[0].record_version, `sha256:${"b".repeat(64)}`);
  assert.notEqual(linked.rule_source_traceability[0].shared_sources[0].source_sha256, linked.rule_source_traceability[0].shared_sources[0].record_version);

  state.sources[0].title = "Updated shared rule record";
  assert.equal((await api.searchRecords("updated shared rule record"))[0].title, "Updated shared rule record");
  assert.equal(state.fileReads, 0, "no packaged source-catalog fallback was used");
  state.sources.push({ id: "source:release-audit-20261004", record_kind: "release_audit", release_state: "PENDING_COUNSEL_REVIEW" });
  const release = (await api.getSharedRuleCatalogContext([])).release_status;
  assert.equal(release.label, "PENDING_COUNSEL_REVIEW");
  assert.equal(release.record_ref, "source:release-audit-20261004");
  assert.equal(release.record_version, `sha256:${"b".repeat(64)}`);
});

test("shared event discovery pages beyond 100 records and observes later catalog updates", async () => {
  state.events = Array.from({ length: 237 }, (_, index) => {
    const key = `shared-event-${String(index).padStart(3, "0")}`;
    return { content_cursor: key, key, kind: "event_pack" };
  });
  state.queries.length = 0;
  const ids = await api.discoverSharedSurvivalGuideEvents();
  assert.equal(ids.length, 237);
  assert.equal(ids[0], "shared-event-000");
  assert.equal(ids.at(-1), "shared-event-236");
  assert.equal(state.queries.length, 3);
  assert.ok(state.queries.every(({ sql }) => /LIMIT 100/.test(sql)));
  state.references.set("shared-event-236", { key: "shared-event-236", kind: "event_pack", data: eventPack("shared-event-236", "Updated after discovery") });
  assert.equal((await api.loadContextPack("shared-event-236")).title, "Updated after discovery");
  state.events.push({ content_cursor: "shared-event-237", key: "shared-event-237", kind: "event_pack" });
  assert.equal((await api.discoverSharedSurvivalGuideEvents()).at(-1), "shared-event-237");
  assert.equal(state.directoryReads, 0, "runtime discovery did not consult packaged filenames");
});

test("named deadline rules use fresh shared configuration, provenance, and no compiled fallback", async () => {
  const previousSources = state.sources;
  const config = (presets) => ({
    key: "deadline-rule-presets",
    kind: "deadline_rule_presets",
    data: { schema: "propria.deadline-rule-presets.v1", presets },
  });
  state.sources = [{
    id: "source:mcr-test-1-2", title: "Synthetic Rule 1.2", citation: "MCR 1.2",
    issuing_body: "Synthetic authority", binding_status: "PROVISIONAL_CURRENCY_NOT_CLEARED",
    official_url: "https://example.invalid/rule", sha256: "e".repeat(64), last_verified: "2026-10-04",
  }];
  state.deadlineConfig = config([{ id: "fixture_before", days: 2, direction: "before", citation: "MCR 1.2", status: "DRAFT_PENDING_REVIEW" }]);
  state.deadlineConfigVersion = "c".repeat(64);

  const first = await api.calculateSharedDeadlinePreset({ anchorDate: "2026-10-05", rule: "fixture_before" });
  assert.equal(first.configured, true);
  assert.equal(first.rawDate, "2026-10-03");
  assert.equal(first.adjustedDate, "2026-10-02");
  assert.equal(first.cite, "MCR 1.2");
  assert.equal(first.configured_status, "DRAFT_PENDING_REVIEW");
  assert.deepEqual(first.preset_provenance.configuration_record, {
    record_ref: "reference:deadline-rule-presets",
    record_version: `sha256:${"c".repeat(64)}`,
    record_hash: "c".repeat(64),
    hash_basis: "caseRecord full shared row with embedding omitted",
    schema: "propria.deadline-rule-presets.v1",
  });
  assert.equal(first.preset_provenance.shared_rule_traceability.traceability_status, "EXACT_SHARED_SOURCE_RECORD");

  state.deadlineConfig = config([
    { id: "fixture_before", days: 4, direction: "before", citation: "MCR 1.2", status: "UPDATED_DRAFT" },
    { id: "new_shared_rule", days: 1, direction: "after", citation: "MCR 1.2", status: "PENDING_REVIEW" },
  ]);
  state.deadlineConfigVersion = "d".repeat(64);
  const updated = await api.calculateSharedDeadlinePreset({ anchorDate: "2026-10-05", rule: "fixture_before" });
  assert.equal(updated.rawDate, "2026-10-01", "the current shared day count is read on every invocation");
  assert.equal(updated.configured_status, "UPDATED_DRAFT");
  assert.equal(updated.preset_provenance.configuration_record.record_version, `sha256:${"d".repeat(64)}`);
  const added = await api.calculateSharedDeadlinePreset({ anchorDate: "2026-10-05", rule: "new_shared_rule" });
  assert.equal(added.rawDate, "2026-10-06", "a newly shared preset is immediately resolvable");

  const unknown = await api.calculateSharedDeadlinePreset({ anchorDate: "2026-10-05", rule: "not_configured" });
  assert.equal(unknown.status, "UNKNOWN_SHARED_DEADLINE_RULE_PRESET");
  assert.equal("rawDate" in unknown, false);
  state.deadlineConfig = null;
  const missing = await api.calculateSharedDeadlinePreset({ anchorDate: "2026-10-05", rule: "referee_objection" });
  assert.equal(missing.status, "PENDING_SHARED_DEADLINE_RULE_PRESET_CONFIGURATION");
  assert.equal("rawDate" in missing, false, "compiled values do not fill a missing shared rule");
  assert.equal(api.calculateDirectionalDeadline({ anchorDate: "2026-10-05", days: 2, direction: "after" }).rawDate, "2026-10-07");
  state.sources = previousSources;
});

test("shared survival-guide packs/templates have visible missing gaps and never fall back to package files", async () => {
  state.events = [{ content_cursor: "referee-hearing", key: "referee-hearing", kind: "event_pack" }];
  state.references.set("referee-hearing", { key: "referee-hearing", kind: "event_pack", data: eventPack("referee-hearing") });
  state.references.delete("survival-guide-template");
  await assert.rejects(api.buildSurvivalGuide({ event: "referee-hearing", format: "full" }), /template missing/);
  assert.equal(state.fileReads, 0);
  assert.equal((await api.loadContextPack("referee-hearing")).id, "referee-hearing");
  state.events = [];
  await assert.rejects(api.discoverSharedSurvivalGuideEvents(), /catalog is empty/);
  assert.equal(state.directoryReads, 0);
});

test("survival-guide output reports current release state and concrete rule-source gaps without blanket warnings", async () => {
  state.events = [{ content_cursor: "referee-hearing", key: "referee-hearing", kind: "event_pack" }];
  const pack = eventPack("referee-hearing");
  pack.applicable_rules = [{ cite: "MCR 3.215", summary: "Synthetic citation", source: "fixture", section: "test" }];
  state.references.set("referee-hearing", { key: "referee-hearing", kind: "event_pack", data: pack });
  const guide = await api.buildSurvivalGuide({ event: "referee-hearing", format: "json" });
  assert.equal(guide.release_status.label, "PENDING_COUNSEL_REVIEW", "current versioned shared audit status is surfaced verbatim");
  assert.equal(guide.release_status.record_ref, "source:release-audit-20261004");
  assert.equal("release_warning" in guide, false);
  const markdown = api.renderSurvivalGuideMarkdown(guide);
  assert.match(markdown, /Shared release status/);
  assert.match(markdown, /Rule-source traceability gaps[\s\S]*MCR 3\.215: GAP_NO_EXACT_SHARED_SOURCE_RECORD/);
  assert.doesNotMatch(markdown, /attorney-client relationship|call 911|stays PROVISIONAL/);
});

test("shared case facts retain full private party and child rows with case_context", async () => {
  const child = { id: "child:fixture", name: "Child Example", initials: "C.E.", age: 9, custom_field: { school: "Fixture School" } };
  const caseContext = {
    court: { id: "court:main", name: "Genesee Circuit Court", custom_field: "court-private-detail" },
    people: [{ id: "person:fixture", name: "Parent Example", role: "party", custom_field: { contact_note: "fixture-private-value" } }],
    children: [child],
  };
  state.summary = {
    configured: true, county: "Genesee", court: "Circuit Court", judge: "Judge", referee: null,
    controlling_orders: [], next_hearing: null, deadlines: [], parties: ["Parent Example"],
    children: { count: 1, entries: [child] }, case_context: caseContext, flags: [],
    source: "surrealdb-case-store", counts: { child: 1 },
  };
  const facts = await api.getSharedCaseFacts();
  assert.equal(facts.county, "Genesee");
  assert.deepEqual(facts.parties, ["Parent Example"]);
  assert.deepEqual(facts.children.entries, [child]);
  assert.deepEqual(facts.case_context, caseContext);
  assert.equal(facts.case_context.people[0].name, "Parent Example");
  assert.deepEqual(facts.case_context.people[0].custom_field, { contact_note: "fixture-private-value" });
  assert.deepEqual(facts.case_context.children[0].custom_field, { school: "Fixture School" });
  assert.ok(String(facts.traceability_gap).includes("does not expose a version/hash"));
  state.summary = {
    configured: true, county: null, court: null, judge: null, referee: null, controlling_orders: [],
    next_hearing: null, deadlines: [], parties: [], children: { count: 0, entries: [] }, flags: [],
    source: "surrealdb-case-store", counts: { source: 5 },
  };
  assert.equal((await api.getSharedCaseFacts()).status, "SHARED_CASE_SUMMARY_EMPTY");
});

test("legacy case facts preserve full names and unknown private party/child fields", () => {
  const party = { name: "Parent Example", initials: "P.E.", role: "party", private_custom_field: "retained-party-value" };
  const child = { name: "Child Example", initials: "C.E.", age: 9, private_custom_field: "retained-child-value" };
  state.caseFile = { county: "Genesee", parties: [party], children: [child] };
  const facts = api.getCaseFacts();
  assert.equal(facts.configured, true);
  assert.deepEqual(facts.parties, [party]);
  assert.deepEqual(facts.children.entries, [child]);
  state.caseFile = null;
});

// Byline: Codex, 2026-10-04.
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { resolve } from "node:path";
import test from "node:test";
import { transformSync } from "esbuild";

/** Run the production library view with bounded UI and transport seams.
 * Inputs: a page-list callback; outputs: appended nodes and observed table requests.
 * Side effects: reads and transforms one source function; no browser or datastore is opened.
 * Choose for verifying working-library routing independently of packaged originals.
 */
async function runLibraryView(load) {
  const source = (await readFile(resolve("web/host.ts"), "utf8")).replace(/\r\n/g, "\n");
  const start = source.indexOf("async function viewLibrary():");
  const end = source.indexOf("\n// Byline: Codex, 2026-10-04.\n/** Browse the files", start);
  assert.ok(start >= 0 && end > start, "production view has a bounded source boundary");
  const code = transformSync(source.slice(start, end), { loader: "ts", target: "es2022" }).code;
  const nodes = [];
  const requests = [];
  const host = { append: (...items) => nodes.push(...items) };
  const view = new Function("page", "el", "pagedRecordList", "errBox", "viewPackagedFiles",
    code + "; return viewLibrary;")(
    () => host,
    (tag, attrs, ...children) => ({ tag, attrs, children }),
    async (_host, tables, ...args) => { requests.push(tables); await load(...args); },
    (error) => ({ error: error.message }),
    () => { throw new Error("packaged view must require an explicit click"); },
  );
  await view();
  return { nodes, requests };
}

/** Verify that the Library reads shared source/reference pages instead of packaged files.
 * Inputs: synthetic page loader; outputs: assertion result.
 * Side effects: none outside source-function evaluation.
 * Choose to catch surface drift even when both stores have identical record titles.
 */
test("working Library loads shared reference and source records", async () => {
  const result = await runLibraryView(async () => {});
  assert.deepEqual(result.requests, [["reference", "source"]]);
  assert.equal(result.nodes.some((node) => node.error), false);
  assert.equal(result.nodes[0].children[0].children[0], "Packaged files");
});

/** Verify that a shared-store error remains visible without a file fallback.
 * Inputs: synthetic store failure; outputs: assertion result.
 * Side effects: none; packaged-file loader is never invoked.
 * Choose alongside successful routing to prevent an outage from displaying stale bundled content.
 */
test("working Library shows store failure without opening packaged content", async () => {
  const result = await runLibraryView(async () => { throw new Error("shared library unavailable"); });
  assert.deepEqual(result.requests, [["reference", "source"]]);
  assert.ok(result.nodes.some((node) => node.error === "shared library unavailable"));
});

/** Verify navigation uses the source catalog's official locator and rejects unsafe schemes.
 * Inputs: bounded synthetic source records; outputs: assertion result.
 * Side effects: transforms one pure production helper; no network requests occur.
 * Choose for the catalog field mismatch that previously left official sources without a link.
 */
test("source details prefer official URLs and reject non-web or credential-bearing locators", async () => {
  const source = (await readFile(resolve("web/host.ts"), "utf8")).replace(/\r\n/g, "\n");
  const start = source.indexOf("export function officialSourceUrl(");
  const end = source.indexOf("\nasync function openRecord(", start);
  const code = transformSync(source.slice(start, end), { loader: "ts", format: "cjs" }).code;
  const module = { exports: {} };
  new Function("module", "exports", code)(module, module.exports);
  const url = module.exports.officialSourceUrl;
  assert.equal(url({ official_url: "https://www.courts.michigan.gov/example", url: "https://mirror.example/" }), "https://www.courts.michigan.gov/example");
  assert.equal(url({ official_url: "javascript:alert(1)" }), null);
  assert.equal(url({ official_url: "https://user:secret@example.org/" }), null);
  assert.equal(url({ official_url: "bad URL", source_url: "https://example.org/source" }), "https://example.org/source");
});

/** Load the pure production citation/proposal gates without starting the app server.
 * Inputs: none; output: helpers transformed from their source module.
 * Side effects: reads and compiles the bounded helper block only; no network or database access occurs.
 * Choose for verifying exact-version proposal arguments independently of phone rendering.
 */
async function loadProposalHelpers() {
  const source = (await readFile(resolve("web/host.ts"), "utf8")).replace(/\r\n/g, "\n");
  const start = source.indexOf("export async function resolveLibraryCitations(");
  const end = source.indexOf("\nasync function openRecord(", start);
  assert.ok(start >= 0 && end > start, "proposal helpers have bounded source boundaries");
  const code = transformSync(source.slice(start, end), { loader: "ts", format: "cjs", target: "es2022" }).code;
  const module = { exports: {} };
  new Function("module", "exports", code)(module, module.exports);
  return module.exports;
}

/** Require exact source case_record versions and retain a full personalized patch in proposal args.
 * Inputs: synthetic records and citations; output: assertion result.
 * Side effects: the injected reader records bounded case_record lookups only.
 * Choose to catch any UI path that accepts user-supplied citation versions or drops record fields.
 */
test("library proposals resolve current source versions and preserve complete record patches", async () => {
  const { resolveLibraryCitations, buildLibraryProposalArguments } = await loadProposalHelpers();
  const reads = [];
  const version = `sha256:${"a".repeat(64)}`;
  const citations = await resolveLibraryCitations("reference:fixture", version, [
    { source_id: "source:fixture", pinpoint: "p. 4", claim: "Supports the synthetic claim." },
  ], async (id) => { reads.push(id); return { found: true, id, version, record: { body: "synthetic" } }; });
  const patch = { title: "Synthetic reference", body: "Full personal-field fixture content", private_context: { names: ["Fixture Person"] } };
  const args = buildLibraryProposalArguments("reference:fixture", version, patch, citations, "Correct the synthetic citation.");
  assert.deepEqual(reads, ["source:fixture"]);
  assert.equal(citations[0].source_version, version);
  assert.deepEqual(args.patch, patch);
  assert.deepEqual(Object.keys(args).sort(), ["citations", "expected_version", "id", "patch", "rationale"]);
  assert.equal("receipt" in args, false);
});

/** Send only editable differences while preserving omission and explicit-null semantics.
 * Inputs: full server record and edited JSON fixtures; output: assertion result for the pure production patch helpers.
 * Side effects: none; fixtures include no live data or datastore calls.
 * Choose to keep server identity/embeddings out of proposals without dropping personal fields or rewriting unchanged timestamps.
 */
test("library editor strips server fields and submits only changed editable keys", async () => {
  const { editableLibraryPatch, changedLibraryPatch } = await loadProposalHelpers();
  const fullRecord = {
    id: "reference:fixture", embedding: [0.2, 0.4], validation: { status: "old" }, validation_status: "old", published_at: "server-time",
    title: "Before", body: "Full personal content", private_context: { names: ["Fixture Person"] }, created_at: "2026-10-01T12:00:00.000Z",
    source_path: "fixtures/source.md", nullable_field: "present",
  };
  const initial = editableLibraryPatch(fullRecord);
  assert.equal("id" in initial, false);
  assert.equal("embedding" in initial, false);
  assert.equal("validation" in initial, false);
  assert.equal("validation_status" in initial, false);
  assert.equal("published_at" in initial, false);
  assert.deepEqual(initial.private_context, fullRecord.private_context);
  const edited = { ...initial, title: "After", nullable_field: null, id: fullRecord.id, embedding: [9], validation: { status: "forged" } };
  const patch = changedLibraryPatch(initial, edited);
  assert.deepEqual(patch, { title: "After", nullable_field: null });
  assert.equal("created_at" in patch, false, "unchanged normalized timestamp is not rewritten");
  assert.equal("body" in patch, false, "omitted keys remain unchanged in the stored base");
  assert.equal("private_context" in patch, false, "unchanged personal data stays in the shared base");
});

/** Permit only the approved new-source self-citation exception and reject missing exact records.
 * Inputs: synthetic new-source/reference citations and shared-reader results; output: assertion result.
 * Side effects: only the injected reader is called; no tool or store mutation occurs.
 * Choose to protect creation flows from invented citation versions and absent non-self sources.
 */
test("new source self-capture uses absent and other citations require an exact record", async () => {
  const { resolveLibraryCitations } = await loadProposalHelpers();
  let reads = 0;
  const self = await resolveLibraryCitations("source:new-fixture", "absent", [
    { source_id: "source:new-fixture", pinpoint: "captured URL", claim: "This is the source being proposed." },
  ], async () => { reads += 1; throw new Error("self-capture must not fetch a published row"); });
  assert.equal(self[0].source_version, "absent");
  assert.equal(reads, 0);
  await assert.rejects(resolveLibraryCitations("reference:new-fixture", "absent", [
    { source_id: "source:missing", pinpoint: "p. 1", claim: "Claim." },
  ], async () => ({ found: false, id: "source:missing" })), /no exact shared record version/);
});

/** Offer publication only from the exact server-read proposal record after currency clearance.
 * Inputs: synthetic proposal snapshots; output: assertion result.
 * Side effects: none; this helper is a display gate and never calls the publish tool.
 * Choose to keep pending/provisional or mismatched records from exposing the publish action.
 */
test("publish affordance requires the exact proposal version and server-cleared currency", async () => {
  const { canonicalLibraryProposalId, canPublishLibraryProposal } = await loadProposalHelpers();
  const proposalId = "library_proposal:12345678-1234-4234-8234-123456789abc";
  const key = proposalId.slice("library_proposal:".length);
  const proposal = { found: true, id: proposalId, version: `sha256:${"b".repeat(64)}`, record: { proposed_hash: `sha256:${"c".repeat(64)}` } };
  const validation = { found: true, id: `library_validation:${key}`, version: `sha256:${"d".repeat(64)}`, record: {
    status: "VERIFIED_PRIMARY", currency_status: "cleared", proposal_id: proposalId, proposed_hash: proposal.record.proposed_hash,
    claim_checks: [{ status: "VERIFIED_PRIMARY", currency_status: "cleared" }],
  } };
  assert.equal(canonicalLibraryProposalId(proposalId), proposalId);
  assert.throws(() => canonicalLibraryProposalId(key), /full shared proposal id/);
  assert.throws(() => canonicalLibraryProposalId(`library_proposal:${proposalId}`), /full shared proposal id/);
  assert.equal(canPublishLibraryProposal(proposalId, proposal, validation), true);
  assert.equal(canPublishLibraryProposal(key, proposal, validation), false);
  assert.equal(canPublishLibraryProposal(proposalId, { ...proposal, record: { ...proposal.record, status: "published" } }, validation), false);
  assert.equal(canPublishLibraryProposal(proposalId, { ...proposal, record: { ...proposal.record, proposed_hash: "not-a-sha256" } }, validation), false);
  assert.equal(canPublishLibraryProposal(proposalId, proposal, { ...validation, record: { ...validation.record, currency_status: "provisional" } }), false);
  assert.equal(canPublishLibraryProposal(proposalId, { ...proposal, id: "library_proposal:12345678-1234-4234-8234-abcdefabcdef" }, validation), false);
  assert.equal(canPublishLibraryProposal(proposalId, { ...proposal, version: "not-a-version" }, validation), false);
  assert.equal(canPublishLibraryProposal(proposalId, proposal, { ...validation, record: { ...validation.record, proposed_hash: "sha256:wrong" } }), false);
  assert.equal(canPublishLibraryProposal(proposalId, proposal, { ...validation, record: { ...validation.record, claim_checks: [{ status: "PENDING", currency_status: "cleared" }] } }), false);
  assert.equal(canPublishLibraryProposal(proposalId, { ...proposal, record: { ...proposal.record, currency_status: "cleared" } }, {}), false);
});

/** Expose validation retry only when the saved server dispatch says it is retryable.
 * Inputs: synthetic dispatch states; output: assertion result.
 * Side effects: none; no validation starter is called.
 * Choose to prevent repeat requests for pending or terminal validation states.
 */
test("validation retry requires a saved queue_failed retryable dispatch", async () => {
  const { canRetryLibraryValidation } = await loadProposalHelpers();
  assert.equal(canRetryLibraryValidation({ state: "queue_failed", retryable: true }), true);
  assert.equal(canRetryLibraryValidation({ state: "queue_failed", retryable: false }), false);
  assert.equal(canRetryLibraryValidation({ state: "queued", retryable: true }), false);
  assert.equal(canRetryLibraryValidation(undefined), false);
});

/** Reopen a synthetic imported draft without changing its target version or complete body.
 * Inputs: fixture case_record proposal; output: assertion result for the editor seed.
 * Side effects: none; no proposal tool is called and the retained source row is unchanged.
 * Choose for migration drafts awaiting citations so a later new proposal still uses the imported compare version.
 */
test("citation-required retained drafts preserve their exact target version and complete proposed record", async () => {
  const { libraryDraftEditSeed } = await loadProposalHelpers();
  const proposedRecord = { title: "Synthetic draft", body: "Full fixture body", private_context: { names: ["Synthetic person"] } };
  const seed = libraryDraftEditSeed({
    found: true, id: "library_proposal:12345678-1234-4234-8234-123456789abc", version: `sha256:${"c".repeat(64)}`,
    record: { target: "reference:fixture", expected_version: `sha256:${"d".repeat(64)}`, proposed_record: proposedRecord, citations: [], status: "citation_required", rationale: "Imported fixture" },
  });
  assert.equal(seed.targetId, "reference:fixture");
  assert.equal(seed.expectedVersion, `sha256:${"d".repeat(64)}`);
  assert.deepEqual(seed.proposedRecord, proposedRecord);
  assert.deepEqual(seed.citations, []);
  assert.throws(() => libraryDraftEditSeed({ found: true, id: "library_proposal:12345678-1234-4234-8234-123456789abc", record: { ...seed, status: "pending_validation" } }), /not an editable citation-required draft/);
});

/** Compare a retained proposal against its exact current base while displaying its retained proposed record.
 * Inputs: citation-required proposal fixture and injected case_record reader; output: assertion result.
 * Side effects: reader observes only fixture ids; no shared store or network is accessed.
 * Choose to make retained imports resubmittable without replacing base personal fields or accepting a stale expected version.
 */
test("retained draft patch compares proposed content to the exact target baseline", async () => {
  const { libraryDraftEditSeed, libraryDraftComparisonBaseline, changedLibraryPatch } = await loadProposalHelpers();
  const expectedVersion = `sha256:${"e".repeat(64)}`;
  const proposedRecord = { title: "Retained edit", body: "Full imported content", private_context: { names: ["Fixture Person"] }, created_at: "2026-10-01T12:00:00.000Z" };
  const proposal = {
    found: true, id: "library_proposal:12345678-1234-4234-8234-123456789abc", version: `sha256:${"c".repeat(64)}`,
    record: { target: "reference:fixture", expected_version: expectedVersion, status: "citation_required", rationale: "Imported fixture", proposed_record: proposedRecord, citations: [] },
  };
  const seed = libraryDraftEditSeed(proposal);
  const reads = [];
  const baseline = await libraryDraftComparisonBaseline(seed, async (id) => {
    reads.push(id);
    return { found: true, id, version: expectedVersion, record: { title: "Published title", body: "Existing body", private_context: { names: ["Fixture Person"] }, created_at: "2026-10-01T12:00:00.000Z" } };
  });
  assert.deepEqual(reads, ["reference:fixture"]);
  assert.deepEqual(seed.proposedRecord, proposedRecord, "the editor displays the complete retained proposal body");
  assert.deepEqual(changedLibraryPatch(baseline, seed.proposedRecord), { title: "Retained edit", body: "Full imported content" });
  await assert.rejects(libraryDraftComparisonBaseline(seed, async (id) => ({ found: true, id, version: `sha256:${"f".repeat(64)}`, record: {} })), /Library version conflict/);
  let absentReads = 0;
  assert.deepEqual(await libraryDraftComparisonBaseline({ targetId: "source:new-fixture", expectedVersion: "absent" }, async () => { absentReads += 1; return {}; }), {});
  assert.equal(absentReads, 0);
});

/** Exercise retained proposal paging against a fake case_query transport and small node doubles.
 * Inputs: page loader; output: visible list and exact read-only query requests.
 * Side effects: transforms the production list function only; no HTTP, database, or browser is started.
 * Choose to ensure imported drafts remain browsable beyond the first bounded page without entering DATA_TABLES.
 */
async function runRetainedDraftList(load) {
  const source = (await readFile(resolve("web/host.ts"), "utf8")).replace(/\r\n/g, "\n");
  const start = source.indexOf("async function viewLibraryDrafts():");
  const end = source.indexOf("\nasync function viewPackagedFiles(", start);
  assert.ok(start >= 0 && end > start, "retained proposal view has bounded source boundaries");
  const code = transformSync(source.slice(start, end), { loader: "ts", target: "es2022" }).code;
  const requests = [];
  const openedDrafts = [];
  const nodes = [];
  const makeNode = (tag, attrs = {}, ...children) => ({ tag, attrs, children, listeners: {}, append(...items) { this.children.push(...items); }, replaceChildren(...items) { this.children = items; }, addEventListener(name, fn) { this.listeners[name] = fn; } });
  const host = makeNode("main");
  const view = new Function("page", "el", "data", "filterInput", "recordRow", "errBox", "openLibraryDraft",
    `${code}; return viewLibraryDrafts;`)(
    () => host,
    makeNode,
    async (tool, args) => { requests.push({ tool, args }); return load(tool, args); },
    (placeholder, onInput) => makeNode("input", { placeholder, onInput }),
    (row, meta, onOpen) => makeNode("button", { row, meta, onOpen }),
    (error) => makeNode("div", { error: error.message }),
    (id) => { openedDrafts.push(id); },
  );
  await view();
  nodes.push(...host.children);
  return { requests, host, nodes, openedDrafts };
}

/** Page through proposal metadata with the fixed SELECT-only offset query.
 * Inputs: two synthetic pages; output: assertion result.
 * Side effects: fake read-only case_query calls only; proposal bodies are fetched only after an exact row click.
 * Choose for confirming drafts after the first 50 remain reachable and no generic-record table allowlist expands.
 */
test("retained proposal list uses bounded SELECT pages beyond the first 50 rows", async () => {
  const pages = [
    Array.from({ length: 50 }, (_, index) => ({ id: `library_proposal:00000000-0000-4000-8000-${String(index + 1).padStart(12, "0")}`, target: "reference:fixture", expected_version: "absent", status: "citation_required" })),
    [{ id: "library_proposal:00000000-0000-4000-8000-000000000051", target: "source:fixture", expected_version: `sha256:${"e".repeat(64)}`, status: "pending_validation" }],
  ];
  const firstProposal = pages[0][0].id;
  const result = await runRetainedDraftList(async (tool, args) => {
    assert.equal(tool, "case_query");
    assert.match(args.surql, /^SELECT id, target, expected_version, status, created_at FROM library_proposal .*LIMIT 50 START (?:0|50)$/);
    assert.equal(args.params, undefined);
    assert.equal(args.write, undefined);
    return { results: [pages.shift()], truncated: false };
  });
  const more = result.host.children.find((node) => node.attrs.text === "Show more");
  assert.equal(more.hidden, false);
  const list = result.host.children.find((node) => node.tag === "div" && node.children.some((child) => child.attrs?.row));
  await list.children[0].attrs.onOpen();
  assert.equal(result.openedDrafts[0], firstProposal);
  await more.listeners.click();
  assert.deepEqual(result.requests.map(({ args }) => Number(/START (\d+)$/.exec(args.surql)[1])), [0, 50]);
  assert.equal(result.host.children.find((node) => node.tag === "div" && node.children.some((child) => child.attrs?.row?.id?.endsWith("000000000051"))) !== undefined, true);
});

/** Keep library source/reference detail on the guarded proposal route while ordinary records retain corrections.
 * Inputs: none; output: assertion result from the production openRecord branch.
 * Side effects: calls only test doubles, never the hosted route or the shared store.
 * Choose to catch regressions that send library records into generic case_put correction flow.
 */
test("openRecord routes source and reference ids through the library proposal detail path", async () => {
  const source = (await readFile(resolve("web/host.ts"), "utf8")).replace(/\r\n/g, "\n");
  const start = source.indexOf("async function openRecord(");
  const end = source.indexOf("\nfunction correctForm(", start);
  assert.ok(start >= 0 && end > start, "record router has bounded source boundaries");
  const code = transformSync(source.slice(start, end), { loader: "ts", target: "es2022" }).code;
  const routed = [];
  const opened = [];
  const openRecord = new Function("openSheet", "openLibraryRecord", "api", "titleOf", "officialSourceUrl", "el", "kv", "markdown", "correctForm", "onChanged", `${code}; return openRecord;`)(
    () => {}, async (id, refresh) => routed.push([id, refresh]), async () => { throw new Error("ordinary record path should not run"); },
    () => "Record", () => null, () => ({}), () => ({}), () => ({}), () => {}, () => {},
  );
  const refresh = () => {};
  await openRecord("reference:fixture", refresh);
  await openRecord("source:fixture", refresh);
  assert.deepEqual(routed, [["reference:fixture", refresh], ["source:fixture", refresh]]);
  assert.deepEqual(opened, []);
});

/** Keep optimistic-version conflicts visible in the record editor instead of replacing the draft.
 * Inputs: a synthetic conflict error; output: accessible record-local alert data.
 * Side effects: transforms one production formatter with a fake element factory; no mutation occurs.
 * Choose to verify failed proposals direct the user to reload the latest shared version.
 */
test("library version conflicts are rendered as explicit per-record alerts", async () => {
  const source = (await readFile(resolve("web/host.ts"), "utf8")).replace(/\r\n/g, "\n");
  const start = source.indexOf("function libraryFailureBox(");
  const end = source.indexOf("\n// ---------------------------------------------------------------------------\n// Widgets", start);
  assert.ok(start >= 0 && end > start, "library failure formatter has bounded source boundaries");
  const code = transformSync(source.slice(start, end), { loader: "ts", target: "es2022" }).code;
  const libraryFailureBox = new Function("el", `${code}; return libraryFailureBox;`)((tag, attrs) => ({ tag, attrs }));
  const conflict = libraryFailureBox(new Error("Library version conflict"));
  assert.equal(conflict.attrs.role, "alert");
  assert.match(conflict.attrs.class, /library-conflict/);
  assert.match(conflict.attrs.text, /Reopen the latest shared record/);
});

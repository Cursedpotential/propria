// Synthetic-only acceptance proof against a disposable API database.
// Attach to an existing approved browser; this script never launches a desktop browser.
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || "playwright");
const crypto = require("crypto");
const fs = require("fs");
const path = require("path");
const base = process.env.CLAIMS_SMOKE_WEB || "http://127.0.0.1:3015";
const api = process.env.CLAIMS_SMOKE_API;
const browserEndpoint = process.env.CLAIMS_SMOKE_BROWSER_CDP;
const signingKey = process.env.CLAIMS_SMOKE_SIGNING_KEY;

function signedHeaders(method, target, body) {
  const timestamp = String(Math.floor(Date.now() / 1000));
  const nonce = crypto.randomBytes(16).toString("hex");
  return {
    "x-legal-bff-timestamp": timestamp,
    "x-legal-bff-nonce": nonce,
    "x-legal-bff-signature": crypto.createHmac("sha256", signingKey)
      .update([timestamp, nonce, method, target, crypto.createHash("sha256").update(body).digest("hex")].join("\n")).digest("hex"),
  };
}
function assert(value, message) { if (!value) throw new Error(message); }

(async () => {
  assert(browserEndpoint, "Set CLAIMS_SMOKE_BROWSER_CDP to an existing approved browser endpoint.");
  assert(process.env.CLAIMS_SMOKE_DISPOSABLE === "yes", "Set CLAIMS_SMOKE_DISPOSABLE=yes only for a disposable synthetic fixture.");
  assert(!api || signingKey, "A direct fixture API requires CLAIMS_SMOKE_SIGNING_KEY.");
  const browser = await chromium.connectOverCDP(browserEndpoint);
  const context = await browser.newContext({ viewport: { width: 1440, height: 1100 } });
  const page = await context.newPage();
  const errors = [];
  page.on("pageerror", error => errors.push(error.message));
  try {
    if (api) await page.route("**/api/legal/**", async route => {
      const req = route.request(), url = new URL(req.url());
      const target = url.pathname.replace("/api/legal", "") + url.search;
      const body = req.postDataBuffer() || Buffer.alloc(0);
      const headers = signedHeaders(req.method(), target, body);
      if (req.headers()["content-type"]) headers["content-type"] = req.headers()["content-type"];
      const response = await page.request.fetch(api + target, { method: req.method(), headers, data: body.length ? body : undefined });
      await route.fulfill({ response });
    });
    await page.goto(`${base}/claims`);
    await page.getByRole("heading", { name: "Claims and evidence", exact: true }).waitFor();
    await page.getByRole("button", { name: "Add claim", exact: true }).click();
    const name = `Synthetic parenting-contact statement ${Date.now()}`;
    await page.getByLabel("Who made this statement?", { exact: true }).fill("Synthetic parent");
    await page.getByLabel("Record type", { exact: true }).selectOption("allegation");
    await page.getByLabel("Statement or allegation", { exact: true }).fill(name);
    await page.getByLabel("Your response", { exact: true }).fill("Synthetic response retained separately.");
    await page.getByRole("button", { name: "Save claim", exact: true }).click();
    await page.getByText("Claim saved.", { exact: true }).waitFor();

    async function fixtureCall(target, method = "GET", body) {
      return page.evaluate(async ({ target, method, body }) => {
        const result = await fetch(`/api/legal${target}`, { method, headers: body ? { "content-type": "application/json" } : {}, body: body ? JSON.stringify(body) : undefined });
        const value = await result.json();
        if (!result.ok) throw Error(`Fixture request failed ${result.status}`);
        return value;
      }, { target, method, body });
    }
    let claim = (await fixtureCall("/v1/claims")).find(item => item.text === name);
    assert(claim && claim.response === "Synthetic response retained separately.", "Claim/response did not persist.");
    assert(claim.evidence_status === "evidence_needed", "Unlinked claim wrongly has evidence support.");

    await page.getByLabel("Relationship to this claim", { exact: true }).selectOption("context");
    await page.getByLabel("Context location", { exact: true }).fill("Synthetic personal note, paragraph 2");
    await page.getByLabel("What does this source show?", { exact: true }).fill("Context is not accepted evidence.");
    await page.getByRole("button", { name: "Save source link", exact: true }).click();
    await page.getByText("Source link saved.", { exact: true }).waitFor();
    claim = await fixtureCall(`/v1/claims/${claim.claim_id}`);
    assert(claim.evidence_status === "context_only", "Context was incorrectly promoted to factual support.");
    const contextLinkId = claim.links.find(link => link.active).link_id;

    await page.getByLabel("Missing evidence or unanswered question", { exact: true }).fill("Find the original parenting-contact message.");
    await page.getByRole("button", { name: "Add gap", exact: true }).click();
    await page.getByText("Evidence gap saved.", { exact: true }).waitFor();
    await page.getByLabel("Next action", { exact: true }).selectOption("investigate");
    await page.getByLabel("Related gap", { exact: true }).selectOption({ index: 1 });
    await page.getByLabel("Follow-up details", { exact: true }).fill("Locate the synthetic original and record its source location.");
    await page.getByRole("button", { name: "Save follow-up plan", exact: true }).click();
    await page.getByText("Follow-up plan saved.", { exact: true }).waitFor();
    await page.getByRole("button", { name: /^Gaps and follow-up/ }).click();
    await page.getByRole("heading", { name, exact: true }).first().waitFor();
    assert((await page.getByText("Locate the synthetic original and record its source location.", { exact: false }).count()) > 0, "Gap report lost planned follow-up.");
    await page.getByRole("button", { name: "Open claim and plan follow-up", exact: true }).first().click();

    // Removing a mistaken link retains it, and restoring it revalidates the source.
    page.once("dialog", dialog => dialog.accept());
    await page.getByRole("button", { name: "Remove link", exact: true }).click();
    await page.getByText("Evidence link removed from the active claim.", { exact: true }).waitFor();
    claim = await fixtureCall(`/v1/claims/${claim.claim_id}`);
    assert(claim.links.find(link => link.link_id === contextLinkId)?.active === false, "Removed link history lost.");
    await page.getByText("Removed links (1)", { exact: true }).click();
    await page.getByRole("button", { name: "Restore link", exact: true }).click();
    await page.getByText("Evidence link restored.", { exact: true }).waitFor();

    const sourceResult = await fixtureCall("/v1/claim-sources");
    let exactSourceLinked = false;
    if (sourceResult.items.length) {
      await page.getByLabel("Relationship to this claim", { exact: true }).selectOption("supports");
      await page.getByLabel("Accepted evidence source", { exact: true }).selectOption("0");
      await page.getByLabel("What does this source show?", { exact: true }).fill("Synthetic exact accepted passage supports this recorded assertion.");
      await page.getByRole("button", { name: "Save source link", exact: true }).click();
      await page.getByText("Source link saved.", { exact: true }).waitFor();
      claim = await fixtureCall(`/v1/claims/${claim.claim_id}`);
      const linked = claim.links.find(link => link.relationship === "supports" && link.active);
      assert(linked?.source?.content_hash === sourceResult.items[0].content_hash && linked.source.span_locator === sourceResult.items[0].span_locator, "Exact provenance changed during linking.");
      assert(claim.evidence_status === "evidence_linked", "Accepted evidence did not produce readable linked status.");
      exactSourceLinked = true;
    } else assert(process.env.CLAIMS_SMOKE_REQUIRE_SOURCE !== "yes", "Fixture requires an accepted synthetic package for exact-source proof.");

    // An independent write must not silently overwrite the editor's stale revision.
    claim = await fixtureCall(`/v1/claims/${claim.claim_id}`);
    await fixtureCall(`/v1/claims/${claim.claim_id}`, "PATCH", { expected_revision: claim.revision, response: "Changed by another synthetic session." });
    await page.getByLabel("Your response", { exact: true }).fill("My unsaved response survives the conflict.");
    await page.getByRole("button", { name: "Save changes", exact: true }).click();
    await page.getByText("This claim changed in another session. Your unsaved entries are still here.", { exact: true }).waitFor();
    assert(await page.getByLabel("Your response", { exact: true }).inputValue() === "My unsaved response survives the conflict.", "Conflict discarded user's input.");
    page.once("dialog", dialog => dialog.accept());
    await page.getByRole("button", { name: "Reload latest claim", exact: true }).click();
    await page.getByRole("button", { name: "View saved history", exact: true }).click();
    await page.getByText(/^Version 1 ·/).waitFor();

    // Check secondary-form unsaved guards, then persistence after browser reload.
    await page.getByLabel("Missing evidence or unanswered question", { exact: true }).fill("Unsaved secondary form text");
    page.once("dialog", dialog => dialog.dismiss());
    await page.getByRole("button", { name: "Add claim", exact: true }).click();
    assert(await page.getByLabel("Missing evidence or unanswered question", { exact: true }).inputValue() === "Unsaved secondary form text", "Secondary form had no unsaved guard.");
    await page.getByLabel("Missing evidence or unanswered question", { exact: true }).fill("");
    await page.reload();
    const claimButton = page.getByRole("button", { name: new RegExp(name) });
    await claimButton.waitFor(); await claimButton.click();
    assert(await page.getByLabel("Your response", { exact: true }).inputValue() === "Changed by another synthetic session.", "Reload persistence failed.");
    await page.getByRole("button", { name: "Mark done", exact: true }).click();
    await page.getByText("Follow-up marked done.", { exact: true }).waitFor();
    claim = await fixtureCall(`/v1/claims/${claim.claim_id}`);
    assert(claim.followups[0].status === "done", "Follow-up completion failed persistence.");

    let sharedProbataOverlay = false;
    const shared = await fixtureCall("/v1/probata/records?kind=event&q=");
    if (shared.available && shared.records.length) {
      await page.getByRole("button", { name: "Probata records", exact: true }).click();
      await page.getByRole("button", { name: "Add legal response", exact: true }).first().waitFor();
      await page.getByRole("button", { name: "Add legal response", exact: true }).first().click();
      await page.getByRole("region", { name: "Linked Probata record", exact: true }).waitFor();
      let overlay = (await fixtureCall("/v1/claims")).find(item => item.origin?.record_id === shared.records[0].origin.record_id);
      assert(overlay && overlay.links.length === 0 && overlay.origin.record_version === shared.records[0].origin.record_version, "Probata source was copied or promoted to accepted evidence.");
      const originalOverlayId = overlay.claim_id;
      await page.getByLabel("Your response", { exact: true }).fill("Synthetic legal response to the shared event.");
      await page.getByRole("button", { name: "Save changes", exact: true }).click();
      await page.getByText("Claim saved.", { exact: true }).waitFor();
      await page.getByRole("button", { name: "Probata records", exact: true }).click();
      await page.getByRole("button", { name: "Add legal response", exact: true }).first().click();
      overlay = await fixtureCall(`/v1/claims/${originalOverlayId}`);
      assert(overlay.response === "Synthetic legal response to the shared event.", "Reopening shared event replaced legal overlay.");
      const overlayMatches = (await fixtureCall("/v1/claims")).filter(item => item.origin?.record_id === shared.records[0].origin.record_id && item.origin?.kind === "event");
      assert(overlayMatches.length === 1, "Reopening shared event created duplicate records.");
      sharedProbataOverlay = true;
    } else assert(process.env.CLAIMS_SMOKE_REQUIRE_PROBATA !== "yes", "Fixture requires a shared synthetic Probata event.");

    const output = process.env.CLAIMS_SMOKE_OUTPUT;
    if (output) { fs.mkdirSync(output, { recursive: true }); await page.screenshot({ path: path.join(output, "claims-desktop.png"), fullPage: true }); }
    await page.setViewportSize({ width: 390, height: 844 });
    const horizontalOverflow = await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth);
    assert(!horizontalOverflow, "Claims page overflows mobile viewport.");
    if (output) await page.screenshot({ path: path.join(output, "claims-mobile.png"), fullPage: true });
    assert(!errors.length, `Browser errors: ${errors.join("; ")}`);
    const proof = { claim_id: claim.claim_id, durable_claim_response: true, context_not_support: true, gaps_report: true,
      followup_planned_and_completed: true, retained_link_restore: true, exact_source_linked: exactSourceLinked,
      stale_revision_protected: true, unsaved_secondary_form_protected: true, history: true, reload_persistence: true,
      mobile_no_horizontal_overflow: true, page_errors: errors, scope: "Disposable synthetic API and attached browser; no external requests dispatched." };
    proof.shared_probata_overlay = sharedProbataOverlay;
    if (output) fs.writeFileSync(path.join(output, "claims-workspace-proof.json"), JSON.stringify(proof, null, 2));
    console.log(JSON.stringify(proof));
  } finally {
    await page.unrouteAll({ behavior: "ignoreErrors" });
    await context.close();
    // A CDP disconnect leaves the user's existing browser process running.
    await browser.close();
  }
})().catch(error => { console.error(error.message); process.exitCode = 1; });

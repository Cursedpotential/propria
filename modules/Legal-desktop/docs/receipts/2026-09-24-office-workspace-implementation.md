# Office workspace implementation and deployment handoff

Date: 2026-09-24. Author: Codex and bounded backend/template/office implementation agents.

## Delivered in source

- Whole-document service in the existing `legal.sqlite`: blank DOCX, imported DOCX/ODT, metadata, immutable original and subsequent byte revisions, downloads and history. Revision bytes, metadata and audit outbox commit together. Stale writes conflict; a failed audit transaction does not leave a saved revision.
- Actual styled DOCX templates derived from the existing outline catalog, with caption, section and signature fields. The template identifier/version/generated hash are recorded in the created document revision. These are working templates, not a newly acquired official court-form collection.
- Collabora WOPI bridge: document-scoped hashed sessions with expiry, CheckFileInfo/GetFile/PutFile, durable locks and atomic revision checks shared with ordinary document saves. Office tokens do not authorize the general API.
- `/drafts` is now Documents and writing: blank/template creation, DOCX/ODT import, saved-document list, embedded office launch, save, download and revision history. Previous section drafts remain accessible in a collapsed section.
- Switching an active editor waits for a successful save response. Failure or missing acknowledgement retains the editor; stale metadata/history responses cannot change a later selection. Navigation links request a save first and browser unload retains a recovery prompt. Save text describes the last confirmed save rather than asserting all subsequent edits are saved.
- Same-origin `/office` gateway configuration supports editor HTTP and WebSockets. The existing tailnet port 3011 is moved from the web container to the gateway; no additional exposed host port is introduced. The API callback remains internal on `legal-api:8010`. Access logs omit token-bearing URLs.

## Runtime selection and references

Collabora CODE image pinned to official registry digest `sha256:4e983196eb9878f339cc506c38c21f1cc3473bca3d6de883c5de08f9c0cc3a6c`, resolved from the [official image registry](https://hub.docker.com/r/collabora/code) on September 24. This is the first implementation/evaluation target for the owner's LibreOffice requirement; production fidelity is not inferred from selecting an image.

The `/office` service-root behavior follows the [official Collabora configuration source](https://github.com/CollaboraOnline/online.mirror/blob/main/coolwsd.xml.in). The host/editor message flow is checked against [Collabora's implementation](https://github.com/CollaboraOnline/online.mirror/blob/main/browser/src/map/handler/Map.WOPI.js). Lock/save responses follow [WOPI Lock](https://learn.microsoft.com/en-us/microsoft-365/cloud-storage-partner-program/rest/files/lock) and [PutFile](https://learn.microsoft.com/en-us/microsoft-365/cloud-storage-partner-program/rest/files/putfile).

## Verification completed

1. Production web build and TypeScript passed with the new workspace.
2. Forty Python tests passed across authentication, document persistence, editor protocol and templates. Tests cover original retention, restart/recreation, concurrent writes, transaction rollback, document scope/expiry, locks, repeated WOPI saves, invalid credentials, discovery URL construction, actual DOCX content/styles and template version metadata. One existing Python deprecation warning remains.
3. Real local API + browser smoke passed: blank creation, template creation, original download (35,255 bytes for the blank fixture), version history and reload persistence. The local store is isolated at `.reconciliation/office-smoke-state`; only synthetic work products were used. A signed test transport adapter connects browser calls to the loopback API; production ingress was not exercised by this test.
4. A separate simulated-office browser test proved rejected saves retain the current iframe and acknowledged saves permit switching. This proves the host UI protocol logic, not an actual Collabora editing session.
5. Desktop/mobile screenshots were inspected from the local browser test. The unavailable-editor state is explicit and still allows retrieval of saved documents.

Reproducible browser scripts: `web/smoke/office-workspace-smoke.cjs` and `web/smoke/office-save-guard-smoke.cjs`. They target loopback Next on 3015 and API on 18010 only. The API must use an isolated store and the synthetic signing key shown in the script. Set `PLAYWRIGHT_MODULE`, optional `CHROMIUM_PATH`, and `OFFICE_SMOKE_OUTPUT` for the installed runtime/output location. They do not exercise a live matter or invoke a model.

## Deployment blocker observed

The live legal health URL returned HTTP 502. Read-only inspection of the registered ovh-app host found `docker.service` inactive with `Result=success`. Its journal records a requested graceful stop at **2026-09-24 10:10:10 UTC / 06:10:10 EDT**, completed at 10:10:16 UTC. This was after the earlier copy-cleanup deployment and before this implementation's deployment.

The owner was asked whether another task intentionally owns that server work. No answer was available when this receipt was written. This lane did not start/restart the host or its shared Docker engine, and did not attempt an application deploy against a stopped engine. The new office workspace is not claimed live.

## Resume procedure

1. Resolve ownership of the deliberate Docker stop; preserve concurrent infrastructure work. Once the application host is available, use the existing Coolify application `gvghzivfmctev8dloetfssnj`, repository `Cursedpotential/Legal-Workspace`, branch `master`.
2. Deploy the reviewed commit through Coolify. Verify gateway, API, web and Collabora health plus the existing legal hostname; retain the current port registry and routes. Check memory/capacity with the new editor service running.
3. Create a synthetic office document through the UI, actually type into Collabora, save, close/reopen, and verify the new immutable revision/download. Repeat with ODT and DOCX fixtures containing tables, footnotes, comments, tracked changes and page formatting. Test interrupted connections and parallel editors. Keep native tracked-change and formatting fidelity marked unverified until these pass.
4. Run the original no-auth/invalid-token checks against the live callback surface without exposing tokens in logs. Verify normal public ingress still enforces identity and that the intended tailnet office origin is the only supported launch origin.
5. Publish the runtime receipt and refresh Docstore. Continue official PDF forms, retroactive template proposals and the four-dimension review workspace from the full workflow report.

## Remaining functionality

This delivery implements the durable document foundation and the interactive-editor bridge/UI, with editable working templates. Actual deployed editor operation remains blocked/unverified. Official PDF catalog/filling, AI template mapping of existing drafts, proposal decisions, full review/release integration and representative office round-trip proof are still open. The larger toolkit/timeline/digital-firm requirements are retained in the parent report.

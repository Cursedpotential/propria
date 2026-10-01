# Coolify state refresh

September 30, 2026 — Codex, using the installed coolify-write 1.1.0 MCP tools.

The September 24 Docker-stop blocker is historical and no longer applies. Coolify infrastructure discovery reports ovh-app reachable. Application `gvghzivfmctev8dloetfssnj` now builds `Cursedpotential/propria`, branch `main`, base directory `/modules/Legal-desktop`, compose `/compose.yaml`. Do not use the archived independent-repository deployment helper.

Latest recorded deployment `v14eduuks932nhepq4yvljhf` finished September 28 at 02:19:58 UTC, commit `51fe27b20d8aee7f8278b10bf7eed3382824ba20`. Comparing that commit to the current checkout showed no changes under `modules/Legal-desktop` before this receipt. No redundant redeployment was triggered.

Live checks returned HTTP 200 for:

- `/drafts`, containing Documents and writing and Start writing.
- `/api/legal/v1/office/status`, reporting configured=true and available=true.
- `/office/hosting/discovery`, returning WOPI discovery with DOCX support.

Coolify's aggregate application status is `running:unknown`, with application health checking disabled; it is not a verified all-container healthy status. These endpoint checks do not establish edit/save/reopen or formatting/tracked-change fidelity. No document was created or modified by this read-only refresh.

Correction: the prior reply should have checked Coolify's current state rather than carrying the September 24 host-stop blocker forward. All further lifecycle operations belong through Coolify.

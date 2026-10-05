# Existing Recovered placement correction

> _Byline: Codex · GPT-6 · 2026-10-04._

STATUS: PLAN FOR REVIEW — no objects moved, removed or overwritten.

Live listing independently confirms both `Recovered/` and `recovery/` beneath the Case Vault. The mistaken lowercase prefix contains 15 ZIP originals (116,293,214 bytes) and 30 preservation/readback receipts, 45 objects total. These recovery originals belong beneath the existing `Recovered/` domain. Active extracted legal material remains under `KnowledgeBase/legal/`; relocation of these archive originals must not change that working-data decision.

Exact mapping: replace only `consignatio/casevault/recovery/` with `consignatio/casevault/Recovered/`, retaining every suffix. The machine map, including bytes, provider version ID and SHA-1, is `wiki-publication-20261004/recovery-placement-map.csv`, SHA-256 `0e832562b0abdacf02ce8bbf86d0bf8b5068f0dd60bb9965db77cff6049a5400`. The snapshot is `wiki-publication-20261004/recovery-listing.json`, SHA-256 `2a72be7c032fb209c77ea5e45eea33d69ce3a578ab95a80245d9d7e16c947417`. Provider SHA-1 is listing evidence, not a new independent SHA-256 body verification.

Before execution: check every destination for collision; independently read and SHA-256 the exact current source versions on the VPS; compare archive identities with the existing preservation receipts. Use the existing governed server-side placement/catalog seam for the approved move, preserve exact historical source/version references, record new destination versions and their verification hashes, and update current locators only after successful readback. Retain historical receipts as written; append relocation lineage rather than rewriting their source claims. No hard deletion or new top-level directory.

The future-write defect is also identified in clean committed source: `modules/Probata/probata/modules/engine/activities/toolkit_package_preservation.go:27` and `toolkit_package_conditional_write_probe.go:21` hard-code the lowercase recovery prefix. The reviewed correction must change those destinations to `Recovered/` and prove the destination contract before activating the worker. No engine source modification/deployment is claimed by this plan.

Execution remains subject to the owner's earlier explicit instruction: "Don't move anything until I review the plan." The tools wiki is already published within existing `Code/wiki/`; it does not depend on this relocation.

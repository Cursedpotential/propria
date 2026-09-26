---
title: Handoff — Consignatio ↔ legal workdesk evidence integration
date_shared: 2026-09-20
shared_by: owner (pasted mid-task into a Claude Code session, 23:07 EDT)
source_service: not stated by the owner
capture: verbatim as pasted
tags: [transcript, handoff, advocatio, consignatio, catalog, backblaze-b2, evidence, promotion, provenance]
status: queued — not yet worked (docs/URGENT-TODO.md Q2)
---

> _Byline: Claude Code · Fable 5.1 · 2026-09-20 (saved verbatim; nothing added)_

## Transcript

CONSIGNATIO — LEGAL WORK DESK EVIDENCE INTEGRATION

Evidence storage remains on Backblaze B2. Do not use:

E:\AI\_Workspace\Projects\Propria\modules\Consignatio\EvidenceVault

as the evidence storage root.

Integrate with the Consignatio catalog to discover and display evidence.
The catalog's exact schema/API is not being prescribed in this handoff.
Reuse its actual contract; do not create a replacement catalog or invent
required field names.

Required behavior:

• Discover evidence through the catalog, then resolve the corresponding
  B2 objects using the catalog/configuration-provided locations.

• Distinguish ordinary context, pending promotion, and completed promotion.
  Only completed promotions belong in the promoted-evidence view.
  Do not infer evidence status from filenames or folder membership alone.

• Promotion creates a separate immutable evidence copy on B2. The source
  context remains in its structured home and stays linked through provenance.

• Expose available catalog metadata, provenance, source relationships,
  and promotion information alongside the evidence.

• Treat promoted evidence as read-only. Keep desk annotations, caches,
  working copies, and generated work product outside immutable evidence
  objects/packages, linked through the catalog's existing identifiers.

• Use the existing B2 access mechanism for viewing or retrieving artifacts.
  A local vault mirror must not be a prerequisite for this integration.

• Keep the catalog connection and storage resolution configurable.
  Do not create buckets, migrate files, or impose a new EvidenceVault
  package layout as a side effect of building the desk.

The internal EvidenceVault layout and catalog contract can be bound to
their actual implementation without changing this integration model.

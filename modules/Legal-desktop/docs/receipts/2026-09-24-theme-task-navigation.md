# Probata palette and task navigation

Owner direction: September 24, 2026.

Use Probata's actual graphite/indigo application colors. Replace the conflicting
palette in Propria's shared design source and vendored consumers, and withdraw
the outdated active Docstore approval. Keep status colors distinct and retain
the shared component geometry and accessibility contract.

Use stable collapsible task categories following the original brainstorming
register at docs/planning/2026-09-13-advocatio-reconciliation/REQUIREMENTS.md.
Reference collection, methods, case context/strategy, documents and review,
evidence/claims/gaps, chronology, digital firm and tools define the product
organization. Phase tabs and phase-dependent menu reordering are superseded.
Only implemented catalog routes are navigation destinations; missing planned
features remain roadmap items rather than empty placeholder screens.

Shared tokens and both Advocatio/Family Court copies pass generation, semantic
parity and contrast verification. Family Court production build passes.
Docstore owner-direction flag note:propria_surface_design_contract_20260912
was updated to revision 3 and independently read back on September 24.

Deployment and browser results will be appended after verification.

Implementation: Advocatio 9cbc158, merged with concurrent source-package work in
4e6290f. Shared Propria palette commits b33245a and 1ef465a. Task navigation uses
Reference library; Documents and drafting; Evidence; Case strategy; Legal research
and analysis; Court and calendar; Skills and processes; Digital firm and tools.
Case dashboard remains a direct link. Command search retains catalog order.

Local production build: 36 routes generated, TypeScript passed. Browser verified
33 unique destinations, no phase buttons, keyboard disclosure toggling, persisted
expansion after reload, and exact canvas/action/surface colors. No uncaught page
errors. The local preview has no BFF credentials, so this is UI proof only.

Docstore decision revision 3 is verified through the hosted tool. File-sync CLI
could not authenticate (CF_MCP_CLIENT_TOKEN absent); no full file-index refresh
is claimed. The receipt will also be summarized in a governed hosted note.

Live verification: Coolify deployment lvyo8e8xh4p7ce5milgtcwrn finished at
2026-09-24T07:43:59Z with 4e6290f. The live /external-sources browser test
confirmed the exact Probata palette, all eight task groups, 33 unique links,
zero phase buttons, keyboard toggling and expansion after reload. No uncaught
browser errors. Dashboard/drafts/templates/laws/calendar/assistant returned 200;
health/matter/drafts/templates API reads returned JSON 200. No live case writes
or provider calls were made. Shared Family Court changes are source/build
verified; this receipt claims deployment only for Advocatio.

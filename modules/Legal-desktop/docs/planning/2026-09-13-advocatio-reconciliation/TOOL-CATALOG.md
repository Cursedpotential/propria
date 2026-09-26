# Resource, tool and integration catalog

This index separates a **resource to read**, a **method to apply**, a **tool to call**, and **code to port**. One package may provide all four. The complete item registers are linked below; useful existing capabilities should become shared/callable before a full code port where that advances the common-surface target.

| Collection or system | Contribution | Planned home / route | Evidence |
|---|---|---|---|
| Current Family Court Toolkit | Full legal material, skills, agents, commands, implemented tool handlers and store access | Shared resource/method collection; generic MCP client and server-side capability adapters | [Toolkit capability audit](inputs/stack/TOOLKIT-CAPABILITIES.md) and [383 inventory entries](inputs/stack/toolkit-capabilities.json). |
| Existing Codex Michigan console | Eight currently connected curated tools | Reuse as currently callable subset; compare with full plugin, preserve differences | Root read-only checklist call succeeded; [MCP client plan](MCP-CLIENT-AND-PORTS.md). |
| Existing remote Surreal/MCP store | Store-first source/reference access with local fallback and remote deployment history | Reconcile remote state before duplicate import; use named service route after registry proof | Toolkit report distinguishes loader counts, migration receipts and current reachability. |
| Legal MCP original ZIP | Foundational tool implementations, playbooks, skills and resource/provider ideas | Port useful pieces into shared typed tool/method registry; call external services through MCP client | [Pack content review](inputs/handoffs/LEGAL-MCP-PACK-REVIEW.md); original source register item SRC-051. |
| Claude/Codex/.agents legal plugins | Additional drafting, evidence, citation, strategy and review methods/code | Deduplicate by source/content; retain useful variants with applicability; no blanket enable | [Additional capability discovery](inputs/current-app/LEGAL-CAPABILITY-DISCOVERY.md) and [16 records](inputs/current-app/legal-capabilities.json). |
| CourtListener | Legal research data through official MCP/REST interfaces | Registered MCP connection and source adapter; manual and agent use | [Official API/MCP documentation](https://wiki.free.law/c/courtlistener/help/api), [client plan](MCP-CLIENT-AND-PORTS.md). |
| Timesketch fork | Advanced chronology, annotations and governed curation | Probata engine/source authority; staged common-surface presentation/adaptation | [handoff recovery](inputs/handoffs/HANDOFF-RECOVERY.md), local fork and accepted ADR. |
| react-calendar-timeline; vis-timeline/vis-data | Distinct lighter timeline rendering options | Shared event projection adapters in unified surface | [visualization recovery](inputs/current-app/VISUALIZATION-RECOVERY.md). |
| Evidence.dev | Frozen reports complementary to live grids | Report artifact integration; restore platform-owned project per D-129 | Visualization recovery and handoff register. |
| NeoDash, React Flow, Surrealist, CopilotKit/AG-UI, Kepler.gl/Leaflet, history viewer | Graphs, workflow DAGs, admin, assistant visual output, geography, session browsing | Per-project disposition; common TypeScript views where appropriate, scoped tool/engine bridge where selected | Historical integration candidates; [rediscovery register](inputs/handoffs/rediscovery.json). |
| LibreOffice and office/editor candidates | Render/convert versus interactive edits/tracked changes | Bounded document jobs and selected editing surface | Original Category 5/later HOLD; [rediscussion](REDISCUSSION.md). **2026-09-21:** LibreOffice render/convert is live; per-tool status in [external tools register](continuation/EXTERNAL-TOOLS-REGISTER.md). Editor choice still open. |
| PDF/form/OCR/redaction/metadata libraries | Form filling, extraction, redaction, stamping and reports | Legal derivative jobs versus evidence/intake extraction according to source ownership | Original Category 5 tool lists in handoff recovery. |
| FreeEed | Actual source for staging/processing/results, family/duplicate/exception/export patterns | Borrow bounded code/process patterns after defect review; processing remains evidence/intake side | `resources/reference-review/2026-09-13/freeeed/SYNTHESIS.md`, pipeline and workflow reports. |
| KAPE | Gathering-side candidate | Evidence/intake contribution assessment | Source-batch pointer; product identification/deeper assessment remains open. |
| Other legal donor apps | Legal terminal, LexRAG, LIGHT-2, THEMIS, Suna/Kortix and skill collections | Named bounded port candidates with source, intended function and language disposition | Twenty named integrations in handoff JSON; later shape-only decisions reconsidered where needed. |
| vLex examples / original digital firm | Cheatsheet format/content and actual role operating design | Reference candidates and versioned firm prompt/protocol library | Pending originals from owner. |

## Per-capability fields required for implementation

Stable ID; display name; kind; package/version/origin; source path/handler; input/output schema where callable; jurisdiction/applicability; source/method versions; current wiring; destination or callable connection; permissions; result provenance; verification fixture; disposition and next action.

Source coverage uses precise states: inventoried, content read, code inspected, extracted, handler registered, connection discovered, call exercised, or integrated. A ZIP listing is not code review; a registered tool is not a successful call; a remote loader count is not a migrated database count.

The original Legal MCP pack now has a substantive source catalog: **27 tools, 11 resources and eight prompts**. All 76 files were safely expanded and matched to the retained donor; 40 files were read fully, three partly and 33 inventoried. Reusable document/report processing, clause comparison, research connectors, drafting/review methods and queue result shapes are recorded in the pack review. Repair prompt/signature drift, research field mapping and inline queue behavior as part of adaptation. The earlier three-file sample was too narrow to assess the pack's code contribution.

When borrowing code, attach upstream revision/license notice and local adaptation record. Preserve the original archive/fork, and test the selected contribution against the shared contracts before adopting it.

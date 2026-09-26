# Application copy and document workflow reconciliation

Date: 2026-09-24. Author: Codex.

Implementation/report commit: `dde633f34c28cbdaac1765b6fb644dfa170cf47e`, pushed to `origin/master`.
Coolify deployment `p2pfzzvgjn8g94ydbifhp8r4` finished at `2026-09-24T09:17:17Z` with that exact commit.

Changes: remove repeated shell/page titles and static policy prose; replace page introductions with functional descriptions; humanize provider/privacy and review controls; label draft fields; expose existing outline creation directly on the drafting page. No authorization/release control was relaxed. Backend data properties were not renamed or removed.

The substantive [document workflow and delivery gap report](../reviews/2026-09-24-DOCUMENT-WORKFLOW-AND-DELIVERY-GAPS.md) recovers the original office/form commitments, compares actual code and recorded conversion proof, specifies write-first/template-later and official-PDF workflows, preserves all 52 requirement groups, and defines seven implementation tasks. Full interactive office editing, native tracked-change round trips, official forms library and retroactive template application remain outstanding.

## Verification

- Production Next build and TypeScript passed; 36 routes generated.
- Shared design contract tests: 4 passed.
- Draft editing and template backend tests: 6 passed; one existing Python deprecation warning.
- Live browser checked all 33 navigation destinations: HTTP 200, exactly one h1 each, no duplicate `.module-title`, no targeted disclaimer/raw-flag phrases and no uncaught page errors.
- Live template catalog displayed. Create draft UI generated the selected template ID. Its POST was intercepted and fulfilled in the browser test to avoid writing to the real matter; this is UI action proof, with persistence separately covered by the backend tests.
- Live Document review, discovery names/options and Evidence ordering passed. Evidence order is catalog, list, missing evidence, discovery requests. Internal investigation and party discovery remain separate.
- Live confidentiality/external-sources checks found no prior repeated privilege disclaimer text or raw trust-posture flag.
- Initial template UI test used an over-specific accessible-label match and timed out. Inspection confirmed the selector existed; corrected scoped locator passed. No application defect was inferred from that test failure.
- No live matter writes or model invocations were used for this verification.

Local browser artifacts are in ignored `.reconciliation/whole-app-copy-proof.json`, `drafts-copy-proof.png`, and `review-discovery-proof.json`. They supplement this committed receipt.

## Documentation publication

Hosted Docstore note `note:advocatio_document_workflow_20260924`, revision 1, was written and independently read back. It records the owner's full office requirement and the report pointer. Full file ingestion/index freshness is not established: the local sync client lacks its required `CF_MCP_CLIENT_TOKEN`. Do not treat the governed note as proof the complete report file was indexed.

## Next bounded delivery

Follow Task 1 in the report: durable whole-document identity/revisions, blank writing/import/save/reopen/conflict behavior, reusing existing storage. Then prove the actual interactive office integration against representative files. The current outline entry is not the completion gate for that work.

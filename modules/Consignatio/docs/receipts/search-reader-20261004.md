# Case Bible search reader verification

Byline: Codex, GPT-6, 2026-10-04.
STATUS: VERIFIED reader and result presentation. Index completeness and automatic synchronization are separate.

Implementation: plugins/case-bible/tools/cb_vsearch.py and cb_search_server.py in the propria-plugins source repository. Server: /data/consignatio/search-tools/cb_search_server.py, isolated interpreter /data/consignatio/search-runtime/bin/python3; DuckDB1.5.6/httpx0.28.1. Search reuses the named-vector/model contracts from the Intake filesystem search boundary and Case Bible message search.

Verified live: hybrid queries across documents/messages/chats (240 candidates), keyword retrieval, literal text/source/date filters (two qualifying results), expanded JSON, indexed record lookup by UUID and human citations. Installed Claude human and Codex compact readers tested independently. No-match result tested. Five server unittest checks pass, covering literal filters, date exclusions, citation-preserving deduplication, excerpt markers, compact column packing and rejected ambiguous requests. Test outputs retain counts/check outcomes only; corpus excerpts are not copied into this receipt.

Live collection metadata: IntakeCorpus37,857, MsgEvents20260918 366,912, AiChatEvents20260918 446 objects. Model in sampled records: nvidia/nemotron-3-embed-1b; named vectors text_vector/text_nim checked against live schema. Counts are indexed objects, not unique files or a completeness proof.

DOC_PATCH_REPORT: replaced the content-search usage row and added search recipes, CocoIndex/index-store/DuckDB responsibilities, output formats, source references, candidate-filter boundaries and index coverage to the owning guide. Replaced active plugin search instructions, dependency/configuration references and restore verification route; preserved originals outside the active plugin in E:/AI_Workspace/plugins/to_be_deleted/case-bible-search-20261004. Generated discoverable skill from executable docstrings and CLI help. No source-data processing, index refresh, index migration or human-surface rebuild is claimed by these changes. The failing hosted chat UI endpoint was observed; this repair targets the plugin tool, not that UI endpoint.

Server implementation SHA256: cae6c1387cddbcfd996f696c09b60ee6aa39e726fce925addf3e873d1d1218af . Usage: ../CASE-BIBLE-CATALOG-GUIDE.md, Search for content.

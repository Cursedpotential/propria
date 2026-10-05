---
title: "atomic-tools"
type: tool-reference
status: SOURCE_VERIFIED
date: 2026-10-04
generated_by: "Codex / GPT-6"
revision: 1
tags: [propria, tools, wiki]
---

# atomic-tools

> _Byline: Codex · GPT-6 · 2026-10-04 — generated from the cited sources._

Read from the existing registry at `2026-10-05T02:19:30.474369+00:00`. 55 entries.

Registry presence is not a successful tool invocation. Preserve the declared side effects and execution policy when selecting a tool.

Gateway tools are callable through the attached `atomic-tools` MCP server or its documented authenticated HTTP run endpoint. ContextForge tools are callable by an MCP client attached to a virtual server that exposes the named tool.

Use the exact tool name and supply the required fields shown in its schema. A placeholder is not a valid real source or workflow ID.

## Browse before executing

ContextForge exposes one directory tool, `atomic-tools-atomic-tools`, over the catalog. Read-only examples:

```json
{"path":""}
{"path":"messages"}
{"path":"messages.sms-xml"}
```

To execute, add `run` containing the source locator and tool options, matching the contract returned by the browse call:

```json
{"path":"messages.sms-xml","run":{"source_ref":"b2://salem-data/<actual-object-key>","args":{}}}
```

This last example is a payload template, not an executed job. A real locator and the tool-specific options must be supplied. The gateway HTTP equivalent is authenticated `POST /tools/{id}/run` with `{source_ref,args}`.

The ContextForge atomic-tool description still reports 43 tools, while the directly read gateway reports 55. This is an observed description-freshness gap; the live gateway inventory is the source for the 55 listed IDs.

## `chunking.chonkie-fast`

Chonkie FastChunker (chonkie-core, SIMD) cutting at newlines into chunks of about 1000 characters.

id: `chunking.chonkie-fast`

capability: `chunk.message_spans`

formats: `['message_lines']`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `chunking.chonkie-fast`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `chunking.chonkie-recursive`

Chonkie RecursiveChunker, splitting on the largest separator that fits and recursing, chunks of about 1000 characters.

id: `chunking.chonkie-recursive`

capability: `chunk.message_spans`

formats: `['message_lines']`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `chunking.chonkie-recursive`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `chunking.chonkie-sentence`

Chonkie SentenceChunker on newline delimiters, packing whole messages into chunks of about 1000 characters.

id: `chunking.chonkie-sentence`

capability: `chunk.message_spans`

formats: `['message_lines']`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `chunking.chonkie-sentence`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `chunking.chonkie-token`

Chonkie TokenChunker over characters, fixed 1000-character windows (no overlap of its own).

id: `chunking.chonkie-token`

capability: `chunk.message_spans`

formats: `['message_lines']`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `chunking.chonkie-token`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `documents.extract-docling`

Docling structured document extraction with layout, tables, reading order, and OCR.

id: `documents.extract-docling`

capability: `extract.text`

formats: `[]`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `documents.extract-docling`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `documents.extract-text`

General text/OCR extraction (run BEFORE format parsing): native PDF text layer -> Tesseract OCR fallback for image/scanned docs. Returns text+pages+stats; low_confidence flags caller to escalate to vision OCR ($-gated).

id: `documents.extract-text`

capability: `extract.text`

formats: `[]`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `documents.extract-text`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `engine.poppler-certify-text`

Run bounded Poppler text extraction and return content-free reproducibility checks.

id: `engine.poppler-certify-text`

capability: `engine.certify`

formats: `['pdf']`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `engine.poppler-certify-text`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `engine.poppler-inspect`

Report the exact Poppler platform profile, executable identity, and readiness.

id: `engine.poppler-inspect`

capability: `engine.inspect`

formats: `['pdf']`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `engine.poppler-inspect`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `geo_map.leaflet`

Build a self-contained interactive Leaflet map (filters, popups, table) from point data given as records/path/csv_text; optional column mapping, reverse geocoding, and Evidence-source CSV export.

id: `geo_map.leaflet`

capability: `viz.geo_map`

formats: `[]`

side_effect: `filesystem_write`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `geo_map.leaflet`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `html.beautifulsoup4`

BeautifulSoup4 over the lxml parser: visible text, script/style/head/noscript removed, one block per line.

id: `html.beautifulsoup4`

capability: `extract.html_text`

formats: `['facebook_export_section_html', 'facebook_messenger_html', 'generic_html_document', 'google_takeout_activity_html', 'google_voice_html', 'imessage_export_html', 'snapchat_export_html', 'whatsapp_chat_html']`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `html.beautifulsoup4`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `html.docling`

Docling HTML backend: structured document (headings, lists, tables) exported as Markdown.

id: `html.docling`

capability: `extract.html_text`

formats: `['facebook_export_section_html', 'facebook_messenger_html', 'generic_html_document', 'google_takeout_activity_html', 'google_voice_html', 'imessage_export_html', 'snapchat_export_html', 'whatsapp_chat_html']`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `html.docling`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `html.html2text`

html2text: HTML to Markdown with no line wrapping; links and images kept as Markdown.

id: `html.html2text`

capability: `extract.html_text`

formats: `['facebook_export_section_html', 'facebook_messenger_html', 'generic_html_document', 'google_takeout_activity_html', 'google_voice_html', 'imessage_export_html', 'snapchat_export_html', 'whatsapp_chat_html']`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `html.html2text`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `html.lxml`

lxml.html (libxml2): visible text in document order, script/style/head/noscript dropped.

id: `html.lxml`

capability: `extract.html_text`

formats: `['facebook_export_section_html', 'facebook_messenger_html', 'generic_html_document', 'google_takeout_activity_html', 'google_voice_html', 'imessage_export_html', 'snapchat_export_html', 'whatsapp_chat_html']`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `html.lxml`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `html.markitdown`

MarkItDown (Microsoft): HTML to Markdown.

id: `html.markitdown`

capability: `extract.html_text`

formats: `['facebook_export_section_html', 'facebook_messenger_html', 'generic_html_document', 'google_takeout_activity_html', 'google_voice_html', 'imessage_export_html', 'snapchat_export_html', 'whatsapp_chat_html']`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `html.markitdown`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `html.selectolax`

selectolax (lexbor): visible text, script/style/head/noscript stripped; the fastest parser measured.

id: `html.selectolax`

capability: `extract.html_text`

formats: `['facebook_export_section_html', 'facebook_messenger_html', 'generic_html_document', 'google_takeout_activity_html', 'google_voice_html', 'imessage_export_html', 'snapchat_export_html', 'whatsapp_chat_html']`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `html.selectolax`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `html.unstructured`

unstructured partition_html: Title / NarrativeText / ListItem / Table elements, joined one per line.

id: `html.unstructured`

capability: `extract.html_text`

formats: `['facebook_export_section_html', 'facebook_messenger_html', 'generic_html_document', 'google_takeout_activity_html', 'google_voice_html', 'imessage_export_html', 'snapchat_export_html', 'whatsapp_chat_html']`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `html.unstructured`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `ingest.context-drain`

Drain pending working.context_record rows into Weaviate platform_context + the Graphiti CASE lane (PG-first change-detection projection, D-048). Idempotent; the manual stand-in for the CDC worker.

id: `ingest.context-drain`

capability: `ingest.context-drain`

formats: `[]`

side_effect: `derived_write`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `ingest.context-drain`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `messages.facebook-html`

Facebook/Messenger HTML export (legacy div.message + card _a6-g layouts) -> normalized messages

id: `messages.facebook-html`

capability: `parse.facebook`

formats: `[]`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `messages.facebook-html`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `messages.facebook-json`

Facebook/Messenger 'Download Your Information' JSON (message_N.json) -> normalized messages + calls (mojibake-repaired)

id: `messages.facebook-json`

capability: `parse.facebook`

formats: `[]`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `messages.facebook-json`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `messages.imessage-html`

ReagentX/imessage-exporter HTML export -> normalized iMessage records (content-sniffs vs Facebook HTML; tapbacks/replies/edits/read receipts/attachments preserved)

id: `messages.imessage-html`

capability: `parse.imessage`

formats: `[]`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `messages.imessage-html`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `messages.imessage-pdf`

iMessage PDF (print-to-PDF of imessage-exporter TXT/HTML) -> normalized records via native text extraction + the shared TXT grammar (tapbacks/replies/edits/receipts preserved)

id: `messages.imessage-pdf`

capability: `parse.imessage`

formats: `[]`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `messages.imessage-pdf`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `messages.imessage-txt`

ReagentX/imessage-exporter TXT export -> normalized iMessage records (tapbacks, replies, edits, read receipts, attachments preserved)

id: `messages.imessage-txt`

capability: `parse.imessage`

formats: `[]`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `messages.imessage-txt`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `messages.messaging-csv`

Tabular messaging CSV export (iMazing/AnyTrans/SMS/iMessage CSV) -> normalized records; column-flexible header mapping, every original column kept in attrs.raw_row

id: `messages.messaging-csv`

capability: `parse.messages-csv`

formats: `[]`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `messages.messaging-csv`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `messages.sms-xml`

"SMS Backup & Restore" XML (sms/mms/call) -> normalized message + call records, with forensic call-block flags

id: `messages.sms-xml`

capability: `parse.sms-xml`

formats: `[]`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `messages.sms-xml`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `messages.sms-xml-sbv`

SMS Backup & Restore XML via SBV (primary) -> normalized message + call records, with forensic call-block flags + MMS media handling

id: `messages.sms-xml-sbv`

capability: `parse.sms-xml`

formats: `[]`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `messages.sms-xml-sbv`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `messages.snapchat-json`

Snapchat 'My Data' chat_history.json -> normalized messages. Handles both the conversation-keyed and history-type-keyed export generations; retains non-text snaps as proof of contact; carries the saved-only export caveat on every record.

id: `messages.snapchat-json`

capability: `parse.snapchat`

formats: `[]`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `messages.snapchat-json`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `messages.transcript-marker`

Transcript-marker messaging export (.txt/.csv) with '[YYYY-MM-DD HH:MM AM/PM] Speaker:' lines -> normalized records; one record per message, speakers never blended, conversation_id derived from content. Sniffs the marker grammar and defers (raises) for tabular/other formats.

id: `messages.transcript-marker`

capability: `parse.messages-transcript`

formats: `[]`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `messages.transcript-marker`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `messages.whatsapp-txt`

WhatsApp exported _chat.txt -> normalized messages. Resolves the locale-dependent date order by declaration or inference and REFUSES when ambiguous; keeps multi-line continuations, colon-bearing sender names, system events (missed calls) and media-omitted markers.

id: `messages.whatsapp-txt`

capability: `parse.whatsapp`

formats: `[]`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `messages.whatsapp-txt`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `repair.audit-verify`

Verify the tool gateway's append-only execution hash chain

id: `repair.audit-verify`

capability: `repair.audit`

formats: `[]`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `repair.audit-verify`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `repair.capabilities`

List repair engines, dependency readiness, versions, and formats

id: `repair.capabilities`

capability: `repair.inspect`

formats: `[]`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `repair.capabilities`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `repair.detect`

Bounded-head format, encoding, engine, and cloud-placeholder detection

id: `repair.detect`

capability: `repair.detect`

formats: `[]`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `repair.detect`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `repair.flag-damaged`

Append a SHA-keyed damage-ledger entry after a complete streaming repair assessment

id: `repair.flag-damaged`

capability: `repair.flag`

formats: `[]`

side_effect: `append_only_ledger`

execution_policy: `manual_approval_required`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `repair.flag-damaged`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `repair.json-repair`

Repair one damaged JSON file (truncated, unquoted, trailing commas, stray text) and return the repaired document.

id: `repair.json-repair`

capability: `repair.json`

formats: `['json']`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `repair.json-repair`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `repair.pdf-derived`

Rebuild a damaged PDF into a separately hashed derived artifact

id: `repair.pdf-derived`

capability: `repair.derive`

formats: `[]`

side_effect: `derived_write`

execution_policy: `manual_approval_required`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `repair.pdf-derived`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `repair.pdf-inspect`

Read-only QPDF structural health inspection

id: `repair.pdf-inspect`

capability: `repair.inspect`

formats: `[]`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `repair.pdf-inspect`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `repair.preview`

Stream repaired structural chunks and return bounded samples plus a repair report

id: `repair.preview`

capability: `repair.preview`

formats: `[]`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `repair.preview`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `repair.quarantine-copy`

Create a verified quarantine copy; original remains and is skipped through the damage ledger

id: `repair.quarantine-copy`

capability: `repair.quarantine`

formats: `[]`

side_effect: `verified_copy_and_ledger`

execution_policy: `manual_approval_required`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `repair.quarantine-copy`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `repair.quarantine-plan`

Create a read-only, hashed quarantine-copy manifest

id: `repair.quarantine-plan`

capability: `repair.quarantine-plan`

formats: `[]`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `repair.quarantine-plan`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `repair.write-derived`

Write a new repaired artifact without modifying the custody original

id: `repair.write-derived`

capability: `repair.derive`

formats: `[]`

side_effect: `derived_write`

execution_policy: `manual_approval_required`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `repair.write-derived`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `transcripts.chatgpt-custom-gpt-md`

ChatGPT Custom GPT markdown export (You asked: / ChatGPT Replied:) -> normalized records

id: `transcripts.chatgpt-custom-gpt-md`

capability: `parse.transcript`

formats: `[]`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `transcripts.chatgpt-custom-gpt-md`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `transcripts.chatgpt-official`

ChatGPT official data-export conversations.json (mapping tree) -> normalized message records

id: `transcripts.chatgpt-official`

capability: `parse.transcript`

formats: `[]`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `transcripts.chatgpt-official`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `transcripts.chatgpt-share`

ChatGPT 'Share' markdown export -> normalized message records

id: `transcripts.chatgpt-share`

capability: `parse.transcript`

formats: `[]`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `transcripts.chatgpt-share`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `transcripts.claude-ai-export`

claude.ai data-export conversations.json -> normalized message records

id: `transcripts.claude-ai-export`

capability: `parse.transcript`

formats: `[]`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `transcripts.claude-ai-export`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `transcripts.claude-code`

Claude Code simple JSONL export (role/content lines) -> normalized message records

id: `transcripts.claude-code`

capability: `parse.transcript`

formats: `[]`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `transcripts.claude-code`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `transcripts.claude-code-jsonl`

Claude Code session .jsonl -> normalized message records (text blocks only)

id: `transcripts.claude-code-jsonl`

capability: `parse.transcript`

formats: `[]`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `transcripts.claude-code-jsonl`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `transcripts.claude-md`

Claude markdown copy-paste transcript -> normalized message records

id: `transcripts.claude-md`

capability: `parse.transcript`

formats: `[]`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `transcripts.claude-md`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `transcripts.gemini-chrome`

Gemini Chrome-extension markdown export -> normalized message records

id: `transcripts.gemini-chrome`

capability: `parse.transcript`

formats: `[]`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `transcripts.gemini-chrome`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `transcripts.gemini-json`

Gemini JSON export -> normalized message records

id: `transcripts.gemini-json`

capability: `parse.transcript`

formats: `[]`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `transcripts.gemini-json`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `transcripts.gemini-md`

Gemini markdown export (**You:** / **Gemini:** markers) -> normalized message records

id: `transcripts.gemini-md`

capability: `parse.transcript`

formats: `[]`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `transcripts.gemini-md`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `transcripts.generic-md`

Generic markdown chat with role markers (fallback) -> normalized message records

id: `transcripts.generic-md`

capability: `parse.transcript`

formats: `[]`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `transcripts.generic-md`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `transcripts.markdown`

Plain .md/.txt transcript -> one whole-file record (fallback)

id: `transcripts.markdown`

capability: `parse.transcript`

formats: `[]`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `transcripts.markdown`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `transcripts.perplexity-contexts`

Perplexity contexts data-export (conversations[].entries[].query/answer) -> normalized records

id: `transcripts.perplexity-contexts`

capability: `parse.transcript`

formats: `[]`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `transcripts.perplexity-contexts`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `transcripts.perplexity-gdpr`

Perplexity GDPR data-export JSON -> normalized message records

id: `transcripts.perplexity-gdpr`

capability: `parse.transcript`

formats: `[]`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `transcripts.perplexity-gdpr`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `transcripts.perplexity-md`

Perplexity generic markdown export -> normalized message records

id: `transcripts.perplexity-md`

capability: `parse.transcript`

formats: `[]`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `transcripts.perplexity-md`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `transcripts.perplexity-plugin`

Perplexity plugin copy-paste markdown -> normalized message records

id: `transcripts.perplexity-plugin`

capability: `parse.transcript`

formats: `[]`

side_effect: `read_only`

execution_policy: `manual_or_auto`

Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.

Source: `assets/tool-inventory.json` → `live_registry.gateway_tools` → `transcripts.perplexity-plugin`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

Back to [[Code/wiki/plugin-inventory|Plugin inventory]].

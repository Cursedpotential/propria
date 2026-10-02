"""html_text — one registered tool per HTML text-extraction library (capability ``extract.html_text``).

One library = one module = one registry tool = one Temporal Activity
(``server/temporal/html_tool_activities.py``). The DuckDB webbed templates
(``facebook_messenger_html_v1``, ``generic_html_document_v1``) are the
engine's primary HTML handlers; these tools are the selectable alternatives
and fallbacks. Which tool is the default for a file type is declared in each
module's ``quality`` map (``primary`` / ``fallback`` / ``experimental``) and
was set from the per-file-type bench in
``docs/receipts/2026-10-02-html-tool-bench/``.

Byline: Claude Code · Sonnet · 2026-10-02
"""

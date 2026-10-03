"""chunking — one registry tool per Chonkie chunker (capability ``chunk.message_spans``).

Byline: Claude Code · Sonnet 5.5 · 2026-10-02

One chunker = one module = one registry tool (owner rule 2026-10-02: one tool, one job, everything selectable). Each tool
wraps the SAME function the Proffer chunk pipeline runs, ``server.context_chunks.chunker.chunk_spans``, so a tool call
and the pipeline cut a thread identically.

Registered here: the model-free Chonkie chunkers (token, fast, sentence, recursive), because the tool-runtime image can
run them (chonkie + chonkie-core, no model). NOT registered: ``neural_distilbert`` (the pipeline default) and
``neural_modernbert``. They load a transformer model (torch, transformers and the model weights), which the tool-runtime
image deliberately does not carry (tests/test_tool_runtime_deploy.py forbids it); they run in the temporal-worker as the
chunk Activities. Registering them here would list tools the runtime cannot run.
"""

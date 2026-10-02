"""server/context_chunks — conversation chunks for Weaviate, built from committed Postgres messages.

Byline: Claude Code · Sonnet 5.5 · 2026-10-02

Owner rules (2026-10-02): Postgres holds every single message and is the record of truth. Weaviate holds
only conversation CHUNKS, never one entry per message; each chunk links back to the Postgres message ids
it covers. Calls are one Weaviate entry per call-log FILE; the individual calls stay in Postgres.

One unit, one job (the ATOMICITY section of the repository AGENTS.md):

    chunker.py   text lines -> message spans (Chonkie; the default is the Neural distilbert chunker)
    source.py    Postgres reads: the committed threads, their messages, the call-log files
    embed.py     NVIDIA NIM nemotron-3-embed-1b, batched, with the NIM input guards
    store.py     the Weaviate REST boundary for ProfferChunks20261002
    service.py   chunk a thread (a plan of spans) and embed+publish a thread's chunks, as separate units
    rechunk.py   the re-chunk entry point (dry-run, then run) over threads already in Postgres
    remove_per_message.py   the owner-run removal of the old per-message objects, after verification

``server/temporal/chunk_activities.py`` wraps service.py as three Temporal Activities on queue
``evidence-pipeline``; the Go ProfferWorkflow calls them after the first-party threads are committed.
"""

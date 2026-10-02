# =============================================================================
# temporal-worker — first P0 build artifact (plan §5 review amendment,
# "the Temporal worker is its own Coolify app whose image carries the full
# server/ dependency stack ... building that image is the first concrete P0
# task, before any workflow code"). Byline: Claude Code · Opus 5 · 2026-08-23
# Full plan: docs/plans/TEMPORAL-INTEGRATION-PLAN-2026-08-23.md (D-067)
#
# Joins Temporal task queue `evidence-pipeline`. Connects to the Temporal
# frontend via TEMPORAL_ADDRESS (default temporal-server:7233 on the shared
# `probata` docker network — see deploy/temporal/compose.temporal.yaml).
#
# Needs the SAME application env as agentos-api (DB_*, WEAVIATE_*, PORTKEY_*,
# etc.) because Activities call the EXISTING pipeline code unchanged
# (custody.py, store.py, workflows.py's Step logic wrapped as Activities) —
# see plan §1: custody hashing and walk derivation stay platform-side,
# invoked FROM Activities, never reimplemented as workflow code.
#
# Base image matches every other docker/*/Dockerfile in this repo
# (python:3.12-slim — checked docker/coolify-mcp, docker/sandbox,
# docker/surreal-phase1-runner; the repo-ROOT Dockerfile uses agnohq/python:3.12
# for the AgentOS app itself, which is a different base for a different app).
#
# Multi-stage is unnecessary here — no compiled frontend/build step, just the
# existing pure-Python server/ package plus the temporalio SDK.
#
# `python -m server.temporal.worker` exists (server/temporal/worker.py); the app is
# declared in deploy/temporal-worker.yaml. Byline: Claude Code · Sonnet · 2026-10-02
# =============================================================================
FROM python:3.12-slim

WORKDIR /app
ENV PYTHONPATH=/app

COPY requirements.txt ./
# DuckDB runs the ELT steps the Go workflows hand to this worker (owner 2026-10-02).
# zipfs reads inside ZIP exports and is loaded at first use (INSTALL zipfs FROM community).
# Byline: Claude Code · Opus 5.5 · 2026-10-02
RUN pip install --no-cache-dir -r requirements.txt \
    && pip install --no-cache-dir temporalio duckdb==1.5.6

# HTML text-extraction libraries, one Activity each (html_tool_activities.py). Pinned exactly in
# requirements-html-tools.txt; libmagic1 is the system library unstructured's file detection loads.
# Byline: Claude Code · Sonnet · 2026-10-02
RUN apt-get update && apt-get install -y --no-install-recommends libmagic1 \
    && rm -rf /var/lib/apt/lists/*
COPY requirements-html-tools.txt ./
RUN pip install --no-cache-dir -r requirements-html-tools.txt
# Conversation chunking (owner 2026-10-02): Chonkie's Neural chunker on distilbert, CPU-only torch, the model baked
# in at build time and loaded offline at run time (no download from the worker, no HF token, nothing fetched when a
# run starts). The model revision is pinned: it is the snapshot the head-to-head used (devbox cache,
# models--mirth--chonky_distilbert_base_uncased_1/snapshots/01d8aae0...). Image growth: about +1.1 GB on disk
# (torch 0.74, transformers and friends 0.15, model 0.27); about +0.5 GB to pull.
# Byline: Claude Code · Sonnet 5.5 · 2026-10-02
COPY deploy/docker/temporal-worker/requirements-chunking.txt ./requirements-chunking.txt
RUN pip install --no-cache-dir --extra-index-url https://download.pytorch.org/whl/cpu -r requirements-chunking.txt
ENV HF_HOME=/opt/hf-cache
RUN python -c "from huggingface_hub import snapshot_download; snapshot_download('mirth/chonky_distilbert_base_uncased_1', revision='01d8aae08726368a1b1645de2a7086610f2e86a5')" \
    && chmod -R a+rX /opt/hf-cache
ENV HF_HUB_OFFLINE=1 TRANSFORMERS_OFFLINE=1
# Prove at build time that the baked model loads offline and classifies, so a bad image never reaches a run.
RUN python -c "import chonkie; c = chonkie.NeuralChunker(model='mirth/chonky_distilbert_base_uncased_1', min_characters_per_chunk=1); print('neural chunker ok,', len(c.chunk('see you at six sounds good did you eat yes thanks')), 'chunk(s)')"

# Activities call into the existing pipeline code, so the worker needs the
# same application surface as agentos-api, not just an SDK stub.
COPY server/ ./server/
COPY scripts/ ./scripts/
COPY sql/ ./sql/

CMD ["python", "-m", "server.temporal.worker"]

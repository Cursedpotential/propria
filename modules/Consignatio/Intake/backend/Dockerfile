# Coco super index — backend service image.
# Byline: Claude Code · Opus 5 · 2026-09-22; staged index, shared chunker and Temporal worker added by Claude Code ·
# Sonnet 5.5 · 2026-10-02
#
# Runs next to the data on ovh-files (owner rule: nothing runs on the desktop). The API
# binds 0.0.0.0:8765 inside the container; the compose file publishes it on the tailnet
# address only, never on a public interface.
#
# One image, two processes (deploy/superindex.compose.yml): the search API (this CMD) and the Temporal worker
# (`python -m casebible_index.temporal_worker`) that runs the scheduled index cycle's Activities.
#
# The image lane (`image-provision` / `image-index` / `image-search`, collection
# `IntakeImageV1`) is a SEPARATE CocoIndex app in the same package, with its own app name,
# state directory, lock and Weaviate collection. It is not folded into the text app and must
# not be. Its two native tools — exiftool and tesseract — are installed here so that app can
# be run from this image later without a second build (owner relay 2026-09-22 10:40).

FROM ghcr.io/astral-sh/uv:0.9.7-python3.12-bookworm-slim

ENV UV_COMPILE_BYTECODE=1 \
    UV_LINK_MODE=copy \
    UV_PYTHON_DOWNLOADS=never \
    PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1

RUN apt-get update \
    && apt-get install --no-install-recommends --yes libimage-exiftool-perl tesseract-ocr

WORKDIR /app

# Dependency layer first so a source edit does not re-resolve the lock.
COPY pyproject.toml uv.lock README.md ./
RUN --mount=type=cache,target=/root/.cache/uv \
    uv sync --frozen --no-install-project --no-dev

COPY src ./src
COPY main.py ./main.py
RUN --mount=type=cache,target=/root/.cache/uv \
    uv sync --frozen --no-dev

# Conversation chunking (owner 2026-10-02): Probata's shared module server/context_chunks is IMPORTED, never copied
# into this source tree. The compose build context `probata_server` is modules/Probata/probata/server; only the
# chunk package is taken (its db.py, source.py and service.py need Postgres and are not imported here). Chonkie's
# Neural chunker needs torch: the CPU wheel, not the 2 GB CUDA one. These installs come AFTER `uv sync`, which would
# remove them as extraneous. The model is baked in so a cold container never depends on huggingface.co.
COPY --from=probata_server context_chunks /app/server/context_chunks
RUN touch /app/server/__init__.py \
    && uv pip install --python /app/.venv/bin/python --index-url https://download.pytorch.org/whl/cpu torch \
    && uv pip install --python /app/.venv/bin/python "chonkie[neural]==1.7.0"
ENV HF_HOME=/opt/hf \
    PYTHONPATH=/app
RUN /app/.venv/bin/python -c "from huggingface_hub import snapshot_download as s; s('mirth/chonky_distilbert_base_uncased_1')" \
    && chmod -R a+rX /opt/hf

ENV PATH="/app/.venv/bin:${PATH}" \
    CASEBIBLE_OUTPUT_DIR=/var/lib/superindex/output \
    INTAKE_LOCK_DIR=/var/lib/superindex/state/.source-locks \
    TMPDIR=/var/lib/superindex/spool

# Non-root: the service only ever reads the bucket and writes its own derived output.
RUN useradd --system --create-home --uid 10001 superindex \
    && mkdir -p /var/lib/superindex/output /var/lib/superindex/state /var/lib/superindex/spool \
    && chown -R superindex:superindex /var/lib/superindex
USER superindex

EXPOSE 8765

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
    CMD python -c "import urllib.request,sys; sys.exit(0 if urllib.request.urlopen('http://127.0.0.1:8765/health', timeout=4).status==200 else 1)"

CMD ["uvicorn", "casebible_index.api:app", "--host", "0.0.0.0", "--port", "8765"]

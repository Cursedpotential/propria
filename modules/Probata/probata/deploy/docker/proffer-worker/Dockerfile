# Toolchain pinned to match modules/engine/go.mod (`go 1.26.6`).
FROM golang:1.26-bookworm AS builder

# Hermetic vendored build (repair post-restructure, 2026-09-02): engine/ moved to
# modules/engine/ and sbv moved to modules/forks/sbv/ (a private, gitignored-locally
# submodule Coolify's clone may not have credentials for). modules/engine/vendor/
# snapshots the full module graph, INCLUDING github.com/lowcarbdev/sbv via the
# go.mod `replace => ../forks/sbv`, so the build needs neither network access nor
# a sibling modules/forks/sbv checkout — proven by building with that directory
# absent. Build context is the repo root (see deploy/*.yaml `context: ..`).
WORKDIR /src/engine
COPY modules/engine/ ./
RUN CGO_ENABLED=1 go build \
    -mod=vendor \
    -tags fts5 \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/proffer-worker \
    ./cmd/proffer-worker
RUN CGO_ENABLED=1 go build -mod=vendor -tags fts5 -trimpath -ldflags="-s -w" \
    -o /out/derive-sms ./cmd/derive-sms
# Byline: Codex, 2026-10-04. Protected operator CLI shares this image's actual parser runtime.
RUN CGO_ENABLED=1 go build -mod=vendor -tags fts5 -trimpath -ldflags="-s -w" \
    -o /out/toolkit-currency-review ./cmd/toolkit-currency-review

FROM debian:bookworm-slim

# Extend this freshly built worker with the existing native HTML/PDF parser runtime.
# Inputs: pinned requirements and selected existing parser modules. Outputs: private bounded extractor environment.
# Effects: image installation only; no OCR, source processing, credentials or separate service.
# Choose for optional toolkit validation; keep the existing Go builder and entrypoint.
# Byline: Codex · GPT-6.1 · 2026-10-04.
RUN apt-get update \
    && apt-get install --no-install-recommends -y ca-certificates tzdata python3 python3-venv \
    && python3 -m venv --copies /opt/library-validation/venv \
    && groupadd --system --gid 10001 platform \
    && useradd --system --uid 10001 --gid platform --home-dir /nonexistent --shell /usr/sbin/nologin platform \
    && install -d -o platform -g platform -m 0750 \
       /data/proffer/source-objects \
       /data/proffer/parser-bundles \
       /data/proffer/normalized-bundles \
       /data/proffer/inventory-manifests \
       /data/proffer/derive-scratch

COPY modules/engine/extraction/libraryvalidation/runtime/requirements.txt /opt/library-validation/requirements.txt
RUN /opt/library-validation/venv/bin/python -m pip install --no-cache-dir --only-binary=:all: --no-deps \
    -r /opt/library-validation/requirements.txt
COPY server/__init__.py /opt/library-validation/project/server/__init__.py
COPY server/tools/__init__.py server/tools/registry.py /opt/library-validation/project/server/tools/
COPY server/tools/extractors/__init__.py server/tools/extractors/extract_text.py /opt/library-validation/project/server/tools/extractors/
COPY server/tools/extractors/html_text/__init__.py server/tools/extractors/html_text/_common.py server/tools/extractors/html_text/_ranks.py server/tools/extractors/html_text/lxml_html.py /opt/library-validation/project/server/tools/extractors/html_text/
COPY modules/engine/extraction/libraryvalidation/extractor_bridge.py /opt/library-validation/project/modules/engine/extraction/libraryvalidation/extractor_bridge.py
RUN install -d -o 10001 -g 10001 -m 0700 /data/proffer/library-validation

COPY --from=builder /out/proffer-worker /usr/local/bin/proffer-worker
# derive-sms (2026-09-20): SBV streams a large SMS XML into per-thread NDJSON + media beside the source.
COPY --from=builder /out/derive-sms /usr/local/bin/derive-sms
COPY --from=builder /out/toolkit-currency-review /usr/local/bin/toolkit-currency-review

USER 10001:10001
ENV SOURCE_OBJECT_DIR=/data/proffer/source-objects \
    PARSER_BUNDLE_DIR=/data/proffer/parser-bundles \
    NORMALIZED_BUNDLE_DIR=/data/proffer/normalized-bundles \
    INVENTORY_MANIFEST_DIR=/data/proffer/inventory-manifests \
    TOOLKIT_VALIDATION_PYTHON=/opt/library-validation/venv/bin/python \
    TOOLKIT_VALIDATION_PROJECT_ROOT=/opt/library-validation/project \
    TOOLKIT_VALIDATION_BRIDGE_FILE=/opt/library-validation/project/modules/engine/extraction/libraryvalidation/extractor_bridge.py \
    TOOLKIT_VALIDATION_SCRATCH_ROOT=/data/proffer/library-validation

# Fail the image build if the non-root runtime cannot import the pinned existing parsers.
RUN PYTHONPATH=/opt/library-validation/project /opt/library-validation/venv/bin/python -B \
    /opt/library-validation/project/modules/engine/extraction/libraryvalidation/extractor_bridge.py --check-runtime

ENTRYPOINT ["/usr/local/bin/proffer-worker"]

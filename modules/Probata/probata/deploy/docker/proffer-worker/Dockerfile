# Toolchain pinned to match modules/engine/go.mod (`go 1.26.6`).
FROM golang:1.26-bookworm AS builder

# Bound compiler/test concurrency within the existing trial host build slot.
# Inputs: vendored Go source; outputs: tested binaries. Effects: image build only.
# Byline: Codex · GPT-6.1-sol · 2026-10-08.
ENV GOMAXPROCS=1 GOMEMLIMIT=1024MiB GOFLAGS=-p=1 AI_WORKPRODUCT_TEST_ROOT=/tmp/to_be_deleted

# Hermetic vendored build (repair post-restructure, 2026-09-02): engine/ moved to
# modules/engine/ and sbv moved to modules/forks/sbv/ (a private, gitignored-locally
# submodule Coolify's clone may not have credentials for). modules/engine/vendor/
# snapshots the full module graph, INCLUDING github.com/lowcarbdev/sbv via the
# go.mod `replace => ../forks/sbv`, so the build needs neither network access nor
# a sibling modules/forks/sbv checkout — proven by building with that directory
# absent. Build context is the repo root (see deploy/*.yaml `context: ..`).
WORKDIR /src/modules/Probata/probata/modules/engine
COPY modules/Probata/probata/modules/engine/ ./
# Preserve repository-relative inputs for the existing focused contract tests.
# Inputs: five tracked schema/deployment/fixture files. Outputs: test-only builder files.
# Effects: build context copy only; final image does not inherit these files.
# Byline: Codex · GPT-6.1-sol · 2026-10-08.
COPY modules/Probata/probata/deploy/proffer-worker.yaml /src/modules/Probata/probata/deploy/proffer-worker.yaml
COPY modules/Probata/probata/sql/bootstrap/schema_snapshot_20260907.sql /src/modules/Probata/probata/sql/bootstrap/schema_snapshot_20260907.sql
COPY modules/Probata/probata/tests/fixtures/probata_mixed_demo/imessage-thread.txt /src/modules/Probata/probata/tests/fixtures/probata_mixed_demo/imessage-thread.txt
COPY modules/Consignatio/casebible/catalog_reconcile/toolkit_recovery_catalog_20261004.sql /src/modules/Consignatio/casebible/catalog_reconcile/toolkit_recovery_catalog_20261004.sql
COPY modules/Probata/probata/sql/bootstrap/toolkit_working_catalog_20261005.sql /src/modules/Probata/probata/sql/bootstrap/toolkit_working_catalog_20261005.sql
# One focused test invocation shares the existing CGO/fts5/vendor build runtime.
RUN CGO_ENABLED=1 go test -mod=vendor -tags fts5 \
    ./surrealsink ./runtimeapi ./temporal/cmd/starter \
    ./approvedgraphprojectionflow ./approvedgraphqueryflow ./extraction/service ./sourceformat ./approvedgraphai ./approvedaigraphprojectionflow ./profferworker \
    ./postgres ./proffer ./aicontextsource ./cmd/context-graph
# Check only the existing AI context packaging Activities without broad activity tests.
RUN CGO_ENABLED=1 go test -mod=vendor -tags fts5 ./activities -run '^TestAIContextPackage' -count=1
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
# Canonical graph schema emission and bounded operator operations use the same image.
RUN CGO_ENABLED=1 go build -mod=vendor -tags fts5 -trimpath -ldflags="-s -w" \
    -o /out/context-graph ./cmd/context-graph

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

COPY modules/Probata/probata/modules/engine/extraction/libraryvalidation/runtime/requirements.txt /opt/library-validation/requirements.txt
RUN /opt/library-validation/venv/bin/python -m pip install --no-cache-dir --only-binary=:all: --no-deps \
    -r /opt/library-validation/requirements.txt
COPY modules/Probata/probata/server/__init__.py /opt/library-validation/project/server/__init__.py
COPY modules/Probata/probata/server/tools/__init__.py modules/Probata/probata/server/tools/registry.py /opt/library-validation/project/server/tools/
COPY modules/Probata/probata/server/tools/extractors/__init__.py modules/Probata/probata/server/tools/extractors/extract_text.py /opt/library-validation/project/server/tools/extractors/
COPY modules/Probata/probata/server/tools/extractors/html_text/__init__.py modules/Probata/probata/server/tools/extractors/html_text/_common.py modules/Probata/probata/server/tools/extractors/html_text/_ranks.py modules/Probata/probata/server/tools/extractors/html_text/lxml_html.py /opt/library-validation/project/server/tools/extractors/html_text/
COPY modules/Probata/probata/modules/engine/extraction/libraryvalidation/extractor_bridge.py /opt/library-validation/project/modules/engine/extraction/libraryvalidation/extractor_bridge.py
RUN install -d -o 10001 -g 10001 -m 0700 /data/proffer/library-validation

COPY --from=builder /out/proffer-worker /usr/local/bin/proffer-worker
# derive-sms (2026-09-20): SBV streams a large SMS XML into per-thread NDJSON + media beside the source.
COPY --from=builder /out/derive-sms /usr/local/bin/derive-sms
COPY --from=builder /out/toolkit-currency-review /usr/local/bin/toolkit-currency-review
COPY --from=builder /out/context-graph /usr/local/bin/context-graph

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

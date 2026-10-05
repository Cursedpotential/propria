# Exercise the bounded existing extractor dependency extension against a parent-built Go worker.
# Inputs: PROFFER_WORKER_IMAGE pinned to the parent-integrated Go build; context: modules/Probata/probata.
# Outputs: the same worker/entrypoint plus private Python and baked existing parser modules.
# Effects: image build only; no credentials, deployment, production writes or source mounts.
# Choose as a reference/smoke fixture only; production copies these steps into the existing worker final stage.
# The existing Coolify worker build supplies fresh Go code; do not create a standalone application or circular image build.
# Byline: Codex · GPT-6.1 · 2026-10-04.
ARG PROFFER_WORKER_IMAGE
FROM ${PROFFER_WORKER_IMAGE}
USER root
RUN apt-get update \
    && apt-get install --no-install-recommends -y python3 python3-venv \
    && python3 -m venv --copies /opt/library-validation/venv
COPY modules/engine/extraction/libraryvalidation/runtime/requirements.txt /opt/library-validation/requirements.txt
RUN /opt/library-validation/venv/bin/python -m pip install --no-cache-dir --only-binary=:all: --no-deps \
    -r /opt/library-validation/requirements.txt
# The imports below need only these existing modules and standard-library registry plumbing.
COPY server/__init__.py /opt/library-validation/project/server/__init__.py
COPY server/tools/__init__.py server/tools/registry.py /opt/library-validation/project/server/tools/
COPY server/tools/extractors/__init__.py server/tools/extractors/extract_text.py /opt/library-validation/project/server/tools/extractors/
COPY server/tools/extractors/html_text/__init__.py server/tools/extractors/html_text/_common.py server/tools/extractors/html_text/_ranks.py server/tools/extractors/html_text/lxml_html.py /opt/library-validation/project/server/tools/extractors/html_text/
COPY modules/engine/extraction/libraryvalidation/extractor_bridge.py /opt/library-validation/project/modules/engine/extraction/libraryvalidation/extractor_bridge.py
ENV TOOLKIT_VALIDATION_PYTHON=/opt/library-validation/venv/bin/python \
    TOOLKIT_VALIDATION_PROJECT_ROOT=/opt/library-validation/project \
    TOOLKIT_VALIDATION_BRIDGE_FILE=/opt/library-validation/project/modules/engine/extraction/libraryvalidation/extractor_bridge.py \
    TOOLKIT_VALIDATION_SCRATCH_ROOT=/data/proffer/library-validation
RUN install -d -o 10001 -g 10001 -m 0700 /data/proffer/library-validation
USER 10001:10001
RUN PYTHONPATH=/opt/library-validation/project /opt/library-validation/venv/bin/python -B \
    /opt/library-validation/project/modules/engine/extraction/libraryvalidation/extractor_bridge.py --check-runtime
# Inherit the parent worker entrypoint; do not start it during build or replace its other registrations.

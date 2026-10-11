# Docstore service code, built from the Propria monorepo root.
# Byline: Codex · GPT-6.1-sol · 2026-10-07.
# The seven documentation roots and private additions are staged on the host at
# /data/probata/volumes/docstore-sources, then bound writable at /app/docs.
# ADR and governed source sync writes remain durable; service.py verifies the
# complete source tree and private additions before starting either API.
FROM python:3.13-slim

WORKDIR /app
ENV PYTHONDONTWRITEBYTECODE=1 PYTHONUNBUFFERED=1 DOCSTORE_MCP_PORT=8084

COPY modules/Probata/probata/deploy/docker/docstore/requirements.lock /app/requirements.lock
RUN pip install --no-cache-dir -r /app/requirements.lock

COPY modules/Probata/probata/scripts/docstore/ /app/scripts/docstore/
COPY modules/Probata/probata/plugins/docstore/control/ /app/plugins/docstore/control/
COPY modules/Probata/probata/tests/release/ /app/tests/
COPY modules/Probata/probata/deploy/docker/docstore/pyproject.toml /app/pyproject.toml
COPY modules/Probata/probata/deploy/docker/docstore/Dockerfile /app/deploy/docker/docstore/Dockerfile
COPY modules/Probata/probata/deploy/docstore.yaml /app/deploy/docstore.yaml

EXPOSE 8000 8084
CMD ["python", "scripts/docstore/service.py"]

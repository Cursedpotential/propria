# =============================================================================
# docstore — the whole Docstore service, built FROM GIT
# =============================================================================
# Byline: Claude Code · Opus 5 · 2026-09-28 (0.9.0)
#
# Replaces the host-built tree at /data/propria/releases/docstore-0.8.1, which was in no
# git repository. That tree had drifted BEHIND main: its scripts/docstore/scope.py was the
# five-root version using pre-2026-09-20 module paths, so the registry validator rejected the
# seven-root registry, `docstore_health` reported five stale `allowed_source_roots`, and the
# 2026-09-27 sync failed. Building from git is the fix, not a repackaging.
#
# BUILD CONTEXT IS THE MONOREPO ROOT (Propria), not the Probata module: the documentation roots
# this service indexes live in seven different modules, and the image has to assemble all of them.
# The Coolify app therefore sets base_directory=/ (see deploy/docstore.yaml).
#
# WHY THE DOCS ARE IN THE IMAGE (owner choice A, 2026-09-28). On the desktop, Propria/docs holds
# a junction per project into that module's docs directory; junctions are not in git, so a clone
# has only the real module paths. This stage rebuilds the SAME layout from those module paths, so
# scripts/docstore/scope.py's roots resolve unchanged and, critically, every canonical_prefix —
# document identity, trailing slash included — stays byte-for-byte what the live store was built
# with. flow_docs.py computes REPO_ROOT as parents[2] of its own file, so with the code at
# /app/scripts/docstore/ the roots must land at /app/docs/<project>. They do.
#
# The previous arrangement fed a host mirror with `client.py sync` from the desktop. That is gone:
# a docs push rebuilds this image and the start-up pass is the change event, which is the contract
# the retired docstore-worker app had.
# -----------------------------------------------------------------------------

# --- Stage 1: assemble the seven documentation roots -------------------------
# Each COPY is a separate layer keyed to one module, so a docs change in one product does not
# invalidate the others. Quarantine and private material is dropped below rather than trusted to
# the registry's excluded_patterns alone — the host mirror was indexing 101 files under
# to_be_deleted/ because nothing enforced it at assembly time.
FROM busybox:1.37 AS docs

WORKDIR /sources
COPY docs/ /sources/docs/
COPY modules/Probata/probata/docs/ /sources/docs/probata/
COPY modules/Consignatio/docs/ /sources/docs/consignatio/
COPY modules/Consignatio/Intake/docs/ /sources/docs/consignatio-intake/
COPY modules/Legal-desktop/docs/ /sources/docs/advocatio/
COPY modules/vestigia-geodata_processor/traceiq-rebuild/docs/ /sources/docs/vestigia/
COPY modules/FL-MCP/docs/ /sources/docs/family-court/

# Drop what the registry excludes, at assembly time, and anything that is not Markdown or the
# registry itself. `find -delete` here removes build-stage copies only; no source is touched.
RUN set -eu; \
    find /sources/docs -type d \( -name to_be_deleted -o -name private -o -name .docstore \
        -o -name .docstore-control -o -name .remember -o -name .cnf -o -name .codex \
        -o -name .agents -o -name node_modules -o -name __pycache__ \) -prune -exec rm -rf {} + ; \
    find /sources/docs -type f ! -name '*.md' ! -name 'docstore-source-registry.json' -delete ; \
    find /sources/docs -type d -empty -delete ; \
    echo "assembled markdown: $(find /sources/docs -name '*.md' | wc -l)" ; \
    for p in probata consignatio consignatio-intake advocatio vestigia family-court; do \
        test -d "/sources/docs/$p" || { echo "MISSING ROOT: docs/$p"; exit 1; }; \
    done ; \
    test -f /sources/docs/docstore-source-registry.json || { echo "MISSING registry"; exit 1; }

# --- Stage 2: the service ----------------------------------------------------
FROM python:3.13-slim

WORKDIR /app
ENV PYTHONDONTWRITEBYTECODE=1 PYTHONUNBUFFERED=1 DOCSTORE_MCP_PORT=8084

# Dependencies first: the lockfile changes far less often than the code or the docs.
COPY modules/Probata/probata/deploy/docker/docstore/requirements.lock /app/requirements.lock
RUN pip install --no-cache-dir -r /app/requirements.lock

# The service: the worker API + pipeline (scripts/docstore), the control MCP server
# (plugins/docstore/control), and the release test suite.
COPY modules/Probata/probata/scripts/docstore/ /app/scripts/docstore/
COPY modules/Probata/probata/plugins/docstore/control/ /app/plugins/docstore/control/
COPY modules/Probata/probata/tests/release/ /app/tests/
COPY modules/Probata/probata/deploy/docker/docstore/pyproject.toml /app/pyproject.toml

# The documentation, assembled above.
COPY --from=docs /sources/docs/ /app/docs/

# Fail the build rather than ship an image whose roots the validator will reject at run time.
RUN python -c "\
import json, pathlib, sys; \
sys.path.insert(0, '/app/scripts/docstore'); \
from scope import ROOTS, validate_registry; \
registry = json.loads(pathlib.Path('/app/docs/docstore-source-registry.json').read_text(encoding='utf-8')); \
validate_registry(registry); \
missing = [r for r, _ in ROOTS.values() if not (pathlib.Path('/app') / r).is_dir()]; \
sys.exit('roots missing in image: ' + ', '.join(missing)) if missing else \
print('registry valid;', len(ROOTS), 'roots present;', \
      len(list(pathlib.Path('/app/docs').rglob('*.md'))), 'markdown files')"

EXPOSE 8000 8084
CMD ["python", "scripts/docstore/service.py"]

"""Supervise Docstore APIs after verifying the complete staged source tree."""
import filecmp
import json
import os
from pathlib import Path
import shutil
import signal
import subprocess
import sys
import time

# Byline: Codex · GPT-6.1-sol · 2026-10-07.


def validate_staged_sources(root: Path, *, require_mount: bool = False) -> dict:
    """Validate the staged registry and seven exact source roots before indexing.

    Inputs: service root and whether production requires a real docs mount. Output:
    registry path and root count. Effects: filesystem reads only. Choose in the
    staged service before starting APIs; legacy additive overlay uses its own path.
    """
    from scope import ROOTS
    from source_registry import load_sources

    root = root.resolve(strict=True)
    docs = root / 'docs'
    if not docs.is_dir() or (require_mount and not docs.is_mount()):
        raise ValueError('Docstore requires the staged /app/docs bind mount')
    quarantine = root / 'to_be_deleted'
    if require_mount and (not quarantine.is_dir() or not quarantine.is_mount()):
        raise ValueError('Docstore requires the durable /app/to_be_deleted bind mount')
    registry = docs / 'docstore-source-registry.json'
    if registry.is_symlink() or not registry.is_file():
        raise ValueError('Docstore staged source registry is missing or linked')
    configured = Path(os.environ.get('DOCSTORE_PROJECT_REGISTRY', str(registry)))
    if configured.resolve(strict=True) != registry.resolve(strict=True):
        raise ValueError('Docstore registry must be the staged docs registry')
    payload = json.loads(registry.read_text(encoding='utf-8'))
    if payload.get('monorepo_root') != str(root):
        raise ValueError(f'Docstore staged registry monorepo_root must be {root}')
    if len(payload.get('projects', [])) != len(ROOTS):
        raise ValueError('Docstore staged registry must contain exactly seven projects')
    sources, _ = load_sources(registry, docs, multi_root_enabled=True)
    if len(sources) != len(ROOTS):
        raise ValueError('Docstore staged registry did not resolve all seven roots')
    for project, (relative, _) in ROOTS.items():
        source = root / relative
        if source.is_symlink() or not source.is_dir() or not source.resolve(strict=True).is_relative_to(docs.resolve(strict=True)):
            raise ValueError(f'Docstore staged source root missing or linked: {project}')
    return {'registry': str(registry), 'roots': len(sources)}


def verify_local_sources(root: Path) -> dict:
    """Require every private addition to have identical bytes in the staged docs bind.

    Inputs: service root and DOCSTORE_LOCAL_SOURCES mount. Output: verified file
    count. Effects: filesystem reads only. Choose for staged sources instead of
    overlay_local_sources, which serves the older additive copy layout.
    """
    source = Path(os.environ.get('DOCSTORE_LOCAL_SOURCES', '/extras'))
    if source.is_symlink() or not source.is_dir():
        raise ValueError('Docstore private source mount is missing or linked')
    docs = (root / 'docs').resolve(strict=True)
    verified = 0
    for item in sorted(source.rglob('*')):
        if item.is_symlink():
            raise ValueError(f'Docstore private source is linked: {item}')
        if item.is_dir() or item.suffix.lower() != '.md':
            continue
        target = docs / item.relative_to(source)
        if not target.resolve(strict=False).is_relative_to(docs) or target.is_symlink() or not target.is_file():
            raise ValueError(f'Docstore staged private source is missing: {item}')
        if not filecmp.cmp(item, target, shallow=False):
            raise ValueError(f'Docstore staged private source differs: {item}')
        verified += 1
    return {'source': str(source), 'verified': verified, 'state': 'verified'}


def overlay_local_sources(root: Path) -> dict:
    """Add private host documents to a writable legacy docs tree.

    Inputs: writable root and DOCSTORE_LOCAL_SOURCES. Output: merge counts.
    Effects: additive file copies only. Choose only for legacy writable layouts;
    staged binds use verify_local_sources before the APIs start.

    Owner ruling 2026-09-28: the session compact summaries are "indexed so that they're
    searchable" but must not "make it to GitHub" -- the same line drawn for dev-resources on
    2026-09-26, where indexing is local and publication is not. Four .gitignore files exclude
    COMPACT-SUMMARY-*.md, so an image built from a git clone cannot carry them. The 14 already in
    the store then read as documents whose sources had vanished, which held every sync at
    `degraded` with cdc_verified false: 872 sources expected against 886 documents observed.

    The mount is read-only and strictly ADDITIVE. A file the image already carries is never
    replaced, so nothing outside git can quietly change a document that came from git. A path
    that escapes docs/ raises rather than being skipped -- that is someone writing into the code
    tree through the mount, not a stray file.
    """
    source = Path(os.environ.get('DOCSTORE_LOCAL_SOURCES', '/extras'))
    report = {'source': str(source), 'added': 0, 'already_in_git': 0, 'skipped_non_markdown': 0}
    if not source.is_dir():
        return {**report, 'state': 'no local source mount'}
    docs = (root / 'docs').resolve()
    for item in sorted(source.rglob('*')):
        if item.is_dir():
            continue
        if item.suffix.lower() != '.md':
            report['skipped_non_markdown'] += 1
            continue
        target = (docs / item.relative_to(source)).resolve()
        if not target.is_relative_to(docs):
            raise ValueError(f'local source escapes the docs tree: {item}')
        if target.exists():
            report['already_in_git'] += 1
            continue
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(item, target)
        report['added'] += 1
    return {**report, 'state': 'merged'}


def main():
    """Start both APIs after validating source and authentication configuration.

    Inputs: environment and staged source mounts. Output: first child exit code.
    Effects: reads sources and starts or stops child processes. Choose as the
    service entry point; the worker owns indexing as a separate operation.
    """
    root=Path(__file__).resolve().parents[2]
    if os.environ.get('DOCSTORE_SOURCES_STAGED') == '1':
        print('docstore: staged sources', validate_staged_sources(root, require_mount=True), flush=True)
        print('docstore: private sources', verify_local_sources(root), flush=True)
    else:
        print('docstore: local sources', overlay_local_sources(root), flush=True)
    if not os.environ.get('DOCSTORE_API_TOKEN') or not os.environ.get('DOCSTORE_CONTROL_TOKEN'):
        raise ValueError('Worker and control tokens are mandatory')
    # 0.8.1-r2 (Claude Code · Opus 5.5, 2026-09-26): DOCSTORE_API_HOST=0.0.0.0 lets the compose publish the
    # worker API on the canonical Docstore port 8072 behind svc:docstore-api (owner 2026-09-10: "make sure
    # there's an API exposed"; 2026-09-12 port families). The default stays loopback.
    api_host=os.environ.get('DOCSTORE_API_HOST','127.0.0.1')
    if api_host not in {'127.0.0.1','0.0.0.0'}:
        raise ValueError('DOCSTORE_API_HOST must be 127.0.0.1 or 0.0.0.0')
    env={**os.environ,'DOCSTORE_API_URL':'http://127.0.0.1:8000','DOCSTORE_MCP_HOST':'0.0.0.0'}
    children=[]
    def stop(*_):
        for child in children:
            if child.poll() is None: child.terminate()
    signal.signal(signal.SIGTERM,stop); signal.signal(signal.SIGINT,stop)
    try:
        children.append(subprocess.Popen([sys.executable,'-m','uvicorn','--app-dir',str(root/'scripts/docstore'),'api:app','--host',api_host,'--port','8000'],env=env))
        children.append(subprocess.Popen([sys.executable,str(root/'plugins/docstore/control/hosted.py')],env=env))
        while all(child.poll() is None for child in children): time.sleep(.5)
        return next((child.returncode for child in children if child.returncode is not None),1)
    finally:
        stop()
        for child in children:
            try: child.wait(timeout=15)
            except subprocess.TimeoutExpired:
                child.kill(); child.wait(timeout=10)


if __name__=='__main__':
    raise SystemExit(main())

"""Single service supervisor: authenticated MCP plus loopback worker API, no startup ingest."""
import os
from pathlib import Path
import shutil
import signal
import subprocess
import sys
import time


def overlay_local_sources(root: Path) -> dict:
    """Add documents that exist on the host but are deliberately not in git.

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
    root=Path(__file__).resolve().parents[2]
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

"""Single service supervisor: authenticated MCP plus loopback worker API, no startup ingest."""
import os
from pathlib import Path
import signal
import subprocess
import sys
import time


def main():
    root=Path(__file__).resolve().parents[2]
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

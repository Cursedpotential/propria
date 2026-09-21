"""Start a governed SELECTED Docstore re-index for the documents whose enrichment failed.

Byline: Claude Code · Fable 5.1 · 2026-09-20

Runs INSIDE the docstore-control container (it holds DOCSTORE_API_URL and
DOCSTORE_API_TOKEN; neither is printed):

    docker exec -i <docstore-control> python - < scripts/docstore/reindex_selected.py

With no arguments it reads the worker's /health, takes the failed enrichment
paths of the latest sync, and posts the same payload the control server's
`docstore_index_selected` tool posts. It prints the run id and nothing secret.
"""

from __future__ import annotations

import json
import os
import sys
import urllib.error
import urllib.request

API = os.environ["DOCSTORE_API_URL"].rstrip("/")
HEADERS = {"Authorization": "Bearer " + os.environ["DOCSTORE_API_TOKEN"], "Content-Type": "application/json"}


def call(method: str, path: str, payload: dict | None = None) -> dict:
    data = json.dumps(payload).encode() if payload is not None else None
    request = urllib.request.Request(API + path, data=data, method=method, headers=HEADERS)
    try:
        with urllib.request.urlopen(request, timeout=60) as response:
            return json.load(response)
    except urllib.error.HTTPError as error:
        raise SystemExit(f"{method} {path}: HTTP {error.code} {error.read().decode()[:300]}") from None


def main() -> None:
    sync = call("GET", "/health").get("startup_or_latest_sync", {})
    failed = [item["source_path"] for item in sync.get("enrichment", {}).get("failed_documents", [])]
    paths = sys.argv[1:] or failed
    if not paths:
        print("nothing to re-index: the latest sync has no failed enrichment")
        return
    print("re-indexing", len(paths), "paths:", ", ".join(paths))
    result = call("POST", "/runs", {"scope": "selected", "paths": paths, "full_reprocess": False, "index_kind": "docs"})
    print(
        json.dumps(
            {key: result.get(key) for key in ("run_id", "status", "state", "accepted", "detail") if key in result}
        )
    )


if __name__ == "__main__":
    main()

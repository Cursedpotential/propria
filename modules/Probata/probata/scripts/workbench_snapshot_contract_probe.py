"""Print WHY the Workbench API rejects a Proffer preview SNAPSHOT.

Byline: Claude Code · Fable 5.1 · 2026-09-21

Sibling of workbench_preview_contract_probe.py (which covers the messages and
content pages). Runs INSIDE the workbench container, read-only:

    docker exec -i -w /app <workbench> python - <handle> < scripts/workbench_snapshot_contract_probe.py
"""

from __future__ import annotations

import asyncio
import json
import sys

from pydantic import ValidationError

from app.service import proffer as service
from app.types.proffer import ProfferPreviewResponse


async def main(handle: str) -> None:
    response = await service._request("GET", f"/reference-import/previews/{handle}")
    payload = response.json()
    print("status", response.status_code, "keys", sorted(payload)[:40])
    for mode in ("TEST", "REAL"):
        candidate = dict(payload)
        candidate.setdefault("matter_mode", mode)
        try:
            ProfferPreviewResponse.model_validate(candidate)
            print(mode, "-> valid")
            return
        except ValidationError as error:
            for item in error.errors()[:12]:
                print(mode, ".".join(str(part) for part in item["loc"]), "|", item["type"], "|", json.dumps(item.get("input"))[:120])


asyncio.run(main(sys.argv[1]))

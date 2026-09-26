"""Print WHY the Workbench API rejects a Proffer preview page.

Byline: Claude Code · Fable 5.1 · 2026-09-20

The BFF answers "Proffer starter returned an invalid preview ... page" and drops
the pydantic error. This runs INSIDE the workbench container (it has the app,
the starter URL and the tailnet route), fetches the same two pages the Review
surface needs and prints the validation errors and the offending values:

    docker exec -i <workbench> python - <handle> < scripts/workbench_preview_contract_probe.py

Read-only. Prints field paths, error types and at most 120 characters of a value.
"""

from __future__ import annotations

import asyncio
import json
import sys

from pydantic import ValidationError

from app.service import proffer as service
from app.types.proffer import ProfferPreviewMessagesResponse

try:
    from app.types.proffer_content import ProfferContentResponse
except ImportError:  # the model has moved between modules before
    from app.types.proffer import ProfferContentResponse  # type: ignore[attr-defined,no-redef]


async def main(handle: str) -> None:
    for label, path, model in (
        ("messages", f"/reference-import/previews/{handle}/messages", ProfferPreviewMessagesResponse),
        ("content", f"/reference-import/previews/{handle}/content", ProfferContentResponse),
    ):
        response = await service._request("GET", path, params={"limit": 3})
        payload = response.json()
        print(f"== {label}: HTTP {response.status_code}; top-level keys: {sorted(payload)[:30]}")
        payload.setdefault("matter_mode", "TEST")
        try:
            model.model_validate(payload)
            print("   validates")
        except ValidationError as error:
            for item in error.errors()[:12]:
                value = json.dumps(item.get("input"), default=str)[:120]
                print("  ", ".".join(str(part) for part in item["loc"]), "|", item["type"], "|", item["msg"], "|", value)


asyncio.run(main(sys.argv[1]))

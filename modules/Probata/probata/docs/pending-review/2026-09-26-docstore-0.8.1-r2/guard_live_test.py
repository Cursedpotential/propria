"""Live negative test of the server-side retraction guard (0.8.1-r3). Changes nothing.

Byline: Claude Code · Opus 5.5 · 2026-09-26

Builds the upload manifest from a Propria-shaped root (argv[1]; use a scratch copy with one document
left out), then through the hosted ctl:
  1. docstore_source_plan must list that document in retracted_sources with its mirror hash;
  2. docstore_source_apply WITHOUT naming it in `retract` must be refused (nothing is written);
  3. docstore_source_read must return the exact mirror copy, whose sha256 equals the plan's hash.
Uses the installed plugin client's own code (sync_sources, manifest, decode_markdown, call) and its
hosted connection. Run with the plugin's pinned requirements:

    uv run --no-project --with-requirements <plugin>/requirements.txt python guard_live_test.py <root> <plugin_dir>
"""
import asyncio
import hashlib
import json
import sys
from pathlib import Path

ROOT = Path(sys.argv[1]).resolve(strict=True)
sys.path.insert(0, sys.argv[2])
import client  # noqa: E402  (the installed propria-docstore client, 0.8.2-sync-r3)


async def main() -> None:
    async with client.connection() as ctl:
        projections = await client.call(ctl, "docstore_adr", {"action": "projections"})
        projected = {p["path"].removeprefix("docs/"): p for p in projections.get("projections", [])}
        entries = {}
        for project, folder, included, excluded in client.sync_sources(ROOT):
            directory = (ROOT / folder).resolve(strict=True)
            for file in sorted(directory.rglob("*.md")):
                relative = file.relative_to(directory)
                rel = relative.as_posix()
                if (any(p.startswith(".") or p.lower() in {"private", "to_be_deleted", "_to_be_deleted"} for p in relative.parts)
                        or (project == "propria" and relative.parts[0].lower() in client.PROPRIA_ALIASES)
                        or not any(r.match(rel) for r in included) or any(r.match(rel) for r in excluded)
                        or file.stat().st_size > client.MAX_DOCUMENT_BYTES):
                    continue
                entries[project + "/" + rel] = client.decode_markdown(file.read_bytes())
            if project == "probata":
                entries.update({"probata/" + p: v["content"] for p, v in projected.items()})
        files, _ = client.manifest(entries)
        plan = await client.call(ctl, "docstore_source_plan", {"files": files})
        missing = plan["retracted_sources"]
        result = {"retracted_sources": missing, "retracted_hashes": plan.get("retracted_hashes")}
        try:
            await client.call(ctl, "docstore_source_apply", {"files": files, "plan_id": plan["plan_id"]})
            result["unnamed_apply"] = "ACCEPTED (guard failed)"
        except Exception as exc:  # the hosted guard refuses; the ctl reports the API's HTTP 409
            result["unnamed_apply"] = "refused: " + str(exc)[:160]
        copies = await client.call(ctl, "docstore_source_read", {"paths": missing})
        result["read"] = [{"key": item["key"], "sha256": item["sha256"],
                           "content_sha256_ok": hashlib.sha256(item["content"].encode()).hexdigest() == item["sha256"],
                           "matches_plan": item["sha256"] == plan.get("retracted_hashes", {}).get(item["key"])}
                          for item in copies.get("files", [])]
        print(json.dumps(result, indent=1))


asyncio.run(main())

"""Run a bounded, content-redacted mixed-format Probata demonstration."""

from __future__ import annotations

import argparse
import asyncio
import hashlib
import json
from pathlib import Path
from typing import Any

from server.analysis.context_chat_ingest import ingest_chat_file
from server.tools.parsers.messaging.imessage_txt import parse as parse_imessage
from server.tools.parsers.messaging.sms_xml import parse as parse_sms
from server.tools.registry import load_builtin_tools, registry


def _sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def _parser_result(
    path: Path,
    source_sha256: str,
    parser_id: str,
    result: dict[str, Any],
    *,
    lane: str,
) -> dict[str, Any]:
    records = result["records"]
    return {
        "file": path.name,
        "sha256": source_sha256,
        "bytes": path.stat().st_size,
        "lane": lane,
        "parser_id": parser_id,
        "record_count": len(records),
        "record_types": sorted({str(record["record_type"]) for record in records}),
        "roles": sorted({str(record["role"]) for record in records}),
        "content_emitted": False,
        "status": "parsed",
    }


async def run(root: Path) -> dict[str, Any]:
    load_builtin_tools()
    imessage = root / "imessage-thread.txt"
    sms = root / "sms-backup.xml"
    claude = root / "claude-conversations.json"
    chatgpt = root / "chatgpt-conversations.json"
    paths = (imessage, sms, claude, chatgpt)

    # H1 is intentionally a separate, pre-parse activity boundary.
    source_hashes = {path: _sha256(path) for path in paths}

    results = [
        _parser_result(
            imessage,
            source_hashes[imessage],
            "messages.imessage-txt",
            parse_imessage({"path": str(imessage)}),
            lane="evidence-parser-proof",
        ),
        _parser_result(
            sms,
            source_hashes[sms],
            "messages.sms-xml",
            parse_sms({"path": str(sms)}),
            lane="evidence-parser-proof",
        ),
        _parser_result(
            chatgpt,
            source_hashes[chatgpt],
            "transcripts.chatgpt-official",
            registry.get("transcripts.chatgpt-official").run({"path": str(chatgpt)}),
            lane="context-chat-parser-proof",
        ),
    ]
    claude_report = await ingest_chat_file(claude, dry_run=True, engine="python", format="claude-ai-export")
    results.append(
        {
            "file": claude.name,
            "sha256": source_hashes[claude],
            "bytes": claude.stat().st_size,
            "lane": "context-chat-dry-run",
            "parser_id": claude_report.parser_id,
            "record_count": claude_report.record_count,
            "conversation_count": len(claude_report.conversation_ids),
            "classified_chunk_assignments": sum(claude_report.lane_counts.values()),
            "lane_counts": claude_report.lane_counts,
            "records_stored": claude_report.records_stored,
            "content_emitted": False,
            "status": "completed" if claude_report.dry_run else "unexpected-live-run",
        }
    )
    return {
        "schema": "probata-mixed-format-demo-v1",
        "fixture_root": str(root.resolve()),
        "test_data_only": True,
        "database_writes": 0,
        "object_store_writes": 0,
        "source_mutations": 0,
        "results": results,
    }


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("fixture_root", type=Path)
    args = parser.parse_args()
    print(json.dumps(asyncio.run(run(args.fixture_root)), indent=2, sort_keys=True))

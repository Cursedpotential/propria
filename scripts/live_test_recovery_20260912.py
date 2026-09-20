"""One-time TEST-only recovery for the 2026-09-12 live mixed-format proof.

Runs inside the live Platform API container. It does not change deployed code:
message evidence stores canonical records with no retired Python chunks, while
AI exports use the independent context-chat store required by D-082.
"""

from __future__ import annotations

import asyncio
import json
from pathlib import Path

from server.analysis.context_chat_ingest import ingest_chat_file
from server.contracts.ingest import IngestRequest
from server.contracts.records import NormalizedRecord
from server.evidence.custody import ArtifactRef
from server.evidence.store import records_exist_for_artifact, store_record_batch
from server.proffer.service import _enrich
from server.tools.parsers.messaging.imessage_txt import parse as parse_imessage
from server.tools.parsers.messaging.sms_xml import parse as parse_sms


TEST_MATTER = "deadbeef-dead-beef-dead-beefdeadbeef"
TEST_RUN = "probata-mixed-live-20260912-01"


def identity(name: str, format_id: str) -> dict[str, object]:
    return {
        "test_only": True,
        "label": "TEST",
        "dataset": "synthetic",
        "source": "portal-test",
        "test_run_id": TEST_RUN,
        "original_name": name,
        "sample_format": format_id,
        "recovery_contract": "record-only-H-04-D-116",
    }


message_inputs = [
    (
        "/data/ingest-staging/460e6f24-d2f4-4e5a-bf5c-c0d0a5589169-sms-backup.xml",
        "sms-backup.xml",
        "messages.sms-xml",
        "01a096be-9787-74d4-ae54-5544d41633d9",
        "3a3290be5d26e4928d11514300b8999a8b9a24aadd0d53c1d2f81ec9f644a88d",
        parse_sms,
    ),
    (
        "/data/ingest-staging/dcba562b-8932-409f-bfa4-ebb254b33dde-imessage-thread.txt",
        "imessage-thread.txt",
        "messages.imessage-txt",
        "01a096c0-88f4-7ac9-bead-4f5e46af38fd",
        "fbb8db88fa09d44639370b30f3f3937d405c93db06151cd95d7d26c4aee4943c",
        parse_imessage,
    ),
]

results: list[dict[str, object]] = []
for staged_path, original_name, parser_id, artifact_id, source_sha256, parser in message_inputs:
    request = IngestRequest(
        staged_path=staged_path,
        source_identity=identity(original_name, parser_id),
        message_corpus="first_party",
        source_principal="TEST_SYNTHETIC_OWNER",
        caller_owns_conversation=True,
        matter_id=TEST_MATTER,
        engine="python",
        allow_fallback=False,
        custody_tier="light",
    )
    artifact = ArtifactRef(
        artifact_id=artifact_id,
        sha256=source_sha256,
        source_ref=staged_path,
        blob_key="",
        size_bytes=Path(staged_path).stat().st_size,
        duplicate=True,
        ingested_at="2026-09-12T17:51:08Z",
        custody_tier="light",
    )
    already_stored = records_exist_for_artifact(artifact_id)
    if already_stored:
        stored = 0
    else:
        parsed = parser({"path": staged_path})["records"]
        records = [
            record if isinstance(record, NormalizedRecord) else NormalizedRecord.model_validate(record)
            for record in parsed
        ]
        records = [
            record.model_copy(update={"sender": record.role})
            if record.record_type.value == "message" and not record.sender and record.role
            else record
            for record in records
        ]
        records = _enrich(records, request, Path(staged_path), parser_id)
        stored = store_record_batch(
            records,
            [],
            artifact,
            case_id=TEST_MATTER,
            domain="context",
            projection_request=request,
            retry=False,
        )
    results.append(
        {
            "name": original_name,
            "writer": "canonical-record-store",
            "artifact_id": artifact_id,
            "status": "already-stored" if already_stored else "completed",
            "records_stored": stored,
            "chunks_stored": 0,
            "parser_id": parser_id,
        }
    )


async def store_ai_context() -> None:
    inputs = [
        (
            "/data/ingest-staging/7ae32b17-c59b-4100-8779-0342cdfab5e4-chatgpt-conversations.json",
            "chatgpt-conversations.json",
            "chatgpt-official",
            "auto",
        ),
        (
            "/data/ingest-staging/01197416-9f67-4924-a812-32c2be0e20c4-claude-conversations.json",
            "claude-conversations.json",
            "claude-ai-export",
            "python",
        ),
    ]
    for staged_path, original_name, format_id, engine in inputs:
        report = await ingest_chat_file(
            Path(staged_path),
            source_meta={**identity(original_name, format_id), "matter_id": TEST_MATTER},
            dry_run=False,
            project=False,
            engine=engine,
            format=format_id,
            classify=False,
        )
        results.append(
            {
                "name": original_name,
                "writer": "context-chat-store",
                "status": "completed",
                "records_stored": report.records_stored,
                "conversation_count": len(report.conversation_ids),
                "parser_id": report.parser_id,
                "lane_counts": report.lane_counts,
            }
        )


asyncio.run(store_ai_context())
print(json.dumps({"test_run_id": TEST_RUN, "results": results}, indent=2, sort_keys=True))

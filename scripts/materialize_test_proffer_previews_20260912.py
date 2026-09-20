"""Materialize three synthetic TEST previews after the live workflow stalled.

This one-time recovery writes only context.proffer_* preview/generation tables.
It never approves, promotes, or writes evidence. Every preview remains awaiting
human decision and carries an explicit manual-recovery reason.
"""

from __future__ import annotations

import hashlib
import json
from pathlib import Path
import uuid

from sqlalchemy import text

from server.contracts.records import NormalizedRecord
from server.evidence.store import _get_engine
from server.tools.parsers.messaging.imessage_txt import parse as parse_imessage
from server.tools.parsers.messaging.sms_xml import parse as parse_sms
from server.tools.registry import load_builtin_tools, registry


RECOVERY = "TEST ONLY - manual preview materialization after repair.detect 422; not approved or evidence"

SAMPLES = (
    {
        "request_id": "TEST-PROBATA-20260912-SMS-001",
        "path": "/data/ingest-staging/460e6f24-d2f4-4e5a-bf5c-c0d0a5589169-sms-backup.xml",
        "format_id": "smsbackuprestore_xml",
        "parser_id": "messages.sms-xml",
        "parse": parse_sms,
    },
    {
        "request_id": "TEST-PROBATA-20260912-IMESSAGE-001",
        "path": "/data/ingest-staging/dcba562b-8932-409f-bfa4-ebb254b33dde-imessage-thread.txt",
        "format_id": "imessage_txt",
        "parser_id": "messages.imessage-txt",
        "parse": parse_imessage,
    },
    {
        "request_id": "TEST-PROBATA-20260912-CHATGPT-001",
        "path": "/data/ingest-staging/7ae32b17-c59b-4100-8779-0342cdfab5e4-chatgpt-conversations.json",
        "format_id": "chatgpt_official_json",
        "parser_id": "transcripts.chatgpt-official",
        "parse": None,
    },
)


def parsed_records(sample: dict) -> list[NormalizedRecord]:
    parser = sample["parse"]
    if parser is None:
        load_builtin_tools()
        result = registry.get(sample["parser_id"]).run({"path": sample["path"]})
    else:
        result = parser({"path": sample["path"]})
    return [
        item if isinstance(item, NormalizedRecord) else NormalizedRecord.model_validate(item)
        for item in result["records"]
    ]


def participant_rows(records: list[NormalizedRecord]) -> list[dict[str, str | None]]:
    names: set[str] = set()
    for record in records:
        names.update(value.strip() for value in record.participants if value and value.strip())
        if record.role and record.role.strip():
            names.add(record.role.strip())
    return [
        {"participant_id": f"test-participant-{index}", "display_name": name, "canonical_address": None}
        for index, name in enumerate(sorted(names), start=1)
    ]


def materialize(sample: dict) -> dict[str, object]:
    records = parsed_records(sample)
    participants = participant_rows(records)
    participant_by_name = {row["display_name"]: row["participant_id"] for row in participants}
    messages = []
    for ordinal, record in enumerate(records):
        participant_ids = [participant_by_name[name] for name in record.participants if name in participant_by_name]
        sender = participant_by_name.get(record.role or "")
        messages.append(
            {
                "message_id": f"test-message-{ordinal:04d}",
                "ordinal": ordinal,
                "sent_at": record.occurred_at,
                "sender_participant_id": sender,
                "body": record.content,
                "participant_ids": participant_ids,
                "source_locator_ref": f"{sample['request_id']}#record={ordinal}",
            }
        )
    digest = hashlib.sha256(
        json.dumps(messages, default=str, sort_keys=True, separators=(",", ":")).encode()
    ).digest()
    config_digest = hashlib.sha256(f"{sample['parser_id']}|{RECOVERY}".encode()).digest()

    with _get_engine().begin() as connection:
        # Use the schema-owned registration path so raw_generation retains its
        # registry FK and each TEST format gets the required append-only subtype.
        connection.execute(
            text("SELECT context.register_raw_format_subtype(:format_id)"),
            {"format_id": sample["format_id"]},
        )
        binding = connection.execute(
            text(
                "SELECT b.preview_handle, sv.id AS source_version_id "
                "FROM context.proffer_preview_binding b "
                "JOIN context.source_version sv ON sv.workflow_id=b.workflow_id "
                "WHERE b.request_id=:request_id"
            ),
            {"request_id": sample["request_id"]},
        ).mappings().one()
        handle = binding["preview_handle"]
        existing = connection.execute(
            text("SELECT count(*) FROM context.proffer_preview_snapshot WHERE preview_handle=:handle"),
            {"handle": handle},
        ).scalar_one()
        if existing:
            return {"request_id": sample["request_id"], "preview_handle": handle, "status": "already-materialized"}

        raw_id, normalized_id = str(uuid.uuid4()), str(uuid.uuid4())
        connection.execute(
            text(
                "INSERT INTO context.raw_generation "
                "(id,source_version_id,generation_ordinal,format_id,parser_id,parser_version,status,sealed_at,sealed_by) "
                "VALUES (CAST(:id AS uuid),CAST(:source AS uuid),1,:format,:parser,:version,'sealed',now(),:sealed_by)"
            ),
            {
                "id": raw_id,
                "source": str(binding["source_version_id"]),
                "format": sample["format_id"],
                "parser": sample["parser_id"],
                "version": "TEST-manual-preview-recovery-v1",
                "sealed_by": "codex-test-recovery",
            },
        )
        connection.execute(
            text(
                "INSERT INTO context.normalized_generation "
                "(id,source_version_id,raw_generation_id,generation_ordinal,normalizer_id,normalizer_version,"
                "status,sealed_at,sealed_by) VALUES (CAST(:id AS uuid),CAST(:source AS uuid),CAST(:raw AS uuid),1,"
                "'probata-test-preview-normalizer','1','sealed',now(),'codex-test-recovery')"
            ),
            {"id": normalized_id, "source": str(binding["source_version_id"]), "raw": raw_id},
        )
        connection.execute(
            text(
                "INSERT INTO context.proffer_preview_snapshot "
                "(preview_handle,snapshot_seq,phase,source_version_id,raw_generation_id,normalized_generation_id,"
                "parser_id,parser_version,parser_config_digest,preview_digest,reason) VALUES "
                "(:handle,1,'awaiting_decision',CAST(:source AS uuid),CAST(:raw AS uuid),CAST(:normalized AS uuid),"
                ":parser,:version,:config_digest,:preview_digest,:reason)"
            ),
            {
                "handle": handle,
                "source": str(binding["source_version_id"]),
                "raw": raw_id,
                "normalized": normalized_id,
                "parser": sample["parser_id"],
                "version": "TEST-manual-preview-recovery-v1",
                "config_digest": config_digest,
                "preview_digest": digest,
                "reason": RECOVERY,
            },
        )
        for row in participants:
            connection.execute(
                text(
                    "INSERT INTO context.proffer_preview_participant "
                    "(preview_handle,snapshot_seq,participant_id,display_name,canonical_address) "
                    "VALUES (:handle,1,:participant_id,:display_name,:canonical_address)"
                ),
                {"handle": handle, **row},
            )
        for row in messages:
            connection.execute(
                text(
                    "INSERT INTO context.proffer_preview_message "
                    "(preview_handle,snapshot_seq,message_id,ordinal,sent_at,sender_participant_id,body,"
                    "participant_ids,source_locator_ref) VALUES (:handle,1,:message_id,:ordinal,:sent_at,"
                    ":sender_participant_id,:body,:participant_ids,:source_locator_ref)"
                ),
                {"handle": handle, **row},
            )
        for receipt_type in ("parser_selection", "parser_execution", "normalization"):
            connection.execute(
                text(
                    "INSERT INTO context.proffer_preview_receipt "
                    "(preview_handle,snapshot_seq,receipt_type,receipt_ref,status,digest,recorded_at) "
                    "VALUES (:handle,1,:type,:ref,'completed',:digest,now())"
                ),
                {
                    "handle": handle,
                    "type": receipt_type,
                    "ref": f"TEST-manual-recovery:{sample['request_id']}:{receipt_type}",
                    "digest": digest,
                },
            )
        next_event = connection.execute(
            text("SELECT COALESCE(max(event_id)+1,0) FROM context.proffer_preview_event WHERE preview_handle=:h"),
            {"h": handle},
        ).scalar_one()
        for offset, event_type in enumerate(("messages_available", "decision_requested")):
            connection.execute(
                text(
                    "INSERT INTO context.proffer_preview_event "
                    "(preview_handle,event_id,event_type,occurred_at,phase,message_count,detail) "
                    "VALUES (:handle,:event_id,:event_type,now(),'awaiting_decision',:count,:detail)"
                ),
                {
                    "handle": handle,
                    "event_id": next_event + offset,
                    "event_type": event_type,
                    "count": len(messages),
                    "detail": RECOVERY,
                },
            )
    return {
        "request_id": sample["request_id"],
        "preview_handle": handle,
        "status": "materialized-awaiting-decision",
        "message_count": len(messages),
        "participant_count": len(participants),
    }


print(json.dumps([materialize(sample) for sample in SAMPLES], indent=2, sort_keys=True))

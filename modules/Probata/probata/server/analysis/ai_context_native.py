"""Extract native AI text fields with bounded in-memory DuckDB projections.

Inputs: original UTF-8 JSON and a native format. Outputs: literal fields and
JSON pointers; effects: one ephemeral DuckDB connection, no PostgreSQL or files.
Pick for the context decode Activity. Projection CTEs reuse the source-native
semantics of engine/postgres/elt_templates/chatgpt_json_array_v2.sql and
claude_ai_export_json_v1.sql; original bytes remain the byte-identity authority.
"""
from __future__ import annotations

import json
from typing import Any


class NativeDecodeInvalid(ValueError):
    """Reject an unsupported or ambiguous native shape without parser fallback.

    Inputs: safe reason. Output: permanent decode error; effects: none. Pick
    instead of silently changing decoder or selecting a guessed conversation.
    """


_CONVERSATIONS = """
WITH document AS (SELECT ?::JSON AS value), conversations AS (
 SELECT try_cast(entry.key AS BIGINT) AS ci, entry.value AS conversation,
 CASE WHEN json_type(document.value)='ARRAY' THEN '/' || entry.key
      WHEN json_type(json_extract(document.value,'$.conversations'))='ARRAY'
      THEN '/conversations/' || entry.key ELSE '' END AS pointer
 FROM document,json_each(CASE WHEN json_type(document.value)='ARRAY' THEN document.value
 WHEN json_type(json_extract(document.value,'$.conversations'))='ARRAY'
 THEN json_extract(document.value,'$.conversations')
 WHEN json_type(document.value)='OBJECT' THEN json_array(document.value)
 ELSE error('native source requires conversations') END) entry
)
"""

_CHATGPT = _CONVERSATIONS + """
SELECT ci,pointer,
 coalesce(json_extract_string(conversation,'$.conversation_id'),
          json_extract_string(conversation,'$.id'),json_extract_string(conversation,'$.uuid')) AS conversation_id,
 coalesce(json_extract_string(conversation,'$.title'),json_extract_string(conversation,'$.name')) AS title,
 json_extract_string(conversation,'$.current_node') AS current_node,
 node.key AS node_key,json_extract_string(node.value,'$.parent') AS parent,
 json_type(node.value) AS node_type,
 json_extract_string(node.value,'$.message.author.role') AS role,
 json_extract(node.value,'$.message.create_time')::VARCHAR AS native_time,
 try_cast(part.key AS BIGINT) AS part_index,
 CASE WHEN json_type(part.value)='VARCHAR' THEN json_extract_string(part.value,'$')
      WHEN json_type(json_extract(part.value,'$.text'))='VARCHAR'
      THEN json_extract_string(part.value,'$.text') END AS body,
 CASE WHEN json_type(part.value)='OBJECT' THEN '/text' ELSE '' END AS part_suffix
FROM conversations
LEFT JOIN LATERAL json_each(CASE WHEN json_type(json_extract(conversation,'$.mapping'))='OBJECT'
 THEN json_extract(conversation,'$.mapping') ELSE error('ChatGPT requires mapping') END) node ON true
LEFT JOIN LATERAL json_each(CASE WHEN json_type(json_extract(node.value,'$.message.content.parts'))='ARRAY'
 THEN json_extract(node.value,'$.message.content.parts') ELSE json_array() END) part ON true
ORDER BY ci,node_key,part_index
"""

_CLAUDE = _CONVERSATIONS + """
SELECT ci,pointer,
 coalesce(json_extract_string(conversation,'$.conversation_id'),json_extract_string(conversation,'$.id'),
          json_extract_string(conversation,'$.uuid')) AS conversation_id,
 coalesce(json_extract_string(conversation,'$.title'),json_extract_string(conversation,'$.name')) AS title,
 try_cast(message.key AS BIGINT) AS message_index,
 coalesce(json_extract_string(message.value,'$.uuid'),json_extract_string(message.value,'$.id')) AS message_id,
 json_extract_string(message.value,'$.sender') AS role,
 json_extract(message.value,'$.created_at')::VARCHAR AS native_time,
 part.key AS part_index,
 CASE WHEN json_type(json_extract(message.value,'$.text'))='VARCHAR'
           AND json_extract_string(message.value,'$.text')<>'' THEN '/text'
      ELSE '/content/' || part.key || '/text' END AS field_pointer,
 CASE WHEN json_type(json_extract(message.value,'$.text'))='VARCHAR'
           AND json_extract_string(message.value,'$.text')<>'' THEN json_extract_string(message.value,'$.text')
      WHEN json_extract_string(part.value,'$.type')='text'
           AND json_type(json_extract(part.value,'$.text'))='VARCHAR'
      THEN json_extract_string(part.value,'$.text') END AS body
FROM conversations
LEFT JOIN LATERAL json_each(CASE WHEN json_type(json_extract(conversation,'$.chat_messages'))='ARRAY'
 THEN json_extract(conversation,'$.chat_messages') ELSE error('Claude requires chat_messages') END) message ON true
LEFT JOIN LATERAL json_each(CASE WHEN json_type(json_extract(message.value,'$.text'))='VARCHAR'
 AND json_extract_string(message.value,'$.text')<>'' THEN json_array(NULL)
 WHEN json_type(json_extract(message.value,'$.content'))='ARRAY'
 THEN json_extract(message.value,'$.content') ELSE json_array() END) part ON true
ORDER BY ci,message_index,try_cast(part_index AS BIGINT)
"""


def _pointer(value: str) -> str:
    """Escape a literal native key for a JSON pointer.

    Inputs: mapping key. Output: escaped segment; effects: none. Pick before
    appending DuckDB-projected mapping identifiers to source coordinates.
    """
    return value.replace("~", "~0").replace("/", "~1")


def project_records(raw: bytes, source_format: str, maximum: int) -> list[dict[str, Any]]:
    """Project native JSON fields through DuckDB and order the explicit active path.

    Inputs: bounded original bytes, ChatGPT/Claude format and record ceiling.
    Output: exact text-field records; effects: ephemeral DuckDB only. Pick before
    chunking; malformed SQL/source fails closed without a Python JSON fallback.
    """
    import duckdb
    if source_format not in {"chatgpt", "claude"}:
        raise NativeDecodeInvalid("native JSON format is unsupported")
    if len(raw) > 32 << 20 or not 0 < maximum <= 1024:
        raise NativeDecodeInvalid("native JSON exceeds decode bounds")
    try:
        text = raw.decode("utf-8-sig")
        with duckdb.connect(":memory:", config={"threads": 2, "memory_limit": "256MB",
                                               "enable_external_access": "false"}) as connection:
            query = connection.execute(_CHATGPT if source_format == "chatgpt" else _CLAUDE, [text])
            names = [column[0] for column in query.description]
            native = [dict(zip(names, row)) for row in query.fetchmany(32769)]
            if len(native) > 32768:
                raise NativeDecodeInvalid("native projection exceeds field bound")
    except NativeDecodeInvalid:
        raise
    except (UnicodeError, duckdb.Error):
        raise NativeDecodeInvalid("native DuckDB projection rejected source shape") from None
    records = []
    grouped: dict[int, list[dict[str, Any]]] = {}
    for row in native:
        grouped.setdefault(row["ci"], []).append(row)
    for ci, rows in grouped.items():
        ordered: list[tuple[int, dict[str, Any]]] = []
        if source_format == "chatgpt":
            nodes: dict[str, list[dict[str, Any]]] = {}
            for row in rows:
                if row["node_key"] is None:
                    raise NativeDecodeInvalid("ChatGPT active path is incomplete")
                nodes.setdefault(row["node_key"], []).append(row)
            current = rows[0]["current_node"]
            if current is not None and current not in nodes:
                raise NativeDecodeInvalid("ChatGPT explicit active path is incomplete")
            if current not in nodes:
                parents = {items[0]["parent"] for items in nodes.values()}
                leaves = [key for key in nodes if key not in parents]
                if len(leaves) != 1:
                    raise NativeDecodeInvalid("ChatGPT active path is ambiguous")
                current = leaves[0]
            selected, seen = [], set()
            while current is not None:
                if current in seen or current not in nodes or nodes[current][0]["node_type"] != "OBJECT":
                    raise NativeDecodeInvalid("ChatGPT active path is incomplete")
                seen.add(current)
                selected.append(nodes[current])
                current = nodes[current][0]["parent"]
            for mi, items in enumerate(reversed(selected)):
                ordered.extend((mi, row) for row in items)
        else:
            ordered = [(row["message_index"], row) for row in rows]
        for mi, row in ordered:
            body = row["body"]
            if not body:
                continue
            if source_format == "chatgpt":
                pointer = f"{row['pointer']}/mapping/{_pointer(row['node_key'])}/message/content/parts/{row['part_index']}{row['part_suffix']}"
                message_id = row["node_key"]
            else:
                pointer = f"{row['pointer']}/chat_messages/{mi}{row['field_pointer']}"
                message_id = row["message_id"]
            records.append({"conversation_index": ci, "conversation_id": row["conversation_id"],
                            "conversation_title": row["title"], "record_id": pointer,
                            "native_message_index": mi, "native_message_id": message_id,
                            "role": row["role"], "native_time": json.loads(row["native_time"]) if row["native_time"] else None,
                            "native_json_pointer": pointer, "body": body,
                            "source_span": {"start": 0, "end": len(body), "unit": "unicode_codepoint"}})
            if len(records) > maximum:
                raise NativeDecodeInvalid("native text fields exceed record bound")
    if not records:
        raise NativeDecodeInvalid("native export contains no supported text fields")
    return records

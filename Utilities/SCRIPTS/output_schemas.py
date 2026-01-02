#!/usr/bin/env python3
"""
Output Schema Definitions and Validators

Defines the expected structure for parser output files
and provides validation functions.
"""

from typing import Dict, Any, List, Optional
from dataclasses import dataclass
from datetime import datetime
import json
import re


@dataclass
class SchemaValidationError:
    """Validation error details"""
    field: str
    message: str
    value: Any


class OutputValidator:
    """Validates parser output against expected schemas"""

    @staticmethod
    def validate_conversation_turn(data: Dict[str, Any]) -> List[SchemaValidationError]:
        """
        Validate a conversation turn record.

        Expected schema:
        {
            "message_hash": str (64-char hex),
            "conversation_id": str,
            "platform": str ("chatgpt"|"claude"|"gemini"|...),
            "timestamp": str (ISO-8601),
            "turn_type": str ("user"|"assistant"),
            "content": str,
            "raw_metadata": dict
        }
        """
        errors = []

        # Required fields
        required = ["message_hash", "conversation_id", "platform", "timestamp", "turn_type", "content", "raw_metadata"]
        for field in required:
            if field not in data:
                errors.append(SchemaValidationError(field, "Missing required field", None))

        # Validate message_hash (SHA256 = 64 hex chars)
        if "message_hash" in data:
            hash_val = data["message_hash"]
            if not isinstance(hash_val, str) or len(hash_val) != 64 or not re.match(r'^[a-f0-9]{64}$', hash_val):
                errors.append(SchemaValidationError("message_hash", "Must be 64-char SHA256 hex string", hash_val))

        # Validate platform
        if "platform" in data:
            platform = data["platform"]
            valid_platforms = ["chatgpt", "claude", "gemini", "perplexity", "qwen", "other"]
            if platform not in valid_platforms:
                errors.append(SchemaValidationError("platform", f"Must be one of {valid_platforms}", platform))

        # Validate timestamp (ISO-8601)
        if "timestamp" in data:
            timestamp = data["timestamp"]
            if not isinstance(timestamp, str):
                errors.append(SchemaValidationError("timestamp", "Must be string", type(timestamp)))
            else:
                # Try parsing ISO-8601
                try:
                    datetime.fromisoformat(timestamp.replace('Z', '+00:00'))
                except ValueError:
                    errors.append(SchemaValidationError("timestamp", "Must be valid ISO-8601 format", timestamp))

        # Validate turn_type
        if "turn_type" in data:
            turn_type = data["turn_type"]
            valid_types = ["user", "assistant", "system", "unknown"]
            if turn_type not in valid_types:
                errors.append(SchemaValidationError("turn_type", f"Must be one of {valid_types}", turn_type))

        # Validate content
        if "content" in data:
            if not isinstance(data["content"], str):
                errors.append(SchemaValidationError("content", "Must be string", type(data["content"])))

        # Validate raw_metadata
        if "raw_metadata" in data:
            if not isinstance(data["raw_metadata"], dict):
                errors.append(SchemaValidationError("raw_metadata", "Must be dict", type(data["raw_metadata"])))

        return errors

    @staticmethod
    def validate_entity(data: Dict[str, Any]) -> List[SchemaValidationError]:
        """
        Validate an entity record.

        Expected schema:
        {
            "entity_id": str (UUID),
            "type": str ("person"|"org"|"location"|"date"|"tech"|"event"|"concept"),
            "name": str,
            "aliases": list[str],
            "confidence": float (0.0-1.0),
            "first_mention": {
                "message_hash": str,
                "timestamp": str
            },
            "mention_count": int (>= 1),
            "extraction_method": str ("spacy"|"gemini"|...)
        }
        """
        errors = []

        # Required fields
        required = ["entity_id", "type", "name", "aliases", "confidence", "first_mention", "mention_count", "extraction_method"]
        for field in required:
            if field not in data:
                errors.append(SchemaValidationError(field, "Missing required field", None))

        # Validate entity_id (UUID format)
        if "entity_id" in data:
            entity_id = data["entity_id"]
            uuid_pattern = r'^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$'
            if not isinstance(entity_id, str) or not re.match(uuid_pattern, entity_id):
                errors.append(SchemaValidationError("entity_id", "Must be valid UUID", entity_id))

        # Validate type
        if "type" in data:
            entity_type = data["type"]
            valid_types = ["person", "org", "location", "date", "tech", "event", "concept", "project"]
            if entity_type not in valid_types:
                errors.append(SchemaValidationError("type", f"Must be one of {valid_types}", entity_type))

        # Validate name
        if "name" in data:
            if not isinstance(data["name"], str) or not data["name"].strip():
                errors.append(SchemaValidationError("name", "Must be non-empty string", data["name"]))

        # Validate aliases
        if "aliases" in data:
            if not isinstance(data["aliases"], list):
                errors.append(SchemaValidationError("aliases", "Must be list", type(data["aliases"])))

        # Validate confidence (0.0-1.0)
        if "confidence" in data:
            confidence = data["confidence"]
            if not isinstance(confidence, (int, float)) or not (0.0 <= confidence <= 1.0):
                errors.append(SchemaValidationError("confidence", "Must be float between 0.0 and 1.0", confidence))

        # Validate first_mention
        if "first_mention" in data:
            first_mention = data["first_mention"]
            if not isinstance(first_mention, dict):
                errors.append(SchemaValidationError("first_mention", "Must be dict", type(first_mention)))
            else:
                if "message_hash" not in first_mention or "timestamp" not in first_mention:
                    errors.append(SchemaValidationError("first_mention", "Must contain message_hash and timestamp", first_mention))

        # Validate mention_count
        if "mention_count" in data:
            count = data["mention_count"]
            if not isinstance(count, int) or count < 1:
                errors.append(SchemaValidationError("mention_count", "Must be integer >= 1", count))

        # Validate extraction_method
        if "extraction_method" in data:
            method = data["extraction_method"]
            valid_methods = ["spacy", "gemini", "nltk", "regex", "manual"]
            if method not in valid_methods:
                errors.append(SchemaValidationError("extraction_method", f"Must be one of {valid_methods}", method))

        return errors

    @staticmethod
    def validate_artifact(data: Dict[str, Any]) -> List[SchemaValidationError]:
        """
        Validate an artifact record.

        Expected schema:
        {
            "artifact_id": str (UUID),
            "type": str ("code"|"file"|"document"|"image"),
            "language": str,
            "content": str,
            "content_hash": str (64-char hex),
            "context": str,
            "source_message": str (64-char hex),
            "timestamp": str (ISO-8601),
            "metadata": dict
        }
        """
        errors = []

        # Required fields
        required = ["artifact_id", "type", "language", "content", "content_hash", "context", "source_message", "timestamp", "metadata"]
        for field in required:
            if field not in data:
                errors.append(SchemaValidationError(field, "Missing required field", None))

        # Validate artifact_id (UUID)
        if "artifact_id" in data:
            artifact_id = data["artifact_id"]
            uuid_pattern = r'^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$'
            if not isinstance(artifact_id, str) or not re.match(uuid_pattern, artifact_id):
                errors.append(SchemaValidationError("artifact_id", "Must be valid UUID", artifact_id))

        # Validate type
        if "type" in data:
            artifact_type = data["type"]
            valid_types = ["code", "file", "document", "image", "other"]
            if artifact_type not in valid_types:
                errors.append(SchemaValidationError("type", f"Must be one of {valid_types}", artifact_type))

        # Validate content_hash (SHA256)
        if "content_hash" in data:
            hash_val = data["content_hash"]
            if not isinstance(hash_val, str) or len(hash_val) != 64 or not re.match(r'^[a-f0-9]{64}$', hash_val):
                errors.append(SchemaValidationError("content_hash", "Must be 64-char SHA256 hex string", hash_val))

        # Validate source_message (SHA256)
        if "source_message" in data:
            hash_val = data["source_message"]
            if not isinstance(hash_val, str) or len(hash_val) != 64 or not re.match(r'^[a-f0-9]{64}$', hash_val):
                errors.append(SchemaValidationError("source_message", "Must be 64-char SHA256 hex string", hash_val))

        # Validate timestamp
        if "timestamp" in data:
            timestamp = data["timestamp"]
            try:
                datetime.fromisoformat(timestamp.replace('Z', '+00:00'))
            except ValueError:
                errors.append(SchemaValidationError("timestamp", "Must be valid ISO-8601 format", timestamp))

        # Validate metadata
        if "metadata" in data:
            if not isinstance(data["metadata"], dict):
                errors.append(SchemaValidationError("metadata", "Must be dict", type(data["metadata"])))

        return errors


def validate_jsonl_file(filepath: str, record_type: str) -> Dict[str, Any]:
    """
    Validate an entire JSONL file.

    Args:
        filepath: Path to JSONL file
        record_type: "conversation"|"entity"|"artifact"

    Returns:
        Validation report dict
    """
    validator_map = {
        "conversation": OutputValidator.validate_conversation_turn,
        "entity": OutputValidator.validate_entity,
        "artifact": OutputValidator.validate_artifact
    }

    if record_type not in validator_map:
        raise ValueError(f"Unknown record type: {record_type}")

    validator = validator_map[record_type]

    report = {
        "filepath": filepath,
        "record_type": record_type,
        "total_records": 0,
        "valid_records": 0,
        "invalid_records": 0,
        "errors": []
    }

    try:
        with open(filepath, 'r', encoding='utf-8') as f:
            for line_num, line in enumerate(f, 1):
                try:
                    record = json.loads(line)
                    report["total_records"] += 1

                    errors = validator(record)
                    if errors:
                        report["invalid_records"] += 1
                        report["errors"].append({
                            "line": line_num,
                            "errors": [{"field": e.field, "message": e.message, "value": str(e.value)} for e in errors]
                        })
                    else:
                        report["valid_records"] += 1

                except json.JSONDecodeError as e:
                    report["invalid_records"] += 1
                    report["errors"].append({
                        "line": line_num,
                        "errors": [{"field": "json", "message": f"Invalid JSON: {e}", "value": line[:50]}]
                    })

    except FileNotFoundError:
        report["errors"].append({"field": "file", "message": f"File not found: {filepath}", "value": None})

    return report


def main():
    """CLI for schema validation"""
    import sys

    if len(sys.argv) < 3:
        print("Usage: python output_schemas.py <jsonl_file> <record_type>")
        print("\nRecord types: conversation, entity, artifact")
        print("\nExample:")
        print("  python output_schemas.py conversations.jsonl conversation")
        sys.exit(1)

    filepath = sys.argv[1]
    record_type = sys.argv[2]

    print(f"Validating {filepath} as {record_type} records...\n")

    report = validate_jsonl_file(filepath, record_type)

    print(f"Total records: {report['total_records']}")
    print(f"Valid records: {report['valid_records']}")
    print(f"Invalid records: {report['invalid_records']}")

    if report['errors']:
        print(f"\nErrors found:\n")
        for error_group in report['errors'][:10]:  # Show first 10 errors
            if "line" in error_group:
                print(f"Line {error_group['line']}:")
                for err in error_group['errors']:
                    print(f"  - {err['field']}: {err['message']}")
                    if err['value']:
                        print(f"    Value: {err['value'][:100]}")
            else:
                print(f"File error: {error_group['errors'][0]['message']}")

        if len(report['errors']) > 10:
            print(f"\n... and {len(report['errors']) - 10} more errors")

        sys.exit(1)
    else:
        print("\n✓ All records are valid!")
        sys.exit(0)


if __name__ == "__main__":
    main()

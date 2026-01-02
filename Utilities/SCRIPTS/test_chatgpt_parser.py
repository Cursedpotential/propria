#!/usr/bin/env python3
"""
Test script for ChatGPT Parser
Creates a sample ChatGPT export and validates parser output
"""

import json
import sys
from pathlib import Path
from datetime import datetime
import tempfile


def create_sample_chatgpt_export(output_path: Path):
    """
    Create a realistic sample ChatGPT export file for testing.

    Args:
        output_path: Path to save the sample export
    """
    sample_export = [
        {
            "id": "conv-001",
            "title": "Python Help Session",
            "create_time": 1700000000,
            "mapping": {
                "msg-001": {
                    "id": "msg-001",
                    "message": {
                        "id": "msg-001",
                        "author": {"role": "user"},
                        "create_time": 1700000000,
                        "content": {
                            "content_type": "text",
                            "parts": [
                                "Hi! I'm working with John Smith from Acme Corp on a Python project. Can you help me write a function to parse JSON files?"
                            ]
                        }
                    }
                },
                "msg-002": {
                    "id": "msg-002",
                    "message": {
                        "id": "msg-002",
                        "author": {"role": "assistant"},
                        "create_time": 1700000030,
                        "content": {
                            "content_type": "text",
                            "parts": [
                                "Of course! Here's a simple JSON parser function:\n\n```python\nimport json\n\ndef parse_json_file(filepath):\n    with open(filepath, 'r') as f:\n        return json.load(f)\n```\n\nThis function reads a JSON file and returns the parsed data."
                            ]
                        }
                    }
                },
                "msg-003": {
                    "id": "msg-003",
                    "message": {
                        "id": "msg-003",
                        "author": {"role": "user"},
                        "create_time": 1700000060,
                        "content": {
                            "content_type": "text",
                            "parts": [
                                "Great! Can you add error handling?"
                            ]
                        }
                    }
                },
                "msg-004": {
                    "id": "msg-004",
                    "message": {
                        "id": "msg-004",
                        "author": {"role": "assistant"},
                        "create_time": 1700000090,
                        "content": {
                            "content_type": "text",
                            "parts": [
                                "Here's the improved version with error handling:\n\n```python\nimport json\n\ndef parse_json_file(filepath):\n    try:\n        with open(filepath, 'r') as f:\n            return json.load(f)\n    except FileNotFoundError:\n        print(f\"Error: {filepath} not found\")\n        return None\n    except json.JSONDecodeError as e:\n        print(f\"Error parsing JSON: {e}\")\n        return None\n```"
                            ]
                        }
                    }
                }
            }
        },
        {
            "id": "conv-002",
            "title": "Meeting with Microsoft",
            "create_time": 1700010000,
            "mapping": {
                "msg-005": {
                    "id": "msg-005",
                    "message": {
                        "id": "msg-005",
                        "author": {"role": "user"},
                        "create_time": 1700010000,
                        "content": {
                            "content_type": "text",
                            "parts": [
                                "I have a meeting with Sarah Johnson from Microsoft in New York on December 15, 2024. Can you help me prepare talking points about our API integration?"
                            ]
                        }
                    }
                },
                "msg-006": {
                    "id": "msg-006",
                    "message": {
                        "id": "msg-006",
                        "author": {"role": "assistant"},
                        "create_time": 1700010030,
                        "content": {
                            "content_type": "text",
                            "parts": [
                                "Here are some key talking points for your meeting:\n\n1. API Performance\n2. Security Features\n3. Integration Timeline\n4. Support and Documentation\n\nWould you like me to elaborate on any of these?"
                            ]
                        }
                    }
                }
            }
        },
        {
            "id": "conv-003",
            "title": "JavaScript Code Review",
            "create_time": 1700020000,
            "mapping": {
                "msg-007": {
                    "id": "msg-007",
                    "message": {
                        "id": "msg-007",
                        "author": {"role": "user"},
                        "create_time": 1700020000,
                        "content": {
                            "content_type": "text",
                            "parts": [
                                "Can you review this JavaScript code?"
                            ]
                        }
                    }
                },
                "msg-008": {
                    "id": "msg-008",
                    "message": {
                        "id": "msg-008",
                        "author": {"role": "assistant"},
                        "create_time": 1700020030,
                        "content": {
                            "content_type": "text",
                            "parts": [
                                "I'd be happy to help! Please share the code you'd like me to review."
                            ]
                        }
                    }
                },
                "msg-009": {
                    "id": "msg-009",
                    "message": {
                        "id": "msg-009",
                        "author": {"role": "user"},
                        "create_time": 1700020060,
                        "content": {
                            "content_type": "text",
                            "parts": [
                                "Here it is:\n\n```javascript\nfunction fetchData(url) {\n    fetch(url)\n        .then(response => response.json())\n        .then(data => console.log(data))\n        .catch(error => console.error(error));\n}\n```"
                            ]
                        }
                    }
                }
            }
        }
    ]

    with open(output_path, 'w', encoding='utf-8') as f:
        json.dump(sample_export, f, indent=2)

    print(f"Created sample export: {output_path}")
    return output_path


def validate_output(output_dir: Path):
    """
    Validate parser output files.

    Args:
        output_dir: Directory containing output JSONL files
    """
    print("\n=== Validating Output ===\n")

    # Check conversations.jsonl
    conversations_file = output_dir / "conversations.jsonl"
    if conversations_file.exists():
        conversations = []
        with open(conversations_file, 'r', encoding='utf-8') as f:
            for line in f:
                conversations.append(json.loads(line))

        print(f"✓ conversations.jsonl: {len(conversations)} messages")

        # Validate schema
        required_fields = ["message_hash", "conversation_id", "platform", "timestamp", "turn_type", "content"]
        for conv in conversations[:3]:  # Check first 3
            for field in required_fields:
                assert field in conv, f"Missing field: {field}"
        print("  - Schema validation: PASSED")

        # Check hash uniqueness
        hashes = [c["message_hash"] for c in conversations]
        assert len(hashes) == len(set(hashes)), "Duplicate message hashes found!"
        print("  - Hash uniqueness: PASSED")

    else:
        print("✗ conversations.jsonl: NOT FOUND")

    # Check entities.jsonl
    entities_file = output_dir / "entities.jsonl"
    if entities_file.exists():
        entities = []
        with open(entities_file, 'r', encoding='utf-8') as f:
            for line in f:
                entities.append(json.loads(line))

        print(f"\n✓ entities.jsonl: {len(entities)} unique entities")

        # Show entity breakdown
        entity_types = {}
        for ent in entities:
            entity_types[ent["type"]] = entity_types.get(ent["type"], 0) + 1

        print("  Entity breakdown:")
        for ent_type, count in sorted(entity_types.items()):
            print(f"    - {ent_type}: {count}")

        # Validate schema
        required_fields = ["entity_id", "type", "name", "confidence", "first_mention"]
        for ent in entities[:3]:
            for field in required_fields:
                assert field in ent, f"Missing field: {field}"
        print("  - Schema validation: PASSED")

    else:
        print("\n✗ entities.jsonl: NOT FOUND")

    # Check artifacts.jsonl
    artifacts_file = output_dir / "artifacts.jsonl"
    if artifacts_file.exists():
        artifacts = []
        with open(artifacts_file, 'r', encoding='utf-8') as f:
            for line in f:
                artifacts.append(json.loads(line))

        print(f"\n✓ artifacts.jsonl: {len(artifacts)} code blocks")

        # Show language breakdown
        languages = {}
        for art in artifacts:
            lang = art["language"]
            languages[lang] = languages.get(lang, 0) + 1

        print("  Language breakdown:")
        for lang, count in sorted(languages.items()):
            print(f"    - {lang}: {count}")

        # Validate schema
        required_fields = ["artifact_id", "type", "language", "content", "content_hash"]
        for art in artifacts[:3]:
            for field in required_fields:
                assert field in art, f"Missing field: {field}"
        print("  - Schema validation: PASSED")

    else:
        print("\n✗ artifacts.jsonl: NOT FOUND")

    print("\n=== Validation Complete ===\n")


def main():
    """Run test workflow"""
    print("ChatGPT Parser Test Script\n")

    # Create temp directory for test
    with tempfile.TemporaryDirectory() as temp_dir:
        temp_path = Path(temp_dir)

        # Step 1: Create sample export
        print("Step 1: Creating sample ChatGPT export...")
        export_file = temp_path / "sample_chatgpt_export.json"
        create_sample_chatgpt_export(export_file)

        # Step 2: Run parser
        print("\nStep 2: Running parser...")
        print("-" * 50)

        # Import and run parser
        try:
            from chatgpt_parser import ChatGPTParser

            parser = ChatGPTParser(export_path=export_file, output_dir=temp_path)
            parser.run()

            print("-" * 50)

        except ImportError:
            print("\nERROR: Cannot import chatgpt_parser.py")
            print("Make sure chatgpt_parser.py is in the same directory as this test script.")
            sys.exit(1)
        except Exception as e:
            print(f"\nERROR during parsing: {e}")
            import traceback
            traceback.print_exc()
            sys.exit(1)

        # Step 3: Validate output
        print("\nStep 3: Validating output...")
        try:
            validate_output(temp_path)
        except AssertionError as e:
            print(f"\n✗ Validation FAILED: {e}")
            sys.exit(1)
        except Exception as e:
            print(f"\n✗ Validation ERROR: {e}")
            import traceback
            traceback.print_exc()
            sys.exit(1)

        print("=" * 50)
        print("ALL TESTS PASSED!")
        print("=" * 50)

        # Show sample outputs
        print("\nSample conversation record:")
        with open(temp_path / "conversations.jsonl", 'r') as f:
            sample = json.loads(f.readline())
            print(json.dumps(sample, indent=2))

        if (temp_path / "entities.jsonl").exists():
            print("\nSample entity record:")
            with open(temp_path / "entities.jsonl", 'r') as f:
                sample = json.loads(f.readline())
                print(json.dumps(sample, indent=2))

        if (temp_path / "artifacts.jsonl").exists():
            print("\nSample artifact record:")
            with open(temp_path / "artifacts.jsonl", 'r') as f:
                sample = json.loads(f.readline())
                # Truncate content for readability
                sample["content"] = sample["content"][:100] + "..." if len(sample["content"]) > 100 else sample["content"]
                print(json.dumps(sample, indent=2))


if __name__ == "__main__":
    main()

#!/usr/bin/env python3
"""
ChatGPT JSON Parser - Sprint 1
Parses ChatGPT export files and extracts conversations, entities, and artifacts.
"""

from pathlib import Path
import json
import hashlib
import spacy
import re
import sys
from typing import Iterator, Dict, List, Optional, Any
from datetime import datetime
from dataclasses import dataclass, asdict
import uuid


@dataclass
class ConversationTurn:
    """Normalized conversation turn schema"""
    message_hash: str
    conversation_id: str
    platform: str
    timestamp: str
    turn_type: str
    content: str
    raw_metadata: Dict[str, Any]


@dataclass
class Entity:
    """Extracted entity schema"""
    entity_id: str
    type: str
    name: str
    aliases: List[str]
    confidence: float
    first_mention: Dict[str, str]
    mention_count: int
    extraction_method: str


@dataclass
class Artifact:
    """Code block or file artifact schema"""
    artifact_id: str
    type: str
    language: str
    content: str
    content_hash: str
    context: str
    source_message: str
    timestamp: str
    metadata: Dict[str, Any]


class ChatGPTParser:
    """
    Parses ChatGPT JSON exports and extracts structured data.

    Features:
    - Auto-detects ChatGPT JSON schema variations
    - Hash-based deduplication
    - Incremental processing (streaming)
    - spaCy-based entity extraction
    - Regex-based code block detection
    """

    def __init__(self, export_path: Path, output_dir: Path = None):
        """
        Initialize parser with export file path.

        Args:
            export_path: Path to ChatGPT JSON export file
            output_dir: Directory for output files (defaults to same as export)
        """
        self.export_path = Path(export_path)
        self.output_dir = Path(output_dir) if output_dir else self.export_path.parent
        self.output_dir.mkdir(exist_ok=True)

        # Schema detection results
        self.schema_map = {}

        # Entity tracking for deduplication
        self.entity_tracker = {}

        # Statistics
        self.stats = {
            "conversations_processed": 0,
            "messages_processed": 0,
            "entities_extracted": 0,
            "artifacts_extracted": 0,
            "errors": 0
        }

        # Load spaCy model
        try:
            self.nlp = spacy.load("en_core_web_sm")
            self._log("Loaded spaCy model: en_core_web_sm")
        except OSError:
            self._log_error("spaCy model 'en_core_web_sm' not found. Install with: python -m spacy download en_core_web_sm")
            sys.exit(1)

    def _log(self, message: str):
        """Log info message to stderr"""
        print(f"[INFO] {message}", file=sys.stderr)

    def _log_error(self, message: str):
        """Log error message to stderr"""
        print(f"[ERROR] {message}", file=sys.stderr)
        self.stats["errors"] += 1

    def _log_progress(self):
        """Log processing progress"""
        print(f"[PROGRESS] Processed {self.stats['messages_processed']} messages, "
              f"{self.stats['entities_extracted']} entities, "
              f"{self.stats['artifacts_extracted']} artifacts",
              file=sys.stderr)

    def pre_scan_schema(self) -> Dict[str, str]:
        """
        Scan the first few conversations to auto-detect field structure.
        ChatGPT exports can vary in structure across versions.

        Returns:
            Dictionary mapping standard fields to detected field names
        """
        self._log("Pre-scanning ChatGPT export to detect schema...")

        try:
            with open(self.export_path, 'r', encoding='utf-8') as f:
                data = json.load(f)

            # ChatGPT exports are typically a list of conversations
            if isinstance(data, list):
                conversations = data[:5]  # Sample first 5
            elif isinstance(data, dict) and 'conversations' in data:
                conversations = data['conversations'][:5]
            else:
                self._log_error(f"Unexpected root structure: {type(data)}")
                return {}

            if not conversations:
                self._log_error("No conversations found in export")
                return {}

            # Analyze structure
            sample_conv = conversations[0]
            self._log(f"Sample conversation keys: {list(sample_conv.keys())}")

            # Common field mappings across ChatGPT versions
            schema_map = {
                "conversation_id": self._detect_field(sample_conv, ["id", "conversation_id", "uuid"]),
                "title": self._detect_field(sample_conv, ["title", "name"]),
                "create_time": self._detect_field(sample_conv, ["create_time", "created_at", "timestamp"]),
                "mapping": self._detect_field(sample_conv, ["mapping", "messages", "turns"]),
            }

            # Check message structure within mapping
            if schema_map["mapping"] and schema_map["mapping"] in sample_conv:
                mapping = sample_conv[schema_map["mapping"]]
                if isinstance(mapping, dict) and mapping:
                    sample_msg_id = next(iter(mapping.keys()))
                    sample_msg = mapping[sample_msg_id]

                    if "message" in sample_msg:
                        msg_content = sample_msg["message"]
                        schema_map["message_author"] = self._detect_field(msg_content, ["author", "role", "sender"])
                        schema_map["message_content"] = self._detect_field(msg_content, ["content", "text", "parts"])
                        schema_map["message_time"] = self._detect_field(msg_content, ["create_time", "timestamp", "created_at"])

            self.schema_map = schema_map
            self._log(f"Detected schema: {schema_map}")
            return schema_map

        except json.JSONDecodeError as e:
            self._log_error(f"Invalid JSON in export file: {e}")
            return {}
        except Exception as e:
            self._log_error(f"Error during schema detection: {e}")
            return {}

    def _detect_field(self, obj: Dict, candidates: List[str]) -> Optional[str]:
        """Find which candidate field name exists in object"""
        for candidate in candidates:
            if candidate in obj:
                return candidate
        return None

    def _generate_message_hash(self, content: str, timestamp: str) -> str:
        """
        Generate SHA256 hash for message deduplication.

        Args:
            content: Message content
            timestamp: ISO-8601 timestamp

        Returns:
            SHA256 hash string
        """
        hash_input = f"{content}{timestamp}chatgpt"
        return hashlib.sha256(hash_input.encode('utf-8')).hexdigest()

    def _normalize_timestamp(self, timestamp: Any) -> str:
        """
        Convert various timestamp formats to ISO-8601.

        Args:
            timestamp: Unix timestamp (int/float) or datetime string

        Returns:
            ISO-8601 formatted timestamp string
        """
        try:
            if isinstance(timestamp, (int, float)):
                dt = datetime.fromtimestamp(timestamp)
                return dt.isoformat() + 'Z'
            elif isinstance(timestamp, str):
                # Try parsing common formats
                for fmt in ["%Y-%m-%dT%H:%M:%S.%fZ", "%Y-%m-%dT%H:%M:%SZ", "%Y-%m-%d %H:%M:%S"]:
                    try:
                        dt = datetime.strptime(timestamp, fmt)
                        return dt.isoformat() + 'Z'
                    except ValueError:
                        continue
                # If already ISO-8601-ish, return as-is
                return timestamp
            else:
                return datetime.now().isoformat() + 'Z'
        except Exception as e:
            self._log_error(f"Error normalizing timestamp {timestamp}: {e}")
            return datetime.now().isoformat() + 'Z'

    def parse_conversations(self) -> Iterator[ConversationTurn]:
        """
        Parse ChatGPT export and yield normalized conversation turns.

        Yields:
            ConversationTurn objects
        """
        try:
            with open(self.export_path, 'r', encoding='utf-8') as f:
                data = json.load(f)

            # Get conversations list
            if isinstance(data, list):
                conversations = data
            elif isinstance(data, dict) and 'conversations' in data:
                conversations = data['conversations']
            else:
                self._log_error("Cannot find conversations in export structure")
                return

            self._log(f"Found {len(conversations)} conversations to process")

            for conv_idx, conversation in enumerate(conversations):
                try:
                    conv_id = conversation.get(self.schema_map.get("conversation_id", "id"), f"unknown_{conv_idx}")
                    conv_title = conversation.get(self.schema_map.get("title", "title"), "Untitled")

                    # Get message mapping
                    mapping_field = self.schema_map.get("mapping", "mapping")
                    if mapping_field not in conversation:
                        self._log_error(f"No mapping field in conversation {conv_id}")
                        continue

                    mapping = conversation[mapping_field]
                    if not isinstance(mapping, dict):
                        self._log_error(f"Mapping is not a dict in conversation {conv_id}")
                        continue

                    # Process messages in order
                    for msg_id, msg_data in mapping.items():
                        try:
                            # Skip empty or malformed messages
                            if not msg_data or "message" not in msg_data:
                                continue

                            message = msg_data["message"]
                            if not message:
                                continue

                            # Extract author/role
                            author_field = self.schema_map.get("message_author", "author")
                            author_data = message.get(author_field, {})

                            if isinstance(author_data, dict):
                                role = author_data.get("role", "unknown")
                            elif isinstance(author_data, str):
                                role = author_data
                            else:
                                role = "unknown"

                            # Map to turn_type
                            if role in ["user", "human"]:
                                turn_type = "user"
                            elif role in ["assistant", "system"]:
                                turn_type = "assistant"
                            else:
                                turn_type = "unknown"

                            # Extract content
                            content_field = self.schema_map.get("message_content", "content")
                            content_data = message.get(content_field, {})

                            if isinstance(content_data, dict):
                                # Content is often in "parts" array
                                parts = content_data.get("parts", [])
                                if parts and isinstance(parts, list):
                                    content = "\n".join(str(p) for p in parts if p)
                                else:
                                    content = str(content_data.get("text", ""))
                            elif isinstance(content_data, list):
                                content = "\n".join(str(p) for p in content_data if p)
                            elif isinstance(content_data, str):
                                content = content_data
                            else:
                                content = ""

                            # Skip empty content
                            if not content.strip():
                                continue

                            # Extract timestamp
                            time_field = self.schema_map.get("message_time", "create_time")
                            timestamp_raw = message.get(time_field, message.get("created_at", None))
                            timestamp = self._normalize_timestamp(timestamp_raw)

                            # Generate hash
                            message_hash = self._generate_message_hash(content, timestamp)

                            # Build conversation turn
                            turn = ConversationTurn(
                                message_hash=message_hash,
                                conversation_id=f"chatgpt_{conv_id}",
                                platform="chatgpt",
                                timestamp=timestamp,
                                turn_type=turn_type,
                                content=content,
                                raw_metadata={
                                    "original_format": "json",
                                    "export_date": datetime.now().strftime("%Y-%m-%d"),
                                    "conversation_title": conv_title,
                                    "message_id": msg_id,
                                    "role": role
                                }
                            )

                            self.stats["messages_processed"] += 1
                            yield turn

                        except Exception as e:
                            self._log_error(f"Error processing message {msg_id} in conversation {conv_id}: {e}")
                            continue

                    self.stats["conversations_processed"] += 1

                    # Progress update every 10 conversations
                    if (conv_idx + 1) % 10 == 0:
                        self._log_progress()

                except Exception as e:
                    self._log_error(f"Error processing conversation {conv_idx}: {e}")
                    continue

        except Exception as e:
            self._log_error(f"Fatal error reading export file: {e}")
            return

    def extract_entities(self, content: str, message_hash: str, timestamp: str) -> List[Entity]:
        """
        Extract named entities using spaCy NER.

        Args:
            content: Message text
            message_hash: Source message hash
            timestamp: Message timestamp

        Returns:
            List of Entity objects
        """
        entities = []

        try:
            # Process with spaCy
            doc = self.nlp(content)

            for ent in doc.ents:
                # Filter for relevant entity types
                if ent.label_ not in ["PERSON", "ORG", "GPE", "DATE", "PRODUCT", "EVENT"]:
                    continue

                entity_name = ent.text.strip()
                entity_type = self._map_entity_type(ent.label_)

                # Track entity for deduplication
                entity_key = f"{entity_type}:{entity_name.lower()}"

                if entity_key in self.entity_tracker:
                    # Update existing entity
                    self.entity_tracker[entity_key]["mention_count"] += 1
                else:
                    # Create new entity
                    entity = Entity(
                        entity_id=str(uuid.uuid4()),
                        type=entity_type,
                        name=entity_name,
                        aliases=[],
                        confidence=0.85,  # spaCy default confidence
                        first_mention={
                            "message_hash": message_hash,
                            "timestamp": timestamp
                        },
                        mention_count=1,
                        extraction_method="spacy"
                    )

                    self.entity_tracker[entity_key] = asdict(entity)
                    entities.append(entity)
                    self.stats["entities_extracted"] += 1

        except Exception as e:
            self._log_error(f"Error extracting entities from message {message_hash}: {e}")

        return entities

    def _map_entity_type(self, spacy_label: str) -> str:
        """Map spaCy entity labels to our entity types"""
        mapping = {
            "PERSON": "person",
            "ORG": "org",
            "GPE": "location",
            "DATE": "date",
            "PRODUCT": "tech",
            "EVENT": "event"
        }
        return mapping.get(spacy_label, "concept")

    def extract_artifacts(self, content: str, message_hash: str, timestamp: str, context: str = "") -> List[Artifact]:
        """
        Extract code blocks and file artifacts using regex.

        Args:
            content: Message text
            message_hash: Source message hash
            timestamp: Message timestamp
            context: Surrounding conversation context

        Returns:
            List of Artifact objects
        """
        artifacts = []

        # Regex pattern for code blocks (```language ... ```)
        code_block_pattern = r'```(\w+)?\n(.*?)```'

        try:
            matches = re.finditer(code_block_pattern, content, re.DOTALL)

            for match in matches:
                language = match.group(1) or "unknown"
                code_content = match.group(2).strip()

                if not code_content:
                    continue

                # Generate content hash
                content_hash = hashlib.sha256(code_content.encode('utf-8')).hexdigest()

                artifact = Artifact(
                    artifact_id=str(uuid.uuid4()),
                    type="code",
                    language=language.lower(),
                    content=code_content,
                    content_hash=content_hash,
                    context=context[:200],  # First 200 chars of context
                    source_message=message_hash,
                    timestamp=timestamp,
                    metadata={
                        "size_bytes": len(code_content.encode('utf-8'))
                    }
                )

                artifacts.append(artifact)
                self.stats["artifacts_extracted"] += 1

        except Exception as e:
            self._log_error(f"Error extracting artifacts from message {message_hash}: {e}")

        return artifacts

    def write_jsonl(self, items: List[Any], output_file: Path):
        """
        Write list of dataclass objects to JSONL file.

        Args:
            items: List of dataclass instances
            output_file: Output file path
        """
        try:
            with open(output_file, 'a', encoding='utf-8') as f:
                for item in items:
                    if hasattr(item, '__dataclass_fields__'):
                        json_line = json.dumps(asdict(item), ensure_ascii=False)
                    else:
                        json_line = json.dumps(item, ensure_ascii=False)
                    f.write(json_line + '\n')
        except Exception as e:
            self._log_error(f"Error writing to {output_file}: {e}")

    def run(self):
        """
        Main pipeline execution.

        Steps:
        1. Pre-scan schema
        2. Parse conversations
        3. Extract entities (per message)
        4. Extract artifacts (per message)
        5. Write JSONL outputs
        """
        self._log(f"Starting ChatGPT parser for: {self.export_path}")

        # Step 1: Pre-scan schema
        if not self.pre_scan_schema():
            self._log_error("Schema detection failed. Cannot proceed.")
            return

        # Prepare output files
        conversations_file = self.output_dir / "conversations.jsonl"
        entities_file = self.output_dir / "entities.jsonl"
        artifacts_file = self.output_dir / "artifacts.jsonl"

        # Clear existing output files
        for file in [conversations_file, entities_file, artifacts_file]:
            if file.exists():
                file.unlink()
                self._log(f"Cleared existing output: {file}")

        # Step 2-5: Process conversations incrementally
        self._log("Processing conversations...")

        conversation_context = []  # For context window

        for turn in self.parse_conversations():
            try:
                # Write conversation turn
                self.write_jsonl([turn], conversations_file)

                # Extract entities (only from user messages for efficiency)
                if turn.turn_type == "user":
                    entities = self.extract_entities(
                        content=turn.content,
                        message_hash=turn.message_hash,
                        timestamp=turn.timestamp
                    )
                    if entities:
                        self.write_jsonl(entities, entities_file)

                # Extract artifacts (from all messages)
                context = "\n".join(conversation_context[-3:])  # Last 3 messages
                artifacts = self.extract_artifacts(
                    content=turn.content,
                    message_hash=turn.message_hash,
                    timestamp=turn.timestamp,
                    context=context
                )
                if artifacts:
                    self.write_jsonl(artifacts, artifacts_file)

                # Update context window
                conversation_context.append(turn.content[:100])  # Keep first 100 chars
                if len(conversation_context) > 10:
                    conversation_context.pop(0)

            except Exception as e:
                self._log_error(f"Error processing turn {turn.message_hash}: {e}")
                continue

        # Final progress report
        self._log_progress()
        self._log("\n=== Processing Complete ===")
        self._log(f"Conversations: {self.stats['conversations_processed']}")
        self._log(f"Messages: {self.stats['messages_processed']}")
        self._log(f"Entities: {self.stats['entities_extracted']}")
        self._log(f"Artifacts: {self.stats['artifacts_extracted']}")
        self._log(f"Errors: {self.stats['errors']}")
        self._log(f"\nOutput files:")
        self._log(f"  - {conversations_file}")
        self._log(f"  - {entities_file}")
        self._log(f"  - {artifacts_file}")


def main():
    """CLI entry point"""
    if len(sys.argv) < 2:
        print("Usage: python chatgpt_parser.py <chatgpt_export.json> [output_dir]")
        print("\nExample:")
        print("  python chatgpt_parser.py conversations.json")
        print("  python chatgpt_parser.py conversations.json ./output")
        sys.exit(1)

    export_path = sys.argv[1]
    output_dir = sys.argv[2] if len(sys.argv) > 2 else None

    parser = ChatGPTParser(export_path=export_path, output_dir=output_dir)
    parser.run()


if __name__ == "__main__":
    main()

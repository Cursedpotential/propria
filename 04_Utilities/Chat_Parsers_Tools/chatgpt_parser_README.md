# ChatGPT Parser - Sprint 1

Production-ready Python script for parsing ChatGPT export files and extracting structured conversation data, entities, and code artifacts.

## Features

- **Auto-schema detection**: Automatically detects ChatGPT JSON structure variations
- **Hash-based deduplication**: SHA256 hashing prevents duplicate messages
- **Incremental processing**: Streams data without loading entire file into memory
- **Entity extraction**: Uses spaCy NER to extract people, organizations, locations, dates
- **Artifact detection**: Regex-based code block extraction with language detection
- **Error resilience**: Never crashes - logs all errors and continues processing
- **Progress tracking**: Real-time progress indicators

## Installation

### Prerequisites

- Python 3.10 or higher
- pip package manager

### Install Dependencies

```bash
# Install required packages
pip install spacy

# Download spaCy language model
python -m spacy download en_core_web_sm
```

Or use the requirements file:

```bash
pip install -r chatgpt_parser_requirements.txt
python -m spacy download en_core_web_sm
```

## Usage

### Basic Usage

```bash
python chatgpt_parser.py <chatgpt_export.json>
```

This will create output files in the same directory as the export file.

### Specify Output Directory

```bash
python chatgpt_parser.py <chatgpt_export.json> <output_dir>
```

### Example

```bash
# Parse ChatGPT export
python chatgpt_parser.py ~/Downloads/conversations.json

# Parse with custom output location
python chatgpt_parser.py ~/Downloads/conversations.json ./parsed_output
```

## Output Files

The parser generates three JSONL files:

### 1. conversations.jsonl

Normalized conversation turns with deduplication hashes.

**Schema:**
```json
{
  "message_hash": "sha256_hash",
  "conversation_id": "chatgpt_conv_123",
  "platform": "chatgpt",
  "timestamp": "2024-11-15T14:30:00Z",
  "turn_type": "user|assistant",
  "content": "Message text here",
  "raw_metadata": {
    "original_format": "json",
    "export_date": "2024-12-20",
    "conversation_title": "Python Help",
    "message_id": "msg_abc123",
    "role": "user"
  }
}
```

### 2. entities.jsonl

Named entities extracted using spaCy NER.

**Schema:**
```json
{
  "entity_id": "uuid-here",
  "type": "person|org|location|date|tech|event|concept",
  "name": "Entity Name",
  "aliases": [],
  "confidence": 0.85,
  "first_mention": {
    "message_hash": "sha256_hash",
    "timestamp": "2024-11-15T14:30:00Z"
  },
  "mention_count": 42,
  "extraction_method": "spacy"
}
```

**Entity Types:**
- `person` - People (PERSON)
- `org` - Organizations (ORG)
- `location` - Geographic locations (GPE)
- `date` - Dates and times (DATE)
- `tech` - Products and technologies (PRODUCT)
- `event` - Events (EVENT)
- `concept` - Other concepts

### 3. artifacts.jsonl

Code blocks and file content extracted from messages.

**Schema:**
```json
{
  "artifact_id": "uuid-here",
  "type": "code",
  "language": "python",
  "content": "def hello():\n    print('world')",
  "content_hash": "sha256_of_content",
  "context": "First 200 chars of surrounding conversation",
  "source_message": "message_hash",
  "timestamp": "2024-11-15T14:30:00Z",
  "metadata": {
    "size_bytes": 1234
  }
}
```

**Supported Code Languages:**
Automatically detected from code fence markers (```python, ```javascript, etc.)

## Implementation Details

### Hash Generation

Messages are hashed using SHA256 of:
```
sha256(content + timestamp + "chatgpt")
```

This enables:
- Deduplication across multiple exports
- Content verification
- Change detection

### Timestamp Normalization

All timestamps are converted to ISO-8601 format:
- Unix timestamps (int/float) → `2024-11-15T14:30:00Z`
- Various string formats → Standardized ISO-8601

### Entity Extraction

Uses spaCy's `en_core_web_sm` model:
- Processes **user messages only** (for efficiency)
- Extracts: PERSON, ORG, GPE, DATE, PRODUCT, EVENT
- Deduplicates entities across conversations
- Tracks mention counts and first occurrence

### Artifact Detection

Regex pattern: ` ```language\ncontent``` `
- Detects fenced code blocks
- Extracts language identifier
- Computes content hash for deduplication
- Stores surrounding context (last 3 messages)

### Memory Efficiency

- **Streaming parser**: Processes one conversation at a time
- **Incremental writes**: Writes to JSONL as data is processed
- **Context window**: Maintains only last 10 message snippets
- **No full file load**: Suitable for large exports (10K+ messages)

## Error Handling

The parser is designed to never crash:

- **Invalid JSON**: Logs error and exits gracefully
- **Malformed messages**: Skips and continues
- **Missing fields**: Uses defaults and logs warning
- **Encoding issues**: Handles UTF-8 errors
- **All errors logged to stderr**

## Progress Tracking

Progress is logged to stderr every 10 conversations:

```
[INFO] Loaded spaCy model: en_core_web_sm
[INFO] Pre-scanning ChatGPT export to detect schema...
[INFO] Detected schema: {...}
[INFO] Found 45 conversations to process
[PROGRESS] Processed 127 messages, 23 entities, 8 artifacts
[PROGRESS] Processed 256 messages, 47 entities, 15 artifacts
...
[INFO] === Processing Complete ===
[INFO] Conversations: 45
[INFO] Messages: 623
[INFO] Entities: 89
[INFO] Artifacts: 34
[INFO] Errors: 0
```

## Validation

To validate output JSONL files:

```bash
# Check conversations.jsonl
cat conversations.jsonl | jq -c '.'

# Count entities by type
cat entities.jsonl | jq -r '.type' | sort | uniq -c

# List all extracted languages
cat artifacts.jsonl | jq -r '.language' | sort | uniq -c
```

## Performance

Expected performance on modern hardware:
- **Processing speed**: 1000+ messages/minute
- **Memory usage**: <500MB for 10K messages
- **Output size**: ~1.5x size of input JSON

## Troubleshooting

### spaCy model not found

```bash
python -m spacy download en_core_web_sm
```

### Out of memory errors

Process export in chunks using `head`/`tail` to split the JSON file.

### No entities extracted

- Check that messages contain proper nouns
- Verify spaCy model is loaded (check stderr logs)
- Entity extraction only runs on user messages

### Empty output files

- Check stderr for errors
- Verify export file is valid JSON
- Ensure export contains "mapping" or "messages" field

## Next Steps

This is Sprint 1 of the chat parser project. Future sprints will add:

- **Sprint 2**: Claude export parser
- **Sprint 3**: Gemini, Perplexity, Qwen parsers
- **Sprint 4**: Event detection and document classification
- **Sprint 5**: LLM-based advanced extraction

## Architecture

```
┌─────────────────────────────────────┐
│  ChatGPT JSON Export                │
└─────────────┬───────────────────────┘
              │
              ▼
┌─────────────────────────────────────┐
│  Pre-scan Schema Detection          │
│  (Auto-detect field variations)     │
└─────────────┬───────────────────────┘
              │
              ▼
┌─────────────────────────────────────┐
│  Parse Conversations                │
│  - Generate SHA256 hashes           │
│  - Normalize timestamps             │
│  - Stream to conversations.jsonl    │
└─────────────┬───────────────────────┘
              │
              ▼
┌─────────────────────────────────────┐
│  Extract Entities (spaCy)           │
│  - PERSON, ORG, GPE, DATE, etc.     │
│  - Deduplicate and track mentions   │
│  - Stream to entities.jsonl         │
└─────────────┬───────────────────────┘
              │
              ▼
┌─────────────────────────────────────┐
│  Extract Artifacts (regex)          │
│  - Detect code blocks               │
│  - Extract language and content     │
│  - Stream to artifacts.jsonl        │
└─────────────┬───────────────────────┘
              │
              ▼
┌─────────────────────────────────────┐
│  Output: 3 JSONL Files              │
│  - conversations.jsonl              │
│  - entities.jsonl                   │
│  - artifacts.jsonl                  │
└─────────────────────────────────────┘
```

## License

Part of the Chat History Parser project.

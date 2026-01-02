# ChatGPT Parser - Quick Reference Card

## Installation (One-Time)

```bash
pip install spacy
python -m spacy download en_core_web_sm
```

## Usage

```bash
# Basic
python chatgpt_parser.py <export.json>

# Custom output location
python chatgpt_parser.py <export.json> <output_dir>
```

## Testing

```bash
# Run automated tests
python test_chatgpt_parser.py

# Validate output files
python output_schemas.py conversations.jsonl conversation
python output_schemas.py entities.jsonl entity
python output_schemas.py artifacts.jsonl artifact
```

## Output Files

| File | Contents |
|------|----------|
| conversations.jsonl | All chat messages with SHA256 hashes |
| entities.jsonl | People, orgs, locations, dates extracted |
| artifacts.jsonl | Code blocks with language detection |

## Common Commands

```bash
# Count messages
wc -l conversations.jsonl

# View first message
head -n 1 conversations.jsonl | jq '.'

# Count entities by type
cat entities.jsonl | jq -r '.type' | sort | uniq -c

# List programming languages found
cat artifacts.jsonl | jq -r '.language' | sort | uniq -c

# Find all Python code blocks
cat artifacts.jsonl | jq 'select(.language=="python")'

# Search for specific entity
cat entities.jsonl | jq 'select(.name | contains("Smith"))'

# Get conversation titles
cat conversations.jsonl | jq -r '.raw_metadata.conversation_title' | sort | uniq
```

## Schema Quick View

### Conversation Turn
```json
{
  "message_hash": "sha256...",
  "conversation_id": "chatgpt_conv_123",
  "platform": "chatgpt",
  "timestamp": "2024-11-15T14:30:00Z",
  "turn_type": "user|assistant",
  "content": "text...",
  "raw_metadata": {...}
}
```

### Entity
```json
{
  "entity_id": "uuid...",
  "type": "person|org|location|date|tech|event|concept",
  "name": "Name",
  "confidence": 0.85,
  "mention_count": 42,
  "extraction_method": "spacy"
}
```

### Artifact
```json
{
  "artifact_id": "uuid...",
  "type": "code",
  "language": "python",
  "content": "code...",
  "content_hash": "sha256...",
  "source_message": "message_hash"
}
```

## Troubleshooting

| Problem | Solution |
|---------|----------|
| "No module 'spacy'" | `pip install spacy` |
| "Model 'en_core_web_sm' not found" | `python -m spacy download en_core_web_sm` |
| "File not found" | Use full path to export file |
| No entities extracted | Normal for code-heavy conversations |
| Out of memory | Close other apps, parser is memory-efficient |

## Get ChatGPT Export

1. ChatGPT → Settings → Data controls → Export data
2. Wait for email with download link
3. Download and extract ZIP
4. Find `conversations.json`

## Performance

- Speed: 1000+ messages/minute
- Memory: <500MB for 10K messages
- Storage: ~1.5x input file size

## Files Reference

| File | Purpose |
|------|---------|
| chatgpt_parser.py | Main parser (run this) |
| test_chatgpt_parser.py | Automated tests |
| output_schemas.py | Validate output |
| chatgpt_parser_README.md | Full documentation |
| QUICKSTART.md | Setup guide |
| SPRINT1_COMPLETE.md | Implementation summary |

## Next Steps

1. Parse your ChatGPT export
2. Explore the output JSONL files
3. Use the data for analysis/search
4. Wait for Sprint 2 (Claude parser)

## Getting Help

- Check QUICKSTART.md for setup
- Read chatgpt_parser_README.md for details
- Run test_chatgpt_parser.py to verify installation
- Check stderr output for error messages

---

**Quick Start:** `python chatgpt_parser.py conversations.json`

# ChatGPT Parser - Quick Start Guide

Get up and running with the ChatGPT parser in 5 minutes.

## Prerequisites

- Python 3.10 or higher
- pip package manager
- A ChatGPT export file (JSON format)

## How to Get ChatGPT Export

1. Go to ChatGPT (chat.openai.com)
2. Click your profile icon (bottom left)
3. Settings → Data controls → Export data
4. Wait for email with download link
5. Download and extract the ZIP file
6. Find `conversations.json` in the extracted folder

## Installation

### 1. Install Dependencies

```bash
# Install spaCy
pip install spacy

# Download English language model
python -m spacy download en_core_web_sm
```

Or use requirements file:

```bash
pip install -r chatgpt_parser_requirements.txt
python -m spacy download en_core_web_sm
```

### 2. Verify Installation

```bash
# Check Python version
python --version  # Should be 3.10+

# Check spaCy installation
python -c "import spacy; print(spacy.load('en_core_web_sm'))"
```

## Usage

### Run the Parser

```bash
python chatgpt_parser.py /path/to/conversations.json
```

### Example

```bash
# Windows
python chatgpt_parser.py C:\Users\YourName\Downloads\conversations.json

# Mac/Linux
python chatgpt_parser.py ~/Downloads/conversations.json
```

### With Custom Output Directory

```bash
python chatgpt_parser.py conversations.json ./output
```

## Testing

Run the test script to validate your setup:

```bash
python test_chatgpt_parser.py
```

This will:
1. Create a sample ChatGPT export
2. Run the parser
3. Validate all output files
4. Show sample records

Expected output:
```
ChatGPT Parser Test Script

Step 1: Creating sample ChatGPT export...
Created sample export: /tmp/.../sample_chatgpt_export.json

Step 2: Running parser...
--------------------------------------------------
[INFO] Loaded spaCy model: en_core_web_sm
[INFO] Pre-scanning ChatGPT export to detect schema...
[INFO] Detected schema: {...}
[INFO] Found 3 conversations to process
[PROGRESS] Processed 9 messages, 4 entities, 3 artifacts
...
--------------------------------------------------

Step 3: Validating output...

=== Validating Output ===

✓ conversations.jsonl: 9 messages
  - Schema validation: PASSED
  - Hash uniqueness: PASSED

✓ entities.jsonl: 4 unique entities
  Entity breakdown:
    - person: 2
    - org: 1
    - location: 1
  - Schema validation: PASSED

✓ artifacts.jsonl: 3 code blocks
  Language breakdown:
    - python: 2
    - javascript: 1
  - Schema validation: PASSED

=== Validation Complete ===

==================================================
ALL TESTS PASSED!
==================================================
```

## Output Files

After running the parser, you'll get three JSONL files:

1. **conversations.jsonl** - All chat messages with hashes
2. **entities.jsonl** - Extracted people, companies, locations, dates
3. **artifacts.jsonl** - Code blocks with language detection

## Inspecting Output

### View conversations
```bash
# Windows PowerShell
Get-Content conversations.jsonl | Select-Object -First 5

# Mac/Linux
head -n 5 conversations.jsonl
```

### Count entities
```bash
# With jq (JSON processor)
cat entities.jsonl | jq -r '.type' | sort | uniq -c
```

### List code languages
```bash
cat artifacts.jsonl | jq -r '.language' | sort | uniq -c
```

## Common Issues

### "ModuleNotFoundError: No module named 'spacy'"

**Fix:**
```bash
pip install spacy
```

### "Can't find model 'en_core_web_sm'"

**Fix:**
```bash
python -m spacy download en_core_web_sm
```

### "No such file or directory: conversations.json"

**Fix:** Provide the full path to your ChatGPT export file:
```bash
python chatgpt_parser.py "C:\Users\YourName\Downloads\conversations.json"
```

### Out of memory on large exports

**Fix:** The parser streams data, so this shouldn't happen. If it does:
1. Check available RAM
2. Close other applications
3. Consider splitting the export file

### No entities extracted

This is normal if:
- Your conversations don't mention people/companies/locations
- Messages are mostly code or technical content

**Check:** Look at the raw conversations.jsonl to verify parsing worked.

## Next Steps

1. Explore the output JSONL files
2. Use the data for analysis, search, or archival
3. Try parsing multiple exports and compare hashes for deduplication
4. Wait for Sprint 2 to parse Claude exports

## Getting Help

If you encounter issues:

1. Check the error messages in the console (stderr)
2. Run the test script to isolate the problem
3. Verify your ChatGPT export is valid JSON
4. Check the README for detailed documentation

## Performance Expectations

On a typical laptop:
- **Speed**: 1000+ messages per minute
- **Memory**: <500MB for 10K messages
- **Storage**: Output files are ~1.5x the size of input

For a typical ChatGPT export (1000-5000 messages):
- **Processing time**: 1-5 minutes
- **Output size**: 2-10 MB total

## File Locations

After running the parser with default settings:

```
/path/to/your/export/
├── conversations.json          (your original export)
├── conversations.jsonl         (parsed messages)
├── entities.jsonl              (extracted entities)
└── artifacts.jsonl             (code blocks)
```

## What's Next?

This is Sprint 1. Future versions will add:
- Claude export parsing
- Gemini, Perplexity, Qwen support
- Event detection (meetings, decisions, milestones)
- Document classification (strategy docs, legal docs)
- LLM-based advanced extraction

---

**You're all set!** Start parsing your ChatGPT history.

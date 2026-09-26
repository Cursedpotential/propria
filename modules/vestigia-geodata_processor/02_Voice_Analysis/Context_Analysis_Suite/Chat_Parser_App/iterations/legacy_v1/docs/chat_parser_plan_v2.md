# Chat History Parser - UPDATED PLAN (Post-Opus Review)

## Opus Strategic Review: ✅ GREEN LIGHT (with conditions)

**Key Takeaway**: Build one complete vertical slice first. Prove the pipeline end-to-end before expanding.

---

## CRITICAL CHANGE: Separate Parser from Database

### This Project Scope (Parser/Tagger)
**Goal**: Extract and tag conversations → Output structured JSON
- Parse exports from ChatGPT, Claude, Gemini, Perplexity, Qwen
- Extract conversation turns (user/assistant)
- Tag entities, events, artifacts, documents
- **Output**: Clean JSON files with all tagged data
- **NO database decisions** - that's Stage 2

### Future Project (Database Ingestion - separate subagent)
- Design database schema
- Handle UUIDs and deduplication
- Create relationships between entities
- Build query interface

**Benefit**: Parser can be tested/iterated independently of database design.

---

## Opus's Critical Feedback Incorporated

### 1. ✅ Use spaCy for NER (not NLTK)
Better accuracy out-of-the-box for entity extraction.

### 2. ✅ Hash-based Deduplication from Day One
Every message gets SHA256 hash of (content + timestamp + platform) to detect duplicates.

### 3. ✅ Add Confidence Scores
All extractions (entities, events, classifications) include confidence 0.0-1.0.

### 4. ✅ Start with ONE Platform (ChatGPT JSON)
Build complete vertical slice before adding more platforms.

### 5. ✅ Defer OCR/PDF Processing
Too complex for v1 - add after core pipeline works.

### 6. ✅ Separate Raw from Processed
- Raw normalized conversations (one file)
- Extracted entities/events (separate files)
- Allows re-extraction without re-parsing

---

## Revised Architecture

```
┌─────────────────────────────────────────────┐
│  Input: Platform Exports                    │
│  - ChatGPT (JSON)                           │
│  - Claude (JSON/Markdown)                   │
│  - Gemini (JSON)                            │
│  - Perplexity (Markdown/HTML)               │
│  - Qwen (JSON)                              │
└─────────────┬───────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────────┐
│  Stage 1: Format Detection & Parsing        │
│  (Gemini codes this)                        │
│                                             │
│  • Detect platform and format               │
│  • Parse to conversation turns              │
│  • Generate message hash for dedup          │
│  • Normalize timestamps                     │
│                                             │
│  Output: normalized_conversations.jsonl     │
└─────────────┬───────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────────┐
│  Stage 2: Local Preprocessing (spaCy)      │
│  (Gemini codes this)                        │
│                                             │
│  • Extract entities (people, orgs, tech)    │
│  • Detect code blocks & language            │
│  • Pattern match (legal, planning keywords) │
│  • Flag conversations for LLM analysis      │
│                                             │
│  Output: preprocessed_data.jsonl            │
└─────────────┬───────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────────┐
│  Stage 3: Selective LLM Analysis (Gemini)  │
│  (Gemini codes this)                        │
│                                             │
│  • Only on flagged conversations (10-20%)   │
│  • Extract strategy documents               │
│  • Classify legal documents                 │
│  • Build entity relationships               │
│  • Rank event importance                    │
│                                             │
│  Output: llm_analysis.jsonl                 │
└─────────────┬───────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────────┐
│  Final Output: Tagged JSON Files            │
│                                             │
│  1. conversations.jsonl - All normalized    │
│  2. entities.jsonl - All extracted entities │
│  3. events.jsonl - Timeline events          │
│  4. artifacts.jsonl - Code, files, docs     │
│  5. documents.jsonl - Strategy, legal docs  │
│                                             │
│  → Hand off to Database Ingestion Agent     │
└─────────────────────────────────────────────┘
```

---

## Output Schema (What Gemini Will Produce)

### 1. conversations.jsonl
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
    "export_date": "2024-12-20"
  }
}
```

### 2. entities.jsonl
```json
{
  "entity_id": "entity_uuid",
  "type": "person|org|project|tech|location|concept",
  "name": "Entity Name",
  "aliases": ["alt name"],
  "confidence": 0.95,
  "first_mention": {
    "message_hash": "...",
    "timestamp": "..."
  },
  "mention_count": 42,
  "extraction_method": "spacy|gemini"
}
```

### 3. events.jsonl
```json
{
  "event_id": "event_uuid",
  "type": "milestone|decision|meeting|incident|change|memory|upcoming",
  "temporal_type": "historical|current|future",
  "title": "Brief description",
  "description": "Detailed context",
  "timestamp": "2024-11-15T00:00:00Z",
  "confidence": 0.85,
  "entities_involved": ["entity_uuid1", "entity_uuid2"],
  "source_messages": ["message_hash1", "message_hash2"],
  "tags": ["project-x", "critical"],
  "sentiment": {
    "polarity": 0.5,
    "subjectivity": 0.6,
    "emotion": "positive|negative|neutral|mixed"
  }
}
```

### 4. artifacts.jsonl
```json
{
  "artifact_id": "artifact_uuid",
  "type": "code|file|document|image",
  "language": "python|javascript|...",
  "content": "Code or file content",
  "content_hash": "sha256_of_content",
  "context": "Surrounding conversation context",
  "source_message": "message_hash",
  "timestamp": "...",
  "metadata": {
    "file_name": "optional",
    "size_bytes": 1234
  }
}
```

### 5. documents.jsonl
```json
{
  "document_id": "doc_uuid",
  "type": "strategy|legal|technical|other",
  "title": "Document title/summary",
  "content": "Full document text or reference",
  "confidence": 0.90,
  "classification_method": "nltk_pattern|gemini",
  "source_messages": ["message_hash1", "message_hash2"],
  "timestamp": "...",
  "entities_mentioned": ["entity_uuid1", "entity_uuid2"]
}
```

---

## Implementation Plan for Gemini

### Sprint 1: ChatGPT Parser (Vertical Slice)
**Goal**: End-to-end proof of concept with ONE platform

1. **Input**: ChatGPT JSON export file
2. **Parse** → conversations.jsonl (with message hashes)
3. **Extract** → entities.jsonl (spaCy NER)
4. **Detect** → artifacts.jsonl (code blocks)
5. **Output**: 3 clean JSON files
6. **Test**: Verify all ChatGPT conversations extracted correctly

**Success Criteria**:
- Can parse 1000 ChatGPT messages
- Extract people/org entities with >80% accuracy
- Detect all code blocks
- No crashes, all errors logged

### Sprint 2: Add Claude Parser
1. Handle Claude JSON format
2. Handle Claude Markdown exports (browser extensions)
3. Merge into same output files
4. Test deduplication (same message from different exports)

### Sprint 3: Add Gemini, Perplexity, Qwen
1. Gemini JSON parser
2. Perplexity Markdown/HTML parser
3. Qwen JSON parser
4. Unified testing across all platforms

### Sprint 4: Enhanced Extraction
1. Event detection (milestones, decisions)
2. Document classification (strategy, legal)
3. Entity relationship mapping (basic)
4. Confidence scoring on all extractions

### Sprint 5: LLM Analysis Integration
1. Gemini API for advanced extraction
2. Batch processing for efficiency
3. Only on flagged conversations (10-20%)
4. Token usage tracking and optimization

---

## Technology Stack

### Required Libraries (Gemini will use these)
```python
# Core
import json
import hashlib
from pathlib import Path
from datetime import datetime

# NLP - spaCy (NOT NLTK for NER)
import spacy
nlp = spacy.load("en_core_web_sm")

# LLM
import google.generativeai as genai  # Gemini API

# Utilities
import re
from typing import Dict, List, Optional
from dataclasses import dataclass
from pydantic import BaseModel  # For schema validation
```

### Deferred/Not Needed for v1
- ❌ Tesseract OCR (no PDF processing in v1)
- ❌ NLTK (using spaCy instead for NER)
- ❌ Database libraries (not this project's scope)
- ❌ Elasticsearch (future maybe)

---

## Opus's Recommended Build Order

1. ✅ **Unified schema definition** (Pydantic models) - START HERE
2. ✅ **ChatGPT JSON parser** - Prove the concept
3. ✅ **Message hashing** - Deduplication from day one
4. ✅ **spaCy entity extraction** - People, orgs, dates
5. ✅ **Code block detector** - Regex-based
6. ✅ **Output to JSONL files** - conversations, entities, artifacts
7. Add Claude parser
8. Add other platforms
9. Add LLM analysis (Gemini API)
10. Add event/document extraction

---

## Explicitly Deferred to v2 or Separate Projects

- ❌ **PDF/OCR Processing** - Complexity trap
- ❌ **Stirling PDF Debugging** - Separate project
- ❌ **Database Schema Design** - Separate ingestion agent
- ❌ **Query Interface** - Post-ingestion feature
- ❌ **Web UI** - CLI is sufficient for v1
- ❌ **Advanced Entity Linking** - Use simple fuzzy matching in v1

---

## Success Metrics for v1

1. ✅ Parse 10K messages from ChatGPT without crashing
2. ✅ Extract entities with >80% precision (manual spot-check)
3. ✅ Detect 95%+ of code blocks
4. ✅ Hash-based deduplication catches duplicate exports
5. ✅ Process 1000 messages/minute
6. ✅ Output valid JSON that validates against schema
7. ✅ Use <$1 in Gemini API tokens for 10K message processing

---

## Ready for Gemini

**Gemini's Task**: Code the parser/tagger system according to this plan.

**Starting Point**: Sprint 1 - ChatGPT JSON parser
- Input: ChatGPT export (JSON format)
- Output: conversations.jsonl, entities.jsonl, artifacts.jsonl
- Use spaCy for NER
- Use SHA256 for message hashing
- No database code - just output clean JSON

**Constraints**:
- Python 3.10+
- Use spaCy, not NLTK
- Output must validate against defined schemas
- Log all errors, never crash silently
- Process incrementally (don't load entire export into memory)

---

## Next Step

Hand this plan to Gemini with instructions:
"Code Sprint 1: ChatGPT JSON parser that outputs conversations.jsonl, entities.jsonl, and artifacts.jsonl. Use spaCy for entity extraction, SHA256 for message hashing. Follow the schemas defined above."

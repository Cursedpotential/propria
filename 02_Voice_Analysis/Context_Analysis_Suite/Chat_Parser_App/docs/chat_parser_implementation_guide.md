# Chat History Parser - Complete Implementation Guide

**Version**: 2.0
**Date**: 2025-12-21
**For**: Gemini Development Agent

---

## TABLE OF CONTENTS

1. [Project Overview](#1-project-overview)
2. [Architecture](#2-architecture)
3. [Project Structure](#3-project-structure)
4. [Development Setup](#4-development-setup)
5. [Implementation Phases](#5-implementation-phases)
6. [Core Components](#6-core-components)
7. [Testing Strategy](#7-testing-strategy)
8. [Security Implementation](#8-security-implementation)
9. [Deployment](#9-deployment)
10. [References](#10-references)

---

## 1. PROJECT OVERVIEW

### Purpose
Parse chat history exports from multiple platforms (ChatGPT, Claude, Gemini, Perplexity, Qwen) into structured, queryable data for legal analysis.

### Key Requirements
- ✅ **Keep PII** (phone numbers, emails needed for legal case)
- ✅ **Self-creating parsers** (formats vary within platforms)
- ✅ **Local processing** (no cloud uploads)
- ✅ **Hybrid storage** (Redis + PostgreSQL + PGVector)
- ✅ **90% Python, 10% LLM** (cost optimization)

### Success Criteria
- Parse 10K messages/sec with Python components
- <$1 cost per 10K messages total
- 100% deduplication accuracy
- Human-verified parser schemas before full processing

---

## 2. ARCHITECTURE

### 2.1 System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                        INPUT LAYER                                  │
│  Platform Exports: ChatGPT, Claude, Gemini, Perplexity, Qwen       │
│  Formats: JSON (single/array), JSONL, Markdown, HTML               │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                  FORMAT DISCOVERY LAYER                             │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐                    │
│  │ Extract    │→ │ Generate   │→ │ Check DB   │                    │
│  │ 100 lines  │  │ Signature  │  │ for Match  │                    │
│  └────────────┘  └────────────┘  └─────┬──────┘                    │
│                                         │                           │
│                     ┌───────────────────┴────────────┐              │
│                     │                                │              │
│                   MATCH                            NO MATCH         │
│                     │                                │              │
│                     ▼                                ▼              │
│            ┌─────────────────┐            ┌─────────────────┐      │
│            │ Load Existing   │            │ LLM Analysis    │      │
│            │ Parser Schema   │            │ + Schema Gen    │      │
│            └────────┬────────┘            └────────┬────────┘      │
│                     │                               │              │
│                     │                               ▼              │
│                     │                      ┌─────────────────┐     │
│                     │                      │ Human Verify &  │     │
│                     │                      │ Approve Schema  │     │
│                     │                      └────────┬────────┘     │
│                     │                               │              │
│                     └───────────┬───────────────────┘              │
│                                 │                                  │
└─────────────────────────────────┼──────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    PARSING LAYER (Python)                           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │
│  │  Streaming   │→ │  Apply       │→ │  Normalize   │              │
│  │  Reader      │  │  Schema      │  │  + Hash      │              │
│  └──────────────┘  └──────────────┘  └──────┬───────┘              │
└─────────────────────────────────────────────┼──────────────────────┘
                                              │
                                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    PROCESSING LAYER (Python + Redis)                │
│                                                                     │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐    │
│  │  spaCy NER      │  │  Regex Patterns │  │  Code Detection │    │
│  │  (entities)     │  │  (legal, etc)   │  │  (artifacts)    │    │
│  └────────┬────────┘  └────────┬────────┘  └────────┬────────┘    │
│           │                    │                     │             │
│           └────────────────────┴─────────────────────┘             │
│                                │                                   │
│                                ▼                                   │
│                      ┌──────────────────┐                          │
│                      │  Redis Buffer    │                          │
│                      │  - Hash dedup    │                          │
│                      │  - Entity temp   │                          │
│                      │  - Checkpoints   │                          │
│                      └────────┬─────────┘                          │
└─────────────────────────────────┼──────────────────────────────────┘
                                  │
                                  ▼ (Flag 10-20% for LLM)
┌─────────────────────────────────────────────────────────────────────┐
│                 SELECTIVE LLM LAYER (Gemini API)                    │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │
│  │  Strategy    │  │  Legal Doc   │  │  Events &    │              │
│  │  Extract     │  │  Classify    │  │  Sentiment   │              │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘              │
│         │                  │                  │                     │
│         └──────────────────┴──────────────────┘                     │
│                            │                                        │
│                   (Batch 50 at a time)                              │
└─────────────────────────────────┬───────────────────────────────────┘
                                  │
                                  ▼ (Batch flush)
┌─────────────────────────────────────────────────────────────────────┐
│              STORAGE LAYER (PostgreSQL + PGVector)                  │
│                                                                     │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐    │
│  │ conversations   │  │ messages        │  │ entities        │    │
│  │ parser_schemas  │  │ events          │  │ artifacts       │    │
│  │ documents       │  │ audit_log       │  │ entity_mentions │    │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘    │
│                                                                     │
│  Features: Vector similarity, encrypted PII, audit logging          │
└─────────────────────────────────────────────────────────────────────┘
```

### 2.2 Data Flow Summary

```
Input File → Format Discovery → Parser Selection/Creation →
Stream Parse (1000 msg chunks) → Python Extraction (spaCy/regex) →
Redis Buffer → Flag 10-20% for LLM → Gemini Batch Analysis →
PostgreSQL Storage → Queryable Output
```

### 2.3 Technology Stack

| Layer | Technology | Purpose |
|-------|-----------|---------|
| **Language** | Python 3.11+ | Core implementation |
| **NLP** | spaCy (en_core_web_lg) | Entity extraction |
| **LLM** | Gemini 2.0 Flash | Schema discovery, classification |
| **Hot Cache** | Redis 7+ | Dedup, rate limits, buffers |
| **Database** | PostgreSQL 16 + PGVector | Persistent storage, vector search |
| **Encryption** | cryptography (Fernet) | PII encryption at rest |
| **Testing** | pytest, pytest-cov | Unit & integration tests |
| **Package Mgmt** | Poetry | Dependency management |
| **Linting** | Ruff | Fast linting (replaces flake8, isort) |
| **Formatting** | Black | Code formatting |
| **Type Checking** | mypy | Static type checking |

---

## 3. PROJECT STRUCTURE

### 3.1 Directory Layout

```
chat-parser/
├── .github/
│   └── workflows/
│       ├── ci.yml                    # CI/CD pipeline
│       └── release.yml               # Release automation
├── src/
│   └── chat_parser/
│       ├── __init__.py
│       ├── cli.py                    # Entry point
│       ├── core/
│       │   ├── __init__.py
│       │   ├── config.py             # Configuration management
│       │   ├── exceptions.py         # Custom exceptions
│       │   └── logging_config.py     # Structured logging
│       ├── discovery/
│       │   ├── __init__.py
│       │   ├── format_detector.py    # Sample extraction & signature
│       │   ├── llm_analyzer.py       # LLM schema generation
│       │   ├── schema_validator.py   # Human verification
│       │   └── models.py             # ParserSchema Pydantic models
│       ├── parsers/
│       │   ├── __init__.py
│       │   ├── base.py               # Abstract parser interface
│       │   ├── json_parser.py        # JSON/JSONL parser
│       │   ├── markdown_parser.py    # Markdown parser
│       │   ├── html_parser.py        # HTML parser
│       │   └── schema_parser.py      # Dynamic schema-driven parser
│       ├── processors/
│       │   ├── __init__.py
│       │   ├── entity_extractor.py   # spaCy NER
│       │   ├── artifact_detector.py  # Code block extraction
│       │   ├── event_detector.py     # Event/timeline extraction
│       │   └── document_classifier.py # Strategy/legal docs
│       ├── storage/
│       │   ├── __init__.py
│       │   ├── redis_client.py       # Redis operations
│       │   ├── postgres_client.py    # PostgreSQL operations
│       │   ├── models.py             # SQLAlchemy models
│       │   └── encryption.py         # PII encryption
│       ├── llm/
│       │   ├── __init__.py
│       │   ├── gemini_client.py      # Gemini API wrapper
│       │   ├── batch_processor.py    # Batch LLM calls
│       │   └── prompts.py            # LLM prompt templates
│       └── utils/
│           ├── __init__.py
│           ├── hashing.py            # SHA256 deduplication
│           ├── streaming.py          # Incremental file reading
│           └── validation.py         # Data validation
├── tests/
│   ├── __init__.py
│   ├── unit/
│   │   ├── test_parsers.py
│   │   ├── test_processors.py
│   │   ├── test_discovery.py
│   │   └── test_storage.py
│   ├── integration/
│   │   ├── test_end_to_end.py
│   │   └── test_database.py
│   └── fixtures/
│       ├── sample_chatgpt.json
│       ├── sample_claude.md
│       └── sample_gemini.jsonl
├── scripts/
│   ├── setup_db.sh                   # Initialize PostgreSQL schema
│   ├── setup_redis.sh                # Configure Redis
│   └── run_tests.sh                  # Test runner
├── docs/
│   ├── architecture.md
│   ├── schema_format.md
│   └── deployment.md
├── pyproject.toml                    # Poetry configuration
├── README.md
├── .env.example                      # Environment template
├── .gitignore
└── LICENSE
```

### 3.2 Key Configuration Files

**pyproject.toml** (Poetry + dependencies):
```toml
[tool.poetry]
name = "chat-parser"
version = "1.0.0"
description = "Multi-platform chat history parser with LLM-assisted schema discovery"
authors = ["Your Name"]
readme = "README.md"
packages = [{include = "chat_parser", from = "src"}]

[tool.poetry.dependencies]
python = "^3.11"
spacy = "^3.7"
pydantic = "^2.5"
redis = "^5.0"
psycopg2-binary = "^2.9"
sqlalchemy = "^2.0"
pgvector = "^0.2"
cryptography = "^42.0"
google-generativeai = "^0.3"
click = "^8.1"
python-dotenv = "^1.0"
structlog = "^24.0"

[tool.poetry.group.dev.dependencies]
pytest = "^7.4"
pytest-cov = "^4.1"
pytest-asyncio = "^0.21"
ruff = "^0.1"
black = "^23.12"
mypy = "^1.7"
pre-commit = "^3.6"

[tool.poetry.scripts]
chat-parser = "chat_parser.cli:main"

[tool.ruff]
line-length = 100
select = ["E", "F", "I", "N", "W", "UP"]
ignore = []

[tool.black]
line-length = 100
target-version = ['py311']

[tool.mypy]
python_version = "3.11"
strict = true
warn_return_any = true
warn_unused_configs = true

[tool.pytest.ini_options]
testpaths = ["tests"]
python_files = "test_*.py"
python_functions = "test_*"
addopts = "--cov=src/chat_parser --cov-report=html --cov-report=term"

[build-system]
requires = ["poetry-core"]
build-backend = "poetry.core.masonry.api"
```

**.env.example**:
```bash
# Gemini API
GEMINI_API_KEY=your_api_key_here

# Database
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=chat_parser
POSTGRES_USER=chat_processor
POSTGRES_PASSWORD=secure_password_here

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_DB=0

# Encryption
MASTER_PASSWORD=your_master_password_here
# Generate with: python -c "import os; print(os.urandom(16).hex())"
ENCRYPTION_SALT=your_hex_salt_here

# Processing
CHUNK_SIZE=1000
BATCH_SIZE=50
LLM_TIMEOUT=30

# Logging
LOG_LEVEL=INFO
LOG_FILE=chat_parser.log
```

---

## 4. DEVELOPMENT SETUP

### 4.1 Prerequisites

```bash
# Python 3.11+
python --version  # Should be 3.11 or higher

# PostgreSQL 16+
psql --version

# Redis 7+
redis-cli --version

# Poetry (https://python-poetry.org/docs/#installation)
curl -sSL https://install.python-poetry.org | python3 -
```

### 4.2 Initial Setup

```bash
# 1. Clone repository (or create new project)
mkdir chat-parser && cd chat-parser

# 2. Initialize Poetry
poetry init

# 3. Install dependencies
poetry install

# 4. Download spaCy model
poetry run python -m spacy download en_core_web_lg

# 5. Setup PostgreSQL
createdb chat_parser
poetry run python scripts/setup_db.py

# 6. Setup Redis (config in redis.conf)
redis-server --daemonize yes

# 7. Copy environment template
cp .env.example .env
# Edit .env with your credentials

# 8. Install pre-commit hooks
poetry run pre-commit install
```

### 4.3 Running Development Server

```bash
# Activate Poetry shell
poetry shell

# Run parser CLI
chat-parser --help

# Example: Process a file
chat-parser process input.json --platform chatgpt

# Run tests
pytest

# Run with coverage
pytest --cov

# Type checking
mypy src/

# Linting
ruff check src/

# Format code
black src/
```

---

## 5. IMPLEMENTATION PHASES

### Phase 1: Core Infrastructure (Week 1)

**Goal**: Set up project skeleton, database schemas, basic parsing

**Deliverables**:
1. ✅ Project structure with Poetry
2. ✅ PostgreSQL schema (tables: conversations, messages, parser_schemas)
3. ✅ Redis client with hash storage
4. ✅ Base parser interface
5. ✅ JSON parser implementation
6. ✅ Basic CLI with `process` command
7. ✅ Unit tests for parsers

**Acceptance Criteria**:
- Can parse ChatGPT JSON export
- Messages stored in PostgreSQL
- Hash dedup working in Redis
- 80%+ test coverage

### Phase 2: Format Discovery (Week 2)

**Goal**: Implement self-creating parser system

**Deliverables**:
1. ✅ Format signature generator
2. ✅ LLM analyzer (Gemini integration)
3. ✅ ParserSchema Pydantic models
4. ✅ Human verification CLI
5. ✅ Schema storage in PostgreSQL
6. ✅ Dynamic schema-driven parser
7. ✅ Integration tests

**Acceptance Criteria**:
- LLM can analyze 100 lines and generate schema
- Human can approve/reject schema via CLI
- Approved schema reused for matching formats
- Works with ChatGPT, Claude formats

### Phase 3: NLP Processing (Week 3)

**Goal**: Implement entity extraction, artifact detection

**Deliverables**:
1. ✅ spaCy NER integration (en_core_web_lg)
2. ✅ Entity deduplication & normalization
3. ✅ Code block detection (regex)
4. ✅ Entity storage with PGVector embeddings
5. ✅ Artifact storage
6. ✅ Unit tests for processors

**Acceptance Criteria**:
- Entities extracted with 90%+ accuracy
- Phone/email entities preserved
- Code blocks extracted correctly
- Vector similarity search working

### Phase 4: LLM Integration (Week 4)

**Goal**: Selective LLM analysis for events, documents

**Deliverables**:
1. ✅ Flagging logic (keyword-based)
2. ✅ Gemini batch processor (50 convs/call)
3. ✅ Event detection prompts
4. ✅ Document classification prompts
5. ✅ Response parsing & validation
6. ✅ Cost tracking

**Acceptance Criteria**:
- Only 10-20% of data sent to LLM
- Batch processing working
- <$1 per 10K messages cost
- Events/documents extracted

### Phase 5: Security & Audit (Week 5)

**Goal**: Implement encryption, access controls, audit logging

**Deliverables**:
1. ✅ Column-level encryption (phone, email)
2. ✅ Hash-based lookups
3. ✅ Audit log table (immutable)
4. ✅ File permission enforcement
5. ✅ Security documentation
6. ✅ Penetration testing

**Acceptance Criteria**:
- PII encrypted at rest
- All access logged
- File permissions 700
- Security audit passed

### Phase 6: Testing & Deployment (Week 6)

**Goal**: Comprehensive testing, CI/CD, documentation

**Deliverables**:
1. ✅ End-to-end integration tests
2. ✅ Performance benchmarks (10K msgs/sec)
3. ✅ CI/CD pipeline (GitHub Actions)
4. ✅ User documentation
5. ✅ Deployment scripts
6. ✅ Production readiness checklist

**Acceptance Criteria**:
- 90%+ code coverage
- All benchmarks met
- CI/CD green
- Documentation complete

---

## 6. CORE COMPONENTS

### 6.1 Format Discovery System

#### 6.1.1 Sample Extraction

```python
# src/chat_parser/discovery/format_detector.py

import hashlib
import re
from pathlib import Path
from typing import Tuple

def extract_sample(file_path: Path, line_count: int = 100) -> Tuple[str, dict]:
    """
    Extract first N lines/turns for format analysis.
    Uses smart boundary detection to avoid cutting mid-message.

    Returns:
        Tuple of (sample_text, metadata)
    """
    with open(file_path, 'r', encoding='utf-8') as f:
        content = f.read()

    # Detect format hints
    format_hints = {
        'is_json': content.strip().startswith(('{', '[')),
        'is_markdown': bool(re.search(r'^#{1,3}\s', content, re.MULTILINE)),
        'is_html': content.strip().startswith(('<', '<!DOCTYPE')),
        'encoding': 'utf-8',
        'file_size': len(content),
    }

    # Extract sample (smart boundary)
    lines = content.splitlines()
    if format_hints['is_json']:
        # For JSON, take first complete object
        sample = _extract_json_sample(content, line_count)
    else:
        # For text formats, take first N lines
        sample = '\n'.join(lines[:line_count])

    return sample, format_hints

def generate_format_signature(sample: str, platform_hint: str = None) -> str:
    """
    Generate structural signature (NOT content-based).
    Used to match against existing schemas.
    """
    features = []

    # Container format
    sample_stripped = sample.strip()
    if sample_stripped.startswith('{'):
        features.append("container:json_object")
    elif sample_stripped.startswith('['):
        features.append("container:json_array")
    elif re.search(r'^#{1,3}\s', sample, re.MULTILINE):
        features.append("container:markdown")
    # ... more patterns

    # Structural patterns (field names, prefixes)
    patterns = [
        (r'"role"\s*:', "field:role"),
        (r'"sender"\s*:', "field:sender"),
        (r'^Human:', "prefix:Human"),
        (r'^Assistant:', "prefix:Assistant"),
        # ... more patterns
    ]

    for pattern, feature in patterns:
        if re.search(pattern, sample, re.MULTILINE):
            features.append(feature)

    if platform_hint:
        features.append(f"platform:{platform_hint}")

    # Sort for consistency, hash
    features.sort()
    signature = hashlib.sha256('|'.join(features).encode()).hexdigest()[:16]
    return signature
```

#### 6.1.2 LLM Schema Analysis

```python
# src/chat_parser/llm/prompts.py

SCHEMA_ANALYSIS_PROMPT = """
Analyze this chat export sample and identify the format structure.

<sample>
{sample_content}
</sample>

Provide a JSON analysis with these EXACT fields:

{{
  "file_format": "json_single|json_array|jsonl|markdown|html|plaintext",
  "message_structure": "nested_object|line_delimited|header_delimited|timestamp_delimited",
  "role_extraction": {{
    "method": "json_path|regex|xpath",
    "pattern": "$.role|^(Human|Assistant):",
    "group": 1
  }},
  "role_mapping": {{
    "human": ["human", "user", "User"],
    "assistant": ["assistant", "claude", "AI"]
  }},
  "content_extraction": {{
    "method": "json_path|regex",
    "pattern": "$.content|$.text"
  }},
  "timestamp_extraction": {{
    "method": "json_path|regex|none",
    "pattern": "$.timestamp|^\\[(\\d{{4}}-\\d{{2}}-\\d{{2}})",
    "format": "%Y-%m-%dT%H:%M:%S.%fZ"
  }},
  "metadata": {{
    "conversation_title_path": "$.title",
    "conversation_id_path": "$.id",
    "platform_indicators": ["Claude", "OpenAI"]
  }},
  "edge_cases": [
    "Message roles sometimes missing",
    "Timestamps in multiple formats"
  ],
  "confidence_score": 0.95,
  "llm_explanation": "This appears to be a ChatGPT export in JSON format..."
}}

Return ONLY valid JSON. No markdown, no explanation outside the JSON.
"""
```

```python
# src/chat_parser/discovery/llm_analyzer.py

from dataclasses import dataclass
import google.generativeai as genai
from .models import ParserSchema
import json

@dataclass
class LLMAnalyzer:
    api_key: str
    model: str = "gemini-2.0-flash-exp"

    def __post_init__(self):
        genai.configure(api_key=self.api_key)
        self.client = genai.GenerativeModel(self.model)

    def analyze_format(self, sample: str, platform_hint: str = None) -> ParserSchema:
        """
        Send sample to LLM for structure analysis.
        Returns ParserSchema object.
        """
        prompt = SCHEMA_ANALYSIS_PROMPT.format(sample_content=sample)

        response = self.client.generate_content(
            prompt,
            generation_config=genai.GenerationConfig(
                temperature=0.1,  # Low for consistency
                response_mime_type="application/json"
            )
        )

        # Parse LLM response
        analysis = json.loads(response.text)

        # Convert to ParserSchema
        schema = ParserSchema(
            platform=platform_hint or "unknown",
            format_signature="",  # Will be set by caller
            **analysis
        )

        return schema
```

#### 6.1.3 Human Verification

```python
# src/chat_parser/discovery/schema_validator.py

from typing import List
from .models import ParserSchema, VerificationResult
from ..parsers.schema_parser import SchemaParser

class SchemaValidator:
    """Handles human verification of LLM-generated schemas"""

    def verify_schema(
        self,
        schema: ParserSchema,
        sample_content: str,
        job_id: str
    ) -> VerificationResult:
        """
        Test schema on sample, display to human, get approval.
        """
        # Test parse sample
        parser = SchemaParser(schema)
        test_results = parser.parse_sample(sample_content, max_messages=10)

        # Display verification report
        self._display_report(schema, test_results)

        # Get human decision
        decision = self._await_decision()

        if decision == 'yes':
            return VerificationResult(
                approved=True,
                schema=schema,
                approved_by=os.getenv('USER', 'unknown')
            )
        elif decision == 'no':
            feedback = input("Feedback (optional): ")
            return VerificationResult(
                approved=False,
                schema=schema,
                feedback=feedback
            )
        else:
            # Provide feedback for retry
            feedback = input("What needs to change? ")
            return VerificationResult(
                approved=False,
                schema=schema,
                feedback=feedback,
                retry=True
            )

    def _display_report(self, schema: ParserSchema, test_results):
        """Format and display verification report"""
        print("\n" + "="*70)
        print("PARSER SCHEMA VERIFICATION")
        print("="*70)
        print(f"\nPlatform: {schema.platform}")
        print(f"File Format: {schema.file_format}")
        print(f"LLM Confidence: {schema.confidence_score:.1%}")
        print(f"\n--- LLM ANALYSIS ---")
        print(schema.llm_explanation)
        print(f"\n--- TEST PARSE RESULTS ---")
        print(f"Messages Parsed: {len(test_results.messages)}")
        print(f"Parse Failures: {test_results.failure_count}")

        print(f"\n--- SAMPLE MESSAGES ---")
        for i, msg in enumerate(test_results.messages[:5], 1):
            print(f"\n[Message {i}]")
            print(f"  Role: {msg.role}")
            print(f"  Timestamp: {msg.timestamp}")
            print(f"  Content: {msg.content[:150]}...")

        if test_results.warnings:
            print(f"\n--- WARNINGS ---")
            for w in test_results.warnings:
                print(f"  ⚠️  {w}")

        print("\n" + "="*70)

    def _await_decision(self) -> str:
        """Get human approval decision"""
        while True:
            response = input("\nAPPROVE THIS PARSER? (yes/no/retry): ").strip().lower()
            if response in ('yes', 'no', 'retry'):
                return response
            print("Please enter 'yes', 'no', or 'retry'")
```

### 6.2 Pydantic Models

```python
# src/chat_parser/discovery/models.py

from pydantic import BaseModel, Field
from typing import Optional, List, Dict, Any
from enum import Enum
from datetime import datetime

class FileFormat(str, Enum):
    JSON_SINGLE = "json_single"
    JSON_ARRAY = "json_array"
    JSONL = "jsonl"
    MARKDOWN = "markdown"
    HTML = "html"

class MessageStructure(str, Enum):
    NESTED_OBJECT = "nested_object"
    LINE_DELIMITED = "line_delimited"
    HEADER_DELIMITED = "header_delimited"

class ExtractionRule(BaseModel):
    method: str  # "json_path", "regex", "xpath"
    pattern: str
    group: Optional[int] = None
    default: Optional[str] = None
    transform: Optional[str] = None

class ParserSchema(BaseModel):
    """Complete schema for parsing a chat export"""

    # Identity
    schema_id: Optional[str] = None
    platform: str
    format_signature: str
    version: int = 1

    # Structure
    file_format: FileFormat
    message_structure: MessageStructure

    # Extraction rules
    role_extraction: ExtractionRule
    role_mapping: Dict[str, List[str]]
    content_extraction: ExtractionRule
    timestamp_extraction: Optional[ExtractionRule] = None
    timestamp_format: Optional[str] = None

    # Metadata
    conversation_title_extraction: Optional[ExtractionRule] = None
    conversation_id_extraction: Optional[ExtractionRule] = None

    # Patterns
    code_block_pattern: str = r"```(\w+)?\n([\s\S]*?)```"

    # Validation
    required_fields: List[str] = ["role", "content"]

    # LLM metadata
    llm_explanation: Optional[str] = None
    edge_cases: List[str] = []
    confidence_score: Optional[float] = None

    # Approval
    approved: bool = False
    approved_by: Optional[str] = None
    approved_at: Optional[datetime] = None

class VerificationResult(BaseModel):
    approved: bool
    schema: ParserSchema
    feedback: Optional[str] = None
    approved_by: Optional[str] = None
    retry: bool = False
```

### 6.3 PostgreSQL Schema

```sql
-- scripts/schema.sql

-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Parser schemas table
CREATE TABLE parser_schemas (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    platform VARCHAR(50) NOT NULL,
    format_signature VARCHAR(64) NOT NULL,
    schema_definition JSONB NOT NULL,
    sample_input TEXT,
    llm_explanation TEXT,
    edge_cases TEXT[],
    confidence_score FLOAT,
    version INTEGER DEFAULT 1,
    approved BOOLEAN DEFAULT FALSE,
    approved_by VARCHAR(100),
    approved_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(platform, format_signature, version)
);
CREATE INDEX parser_schemas_signature_idx ON parser_schemas(format_signature);
CREATE INDEX parser_schemas_platform_idx ON parser_schemas(platform);

-- Conversations table
CREATE TABLE conversations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    platform VARCHAR(50) NOT NULL,
    title TEXT,
    external_id VARCHAR(255),
    source_file VARCHAR(500) NOT NULL,
    parser_schema_id UUID REFERENCES parser_schemas(id),
    parsed_at TIMESTAMP DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'
);
CREATE INDEX conversations_platform_idx ON conversations(platform);
CREATE INDEX conversations_source_file_idx ON conversations(source_file);

-- Messages table
CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL,
    content TEXT NOT NULL,
    content_hash VARCHAR(64) NOT NULL,
    timestamp TIMESTAMP,
    message_index INTEGER,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(content_hash, conversation_id)
);
CREATE INDEX messages_conversation_idx ON messages(conversation_id);
CREATE INDEX messages_hash_idx ON messages(content_hash);
CREATE INDEX messages_timestamp_idx ON messages(timestamp);

-- Entities table (with encryption)
CREATE TABLE entities (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(500) NOT NULL,
    canonical_name VARCHAR(500),
    entity_type VARCHAR(50) NOT NULL,

    -- Encrypted PII
    phone_encrypted TEXT,
    email_encrypted TEXT,

    -- Searchable hashes
    phone_hash VARCHAR(64),
    email_hash VARCHAR(64),

    -- Vector embedding
    embedding vector(384),

    metadata JSONB DEFAULT '{}',
    first_seen TIMESTAMP DEFAULT NOW(),
    last_seen TIMESTAMP DEFAULT NOW(),
    mention_count INTEGER DEFAULT 1
);
CREATE INDEX entities_type_idx ON entities(entity_type);
CREATE INDEX entities_canonical_idx ON entities(canonical_name);
CREATE INDEX entities_phone_hash_idx ON entities(phone_hash);
CREATE INDEX entities_email_hash_idx ON entities(email_hash);
CREATE INDEX entities_embedding_idx ON entities USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- Entity mentions (link entities to messages)
CREATE TABLE entity_mentions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    entity_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    span_start INTEGER,
    span_end INTEGER,
    context TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);
CREATE INDEX entity_mentions_entity_idx ON entity_mentions(entity_id);
CREATE INDEX entity_mentions_message_idx ON entity_mentions(message_id);

-- Artifacts table
CREATE TABLE artifacts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    artifact_type VARCHAR(50) NOT NULL,
    language VARCHAR(50),
    content TEXT NOT NULL,
    content_hash VARCHAR(64) NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(content_hash)
);
CREATE INDEX artifacts_message_idx ON artifacts(message_id);
CREATE INDEX artifacts_type_idx ON artifacts(artifact_type);

-- Events table
CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    conversation_id UUID REFERENCES conversations(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL,
    temporal_type VARCHAR(50) NOT NULL,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    event_date TIMESTAMP,
    sentiment_polarity FLOAT,
    sentiment_subjectivity FLOAT,
    sentiment_emotion VARCHAR(50),
    entities UUID[],
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW()
);
CREATE INDEX events_conversation_idx ON events(conversation_id);
CREATE INDEX events_type_idx ON events(event_type);
CREATE INDEX events_temporal_idx ON events(temporal_type);

-- Documents table
CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    conversation_id UUID REFERENCES conversations(id) ON DELETE CASCADE,
    document_type VARCHAR(50) NOT NULL,
    title VARCHAR(500),
    summary TEXT,
    entities UUID[],
    source_messages UUID[],
    confidence FLOAT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW()
);
CREATE INDEX documents_conversation_idx ON documents(conversation_id);
CREATE INDEX documents_type_idx ON documents(document_type);

-- Audit log (immutable)
CREATE TABLE audit_log (
    id BIGSERIAL PRIMARY KEY,
    timestamp TIMESTAMP DEFAULT NOW(),
    event_type VARCHAR(50) NOT NULL,
    actor VARCHAR(255) NOT NULL,
    resource_type VARCHAR(50),
    resource_id VARCHAR(255),
    resource_hash VARCHAR(64),
    action_details JSONB,
    ip_address INET,
    session_id VARCHAR(100),
    CONSTRAINT audit_log_immutable CHECK (TRUE)
);
REVOKE UPDATE, DELETE ON audit_log FROM PUBLIC;
CREATE INDEX audit_log_timestamp_idx ON audit_log(timestamp);
CREATE INDEX audit_log_actor_idx ON audit_log(actor);
CREATE INDEX audit_log_event_type_idx ON audit_log(event_type);

-- Database users
CREATE ROLE chat_processor_role;
GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA public TO chat_processor_role;
GRANT DELETE ON entity_mentions TO chat_processor_role;

CREATE ROLE chat_reader_role;
GRANT SELECT ON conversations, messages TO chat_reader_role;
GRANT SELECT (id, name, entity_type, metadata) ON entities TO chat_reader_role;
```

### 6.4 Redis Keys Structure

```python
# src/chat_parser/storage/redis_client.py

from redis import Redis
from typing import Set, Optional
import json

class RedisClient:
    """Redis operations for caching and deduplication"""

    KEY_PATTERNS = {
        "message_hashes": "chat:hashes:{conversation_id}",
        "llm_cache": "llm:response:{prompt_hash}",
        "rate_limit": "ratelimit:gemini:{minute}",
        "parser_state": "parser:state:{job_id}",
        "checkpoint": "checkpoint:{job_id}",
        "entity_buffer": "entity:buffer:{job_id}",
    }

    def __init__(self, host='localhost', port=6379, db=0):
        self.client = Redis(host=host, port=port, db=db, decode_responses=True)

    def check_message_hash(self, conversation_id: str, content_hash: str) -> bool:
        """Check if message hash already seen"""
        key = self.KEY_PATTERNS["message_hashes"].format(conversation_id=conversation_id)
        return self.client.sismember(key, content_hash)

    def add_message_hash(self, conversation_id: str, content_hash: str):
        """Add message hash to dedup set"""
        key = self.KEY_PATTERNS["message_hashes"].format(conversation_id=conversation_id)
        self.client.sadd(key, content_hash)

    def cache_llm_response(self, prompt_hash: str, response: str, ttl: int = 86400):
        """Cache LLM response for 24 hours"""
        key = self.KEY_PATTERNS["llm_cache"].format(prompt_hash=prompt_hash)
        self.client.setex(key, ttl, response)

    def get_llm_cache(self, prompt_hash: str) -> Optional[str]:
        """Retrieve cached LLM response"""
        key = self.KEY_PATTERNS["llm_cache"].format(prompt_hash=prompt_hash)
        return self.client.get(key)

    def save_checkpoint(self, job_id: str, state: dict):
        """Save processing checkpoint"""
        key = self.KEY_PATTERNS["checkpoint"].format(job_id=job_id)
        self.client.set(key, json.dumps(state))

    def load_checkpoint(self, job_id: str) -> Optional[dict]:
        """Load processing checkpoint"""
        key = self.KEY_PATTERNS["checkpoint"].format(job_id=job_id)
        data = self.client.get(key)
        return json.loads(data) if data else None
```

---

## 7. TESTING STRATEGY

### 7.1 Testing Pyramid

```
              ┌─────────────────┐
              │   E2E Tests     │  < 10%
              │  (Integration)  │
              └─────────────────┘
            ┌─────────────────────┐
            │  Integration Tests  │  ~ 20%
            │  (DB, API, Files)   │
            └─────────────────────┘
        ┌───────────────────────────┐
        │      Unit Tests           │  ~ 70%
        │  (Functions, Classes)     │
        └───────────────────────────┘
```

### 7.2 Unit Tests

```python
# tests/unit/test_format_detector.py

import pytest
from chat_parser.discovery.format_detector import (
    extract_sample,
    generate_format_signature
)

def test_extract_sample_json():
    """Test JSON sample extraction"""
    content = '{"messages": [{"role": "user", "content": "test"}]}'
    with open('test.json', 'w') as f:
        f.write(content)

    sample, hints = extract_sample(Path('test.json'), line_count=10)

    assert hints['is_json'] is True
    assert '"messages"' in sample
    assert hints['file_size'] == len(content)

def test_generate_format_signature_consistency():
    """Same structure should produce same signature"""
    sample1 = '{"role": "user", "content": "hello"}'
    sample2 = '{"role": "assistant", "content": "hi"}'

    sig1 = generate_format_signature(sample1)
    sig2 = generate_format_signature(sample2)

    assert sig1 == sig2  # Same structure despite different content

@pytest.mark.parametrize("sample,expected_features", [
    ('{"role": "user"}', ["container:json_object", "field:role"]),
    ('[{"sender": "human"}]', ["container:json_array", "field:sender"]),
    ('## Human:\nHello', ["container:markdown", "prefix:Human"]),
])
def test_signature_features(sample, expected_features):
    """Test feature detection"""
    sig = generate_format_signature(sample)
    # Signature should be deterministic
    assert len(sig) == 16
    assert sig.isalnum()
```

### 7.3 Integration Tests

```python
# tests/integration/test_end_to_end.py

import pytest
from pathlib import Path
from chat_parser.cli import process_file
from chat_parser.storage.postgres_client import PostgresClient

@pytest.fixture
def db():
    """Test database connection"""
    client = PostgresClient(database="chat_parser_test")
    yield client
    # Cleanup
    client.execute("TRUNCATE conversations, messages, entities CASCADE")

@pytest.fixture
def redis():
    """Test Redis connection"""
    from chat_parser.storage.redis_client import RedisClient
    client = RedisClient(db=15)  # Use separate test DB
    yield client
    client.client.flushdb()

def test_chatgpt_end_to_end(db, redis, tmp_path):
    """Test complete ChatGPT parsing pipeline"""
    # Setup: Create sample ChatGPT export
    input_file = tmp_path / "chatgpt_sample.json"
    input_file.write_text('''
    {
      "title": "Test Conversation",
      "mapping": {
        "msg1": {
          "message": {
            "author": {"role": "user"},
            "content": {"parts": ["Hello"]},
            "create_time": 1234567890
          }
        },
        "msg2": {
          "message": {
            "author": {"role": "assistant"},
            "content": {"parts": ["Hi there!"]},
            "create_time": 1234567891
          }
        }
      }
    }
    ''')

    # Execute: Process file
    result = process_file(
        file_path=input_file,
        platform="chatgpt",
        db_client=db,
        redis_client=redis
    )

    # Verify: Check database
    conversations = db.query("SELECT * FROM conversations")
    assert len(conversations) == 1
    assert conversations[0]['platform'] == 'chatgpt'

    messages = db.query("SELECT * FROM messages ORDER BY message_index")
    assert len(messages) == 2
    assert messages[0]['role'] == 'user'
    assert messages[0]['content'] == 'Hello'
    assert messages[1]['role'] == 'assistant'

    # Verify: Check Redis
    conv_id = conversations[0]['id']
    hash_key = f"chat:hashes:{conv_id}"
    hash_count = redis.client.scard(hash_key)
    assert hash_count == 2  # Both message hashes stored
```

### 7.4 Test Coverage Requirements

```bash
# Minimum coverage thresholds
pytest --cov=src/chat_parser --cov-fail-under=80

# Coverage by module:
# - discovery/: 90%+
# - parsers/: 85%+
# - processors/: 80%+
# - storage/: 85%+
# - llm/: 75%+
```

---

## 8. SECURITY IMPLEMENTATION

### 8.1 Encryption

```python
# src/chat_parser/storage/encryption.py

from cryptography.fernet import Fernet
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.primitives.kdf.pbkdf2 import PBKDF2HMAC
import base64
import os
import hashlib

class PIIEncryption:
    """Column-level encryption for PII fields"""

    def __init__(self, master_password: str, salt: bytes = None):
        self.salt = salt or bytes.fromhex(os.getenv('ENCRYPTION_SALT'))
        self.fernet = self._derive_key(master_password)

    def _derive_key(self, password: str) -> Fernet:
        """Derive encryption key from password using PBKDF2"""
        kdf = PBKDF2HMAC(
            algorithm=hashes.SHA256(),
            length=32,
            salt=self.salt,
            iterations=480000,  # OWASP recommended
        )
        key = base64.urlsafe_b64encode(kdf.derive(password.encode()))
        return Fernet(key)

    def encrypt_field(self, plaintext: str) -> str:
        """Encrypt PII field (phone, email)"""
        if not plaintext:
            return None
        return self.fernet.encrypt(plaintext.encode()).decode()

    def decrypt_field(self, ciphertext: str) -> str:
        """Decrypt PII field"""
        if not ciphertext:
            return None
        return self.fernet.decrypt(ciphertext.encode()).decode()

    @staticmethod
    def hash_for_lookup(value: str) -> str:
        """Generate searchable hash (one-way)"""
        normalized = value.lower().strip()
        return hashlib.sha256(normalized.encode()).hexdigest()

# Usage in entity storage
def store_entity_with_pii(
    name: str,
    entity_type: str,
    phone: str = None,
    email: str = None,
    encryption: PIIEncryption = None
):
    """Store entity with encrypted PII"""
    encrypted_phone = encryption.encrypt_field(phone) if phone else None
    phone_hash = PIIEncryption.hash_for_lookup(phone) if phone else None

    encrypted_email = encryption.encrypt_field(email) if email else None
    email_hash = PIIEncryption.hash_for_lookup(email) if email else None

    db.execute("""
        INSERT INTO entities (name, entity_type, phone_encrypted, phone_hash,
                             email_encrypted, email_hash)
        VALUES (%s, %s, %s, %s, %s, %s)
    """, (name, entity_type, encrypted_phone, phone_hash,
          encrypted_email, email_hash))
```

### 8.2 Audit Logging

```python
# src/chat_parser/core/audit.py

from enum import Enum
from typing import Optional, Dict, Any
import hashlib
import json
from datetime import datetime

class AuditEventType(str, Enum):
    FILE_READ = "file_read"
    FILE_WRITE = "file_write"
    DB_QUERY_PII = "db_query_pii"
    SCHEMA_APPROVED = "schema_approved"
    DECRYPTION = "decryption"
    ACCESS_DENIED = "access_denied"

class AuditLogger:
    """Immutable audit logging"""

    def __init__(self, db_client):
        self.db = db_client

    def log(
        self,
        event_type: AuditEventType,
        actor: str,
        resource_type: Optional[str] = None,
        resource_id: Optional[str] = None,
        content_for_hash: Optional[bytes] = None,
        details: Optional[Dict[str, Any]] = None
    ):
        """Log audit event (cannot be modified)"""
        resource_hash = None
        if content_for_hash:
            resource_hash = hashlib.sha256(content_for_hash).hexdigest()

        self.db.execute("""
            INSERT INTO audit_log
            (event_type, actor, resource_type, resource_id, resource_hash, action_details)
            VALUES (%s, %s, %s, %s, %s, %s)
        """, (
            event_type.value,
            actor,
            resource_type,
            resource_id,
            resource_hash,
            json.dumps(details) if details else None
        ))

# Usage in processing
def read_file_with_audit(file_path: str, audit: AuditLogger, actor: str) -> str:
    """Read file with audit trail"""
    with open(file_path, 'rb') as f:
        content = f.read()

    audit.log(
        event_type=AuditEventType.FILE_READ,
        actor=actor,
        resource_type="input_file",
        resource_id=str(file_path),
        content_for_hash=content,
        details={"size_bytes": len(content)}
    )

    return content.decode('utf-8')
```

---

## 9. DEPLOYMENT

### 9.1 CI/CD Pipeline

```yaml
# .github/workflows/ci.yml

name: CI

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest

    services:
      postgres:
        image: pgvector/pgvector:pg16
        env:
          POSTGRES_DB: chat_parser_test
          POSTGRES_PASSWORD: testpass
        ports:
          - 5432:5432
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

      redis:
        image: redis:7
        ports:
          - 6379:6379

    steps:
    - uses: actions/checkout@v3

    - name: Set up Python
      uses: actions/setup-python@v4
      with:
        python-version: '3.11'

    - name: Install Poetry
      run: |
        curl -sSL https://install.python-poetry.org | python3 -
        echo "$HOME/.local/bin" >> $GITHUB_PATH

    - name: Install dependencies
      run: |
        poetry install
        poetry run python -m spacy download en_core_web_lg

    - name: Run linters
      run: |
        poetry run ruff check src/
        poetry run black --check src/
        poetry run mypy src/

    - name: Run tests
      env:
        POSTGRES_HOST: localhost
        POSTGRES_DB: chat_parser_test
        POSTGRES_USER: postgres
        POSTGRES_PASSWORD: testpass
        REDIS_HOST: localhost
      run: |
        poetry run pytest --cov --cov-report=xml

    - name: Upload coverage
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.xml
```

### 9.2 Production Deployment Checklist

```markdown
## Pre-Deployment Checklist

### Environment
- [ ] Python 3.11+ installed
- [ ] PostgreSQL 16 with PGVector extension
- [ ] Redis 7+ configured
- [ ] Sufficient disk space (estimate: 10x input file size)
- [ ] SSL certificates for PostgreSQL

### Security
- [ ] Master password set (not in code/config)
- [ ] Encryption salt generated and stored securely
- [ ] Database users created with minimal permissions
- [ ] File permissions set (700 for processing dirs)
- [ ] Redis bound to localhost only
- [ ] PostgreSQL SSL required
- [ ] Firewall rules configured

### Configuration
- [ ] .env file created with production values
- [ ] Gemini API key configured
- [ ] Database connection tested
- [ ] Redis connection tested
- [ ] spaCy model downloaded

### Testing
- [ ] All tests passing (pytest)
- [ ] Code coverage ≥80%
- [ ] Type checking passing (mypy)
- [ ] Linting passing (ruff)
- [ ] Integration tests on sample data

### Data Migration
- [ ] Database schema created (schema.sql)
- [ ] Indexes created
- [ ] Audit log table verified immutable
- [ ] Backup strategy established

### Monitoring
- [ ] Logging configured
- [ ] Error tracking enabled
- [ ] Performance metrics baseline established
```

---

## 10. REFERENCES

### 10.1 Documentation Sources

- [spaCy Pipeline Architecture](https://github.com/explosion/spacy/blob/master/website/docs/usage/processing-pipelines.mdx)
- [ETL Pipeline Best Practices - dbt Labs](https://www.getdbt.com/blog/etl-pipeline-best-practices)
- [Data Pipeline Best Practices 2025 - Ascend.io](https://www.ascend.io/blog/data-pipeline-best-practices)
- [Python Best Practices 2025 - Nerd Level Tech](https://nerdleveltech.com/python-best-practices-the-2025-guide-for-clean-fast-and-secure-code)
- [Poetry Dependency Management - Real Python](https://realpython.com/dependency-management-python-poetry/)
- [Testing Python with Poetry, Tox, Nox - DEV Community](https://dev.to/wallaceespindola/test-python-code-like-a-pro-with-poetry-tox-nox-and-cicd-1i6p)

### 10.2 Libraries & Tools

| Tool | Version | Purpose |
|------|---------|---------|
| Python | 3.11+ | Core language |
| spaCy | 3.7+ | NLP & NER |
| Pydantic | 2.5+ | Data validation |
| SQLAlchemy | 2.0+ | Database ORM |
| PGVector | 0.2+ | Vector similarity |
| Redis | 5.0+ | Caching |
| Poetry | 1.7+ | Package mgmt |
| pytest | 7.4+ | Testing |
| Ruff | 0.1+ | Linting |
| Black | 23.12+ | Formatting |

---

## APPENDIX A: SAMPLE CLI USAGE

```bash
# Process single file
chat-parser process \
  --input exports/chatgpt_2024.json \
  --platform chatgpt \
  --output-dir ./parsed

# Process with schema approval workflow
chat-parser process \
  --input exports/unknown_format.json \
  --auto-discover \
  --verify-schema  # Will prompt for human approval

# Batch process directory
chat-parser batch \
  --input-dir exports/ \
  --platform-map chatgpt:*.json,claude:*.md \
  --parallel 4

# Export to JSONL
chat-parser export \
  --conversation-id abc-123 \
  --format jsonl \
  --output conversation.jsonl

# Query entities
chat-parser query \
  --type entity \
  --filter "entity_type=PERSON" \
  --similar "John Smith" \
  --top-k 10
```

---

**END OF IMPLEMENTATION GUIDE**

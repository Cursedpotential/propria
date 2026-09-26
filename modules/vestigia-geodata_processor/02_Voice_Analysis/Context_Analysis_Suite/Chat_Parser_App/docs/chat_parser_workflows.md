# Chat History Parser - Complete Workflows & Diagrams

## 1. SYSTEM ARCHITECTURE OVERVIEW

```
┌─────────────────────────────────────────────────────────────────────┐
│                     INPUT LAYER                                     │
│  Platform Exports: ChatGPT, Claude, Gemini, Perplexity, Qwen       │
│  Formats: JSON, Markdown, PDF, HTML                                │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                  STAGE 1: FORMAT DETECTION                          │
│  Tool: Python (regex, magic bytes)                                 │
│  Output: format_type, platform_type                                │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                  STAGE 2: PARSING LAYER                             │
│                                                                     │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                │
│  │  ChatGPT    │  │   Claude    │  │   Gemini    │                │
│  │  JSON       │  │   JSON/MD   │  │   JSON      │                │
│  │  Parser     │  │   Parser    │  │   Parser    │                │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘                │
│         │                 │                 │                       │
│         └─────────────────┴─────────────────┘                       │
│                           │                                         │
│  Output: normalized conversations + message_hash                   │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│              STAGE 3: LOCAL PREPROCESSING (Python)                  │
│                                                                     │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐ │
│  │  spaCy NER       │  │  Regex Patterns  │  │  Code Detector   │ │
│  │  (entities)      │  │  (legal, plan)   │  │  (artifacts)     │ │
│  └────────┬─────────┘  └────────┬─────────┘  └────────┬─────────┘ │
│           │                     │                     │            │
│           └─────────────────────┴─────────────────────┘            │
│                                 │                                  │
│  Output: entities.jsonl + artifacts.jsonl + flagged_convs.json    │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│         STAGE 4: SELECTIVE LLM ANALYSIS (Gemini API)                │
│         Only on flagged conversations (10-20%)                      │
│                                                                     │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐             │
│  │  Strategy    │  │  Legal Doc   │  │  Event       │             │
│  │  Extraction  │  │  Classify    │  │  Detection   │             │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘             │
│         │                  │                  │                    │
│         └──────────────────┴──────────────────┘                    │
│                            │                                       │
│  Output: documents.jsonl + events.jsonl + enhanced entities       │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                  STAGE 5: VALIDATION & OUTPUT                       │
│                                                                     │
│  Schema Validation (Pydantic) → Final JSONL Files:                 │
│  • conversations.jsonl                                             │
│  • entities.jsonl                                                  │
│  • events.jsonl                                                    │
│  • artifacts.jsonl                                                 │
│  • documents.jsonl                                                 │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 2. TOOL ASSIGNMENT DECISION TREE

```
                    ┌─────────────────────┐
                    │  Processing Task    │
                    └──────────┬──────────┘
                               │
                 ┌─────────────┴─────────────┐
                 │  Can pattern match solve? │
                 └──────────┬────────────────┘
                            │
              ┌─────────────┴─────────────┐
              │                           │
            YES                          NO
              │                           │
              ▼                           ▼
    ┌─────────────────────┐    ┌─────────────────────┐
    │  Use Python/Regex   │    │  Need context?      │
    │  Examples:          │    └──────┬──────────────┘
    │  • Code blocks      │           │
    │  • Message hashing  │     ┌─────┴─────┐
    │  • Timestamps       │    YES          NO
    │  • Basic NER        │     │            │
    └─────────────────────┘     ▼            ▼
                      ┌──────────────┐  ┌──────────────┐
                      │  Use LLM     │  │  Use spaCy   │
                      │  Examples:   │  │  Examples:   │
                      │  • Strategy  │  │  • NER       │
                      │  • Legal doc │  │  • POS tag   │
                      │  • Sentiment │  │  • Basic ent │
                      │  • Relations │  │              │
                      └──────────────┘  └──────────────┘
```

**Cost Optimization Rules**:
1. **Always Python first** - regex, patterns, structure
2. **spaCy for NER** - people, orgs, locations (free)
3. **LLM only when needed** - classification, relationships, context
4. **Batch LLM calls** - process 10-50 conversations at once
5. **Flag before LLM** - prefilter with Python to reduce LLM calls

---

## 3. ENTITY EXTRACTION WORKFLOW

```
Input: Conversation turns
         │
         ▼
┌──────────────────────────────────────┐
│  Step 1: spaCy NER Pipeline          │
│  Load: en_core_web_sm                │
│  Extract: PERSON, ORG, GPE, PRODUCT  │
│  Confidence: spaCy built-in scores   │
└────────────┬─────────────────────────┘
             │
             ▼
┌──────────────────────────────────────┐
│  Step 2: Pattern Enhancement         │
│  • Tech stack: regex (Python, React) │
│  • Projects: capital names pattern   │
│  • Concepts: quoted terms            │
└────────────┬─────────────────────────┘
             │
             ▼
┌──────────────────────────────────────┐
│  Step 3: Deduplication                │
│  • Normalize names (Bob → Robert)     │
│  • Fuzzy match aliases (90% sim)      │
│  • Merge duplicate entity_ids         │
└────────────┬─────────────────────────┘
             │
             ▼
┌──────────────────────────────────────┐
│  Step 4: Generate Entity Objects      │
│  Schema: entities.jsonl               │
│  Fields:                              │
│  • entity_id (UUID)                   │
│  • type (person|org|tech|...)         │
│  • name + aliases                     │
│  • confidence (0.0-1.0)               │
│  • first_mention (message_hash)       │
│  • mention_count                      │
│  • extraction_method (spacy|regex)    │
└────────────┬─────────────────────────┘
             │
             ▼
     entities.jsonl output
```

**Performance**:
- **spaCy NER**: ~1000 messages/sec
- **Regex patterns**: ~5000 messages/sec
- **No LLM calls** for basic entity extraction
- **LLM used later** for relationship mapping (Stage 4)

---

## 4. EVENT DETECTION WORKFLOW

```
Input: Conversation turns with entities
         │
         ▼
┌──────────────────────────────────────┐
│  Step 1: Pattern-Based Detection     │
│  Regex patterns for:                 │
│  • Temporal markers (yesterday, next)│
│  • Action verbs (decided, launched)  │
│  • Milestone keywords (completed)    │
│  Flag conversations with 2+ markers  │
└────────────┬─────────────────────────┘
             │
             ▼
┌──────────────────────────────────────┐
│  Step 2: Sentiment Analysis (NLTK)   │
│  VADER on flagged conversations:     │
│  • Polarity: -1.0 to 1.0             │
│  • Subjectivity: 0.0 to 1.0          │
│  • Emotion: pos|neg|neutral|mixed    │
└────────────┬─────────────────────────┘
             │
             ▼
┌──────────────────────────────────────┐
│  Step 3: LLM Context Extraction       │
│  Send flagged convs to Gemini:       │
│  Prompt: "Extract events with:"      │
│  • Type (milestone|decision|...)     │
│  • Temporal (historical|future)      │
│  • Title + description               │
│  • Entities involved                 │
│  • Confidence score                  │
└────────────┬─────────────────────────┘
             │
             ▼
┌──────────────────────────────────────┐
│  Step 4: Validation & Output          │
│  Pydantic schema validation          │
│  Merge sentiment from Step 2         │
│  Link to entity_ids from Step 3      │
└────────────┬─────────────────────────┘
             │
             ▼
      events.jsonl output
```

**Token Efficiency**:
- **Pattern flagging**: Reduce LLM calls by 80-90%
- **Only 10-20%** of conversations analyzed by LLM
- **Batch processing**: 10-50 flagged convs per API call

---

## 5. ARTIFACT EXTRACTION WORKFLOW

```
Input: Conversation turns
         │
         ▼
┌──────────────────────────────────────┐
│  Step 1: Code Block Detection        │
│  Regex patterns:                     │
│  • Markdown fences (```lang)         │
│  • Indented blocks (4 spaces)        │
│  • XML/HTML tags (<code>)            │
│  Extract: language, content          │
└────────────┬─────────────────────────┘
             │
             ▼
┌──────────────────────────────────────┐
│  Step 2: Language Detection           │
│  Priority order:                     │
│  1. Markdown language tag            │
│  2. Regex heuristics (import, def)   │
│  3. File extension if mentioned      │
│  Fallback: "unknown"                 │
└────────────┬─────────────────────────┘
             │
             ▼
┌──────────────────────────────────────┐
│  Step 3: Content Hashing              │
│  SHA256 hash of code content         │
│  Dedup: identical code blocks        │
│  Store: content_hash                 │
└────────────┬─────────────────────────┘
             │
             ▼
┌──────────────────────────────────────┐
│  Step 4: Context Extraction           │
│  Capture:                            │
│  • Surrounding message text          │
│  • Timestamp + platform              │
│  • Conversation ID                   │
│  • Source message hash               │
└────────────┬─────────────────────────┘
             │
             ▼
     artifacts.jsonl output
```

**100% Python** - No LLM needed for artifacts

---

## 6. DOCUMENT CLASSIFICATION WORKFLOW

```
Input: Conversation turns
         │
         ▼
┌──────────────────────────────────────┐
│  Step 1: Keyword Flagging (Python)   │
│  Strategy keywords:                  │
│  • plan, roadmap, propose, design    │
│  Legal keywords:                     │
│  • contract, terms, liability, NDA   │
│  Flag if 3+ keywords in 500 chars    │
└────────────┬─────────────────────────┘
             │
             ▼
┌──────────────────────────────────────┐
│  Step 2: Length Filter                │
│  Require:                            │
│  • 200+ words for strategy doc       │
│  • 100+ words for legal doc          │
│  Skip short messages                 │
└────────────┬─────────────────────────┘
             │
             ▼
┌──────────────────────────────────────┐
│  Step 3: LLM Classification           │
│  Send flagged convs to Gemini:       │
│  Prompt: "Classify document as:"     │
│  • Type: strategy|legal|technical    │
│  • Confidence: 0.0-1.0               │
│  • Key entities mentioned            │
│  • Summary (1-2 sentences)           │
└────────────┬─────────────────────────┘
             │
             ▼
┌──────────────────────────────────────┐
│  Step 4: Output Generation            │
│  Schema: documents.jsonl             │
│  Link to entity_ids                  │
│  Store source message hashes         │
└────────────┬─────────────────────────┘
             │
             ▼
     documents.jsonl output
```

**Efficiency**: Keyword filter reduces LLM calls by 95%

---

## 7. ERROR HANDLING & VALIDATION FLOW

```
                  ┌─────────────────┐
                  │  Parse Attempt  │
                  └────────┬────────┘
                           │
              ┌────────────┴────────────┐
              │  Success?               │
              └────┬──────────────┬─────┘
                   │             │
                 YES            NO
                   │             │
                   ▼             ▼
         ┌─────────────┐  ┌─────────────────┐
         │  Validate   │  │  Log Error      │
         │  Schema     │  │  • File path    │
         └──────┬──────┘  │  • Line number  │
                │         │  • Error type   │
                │         │  • Raw content  │
                │         └─────────┬───────┘
                │                   │
         ┌──────┴──────┐           │
         │  Valid?     │           │
         └──┬───────┬──┘           │
            │      │               │
          YES     NO               │
            │      │               │
            │      └───────────────┘
            │                      │
            ▼                      ▼
    ┌──────────────┐      ┌──────────────────┐
    │  Write to    │      │  Add to          │
    │  JSONL       │      │  errors.jsonl    │
    └──────────────┘      │  Continue parse  │
                          └──────────────────┘
```

**Error Categories**:
1. **Format errors**: Unknown platform, corrupted JSON
2. **Schema errors**: Missing required fields
3. **Encoding errors**: Invalid UTF-8
4. **LLM errors**: API failures, rate limits

**Error Recovery**:
- **Never crash** - log and continue
- **Partial success** - save valid data
- **Retry LLM** - exponential backoff (3 attempts)
- **Manual review** - errors.jsonl for inspection

---

## 8. DEDUPLICATION STRATEGY

```
Message arrives
      │
      ▼
┌─────────────────────────────────────┐
│  Generate message_hash               │
│  SHA256(content + timestamp +       │
│         platform)                   │
└────────────┬────────────────────────┘
             │
             ▼
┌─────────────────────────────────────┐
│  Check hash in conversations.jsonl  │
│  (in-memory set for speed)          │
└────────┬───────────────┬────────────┘
         │               │
    EXISTS           NEW
         │               │
         ▼               ▼
┌─────────────┐   ┌──────────────────┐
│  Skip       │   │  Add to output   │
│  (dup)      │   │  Update hash set │
└─────────────┘   └──────────────────┘
```

**Hash Components**:
- `content`: Message text (normalized whitespace)
- `timestamp`: ISO-8601 format
- `platform`: chatgpt|claude|gemini|...

**Why this works**: Same message exported from multiple sources will have identical hash

---

## 9. INCREMENTAL PROCESSING ARCHITECTURE

```
                ┌──────────────────┐
                │  Large Export    │
                │  File (50K msgs) │
                └────────┬─────────┘
                         │
                         ▼
              ┌──────────────────────┐
              │  Stream Reader        │
              │  Read line-by-line   │
              └────────┬─────────────┘
                       │
          ┌────────────┴────────────┐
          │                         │
          ▼                         ▼
    ┌──────────┐            ┌──────────┐
    │  Chunk 1 │            │  Chunk N │
    │  (1000)  │    ...     │  (1000)  │
    └────┬─────┘            └────┬─────┘
         │                       │
         └───────────┬───────────┘
                     │
                     ▼
           ┌─────────────────────┐
           │  Process Chunk      │
           │  • Parse            │
           │  • Extract          │
           │  • Validate         │
           └──────────┬──────────┘
                      │
                      ▼
           ┌─────────────────────┐
           │  Append to JSONL    │
           │  (streaming write)  │
           └──────────┬──────────┘
                      │
                      ▼
              ┌───────────────┐
              │  Free memory  │
              │  Next chunk   │
              └───────────────┘
```

**Memory Management**:
- **Max chunk**: 1000 messages in RAM
- **Streaming I/O**: Never load full file
- **Progressive output**: Write immediately after validation
- **Memory limit**: ~500MB peak regardless of file size

---

## 10. GEMINI API BATCH PROCESSING

```
Flagged conversations (10-20% of total)
         │
         ▼
┌──────────────────────────────────────┐
│  Batch Accumulator                   │
│  Collect until:                      │
│  • 50 conversations OR               │
│  • 30 seconds timeout                │
└────────────┬─────────────────────────┘
             │
             ▼
┌──────────────────────────────────────┐
│  Build Batch Prompt                  │
│  Format:                             │
│  [                                   │
│    {id: 1, text: "..."},             │
│    {id: 2, text: "..."},             │
│    ...                               │
│  ]                                   │
│  Instruction: Extract events/docs    │
└────────────┬─────────────────────────┘
             │
             ▼
┌──────────────────────────────────────┐
│  Gemini API Call                     │
│  Model: gemini-2.0-flash-exp         │
│  Temperature: 0.1 (consistency)      │
│  Response: JSON array of results     │
└────────────┬─────────────────────────┘
             │
         ┌───┴───┐
         │Success│
         └───┬───┘
             │
    ┌────────┴────────┐
    │                 │
  YES                NO
    │                 │
    ▼                 ▼
┌─────────┐     ┌──────────────┐
│ Parse   │     │ Retry (3x)   │
│ Results │     │ Exponential  │
│ Write   │     │ Backoff      │
└─────────┘     └──────────────┘
```

**Cost Optimization**:
- **Batch 50x**: 50 conversations per API call
- **Input tokens**: ~10K per batch
- **Output tokens**: ~5K structured JSON
- **Cost**: ~$0.01 per batch (Gemini 2.0 Flash pricing)

---

## 11. COMPLETE DATA FLOW SUMMARY

```
Platform Exports
      ↓
Format Detection (Python)
      ↓
Parser Selection (ChatGPT|Claude|Gemini|...)
      ↓
Normalized Conversations + Message Hash
      ↓
┌─────────────────────────────────────────────────┐
│  PARALLEL PROCESSING                            │
│                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────┐ │
│  │  spaCy NER  │  │  Regex      │  │  Code   │ │
│  │  Entities   │  │  Patterns   │  │  Detect │ │
│  └──────┬──────┘  └──────┬──────┘  └────┬────┘ │
│         │                │               │      │
│         └────────────────┴───────────────┘      │
│                          │                      │
└──────────────────────────┼──────────────────────┘
                           │
                           ▼
            ┌──────────────────────────┐
            │  Flag for LLM Analysis   │
            │  (10-20% of total)       │
            └────────────┬─────────────┘
                         │
                         ▼
            ┌──────────────────────────┐
            │  Gemini Batch Processing │
            │  • Strategy docs         │
            │  • Legal docs            │
            │  • Event extraction      │
            │  • Entity relations      │
            └────────────┬─────────────┘
                         │
                         ▼
            ┌──────────────────────────┐
            │  Schema Validation       │
            │  (Pydantic)              │
            └────────────┬─────────────┘
                         │
                         ▼
            ┌──────────────────────────┐
            │  FINAL OUTPUT            │
            │  • conversations.jsonl   │
            │  • entities.jsonl        │
            │  • events.jsonl          │
            │  • artifacts.jsonl       │
            │  • documents.jsonl       │
            │  • errors.jsonl          │
            └──────────────────────────┘
```

---

## 12. PERFORMANCE TARGETS

| Metric | Target | Tool |
|--------|--------|------|
| Parsing speed | 1000 msgs/sec | Python |
| Entity extraction | 1000 msgs/sec | spaCy |
| Code detection | 5000 msgs/sec | Regex |
| LLM processing | 10-20% coverage | Gemini batch |
| Memory usage | <500MB peak | Streaming |
| Dedup accuracy | 100% | SHA256 hash |
| Schema validation | 100% | Pydantic |
| Token cost | <$1 per 10K msgs | Selective LLM |

---

## 13. MISSING WORKFLOWS TO REVIEW

**Potential gaps for Opus to evaluate**:

1. **Relationship extraction** - How to map entity relationships?
2. **Conflict resolution** - Multiple LLM extractions contradict?
3. **Quality scoring** - Confidence thresholds for filtering?
4. **Output merging** - Multiple exports from same platform?
5. **Incremental updates** - New exports added later?
6. **Manual corrections** - How to incorporate human edits?
7. **Export format** - CSV/JSON/Timeline visualization?
8. **Privacy** - PII scrubbing workflow?
9. **Monitoring** - Progress tracking for large files?
10. **Rollback** - Recovery from corrupted output?

---

## REFERENCES

- [spaCy NER Pipeline Best Practices](https://github.com/explosion/spacy/blob/master/website/docs/usage/processing-pipelines.mdx)
- [Building NLP Pipelines for Sentiment Analysis](https://codezup.com/building-a-nlp-pipeline-for-sentiment-analysis-using-python-and-spacy/)
- [NLP Techniques 2025](https://labelyourdata.com/articles/natural-language-processing/techniques)
- [Data Engineering for NLP Pipelines](https://iabac.org/blog/data-engineering-for-natural-language-processing-building-nlp-pipelines)

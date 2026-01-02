# Multi-Platform Chat History Processor - Master Plan

## Project Goal
Process historical chat exports from multiple AI platforms, extract structured knowledge, and create queryable databases of artifacts, entities, events, and timelines.

## Input Sources

### Platforms
1. **ChatGPT** - OpenAI conversations
2. **Claude** - Anthropic conversations
3. **Perplexity** - Search-based conversations
4. **Gemini** - Google AI conversations
5. **Qwen** - Alibaba AI conversations

### Export Formats
- **Markdown** - Browser extension exports
- **PDF** - Requires OCR (Tesseract)
- **JSON** - Native platform exports
- **HTML** - Web conversation exports
- **Other** - Various proprietary formats

## Core Requirements

### 1. Format Normalization
**Challenge**: Each platform has different schema/structure

**Solution**:
- Parse each platform's export format
- Extract common fields: timestamp, user, assistant, message content
- Handle platform-specific metadata
- Normalize to unified schema

### 2. OCR Processing
**For PDF exports**:
- Use Tesseract OCR for text extraction
- Alternative: Stirling PDF (if OCR can be fixed)
- Handle multi-page documents
- Preserve formatting/structure where possible

### 3. Content Extraction

#### A. **Artifacts** (Code, Files, Attachments)
- **Code Blocks**: Extract language, content, context
- **File Attachments**: Links, downloads, embedded files
- **Generated Content**: AI-created documents, images
- **Metadata**: Creation date, platform, conversation ID

**Output Format**:
```json
{
  "artifact_id": "uuid",
  "type": "code|file|document",
  "language": "python|javascript|...",
  "content": "...",
  "conversation_id": "...",
  "timestamp": "ISO-8601",
  "platform": "chatgpt|claude|...",
  "context": "surrounding conversation"
}
```

#### B. **Strategy Documents**
- **Planning Discussions**: Project plans, roadmaps
- **Decision Documents**: Architecture decisions, trade-offs
- **Proposals**: Feature proposals, business plans
- **Brainstorming**: Idea generation sessions

**Extraction Method**:
- NLTK preprocessing: Identify planning keywords (plan, strategy, roadmap, propose)
- LLM analysis: Understand context, extract structured strategy
- Classification: Type of strategy document

#### C. **Legal Documents**
- **Contracts**: Agreements, terms
- **Compliance**: Legal requirements, regulations
- **Disclaimers**: Warnings, liability statements
- **Privacy**: Data handling, GDPR discussions

**Extraction Method**:
- Pattern matching: Legal keywords (contract, agreement, terms, liability)
- LLM analysis: Classify legal document type
- Entity extraction: Parties, dates, obligations

#### D. **Entity Maps**
Extract and link entities across conversations:
- **People**: Names, roles, relationships
- **Organizations**: Companies, teams, projects
- **Projects**: Codenames, initiatives
- **Technologies**: Tools, frameworks, languages
- **Locations**: Cities, offices, regions
- **Concepts**: Ideas, methodologies

**Output Format**:
```json
{
  "entity_id": "uuid",
  "type": "person|org|project|tech|location|concept",
  "name": "Entity Name",
  "aliases": ["alt name 1", "alt name 2"],
  "first_mention": "timestamp",
  "mention_count": 42,
  "relationships": [
    {"entity_id": "...", "relationship": "works_at|uses|located_in"}
  ],
  "context": "Description from conversations"
}
```

#### E. **Events & Timelines**
Extract temporal events for timeline reconstruction:
- **Milestones**: Project completions, launches
- **Decisions**: Key choices made
- **Meetings**: Discussions, agreements
- **Incidents**: Problems, bugs, outages
- **Changes**: Updates, pivots, transitions

**Output Format**:
```json
{
  "event_id": "uuid",
  "type": "milestone|decision|meeting|incident|change",
  "title": "Brief description",
  "description": "Detailed context",
  "timestamp": "ISO-8601",
  "entities_involved": ["entity_id1", "entity_id2"],
  "conversation_id": "...",
  "platform": "...",
  "tags": ["project-x", "critical", "success"]
}
```

## Architecture

### Phase 1: Ingestion (Gemini handles)
```
Input: Raw exports (Markdown, PDF, JSON, HTML)
   ↓
OCR Processing (if PDF)
   ↓
Format Detection & Parsing
   ↓
Schema Normalization
   ↓
Unified Conversation Format
```

### Phase 2: Preprocessing (NLTK - Local)
```
Unified Conversations
   ↓
Extract conversation turns
   ↓
Identify message types (code, question, answer, planning)
   ↓
Pattern matching: Legal keywords, planning keywords, entities
   ↓
Basic entity extraction (names, dates, technologies)
   ↓
Flag conversations for LLM analysis
```

### Phase 3: Analysis (LLM - Selective)
```
Flagged Conversations
   ↓
Context-aware extraction:
  - Strategy document classification
  - Legal document understanding
  - Entity relationship mapping
  - Event importance ranking
  - Artifact context understanding
   ↓
Structured Knowledge Graphs
```

### Phase 4: Storage & Indexing
```
Extracted Data
   ↓
Database Schema:
  - conversations (normalized)
  - artifacts (code, files)
  - entities (people, orgs, projects)
  - events (timeline items)
  - documents (strategy, legal)
  - relationships (entity links)
   ↓
Search Indexes:
  - Full-text search (conversations)
  - Entity search
  - Timeline query
  - Artifact search
```

## Technology Stack

### Processing Pipeline
- **Python** - Main orchestration
- **NLTK** - Preprocessing, pattern matching, entity extraction
- **Tesseract OCR** - PDF text extraction
- **spaCy** - Named Entity Recognition (NER)
- **Gemini API** - Bulk processing, format parsing (unlimited usage)
- **Claude/Opus** - Strategy review, quality control
- **SQLite/PostgreSQL** - Structured storage
- **Elasticsearch** - Full-text search (optional)

### Data Flow
```
┌─────────────────────────────────────────────┐
│  Platform Exports                           │
│  (ChatGPT, Claude, Gemini, Perplexity, etc) │
└─────────────┬───────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────────┐
│  Format Parser (Gemini)                     │
│  • Detect format (MD, PDF, JSON, HTML)      │
│  • OCR if needed (Tesseract)                │
│  • Parse to unified schema                  │
└─────────────┬───────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────────┐
│  NLTK Preprocessing (Local - Free)          │
│  • Tokenization                             │
│  • Pattern matching (legal, code, etc)      │
│  • Basic NER (people, orgs, dates)          │
│  • Flag important conversations             │
└─────────────┬───────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────────┐
│  Selective LLM Analysis (Gemini)            │
│  • Strategy document extraction             │
│  • Legal document classification            │
│  • Entity relationship mapping              │
│  • Event importance ranking                 │
│  • Only on flagged content (10-20% of data) │
└─────────────┬───────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────────┐
│  Structured Database                        │
│  • Conversations (normalized)               │
│  • Artifacts (code, files)                  │
│  • Entities (knowledge graph)               │
│  • Events (timeline)                        │
│  • Documents (strategy, legal)              │
└─────────────────────────────────────────────┘
```

## Implementation Phases

### Phase 1: Foundation (Gemini codes)
- [ ] Format parser for each platform (ChatGPT, Claude, Gemini, Perplexity, Qwen)
- [ ] OCR integration (Tesseract)
- [ ] Unified schema definition
- [ ] Basic conversation ingestion

### Phase 2: NLTK Preprocessing (Gemini codes)
- [ ] Conversation turn extraction
- [ ] Pattern matchers (code blocks, legal terms, planning keywords)
- [ ] Basic NER (people, organizations, technologies)
- [ ] Conversation flagging logic

### Phase 3: Extraction (Gemini codes)
- [ ] Artifact extractor (code, files, attachments)
- [ ] Entity extractor with relationship mapping
- [ ] Event detector and timeline builder
- [ ] Document classifier (strategy, legal)

### Phase 4: LLM Analysis Integration (Gemini codes)
- [ ] Gemini API integration for bulk analysis
- [ ] Prompts for each extraction type
- [ ] Result validation and quality checks
- [ ] Token usage optimization

### Phase 5: Storage & Query (Gemini codes)
- [ ] Database schema implementation
- [ ] Data insertion pipeline
- [ ] Query interface (CLI or web)
- [ ] Export functionality (JSON, CSV, timeline visualization)

## Success Metrics

1. **Coverage**: Successfully parse 95%+ of exports
2. **Accuracy**: Entity extraction 90%+ precision
3. **Cost**: <$5 total for processing (using Gemini unlimited)
4. **Speed**: Process 10K messages/minute
5. **Usability**: Query interface returns results in <1 second

## Open Questions for Opus Review

1. **Database Choice**: SQLite vs PostgreSQL vs NoSQL?
2. **Entity Linking**: How to handle name variations (Bob vs Robert)?
3. **Timeline Granularity**: Hour-level or day-level events?
4. **Artifact Storage**: Store code inline or in separate files?
5. **Search Strategy**: Full-text search needed or just structured queries?
6. **Privacy**: Any PII scrubbing required before storage?
7. **Incremental Updates**: How to handle new chat exports later?

## Side Quest: Fix Stirling PDF OCR

If we can fix the Stirling PDF deployment, we could use it instead of Tesseract:
- **Problem**: OCR doesn't work, can't read logs
- **Benefit**: Web interface, better quality, already deployed
- **Next Step**: Debug log access and OCR configuration

---

**Ready for Opus Review**: This plan needs expert validation before Gemini starts coding.

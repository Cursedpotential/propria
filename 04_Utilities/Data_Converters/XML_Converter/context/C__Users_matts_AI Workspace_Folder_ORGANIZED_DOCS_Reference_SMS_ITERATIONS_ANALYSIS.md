# SMS ITERATIONS ANALYSIS

> _Naming (D-140, 2026-09-05; applied 2026-09-06): this product is **vestigia** (formerly traceIQ / TraceIQ - Latin: footprints, tracks). Working copy: `probata/modules/vestigia/` (directory rename from `modules/traceIQ/` landed 2026-09-06; old name kept as a junction). GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._

**Date**: December 27, 2025
**Case**: Salem v. Kinzel (2025-53985-DC)
**Analysis**: Cross-app SMS table iteration review

---

## EXECUTIVE SUMMARY

After searching and reviewing all organized apps, **SUPABASE_PRODUCTION_SCHEMA.SQL is the definitive SMS tables source**. The other apps (Chat Miner, TraceIQ) had different purposes and don't contain SMS-specific table designs.

**Deliverable Created**: `SMS_TABLES_FINAL.sql` (900+ lines) - Consolidates best from supabase_production_schema.sql with forensic enhancements for Salem case.

---

## APP-BY-APP FINDINGS

### 1. SUPABASE PRODUCTION SCHEMA ⭐ (Best Source)
**Location**: `STACK_Deployment/n8n-local/supabase_production_schema.sql`
**Purpose**: Production evidence management for Salem case
**SMS Features**:
- ✅ `messaging_documents` - Source file tracking with SHA-256 hashing
- ✅ `messaging_conversations` - Thread grouping by platform
- ✅ `messaging_messages` - Core message table with forensic fields
- ✅ `behavior_categories` - Coercive control taxonomy (18 categories)
- ✅ `messaging_behaviors` - Detected manipulative patterns
- ✅ `messaging_evidence_items` - Court-ready evidence linking
- ✅ `mcl_factors` - MCL 722.23 legal factors reference
- ✅ `messaging_factor_citations` - Links evidence to legal factors
- ✅ Full-text search with pg_trgm
- ✅ Automated triggers for counts/updates
- ✅ Materialized views for reporting

**Strengths**:
- Explicitly designed for Salem case MCL 722.23 factors
- Chain of custody tracking built-in
- Behavioral analysis taxonomy already defined
- Multi-platform support (SMS, Facebook, Snapchat, etc.)
- Evidence admissibility focus

**Gaps Fixed in SMS_TABLES_FINAL.sql**:
- Added MMS attachments table
- Added OCR/transcription fields for media
- Added linguistic markers (caps ratio, question/exclamation counts)
- Added message-to-message linking (previous_message_id, next_message_id)
- Added time_since_previous_seconds for response time analysis
- Added behavior patterns table for multi-message analysis
- Enhanced entity tracking with relationship_to_petitioner/respondent fields
- Added timeline_events integration

---

### 2. CHAT MINER APP ❌ (Not SMS-Specific)
**Location**: `Chat_Miner_App/`
**Purpose**: Generic chat history parser (ChatGPT, Claude, Gemini exports)
**Key Files Reviewed**:
- `timeline_ingestion_schema.sql` (1066 lines)
- `chat_parser_implementation_guide.md` (1626 lines)
- `batch-processing.md`
- `SESSION_STATE_CURRENT.md`

**What It Actually Does**:
- Parses AI chat conversation exports (JSON, JSONL, Markdown)
- Generic timeline event storage (not SMS-specific)
- Format auto-discovery with LLM
- Entity extraction via spaCy NER
- Redis buffer for deduplication
- PostgreSQL + PGVector storage

**SMS Relevance**: ❌ NONE
- No SMS XML parsing
- No phone number normalization
- No MMS attachment handling
- No SMS-specific fields
- Designed for AI chat platforms, not text messages

**Schema Highlights** (Generic, Not SMS):
```sql
CREATE TABLE timeline_events (
    event_id UUID PRIMARY KEY,
    source_id UUID,
    event_timestamp TIMESTAMPTZ NOT NULL,
    event_type VARCHAR(100),
    title VARCHAR(500),
    description TEXT,
    raw_data JSONB,
    ...
);

CREATE TABLE chat_messages (
    message_id UUID PRIMARY KEY,
    channel_id UUID,
    sender_id UUID,
    message_text TEXT,
    message_html TEXT,
    reply_count INTEGER,
    reaction_count INTEGER,
    ...
);
```

**Useful Concepts Borrowed**:
- ✅ Timeline event abstraction (adapted for SMS_TABLES_FINAL)
- ✅ Entity extraction patterns
- ✅ Full-text search indexing strategy
- ✅ Materialized views for reporting
- ✅ Audit trail table design

---

### 3. TRACEIQ TIMELINE APP ⚠️ (Location Data Focus)
**Location**: `TraceIQ_Timeline_App/`
**Purpose**: Google Timeline geospatial data processing
**Key Files Reviewed**:
- `schema_complete.sql` (542 lines)
- `traceiq/schema.sql` (420 lines)

**What It Actually Does**:
- Processes Google Maps Timeline JSON exports
- Enriches location data with Radar/Google APIs
- Tracks visits, activities, paths, overnight stays
- Anomaly detection (bouncy trips, schedule deviations)

**SMS Feature Found**:
```sql
CREATE TABLE messages (
    message_id TEXT PRIMARY KEY,
    event_id TEXT NOT NULL UNIQUE,
    timestamp_utc TEXT NOT NULL,
    sender TEXT,
    receiver TEXT,
    body TEXT,
    message_type TEXT,  -- SMS_IN, SMS_OUT, FB_CHAT, SNAPCHAT, GCHAT
    source_file TEXT,
    is_private INTEGER DEFAULT 0,
    linked_location_event_id TEXT,  -- FK to timeline_enriched
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**SMS Relevance**: ⚠️ MINIMAL
- Basic message table exists BUT
- No parsing logic for SMS XML
- No behavioral analysis
- No court evidence features
- Just a placeholder for linking messages to location events

**Useful Concepts Borrowed**:
- ✅ Event linking (messages ↔ locations)
- ✅ Forensic evidence view design
- ✅ Data quality metrics tracking
- ✅ Chain of custody patterns

---

### 4. UTILITIES FOLDER 🔧 (Scripts Only)
**Location**: `Utilities/_SCRIPTS/`
**SMS-Related Scripts Found**:
- `organize_downloads_smart.py` - File categorization (mentions "sms" in Court_Evidence patterns)
- No SMS XML parsers
- No SMS import/export scripts
- No behavioral analysis tools

**SMS Relevance**: ❌ NONE for parsing

---

### 5. ARCHIVE FOLDER 📦 (Historical Reference)
**Location**: `ARCHIVE_Old_Versions/External_Tools_Reference/`
**SMS-Related Tools**:
- None found specific to SMS processing
- Various third-party forensic tools referenced but not integrated

---

## CONSOLIDATION DECISIONS

### What Went Into SMS_TABLES_FINAL.sql

**From SUPABASE_PRODUCTION_SCHEMA.SQL** (Primary Source):
- ✅ Core table structure (documents, conversations, messages)
- ✅ Behavior categories taxonomy (18 types)
- ✅ MCL 722.23 factors integration
- ✅ Evidence items and factor citations
- ✅ Full-text search setup
- ✅ Trigger functions for auto-updates
- ✅ Materialized views

**Enhancements Added**:
- ✅ `messaging_attachments` table (MMS support)
  - OCR text extraction fields
  - Face detection flags
  - EXIF metadata storage
  - Thumbnail paths
  
- ✅ `messaging_behavior_patterns` table
  - Multi-message pattern detection
  - Escalation tracking
  - Cycle identification
  - Temporal pattern analysis
  
- ✅ `messaging_timeline_events` table
  - Integration with Chronicle/Video Analyzer
  - Cross-app event correlation
  
- ✅ Enhanced `messaging_messages` columns:
  - `previous_message_id` / `next_message_id` (context chain)
  - `time_since_previous_seconds` (response time analysis)
  - `contains_apology` / `contains_blame` / etc. (linguistic markers)
  - `question_count` / `exclamation_count` / `caps_ratio`
  - `is_redacted` / `redaction_reason` (court sensitivity)
  
- ✅ Enhanced `messaging_entities` columns:
  - `relationship_to_petitioner`
  - `relationship_to_respondent`
  - `is_flagged` / `flag_reason`
  
- ✅ Additional indexes for performance:
  - Composite indexes for common query patterns
  - GIN indexes for JSONB and text search
  - Partial indexes for filtered queries

**From CHAT_MINER** (Concepts Only):
- ✅ Audit trail table pattern
- ✅ Sync logs design
- ✅ Entity timeline associations
- ✅ Full-text search function patterns

**From TRACEIQ** (Concepts Only):
- ✅ Event linking patterns
- ✅ Data quality metrics approach
- ✅ Forensic evidence packaging view

---

## MISSING COMPONENTS (Still Need to Build)

### 1. SMS XML Parser Scripts 🚧
**Status**: NOT FOUND in any app
**Need**: Python scripts to parse Android/iOS SMS XML exports
**Requirements**:
- Read SMS Backup & Restore XML format
- Extract: timestamp, sender, recipient, body, type (sent/received)
- Handle MMS attachments (extract paths, types)
- Normalize phone numbers (E.164 format)
- Generate SHA-256 hashes for deduplication
- Insert into `messaging_documents` + `messaging_messages`

**File Location**: Should create in `Chat_Miner_App/sms_parsers/`

### 2. MMS Attachment Processor 🚧
**Status**: NOT FOUND in any app
**Need**: Script to process MMS media files
**Requirements**:
- Extract images/videos/audio from MMS folders
- Run OCR on images (Tesseract)
- Transcribe audio (Whisper)
- Detect faces in images
- Extract EXIF data
- Upload to Cloudflare R2
- Insert into `messaging_attachments`

**File Location**: Should create in `Chat_Miner_App/mms_processors/`

### 3. Behavioral Analysis Pipeline 🚧
**Status**: Partially in supabase schema, NO implementation code
**Need**: Detection scripts for coercive control patterns
**Requirements**:
- Keyword pattern matching (regex)
- LLM-based classification (Gemini 2.5 Pro)
- Confidence scoring
- Multi-message pattern detection
- Insert into `messaging_behaviors` + `messaging_behavior_patterns`

**File Location**: Should create in `Chat_Miner_App/behavior_analysis/`

### 4. Evidence Package Generator 🚧
**Status**: NOT FOUND in any app
**Need**: Generate court-ready evidence packages
**Requirements**:
- Query messages by date range / participant
- Extract quotes with context
- Link to MCL 722.23 factors
- Generate PDF exhibits with metadata
- Create evidence summary reports

**File Location**: Should create in `Chat_Miner_App/evidence_gen/`

---

## DEPLOYMENT PLAN

### Phase 1: Database Setup ✅ READY
1. Import `SMS_TABLES_FINAL.sql` to Supabase
2. Verify all tables/indexes/triggers created
3. Test sample inserts
4. Configure RLS policies if needed

### Phase 2: SMS Import Scripts 🚧 TO BUILD
1. Build Android SMS XML parser
2. Build iOS Messages parser
3. Test on sample exports
4. Implement batch processing

### Phase 3: MMS Processing 🚧 TO BUILD
1. Build attachment extractor
2. Integrate OCR (Tesseract)
3. Integrate transcription (Whisper)
4. Test media uploads to R2

### Phase 4: Behavioral Analysis 🚧 TO BUILD
1. Implement keyword detection
2. Integrate Gemini API for classification
3. Build pattern detection engine
4. Test on real message corpus

### Phase 5: Evidence Generation 🚧 TO BUILD
1. Build query interface
2. Build PDF exhibit generator
3. Build timeline integration
4. Test with real case data

---

## TECHNICAL DEBT / KNOWN ISSUES

### From SUPABASE_PRODUCTION_SCHEMA.SQL
- ❌ No actual SMS parsing code (just tables)
- ❌ No behavioral detection implementation (just categories)
- ❌ No MMS attachment handling
- ⚠️ RLS policies commented out (security risk if multi-user)

### From CHAT_MINER
- ❌ Wrong purpose (AI chats, not SMS)
- ❌ Incomplete implementation (just architecture docs)
- ⚠️ Redis dependency (adds complexity)

### From TRACEIQ
- ❌ Messages table is a stub (no real SMS support)
- ⚠️ SQLite format (would need conversion for Supabase)

---

## FILE MANIFEST

### Created Files
1. **SMS_TABLES_FINAL.sql** (900+ lines)
   - Location: `DOCS_Reference/SMS_TABLES_FINAL.sql`
   - Complete schema ready for Supabase deployment
   - All tables, indexes, triggers, views, functions

2. **SMS_ITERATIONS_ANALYSIS.md** (this file)
   - Location: `DOCS_Reference/SMS_ITERATIONS_ANALYSIS.md`
   - Analysis of all app iterations
   - Deployment recommendations

### Source Files Reviewed
1. `STACK_Deployment/n8n-local/supabase_production_schema.sql` ⭐
2. `Chat_Miner_App/timeline_ingestion_schema.sql`
3. `Chat_Miner_App/docs/chat_parser_implementation_guide.md`
4. `TraceIQ_Timeline_App/schema_complete.sql`
5. `TraceIQ_Timeline_App/traceiq/schema.sql`

---

## NEXT ACTIONS

### Immediate (Before You Return from Work)
- [x] Create SMS_TABLES_FINAL.sql
- [x] Create SMS_ITERATIONS_ANALYSIS.md
- [ ] Save findings to Supermemory/Pieces
- [ ] Create SMS_DEPLOYMENT_CHECKLIST.md

### Short-Term (When You're Back)
1. Review SMS_TABLES_FINAL.sql
2. Import to Supabase (oflqpddqaecotsdsxbzp project)
3. Test sample data inserts
4. Verify triggers/views working

### Medium-Term (Next Dev Session)
1. Build Android SMS XML parser
2. Build MMS attachment processor
3. Test on real SMS exports
4. Begin behavioral analysis implementation

### Long-Term (Full Salem Trinity Integration)
1. Link SMS timeline to Chronicle voice narratives
2. Link SMS to Video Analyzer surveillance footage
3. Cross-correlate messages with TraceIQ location data
4. Build unified Master Timeline view

---

## COST ESTIMATE

### Current State
- Tables: ✅ FREE (schema only)
- Storage: Depends on message volume
  - 100K messages ≈ 500MB ≈ $0.13/month (Supabase)
  - 1M messages ≈ 5GB ≈ $1.25/month

### To Build (Processing Pipeline)
- SMS Parsing: FREE (Python)
- OCR (Tesseract): FREE (local)
- Transcription (Whisper): FREE (local) or $0.006/min (API)
- Behavioral Analysis (Gemini):
  - Free tier: 1M tokens/month
  - Paid: $1.50/1M tokens (Flash) or $7.50/1M (Pro)
  - Estimate: $5-20/month for full corpus analysis

**Total Estimated Monthly Cost**: $5-25 depending on volume

---

## CONCLUSION

**The SMS tables are READY for deployment.** The schema is comprehensive, forensically sound, and court-focused.

**What's NOT ready**: The actual parsing/processing code. Chat Miner was designed for AI chats (not SMS), and TraceIQ is for location data. You need to build:
1. SMS XML parsers (Python)
2. MMS processors (Python + OCR/transcription)
3. Behavioral analysis pipeline (Python + Gemini API)
4. Evidence package generator (Python + PDF libs)

**Recommendation**: Deploy SMS_TABLES_FINAL.sql to Supabase NOW, then build the parsers incrementally as you process your SMS exports.

---

**Analysis Complete**: December 27, 2025
**Next Review**: After parser implementation

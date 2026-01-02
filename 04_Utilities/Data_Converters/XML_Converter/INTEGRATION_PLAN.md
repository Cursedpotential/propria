# Salem Forensic Ingestion - Complete Integration Analysis

## Executive Summary

**Current State**: All core components are **built and ready** but NOT connected. The forensic schema, NLP service, rules loader, and multi-provider AI are complete standalone services. The XML processor and Supabase uploader still route to simple flat tables.

**Next Phase**: Wire everything together - connect XML processor → forensic tables → NLP analysis → behavior detection.

---

## CURRENT ARCHITECTURE (As-Built)

### ✅ COMPLETED COMPONENTS

#### 1. **Forensic Database Schema** (`services/supabaseService.ts:193-610`)

**10 Court-Admissible Tables**:
- `messaging_documents` - Source file chain of custody (SHA-256 hash, acquisition metadata)
- `messaging_conversations` - Thread grouping across platforms (SMS/MMS/Facebook/WhatsApp)
- `messaging_messages` - Core forensic message records with 40+ columns
- `messaging_calls` - Call log records (separate from messages)
- `messaging_attachments` - MMS/media files with storage paths
- `messaging_behaviors` - Detected behavioral patterns with **ACTUAL MATCHED TEXT**
- `messaging_behavior_categories` - 10 pre-loaded categories mapped to MCL 722.23 factors
- `mcl_factors` - Michigan Child Custody Act factors A-L
- `messaging_entities` - Extracted people, places, orgs
- `messaging_audit_log` - Complete forensic audit trail

**Auto-Triggers**:
- Content hash generation (SHA-256)
- Behavior count updates
- Conversation stats updates

**Indexes**: 25+ optimized for performance

**Status**: ✅ Schema complete, ready to deploy
**Location**: `services/supabaseService.ts:193-610`

---

#### 2. **Multi-Provider AI Service** (`services/aiService.ts`)

**Supported Providers** (No Vendor Lock-In):
- OpenRouter (default if `VITE_OPENROUTER_API_KEY` set)
- Anthropic Claude
- OpenAI
- Google Gemini

**Auto-Detection**: Reads environment variables

**Status**: ✅ Complete and working
**Location**: `services/aiService.ts`

---

#### 3. **Browser NLP Service** (`services/browserNlpService.ts`)

**Libraries**:
- `compromise` (180kb) - Entity extraction, rule-based patterns
- `node-nlp` (nlp.js) - Sentiment analysis

**Micro-Level Analysis** (Surface-Level Only):
- ✅ Entity extraction (people, places, orgs, phones, emails, dates)
- ✅ Sentiment (positive/negative/neutral)
- ✅ Linguistic markers (question_count, exclamation_count, caps_ratio)
- ✅ Simple flags (contains_apology, contains_blame, contains_threat, contains_profanity)
- ✅ Basic stats (word_count, character_count)

**What It DOESN'T Do** (Macro Analysis - Later):
- ❌ Complex pattern detection (gaslighting cycles, DARVO)
- ❌ Multi-message analysis
- ❌ Temporal pattern detection
- ❌ Cross-platform correlation

**Missing Method**: `detectBehaviors()` - referenced on line 355 but not implemented yet

**Status**: ✅ Built, NOT integrated yet
**Location**: `services/browserNlpService.ts`

---

#### 4. **Rules Loader System** (`services/rulesLoader.ts`)

**21 Behavioral Pattern Categories** (140+ patterns):
1. manipulation_gaslighting
2. projection_blameshift
3. victimhood_narrative
4. certainty_absolutism
5. character_assassination (with actual slurs/profanity)
6. coercive_control
7. stonewalling_blocking
8. last_minute_plan_change
9. parental_alienation
10. financial_abuse
11. healthcare_denial
12. sexual_weaponization
13. love_bombing
14. feigned_incompetence
15. social_media_deception
16. infidelity_betrayal
17. substance_weaponization
18. stimulant_control
19. alcohol_risk_child
20. reactive_abuse_gotcha
21. dismissiveness

**Key Features**:
- Captures **ACTUAL matched text** (e.g., "motherfucker", "crackhead", "psycho")
- Position tracking (start_char, end_char, context)
- Regex + simple string matching
- Configurable rules path (reusable across cases)

**Entities Loaded**:
- People: Katrina (Kat), Salem (Matt, Matthew)
- Places: Huckleberry Junction (Huckleberry, Huck)
- Platforms: snapchat, instagram, facebook, tiktok
- Substances: alcohol (fireball, vodka, tequila), stimulants (adderall, addy)

**Status**: ✅ Complete with ALL 21 categories embedded
**Location**: `services/rulesLoader.ts`

---

### 🔴 DISCONNECTED COMPONENTS (Not Integrated)

#### 5. **XML Stream Processor** (`services/xmlStreamService.ts`)

**Current Behavior**:
- Parses SMS/MMS/call XML tags ✅
- Generates UUIDs, hashes, deduplication ✅
- Handles attachments, images ✅
- Routes EVERYTHING to simple flat tables via `uploadBatch()` ❌

**What's Missing**:
1. NO document metadata extraction → `messaging_documents`
2. NO conversation grouping → `messaging_conversations`
3. NO message vs call routing (everything goes to "messages" table)
4. NO attachment table population
5. NO NLP analysis during ingestion
6. NO behavior detection

**Critical Methods**:
- `convert()` - Main ingestion loop (lines 175-516)
- `processItem()` - XML parsing (lines 535-607)
- `flushSupabaseBuffer()` - Upload batching (lines 527-534)

**Status**: 🔴 Needs complete integration overhaul
**Location**: `services/xmlStreamService.ts`

---

#### 6. **Supabase Upload Service** (`services/supabaseService.ts:150-191`)

**Current Behavior**:
```typescript
async uploadBatch(messages: any[], entities: any[], attachments: any[]) {
    // Uploads to:
    // - this.table (flat "messages" table) ❌
    // - this.entityTable (flat "entities" table) ❌
    // - this.attachmentTable (flat "attachments" table) ❌
}
```

**What's Missing**:
1. NO `messaging_documents` record creation
2. NO `messaging_conversations` lookup/creation
3. NO routing to `messaging_messages` table
4. NO routing to `messaging_calls` table
5. NO routing to `messaging_attachments` table
6. NO behavior insertion into `messaging_behaviors`

**Status**: 🔴 Needs complete rewrite
**Location**: `services/supabaseService.ts:150-191`

---

## DATA FLOW ANALYSIS

### Current Flow (Broken)

```
XML File
  ↓
xmlStreamService.convert()
  ↓
processItem() - Parse XML attributes
  ↓
flushSupabaseBuffer() - Batch messages
  ↓
supabaseService.uploadBatch()
  ↓
WRITES TO FLAT TABLES ❌
  - messages (simple table)
  - entities (simple table)
  - attachments (simple table)
```

**Problem**: Forensic schema tables never get used!

---

### Target Flow (Integrated)

```
XML File
  ↓
xmlStreamService.convert()
  ↓
1. CREATE/LOOKUP document record → messaging_documents
  ↓
2. processItem() - Parse XML + NLP analysis
  ↓
   a. Extract entities → browserNlpService.extractEntities()
   b. Detect behaviors → rulesLoader.matchesPattern()
   c. Analyze sentiment → browserNlpService.analyzeLinguisticMarkers()
  ↓
3. CREATE/LOOKUP conversation → messaging_conversations
  ↓
4. flushSupabaseBuffer() - Batch processed records
  ↓
5. supabaseService.uploadBatch() - FORENSIC ROUTING
  ↓
   → messaging_messages (if SMS/MMS)
   → messaging_calls (if call log)
   → messaging_attachments (MMS parts)
   → messaging_behaviors (detected patterns)
   → messaging_entities (people, places, orgs)
```

---

## CRITICAL GAPS

### Gap 1: Missing `detectBehaviors()` Implementation

**File**: `services/browserNlpService.ts`
**Line**: 355 (referenced but not implemented)

**Needed**:
```typescript
detectBehaviors(text: string): BehaviorMatch[] {
    const matches: BehaviorMatch[] = [];
    const rules = await RulesLoader.getInstance().getBehaviorPatterns();

    for (const [category, patterns] of Object.entries(rules)) {
        const result = rulesLoader.matchesPattern(text, patterns);
        if (result.matched) {
            matches.push({
                category,
                matched_pattern: result.matchedText,
                matched_text: result.matchedText, // ACTUAL words
                start_char: text.indexOf(result.matchedText),
                end_char: text.indexOf(result.matchedText) + result.matchedText.length,
                context_before: ...,
                context_after: ...,
                confidence: 0.90,
                severity: determineSeverity(category),
                detection_method: 'rule_based'
            });
        }
    }
    return matches;
}
```

---

### Gap 2: Document Metadata Extraction

**File**: `services/xmlStreamService.ts`
**Location**: Before `convert()` loop starts

**Needed**:
```typescript
// Extract filename, hash, acquisition metadata
const documentRecord = {
    id: UuidV7.generate(),
    filename: this.metadata.originalFileName,
    file_hash: await this.calculateFileHash(),
    acquired_by: this.config.acquiredBy || 'Salem (Device Owner)',
    acquired_date: new Date().toISOString(),
    acquisition_method: 'XML Export',
    source_label: this.config.sourceLabel,
    file_size_bytes: this.metadata.fileSize
};

// Insert into messaging_documents
const documentId = await this.supabase.createDocument(documentRecord);
```

---

### Gap 3: Conversation Grouping

**File**: `services/xmlStreamService.ts`
**Location**: Inside `convert()` loop after `processItem()`

**Needed**:
```typescript
// Group by phone number + platform
const conversationKey = `${parsed.address}_${itemTag}`;
let conversationId = conversationCache.get(conversationKey);

if (!conversationId) {
    conversationId = await this.supabase.createConversation({
        document_id: documentId,
        platform: itemTag, // 'sms', 'mms', 'facebook'
        participants: [parsed.address, 'ME'],
        start_date: parsed.date_iso,
        message_count: 0
    });
    conversationCache.set(conversationKey, conversationId);
}

parsed.conversation_id = conversationId;
```

---

### Gap 4: Message vs Call Routing

**File**: `services/supabaseService.ts:150-191`
**Location**: Complete rewrite of `uploadBatch()`

**Needed**:
```typescript
async uploadBatch(records: any[], documentId: string) {
    const messages = [];
    const calls = [];
    const attachments = [];
    const behaviors = [];
    const entities = [];

    for (const record of records) {
        if (record.type === 'call') {
            calls.push({
                id: record.uuid_v7,
                conversation_id: record.conversation_id,
                phone_number: record.address,
                call_type: record.call_type,
                timestamp: record.date_iso,
                duration: record.duration
            });
        } else {
            // SMS/MMS message
            messages.push({
                id: record.uuid_v7,
                conversation_id: record.conversation_id,
                sender: record.address,
                content: record.body,
                timestamp: record.date_iso,
                // NLP fields
                contains_apology: record.nlp_markers?.contains_apology,
                contains_blame: record.nlp_markers?.contains_blame,
                sentiment: record.nlp_markers?.sentiment,
                word_count: record.nlp_markers?.word_count
            });

            // Detected behaviors
            if (record.behaviors && record.behaviors.length > 0) {
                record.behaviors.forEach(b => {
                    behaviors.push({
                        message_id: record.uuid_v7,
                        category: b.category,
                        matched_text: b.matched_text, // ACTUAL words
                        confidence: b.confidence,
                        severity: b.severity
                    });
                });
            }

            // MMS attachments
            if (record._parts) {
                record._parts.forEach(part => {
                    attachments.push({
                        message_id: record.uuid_v7,
                        content_type: part.ct,
                        storage_path: part.storage_path
                    });
                });
            }
        }
    }

    // Batch insert to forensic tables
    await this.client.from('messaging_messages').insert(messages);
    await this.client.from('messaging_calls').insert(calls);
    await this.client.from('messaging_attachments').insert(attachments);
    await this.client.from('messaging_behaviors').insert(behaviors);
}
```

---

### Gap 5: NLP Integration Point

**File**: `services/xmlStreamService.ts`
**Location**: Inside `processItem()` or `convert()` loop

**Needed**:
```typescript
// After parsing XML
if (parsed.body && this.nlpService.isEnabled()) {
    const analysis = this.nlpService.analyzeMessage(parsed.uuid_v7, parsed.body);

    parsed.nlp_markers = analysis.linguistic_markers;
    parsed.word_count = analysis.word_count;
    parsed.character_count = analysis.character_count;
    parsed.entities = analysis.entities;
    parsed.behaviors = analysis.behaviors; // From detectBehaviors()
}
```

---

## INTEGRATION CHECKLIST

### Phase 1: Core Schema Deployment ⏳
- [ ] Run `npm install` to install compromise, node-nlp
- [ ] Deploy forensic schema to Supabase
- [ ] Verify all 10 tables created
- [ ] Verify triggers working (content_hash auto-generation)
- [ ] Verify 25+ indexes created

### Phase 2: Implement Missing Methods ⏳
- [ ] Implement `browserNlpService.detectBehaviors()` method
- [ ] Integrate RulesLoader into browserNlpService
- [ ] Test behavior detection with sample messages

### Phase 3: Supabase Upload Rewrite ⏳
- [ ] Add `createDocument()` method to supabaseService
- [ ] Add `createConversation()` method to supabaseService
- [ ] Rewrite `uploadBatch()` to route to forensic tables
- [ ] Add message vs call routing logic
- [ ] Add attachment handling
- [ ] Add behavior insertion

### Phase 4: XML Processor Integration ⏳
- [ ] Add document creation at start of `convert()`
- [ ] Add conversation caching/lookup
- [ ] Add NLP analysis to `processItem()` or loop
- [ ] Pass NLP results to buffer
- [ ] Update `flushSupabaseBuffer()` to pass documentId

### Phase 5: End-to-End Testing ⏳
- [ ] Upload small XML file (10-50 records)
- [ ] Verify document record created
- [ ] Verify conversations grouped correctly
- [ ] Verify messages vs calls routed correctly
- [ ] Verify attachments linked
- [ ] Verify behaviors detected
- [ ] Verify actual matched text captured
- [ ] Verify chain of custody preserved (hashes, timestamps)

---

## ANSWERING USER QUESTION

**User Asked**: "Should we actually create a database for some tables in the database for our behaviors just in general for later use and other things"

### Answer: YES, but not urgent for current phase

**What We Already Have**:
1. `messaging_behavior_categories` - 10 pre-loaded categories with MCL mappings
2. `messaging_behaviors` - Table for storing detected pattern instances

**What You're Suggesting** (Smart Idea):
Create additional tables to store the **RULES/PATTERNS THEMSELVES** in the database:

```sql
-- Store behavioral pattern definitions
CREATE TABLE behavior_rule_patterns (
    id UUID PRIMARY KEY,
    category_id UUID REFERENCES messaging_behavior_categories(id),
    pattern_text TEXT NOT NULL,  -- "motherfucker", "you never", "crackhead"
    pattern_type VARCHAR(20),    -- 'string', 'regex'
    severity VARCHAR(20),
    confidence_weight DECIMAL(3,2),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    created_by VARCHAR(100),
    version INTEGER DEFAULT 1
);

-- Store case-specific entity definitions
CREATE TABLE case_entities (
    id UUID PRIMARY KEY,
    entity_type VARCHAR(50),  -- 'person', 'place', 'substance'
    name VARCHAR(255),
    aliases TEXT[],
    notes TEXT,
    case_id UUID  -- For multi-case support
);
```

**Benefits**:
- ✅ Version control for rules (track changes over time)
- ✅ Different rule sets for different cases
- ✅ Web UI for rule editing (lawyers can add patterns)
- ✅ Collaborative rule development
- ✅ A/B testing of pattern effectiveness
- ✅ Export/import rule sets

**Current Status**:
- ❌ NOT urgent - rules are embedded in `rulesLoader.ts` (works fine for now)
- ✅ YAML files in `ConflictAnalysisApp/rules/` are the source of truth
- ⏳ Database storage would be Phase 3+ (after core ingestion works)

**Recommendation**:
- **NOW**: Focus on integration (get ingestion working with embedded rules)
- **LATER**: Add rule management tables when building web UI for case review

---

## PRIORITY ORDER

### URGENT (Do Now)
1. Install NLP dependencies: `npm install`
2. Implement `detectBehaviors()` method
3. Rewrite `supabaseService.uploadBatch()` for forensic tables
4. Add document/conversation creation

### HIGH (This Week)
5. Integrate NLP into XML processor
6. Test with small XML file
7. Verify chain of custody working

### MEDIUM (Next Week)
8. Performance optimization
9. Error handling improvements
10. Build web UI for reviewing ingested data

### LOW (Future)
11. Database-backed rule storage
12. Multi-case support
13. Advanced macro-analysis (gaslighting, DARVO cycles)

---

## FILE MODIFICATION SUMMARY

### Files That Need Changes:
1. ✏️ `services/browserNlpService.ts` - Add `detectBehaviors()` method
2. ✏️ `services/supabaseService.ts` - Rewrite `uploadBatch()`, add helper methods
3. ✏️ `services/xmlStreamService.ts` - Add document/conversation creation, NLP integration

### Files That Are Complete:
1. ✅ `services/rulesLoader.ts` - All 21 categories loaded
2. ✅ `services/aiService.ts` - Multi-provider support ready
3. ✅ `ConflictAnalysisApp/rules/behaviors.yaml` - Source of truth
4. ✅ Schema SQL (lines 193-610) - Ready to deploy

---

## NEXT STEP

**Start with Phase 1**: Install dependencies and deploy schema
```bash
cd "C:\Users\matts\Downloads\massive-xml-to-csv-converter (2)"
npm install
```

Then test schema deployment through the UI.

**Then Phase 2**: Implement missing `detectBehaviors()` method and connect everything.

---

## REUSABILITY NOTES

**Multi-Platform Support** ✅:
- Platform field in conversations table ('sms', 'mms', 'facebook', 'whatsapp')
- Generic XML/JSON parsing (not locked to SMS format)
- Configurable rules path in RulesLoader

**Multi-Case Support** 🔄:
- Add `case_id` column to documents table
- Different rule sets per case
- Isolated data per case

**Multi-File-Type Support** ✅:
- XML (current)
- JSON (easy to add)
- CSV (easy to add)
- Direct API ingestion (future)

---

**END OF ANALYSIS**

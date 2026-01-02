# Salem Forensic Ingestion - Implementation Status

## ✅ COMPLETED

### 1. Forensic Database Schema
**File**: `services/supabaseService.ts:193-610`

Court-admissible PostgreSQL schema with:
- `messaging_documents` - Source file chain of custody
- `messaging_conversations` - Thread grouping across platforms
- `messaging_messages` - Core forensic message records
- `messaging_calls` - Forensic call log records
- `messaging_attachments` - MMS/media files
- `messaging_behaviors` - Detected behavioral patterns
- `messaging_behavior_categories` - Reference table with 10 coercive control categories
- `mcl_factors` - Michigan Child Custody Act (MCL 722.23) factors A-L
- `messaging_audit_log` - Forensic audit trail

**Auto-Triggers**:
- Content hash generation on insert/update
- Behavior count updates when behaviors detected
- Conversation stats updates when messages added

**Indexes**: 25+ optimized indexes for performance

---

### 2. Multi-Provider AI Service ✅
**File**: `services/aiService.ts`

Supports multiple AI providers (no vendor lock-in):
- OpenRouter (default if `VITE_OPENROUTER_API_KEY` set)
- Anthropic Claude
- OpenAI
- Google Gemini

Auto-detects from environment variables.

---

### 3. Browser NLP Service (Built, Not Integrated Yet)
**File**: `services/browserNlpService.ts`

**Libraries**:
- `compromise` (180kb) - Fast entity extraction, rule-based pattern matching
- `node-nlp (nlp.js)` - Sentiment analysis (AFINN + negation support)

**Micro-Level Analysis** (Surface-Level Only):
- ✅ Entity extraction (people, places, phones, emails, dates)
- ✅ Simple content flags (apology, blame, threat, minimizing, profanity)
- ✅ Linguistic markers (questions, exclamations, caps ratio)
- ✅ Sentiment (positive/negative/neutral)
- ✅ Word/character counts
- ✅ **CAPTURES ACTUAL WORDS** - Not just boolean flags!

**What It Doesn't Do** (Macro Analysis - Later):
- ❌ Complex pattern detection (gaslighting, DARVO cycles)
- ❌ Multi-message analysis
- ❌ Temporal pattern detection
- ❌ Cross-platform correlation

---

### 4. Rules System
**File**: `services/rulesLoader.ts`
**Source**: `ConflictAnalysisApp/rules/*.yaml`

**Loaded Rules**:
- `behaviors.yaml` - 20+ behavioral pattern categories with actual phrases
- `entities.yaml` - Case-specific people, places, platforms, substances
- Ready to extend with additional YAML files

**Key Features**:
- Configurable rules path (reusable across different cases)
- Captures ACTUAL matched text (e.g., "motherfucker", "psycho", "crackhead")
- Position tracking (start_char, end_char, context)

**Pattern Categories Loaded**:
1. manipulation_gaslighting
2. projection_blameshift
3. character_assassination
4. parental_alienation
5. coercive_control
6. substance_weaponization
7. reactive_abuse_gotcha
8. dismissiveness
9. ...and more

---

## 🔧 IN PROGRESS

### Update Supabase Upload to Forensic Schema
**File**: `services/supabaseService.ts:150-191`

**Current**: Uploads to simple flat `messages` table

**Needed**:
1. Create `messaging_documents` record first (source file metadata)
2. Create/find `messaging_conversations` record (group by thread/phone)
3. Insert into `messaging_messages` table (not old `messages` table)
4. Insert into `messaging_calls` table (if call logs)
5. Insert into `messaging_attachments` table (MMS parts)

---

### Update XML Processor
**File**: `services/xmlStreamService.ts`

**Current**: Routes everything to simple tables

**Needed**:
1. Extract document metadata → `messaging_documents`
2. Group messages by conversation → `messaging_conversations`
3. Route SMS/MMS → `messaging_messages`
4. Route call logs → `messaging_calls`
5. Route MMS parts → `messaging_attachments`

---

## 📦 NEXT STEPS

### 1. Install Dependencies
```bash
npm install
```

This installs:
- `compromise` - Rule-based NLP (180kb)
- `node-nlp` - Sentiment analysis
- `wink-nlp` - NOT installed (decided not needed)
- `@xenova/transformers` - NOT installed (too heavy for micro-analysis)

### 2. Test Schema Deployment
```bash
# Start dev server
npm run dev

# In browser:
1. Open Supabase settings
2. Enter URL, API key, Personal Access Token
3. Click "Deploy Schema"
4. Verify tables created in Supabase dashboard
```

### 3. Update Upload Logic
Modify `services/supabaseService.ts` `uploadBatch()` method to:
- Write to forensic tables instead of flat tables
- Handle document/conversation creation
- Route messages vs calls correctly

### 4. Test End-to-End
Upload a small XML file and verify:
- Document record created
- Conversations grouped
- Messages inserted with proper references
- Calls in separate table
- Attachments linked correctly

---

## 🎯 DESIGN PHILOSOPHY

### Two-Stage Analysis

**STAGE 1: Browser Ingestion (Current Focus)**
- **Micro-level tagging** - Fast, surface-level flags
- Capture actual words, entities, simple patterns
- Store raw data + basic flags in forensic schema
- NO complex behavioral analysis

**STAGE 2: Post-Processing (Later)**
- **Macro-level analysis** - After all data centralized
- Gaslighting patterns (contradictions across messages)
- DARVO cycles (victim reversal over time)
- Escalation detection (sentiment degradation)
- Cross-platform correlation
- Temporal pattern analysis
- Likely: Python + LLM-based analysis

---

## 📁 FILE STRUCTURE

```
massive-xml-to-csv-converter/
├── services/
│   ├── supabaseService.ts       ← Forensic schema ✅
│   ├── aiService.ts              ← Multi-provider AI ✅
│   ├── browserNlpService.ts      ← NLP service ✅ (not integrated)
│   ├── rulesLoader.ts            ← Rules system ✅
│   └── xmlStreamService.ts       ← Needs update 🔧
├── ConflictAnalysisApp/
│   └── rules/
│       ├── behaviors.yaml        ← Loaded ✅
│       ├── entities.yaml         ← Loaded ✅
│       └── ...
├── python-backend/               ← Built but not needed yet
│   ├── nlp_service.py
│   └── requirements.txt
└── package.json                  ← Dependencies defined ✅
```

---

## 🚀 READY TO TEST

The forensic schema and NLP service are **built and ready**. Next step is updating the XML processor to actually USE them.

Shall I proceed with updating the Supabase upload logic?

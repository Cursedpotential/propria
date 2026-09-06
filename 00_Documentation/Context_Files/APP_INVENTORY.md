# TheBigOne - Complete App Inventory

> _Naming (D-140, 2026-09-05; applied 2026-09-06): this product is **vestigia** (formerly traceIQ / TraceIQ - Latin: footprints, tracks). Working copy: `probata/modules/vestigia/` (directory rename from `modules/traceIQ/` landed 2026-09-06; old name kept as a junction). GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._

Generated: 2025-12-30

## PROBLEM IDENTIFIED
Previous agents created a mess:
1. Multiple versions of same apps scattered across directories
2. Nested copies inside copies (e.g., TraceIQ/traceiq/)
3. Numbered iteration files (helpers (2).py, orchestrator (6).py) mixed with clean versions
4. Some apps may have been "chopped" - components separated into type-based folders

---

## CATEGORY 1: TIMELINE/LOCATION FORENSICS APPS

### 1A. TraceIQ_Complete/ (77.89 MB)
**Location:** `C:\Users\matts\AI Workspace\TheBigOne\TraceIQ_Complete`
**Type:** Flask Python backend for Google Timeline JSON processing
**Structure:**
- Root level: Clean version (app.py, 6 .py files)
- `traceiq/` subdirectory: Messy iteration versions with numbered files

**Root Files:**
- app.py (main Flask app - 365 lines)
- anylizer.py
- collect_and_zip_refs.py
- import_cache_batches.py
- precision.py
- quick_audit.py

**Nested traceiq/ has iterations:**
- helpers (2).py
- orchestrator (6).py
- pass4_5_generate_api_orders (2).py
- pass4_analytics (3).py
- resolve_geocodes (4).py
- sqlite_schema (5).sql
- human_readable_view (4).sql
- evidence_processor_v4.js, evidence_processor_v7.js

---

### 1B. TraceIQ_Timeline_App/ (77.91 MB)
**Location:** `C:\Users\matts\AI Workspace\TheBigOne\TraceIQ_Timeline_App`
**Type:** Flask Python backend for Google Timeline JSON processing
**Structure:** Same nested structure as Complete

**Root Files (14 .py files - MORE than Complete):**
- app.py (main Flask app - 365 lines)
- anylizer.py
- collect_and_zip_refs.py
- import_cache_batches.py
- precision.py
- quick_audit.py
- **UNIQUE TO THIS VERSION:**
  - fix_json_manually.py
  - fix_place_cache.py
  - logging_config.py
  - overnight_analyzer.py
  - schedule_analyzer.py
  - test_db_connection.py
  - utils_io.py
  - validate_timeline_data.py

**Nested traceiq/ has same iterations as Complete**

---

### 1C. location-admin/ (2.06 MB)
**Location:** `C:\Users\matts\AI Workspace\TheBigOne\location-admin`
**Type:** React + Vite + TypeScript admin panel
**Purpose:** NOT a TraceIQ duplicate - manages Supabase geocode_cache_google table
**Stack:** React, MUI, Supabase, Gemini AI

**Files:**
- src/App.tsx (main component)
- src/supabaseClient.ts
- server.js (Express backend?)
- agent_prompt.txt

---

## CATEGORY 2: VOICE/TRANSCRIPT ANALYSIS APPS

### 2A. story-voice/ (0.03 MB)
**Location:** `C:\Users\matts\AI Workspace\TheBigOne\story-voice`
**Type:** Python Gradio backend + HTML frontend
**Purpose:** Transcript analysis with Gemini 3 Pro

**Files:**
- backend/app.py (303 lines) - Gradio app
- backend/requirements.txt
- frontend/index.html

**Features:**
- Calls Gemini 3 Pro API
- Extracts entities, timeline, patterns, themes, emotional markers
- Saves to Supabase + Qdrant vector DB
- Has profanity filter (CLEAN_TRANSCRIPT_PROMPT)

---

### 2B. voice app react/ (0.04 MB)
**Location:** `C:\Users\matts\AI Workspace\TheBigOne\voice app react`
**Type:** React single-file app
**Purpose:** Voice/transcript processing

**Files:**
- index.tsx (44KB - everything in one file)
- index.html
- package.json
- vite.config.ts

---

### 2C. Chronicle_Voice_App/ (inside Context analysis extraction apps)
**Location:** `C:\Users\matts\AI Workspace\TheBigOne\Context analysis extraction apps\Chronicle_Voice_App`
**Type:** Full React + TypeScript application
**Purpose:** Most feature-rich voice app

**Files:**
- App.tsx (755 lines)
- components/ directory
- utils/ directory
- types.ts

**Features:**
- VoiceSession, Timeline, KnowledgeGraph, Reports, ContextGrid components
- Multi-database: Supabase + Weaviate + Neo4j
- Cloud sync
- Preloaded entities (Rita, Rick, Jessica, etc.)
- GoogleGenAI Live API (gemini-2.5-flash-native-audio-preview)

---

### 2D. ai_studio_app/ (0.05 MB)
**Location:** `C:\Users\matts\AI Workspace\TheBigOne\ai_studio_app`
**Type:** Google AI Studio generated React app
**Purpose:** Voice/behavioral analysis

**Files:**
- index.tsx (single file)
- package.json
- README.md (points to AI Studio URL)

---

## CATEGORY 3: CONFLICT/EVIDENCE ANALYSIS APPS

### 3A. ConflictAnalysisApp/ (0.99 MB)
**Location:** `C:\Users\matts\AI Workspace\TheBigOne\ConflictAnalysisApp`
**Type:** Python analysis toolkit
**Purpose:** Court evidence parsing and analysis

**Files (11 .py):**
- app.py
- parsers.py
- taggers.py
- sequences.py
- rules_loader.py
- script.py, script_1.py, script_2.py
- find_comply.py
- preprocess_for_autopsy.py
- sms_backup_parser.py

**Data files:**
- Multiple .xlsx and .csv court evidence files
- rules/ directory

---

### 3B. forensic-data-refinery/ (0.04 MB)
**Location:** `C:\Users\matts\AI Workspace\TheBigOne\forensic-data-refinery`
**Type:** React + TypeScript
**Purpose:** Data processing UI

**Files:**
- App.tsx
- components/
- hooks/
- lib/
- types.ts
- worker.js

---

## CATEGORY 4: VIDEO ANALYSIS

### 4A. Video_Analyzer_App/
**Location:** `C:\Users\matts\AI Workspace\TheBigOne\Context analysis extraction apps\Video_Analyzer_App`
**Type:** React + TypeScript
**Purpose:** Video forensic analysis

**Files:**
- App.tsx
- components/
- hooks/
- utils/

---

## CATEGORY 5: UTILITIES (Not apps - helper tools)

### Utilities/ (578 MB - mostly due to stanford-corenlp)
Contains standalone tools, not integrated apps:
- Chunker/
- Document-Analyser-MCP-main/
- Manipulative-Expression-Recognition-main/
- NLP-Toxicity-Analyzer/
- TimelineExtractor-master/
- etc.

---

## NEXT STEPS

### For TraceIQ (Category 1A/1B):
1. Compare root app.py files between Complete and Timeline_App
2. Identify UNIQUE code in each version's nested traceiq/ folder
3. Extract valuable functions from iterations (orchestrator (6).py, etc.)
4. Merge into single consolidated version
5. Archive duplicates

### For Voice Apps (Category 2A-2D):
1. Analyze features in each version
2. Identify unique code/integrations in each
3. Determine which is the best base
4. Extract valuable features from others
5. Merge and consolidate

---

## STATUS
- [ ] TraceIQ versions diffed and merged
- [ ] Voice apps analyzed and merged
- [ ] Numbered iteration files processed
- [ ] Chopped components reassembled (if any found)

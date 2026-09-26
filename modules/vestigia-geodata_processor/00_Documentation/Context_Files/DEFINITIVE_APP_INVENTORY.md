# DEFINITIVE APP INVENTORY - TheBigOne

> _Naming (D-140, 2026-09-05; applied 2026-09-06): this product is **vestigia** (formerly traceIQ / TraceIQ - Latin: footprints, tracks). Working copy: `probata/modules/vestigia/` (directory rename from `modules/traceIQ/` landed 2026-09-06; old name kept as a junction). GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._

**Generated:** 2025-12-30
**Purpose:** Identify all distinct applications, duplicates, and scattered components

---

## DISCOVERY SUMMARY

Found **31 entry point files** (app.py, App.tsx, index.tsx, main.py) across the workspace.
After analysis, these represent approximately **12 distinct application concepts** with multiple duplicates/iterations.

---

## CATEGORY 1: TRACEIQ / TIMELINE PROCESSING (Google Timeline Forensics)

### Apps Found:
| Location | Type | Notes |
|----------|------|-------|
| `TraceIQ_Complete/app.py` | Flask | Root-level "clean" version |
| `TraceIQ_Complete/traceiq/app.py` | Flask | NESTED copy with iteration files |
| `TraceIQ_Timeline_App/app.py` | Flask | Another root-level version |
| `TraceIQ_Timeline_App/traceiq/app.py` | Flask | NESTED copy (duplicate of above) |

### Analysis:
- **TraceIQ_Complete** and **TraceIQ_Timeline_App** are near-identical at root level
- **TraceIQ_Timeline_App** has 8 EXTRA utility scripts at root:
  - `overnight_analyzer.py` - Analyze overnight stays
  - `schedule_analyzer.py` - Analyze schedules
  - `fix_json_manually.py` - JSON repair tool
  - `fix_place_cache.py` - Cache repair tool
  - `logging_config.py` - Logging configuration
  - `test_db_connection.py` - DB testing
  - `utils_io.py` - I/O utilities
  - `validate_timeline_data.py` - Data validation
- Both have nested `traceiq/` subdirectory with NUMBERED ITERATION FILES:
  - `helpers (2).py`
  - `orchestrator (6).py`
  - `pass4_analytics (3).py`
  - `pass4_5_generate_api_orders (2).py`
  - `resolve_geocodes (4).py`
  - `sqlite_schema (5).sql`
  - `human_readable_view (4).sql`
  - `evidence_processor_v4.js`, `evidence_processor_v7.js`

### RECOMMENDATION:
**BEST BASE:** `TraceIQ_Timeline_App/` (has extra utilities)
**ACTION:** 
1. Diff the numbered iteration files against base versions
2. Extract unique code from iterations before archiving
3. Merge TraceIQ_Complete's nested traceiq/ unique content if any
4. Archive duplicates

---

## CATEGORY 2: LOCATION-ADMIN (Supabase Geocode Cache Manager)

### Apps Found:
| Location | Type | Notes |
|----------|------|-------|
| `location-admin/src/App.tsx` | React/Vite | Supabase geocode_cache_google admin |

### Analysis:
- **NOT a TraceIQ duplicate** - completely different app
- React + MUI + Supabase client
- Manages geocode cache table
- Has `server.js` (Express backend?)
- Standalone app, no duplicates found

### RECOMMENDATION:
**STATUS:** Keep as-is (single instance)

---

## CATEGORY 3: VOICE/TRANSCRIPT ANALYSIS APPS

### Apps Found:
| Location | Type | Notes |
|----------|------|-------|
| `story-voice/backend/app.py` | Python/Gradio | Gemini 3 Pro, Supabase+Qdrant |
| `voice app react/index.tsx` | React | Single-file minimal implementation |
| `Chronicle_Voice_App/App.tsx` | React/TS | Most complete - multi-DB, components |
| `Chronicle_Voice_App/index.tsx` | React | Entry point for above |
| `Chronicle_Voice_App/chronicle_-empathetic-timeline.../App.tsx` | React | DUPLICATE nested inside Chronicle |
| `ai_studio_app/index.tsx` | React | AI Studio generated, minimal |

### Analysis:
- **Chronicle_Voice_App** is the MOST COMPLETE:
  - Components: VoiceSession, Timeline, KnowledgeGraph, Reports, ContextGrid
  - Multi-database: Supabase + Weaviate + Neo4j
  - GoogleGenAI Live API (gemini-2.5-flash-native-audio-preview)
  - Preloaded entities for the case
  - Has nested duplicate inside itself: `chronicle_-empathetic-timeline-investigator/`
  
- **story-voice** has DIFFERENT approach:
  - Python Gradio backend
  - Profanity filter (CLEAN_TRANSCRIPT_PROMPT)
  - Gemini 3 Pro API
  - Saves to Supabase + Qdrant vector DB
  
- **voice app react** is minimal single-file (44KB index.tsx)
- **ai_studio_app** is AI Studio generated minimal

### RECOMMENDATION:
**BEST BASE:** `Chronicle_Voice_App/` (most features)
**EXTRACT FROM:**
- `story-voice/` - Profanity filter, Qdrant integration, Gradio patterns
- `voice app react/` - Check for any unique UI patterns
**ARCHIVE:** 
- `Chronicle_Voice_App/chronicle_-empathetic-timeline.../` (nested duplicate)
- `ai_studio_app/` (unless needed for reference)

---

## CATEGORY 4: CONFLICT/EVIDENCE ANALYSIS

### Apps Found:
| Location | Type | Notes |
|----------|------|-------|
| `ConflictAnalysisApp/app.py` | Python | Court evidence parsing toolkit |
| `massive-xml-to-csv-converter (2)/ConflictAnalysisApp/app.py` | Python | DUPLICATE inside xml converter |

### Analysis:
- **ConflictAnalysisApp** is a Python toolkit for court evidence:
  - `parsers.py`, `taggers.py`, `sequences.py`
  - `rules_loader.py` with `rules/` directory
  - Multiple .xlsx and .csv court evidence files
  - `sms_backup_parser.py`, `preprocess_for_autopsy.py`
  
- There's a DUPLICATE inside `massive-xml-to-csv-converter (2)/`

### RECOMMENDATION:
**BEST BASE:** `ConflictAnalysisApp/` (root level)
**ACTION:** Verify duplicate is identical, then delete from xml-converter folder

---

## CATEGORY 5: VIDEO ANALYZER

### Apps Found:
| Location | Type | Notes |
|----------|------|-------|
| `Video_Analyzer_App/App.tsx` | React | Root level |
| `Video_Analyzer_App/index.tsx` | React | Entry point |
| `Video_Analyzer_App/forensic-video-analyzer/App.tsx` | React | NESTED duplicate |
| `Video_Analyzer_App/forensic-video-analyzer/index.tsx` | React | NESTED duplicate |

### Analysis:
- React + TypeScript video forensic analyzer
- Has components/, hooks/, utils/ structure
- Contains NESTED duplicate: `forensic-video-analyzer/`

### RECOMMENDATION:
**BEST BASE:** `Video_Analyzer_App/` root level files
**ARCHIVE:** Nested `forensic-video-analyzer/` after verifying identical

---

## CATEGORY 6: FORENSIC DATA REFINERY

### Apps Found:
| Location | Type | Notes |
|----------|------|-------|
| `forensic-data-refinery/App.tsx` | React | Data processing UI |
| `forensic-data-refinery/index.tsx` | React | Entry point |

### Analysis:
- React + TypeScript
- Has components/, hooks/, lib/, worker.js
- Single instance, no duplicates

### RECOMMENDATION:
**STATUS:** Keep as-is (single instance)

---

## CATEGORY 7: CHAT PARSER / MINER

### Apps Found:
| Location | Type | Notes |
|----------|------|-------|
| `chat-parser-workspace/app.py` | Flask | Chat history processor |
| `chat-parser-workspace/Chat_Miner_App/app.py` | Flask | NESTED duplicate |

### Analysis:
- Flask app for processing chat exports
- PostgreSQL integration
- Has `docs/` with implementation guides
- NESTED duplicate: `Chat_Miner_App/`

### RECOMMENDATION:
**BEST BASE:** `chat-parser-workspace/` root
**ARCHIVE:** Nested `Chat_Miner_App/` after verification

---

## CATEGORY 8: CHUNKER (Document Chunking Tool)

### Apps Found:
| Location | Type | Notes |
|----------|------|-------|
| `Utilities/Chunker/main.py` | Python/Flet | Cross-platform document chunker |
| `Chronicle_Voice_App/Chunker/main.py` | Python/Flet | DUPLICATE nested in Chronicle |

### Analysis:
- Python Flet app for intelligent document chunking
- Has parsers/, chunkers/, exporters/, schemas/, ui/ structure
- Supports HTML, CSV, with custom schemas
- DUPLICATE nested inside Chronicle_Voice_App

### RECOMMENDATION:
**BEST BASE:** `Utilities/Chunker/`
**ARCHIVE:** `Chronicle_Voice_App/Chunker/` (should not be inside Voice app)

---

## CATEGORY 9: DIRECTORY SCANNER

### Apps Found:
| Location | Type | Notes |
|----------|------|-------|
| `DirectoryScanner/frontend/app.TSX` | React | Directory scanning UI |

### Analysis:
- React frontend for directory scanning
- 22MB total (likely includes scanned data)
- Single instance

### RECOMMENDATION:
**STATUS:** Keep as-is, evaluate if still needed

---

## CATEGORY 10: MASSIVE XML TO CSV CONVERTER

### Apps Found:
| Location | Type | Notes |
|----------|------|-------|
| `massive-xml-to-csv-converter (2)/App.tsx` | React | XML processing |
| `massive-xml-to-csv-converter (2)/index.tsx` | React | Entry point |

### Analysis:
- 407MB total (huge - likely has data files)
- Contains ORPHANED ConflictAnalysisApp/ inside
- React-based XML processor

### RECOMMENDATION:
**ACTION:** 
1. Remove orphaned ConflictAnalysisApp/ from inside
2. Evaluate if app is still needed
3. Consider archiving if data processing is complete

---

## CATEGORY 11: EXTERNAL TOOLS

### Apps Found:
| Location | Type | Notes |
|----------|------|-------|
| `EXTERNAL_TOOLS/Context-Cruncher-main/app.py` | Python | Third-party tool |

### Analysis:
- Third-party/downloaded tool
- Not custom-built

### RECOMMENDATION:
**STATUS:** Keep in EXTERNAL_TOOLS (correctly categorized)

---

## CATEGORY 12: UTILITIES (Tether/TetherPro, NLP-Toxicity)

### Apps Found:
| Location | Type | Notes |
|----------|------|-------|
| `Utilities/Tether/app.py` | Python | Tether tool |
| `Utilities/TetherPro/app.py` | Python | Tether Pro version |
| `Utilities/NLP-Toxicity-Analyzer.../app.py` | Python/Streamlit | NLP toxicity analyzer |

### Analysis:
- These are utility tools, not main applications
- Correctly located in Utilities/

### RECOMMENDATION:
**STATUS:** Keep as-is in Utilities/

---

## PRIORITY ACTION LIST

### HIGH PRIORITY (Merge & Dedupe):

1. **TraceIQ Apps** - Most complex, most duplicates
   - [ ] Diff TraceIQ_Complete vs TraceIQ_Timeline_App root files
   - [ ] Extract unique code from numbered iteration files
   - [ ] Create single consolidated TraceIQ
   - [ ] Archive duplicates

2. **Voice Apps** - Multiple implementations
   - [ ] Use Chronicle_Voice_App as base
   - [ ] Extract profanity filter from story-voice
   - [ ] Extract Qdrant patterns from story-voice
   - [ ] Archive nested duplicate, minimal versions

3. **Nested Duplicates** - Quick wins
   - [ ] Remove Chronicle_Voice_App/chronicle_-empathetic.../
   - [ ] Remove Chronicle_Voice_App/Chunker/ (belongs in Utilities)
   - [ ] Remove Video_Analyzer_App/forensic-video-analyzer/
   - [ ] Remove chat-parser-workspace/Chat_Miner_App/
   - [ ] Remove massive-xml-to-csv/ConflictAnalysisApp/

### MEDIUM PRIORITY:
4. **Evaluate Large Folders**
   - [ ] massive-xml-to-csv-converter (407MB) - still needed?
   - [ ] DirectoryScanner (22MB) - still needed?

---

## NEXT STEPS

Would you like me to:
1. Start diffing the TraceIQ versions to extract unique code?
2. Remove the obvious nested duplicates first?
3. Something else?

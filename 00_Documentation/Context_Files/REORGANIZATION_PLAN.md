# TheBigOne Workspace - Reorganization Plan
**Generated:** 2025-12-30
**Total Files:** 15,500+ (excluding venv/node_modules)
**Duplicate Sets Found:** 1,601

---

## PHASE 1: MOVE 100% DUPLICATES TO _MARKED_FOR_DELETE

### Confirmed 100% Identical (By Hash):

1. **forensic-data-refinery (1)/** → MARK FOR DELETE
   - 100% byte-for-byte identical to `forensic-data-refinery/`

2. **story-voice - 1/** → MARK FOR DELETE
   - 100% identical code to `story-voice/`, just extra nesting level

3. **Root-level config files duplicated in STACK_Deployment/CONFIG_Auth_Keys/**:
   - ` (3).env` → MARK FOR DELETE (duplicate of STACK_Deployment version)
   - `.api-keys-reference.txt` → MARK FOR DELETE (duplicate)
   - `.env.example` → MARK FOR DELETE (duplicate)
   - `.gitignore` → MARK FOR DELETE (duplicate)
   - `claude_desktop_mcp_config.jsonc` → MARK FOR DELETE (duplicate)
   - `client_secret_505045767698-ckdl2qqcpr9hovjhfmvh2ljj67jo1s95.apps.googleusercontent.com.json` → MARK FOR DELETE
   - `client_secret_2_505045767698-ckdl2qqcpr9hovjhfmvh2ljj67jo1s95.apps.googleusercontent.com.json` → MARK FOR DELETE

4. **conversations-20251215_153247-2f08f0ab.json** (16MB) → MARK FOR DELETE
   - Duplicate in `DOCS_Reference/context/`

5. **437 empty (0-byte) files** → Most are `__init__.py` (keep), but scan for accidental empties

---

## PHASE 2: RENAME "COPY OF" FILES (If No Original Exists)

### Files to Check:
1. `Copy of sms_export Matt & Katrina (1) - Copy - Copy.csv`
   - **NO ORIGINAL FOUND** → Rename to: `sms_export_Matt_Katrina.csv`
   - Move to: `DATA/Communications/`

2. `Court Statement Narrative Synthesis - Copy - Copy.docx`
   - Check if original exists → If not, rename to base name

3. `Advanced_Evidentiary_Strategy_MRE_Research - Copy.docx`
   - Check if original exists → If not, rename to base name

4. `AFFIDAVIT%20OF%20MATTHEW%20SALEM - Copy.docx`
   - **URL encoded name** → Rename to: `AFFIDAVIT_OF_MATTHEW_SALEM.docx`
   - Check for original first

5. `Autopsy_Analysis_Keywords_Lucene - Copy.xml`
   - Check if original exists → If not, rename to base name

---

## PHASE 3: DIFF & COMPARE NUMBERED FILES

### Files with (1), (2), (3), etc:

1. **ENV FILES** (CRITICAL - CREDENTIALS):
   - ` (2).env` (25 lines, DB password: `Ms10238512ms!!`)
   - ` (3).env` (77 lines, DB password: `Ms10238512ms!`) ← WRONG
   - `MASTER_ENV_COMPILED.env` (87 lines, labeled "THE CORRECT ONE", DB password: `@@Kailah2020!#`)
   - **DECISION:** Keep MASTER_ENV_COMPILED as `.env`, archive the (2) and (3)

2. **Timeline (1).json** (13MB)
   - Need to check if base `Timeline.json` exists
   - If original exists, diff them to see what changed

3. **SALEM_api_key (1).txt**
   - Check for base `SALEM_api_key.txt`
   - If exists, compare values

4. **_folder_metadata (2).json**
   - Compare with `_folder_metadata.json`
   - Determine which is newer/correct

5. **Analysis Complete_ Salem v. Kinzel - What the Prev (1).md**
   - Check for base version

6. **Reactive behaviors by victims...psychological effects (1).pdf**
   - Check for original PDF

7. **TraceIQ Module Files (Multiple numbered versions within both apps)**:
   - `helpers (2).py`
   - `pass4_5_generate_api_orders (2).py`
   - `resolve_geocodes (4).py`
   - `pass4_analytics (3).py`
   - `orchestrator (6).py`
   - **ACTION:** Need to diff these against base versions (0) or (1)

---

## PHASE 4: APP VERSION COMPARISON & CONSOLIDATION

### A. TraceIQ (2 versions)

**TraceIQ_Timeline_App** = MORE COMPLETE VERSION
- Has 8 additional utilities not in "Complete":
  - `overnight_analyzer.py` - Overnight stay detection
  - `validate_timeline_data.py` - Data integrity validation
  - `schedule_analyzer.py` - Schedule analysis
  - `fix_json_manually.py` - JSON repair
  - `fix_place_cache.py` - Place cache repair
  - `test_db_connection.py` - DB testing
  - `logging_config.py` - Centralized logging
  - `utils_io.py` - I/O utilities

**TraceIQ_Complete** = LESS COMPLETE (ironic name!)
- Missing the 8 utility scripts above
- Same core app.py (first 50 lines identical)

**RECOMMENDATION:**
- **KEEP:** TraceIQ_Timeline_App → Rename to just `TraceIQ/`
- **ARCHIVE:** TraceIQ_Complete → Move to `_ARCHIVE/TraceIQ_Complete_old/`

---

### B. Voice/Story Apps (User says 3-4 versions exist)

**FOUND SO FAR:**
1. **story-voice/** - Python backend + React frontend
2. **story-voice - 1/** - Identical code, extra nesting (DUPLICATE - MARK FOR DELETE)
3. **voice app react/** - React voice app (package.json present)
4. **Context analysis extraction apps/Chronicle_Voice_App/** - Voice/timeline analysis with chunking

**TODO:** Search for other voice app versions user mentioned
- Look for variations like: story-voice-v2, voice-app-old, etc.

**NEXT STEP:** Diff all voice apps to determine:
- Which has most features
- Which has best code quality
- What unique features each has
- Consolidation strategy

---

### C. Context Analysis Apps (3 SEPARATE apps - NOT duplicates)

**User Confirmed These Are Different:**
1. **Chat_Miner_App** - Chat conversation parsing
2. **Chronicle_Voice_App** - Voice/timeline analysis with chunking
3. **Video_Analyzer_App** - Video analysis with forensic capabilities

**STATUS:** Keep all three as separate applications
**ACTION:** Ensure each has clear naming and proper folder structure

---

## PHASE 5: LOGICAL FOLDER STRUCTURE

Proposed structure:
```
TheBigOne/
├── APPLICATIONS/
│   ├── TraceIQ/                    (Timeline & forensic analysis)
│   ├── ConflictAnalysisApp/        (Abuse pattern detection)
│   ├── Chat_Miner_App/             (Chat parsing)
│   ├── Chronicle_Voice_App/        (Voice/timeline analysis)
│   ├── Video_Analyzer_App/         (Video forensics)
│   ├── story-voice/                (Story/voice interface)
│   ├── voice-app-react/            (React voice app)
│   ├── forensic-data-refinery/     (Data refinery)
│   ├── DirectoryScanner/           (Go/Wails scanner)
│   ├── ai_studio_app/              (AI studio)
│   └── location-admin/             (Location management)
├── UTILITIES/
│   ├── Parsers/
│   │   ├── massive-xml-to-csv-converter/
│   │   ├── location-history-json-converter/
│   │   └── parser.py, test_parser.py
│   └── Scripts/
│       └── (utility scripts)
├── DATA/
│   ├── Communications/
│   │   └── sms_export_Matt_Katrina.csv
│   ├── Legal/
│   │   ├── affidavits/
│   │   ├── court_statements/
│   │   └── legal_strategy/
│   ├── Research/
│   │   └── (PDF research papers)
│   └── Timelines/
│       └── Timeline.json (13MB)
├── CONFIGURATION/
│   ├── .env                        (from MASTER_ENV_COMPILED.env)
│   ├── docker-compose files
│   ├── CREDENTIALS/                (move all API keys here)
│   │   ├── client_secret_*.json
│   │   ├── SALEM_api_key.txt
│   │   └── personal_access_token.txt
│   └── STACK_Deployment/           (deployment configs)
├── DOCUMENTATION/
│   ├── Architecture/
│   │   └── ARCHITECTURE_HANDOFF.md
│   ├── Guides/
│   │   └── (markdown guides)
│   └── Reference/
│       └── DOCS_Reference/
├── DEPLOYMENT/
│   ├── docker-compose files
│   └── STACK_Deployment/
├── EXTERNAL_TOOLS/
│   ├── claude-context-master/
│   ├── Context-Cruncher-main/
│   └── Claude-Slash-Commands-main/
├── CONVERSATION_LOGS/
│   ├── gemini_conversations/
│   ├── claude_conversations/
│   └── Context_Logs/
└── _ARCHIVE/
    ├── TraceIQ_Complete_old/
    ├── old_env_files/
    │   ├── (2).env
    │   └── (3).env
    └── _MARKED_FOR_DELETE/
        ├── forensic-data-refinery (1)/
        ├── story-voice - 1/
        └── (duplicate config files)
```

---

## SECURITY NOTES

**CRITICAL - EXPOSED CREDENTIALS:**
The following files contain plaintext credentials and should be:
1. Moved to `CONFIGURATION/CREDENTIALS/` (with restricted permissions)
2. Added to `.gitignore`
3. Never committed to version control

**Files:**
- `MASTER_ENV_COMPILED.env` - Multiple API keys, database passwords
- ` (2).env`, ` (3).env` - Old credentials (archive after verification)
- `SALEM_api_key (1).txt`
- `personal_access_token.txt`
- `client_secret_*.json` (Google OAuth)
- `drive-479520-4833b9f83011.json` (Google Drive credentials)

**Recommendation:** After reorganization, rotate all exposed credentials.

---

## NEXT ACTIONS

1. **User Approval Required:**
   - Approve the proposed folder structure
   - Confirm which voice app versions you want to keep
   - Approve moving duplicates to _MARKED_FOR_DELETE (not deleting)

2. **Automated Tasks:**
   - Create `_MARKED_FOR_DELETE/` folder
   - Move confirmed 100% duplicates
   - Rename "Copy of" files without originals
   - Create new folder structure

3. **Manual Review Required:**
   - Diff numbered files (1), (2), (3)
   - Compare voice app versions
   - Verify TraceIQ consolidation
   - Review ARCHITECTURE_HANDOFF.md copies (may be intentional per-app docs)

**Ready to proceed?** Let me know which phase to start with or if you want to adjust the plan.

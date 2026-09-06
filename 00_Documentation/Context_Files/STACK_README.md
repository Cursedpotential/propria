# TheBigOne: Unified Forensic AI Stack

> _Naming (D-140, 2026-09-05; applied 2026-09-06): this product is **vestigia** (formerly traceIQ / TraceIQ - Latin: footprints, tracks). Working copy: `probata/modules/vestigia/` (directory rename from `modules/traceIQ/` lands with the workspace directory-rename step; old name kept as a junction). GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._

**Root Directory:** `C:\Users\matts\AI Workspace\TheBigOne`
**Generated:** 2026-01-01

## 📂 Stack Hierarchy

### [00_Documentation]
Global documentation, transcripts, and secrets audit.
- `Global_Transcripts/`: Chat logs not tied to a specific app.
- `SECRETS_AUDIT.md`: List of exposed keys to rotate.

### [00_Code_Vault]
Extracted logic modules and snippets from legacy/iterated versions.
- `TraceIQ_Snippets/`: Logic extracted from TraceIQ iterations (v6 orchestrator, v3 analytics).

### [01_Timeline_Forensics]
**Core Domain: Google Location History & Metadata Analysis**
- **TraceIQ_Main/** (formerly TraceIQ_Timeline_App): The primary backend.
  - `src/`: Main application code.
  - `iterations/`: Archived iteration files (`helpers (2).py`, etc).
  - `transcripts/`: Developer chat logs.
- **location-admin/**: React/Supabase admin panel for geocode caching.

### [02_Voice_Analysis]
**Core Domain: Audio Transcripts & Behavioral Analysis**
- **Chronicle_Voice_App/**: Primary React frontend with multi-DB support.
- **story-voice-backend/**: Python/Gradio backend (kept for Qdrant/Profanity logic).

### [03_Evidence_Analysis]
**Core Domain: Text, Video & Document Forensics**
- **ConflictAnalysisApp/**: Python toolkit for court evidence parsing.
- **Video_Analyzer_App/**: React-based video forensic tool.
- **forensic-data-refinery/**: Data cleaning UI.

### [04_Utilities]
**Shared Tools & Micro-services**
- **Data_Converters/**: XML->CSV, JSON converters.
- **Text_Analysis/**: NLP Toxicity, Manipulative Pattern recognition.
- **MCP_Tools/**: Document Analyzer, NotebookLM connectors.
- **Automation/**: n8n-local instance.
- **DirectoryScanner/**: File system analysis tool.

### [99_Archive]
Deprecated or duplicate versions.
- `TraceIQ_Complete_Legacy`
- `voice_app_react_legacy`
- `ai_studio_app_legacy`

---

## 🔄 Versioning Strategy

### Application Iterations
Apps often contain file versions like `app (2).py` or `orchestrator (6).py`.
**Policy:**
1.  **Do not delete** these immediately.
2.  Move them to `[AppName]/iterations/`.
3.  Use **Merge Strategy** (see `MERGE_STRATEGY.md`) to extract unique logic.
4.  Once extracted to `00_Code_Vault`, the iteration file can be archived.

### Transcripts & Context
- Chat logs are now located in `[AppName]/transcripts/`.
- When starting a new task on an app, **check the transcripts folder first** for context.

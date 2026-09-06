# App Merge & Diff Strategy

> _Naming (D-140, 2026-09-05; applied 2026-09-06): this product is **vestigia** (formerly traceIQ / TraceIQ - Latin: footprints, tracks). Working copy: `probata/modules/vestigia/` (directory rename from `modules/traceIQ/` lands with the workspace directory-rename step; old name kept as a junction). GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._

**Target:** Consolidate "Iterated" logic into "Main" branches.

## 🎯 TraceIQ Merge Plan

**Source:** `01_Timeline_Forensics/TraceIQ_Main/iterations/`
**Target:** `01_Timeline_Forensics/TraceIQ_Main/app.py` (and helper modules)

### Identified Iterations:
1.  **Orchestrator Logic** (`orchestrator (6).py`)
    *   *Status:* Extracted to `00_Code_Vault/TraceIQ_Snippets/orchestrator_v6_pipeline.py`.
    *   *Action:* Refactor `app.py` to import this pipeline instead of using inline logic.

2.  **Analytics Logic** (`pass4_analytics (3).py`)
    *   *Status:* Extracted to `00_Code_Vault/TraceIQ_Snippets/analytics_v3_path_inference.py`.
    *   *Action:* Replace current analytics function in `anylizer.py` with this v3 version.

3.  **Geocoding Logic** (`resolve_geocodes (4).py`)
    *   *Status:* Extracted to `00_Code_Vault/TraceIQ_Snippets/geocoding_v4_resolver.py`.
    *   *Action:* Create `utils/geocoding.py` in Main app and paste this logic there.

## 🎯 Voice App Merge Plan

**Source:** `02_Voice_Analysis/story-voice-backend/app.py`
**Target:** `02_Voice_Analysis/Chronicle_Voice_App/`

### Features to Migrate:
1.  **Profanity Filter (`CLEAN_TRANSCRIPT_PROMPT`)**
    *   *Logic:* A specific prompt template for cleaning text.
    *   *Action:* Copy prompt constant to `Chronicle_Voice_App/utils/prompts.ts`.

2.  **Qdrant Integration**
    *   *Logic:* Code for embedding and saving vectors.
    *   *Action:* Chronicle uses Weaviate. Decide if Qdrant is needed. If so, port the Python logic to TypeScript/Supabase Edge Function.

## 🛠️ Automated Diffing Script

To diff future iterations, use this PowerShell command pattern:

```powershell
# Diff iteration against main file
diff (Get-Content "iterations/helper (2).py") (Get-Content "src/helper.py")
```

Or ask the **App Diff Agent** to run a syntax-aware comparison.

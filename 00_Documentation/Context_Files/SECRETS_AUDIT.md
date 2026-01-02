# SECRETS AUDIT REPORT
Generated: 2025-12-30

## CRITICAL FINDINGS - ACTION REQUIRED

The following files contain sensitive information or configuration that needs to be consolidated.
**DO NOT COMMIT THESE FILES.**

### 1. Root Level Secrets
- `C:\Users\matts\AI Workspace\TheBigOne\personal_access_token.txt`
  - **Type:** Raw Access Token (likely GitHub or generic API token)
  - **Action:** Rotate if exposed. Move to `.env` or password manager.

- `C:\Users\matts\AI Workspace\TheBigOne\client_secret_2_505045767698-....json`
- `C:\Users\matts\AI Workspace\TheBigOne\client_secret_505045767698-....json`
  - **Type:** Google OAuth Client Secrets (Desktop/Web App)
  - **Action:** Move to secure `secrets/` folder. Do not leave in root.

### 2. Configuration Files
- `C:\Users\matts\AI Workspace\TheBigOne\.env.example`
  - **Status:** Good template. Contains placeholders for:
    - Supabase (DB, API, Auth)
    - Neo4j
    - Qdrant
    - Cloudflare R2
    - LLM Keys (OpenAI, Anthropic, OpenRouter)
    - LiteLLM
    - Mem0
    - Directus

- `C:\Users\matts\AI Workspace\TheBigOne\litellm_config.yaml`
  - **Status:** Secure. Uses `os.environ` references.
  - **Action:** Ensure all keys referenced here (`ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `OPENROUTER_API_KEY`, `LITELLM_MASTER_KEY`) are present in the final `.env`.

### 3. App-Specific Secrets
- `TraceIQ_Timeline_App/test_db_connection.py` (Likely)
  - **Risk:** Check for hardcoded DB credentials.

- `location-admin/.env.template`
  - **Status:** Check for unique admin keys not in main .env.

- `story-voice/backend/app.py`
  - **Status:** Reads from `os.getenv` (QDRANT_API_KEY). Safe.

## CONSOLIDATION PLAN

1.  Create `unified_template.env` combining all detected requirements.
2.  Create `secrets/` directory (added to .gitignore).
3.  Move client_secret json files to `secrets/`.
4.  Instruct user to fill `unified_template.env` and save as `.env`.

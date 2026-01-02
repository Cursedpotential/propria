# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

---

## Project Overview

**TheBigOne** is a comprehensive forensic legal case management system centered around the "Salem Processing Stack" - a multi-application ecosystem for timeline reconstruction, evidence analysis, and abuse pattern detection. The system is designed around a provenance-based data architecture where three independent applications (Chronicle, Sentinel, Anamnesis) each maintain their own data with strict origin tagging for future conflict resolution.

**Primary Use Case:** Legal case preparation involving timeline reconstruction from multiple evidence sources (voice recordings, chat logs, video footage, location data, documents).

---

## System Architecture

### Three-Application Ecosystem

The system implements a **provenance-based architecture** where data from different sources is tagged and isolated:

1. **Chronicle (App 1)** - User Recall & Voice Interface
   - Captures user's perspective through voice sessions and manual entry
   - All data tagged with `origin: 'Chronicle'`
   - Stores to `chronicle_cases` table
   - Status: Active development

2. **Sentinel (App 2)** - Video Evidence Processing
   - Planned: Video analysis and forensic capabilities
   - Will tag data with `origin: 'Sentinel'`

3. **Anamnesis (App 3)** - Chat Log Mining
   - Planned: Historical communication analysis
   - Will tag data with `origin: 'Anamnesis'`

**Critical Design Pattern:** Each app maintains data isolation via the `origin` field. A future "Merger" system (managed by an Independent Advisor) will reconcile these perspectives, identifying duplicates vs. conflicts.

### Core Data Model

All timeline events follow this provenance pattern:

```typescript
interface TimelineEvent {
  id: string;              // UUID
  origin: string;          // CRITICAL: 'Chronicle' | 'Sentinel' | 'Anamnesis'
  date: string;
  category: EventCategory;
  description: string;
  location?: string;
  witnesses?: string[];
  isSignificant: boolean;
  emotionalContext?: string;
}
```

**Key Reference:** `timeline_schema_design_doc.md` - 32 tables, 100+ indexes, inheritance pattern with `timeline_events` as universal parent.

---

## Technology Stack

### Infrastructure (Docker-based)
- **Database:** Supabase PostgreSQL (direct IPv6 connection)
- **ORM:** Drizzle (type-safe queries, migrations)
- **Admin UI:** Directus (visual data browser, file management)
- **File Storage:** Cloudflare R2 (blob storage for evidence)
- **Graph Database:** Neo4j Aura (entity relationships)
- **Vector Search:** Qdrant Cloud (semantic search)
- **Cache/Queue:** Dragonfly (Redis-compatible)
- **Workflows:** N8N (evidence processing automation)
- **Memory:** Mem0 (AI conversation memory)
- **LLM Gateway:** LiteLLM (routes to OpenRouter, Anthropic, OpenAI, Ollama)
- **Reverse Proxy:** Traefik v3 (HTTPS, Let's Encrypt)

### Applications
- **TraceIQ** - Timeline & forensic analysis (Python)
- **ConflictAnalysisApp** - Abuse pattern detection (Python)
- **Chat Parsers** - Message extraction tools
- **Voice Apps** - Multiple voice interface implementations (React + Python backends)
- **DirectoryScanner** - Go/Wails application for filesystem analysis
- **forensic-data-refinery** - Data processing pipeline

### Languages & Frameworks
- **Python:** Backend services, data processing, ML workflows
- **TypeScript/React:** Frontend applications
- **Go:** DirectoryScanner utility (Wails framework)
- **Node.js:** Various frontend tooling

---

## Development Commands

### Environment Setup

**Required:** Copy `.env.example` to `.env` and fill in credentials:
```bash
cp .env.example .env
```

**Reference:** `MASTER_ENV_COMPILED.env` contains the canonical environment configuration (87 lines, includes all service credentials).

### Docker Stack Deployment

**Full stack deployment:**
```bash
# Using latest version (v3.1)
docker-compose -f docker-compose-salem-v3.1.yml up -d

# Or use unified deployment
docker-compose -f STACK_Deployment/docker-compose-salem-v3.1.yml up -d
```

**Individual services:**
```bash
# Start only core services
docker-compose up -d dragonfly directus n8n

# View logs
docker-compose logs -f [service-name]

# Stop all services
docker-compose down
```

**Service URLs (local development):**
- Traefik Dashboard: `http://localhost:8080`
- Directus Admin: `http://localhost:8055`
- N8N Workflows: `http://localhost:5678`
- LiteLLM Gateway: `http://localhost:4000`
- Open WebUI: `http://localhost:3000`

### Python Applications

**TraceIQ Timeline App (PREFERRED VERSION):**
```bash
cd TraceIQ_Timeline_App

# Install dependencies
pip install -r requirements.txt

# Test database connection
python test_db_connection.py

# Run main app
python app.py

# Utilities
python overnight_analyzer.py     # Overnight stay detection
python validate_timeline_data.py # Data integrity validation
python schedule_analyzer.py      # Schedule analysis
```

**Note:** `TraceIQ_Timeline_App` is the more complete version - has 8 additional utilities not in `TraceIQ_Complete`.

**ConflictAnalysisApp:**
```bash
cd ConflictAnalysisApp
pip install -r requirements.txt
python app.py
```

**Story-Voice Backend:**
```bash
cd story-voice/backend
pip install -r requirements.txt
python app.py
```

### Node.js/React Applications

**AI Studio App:**
```bash
cd ai_studio_app
npm install
npm run dev
```

**Voice App (React):**
```bash
cd "voice app react"
npm install
npm start
```

**Chronicle Voice App:**
```bash
cd "Context analysis extraction apps/Chronicle_Voice_App"
npm install
npm run dev
```

### Utility Scripts

**Find duplicate files:**
```bash
python find_duplicates.py
# Generates DUPLICATE_REPORT.json
```

**Test parsers:**
```bash
python test_parser.py  # Main parser tests
python Utilities/SCRIPTS/test_chatgpt_parser.py
```

---

## Critical Files & Directories

### Configuration
- `.env` - Active environment variables (DO NOT COMMIT)
- `.env.example` - Template for environment setup
- `MASTER_ENV_COMPILED.env` - Reference configuration (87 lines, canonical)
- `docker-compose-salem-v3.1.yml` - Latest stack definition
- `litellm_config.yaml` - LLM routing configuration

### Documentation
- `README.md` - Chronicle app philosophy & data contracts
- `ARCHITECTURE_HANDOFF.md` - System integration contract
- `timeline_schema_design_doc.md` - Database schema (32 tables)
- `unified_deployment_v3.md` - Deployment architecture
- `REORGANIZATION_PLAN.md` - Workspace cleanup plan (1,601 duplicate sets found)

### Data Specifications
- `timeline_parser_prompt.md` - Timeline extraction guidelines
- `Salem_Workspace_Project_Prompt.md` - Project context

### Security & Credentials
**NEVER COMMIT THESE:**
- `personal_access_token.txt`
- `SALEM_api_key.txt`
- `client_secret_*.json` (Google OAuth)
- `drive-*.json` (Google Drive credentials)
- Any `.env` files

---

## Key Design Patterns

### 1. Origin-Tagged Provenance System
Every data record includes an `origin` field identifying its source application. This enables future conflict resolution when the same event appears from multiple perspectives.

**Example:**
- Chronicle says "I was alone at dinner" (`origin: 'Chronicle'`)
- Sentinel video shows two people (`origin: 'Sentinel'`)
- Merger flags this as a **conflict** rather than a duplicate

### 2. JSONB for Schema Flexibility
Extensive use of PostgreSQL JSONB columns for:
- `raw_data` - Original source data preservation
- `processed_data` - Extracted/enriched fields
- Platform-specific metadata
- Extensible custom fields

**Benefit:** No schema migrations needed for new source types.

### 3. Soft Deletes
Use `is_deleted` and `deleted_at` flags rather than hard deletes for:
- Audit trail preservation
- Data recovery capability
- Compliance requirements

### 4. Universal Timeline Events
The `timeline_events` table serves as the parent for all temporal data:
- Unified querying across event types
- Consistent temporal indexing
- Shared metadata (tags, attachments)
- Polymorphic relationships via UUIDs

---

## Project Structure Notes

### Application Consolidation Status
Per `REORGANIZATION_PLAN.md`, the workspace contains multiple versions of some apps:

**Confirmed Duplicates (to be archived):**
- `forensic-data-refinery (1)/` - 100% identical to `forensic-data-refinery/`
- `story-voice - 1/` - 100% identical to `story-voice/`
- Multiple config file duplicates in `_MARKED_FOR_DELETE/`

**Active Applications (Keep):**
- `TraceIQ_Timeline_App/` - PREFERRED (has 8 extra utilities)
- `TraceIQ_Complete/` - Less complete despite name (archive)
- `Chronicle_Voice_App/` - Voice/timeline analysis with chunking
- `ConflictAnalysisApp/` - Abuse pattern detection
- `Video_Analyzer_App/` - Video forensics (planned)

### Directory Organization Philosophy
The workspace is being reorganized into:
- `APPLICATIONS/` - Main apps
- `UTILITIES/` - Helper scripts and tools
- `DATA/` - Evidence files, timelines, communications
- `CONFIGURATION/` - Deployment configs, credentials
- `DOCUMENTATION/` - Guides and references
- `EXTERNAL_TOOLS/` - Third-party tools
- `_ARCHIVE/` - Old versions
- `_MARKED_FOR_DELETE/` - Confirmed duplicates

---

## Data Integration Contract

When working with this codebase:

1. **Never overwrite data from another app** - Check the `origin` field
2. **Always tag your data** - Set `origin` to the appropriate app name
3. **Preserve provenance** - The `origin` field is CRITICAL for conflict resolution
4. **Use UUIDs** - All IDs are UUIDs for distributed system compatibility
5. **Respect soft deletes** - Filter by `is_deleted = false` in queries
6. **Preserve raw data** - Always store original source data in `raw_data` JSONB

---

## Common Workflows

### Adding a New Evidence Source
1. Define source in `sources` table with unique identifier
2. Create events in `timeline_events` with proper `origin` tag
3. Store raw data in `raw_data` JSONB column
4. Extract structured fields to `processed_data`
5. Add attachments to `event_attachments` if applicable
6. Apply auto-tags via `event_tags` with confidence scores

### Running Timeline Analysis
1. Ensure database connection: `python test_db_connection.py`
2. Validate data integrity: `python validate_timeline_data.py`
3. Run overnight analysis: `python overnight_analyzer.py`
4. Generate schedule patterns: `python schedule_analyzer.py`

### Deploying Stack Updates
1. Update configuration in `docker-compose-salem-v3.1.yml`
2. Update environment variables in `.env`
3. Test configuration: `docker-compose config`
4. Deploy: `docker-compose up -d`
5. Verify services: `docker-compose ps`
6. Check logs: `docker-compose logs -f`

---

## Testing

**Python Tests:**
```bash
# Parser tests
python test_parser.py

# Database connectivity
python TraceIQ_Timeline_App/test_db_connection.py

# ChatGPT parser
python Utilities/SCRIPTS/test_chatgpt_parser.py

# JSON utilities
python Utilities/SCRIPTS/test_json_splitter.py
```

**Note:** No centralized test runner currently. Tests are per-component.

---

## Security Considerations

### Exposed Credentials Alert
Per `REORGANIZATION_PLAN.md`, the workspace contains plaintext credentials in multiple files. After reorganization:
1. Move all credentials to `CONFIGURATION/CREDENTIALS/` with restricted permissions
2. Ensure all credential files are in `.gitignore`
3. **Rotate all exposed credentials** after cleanup

### Current Credential Locations (DO NOT COMMIT):
- `MASTER_ENV_COMPILED.env` - Multiple API keys, DB passwords
- `.env` files (various versions)
- `personal_access_token.txt`
- `SALEM_api_key.txt`
- `client_secret_*.json`
- `drive-*.json`

---

## MCP Servers & AI Integration

**Available MCP Servers:**
- OpenMemory (`openmemory`) - Persistent conversational memory across sessions
- Zep - Personal life context (ONLY for personal relationships, NOT technical/legal)
- Mem0 - AI conversation memory (integrated with stack)

**Usage Pattern:**
1. Search OpenMemory at conversation start for relevant context
2. Store key learnings after completing significant tasks
3. Use for user preferences, project patterns, important decisions

---

## Important Notes

### When to Use Which App
- **Chronicle:** User-reported events, voice sessions, manual timeline entry
- **TraceIQ:** Location data analysis, Google Maps timeline processing
- **ConflictAnalysisApp:** Pattern detection in communication, abuse indicators
- **Chat Parsers:** Extracting structured data from message exports

### Database Best Practices
- Use Drizzle ORM for type-safe queries
- Leverage JSONB GIN indexes for metadata queries
- Apply importance_score (0.0-1.0) for filtering significant events
- Use tsquery/tsvector for full-text search on descriptions

### LiteLLM Routing
The LiteLLM gateway routes requests based on model names:
- `gpt-4` → OpenAI
- `claude-*` → Anthropic
- `openrouter/*` → OpenRouter
- Local models → Ollama instance

**Config:** `litellm_config.yaml`

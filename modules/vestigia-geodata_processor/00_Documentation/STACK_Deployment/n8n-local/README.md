# Evidence Processing Pipeline Setup

## Current Status: Ready for Deployment

### Stack Overview
```
┌─────────────────────────────────────────────────────────────┐
│                    LOCAL PROCESSING                          │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐   │
│  │  N8N        │────▶│ PostgreSQL  │────▶│ Python      │   │
│  │  Workflows  │     │ (Staging)   │     │ Processors  │   │
│  └─────────────┘     └─────────────┘     └─────────────┘   │
│         │                                       │           │
└─────────┼───────────────────────────────────────┼───────────┘
          │                                       │
          ▼                                       ▼
┌─────────────────────────────────────────────────────────────┐
│                    HOSTED SERVICES                           │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐   │
│  │  Supabase   │     │  Weaviate   │     │   Neo4j     │   │
│  │  (Postgres) │     │  (Vectors)  │     │  (Graph)    │   │
│  └─────────────┘     └─────────────┘     └─────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

---

## IMMEDIATE ACTION ITEMS

### 1. Start N8N + PostgreSQL
```powershell
cd "C:\Users\matts\AI Workspace\n8n-local"
docker-compose up -d
```

Access N8N at: http://localhost:5678
PostgreSQL at: localhost:5432 (user: n8n, pass: n8n_local_2024)

### 2. Deploy Supabase Schema
1. Go to: https://supabase.com/dashboard/project/oflqpddqaecotsdsxbzp/sql
2. Copy contents of `supabase_production_schema.sql`
3. Run it

### 3. Get LlamaCloud API Key
1. Go to: https://cloud.llamaindex.ai
2. Sign up (free tier: 10k credits)
3. Get API key
4. Add to .env:
   ```
   LLAMA_CLOUD_API_KEY=your_key_here
   ```

### 4. Provide SMS XML Path
Tell Claude where your SMS backup XML file is located so we can start processing.

---

## FILES CREATED

| File | Purpose |
|------|---------|
| `docker-compose.yml` | N8N + PostgreSQL local stack |
| `init-scripts/01_create_tables.sql` | Local staging tables |
| `supabase_production_schema.sql` | Production schema for Supabase |

---

## CREDENTIALS NEEDED

| Service | Status | Action |
|---------|--------|--------|
| Supabase | ✅ Have | oflqpddqaecotsdsxbzp project |
| LlamaCloud | ❓ Need | Sign up + get API key |
| Weaviate | ❓ Need URL | Provide cluster URL + key |
| Neo4j | ❓ Need URL | Provide connection details |
| Google Doc AI | ❓ Partial | Need to verify processor setup |

---

## PROCESSING PIPELINE

### Phase 1: SMS/MMS Processing
```
sms_backup.xml
    ↓ [sms_backup_parser.py - strips base64]
Chunked Messages
    ↓ [LlamaExtract - entities]
    ↓ [YAML Rules - behaviors]
Classified Data
    ↓ [Push to databases]
Supabase + Weaviate + Neo4j
```

### Phase 2: Conversation Logs (Claude, Facebook, etc.)
Same pipeline, different parsers.

### Phase 3: Documents (PDFs, images)
Google Document AI for OCR → Same classification pipeline.

---

## BEHAVIORAL DETECTION

Already have YAML configs at:
`C:\Users\matts\AI Workspace\Analysis_Notes\analysis_configs_bundle\`

Categories:
- Gaslighting
- Blame Shifting  
- Coercive Control
- Parental Alienation
- DARVO
- Love Bombing
- Stonewalling
- Financial Abuse
- Substance Weaponization
- And more...

---

## NEXT STEPS AFTER SETUP

1. **Test pipeline** with small sample (100 messages)
2. **Refine extraction schema** based on results
3. **Build N8N workflows** for automated processing
4. **Create search interface** for evidence queries
5. **Generate reports** by MCL factor

---

## Quick Commands

```powershell
# Start local stack
cd "C:\Users\matts\AI Workspace\n8n-local"
docker-compose up -d

# Check logs
docker-compose logs -f

# Stop stack
docker-compose down

# Reset (delete all data)
docker-compose down -v
```

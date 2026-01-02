# Story Capture Setup

## Architecture
```
Cloudflare Pages (frontend)  →  HuggingFace Space (backend)
       ↓                              ↓
  Google OAuth2               Gemini 3 Pro analysis
  Gemini 2.5 Flash              Supabase writes
  Web Speech API                Qdrant writes
```

## Step 1: Google OAuth2 (5 min)

1. Go to https://console.cloud.google.com/apis/credentials
2. Create OAuth 2.0 Client ID (Web application)
3. Add authorized redirect URIs:
   - `https://your-site.pages.dev/`
   - `http://localhost:8000/` (for testing)
4. Copy Client ID → replace `YOUR_CLIENT_ID` in `index.html`

## Step 2: HuggingFace Space (5 min)

1. Create new Space at https://huggingface.co/new-space
2. Choose "Gradio" SDK
3. Upload `backend/app.py` and `backend/requirements.txt`
4. Add secrets in Settings:
   - `SUPABASE_URL` - your Supabase project URL
   - `SUPABASE_KEY` - your Supabase anon key
   - `QDRANT_URL` - your Qdrant Cloud URL
   - `QDRANT_API_KEY` - your Qdrant API key
5. Copy Space URL → replace `YOUR_SPACE` in `index.html`

## Step 3: Supabase Tables

Run in Supabase SQL editor:

```sql
-- Sessions
CREATE TABLE story_sessions (
  id UUID PRIMARY KEY,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  raw_transcript JSONB,
  clean_transcript TEXT,
  extraction JSONB,
  summary TEXT
);

-- Entities
CREATE TABLE story_entities (
  id SERIAL PRIMARY KEY,
  session_id UUID REFERENCES story_sessions(id),
  entity_type TEXT,
  name TEXT,
  metadata JSONB,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Timeline
CREATE TABLE story_timeline (
  id SERIAL PRIMARY KEY,
  session_id UUID REFERENCES story_sessions(id),
  event_description TEXT,
  date_mentioned TEXT,
  date_estimated DATE,
  confidence INT,
  turn_index INT,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for vector search later
CREATE INDEX idx_entities_name ON story_entities(name);
CREATE INDEX idx_timeline_date ON story_timeline(date_estimated);
```

## Step 4: Qdrant Collection

Create via Qdrant Cloud dashboard or API:

```bash
curl -X PUT "https://YOUR_CLUSTER.qdrant.io/collections/story_sessions" \
  -H "api-key: YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "vectors": {
      "size": 768,
      "distance": "Cosine"
    }
  }'
```

## Step 5: Cloudflare Pages

1. Create new Pages project at https://dash.cloudflare.com/
2. Connect to Git or Direct Upload
3. Upload `frontend/index.html`
4. Deploy

## Usage

1. Open your Cloudflare Pages URL
2. Sign in with Google
3. Click "Start Session"
4. Talk - the AI will respond and track everything
5. Click "End Session" when done
6. Click "Analyze & Save" to run Gemini 3 Pro extraction
7. Export MD/JSON locally if needed

## Models Used

- **Live conversation**: Gemini 2.5 Flash (via OAuth2 - free)
- **Analysis/Extraction**: Gemini 3 Pro (via OAuth2 - free)
- **Embeddings**: text-embedding-004 (via OAuth2 - free)

## Costs

- Cloudflare Pages: Free
- HuggingFace Spaces: Free
- Google Gemini: Free via Workspace OAuth2
- Supabase: Free tier (500MB)
- Qdrant Cloud: Free tier (1GB)

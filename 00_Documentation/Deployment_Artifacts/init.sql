-- Core for AI memory + GIS
CREATE EXTENSION IF NOT EXISTS pgvector;
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS pg_uuidv7;
CREATE EXTENSION IF NOT EXISTS hstore;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS ltree;

-- SPECIFIED EXTENSIONS
CREATE EXTENSION IF NOT EXISTS pgai;
CREATE EXTENSION IF NOT EXISTS pg_jsonschema;
CREATE EXTENSION IF NOT EXISTS timescaledb;
CREATE EXTENSION IF NOT EXISTS pg_partman;
CREATE EXTENSION IF NOT EXISTS fuzzystrmatch;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Verify all loaded
\dx

-- 1. Partitioned agent memories (pg_partman + timescaledb)
CREATE TABLE IF NOT EXISTS agent_memories (
  id UUID PRIMARY KEY DEFAULT gen_uuidv7(),
  user_id UUID NOT NULL,
  session_id UUID,
  embedding VECTOR(1536),
  content TEXT,
  metadata JSONB,
  content_hash BYTEA DEFAULT digest(content, 'sha256'),
  created_at TIMESTAMPTZ DEFAULT NOW()
) PARTITION BY RANGE (created_at);

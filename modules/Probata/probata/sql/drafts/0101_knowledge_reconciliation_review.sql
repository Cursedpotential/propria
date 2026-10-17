-- Propria knowledge reconciler: review records within EXISTING canonical platform PostgreSQL.
-- Status: DRAFT migration; DO NOT apply until confirmed against live schema and reviewer ACLs.
-- Input: n8n reference-only review candidates. Output: durable versioned human review and decisions.
-- Effects: additive tables in ops; NO evidence, memory, original or other source updates.
-- The same Workbench Activity surface displays these records; this is NOT a replacement evidence review gate.
-- Byline: ChatGPT GPT-6, 2026-10-17.
BEGIN;
CREATE SCHEMA IF NOT EXISTS ops;

CREATE TABLE IF NOT EXISTS ops.knowledge_review_case (
  review_ref uuid PRIMARY KEY DEFAULT uuidv7(),
  fingerprint text NOT NULL UNIQUE CHECK (length(fingerprint)=64),
  scope text NOT NULL CHECK (length(scope) BETWEEN 1 AND 120),
  kind text NOT NULL CHECK (length(kind) BETWEEN 1 AND 80),
  severity text NOT NULL CHECK (severity IN ('low','medium','high','critical')),
  summary text NOT NULL CHECK (length(summary) BETWEEN 1 AND 240),
  source_refs jsonb NOT NULL CHECK (jsonb_typeof(source_refs)='array'),
  source_revisions jsonb NOT NULL CHECK (jsonb_typeof(source_revisions)='array'),
  status text NOT NULL DEFAULT 'queued' CHECK(status IN ('queued','deferred','decided')),
  case_version integer NOT NULL DEFAULT 1 CHECK(case_version>=1),
  final_disposition text CHECK(final_disposition IN ('approve_plan','reject','defer')),
  native_write_receipt jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  last_seen_at timestamptz NOT NULL DEFAULT now(),
  decided_at timestamptz,
  CHECK (final_disposition IS NULL OR status='decided')
);

CREATE TABLE IF NOT EXISTS ops.knowledge_review_action (
  action_id uuid PRIMARY KEY DEFAULT uuidv7(),
  review_ref uuid NOT NULL REFERENCES ops.knowledge_review_case(review_ref),
  case_version integer NOT NULL CHECK(case_version>=1),
  idempotency_key text NOT NULL UNIQUE CHECK(length(idempotency_key) BETWEEN 16 AND 256),
  reviewer_ref text NOT NULL CHECK(length(reviewer_ref) BETWEEN 1 AND 256),
  decision text NOT NULL CHECK(decision IN ('approve_plan','reject','defer')),
  rationale text NOT NULL CHECK(length(rationale) BETWEEN 1 AND 2000),
  decided_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS knowledge_review_active
  ON ops.knowledge_review_case (status, severity, created_at DESC)
  WHERE status IN ('queued','deferred');

-- The app's existing DB role gets bounded access, while authenticated API handlers
-- enforce separate internal writer and owner-decision capability checks.
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='platform_runtime') THEN
    GRANT USAGE ON SCHEMA ops TO platform_runtime;
    GRANT SELECT, INSERT, UPDATE ON ops.knowledge_review_case TO platform_runtime;
    GRANT SELECT, INSERT ON ops.knowledge_review_action TO platform_runtime;
  END IF;
END $$;
COMMIT;

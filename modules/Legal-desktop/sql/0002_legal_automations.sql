-- Byline: Grok · grok-4.6 · 2026-08-18
-- Schema file ONLY. Do not apply to live PG (Type 1 HOLD).
-- legal_core.automation_job is the later APScheduler Postgres jobstore.
-- Local runtime uses MemoryJobStore + data/workspace JSONL.

CREATE TABLE IF NOT EXISTS legal_core.playbook (
  playbook_id   text PRIMARY KEY,
  title         text NOT NULL,
  description   text NOT NULL DEFAULT '',
  kind          text NOT NULL DEFAULT 'structural_labels',
  court_safe    boolean NOT NULL DEFAULT false,
  created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS legal_core.playbook_step (
  playbook_id   text NOT NULL REFERENCES legal_core.playbook (playbook_id),
  step_index    integer NOT NULL,
  step_name     text NOT NULL,
  PRIMARY KEY (playbook_id, step_index)
);

CREATE TABLE IF NOT EXISTS legal_core.automation (
  automation_id uuid PRIMARY KEY,
  playbook_id   text REFERENCES legal_core.playbook (playbook_id),
  enabled       boolean NOT NULL DEFAULT true,
  trigger_kind  text NOT NULL,
  trigger_spec  jsonb NOT NULL DEFAULT '{}',
  last_run_at   timestamptz,
  created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS legal_audit.automation_run (
  run_id        uuid PRIMARY KEY,
  playbook_id   text NOT NULL,
  automation_id uuid,
  started_at    timestamptz NOT NULL DEFAULT now(),
  finished_at   timestamptz,
  ok            boolean NOT NULL,
  court_safe    boolean NOT NULL DEFAULT false,
  steps         jsonb NOT NULL DEFAULT '[]'
);

-- Later APScheduler SQLAlchemyJobStore. Not wired. Not applied.
CREATE TABLE IF NOT EXISTS legal_core.automation_job (
  id            varchar(191) PRIMARY KEY,
  next_run_time double precision,
  job_state     bytea NOT NULL
);

CREATE INDEX IF NOT EXISTS automation_job_next_run_time_idx
  ON legal_core.automation_job (next_run_time);

INSERT INTO legal_core.playbook (playbook_id, title, description)
VALUES (
  'foc-hearing-prep',
  'FOC hearing prep',
  'Structural labels only. No hearing date is invented. Clerk docket is not fetched.'
) ON CONFLICT (playbook_id) DO NOTHING;

INSERT INTO legal_core.playbook_step (playbook_id, step_index, step_name) VALUES
  ('foc-hearing-prep', 1, 'confirm-clerk-docket'),
  ('foc-hearing-prep', 2, 'review-factor-j'),
  ('foc-hearing-prep', 3, 'build-release')
ON CONFLICT (playbook_id, step_index) DO NOTHING;

GRANT SELECT, INSERT, UPDATE, DELETE ON legal_core.playbook TO legal_os_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON legal_core.playbook_step TO legal_os_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON legal_core.automation TO legal_os_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON legal_core.automation_job TO legal_os_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON legal_audit.automation_run TO legal_os_app;

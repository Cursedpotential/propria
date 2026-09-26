-- Byline: Grok · grok-4.6 · 2026-08-18
-- Isolated schemas on the existing Agno PostgreSQL 18 cluster.
-- Numbered raw SQL, same convention as Agno. Not Alembic.
-- Do not apply to production until an owner Type 1 review.

CREATE SCHEMA IF NOT EXISTS legal_core;
CREATE SCHEMA IF NOT EXISTS legal_research;
CREATE SCHEMA IF NOT EXISTS legal_work_product;
CREATE SCHEMA IF NOT EXISTS legal_release;
CREATE SCHEMA IF NOT EXISTS legal_audit;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'legal_os_app') THEN
    CREATE ROLE legal_os_app LOGIN;
  END IF;
END
$$;

GRANT USAGE ON SCHEMA legal_core, legal_research, legal_work_product, legal_release, legal_audit
  TO legal_os_app;

CREATE TABLE IF NOT EXISTS legal_core.app_settings (
  id              boolean PRIMARY KEY DEFAULT true CHECK (id),
  theme           text NOT NULL DEFAULT 'dark',
  display_timezone text NOT NULL DEFAULT 'America/New_York',
  confidential_mode boolean NOT NULL DEFAULT false,
  case_phase      text NOT NULL DEFAULT 'motions',
  updated_at      timestamptz NOT NULL DEFAULT now()
);

INSERT INTO legal_core.app_settings (id) VALUES (true) ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS legal_core.matter_ref (
  matter_id     uuid PRIMARY KEY,
  display_name  text NOT NULL,
  is_friendly_primary boolean NOT NULL DEFAULT true,
  source_revision text,
  synced_at     timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS legal_core.court_case_ref (
  court_case_id uuid PRIMARY KEY,
  matter_id     uuid NOT NULL REFERENCES legal_core.matter_ref (matter_id),
  display_name  text NOT NULL,
  docket_number text,
  court         text,
  is_primary    boolean NOT NULL DEFAULT true,
  synced_at     timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS legal_core.source_package (
  package_id      uuid PRIMARY KEY,
  schema_version  text NOT NULL,
  manifest_hash   text NOT NULL,
  matter_id       uuid NOT NULL,
  court_case_id   uuid,
  payload         jsonb NOT NULL,
  imported_at     timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS legal_core.source_package_omission (
  package_id  uuid NOT NULL REFERENCES legal_core.source_package (package_id),
  item_id     uuid NOT NULL,
  reason      text NOT NULL,
  PRIMARY KEY (package_id, item_id)
);

CREATE TABLE IF NOT EXISTS legal_work_product.work_product (
  work_product_id uuid PRIMARY KEY,
  matter_id       uuid NOT NULL,
  court_case_id   uuid,
  kind            text NOT NULL,
  title           text NOT NULL,
  created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS legal_work_product.work_product_version (
  work_product_id uuid NOT NULL REFERENCES legal_work_product.work_product (work_product_id),
  version         integer NOT NULL,
  state           text NOT NULL,
  source_package_id uuid,
  content_hash    text NOT NULL,
  body_md         text,
  cited_assertion_ids uuid[] NOT NULL DEFAULT '{}',
  cited_authority_ids text[] NOT NULL DEFAULT '{}',
  created_at      timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (work_product_id, version)
);

CREATE TABLE IF NOT EXISTS legal_audit.event_outbox (
  event_id           uuid PRIMARY KEY,
  event_type         text NOT NULL,
  schema_version     text NOT NULL,
  occurred_at        timestamptz NOT NULL,
  matter_id          uuid NOT NULL,
  court_case_id      uuid,
  aggregate_id       uuid NOT NULL,
  aggregate_version  integer NOT NULL,
  trace_id           text NOT NULL,
  payload_hash       text NOT NULL,
  payload            jsonb NOT NULL,
  published_at       timestamptz
);

CREATE TABLE IF NOT EXISTS legal_audit.consumed_event (
  event_id     uuid PRIMARY KEY,
  consumed_at  timestamptz NOT NULL DEFAULT now()
);

GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA legal_core TO legal_os_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA legal_work_product TO legal_os_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA legal_audit TO legal_os_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA legal_research TO legal_os_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA legal_release TO legal_os_app;

CREATE TABLE ops.legal_investigation_request (
 request_id uuid PRIMARY KEY,
 mode text NOT NULL CHECK (mode IN ('REAL','TEST')),
 matter_id uuid NOT NULL,
 court_case_id uuid NOT NULL,
 legal_matter_id uuid NOT NULL,
 claim_id uuid NOT NULL,
 followup_id uuid NOT NULL,
 actor_uid text NOT NULL CHECK (length(actor_uid) BETWEEN 1 AND 200),
 actor_username text NOT NULL CHECK (length(actor_username) BETWEEN 1 AND 200),
 idempotency_key uuid NOT NULL,
 idempotency_aliases jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(idempotency_aliases)='array'),
 payload_hash text NOT NULL CHECK (payload_hash ~ '^[0-9a-f]{64}$'),
 payload jsonb NOT NULL CHECK (jsonb_typeof(payload)='object' AND length(payload->>'question') BETWEEN 1 AND 5000 AND jsonb_typeof(payload->'sources')='array' AND jsonb_array_length(payload->'sources')<=30),
 status text NOT NULL DEFAULT 'received' CHECK (status IN ('received','running','completed','failed','cancelled')),
 results jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(results)='array' AND jsonb_array_length(results)<=100 AND octet_length(results::text)<=262144),
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 UNIQUE(actor_uid,idempotency_key),
 UNIQUE(mode,matter_id,court_case_id,legal_matter_id,claim_id,followup_id),
 CHECK (updated_at>=created_at),
 CHECK (payload->>'mode'=mode AND payload->>'matter_id'=matter_id::text AND payload->>'court_case_id'=court_case_id::text AND payload->>'legal_matter_id'=legal_matter_id::text AND payload->>'claim_id'=claim_id::text AND payload->>'followup_id'=followup_id::text)
);
COMMENT ON TABLE ops.legal_investigation_request IS 'Native investigation request receipts; received means awaiting execution, not evidence or an executed result. Legal IDs are opaque correlation. Registry entity references do not imply case participation.';
GRANT SELECT, INSERT, UPDATE ON TABLE ops.legal_investigation_request TO platform_app;

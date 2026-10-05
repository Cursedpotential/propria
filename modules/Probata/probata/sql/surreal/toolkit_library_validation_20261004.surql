-- Provision record-access principals and enforce separate library receipt and currency write boundaries.
-- Inputs: selected namespace/database; bound $validator_username, $validator_password_hash,
-- $currency_username, $currency_password_hash (precomputed Argon2 PHC hashes, no plaintext secrets).
-- Outputs: schema/principals after operator review; effects: only the listed permission definitions and service principals.
-- Choose for scoped record users, never DEFINE USER ... ROLES EDITOR: system users bypass table permissions.
-- Default is a cancelled transaction. Parent reviews then replaces CANCEL with COMMIT; no deployment is implied.
-- Existing source/reference/proposal schemas and fields are retained by ALTER TABLE.
-- Other tables must retain permissions NONE for these principals; audit any legacy PERMISSIONS FULL before cutover.
-- Existing privileged system users remain privileged and must never be described as table-scoped by this template.
-- No signup. Operator-only credential provisioning; neither principal may edit itself, a source, a proposal or delete records.
-- Byline: Codex · GPT-6.1 · 2026-10-04.

BEGIN TRANSACTION;
  -- Byline: Codex, 2026-10-04. Fresh live stores may not yet have the additive proposal table.
  DEFINE TABLE IF NOT EXISTS library_proposal SCHEMALESS;
  DEFINE TABLE IF NOT EXISTS toolkit_service SCHEMAFULL
    PERMISSIONS
      FOR select WHERE id = $auth
      FOR create, update, delete NONE;
  DEFINE FIELD IF NOT EXISTS name ON toolkit_service TYPE string;
  DEFINE FIELD IF NOT EXISTS password_hash ON toolkit_service TYPE string
    PERMISSIONS
      FOR select NONE;
  DEFINE FIELD IF NOT EXISTS role ON toolkit_service TYPE string
    ASSERT $value IN ['validator', 'currency'];
  DEFINE FIELD IF NOT EXISTS enabled ON toolkit_service TYPE bool;
  DEFINE INDEX IF NOT EXISTS toolkit_service_name ON toolkit_service FIELDS name
    UNIQUE;
  UPSERT toolkit_service:validator CONTENT {
    name: $validator_username,
    password_hash: $validator_password_hash,
    role: 'validator',
    enabled: true,
  };
  UPSERT toolkit_service:currency CONTENT {
    name: $currency_username,
    password_hash: $currency_password_hash,
    role: 'currency',
    enabled: true,
  };
  DEFINE ACCESS IF NOT EXISTS toolkit_validator ON DATABASE TYPE RECORD
    SIGNIN (
      SELECT *
      FROM toolkit_service
      WHERE
        id = toolkit_service:validator
        AND name = $username
        AND enabled = true
        AND crypto::argon2::compare(password_hash, $password)
    ) DURATION FOR TOKEN 15m, FOR SESSION 15m;
  DEFINE ACCESS IF NOT EXISTS toolkit_currency ON DATABASE TYPE RECORD
    SIGNIN (
      SELECT *
      FROM toolkit_service
      WHERE
        id = toolkit_service:currency
        AND name = $username
        AND enabled = true
        AND crypto::argon2::compare(password_hash, $password)
    ) DURATION FOR TOKEN 15m, FOR SESSION 15m;
  DEFINE TABLE IF NOT EXISTS library_validation SCHEMALESS;
  ALTER TABLE library_validation PERMISSIONS
    FOR select WHERE $auth IN [
      toolkit_service:validator,
      toolkit_service:currency,
    ]
    FOR create, update WHERE $auth = toolkit_service:validator
    FOR delete NONE;
  DEFINE TABLE IF NOT EXISTS library_currency SCHEMAFULL;
  ALTER TABLE library_currency PERMISSIONS
    FOR select WHERE $auth IN [
      toolkit_service:validator,
      toolkit_service:currency,
    ]
    FOR create, update WHERE $auth = toolkit_service:currency
    FOR delete NONE;
  DEFINE FIELD IF NOT EXISTS source_id ON library_currency TYPE string;
  DEFINE FIELD IF NOT EXISTS source_version ON library_currency TYPE string;
  DEFINE FIELD IF NOT EXISTS snapshot_sha256 ON library_currency TYPE string;
  DEFINE FIELD IF NOT EXISTS primary_url ON library_currency TYPE string;
  DEFINE FIELD IF NOT EXISTS release_url ON library_currency TYPE string;
  DEFINE FIELD IF NOT EXISTS release_version ON library_currency TYPE string;
  DEFINE FIELD IF NOT EXISTS release_sha256 ON library_currency TYPE string;
  DEFINE FIELD IF NOT EXISTS review_id ON library_currency TYPE string;
  DEFINE FIELD IF NOT EXISTS reviewer_id ON library_currency TYPE string;
  DEFINE FIELD IF NOT EXISTS decision_id ON library_currency TYPE string;
  DEFINE FIELD IF NOT EXISTS release_pinpoint ON library_currency TYPE string;
  DEFINE FIELD IF NOT EXISTS release_quote_sha256 ON library_currency TYPE string;
  DEFINE FIELD IF NOT EXISTS approval_ref ON library_currency TYPE object FLEXIBLE;
  DEFINE FIELD IF NOT EXISTS source_snapshot_ref ON library_currency TYPE object FLEXIBLE;
  DEFINE FIELD IF NOT EXISTS release_snapshot_ref ON library_currency TYPE object FLEXIBLE;
  DEFINE FIELD IF NOT EXISTS effective_at ON library_currency TYPE datetime;
  DEFINE FIELD IF NOT EXISTS reviewed_through ON library_currency TYPE datetime;
  DEFINE FIELD IF NOT EXISTS checked_at ON library_currency TYPE datetime;
  DEFINE FIELD IF NOT EXISTS expires_at ON library_currency TYPE datetime;
  DEFINE FIELD IF NOT EXISTS status ON library_currency TYPE string
    ASSERT $value = 'cleared';
  DEFINE FIELD IF NOT EXISTS signature ON library_currency TYPE string;
  ALTER TABLE source PERMISSIONS
    FOR select WHERE $auth IN [
      toolkit_service:validator,
      toolkit_service:currency,
    ]
    FOR create, update, delete NONE;
  ALTER TABLE reference PERMISSIONS
    FOR select WHERE $auth IN [
      toolkit_service:validator,
      toolkit_service:currency,
    ]
    FOR create, update, delete NONE;
  ALTER TABLE library_proposal PERMISSIONS
    FOR select WHERE $auth IN [
      toolkit_service:validator,
      toolkit_service:currency,
    ]
    FOR create, update, delete NONE;
CANCEL TRANSACTION;

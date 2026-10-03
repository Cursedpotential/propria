-- Byline: Claude Code · Sonnet 5.5 · 2026-10-02
--
-- Schema for conversations sent to surreal-case (SurrealDB 3.2.4, ovh-files :8471, tailnet only): the conv_* tables the
-- engine's send_to_surreal_workflow writes (modules/engine/surrealsink), and the login it uses.
--
-- The tables are NEW. Nothing here touches the Family Court Toolkit's tables (person, message, event, source, note, ...).
-- Every record id is deterministic (thread: a v5 uuid of matter + export + conversation; message: the PostgreSQL
-- normalized record id; entity/event/run: the PostgreSQL candidate/run id), so sending a conversation twice leaves one copy.
--
-- OWNER DECISION PENDING (see the task report): which database holds these tables. This file targets fct / case (option A).
-- For a separate database (option B) change the USE line, and set SURREAL_CASE_DATABASE on the proffer-worker.
--
-- Dry run (changes nothing; the transaction is cancelled):
--     run this file as is. NOTE: this script has NOT been executed anywhere (the task that wrote it could not write to Surreal);
--     the dry run is also its first syntax check. FULLTEXT is the 3.x spelling (the live docs store uses it).
-- Apply: replace the last line `CANCEL TRANSACTION;` with `COMMIT TRANSACTION;`, and bind $conv_password.
-- The password is bound as a parameter (never written in this file or in the chat); the engine reads it from
-- /data/probata/secrets/surreal-case/password on ovh-files (deploy/proffer-worker.yaml).

USE NS fct DB case;

BEGIN TRANSACTION;

-- One conversation: an export file plus a conversation key, read from its source versions.
DEFINE TABLE IF NOT EXISTS conv_thread SCHEMALESS PERMISSIONS NONE;
DEFINE FIELD IF NOT EXISTS matter_id ON conv_thread TYPE string;
DEFINE FIELD IF NOT EXISTS export_key ON conv_thread TYPE string;
DEFINE FIELD IF NOT EXISTS conv ON conv_thread TYPE string;
DEFINE FIELD IF NOT EXISTS case_ref ON conv_thread TYPE option<record<case_status>>;
DEFINE FIELD IF NOT EXISTS source_files ON conv_thread TYPE array;
DEFINE FIELD IF NOT EXISTS participants ON conv_thread TYPE array;
DEFINE FIELD IF NOT EXISTS message_count ON conv_thread TYPE int;
DEFINE FIELD IF NOT EXISTS first_at ON conv_thread TYPE option<datetime>;
DEFINE FIELD IF NOT EXISTS last_at ON conv_thread TYPE option<datetime>;
DEFINE FIELD IF NOT EXISTS sent_at ON conv_thread TYPE datetime;
DEFINE FIELD IF NOT EXISTS sent_by ON conv_thread TYPE string;
DEFINE FIELD IF NOT EXISTS send_request_id ON conv_thread TYPE string;
DEFINE INDEX IF NOT EXISTS conv_thread_matter ON conv_thread FIELDS matter_id;

-- One message of a conversation.
DEFINE TABLE IF NOT EXISTS conv_message SCHEMALESS PERMISSIONS NONE;
DEFINE FIELD IF NOT EXISTS thread ON conv_message TYPE record<conv_thread>;
DEFINE FIELD IF NOT EXISTS matter_id ON conv_message TYPE string;
DEFINE FIELD IF NOT EXISTS at ON conv_message TYPE option<datetime>;
DEFINE FIELD IF NOT EXISTS body ON conv_message TYPE string;
DEFINE FIELD IF NOT EXISTS sender ON conv_message TYPE option<string>;
DEFINE FIELD IF NOT EXISTS recipients ON conv_message TYPE array<string>;
DEFINE FIELD IF NOT EXISTS participants ON conv_message TYPE array;
DEFINE FIELD IF NOT EXISTS party ON conv_message TYPE option<string>;
DEFINE FIELD IF NOT EXISTS attachments ON conv_message TYPE int;
DEFINE FIELD IF NOT EXISTS certainty ON conv_message TYPE option<string>;
DEFINE FIELD IF NOT EXISTS source_version_id ON conv_message TYPE string;
DEFINE FIELD IF NOT EXISTS ordinal ON conv_message TYPE int;
DEFINE INDEX IF NOT EXISTS conv_message_thread_at ON conv_message FIELDS thread, at;
DEFINE INDEX IF NOT EXISTS conv_message_body_ft ON conv_message FIELDS body FULLTEXT ANALYZER en BM25;

-- One extraction run, tagged with the extractor that made it.
DEFINE TABLE IF NOT EXISTS conv_extraction_run SCHEMALESS PERMISSIONS NONE;
DEFINE FIELD IF NOT EXISTS thread ON conv_extraction_run TYPE record<conv_thread>;
DEFINE FIELD IF NOT EXISTS extractor ON conv_extraction_run TYPE string;
DEFINE FIELD IF NOT EXISTS extractor_version ON conv_extraction_run TYPE string;
DEFINE FIELD IF NOT EXISTS model_id ON conv_extraction_run TYPE option<string>;
DEFINE FIELD IF NOT EXISTS status ON conv_extraction_run TYPE string;
DEFINE FIELD IF NOT EXISTS compare_only ON conv_extraction_run TYPE bool;
DEFINE FIELD IF NOT EXISTS stats ON conv_extraction_run FLEXIBLE TYPE option<object>;
DEFINE FIELD IF NOT EXISTS started_at ON conv_extraction_run TYPE option<datetime>;
DEFINE FIELD IF NOT EXISTS finished_at ON conv_extraction_run TYPE option<datetime>;
DEFINE INDEX IF NOT EXISTS conv_extraction_run_thread ON conv_extraction_run FIELDS thread, extractor;

-- One extracted entity (person, organization, location, ...), tagged with its extractor.
DEFINE TABLE IF NOT EXISTS conv_entity SCHEMALESS PERMISSIONS NONE;
DEFINE FIELD IF NOT EXISTS thread ON conv_entity TYPE record<conv_thread>;
DEFINE FIELD IF NOT EXISTS run ON conv_entity TYPE record<conv_extraction_run>;
DEFINE FIELD IF NOT EXISTS extractor ON conv_entity TYPE option<string>;
DEFINE FIELD IF NOT EXISTS name ON conv_entity TYPE string;
DEFINE FIELD IF NOT EXISTS entity_type ON conv_entity TYPE string;
DEFINE FIELD IF NOT EXISTS aliases ON conv_entity TYPE array<string>;
DEFINE FIELD IF NOT EXISTS mention_count ON conv_entity TYPE int;
DEFINE FIELD IF NOT EXISTS mentions ON conv_entity TYPE array;
DEFINE FIELD IF NOT EXISTS confidence ON conv_entity TYPE float;
DEFINE FIELD IF NOT EXISTS review_state ON conv_entity TYPE string;
DEFINE INDEX IF NOT EXISTS conv_entity_thread ON conv_entity FIELDS thread, extractor;

-- One extracted event, tagged with its extractor.
DEFINE TABLE IF NOT EXISTS conv_event SCHEMALESS PERMISSIONS NONE;
DEFINE FIELD IF NOT EXISTS thread ON conv_event TYPE record<conv_thread>;
DEFINE FIELD IF NOT EXISTS run ON conv_event TYPE record<conv_extraction_run>;
DEFINE FIELD IF NOT EXISTS extractor ON conv_event TYPE option<string>;
DEFINE FIELD IF NOT EXISTS title ON conv_event TYPE string;
DEFINE FIELD IF NOT EXISTS event_type ON conv_event TYPE string;
DEFINE FIELD IF NOT EXISTS occurred_at ON conv_event TYPE option<datetime>;
DEFINE FIELD IF NOT EXISTS precision ON conv_event TYPE option<string>;
DEFINE FIELD IF NOT EXISTS confidence ON conv_event TYPE float;
DEFINE FIELD IF NOT EXISTS record_ids ON conv_event TYPE array<string>;
DEFINE FIELD IF NOT EXISTS review_state ON conv_event TYPE string;
DEFINE INDEX IF NOT EXISTS conv_event_thread ON conv_event FIELDS thread, extractor;
DEFINE INDEX IF NOT EXISTS conv_event_time ON conv_event FIELDS occurred_at;

-- The engine's login: a database-level EDITOR (SurrealDB has no finer role). A separate database (option B) is the
-- only way to keep this login away from the toolkit's tables; the code writes conv_* only.
DEFINE USER IF NOT EXISTS proffer_conversations ON DATABASE PASSWORD $conv_password ROLES EDITOR DURATION FOR TOKEN 1h, FOR SESSION 1h;

-- Checks (read them in the dry-run output).
INFO FOR DB;
INFO FOR TABLE conv_message;

CANCEL TRANSACTION;

-- Additive 0.8 migration. Never redefine existing vector indexes or history.
DEFINE TABLE IF NOT EXISTS docstore_meta SCHEMALESS;
DEFINE TABLE IF NOT EXISTS docstore_migration SCHEMALESS;
DEFINE TABLE IF NOT EXISTS docstore_enrichment SCHEMALESS;
DEFINE TABLE IF NOT EXISTS statement SCHEMALESS;
DEFINE TABLE IF NOT EXISTS entity_alias SCHEMALESS;
DEFINE FIELD IF NOT EXISTS version ON adr TYPE int DEFAULT 1;
DEFINE FIELD IF NOT EXISTS projection_path ON adr TYPE option<string>;
DEFINE FIELD IF NOT EXISTS projection_hash ON adr TYPE option<string>;
DEFINE FIELD IF NOT EXISTS projection_doc ON adr TYPE option<record<document>>;
DEFINE FIELD IF NOT EXISTS legacy_status ON adr TYPE option<string>;
DEFINE FIELD IF NOT EXISTS migration_key ON adr TYPE option<string>;
DEFINE INDEX IF NOT EXISTS adr_migration_key ON adr FIELDS migration_key UNIQUE;
DEFINE TABLE IF NOT EXISTS about TYPE RELATION IN document OUT entity SCHEMALESS;
DEFINE TABLE IF NOT EXISTS asserts TYPE RELATION IN document OUT statement SCHEMALESS;
DEFINE TABLE IF NOT EXISTS statement_mentions TYPE RELATION IN statement OUT entity SCHEMALESS;
DEFINE TABLE IF NOT EXISTS related_to TYPE RELATION IN entity|document|adr|statement OUT entity|document|adr|statement SCHEMALESS;
DEFINE TABLE IF NOT EXISTS supports TYPE RELATION IN document|adr|statement OUT document|adr|statement SCHEMALESS;
DEFINE TABLE IF NOT EXISTS contradicts TYPE RELATION IN document|adr|statement OUT document|adr|statement SCHEMALESS;
-- Content equality is not document identity: preserve copies and their provenance.
DEFINE INDEX OVERWRITE document_hash ON document FIELDS content_hash;
DEFINE TABLE IF NOT EXISTS links_to TYPE RELATION IN document OUT document SCHEMAFULL;
DEFINE FIELD IF NOT EXISTS kind ON links_to TYPE string DEFAULT "md_link";
DEFINE FIELD IF NOT EXISTS at ON links_to TYPE datetime VALUE time::now();
DEFINE TABLE IF NOT EXISTS cites TYPE RELATION IN document OUT document SCHEMAFULL;
DEFINE FIELD IF NOT EXISTS kind ON cites TYPE string ASSERT $value IN ["adr","decision"];
DEFINE FIELD IF NOT EXISTS ids ON cites TYPE array<string> DEFAULT [];
DEFINE FIELD IF NOT EXISTS at ON cites TYPE datetime VALUE time::now();

-- Optional contextual retrieval fields; preparation only, not applied automatically.
-- Inputs: existing chunk table. Outputs: three derived search columns and BM25
-- index. Side effects: schema/index additions. Apply only before explicit source-
-- scoped activation; the raw chunk_text_ft and raw text remain authoritative.
-- option<string> accepts omitted fields on legacy UPSERT CONTENT rewrites;
-- DEFAULT alone applies only on CREATE and cannot supply them on an update.
DEFINE FIELD IF NOT EXISTS context_description ON chunk TYPE option<string> DEFAULT "";
DEFINE FIELD IF NOT EXISTS search_text ON chunk TYPE option<string> DEFAULT "";
DEFINE FIELD IF NOT EXISTS context_provenance ON chunk TYPE option<string> DEFAULT "";
DEFINE INDEX IF NOT EXISTS chunk_context_ft ON chunk FIELDS search_text
  FULLTEXT ANALYZER doc_text BM25 CONCURRENTLY;

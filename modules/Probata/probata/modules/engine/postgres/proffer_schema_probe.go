// Byline: Codex · GPT-6.1-sol · 2026-10-08 (D-153 fresh snapshot admission)
// Byline: Codex · GPT-5 · 2026-10-05 (single-case operating contract)
// Byline: Codex · GPT-5.6-Sol · 2026-08-30 (shared Proffer schema admission)
// Retarget · Claude Code · Sonnet 5 · 2026-09-02 (BUILD LANE S2): ledger check
// moved from public.schema_version to ops.migration_ledger per D-109 (see
// comment at the ledgerCount subquery below).
// Dev-flag identity/receipt sentinel · Claude Code · Sonnet 5 · 2026-09-02
// (BUILD LANE S3, D-126): PLATFORM_DEV_AUTH_BYPASS (D-125) points the
// identity + receipt checks at a fixed, obviously-synthetic pre-launch
// sentinel instead of the real go-live identity. Both checks stay fully
// enforced in both modes -- see the doc comment on devMatterID below for the
// owner's exact scoping ruling and why this is not a skip.
// Oct 5 owner correction: D-126 identity selection is superseded. D-125 auth
// and D-110 evidence guards remain independent; this probe always checks the
// approved real identity and receipt. Legacy sentinel constants below are
// historical fixtures only and never admitted identities.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/jackc/pgx/v5"
)

// SchemaProbeDB is the read-only database surface needed for startup admission.
type SchemaProbeDB interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

var requiredProfferTables = []string{
	"registry.matter", "registry.court_case", "analysis.matter_knowledge_partition", "analysis.case_registry_import_receipt",
	"context.activity_execution", "context.activity_receipt", "context.hash_batch",
	"context.hash_batch_member", "context.hash_manifest", "context.hash_manifest_member",
	"context.hash_receipt", "context.normalization_lineage", "context.normalized_generation",
	"context.normalized_generation_publication", "context.normalized_record_identity",
	"context.raw_format_registry", "context.raw_generation", "context.raw_record_identity",
	"context.reconciliation_receipt", "context.retained_object", "context.source",
	"context.source_metadata", "context.source_version", "context.source_version_object",
	"context.proffer_preview_binding", "context.proffer_preview_snapshot", "context.proffer_preview_receipt",
	"context.proffer_preview_participant", "context.proffer_preview_message", "context.proffer_preview_attachment",
	"context.proffer_preview_event", "context.proffer_preview_decision", "context.repair_assessment",
	"context.repair_decision", "context.repair_resolution", "context.proffer_source_context_revision",
	"context.handler_content_signature", "context.handler_detected_format", "context.handler_compatibility",
	"context.handler_recommendation", "context.handler_selection_decision", "context.handler_selection_validation",
}

var requiredProfferColumns = []string{
	"context.source_version.matter_id", "context.source_version.court_case_id",
	"context.source_version.source_context_ref",
	"context.proffer_source_context_revision.matter_id",
	"context.proffer_source_context_revision.court_case_id",
	"context.proffer_source_context_revision.source_context_ref",
	"analysis.case_registry_import_receipt.source_migration_uri",
	"analysis.case_registry_import_receipt.source_migration_sha256",
	"analysis.case_registry_import_receipt.source_git_commit",
	"analysis.case_registry_import_receipt.payload_schema_version",
	"analysis.case_registry_import_receipt.payload_byte_length",
	"analysis.case_registry_import_receipt.canonical_payload_sha256",
	"analysis.case_registry_import_receipt.api_payload_sha256",
	"analysis.case_registry_import_receipt.source_observed_at",
	"analysis.case_registry_import_receipt.approved_by",
	"analysis.case_registry_import_receipt.approved_on",
}

// The real go-live case identity (OD-05), minted by sql/bootstrap/seed_live_case_registry_20261001.sql from the
// owner-approved payload sql/bootstrap/case_registry_live_identity_20261001.json (owner 2026-10-01 07:17 EDT,
// "Mint now, I approve"). Replaces the never-minted 0030 handoff values. Claude Code · Opus 5.5 · 2026-10-02.
const authoritativeMatterID = caseidentity.AuthoritativeMatterID
const authoritativeCourtCaseID = caseidentity.AuthoritativeCourtCaseID
const registrySourceMigrationURI = "sql/bootstrap/case_registry_live_identity_20261001.json"
const registrySourceMigrationSHA256 = "b39561e99a111c55f86d258e97cdbffec3fa0f84efb1d97fb548aa301b8fe544"
const registrySourceGitCommit = "5643178cf0beba05a11bd357544e0c32f170f840"
const registryPayloadSchemaVersion = "live-case-registry-identity-v1"
const registryCanonicalPayloadSHA256 = "e51c7fcfcc59d422255e173cbad9455ff010e8e052807190b7872b49cfaf113a"
const registryAPIPayloadSHA256 = "b39561e99a111c55f86d258e97cdbffec3fa0f84efb1d97fb548aa301b8fe544"

// The receipt predicates bound in STRICT mode (DEV mode binds the devReceipt* values below).
const registryReceiptPayloadByteLength = 4532
const registryReceiptApprovedBy = "owner"
const registryReceiptApprovedOn = "2026-10-01"

// platformDevAuthBypassEnv is the one flag D-125 defines for every ingest
// surface (Proffer starter, Workbench BFF, and -- as of D-126 -- this admission
// probe). Default OFF, fail-closed: unset or anything but a truthy value
// means STRICT (the real go-live identity is required, unmet until go-live).
const platformDevAuthBypassEnv = "PLATFORM_DEV_AUTH_BYPASS"

// D-126 (2026-09-02, owner refinement): "The only thing the feature flag
// should really do is bypass the UUID type requirement. And allow for the
// UUID to persist. And add a fake one instead of an auto created one, but
// everything else is still going to look for it, still going to reference
// it. But it's going to be referencing a fake one that's not an actual
// UUID." I.e. identity and receipt checking stay fully ON under the flag --
// only WHICH constants they must match changes. This is not "skip the
// check"; it is "check against the known-fake pre-launch value."
//
// registry.matter.id / registry.court_case.id are Postgres `uuid`-typed
// columns with live FK referrers across sql/0043, 0047, 0053 and 0054
// (context.source_version, context.proffer_source_context_revision,
// working.first_party_context_thread/third_party_context_thread,
// analysis.matter_knowledge_partition, analysis.case_registry_import_receipt
// all carry `matter_id UUID`/`court_case_id UUID` FKs) -- a non-UUID-shaped
// literal ("dev1" etc.) cannot be stored without a destabilizing type change
// across every one of them. So the sentinel is UUID-SHAPED but built
// entirely from classic "this is obviously fake" hex magic numbers (every
// digit is valid hex, 0-9/a-f): DEADBEEF for the matter, CAFEBABE for the
// court case. Neither uuidv7() nor any real UUID generator emits either
// pattern, and both read as fake at a glance next to a real time-ordered
// uuidv7 id, which always starts with a timestamp prefix (e.g. 01a0...).
// sql/0069_dev_case_registry_identity.sql seeds exactly these two values.
const devMatterID = "deadbeef-dead-beef-dead-beefdeadbeef"
const devCourtCaseID = "cafebabe-cafe-babe-cafe-babecafebabe"

// The dev receipt is written HONESTLY: D-126 forbids ever recording
// approved_by='owner' for an approval the owner did not give -- that would
// be exactly the fabricated-record class of defect
// docs/CLAIMED_COMPLETE_LIKELY_LIES/ exists to catch. approved_by here names
// the mechanism, not a person. Every hash/commit field is a fixed hex
// "magic number" placeholder (never derived from a real payload) so it is
// obviously not asserting real content-integrity -- sql/0069 seeds the
// identical literals; the two files must be changed together.
const devReceiptSourceMigrationURI = "sql/0069_dev_case_registry_identity.sql"
const devReceiptSourceMigrationSHA256 = "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
const devReceiptSourceGitCommit = "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
const devReceiptPayloadSchemaVersion = "dev-placeholder-v1"
const devReceiptPayloadByteLength = 1
const devReceiptCanonicalPayloadSHA256 = "cafebabecafebabecafebabecafebabecafebabecafebabecafebabecafebabe"
const devReceiptAPIPayloadSHA256 = "deadfacedeadfacedeadfacedeadfacedeadfacedeadfacedeadfacedeadface"
const devReceiptApprovedBy = "dev-mode-placeholder"
const devReceiptApprovedOn = "2026-09-02"

// devAuthBypassEnabled reads PLATFORM_DEV_AUTH_BYPASS directly rather than
// taking a parameter: ProbeProfferSchema is called from modules/engine/temporal/
// cmd/starter/main.go and modules/engine/profferworker/worker.go with a fixed
// two-argument signature, and D-125's contract is one process-wide flag, not
// a value threaded through every caller. Truthy values match D-125's own
// documented example (PLATFORM_DEV_AUTH_BYPASS=1) plus the usual spellings;
// anything else, including unset, is OFF (fail-closed default).
func devAuthBypassEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(platformDevAuthBypassEnv))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// ProbeProfferSchema admits the canonical snapshot schema and exact real-case receipt.
// Inputs: context and read-only catalog client. Output: admission error or nil.
// Effects: one catalog query; choose before polling any Proffer Temporal queue.
// D-153 and sql/bootstrap/README.md make the snapshot the fresh-bootstrap authority:
// retained migration history is protected state, not required proof of retired DDL.
func ProbeProfferSchema(ctx context.Context, db SchemaProbeDB) error {
	if db == nil {
		return errors.New("Proffer schema admission: database is required")
	}
	devBypass := devAuthBypassEnabled()
	matterID, courtCaseID := authoritativeMatterID, authoritativeCourtCaseID
	receiptURI, receiptSHA256Hex := registrySourceMigrationURI, registrySourceMigrationSHA256
	gitCommit, schemaVersion := registrySourceGitCommit, registryPayloadSchemaVersion
	canonicalSHA256Hex, apiSHA256Hex := registryCanonicalPayloadSHA256, registryAPIPayloadSHA256
	payloadByteLength, approvedBy, approvedOn := registryReceiptPayloadByteLength, registryReceiptApprovedBy, registryReceiptApprovedOn
	// The identity check no longer follows the dev flag (owner 2026-10-02: the flag only governs login).
	// The real go-live receipt is required in every mode.
	var database, currentUser, databaseOwner string
	var tableCount, columnCount int
	var constraintsExact, substrateExact, roleSafe, grantsExact, receiptExact bool
	err := db.QueryRow(ctx, `
		SELECT current_database(), current_user,
		       pg_get_userbyid((SELECT datdba FROM pg_database WHERE datname=current_database())),
		       (SELECT count(*) FROM information_schema.tables
		         WHERE format('%s.%s',table_schema,table_name)=ANY($1::text[])),
		       (SELECT count(*) FROM information_schema.columns
		         WHERE format('%s.%s.%s',table_schema,table_name,column_name)=ANY($2::text[])),
		       (NOT EXISTS (
		         SELECT 1 FROM (VALUES
		           ('context.source_version','source_version_matter_fk','registry.matter',ARRAY['matter_id'],ARRAY['id']),
		           ('context.source_version','source_version_court_case_scope_fk','registry.court_case',ARRAY['court_case_id','matter_id'],ARRAY['id','matter_id']),
		           ('context.source_version','source_version_source_context_scope_fk','context.proffer_source_context_revision',ARRAY['source_context_ref','matter_id','court_case_id'],ARRAY['source_context_ref','matter_id','court_case_id']),
		           ('context.proffer_source_context_revision','proffer_source_context_matter_fk','registry.matter',ARRAY['matter_id'],ARRAY['id']),
		           ('context.proffer_source_context_revision','proffer_source_context_court_case_scope_fk','registry.court_case',ARRAY['court_case_id','matter_id'],ARRAY['id','matter_id'])
		         ) AS required(relation_name,constraint_name,referenced_name,columns,referenced_columns)
		         WHERE NOT EXISTS (
		           SELECT 1 FROM pg_constraint c
		           WHERE c.conrelid=required.relation_name::regclass
		             AND c.confrelid=required.referenced_name::regclass AND c.contype='f'
		             AND c.conname=required.constraint_name AND c.convalidated AND c.confdeltype='r'
		             AND ARRAY(SELECT a.attname::text FROM unnest(c.conkey) WITH ORDINALITY k(attnum,ord)
		                       JOIN pg_attribute a ON a.attrelid=c.conrelid AND a.attnum=k.attnum ORDER BY k.ord)=required.columns
		             AND ARRAY(SELECT a.attname::text FROM unnest(c.confkey) WITH ORDINALITY k(attnum,ord)
		                       JOIN pg_attribute a ON a.attrelid=c.confrelid AND a.attnum=k.attnum ORDER BY k.ord)=required.referenced_columns))
		         AND EXISTS (SELECT 1 FROM pg_constraint c
		           WHERE c.conrelid='context.source_version'::regclass
		             AND c.conname='source_version_matter_case_pair_check' AND c.contype='c' AND c.convalidated)
		         AND EXISTS (SELECT 1 FROM pg_constraint c
		           WHERE c.conrelid='context.source_version'::regclass
		             AND c.conname='source_version_source_context_scope_check' AND c.contype='c' AND c.convalidated)
		         AND EXISTS (SELECT 1 FROM pg_constraint c
		           WHERE c.conrelid='context.proffer_source_context_revision'::regclass
		             AND c.conname='proffer_source_context_scope_key' AND c.contype='u' AND c.convalidated
		             AND ARRAY(SELECT a.attname::text FROM unnest(c.conkey) WITH ORDINALITY k(attnum,ord)
		                       JOIN pg_attribute a ON a.attrelid=c.conrelid AND a.attnum=k.attnum ORDER BY k.ord)
		                 =ARRAY['source_context_ref','matter_id','court_case_id'])),
		       (SELECT count(*)=4 FROM pg_constraint WHERE convalidated AND conname=ANY(ARRAY[
		         'raw_record_context_fingerprint_canon_check','hash_batch_context_kind_check',
		         'hash_manifest_context_kind_check','hash_receipt_context_kind_check'])),
		       COALESCE((SELECT rolcanlogin AND NOT (rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls)
		         AND pg_has_role('platform_runtime','context_import_writer','MEMBER')
		         FROM pg_roles WHERE rolname='platform_runtime'),false),
		       has_schema_privilege('platform_runtime','analysis','USAGE')
		         AND NOT has_schema_privilege('platform_runtime','analysis','CREATE')
		         AND has_table_privilege('platform_runtime','registry.matter','SELECT')
		         AND has_table_privilege('platform_runtime','registry.court_case','SELECT')
		         AND has_table_privilege('platform_runtime','analysis.matter_knowledge_partition','SELECT')
		         AND has_table_privilege('platform_runtime','analysis.case_registry_import_receipt','SELECT')
		         AND NOT has_table_privilege('platform_runtime','registry.matter','INSERT')
		         AND NOT has_table_privilege('platform_runtime','registry.matter','UPDATE')
		         AND NOT has_table_privilege('platform_runtime','registry.matter','DELETE')
		         -- D-153 retains historical state but does not require fabricated
		         -- migration rows on fresh bootstrap; runtime still cannot forge it.
		         AND NOT has_table_privilege('platform_runtime','ops.migration_ledger','INSERT')
		         AND (NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='agno_app') OR NOT (
		           has_table_privilege('agno_app','registry.matter','INSERT')
		           OR has_table_privilege('agno_app','registry.matter','UPDATE')
		           OR has_table_privilege('agno_app','registry.matter','DELETE')
		           OR has_table_privilege('agno_app','registry.court_case','INSERT')
		           OR has_table_privilege('agno_app','registry.court_case','UPDATE')
		           OR has_table_privilege('agno_app','registry.court_case','DELETE')
		           OR has_table_privilege('agno_app','analysis.matter_knowledge_partition','INSERT')
		           OR has_table_privilege('agno_app','analysis.matter_knowledge_partition','UPDATE')
		           OR has_table_privilege('agno_app','analysis.matter_knowledge_partition','DELETE'))),
		       -- The exact owner-approved real receipt is required in every mode.
		       (SELECT count(*)=1 AND count(*) FILTER (WHERE matter_id=$3::uuid AND court_case_id=$4::uuid
		          AND source_migration_uri=$5 AND encode(source_migration_sha256,'hex')=$6
		          AND source_git_commit=$7 AND payload_schema_version=$8 AND payload_byte_length=$11
		          AND encode(canonical_payload_sha256,'hex')=$9 AND encode(api_payload_sha256,'hex')=$10
		          AND approved_by=$12 AND approved_on=$13::date)=1
		          FROM analysis.case_registry_import_receipt)`,
		requiredProfferTables, requiredProfferColumns, matterID, courtCaseID,
		receiptURI, receiptSHA256Hex, gitCommit,
		schemaVersion, canonicalSHA256Hex, apiSHA256Hex,
		payloadByteLength, approvedBy, approvedOn,
	).Scan(&database, &currentUser, &databaseOwner, &tableCount, &columnCount,
		&constraintsExact, &substrateExact, &roleSafe, &grantsExact, &receiptExact)
	if err != nil {
		return errors.New("Proffer schema admission: catalog verification unavailable")
	}
	if database != "platform" || currentUser != "platform_runtime" || databaseOwner != "platform_admin" {
		return fmt.Errorf("Proffer schema admission: identity rejected: database=%q role=%q owner=%q", database, currentUser, databaseOwner)
	}
	if tableCount != len(requiredProfferTables) || columnCount != len(requiredProfferColumns) || !constraintsExact || !substrateExact || !roleSafe || !grantsExact || !receiptExact {
		return fmt.Errorf("Proffer schema admission failed (dev_bypass=%t): tables=%d/%d columns=%d/%d constraints=%t substrate=%t role=%t grants=%t receipt=%t",
			devBypass, tableCount, len(requiredProfferTables), columnCount,
			len(requiredProfferColumns), constraintsExact, substrateExact, roleSafe, grantsExact, receiptExact)
	}
	return nil
}

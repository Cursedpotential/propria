package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go.temporal.io/sdk/activity"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/humancontext"
	"github.com/Cursedpotential/probata/engine/xmlsourceframes"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// HumanSourceMetadata stores native bundle references and real stage receipts in existing context metadata.
// Inputs: platform metadata DB. Outputs: idempotent stage references; effects: metadata-only transactions.
// Pick for unretained context sources, never legacy normalized message storage.
type HumanSourceMetadata struct{ DB DB }

type humanStageMetadata struct {
	Contract         string              `json:"contract"`
	WorkflowID       string              `json:"workflow_id"`
	Source           humancontext.Source `json:"request_source"`
	InputManifestRef string              `json:"input_manifest_ref"`
	Result           humancontext.Result `json:"result"`
	Temporal         *humanStageTemporal `json:"temporal,omitempty"`
}

// humanStageTemporal records SDK-native execution clocks separately from receipt persistence.
// Inputs: actual Activity info. Outputs: metadata only; unavailable clocks are never invented.
type humanStageTemporal struct {
	WorkflowID           string    `json:"workflow_id"`
	WorkflowRunID        string    `json:"workflow_run_id"`
	ActivityID           string    `json:"activity_id"`
	Attempt              int32     `json:"attempt"`
	StartedAt            time.Time `json:"started_at"`
	OperationCompletedAt time.Time `json:"operation_completed_at"`
	ReceiptPersistedAt   time.Time `json:"receipt_persisted_at"`
}

// nativeStageTiming preserves retry attempt and start clock from the current Temporal Activity.
// Inputs: SDK Activity info and observed completion. Output: actual coordinates or explicit unavailable error.
func nativeStageTiming(info activity.Info, completed time.Time) (humanStageTemporal, error) {
	if info.StartedTime.IsZero() || info.Attempt < 1 || info.WorkflowExecution.ID == "" {
		return humanStageTemporal{}, errors.New("native Activity timing unavailable; receipt started_at cannot be invented")
	}
	return humanStageTemporal{WorkflowID: info.WorkflowExecution.ID, WorkflowRunID: info.WorkflowExecution.RunID, ActivityID: info.ActivityID, Attempt: info.Attempt, StartedAt: info.StartedTime, OperationCompletedAt: completed}, nil
}

// decodeHumanStage verifies a recovered receipt against its complete registered request coordinates.
// Inputs: metadata and requested stage/source. Outputs: recovered result or mismatch; effects: none.
func decodeHumanStage(raw []byte, req humancontext.Request, stage string) (humancontext.Result, error) {
	var m humanStageMetadata
	if err := json.Unmarshal(raw, &m); err != nil {
		return m.Result, err
	}
	r := m.Result
	if m.Contract != humancontext.Contract || m.WorkflowID != req.WorkflowID || !reflect.DeepEqual(m.Source, req.Source) || m.InputManifestRef != req.ManifestRef || r.ContractVersion != humancontext.Contract || r.Stage != stage || r.ManifestRef == "" || r.ReceiptRef == "" {
		return r, errors.New("human recovered stage request differs")
	}
	return r, nil
}

// LookupStage recovers the first completed native-human stage for this registered workflow.
// Inputs: actual workflow/source version and stage. Outputs: durable result or absent.
// Effects: read-only SQL; choose before a retry to avoid re-reading originals or duplicating bundles.
func (s *HumanSourceMetadata) LookupStage(ctx context.Context, req humancontext.Request, stage string) (humancontext.Result, bool, error) {
	var raw []byte
	err := s.DB.QueryRow(ctx, `SELECT metadata FROM context.source_metadata
WHERE source_version_id=$1::uuid AND extractor_id=$2 AND metadata->>'workflow_id'=$3
ORDER BY generated_at LIMIT 1`, req.Source.SourceVersionRef, "context-human-"+stage, req.WorkflowID).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return humancontext.Result{}, false, nil
	}
	if err != nil {
		return humancontext.Result{}, false, err
	}
	result, err := decodeHumanStage(raw, req, stage)
	return result, err == nil, err
}

// RecordStage atomically persists one stage receipt and its external native bundle locator.
// Inputs: registered source request and completed stage result. Outputs: recovered or newly persisted result.
// Effects: context activity/receipt/source_metadata writes only; native bodies remain in bundle files.
func (s *HumanSourceMetadata) RecordStage(ctx context.Context, req humancontext.Request, result humancontext.Result) (humancontext.Result, error) {
	completed := time.Now().UTC()
	if !activity.IsActivity(ctx) {
		return result, errors.New("native Activity metadata unavailable for required receipt start clock")
	}
	timing, err := nativeStageTiming(activity.GetInfo(ctx), completed)
	if err != nil {
		return result, err
	}
	if timing.WorkflowID != req.WorkflowID {
		return result, errors.New("native Activity workflow differs")
	}
	version, err := uuid.Parse(req.Source.SourceVersionRef)
	if err != nil {
		return result, err
	}
	if req.ContractVersion != humancontext.Contract || result.ContractVersion != humancontext.Contract || result.ManifestRef == "" || result.Stage == "" {
		return result, errors.New("invalid human stage receipt")
	}
	tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return result, err
	}
	defer func() { cleanup, cancel := lifecycleCleanup(ctx); defer cancel(); _ = tx.Rollback(cleanup) }()
	var workflow, source string
	if err := tx.QueryRow(ctx, `SELECT v.workflow_id,s.source_key FROM context.source_version v JOIN context.source s ON s.id=v.source_id WHERE v.id=$1::uuid FOR UPDATE OF v`, version).Scan(&workflow, &source); err != nil {
		return result, err
	}
	if workflow != req.WorkflowID || source != req.Source.SourceRef {
		return result, errors.New("human stage registration differs")
	}
	var prior []byte
	err = tx.QueryRow(ctx, `SELECT metadata FROM context.source_metadata WHERE source_version_id=$1::uuid AND extractor_id=$2 AND metadata->>'workflow_id'=$3 ORDER BY generated_at LIMIT 1`, version, "context-human-"+result.Stage, workflow).Scan(&prior)
	if err == nil {
		existing, decodeErr := decodeHumanStage(prior, req, result.Stage)
		if decodeErr != nil {
			return result, decodeErr
		}
		if err := tx.Commit(ctx); err != nil {
			return result, err
		}
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return result, err
	}
	execution, err := lifecycleEnsureExecution(ctx, tx, version, workflow, "context_human_"+result.Stage+"_activity", workflow, nil)
	if err != nil {
		return result, err
	}
	receipt, err := uuid.NewV7()
	if err != nil {
		return result, err
	}
	result.ReceiptRef = receipt.String()
	ref, err := resultReference("human_context_bundle", result.ManifestRef)
	if err != nil {
		return result, err
	}
	now := time.Now().UTC()
	if _, err := tx.Exec(ctx, `INSERT INTO context.activity_receipt (id,activity_execution_id,attempt,status,started_at,completed_at,result_ref) VALUES ($1::uuid,$2::uuid,$3,'success',$4,$5,$6::jsonb)`, receipt, execution, timing.Attempt, timing.StartedAt, completed, ref); err != nil {
		return result, err
	}
	timing.ReceiptPersistedAt = now
	metadata, err := json.Marshal(humanStageMetadata{Contract: humancontext.Contract, WorkflowID: workflow, Source: req.Source, InputManifestRef: req.ManifestRef, Result: result, Temporal: &timing})
	if err != nil {
		return result, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO context.source_metadata (source_version_id,metadata_class,metadata,extractor_id,extractor_version,extraction_activity_receipt_id,generated_at) VALUES ($1::uuid,'record_native',$2::jsonb,$3,'1.0.0',$4::uuid,$5)`, version, metadata, "context-human-"+result.Stage, receipt, now); err != nil {
		return result, err
	}
	if err := tx.Commit(ctx); err != nil {
		return result, err
	}
	return result, nil
}

// HumanSourceDuckDB executes bounded lexical JSON through pg_duckdb without storing raw message rows.
// Inputs: existing pg_duckdb pool. Outputs: native record JSON; effects: leased read-only DuckDB queries.
// Pick instead of StructuredELTRepository, which requires legacy retained/handler contracts.
type HumanSourceDuckDB struct{ Pool *pgxpool.Pool }

var humanSQLParameter = regexp.MustCompile(`\$[a-z_]+`)

// Extract maps one bounded generic lexical unit using the isolated human-source DuckDB SQL template.
// Inputs: actual source pins and complete lexical record plus root context. Output: one native record.
// Effects: no persistent SQL writes; external entity errors remain pending rather than becoming invented text.
func (s *HumanSourceDuckDB) Extract(ctx context.Context, source humancontext.Source, rows []xmlsourceframes.Lexeme) (humancontext.Record, error) {
	raw, err := json.Marshal(rows)
	if err != nil {
		return humancontext.Record{}, err
	}
	if len(raw) > humancontext.MaxMetadataBytes {
		return humancontext.Record{}, errors.New("human lexical query exceeds metadata bound")
	}
	args := map[string]string{"$lexemes": string(raw), "$source_ref": source.SourceRef, "$provider_version_id": source.ProviderVersionID, "$source_version_ref": source.SourceVersionRef}
	query := humanSQLParameter.ReplaceAllStringFunc(contextHumanSourceSQL, func(key string) string { return "'" + strings.ReplaceAll(args[key], "'", "''") + "'" })
	lease, err := s.Pool.Acquire(ctx)
	if err != nil {
		return humancontext.Record{}, err
	}
	defer lease.Release()
	if err := ensureWebbedLoaded(ctx, lease); err != nil {
		return humancontext.Record{}, fmt.Errorf("verify human XML decoder: %w", err)
	}
	var ordinal int64
	var status, native string
	// Dollar delimiter is fixed; lexical payload lives inside an escaped SQL literal.
	// A PostgreSQL bind carries the entire inner query, so no dollar-delimiter injection is possible.
	if err := lease.QueryRow(ctx, "SELECT (row['record_ordinal'])::bigint,(row['extraction_status'])::text,(row['native_fields'])::text FROM duckdb.query($1) AS row", query).Scan(&ordinal, &status, &native); err != nil {
		return humancontext.Record{}, err
	}
	return humancontext.Record{Ordinal: ordinal, Status: status, Native: json.RawMessage(native)}, nil
}

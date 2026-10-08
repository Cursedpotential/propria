// Byline: Codex · GPT-6.1-sol · 2026-10-08.
package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
)

// RegisterAIContextPackage uses the existing restricted writer and independent native readback functions.
// Inputs: complete canonical package metadata and exact byte SHA. Outputs: independently reconstructed payload.
// Effects: one serializable registration transaction followed by read-only verification; no broad table writes or DDL.
// Choose for native packages, leaving F:/Downloads and historical recovery identities in their existing siblings.
func (s *ToolkitRecoveryCatalog) RegisterAIContextPackage(ctx context.Context, raw []byte, sha string) ([]byte, error) {
	if s == nil || s.db == nil || len(raw) == 0 || len(raw) > 256<<10 || len(sha) != 64 {
		return nil, errors.New("native catalog metadata/writer unavailable or outside bounds")
	}
	actual := sha256.Sum256(raw)
	if hex.EncodeToString(actual[:]) != sha {
		return nil, errors.New("native catalog exact JSON byte digest differs")
	}
	var header struct {
		Contract  string            `json:"contract"`
		Operation string            `json:"operation"`
		Members   []json.RawMessage `json:"members"`
	}
	if json.Unmarshal(raw, &header) != nil || header.Contract != "ai-context-catalog/v1" || len(header.Operation) != 72 || len(header.Members) < 4 || len(header.Members) > 65 {
		return nil, errors.New("native catalog contract or member count invalid")
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return nil, errors.New("native catalog registration unavailable")
	}
	defer tx.Rollback(ctx)
	var count int
	if err = tx.QueryRow(ctx, `SELECT source_occurrence_api.register_ai_context_package($1::text,$2::text)`, string(raw), sha).Scan(&count); err != nil {
		return nil, errors.New("narrow native catalog registration failed")
	}
	if count != len(header.Members) {
		return nil, errors.New("native catalog registration member count differs")
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, errors.New("native catalog commit outcome unknown; retry identical pinned payload")
	}
	read, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, errors.New("native catalog independent readback unavailable")
	}
	defer read.Rollback(ctx)
	var observed []byte
	if err = read.QueryRow(ctx, `SELECT source_occurrence_api.read_ai_context_package($1::text,$2::text)`, header.Operation, sha).Scan(&observed); err != nil || len(observed) == 0 || len(observed) > 256<<10 {
		return nil, errors.New("native catalog stored rows/readback unavailable or outside bounds")
	}
	if err = read.Commit(ctx); err != nil {
		return nil, errors.New("native catalog readback transaction failed")
	}
	return observed, nil
}

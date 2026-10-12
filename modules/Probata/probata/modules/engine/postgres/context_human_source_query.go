package postgres

import _ "embed"

// contextHumanSourceSQL maps bounded original lexical values to native human-message fields.
// Inputs are four bound DuckDB parameters, not interpolated source text. Outputs are streaming bundle rows.
// Effects: none until executed by the context-native repository; this never changes the legacy ELT query.
//
//go:embed context_human_source.sql
var contextHumanSourceSQL string

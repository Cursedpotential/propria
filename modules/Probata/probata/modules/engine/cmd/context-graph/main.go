// Command context-graph projects and reads retained real extraction bundles in fct/analysis.
// Inputs are an explicit private bundle file and mounted ANALYSIS_SURREAL credentials.
// Outputs are schema text or reference-only checkpoints; effects are limited to the
// selected schema/project/read/deactivate operation. Pick downstream of extraction;
// the command never ingests, calls a model, applies schema or prints source bodies.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/Cursedpotential/probata/engine/surrealsink"
)

// run executes exactly one bounded operator-selected projection operation.
// Inputs are CLI arguments and stdout; output is an error or checkpoint/schema text.
// Effects read a private bundle and optionally project/deactivate its scoped generation.
// Pick as the CLI adapter for the independently callable repository units.
func run(args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("operation required: schema, project, read, deactivate or metadata")
	}
	operation := args[0]
	if operation != "schema" && operation != "project" && operation != "read" && operation != "deactivate" && operation != "metadata" {
		return errors.New("unsupported context graph operation")
	}
	f := flag.NewFlagSet("context-graph", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	path := f.String("bundle", "", "exact private retained bundle path")
	node := f.String("node", "", "existing node ID for read traversal")
	hops := f.Int("hops", 1, "read traversal hops: 1 or 2")
	if f.Parse(args[1:]) != nil || f.NArg() != 0 || (*path == "" && operation != "metadata") {
		return errors.New("explicit --bundle path required")
	}
	if operation == "metadata" {
		cfg, e := surrealsink.AnalysisConfigFromEnv()
		if e != nil {
			return e
		}
		return json.NewEncoder(out).Encode(map[string]string{"namespace": cfg.Namespace, "database": cfg.Database, "user": cfg.User, "auth_level": cfg.AuthLevel})
	}
	file, e := os.Open(*path)
	if e != nil {
		return errors.New("private bundle unavailable")
	}
	bundle, e := surrealsink.DecodeContextGraphBundle(file)
	_ = file.Close()
	if e != nil {
		return e
	}
	if operation == "schema" {
		schema, e := surrealsink.MinimalContextGraphSchema(bundle)
		if e != nil {
			return e
		}
		_, e = io.WriteString(out, schema)
		return e
	}
	cfg, e := surrealsink.AnalysisConfigFromEnv()
	if e != nil {
		return e
	}
	client, e := surrealsink.NewAnalysis(cfg)
	if e != nil {
		return e
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	var receipt surrealsink.ContextGraphReceipt
	switch operation {
	case "project":
		receipt, e = client.ProjectContextGraph(ctx, bundle)
	case "deactivate":
		receipt, e = client.DeactivateContextGraph(ctx, bundle.Scope, bundle.GenerationID)
	case "read":
		var view surrealsink.ContextGraphReadback
		view, e = client.TraverseContextGraph(ctx, bundle.Scope, bundle.GenerationID, *node, *hops)
		receipt = view.Receipt
	}
	if e != nil {
		return e
	}
	return json.NewEncoder(out).Encode(receipt)
}

// main writes static failures to stderr and the selected bounded result to stdout.
// Input is the process argument vector; output is an exit code and reviewed result.
// Effects belong to run; pick as the executable entrypoint only.
func main() {
	if e := run(os.Args[1:], os.Stdout); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}

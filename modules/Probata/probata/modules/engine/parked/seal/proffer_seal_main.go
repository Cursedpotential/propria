//go:build parked

// PARKED 2026-09-06 (owner ruling 11:16/11:22): sealing is custody-at-PROMOTION work,
// not an ingest-time step. Keep for reuse; do not rewrite. Excluded from
// `go build ./...` by the build tag above. Original path: cmd/proffer-seal/main.go.
//
// proffer-seal: hash a local file and put it in the parsers' working store.
// Usage:
//
//	proffer-seal -root /data/proffer/source-objects <file> [<file>...]
//
// For each file it prints one line: <upload://sha256>  <bytes>  <path>.
// Files are never moved or modified; sealing is content-addressed and
// idempotent. Data movement between hosts is not this command's job.
//
// Byline: Claude Code · Fable 5.1 · 2026-09-06.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/Cursedpotential/probata/engine/acquisition"
)

func main() {
	root := flag.String("root", os.Getenv("SOURCE_OBJECT_DIR"), "seal root (the source-objects store); defaults to $SOURCE_OBJECT_DIR")
	flag.Parse()
	if *root == "" || flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: proffer-seal -root <source-objects dir> <file> [<file>...]")
		os.Exit(2)
	}
	failed := 0
	for _, path := range flag.Args() {
		sealed, ref, err := acquisition.SealLocalFile(context.Background(), *root, path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "FAIL  %s: %v\n", path, err)
			failed++
			continue
		}
		fmt.Printf("%s  %d  %s\n", ref, sealed.ByteLength, path)
	}
	if failed > 0 {
		os.Exit(1)
	}
}

package humancontext

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/Cursedpotential/probata/engine/xmlsourceframes"
)

const MaxMetadataBytes = 2 * 1024 * 1024

// Bundles executes independent framing and DuckDB extraction units on the shared bundle volume.
// Inputs: configured root, read-only original opener, DuckDB extractor and optional metadata receipt sink.
// Outputs: immutable file references. Effects: derivative files and metadata only, never original writes.
type Bundles struct {
	Root      string
	Opener    SourceOpener
	Extractor Extractor
	Metadata  MetadataStore
}

func fileRef(path string) string {
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(path)}).String()
}
func (b *Bundles) path(ref string) (string, error) {
	u, err := url.Parse(ref)
	if err != nil || u.Scheme != "file" || u.Host != "" || u.RawQuery != "" || u.Fragment != "" {
		return "", errors.New("human bundle requires local file reference")
	}
	root, err := filepath.Abs(b.Root)
	if err != nil {
		return "", err
	}
	p := filepath.Clean(filepath.FromSlash(u.Path))
	rel, err := filepath.Rel(root, p)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", errors.New("human bundle reference escaped root")
	}
	current := root
	for _, part := range append([]string{""}, strings.Split(rel, string(os.PathSeparator))...) {
		if part != "" {
			current = filepath.Join(current, part)
		}
		info, e := os.Lstat(current)
		if e != nil || info.Mode()&os.ModeSymlink != 0 {
			return "", errors.New("human bundle path unavailable or linked")
		}
	}
	return p, nil
}

// ReadManifest reads a bounded first-party human manifest from the configured bundle root.
// Inputs: exact file reference. Outputs: typed source/stage pins. Effects: read-only file access.
// Choose for every stage and viewer readback rather than accepting arbitrary caller paths.
func (b *Bundles) ReadManifest(ref string) (Manifest, error) {
	p, err := b.path(ref)
	if err != nil {
		return Manifest{}, err
	}
	file, err := os.Open(p)
	if err != nil {
		return Manifest{}, err
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, MaxMetadataBytes+1))
	if err != nil || len(raw) > MaxMetadataBytes {
		return Manifest{}, errors.New("human manifest exceeds bound")
	}
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return m, err
	}
	if m.ContractVersion != Contract || m.Source.SourceVersionRef == "" {
		return m, errors.New("human manifest contract invalid")
	}
	return m, nil
}
func (b *Bundles) begin(req Request, stage string) (string, error) {
	if req.ContractVersion != Contract || req.WorkflowID == "" || req.Source.SourceVersionRef == "" || req.Source.SourceRef == "" || req.Source.RegistrationReceiptRef == "" {
		return "", errors.New("human stage requires actual registration references")
	}
	root, err := filepath.Abs(b.Root)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(root, 0750); err != nil {
		return "", err
	}
	return os.MkdirTemp(root, stage+"-")
}
func (b *Bundles) save(ctx context.Context, req Request, dir string, m Manifest) (Result, error) {
	path := filepath.Join(dir, "manifest.json")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0640)
	if err != nil {
		return Result{}, err
	}
	err = json.NewEncoder(file).Encode(m)
	if err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return Result{}, err
	}
	if closeErr != nil {
		return Result{}, closeErr
	}
	status := "complete"
	if m.Pending > 0 || !m.Framing.Complete {
		status = "partial"
	}
	r := Result{ContractVersion: Contract, Stage: m.Stage, Status: status, ManifestRef: fileRef(path), IndexKind: m.IndexKind, Records: m.Records, Pending: m.Pending, Chunks: m.Chunks, Published: m.Published, Verified: m.Verified}
	if b.Metadata != nil {
		return b.Metadata.RecordStage(ctx, req, r)
	}
	return r, nil
}

// Frame reads original bytes once and writes lossless lexical/span manifests with bounded memory.
// Inputs: registered source request. Outputs: framing receipt and manifest reference.
// Effects: derivative-only files and metadata; registration has already happened. Hashing remains separate.
func (b *Bundles) Frame(ctx context.Context, req Request) (Result, error) {
	if r, found, err := b.cachedStage(ctx, req, "framed", req.Source); err != nil || found {
		return r, err
	}
	if b.Opener == nil {
		return Result{}, errors.New("human source opener unavailable")
	}
	if err := ValidateSourcePins(req.Source, req.Source); err != nil {
		return Result{}, err
	}
	reader, pin, err := b.Opener.Open(ctx, req.Source)
	if err != nil {
		return Result{}, err
	}
	if err := ValidateSourcePins(req.Source, pin); err != nil {
		_ = reader.Close()
		return Result{}, err
	}
	dir, err := b.begin(req, "framed")
	if err != nil {
		_ = reader.Close()
		return Result{}, err
	}
	lexPath, spanPath := filepath.Join(dir, "lexemes.jsonl"), filepath.Join(dir, "spans.jsonl")
	lex, err := os.OpenFile(lexPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0640)
	if err != nil {
		_ = reader.Close()
		return Result{}, err
	}
	defer lex.Close()
	spans, err := os.OpenFile(spanPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0640)
	if err != nil {
		_ = reader.Close()
		return Result{}, err
	}
	defer spans.Close()
	lexWriter, spanWriter := bufio.NewWriterSize(lex, 65536), bufio.NewWriterSize(spans, 65536)
	lexEncoder, spanEncoder := json.NewEncoder(lexWriter), json.NewEncoder(spanWriter)
	receipt, err := xmlsourceframes.ScanLexemes(ctx, reader, 2*1024*1024, 64*1024, func(s xmlsourceframes.Span) error { return spanEncoder.Encode(s) }, func(l xmlsourceframes.Lexeme) error { return lexEncoder.Encode(l) })
	if err != nil {
		return Result{}, err
	}
	if pin.ByteLength > 0 && receipt.Bytes != pin.ByteLength {
		return Result{}, errors.New("human original length changed")
	}
	if err := lexWriter.Flush(); err != nil {
		return Result{}, err
	}
	if err := spanWriter.Flush(); err != nil {
		return Result{}, err
	}
	if err := lex.Sync(); err != nil {
		return Result{}, err
	}
	if err := spans.Sync(); err != nil {
		return Result{}, err
	}
	pin.ByteLength = receipt.Bytes
	m := Manifest{ContractVersion: Contract, WorkflowID: req.WorkflowID, Stage: "framed", Source: pin, Framing: receipt, LexemesRef: fileRef(lexPath), SpansRef: fileRef(spanPath), Records: receipt.Elements}
	return b.save(ctx, req, dir, m)
}

// Extract streams one bounded lexical record at a time through DuckDB into a native human bundle.
// Inputs: framed manifest reference and original registration coordinates. Outputs: native bundle receipt.
// Effects: derivative JSONL plus fixed-width pagination offsets; no raw/normalized PostgreSQL rows.
func (b *Bundles) Extract(ctx context.Context, req Request) (Result, error) {

	if b.Extractor == nil {
		return Result{}, errors.New("human DuckDB extractor unavailable")
	}
	m, err := b.ReadManifest(req.ManifestRef)
	if err != nil {
		return Result{}, err
	}
	if err := validateStageManifest(req, m, "framed"); err != nil {
		return Result{}, err
	}
	if r, found, err := b.cachedStage(ctx, req, "extracted", m.Source); err != nil || found {
		return r, err
	}
	dir, err := b.begin(req, "extracted")
	if err != nil {
		return Result{}, err
	}
	path, err := b.path(m.LexemesRef)
	if err != nil {
		return Result{}, err
	}
	input, err := os.Open(path)
	if err != nil {
		return Result{}, err
	}
	defer input.Close()
	recordPath, offsetPath := filepath.Join(dir, "records.jsonl"), filepath.Join(dir, "offsets.bin")
	output, err := os.OpenFile(recordPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0640)
	if err != nil {
		return Result{}, err
	}
	defer output.Close()
	offsets, err := os.OpenFile(offsetPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0640)
	if err != nil {
		return Result{}, err
	}
	defer offsets.Close()
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 65536), 128*1024)
	var root, unit []xmlsourceframes.Lexeme
	var ordinal int64
	size, rootSize := 0, 0
	oversize := false
	count, pending := int64(0), int64(0)
	flush := func() error {
		if len(unit) == 0 {
			return nil
		}
		var record Record
		completeStart := false
		for _, row := range unit {
			if row.Kind == "element_start" && row.ParentID == rootElementID(root) {
				completeStart = true
			}
		}
		pendingRecord := func(reason string) Record {
			native, _ := json.Marshal(map[string]any{"reason": reason, "source_locator": map[string]any{"source_ref": m.Source.SourceRef, "source_version_ref": m.Source.SourceVersionRef, "provider_version_id": m.Source.ProviderVersionID, "record_ordinal": ordinal}, "lexemes_ref": m.LexemesRef, "spans_ref": m.SpansRef})
			return Record{Ordinal: ordinal, Status: "pending", Native: native}
		}
		if oversize {
			record = pendingRecord("lexical_metadata_bound")
		} else if !completeStart {
			record = pendingRecord("incomplete_lexical_record")
		} else {
			var e error
			record, e = b.Extractor.Extract(ctx, m.Source, append(append([]xmlsourceframes.Lexeme{}, root...), unit...))
			if e != nil {
				return e
			}
		}
		if len(record.Native) > MaxMetadataBytes-512 {
			record = pendingRecord("native_metadata_bound")
		}
		if record.Ordinal != ordinal || !json.Valid(record.Native) {
			return errors.New("human DuckDB row identity invalid")
		}
		if record.Status != "complete" && record.Status != "pending" && record.Status != "unparsed" {
			return errors.New("human DuckDB row status invalid")
		}
		position, e := output.Seek(0, io.SeekCurrent)
		if e != nil {
			return e
		}
		if e := binary.Write(offsets, binary.LittleEndian, uint64(position)); e != nil {
			return e
		}
		if e := json.NewEncoder(output).Encode(record); e != nil {
			return e
		}
		count++
		if record.Status != "complete" {
			pending++
		}
		unit = nil
		size = 0
		oversize = false
		return nil
	}
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		var row xmlsourceframes.Lexeme
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
			return Result{}, err
		}
		if row.RecordOrdinal == 0 {
			if row.Kind != "element_end" {
				root = append(root, row)
				rootSize += len(scanner.Bytes())
				if len(root) > 256 || rootSize > MaxMetadataBytes/2 {
					return Result{}, errors.New("human root metadata bound")
				}
			}
			continue
		}
		if ordinal != 0 && row.RecordOrdinal != ordinal {
			if err := flush(); err != nil {
				return Result{}, err
			}
		}
		ordinal = row.RecordOrdinal
		size += len(scanner.Bytes())
		if size+rootSize > MaxMetadataBytes-4096 {
			oversize = true
		}
		if !oversize {
			unit = append(unit, row)
		}
	}
	if err := scanner.Err(); err != nil {
		return Result{}, err
	}
	if err := flush(); err != nil {
		return Result{}, err
	}
	if count != m.Framing.Elements && m.Framing.Complete {
		return Result{}, fmt.Errorf("human extraction accounting: framed %d extracted %d", m.Framing.Elements, count)
	}
	if err := output.Sync(); err != nil {
		return Result{}, err
	}
	if err := offsets.Sync(); err != nil {
		return Result{}, err
	}
	m.Stage = "extracted"
	m.RecordsRef = fileRef(recordPath)
	m.OffsetsRef = fileRef(offsetPath)
	m.Records = count
	m.Pending = pending
	return b.save(ctx, req, dir, m)
}

// rootElementID finds the actual lexical document element used to identify direct record children.
// Inputs: bounded root lexemes. Outputs: lexical ID only. Effects: none; no message semantics.
func rootElementID(root []xmlsourceframes.Lexeme) int64 {
	for _, row := range root {
		if row.Kind == "element_start" && row.ParentID == 0 {
			return row.ElementID
		}
	}
	return 0
}

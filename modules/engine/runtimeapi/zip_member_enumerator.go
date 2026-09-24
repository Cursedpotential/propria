package runtimeapi

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/Cursedpotential/probata/engine/activities"
	platformpostgres "github.com/Cursedpotential/probata/engine/postgres"
	"github.com/Cursedpotential/probata/engine/proffer"
)

// These are admission limits, not claims about the largest valid ZIP. A source
// above a limit stays retained and gets a failed inventory attempt for review.
const (
	zipMaxArchiveBytes   int64 = 2 << 30
	zipMaxMembers              = 10_000
	zipMaxDirectoryBytes int64 = 64 << 20
	zipMaxMemberBytes    int64 = 128 << 20
	zipMaxExpandedBytes  int64 = 1 << 30
	zipMaxExpansionRatio int64 = 100
)

// NewZIPMemberEnumerator inventories retained filesystem ZIP/OOXML objects.
// It never extracts members or creates a repaired source. The exact retained
// original membership is checked before opening the file.
func NewZIPMemberEnumerator(db platformpostgres.DB) (activities.MemberEnumerator, error) {
	if db == nil {
		return nil, errors.New("ZIP member enumerator: database is required")
	}
	return zipMemberEnumerator{db: db}, nil
}

type zipMemberEnumerator struct{ db platformpostgres.DB }

func (e zipMemberEnumerator) EnumerateMembers(ctx context.Context, input activities.SourceObservationInput) (activities.MemberStream, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	format := strings.ToLower(strings.TrimSpace(input.DeclaredFormat))
	switch format {
	case "zip", "archive", "xlsx", "docx", "pptx", "ooxml":
	default:
		return nonContainerMemberEnumerator{}.EnumerateMembers(ctx, input)
	}
	object, _, err := resolveRetainedObjectForObservation(ctx, e.db, input)
	if err != nil {
		return nil, fmt.Errorf("resolve ZIP original: %w", err)
	}
	if object.storageClass != "filesystem" {
		return nil, fmt.Errorf("ZIP inventory unsupported retained storage class %q", object.storageClass)
	}
	if object.byteLength < 0 || object.byteLength > zipMaxArchiveBytes {
		return nil, fmt.Errorf("ZIP inventory archive byte limit exceeded: %d", object.byteLength)
	}
	filename, err := pathFromFileURI(object.objectURI)
	if err != nil {
		return nil, fmt.Errorf("resolve ZIP file URI: %w", err)
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("open retained ZIP: %w", err)
	}
	closeOnError := true
	defer func() {
		if closeOnError {
			_ = file.Close()
		}
	}()
	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat retained ZIP: %w", err)
	}
	if !stat.Mode().IsRegular() || stat.Size() != object.byteLength {
		return nil, errors.New("retained ZIP is not a regular file or its byte length changed")
	}
	if len(object.contentSHA256) != sha256.Size {
		return nil, errors.New("retained ZIP has no valid content SHA-256")
	}
	sourceHash := sha256.New()
	buffer := make([]byte, 64*1024)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		n, readErr := file.Read(buffer)
		if n > 0 {
			_, _ = sourceHash.Write(buffer[:n])
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("read retained ZIP for digest verification: %w", readErr)
		}
	}
	if subtle.ConstantTimeCompare(sourceHash.Sum(nil), object.contentSHA256) != 1 {
		return nil, errors.New("retained ZIP content SHA-256 changed")
	}
	archive, err := openAdmittedZIP(ctx, file, stat.Size(), zip.NewReader)
	if err != nil {
		return nil, fmt.Errorf("invalid or unsupported ZIP structure: %w", err)
	}
	if len(archive.File) > zipMaxMembers {
		return nil, fmt.Errorf("ZIP inventory member limit exceeded: %d", len(archive.File))
	}
	entries := make([]zipMemberEntry, 0, len(archive.File))
	seen := make(map[string]struct{}, len(archive.File))
	var total int64
	for _, member := range archive.File {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := validZIPMemberName(member.Name); err != nil {
			return nil, fmt.Errorf("invalid ZIP member name: %w", err)
		}
		normalizedName := strings.TrimSuffix(member.Name, "/")
		if _, exists := seen[normalizedName]; exists {
			return nil, fmt.Errorf("duplicate normalized ZIP member name %q", normalizedName)
		}
		seen[normalizedName] = struct{}{}
		if member.Flags&1 != 0 {
			return nil, fmt.Errorf("unsupported encrypted ZIP member %q", member.Name)
		}
		mode := member.Mode()
		if (!mode.IsRegular() && !mode.IsDir()) || mode.IsDir() != strings.HasSuffix(member.Name, "/") {
			return nil, fmt.Errorf("unsupported ZIP member file mode for %q", member.Name)
		}
		if member.UncompressedSize64 > uint64(zipMaxMemberBytes) || member.UncompressedSize64 > uint64(zipMaxExpandedBytes-total) {
			return nil, fmt.Errorf("ZIP member expanded byte limit exceeded at %q", member.Name)
		}
		offset, err := member.DataOffset()
		if err != nil {
			return nil, fmt.Errorf("invalid ZIP member offset for %q: %w", member.Name, err)
		}
		if offset < 0 || uint64(offset) > uint64(stat.Size()) || member.CompressedSize64 > uint64(stat.Size()-offset) {
			return nil, fmt.Errorf("ZIP member compressed range exceeds source at %q", member.Name)
		}
		if member.UncompressedSize64 > 0 && (member.CompressedSize64 == 0 ||
			member.UncompressedSize64 > member.CompressedSize64*uint64(zipMaxExpansionRatio)) {
			return nil, fmt.Errorf("ZIP member expansion ratio limit exceeded at %q", member.Name)
		}
		total += int64(member.UncompressedSize64)
		entries = append(entries, zipMemberEntry{file: member, offset: offset})
	}
	if format == "xlsx" || format == "docx" || format == "pptx" || format == "ooxml" {
		if _, ok := seen["[Content_Types].xml"]; !ok {
			return nil, errors.New("OOXML package is missing [Content_Types].xml")
		}
		if _, ok := seen["_rels/.rels"]; !ok {
			return nil, errors.New("OOXML package is missing _rels/.rels")
		}
	}
	// ZIP central-directory order can differ from on-disk order. The latter is
	// the source order preserved by the inventory ordinal.
	sort.Slice(entries, func(i, j int) bool { return entries[i].offset < entries[j].offset })
	var previousEnd int64
	for i, entry := range entries {
		if i > 0 && entry.offset <= entries[i-1].offset {
			return nil, errors.New("ZIP member compressed ranges share an offset")
		}
		if i > 0 && entry.offset < previousEnd {
			return nil, errors.New("ZIP member compressed ranges overlap")
		}
		previousEnd = entry.offset + int64(entry.file.CompressedSize64)
	}
	closeOnError = false
	return &zipMemberStream{file: file, entries: entries, original: input.OriginalRef, sourceVersion: input.SourceVersionRef}, nil
}

func openAdmittedZIP(
	ctx context.Context,
	source io.ReaderAt,
	size int64,
	parse func(io.ReaderAt, int64) (*zip.Reader, error),
) (*zip.Reader, error) {
	// archive/zip allocates and parses the central directory before returning.
	// Bound physical records and bytes independently of the EOCD count first.
	if err := admitZIPCentralDirectory(ctx, source, size); err != nil {
		return nil, err
	}
	return parse(source, size)
}

// admitZIPCentralDirectory performs constant-memory structural admission before
// archive/zip can allocate File records, names, comments, or extra fields. It
// intentionally accepts only a single-disk, non-ZIP64 directory ending exactly
// at the EOCD; unsupported layouts stay retained for a failed inventory attempt.
func admitZIPCentralDirectory(ctx context.Context, source io.ReaderAt, size int64) error {
	const endLength = 22
	const headerLength = 46
	const endSignature = 0x06054b50
	const headerSignature = 0x02014b50
	if size < endLength {
		return errors.New("missing ZIP end of central directory")
	}
	tailSize := int64(endLength + 65535)
	if size < tailSize {
		tailSize = size
	}
	tail := make([]byte, int(tailSize))
	if _, err := source.ReadAt(tail, size-tailSize); err != nil {
		return fmt.Errorf("read ZIP end of central directory: %w", err)
	}
	endAt := -1
	for i := len(tail) - endLength; i >= 0; i-- {
		if binary.LittleEndian.Uint32(tail[i:]) == endSignature {
			commentEnd := i + endLength + int(binary.LittleEndian.Uint16(tail[i+20:]))
			if commentEnd > len(tail) {
				return errors.New("truncated ZIP end of central directory comment")
			}
			if commentEnd != len(tail) {
				return errors.New("trailing bytes after ZIP end of central directory")
			}
			endAt = i
			break
		}
	}
	if endAt < 0 {
		return errors.New("missing or truncated ZIP end of central directory")
	}
	end := tail[endAt:]
	if binary.LittleEndian.Uint16(end[4:]) != 0 || binary.LittleEndian.Uint16(end[6:]) != 0 {
		return errors.New("multi-disk ZIP is unsupported")
	}
	recordsOnDisk := binary.LittleEndian.Uint16(end[8:])
	records := binary.LittleEndian.Uint16(end[10:])
	directoryBytes := binary.LittleEndian.Uint32(end[12:])
	directoryOffset := binary.LittleEndian.Uint32(end[16:])
	if records == 0xffff || directoryBytes == 0xffffffff || directoryOffset == 0xffffffff {
		return errors.New("ZIP64 central directory is unsupported")
	}
	if recordsOnDisk != records {
		return errors.New("ZIP central directory record counts disagree")
	}
	if records > zipMaxMembers {
		return fmt.Errorf("ZIP inventory member limit exceeded: %d", records)
	}
	if int64(directoryBytes) > zipMaxDirectoryBytes {
		return fmt.Errorf("ZIP central directory byte limit exceeded: %d", directoryBytes)
	}
	endOffset := size - tailSize + int64(endAt)
	if int64(directoryOffset)+int64(directoryBytes) != endOffset {
		return errors.New("ZIP central directory extent does not match end record")
	}
	position := int64(directoryOffset)
	count := 0
	var header [headerLength]byte
	for position < endOffset {
		if err := ctx.Err(); err != nil {
			return err
		}
		if endOffset-position < headerLength {
			return errors.New("truncated ZIP central directory header")
		}
		if _, err := source.ReadAt(header[:], position); err != nil {
			return fmt.Errorf("read ZIP central directory header: %w", err)
		}
		if binary.LittleEndian.Uint32(header[:]) != headerSignature {
			return errors.New("invalid ZIP central directory header")
		}
		entryBytes := int64(headerLength) + int64(binary.LittleEndian.Uint16(header[28:])) +
			int64(binary.LittleEndian.Uint16(header[30:])) + int64(binary.LittleEndian.Uint16(header[32:]))
		if entryBytes > endOffset-position {
			return errors.New("truncated ZIP central directory entry")
		}
		count++
		if count > zipMaxMembers {
			return fmt.Errorf("ZIP inventory member limit exceeded: more than %d physical entries", zipMaxMembers)
		}
		position += entryBytes
	}
	if count != int(records) {
		return fmt.Errorf("ZIP central directory count mismatch: declared %d, found %d", records, count)
	}
	return nil
}

func validZIPMemberName(name string) error {
	if name == "" || strings.ContainsAny(name, "\\\x00") || strings.HasPrefix(name, "/") ||
		strings.Contains(name, ":") || path.Clean(name) != strings.TrimSuffix(name, "/") {
		return fmt.Errorf("unsafe path %q", name)
	}
	for _, part := range strings.Split(name, "/") {
		if part == ".." || part == "." {
			return fmt.Errorf("unsafe path %q", name)
		}
	}
	return nil
}

type zipMemberEntry struct {
	file   *zip.File
	offset int64
}

type zipMemberStream struct {
	file          *os.File
	entries       []zipMemberEntry
	original      proffer.Ref
	sourceVersion proffer.Ref
	next          int
	closed        bool
}

func (s *zipMemberStream) Next(ctx context.Context) (activities.InventoryMember, error) {
	if s.closed {
		return activities.InventoryMember{}, errors.New("ZIP member stream is closed")
	}
	if err := ctx.Err(); err != nil {
		return activities.InventoryMember{}, err
	}
	if s.next == len(s.entries) {
		return activities.InventoryMember{}, io.EOF
	}
	entry := s.entries[s.next]
	reader, err := entry.file.Open()
	if err != nil {
		return activities.InventoryMember{}, fmt.Errorf("open ZIP member %q: %w", entry.file.Name, err)
	}
	hash := sha256.New()
	var count int64
	buffer := make([]byte, 64*1024)
	for {
		if err := ctx.Err(); err != nil {
			_ = reader.Close()
			return activities.InventoryMember{}, err
		}
		n, readErr := reader.Read(buffer)
		if n > 0 {
			count += int64(n)
			if count > zipMaxMemberBytes || count > int64(entry.file.UncompressedSize64) {
				_ = reader.Close()
				return activities.InventoryMember{}, fmt.Errorf("ZIP member expanded byte limit exceeded at %q", entry.file.Name)
			}
			_, _ = hash.Write(buffer[:n])
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			_ = reader.Close()
			return activities.InventoryMember{}, fmt.Errorf("read ZIP member %q: %w", entry.file.Name, readErr)
		}
	}
	if err := reader.Close(); err != nil {
		return activities.InventoryMember{}, fmt.Errorf("close ZIP member %q: %w", entry.file.Name, err)
	}
	if count != int64(entry.file.UncompressedSize64) {
		return activities.InventoryMember{}, fmt.Errorf("ZIP member byte count mismatch at %q", entry.file.Name)
	}
	ordinal := s.next
	s.next++
	offset := entry.offset
	return activities.InventoryMember{
		Ordinal:              int64(ordinal),
		MemberRef:            proffer.Ref(fmt.Sprintf("zip-member:%s:%d", s.sourceVersion, ordinal)),
		ParentRef:            s.original,
		ByteLength:           count,
		Name:                 entry.file.Name,
		SourceByteOffset:     &offset,
		CompressedByteLength: int64(entry.file.CompressedSize64),
		SHA256:               hex.EncodeToString(hash.Sum(nil)),
	}, nil
}

func (s *zipMemberStream) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	return s.file.Close()
}

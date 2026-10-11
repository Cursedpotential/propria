// Byline: Codex · GPT-6.1 · 2026-10-06. Selective port from held 7a1db624; independent byte-only operation.
// Package sourceintegrity records whole-stream byte coverage independently of format detection, credentials, and custody.
package sourceintegrity

import (
	"context"
	"errors"
	"io"
)

// CheckVersion names the whole-stream zero-content check, not a format validator.
const CheckVersion = "whole_stream_zero_v1"

// Assessment records byte observations and their coverage without claiming usable content.
// Inputs: source stream and an optional expected length. Outputs: fixed evidence fields, never source bytes.
// Complete requires EOF and agreement with a known expected length; nonzero observations alone leave Status unknown.
// FormatValidation remains unassessed because this unit performs no parsing, member validation, or hashing.
type Assessment struct {
	CheckVersion     string `json:"check_version"`
	Status           string `json:"status"`
	Reason           string `json:"reason"`
	CheckedBytes     int64  `json:"checked_bytes"`
	ExpectedBytes    int64  `json:"expected_bytes"`
	Complete         bool   `json:"complete"`
	EOFReached       bool   `json:"eof_reached"`
	NonzeroObserved  bool   `json:"nonzero_observed"`
	FormatValidation string `json:"format_validation"`
}

// Unchecked creates explicit unknown evidence for a source whose stream has not been inspected.
// Input: expected byte length, or -1 for unknown. Output: an incomplete assessment with no bytes checked.
// Side effects: none. Pick when access fails or a prior metadata manifest lacks whole-stream coverage.
func Unchecked(expectedBytes int64) Assessment {
	return Assessment{CheckVersion: CheckVersion, Status: "unknown", Reason: "stream_not_checked", ExpectedBytes: expectedBytes, FormatValidation: "unassessed"}
}

// InspectStream checks every byte for zero content using a fixed 64 KiB buffer.
// Inputs: non-nil context, read-only reader positioned at source start, expected length (-1 means unknown).
// Outputs: checked bytes, EOF/size coverage, zero_byte/all_zero/unknown status, and any read or cancellation error.
// Side effects: reads the stream without closing it; no writes, hashes, format validation, or body logging.
// Pick over prefix detection when all_zero requires whole-source evidence; credentials remain a separate unit.
// Cancellation is checked between reads; callers must supply a context-aware reader to interrupt blocked I/O.
func InspectStream(ctx context.Context, reader io.Reader, expectedBytes int64) (Assessment, error) {
	result := Unchecked(expectedBytes)
	if ctx == nil || reader == nil || expectedBytes < -1 {
		result.Reason = "invalid_stream_input"
		return result, errors.New("invalid integrity stream input")
	}
	buffer := make([]byte, 64<<10)
	idleReads := 0
	for {
		if err := ctx.Err(); err != nil {
			result.Reason = "context_canceled"
			return result, err
		}
		n, err := reader.Read(buffer)
		result.CheckedBytes += int64(n)
		for _, value := range buffer[:n] {
			if value != 0 {
				result.NonzeroObserved = true
				break
			}
		}
		if cancelErr := ctx.Err(); cancelErr != nil {
			result.Reason = "context_canceled"
			return result, cancelErr
		}
		if err == io.EOF {
			result.EOFReached = true
			if expectedBytes >= 0 && result.CheckedBytes != expectedBytes {
				result.Reason = "source_size_mismatch"
				return result, errors.New("integrity source size mismatch")
			}
			result.Complete = true
			switch {
			case result.CheckedBytes == 0:
				result.Status, result.Reason = "zero_byte", "source_zero_bytes"
			case !result.NonzeroObserved:
				result.Status, result.Reason = "all_zero", "every_source_byte_zero"
			default:
				result.Reason = "format_validation_unassessed"
			}
			return result, nil
		}
		if err != nil {
			result.Reason = "stream_read_failed"
			return result, err
		}
		if n == 0 {
			idleReads++
			if idleReads >= 100 {
				result.Reason = "stream_no_progress"
				return result, io.ErrNoProgress
			}
		} else {
			idleReads = 0
		}
	}
}

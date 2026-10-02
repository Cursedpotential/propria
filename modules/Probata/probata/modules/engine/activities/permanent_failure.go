// Byline: Claude Code · Fable 5.1 · 2026-09-20
package activities

import (
	"errors"
	"strings"

	"go.temporal.io/sdk/temporal"
)

// PermanentFailureType marks an Activity failure that cannot succeed on a
// retry: the same request over the same bytes fails the same way. Temporal
// stops retrying it at once, so the workflow's own recovery (handler
// alternatives, repair, operator decision) is offered immediately instead of
// after every bounded attempt has been spent. Owner 2026-09-20: a retry cap
// with a "cannot succeed, stop" classification.
const PermanentFailureType = "permanent_input_failure"

type permanentFailure struct{ err error }

func (p permanentFailure) Error() string { return p.err.Error() }
func (p permanentFailure) Unwrap() error { return p.err }

// permanent marks err as a failure of the request or of the source content.
func permanent(err error) error {
	if err == nil {
		return nil
	}
	return permanentFailure{err: err}
}

// PermanentSourceError is permanent for callers outside this package: a repository
// that proves, before running anything, that the source cannot be read by its template.
func PermanentSourceError(err error) error { return permanent(err) }

// duckDBDeterministicErrors are DuckDB error classes raised by the query or the
// content it reads. "IO Error", "HTTP Error" and out-of-memory are deliberately
// absent: storage, network and capacity trouble can clear on its own.
var duckDBDeterministicErrors = []string{
	"Invalid Input Error:", "Conversion Error:", "Parser Error:", "Binder Error:",
	"Catalog Error:", "Not implemented Error:", "Out of Range Error:",
}

// stopRetryingPermanent converts a permanent failure into Temporal's
// non-retryable application error. Every other error passes through untouched
// and keeps the stage's bounded RetryPolicy.
func stopRetryingPermanent(err error) error {
	if err == nil {
		return nil
	}
	var marked permanentFailure
	deterministic := errors.As(err, &marked)
	if !deterministic {
		message := err.Error()
		for _, class := range duckDBDeterministicErrors {
			if strings.Contains(message, class) {
				deterministic = true
				break
			}
		}
	}
	if !deterministic {
		return err
	}
	return temporal.NewNonRetryableApplicationError(err.Error(), PermanentFailureType, err)
}

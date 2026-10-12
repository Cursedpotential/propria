package flow

import (
	failurepb "go.temporal.io/api/failure/v1"
	"go.temporal.io/sdk/temporal"
	"regexp"
)

const extractionFailureCauseLimit = 6
const extractionFailureSizeWrapper = "Failure exceeds size limit."
const extractionRateLimitReason = "Upstream HTTP 429: Too Many Requests"

var extractionHTTP429 = regexp.MustCompile(`\b429\b`)

// pendingExtractionFailure exposes a fixed safe reason for a structured rate limit hidden by Temporal's size wrapper.
// Inputs: actual server failure chain. Output: bounded outer message or known safe reason.
// Effects: none; inspect at most six nodes, never return nested provider text, stacks or payloads.
// Choose for pending Activity status; ordinary failures retain their existing display.
func pendingExtractionFailure(failure *failurepb.Failure) string {
	if failure.GetMessage() == extractionFailureSizeWrapper {
		for cause, n := failure, 0; cause != nil && n < extractionFailureCauseLimit; cause, n = cause.GetCause(), n+1 {
			if cause.GetApplicationFailureInfo().GetType() == "RateLimitError" && extractionHTTP429.MatchString(cause.GetMessage()) {
				return extractionRateLimitReason
			}
		}
	}
	return boundedExtractionFailure(failure.GetMessage())
}

// terminalExtractionFailureReason recognizes the same safe cause within a bounded SDK error chain.
// Inputs: actual terminal error. Output: fixed safe reason or empty when unverified.
// Effects: none; choose to annotate recovered Progress without replacing its original details.
func terminalExtractionFailureReason(err error) string {
	wrapped := false
	for n := 0; err != nil && n < extractionFailureCauseLimit; n++ {
		if app, ok := err.(*temporal.ApplicationError); ok {
			wrapped = wrapped || app.Message() == extractionFailureSizeWrapper
			if wrapped && app.Type() == "RateLimitError" && extractionHTTP429.MatchString(app.Message()) {
				return extractionRateLimitReason
			}
		}
		next, ok := err.(interface{ Unwrap() error })
		if !ok {
			break
		}
		err = next.Unwrap()
	}
	return ""
}

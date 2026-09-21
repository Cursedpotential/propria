// Byline: Claude Code · Fable 5.1 · 2026-09-20
package activities

import (
	"context"
	"errors"
	"testing"

	"go.temporal.io/sdk/temporal"
)

func nonRetryable(err error) bool {
	var application *temporal.ApplicationError
	return errors.As(err, &application) && application.NonRetryable() && application.Type() == PermanentFailureType
}

// The exact failure the owner hit on 2026-09-20: the same bytes fail the same
// way on every attempt, so Temporal must stop and let the workflow offer the
// handler alternatives instead of burning five 30-minute attempts.
func TestExecuteStructuredELTStopsRetryingAFailureThatCannotSucceed(t *testing.T) {
	duckdb := errors.New(`ERROR: (PGDuckDB/CreatePlan) Prepared query returned an error: Invalid Input Error: File s3://salem-data/consignatio/vault/v1/sms-20260110021338.xml exceeds maximum size limit (16777216 bytes) (SQLSTATE XX000)`)
	for name, openErr := range map[string]error{
		"open": duckdb,
	} {
		req, store, _ := structuredELTStageFixture()
		repo := &fakeStructuredELTRowRepository{err: openErr}
		_, err := (StructuredELTActivities{Rows: repo, Store: store, Authorization: structuredELTAuthorization()}).ExecuteStructuredELT(context.Background(), req)
		if !nonRetryable(err) {
			t.Fatalf("%s: deterministic DuckDB input failure stayed retryable: %v", name, err)
		}
	}
}

func TestExecuteStructuredELTEmptyAndMalformedRequestsAreNotRetried(t *testing.T) {
	req, store, _ := structuredELTStageFixture()
	repo := &fakeStructuredELTRowRepository{reader: &fakeStructuredELTRowReader{}}
	activity := StructuredELTActivities{Rows: repo, Store: store, Authorization: structuredELTAuthorization()}
	if _, err := activity.ExecuteStructuredELT(context.Background(), req); !nonRetryable(err) {
		t.Fatalf("an extraction that produced no records stayed retryable: %v", err)
	}
	delete(req.Refs, "original")
	if _, err := activity.ExecuteStructuredELT(context.Background(), req); !nonRetryable(err) {
		t.Fatalf("a request missing a required reference stayed retryable: %v", err)
	}
}

// Connectivity and storage trouble can clear on its own: it must keep retrying.
func TestExecuteStructuredELTKeepsRetryingTransientFailures(t *testing.T) {
	for _, transient := range []error{
		errors.New("failed to connect to `host=100.91.190.107`: dial tcp: i/o timeout"),
		errors.New("ERROR: (PGDuckDB/CreatePlan) IO Error: Connection error for HTTP GET to 's3://salem-data/x.xml'"),
		context.DeadlineExceeded,
	} {
		req, store, _ := structuredELTStageFixture()
		repo := &fakeStructuredELTRowRepository{err: transient}
		_, err := (StructuredELTActivities{Rows: repo, Store: store, Authorization: structuredELTAuthorization()}).ExecuteStructuredELT(context.Background(), req)
		if err == nil || nonRetryable(err) {
			t.Fatalf("transient failure was marked permanent: %v", err)
		}
	}
}

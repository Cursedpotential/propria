package sourcepreview

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/objectstores"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

func TestWorkflowCarriesOnlyOpaqueReferencesAndCounts(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	id := strings.Repeat("a", 43)
	env.RegisterActivityWithOptions(func(context.Context, string) (Receipt, error) { return Receipt{}, nil }, activity.RegisterOptions{Name: ActivityName})
	env.OnActivity(ActivityName, mock.Anything, id).Return(Receipt{PageID: id, Rows: 1, ScannedBytes: 100}, nil).Once()
	env.ExecuteWorkflow(Workflow, id)
	if err := env.GetWorkflowError(); err != nil {
		t.Fatal(err)
	}
	var receipt Receipt
	if err := env.GetWorkflowResult(&receipt); err != nil || receipt.PageID != id || receipt.Rows != 1 {
		t.Fatal("receipt mismatch")
	}
	env.AssertExpectations(t)
}

type syntheticObjectClient struct{ fixture string }

func (c syntheticObjectClient) HeadObject(context.Context, *s3.HeadObjectInput, ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	return &s3.HeadObjectOutput{ETag: aws.String("etag"), VersionId: aws.String("v1"), ContentLength: aws.Int64(int64(len(c.fixture)))}, nil
}
func (c syntheticObjectClient) GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return &s3.GetObjectOutput{ETag: aws.String("etag"), VersionId: aws.String("v1"), ContentLength: aws.Int64(int64(len(c.fixture))), ContentRange: aws.String(fmt.Sprintf("bytes 0-%d/%d", len(c.fixture)-1, len(c.fixture))), Body: io.NopCloser(strings.NewReader(c.fixture))}, nil
}
func TestActivityUsesPrivateCallbackAndReturnsNoMessagePayload(t *testing.T) {
	id := strings.Repeat("a", 43)
	fixture := "<smses><sms body='private synthetic text'/></smses>"
	token := strings.Repeat("s", 32)
	tokenFile := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(tokenFile, []byte(token), 0600); err != nil {
		t.Fatal(err)
	}
	var completed Completion
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			t.Error("missing callback auth")
		}
		if strings.HasSuffix(r.URL.Path, "/task") {
			json.NewEncoder(w).Encode(Task{Request: Request{SourceRef: "b2://synthetic/messages.xml", ExpectedETag: "etag", VersionID: "v1", Limit: 25}})
			return
		}
		if strings.HasSuffix(r.URL.Path, "/result") {
			calls++
			if err := json.NewDecoder(r.Body).Decode(&completed); err != nil {
				t.Error(err)
			}
			json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
			return
		}
		t.Error("unexpected callback")
	}))
	defer server.Close()
	roots, _ := objectstores.ParseRoots(`[{"id":"synthetic","label":"Synthetic","url":"b2://synthetic/"}]`)
	a := &Activity{Provider: &Provider{Roots: roots, Clients: map[string]ObjectClient{"b2": syntheticObjectClient{fixture}}}, CallbackURL: server.URL, TokenFile: tokenFile, Client: server.Client()}
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestActivityEnvironment()
	env.RegisterActivity(a.Read)
	value, err := env.ExecuteActivity(a.Read, id)
	if err != nil {
		t.Fatal(err)
	}
	var receipt Receipt
	if err := value.Get(&receipt); err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(receipt)
	if calls != 1 || len(completed.Page.Messages) != 1 || completed.Page.Messages[0].Body != "private synthetic text" || strings.Contains(string(encoded), "private") || receipt.Rows != 1 {
		t.Fatal("private callback boundary failed")
	}
}

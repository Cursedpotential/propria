package sourcepreview

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const WorkflowName = "SourceReadPreviewWorkflow"
const ActivityName = "read_raw_source_preview_activity"

// Workflow schedules exactly one raw-source preview Activity without carrying message content.
// Inputs: opaque pending page ID. Outputs: opaque receipt and bounded counts only.
// Effects: Temporal orchestration history; choose for viewing raw SMS/MMS before extraction or ingestion.
func Workflow(ctx workflow.Context, pageID string) (Receipt, error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 12 * time.Second, ScheduleToCloseTimeout: 20 * time.Second,
		HeartbeatTimeout: 4 * time.Second, WaitForCancellation: true,
		RetryPolicy: &temporal.RetryPolicy{MaximumAttempts: 1},
	})
	var result Receipt
	err := workflow.ExecuteActivity(ctx, ActivityName, pageID).Get(ctx, &result)
	return result, err
}

type Activity struct {
	Provider    *Provider
	CallbackURL string
	TokenFile   string
	Client      *http.Client
}

// Read fetches a private task, scans one pinned range and sends its bounded view to the starter cache.
// Inputs: opaque page ID and server-configured provider/callback/token-file dependencies.
// Outputs: opaque page receipt/counts; effects: bounded source GET and authenticated ephemeral-cache callback.
// Choose as the sole Activity of Workflow; message bodies, continuation fields and tokens never enter history.
func (a *Activity) Read(ctx context.Context, pageID string) (Receipt, error) {
	if len(pageID) != 43 || a.Provider == nil || a.Client == nil {
		return Receipt{}, ErrInvalid
	}
	for _, r := range pageID {
		if !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' || r == '-') {
			return Receipt{}, ErrInvalid
		}
	}
	base, err := url.Parse(a.CallbackURL)
	if err != nil || base.Host == "" || (base.Scheme != "http" && base.Scheme != "https") || base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return Receipt{}, ErrUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				activity.RecordHeartbeat(ctx)
			}
		}
	}()
	var task Task
	endpoint := strings.TrimRight(a.CallbackURL, "/") + "/reference-import/source-preview/" + pageID
	if err = a.exchange(ctx, http.MethodGet, endpoint+"/task", nil, &task); err != nil {
		return Receipt{}, err
	}
	source, reader, err := a.Provider.Open(ctx, task.Request, task.State.Offset)
	var completion Completion
	if err == nil {
		completion, err = Scan(ctx, reader, source, task.State, task.Request.Limit)
		reader.Close()
	}
	if err != nil {
		code := "source_preview_unavailable"
		if err == ErrInvalid {
			code = ErrInvalid.Error()
		}
		if err == ErrChanged {
			code = ErrChanged.Error()
		}
		completion = Completion{ErrorCode: code}
	}
	completion.Page.PageID = pageID
	if callbackErr := a.exchange(ctx, http.MethodPost, endpoint+"/result", completion, nil); callbackErr != nil {
		return Receipt{}, callbackErr
	}
	return Receipt{PageID: pageID, Rows: len(completion.Page.Messages), ScannedBytes: completion.Page.ScannedBytes}, nil
}

func (a *Activity) exchange(ctx context.Context, method, endpoint string, value, result any) error {
	file, err := os.Open(a.TokenFile)
	if err != nil {
		return ErrUnavailable
	}
	token, err := io.ReadAll(io.LimitReader(file, 4097))
	file.Close()
	if err != nil || len(token) > 4096 || len(bytes.TrimSpace(token)) < 32 {
		return ErrUnavailable
	}
	token = bytes.TrimSpace(token)
	var body []byte
	if value != nil {
		body, err = json.Marshal(value)
		if err != nil || len(body) > MaxPayloadBytes {
			return ErrInvalid
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return ErrUnavailable
	}
	req.Header.Set("Authorization", "Bearer "+string(token))
	req.Header.Set("Content-Type", "application/json")
	out, err := a.Client.Do(req)
	if err != nil {
		return ErrUnavailable
	}
	defer out.Body.Close()
	if out.StatusCode != http.StatusOK {
		return ErrUnavailable
	}
	if result != nil {
		decoder := json.NewDecoder(io.LimitReader(out.Body, MaxPayloadBytes+1))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(result); err != nil {
			return ErrUnavailable
		}
	}
	return nil
}

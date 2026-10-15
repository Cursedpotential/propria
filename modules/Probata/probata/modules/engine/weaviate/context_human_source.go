package weaviate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Cursedpotential/probata/engine/humancontext"
)

// HumanSourceTarget adapts the existing HTTP transport to separately typed native-human objects.
// Inputs: configured existing Store/collection/vector. Outputs: idempotent writes and exact GET readback.
// Effects: object requests only; no collection creation or fabricated PostgreSQL coordinates.
type HumanSourceTarget struct{ Store Store }

// validate pins native conversation chunks to the existing approved collection and named vector.
// Inputs: configured target. Outputs: configuration error; effects: none, before any HTTP request.
func (t *HumanSourceTarget) validate() error {
	if err := t.Store.validate(); err != nil {
		return err
	}
	if t.Store.Collection != "ProfferChunks20261002" || t.Store.VectorName != "text_nim" {
		return errors.New("native conversation chunks require ProfferChunks20261002/text_nim")
	}
	return nil
}

// request bounds native search response bytes without changing the legacy transport.
// Inputs: method, relative path and optional body. Outputs: status and at most four MiB; effects: one HTTP request.
func (t *HumanSourceTarget) request(ctx context.Context, method, path string, body any) (int, []byte, error) {
	if err := t.validate(); err != nil {
		return 0, nil, err
	}
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(t.Store.BaseURL, "/")+path, reader)
	if err != nil {
		return 0, nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if t.Store.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+t.Store.APIKey)
	}
	resp, err := t.Store.client().Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	const limit = 4 * 1024 * 1024
	payload, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return 0, nil, err
	}
	if len(payload) > limit {
		return 0, nil, errors.New("native search response exceeds four MiB budget")
	}
	return resp.StatusCode, payload, nil
}

// Put upserts one deterministic native-human projection into the configured existing collection.
// Inputs: bounded human object. Outputs: write error or nil. Effects: one search object upsert.
// Choose for native context bundles; PublishObjects retains its legacy PostgreSQL contract.
func (t *HumanSourceTarget) Put(ctx context.Context, object humancontext.SearchObject) error {
	if err := t.validate(); err != nil {
		return err
	}
	if object.ID == "" || len(object.Vector) == 0 {
		return errors.New("invalid native human object")
	}
	body := map[string]any{"id": object.ID, "class": t.Store.Collection, "properties": object.Properties, "vectors": map[string][]float32{t.Store.VectorName: object.Vector}}
	status, payload, err := t.request(ctx, http.MethodPost, "/v1/batch/objects", map[string]any{"objects": []any{body}})
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return errors.New("native human search write failed")
	}
	var results []struct {
		ID     string `json:"id"`
		Result struct {
			Status string          `json:"status"`
			Errors json.RawMessage `json:"errors"`
		} `json:"result"`
	}
	if err := json.Unmarshal(payload, &results); err != nil {
		return err
	}
	if len(results) != 1 || results[0].ID != object.ID || results[0].Result.Status != "SUCCESS" || (len(results[0].Result.Errors) > 0 && string(results[0].Result.Errors) != "null") {
		return errors.New("native human search batch rejected object")
	}
	return nil
}

// Get reads the complete stored properties and the configured named vector for independent verification.
// Inputs: exact deterministic projection ID. Outputs: typed native object. Effects: one search GET.
// Choose over acknowledged-write counts when finishing the indexing path.
func (t *HumanSourceTarget) Get(ctx context.Context, id string) (humancontext.SearchObject, error) {
	if strings.TrimSpace(id) == "" {
		return humancontext.SearchObject{}, errors.New("native search object requires id")
	}
	status, payload, err := t.request(ctx, http.MethodGet, "/v1/objects/"+t.Store.Collection+"/"+url.PathEscape(id)+"?include=vector", nil)
	if err != nil {
		return humancontext.SearchObject{}, err
	}
	if status != http.StatusOK {
		return humancontext.SearchObject{}, errors.New("native human search object read failed")
	}
	var object StoredObject
	if err := json.Unmarshal(payload, &object); err != nil {
		return humancontext.SearchObject{}, err
	}
	if object.ID != id || object.Class != t.Store.Collection {
		return humancontext.SearchObject{}, errors.New("native human search object identity mismatch")
	}
	return humancontext.SearchObject{ID: object.ID, Properties: object.Properties, Vector: object.Vectors[t.Store.VectorName]}, nil
}

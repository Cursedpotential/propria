package humancontext

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
)

// SearchObject is a separately typed human-native search projection, without PostgreSQL row coordinates.
// Original locators remain in provenance; its ID is a deterministic index coordinate, not a native source ID.
type SearchObject struct {
	ID         string         `json:"id"`
	Properties map[string]any `json:"properties"`
	Vector     []float32      `json:"vector"`
}

// SearchTarget publishes and independently reads exact human search projections in one existing collection.
// Implementations must not create a replacement corpus or fabricate legacy normalized-generation properties.
type SearchTarget interface {
	Put(context.Context, SearchObject) error
	Get(context.Context, string) (SearchObject, error)
}

// Embedder is the existing text embedding boundary; no model call is hidden in framing or extraction.
type Embedder interface {
	Embed(context.Context, []string) ([][]float32, error)
}

// Indexer owns separate native-human publication and independent exact readback units.
// Inputs: immutable bundles, existing embedder and configured existing collection target.
// Outputs: distinct publication/verification receipts; effects: idempotent search writes and readback files.
type Indexer struct {
	Bundles  *Bundles
	Embedder Embedder
	Target   SearchTarget
}

// Publish embeds complete native records in bounded batches and records every exact expected object externally.
// Inputs: extraction manifest and source registration refs. Outputs: publication receipt only.
// Effects: idempotent search upserts and derivative expected-object JSONL; verification is a separate Activity.
func (i *Indexer) Publish(ctx context.Context, req Request) (Result, error) {
	if req.ChunkManifestRef != "" && req.ChunkManifestRef != req.ManifestRef {
		return Result{}, errors.New("native chunk plan differs")
	}
	if i.Bundles == nil || i.Embedder == nil || i.Target == nil {
		return Result{}, errors.New("native index dependencies unavailable")
	}

	m, err := i.Bundles.ReadManifest(req.ManifestRef)
	if err != nil {
		return Result{}, err
	}
	if err := validateStageManifest(req, m, "chunked"); err != nil {
		return Result{}, err
	}
	if r, found, err := i.Bundles.cachedStage(ctx, req, "published", m.Source); err != nil || found {
		return r, err
	}
	dir, err := i.Bundles.begin(req, "published")
	if err != nil {
		return Result{}, err
	}
	path, err := i.Bundles.path(m.ChunksRef)
	if err != nil {
		return Result{}, err
	}
	input, err := os.Open(path)
	if err != nil {
		return Result{}, err
	}
	defer input.Close()
	pubPath := filepath.Join(dir, "objects.jsonl")
	output, err := os.OpenFile(pubPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0640)
	if err != nil {
		return Result{}, err
	}
	defer output.Close()
	encoder := json.NewEncoder(output)
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 65536), MaxMetadataBytes+1)
	batch := make([]SearchObject, 0, 16)
	published := int64(0)
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		texts := make([]string, len(batch))
		for j := range batch {
			texts[j] = batch[j].Properties["text"].(string)
		}
		vectors, err := i.Embedder.Embed(ctx, texts)
		if err != nil {
			return err
		}
		if len(vectors) != len(batch) {
			return errors.New("native embedding count differs")
		}
		for j := range batch {
			if len(vectors[j]) == 0 {
				return errors.New("native embedding missing")
			}
			batch[j].Vector = vectors[j]
			if err := i.Target.Put(ctx, batch[j]); err != nil {
				return err
			}
			if err := encoder.Encode(batch[j]); err != nil {
				return err
			}
			published++
		}
		batch = batch[:0]
		return nil
	}
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		var chunk NativeChunk
		if err := json.Unmarshal(scanner.Bytes(), &chunk); err != nil {
			return Result{}, err
		}
		object, err := nativeChunkObject(m.Source, req.ManifestRef, chunk)
		if err != nil {
			return Result{}, err
		}
		if model, ok := i.Embedder.(interface{ ModelName() string }); ok {
			object.Properties["embed_model"] = model.ModelName()
		}
		batch = append(batch, object)
		if len(batch) == 16 {
			if err := flush(); err != nil {
				return Result{}, err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return Result{}, err
	}
	if err := flush(); err != nil {
		return Result{}, err
	}
	if err := output.Sync(); err != nil {
		return Result{}, err
	}
	if published != m.Chunks {
		return Result{}, errors.New("native publication accounting differs")
	}
	m.Stage = "published"
	m.PublicationRef = fileRef(pubPath)
	m.Published = published
	m.IndexKind = IndexKind
	return i.Bundles.save(ctx, req, dir, m)
}

// Verify independently GETs every expected projection and compares all properties and its exact named vector.
// Inputs: publication manifest. Outputs: distinct readback receipt; effects: search reads and verification metadata.
// Pick after Publish even when the target acknowledged every write; counts alone are insufficient.
func (i *Indexer) Verify(ctx context.Context, req Request) (Result, error) {
	if i.Bundles == nil || i.Target == nil {
		return Result{}, errors.New("native readback dependencies unavailable")
	}

	m, err := i.Bundles.ReadManifest(req.ManifestRef)
	if err != nil {
		return Result{}, err
	}
	if err := validateStageManifest(req, m, "published"); err != nil {
		return Result{}, err
	}
	if r, found, err := i.Bundles.cachedStage(ctx, req, "verified", m.Source); err != nil || found {
		return r, err
	}
	p, err := i.Bundles.path(m.PublicationRef)
	if err != nil {
		return Result{}, err
	}
	file, err := os.Open(p)
	if err != nil {
		return Result{}, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 65536), MaxMetadataBytes+256*1024)
	verified := int64(0)
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		var expected SearchObject
		if err := json.Unmarshal(scanner.Bytes(), &expected); err != nil {
			return Result{}, err
		}
		actual, err := i.Target.Get(ctx, expected.ID)
		if err != nil {
			return Result{}, err
		}
		left, _ := json.Marshal(expected.Properties)
		right, _ := json.Marshal(actual.Properties)
		if expected.ID != actual.ID || string(left) != string(right) || !reflect.DeepEqual(expected.Vector, actual.Vector) {
			return Result{}, errors.New("native search readback differs")
		}
		verified++
	}
	if err := scanner.Err(); err != nil {
		return Result{}, err
	}
	if verified != m.Published {
		return Result{}, errors.New("native verification accounting differs")
	}
	dir, err := i.Bundles.begin(req, "verified")
	if err != nil {
		return Result{}, err
	}
	m.Stage = "verified"
	m.Verified = verified
	return i.Bundles.save(ctx, req, dir, m)
}

// Worker wiring for publish_context_search_activity, the Weaviate-first stage
// (activities/publish_context_search.go).
//
// The stage is scheduled on every new Proffer run, so its configuration is a
// startup contract: a worker that cannot reach the search surface must not
// poll the queue and fail every run at that stage. Collection names have no
// default (owner 2026-09-24: "from now on i approve collection names"). The
// deployed values are the owner-approved MsgEvents20260918 (OD-06,
// 2026-10-01), AiChatEvents20260918 (2026-09-24) and DocEvents20261001
// (2026-10-02, the one collection this worker may create).
//
// Byline: Claude Code · Opus 5.5 · 2026-10-01
// Byline: Claude Code · Opus 5.5 · 2026-10-02 (one collection per record kind)
package profferworker

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/contextsearch"
	"github.com/Cursedpotential/probata/engine/embedding"
	platformpostgres "github.com/Cursedpotential/probata/engine/postgres"
	platformtemporal "github.com/Cursedpotential/probata/engine/temporal"
	"github.com/Cursedpotential/probata/engine/weaviate"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ContextSearchConfig is the Weaviate targets plus the embedder. The API key
// stays in memory and is never logged.
type ContextSearchConfig struct {
	WeaviateURL string
	// Collections maps record kind -> collection.
	Collections map[string]string
	// Creatable names the collections the worker may create when absent.
	Creatable       map[string]bool
	VectorName      string
	EmbedBaseURL    string
	EmbedModel      string
	EmbedAPIKeyFile string
	EmbedAPIKey     string
}

const defaultContextSearchVectorName = "text_nim"

// contextSearchCollectionEnv names the variable that configures each kind.
var contextSearchCollectionEnv = map[string]string{
	contextsearch.RecordKindMessage:  "CONTEXT_SEARCH_MESSAGE_COLLECTION",
	contextsearch.RecordKindCall:     "CONTEXT_SEARCH_MESSAGE_COLLECTION",
	contextsearch.RecordKindAIChat:   "CONTEXT_SEARCH_AI_CHAT_COLLECTION",
	contextsearch.RecordKindDocument: "CONTEXT_SEARCH_DOCUMENT_COLLECTION",
}

// loadContextSearchConfig reads CONTEXT_SEARCH_*. Every value the stage cannot
// run without is required.
func loadContextSearchConfig() (ContextSearchConfig, []string) {
	var problems []string
	cfg := ContextSearchConfig{
		WeaviateURL:     strings.TrimRight(firstEnvironment("CONTEXT_SEARCH_WEAVIATE_URL"), "/"),
		Collections:     map[string]string{},
		Creatable:       map[string]bool{},
		VectorName:      firstEnvironment("CONTEXT_SEARCH_VECTOR_NAME"),
		EmbedBaseURL:    firstEnvironment("CONTEXT_SEARCH_EMBED_BASE_URL"),
		EmbedModel:      firstEnvironment("CONTEXT_SEARCH_EMBED_MODEL"),
		EmbedAPIKeyFile: firstEnvironment("CONTEXT_SEARCH_EMBED_API_KEY_FILE"),
	}
	if cfg.VectorName == "" {
		cfg.VectorName = defaultContextSearchVectorName
	}
	if parsed, err := url.Parse(cfg.WeaviateURL); cfg.WeaviateURL == "" || err != nil || parsed.Scheme == "" || parsed.Host == "" {
		problems = append(problems, "CONTEXT_SEARCH_WEAVIATE_URL must be an absolute http(s) URL")
	}
	reported := map[string]bool{}
	for _, kind := range activities.ContextSearchRecordKinds {
		name := contextSearchCollectionEnv[kind]
		value := firstEnvironment(name)
		if value == "" {
			if !reported[name] {
				problems = append(problems, name+" is required and has no default (collection names are owner-approved)")
				reported[name] = true
			}
			continue
		}
		cfg.Collections[kind] = value
	}
	for _, name := range strings.Split(firstEnvironment("CONTEXT_SEARCH_CREATABLE_COLLECTIONS"), ",") {
		if name = strings.TrimSpace(name); name != "" {
			cfg.Creatable[name] = true
		}
	}
	for name := range cfg.Creatable {
		routed := false
		for _, collection := range cfg.Collections {
			routed = routed || collection == name
		}
		if !routed {
			problems = append(problems, fmt.Sprintf("CONTEXT_SEARCH_CREATABLE_COLLECTIONS names %s, which no record kind routes to", name))
		}
	}
	if cfg.EmbedAPIKeyFile == "" || !absoluteRuntimePath(cfg.EmbedAPIKeyFile) {
		problems = append(problems, "CONTEXT_SEARCH_EMBED_API_KEY_FILE must be an absolute path")
	} else if value, err := platformtemporal.ReadRuntimeSecretFile(cfg.EmbedAPIKeyFile, 4096); err != nil || strings.TrimSpace(value) == "" {
		problems = append(problems, "CONTEXT_SEARCH_EMBED_API_KEY_FILE is unavailable or empty")
	} else {
		cfg.EmbedAPIKey = strings.TrimSpace(value)
	}
	return cfg, problems
}

// contextSearchTarget routes each collection to its own weaviate.Store.
type contextSearchTarget struct{ stores map[string]weaviate.Store }

func (t contextSearchTarget) store(collection string) (weaviate.Store, error) {
	store, ok := t.stores[collection]
	if !ok {
		return weaviate.Store{}, fmt.Errorf("collection %s is not configured on this worker", collection)
	}
	return store, nil
}

func (t contextSearchTarget) EnsureCollection(ctx context.Context, collection string) ([]string, error) {
	store, err := t.store(collection)
	if err != nil {
		return nil, err
	}
	return store.EnsureCollection(ctx)
}

func (t contextSearchTarget) PublishObjects(ctx context.Context, collection string, objects []contextsearch.Object) (activities.ContextSearchPublishResult, error) {
	store, err := t.store(collection)
	if err != nil {
		return activities.ContextSearchPublishResult{}, err
	}
	outcome, err := store.PublishObjects(ctx, objects)
	return activities.ContextSearchPublishResult{Requested: outcome.Requested, Written: outcome.Written, ObjectIDs: outcome.ObjectIDs}, err
}

// contextSearchDescriptions describe a collection this worker creates.
var contextSearchDescriptions = map[string]string{
	contextsearch.RecordKindDocument: "Documents; written by Probata's Weaviate-first stage before approval (owner-approved name 2026-10-02)",
}

func buildContextSearch(pool *pgxpool.Pool, cfg ContextSearchConfig) (activities.PublishContextSearchActivities, error) {
	source, err := platformpostgres.NewContextSearchStore(pool)
	if err != nil {
		return activities.PublishContextSearchActivities{}, err
	}
	embedder := embedding.NIM{BaseURL: cfg.EmbedBaseURL, Model: cfg.EmbedModel, APIKey: cfg.EmbedAPIKey}
	stores := map[string]weaviate.Store{}
	for kind, collection := range cfg.Collections {
		if _, done := stores[collection]; done {
			continue
		}
		stores[collection] = weaviate.Store{
			BaseURL: cfg.WeaviateURL, Collection: collection, VectorName: cfg.VectorName, EmbedModel: embedder.ModelName(),
			CreateIfAbsent: cfg.Creatable[collection], Description: contextSearchDescriptions[kind],
		}
	}
	activity := activities.NewPublishContextSearchActivities(source, embedder, contextSearchTarget{stores: stores}, cfg.Collections)
	return activity, nil
}

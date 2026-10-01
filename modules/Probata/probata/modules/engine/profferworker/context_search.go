// Worker wiring for publish_context_search_activity, the Weaviate-first stage
// (activities/publish_context_search.go).
//
// The stage is scheduled on every new Proffer run, so its configuration is a
// startup contract: a worker that cannot reach the search surface must not
// poll the queue and fail every run at that stage. The collection name has no
// default (owner 2026-09-24: "from now on i approve collection names"); the
// deployed value is the owner-approved MsgEvents20260918 (OD-06, 2026-10-01).
//
// Byline: Claude Code · Opus 5.5 · 2026-10-01
package profferworker

import (
	"context"
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

// ContextSearchConfig is the Weaviate target plus the embedder. The API key
// stays in memory and is never logged.
type ContextSearchConfig struct {
	WeaviateURL     string
	Collection      string
	VectorName      string
	EmbedBaseURL    string
	EmbedModel      string
	EmbedAPIKeyFile string
	EmbedAPIKey     string
}

const defaultContextSearchVectorName = "text_nim"

// loadContextSearchConfig reads CONTEXT_SEARCH_*. Every value the stage cannot
// run without is required.
func loadContextSearchConfig() (ContextSearchConfig, []string) {
	var problems []string
	cfg := ContextSearchConfig{
		WeaviateURL:     strings.TrimRight(firstEnvironment("CONTEXT_SEARCH_WEAVIATE_URL"), "/"),
		Collection:      firstEnvironment("CONTEXT_SEARCH_COLLECTION"),
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
	if cfg.Collection == "" {
		problems = append(problems, "CONTEXT_SEARCH_COLLECTION is required and has no default (collection names are owner-approved)")
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

// contextSearchTarget adapts weaviate.Store to the Activity's target seam.
type contextSearchTarget struct{ store weaviate.Store }

func (t contextSearchTarget) EnsureCollection(ctx context.Context) ([]string, error) {
	return t.store.EnsureCollection(ctx)
}

func (t contextSearchTarget) PublishObjects(ctx context.Context, objects []contextsearch.Object) (activities.ContextSearchPublishResult, error) {
	outcome, err := t.store.PublishObjects(ctx, objects)
	return activities.ContextSearchPublishResult{Requested: outcome.Requested, Written: outcome.Written, ObjectIDs: outcome.ObjectIDs}, err
}

func buildContextSearch(pool *pgxpool.Pool, cfg ContextSearchConfig) (activities.PublishContextSearchActivities, error) {
	source, err := platformpostgres.NewContextSearchStore(pool)
	if err != nil {
		return activities.PublishContextSearchActivities{}, err
	}
	embedder := embedding.NIM{BaseURL: cfg.EmbedBaseURL, Model: cfg.EmbedModel, APIKey: cfg.EmbedAPIKey}
	target := contextSearchTarget{store: weaviate.Store{
		BaseURL: cfg.WeaviateURL, Collection: cfg.Collection, VectorName: cfg.VectorName, EmbedModel: embedder.ModelName(),
	}}
	return activities.NewPublishContextSearchActivities(source, embedder, target, cfg.Collection), nil
}

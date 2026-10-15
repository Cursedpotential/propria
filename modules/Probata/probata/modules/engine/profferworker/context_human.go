package profferworker

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/Cursedpotential/probata/engine/acquisition"
	"github.com/Cursedpotential/probata/engine/embedding"
	"github.com/Cursedpotential/probata/engine/humancontext"
	"github.com/Cursedpotential/probata/engine/objectstores"
	platformpostgres "github.com/Cursedpotential/probata/engine/postgres"
	"github.com/Cursedpotential/probata/engine/weaviate"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jackc/pgx/v5/pgxpool"
)

// buildHumanContext composes native human bundles and separate chunk publication/readback using existing configured transports.
// Inputs: platform metadata/DuckDB pool, worker scratch root, object-store configuration and CONTEXT_CHUNKS_COLLECTION.
// Outputs: four independently registered Go Activities; absent collection configuration leaves indexing unavailable.
// Effects: reads existing credential configuration only; no source reads, database queries, schema creation or model calls.
// Choose for registered native human sources rather than legacy normalized SQL, retained originals or AI packages.
func buildHumanContext(pool *pgxpool.Pool, cfg Config, stores objectstores.Stores) (*humancontext.Activities, error) {
	collection := strings.TrimSpace(os.Getenv("CONTEXT_CHUNKS_COLLECTION"))
	if collection != "" && collection != "ProfferChunks20261002" {
		return nil, errors.New("native human indexing requires the approved CONTEXT_CHUNKS_COLLECTION")
	}
	configured := stores
	if len(configured) == 0 {
		configured = objectstores.Stores{}
		for scheme, file := range map[string]string{"r2": os.Getenv("CASEBIBLE_R2_CONFIG_PATH"), "b2": os.Getenv("B2_CONFIG_PATH")} {
			if file != "" {
				configured[scheme] = file
			}
		}
	}
	clients := make(map[string]*s3.Client, len(configured))
	for _, scheme := range configured.Schemes() {
		storeConfig, err := acquisition.LoadObjectStorageConfigFile(configured[scheme])
		if err != nil {
			return nil, errors.New("native human object-store configuration unavailable")
		}
		client, err := acquisition.NewS3Client(storeConfig)
		if err != nil {
			return nil, errors.New("native human object-store client unavailable")
		}
		clients[scheme] = client
	}
	bundles := &humancontext.Bundles{
		Root:      filepath.Join(cfg.DeriveScratchDir, "human-context"),
		Opener:    &acquisition.HumanObjectSource{Clients: clients},
		Extractor: &platformpostgres.HumanSourceDuckDB{Pool: pool},
		Metadata:  &platformpostgres.HumanSourceMetadata{DB: pool},
	}
	embedder := embedding.NIM{BaseURL: cfg.ContextSearch.EmbedBaseURL, Model: cfg.ContextSearch.EmbedModel, APIKey: cfg.ContextSearch.EmbedAPIKey, StrictInput: true}
	indexer := &humancontext.Indexer{Bundles: bundles, Embedder: embedder}
	if collection != "" {
		indexer.Target = &weaviate.HumanSourceTarget{Store: weaviate.Store{
			BaseURL: cfg.ContextSearch.WeaviateURL, Collection: collection,
			VectorName: cfg.ContextSearch.VectorName, EmbedModel: embedder.ModelName(),
			CreateIfAbsent: false,
		}}
	}
	return &humancontext.Activities{Bundles: bundles, Indexer: indexer}, nil
}

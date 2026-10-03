// Byline: Claude Code · Sonnet 5.5 · 2026-10-02

package cfjobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// Config is every environment setting the cf-jobs worker needs. Secrets are read from files, never from the environment.
type Config struct {
	TemporalHostPort  string
	TemporalNamespace string
	TaskQueue         string
	// CatalogDSNFile holds the PostgreSQL connection URL of the Case Bible database (`casebible`).
	CatalogDSNFile string
	Sniffer        WorkerEndpoint
	ZipLister      WorkerEndpoint
	B2Hasher       WorkerEndpoint
}

// WorkerEndpoint is a Worker's URL and the file holding its bearer token.
type WorkerEndpoint struct {
	URL       string
	TokenFile string
}

// LoadConfig reads the worker's environment and reports every missing variable at once.
//
// Variables: TEMPORAL_HOST_PORT, TEMPORAL_NAMESPACE, CF_TASK_QUEUE (optional, default casebible-cf-v1), CASEBIBLE_DATABASE_URL_FILE,
// and for each Worker CF_SNIFFER_URL / CF_SNIFFER_TOKEN_FILE, CF_ZIP_LISTER_URL / CF_ZIP_LISTER_TOKEN_FILE,
// CF_B2_HASHER_URL / CF_B2_HASHER_TOKEN_FILE. It fails closed: a half-configured worker does not start.
func LoadConfig() (Config, error) {
	var problems []string
	require := func(name string) string {
		value := strings.TrimSpace(os.Getenv(name))
		if value == "" {
			problems = append(problems, name+" is required")
		}
		return value
	}
	cfg := Config{
		TemporalHostPort:  require("TEMPORAL_HOST_PORT"),
		TemporalNamespace: require("TEMPORAL_NAMESPACE"),
		TaskQueue:         strings.TrimSpace(os.Getenv("CF_TASK_QUEUE")),
		CatalogDSNFile:    require("CASEBIBLE_DATABASE_URL_FILE"),
		Sniffer:           WorkerEndpoint{URL: require("CF_SNIFFER_URL"), TokenFile: require("CF_SNIFFER_TOKEN_FILE")},
		ZipLister:         WorkerEndpoint{URL: require("CF_ZIP_LISTER_URL"), TokenFile: require("CF_ZIP_LISTER_TOKEN_FILE")},
		B2Hasher:          WorkerEndpoint{URL: require("CF_B2_HASHER_URL"), TokenFile: require("CF_B2_HASHER_TOKEN_FILE")},
	}
	if cfg.TaskQueue == "" {
		cfg.TaskQueue = QueueName
	}
	if len(problems) > 0 {
		return Config{}, errors.New("cf-jobs worker configuration invalid: " + strings.Join(problems, "; "))
	}
	return cfg, nil
}

func readSecret(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read secret file %s: %w", path, err)
	}
	value := strings.TrimSpace(string(raw))
	if value == "" {
		return "", fmt.Errorf("secret file %s is empty", path)
	}
	return value, nil
}

func (e WorkerEndpoint) client() (*WorkerClient, error) {
	token, err := readSecret(e.TokenFile)
	if err != nil {
		return nil, err
	}
	return &WorkerClient{BaseURL: e.URL, Token: token, HTTP: &http.Client{}}, nil
}

// Run connects to the catalog and Temporal and serves the cf-jobs worker until the context ends.
//
// Side effects: opens a PostgreSQL pool and a Temporal client, and polls Config.TaskQueue. It writes nothing until a workflow
// is started against it.
func Run(ctx context.Context, cfg Config) error {
	dsn, err := readSecret(cfg.CatalogDSNFile)
	if err != nil {
		return err
	}
	sniffer, err := cfg.Sniffer.client()
	if err != nil {
		return err
	}
	zipLister, err := cfg.ZipLister.client()
	if err != nil {
		return err
	}
	hasher, err := cfg.B2Hasher.client()
	if err != nil {
		return err
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return fmt.Errorf("open catalog pool: %w", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping catalog: %w", err)
	}
	temporalClient, err := client.Dial(client.Options{HostPort: cfg.TemporalHostPort, Namespace: cfg.TemporalNamespace})
	if err != nil {
		return fmt.Errorf("dial Temporal: %w", err)
	}
	defer temporalClient.Close()
	w := worker.New(temporalClient, cfg.TaskQueue, worker.Options{
		MaxConcurrentActivityExecutionSize: 8,
		WorkerStopTimeout:                  30 * time.Second,
	})
	Register(w, &Activities{Catalog: &PGCatalog{Pool: pool}, Sniffer: sniffer, ZipLister: zipLister, B2Hasher: hasher})
	slog.Info("cf-jobs worker started", "queue", cfg.TaskQueue)
	return w.Run(worker.InterruptCh())
}

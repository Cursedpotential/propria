// Byline: Codex · GPT-6 · 2026-10-11
// Package discoveryconfig admits explicit optional analysis configuration without affecting intake.
package discoveryconfig

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Cursedpotential/probata/engine/discovery"
	platformtemporal "github.com/Cursedpotential/probata/engine/temporal"
)

// Config holds server-selected datasets and existing shared derivative storage.
// Inputs: explicit runtime settings. Outputs: bounded configuration; no secret is logged.
// Effects: none until constructed. Choose independently of source or operating-mode flags.
type Config struct {
	Enabled     bool
	Root        string
	Datasets    []discovery.Dataset
	ModelConfig json.RawMessage
	Surreal     discovery.SurrealConfig
}

type datasetConfiguration struct {
	discovery.Dataset
	SurrealTable string `json:"surreal_table,omitempty"`
}

// FromEnvironment validates discovery only when its independent feature flag is enabled.
// Inputs: ADAPTIVE_DISCOVERY_* variables and optional mounted Surreal credential.
// Outputs: bounded configuration or a setup error. Effects: reads runtime settings/credential.
// Choose at existing starter/worker composition; disabled discovery reads no model or source.
func FromEnvironment() (Config, error) {
	var cfg Config
	flag := strings.TrimSpace(os.Getenv("ADAPTIVE_DISCOVERY_ENABLED"))
	if flag == "" {
		return cfg, nil
	}
	enabled, err := strconv.ParseBool(flag)
	if err != nil {
		return cfg, errors.New("ADAPTIVE_DISCOVERY_ENABLED must be a boolean")
	}
	cfg.Enabled = enabled
	if !enabled {
		return cfg, nil
	}
	cfg.Root = strings.TrimSpace(os.Getenv("ADAPTIVE_DISCOVERY_ROOT"))
	if cfg.Root == "" || !filepath.IsAbs(cfg.Root) {
		return cfg, errors.New("ADAPTIVE_DISCOVERY_ROOT requires an absolute shared derivative directory")
	}
	raw := os.Getenv("ADAPTIVE_DISCOVERY_DATASETS_JSON")
	if len(raw) == 0 || len(raw) > 128<<10 {
		return cfg, errors.New("ADAPTIVE_DISCOVERY_DATASETS_JSON requires bounded server-selected datasets")
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	var selected []datasetConfiguration
	if err := decoder.Decode(&selected); err != nil {
		return cfg, errors.New("discovery dataset configuration is invalid")
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return cfg, errors.New("discovery dataset configuration contains trailing JSON")
	}
	cfg.Surreal.Tables = map[string]string{}
	for _, item := range selected {
		item.Dataset.SurrealTable = item.SurrealTable
		cfg.Datasets = append(cfg.Datasets, item.Dataset)
		if item.InputBackend == "surrealdb" {
			cfg.Surreal.Tables[item.DatasetID] = item.SurrealTable
		}
	}
	if len(cfg.Datasets) == 0 || discovery.ValidateDatasets(cfg.Datasets) != nil {
		return cfg, errors.New("discovery dataset definitions are not admitted")
	}
	model := strings.TrimSpace(os.Getenv("ADAPTIVE_DISCOVERY_MODEL_CONFIG_JSON"))
	if model == "" {
		return cfg, errors.New("ADAPTIVE_DISCOVERY_MODEL_CONFIG_JSON requires a pinned local CPU model")
	}
	if len(model) > 16<<10 || !json.Valid([]byte(model)) {
		return cfg, errors.New("discovery model configuration is invalid")
	}
	var shape map[string]json.RawMessage
	if err := json.Unmarshal([]byte(model), &shape); err != nil || shape == nil {
		return cfg, errors.New("discovery model configuration requires an object")
	}
	cfg.ModelConfig = append(json.RawMessage(nil), model...)
	if len(cfg.Surreal.Tables) > 0 {
		cfg.Surreal.URL = strings.TrimSpace(os.Getenv("ADAPTIVE_DISCOVERY_SURREAL_URL"))
		cfg.Surreal.Namespace = strings.TrimSpace(os.Getenv("ADAPTIVE_DISCOVERY_SURREAL_NAMESPACE"))
		cfg.Surreal.Database = strings.TrimSpace(os.Getenv("ADAPTIVE_DISCOVERY_SURREAL_DATABASE"))
		tokenPath := strings.TrimSpace(os.Getenv("ADAPTIVE_DISCOVERY_SURREAL_TOKEN_FILE"))
		if tokenPath == "" {
			return cfg, errors.New("selected Surreal discovery input requires its mounted token")
		}
		token, err := platformtemporal.ReadRuntimeSecretFile(tokenPath, 4096)
		if err != nil || strings.TrimSpace(token) == "" {
			return cfg, errors.New("selected Surreal discovery token is unavailable")
		}
		cfg.Surreal.Token = token
		if _, err := discovery.NewSurrealHTTPReader(cfg.Surreal); err != nil {
			return cfg, errors.New("selected Surreal discovery connection is invalid")
		}
	}
	return cfg, nil
}

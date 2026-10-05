// Byline: Codex · GPT-6.1 · 2026-10-04.
package libraryvalidation

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Cursedpotential/probata/engine/extraction/model"
	"github.com/Cursedpotential/probata/engine/surrealsink"
)

const (
	EnvCaseMCPURL      = "TOOLKIT_VALIDATION_CASE_MCP_URL"
	EnvCaseTokenFile   = "TOOLKIT_VALIDATION_CASE_TOKEN_FILE"
	EnvSigningKeyFile  = "TOOLKIT_VALIDATION_SIGNING_KEY_FILE"
	EnvCurrencyKeyFile = "TOOLKIT_VALIDATION_CURRENCY_KEY_FILE"
	EnvDBUserFile      = "TOOLKIT_VALIDATION_DB_USER_FILE"
	EnvDBPasswordFile  = "TOOLKIT_VALIDATION_DB_PASSWORD_FILE"
	EnvB2ConfigFile    = "TOOLKIT_VALIDATION_B2_CONFIG_FILE"
	EnvB2Bucket        = "TOOLKIT_VALIDATION_B2_BUCKET"
	EnvB2Prefix        = "TOOLKIT_VALIDATION_B2_PREFIX"
	EnvPython          = "TOOLKIT_VALIDATION_PYTHON"
	EnvProjectRoot     = "TOOLKIT_VALIDATION_PROJECT_ROOT"
	EnvScratchRoot     = "TOOLKIT_VALIDATION_SCRATCH_ROOT"
	EnvBridgeFile      = "TOOLKIT_VALIDATION_BRIDGE_FILE"
)

// ReadCredentialFile reads a bounded literal value or explicitly selected env assignment without evaluating or printing it.
// Inputs: absolute regular file and accepted key names. Outputs: secret value. Effects: file read only.
// Choose for mounted server credentials; never source a secret file or expose its contents in an error.
func ReadCredentialFile(file string, names ...string) (string, error) {
	if !filepath.IsAbs(file) {
		return "", errors.New("credential file must be an absolute mounted path")
	}
	info, err := os.Lstat(file)
	if err != nil || !info.Mode().IsRegular() || info.Size() == 0 || info.Size() > 16384 {
		return "", errors.New("credential file is absent, unsafe or exceeds budget")
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		return "", errors.New("credential file cannot be read")
	}
	value := strings.TrimSpace(string(raw))
	if regexp.MustCompile(`(?m)^\s*[A-Z_][A-Z0-9_]*\s*=`).MatchString(value) {
		found := false
		for _, name := range names {
			pattern := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(name) + `\s*=\s*([^\r\n]+)\s*$`)
			matches := pattern.FindAllStringSubmatch(value, -1)
			if len(matches) > 1 {
				return "", errors.New("credential assignment is ambiguous")
			}
			if len(matches) == 1 {
				if found {
					return "", errors.New("credential file contains multiple accepted keys")
				}
				value = strings.Trim(strings.TrimSpace(matches[0][1]), `"'`)
				found = true
			}
		}
		if !found {
			return "", errors.New("credential file does not contain the expected literal key")
		}
	}
	if value == "" || strings.ContainsAny(value, "\r\n\x00") {
		return "", errors.New("credential file must hold a single literal value")
	}
	return value, nil
}

// NewServiceFromEnv builds the validation dependencies from existing NIM/B2/shared-case contracts and mounted server files.
// Inputs: parent-constructed existing versioned B2 store, named *_FILE paths and extractor configuration.
// Outputs: ready Service or visible configuration error; parent loads EnvB2ConfigFile with acquisition.LoadObjectStorageConfigFile.
// Effects: credential reads and bounded existing parser import preflight; no network calls, DB writes or deployment.
// Choose at parent-owned worker boot and register the four Activities plus RegisterWorkflow separately.
func NewServiceFromEnv(store VersionStore) (*Service, error) {
	if validateB2ConfigFile(os.Getenv(EnvB2ConfigFile)) != nil {
		return nil, errors.New(EnvB2ConfigFile + " must configure current Backblaze B2 storage")
	}
	read := func(env string, names ...string) (string, error) {
		value, err := ReadCredentialFile(strings.TrimSpace(os.Getenv(env)), names...)
		if err != nil {
			return "", errors.New(env + " is missing or invalid")
		}
		return value, nil
	}
	token, err := read(EnvCaseTokenFile, "MCP_BEARER_TOKEN", "TOOLKIT_VALIDATION_CASE_TOKEN")
	if err != nil {
		return nil, err
	}
	signing, err := read(EnvSigningKeyFile, "TOOLKIT_VALIDATION_SIGNING_KEY")
	if err != nil || len(signing) < 32 {
		return nil, errors.New(EnvSigningKeyFile + " needs a key of at least 32 bytes")
	}
	var currency string
	if strings.TrimSpace(os.Getenv(EnvCurrencyKeyFile)) != "" {
		currency, err = read(EnvCurrencyKeyFile, "TOOLKIT_VALIDATION_CURRENCY_KEY")
		if err != nil || len(currency) < 32 {
			return nil, errors.New(EnvCurrencyKeyFile + " is invalid")
		}
	}
	if currency != "" && currency == signing {
		return nil, errors.New("validation and currency signing keys must be independent")
	}
	user, err := read(EnvDBUserFile, "TOOLKIT_VALIDATION_DB_USER", "SURREAL_CASE_USER", "USER", "USERNAME")
	if err != nil {
		return nil, err
	}
	password, err := read(EnvDBPasswordFile, "TOOLKIT_VALIDATION_DB_PASSWORD", "SURREAL_CASE_PASSWORD", "PASSWORD")
	if err != nil {
		return nil, err
	}
	db := surrealsink.Config{URL: strings.TrimRight(os.Getenv(surrealsink.EnvURL), "/"), Namespace: envDefault(surrealsink.EnvNamespace, surrealsink.DefaultNamespace), Database: envDefault(surrealsink.EnvDatabase, surrealsink.DefaultDatabase), AuthLevel: envDefault(surrealsink.EnvAuthLevel, "database"), User: user, Password: password}
	if err = db.Validate(); err != nil {
		return nil, errors.New("existing surreal-case connection is invalid")
	}
	endpoint := strings.TrimSpace(os.Getenv(EnvCaseMCPURL))
	if endpoint == "" {
		return nil, errors.New(EnvCaseMCPURL + " is required")
	}
	artifacts := B2Artifacts{Store: store, Bucket: strings.TrimSpace(os.Getenv(EnvB2Bucket)), Prefix: strings.TrimSpace(os.Getenv(EnvB2Prefix))}
	if err = artifacts.Validate(); err != nil {
		return nil, err
	}
	modelConfig, err := model.ConfigFromEnv()
	if err != nil {
		return nil, errors.New("existing ENTITY_MODEL_* NIM configuration is missing or invalid")
	}
	if modelConfig.ModelID != model.DefaultModelID || strings.TrimRight(modelConfig.BaseURL, "/") != model.DefaultBaseURL {
		return nil, errors.New("library validation requires the owner's existing NIM kimi-k3 configuration")
	}
	modelClient, err := model.NewClient(modelConfig)
	if err != nil {
		return nil, errors.New("existing NIM model client is invalid")
	}
	rejectRedirect := func(_ *http.Request, _ []*http.Request) error {
		return errors.New("service credential redirect refused")
	}
	httpClient := &http.Client{Timeout: FetchTimeout, CheckRedirect: rejectRedirect}
	repo := HTTPRepository{Reader: MCPRecords{Endpoint: endpoint, Token: token, HTTP: httpClient}, DB: SQLClient{Config: db, HTTP: httpClient}, SigningKey: []byte(signing)}
	extractor := ProcessExtractor{Artifacts: artifacts, Python: strings.TrimSpace(os.Getenv(EnvPython)), ProjectRoot: strings.TrimSpace(os.Getenv(EnvProjectRoot)), ScratchRoot: strings.TrimSpace(os.Getenv(EnvScratchRoot)), BridgePath: strings.TrimSpace(os.Getenv(EnvBridgeFile))}
	for _, file := range []string{extractor.Python, extractor.ProjectRoot, extractor.ScratchRoot, extractor.BridgePath} {
		if !filepath.IsAbs(file) {
			return nil, errors.New("pinned extractor configuration requires absolute same-host paths")
		}
	}
	if _, err := extractor.ValidateRuntime(context.Background()); err != nil {
		return nil, err
	}
	return &Service{Repository: repo, Artifacts: artifacts, Fetcher: NewPrimaryFetcher(), Extractor: extractor, Verifier: ClaimVerifier{Model: modelClient, CurrencyKey: []byte(currency)}, SigningKey: []byte(signing)}, nil
}

// envDefault chooses one existing shared setting; inputs: env/fallback; outputs: value; effects: env read only.
func envDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

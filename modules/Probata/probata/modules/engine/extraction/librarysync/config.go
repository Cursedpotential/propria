// Byline: Codex · GPT-6.1 · 2026-10-04.
package librarysync

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/Cursedpotential/probata/engine/extraction/libraryvalidation"
)

const (
	EnvBackendURL   = "TOOLKIT_LIBRARY_SYNC_BACKEND_URL"
	EnvTokenFile    = "TOOLKIT_LIBRARY_SYNC_TOKEN_FILE"
	EnvB2ConfigFile = "TOOLKIT_LIBRARY_SYNC_B2_CONFIG_FILE"
	EnvB2Bucket     = "TOOLKIT_LIBRARY_SYNC_B2_BUCKET"
	EnvAccountScope = "TOOLKIT_LIBRARY_SYNC_ACCOUNT_SCOPE"
	EnvLegalRoot    = "TOOLKIT_LIBRARY_SYNC_LEGAL_ROOT"
)

// NewServiceFromEnv composes sync using current B2 config and existing validator artifact/extractor dependencies.
// Inputs: parent-owned artifacts, pinned extractor, server signing key, explicit *_FILE/origin/scope configuration.
// Outputs: ready service or visible configuration error. Effects: bounded mounted-file reads only; no DB, deploy or new runtime.
// Choose at existing worker composition; register workflow/Activities separately. Approval/currency reviewer keys are never required.
func NewServiceFromEnv(artifacts libraryvalidation.Artifacts, extractor libraryvalidation.Extractor, signingKey []byte) (*Service, error) {
	deps, err := readStorageBackendFromEnv()
	if err != nil {
		return nil, err
	}
	return NewService(deps.Scope, deps.Storage, deps.Backend, artifacts, extractor, signingKey)
}

// NewOriginalReaderFromEnv composes the starter's original-download reader from the same approved B2/service mounts.
// Inputs: six TOOLKIT_LIBRARY_SYNC_* settings; outputs: read-only reader or visible configuration error.
// Effects: bounded credential-file reads only. Choose in the starter; no Python, artifacts, signing or currency key is required.
func NewOriginalReaderFromEnv() (*PinnedOriginalReader, error) {
	return readStorageBackendFromEnv()
}

func readStorageBackendFromEnv() (*PinnedOriginalReader, error) {
	token, err := libraryvalidation.ReadCredentialFile(os.Getenv(EnvTokenFile), "TOOLKIT_LIBRARY_SYNC_TOKEN")
	if err != nil {
		return nil, errors.New(EnvTokenFile + " missing or invalid")
	}
	backend, err := NewHTTPBackend(strings.TrimSpace(os.Getenv(EnvBackendURL)), token, nil)
	if err != nil {
		return nil, err
	}
	file := os.Getenv(EnvB2ConfigFile)
	if !filepath.IsAbs(file) {
		return nil, errors.New(EnvB2ConfigFile + " must name a mounted absolute file")
	}
	info, err := os.Lstat(file)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 16384 {
		return nil, errors.New("sync B2 mounted config invalid")
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		return nil, errors.New("sync B2 mounted config unavailable")
	}
	var cfg B2Config
	if json.Unmarshal(raw, &cfg) != nil {
		return nil, errors.New("sync B2 literal JSON config invalid")
	}
	scope := Scope{AccountScope: os.Getenv(EnvAccountScope), Bucket: os.Getenv(EnvB2Bucket), LegalRoot: os.Getenv(EnvLegalRoot)}
	store, err := NewB2Storage(cfg, scope)
	if err != nil {
		return nil, err
	}
	return NewOriginalReader(scope, store, backend)
}

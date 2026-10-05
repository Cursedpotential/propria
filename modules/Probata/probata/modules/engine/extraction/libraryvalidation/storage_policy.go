// Byline: Codex · GPT-6.1 · 2026-10-04; owner B2-only storage ruling, 21:36.
package libraryvalidation

import (
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// ValidateB2StorageEndpoint admits only the current Backblaze HTTPS endpoint before constructing a validator storage client.
// Inputs: parsed mounted endpoint; outputs: safe error. Effects: none; historical R2 configuration is never routed or rewritten.
func ValidateB2StorageEndpoint(endpoint string) error {
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "https" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || (u.Port() != "" && u.Port() != "443") || !strings.HasSuffix(strings.ToLower(u.Hostname()), ".backblazeb2.com") {
		return errors.New("library validation storage requires a Backblaze B2 HTTPS endpoint")
	}
	return nil
}

// validateB2ConfigFile checks the endpoint in the existing mounted JSON format without importing acquisition's worker dependencies.
// Inputs: absolute bounded regular config path; outputs: safe error. Effects: local read only; credentials are never output or evaluated.
func validateB2ConfigFile(file string) error {
	info, err := os.Lstat(file)
	if !filepath.IsAbs(file) || err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 64<<10 {
		return errors.New("B2 storage config is absent or unsafe")
	}
	raw, err := os.ReadFile(file)
	if err != nil || len(raw) > 64<<10 {
		return errors.New("B2 storage config unavailable")
	}
	var config struct {
		Endpoint string `json:"endpoint_url"`
	}
	if json.Unmarshal(raw, &config) != nil {
		return errors.New("B2 storage config invalid")
	}
	return ValidateB2StorageEndpoint(strings.TrimSpace(config.Endpoint))
}

// b2Reference checks the active provider identity without interpreting a version as permission to change providers.
// Inputs: artifact URI; outputs: B2 identity validity. Effects: none; exact version/path/hash checks remain in B2Artifacts.Read.
func b2Reference(ref string) bool {
	u, err := url.Parse(ref)
	return err == nil && u.Scheme == "b2" && u.Host != "" && u.Path != "" && u.User == nil && u.Fragment == ""
}

// activeR2Source detects retired provider coordinates in current source fields while preserving personal text and migration history.
// Inputs: unchanged shared source fields; outputs: blocked-provider indication. Effects: none; never rewrites records or falls back.
// Choose before official fetch or currency approval; only active coordinate fields/containers are interpreted as storage references.
func activeR2Source(fields map[string]json.RawMessage) bool {
	for key, raw := range fields {
		if !storageCoordinate(key) {
			continue
		}
		var value any
		if json.Unmarshal(raw, &value) == nil && retiredStorage(value, strings.ToLower(key)) {
			return true
		}
	}
	return false
}

// storageCoordinate identifies active source coordinates, excluding historical/audit records and free-text bodies.
// Inputs: field name; outputs: coordinate membership; effects: none.
func storageCoordinate(key string) bool {
	key = strings.ToLower(key)
	for _, prefix := range []string{"legacy_", "historical_", "migration_", "retired_"} {
		if strings.HasPrefix(key, prefix) {
			return false
		}
	}
	if strings.HasSuffix(key, "_ref") || strings.HasSuffix(key, "_uri") || strings.HasSuffix(key, "_url") {
		return true
	}
	switch key {
	case "url", "uri", "ref", "locator", "source_locator", "source", "storage", "object_storage", "artifact", "provider", "scheme", "store", "endpoint":
		return true
	}
	return false
}

// retiredStorage checks explicit R2 schemes/endpoints inside admitted coordinates and skips migration/history subtrees.
// Inputs: decoded coordinate value/name; outputs: retired provider use; effects: none.
func retiredStorage(value any, key string) bool {
	switch item := value.(type) {
	case string:
		text := strings.ToLower(strings.TrimSpace(item))
		if (key == "provider" || key == "scheme" || key == "store") && (text == "r2" || text == "cloudflare_r2") {
			return true
		}
		u, err := url.Parse(text)
		return err == nil && (u.Scheme == "r2" || u.Hostname() == "r2.cloudflarestorage.com" || strings.HasSuffix(u.Hostname(), ".r2.cloudflarestorage.com"))
	case map[string]any:
		for child, nested := range item {
			if storageCoordinate(child) && retiredStorage(nested, strings.ToLower(child)) {
				return true
			}
		}
	case []any:
		for _, nested := range item {
			if retiredStorage(nested, key) {
				return true
			}
		}
	}
	return false
}

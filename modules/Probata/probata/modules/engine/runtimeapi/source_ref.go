// Byline: Codex · GPT-5.6-Sol · 2026-08-30 (fixed Proffer source authority)
package runtimeapi

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"os"
	"strings"

	"github.com/Cursedpotential/probata/engine/objectstores"
)

// devFixtureBucket / devFixturePrefix name the synthetic non-canonical R2 location a
// Proffer run may start from, and only while PLATFORM_DEV_AUTH_BYPASS is set
// (D-125/D-127: the dev flag is the single switch for every dev-only
// admission). Synthetic rehearsal fixtures live there so they never touch
// casebible-sorted (owner rule: test data must never become canonical).
// Live finding 2026-09-05: the tool gateway's R2 resolver is proven against
// r2://nexus/proffer/test-fixtures/…, but this allowlist rejected it at the API.
// Production source authority (upload:// or Case Bible Sorted) is unchanged.
const (
	devFixtureBucket       = "nexus"
	devFixturePrefix       = "proffer/test-fixtures/"
	workbenchStagingPrefix = "workbench/staging/"
)

var devSourceBuckets = map[string]struct{}{
	"casebible-raw":        {},
	"casebible-quarantine": {},
}

func devFixtureSourcesEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("PLATFORM_DEV_AUTH_BYPASS"))) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

func safeObjectKey(key string) bool {
	if key == "" || strings.HasPrefix(key, "/") || strings.Contains(key, `\`) {
		return false
	}
	for _, segment := range strings.Split(key, "/") {
		if segment == ".." {
			return false
		}
	}
	return true
}

// validSourceVersionQuery admits only the exact object-version selector used by acquisition.
// Inputs: parsed source URI. Outputs: whether its query is absent or one bounded versionId.
// Effects: none; choose before root matching so version pins never broaden source authority.
// Byline: Codex / 2026-10-06.
func validSourceVersionQuery(parsed *url.URL) bool {
	if parsed.RawQuery == "" {
		return !parsed.ForceQuery
	}
	if parsed.Scheme == "upload" {
		return false
	}
	query, err := url.ParseQuery(parsed.RawQuery)
	if err != nil || len(query) != 1 {
		return false
	}
	versions := query["versionId"]
	return len(versions) == 1 && versions[0] != "" && versions[0] != "null" && len(versions[0]) <= 2048 && !strings.ContainsAny(versions[0], "\r\n\x00")
}

// validateAuthorizedSourceRef verifies source roots while preserving an optional exact version pin.
// Inputs: caller source URI. Outputs: admitted provider and object key or error.
// Effects: reads configured roots; the original URI is forwarded intact to acquisition.
// Choose for individual import admission; a query never admits a different root.
func validateAuthorizedSourceRef(value string) (string, string, error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.User != nil || parsed.Fragment != "" || !validSourceVersionQuery(parsed) {
		return "", "", errors.New("source_ref is outside the authorized Case Bible intake roots")
	}
	if parsed.Scheme == "upload" && parsed.Path == "" {
		digest := strings.ToLower(parsed.Host)
		raw, decodeErr := hex.DecodeString(digest)
		if decodeErr == nil && len(raw) == sha256.Size {
			return "upload", digest, nil
		}
	}
	// Configured roots (SOURCE_ROOTS_JSON) are production source authority for any
	// configured object store; the fixed R2 rules below remain for existing data.
	if roots, rootsErr := objectstores.RootsFromEnv(); rootsErr == nil && parsed.Host != "" {
		if key, keyErr := url.PathUnescape(strings.TrimPrefix(parsed.EscapedPath(), "/")); keyErr == nil && safeObjectKey(key) {
			if _, ok := roots.Match(parsed.Scheme, parsed.Host, key); ok {
				return strings.ToLower(parsed.Scheme), key, nil
			}
		}
	}
	devMode := devFixtureSourcesEnabled()
	stagedUpload := parsed.Scheme == "r2" && parsed.Host == "nexus" && strings.HasPrefix(parsed.Path, "/"+workbenchStagingPrefix)
	devFixture := parsed.Scheme == "r2" && parsed.Host == devFixtureBucket && devMode
	_, devSourceBucket := devSourceBuckets[parsed.Host]
	devSourceBucket = parsed.Scheme == "r2" && devSourceBucket && devMode
	if parsed.Scheme == "r2" && (parsed.Host == "casebible-sorted" || devFixture || devSourceBucket || stagedUpload) {
		key, unescapeErr := url.PathUnescape(strings.TrimPrefix(parsed.EscapedPath(), "/"))
		if devFixture && !stagedUpload && !strings.HasPrefix(key, devFixturePrefix) {
			return "", "", errors.New("source_ref is outside the authorized Case Bible intake roots")
		}
		if stagedUpload {
			parts := strings.Split(strings.TrimPrefix(key, workbenchStagingPrefix), "/")
			if len(parts) != 2 || parts[1] == "" || parts[1] == "." {
				return "", "", errors.New("source_ref is not a content-addressed Workbench staging object")
			}
			digest, decodeErr := hex.DecodeString(parts[0])
			if decodeErr != nil || len(digest) != sha256.Size {
				return "", "", errors.New("source_ref staging object requires a SHA-256 coordinate")
			}
		}
		if unescapeErr == nil && key != "" && !strings.HasPrefix(key, "/") && !strings.Contains(key, `\`) {
			valid := true
			for _, segment := range strings.Split(key, "/") {
				if segment == ".." {
					valid = false
					break
				}
			}
			if valid {
				return "r2", key, nil
			}
		}
	}
	return "", "", errors.New("source_ref is outside the authorized Case Bible intake roots")
}

// Byline: Claude Code · Fable 5.1 · 2026-09-20 (owner: storage providers are configuration, not code)

// Package objectstores is the single place the engine learns which
// S3-compatible object stores exist and which source roots a run may start
// from. Cloudflare R2, Backblaze B2 and any other S3-compatible provider differ
// only in an endpoint and a credential file, so neither is named in code:
//
//	OBJECT_STORES_JSON = {"b2": "/run/secrets/casebible-b2.json", "r2": "/run/secrets/casebible-r2.json"}
//	SOURCE_ROOTS_JSON  = [{"id": "b2-vault", "label": "B2 / Vault", "url": "b2://salem-data/consignatio/vault/v1/"}]
//
// A store's key is the locator scheme ("b2://bucket/key"). Adding or switching a
// provider is a credential file plus these two values; no resolver, allowlist
// or DuckDB mapping is edited. The package has no engine dependencies so every
// layer (acquisition, runtimeapi, postgres, cmd) can import it.
package objectstores

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
)

const (
	// StoresEnv maps a locator scheme to the absolute path of its credential document.
	StoresEnv = "OBJECT_STORES_JSON"
	// RootsEnv lists the object-store prefixes a Proffer run may start from.
	RootsEnv = "SOURCE_ROOTS_JSON"
)

var schemePattern = regexp.MustCompile(`^[a-z][a-z0-9]{0,15}$`)

// reservedSchemes are locator schemes owned by other resolvers.
var reservedSchemes = map[string]struct{}{"upload": {}, "file": {}, "http": {}, "https": {}, "s3": {}}

// Stores maps a locator scheme to its credential document path.
type Stores map[string]string

// ParseStores decodes OBJECT_STORES_JSON. Empty input is a valid empty set.
func ParseStores(raw string) (Stores, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Stores{}, nil
	}
	var stores Stores
	if err := json.Unmarshal([]byte(raw), &stores); err != nil {
		return nil, fmt.Errorf("objectstores: %s is not a JSON object of scheme to credential path: %w", StoresEnv, err)
	}
	for scheme, path := range stores {
		if !schemePattern.MatchString(scheme) {
			return nil, fmt.Errorf("objectstores: store scheme %q must be lower-case letters and digits", scheme)
		}
		if _, reserved := reservedSchemes[scheme]; reserved {
			return nil, fmt.Errorf("objectstores: store scheme %q is reserved", scheme)
		}
		// Container paths: POSIX-absolute regardless of the OS running the tests.
		if path != strings.TrimSpace(path) || !strings.HasPrefix(path, "/") {
			return nil, fmt.Errorf("objectstores: credential path for %q must be absolute and unpadded", scheme)
		}
	}
	return stores, nil
}

// StoresFromEnv reads OBJECT_STORES_JSON.
func StoresFromEnv() (Stores, error) { return ParseStores(os.Getenv(StoresEnv)) }

// Schemes returns the configured schemes in a stable order.
func (s Stores) Schemes() []string {
	schemes := make([]string, 0, len(s))
	for scheme := range s {
		schemes = append(schemes, scheme)
	}
	sort.Strings(schemes)
	return schemes
}

// Has reports whether scheme names a configured object store.
func (s Stores) Has(scheme string) bool {
	_, ok := s[strings.ToLower(scheme)]
	return ok
}

// Root is one object-store prefix a run may start from.
type Root struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	URL   string `json:"url"`
	// Temporary marks a root that is scheduled for retirement; surfaces show it.
	Temporary bool `json:"temporary,omitempty"`

	scheme, bucket, prefix string
}

// Scheme, Bucket and Prefix are the parsed parts of URL; Prefix is "" or ends in "/".
func (r Root) Scheme() string { return r.scheme }
func (r Root) Bucket() string { return r.bucket }
func (r Root) Prefix() string { return r.prefix }

// Roots is the configured allowlist.
type Roots []Root

// ParseRoots decodes SOURCE_ROOTS_JSON. Empty input is a valid empty set.
func ParseRoots(raw string) (Roots, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Roots{}, nil
	}
	var roots Roots
	if err := json.Unmarshal([]byte(raw), &roots); err != nil {
		return nil, fmt.Errorf("objectstores: %s is not a JSON array of roots: %w", RootsEnv, err)
	}
	seen := map[string]struct{}{}
	for i := range roots {
		root := &roots[i]
		if strings.TrimSpace(root.ID) == "" || strings.TrimSpace(root.Label) == "" {
			return nil, errors.New("objectstores: every source root needs an id and a label")
		}
		if _, dup := seen[root.ID]; dup {
			return nil, fmt.Errorf("objectstores: duplicate source root id %q", root.ID)
		}
		seen[root.ID] = struct{}{}
		parsed, err := url.Parse(root.URL)
		if err != nil || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Host == "" {
			return nil, fmt.Errorf("objectstores: source root %q url must be <scheme>://<bucket>/<prefix/>", root.ID)
		}
		scheme := strings.ToLower(parsed.Scheme)
		if !schemePattern.MatchString(scheme) {
			return nil, fmt.Errorf("objectstores: source root %q has an invalid scheme", root.ID)
		}
		prefix := strings.TrimPrefix(parsed.Path, "/")
		if prefix != "" && !strings.HasSuffix(prefix, "/") {
			return nil, fmt.Errorf("objectstores: source root %q prefix must end in /", root.ID)
		}
		for _, part := range strings.Split(prefix, "/") {
			if part == ".." || part == "." {
				return nil, fmt.Errorf("objectstores: source root %q prefix must not contain dot segments", root.ID)
			}
		}
		root.scheme, root.bucket, root.prefix = scheme, parsed.Host, prefix
	}
	return roots, nil
}

// RootsFromEnv reads SOURCE_ROOTS_JSON.
func RootsFromEnv() (Roots, error) { return ParseRoots(os.Getenv(RootsEnv)) }

// Match returns the root that contains scheme://bucket/key, if any. key is the
// unescaped object key. A key equal to the bare prefix is not an object.
func (r Roots) Match(scheme, bucket, key string) (Root, bool) {
	scheme = strings.ToLower(scheme)
	for _, root := range r {
		if root.scheme == scheme && root.bucket == bucket && strings.HasPrefix(key, root.prefix) && len(key) > len(root.prefix) {
			return root, true
		}
	}
	return Root{}, false
}

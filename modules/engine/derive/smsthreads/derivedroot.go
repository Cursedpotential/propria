// Byline: Claude Code · Opus 5 · 2026-09-20
//
// Where a derivation is published.
//
// Owner ruling 2026-09-20 23:51 SUPERSEDES the 18:51 "next to original"
// ruling: derived output goes under ONE separate top-level vault directory
// whose inner path mirrors the source tree, not beside the original as
// <key>.derived/.
//
// The mapping is configuration, never code: DERIVED_ROOTS_JSON carries a list
// of {"source": "...", "derived": "..."} locator-prefix pairs. A source that
// matches no pair falls back to the old beside-the-original placement, so
// nothing breaks before the variable is set. No provider, bucket or vault
// path is named in code, and there is no default pair.
//
// The target vault is NOT built yet (owner, 2026-09-21 00:05: "its not built
// yet make it configurable"), and its folder names are deliberately absent
// from this package. What this unit publishes is machine output — thread
// NDJSON and extracted media, source-associated processing artifacts — not a
// readable rendering, so nothing here assumes a reader-facing home either.
// The only shape this package knows is a pair of locator prefixes:
//
//	DERIVED_ROOTS_JSON=[{"source":"b2://bucket/source-root/",
//	                     "derived":"b2://bucket/derived-root/"}]
//
// Switching the pair on must not orphan or re-derive anything already
// published beside an original, so a lookup checks the mapped location first
// and then the legacy beside-the-original location (see ResolvePublished).

package smsthreads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

// DerivedRootsEnv is the only configuration input for derived placement.
const DerivedRootsEnv = "DERIVED_ROOTS_JSON"

// DerivedRootRule maps one source locator prefix to one derived locator
// prefix. Both are <scheme>://<bucket>/<prefix>; a missing trailing slash is
// added, because a prefix is a directory boundary and "v1" must not match
// "v10".
type DerivedRootRule struct {
	Source  string `json:"source"`
	Derived string `json:"derived"`
}

// DerivedRoots is the ordered rule list. The longest matching source prefix
// wins, so a more specific rule always beats a broader one regardless of order.
type DerivedRoots []DerivedRootRule

// DerivedLocation is the resolved publication target for one source object:
// everything the derivation writes lives under Prefix.
type DerivedLocation struct {
	Scheme string
	Bucket string
	// Prefix always ends in "/". Objects are published at
	// Prefix+"threads/...", Prefix+"media/...", Prefix+ManifestName.
	Prefix string
	// Mapped is false when no configured pair matched and the derivation fell
	// back to <key>.derived/ beside the original.
	Mapped bool
	// MatchedSource is the configured source prefix that matched, for logging.
	MatchedSource string
}

// URI spells the derived prefix back as a locator.
func (l DerivedLocation) URI() string {
	return fmt.Sprintf("%s://%s/%s", l.Scheme, l.Bucket, l.Prefix)
}

// ManifestKey is the object key of this location's manifest.
func (l DerivedLocation) ManifestKey() string { return l.Prefix + ManifestName }

// DerivedRootsFromEnv reads DERIVED_ROOTS_JSON. An unset or empty variable is
// not an error: every source then falls back to beside-the-original.
func DerivedRootsFromEnv() (DerivedRoots, error) {
	return ParseDerivedRoots(os.Getenv(DerivedRootsEnv))
}

// ParseDerivedRoots validates the configured pairs up front, so a typo is a
// startup failure rather than a derivation published in the wrong place.
func ParseDerivedRoots(raw string) (DerivedRoots, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	var rules DerivedRoots
	if err := json.Unmarshal([]byte(trimmed), &rules); err != nil {
		return nil, fmt.Errorf("smsthreads: %s is not a JSON list of {source,derived} pairs: %w", DerivedRootsEnv, err)
	}
	for index, rule := range rules {
		source, err := parsePrefixLocator(rule.Source)
		if err != nil {
			return nil, fmt.Errorf("smsthreads: %s[%d].source: %w", DerivedRootsEnv, index, err)
		}
		derived, err := parsePrefixLocator(rule.Derived)
		if err != nil {
			return nil, fmt.Errorf("smsthreads: %s[%d].derived: %w", DerivedRootsEnv, index, err)
		}
		if source.Scheme != derived.Scheme {
			return nil, fmt.Errorf(
				"smsthreads: %s[%d] crosses object-store schemes (%q -> %q); one derivation writes through one store client",
				DerivedRootsEnv, index, source.Scheme, derived.Scheme)
		}
		if source.Bucket == derived.Bucket && strings.HasPrefix(derived.Prefix, source.Prefix) {
			return nil, fmt.Errorf(
				"smsthreads: %s[%d] publishes derived output inside its own source root (%q under %q)",
				DerivedRootsEnv, index, derived.Prefix, source.Prefix)
		}
	}
	return rules, nil
}

// prefixLocator is one parsed <scheme>://<bucket>/<prefix>.
type prefixLocator struct {
	Scheme string
	Bucket string
	Prefix string // "" (bucket root) or ends in "/"
}

func parsePrefixLocator(value string) (prefixLocator, error) {
	scheme, rest, found := strings.Cut(strings.TrimSpace(value), "://")
	if !found || strings.TrimSpace(scheme) == "" {
		return prefixLocator{}, errors.New("must be <scheme>://<bucket>/<prefix>")
	}
	bucket, prefix, _ := strings.Cut(rest, "/")
	if strings.TrimSpace(bucket) == "" {
		return prefixLocator{}, errors.New("must name a bucket")
	}
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return prefixLocator{Scheme: strings.ToLower(strings.TrimSpace(scheme)), Bucket: bucket, Prefix: prefix}, nil
}

// LegacyLocation is the pre-2026-09-21 placement: <key>.derived/ beside the
// original. It stays readable forever so switching DERIVED_ROOTS_JSON on
// neither orphans nor re-derives an existing derivation.
func LegacyLocation(scheme, bucket, key string) DerivedLocation {
	return DerivedLocation{
		Scheme: strings.ToLower(strings.TrimSpace(scheme)), Bucket: bucket,
		Prefix: key + DerivedSuffix + "/",
	}
}

// ResolvePublished says where this source's derivation lives NOW and whether
// one is published. The mapped location wins; a derivation already published
// at the legacy beside-the-original location is found there and reused in
// place. When neither holds a manifest, the mapped location is returned as
// the place a new derivation will be written.
func ResolvePublished(
	ctx context.Context, store ObjectStore, roots DerivedRoots, scheme, bucket, key string,
) (DerivedLocation, bool, error) {
	mapped, err := roots.Locate(scheme, bucket, key)
	if err != nil {
		return DerivedLocation{}, false, err
	}
	if store == nil {
		return mapped, false, errors.New("smsthreads: object store is required to resolve a published derivation")
	}
	if done, err := store.Exists(ctx, mapped.Bucket, mapped.ManifestKey()); err != nil {
		return mapped, false, fmt.Errorf("smsthreads: check existing manifest: %w", err)
	} else if done {
		return mapped, true, nil
	}
	legacy := LegacyLocation(scheme, bucket, key)
	if legacy.Prefix == mapped.Prefix && legacy.Bucket == mapped.Bucket {
		return mapped, false, nil
	}
	if done, err := store.Exists(ctx, legacy.Bucket, legacy.ManifestKey()); err != nil {
		return mapped, false, fmt.Errorf("smsthreads: check legacy manifest: %w", err)
	} else if done {
		return legacy, true, nil
	}
	return mapped, false, nil
}

// RequireConfiguredSchemes fails a startup check when a configured pair names
// an object-store scheme the process has no credentials for. A typo must be a
// loud boot failure, never a silent fallback to beside-the-original.
func (r DerivedRoots) RequireConfiguredSchemes(configured []string) error {
	known := make(map[string]struct{}, len(configured))
	for _, scheme := range configured {
		known[strings.ToLower(strings.TrimSpace(scheme))] = struct{}{}
	}
	for index, rule := range r {
		source, err := parsePrefixLocator(rule.Source)
		if err != nil {
			return fmt.Errorf("smsthreads: %s[%d].source: %w", DerivedRootsEnv, index, err)
		}
		if _, ok := known[source.Scheme]; !ok {
			return fmt.Errorf(
				"smsthreads: %s[%d] uses object-store scheme %q, which is not configured (configured: %v)",
				DerivedRootsEnv, index, source.Scheme, configured)
		}
	}
	return nil
}

// Locate resolves where one source object's derivation is published.
//
// A matching pair maps
//
//	<scheme>://<bucket>/<source_root>/<path>/<file>
//
// to
//
//	<scheme>://<derived_bucket>/<derived_root>/<path>/<file>/
//
// so the inner path mirrors the source tree and the source file name becomes
// the derivation's own directory. No match falls back to <key>.derived/ beside
// the original and says so in the returned Mapped flag.
func (r DerivedRoots) Locate(scheme, bucket, key string) (DerivedLocation, error) {
	if strings.TrimSpace(scheme) == "" || strings.TrimSpace(bucket) == "" || strings.TrimSpace(key) == "" {
		return DerivedLocation{}, errors.New("smsthreads: derived placement requires a scheme, bucket and key")
	}
	fallback := LegacyLocation(scheme, bucket, key)
	lowerScheme := strings.ToLower(strings.TrimSpace(scheme))

	var bestSource, bestDerived prefixLocator
	matched := false
	for index, rule := range r {
		source, err := parsePrefixLocator(rule.Source)
		if err != nil {
			return DerivedLocation{}, fmt.Errorf("smsthreads: %s[%d].source: %w", DerivedRootsEnv, index, err)
		}
		derived, err := parsePrefixLocator(rule.Derived)
		if err != nil {
			return DerivedLocation{}, fmt.Errorf("smsthreads: %s[%d].derived: %w", DerivedRootsEnv, index, err)
		}
		if source.Scheme != lowerScheme || source.Bucket != bucket || !strings.HasPrefix(key, source.Prefix) {
			continue
		}
		if matched && len(source.Prefix) <= len(bestSource.Prefix) {
			continue
		}
		bestSource, bestDerived, matched = source, derived, true
	}
	if !matched {
		return fallback, nil
	}
	if derivedScheme := bestDerived.Scheme; derivedScheme != lowerScheme {
		return DerivedLocation{}, fmt.Errorf(
			"smsthreads: derived root for %q uses scheme %q but the source uses %q", bestSource.Prefix, derivedScheme, lowerScheme)
	}
	relative := strings.TrimPrefix(key, bestSource.Prefix)
	if strings.TrimSpace(relative) == "" {
		return DerivedLocation{}, fmt.Errorf("smsthreads: source key %q is the configured source root itself", key)
	}
	return DerivedLocation{
		Scheme: lowerScheme, Bucket: bestDerived.Bucket,
		Prefix: bestDerived.Prefix + relative + "/",
		Mapped: true, MatchedSource: bestSource.Prefix,
	}, nil
}

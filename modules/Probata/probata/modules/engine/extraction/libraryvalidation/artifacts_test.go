// Byline: Codex · GPT-6.1 · 2026-10-04.
package libraryvalidation

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type fixtureVersionStore struct {
	version, readVersion, key string
	body, returned            []byte
	writeErr, readErr         error
}

func (s *fixtureVersionStore) PutRecoveredVersion(_ context.Context, _ string, key string, body io.ReadSeeker, size int64, _ string, hash string, _ func(int64)) (string, error) {
	s.key = key
	s.body, _ = io.ReadAll(body)
	if int64(len(s.body)) != size || Hash(s.body) != "sha256:"+hash {
		return "", errors.New("provider write descriptor mismatch")
	}
	return s.version, s.writeErr
}

func (s *fixtureVersionStore) OpenVersion(_ context.Context, _ string, _ string, version string) (io.ReadCloser, error) {
	s.readVersion = version
	body := s.body
	if s.returned != nil {
		body = s.returned
	}
	return io.NopCloser(bytes.NewReader(body)), s.readErr
}

func TestB2ArtifactsRequireExactRetainedVersionAndFullBytes(t *testing.T) {
	provider := &fixtureVersionStore{version: "provider/version+encoded="}
	store := B2Artifacts{Store: provider, Bucket: "derivatives", Prefix: "library-validation"}
	ref, err := store.Put(t.Context(), "proposal/snapshot.html", []byte(fixtureText), "text/html")
	require.NoError(t, err)
	require.Equal(t, provider.version, provider.readVersion)
	require.Contains(t, provider.key, "library-validation/proposal/snapshot/")
	require.Contains(t, ref.URI, "versionId=provider%2Fversion%2Bencoded%3D")
	raw, err := store.Read(t.Context(), ref, MaxSourceBytes)
	require.NoError(t, err)
	require.Equal(t, []byte(fixtureText), raw)
	provider.returned = []byte(strings.Repeat("x", len(raw)))
	_, err = store.Read(t.Context(), ref, MaxSourceBytes)
	require.ErrorContains(t, err, "integrity")
	provider.returned = append(raw, 'x')
	_, err = store.Read(t.Context(), ref, MaxSourceBytes)
	require.ErrorContains(t, err, "integrity")
	provider.returned = nil
	_, err = store.Read(t.Context(), ref, ref.Bytes-1)
	require.ErrorContains(t, err, "budget")
	ref.URI = strings.Replace(ref.URI, "library-validation", "originals", 1)
	_, err = store.Read(t.Context(), ref, MaxSourceBytes)
	require.Error(t, err)
}

func TestB2ArtifactsFailVisibleForMissingVersionAndProviderErrors(t *testing.T) {
	for _, name := range []string{"missing", "null", "write failed", "read failed"} {
		t.Run(name, func(t *testing.T) {
			provider := &fixtureVersionStore{version: "fixture-version"}
			switch name {
			case "missing":
				provider.version = ""
			case "null":
				provider.version = "null"
			case "write failed":
				provider.writeErr = errors.New("provider secret must not surface")
			case "read failed":
				provider.readErr = errors.New("provider secret must not surface")
			}
			_, err := (B2Artifacts{Store: provider, Bucket: "derivatives", Prefix: "library-validation"}).Put(t.Context(), "snapshot.html", []byte(fixtureText), "text/html")
			require.Error(t, err)
			require.NotContains(t, err.Error(), "secret")
		})
	}
}

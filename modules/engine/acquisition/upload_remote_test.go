// Byline: Claude Code · Fable 5.1 · 2026-09-20 (cross-host upload:// resolution tests)
package acquisition

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/stretchr/testify/require"
)

// originServer mounts the object handler the way the starter does and presents
// every request as a tailnet peer, which httptest's loopback socket is not.
func originServer(t *testing.T, root string, peer string) *httptest.Server {
	t.Helper()
	handler, err := NewUploadObjectHandler(root)
	require.NoError(t, err)
	mux := http.NewServeMux()
	mux.Handle("GET "+UploadObjectPathPrefix+"{sha256}", handler)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.RemoteAddr = peer
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)
	return server
}

func sealForTest(t *testing.T, root string, content []byte) string {
	t.Helper()
	sealed, err := sealStream(context.Background(), root, bytes.NewReader(content))
	require.NoError(t, err)
	return hex.EncodeToString(sealed.ContentSHA256)
}

func TestRemoteUploadResolverFetchesAndSealsAcrossRoots(t *testing.T) {
	originRoot, gatewayRoot := t.TempDir(), t.TempDir()
	content := bytes.Repeat([]byte("uploaded-on-another-host"), 4_000)
	digestHex := sealForTest(t, originRoot, content)
	server := originServer(t, originRoot, "100.64.1.2:1234")

	resolver, err := NewRemoteUploadResolver(gatewayRoot, server.URL, 1<<20, server.Client())
	require.NoError(t, err)
	result, err := resolver(context.Background(), proffer.Ref("upload://"+digestHex))
	require.NoError(t, err)

	want := sha256.Sum256(content)
	require.Equal(t, want[:], result.ContentSHA256)
	require.Equal(t, int64(len(content)), result.ByteLength)
	require.Equal(t, storageClassSealed, result.StorageClass)

	// The gateway's own seal root now resolves it locally, byte for byte.
	local, err := NewUploadIngressResolver(gatewayRoot)
	require.NoError(t, err)
	again, err := local(context.Background(), proffer.Ref("upload://"+digestHex))
	require.NoError(t, err)
	require.Equal(t, want[:], again.ContentSHA256)
}

func TestRemoteUploadResolverFailsClosedWhenOriginReturnsOtherBytes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not the bytes that were asked for"))
	}))
	t.Cleanup(server.Close)
	resolver, err := NewRemoteUploadResolver(t.TempDir(), server.URL, 1<<20, server.Client())
	require.NoError(t, err)

	asked := sha256.Sum256([]byte("the real upload"))
	_, err = resolver(context.Background(), proffer.Ref("upload://"+hex.EncodeToString(asked[:])))
	require.ErrorContains(t, err, "do not hash to")
}

func TestRemoteUploadResolverRejectsMissingOversizedAndMalformed(t *testing.T) {
	originRoot := t.TempDir()
	content := bytes.Repeat([]byte("x"), 4_096)
	digestHex := sealForTest(t, originRoot, content)
	server := originServer(t, originRoot, "100.64.1.2:1234")

	resolver, err := NewRemoteUploadResolver(t.TempDir(), server.URL, 1<<20, server.Client())
	require.NoError(t, err)
	absent := sha256.Sum256([]byte("never uploaded"))
	_, err = resolver(context.Background(), proffer.Ref("upload://"+hex.EncodeToString(absent[:])))
	require.ErrorContains(t, err, "returned 404")

	_, err = resolver(context.Background(), proffer.Ref("r2://bucket/key"))
	require.Error(t, err)

	bounded, err := NewRemoteUploadResolver(t.TempDir(), server.URL, 1_024, server.Client())
	require.NoError(t, err)
	_, err = bounded(context.Background(), proffer.Ref("upload://"+digestHex))
	require.ErrorContains(t, err, "exceeds")
}

func TestUploadObjectHandlerRejectsNonTailnetPeerMalformedDigestAndWrongMethod(t *testing.T) {
	root := t.TempDir()
	digestHex := sealForTest(t, root, []byte("payload"))

	for _, peer := range []string{"192.0.2.1:1234", "100.63.1.2:1234", "100.128.1.2:1234", "127.0.0.1:1234"} {
		server := originServer(t, root, peer)
		resp, err := server.Client().Get(server.URL + UploadObjectPathPrefix + digestHex)
		require.NoError(t, err)
		resp.Body.Close()
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode, peer)
	}

	server := originServer(t, root, "100.64.1.2:1234")
	for _, bad := range []string{"abc", "..%2f..%2fetc%2fpasswd", "ZZ" + digestHex[2:]} {
		resp, err := server.Client().Get(server.URL + UploadObjectPathPrefix + bad)
		require.NoError(t, err)
		resp.Body.Close()
		require.NotEqual(t, http.StatusOK, resp.StatusCode, bad)
	}

	handler, err := NewUploadObjectHandler(root)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, UploadObjectPathPrefix+digestHex, nil)
	req.RemoteAddr = "100.64.1.2:1234"
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusMethodNotAllowed, recorder.Code)
}

func TestRemoteUploadResolverConfigValidation(t *testing.T) {
	client := http.DefaultClient
	for _, tc := range []struct {
		root, origin string
		max          int64
		client       *http.Client
	}{
		{"", "http://100.64.1.2:8091", 1, client},
		{t.TempDir(), "ftp://100.64.1.2", 1, client},
		{t.TempDir(), "not a url", 1, client},
		{t.TempDir(), "http://100.64.1.2:8091", 0, client},
		{t.TempDir(), "http://100.64.1.2:8091", 1, nil},
	} {
		_, err := NewRemoteUploadResolver(tc.root, tc.origin, tc.max, tc.client)
		require.Error(t, err)
	}
	_, err := NewUploadObjectHandler("")
	require.Error(t, err)
}

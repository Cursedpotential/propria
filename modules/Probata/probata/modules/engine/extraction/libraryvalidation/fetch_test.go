// Byline: Codex · GPT-6.1 · 2026-10-04.
package libraryvalidation

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConnectTimeDNSRejectsPrivateMixedAnswersAndDialsCheckedIP(t *testing.T) {
	for _, answers := range [][]netip.Addr{
		{netip.MustParseAddr("127.0.0.1")},
		{netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("10.0.0.1")},
		{netip.MustParseAddr("100.91.190.107")},
	} {
		called := false
		dial := checkedSourceDial(func(context.Context, string, string) ([]netip.Addr, error) { return answers, nil }, func(context.Context, string, string) (net.Conn, error) {
			called = true
			return nil, errors.New("unexpected dial")
		})
		_, err := dial(t.Context(), "tcp", "www.courts.michigan.gov:443")
		require.ErrorContains(t, err, "nonpublic")
		require.False(t, called)
	}
	lookups := 0
	dial := checkedSourceDial(func(_ context.Context, network, host string) ([]netip.Addr, error) {
		lookups++
		require.Equal(t, "ip", network)
		require.Equal(t, "www.courts.michigan.gov", host)
		return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
	}, func(_ context.Context, _ string, address string) (net.Conn, error) {
		require.Equal(t, "8.8.8.8:443", address)
		return nil, errors.New("fixture stop before socket creation")
	})
	_, err := dial(t.Context(), "tcp", "www.courts.michigan.gov:443")
	require.ErrorContains(t, err, "fixture stop")
	require.Equal(t, 1, lookups)
}

func fixtureFetcher(t *testing.T, handler http.HandlerFunc) *PrimaryFetcher {
	t.Helper()
	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)
	address := strings.TrimPrefix(server.URL, "https://")
	transport := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, address)
	}}
	t.Cleanup(transport.CloseIdleConnections)
	return &PrimaryFetcher{client: &http.Client{Transport: transport, CheckRedirect: sourceRedirect}, MaxBytes: MaxSourceBytes}
}

func TestOfficialSourceNetworkFixtures(t *testing.T) {
	for _, test := range []struct {
		name    string
		handler http.HandlerFunc
		budget  int64
		code    string
	}{
		{"official HTML", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte("<h1>Rule 3.215</h1>"))
		}, 1024, ""},
		{"legislature forbidden", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(403) }, 1024, "HTTP_403"},
		{"host redirect", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "https://attacker.invalid/source", 302)
		}, 1024, "FETCH_OR_REDIRECT_BLOCKED"},
		{"private IP redirect", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "https://127.0.0.1/private", 302) }, 1024, "FETCH_OR_REDIRECT_BLOCKED"},
		{"downgrade redirect", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "http://www.courts.michigan.gov/rules", 302)
		}, 1024, "FETCH_OR_REDIRECT_BLOCKED"},
		{"redirect loop", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "https://www.courts.michigan.gov/loop", 302)
		}, 1024, "FETCH_OR_REDIRECT_BLOCKED"},
		{"stream body budget", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
			_, _ = w.Write([]byte(strings.Repeat("x", 2048)))
		}, 128, "BODY_BUDGET"},
		{"declared body budget", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.Header().Set("Content-Length", "999999")
			w.WriteHeader(200)
		}, 128, "BODY_BUDGET"},
		{"PDF cannot be UTF8 fiction", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/pdf")
			_, _ = w.Write([]byte("rule text pretending to be a PDF"))
		}, 1024, "PDF_MAGIC"},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := fixtureFetcher(t, test.handler)
			f.MaxBytes = test.budget
			body, err := f.Fetch(t.Context(), "https://www.courts.michigan.gov/source")
			if test.code == "" {
				require.NoError(t, err)
				require.NotEmpty(t, body.Bytes)
			} else {
				var blocked *FetchError
				require.ErrorAs(t, err, &blocked)
				require.Equal(t, test.code, blocked.Code)
			}
		})
	}
}

func TestSourceURLAndPrivateAddressPolicy(t *testing.T) {
	for _, raw := range []string{"https://www.courts.michigan.gov/official", "https://legislature.mi.gov/Laws/MCL?objectName=mcl-552-507", "https://michigan.gov/policy"} {
		_, err := PrimaryURL(raw)
		require.NoError(t, err)
	}
	for _, raw := range []string{"https://courts.michigan.gov.attacker.invalid/x", "https://evil.courts.michigan.gov/x", "https://user:secret@www.courts.michigan.gov/x", "https://www.courts.michigan.gov:8080/x", "https://[::1]/x", "https://169.254.169.254/latest/meta-data", "http://michigan.gov/x"} {
		_, err := PrimaryURL(raw)
		require.Error(t, err)
	}
	for _, raw := range []string{"127.0.0.1", "10.0.0.1", "172.16.0.1", "192.168.1.1", "100.91.190.107", "169.254.169.254", "0.0.0.0", "224.0.0.1", "::1", "fc00::1", "fe80::1", "::ffff:127.0.0.1", "2002:7f00:1::1", "64:ff9b::a00:1"} {
		require.False(t, publicAddress(netip.MustParseAddr(raw)), raw)
	}
	require.True(t, publicAddress(netip.MustParseAddr("8.8.8.8")))
	transport := NewPrimaryFetcher().client.Transport.(*http.Transport)
	_, err := transport.DialContext(t.Context(), "tcp", "127.0.0.1:443")
	require.Error(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = NewPrimaryFetcher().Fetch(ctx, "https://www.courts.michigan.gov/source")
	require.Error(t, err)
	require.True(t, errors.Is(ctx.Err(), context.Canceled))
}

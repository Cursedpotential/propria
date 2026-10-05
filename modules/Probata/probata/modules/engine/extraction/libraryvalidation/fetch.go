// Byline: Codex · GPT-6.1 · 2026-10-04.
package libraryvalidation

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

var officialHosts = map[string]bool{
	"courts.michigan.gov": true, "www.courts.michigan.gov": true,
	"legislature.mi.gov": true, "www.legislature.mi.gov": true,
	"michigan.gov": true, "www.michigan.gov": true,
}

// PrimaryURL permits only exact official HTTPS hosts and standard TLS ports.
// Inputs: stored official URL; outputs: parsed URL or error. Effects: none; choose at initial and every redirect boundary.
func PrimaryURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || !officialHosts[strings.ToLower(u.Hostname())] || u.User != nil || u.Fragment != "" || (u.Port() != "" && u.Port() != "443") || strings.ContainsAny(raw, "\r\n\x00") {
		return nil, errors.New("primary URL is outside the official HTTPS allowlist")
	}
	return u, nil
}

// publicAddress rejects private, special-purpose and non-unicast destinations even for an allowed hostname.
// Inputs: one DNS address; outputs: admission decision. Effects: none; choose immediately before dialing the checked address.
func publicAddress(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsValid() || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return false
	}
	for _, raw := range []string{"100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4", "2001:db8::/32", "2001::/32", "2002::/16", "64:ff9b::/96"} {
		if netip.MustParsePrefix(raw).Contains(ip) {
			return false
		}
	}
	return true
}

// PrimaryBody carries bounded raw source bytes and observed transport metadata inside an Activity only.
// Inputs: admitted HTTP response; outputs: complete bytes and final URL. Effects: none; never use it as a validation verdict.
type PrimaryBody struct {
	Bytes                                   []byte
	FinalURL, MediaType, ETag, LastModified string
}

// PrimaryFetcher uses a proxy-free DNS-pinned transport; tests replace its private client with network fixtures.
// Inputs: official URL; outputs: bounded source body. Effects: HTTPS reads only.
type PrimaryFetcher struct {
	client   *http.Client
	MaxBytes int64
}

// NewPrimaryFetcher creates bounded network acquisition with allowlisted redirects and connect-time public-IP checks.
// Inputs: none; outputs: safe fetcher. Effects: none until Fetch; choose over ambient/default HTTP clients.
func NewPrimaryFetcher() *PrimaryFetcher {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.TLSHandshakeTimeout = 10 * time.Second
	transport.ResponseHeaderTimeout = 15 * time.Second
	transport.MaxResponseHeaderBytes = 64 << 10
	dialer := net.Dialer{Timeout: 10 * time.Second}
	transport.DialContext = checkedSourceDial(net.DefaultResolver.LookupNetIP, dialer.DialContext)
	return &PrimaryFetcher{client: &http.Client{Transport: transport, Timeout: FetchTimeout, CheckRedirect: sourceRedirect}, MaxBytes: MaxSourceBytes}
}

// checkedSourceDial checks the entire DNS answer and dials numeric addresses without a second lookup.
// Inputs: context-aware lookup/dial contracts; outputs: transport dial function. Effects: DNS and bounded socket I/O.
func checkedSourceDial(lookup func(context.Context, string, string) ([]netip.Addr, error), dial func(context.Context, string, string) (net.Conn, error)) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil || !officialHosts[strings.ToLower(host)] || port != "443" {
			return nil, errors.New("source dial is not allowlisted")
		}
		ips, err := lookup(ctx, "ip", host)
		if err != nil || len(ips) == 0 {
			return nil, errors.New("source DNS lookup failed")
		}
		for _, ip := range ips {
			if !publicAddress(ip) {
				return nil, errors.New("source DNS includes a nonpublic address")
			}
		}
		// Dial the checked address itself, never perform a second hostname lookup.
		var last error
		for _, ip := range ips {
			conn, err := dial(ctx, network, net.JoinHostPort(ip.String(), port))
			if err == nil {
				return conn, nil
			}
			last = err
		}
		return nil, last
	}
}

// sourceRedirect reapplies the exact URL policy at each hop; inputs: next request/history; outputs: refusal or nil.
func sourceRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 5 {
		return errors.New("source redirect budget exceeded")
	}
	_, err := PrimaryURL(req.URL.String())
	return err
}

// Fetch reads one complete official source within byte/time budgets and returns explicit transport failure codes.
// Inputs: URL; outputs: body or FetchError. Effects: network reads; choose before separately validating text and currency.
func (f *PrimaryFetcher) Fetch(ctx context.Context, raw string) (PrimaryBody, error) {
	if _, err := PrimaryURL(raw); err != nil {
		return PrimaryBody{}, &FetchError{Code: "URL_POLICY"}
	}
	if f == nil || f.client == nil || f.MaxBytes <= 0 || f.MaxBytes > MaxSourceBytes {
		return PrimaryBody{}, errors.New("primary fetcher is not configured")
	}
	ctx, cancel := context.WithTimeout(ctx, FetchTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return PrimaryBody{}, &FetchError{Code: "REQUEST"}
	}
	req.Header.Set("Accept", "text/html,application/pdf,text/plain;q=0.8")
	resp, err := f.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return PrimaryBody{}, &FetchError{Code: "TIME_BUDGET"}
		}
		return PrimaryBody{}, &FetchError{Code: "FETCH_OR_REDIRECT_BLOCKED"}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return PrimaryBody{}, &FetchError{Code: fmt.Sprintf("HTTP_%d", resp.StatusCode)}
	}
	if _, err = PrimaryURL(resp.Request.URL.String()); err != nil {
		return PrimaryBody{}, &FetchError{Code: "FINAL_URL_POLICY"}
	}
	if resp.ContentLength > f.MaxBytes {
		return PrimaryBody{}, &FetchError{Code: "BODY_BUDGET"}
	}
	contentType, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil {
		return PrimaryBody{}, &FetchError{Code: "CONTENT_TYPE"}
	}
	switch contentType {
	case "text/html", "application/xhtml+xml", "application/pdf", "text/plain":
	default:
		return PrimaryBody{}, &FetchError{Code: "UNSUPPORTED_MEDIA"}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, f.MaxBytes+1))
	if err != nil {
		return PrimaryBody{}, &FetchError{Code: "BODY_READ"}
	}
	if len(body) == 0 || int64(len(body)) > f.MaxBytes {
		return PrimaryBody{}, &FetchError{Code: "BODY_BUDGET"}
	}
	if contentType == "application/pdf" && !strings.HasPrefix(string(body), "%PDF-") {
		return PrimaryBody{}, &FetchError{Code: "PDF_MAGIC"}
	}
	return PrimaryBody{Bytes: body, FinalURL: resp.Request.URL.String(), MediaType: contentType, ETag: resp.Header.Get("ETag"), LastModified: resp.Header.Get("Last-Modified")}, nil
}

// FetchError exposes a stable failure code without source bodies, credentials or personal content.
// Inputs: transport refusal; outputs: safe Activity diagnostics. Effects: none; choose over raw response snippets.
type FetchError struct{ Code string }

// Error returns the bounded network failure; inputs: none; outputs: safe code; effects: none.
func (e *FetchError) Error() string { return "primary source fetch blocked: " + e.Code }

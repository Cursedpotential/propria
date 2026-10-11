// Byline: Codex · GPT-5 · 2026-10-07 (owner-authorized internal service network repair).
package runtimeapi

import (
	"crypto/hmac"
	"errors"
	"net/http"
	"strings"

	"github.com/Cursedpotential/probata/engine/servicepeer"
)

const profferInternalServiceCIDRsEnv = servicepeer.InternalCIDRsEnv

// profferServiceNetworks validates explicit internal IPv4 networks beside the tailnet.
// Inputs: comma-separated canonical RFC1918 CIDRs, /16 or narrower; empty means tailnet only.
// Outputs: complete allowlist or an error (never a partially accepted list). Effects: none.
// Choose for shared mounted-token Proffer routes, not independently governed toolkit/user gates.
func profferServiceNetworks(value string) (servicepeer.Networks, error) {
	return servicepeer.Parse(value)
}

// profferServiceAuth authenticates direct internal socket peers with the mounted service token.
// Inputs: token path, denial text and next handler; network configuration is captured at route construction.
// Outputs: handler; invalid network configuration denies every request, including tailnet peers.
// Effects: reloads the credential on every request and applies the existing canonical-write fence.
// Choose for both reads and writes; downstream actor, idempotency and case-binding gates still apply.
func profferServiceAuth(tokenPath, denied string, next http.HandlerFunc) http.HandlerFunc {
	networks, networkErr := servicepeer.FromEnvironment()
	return func(w http.ResponseWriter, r *http.Request) {
		allowed := networkErr == nil && networks.Allows(r.RemoteAddr)
		credential, tokenErr := loadServiceToken(tokenPath)
		header := strings.TrimSpace(r.Header.Get("Authorization"))
		provided := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		trusted := tokenErr == nil && strings.HasPrefix(header, "Bearer ") && hmac.Equal([]byte(provided), credential)
		if !allowed || !trusted {
			previewError(w, http.StatusUnauthorized, errors.New(denied))
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if !canonicalRequestWrite(w, r) {
			return
		}
		next(w, r)
	}
}

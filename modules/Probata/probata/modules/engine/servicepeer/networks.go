// Byline: Codex · GPT-5 · 2026-10-07 (owner-authorized shared internal socket-peer policy).
// Package servicepeer validates explicit owned internal networks without depending on HTTP services.
package servicepeer

import (
	"errors"
	"net"
	"net/netip"
	"os"
	"strings"
)

// InternalCIDRsEnv names the shared explicit service-network configuration.
const InternalCIDRsEnv = "PROFFER_INTERNAL_SERVICE_CIDRS"

// Networks is an immutable parsed direct-peer policy; its zero value admits nobody.
type Networks struct{ prefixes []netip.Prefix }

// Parse validates complete canonical RFC1918 networks beside the existing IPv4 tailnet.
// Inputs: comma-separated CIDRs, /16 or narrower; empty means tailnet only.
// Outputs: policy or error, never partial admission. Effects: none.
// Choose for Proffer socket admission; credential and write checks remain the caller's responsibility.
func Parse(value string) (Networks, error) {
	networks := Networks{prefixes: []netip.Prefix{netip.MustParsePrefix("100.64.0.0/10")}}
	if strings.TrimSpace(value) == "" {
		return networks, nil
	}
	for _, raw := range strings.Split(value, ",") {
		prefix, err := netip.ParsePrefix(strings.TrimSpace(raw))
		if err != nil || !prefix.Addr().Is4() || !prefix.Addr().IsPrivate() || prefix.Bits() < 16 || prefix != prefix.Masked() || prefix.String() != strings.TrimSpace(raw) || prefix.String() == "192.168.0.0/16" {
			return Networks{}, errors.New("PROFFER_INTERNAL_SERVICE_CIDRS must contain canonical private IPv4 networks, /16 or narrower; RFC1918 blankets are forbidden")
		}
		networks.prefixes = append(networks.prefixes, prefix)
	}
	return networks, nil
}

// FromEnvironment reads and validates the shared Proffer service-network configuration.
// Inputs: InternalCIDRsEnv. Outputs: policy or configuration error. Effects: environment read only.
// Choose at route construction or a network-only admission boundary; invalid config must never be bypassed.
func FromEnvironment() (Networks, error) { return Parse(os.Getenv(InternalCIDRsEnv)) }

// Allows matches the actual socket endpoint, never proxy or identity headers.
// Inputs: RemoteAddr in host:port format. Outputs: admission boolean. Effects: none.
// Choose before any service effects; IPv4-mapped peers retain the previous IPv4 admission semantics.
func (n Networks) Allows(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddr))
	peer, parseErr := netip.ParseAddr(host)
	if err != nil || parseErr != nil || peer.Zone() != "" {
		return false
	}
	peer = peer.Unmap()
	for _, network := range n.prefixes {
		if network.Contains(peer) {
			return true
		}
	}
	return false
}

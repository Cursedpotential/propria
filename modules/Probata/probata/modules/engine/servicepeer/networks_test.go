// Byline: Codex · GPT-5 · 2026-10-07.
package servicepeer

import (
	"fmt"
	"strings"
	"testing"
)

// TestDirectPeerPolicy checks zero-value, canonical prefix and socket endpoint behavior.
// Inputs: synthetic networks/endpoints. Outputs: admission assertions. Effects: none.
// Choose for the leaf policy used by runtime API, acquisition and Temporal HTTP admission.
func TestDirectPeerPolicy(t *testing.T) {
	if (Networks{}).Allows("100.91.190.107:4444") {
		t.Fatal("zero policy admits peers")
	}
	networks, err := Parse("172.25.0.0/16,192.168.8.0/24,10.8.0.0/24")
	if err != nil {
		t.Fatal(err)
	}
	for _, peer := range []string{"100.91.190.107:4444", "172.25.0.27:4444", "192.168.8.7:4444", "10.8.0.7:4444", "[::ffff:172.25.0.27]:4444"} {
		if !networks.Allows(peer) {
			t.Errorf("expected admission: %s", peer)
		}
	}
	for _, peer := range []string{"172.26.0.27:4444", "203.0.113.9:4444", "[fd00::1]:4444", "[::ffff:172.25.0.27%zone]:4444", "172.25.0.27", "not-an-ip:4444"} {
		if networks.Allows(peer) {
			t.Errorf("unexpected admission: %s", peer)
		}
	}
}

// TestOwnedNetworkInventory accepts the owner's measured two-host CIDR shapes without aggregating them.
// Inputs: 57 individual private Docker/VPS networks. Outputs: complete policy and per-network admission.
// Effects: none. Choose to prevent a policy change from silently excluding measured legitimate networks.
func TestOwnedNetworkInventory(t *testing.T) {
	var cidrs, peers []string
	for subnet := 0; subnet <= 23; subnet++ {
		cidrs = append(cidrs, fmt.Sprintf("10.200.%d.0/24", subnet))
		peers = append(peers, fmt.Sprintf("10.200.%d.1:4444", subnet))
	}
	cidrs = append(cidrs, "10.201.0.0/29", "10.201.8.0/29", "10.250.72.0/24")
	peers = append(peers, "10.201.0.1:4444", "10.201.8.1:4444", "10.250.72.1:4444")
	for subnet := 17; subnet <= 31; subnet++ {
		cidrs = append(cidrs, fmt.Sprintf("172.%d.0.0/16", subnet))
		peers = append(peers, fmt.Sprintf("172.%d.0.1:4444", subnet))
	}
	for _, subnet := range []int{0, 16, 32, 48, 64, 96, 112, 128, 144, 160, 176, 192, 208, 224, 240} {
		cidrs = append(cidrs, fmt.Sprintf("192.168.%d.0/20", subnet))
		peers = append(peers, fmt.Sprintf("192.168.%d.1:4444", subnet))
	}
	if len(cidrs) != 57 {
		t.Fatalf("inventory length: %d", len(cidrs))
	}
	networks, err := Parse(strings.Join(cidrs, ","))
	if err != nil {
		t.Fatal(err)
	}
	for _, peer := range peers {
		if !networks.Allows(peer) {
			t.Errorf("inventory peer rejected: %s", peer)
		}
	}
}

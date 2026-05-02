package amnezigo

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNetworkConfig_JSONRoundTrip(t *testing.T) {
	nc := NetworkConfig{
		MTU: 1280,
		DNS: []string{"1.1.1.1", "8.8.8.8"},
	}
	b, err := json.Marshal(nc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded NetworkConfig
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.MTU != nc.MTU {
		t.Errorf("MTU = %d, want %d", decoded.MTU, nc.MTU)
	}
	if len(decoded.DNS) != len(nc.DNS) {
		t.Fatalf("DNS len = %d, want %d", len(decoded.DNS), len(nc.DNS))
	}
	for i := range nc.DNS {
		if decoded.DNS[i] != nc.DNS[i] {
			t.Errorf("DNS[%d] = %q, want %q", i, decoded.DNS[i], nc.DNS[i])
		}
	}
}

func TestNetworkConfig_OmitsEmptyFields(t *testing.T) {
	nc := NetworkConfig{}
	b, err := json.Marshal(nc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(b)
	if strings.Contains(got, `"mtu"`) {
		t.Errorf("empty MTU (0) should be omitted, got %q", got)
	}
	if strings.Contains(got, `"dns"`) {
		t.Errorf("nil DNS should be omitted, got %q", got)
	}
}

func TestObfuscationManifest_ExplicitParams(t *testing.T) {
	input := `{
		"s1": 30, "s2": 35, "s3": 20, "s4": 12,
		"h1": {"min": 100, "max": 5000000},
		"jc": 5, "jmin": 250, "jmax": 750
	}`
	var om ObfuscationManifest
	if err := json.Unmarshal([]byte(input), &om); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if om.S1 == nil || *om.S1 != 30 {
		t.Errorf("S1 = %v, want 30", om.S1)
	}
	if om.H1 == nil || om.H1.Min != 100 || om.H1.Max != 5000000 {
		t.Errorf("H1 = %v, want {100, 5000000}", om.H1)
	}
	if om.Jc == nil || *om.Jc != 5 {
		t.Errorf("Jc = %v, want 5", om.Jc)
	}
}

func TestObfuscationManifest_ZeroValueDistinctFromAbsent(t *testing.T) {
	withZero := `{"s1": 0}`
	var om ObfuscationManifest
	if err := json.Unmarshal([]byte(withZero), &om); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if om.S1 == nil {
		t.Fatal("S1 = nil after explicit 0; pointer must distinguish 0 from absent")
	}
	if *om.S1 != 0 {
		t.Errorf("S1 = %d, want 0", *om.S1)
	}

	withoutS1 := `{}`
	var om2 ObfuscationManifest
	if err := json.Unmarshal([]byte(withoutS1), &om2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if om2.S1 != nil {
		t.Errorf("S1 should be nil when absent, got %v", om2.S1)
	}
}

func TestObfuscationManifest_OmitsNilFieldsOnMarshal(t *testing.T) {
	om := ObfuscationManifest{Protocol: "quic"}
	b, err := json.Marshal(om)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(b)
	if strings.Contains(got, `"s1"`) {
		t.Errorf("nil S1 should be omitted, got %q", got)
	}
	if strings.Contains(got, `"h1"`) {
		t.Errorf("nil H1 should be omitted, got %q", got)
	}
	if !strings.Contains(got, `"protocol":"quic"`) {
		t.Errorf("protocol should be present, got %q", got)
	}
}

func TestPeerManifest_JSONRoundTrip_Server(t *testing.T) {
	input := `{
		"address": "10.0.0.1/24",
		"tun_name": "awg0",
		"main_iface": "eth0",
		"endpoint": "vpn.example.com:51820",
		"listen_port": 51820
	}`
	var pm PeerManifest
	if err := json.Unmarshal([]byte(input), &pm); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if pm.Address != "10.0.0.1/24" {
		t.Errorf("Address = %q, want %q", pm.Address, "10.0.0.1/24")
	}
	if pm.Endpoint != "vpn.example.com:51820" {
		t.Errorf("Endpoint = %q", pm.Endpoint)
	}
	if pm.ListenPort != 51820 {
		t.Errorf("ListenPort = %d", pm.ListenPort)
	}
	if pm.TunName != "awg0" {
		t.Errorf("TunName = %q, want %q", pm.TunName, "awg0")
	}
	if pm.MainIface != "eth0" {
		t.Errorf("MainIface = %q, want %q", pm.MainIface, "eth0")
	}
}

func TestPeerManifest_JSONRoundTrip_Client(t *testing.T) {
	input := `{
		"address": "10.0.0.2/32",
		"protocol": "sip",
		"keepalive": 25
	}`
	var pm PeerManifest
	if err := json.Unmarshal([]byte(input), &pm); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if pm.Address != "10.0.0.2/32" {
		t.Errorf("Address = %q", pm.Address)
	}
	if pm.Protocol != "sip" {
		t.Errorf("Protocol = %q", pm.Protocol)
	}
	if pm.Keepalive == nil || *pm.Keepalive != 25 {
		t.Errorf("Keepalive = %v, want 25", pm.Keepalive)
	}
	if pm.Endpoint != "" {
		t.Errorf("Endpoint should be empty for client, got %q", pm.Endpoint)
	}
	if pm.ListenPort != 0 {
		t.Errorf("ListenPort should be 0 for client, got %d", pm.ListenPort)
	}
	if pm.MainIface != "" {
		t.Errorf("MainIface should be empty for client, got %q", pm.MainIface)
	}
}

func TestPeerManifest_IsServer(t *testing.T) {
	server := PeerManifest{
		Address:    "10.0.0.1/24",
		Endpoint:   "vpn.example.com:51820",
		ListenPort: 51820,
	}
	if !server.IsServer() {
		t.Error("peer with endpoint + listen_port should be a server")
	}

	client := PeerManifest{
		Address: "10.0.0.2/32",
	}
	if client.IsServer() {
		t.Error("peer without endpoint should not be a server")
	}

	// Edge case: endpoint without listen_port — not a valid server.
	partial := PeerManifest{
		Address:  "10.0.0.3/32",
		Endpoint: "vpn.example.com:51820",
	}
	if partial.IsServer() {
		t.Error("peer with endpoint but no listen_port should not be a server")
	}

	// Edge case: listen_port without endpoint — not a valid server.
	partial2 := PeerManifest{
		Address:    "10.0.0.4/32",
		ListenPort: 51820,
	}
	if partial2.IsServer() {
		t.Error("peer with listen_port but no endpoint should not be a server")
	}
}

func TestPeerManifest_OmitsEmptyFields(t *testing.T) {
	pm := PeerManifest{Address: "10.0.0.2/32"}
	b, err := json.Marshal(pm)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(b)
	for _, field := range []string{"tun_name", "main_iface", "endpoint", "listen_port", "protocol", "keepalive"} {
		if strings.Contains(got, `"`+field+`"`) {
			t.Errorf("empty %s should be omitted, got %q", field, got)
		}
	}
	if !strings.Contains(got, `"address"`) {
		t.Errorf("address should always be present, got %q", got)
	}
}

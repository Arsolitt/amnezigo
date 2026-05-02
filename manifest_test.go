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

func TestManifest_JSONRoundTrip_Full(t *testing.T) {
	// JSON matching the roadmap draft schema (docs/roadmap.md:401-431).
	input := `{
		"version": 1,
		"network": {
			"mtu": 1280,
			"dns": ["1.1.1.1", "8.8.8.8"]
		},
		"obfuscation": {
			"s1": 30, "s2": 35, "s3": 20, "s4": 12,
			"h1": {"min": 100, "max": 5000000},
			"jc": 5, "jmin": 250, "jmax": 750,
			"protocol": "quic"
		},
		"peers": {
			"server": {
				"address": "10.0.0.1/24",
				"tun_name": "awg0",
				"main_iface": "eth0",
				"endpoint": "vpn.example.com:51820",
				"listen_port": 51820
			},
			"phone": {
				"address": "10.0.0.2/32",
				"protocol": "sip"
			},
			"laptop": {
				"address": "10.0.0.3/32"
			}
		}
	}`

	var m Manifest
	if err := json.Unmarshal([]byte(input), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Version
	if m.Version != 1 {
		t.Errorf("Version = %d, want 1", m.Version)
	}

	// Network
	if m.Network.MTU != 1280 {
		t.Errorf("Network.MTU = %d, want 1280", m.Network.MTU)
	}
	if len(m.Network.DNS) != 2 || m.Network.DNS[0] != "1.1.1.1" {
		t.Errorf("Network.DNS = %v", m.Network.DNS)
	}

	// Obfuscation
	if m.Obfuscation.S1 == nil || *m.Obfuscation.S1 != 30 {
		t.Errorf("Obfuscation.S1 = %v, want 30", m.Obfuscation.S1)
	}
	if m.Obfuscation.Protocol != "quic" {
		t.Errorf("Obfuscation.Protocol = %q", m.Obfuscation.Protocol)
	}

	// Peers
	if len(m.Peers) != 3 {
		t.Fatalf("Peers count = %d, want 3", len(m.Peers))
	}

	server, ok := m.Peers["server"]
	if !ok {
		t.Fatal("missing 'server' peer")
	}
	if !server.IsServer() {
		t.Error("'server' peer should be identified as server")
	}
	if server.Address != "10.0.0.1/24" {
		t.Errorf("server.Address = %q", server.Address)
	}

	phone, ok := m.Peers["phone"]
	if !ok {
		t.Fatal("missing 'phone' peer")
	}
	if phone.IsServer() {
		t.Error("'phone' should be a client peer")
	}
	if phone.Protocol != "sip" {
		t.Errorf("phone.Protocol = %q", phone.Protocol)
	}

	laptop, ok := m.Peers["laptop"]
	if !ok {
		t.Fatal("missing 'laptop' peer")
	}
	if laptop.IsServer() {
		t.Error("'laptop' should be a client peer")
	}

	// Round-trip: marshal and unmarshal again.
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m2 Manifest
	if err := json.Unmarshal(b, &m2); err != nil {
		t.Fatalf("round-trip unmarshal: %v", err)
	}
	if m2.Version != m.Version {
		t.Errorf("round-trip Version mismatch: %d vs %d", m2.Version, m.Version)
	}
	if len(m2.Peers) != len(m.Peers) {
		t.Errorf("round-trip Peers count mismatch: %d vs %d", len(m2.Peers), len(m.Peers))
	}
}

func TestManifest_JSONRoundTrip_ExplicitObfuscation(t *testing.T) {
	input := `{
		"version": 1,
		"network": {"mtu": 1280},
		"obfuscation": {
			"s1": 30, "s2": 35, "s3": 20, "s4": 12,
			"h1": {"min": 100, "max": 5000000},
			"h2": {"min": 10000000, "max": 200000000},
			"h3": {"min": 400000000, "max": 800000000},
			"h4": {"min": 1000000000, "max": 2100000000},
			"jc": 5, "jmin": 250, "jmax": 750,
			"protocol": "quic"
		},
		"peers": {
			"server": {
				"address": "10.0.0.1/24",
				"endpoint": "vpn.example.com:51820",
				"listen_port": 51820
			},
			"client": {
				"address": "10.0.0.2/32"
			}
		}
	}`

	var m Manifest
	if err := json.Unmarshal([]byte(input), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m.Obfuscation.S1 == nil || *m.Obfuscation.S1 != 30 {
		t.Errorf("S1 = %v, want 30", m.Obfuscation.S1)
	}
	if m.Obfuscation.H1 == nil || m.Obfuscation.H1.Min != 100 {
		t.Errorf("H1.Min = %v, want 100", m.Obfuscation.H1)
	}
}

func TestManifest_EmptyPeers(t *testing.T) {
	input := `{
		"version": 1,
		"network": {},
		"obfuscation": {"protocol": "quic"},
		"peers": {}
	}`
	var m Manifest
	if err := json.Unmarshal([]byte(input), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(m.Peers) != 0 {
		t.Errorf("Peers should be empty, got %d", len(m.Peers))
	}
}

func TestManifest_ServerPeerCount(t *testing.T) {
	m := Manifest{
		Version: 1,
		Peers: map[string]PeerManifest{
			"server": {
				Address: "10.0.0.1/24", Endpoint: "vpn.example.com:51820",
				ListenPort: 51820,
			},
			"phone":  {Address: "10.0.0.2/32"},
			"laptop": {Address: "10.0.0.3/32"},
		},
	}
	name, count := m.ServerPeer()
	if count != 1 {
		t.Errorf("expected 1 server peer, got %d", count)
	}
	if name != "server" {
		t.Errorf("server peer name = %q, want %q", name, "server")
	}
}

func TestManifest_ServerPeerCount_None(t *testing.T) {
	m := Manifest{
		Version: 1,
		Peers: map[string]PeerManifest{
			"phone":  {Address: "10.0.0.2/32"},
			"laptop": {Address: "10.0.0.3/32"},
		},
	}
	_, count := m.ServerPeer()
	if count != 0 {
		t.Errorf("expected 0 server peers, got %d", count)
	}
}

func TestManifest_ServerPeerCount_Multiple(t *testing.T) {
	m := Manifest{
		Version: 1,
		Peers: map[string]PeerManifest{
			"server1": {
				Address: "10.0.0.1/24", Endpoint: "s1.example.com:51820",
				ListenPort: 51820,
			},
			"server2": {
				Address: "10.0.1.1/24", Endpoint: "s2.example.com:51820",
				ListenPort: 51820,
			},
		},
	}
	_, count := m.ServerPeer()
	if count != 2 {
		t.Errorf("expected 2 server peers, got %d", count)
	}
}

func TestManifest_PeerNames(t *testing.T) {
	m := Manifest{
		Version: 1,
		Peers: map[string]PeerManifest{
			"charlie": {Address: "10.0.0.3/32"},
			"alice":   {Address: "10.0.0.1/24", Endpoint: "vpn.example.com:51820", ListenPort: 51820},
			"bob":     {Address: "10.0.0.2/32"},
		},
	}
	names := m.PeerNames()
	expected := []string{"alice", "bob", "charlie"}
	if len(names) != len(expected) {
		t.Fatalf("PeerNames len = %d, want %d", len(names), len(expected))
	}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("PeerNames[%d] = %q, want %q", i, name, expected[i])
		}
	}
}

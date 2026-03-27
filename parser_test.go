package amnezigo

import (
	"strings"
	"testing"
	"time"
)

func TestParseServerConfig(t *testing.T) {
	input := `
[Interface]
PrivateKey = abc123
PublicKey = xyz789
Address = 10.8.0.1/24
ListenPort = 55424

[Peer]
#_Role = client
#_Name = client1
PublicKey = peerpub1
AllowedIPs = 10.8.0.2/32
`

	cfg, err := ParseServerConfig(strings.NewReader(input))

	if err != nil {
		t.Fatalf("ParseServerConfig failed: %v", err)
	}

	// Verify Interface section
	if cfg.Interface.PrivateKey != "abc123" {
		t.Errorf("Expected PrivateKey 'abc123', got '%s'", cfg.Interface.PrivateKey)
	}
	if cfg.Interface.PublicKey != "xyz789" {
		t.Errorf("Expected PublicKey 'xyz789', got '%s'", cfg.Interface.PublicKey)
	}
	if cfg.Interface.Address != "10.8.0.1/24" {
		t.Errorf("Expected Address '10.8.0.1/24', got '%s'", cfg.Interface.Address)
	}
	if cfg.Interface.ListenPort != 55424 {
		t.Errorf("Expected ListenPort 55424, got %d", cfg.Interface.ListenPort)
	}

	// Verify Peer section
	if len(cfg.Clients) != 1 {
		t.Fatalf("Expected 1 peer, got %d", len(cfg.Clients))
	}
	if cfg.Clients[0].Name != "client1" {
		t.Errorf("Expected Peer Name 'client1', got '%s'", cfg.Clients[0].Name)
	}
	if cfg.Clients[0].PublicKey != "peerpub1" {
		t.Errorf("Expected Peer PublicKey 'peerpub1', got '%s'", cfg.Clients[0].PublicKey)
	}
	if cfg.Clients[0].AllowedIPs != "10.8.0.2/32" {
		t.Errorf("Expected Peer AllowedIPs '10.8.0.2/32', got '%s'", cfg.Clients[0].AllowedIPs)
	}
}

func TestParseObfuscationParams(t *testing.T) {
	input := `
[Interface]
Jc = 3
Jmin = 64
Jmax = 512
S1 = 1
S2 = 2
S3 = 3
S4 = 4
H1 = 1020325451-1020325451
H2 = 2020325452-2020325452
H3 = 3020325453-3020325453
H4 = 4020325454-4020325454
`

	cfg, err := ParseServerConfig(strings.NewReader(input))

	if err != nil {
		t.Fatalf("ParseServerConfig failed: %v", err)
	}

	// Verify obfuscation params
	if cfg.Obfuscation.Jc != 3 {
		t.Errorf("Expected Jc 3, got %d", cfg.Obfuscation.Jc)
	}
	if cfg.Obfuscation.Jmin != 64 {
		t.Errorf("Expected Jmin 64, got %d", cfg.Obfuscation.Jmin)
	}
	if cfg.Obfuscation.Jmax != 512 {
		t.Errorf("Expected Jmax 512, got %d", cfg.Obfuscation.Jmax)
	}
	if cfg.Obfuscation.S1 != 1 {
		t.Errorf("Expected S1 1, got %d", cfg.Obfuscation.S1)
	}
	if cfg.Obfuscation.H1.Min != 1020325451 || cfg.Obfuscation.H1.Max != 1020325451 {
		t.Errorf("Expected H1 {1020325451,1020325451}, got {%d,%d}", cfg.Obfuscation.H1.Min, cfg.Obfuscation.H1.Max)
	}
	// I1-I5 are client-only fields, not in ServerObfuscationConfig
}

func TestParseMultiplePeers(t *testing.T) {
	input := `
[Interface]
PrivateKey = abc123
PublicKey = xyz789
Address = 10.8.0.1/24
ListenPort = 55424

[Peer]
#_Role = client
#_Name = client1
PublicKey = peerpub1
AllowedIPs = 10.8.0.2/32
#_GenKeyTime = 2024-01-15T10:30:00Z

[Peer]
#_Role = client
#_Name = client2
PublicKey = peerpub2
AllowedIPs = 10.8.0.3/32
#_GenKeyTime = 2024-01-16T11:45:00Z
`

	cfg, err := ParseServerConfig(strings.NewReader(input))

	if err != nil {
		t.Fatalf("ParseServerConfig failed: %v", err)
	}

	// Verify two peers
	if len(cfg.Clients) != 2 {
		t.Fatalf("Expected 2 peers, got %d", len(cfg.Clients))
	}

	// First peer
	if cfg.Clients[0].Name != "client1" {
		t.Errorf("Expected Peer[0] Name 'client1', got '%s'", cfg.Clients[0].Name)
	}
	expectedTime1, _ := time.Parse(time.RFC3339, "2024-01-15T10:30:00Z")
	if !cfg.Clients[0].CreatedAt.Equal(expectedTime1) {
		t.Errorf("Expected Peer[0] CreatedAt %v, got %v", expectedTime1, cfg.Clients[0].CreatedAt)
	}

	// Second peer
	if cfg.Clients[1].Name != "client2" {
		t.Errorf("Expected Peer[1] Name 'client2', got '%s'", cfg.Clients[1].Name)
	}
	expectedTime2, _ := time.Parse(time.RFC3339, "2024-01-16T11:45:00Z")
	if !cfg.Clients[1].CreatedAt.Equal(expectedTime2) {
		t.Errorf("Expected Peer[1] CreatedAt %v, got %v", expectedTime2, cfg.Clients[1].CreatedAt)
	}
}

func TestParsePostScripts(t *testing.T) {
	input := `
[Interface]
PrivateKey = abc123
PublicKey = xyz789
Address = 10.8.0.1/24
ListenPort = 55424
PostUp = iptables -A FORWARD -i wg0 -j ACCEPT
PostDown = iptables -D FORWARD -i wg0 -j ACCEPT
MTU = 1420

[Peer]
#_Role = client
PublicKey = peerpub1
AllowedIPs = 10.8.0.2/32
`

	cfg, err := ParseServerConfig(strings.NewReader(input))

	if err != nil {
		t.Fatalf("ParseServerConfig failed: %v", err)
	}

	if cfg.Interface.PostUp != "iptables -A FORWARD -i wg0 -j ACCEPT" {
		t.Errorf("Expected PostUp 'iptables -A FORWARD -i wg0 -j ACCEPT', got '%s'", cfg.Interface.PostUp)
	}
	if cfg.Interface.PostDown != "iptables -D FORWARD -i wg0 -j ACCEPT" {
		t.Errorf("Expected PostDown 'iptables -D FORWARD -i wg0 -j ACCEPT', got '%s'", cfg.Interface.PostDown)
	}
	if cfg.Interface.MTU != 1420 {
		t.Errorf("Expected MTU 1420, got %d", cfg.Interface.MTU)
	}
}

func TestParsePeerPresharedKey(t *testing.T) {
	input := `
[Interface]
PrivateKey = abc123
PublicKey = xyz789
Address = 10.8.0.1/24
ListenPort = 55424

[Peer]
#_Role = client
PublicKey = peerpub1
PresharedKey = psk123
AllowedIPs = 10.8.0.2/32
`

	cfg, err := ParseServerConfig(strings.NewReader(input))

	if err != nil {
		t.Fatalf("ParseServerConfig failed: %v", err)
	}

	if len(cfg.Clients) != 1 {
		t.Fatalf("Expected 1 peer, got %d", len(cfg.Clients))
	}
	if cfg.Clients[0].PublicKey != "peerpub1" {
		t.Errorf("Expected Peer PublicKey 'peerpub1', got '%s'", cfg.Clients[0].PublicKey)
	}
	if cfg.Clients[0].PresharedKey != "psk123" {
		t.Errorf("Expected Peer PresharedKey 'psk123', got '%s'", cfg.Clients[0].PresharedKey)
	}
	if cfg.Clients[0].AllowedIPs != "10.8.0.2/32" {
		t.Errorf("Expected Peer AllowedIPs '10.8.0.2/32', got '%s'", cfg.Clients[0].AllowedIPs)
	}
}

func TestParsePeerRejectsMissingRole(t *testing.T) {
	input := `
[Interface]
PrivateKey = abc123
PublicKey = xyz789
Address = 10.8.0.1/24
ListenPort = 55424

[Peer]
#_Name = client1
PublicKey = peerpub1
AllowedIPs = 10.8.0.2/32
`
	_, err := ParseServerConfig(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for peer without #_Role, got nil")
	}
	if !strings.Contains(err.Error(), "role") {
		t.Errorf("expected error about missing role, got: %v", err)
	}
}

func TestParsePeerRejectsInvalidRole(t *testing.T) {
	input := `
[Interface]
PrivateKey = abc123
Address = 10.8.0.1/24
ListenPort = 55424

[Peer]
#_Role = server
#_Name = client1
PublicKey = peerpub1
AllowedIPs = 10.8.0.2/32
`
	_, err := ParseServerConfig(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for invalid role, got nil")
	}
}

func TestParseSplitsClientsAndEdges(t *testing.T) {
	input := `
[Interface]
PrivateKey = abc123
Address = 10.8.0.1/24
ListenPort = 55424

[Peer]
#_Role = client
#_Name = user1
PublicKey = pub1
AllowedIPs = 10.8.0.2/32

[Peer]
#_Role = edge
#_Name = edge1
PublicKey = pub2
AllowedIPs = 10.8.0.3/32

[Peer]
#_Role = client
#_Name = user2
PublicKey = pub3
AllowedIPs = 10.8.0.4/32
`
	cfg, err := ParseServerConfig(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseServerConfig failed: %v", err)
	}
	if len(cfg.Clients) != 2 {
		t.Fatalf("expected 2 clients, got %d", len(cfg.Clients))
	}
	if len(cfg.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(cfg.Edges))
	}
	if cfg.Clients[0].Name != "user1" {
		t.Errorf("expected client name 'user1', got '%s'", cfg.Clients[0].Name)
	}
	if cfg.Clients[0].Role != RoleClient {
		t.Errorf("expected role '%s', got '%s'", RoleClient, cfg.Clients[0].Role)
	}
	if cfg.Edges[0].Name != "edge1" {
		t.Errorf("expected edge name 'edge1', got '%s'", cfg.Edges[0].Name)
	}
	if cfg.Edges[0].Role != RoleEdge {
		t.Errorf("expected role '%s', got '%s'", RoleEdge, cfg.Edges[0].Role)
	}
}

package config

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
	if len(cfg.Peers) != 1 {
		t.Fatalf("Expected 1 peer, got %d", len(cfg.Peers))
	}
	if cfg.Peers[0].Name != "client1" {
		t.Errorf("Expected Peer Name 'client1', got '%s'", cfg.Peers[0].Name)
	}
	if cfg.Peers[0].PublicKey != "peerpub1" {
		t.Errorf("Expected Peer PublicKey 'peerpub1', got '%s'", cfg.Peers[0].PublicKey)
	}
	if cfg.Peers[0].AllowedIPs != "10.8.0.2/32" {
		t.Errorf("Expected Peer AllowedIPs '10.8.0.2/32', got '%s'", cfg.Peers[0].AllowedIPs)
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
H1 = 1020325451
H2 = 2020325452
H3 = 3020325453
H4 = 4020325454
I1 = <b 0xc00000000108><r 8>
I2 = <b 0xc00000000108><r 9>
I3 = <b 0xc00000000108><r 10>
I4 = <b 0xc00000000108><r 11>
I5 = <b 0xc00000000108><r 12>
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
	if cfg.Obfuscation.H1 != 1020325451 {
		t.Errorf("Expected H1 1020325451, got %d", cfg.Obfuscation.H1)
	}
	if cfg.Obfuscation.I1 != "<b 0xc00000000108><r 8>" {
		t.Errorf("Expected I1 '<b 0xc00000000108><r 8>', got '%s'", cfg.Obfuscation.I1)
	}
}

func TestParseMultiplePeers(t *testing.T) {
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
#_GenKeyTime = 2024-01-15T10:30:00Z

[Peer]
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
	if len(cfg.Peers) != 2 {
		t.Fatalf("Expected 2 peers, got %d", len(cfg.Peers))
	}

	// First peer
	if cfg.Peers[0].Name != "client1" {
		t.Errorf("Expected Peer[0] Name 'client1', got '%s'", cfg.Peers[0].Name)
	}
	expectedTime1, _ := time.Parse(time.RFC3339, "2024-01-15T10:30:00Z")
	if !cfg.Peers[0].CreatedAt.Equal(expectedTime1) {
		t.Errorf("Expected Peer[0] CreatedAt %v, got %v", expectedTime1, cfg.Peers[0].CreatedAt)
	}

	// Second peer
	if cfg.Peers[1].Name != "client2" {
		t.Errorf("Expected Peer[1] Name 'client2', got '%s'", cfg.Peers[1].Name)
	}
	expectedTime2, _ := time.Parse(time.RFC3339, "2024-01-16T11:45:00Z")
	if !cfg.Peers[1].CreatedAt.Equal(expectedTime2) {
		t.Errorf("Expected Peer[1] CreatedAt %v, got %v", expectedTime2, cfg.Peers[1].CreatedAt)
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
PublicKey = peerpub1
PresharedKey = psk123
AllowedIPs = 10.8.0.2/32
`

	cfg, err := ParseServerConfig(strings.NewReader(input))

	if err != nil {
		t.Fatalf("ParseServerConfig failed: %v", err)
	}

	if len(cfg.Peers) != 1 {
		t.Fatalf("Expected 1 peer, got %d", len(cfg.Peers))
	}
	if cfg.Peers[0].PublicKey != "peerpub1" {
		t.Errorf("Expected Peer PublicKey 'peerpub1', got '%s'", cfg.Peers[0].PublicKey)
	}
	if cfg.Peers[0].PresharedKey != "psk123" {
		t.Errorf("Expected Peer PresharedKey 'psk123', got '%s'", cfg.Peers[0].PresharedKey)
	}
	if cfg.Peers[0].AllowedIPs != "10.8.0.2/32" {
		t.Errorf("Expected Peer AllowedIPs '10.8.0.2/32', got '%s'", cfg.Peers[0].AllowedIPs)
	}
}

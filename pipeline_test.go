package amnezigo

import (
	"os"
	"strings"
	"testing"
)

func TestGenerateOptions_ZeroValue(t *testing.T) {
	var opts GenerateOptions
	if opts.ProjectDir != "" {
		t.Error("expected empty ProjectDir")
	}
	if opts.FullReset {
		t.Error("expected FullReset false")
	}
	if opts.PeerFilter != nil {
		t.Error("expected nil PeerFilter")
	}
}

func TestFileOutput_ZeroValue(t *testing.T) {
	var f FileOutput
	if f.RelPath != "" {
		t.Error("expected empty RelPath")
	}
	if f.Content != nil {
		t.Error("expected nil Content")
	}
}

func TestResolveObfuscation_Empty(t *testing.T) {
	obf := ObfuscationManifest{}
	result, err := resolveObfuscation(obf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All S fields should be non-zero
	if result.S1 == 0 {
		t.Error("expected S1 != 0")
	}
	if result.S2 == 0 {
		t.Error("expected S2 != 0")
	}
	if result.S3 == 0 {
		t.Error("expected S3 != 0")
	}
	if result.S4 == 0 {
		t.Error("expected S4 != 0")
	}

	// All J fields should be non-zero
	if result.Jc == 0 {
		t.Error("expected Jc != 0")
	}
	if result.Jmin == 0 {
		t.Error("expected Jmin != 0")
	}
	if result.Jmax == 0 {
		t.Error("expected Jmax != 0")
	}

	// All H fields should be non-zero (check if not HeaderRange{0, 0})
	if result.H1.Min == 0 && result.H1.Max == 0 {
		t.Error("expected H1 to be non-zero")
	}
	if result.H2.Min == 0 && result.H2.Max == 0 {
		t.Error("expected H2 to be non-zero")
	}
	if result.H3.Min == 0 && result.H3.Max == 0 {
		t.Error("expected H3 to be non-zero")
	}
	if result.H4.Min == 0 && result.H4.Max == 0 {
		t.Error("expected H4 to be non-zero")
	}
}

func TestResolveObfuscation_ExplicitValues(t *testing.T) {
	s1, s2, s3, s4 := 50, 100, 150, 200
	obf := ObfuscationManifest{
		S1: &s1,
		S2: &s2,
		S3: &s3,
		S4: &s4,
	}
	result, err := resolveObfuscation(obf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Explicit S values should be preserved
	if result.S1 != 50 {
		t.Errorf("expected S1 = 50, got %d", result.S1)
	}
	if result.S2 != 100 {
		t.Errorf("expected S2 = 100, got %d", result.S2)
	}
	if result.S3 != 150 {
		t.Errorf("expected S3 = 150, got %d", result.S3)
	}
	if result.S4 != 200 {
		t.Errorf("expected S4 = 200, got %d", result.S4)
	}

	// J fields should be generated
	if result.Jc == 0 {
		t.Error("expected Jc != 0")
	}
	if result.Jmin == 0 {
		t.Error("expected Jmin != 0")
	}
	if result.Jmax == 0 {
		t.Error("expected Jmax != 0")
	}

	// H fields should be generated
	if result.H1.Min == 0 && result.H1.Max == 0 {
		t.Error("expected H1 to be non-zero")
	}
	if result.H2.Min == 0 && result.H2.Max == 0 {
		t.Error("expected H2 to be non-zero")
	}
	if result.H3.Min == 0 && result.H3.Max == 0 {
		t.Error("expected H3 to be non-zero")
	}
	if result.H4.Min == 0 && result.H4.Max == 0 {
		t.Error("expected H4 to be non-zero")
	}
}

func TestResolveObfuscation_PartialExplicit(t *testing.T) {
	s1 := 50
	obf := ObfuscationManifest{
		S1: &s1,
	}
	result, err := resolveObfuscation(obf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// S1 should be preserved
	if result.S1 != 50 {
		t.Errorf("expected S1 = 50, got %d", result.S1)
	}

	// Other S values should be non-zero
	if result.S2 == 0 {
		t.Error("expected S2 != 0")
	}
	if result.S3 == 0 {
		t.Error("expected S3 != 0")
	}
	if result.S4 == 0 {
		t.Error("expected S4 != 0")
	}

	// All S values should be distinct
	sValues := []int{result.S1, result.S2, result.S3, result.S4}
	for i := range sValues {
		for j := i + 1; j < len(sValues); j++ {
			if sValues[i] == sValues[j] {
				t.Errorf("S values should be distinct: S[%d]=%d equals S[%d]=%d", i, sValues[i], j, sValues[j])
			}
		}
	}
}

func TestResolvePeerCredentials_FreshGeneration(t *testing.T) {
	peers := map[string]PeerManifest{
		"server":  {Address: "10.0.0.1/24", ListenPort: 51820, Endpoint: "example.com:51820"},
		"client1": {Address: "10.0.0.2/24"},
		"client2": {Address: "10.0.0.3/24"},
	}
	persisted := EmptyCredentials()
	serverName := "server"
	fullReset := false

	result := resolvePeerCredentials(peers, persisted, serverName, fullReset)

	// Check server credentials
	serverCreds := result[serverName]
	if serverCreds.PrivateKey == "" {
		t.Error("expected server PrivateKey to be non-empty")
	}
	if serverCreds.PublicKey == "" {
		t.Error("expected server PublicKey to be non-empty")
	}
	if serverCreds.PresharedKey != "" {
		t.Error("expected server PresharedKey to be empty (servers don't get PSK)")
	}

	// Check client credentials - all should have keys
	for _, name := range []string{"client1", "client2"} {
		creds := result[name]
		if creds.PrivateKey == "" {
			t.Errorf("expected %s PrivateKey to be non-empty", name)
		}
		if creds.PublicKey == "" {
			t.Errorf("expected %s PublicKey to be non-empty", name)
		}
		if creds.PresharedKey == "" {
			t.Errorf("expected %s PresharedKey to be non-empty (clients need PSK)", name)
		}
	}

	// Verify PublicKey is derived from PrivateKey
	for name, creds := range result {
		expectedPub := DerivePublicKey(creds.PrivateKey)
		if creds.PublicKey != expectedPub {
			t.Errorf("%s: PublicKey mismatch with derived key from PrivateKey", name)
		}
	}
}

func TestResolvePeerCredentials_ReuseExisting(t *testing.T) {
	// Generate known keys for server and client1
	serverPriv, serverPub := GenerateKeyPair()
	client1Priv, client1Pub := GenerateKeyPair()
	client1PSK := GeneratePSK()

	persisted := &PersistedCredentials{
		Server: PeerCredentials{
			PrivateKey: serverPriv,
			PublicKey:  serverPub,
		},
		Peers: map[string]PeerCredentials{
			"client1": {
				PrivateKey:   client1Priv,
				PublicKey:    client1Pub,
				PresharedKey: client1PSK,
			},
		},
	}

	peers := map[string]PeerManifest{
		"server":  {Address: "10.0.0.1/24", ListenPort: 51820, Endpoint: "example.com:51820"},
		"client1": {Address: "10.0.0.2/24"},
		"client2": {Address: "10.0.0.3/24"},
	}
	serverName := "server"
	fullReset := false

	result := resolvePeerCredentials(peers, persisted, serverName, fullReset)

	// Server keys should be preserved
	if result[serverName].PrivateKey != serverPriv {
		t.Error("expected server PrivateKey to be preserved from persisted")
	}
	if result[serverName].PublicKey != serverPub {
		t.Error("expected server PublicKey to be preserved from persisted")
	}

	// Client1 keys should be preserved
	if result["client1"].PrivateKey != client1Priv {
		t.Error("expected client1 PrivateKey to be preserved from persisted")
	}
	if result["client1"].PublicKey != client1Pub {
		t.Error("expected client1 PublicKey to be preserved from persisted")
	}
	if result["client1"].PresharedKey != client1PSK {
		t.Error("expected client1 PresharedKey to be preserved from persisted")
	}

	// Client2 should get freshly generated keys
	client2Creds := result["client2"]
	if client2Creds.PrivateKey == "" {
		t.Error("expected client2 PrivateKey to be generated")
	}
	if client2Creds.PublicKey == "" {
		t.Error("expected client2 PublicKey to be generated")
	}
	if client2Creds.PresharedKey == "" {
		t.Error("expected client2 PresharedKey to be generated")
	}
	// Verify client2 keys are different from persisted ones
	if client2Creds.PrivateKey == client1Priv {
		t.Error("expected client2 PrivateKey to be different from client1")
	}
	if client2Creds.PresharedKey == client1PSK {
		t.Error("expected client2 PresharedKey to be different from client1")
	}
}

func TestResolvePeerCredentials_FullReset(t *testing.T) {
	// Generate known keys for all peers
	serverPriv, serverPub := GenerateKeyPair()
	client1Priv, client1Pub := GenerateKeyPair()
	client1PSK := GeneratePSK()
	client2Priv, client2Pub := GenerateKeyPair()
	client2PSK := GeneratePSK()

	persisted := &PersistedCredentials{
		Server: PeerCredentials{
			PrivateKey: serverPriv,
			PublicKey:  serverPub,
		},
		Peers: map[string]PeerCredentials{
			"client1": {
				PrivateKey:   client1Priv,
				PublicKey:    client1Pub,
				PresharedKey: client1PSK,
			},
			"client2": {
				PrivateKey:   client2Priv,
				PublicKey:    client2Pub,
				PresharedKey: client2PSK,
			},
		},
	}

	peers := map[string]PeerManifest{
		"server":  {Address: "10.0.0.1/24", ListenPort: 51820, Endpoint: "example.com:51820"},
		"client1": {Address: "10.0.0.2/24"},
		"client2": {Address: "10.0.0.3/24"},
	}
	serverName := "server"
	fullReset := true

	result := resolvePeerCredentials(peers, persisted, serverName, fullReset)

	// ALL keys should be different from persisted ones
	if result[serverName].PrivateKey == serverPriv {
		t.Error("expected server PrivateKey to be regenerated (fullReset)")
	}
	if result["client1"].PrivateKey == client1Priv {
		t.Error("expected client1 PrivateKey to be regenerated (fullReset)")
	}
	if result["client2"].PrivateKey == client2Priv {
		t.Error("expected client2 PrivateKey to be regenerated (fullReset)")
	}

	// All peers should still have valid keys
	for name, creds := range result {
		if creds.PrivateKey == "" {
			t.Errorf("expected %s PrivateKey to be non-empty after fullReset", name)
		}
		if creds.PublicKey == "" {
			t.Errorf("expected %s PublicKey to be non-empty after fullReset", name)
		}
		if name != serverName && creds.PresharedKey == "" {
			t.Errorf("expected %s PresharedKey to be non-empty after fullReset", name)
		}
		if name == serverName && creds.PresharedKey != "" {
			t.Error("expected server PresharedKey to be empty after fullReset")
		}
	}
}

func TestBuildServerConfig(t *testing.T) {
	// Create minimal manifest with server and client peers
	manifest := Manifest{
		Peers: map[string]PeerManifest{
			"server": {
				Address:    "10.0.0.1/24",
				Endpoint:   "vpn.example.com:51820",
				ListenPort: 51820,
				TunName:    "awg0",
				MainIface:  "eth0",
			},
			"phone": {
				Address: "10.0.0.2/32",
			},
		},
		Network: NetworkConfig{
			MTU: 1420,
			DNS: []string{"1.1.1.1", "8.8.8.8"},
		},
	}

	// Create ServerObfuscationConfig with realistic values
	obf := ServerObfuscationConfig{
		S1: 50, S2: 100, S3: 150, S4: 200,
		H1: HeaderRange{Min: 1, Max: 10},
		H2: HeaderRange{Min: 11, Max: 20},
		H3: HeaderRange{Min: 21, Max: 30},
		H4: HeaderRange{Min: 31, Max: 40},
		Jc: 4, Jmin: 50, Jmax: 1000,
	}

	// Create credentials map with known keys
	serverPriv, serverPub := GenerateKeyPair()
	phonePriv, phonePub := GenerateKeyPair()
	phonePSK := GeneratePSK()

	creds := map[string]PeerCredentials{
		"server": {
			PrivateKey: serverPriv,
			PublicKey:  serverPub,
		},
		"phone": {
			PrivateKey:   phonePriv,
			PublicKey:    phonePub,
			PresharedKey: phonePSK,
		},
	}

	// Call buildServerConfig
	output, err := buildServerConfig(manifest, "server", obf, creds)
	if err != nil {
		t.Fatalf("buildServerConfig failed: %v", err)
	}

	// Verify output contains server address
	if !contains(output, "Address = 10.0.0.1/24") {
		t.Error("expected server address in output")
	}

	// Verify output contains listen port
	if !contains(output, "ListenPort = 51820") {
		t.Error("expected listen port in output")
	}

	// Verify output contains private key
	if !contains(output, "PrivateKey = "+serverPriv) {
		t.Error("expected server private key in output")
	}

	// Verify output contains PostUp iptables rules (IPv4)
	if !contains(output, "PostUp = ") {
		t.Error("expected PostUp in output")
	}
	if !contains(output, "iptables -A INPUT -i awg0 -j ACCEPT") {
		t.Error("expected IPv4 INPUT rule in PostUp")
	}

	// Verify output contains PostDown iptables rules
	if !contains(output, "PostDown = ") {
		t.Error("expected PostDown in output")
	}

	// Verify phone peer is present
	if !contains(output, "PublicKey = "+phonePub) {
		t.Error("expected phone public key in output")
	}
	if !contains(output, "PresharedKey = "+phonePSK) {
		t.Error("expected phone preshared key in output")
	}
	if !contains(output, "AllowedIPs = 10.0.0.2/32") {
		t.Error("expected phone allowed IPs in output")
	}

	// Verify obfuscation parameters
	if !contains(output, "Jc = 4") {
		t.Error("expected Jc in output")
	}
	if !contains(output, "S1 = 50") {
		t.Error("expected S1 in output")
	}
	// Verify iptables rules use the network address (10.0.0.0/24), not the host address.
	// Regression: the old extractSubnet returned the host 10.0.0.1/24 here.
	if !contains(output, "-s 10.0.0.0/24") {
		t.Error("expected iptables rules to reference network address 10.0.0.0/24")
	}
	if contains(output, "-s 10.0.0.1/24") {
		t.Error("iptables rules must not reference host address 10.0.0.1/24")
	}
}

// TestBuildServerConfig_IptablesUsesNetworkAddress is a regression test for the
// third-octet bug: extractSubnet previously replaced parts[2] (the third octet)
// instead of the last octet, so a server at 10.0.50.1/24 produced iptables rules
// referencing the wrong network 10.0.0.1/24. ExtractSubnet (helpers.go) computes
// the correct network address via net.ParseCIDR.
func TestBuildServerConfig_IptablesUsesNetworkAddress(t *testing.T) {
	manifest := Manifest{
		Peers: map[string]PeerManifest{
			"server": {
				Address:    "10.0.50.1/24",
				Endpoint:   "vpn.example.com:51820",
				ListenPort: 51820,
				TunName:    "awg0",
				MainIface:  "eth0",
			},
			"phone": {
				Address: "10.0.50.2/32",
			},
		},
		Network: NetworkConfig{
			MTU: 1420,
		},
	}

	obf := ServerObfuscationConfig{
		S1: 50, S2: 100, S3: 150, S4: 200,
		H1: HeaderRange{Min: 1, Max: 10},
		H2: HeaderRange{Min: 11, Max: 20},
		H3: HeaderRange{Min: 21, Max: 30},
		H4: HeaderRange{Min: 31, Max: 40},
		Jc: 4, Jmin: 50, Jmax: 1000,
	}

	serverPriv, serverPub := GenerateKeyPair()
	phonePriv, phonePub := GenerateKeyPair()
	phonePSK := GeneratePSK()
	creds := map[string]PeerCredentials{
		"server": {PrivateKey: serverPriv, PublicKey: serverPub},
		"phone":  {PrivateKey: phonePriv, PublicKey: phonePub, PresharedKey: phonePSK},
	}

	output, err := buildServerConfig(manifest, "server", obf, creds)
	if err != nil {
		t.Fatalf("buildServerConfig failed: %v", err)
	}

	// Correct network address for 10.0.50.1/24 is 10.0.50.0/24.
	if !contains(output, "-s 10.0.50.0/24") {
		t.Error("expected iptables rules to reference network address 10.0.50.0/24")
	}
	if !contains(output, "-d 10.0.50.0/24") {
		t.Error("expected iptables rules to reference network address 10.0.50.0/24 in -d rule")
	}

	// Must NOT contain the buggy output (10.0.0.1/24) nor the host address.
	if contains(output, "10.0.0.1/24") {
		t.Error("iptables rules must not reference wrong subnet 10.0.0.1/24 (third-octet bug)")
	}
	if contains(output, "-s 10.0.50.1/24") {
		t.Error("iptables rules must not reference host address 10.0.50.1/24")
	}
}

func TestBuildClientConfig(t *testing.T) {
	// Create minimal manifest with server and client peers
	manifest := Manifest{
		Peers: map[string]PeerManifest{
			"server": {
				Address:    "10.0.0.1/24",
				Endpoint:   "vpn.example.com:51820",
				ListenPort: 51820,
				TunName:    "awg0",
				MainIface:  "eth0",
			},
			"phone": {
				Address:  "10.0.0.2/32",
				Protocol: "quic",
			},
		},
		Network: NetworkConfig{
			MTU: 1420,
			DNS: []string{"1.1.1.1", "8.8.8.8"},
		},
	}

	// Create ServerObfuscationConfig
	obf := ServerObfuscationConfig{
		S1: 50, S2: 100, S3: 150, S4: 200,
		H1: HeaderRange{Min: 1, Max: 10},
		H2: HeaderRange{Min: 11, Max: 20},
		H3: HeaderRange{Min: 21, Max: 30},
		H4: HeaderRange{Min: 31, Max: 40},
		Jc: 4, Jmin: 50, Jmax: 1000,
	}

	// Create credentials map
	serverPriv, serverPub := GenerateKeyPair()
	phonePriv, phonePub := GenerateKeyPair()
	phonePSK := GeneratePSK()

	creds := map[string]PeerCredentials{
		"server": {
			PrivateKey: serverPriv,
			PublicKey:  serverPub,
		},
		"phone": {
			PrivateKey:   phonePriv,
			PublicKey:    phonePub,
			PresharedKey: phonePSK,
		},
	}

	// Call buildClientConfig
	output, err := buildClientConfig(manifest, "phone", "server", obf, creds)
	if err != nil {
		t.Fatalf("buildClientConfig failed: %v", err)
	}

	// Verify output contains phone's address
	if !contains(output, "Address = 10.0.0.2/32") {
		t.Error("expected phone address in output")
	}

	// Verify output contains phone's private key
	if !contains(output, "PrivateKey = "+phonePriv) {
		t.Error("expected phone private key in output")
	}

	// Verify output contains DNS
	if !contains(output, "DNS = 1.1.1.1, 8.8.8.8") {
		t.Error("expected DNS in output")
	}

	// Verify output contains server's public key
	if !contains(output, "PublicKey = "+serverPub) {
		t.Error("expected server public key in output")
	}

	// Verify output contains server's endpoint
	if !contains(output, "Endpoint = vpn.example.com:51820") {
		t.Error("expected server endpoint in output")
	}

	// Verify output contains preshared key
	if !contains(output, "PresharedKey = "+phonePSK) {
		t.Error("expected preshared key in output")
	}

	// Verify output contains allowed IPs (0.0.0.0/0, ::/0)
	if !contains(output, "AllowedIPs = 0.0.0.0/0, ::/0") {
		t.Error("expected allowed IPs in output")
	}

	// Verify obfuscation parameters
	if !contains(output, "Jc = 4") {
		t.Error("expected Jc in output")
	}
	if !contains(output, "S1 = 50") {
		t.Error("expected S1 in output")
	}

	// Verify I1-I5 are present (custom packet strings)
	if !contains(output, "I1 = ") {
		t.Error("expected I1 in output")
	}
}

// Helper function to check if byte slice contains a string.
func contains(data []byte, substr string) bool {
	return string(data) != "" && indexOf(data, substr) >= 0
}

// Helper function to find substring index in byte slice.
func indexOf(data []byte, substr string) int {
	str := string(data)
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func TestGenerate_MinimalManifest(t *testing.T) {
	manifest := Manifest{
		Version: 1,
		Network: NetworkConfig{
			MTU: 1280,
			DNS: []string{"1.1.1.1"},
		},
		Obfuscation: ObfuscationManifest{},
		Peers: map[string]PeerManifest{
			"server": {
				Address:    "10.0.0.1/24",
				Endpoint:   "vpn.example.com:51820",
				ListenPort: 51820,
			},
			"phone": {
				Address: "10.0.0.2/32",
			},
		},
	}

	outputDir := t.TempDir()
	result, err := Generate(manifest, GenerateOptions{OutputDir: outputDir})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify: result has 2 files
	if len(result.Files) != 2 {
		t.Errorf("expected 2 files, got %d", len(result.Files))
	}

	// Verify: result.ServerPeer == "server"
	if result.ServerPeer != "server" {
		t.Errorf("expected ServerPeer = 'server', got '%s'", result.ServerPeer)
	}

	// Verify: result.ClientPeers contains "phone"
	if len(result.ClientPeers) != 1 {
		t.Errorf("expected 1 client peer, got %d", len(result.ClientPeers))
	}
	if len(result.ClientPeers) > 0 && result.ClientPeers[0] != "phone" {
		t.Errorf("expected ClientPeers[0] = 'phone', got '%s'", result.ClientPeers[0])
	}

	// Verify: each FileOutput has non-empty Content
	for _, f := range result.Files {
		if len(f.Content) == 0 {
			t.Errorf("file %s has empty content", f.RelPath)
		}
	}

	// Verify: files were written to disk
	for _, f := range result.Files {
		fullPath := outputDir + "/" + f.RelPath
		if _, err := os.Stat(fullPath); err != nil {
			t.Errorf("file %s was not written to disk: %v", f.RelPath, err)
		}
	}
}

func TestGenerate_DryRun(t *testing.T) {
	manifest := Manifest{
		Version: 1,
		Network: NetworkConfig{
			MTU: 1280,
			DNS: []string{"1.1.1.1"},
		},
		Obfuscation: ObfuscationManifest{},
		Peers: map[string]PeerManifest{
			"server": {
				Address:    "10.0.0.1/24",
				Endpoint:   "vpn.example.com:51820",
				ListenPort: 51820,
			},
			"phone": {
				Address: "10.0.0.2/32",
			},
		},
	}

	outputDir := t.TempDir()
	result, err := Generate(manifest, GenerateOptions{
		OutputDir: outputDir,
		DryRun:    true,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify: result has files (computed in memory)
	if len(result.Files) != 2 {
		t.Errorf("expected 2 files in dry run, got %d", len(result.Files))
	}

	// Verify: NO files written to disk
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		t.Fatalf("failed to read output dir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected empty output dir in dry run, found %d entries", len(entries))
	}
}

func TestGenerate_PeerFilter(t *testing.T) {
	manifest := Manifest{
		Version: 1,
		Network: NetworkConfig{
			MTU: 1280,
			DNS: []string{"1.1.1.1"},
		},
		Obfuscation: ObfuscationManifest{},
		Peers: map[string]PeerManifest{
			"server": {
				Address:    "10.0.0.1/24",
				Endpoint:   "vpn.example.com:51820",
				ListenPort: 51820,
			},
			"phone": {
				Address: "10.0.0.2/32",
			},
			"laptop": {
				Address: "10.0.0.3/32",
			},
		},
	}

	outputDir := t.TempDir()
	result, err := Generate(manifest, GenerateOptions{
		OutputDir:  outputDir,
		PeerFilter: []string{"phone"},
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify: result has 2 files (server + phone, not laptop)
	if len(result.Files) != 2 {
		t.Errorf("expected 2 files with peer filter, got %d", len(result.Files))
	}

	// Verify: ClientPeers contains only "phone"
	if len(result.ClientPeers) != 1 {
		t.Errorf("expected 1 client peer with filter, got %d", len(result.ClientPeers))
	}
	if len(result.ClientPeers) > 0 && result.ClientPeers[0] != "phone" {
		t.Errorf("expected ClientPeers[0] = 'phone', got '%s'", result.ClientPeers[0])
	}

	// Verify: server is always included regardless of filter
	serverFound := false
	for _, f := range result.Files {
		if f.RelPath == "server/awg0.conf" {
			serverFound = true
			break
		}
	}
	if !serverFound {
		t.Error("server config should always be included regardless of PeerFilter")
	}
}

func TestGenerate_MultipleServersError(t *testing.T) {
	manifest := Manifest{
		Version: 1,
		Network: NetworkConfig{
			MTU: 1280,
			DNS: []string{"1.1.1.1"},
		},
		Obfuscation: ObfuscationManifest{},
		Peers: map[string]PeerManifest{
			"server1": {
				Address:    "10.0.0.1/24",
				Endpoint:   "vpn1.example.com:51820",
				ListenPort: 51820,
			},
			"server2": {
				Address:    "10.0.0.2/24",
				Endpoint:   "vpn2.example.com:51820",
				ListenPort: 51821,
			},
		},
	}

	outputDir := t.TempDir()
	_, err := Generate(manifest, GenerateOptions{OutputDir: outputDir})
	if err == nil {
		t.Error("expected error for multiple server peers, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "exactly one server peer required") {
		t.Errorf("expected 'exactly one server peer required' error, got: %v", err)
	}
}

func TestGenerate_NoServerError(t *testing.T) {
	manifest := Manifest{
		Version: 1,
		Network: NetworkConfig{
			MTU: 1280,
			DNS: []string{"1.1.1.1"},
		},
		Obfuscation: ObfuscationManifest{},
		Peers: map[string]PeerManifest{
			"phone": {
				Address: "10.0.0.2/32",
			},
			"laptop": {
				Address: "10.0.0.3/32",
			},
		},
	}

	outputDir := t.TempDir()
	_, err := Generate(manifest, GenerateOptions{OutputDir: outputDir})
	if err == nil {
		t.Error("expected error for no server peer, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "exactly one server peer required") {
		t.Errorf("expected 'exactly one server peer required' error, got: %v", err)
	}
}

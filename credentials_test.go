package amnezigo

import (
	"os"
	"path/filepath"
	"testing"
)

// ---- Step A: type constructor tests ----

func TestEmptyCredentials_InitializedMap(t *testing.T) {
	creds := EmptyCredentials()
	if creds.Peers == nil {
		t.Fatal("EmptyCredentials().Peers must be initialized, not nil")
	}
	if len(creds.Peers) != 0 {
		t.Errorf("expected 0 peers, got %d", len(creds.Peers))
	}
	if creds.Server.PrivateKey != "" {
		t.Error("server PrivateKey must be empty on EmptyCredentials")
	}
}

func TestPeerCredentials_ZeroValue(t *testing.T) {
	var pc PeerCredentials
	if pc.PrivateKey != "" || pc.PublicKey != "" || pc.PresharedKey != "" {
		t.Error("zero-value PeerCredentials must have empty string fields")
	}
}

// ---- Helpers shared across steps ----

func validObfuscation(t *testing.T) ServerObfuscationConfig {
	t.Helper()
	return GenerateServerConfig(1280, 32, 5)
}

func writeTestConfig(t *testing.T, path string, cfg ServerConfig) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := WriteServerConfig(f, cfg); err != nil {
		t.Fatal(err)
	}
}

// writePeerClientConfig writes a minimal client config carrying the peer's
// PrivateKey, mirroring what the generate pipeline produces under
// <outputDir>/<peer>/awg0.conf. LoadCredentials reads peer private keys from
// these per-peer files (never from the server config, which stores only
// public material — a peer private key is a client secret).
func writePeerClientConfig(t *testing.T, dir, peerName, privKey, psk string) {
	t.Helper()
	peerDir := filepath.Join(dir, peerName)
	if err := os.MkdirAll(peerDir, 0o750); err != nil {
		t.Fatal(err)
	}
	content := "[Interface]\nPrivateKey = " + privKey + "\n"
	if psk != "" {
		content += "\n[Peer]\nPresharedKey = " + psk + "\n"
	}
	if err := os.WriteFile(filepath.Join(peerDir, outputConfigName), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// ---- Step B: server credential extraction ----

func TestLoadCredentials_ServerConfig(t *testing.T) {
	dir := t.TempDir()
	serverDir := filepath.Join(dir, "gateway")
	if err := os.MkdirAll(serverDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: "server-priv-key-base64",
			PublicKey:  "server-pub-key-base64",
			Address:    "10.0.0.1/24",
			ListenPort: 51820,
			MTU:        1280,
		},
		Obfuscation: validObfuscation(t),
	}
	path := filepath.Join(serverDir, "awg0.conf")
	writeTestConfig(t, path, cfg)

	creds, err := LoadCredentials(dir, "gateway")
	if err != nil {
		t.Fatalf("LoadCredentials: %v", err)
	}
	if creds.Server.PrivateKey != "server-priv-key-base64" {
		t.Errorf("server PrivateKey = %q, want %q",
			creds.Server.PrivateKey, "server-priv-key-base64")
	}
	if creds.Server.PublicKey != "server-pub-key-base64" {
		t.Errorf("server PublicKey = %q, want %q",
			creds.Server.PublicKey, "server-pub-key-base64")
	}
}

func TestLoadCredentials_ServerConfigWithPeers(t *testing.T) {
	dir := t.TempDir()
	serverDir := filepath.Join(dir, "server")
	if err := os.MkdirAll(serverDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: "srv-priv",
			PublicKey:  "srv-pub",
			Address:    "10.0.0.1/24",
			ListenPort: 51820,
			MTU:        1280,
		},
		Obfuscation: validObfuscation(t),
		Peers: []PeerConfig{
			{
				Name:         "phone",
				PrivateKey:   "phone-priv",
				PublicKey:    "phone-pub",
				PresharedKey: "phone-psk",
				AllowedIPs:   "10.0.0.2/32",
			},
			{
				Name:         "laptop",
				PrivateKey:   "laptop-priv",
				PublicKey:    "laptop-pub",
				PresharedKey: "laptop-psk",
				AllowedIPs:   "10.0.0.3/32",
			},
		},
	}
	path := filepath.Join(serverDir, "awg0.conf")
	writeTestConfig(t, path, cfg)
	// Peer private keys live in each peer's own client config; the server
	// config carries only public material (mirrors the generate pipeline).
	writePeerClientConfig(t, dir, "phone", "phone-priv", "")
	writePeerClientConfig(t, dir, "laptop", "laptop-priv", "")

	creds, err := LoadCredentials(dir, "server")
	if err != nil {
		t.Fatalf("LoadCredentials: %v", err)
	}

	// Server credentials
	if creds.Server.PrivateKey != "srv-priv" {
		t.Errorf("server priv = %q", creds.Server.PrivateKey)
	}

	// Phone credentials
	phone, ok := creds.Peers["phone"]
	if !ok {
		t.Fatal("phone peer not found in credentials")
	}
	if phone.PrivateKey != "phone-priv" {
		t.Errorf("phone priv = %q", phone.PrivateKey)
	}
	if phone.PublicKey != "phone-pub" {
		t.Errorf("phone pub = %q", phone.PublicKey)
	}
	if phone.PresharedKey != "phone-psk" {
		t.Errorf("phone psk = %q", phone.PresharedKey)
	}

	// Laptop credentials
	laptop, ok := creds.Peers["laptop"]
	if !ok {
		t.Fatal("laptop peer not found in credentials")
	}
	if laptop.PrivateKey != "laptop-priv" {
		t.Errorf("laptop priv = %q", laptop.PrivateKey)
	}
}

// ---- Step C: edge cases ----

func TestLoadCredentials_MissingOutputDir(t *testing.T) {
	creds, err := LoadCredentials("/nonexistent/path", "server")
	if err != nil {
		t.Fatalf("expected nil error for missing dir, got: %v", err)
	}
	if len(creds.Peers) != 0 {
		t.Errorf("expected 0 peers, got %d", len(creds.Peers))
	}
	if creds.Server.PrivateKey != "" {
		t.Error("expected empty server credentials")
	}
}

func TestLoadCredentials_EmptyOutputDir(t *testing.T) {
	dir := t.TempDir()
	// Server dir exists but no config file inside.
	serverDir := filepath.Join(dir, "server")
	if err := os.MkdirAll(serverDir, 0o755); err != nil {
		t.Fatal(err)
	}
	creds, err := LoadCredentials(dir, "server")
	if err != nil {
		t.Fatalf("expected nil error for empty dir, got: %v", err)
	}
	if len(creds.Peers) != 0 {
		t.Errorf("expected 0 peers, got %d", len(creds.Peers))
	}
}

func TestLoadCredentials_PeerWithoutName(t *testing.T) {
	dir := t.TempDir()
	serverDir := filepath.Join(dir, "server")
	if err := os.MkdirAll(serverDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: "srv-priv",
			Address:    "10.0.0.1/24",
			ListenPort: 51820,
			MTU:        1280,
		},
		Obfuscation: validObfuscation(t),
		Peers: []PeerConfig{
			{
				// No Name — #_Name comment not present.
				PublicKey:    "anon-pub",
				PresharedKey: "anon-psk",
				AllowedIPs:   "10.0.0.2/32",
			},
		},
	}
	path := filepath.Join(serverDir, "awg0.conf")
	writeTestConfig(t, path, cfg)

	creds, err := LoadCredentials(dir, "server")
	if err != nil {
		t.Fatalf("LoadCredentials: %v", err)
	}
	// Unnamed peer must be skipped — cannot match to manifest entries.
	if len(creds.Peers) != 0 {
		t.Errorf("expected 0 named peers, got %d", len(creds.Peers))
	}
}

func TestLoadCredentials_ServerWithoutPublicKey(t *testing.T) {
	dir := t.TempDir()
	serverDir := filepath.Join(dir, "server")
	if err := os.MkdirAll(serverDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: "srv-priv",
			// PublicKey intentionally omitted — derived at export time.
			Address:    "10.0.0.1/24",
			ListenPort: 51820,
			MTU:        1280,
		},
		Obfuscation: validObfuscation(t),
	}
	path := filepath.Join(serverDir, "awg0.conf")
	writeTestConfig(t, path, cfg)

	creds, err := LoadCredentials(dir, "server")
	if err != nil {
		t.Fatalf("LoadCredentials: %v", err)
	}
	if creds.Server.PrivateKey != "srv-priv" {
		t.Errorf("server priv = %q", creds.Server.PrivateKey)
	}
	// PublicKey is empty — the pipeline will derive it from PrivateKey if needed.
	if creds.Server.PublicKey != "" {
		t.Errorf("expected empty pub when not in config, got %q", creds.Server.PublicKey)
	}
}

// ---- Step D: client config fallback extraction ----

func TestExtractClientCredentials_ValidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "awg0.conf")

	clientCfg := ClientConfig{
		Interface: ClientInterfaceConfig{
			PrivateKey: "client-priv-key",
			Address:    "10.0.0.2/32",
			DNS:        "1.1.1.1",
			MTU:        1280,
			Obfuscation: ClientObfuscationConfig{
				ServerObfuscationConfig: validObfuscation(t),
			},
		},
		Peer: ClientPeerConfig{
			PublicKey:           "server-pub",
			PresharedKey:        "client-psk",
			Endpoint:            "vpn.example.com:51820",
			AllowedIPs:          "0.0.0.0/0",
			PersistentKeepalive: 25,
		},
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := WriteClientConfig(f, clientCfg); err != nil {
		t.Fatal(err)
	}

	creds, err := extractClientCredentials(path)
	if err != nil {
		t.Fatalf("extractClientCredentials: %v", err)
	}
	if creds.PrivateKey != "client-priv-key" {
		t.Errorf("PrivateKey = %q, want %q", creds.PrivateKey, "client-priv-key")
	}
	if creds.PresharedKey != "client-psk" {
		t.Errorf("PresharedKey = %q, want %q", creds.PresharedKey, "client-psk")
	}
}

func TestExtractClientCredentials_MissingFile(t *testing.T) {
	_, err := extractClientCredentials("/nonexistent/awg0.conf")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadCredentials_FallbackToClientConfigs(t *testing.T) {
	dir := t.TempDir()
	// No server config — server dir does not exist.
	// But phone dir exists with a client config.
	phoneDir := filepath.Join(dir, "phone")
	if err := os.MkdirAll(phoneDir, 0o755); err != nil {
		t.Fatal(err)
	}

	clientCfg := ClientConfig{
		Interface: ClientInterfaceConfig{
			PrivateKey: "phone-priv",
			Address:    "10.0.0.2/32",
			DNS:        "1.1.1.1",
			MTU:        1280,
			Obfuscation: ClientObfuscationConfig{
				ServerObfuscationConfig: validObfuscation(t),
			},
		},
		Peer: ClientPeerConfig{
			PublicKey:           "server-pub",
			PresharedKey:        "phone-psk",
			Endpoint:            "vpn.example.com:51820",
			AllowedIPs:          "0.0.0.0/0",
			PersistentKeepalive: 25,
		},
	}
	path := filepath.Join(phoneDir, "awg0.conf")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := WriteClientConfig(f, clientCfg); err != nil {
		t.Fatal(err)
	}

	creds, err := LoadCredentials(dir, "server")
	if err != nil {
		t.Fatalf("LoadCredentials: %v", err)
	}
	// Server credentials should be empty (no server config).
	if creds.Server.PrivateKey != "" {
		t.Error("expected empty server credentials")
	}
	// Phone credentials should be recovered from client config.
	phone, ok := creds.Peers["phone"]
	if !ok {
		t.Fatal("phone peer not found after client fallback")
	}
	if phone.PrivateKey != "phone-priv" {
		t.Errorf("phone priv = %q", phone.PrivateKey)
	}
	if phone.PresharedKey != "phone-psk" {
		t.Errorf("phone psk = %q", phone.PresharedKey)
	}
}

// ---- Step E: round-trip and idempotency ----

func TestLoadCredentials_RoundTrip_WriteAndReload(t *testing.T) {
	dir := t.TempDir()
	serverDir := filepath.Join(dir, "server")
	if err := os.MkdirAll(serverDir, 0o755); err != nil {
		t.Fatal(err)
	}

	serverPriv, serverPub := GenerateKeyPair()
	phonePriv, phonePub := GenerateKeyPair()
	phonePSK := GeneratePSK()
	laptopPriv, laptopPub := GenerateKeyPair()
	laptopPSK := GeneratePSK()

	cfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: serverPriv,
			PublicKey:  serverPub,
			Address:    "10.0.0.1/24",
			ListenPort: 51820,
			MTU:        1280,
		},
		Obfuscation: validObfuscation(t),
		Peers: []PeerConfig{
			{
				Name: "phone", PrivateKey: phonePriv,
				PublicKey: phonePub, PresharedKey: phonePSK,
				AllowedIPs: "10.0.0.2/32",
			},
			{
				Name: "laptop", PrivateKey: laptopPriv,
				PublicKey: laptopPub, PresharedKey: laptopPSK,
				AllowedIPs: "10.0.0.3/32",
			},
		},
	}
	writeTestConfig(t, filepath.Join(serverDir, "awg0.conf"), cfg)
	// Peer private keys live in per-peer client configs, not the server config.
	writePeerClientConfig(t, dir, "phone", phonePriv, phonePSK)
	writePeerClientConfig(t, dir, "laptop", laptopPriv, laptopPSK)

	creds, err := LoadCredentials(dir, "server")
	if err != nil {
		t.Fatalf("LoadCredentials: %v", err)
	}

	// Server round-trip.
	if creds.Server.PrivateKey != serverPriv {
		t.Errorf("server priv mismatch")
	}
	if creds.Server.PublicKey != serverPub {
		t.Errorf("server pub mismatch")
	}

	// Phone round-trip.
	if creds.Peers["phone"].PrivateKey != phonePriv {
		t.Errorf("phone priv mismatch")
	}
	if creds.Peers["phone"].PublicKey != phonePub {
		t.Errorf("phone pub mismatch")
	}
	if creds.Peers["phone"].PresharedKey != phonePSK {
		t.Errorf("phone psk mismatch")
	}

	// Laptop round-trip.
	if creds.Peers["laptop"].PrivateKey != laptopPriv {
		t.Errorf("laptop priv mismatch")
	}
	if creds.Peers["laptop"].PresharedKey != laptopPSK {
		t.Errorf("laptop psk mismatch")
	}

	// Verify derived public keys match.
	if DerivePublicKey(creds.Peers["phone"].PrivateKey) != phonePub {
		t.Error("phone pubkey derivation mismatch")
	}
	if DerivePublicKey(creds.Peers["laptop"].PrivateKey) != laptopPub {
		t.Error("laptop pubkey derivation mismatch")
	}
}

func TestLoadCredentials_Idempotent(t *testing.T) {
	dir := t.TempDir()
	serverDir := filepath.Join(dir, "server")
	if err := os.MkdirAll(serverDir, 0o755); err != nil {
		t.Fatal(err)
	}

	serverPriv, serverPub := GenerateKeyPair()
	peerPriv, peerPub := GenerateKeyPair()
	peerPSK := GeneratePSK()

	cfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: serverPriv, PublicKey: serverPub,
			Address: "10.0.0.1/24", ListenPort: 51820, MTU: 1280,
		},
		Obfuscation: validObfuscation(t),
		Peers: []PeerConfig{
			{
				Name: "phone", PrivateKey: peerPriv,
				PublicKey: peerPub, PresharedKey: peerPSK,
				AllowedIPs: "10.0.0.2/32",
			},
		},
	}
	writeTestConfig(t, filepath.Join(serverDir, "awg0.conf"), cfg)
	// Peer private key lives in the per-peer client config, not the server config.
	writePeerClientConfig(t, dir, "phone", peerPriv, peerPSK)

	creds1, _ := LoadCredentials(dir, "server")
	creds2, _ := LoadCredentials(dir, "server")

	if creds1.Server.PrivateKey != creds2.Server.PrivateKey {
		t.Error("non-idempotent server key")
	}
	if creds1.Peers["phone"].PrivateKey != creds2.Peers["phone"].PrivateKey {
		t.Error("non-idempotent peer key")
	}
	if creds1.Peers["phone"].PresharedKey != creds2.Peers["phone"].PresharedKey {
		t.Error("non-idempotent peer psk")
	}
}

// TestLoadCredentials_PeerPrivkeyFromClientConfig pins the fix for the
// credential-reuse bug: a peer's PrivateKey must come from its own client
// config, never from the server config. The server config [Peer] section is
// seeded with a stale "server-side" private key; LoadCredentials must ignore it
// and return the key from output/<peer>/awg0.conf instead. Regression guard
// for the rotate-on-every-generate bug.
func TestLoadCredentials_PeerPrivkeyFromClientConfig(t *testing.T) {
	dir := t.TempDir()
	serverDir := filepath.Join(dir, "server")
	if err := os.MkdirAll(serverDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: "srv-priv", PublicKey: "srv-pub",
			Address: "10.0.0.1/24", ListenPort: 51820, MTU: 1280,
		},
		Obfuscation: validObfuscation(t),
		Peers: []PeerConfig{
			{
				Name: "phone",
				// Stale server-side copy — must be ignored in favour of the
				// peer's own client config.
				PrivateKey:   "STALE-SERVER-SIDE-PRIV",
				PublicKey:    "phone-pub",
				PresharedKey: "phone-psk",
				AllowedIPs:   "10.0.0.2/32",
			},
		},
	}
	writeTestConfig(t, filepath.Join(serverDir, "awg0.conf"), cfg)
	writePeerClientConfig(t, dir, "phone", "phone-real-priv", "phone-psk")

	creds, err := LoadCredentials(dir, "server")
	if err != nil {
		t.Fatalf("LoadCredentials: %v", err)
	}
	phone := creds.Peers["phone"]
	if phone.PrivateKey != "phone-real-priv" {
		t.Errorf("peer priv = %q, want %q (from client config, not server)",
			phone.PrivateKey, "phone-real-priv")
	}
	if phone.PublicKey != "phone-pub" {
		t.Errorf("peer pub = %q, want %q (from server config)", phone.PublicKey, "phone-pub")
	}
	if phone.PresharedKey != "phone-psk" {
		t.Errorf("peer psk = %q, want %q", phone.PresharedKey, "phone-psk")
	}
}

// TestLoadCredentials_PeerWithoutClientConfig covers a freshly declared peer
// whose client config has not been generated yet: PrivateKey must be empty so
// the pipeline generates a fresh keypair on the next run.
func TestLoadCredentials_PeerWithoutClientConfig(t *testing.T) {
	dir := t.TempDir()
	serverDir := filepath.Join(dir, "server")
	if err := os.MkdirAll(serverDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: "srv-priv", PublicKey: "srv-pub",
			Address: "10.0.0.1/24", ListenPort: 51820, MTU: 1280,
		},
		Obfuscation: validObfuscation(t),
		Peers: []PeerConfig{
			{Name: "phone", PublicKey: "phone-pub", PresharedKey: "phone-psk",
				AllowedIPs: "10.0.0.2/32"},
		},
	}
	writeTestConfig(t, filepath.Join(serverDir, "awg0.conf"), cfg)
	// No client config for "phone" — simulates a newly added manifest peer.

	creds, err := LoadCredentials(dir, "server")
	if err != nil {
		t.Fatalf("LoadCredentials: %v", err)
	}
	phone := creds.Peers["phone"]
	if phone.PrivateKey != "" {
		t.Errorf("peer priv = %q, want empty (no client config → fresh key on next run)",
			phone.PrivateKey)
	}
}

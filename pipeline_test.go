package amnezigo

import "testing"

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

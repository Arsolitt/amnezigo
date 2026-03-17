package config

import (
	"strings"
	"testing"
)

func TestWriteServerConfig(t *testing.T) {
	cfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: "test_priv_key_1",
			PublicKey:  "test_pub_key_1",
			Address:    "10.0.0.1/24",
			ListenPort: 51820,
			MTU:        1420,
			PostUp:     "iptables -A FORWARD -i %i -j ACCEPT; iptables -A FORWARD -o %i -j ACCEPT; iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE",
			PostDown:   "iptables -D FORWARD -i %i -j ACCEPT; iptables -D FORWARD -o %i -j ACCEPT; iptables -t nat -D POSTROUTING -o eth0 -j MASQUERADE",
		},
		Peers: []PeerConfig{
			{
				Name:       "peer1",
				PrivateKey: "peer1_priv_key",
				PublicKey:  "peer1_pub_key",
				AllowedIPs: "10.0.0.2/32",
			},
		},
		Obfuscation: ObfuscationConfig{
			Jc:   50,
			Jmin: 10,
			Jmax: 30,
			S1:   100,
			S2:   200,
			S3:   300,
			S4:   400,
			H1:   1,
			H2:   2,
			H3:   3,
			H4:   4,
			I1:   "i1_value",
			I2:   "i2_value",
			I3:   "i3_value",
			I4:   "i4_value",
			I5:   "i5_value",
		},
	}

	var buf strings.Builder
	err := WriteServerConfig(&buf, cfg)
	if err != nil {
		t.Fatalf("WriteServerConfig failed: %v", err)
	}

	output := buf.String()

	// Check [Interface] section
	if !strings.Contains(output, "[Interface]") {
		t.Error("Output should contain [Interface] section")
	}
	if !strings.Contains(output, "PrivateKey = test_priv_key_1") {
		t.Error("Output should contain PrivateKey")
	}
	if !strings.Contains(output, "Address = 10.0.0.1/24") {
		t.Error("Output should contain Address")
	}
	if !strings.Contains(output, "ListenPort = 51820") {
		t.Error("Output should contain ListenPort")
	}
	if !strings.Contains(output, "MTU = 1420") {
		t.Error("Output should contain MTU")
	}

	// Check PostUp and PostDown
	if !strings.Contains(output, "PostUp =") {
		t.Error("Output should contain PostUp")
	}
	if !strings.Contains(output, "PostDown =") {
		t.Error("Output should contain PostDown")
	}

	// Check obfuscation params
	if !strings.Contains(output, "Jc = 50") {
		t.Error("Output should contain Jc")
	}
	if !strings.Contains(output, "Jmin = 10") {
		t.Error("Output should contain Jmin")
	}
	if !strings.Contains(output, "Jmax = 30") {
		t.Error("Output should contain Jmax")
	}
	if !strings.Contains(output, "S1 = 100") {
		t.Error("Output should contain S1")
	}
	if !strings.Contains(output, "H1 = 1") {
		t.Error("Output should contain H1")
	}
	if !strings.Contains(output, "I1 = i1_value") {
		t.Error("Output should contain I1")
	}

	// Check [Peer] section
	if !strings.Contains(output, "[Peer]") {
		t.Error("Output should contain [Peer] section")
	}
	if !strings.Contains(output, "#_Name = peer1") {
		t.Error("Output should contain commented #_Name")
	}
	if !strings.Contains(output, "#_PrivateKey = peer1_priv_key") {
		t.Error("Output should contain commented #_PrivateKey")
	}
	if !strings.Contains(output, "PublicKey = peer1_pub_key") {
		t.Error("Output should contain PublicKey")
	}
	if !strings.Contains(output, "AllowedIPs = 10.0.0.2/32") {
		t.Error("Output should contain AllowedIPs")
	}
}

func TestWriteServerConfigWithPresharedKey(t *testing.T) {
	cfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: "test_priv_key_1",
			PublicKey:  "test_pub_key_1",
			Address:    "10.0.0.1/24",
			ListenPort: 51820,
			MTU:        1420,
		},
		Peers: []PeerConfig{
			{
				Name:         "peer1",
				PrivateKey:   "peer1_priv_key",
				PublicKey:    "peer1_pub_key",
				PresharedKey: "peer1_psk_value",
				AllowedIPs:   "10.0.0.2/32",
			},
		},
	}

	var buf strings.Builder
	err := WriteServerConfig(&buf, cfg)
	if err != nil {
		t.Fatalf("WriteServerConfig failed: %v", err)
	}

	output := buf.String()

	// Check [Interface] section exists
	if !strings.Contains(output, "[Interface]") {
		t.Error("Output should contain [Interface] section")
	}

	// Check [Peer] section
	if !strings.Contains(output, "[Peer]") {
		t.Error("Output should contain [Peer] section")
	}

	// Check PublicKey is present
	if !strings.Contains(output, "PublicKey = peer1_pub_key") {
		t.Error("Output should contain PublicKey")
	}

	// Check PresharedKey is present
	if !strings.Contains(output, "PresharedKey = peer1_psk_value") {
		t.Error("Output should contain PresharedKey")
	}

	// Check that PresharedKey comes after PublicKey
	pubKeyIndex := strings.Index(output, "PublicKey = peer1_pub_key")
	pskIndex := strings.Index(output, "PresharedKey = peer1_psk_value")
	if pubKeyIndex == -1 || pskIndex == -1 {
		t.Error("Both PublicKey and PresharedKey should be present")
	} else if pskIndex < pubKeyIndex {
		t.Error("PresharedKey should appear after PublicKey")
	}

	// Check AllowedIPs is present
	if !strings.Contains(output, "AllowedIPs = 10.0.0.2/32") {
		t.Error("Output should contain AllowedIPs")
	}
}

func TestWriteClientConfig(t *testing.T) {
	cfg := ClientConfig{
		Interface: ClientInterfaceConfig{
			PrivateKey: "client_priv_key",
			Address:    "10.0.0.2/32",
			DNS:        "1.1.1.1",
			MTU:        1420,
			Obfuscation: ObfuscationConfig{
				Jc:   50,
				Jmin: 10,
				Jmax: 30,
				S1:   100,
				S2:   200,
				S3:   300,
				S4:   400,
				H1:   1,
				H2:   2,
				H3:   3,
				H4:   4,
				I1:   "i1_value",
				I2:   "i2_value",
				I3:   "i3_value",
				I4:   "i4_value",
				I5:   "i5_value",
			},
		},
		Peer: ClientPeerConfig{
			PublicKey:           "server_pub_key",
			PresharedKey:        "client_psk",
			Endpoint:            "example.com:51820",
			AllowedIPs:          "0.0.0.0/0",
			PersistentKeepalive: 25,
		},
	}

	var buf strings.Builder
	err := WriteClientConfig(&buf, cfg)
	if err != nil {
		t.Fatalf("WriteClientConfig failed: %v", err)
	}

	output := buf.String()

	// Check [Interface] section
	if !strings.Contains(output, "[Interface]") {
		t.Error("Output should contain [Interface] section")
	}
	if !strings.Contains(output, "PrivateKey = client_priv_key") {
		t.Error("Output should contain PrivateKey")
	}
	if !strings.Contains(output, "Address = 10.0.0.2/32") {
		t.Error("Output should contain Address")
	}
	if !strings.Contains(output, "DNS = 1.1.1.1") {
		t.Error("Output should contain DNS")
	}
	if !strings.Contains(output, "MTU = 1420") {
		t.Error("Output should contain MTU")
	}

	// Check obfuscation params
	if !strings.Contains(output, "Jc = 50") {
		t.Error("Output should contain Jc")
	}
	if !strings.Contains(output, "I1 = i1_value") {
		t.Error("Output should contain I1")
	}

	// Check [Peer] section
	if !strings.Contains(output, "[Peer]") {
		t.Error("Output should contain [Peer] section")
	}
	if !strings.Contains(output, "PublicKey = server_pub_key") {
		t.Error("Output should contain PublicKey")
	}
	if !strings.Contains(output, "PresharedKey = client_psk") {
		t.Error("Output should contain PresharedKey")
	}
	if !strings.Contains(output, "Endpoint = example.com:51820") {
		t.Error("Output should contain Endpoint")
	}
	if !strings.Contains(output, "AllowedIPs = 0.0.0.0/0") {
		t.Error("Output should contain AllowedIPs")
	}
	if !strings.Contains(output, "PersistentKeepalive = 25") {
		t.Error("Output should contain PersistentKeepalive")
	}
}

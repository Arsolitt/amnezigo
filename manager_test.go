package amnezigo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewManager(t *testing.T) {
	mgr := NewManager("test.conf")
	if mgr.ConfigPath != "test.conf" {
		t.Errorf("expected ConfigPath 'test.conf', got '%s'", mgr.ConfigPath)
	}
}

func TestManagerLoadSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")

	cfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: "testpriv=",
			Address:    "10.8.0.1/24",
			ListenPort: 12345,
			MTU:        1280,
		},
		Obfuscation: ServerObfuscationConfig{
			Jc: 5, Jmin: 100, Jmax: 200, S1: 10, S2: 20, S3: 30, S4: 5,
			H1: HeaderRange{Min: 100, Max: 200},
			H2: HeaderRange{Min: 300, Max: 400},
			H3: HeaderRange{Min: 500, Max: 600},
			H4: HeaderRange{Min: 700, Max: 800},
		},
		Peers: []PeerConfig{},
	}

	mgr := NewManager(path)
	if err := mgr.Save(cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := mgr.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.Interface.PrivateKey != cfg.Interface.PrivateKey {
		t.Errorf("PrivateKey mismatch")
	}
	if loaded.Interface.Address != cfg.Interface.Address {
		t.Errorf("Address mismatch")
	}
	if loaded.Interface.ListenPort != cfg.Interface.ListenPort {
		t.Errorf("ListenPort mismatch")
	}
	if loaded.Obfuscation.Jc != cfg.Obfuscation.Jc {
		t.Errorf("Jc mismatch")
	}
}

func TestManagerAddPeer(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")

	initialCfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: "serverpriv=",
			PublicKey:  "serverpub=",
			Address:    "10.8.0.1/24",
			ListenPort: 12345,
			MTU:        1280,
		},
		Obfuscation: ServerObfuscationConfig{
			Jc: 5, Jmin: 100, Jmax: 200, S1: 10, S2: 20, S3: 30, S4: 5,
			H1: HeaderRange{Min: 100, Max: 200},
			H2: HeaderRange{Min: 300, Max: 400},
			H3: HeaderRange{Min: 500, Max: 600},
			H4: HeaderRange{Min: 700, Max: 800},
		},
		Peers: []PeerConfig{},
	}

	mgr := NewManager(path)
	if err := mgr.Save(initialCfg); err != nil {
		t.Fatalf("initial Save failed: %v", err)
	}

	peer, err := mgr.AddPeer("testclient", "")
	if err != nil {
		t.Fatalf("AddPeer failed: %v", err)
	}

	if peer.Name != "testclient" {
		t.Errorf("expected name 'testclient', got '%s'", peer.Name)
	}
	if !strings.HasSuffix(peer.AllowedIPs, "/32") {
		t.Errorf("expected AllowedIPs to end with /32, got '%s'", peer.AllowedIPs)
	}
	if peer.PublicKey == "" {
		t.Error("expected non-empty PublicKey")
	}
	if peer.PresharedKey == "" {
		t.Error("expected non-empty PresharedKey")
	}

	loaded, err := mgr.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(loaded.Peers) != 1 {
		t.Fatalf("expected 1 peer, got %d", len(loaded.Peers))
	}
	if loaded.Peers[0].Name != "testclient" {
		t.Errorf("expected peer name 'testclient', got '%s'", loaded.Peers[0].Name)
	}
}

func TestManagerAddPeerDuplicate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")

	cfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: "priv=", Address: "10.8.0.1/24",
			ListenPort: 12345, MTU: 1280,
		},
		Obfuscation: ServerObfuscationConfig{
			Jc: 5, Jmin: 100, Jmax: 200, S1: 10, S2: 20, S3: 30, S4: 5,
			H1: HeaderRange{Min: 100, Max: 200},
			H2: HeaderRange{Min: 300, Max: 400},
			H3: HeaderRange{Min: 500, Max: 600},
			H4: HeaderRange{Min: 700, Max: 800},
		},
		Peers: []PeerConfig{},
	}

	mgr := NewManager(path)
	_ = mgr.Save(cfg)
	_, _ = mgr.AddPeer("dup", "")

	_, err := mgr.AddPeer("dup", "")
	if err == nil {
		t.Fatal("expected error for duplicate peer name")
	}
}

func TestManagerAddPeerWithIP(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")

	cfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: "priv=", Address: "10.8.0.1/24",
			ListenPort: 12345, MTU: 1280,
		},
		Obfuscation: ServerObfuscationConfig{
			Jc: 5, Jmin: 100, Jmax: 200, S1: 10, S2: 20, S3: 30, S4: 5,
			H1: HeaderRange{Min: 100, Max: 200},
			H2: HeaderRange{Min: 300, Max: 400},
			H3: HeaderRange{Min: 500, Max: 600},
			H4: HeaderRange{Min: 700, Max: 800},
		},
		Peers: []PeerConfig{},
	}

	mgr := NewManager(path)
	_ = mgr.Save(cfg)

	peer, err := mgr.AddPeer("manual", "10.8.0.50")
	if err != nil {
		t.Fatalf("AddPeer with IP failed: %v", err)
	}

	if peer.AllowedIPs != "10.8.0.50/32" {
		t.Errorf("expected AllowedIPs '10.8.0.50/32', got '%s'", peer.AllowedIPs)
	}
}

func TestManagerRemovePeer(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")

	cfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: "priv=", Address: "10.8.0.1/24",
			ListenPort: 12345, MTU: 1280,
		},
		Obfuscation: ServerObfuscationConfig{
			Jc: 5, Jmin: 100, Jmax: 200, S1: 10, S2: 20, S3: 30, S4: 5,
			H1: HeaderRange{Min: 100, Max: 200},
			H2: HeaderRange{Min: 300, Max: 400},
			H3: HeaderRange{Min: 500, Max: 600},
			H4: HeaderRange{Min: 700, Max: 800},
		},
		Peers: []PeerConfig{
			{Name: "keep", PublicKey: "pub1", AllowedIPs: "10.8.0.2/32"},
			{Name: "remove", PublicKey: "pub2", AllowedIPs: "10.8.0.3/32"},
		},
	}

	mgr := NewManager(path)
	_ = mgr.Save(cfg)

	err := mgr.RemovePeer("remove")
	if err != nil {
		t.Fatalf("RemovePeer failed: %v", err)
	}

	loaded, _ := mgr.Load()
	if len(loaded.Peers) != 1 {
		t.Fatalf("expected 1 peer after removal, got %d", len(loaded.Peers))
	}
	if loaded.Peers[0].Name != "keep" {
		t.Errorf("expected remaining peer 'keep', got '%s'", loaded.Peers[0].Name)
	}
}

func TestManagerRemovePeerNotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")

	cfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: "priv=", Address: "10.8.0.1/24",
			ListenPort: 12345, MTU: 1280,
		},
		Obfuscation: ServerObfuscationConfig{
			Jc: 5, Jmin: 100, Jmax: 200, S1: 10, S2: 20, S3: 30, S4: 5,
			H1: HeaderRange{Min: 100, Max: 200},
			H2: HeaderRange{Min: 300, Max: 400},
			H3: HeaderRange{Min: 500, Max: 600},
			H4: HeaderRange{Min: 700, Max: 800},
		},
		Peers: []PeerConfig{},
	}

	mgr := NewManager(path)
	_ = mgr.Save(cfg)

	err := mgr.RemovePeer("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent peer")
	}
}

func TestManagerFindPeer(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")

	cfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: "priv=", Address: "10.8.0.1/24",
			ListenPort: 12345, MTU: 1280,
		},
		Obfuscation: ServerObfuscationConfig{
			Jc: 5, Jmin: 100, Jmax: 200, S1: 10, S2: 20, S3: 30, S4: 5,
			H1: HeaderRange{Min: 100, Max: 200},
			H2: HeaderRange{Min: 300, Max: 400},
			H3: HeaderRange{Min: 500, Max: 600},
			H4: HeaderRange{Min: 700, Max: 800},
		},
		Peers: []PeerConfig{
			{Name: "target", PublicKey: "pub1", AllowedIPs: "10.8.0.2/32"},
		},
	}

	mgr := NewManager(path)
	_ = mgr.Save(cfg)

	peer, err := mgr.FindPeer("target")
	if err != nil {
		t.Fatalf("FindPeer failed: %v", err)
	}
	if peer.Name != "target" {
		t.Errorf("expected name 'target', got '%s'", peer.Name)
	}
}

func TestManagerListPeers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")

	cfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: "priv=", Address: "10.8.0.1/24",
			ListenPort: 12345, MTU: 1280,
		},
		Obfuscation: ServerObfuscationConfig{
			Jc: 5, Jmin: 100, Jmax: 200, S1: 10, S2: 20, S3: 30, S4: 5,
			H1: HeaderRange{Min: 100, Max: 200},
			H2: HeaderRange{Min: 300, Max: 400},
			H3: HeaderRange{Min: 500, Max: 600},
			H4: HeaderRange{Min: 700, Max: 800},
		},
		Peers: []PeerConfig{
			{Name: "a", PublicKey: "pub1", AllowedIPs: "10.8.0.2/32"},
			{Name: "b", PublicKey: "pub2", AllowedIPs: "10.8.0.3/32"},
		},
	}

	mgr := NewManager(path)
	_ = mgr.Save(cfg)

	peers := mgr.ListPeers()
	if len(peers) != 2 {
		t.Fatalf("expected 2 peers, got %d", len(peers))
	}
}

func TestManagerExportPeer(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")

	cfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: "serverpriv=",
			PublicKey:  "serverpub=",
			Address:    "10.8.0.1/24",
			ListenPort: 12345,
			MTU:        1280,
		},
		Obfuscation: ServerObfuscationConfig{
			Jc: 5, Jmin: 100, Jmax: 200, S1: 10, S2: 20, S3: 30, S4: 5,
			H1: HeaderRange{Min: 100, Max: 200},
			H2: HeaderRange{Min: 300, Max: 400},
			H3: HeaderRange{Min: 500, Max: 600},
			H4: HeaderRange{Min: 700, Max: 800},
		},
		Peers: []PeerConfig{
			{
				Name:         "exportme",
				PrivateKey:   "clientpriv=",
				PublicKey:    "clientpub=",
				PresharedKey: "psk=",
				AllowedIPs:   "10.8.0.2/32",
			},
		},
	}

	mgr := NewManager(path)
	_ = mgr.Save(cfg)

	clientCfg, err := mgr.ExportPeer("exportme", "random", "1.2.3.4:12345")
	if err != nil {
		t.Fatalf("ExportPeer failed: %v", err)
	}

	if clientCfg.Peer.Endpoint != "1.2.3.4:12345" {
		t.Errorf("expected endpoint '1.2.3.4:12345', got '%s'", clientCfg.Peer.Endpoint)
	}
	if clientCfg.Peer.PublicKey != "serverpub=" {
		t.Errorf("expected server public key, got '%s'", clientCfg.Peer.PublicKey)
	}
	if clientCfg.Interface.PrivateKey != "clientpriv=" {
		t.Errorf("expected client private key, got '%s'", clientCfg.Interface.PrivateKey)
	}
}

func TestLoadServerConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")

	cfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: "priv=", Address: "10.8.0.1/24",
			ListenPort: 12345, MTU: 1280,
		},
		Obfuscation: ServerObfuscationConfig{
			Jc: 5, Jmin: 100, Jmax: 200, S1: 10, S2: 20, S3: 30, S4: 5,
			H1: HeaderRange{Min: 100, Max: 200},
			H2: HeaderRange{Min: 300, Max: 400},
			H3: HeaderRange{Min: 500, Max: 600},
			H4: HeaderRange{Min: 700, Max: 800},
		},
		Peers: []PeerConfig{},
	}

	if err := SaveServerConfig(path, cfg); err != nil {
		t.Fatalf("SaveServerConfig failed: %v", err)
	}

	loaded, err := LoadServerConfig(path)
	if err != nil {
		t.Fatalf("LoadServerConfig failed: %v", err)
	}

	if loaded.Interface.PrivateKey != cfg.Interface.PrivateKey {
		t.Error("PrivateKey mismatch")
	}
}

func TestBuildPeerConfig_CustomDNSAndKeepalive(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")

	cfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey:          "serverpriv=",
			PublicKey:           "serverpub=",
			Address:             "10.8.0.1/24",
			ListenPort:          12345,
			MTU:                 1280,
			DNS:                 "9.9.9.9, 1.0.0.1",
			PersistentKeepalive: 15,
		},
		Obfuscation: ServerObfuscationConfig{
			Jc: 5, Jmin: 100, Jmax: 200, S1: 10, S2: 20, S3: 30, S4: 5,
			H1: HeaderRange{Min: 100, Max: 200},
			H2: HeaderRange{Min: 300, Max: 400},
			H3: HeaderRange{Min: 500, Max: 600},
			H4: HeaderRange{Min: 700, Max: 800},
		},
		Peers: []PeerConfig{
			{
				Name:         "testpeer",
				PrivateKey:   "clientpriv=",
				PublicKey:    "clientpub=",
				PresharedKey: "psk=",
				AllowedIPs:   "10.8.0.2/32",
			},
		},
	}

	mgr := NewManager(path)
	_ = mgr.Save(cfg)

	clientCfg, err := mgr.ExportPeer("testpeer", "quic", "1.2.3.4:12345")
	if err != nil {
		t.Fatalf("ExportPeer failed: %v", err)
	}

	if clientCfg.Interface.DNS != "9.9.9.9, 1.0.0.1" {
		t.Errorf("expected DNS '9.9.9.9, 1.0.0.1', got '%s'", clientCfg.Interface.DNS)
	}
	if clientCfg.Peer.PersistentKeepalive != 15 {
		t.Errorf("expected PersistentKeepalive 15, got %d", clientCfg.Peer.PersistentKeepalive)
	}
}

func TestBuildPeerConfig_DefaultDNSAndKeepalive(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")

	cfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: "serverpriv=",
			PublicKey:  "serverpub=",
			Address:    "10.8.0.1/24",
			ListenPort: 12345,
			MTU:        1280,
		},
		Obfuscation: ServerObfuscationConfig{
			Jc: 5, Jmin: 100, Jmax: 200, S1: 10, S2: 20, S3: 30, S4: 5,
			H1: HeaderRange{Min: 100, Max: 200},
			H2: HeaderRange{Min: 300, Max: 400},
			H3: HeaderRange{Min: 500, Max: 600},
			H4: HeaderRange{Min: 700, Max: 800},
		},
		Peers: []PeerConfig{
			{
				Name:         "testpeer",
				PrivateKey:   "clientpriv=",
				PublicKey:    "clientpub=",
				PresharedKey: "psk=",
				AllowedIPs:   "10.8.0.2/32",
			},
		},
	}

	mgr := NewManager(path)
	_ = mgr.Save(cfg)

	clientCfg, err := mgr.ExportPeer("testpeer", "quic", "1.2.3.4:12345")
	if err != nil {
		t.Fatalf("ExportPeer failed: %v", err)
	}

	if clientCfg.Interface.DNS != "1.1.1.1, 8.8.8.8" {
		t.Errorf("expected default DNS '1.1.1.1, 8.8.8.8', got '%s'", clientCfg.Interface.DNS)
	}
	if clientCfg.Peer.PersistentKeepalive != 25 {
		t.Errorf("expected default PersistentKeepalive 25, got %d", clientCfg.Peer.PersistentKeepalive)
	}
}

func TestSaveServerConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")

	cfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: "priv=", Address: "10.8.0.1/24",
			ListenPort: 12345, MTU: 1280,
		},
		Obfuscation: ServerObfuscationConfig{
			Jc: 5, Jmin: 100, Jmax: 200, S1: 10, S2: 20, S3: 30, S4: 5,
			H1: HeaderRange{Min: 100, Max: 200},
			H2: HeaderRange{Min: 300, Max: 400},
			H3: HeaderRange{Min: 500, Max: 600},
			H4: HeaderRange{Min: 700, Max: 800},
		},
		Peers: []PeerConfig{},
	}

	err := SaveServerConfig(path, cfg)
	if err != nil {
		t.Fatalf("SaveServerConfig failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "[Interface]") {
		t.Error("expected [Interface] section")
	}
	if !strings.Contains(content, "PrivateKey = priv=") {
		t.Error("expected PrivateKey in output")
	}
}
